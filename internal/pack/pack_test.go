package pack_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"taxsoftware/internal/pack"
	"taxsoftware/internal/portalmap"
	"taxsoftware/internal/returns"
	"taxsoftware/internal/rulesyaml"
	"taxsoftware/taxengine"
)

// ---------------------------------------------------------------------------
// Test infrastructure
// ---------------------------------------------------------------------------

var t0 = time.Date(2027, 1, 15, 9, 0, 0, 0, time.UTC)

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func loadPortalMap(t *testing.T) portalmap.PortalMap {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(), "rules", "portals", "ng-la-etax.yaml"))
	if err != nil {
		t.Fatalf("read portal map: %v", err)
	}
	pm, err := portalmap.Load(raw, rulesyaml.Unmarshal, portalmap.LoadOptions{AllowDraft: true})
	if err != nil {
		t.Fatalf("load portal map: %v", err)
	}
	return pm
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return json.RawMessage(b)
}

func openEvent(seq int) returns.Event {
	return returns.Event{
		Seq:      seq,
		ReturnID: "R-PACK-TEST",
		Kind:     returns.EvReturnOpened,
		Payload: mustPayload(map[string]any{
			"taxpayer_id":        "TP-001",
			"year_of_assessment": 2026,
			"jurisdiction":       "NG-LA",
		}),
		Actor:      returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-001"},
		OccurredAt: t0,
	}
}

// mustPayload marshals v to JSON without a testing.T (used in package-level helpers).
func mustPayload(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(b)
}

// fakeAssessmentPayload builds an EvAssessmentComputed payload with a
// small trace, suitable for tests that do not call taxengine.Compute.
func fakeAssessmentPayload(t *testing.T, published bool) json.RawMessage {
	t.Helper()
	traceSteps := []taxengine.Step{
		{Op: "ADD", Label: "Employment income", AmountKobo: 500_000_000, RunningKobo: 500_000_000},
		{Op: "TOTAL", Label: "Gross income", AmountKobo: 500_000_000, RunningKobo: 500_000_000},
	}
	traceJSON, err := json.Marshal(traceSteps)
	if err != nil {
		t.Fatalf("marshal trace: %v", err)
	}
	snap := returns.AssessmentSnapshot{
		FactsHash:            "test-hash",
		RuleSetVersion:       "v1.0.0",
		RuleSetPublished:     published,
		ChargeableIncomeKobo: 400_000_000,
		TaxPayableKobo:       30_000_000,
		BalanceKobo:          30_000_000,
		ComputedAt:           t0,
		Trace:                json.RawMessage(traceJSON),
	}
	return mustJSON(t, snap)
}

// buildSignedProjection produces a minimal SIGNED projection for tests that
// only need to check pack generation mechanics, not field values.
func buildSignedProjection(t *testing.T) *returns.Projection {
	t.Helper()
	events := []returns.Event{
		openEvent(1),
		{Seq: 2, Kind: returns.EvIncomeDeclared, ReturnID: "R-PACK-TEST",
			Actor:   returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: mustJSON(t, returns.IncomeLine{ID: "e1", Category: "EMPLOYMENT", AmountKobo: 500_000_000, Label: "Acme Ltd"})},
		{Seq: 3, Kind: returns.EvAssessmentComputed, ReturnID: "R-PACK-TEST",
			Actor:   returns.Actor{Kind: returns.ActorSystem},
			Payload: fakeAssessmentPayload(t, true)},
		{Seq: 4, Kind: returns.EvValidationRun, ReturnID: "R-PACK-TEST",
			Payload: []byte(`{"blocking":[],"warnings":[]}`)},
		{Seq: 5, Kind: returns.EvReturnSigned, ReturnID: "R-PACK-TEST",
			OccurredAt: t0},
	}
	p, err := returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if p.State != returns.StateSigned {
		t.Fatalf("buildSignedProjection: expected SIGNED, got %s", p.State)
	}
	return p
}

// ---------------------------------------------------------------------------
// Pack field assertion helpers
// ---------------------------------------------------------------------------

func findPage(t *testing.T, pk pack.Pack, id string) pack.PackPage {
	t.Helper()
	for _, page := range pk.Pages {
		if page.ID == id {
			return page
		}
	}
	t.Fatalf("page %q not found in pack (pages: %v)", id, pageIDs(pk))
	return pack.PackPage{}
}

func pageIDs(pk pack.Pack) []string {
	ids := make([]string, len(pk.Pages))
	for i, p := range pk.Pages {
		ids[i] = p.ID
	}
	return ids
}

// allFields collects all PackFields from all rows of a page.
func allFields(page pack.PackPage) []pack.PackField {
	var out []pack.PackField
	for _, row := range page.Rows {
		out = append(out, row.Fields...)
	}
	return out
}

// findField finds the first field with the given portal label.
func findField(page pack.PackPage, label string) (pack.PackField, bool) {
	for _, f := range allFields(page) {
		if f.PortalLabel == label {
			return f, true
		}
	}
	return pack.PackField{}, false
}

// countFieldsByLabel counts how many fields with the given label exist on the page.
func countFieldsByLabel(page pack.PackPage, label string) int {
	n := 0
	for _, f := range allFields(page) {
		if f.PortalLabel == label {
			n++
		}
	}
	return n
}

func assertFieldValue(t *testing.T, page pack.PackPage, label, want string) {
	t.Helper()
	f, ok := findField(page, label)
	if !ok {
		t.Errorf("field %q not found in page %s", label, page.ID)
		return
	}
	if f.Value != want {
		t.Errorf("field %q in page %s: want %q, got %q", label, page.ID, want, f.Value)
	}
}

func assertFieldMissing(t *testing.T, page pack.PackPage, label string) {
	t.Helper()
	f, ok := findField(page, label)
	if !ok {
		// Field not present at all — could mean it was hidden by a conditional.
		// The test must distinguish "hidden by conditional" from "present but Missing".
		t.Errorf("field %q not found in page %s (was it hidden by a conditional?)", label, page.ID)
		return
	}
	if !f.Missing {
		t.Errorf("field %q in page %s: expected Missing=true, got Missing=false (value=%q)", label, page.ID, f.Value)
	}
}

func assertFieldAbsent(t *testing.T, page pack.PackPage, label string) {
	t.Helper()
	if _, ok := findField(page, label); ok {
		t.Errorf("field %q should be absent from page %s (conditional not satisfied)", label, page.ID)
	}
}

func findPrecondition(pk pack.Pack, id string) (pack.PreconditionResult, bool) {
	for _, pc := range pk.Preconditions {
		if pc.ID == id {
			return pc, true
		}
	}
	return pack.PreconditionResult{}, false
}

// ---------------------------------------------------------------------------
// T-001: Happy path — employment + pension + rent, accommodation declared
// ---------------------------------------------------------------------------

func TestGenerate_HappyPath(t *testing.T) {
	pm := loadPortalMap(t)

	events := []returns.Event{
		openEvent(1),
		// Single employer
		{Seq: 2, Kind: returns.EvIncomeDeclared, ReturnID: "R-PACK-TEST",
			Actor:   returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: mustJSON(t, returns.IncomeLine{ID: "emp1", Category: "EMPLOYMENT", AmountKobo: 500_000_000, Label: "Acme Ltd"})},
		// Pension relief (₦400,000 = 40_000_000 kobo)
		{Seq: 3, Kind: returns.EvReliefClaimed, ReturnID: "R-PACK-TEST",
			Payload: mustJSON(t, returns.ReliefLine{Code: "PENSION", AmountKobo: 40_000_000})},
		// Annual rent paid base (₦1,000,000 = 100_000_000 kobo)
		{Seq: 4, Kind: returns.EvBaseDeclared, ReturnID: "R-PACK-TEST",
			Payload: mustJSON(t, map[string]any{"name": "annual_rent_paid", "amount_kobo": int64(100_000_000)})},
		// Rent relief claim (amount computed by engine from base)
		{Seq: 5, Kind: returns.EvReliefClaimed, ReturnID: "R-PACK-TEST",
			Payload: mustJSON(t, returns.ReliefLine{Code: "RENT_RELIEF"})},
		// Accommodation — Rent, Tenant
		{Seq: 6, Kind: returns.EvAccommodationDeclared, ReturnID: "R-PACK-TEST",
			Actor: returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: mustJSON(t, map[string]any{
				"type":             "Rent",
				"ownership_type":   "Tenant",
				"landlord_name":    "Mr. Adekunle",
				"landlord_address": "22 Awolowo Road, Ikoyi",
			})},
		{Seq: 7, Kind: returns.EvAssessmentComputed, ReturnID: "R-PACK-TEST",
			Actor:   returns.Actor{Kind: returns.ActorSystem},
			Payload: fakeAssessmentPayload(t, true)},
		{Seq: 8, Kind: returns.EvValidationRun, ReturnID: "R-PACK-TEST",
			Payload: []byte(`{"blocking":[],"warnings":[]}`)},
		{Seq: 9, Kind: returns.EvReturnSigned, ReturnID: "R-PACK-TEST", OccurredAt: t0},
	}

	p, err := returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	pk, err := pack.Generate(pack.GenerateInput{
		Projection:       p,
		PortalMap:        pm,
		PreconditionsMet: map[string]bool{"PC01": true, "PC02": true},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Page 1: Employer Name and Salary must have correct values.
	p1 := findPage(t, pk, "P1")
	assertFieldValue(t, p1, "Employer Name", "Acme Ltd")
	assertFieldValue(t, p1, "Salary", taxengine.Kobo(500_000_000).Format())

	// Employer Name and Salary must not be Missing.
	if f, ok := findField(p1, "Employer Name"); ok && f.Missing {
		t.Error("Employer Name should not be Missing")
	}
	if f, ok := findField(p1, "Salary"); ok && f.Missing {
		t.Error("Salary should not be Missing")
	}

	// Page 2: Accommodation Type resolved, Annual Rent Paid resolved.
	p2 := findPage(t, pk, "P2")
	assertFieldValue(t, p2, "Accommodation Type", "Rent")
	assertFieldValue(t, p2, "Annual Rent Paid", taxengine.Kobo(100_000_000).Format())
	assertFieldValue(t, p2, "Landlord Details", "Mr. Adekunle, 22 Awolowo Road, Ikoyi")

	// Page 3: Pension Contributions resolved.
	p3 := findPage(t, pk, "P3")
	assertFieldValue(t, p3, "Pension Contributions", taxengine.Kobo(40_000_000).Format())
}

// ---------------------------------------------------------------------------
// T-002: Multi-employer — two EMPLOYMENT lines produce two rows each
// ---------------------------------------------------------------------------

func TestGenerate_MultiEmployer(t *testing.T) {
	pm := loadPortalMap(t)

	events := []returns.Event{
		openEvent(1),
		// Two employers — IDs are sorted so "emp-a" < "emp-b".
		{Seq: 2, Kind: returns.EvIncomeDeclared, ReturnID: "R-PACK-TEST",
			Actor:   returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: mustJSON(t, returns.IncomeLine{ID: "emp-a", Category: "EMPLOYMENT", AmountKobo: 500_000_000, Label: "Acme Ltd"})},
		{Seq: 3, Kind: returns.EvIncomeDeclared, ReturnID: "R-PACK-TEST",
			Actor:   returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: mustJSON(t, returns.IncomeLine{ID: "emp-b", Category: "EMPLOYMENT", AmountKobo: 300_000_000, Label: "TechCorp"})},
		{Seq: 4, Kind: returns.EvAssessmentComputed, ReturnID: "R-PACK-TEST",
			Actor:   returns.Actor{Kind: returns.ActorSystem},
			Payload: fakeAssessmentPayload(t, true)},
		{Seq: 5, Kind: returns.EvValidationRun, ReturnID: "R-PACK-TEST",
			Payload: []byte(`{"blocking":[],"warnings":[]}`)},
		{Seq: 6, Kind: returns.EvReturnSigned, ReturnID: "R-PACK-TEST", OccurredAt: t0},
	}

	p, err := returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	pk, err := pack.Generate(pack.GenerateInput{Projection: p, PortalMap: pm})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	p1 := findPage(t, pk, "P1")

	nameCount := countFieldsByLabel(p1, "Employer Name")
	if nameCount != 2 {
		t.Errorf("Employer Name rows: want 2, got %d", nameCount)
	}

	salaryCount := countFieldsByLabel(p1, "Salary")
	if salaryCount != 2 {
		t.Errorf("Salary rows: want 2, got %d", salaryCount)
	}

	// Verify both employer names appear (order determined by ID sort: emp-a first).
	fields := allFields(p1)
	nameValues := []string{}
	for _, f := range fields {
		if f.PortalLabel == "Employer Name" {
			nameValues = append(nameValues, f.Value)
		}
	}
	if len(nameValues) < 2 || nameValues[0] != "Acme Ltd" || nameValues[1] != "TechCorp" {
		t.Errorf("Employer Name values: want [Acme Ltd TechCorp], got %v", nameValues)
	}
}

// ---------------------------------------------------------------------------
// T-003: Unmapped income — DIGITAL_ASSET_GAIN flagged; GRATUITY not flagged
// ---------------------------------------------------------------------------

func TestGenerate_UnmappedIncome(t *testing.T) {
	pm := loadPortalMap(t)

	events := []returns.Event{
		openEvent(1),
		// DIGITAL_ASSET_GAIN — unmapped on Page 1
		{Seq: 2, Kind: returns.EvIncomeDeclared, ReturnID: "R-PACK-TEST",
			Actor:   returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: mustJSON(t, returns.IncomeLine{ID: "dag1", Category: "DIGITAL_ASSET_GAIN", AmountKobo: 100_000_000})},
		// GRATUITY — mapped on Page 3 (income.GRATUITY); must NOT appear in UnmappedIncome.
		{Seq: 3, Kind: returns.EvIncomeDeclared, ReturnID: "R-PACK-TEST",
			Actor:   returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: mustJSON(t, returns.IncomeLine{ID: "grat1", Category: "GRATUITY", AmountKobo: 50_000_000})},
		// Employment income so DeclaredByTaxpayer stays true.
		{Seq: 4, Kind: returns.EvIncomeDeclared, ReturnID: "R-PACK-TEST",
			Actor:   returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: mustJSON(t, returns.IncomeLine{ID: "emp1", Category: "EMPLOYMENT", AmountKobo: 500_000_000, Label: "Acme Ltd"})},
		{Seq: 5, Kind: returns.EvAssessmentComputed, ReturnID: "R-PACK-TEST",
			Actor:   returns.Actor{Kind: returns.ActorSystem},
			Payload: fakeAssessmentPayload(t, true)},
		{Seq: 6, Kind: returns.EvValidationRun, ReturnID: "R-PACK-TEST",
			Payload: []byte(`{"blocking":[],"warnings":[]}`)},
		{Seq: 7, Kind: returns.EvReturnSigned, ReturnID: "R-PACK-TEST", OccurredAt: t0},
	}

	p, err := returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	pk, err := pack.Generate(pack.GenerateInput{Projection: p, PortalMap: pm})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(pk.UnmappedIncome) != 1 {
		t.Fatalf("UnmappedIncome count: want 1, got %d (items: %v)", len(pk.UnmappedIncome), pk.UnmappedIncome)
	}
	if pk.UnmappedIncome[0].Category != "DIGITAL_ASSET_GAIN" {
		t.Errorf("UnmappedIncome[0].Category: want DIGITAL_ASSET_GAIN, got %s", pk.UnmappedIncome[0].Category)
	}
	if pk.UnmappedIncome[0].Amount != taxengine.Kobo(100_000_000) {
		t.Errorf("UnmappedIncome[0].Amount: want %s, got %s",
			taxengine.Kobo(100_000_000).Format(), pk.UnmappedIncome[0].Amount.Format())
	}

	// GRATUITY must NOT appear in UnmappedIncome — it is mapped on Page 3.
	for _, ui := range pk.UnmappedIncome {
		if ui.Category == "GRATUITY" {
			t.Error("GRATUITY must not appear in UnmappedIncome — it is mapped on Page 3 (income.GRATUITY)")
		}
	}
}

// ---------------------------------------------------------------------------
// T-004: Missing accommodation — Page 2 fields are Missing when not declared
// ---------------------------------------------------------------------------

func TestGenerate_MissingAccommodation(t *testing.T) {
	pm := loadPortalMap(t)

	// No EvAccommodationDeclared emitted — AccommodationType is empty.
	p := buildSignedProjection(t)
	if p.AccommodationType != "" {
		t.Fatalf("expected empty AccommodationType, got %q", p.AccommodationType)
	}

	pk, err := pack.Generate(pack.GenerateInput{Projection: p, PortalMap: pm})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	p2 := findPage(t, pk, "P2")

	// These fields are not conditional — they must appear but be Missing.
	assertFieldMissing(t, p2, "Accommodation Type")
	assertFieldMissing(t, p2, "Ownership Type")

	// "Annual Rent Paid" and "Landlord Details" are conditional on
	// accommodation.type == Rent. With AccommodationType = "", the conditional
	// evaluates to false, so these fields are absent entirely (not Missing).
	assertFieldAbsent(t, p2, "Annual Rent Paid")
	assertFieldAbsent(t, p2, "Landlord Details")
}

// ---------------------------------------------------------------------------
// T-005: Too-early state — DRAFT and VALIDATED are rejected
// ---------------------------------------------------------------------------

func TestGenerate_TooEarlyState(t *testing.T) {
	pm := loadPortalMap(t)

	// DRAFT
	pDraft, err := returns.Replay([]returns.Event{openEvent(1)})
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if _, err := pack.Generate(pack.GenerateInput{Projection: pDraft, PortalMap: pm}); err == nil {
		t.Error("expected error for DRAFT state, got nil")
	}

	// VALIDATED (assessment computed + validation run, but not yet signed)
	events := []returns.Event{
		openEvent(1),
		{Seq: 2, Kind: returns.EvAssessmentComputed, ReturnID: "R-PACK-TEST",
			Payload: fakeAssessmentPayload(t, true)},
		{Seq: 3, Kind: returns.EvValidationRun, ReturnID: "R-PACK-TEST",
			Payload: []byte(`{"blocking":[],"warnings":[]}`)},
	}
	pValidated, err := returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if pValidated.State != returns.StateValidated {
		t.Fatalf("expected VALIDATED, got %s", pValidated.State)
	}
	if _, err := pack.Generate(pack.GenerateInput{Projection: pValidated, PortalMap: pm}); err == nil {
		t.Error("expected error for VALIDATED state, got nil")
	}
}

// ---------------------------------------------------------------------------
// T-006: SIGNED state is accepted (pack preview before ISSUE_PACK fires)
// ---------------------------------------------------------------------------

func TestGenerate_SignedStateAccepted(t *testing.T) {
	pm := loadPortalMap(t)
	p := buildSignedProjection(t)

	if _, err := pack.Generate(pack.GenerateInput{Projection: p, PortalMap: pm}); err != nil {
		t.Errorf("expected success for SIGNED state, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// T-007: Preconditions — BlocksPack and BlocksTCC reported correctly
// ---------------------------------------------------------------------------

func TestGenerate_PreconditionsMet(t *testing.T) {
	pm := loadPortalMap(t)
	p := buildSignedProjection(t)

	pk, err := pack.Generate(pack.GenerateInput{
		Projection: p,
		PortalMap:  pm,
		PreconditionsMet: map[string]bool{
			"PC01": true,  // Payer ID — blocks pack, is met
			"PC02": true,  // Profile complete — met
			"PC03": false, // Passport photograph — blocks TCC, not met
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Pack must still be generated regardless of precondition state.
	if len(pk.Pages) == 0 {
		t.Error("pack must have pages even with unmet preconditions")
	}

	pc01, ok := findPrecondition(pk, "PC01")
	if !ok {
		t.Fatal("PC01 not in pack Preconditions")
	}
	if !pc01.Met {
		t.Error("PC01: want Met=true")
	}
	if !pc01.BlocksPack {
		t.Error("PC01: want BlocksPack=true")
	}

	pc03, ok := findPrecondition(pk, "PC03")
	if !ok {
		t.Fatal("PC03 not in pack Preconditions")
	}
	if pc03.Met {
		t.Error("PC03: want Met=false")
	}
	if !pc03.BlocksTCC {
		t.Error("PC03: want BlocksTCC=true")
	}
	if pc03.BlocksPack {
		t.Error("PC03: must NOT block pack (only the TCC)")
	}
}

// ---------------------------------------------------------------------------
// T-008: PC01 not met — reported, pack still generated
// ---------------------------------------------------------------------------

func TestGenerate_PC01NotMet(t *testing.T) {
	pm := loadPortalMap(t)
	p := buildSignedProjection(t)

	// PC01 is absent from the map — defaults to false (not met).
	pk, err := pack.Generate(pack.GenerateInput{
		Projection: p,
		PortalMap:  pm,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	pc01, ok := findPrecondition(pk, "PC01")
	if !ok {
		t.Fatal("PC01 not in pack Preconditions")
	}
	if pc01.Met {
		t.Error("PC01: want Met=false (caller did not confirm it)")
	}
	if !pc01.BlocksPack {
		t.Error("PC01: want BlocksPack=true")
	}
	// Pack must still be generated — Generate reports, not aborts.
	if len(pk.Pages) == 0 {
		t.Error("pack must still be generated when PC01 is not met")
	}
}

// ---------------------------------------------------------------------------
// T-009: Assessment trace is populated from the snapshot
// ---------------------------------------------------------------------------

func TestGenerate_AssessmentTrace(t *testing.T) {
	pm := loadPortalMap(t)

	// fakeAssessmentPayload includes two trace steps.
	p := buildSignedProjection(t)

	pk, err := pack.Generate(pack.GenerateInput{Projection: p, PortalMap: pm})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(pk.AssessmentTrace) == 0 {
		t.Error("AssessmentTrace should be populated from AssessmentSnapshot.Trace")
	}
	// Verify the first step came through correctly.
	if pk.AssessmentTrace[0].Op != "ADD" {
		t.Errorf("AssessmentTrace[0].Op: want ADD, got %s", pk.AssessmentTrace[0].Op)
	}
}

// ---------------------------------------------------------------------------
// T-010: Conditional field — Self-owned hides Annual Rent Paid and Landlord
// ---------------------------------------------------------------------------

func TestGenerate_ConditionalField_SelfOwned(t *testing.T) {
	pm := loadPortalMap(t)

	events := []returns.Event{
		openEvent(1),
		{Seq: 2, Kind: returns.EvIncomeDeclared, ReturnID: "R-PACK-TEST",
			Actor:   returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: mustJSON(t, returns.IncomeLine{ID: "emp1", Category: "EMPLOYMENT", AmountKobo: 500_000_000, Label: "Acme Ltd"})},
		// Self-owned accommodation — conditional fields must be absent.
		{Seq: 3, Kind: returns.EvAccommodationDeclared, ReturnID: "R-PACK-TEST",
			Actor: returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: mustJSON(t, map[string]any{
				"type": "Self-owned", "ownership_type": "Owner",
			})},
		{Seq: 4, Kind: returns.EvAssessmentComputed, ReturnID: "R-PACK-TEST",
			Actor:   returns.Actor{Kind: returns.ActorSystem},
			Payload: fakeAssessmentPayload(t, true)},
		{Seq: 5, Kind: returns.EvValidationRun, ReturnID: "R-PACK-TEST",
			Payload: []byte(`{"blocking":[],"warnings":[]}`)},
		{Seq: 6, Kind: returns.EvReturnSigned, ReturnID: "R-PACK-TEST", OccurredAt: t0},
	}

	p, err := returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	pk, err := pack.Generate(pack.GenerateInput{Projection: p, PortalMap: pm})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	p2 := findPage(t, pk, "P2")

	// Non-conditional fields must be present with their values.
	assertFieldValue(t, p2, "Accommodation Type", "Self-owned")
	assertFieldValue(t, p2, "Ownership Type", "Owner")

	// Conditional fields must be absent entirely (not just Missing).
	assertFieldAbsent(t, p2, "Annual Rent Paid")
	assertFieldAbsent(t, p2, "Landlord Details")
}
