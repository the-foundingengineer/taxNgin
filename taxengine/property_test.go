package taxengine_test

import (
	"testing"

	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"

	"taxsoftware/taxengine"
)

func loadPropertyRuleSet(t *testing.T) *taxengine.RuleSet {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "rules", "ng-2026.yaml"))
	if err != nil {
		t.Fatalf("read rule set: %v", err)
	}
	rs, err := taxengine.Load(raw, yaml.Unmarshal, taxengine.LoadOptions{AllowDraft: true})
	if err != nil {
		t.Fatalf("load rule set: %v", err)
	}
	return rs
}

func nairaK(n int64) taxengine.Kobo {
	return taxengine.Kobo(n * 100)
}

func computeForEmployment(t *testing.T, rs *taxengine.RuleSet, grossNaira int64) (grossKobo, chargeableKobo, taxPayableKobo taxengine.Kobo) {
	t.Helper()
	facts := taxengine.TaxFacts{
		YearOfAssessment: 2026,
		Jurisdiction:     "NG",
		IsResident:       true,
		Income: []taxengine.IncomeItem{
			{Category: taxengine.IncEmployment, AmountKobo: nairaK(grossNaira), Label: "Salary"},
		},
		Reliefs:        nil,
		Credits:        nil,
		DeclaredBases:  nil,
		MonthsEmployed: 12,
	}
	a, _, err := taxengine.Compute(facts, rs)
	if err != nil {
		t.Fatalf("Compute for gross %d naira: %v", grossNaira, err)
	}
	return a.GrossIncomeKobo, a.ChargeableIncomeKobo, a.TaxPayableKobo
}

func TestPropertyMonotonicTax(t *testing.T) {
	rs := loadPropertyRuleSet(t)

	samples := []int64{
		0, 1, 799999, 800000, 800001, 2999999, 3000000, 3000001,
		11999999, 12000000, 12000001,
		24999999, 25000000, 25000001,
		49999999, 50000000, 50000001,
		60000000, 100000000, 1000000000,
	}

	var prevTax taxengine.Kobo = -1
	for i, n := range samples {
		_, _, tax := computeForEmployment(t, rs, n)
		if i > 0 && tax < prevTax {
			t.Errorf("tax decreased at gross %d: tax=%d, prev=%d", n, tax, prevTax)
		}
		prevTax = tax
	}

	for n := int64(0); n < 2000000; n += 50000 {
		_, _, tax1 := computeForEmployment(t, rs, n)
		_, _, tax2 := computeForEmployment(t, rs, n+1)
		if tax2 < tax1 {
			t.Fatalf("tax decreased at gross %d -> %d: %d -> %d", n, n+1, tax1, tax2)
		}
	}
}

func TestPropertyNoCliff(t *testing.T) {
	rs := loadPropertyRuleSet(t)

	steps := []int64{
		799999, 800000,
		2999999, 3000000,
		11999999, 12000000,
		24999999, 25000000,
		49999999, 50000000,
	}

	for _, base := range steps {
		for delta := int64(-2); delta <= 2; delta++ {
			n := base + delta
			if n < 0 {
				continue
			}
			g1, _, t1 := computeForEmployment(t, rs, n)
			g2, _, t2 := computeForEmployment(t, rs, n+1)
			takeHome1 := g1 - t1
			takeHome2 := g2 - t2
			if takeHome2 < takeHome1 {
				t.Errorf("cliff at gross %d -> %d: take-home %d -> %d (tax %d -> %d",
					n, n+1, takeHome1, takeHome2, t1, t2)
			}
		}
	}

	for n := int64(0); n < 5000000; n += 100000 {
		g1, _, t1 := computeForEmployment(t, rs, n)
		g2, _, t2 := computeForEmployment(t, rs, n+1)
		takeHome1 := g1 - t1
		takeHome2 := g2 - t2
		if takeHome2 < takeHome1 {
			t.Fatalf("cliff at gross %d -> %d: take-home %d -> %d", n, n+1, takeHome1, takeHome2)
		}
	}
}

func TestPropertyEffectiveRateCap(t *testing.T) {
	rs := loadPropertyRuleSet(t)

	topMarginalRate := taxengine.Kobo(2500)
	bpScale := int64(10000)

	samples := []int64{
		1, 800000, 3000000, 12000000, 25000000, 50000000,
		50000001, 60000000, 100000000, 1000000000,
	}

	for _, n := range samples {
		_, chargeable, tax := computeForEmployment(t, rs, n)
		if chargeable == 0 {
			continue
		}
		taxBP := int64(tax) * bpScale / int64(chargeable)
		if taxBP > int64(topMarginalRate) {
			t.Errorf("effective rate exceeded top marginal at chargeable %s, tax %s, rate %d bp > %d bp",
				chargeable.Format(), tax.Format(), taxBP, topMarginalRate)
		}
	}

	for n := int64(100000); n < 100000000; n += 500000 {
		_, chargeable, tax := computeForEmployment(t, rs, n)
		if chargeable == 0 {
			continue
		}
		taxBP := int64(tax) * bpScale / int64(chargeable)
		if taxBP > int64(topMarginalRate) {
			t.Fatalf("effective rate cap violated at chargeable %d kobo", chargeable)
		}
	}
}

func TestPropertyIdempotentCompute(t *testing.T) {
	rs := loadPropertyRuleSet(t)

	facts := taxengine.TaxFacts{
		YearOfAssessment: 2026,
		Jurisdiction:     "NG",
		IsResident:       true,
		Income: []taxengine.IncomeItem{
			{Category: taxengine.IncEmployment, AmountKobo: nairaK(9000000), Label: "Salary", SourceRef: "acme"},
			{Category: taxengine.IncProfessional, AmountKobo: nairaK(1800000), Label: "Consulting"},
			{Category: taxengine.IncRent, AmountKobo: nairaK(1200000), Label: "Rent"},
		},
		Reliefs: []taxengine.ReliefClaim{
			{Code: "PENSION", AmountKobo: nairaK(720000), HasEvidence: true},
			{Code: "RENT_RELIEF", HasEvidence: true},
		},
		Credits: []taxengine.CreditClaim{
			{Code: "PAYE_REMITTED", AmountKobo: nairaK(1600000)},
		},
		DeclaredBases: map[string]taxengine.Kobo{
			"annual_rent_paid": nairaK(3600000),
		},
		MonthsEmployed: 12,
	}

	h1, err := facts.Hash()
	if err != nil {
		t.Fatalf("first Hash: %v", err)
	}

	a1, tr1, err := taxengine.Compute(facts, rs)
	if err != nil {
		t.Fatalf("first Compute: %v", err)
	}

	h2, err := facts.Hash()
	if err != nil {
		t.Fatalf("second Hash: %v", err)
	}

	if h1 != h2 {
		t.Errorf("Hash not equal across calls: %s vs %s", h1, h2)
	}

	a2, tr2, err := taxengine.Compute(facts, rs)
	if err != nil {
		t.Fatalf("second Compute: %v", err)
	}

	if a1.FactsHash != a2.FactsHash {
		t.Errorf("assessment facts hash mismatch")
	}
	if a1.TaxPayableKobo != a2.TaxPayableKobo {
		t.Errorf("assessment tax payable mismatch: %d vs %d", a1.TaxPayableKobo, a2.TaxPayableKobo)
	}
	if a1.BalanceKobo != a2.BalanceKobo {
		t.Errorf("assessment balance mismatch: %d vs %d", a1.BalanceKobo, a2.BalanceKobo)
	}
	if len(tr1.Steps) != len(tr2.Steps) {
		t.Errorf("trace step count mismatch: %d vs %d", len(tr1.Steps), len(tr2.Steps))
	}
}

func TestPropertyEqualFactsEqualHash(t *testing.T) {
	rs := loadPropertyRuleSet(t)
	_ = rs

	mkFacts := func() taxengine.TaxFacts {
		return taxengine.TaxFacts{
			YearOfAssessment: 2026,
			Jurisdiction:     "NG",
			IsResident:       true,
			Income: []taxengine.IncomeItem{
				{Category: taxengine.IncEmployment, AmountKobo: nairaK(5000000), Label: "Salary"},
			},
			MonthsEmployed: 12,
		}
	}

	f1 := mkFacts()
	f2 := mkFacts()

	h1, err := f1.Hash()
	if err != nil {
		t.Fatal(err)
	}
	h2, err := f2.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if h1 != h2 {
		t.Errorf("equal facts have different hashes: %s vs %s", h1, h2)
	}
}
