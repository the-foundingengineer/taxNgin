package employer_test

import (
	"encoding/json"
	"testing"

	"taxsoftware/internal/coordinator"
	"taxsoftware/internal/employer"
	"taxsoftware/internal/golden"
	"taxsoftware/internal/returns"
	"taxsoftware/internal/validate"
	"taxsoftware/taxengine"
)

// ---------------------------------------------------------------------------
// S-EMPLOYER-001: Mid-year job change — two employers, income summed
// ---------------------------------------------------------------------------

func TestSEmployer001_MidYearJobChange(t *testing.T) {
	// 1. Open a return.
	events := []returns.Event{openEvent(1)}
	p, err := returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay open: %v", err)
	}

	empA := employer.Employer{ID: "EMP-A", LegalName: "Acme Ltd"}
	empB := employer.Employer{ID: "EMP-B", LegalName: "Beta Corp"}
	recA := employer.PayrollRecord{ID: "PR-A", GrossKobo: 600_000_000, PAYEKobo: 80_000_000}
	recB := employer.PayrollRecord{ID: "PR-B", GrossKobo: 400_000_000, PAYEKobo: 40_000_000}

	// 2. Employer A proposes.
	propA := employer.ProposalEvent(recA, empA, "R-EMP-TEST", 2, t0)
	events = append(events, propA)
	p, err = returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay after propA: %v", err)
	}
	if len(p.Income) != 0 {
		t.Error("Proposal A must not create income lines")
	}
	if p.AssessmentStale {
		t.Error("Proposal A must not mark assessment stale")
	}

	// 3. Employer B proposes.
	propB := employer.ProposalEvent(recB, empB, "R-EMP-TEST", 3, t0)
	events = append(events, propB)
	p, err = returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay after propB: %v", err)
	}
	if len(p.Income) != 0 {
		t.Error("Proposal B must not create income lines")
	}

	// 4. Taxpayer accepts Employer A.
	lineA := returns.IncomeLine{ID: "inc-A", Category: "EMPLOYMENT", AmountKobo: 600_000_000, Label: "Acme Ltd"}
	acceptA, err := employer.AcceptanceEvent("PR-A", lineA, "R-EMP-TEST", 4,
		returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-001"}, t0)
	if err != nil {
		t.Fatalf("AcceptanceEvent A: %v", err)
	}
	events = append(events, acceptA)
	p, err = returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay after acceptA: %v", err)
	}
	if _, ok := p.Income["inc-A"]; !ok {
		t.Error("inc-A should be in Income after acceptance")
	}
	if len(p.Income) != 1 {
		t.Errorf("after accepting A: want 1 income line, got %d", len(p.Income))
	}

	// 5. Taxpayer accepts Employer B.
	lineB := returns.IncomeLine{ID: "inc-B", Category: "EMPLOYMENT", AmountKobo: 400_000_000, Label: "Beta Corp"}
	acceptB, err := employer.AcceptanceEvent("PR-B", lineB, "R-EMP-TEST", 5,
		returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-001"}, t0)
	if err != nil {
		t.Fatalf("AcceptanceEvent B: %v", err)
	}
	events = append(events, acceptB)
	p, err = returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay after acceptB: %v", err)
	}
	if len(p.Income) != 2 {
		t.Errorf("after accepting B: want 2 income lines, got %d", len(p.Income))
	}
	var totalIncome int64
	for _, line := range p.Income {
		totalIncome += line.AmountKobo
	}
	if totalIncome != 1_000_000_000 {
		t.Errorf("total income: want 1,000,000,000 kobo (₦10m), got %d", totalIncome)
	}
	if !p.DeclaredByTaxpayer {
		t.Error("DeclaredByTaxpayer must be true after both records accepted by TAXPAYER")
	}

	// 6. SIGN guard requires a computed assessment.
	_, err = returns.Apply(p.State, returns.TrSign, p)
	if err == nil {
		t.Error("SIGN must fail: no computed assessment exists yet")
	}

	// 7. AcceptanceEvent with SYSTEM actor must be rejected before emitting any event.
	_, err = employer.AcceptanceEvent("PR-A", lineA, "R-EMP-TEST", 99,
		returns.Actor{Kind: returns.ActorSystem, ID: "auto"}, t0)
	if err == nil {
		t.Error("AcceptanceEvent with SYSTEM actor must return error — auto-accept is not permitted")
	}
}

// ---------------------------------------------------------------------------
// S-EMPLOYER-002: Partial acceptance — one accepted, one rejected
// ---------------------------------------------------------------------------

func TestSEmployer002_PartialAcceptance(t *testing.T) {
	empA := employer.Employer{ID: "EMP-A", LegalName: "Acme Ltd"}
	empB := employer.Employer{ID: "EMP-B", LegalName: "Beta Corp"}
	recA := employer.PayrollRecord{ID: "PR-A", GrossKobo: 500_000_000}
	recB := employer.PayrollRecord{ID: "PR-B", GrossKobo: 300_000_000}

	lineA := returns.IncomeLine{ID: "inc-A", Category: "EMPLOYMENT", AmountKobo: 500_000_000, Label: "Acme Ltd"}
	acceptA, err := employer.AcceptanceEvent("PR-A", lineA, "R-EMP-TEST", 4,
		returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-001"}, t0)
	if err != nil {
		t.Fatalf("AcceptanceEvent A: %v", err)
	}

	rejectBPayload, _ := json.Marshal(map[string]any{"record_id": "PR-B"})
	rejectB := returns.Event{
		Seq:      5,
		Kind:     returns.EvPayrollRejected,
		ReturnID: "R-EMP-TEST",
		Actor:    returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-001"},
		Payload:  json.RawMessage(rejectBPayload),
	}

	events := []returns.Event{
		openEvent(1),
		employer.ProposalEvent(recA, empA, "R-EMP-TEST", 2, t0),
		employer.ProposalEvent(recB, empB, "R-EMP-TEST", 3, t0),
		acceptA,
		rejectB,
	}
	p, err := returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	if _, ok := p.Income["inc-A"]; !ok {
		t.Error("inc-A should be in Income")
	}
	if len(p.Income) != 1 {
		t.Errorf("want 1 income line (accepted only), got %d", len(p.Income))
	}
	if prA := p.ProposedPayroll["PR-A"]; prA.Status != "ACCEPTED" {
		t.Errorf("PR-A status: want ACCEPTED, got %s", prA.Status)
	}
	if prB := p.ProposedPayroll["PR-B"]; prB.Status != "REJECTED" {
		t.Errorf("PR-B status: want REJECTED, got %s", prB.Status)
	}
	if !p.DeclaredByTaxpayer {
		t.Error("DeclaredByTaxpayer must be true (only TAXPAYER-actor income exists)")
	}
}

// ---------------------------------------------------------------------------
// S-EMPLOYER-003: W102 wire-up via coordinator
// ---------------------------------------------------------------------------

func TestSEmployer003_W102WireUp(t *testing.T) {
	rs, err := golden.LoadRuleSet()
	if err != nil {
		t.Fatalf("LoadRuleSet: %v", err)
	}

	emp := employer.Employer{ID: "EMP-A", LegalName: "Acme Ltd"}
	rec := employer.PayrollRecord{ID: "PR-1", GrossKobo: 900_000_000, PAYEKobo: 100_000_000}

	// Accept the payroll at the employer-reported gross.
	lineAtEmpGross := returns.IncomeLine{
		ID: "inc-1", Category: "EMPLOYMENT", AmountKobo: 900_000_000, Label: "Acme Ltd",
	}
	acceptEv, err := employer.AcceptanceEvent("PR-1", lineAtEmpGross, "R-EMP-TEST", 3,
		returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-001"}, t0)
	if err != nil {
		t.Fatalf("AcceptanceEvent: %v", err)
	}

	// Taxpayer revises income downward (payslips show different figure).
	revisedPayload, _ := json.Marshal(returns.IncomeLine{
		ID: "inc-1", Category: "EMPLOYMENT", AmountKobo: 850_000_000, Label: "Acme Ltd",
	})
	reviseEv := returns.Event{
		Seq: 4, Kind: returns.EvIncomeRevised, ReturnID: "R-EMP-TEST",
		Actor:   returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-001"},
		Payload: json.RawMessage(revisedPayload),
	}

	events := []returns.Event{
		openEvent(1),
		employer.ProposalEvent(rec, emp, "R-EMP-TEST", 2, t0),
		acceptEv,
		reviseEv,
	}
	target, err := returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	// The declared income is now ₦8,500,000; the employer reported ₦9,000,000.
	if inc := target.Income["inc-1"].AmountKobo; inc != 850_000_000 {
		t.Errorf("revised income: want 850000000, got %d", inc)
	}

	facts := taxengine.TaxFacts{
		YearOfAssessment: 2026,
		Jurisdiction:     "NG",
		IsResident:       true,
		Income:           []taxengine.IncomeItem{{Category: taxengine.IncEmployment, AmountKobo: 850_000_000}},
		DeclaredBases:    map[string]taxengine.Kobo{},
	}
	base := validate.FilingContext{
		Facts:              facts,
		TaxID:              "TIN-001",
		TaxIDVerified:      true,
		NIN:                "11122233344",
		Signed:             true,
		DeclaredByTaxpayer: target.DeclaredByTaxpayer,
		RelevantAuthority:  "LIRS",
	}
	ctx := coordinator.BuildFilingContext(target, nil, base)

	// Coordinator must derive EmployerGrossKobo from the accepted proposal.
	if ctx.EmployerGrossKobo == nil {
		t.Fatal("EmployerGrossKobo must be set from accepted payroll proposal")
	}
	if *ctx.EmployerGrossKobo != taxengine.Kobo(900_000_000) {
		t.Errorf("EmployerGrossKobo: want 900,000,000, got %d", *ctx.EmployerGrossKobo)
	}

	result, err := validate.Validate(ctx, rs)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	hasW102 := false
	for _, iss := range result.Issues {
		if iss.Code == "W102" {
			hasW102 = true
		}
	}
	if !hasW102 {
		t.Errorf("W102 must fire when employer reported ₦9m and taxpayer declared ₦8.5m; issues: %v", result.Issues)
	}
}
