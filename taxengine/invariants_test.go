package taxengine

import (
	"testing"
	"time"
)

func validDraft() *RuleSet {
	upper800K := Naira(800000)
	upper3M := Naira(3000000)
	return &RuleSet{
		Jurisdiction:     "NG",
		YearOfAssessment: 2026,
		Version:          "0.1.0-test",
		EffectiveFrom:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:           StatusDraft,
		Arithmetic: Arithmetic{
			Unit:         "kobo",
			RateUnit:     "basis_points",
			RoundingMode: RoundHalfUp,
		},
		Bands: []Band{
			{UpperKobo: &upper800K, RateBP: 0, Label: "First 800k"},
			{UpperKobo: &upper3M, RateBP: 1500, Label: "800k-3M"},
			{UpperKobo: nil, RateBP: 2500, Label: "Above 3M"},
		},
	}
}

func ptrKobo(k Kobo) *Kobo {
	return &k
}


func TestComputeIdempotentAndHashStable(t *testing.T) {
	rs := validDraft()
	f := TaxFacts{
		YearOfAssessment: 2026,
		Jurisdiction:     "NG",
		Income: []IncomeItem{
			{Category: IncDividend, AmountKobo: Naira(3_000_000), Label: "div"},
		},
		MonthsEmployed: 12,
	}
	h1, err := f.Hash()
	if err != nil {
		t.Fatal(err)
	}
	a1, tr1, err := Compute(f, rs)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := f.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatal("Hash mutated inputs")
	}
	if f.Income[0].Label != "div" {
		t.Fatal("Hash reordered caller slice in place")
	}
	a2, tr2, err := Compute(f, rs)
	if err != nil {
		t.Fatal(err)
	}
	if a1 != a2 {
		t.Fatalf("Compute not idempotent: %+v vs %+v", a1, a2)
	}
	if len(tr1.Steps) != len(tr2.Steps) {
		t.Fatal("trace length changed")
	}
	equal := f
	h3, _ := equal.Hash()
	if h3 != h1 {
		t.Fatal("equal facts produced unequal Hash")
	}
}

func TestInvariantsOverChargeableGrid(t *testing.T) {
	rs := validDraft()
	rs.Bands = []Band{
		{UpperKobo: ptrKobo(Naira(800_000)), RateBP: 0, Label: "0"},
		{UpperKobo: ptrKobo(Naira(3_000_000)), RateBP: 1500, Label: "15"},
		{UpperKobo: ptrKobo(Naira(12_000_000)), RateBP: 1800, Label: "18"},
		{UpperKobo: ptrKobo(Naira(25_000_000)), RateBP: 2100, Label: "21"},
		{UpperKobo: ptrKobo(Naira(50_000_000)), RateBP: 2300, Label: "23"},
		{UpperKobo: nil, RateBP: 2500, Label: "25"},
	}
	top := rs.Bands[len(rs.Bands)-1].RateBP

	var prevTax Kobo = -1
	var prevTakeHome Kobo
	for naira := int64(0); naira <= 60_000_000; naira += 17_777 {
		f := TaxFacts{
			YearOfAssessment: 2026,
			Income:           []IncomeItem{{Category: IncBusiness, AmountKobo: Naira(naira)}},
			MonthsEmployed:   12,
		}
		a, _, err := Compute(f, rs)
		if err != nil {
			t.Fatal(err)
		}
		if prevTax >= 0 && a.TaxPayableKobo < prevTax {
			t.Fatalf("tax fell at ₦%d: %s -> %s", naira, prevTax.Format(), a.TaxPayableKobo.Format())
		}
		takeHome := a.GrossIncomeKobo - a.TaxPayableKobo
		if naira > 0 && takeHome < prevTakeHome {
			t.Fatalf("cliff at ₦%d: take-home %s -> %s", naira, prevTakeHome.Format(), takeHome.Format())
		}
		if a.GrossIncomeKobo > 0 {
			eff, err := applyRate(a.GrossIncomeKobo, top, RoundHalfUp)
			if err != nil {
				t.Fatal(err)
			}
			if a.TaxPayableKobo > Naira(naira)*Kobo(top)/Kobo(BasisPointScale)+1 {
				_ = eff
			}
			if a.TaxPayableKobo > applyTopBound(a.GrossIncomeKobo, top) {
				t.Fatalf("effective rate exceeded top band at ₦%d: tax %s", naira, a.TaxPayableKobo.Format())
			}
		}
		a2, _, _ := Compute(f, rs)
		if a != a2 {
			t.Fatal("idempotency failed in grid")
		}
		prevTax = a.TaxPayableKobo
		prevTakeHome = takeHome
	}
}

func applyTopBound(gross Kobo, top BasisPoints) Kobo {
	s, err := scaled(gross, top)
	if err != nil {
		return gross
	}
	k, err := descale(s, RoundHalfUp)
	if err != nil {
		return gross
	}
	return k
}

func TestUnknownReliefErrors(t *testing.T) {
	rs := validDraft()
	f := TaxFacts{
		YearOfAssessment: 2026,
		Income:           []IncomeItem{{Category: IncBusiness, AmountKobo: Naira(1_000_000)}},
		Reliefs:          []ReliefClaim{{Code: "NOT_A_RELIEF", AmountKobo: Naira(1)}},
		MonthsEmployed:   12,
	}
	_, _, err := Compute(f, rs)
	if err == nil {
		t.Fatal("want unknown relief error")
	}
}
