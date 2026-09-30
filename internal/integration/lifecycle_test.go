package integration_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"taxsoftware/internal/bridge"
	"taxsoftware/internal/golden"
	"taxsoftware/internal/returns"
	"taxsoftware/internal/validate"
	"taxsoftware/taxengine"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var fixedTime = time.Date(2027, 4, 1, 10, 0, 0, 0, time.UTC)

func loadRuleSet(t *testing.T) *taxengine.RuleSet {
	t.Helper()
	rs, err := golden.LoadRuleSet()
	if err != nil {
		t.Fatal(err)
	}
	return rs
}

func payload(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func applyOrFatal(t *testing.T, cur returns.State, tr returns.Transition, p *returns.Projection) returns.State {
	t.Helper()
	next, err := returns.Apply(cur, tr, p)
	if err != nil {
		t.Fatalf("Apply(%s, %s) failed: %v", cur, tr, err)
	}
	return next
}

func applyMustFail(t *testing.T, cur returns.State, tr returns.Transition, p *returns.Projection, wantSubstr string) {
	t.Helper()
	_, err := returns.Apply(cur, tr, p)
	if err == nil {
		t.Fatalf("expected Apply(%s, %s) to fail containing %q, but it succeeded", cur, tr, wantSubstr)
	}
	if wantSubstr != "" && !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("expected error containing %q, got %q", wantSubstr, err.Error())
	}
}

func openReturn(seq int) returns.Event {
	return returns.Event{
		Seq:      seq,
		ReturnID: "R-INT-TEST",
		Kind:     returns.EvReturnOpened,
		Payload: []byte(`{
			"taxpayer_id":        "TIN-INT-001",
			"year_of_assessment": 2026,
			"jurisdiction":       "NG"
		}`),
		Actor:      returns.Actor{Kind: returns.ActorTaxpayer, ID: "TIN-INT-001"},
		OccurredAt: fixedTime,
	}
}

func assessmentSnapshot(t *testing.T, a taxengine.Assessment, tr taxengine.Trace, published bool) json.RawMessage {
	t.Helper()
	traceJSON, err := json.Marshal(tr.Steps)
	if err != nil {
		t.Fatalf("marshal trace: %v", err)
	}
	return payload(t, returns.AssessmentSnapshot{
		FactsHash:            a.FactsHash,
		RuleSetVersion:       a.RuleSetVersion,
		RuleSetDigest:        a.RuleSetDigest,
		RuleSetPublished:     published,
		ChargeableIncomeKobo: int64(a.ChargeableIncomeKobo),
		TaxPayableKobo:       int64(a.TaxPayableKobo),
		TotalCreditsKobo:     int64(a.TotalCreditsKobo),
		BalanceKobo:          int64(a.BalanceKobo),
		ComputedAt:           fixedTime,
		Trace:                json.RawMessage(traceJSON),
	})
}

// ---------------------------------------------------------------------------
// S-001: Happy path — draft to cleared
// ---------------------------------------------------------------------------
func TestS001_HappyPath_DraftToCleared(t *testing.T) {
	rs := loadRuleSet(t)

	// 1. Open return, declare income, declare base, claim relief
	events := []returns.Event{
		openReturn(1),
		{Seq: 2, Kind: returns.EvIncomeDeclared, Actor: returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: payload(t, returns.IncomeLine{ID: "sal", Category: "EMPLOYMENT", AmountKobo: 500_000_000, Label: "Salary"})},
		{Seq: 3, Kind: returns.EvBaseDeclared, Payload: payload(t, struct {
			Name       string `json:"name"`
			AmountKobo int64  `json:"amount_kobo"`
		}{"annual_rent_paid", 100_000_000})},
		{Seq: 4, Kind: returns.EvReliefClaimed, Payload: payload(t, returns.ReliefLine{Code: "RENT_RELIEF"})},
	}

	p, err := returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}
	if p.State != returns.StateDraft {
		t.Fatalf("expected DRAFT, got %s", p.State)
	}

	// 2. Bridge → Compute
	facts := bridge.ProjectionToFacts(p)
	assessment, trace, err := taxengine.Compute(facts, rs)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	// 3. Fold AssessmentComputed + ValidationRun (no blocking errors)
	published := rs.Status == taxengine.StatusPublished
	events = append(events,
		returns.Event{Seq: 5, Kind: returns.EvAssessmentComputed, Actor: returns.Actor{Kind: returns.ActorSystem},
			Payload: assessmentSnapshot(t, assessment, trace, published)},
		returns.Event{Seq: 6, Kind: returns.EvValidationRun, Actor: returns.Actor{Kind: returns.ActorSystem},
			Payload: []byte(`{"blocking":[],"warnings":[]}`)},
	)
	p, err = returns.Replay(events)
	if err != nil {
		t.Fatal(err)
	}
	if p.State != returns.StateValidated {
		t.Fatalf("expected VALIDATED, got %s", p.State)
	}

	// 4. Walk transitions: SIGN → ISSUE_PACK → ATTEST → UPLOAD_ACK → RECORD_ASSESSMENT → RECORD_PAYMENT → CLEAR
	// Note: SIGN guard requires RuleSetPublished=true. ng-2026 is DRAFT so published=false.
	// This proves the SIGN guard correctly blocks an unpublished rule set.
	if !published {
		applyMustFail(t, p.State, returns.TrSign, p, "unpublished rule set")
		// For the happy-path continuation, force-publish for test completeness.
		p.RuleSetPublished = true
	}

	steps := []struct {
		from returns.State
		tr   returns.Transition
		ev   returns.Event
		want returns.State
	}{
		{returns.StateValidated, returns.TrSign, returns.Event{Seq: 7, Kind: returns.EvReturnSigned, OccurredAt: fixedTime}, returns.StateSigned},
		{returns.StateSigned, returns.TrIssuePack, returns.Event{Seq: 8, Kind: returns.EvPackIssued, Payload: []byte(`{"portal_map_version":"1.0"}`), OccurredAt: fixedTime}, returns.StatePackIssued},
		{returns.StatePackIssued, returns.TrAttest, returns.Event{Seq: 9, Kind: returns.EvFilingAttested, OccurredAt: fixedTime}, returns.StateSelfAttested},
		{returns.StateSelfAttested, returns.TrUploadAck, returns.Event{Seq: 10, Kind: returns.EvAcknowledgementUploaded, OccurredAt: fixedTime}, returns.StateAckUploaded},
		{returns.StateAckUploaded, returns.TrRecordAssessment, returns.Event{Seq: 11, Kind: returns.EvAssessmentServed, Payload: payload(t, struct {
			ServedAt   time.Time `json:"served_at"`
			WindowDays int       `json:"objection_window_days"`
		}{fixedTime, 30})}, returns.StateAssessed},
		{returns.StateAssessed, returns.TrRecordPayment, returns.Event{Seq: 12, Kind: returns.EvPaymentRecorded, OccurredAt: fixedTime}, returns.StatePaid},
		{returns.StatePaid, returns.TrClear, returns.Event{Seq: 13, Kind: returns.EvReturnCleared, OccurredAt: fixedTime}, returns.StateCleared},
	}

	for _, s := range steps {
		cur := applyOrFatal(t, s.from, s.tr, p)
		if cur != s.want {
			t.Fatalf("after %s: expected %s, got %s", s.tr, s.want, cur)
		}
		events = append(events, s.ev)
		p, err = returns.Replay(events)
		if err != nil {
			t.Fatalf("Replay after %s: %v", s.tr, err)
		}
	}

	if p.State != returns.StateCleared {
		t.Fatalf("final state: expected CLEARED, got %s", p.State)
	}
	if !p.State.IsTerminal() {
		t.Fatal("CLEARED should be terminal")
	}
}

// ---------------------------------------------------------------------------
// S-002: SIGN guard — stale assessment
// ---------------------------------------------------------------------------
func TestS002_SignGuard_StaleAssessment(t *testing.T) {
	rs := loadRuleSet(t)

	events := []returns.Event{
		openReturn(1),
		{Seq: 2, Kind: returns.EvIncomeDeclared, Actor: returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: payload(t, returns.IncomeLine{ID: "sal", Category: "EMPLOYMENT", AmountKobo: 500_000_000})},
	}
	p, _ := returns.Replay(events)
	facts := bridge.ProjectionToFacts(p)
	a, trA, _ := taxengine.Compute(facts, rs)

	events = append(events,
		returns.Event{Seq: 3, Kind: returns.EvAssessmentComputed, Payload: assessmentSnapshot(t, a, trA, true)},
		returns.Event{Seq: 4, Kind: returns.EvValidationRun, Payload: []byte(`{"blocking":[],"warnings":[]}`)},
	)
	p, _ = returns.Replay(events)

	// Revise income → stale
	events = append(events, returns.Event{
		Seq: 5, Kind: returns.EvIncomeRevised, Actor: returns.Actor{Kind: returns.ActorTaxpayer},
		Payload: payload(t, returns.IncomeLine{ID: "sal", Category: "EMPLOYMENT", AmountKobo: 600_000_000}),
	})
	p, _ = returns.Replay(events)

	if !p.AssessmentStale {
		t.Fatal("expected stale assessment after income revision")
	}
	applyMustFail(t, returns.StateValidated, returns.TrSign, p, "facts changed since the assessment was computed")

	// Recompute → un-stale → SIGN succeeds
	facts = bridge.ProjectionToFacts(p)
	a2, trA2, _ := taxengine.Compute(facts, rs)
	events = append(events,
		returns.Event{Seq: 6, Kind: returns.EvAssessmentComputed, Payload: assessmentSnapshot(t, a2, trA2, true)},
	)
	p, _ = returns.Replay(events)

	if p.AssessmentStale {
		t.Fatal("expected non-stale after recompute")
	}
	applyOrFatal(t, returns.StateValidated, returns.TrSign, p)
}

// ---------------------------------------------------------------------------
// S-003: SIGN guard — unpublished rule set
// ---------------------------------------------------------------------------
func TestS003_SignGuard_UnpublishedRuleSet(t *testing.T) {
	events := []returns.Event{
		openReturn(1),
		{Seq: 2, Kind: returns.EvAssessmentComputed,
			Payload: payload(t, returns.AssessmentSnapshot{RuleSetPublished: false})},
		{Seq: 3, Kind: returns.EvValidationRun, Payload: []byte(`{"blocking":[],"warnings":[]}`)},
	}
	p, _ := returns.Replay(events)

	if p.RuleSetPublished {
		t.Fatal("expected RuleSetPublished=false")
	}
	applyMustFail(t, returns.StateValidated, returns.TrSign, p, "unpublished rule set")

	// With RuleSetPublished=true, SIGN would pass (if other guards satisfied)
	events2 := []returns.Event{
		openReturn(1),
		{Seq: 2, Kind: returns.EvAssessmentComputed,
			Payload: payload(t, returns.AssessmentSnapshot{RuleSetPublished: true})},
		{Seq: 3, Kind: returns.EvValidationRun, Payload: []byte(`{"blocking":[],"warnings":[]}`)},
	}
	p2, _ := returns.Replay(events2)
	applyOrFatal(t, returns.StateValidated, returns.TrSign, p2)
}

// ---------------------------------------------------------------------------
// S-004: SIGN guard — DeclaredByTaxpayer false
// ---------------------------------------------------------------------------
func TestS004_SignGuard_DeclaredByTaxpayer(t *testing.T) {
	events := []returns.Event{
		openReturn(1),
		{Seq: 2, Kind: returns.EvIncomeDeclared, Actor: returns.Actor{Kind: returns.ActorEmployer, ID: "EMP-1"},
			Payload: payload(t, returns.IncomeLine{ID: "e1", Category: "EMPLOYMENT", AmountKobo: 500_000_000})},
		{Seq: 3, Kind: returns.EvAssessmentComputed,
			Payload: payload(t, returns.AssessmentSnapshot{RuleSetPublished: true})},
		{Seq: 4, Kind: returns.EvValidationRun, Payload: []byte(`{"blocking":[],"warnings":[]}`)},
	}
	p, _ := returns.Replay(events)

	if p.DeclaredByTaxpayer {
		t.Fatal("expected DeclaredByTaxpayer=false when employer declares income")
	}
	applyMustFail(t, returns.StateValidated, returns.TrSign, p, "not all income was declared by the taxpayer actor")

	// Accept payroll — income becomes taxpayer-declared
	events = append(events,
		returns.Event{Seq: 5, Kind: returns.EvPayrollAccepted, Actor: returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: payload(t, struct {
				RecordID string             `json:"record_id"`
				Income   returns.IncomeLine `json:"income"`
			}{"r1", returns.IncomeLine{ID: "e1", Category: "EMPLOYMENT", AmountKobo: 500_000_000}})},
	)
	p, _ = returns.Replay(events)
	if !p.DeclaredByTaxpayer {
		t.Fatal("expected DeclaredByTaxpayer=true after PayrollAccepted by TAXPAYER")
	}
}

// ---------------------------------------------------------------------------
// S-005: SIGN guard — blocking validation errors
// ---------------------------------------------------------------------------
func TestS005_SignGuard_BlockingValidationErrors(t *testing.T) {
	events := []returns.Event{
		openReturn(1),
		{Seq: 2, Kind: returns.EvAssessmentComputed,
			Payload: payload(t, returns.AssessmentSnapshot{RuleSetPublished: true})},
		{Seq: 3, Kind: returns.EvValidationRun,
			Payload: []byte(`{"blocking":["E005: no income"],"warnings":[]}`)},
	}
	p, _ := returns.Replay(events)

	// VALIDATE guard rejects because blocking errors remain
	applyMustFail(t, returns.StateDraft, returns.TrValidate, p, "blocking validation errors remain")

	// Clear blocking errors with a clean validation run
	events = append(events,
		returns.Event{Seq: 4, Kind: returns.EvIncomeDeclared, Actor: returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: payload(t, returns.IncomeLine{ID: "i1", Category: "EMPLOYMENT", AmountKobo: 100_000_000})},
		returns.Event{Seq: 5, Kind: returns.EvAssessmentComputed,
			Payload: payload(t, returns.AssessmentSnapshot{RuleSetPublished: true})},
		returns.Event{Seq: 6, Kind: returns.EvValidationRun,
			Payload: []byte(`{"blocking":[],"warnings":[]}`)},
	)
	p, _ = returns.Replay(events)
	applyOrFatal(t, returns.StateValidated, returns.TrSign, p)
}

// ---------------------------------------------------------------------------
// S-006: Edit after signing invalidates
// ---------------------------------------------------------------------------
func TestS006_EditAfterSigningInvalidates(t *testing.T) {
	events := []returns.Event{
		openReturn(1),
		{Seq: 2, Kind: returns.EvIncomeDeclared, Actor: returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: payload(t, returns.IncomeLine{ID: "s", Category: "EMPLOYMENT", AmountKobo: 500_000_000})},
		{Seq: 3, Kind: returns.EvAssessmentComputed,
			Payload: payload(t, returns.AssessmentSnapshot{RuleSetPublished: true})},
		{Seq: 4, Kind: returns.EvValidationRun, Payload: []byte(`{"blocking":[],"warnings":[]}`)},
		{Seq: 5, Kind: returns.EvReturnSigned, OccurredAt: fixedTime},
	}
	p, _ := returns.Replay(events)
	if p.State != returns.StateSigned {
		t.Fatalf("expected SIGNED, got %s", p.State)
	}

	// INVALIDATE from SIGNED → DRAFT
	next := applyOrFatal(t, returns.StateSigned, returns.TrInvalidate, p)
	if next != returns.StateDraft {
		t.Fatalf("expected DRAFT after INVALIDATE, got %s", next)
	}

	// Also verify from PACK_ISSUED → DRAFT
	events2 := append(events,
		returns.Event{Seq: 6, Kind: returns.EvPackIssued, Payload: []byte(`{"portal_map_version":"1.0"}`), OccurredAt: fixedTime},
	)
	p2, _ := returns.Replay(events2)
	next2 := applyOrFatal(t, returns.StatePackIssued, returns.TrInvalidate, p2)
	if next2 != returns.StateDraft {
		t.Fatalf("expected DRAFT after INVALIDATE from PACK_ISSUED, got %s", next2)
	}
}

// ---------------------------------------------------------------------------
// S-007: Payroll proposal does not touch income
// ---------------------------------------------------------------------------
func TestS007_PayrollProposalDoesNotTouchIncome(t *testing.T) {
	events := []returns.Event{
		openReturn(1),
		{Seq: 2, Kind: returns.EvPayrollProposed, Actor: returns.Actor{Kind: returns.ActorEmployer},
			Payload: payload(t, returns.PayrollProposal{RecordID: "pr1", EmployerID: "EMP-1", GrossKobo: 500_000_000, PayeKobo: 50_000_000})},
	}
	p, _ := returns.Replay(events)

	if len(p.Income) != 0 {
		t.Fatal("EvPayrollProposed must NOT create an income line")
	}
	if p.AssessmentStale {
		t.Fatal("EvPayrollProposed must NOT mark assessment stale")
	}
	if pr, ok := p.ProposedPayroll["pr1"]; !ok || pr.Status != "PROPOSED" {
		t.Fatal("expected payroll proposal to exist with status PROPOSED")
	}
}

// ---------------------------------------------------------------------------
// S-008: Objection deadline
// ---------------------------------------------------------------------------
func TestS008_ObjectionDeadline(t *testing.T) {
	served := time.Date(2027, 5, 15, 9, 0, 0, 0, time.UTC)
	events := []returns.Event{
		openReturn(1),
		{Seq: 2, Kind: returns.EvAssessmentServed, Payload: payload(t, struct {
			ServedAt   time.Time `json:"served_at"`
			WindowDays int       `json:"objection_window_days"`
		}{served, 30})},
	}
	p, _ := returns.Replay(events)

	if p.AssessmentServedAt == nil {
		t.Fatal("AssessmentServedAt should be set")
	}
	if p.ObjectionDeadline == nil {
		t.Fatal("ObjectionDeadline should be set")
	}
	wantDeadline := served.AddDate(0, 0, 30)
	if !p.ObjectionDeadline.Equal(wantDeadline) {
		t.Fatalf("expected deadline %v, got %v", wantDeadline, *p.ObjectionDeadline)
	}
	if p.State != returns.StateAssessed {
		t.Fatalf("expected ASSESSED, got %s", p.State)
	}

	// Object → OBJECTED, then Withdraw → ASSESSED roundtrip
	applyOrFatal(t, returns.StateAssessed, returns.TrObject, p)
	events = append(events, returns.Event{Seq: 3, Kind: returns.EvObjectionLodged})
	p, _ = returns.Replay(events)
	if p.State != returns.StateObjected {
		t.Fatalf("expected OBJECTED, got %s", p.State)
	}

	applyOrFatal(t, returns.StateObjected, returns.TrWithdrawObjection, p)
	events = append(events, returns.Event{Seq: 4, Kind: returns.EvObjectionWithdrawn})
	p, _ = returns.Replay(events)
	if p.State != returns.StateAssessed {
		t.Fatalf("expected ASSESSED after withdrawal, got %s", p.State)
	}
}

// ---------------------------------------------------------------------------
// S-009: Idempotency deduplication
// ---------------------------------------------------------------------------
func TestS009_Idempotency(t *testing.T) {
	events := []returns.Event{
		openReturn(1),
		{Seq: 2, Kind: returns.EvIncomeDeclared, IdempotencyKey: "dedup-key-1",
			Actor: returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: payload(t, returns.IncomeLine{ID: "inc-orig", Category: "EMPLOYMENT", AmountKobo: 500_000_000})},
		// Duplicate: same IdempotencyKey, different payload — must be ignored
		{Seq: 3, Kind: returns.EvIncomeDeclared, IdempotencyKey: "dedup-key-1",
			Actor: returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: payload(t, returns.IncomeLine{ID: "inc-dup", Category: "EMPLOYMENT", AmountKobo: 999_999_999})},
	}
	p, err := returns.Replay(events)
	if err != nil {
		t.Fatal(err)
	}

	// Version must advance even for deduplicated events
	if p.Version != 3 {
		t.Fatalf("expected version 3, got %d", p.Version)
	}
	// Original income line must exist
	if _, ok := p.Income["inc-orig"]; !ok {
		t.Fatal("original income line 'inc-orig' must exist")
	}
	// Duplicate income line must NOT exist
	if _, ok := p.Income["inc-dup"]; ok {
		t.Fatal("duplicate income line 'inc-dup' must NOT exist — idempotency failed")
	}
}

// ---------------------------------------------------------------------------
// S-010: Unknown event kind tolerance
// ---------------------------------------------------------------------------
func TestS010_UnknownEventKindTolerant(t *testing.T) {
	events := []returns.Event{
		openReturn(1),
		{Seq: 2, Kind: returns.EventKind("FutureFeatureAdded"), Payload: []byte(`{"some":"data"}`)},
	}
	p, err := returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay must not fail on unknown event kind: %v", err)
	}
	if p.Version != 2 {
		t.Fatalf("expected version 2, got %d", p.Version)
	}
	if p.State != returns.StateDraft {
		t.Fatalf("unknown event should not change state, got %s", p.State)
	}
}

// ---------------------------------------------------------------------------
// Assessment consistency invariant — bridge roundtrip
// ---------------------------------------------------------------------------
func TestAssessmentConsistency_BridgeRoundtrip(t *testing.T) {
	rs := loadRuleSet(t)

	events := []returns.Event{
		openReturn(1),
		{Seq: 2, Kind: returns.EvIncomeDeclared, Actor: returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: payload(t, returns.IncomeLine{ID: "s1", Category: "EMPLOYMENT", AmountKobo: 800_000_000})},
		{Seq: 3, Kind: returns.EvReliefClaimed,
			Payload: payload(t, returns.ReliefLine{Code: "PENSION", AmountKobo: 50_000_000})},
	}
	p, _ := returns.Replay(events)

	// Compute via bridge
	facts := bridge.ProjectionToFacts(p)
	a, tr, err := taxengine.Compute(facts, rs)
	if err != nil {
		t.Fatal(err)
	}

	// Fold the snapshot, then verify it matches a fresh Compute
	events = append(events, returns.Event{
		Seq: 4, Kind: returns.EvAssessmentComputed,
		Payload: assessmentSnapshot(t, a, tr, rs.Status == taxengine.StatusPublished),
	})
	p, _ = returns.Replay(events)

	// Fresh compute from the same projection
	facts2 := bridge.ProjectionToFacts(p)
	a2, _, err := taxengine.Compute(facts2, rs)
	if err != nil {
		t.Fatal(err)
	}

	if p.Assessment.ChargeableIncomeKobo != int64(a2.ChargeableIncomeKobo) {
		t.Errorf("ChargeableIncome: snapshot %d, fresh %d", p.Assessment.ChargeableIncomeKobo, a2.ChargeableIncomeKobo)
	}
	if p.Assessment.TaxPayableKobo != int64(a2.TaxPayableKobo) {
		t.Errorf("TaxPayable: snapshot %d, fresh %d", p.Assessment.TaxPayableKobo, a2.TaxPayableKobo)
	}
	if p.Assessment.BalanceKobo != int64(a2.BalanceKobo) {
		t.Errorf("Balance: snapshot %d, fresh %d", p.Assessment.BalanceKobo, a2.BalanceKobo)
	}
}

// ---------------------------------------------------------------------------
// Validation consistency — bridge produces correct result
// ---------------------------------------------------------------------------
func TestValidationConsistency_Bridge(t *testing.T) {
	rs := loadRuleSet(t)

	events := []returns.Event{
		openReturn(1),
		{Seq: 2, Kind: returns.EvIncomeDeclared, Actor: returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: payload(t, returns.IncomeLine{ID: "s1", Category: "EMPLOYMENT", AmountKobo: 500_000_000})},
	}
	p, _ := returns.Replay(events)

	ctx := validate.FilingContext{
		Facts:              bridge.ProjectionToFacts(p),
		TaxID:              "TIN-INT-001",
		TaxIDVerified:      true,
		NIN:                "11111111111",
		DeclaredByTaxpayer: p.DeclaredByTaxpayer,
		RelevantAuthority:  "LIRS",
		YearsOnFile:        3,
	}
	res, err := validate.Validate(ctx, rs)
	if err != nil {
		t.Fatal(err)
	}

	// With a DRAFT rule set we expect E008 (unpublished)
	foundE008 := false
	for _, iss := range res.Issues {
		if iss.Code == "E008" {
			foundE008 = true
		}
	}
	if rs.Status != taxengine.StatusPublished && !foundE008 {
		t.Error("expected E008 (rule set not published) but it was missing")
	}

	t.Logf("Validation produced %d issues", len(res.Issues))
	for _, iss := range res.Issues {
		t.Logf("  %s [%s] %s", iss.Code, iss.Severity, iss.Message)
	}
}
