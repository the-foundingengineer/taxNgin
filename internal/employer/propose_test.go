package employer_test

import (
	"encoding/json"
	"testing"
	"time"

	"taxsoftware/internal/employer"
	"taxsoftware/internal/returns"
	"taxsoftware/taxengine"
)

var t0 = time.Date(2027, 4, 1, 9, 0, 0, 0, time.UTC)

func openEvent(seq int) returns.Event {
	payload, _ := json.Marshal(map[string]any{
		"taxpayer_id":        "TP-001",
		"year_of_assessment": 2026,
		"jurisdiction":       "NG-LA",
	})
	return returns.Event{
		Seq:      seq,
		ReturnID: "R-EMP-TEST",
		Kind:     returns.EvReturnOpened,
		Payload:  json.RawMessage(payload),
		Actor:    returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-001"},
		OccurredAt: t0,
	}
}

var testEmp = employer.Employer{ID: "EMP-ACME", LegalName: "Acme Ltd"}
var testRec = employer.PayrollRecord{
	ID:        "PR-001",
	ImportID:  "IMP-001",
	GrossKobo: taxengine.Kobo(900_000_000),
	PAYEKobo:  taxengine.Kobo(160_000_000),
}

func TestProposalEvent_Shape(t *testing.T) {
	ev := employer.ProposalEvent(testRec, testEmp, "R-EMP-TEST", 2, t0)

	if ev.Kind != returns.EvPayrollProposed {
		t.Errorf("Kind: want %s, got %s", returns.EvPayrollProposed, ev.Kind)
	}
	if ev.Actor.Kind != returns.ActorEmployer {
		t.Errorf("Actor.Kind: want %s, got %s", returns.ActorEmployer, ev.Actor.Kind)
	}
	if ev.Actor.ID != "EMP-ACME" {
		t.Errorf("Actor.ID: want EMP-ACME, got %s", ev.Actor.ID)
	}
	if ev.Seq != 2 || ev.ReturnID != "R-EMP-TEST" {
		t.Errorf("Seq or ReturnID wrong: seq=%d returnID=%s", ev.Seq, ev.ReturnID)
	}

	// PayrollProposal has no json tags; Fold unmarshals by capital-case field name.
	// Assert the same way so this test mirrors what Fold actually reads.
	var prop returns.PayrollProposal
	if err := json.Unmarshal(ev.Payload, &prop); err != nil {
		t.Fatalf("payload unmarshal: %v", err)
	}
	if prop.RecordID != "PR-001" {
		t.Errorf("RecordID: want PR-001, got %s", prop.RecordID)
	}
	if prop.GrossKobo != 900_000_000 {
		t.Errorf("GrossKobo: want 900000000, got %d", prop.GrossKobo)
	}
}

func TestProposalEvent_DoesNotTouchIncome(t *testing.T) {
	// Fold an open event then the proposal; assert no income line and no stale assessment.
	p, err := returns.Replay([]returns.Event{
		openEvent(1),
		employer.ProposalEvent(testRec, testEmp, "R-EMP-TEST", 2, t0),
	})
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if len(p.Income) != 0 {
		t.Errorf("EvPayrollProposed must not create income lines; got %d", len(p.Income))
	}
	if p.AssessmentStale {
		t.Error("EvPayrollProposed must not mark assessment stale")
	}
	if pr, ok := p.ProposedPayroll["PR-001"]; !ok || pr.Status != "PROPOSED" {
		t.Errorf("ProposedPayroll[PR-001].Status: want PROPOSED, got %q", p.ProposedPayroll["PR-001"].Status)
	}
}

func TestAcceptanceEvent_TaxpayerActor(t *testing.T) {
	actor := returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-001"}
	line := returns.IncomeLine{ID: "inc-1", Category: "EMPLOYMENT", AmountKobo: 900_000_000, Label: "Acme Ltd"}
	_, err := employer.AcceptanceEvent("PR-001", line, "R-EMP-TEST", 3, actor, t0)
	if err != nil {
		t.Errorf("expected success for TAXPAYER actor, got: %v", err)
	}
}

func TestAcceptanceEvent_NonTaxpayerActorRejected(t *testing.T) {
	for _, kind := range []returns.ActorKind{returns.ActorEmployer, returns.ActorSystem, returns.ActorStaff} {
		actor := returns.Actor{Kind: kind, ID: "X"}
		line := returns.IncomeLine{ID: "inc-1", Category: "EMPLOYMENT", AmountKobo: 900_000_000}
		_, err := employer.AcceptanceEvent("PR-001", line, "R-EMP-TEST", 3, actor, t0)
		if err == nil {
			t.Errorf("AcceptanceEvent with actor %s: expected error, got nil", kind)
		}
	}
}

func TestAcceptanceEvent_AutoAcceptPatternDetected(t *testing.T) {
	sysActor := returns.Actor{Kind: returns.ActorSystem, ID: "auto"}
	line := returns.IncomeLine{ID: "inc-1", Category: "EMPLOYMENT", AmountKobo: 900_000_000}
	_, err := employer.AcceptanceEvent("PR-001", line, "R-EMP-TEST", 3, sysActor, t0)
	if err == nil {
		t.Fatal("SYSTEM actor auto-accept must be rejected")
	}
	if err.Error() == "" {
		t.Error("error message must explain the DeclaredByTaxpayer guard")
	}
}

func TestAcceptanceEvent_CreatesDeclaredByTaxpayer(t *testing.T) {
	taxpayerActor := returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-001"}
	line := returns.IncomeLine{ID: "inc-1", Category: "EMPLOYMENT", AmountKobo: 900_000_000, Label: "Acme Ltd"}
	acceptEv, err := employer.AcceptanceEvent("PR-001", line, "R-EMP-TEST", 3, taxpayerActor, t0)
	if err != nil {
		t.Fatalf("AcceptanceEvent: %v", err)
	}

	p, err := returns.Replay([]returns.Event{
		openEvent(1),
		employer.ProposalEvent(testRec, testEmp, "R-EMP-TEST", 2, t0),
		acceptEv,
	})
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	if !p.DeclaredByTaxpayer {
		t.Error("DeclaredByTaxpayer must be true after TAXPAYER accepts the payroll record")
	}
	if inc, ok := p.Income["inc-1"]; !ok || inc.AmountKobo != 900_000_000 {
		t.Errorf("Income[inc-1]: want 900000000, got %d", p.Income["inc-1"].AmountKobo)
	}
}
