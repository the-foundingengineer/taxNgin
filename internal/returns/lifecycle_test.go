package returns_test

import (
	"encoding/json"
	"testing"
	"time"

	"taxsoftware/internal/returns"
	"taxsoftware/taxengine"
)

func baseProjection() *returns.Projection {
	p := &returns.Projection{
		State:         returns.StateValidated,
		RuleSetPublished: true,
		DeclaredByTaxpayer: true,
		AssessmentStale: false,
	}
	t := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	p.Assessment = &returns.AssessmentSnapshot{
		FactsHash:      "abc123",
		RuleSetVersion: "0.1.0",
		RuleSetDigest:  "def456",
		ComputedAt:     t,
	}
	p.BlockingErrors = []string{}
	return p
}

func TestSignGuardBlockingErrors(t *testing.T) {
	p := baseProjection()
	p.BlockingErrors = []string{"E001 missing tax ID"}

	_, err := returns.Apply(returns.StateValidated, returns.TrSign, p)
	if err == nil {
		t.Fatal("expected SIGN to fail with blocking errors outstanding")
	}
}

func TestSignGuardNoAssessment(t *testing.T) {
	p := baseProjection()
	p.Assessment = nil

	_, err := returns.Apply(returns.StateValidated, returns.TrSign, p)
	if err == nil {
		t.Fatal("expected SIGN to fail with no assessment")
	}
}

func TestSignGuardStaleAssessment(t *testing.T) {
	p := baseProjection()
	p.AssessmentStale = true

	_, err := returns.Apply(returns.StateValidated, returns.TrSign, p)
	if err == nil {
		t.Fatal("expected SIGN to fail with stale assessment")
	}
}

func TestSignGuardUnpublishedRuleSet(t *testing.T) {
	p := baseProjection()
	p.RuleSetPublished = false

	_, err := returns.Apply(returns.StateValidated, returns.TrSign, p)
	if err == nil {
		t.Fatal("expected SIGN to fail with unpublished rule set")
	}
}

func TestSignGuardNotDeclaredByTaxpayer(t *testing.T) {
	p := baseProjection()
	p.DeclaredByTaxpayer = false

	_, err := returns.Apply(returns.StateValidated, returns.TrSign, p)
	if err == nil {
		t.Fatal("expected SIGN to fail when DeclaredByTaxpayer is false")
	}
}

func TestSignSuccessAllGuardsPass(t *testing.T) {
	p := baseProjection()

	next, err := returns.Apply(returns.StateValidated, returns.TrSign, p)
	if err != nil {
		t.Fatalf("expected SIGN to succeed, got error: %v", err)
	}
	if next != returns.StateSigned {
		t.Errorf("expected SIGN from VALIDATED to go to SIGNED, got %s", next)
	}
}

func TestValidateGuardBlockingErrors(t *testing.T) {
	p := baseProjection()
	p.State = returns.StateDraft
	p.BlockingErrors = []string{"E001 missing tax ID"}

	_, err := returns.Apply(returns.StateDraft, returns.TrValidate, p)
	if err == nil {
		t.Fatal("expected VALIDATE to fail with blocking errors")
	}
}

func TestValidateSuccess(t *testing.T) {
	p := baseProjection()
	p.State = returns.StateDraft
	p.Assessment = nil
	p.BlockingErrors = []string{}

	next, err := returns.Apply(returns.StateDraft, returns.TrValidate, p)
	if err != nil {
		t.Fatalf("VALIDATE failed: %v", err)
	}
	if next != returns.StateValidated {
		t.Errorf("expected VALIDATE -> VALIDATED, got %s", next)
	}
}

func TestInvalidTransition(t *testing.T) {
	p := baseProjection()
	p.State = returns.StateDraft
	_, err := returns.Apply(returns.StateDraft, returns.TrClear, p)
	if err == nil {
		t.Fatal("expected error for invalid transition from DRAFT -> CLEAR")
	}
}

func TestStateTerminal(t *testing.T) {
	terminals := []returns.State{returns.StateCleared, returns.StateSuperseded, returns.StateAbandoned}
	for _, s := range terminals {
		if !s.IsTerminal() {
			t.Errorf("%s should be terminal", s)
		}
	}
	nonTerminals := []returns.State{returns.StateDraft, returns.StateValidated, returns.StateSigned, returns.StatePackIssued}
	for _, s := range nonTerminals {
		if s.IsTerminal() {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

func TestObjectionDeadline(t *testing.T) {
	served := time.Date(2027, 5, 1, 0, 0, 0, 0, time.UTC)
	got := returns.ObjectionDeadline(served, 30)
	want := time.Date(2027, 5, 31, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("ObjectionDeadline = %v, want %v", got, want)
	}
}

func TestPackIssuedGuardUnsigned(t *testing.T) {
	p := baseProjection()
	p.State = returns.StateSigned
	p.SignedAt = nil
	p.PortalMapVersion = "1.0"

	_, err := returns.Apply(returns.StateSigned, returns.TrIssuePack, p)
	if err == nil {
		t.Fatal("expected ISSUE_PACK to fail when unsigned")
	}
}

func TestPackIssuedGuardNoPortalMap(t *testing.T) {
	p := baseProjection()
	p.State = returns.StateSigned
	signTime := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	p.SignedAt = &signTime
	p.PortalMapVersion = ""

	_, err := returns.Apply(returns.StateSigned, returns.TrIssuePack, p)
	if err == nil {
		t.Fatal("expected ISSUE_PACK to fail with no portal map")
	}
}

var _ = taxengine.Naira
