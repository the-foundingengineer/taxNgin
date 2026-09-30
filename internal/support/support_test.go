package support_test

import (
	"regexp"
	"testing"
	"time"

	"taxsoftware/internal/returns"
	"taxsoftware/internal/support"
)

// ninPattern matches any run of 11 or more consecutive digits.
var ninPattern = regexp.MustCompile(`\d{11,}`)

// allStringFields collects every string value from a SupportView so tests can
// check that no financial data or identity information has leaked into them.
func allStringFields(sv support.SupportView) []string {
	var out []string
	out = append(out, sv.ReturnID)
	out = append(out, sv.Jurisdiction)
	out = append(out, string(sv.State))
	out = append(out, sv.BlockingIssueCodes...)
	out = append(out, sv.WarningCodes...)
	for k := range sv.IncomeCategoryCount {
		out = append(out, k)
	}
	for _, e := range sv.Timeline {
		out = append(out, string(e.State))
	}
	return out
}

// containsSubstr is a plain substring check used in preference to strings.Contains
// so the test has no implicit dependency on strings package behaviour changes.
func containsSubstr(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// minimalProjection returns a Projection with only the mandatory maps
// initialised so callers can add specific fields without nil-map panics.
func minimalProjection() *returns.Projection {
	return &returns.Projection{
		ReturnID:             "R-SUPPORT-001",
		YearOfAssessment:     2026,
		Jurisdiction:         "NG",
		State:                returns.StateDraft,
		Income:               make(map[string]returns.IncomeLine),
		Reliefs:              make(map[string]returns.ReliefLine),
		Bases:                make(map[string]int64),
		ProposedPayroll:      make(map[string]returns.PayrollProposal),
		AcknowledgedWarnings: make(map[string]bool),
		SeenIdempotencyKeys:  make(map[string]bool),
	}
}

// TestBuildSupportView_NeverExposesAmounts asserts that a SupportView built
// from a projection with employment income of ₦9m and a PAYE credit of ₦1.6m
// contains no string field that could reconstruct either amount.
func TestBuildSupportView_NeverExposesAmounts(t *testing.T) {
	p := minimalProjection()
	p.Income["emp1"] = returns.IncomeLine{
		ID:         "emp1",
		Category:   "EMPLOYMENT",
		AmountKobo: 900_000_000, // ₦9,000,000
	}
	p.Assessment = &returns.AssessmentSnapshot{
		TotalCreditsKobo: 160_000_000, // ₦1,600,000 PAYE
	}

	sv := support.BuildSupportView(p)

	// These are the amount fragments (kobo and naira representations) that must
	// not appear in any string field. A substring match is sufficient — the
	// point is that no amount string leaks through the redaction boundary.
	forbidden := []string{
		"900000000",  // ₦9m in kobo
		"9000000",    // ₦9m in naira (no decimal)
		"160000000",  // ₦1.6m in kobo
		"1600000",    // ₦1.6m in naira
	}

	for _, s := range allStringFields(sv) {
		for _, frag := range forbidden {
			if containsSubstr(s, frag) {
				t.Errorf("SupportView string field %q contains amount fragment %q — financial data leaked through redaction", s, frag)
			}
		}
	}
}

// TestBuildSupportView_IncomeCategoriesCorrect asserts that IncomeCategoryCount
// reflects two EMPLOYMENT lines and one DIVIDEND line by category only.
func TestBuildSupportView_IncomeCategoriesCorrect(t *testing.T) {
	p := minimalProjection()
	p.Income["emp1"] = returns.IncomeLine{ID: "emp1", Category: "EMPLOYMENT", AmountKobo: 500_000_000}
	p.Income["emp2"] = returns.IncomeLine{ID: "emp2", Category: "EMPLOYMENT", AmountKobo: 300_000_000}
	p.Income["div1"] = returns.IncomeLine{ID: "div1", Category: "DIVIDEND", AmountKobo: 100_000_000}

	sv := support.BuildSupportView(p)

	if sv.IncomeCategoryCount["EMPLOYMENT"] != 2 {
		t.Errorf("expected EMPLOYMENT count 2, got %d", sv.IncomeCategoryCount["EMPLOYMENT"])
	}
	if sv.IncomeCategoryCount["DIVIDEND"] != 1 {
		t.Errorf("expected DIVIDEND count 1, got %d", sv.IncomeCategoryCount["DIVIDEND"])
	}
	if len(sv.IncomeCategoryCount) != 2 {
		t.Errorf("expected 2 categories total, got %d", len(sv.IncomeCategoryCount))
	}
}

// TestBuildSupportView_BlockingCodesPresent asserts that blocking issue codes
// are surfaced verbatim from the projection's BlockingErrors slice.
func TestBuildSupportView_BlockingCodesPresent(t *testing.T) {
	p := minimalProjection()
	p.BlockingErrors = []string{"E001", "E005"}

	sv := support.BuildSupportView(p)

	if len(sv.BlockingIssueCodes) != 2 {
		t.Fatalf("expected 2 blocking codes, got %d", len(sv.BlockingIssueCodes))
	}
	if sv.BlockingIssueCodes[0] != "E001" || sv.BlockingIssueCodes[1] != "E005" {
		t.Errorf("unexpected blocking codes: %v", sv.BlockingIssueCodes)
	}
}

// TestBuildSupportView_PayrollProposalCounts asserts that proposal counts are
// correctly tallied from the ProposedPayroll map.
func TestBuildSupportView_PayrollProposalCounts(t *testing.T) {
	p := minimalProjection()
	p.ProposedPayroll = map[string]returns.PayrollProposal{
		"pr1": {RecordID: "pr1", EmployerID: "EMP-A", GrossKobo: 600_000_000, Status: "ACCEPTED"},
		"pr2": {RecordID: "pr2", EmployerID: "EMP-B", GrossKobo: 400_000_000, Status: "REJECTED"},
	}

	sv := support.BuildSupportView(p)

	if sv.PayrollProposalCount != 2 {
		t.Errorf("expected 2 proposals, got %d", sv.PayrollProposalCount)
	}
	if sv.PayrollAcceptedCount != 1 {
		t.Errorf("expected 1 accepted, got %d", sv.PayrollAcceptedCount)
	}
	if sv.PayrollRejectedCount != 1 {
		t.Errorf("expected 1 rejected, got %d", sv.PayrollRejectedCount)
	}
}

// TestBuildSupportView_TimelineOrdered folds events in ascending seq order and
// asserts that the resulting Timeline slice is also ascending by OccurredAt.
func TestBuildSupportView_TimelineOrdered(t *testing.T) {
	base := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	t1 := base
	t2 := base.Add(24 * time.Hour)
	t3 := base.Add(48 * time.Hour)

	p := minimalProjection()
	p.State = returns.StateSelfAttested
	p.SignedAt = &t1
	p.PackIssuedAt = &t2
	p.AttestedAt = &t3

	sv := support.BuildSupportView(p)

	if len(sv.Timeline) != 3 {
		t.Fatalf("expected 3 timeline entries, got %d", len(sv.Timeline))
	}
	for i := 1; i < len(sv.Timeline); i++ {
		if !sv.Timeline[i].OccurredAt.After(sv.Timeline[i-1].OccurredAt) {
			t.Errorf("timeline not ascending at index %d: %v not after %v",
				i, sv.Timeline[i].OccurredAt, sv.Timeline[i-1].OccurredAt)
		}
	}
	if sv.Timeline[0].State != returns.StateSigned {
		t.Errorf("expected first entry SIGNED, got %s", sv.Timeline[0].State)
	}
	if sv.Timeline[1].State != returns.StatePackIssued {
		t.Errorf("expected second entry PACK_ISSUED, got %s", sv.Timeline[1].State)
	}
	if sv.Timeline[2].State != returns.StateSelfAttested {
		t.Errorf("expected third entry SELF_ATTESTED, got %s", sv.Timeline[2].State)
	}
}

// TestBuildSupportView_NeverExposesNIN asserts that no 11-digit sequence (the
// shape of a Nigerian NIN) appears in any string field of the SupportView, even
// when the underlying projection's TaxpayerID is NIN-shaped.
func TestBuildSupportView_NeverExposesNIN(t *testing.T) {
	p := minimalProjection()
	p.TaxpayerID = "12345678901" // 11-digit NIN-like identifier

	sv := support.BuildSupportView(p)

	for _, s := range allStringFields(sv) {
		if ninPattern.MatchString(s) {
			t.Errorf("SupportView field %q contains an 11-digit sequence — possible NIN or identity data exposure", s)
		}
	}
}
