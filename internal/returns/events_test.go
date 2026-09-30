package returns_test

import (
	"encoding/json"
	"testing"
	"time"

	"taxsoftware/internal/returns"
)

func seqEvents(t *testing.T, es []returns.Event) []returns.Event {
	t.Helper()
	for i := range es {
		es[i].Seq = i + 1
	}
	return es
}

func baseOpenEvent() returns.Event {
	occurred := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	payload, _ := json.Marshal(map[string]any{
		"taxpayer_id":        "tp-1",
		"year_of_assessment": 2026,
		"basis_period_start": "2026-01-01T00:00:00Z",
		"basis_period_end":   "2026-12-31T00:00:00Z",
		"jurisdiction":       "NG",
	})
	return returns.Event{
		ReturnID:   "ret-1",
		Kind:       returns.EvReturnOpened,
		Payload:    payload,
		Actor:      returns.Actor{Kind: returns.ActorTaxpayer, ID: "tp-1"},
		OccurredAt: occurred,
	}
}

func TestFoldUnknownEventKind(t *testing.T) {
	e1 := baseOpenEvent()
	e1.Seq = 1

	e2 := returns.Event{
		Seq:      2,
		ReturnID: "ret-1",
		Kind:     returns.EventKind("ThisKindDoesNotExist"),
		Payload:  json.RawMessage(`{"foo": "bar"}`),
		Actor:    returns.Actor{Kind: returns.ActorSystem, ID: "sys"},
	}

	p := &returns.Projection{}
	if err := p.Fold(e1); err != nil {
		t.Fatalf("fold open: %v", err)
	}
	if err := p.Fold(e2); err != nil {
		t.Fatalf("fold unknown: %v", err)
	}
	if got, want := p.Version, 2; got != want {
		t.Errorf("version after unknown event: got %d, want %d", got, want)
	}
	if p.State != returns.StateDraft {
		t.Errorf("state should remain DRAFT after unknown event, got %s", p.State)
	}
}

func TestFoldOutOfOrderFails(t *testing.T) {
	e1 := baseOpenEvent()
	e1.Seq = 2

	p := &returns.Projection{}
	err := p.Fold(e1)
	if err == nil {
		t.Fatal("expected out-of-order event to fail")
	}
}

func TestFoldIsPureReplayable(t *testing.T) {
	events := []returns.Event{baseOpenEvent()}

	occurred := time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC)
	payload, _ := json.Marshal(map[string]any{
		"id":          "inc-1",
		"category":    "EMPLOYMENT",
		"amount_kobo": 900000000,
		"label":       "Salary",
		"source_ref":  "acme-ltd",
	})
	events = append(events, returns.Event{
		Seq:        2,
		ReturnID:   "ret-1",
		Kind:       returns.EvIncomeDeclared,
		Payload:    payload,
		Actor:      returns.Actor{Kind: returns.ActorTaxpayer, ID: "tp-1"},
		OccurredAt: occurred,
	})

	p1, err := returns.Replay(seqEvents(t, events))
	if err != nil {
		t.Fatalf("Replay #1: %v", err)
	}
	p2, err := returns.Replay(seqEvents(t, events))
	if err != nil {
		t.Fatalf("Replay #2: %v", err)
	}

	if p1.Version != p2.Version {
		t.Errorf("version mismatch: %d vs %d", p1.Version, p2.Version)
	}
	if p1.State != p2.State {
		t.Errorf("state mismatch: %s vs %s", p1.State, p2.State)
	}
	if p1.Income["inc-1"].AmountKobo != p2.Income["inc-1"].AmountKobo {
		t.Errorf("income amount mismatch")
	}
	if p1.DeclaredByTaxpayer != p2.DeclaredByTaxpayer {
		t.Errorf("DeclaredByTaxpayer mismatch")
	}
}

func TestFoldIncomeDeclaredMarksStale(t *testing.T) {
	p := &returns.Projection{}
	e := baseOpenEvent()
	e.Seq = 1
	if err := p.Fold(e); err != nil {
		t.Fatalf("open: %v", err)
	}

	compTime := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	snapPayload, _ := json.Marshal(returns.AssessmentSnapshot{
		FactsHash:      "abc",
		RuleSetVersion: "v1",
		ComputedAt:     compTime,
	})
	assessEvent := returns.Event{
		Seq:        2,
		ReturnID:   "ret-1",
		Kind:       returns.EvAssessmentComputed,
		Payload:    snapPayload,
		Actor:      returns.Actor{Kind: returns.ActorSystem, ID: "sys"},
		OccurredAt: compTime,
	}
	if err := p.Fold(assessEvent); err != nil {
		t.Fatalf("assessment: %v", err)
	}
	if p.AssessmentStale {
		t.Fatal("assessment should not be stale right after compute")
	}

	occurred := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	incPayload, _ := json.Marshal(map[string]any{
		"id":          "inc-1",
		"category":    "EMPLOYMENT",
		"amount_kobo": 100000,
		"label":       "Salary",
	})
	incEvent := returns.Event{
		Seq:        3,
		ReturnID:   "ret-1",
		Kind:       returns.EvIncomeDeclared,
		Payload:    incPayload,
		Actor:      returns.Actor{Kind: returns.ActorTaxpayer, ID: "tp-1"},
		OccurredAt: occurred,
	}
	if err := p.Fold(incEvent); err != nil {
		t.Fatalf("income: %v", err)
	}
	if !p.AssessmentStale {
		t.Error("assessment should be stale after income added")
	}
}

func TestFoldEmployerIncomeRejectedBySignGuard(t *testing.T) {
	p := &returns.Projection{}
	e := baseOpenEvent()
	e.Seq = 1
	if err := p.Fold(e); err != nil {
		t.Fatalf("open: %v", err)
	}

	occurred := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	incPayload, _ := json.Marshal(map[string]any{
		"id":          "inc-1",
		"category":    "EMPLOYMENT",
		"amount_kobo": 100000,
		"label":       "Salary",
	})
	// EMPLOYER actor, not TAXPAYER
	incEvent := returns.Event{
		Seq:        2,
		ReturnID:   "ret-1",
		Kind:       returns.EvIncomeDeclared,
		Payload:    incPayload,
		Actor:      returns.Actor{Kind: returns.ActorEmployer, ID: "acme"},
		OccurredAt: occurred,
	}
	if err := p.Fold(incEvent); err != nil {
		t.Fatalf("income: %v", err)
	}
	if p.DeclaredByTaxpayer {
		t.Error("DeclaredByTaxpayer should be false when EMPLOYER actor declares income")
	}
}

func TestFoldPayrollProposalDoesNotMarkStale(t *testing.T) {
	p := &returns.Projection{}
	e := baseOpenEvent()
	e.Seq = 1
	if err := p.Fold(e); err != nil {
		t.Fatalf("open: %v", err)
	}

	compTime := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	snapPayload, _ := json.Marshal(returns.AssessmentSnapshot{
		FactsHash:      "abc",
		RuleSetVersion: "v1",
		ComputedAt:     compTime,
	})
	assessEvent := returns.Event{
		Seq:        2,
		ReturnID:   "ret-1",
		Kind:       returns.EvAssessmentComputed,
		Payload:    snapPayload,
		Actor:      returns.Actor{Kind: returns.ActorSystem, ID: "sys"},
		OccurredAt: compTime,
	}
	if err := p.Fold(assessEvent); err != nil {
		t.Fatalf("assessment: %v", err)
	}

	occurred := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	payrollPayload, _ := json.Marshal(map[string]any{
		"record_id":   "pr-1",
		"employer_id": "acme",
		"gross_kobo":  900000000,
		"paye_kobo":   160000000,
	})
	payrollEvent := returns.Event{
		Seq:        3,
		ReturnID:   "ret-1",
		Kind:       returns.EvPayrollProposed,
		Payload:    payrollPayload,
		Actor:      returns.Actor{Kind: returns.ActorEmployer, ID: "acme"},
		OccurredAt: occurred,
	}
	if err := p.Fold(payrollEvent); err != nil {
		t.Fatalf("payroll proposed: %v", err)
	}
	if p.AssessmentStale {
		t.Error("PayrollProposed should NOT mark assessment stale")
	}
	if len(p.Income) != 0 {
		t.Error("PayrollProposed should NOT add income lines")
	}
}

func TestFold_AccommodationDeclared(t *testing.T) {
	p := &returns.Projection{}
	e := baseOpenEvent()
	e.Seq = 1
	if err := p.Fold(e); err != nil {
		t.Fatalf("open: %v", err)
	}

	acPayload, _ := json.Marshal(map[string]any{
		"type":             "Rent",
		"ownership_type":   "Tenant",
		"landlord_name":    "Mrs. Coker",
		"landlord_address": "14 Bode Thomas Street, Lagos",
	})
	acEvent := returns.Event{
		Seq:        2,
		ReturnID:   "ret-1",
		Kind:       returns.EvAccommodationDeclared,
		Payload:    acPayload,
		Actor:      returns.Actor{Kind: returns.ActorTaxpayer, ID: "tp-1"},
		OccurredAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := p.Fold(acEvent); err != nil {
		t.Fatalf("accommodation: %v", err)
	}

	if p.AccommodationType != "Rent" {
		t.Errorf("AccommodationType: want %q, got %q", "Rent", p.AccommodationType)
	}
	if p.OwnershipType != "Tenant" {
		t.Errorf("OwnershipType: want %q, got %q", "Tenant", p.OwnershipType)
	}
	if p.LandlordName != "Mrs. Coker" {
		t.Errorf("LandlordName: want %q, got %q", "Mrs. Coker", p.LandlordName)
	}
	if p.LandlordAddress != "14 Bode Thomas Street, Lagos" {
		t.Errorf("LandlordAddress: want %q, got %q", "14 Bode Thomas Street, Lagos", p.LandlordAddress)
	}
	if p.Version != 2 {
		t.Errorf("Version: want 2, got %d", p.Version)
	}
}

func TestFold_AccommodationDoesNotMarkStale(t *testing.T) {
	p := &returns.Projection{}
	e := baseOpenEvent()
	e.Seq = 1
	if err := p.Fold(e); err != nil {
		t.Fatalf("open: %v", err)
	}

	// Compute an assessment first.
	compTime := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	snapPayload, _ := json.Marshal(returns.AssessmentSnapshot{
		FactsHash:      "abc",
		RuleSetVersion: "v1",
		ComputedAt:     compTime,
	})
	if err := p.Fold(returns.Event{
		Seq: 2, Kind: returns.EvAssessmentComputed, Payload: snapPayload,
		Actor: returns.Actor{Kind: returns.ActorSystem},
	}); err != nil {
		t.Fatalf("assessment: %v", err)
	}
	if p.AssessmentStale {
		t.Fatal("assessment should not be stale right after compute")
	}

	// Declare accommodation — must NOT mark assessment stale.
	acPayload, _ := json.Marshal(map[string]any{
		"type": "Self-owned", "ownership_type": "Owner",
	})
	if err := p.Fold(returns.Event{
		Seq: 3, Kind: returns.EvAccommodationDeclared, Payload: acPayload,
		Actor: returns.Actor{Kind: returns.ActorTaxpayer},
	}); err != nil {
		t.Fatalf("accommodation: %v", err)
	}

	if p.AssessmentStale {
		t.Error("EvAccommodationDeclared must NOT mark assessment stale")
	}
	if p.AccommodationType != "Self-owned" {
		t.Errorf("AccommodationType: want %q, got %q", "Self-owned", p.AccommodationType)
	}
}

func TestFoldValidationRunSetsState(t *testing.T) {
	p := &returns.Projection{}
	e := baseOpenEvent()
	e.Seq = 1
	if err := p.Fold(e); err != nil {
		t.Fatalf("open: %v", err)
	}

	occurred := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	valPayload, _ := json.Marshal(map[string]any{
		"blocking": []string{},
		"warnings": []string{"W101 rent without evidence"},
	})
	valEvent := returns.Event{
		Seq:        2,
		ReturnID:   "ret-1",
		Kind:       returns.EvValidationRun,
		Payload:    valPayload,
		Actor:      returns.Actor{Kind: returns.ActorSystem, ID: "sys"},
		OccurredAt: occurred,
	}
	if err := p.Fold(valEvent); err != nil {
		t.Fatalf("validation: %v", err)
	}
	if p.State != returns.StateValidated {
		t.Errorf("state after validation with no blocking errors: got %s, want VALIDATED", p.State)
	}
}
