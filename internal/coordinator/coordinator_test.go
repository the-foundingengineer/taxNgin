package coordinator_test

import (
	"encoding/json"
	"testing"
	"time"

	"taxsoftware/internal/coordinator"
	"taxsoftware/internal/golden"
	"taxsoftware/internal/returns"
	"taxsoftware/internal/validate"
	"taxsoftware/taxengine"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var t0 = time.Date(2027, 4, 1, 9, 0, 0, 0, time.UTC)

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(b)
}

// buildProjection creates a projection for the given year with a fixed set of
// income lines (amounts in kobo).
func buildProjection(t *testing.T, yoa int, incomeAmountsKobo []int64) *returns.Projection {
	t.Helper()
	events := []returns.Event{
		{
			Seq: 1, Kind: returns.EvReturnOpened,
			Payload: mustJSON(map[string]any{
				"taxpayer_id":        "TP-COORD",
				"year_of_assessment": yoa,
				"jurisdiction":       "NG-LA",
			}),
			Actor: returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-COORD"},
		},
	}
	for i, amt := range incomeAmountsKobo {
		events = append(events, returns.Event{
			Seq:  i + 2,
			Kind: returns.EvIncomeDeclared,
			Actor: returns.Actor{Kind: returns.ActorTaxpayer},
			Payload: mustJSON(returns.IncomeLine{
				ID:         "inc-" + string(rune('a'+i)),
				Category:   "EMPLOYMENT",
				AmountKobo: amt,
			}),
		})
	}
	p, err := returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay YoA=%d: %v", yoa, err)
	}
	return p
}

// buildProjectionWithPayroll creates a projection with an accepted payroll
// proposal. employerGrossKobo is what the employer reported; declaredKobo is
// what the taxpayer has on their income line (may differ to trigger W102).
func buildProjectionWithPayroll(t *testing.T, employerGrossKobo, declaredKobo int64) *returns.Projection {
	t.Helper()
	events := []returns.Event{
		{
			Seq: 1, Kind: returns.EvReturnOpened,
			Payload: mustJSON(map[string]any{
				"taxpayer_id":        "TP-COORD",
				"year_of_assessment": 2026,
				"jurisdiction":       "NG-LA",
			}),
			Actor: returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-COORD"},
		},
		{
			Seq:  2,
			Kind: returns.EvPayrollProposed,
			Actor: returns.Actor{Kind: returns.ActorEmployer, ID: "EMP-1"},
			Payload: mustJSON(returns.PayrollProposal{
				RecordID:   "pr-1",
				EmployerID: "EMP-1",
				GrossKobo:  employerGrossKobo,
				PayeKobo:   0,
			}),
		},
		{
			Seq:  3,
			Kind: returns.EvPayrollAccepted,
			Actor: returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-COORD"},
			Payload: mustJSON(map[string]any{
				"record_id": "pr-1",
				"income": returns.IncomeLine{
					ID:         "inc-1",
					Category:   "EMPLOYMENT",
					AmountKobo: declaredKobo,
					Label:      "Employer A",
				},
			}),
		},
	}
	p, err := returns.Replay(events)
	if err != nil {
		t.Fatalf("Replay with payroll: %v", err)
	}
	return p
}

// baseCtx returns a FilingContext sufficient to pass all identity checks so
// that W105/W102/W106 findings are not obscured by identity errors.
func baseCtx(facts taxengine.TaxFacts) validate.FilingContext {
	return validate.FilingContext{
		Facts:              facts,
		TaxID:              "TIN-COORD",
		TaxIDVerified:      true,
		NIN:                "11122233344",
		Signed:             true,
		DeclaredByTaxpayer: true,
		RelevantAuthority:  "LIRS",
	}
}

func hasIssue(t *testing.T, result validate.Result, code string) bool {
	t.Helper()
	for _, iss := range result.Issues {
		if iss.Code == code {
			return true
		}
	}
	return false
}

func loadRS(t *testing.T) *taxengine.RuleSet {
	t.Helper()
	rs, err := golden.LoadRuleSet()
	if err != nil {
		t.Fatalf("LoadRuleSet: %v", err)
	}
	return rs
}

// ---------------------------------------------------------------------------
// W105 — prior year gross
// ---------------------------------------------------------------------------

func TestBuildFilingContext_W105_FiresOnDrop(t *testing.T) {
	rs := loadRS(t)
	// Prior year: ₦3,000,000 gross (300,000,000 kobo)
	prior := buildProjection(t, 2025, []int64{300_000_000})
	// Target year: ₦1,000,000 gross — less than 50% of prior → W105
	target := buildProjection(t, 2026, []int64{100_000_000})

	facts := taxengine.TaxFacts{
		YearOfAssessment: 2026, Jurisdiction: "NG",
		IsResident:    true,
		Income:        []taxengine.IncomeItem{{Category: taxengine.IncEmployment, AmountKobo: 100_000_000}},
		DeclaredBases: map[string]taxengine.Kobo{},
	}
	ctx := coordinator.BuildFilingContext(target, []*returns.Projection{prior}, baseCtx(facts))

	if ctx.PriorYearGrossKobo == nil {
		t.Fatal("PriorYearGrossKobo must be set when prior year has income")
	}
	if *ctx.PriorYearGrossKobo != taxengine.Kobo(300_000_000) {
		t.Errorf("PriorYearGrossKobo: want 300,000,000, got %d", *ctx.PriorYearGrossKobo)
	}

	result, err := validate.Validate(ctx, rs)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !hasIssue(t, result, "W105") {
		t.Errorf("W105 must fire when income dropped >50%%; issues: %v", result.Issues)
	}
}

func TestBuildFilingContext_W105_NoFireOnSmallDrop(t *testing.T) {
	rs := loadRS(t)
	// Prior: ₦3,000,000. Target: ₦2,000,000 (>50% of prior → no W105)
	prior := buildProjection(t, 2025, []int64{300_000_000})
	target := buildProjection(t, 2026, []int64{200_000_000})

	facts := taxengine.TaxFacts{
		YearOfAssessment: 2026, Jurisdiction: "NG",
		IsResident:    true,
		Income:        []taxengine.IncomeItem{{Category: taxengine.IncEmployment, AmountKobo: 200_000_000}},
		DeclaredBases: map[string]taxengine.Kobo{},
	}
	ctx := coordinator.BuildFilingContext(target, []*returns.Projection{prior}, baseCtx(facts))

	result, err := validate.Validate(ctx, rs)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if hasIssue(t, result, "W105") {
		t.Error("W105 must NOT fire when income dropped <50%")
	}
}

func TestBuildFilingContext_W105_NoPriorYear(t *testing.T) {
	target := buildProjection(t, 2026, []int64{100_000_000})
	facts := taxengine.TaxFacts{YearOfAssessment: 2026, DeclaredBases: map[string]taxengine.Kobo{}}
	ctx := coordinator.BuildFilingContext(target, nil, baseCtx(facts))

	if ctx.PriorYearGrossKobo != nil {
		t.Error("PriorYearGrossKobo must be nil when no prior year is in history")
	}
}

func TestBuildFilingContext_W105_NilReturnPriorYear(t *testing.T) {
	// Prior year has NO income lines (nil return)
	prior := buildProjection(t, 2025, nil)
	target := buildProjection(t, 2026, []int64{100_000_000})
	facts := taxengine.TaxFacts{YearOfAssessment: 2026, DeclaredBases: map[string]taxengine.Kobo{}}
	ctx := coordinator.BuildFilingContext(target, []*returns.Projection{prior}, baseCtx(facts))

	if ctx.PriorYearGrossKobo != nil {
		t.Error("PriorYearGrossKobo must be nil when prior year was a nil return (zero income)")
	}
}

// ---------------------------------------------------------------------------
// W102 — employer vs. taxpayer gross
// ---------------------------------------------------------------------------

func TestBuildFilingContext_W102_FiresOnMismatch(t *testing.T) {
	rs := loadRS(t)
	// Employer reported ₦9,000,000; taxpayer has revised to ₦8,500,000.
	target := buildProjectionWithPayroll(t, 900_000_000, 850_000_000)

	facts := taxengine.TaxFacts{
		YearOfAssessment: 2026, Jurisdiction: "NG",
		IsResident:    true,
		Income:        []taxengine.IncomeItem{{Category: taxengine.IncEmployment, AmountKobo: 850_000_000}},
		DeclaredBases: map[string]taxengine.Kobo{},
	}
	ctx := coordinator.BuildFilingContext(target, nil, baseCtx(facts))

	if ctx.EmployerGrossKobo == nil {
		t.Fatal("EmployerGrossKobo must be set when payroll was accepted")
	}
	if *ctx.EmployerGrossKobo != taxengine.Kobo(900_000_000) {
		t.Errorf("EmployerGrossKobo: want 900,000,000, got %d", *ctx.EmployerGrossKobo)
	}

	result, err := validate.Validate(ctx, rs)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !hasIssue(t, result, "W102") {
		t.Errorf("W102 must fire when employer and taxpayer gross differ; issues: %v", result.Issues)
	}
}

func TestBuildFilingContext_W102_NoFireOnMatch(t *testing.T) {
	rs := loadRS(t)
	// Employer and taxpayer both say ₦9,000,000.
	target := buildProjectionWithPayroll(t, 900_000_000, 900_000_000)

	facts := taxengine.TaxFacts{
		YearOfAssessment: 2026, Jurisdiction: "NG",
		IsResident:    true,
		Income:        []taxengine.IncomeItem{{Category: taxengine.IncEmployment, AmountKobo: 900_000_000}},
		DeclaredBases: map[string]taxengine.Kobo{},
	}
	ctx := coordinator.BuildFilingContext(target, nil, baseCtx(facts))

	result, err := validate.Validate(ctx, rs)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if hasIssue(t, result, "W102") {
		t.Error("W102 must NOT fire when employer and taxpayer gross match exactly")
	}
}

func TestBuildFilingContext_W102_NoProposals(t *testing.T) {
	target := buildProjection(t, 2026, []int64{900_000_000})
	facts := taxengine.TaxFacts{YearOfAssessment: 2026, DeclaredBases: map[string]taxengine.Kobo{}}
	ctx := coordinator.BuildFilingContext(target, nil, baseCtx(facts))

	if ctx.EmployerGrossKobo != nil {
		t.Error("EmployerGrossKobo must be nil when no payroll proposals exist")
	}
}

// ---------------------------------------------------------------------------
// YearsOnFile
// ---------------------------------------------------------------------------

func TestBuildFilingContext_YearsOnFile(t *testing.T) {
	// Build two history projections in filed states.
	makeFiledProjection := func(yoa int, state returns.State) *returns.Projection {
		p := buildProjection(t, yoa, []int64{100_000_000})
		// Advance through the lifecycle to reach ACK_UPLOADED.
		// We do this by folding the necessary events directly.
		seq := p.Version + 1
		stateEvents := []returns.Event{
			{Seq: seq, Kind: returns.EvAssessmentComputed,
				Payload: mustJSON(returns.AssessmentSnapshot{RuleSetPublished: true, ComputedAt: t0})},
			{Seq: seq + 1, Kind: returns.EvValidationRun, Payload: []byte(`{"blocking":[],"warnings":[]}`)},
			{Seq: seq + 2, Kind: returns.EvReturnSigned, OccurredAt: t0},
			{Seq: seq + 3, Kind: returns.EvPackIssued, Payload: mustJSON(map[string]any{"portal_map_version": "0.1"})},
		}
		if state == returns.StateAckUploaded || state == returns.StateCleared {
			stateEvents = append(stateEvents,
				returns.Event{Seq: seq + 4, Kind: returns.EvFilingAttested, OccurredAt: t0},
				returns.Event{Seq: seq + 5, Kind: returns.EvAcknowledgementUploaded, OccurredAt: t0},
			)
		}
		if state == returns.StateCleared {
			stateEvents = append(stateEvents,
				returns.Event{Seq: seq + 6, Kind: returns.EvAssessmentServed,
					Payload: mustJSON(map[string]any{"served_at": t0, "objection_window_days": 30})},
				returns.Event{Seq: seq + 7, Kind: returns.EvPaymentRecorded, OccurredAt: t0},
				returns.Event{Seq: seq + 8, Kind: returns.EvReturnCleared, OccurredAt: t0},
			)
		}
		// Rebuild with all events
		allEvents := make([]returns.Event, 0)
		// We need to collect the original events from the projection. Since Replay
		// is the source of truth, reconstruct from scratch by replaying.
		base := []returns.Event{
			{
				Seq: 1, Kind: returns.EvReturnOpened,
				Payload: mustJSON(map[string]any{
					"taxpayer_id":        "TP-COORD",
					"year_of_assessment": yoa,
					"jurisdiction":       "NG-LA",
				}),
				Actor: returns.Actor{Kind: returns.ActorTaxpayer, ID: "TP-COORD"},
			},
			{
				Seq: 2, Kind: returns.EvIncomeDeclared,
				Actor:   returns.Actor{Kind: returns.ActorTaxpayer},
				Payload: mustJSON(returns.IncomeLine{ID: "inc-a", Category: "EMPLOYMENT", AmountKobo: 100_000_000}),
			},
		}
		for i := range stateEvents {
			stateEvents[i].Seq = len(base) + i + 1
		}
		allEvents = append(base, stateEvents...)
		p2, err := returns.Replay(allEvents)
		if err != nil {
			t.Fatalf("Replay to state %s: %v", state, err)
		}
		return p2
	}

	h1 := makeFiledProjection(2024, returns.StateCleared)    // CLEARED = filed
	h2 := makeFiledProjection(2025, returns.StateAckUploaded) // ACK_UPLOADED = filed
	target := buildProjection(t, 2026, []int64{100_000_000}) // DRAFT = not filed

	facts := taxengine.TaxFacts{YearOfAssessment: 2026, DeclaredBases: map[string]taxengine.Kobo{}}
	ctx := coordinator.BuildFilingContext(target, []*returns.Projection{h1, h2}, baseCtx(facts))

	// History has 2 filed, target is DRAFT (not filed) → YearsOnFile = 2.
	if ctx.YearsOnFile != 2 {
		t.Errorf("YearsOnFile: want 2 (2 history years filed, target DRAFT), got %d", ctx.YearsOnFile)
	}
}
