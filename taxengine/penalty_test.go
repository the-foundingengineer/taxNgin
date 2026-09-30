package taxengine

import (
	"testing"
	"time"
)

func makePenaltyRuleSet(t *testing.T) *RuleSet {
	t.Helper()
	upper800K := Naira(800000)
	upper3M := Naira(3000000)
	deadline := time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)
	app := mustApproval()
	rs := &RuleSet{
		Jurisdiction:     "NG",
		YearOfAssessment: 2026,
		Version:          "0.1.0-test",
		EffectiveFrom:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:           StatusPublished,
		ApprovedBy:       app,
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
	rs.Penalties.FilingDeadline = deadline
	rs.Penalties.LateFiling.FirstMonthKobo = Naira(100000)
	rs.Penalties.LateFiling.PerSubsequentMonthKobo = Naira(50000)
	rs.Penalties.LateFiling.Citation = "NTAA 2025 s.101"
	rs.Penalties.LatePayment.SurchargeBP = 1000
	rs.Penalties.LatePayment.InterestBasis = "CBN_MPR"
	rs.Penalties.ObjectionWindowDays = 30
	return rs
}

func TestPenaltyOnTime(t *testing.T) {
	rs := makePenaltyRuleSet(t)
	deadline := rs.Penalties.FilingDeadline

	pa, err := ComputePenalties(PenaltyInput{
		FilingDeadline: deadline,
		FiledOn:        deadline,
		AsOf:           deadline,
		UnpaidKobo:     0,
	}, rs)
	if err != nil {
		t.Fatalf("ComputePenalties error: %v", err)
	}
	if got, want := pa.MonthsInDefault, 0; got != want {
		t.Errorf("months in default: got %d, want %d", got, want)
	}
	if got, want := pa.LateFilingKobo, Kobo(0); got != want {
		t.Errorf("late filing: got %d, want %d", got, want)
	}
	if got, want := pa.TotalKobo, Kobo(0); got != want {
		t.Errorf("total: got %d, want %d", got, want)
	}
}

func TestPenaltyOneDayLate(t *testing.T) {
	rs := makePenaltyRuleSet(t)
	deadline := rs.Penalties.FilingDeadline
	filed := deadline.AddDate(0, 0, 1) // 1 April

	pa, err := ComputePenalties(PenaltyInput{
		FilingDeadline: deadline,
		FiledOn:        filed,
		AsOf:           filed,
		UnpaidKobo:     0,
	}, rs)
	if err != nil {
		t.Fatalf("ComputePenalties error: %v", err)
	}
	if got, want := pa.MonthsInDefault, 1; got != want {
		t.Errorf("months in default: got %d, want %d", got, want)
	}
	if got, want := pa.LateFilingKobo, Naira(100000); got != want {
		t.Errorf("late filing: got %s, want %s", got.Format(), want.Format())
	}
	if got, want := pa.TotalKobo, Naira(100000); got != want {
		t.Errorf("total: got %s, want %s", got.Format(), want.Format())
	}
}

func TestPenaltySixMonthsLate(t *testing.T) {
	rs := makePenaltyRuleSet(t)
	deadline := rs.Penalties.FilingDeadline
	// 30 September 2027 is roughly 6 months after 31 March
	filed := time.Date(2027, 9, 30, 0, 0, 0, 0, time.UTC)

	pa, err := ComputePenalties(PenaltyInput{
		FilingDeadline: deadline,
		FiledOn:        filed,
		AsOf:           filed,
		UnpaidKobo:     0,
	}, rs)
	if err != nil {
		t.Fatalf("ComputePenalties error: %v", err)
	}
	// April = month 1, Sep = month 6
	wantMonths := 6
	if got := pa.MonthsInDefault; got != wantMonths {
		t.Errorf("months in default: got %d, want %d", got, wantMonths)
	}
	// 100k + 50k * 5 = 350k
	wantFiling := Naira(350000)
	if got := pa.LateFilingKobo; got != wantFiling {
		t.Errorf("late filing: got %s, want %s", got.Format(), wantFiling.Format())
	}
}

func TestPenaltyTwelveMonthsLate(t *testing.T) {
	rs := makePenaltyRuleSet(t)
	deadline := rs.Penalties.FilingDeadline
	// 31 March following year
	filed := time.Date(2028, 3, 31, 0, 0, 0, 0, time.UTC)

	pa, err := ComputePenalties(PenaltyInput{
		FilingDeadline: deadline,
		FiledOn:        filed,
		AsOf:           filed,
		UnpaidKobo:     0,
	}, rs)
	if err != nil {
		t.Fatalf("ComputePenalties error: %v", err)
	}
	// April 2027 -> March 2028 = 12 months
	wantMonths := 12
	if got := pa.MonthsInDefault; got != wantMonths {
		t.Errorf("months in default: got %d, want %d", got, wantMonths)
	}
	// 100k + 50k * 11 = 650k
	wantFiling := Naira(650000)
	if got := pa.LateFilingKobo; got != wantFiling {
		t.Errorf("late filing: got %s, want %s", got.Format(), wantFiling.Format())
	}
}

func TestPenaltyLatePaymentSurchargeAndInterest(t *testing.T) {
	rs := makePenaltyRuleSet(t)
	deadline := rs.Penalties.FilingDeadline
	paymentDue := deadline
	asOf := paymentDue.AddDate(0, 0, 30)
	mpr := BasisPoints(2750) // 27.5% CBN MPR

	pa, err := ComputePenalties(PenaltyInput{
		FilingDeadline:          deadline,
		FiledOn:                 deadline,
		AsOf:                    asOf,
		UnpaidKobo:              Naira(1730400),
		PaymentDueOn:            paymentDue,
		CBNMonetaryPolicyRateBP: mpr,
	}, rs)
	if err != nil {
		t.Fatalf("ComputePenalties error: %v", err)
	}
	// Surcharge: 10% of 1,730,400 = 173,040
	wantSurcharge := Naira(173040)
	if got := pa.LatePaymentSurchargeKobo; got != wantSurcharge {
		t.Errorf("surcharge: got %s, want %s", got.Format(), wantSurcharge.Format())
	}
	// Interest: 1,730,400 * 27.5% * 30 / 365 ~= 39,111.78 -> 39,112 naira with half-up
	// Let's compute: 173040000 * 2750 * 30 / (10000*365) = 173040000 * 82500 / 3650000
	// = 14275800000000 / 3650000 = 3,911,178 kobo = 39,111.78 naira
	// Half-up: 3911178 kobo? Let me be more precise:
	// 173040000 (kobo) * 2750 (bp) = 475860000000; * 30 days = 14275800000000
	// / (10000 * 365) = / 3650000
	// 14275800000000 / 3650000 = 3911178.082... -> half-up -> 3911178 kobo = 39,111.78
	wantInterestKobo := Kobo(3911178)
	if got := pa.LatePaymentInterestKobo; got != wantInterestKobo {
		t.Errorf("interest: got %s (%d), want %s (%d)", got.Format(), got, wantInterestKobo.Format(), wantInterestKobo)
	}
}

func TestPenaltyNoMPRReturnsError(t *testing.T) {
	rs := makePenaltyRuleSet(t)
	deadline := rs.Penalties.FilingDeadline
	paymentDue := deadline
	asOf := paymentDue.AddDate(0, 0, 30)

	_, err := ComputePenalties(PenaltyInput{
		FilingDeadline:          deadline,
		FiledOn:                 deadline,
		AsOf:                    asOf,
		UnpaidKobo:              Naira(1000000),
		PaymentDueOn:            paymentDue,
		CBNMonetaryPolicyRateBP: 0,
	}, rs)
	if err == nil {
		t.Fatal("expected error when MPR is zero but interest would be due")
	}
}

func TestPenaltyNilRuleSet(t *testing.T) {
	_, err := ComputePenalties(PenaltyInput{}, nil)
	if err == nil {
		t.Fatal("expected error for nil rule set")
	}
}
