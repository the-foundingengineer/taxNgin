package taxengine

import (
	"strings"
	"testing"
	"time"
)

func mustApproval() Approval {
	t := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	return Approval{
		Name:         "Dr. Test CITN",
		MembershipNo: "ACTI/00123",
		ApprovedAt:   &t,
		Scope:        "test scope",
	}
}

func baseRuleSet(t *testing.T) RuleSet {
	t.Helper()
	upper800K := Naira(800000)
	upper3M := Naira(3000000)
	return RuleSet{
		Jurisdiction:     "NG",
		YearOfAssessment: 2026,
		Version:          "0.1.0-test",
		EffectiveFrom:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:           StatusPublished,
		ApprovedBy:       mustApproval(),
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

func TestValidateDraftRejectedWithoutAllowDraft(t *testing.T) {
	rs := baseRuleSet(t)
	rs.Status = StatusDraft
	rs.ApprovedBy = Approval{}

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected validation to reject DRAFT rule set without AllowDraft")
	}
	if !strings.Contains(err.Error(), "not signed off") {
		t.Errorf("expected 'not signed off' in error, got: %v", err)
	}
}

func TestValidateDraftAllowed(t *testing.T) {
	rs := baseRuleSet(t)
	rs.Status = StatusDraft
	rs.ApprovedBy = Approval{}

	err := rs.Validate(LoadOptions{AllowDraft: true})
	if err != nil {
		t.Fatalf("AllowDraft=true should pass DRAFT, got error: %v", err)
	}
}

func TestValidatePublishedWithoutApprover(t *testing.T) {
	rs := baseRuleSet(t)
	rs.Status = StatusPublished
	rs.ApprovedBy = Approval{}

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for PUBLISHED rule set with no approver")
	}
	if !strings.Contains(err.Error(), "no approver") {
		t.Errorf("expected 'no approver' in error, got: %v", err)
	}
}

func TestValidatePublishedWithoutApprovalDate(t *testing.T) {
	rs := baseRuleSet(t)
	rs.Status = StatusPublished
	rs.ApprovedBy.ApprovedAt = nil

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for PUBLISHED rule set with no approval date")
	}
}

func TestValidateFinalBandHasNilUpper(t *testing.T) {
	rs := baseRuleSet(t)
	notNil := Naira(100000000)
	rs.Bands[2].UpperKobo = &notNil

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for final band with non-nil upper_kobo")
	}
	if !strings.Contains(err.Error(), "final band must have upper_kobo: null") {
		t.Errorf("expected final band error, got: %v", err)
	}
}

func TestValidateNonFinalBandHasNilUpper(t *testing.T) {
	rs := baseRuleSet(t)
	rs.Bands[1].UpperKobo = nil

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for non-final band with nil upper_kobo")
	}
	if !strings.Contains(err.Error(), "is not final") {
		t.Errorf("expected non-final band error, got: %v", err)
	}
}

func TestValidateBandsOutOfOrder(t *testing.T) {
	rs := baseRuleSet(t)
	upper1 := Naira(3000000)
	upper2 := Naira(800000)
	rs.Bands[0].UpperKobo = &upper1
	rs.Bands[1].UpperKobo = &upper2

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for bands out of order")
	}
	if !strings.Contains(err.Error(), "does not exceed previous") {
		t.Errorf("expected band ordering error, got: %v", err)
	}
}

func TestValidateBandsEqualUpper(t *testing.T) {
	rs := baseRuleSet(t)
	same := Naira(800000)
	rs.Bands[1].UpperKobo = &same

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for bands with equal upper bounds")
	}
}

func TestValidateDuplicateReliefCode(t *testing.T) {
	rs := baseRuleSet(t)
	rs.Reliefs = []ReliefDef{
		{Code: "PENSION", Kind: ReliefActual},
		{Code: "PENSION", Kind: ReliefActual},
	}

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for duplicate relief code")
	}
	if !strings.Contains(err.Error(), "duplicate relief code") {
		t.Errorf("expected duplicate relief code error, got: %v", err)
	}
}

func TestValidateRateOfBaseNoBase(t *testing.T) {
	rs := baseRuleSet(t)
	rs.Reliefs = []ReliefDef{
		{Code: "RENT", Kind: ReliefRateOfBase, Base: "", RateBP: 2000},
	}

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for rate_of_base relief with no base")
	}
	if !strings.Contains(err.Error(), "names no base") {
		t.Errorf("expected 'names no base' error, got: %v", err)
	}
}

func TestValidateRateOfBaseNoRate(t *testing.T) {
	rs := baseRuleSet(t)
	rs.Reliefs = []ReliefDef{
		{Code: "RENT", Kind: ReliefRateOfBase, Base: "rent", RateBP: 0},
	}

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for rate_of_base relief with no rate")
	}
	if !strings.Contains(err.Error(), "has no rate") {
		t.Errorf("expected 'has no rate' error, got: %v", err)
	}
}

func TestValidateUnknownRoundingMode(t *testing.T) {
	rs := baseRuleSet(t)
	rs.Arithmetic.RoundingMode = "bankers"

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for unknown rounding mode")
	}
	if !strings.Contains(err.Error(), "unknown rounding_mode") {
		t.Errorf("expected unknown rounding_mode error, got: %v", err)
	}
}

func TestValidateUnitNotKobo(t *testing.T) {
	rs := baseRuleSet(t)
	rs.Arithmetic.Unit = "naira"

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for unit != kobo")
	}
	if !strings.Contains(err.Error(), "unit must be kobo") {
		t.Errorf("expected 'unit must be kobo' error, got: %v", err)
	}
}

func TestValidateNoJurisdiction(t *testing.T) {
	rs := baseRuleSet(t)
	rs.Jurisdiction = ""

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for no jurisdiction")
	}
}

func TestValidateNoYearOfAssessment(t *testing.T) {
	rs := baseRuleSet(t)
	rs.YearOfAssessment = 0

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for no year_of_assessment")
	}
}

func TestValidateNoVersion(t *testing.T) {
	rs := baseRuleSet(t)
	rs.Version = ""

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for no version")
	}
}

func TestValidateNoBands(t *testing.T) {
	rs := baseRuleSet(t)
	rs.Bands = nil

	err := rs.Validate(LoadOptions{AllowDraft: false})
	if err == nil {
		t.Fatal("expected error for no bands")
	}
}
