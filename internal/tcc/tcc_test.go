package tcc_test

import (
	"testing"

	"taxsoftware/internal/returns"
	"taxsoftware/internal/tcc"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func threeFiledYears() []tcc.YearSummary {
	return []tcc.YearSummary{
		{YearOfAssessment: 2024, State: returns.StateCleared, AssessmentServed: true, BalancePaid: true},
		{YearOfAssessment: 2025, State: returns.StateAckUploaded, AssessmentServed: true, BalancePaid: true},
		{YearOfAssessment: 2026, State: returns.StateAckUploaded, AssessmentServed: true, BalancePaid: true},
	}
}

func findCondition(t *testing.T, status tcc.TCCStatus, id string) tcc.Condition {
	t.Helper()
	for _, c := range status.Conditions {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("condition %q not found in TCCStatus", id)
	return tcc.Condition{}
}

// ---------------------------------------------------------------------------
// Test cases
// ---------------------------------------------------------------------------

// TestReadiness_AllMet verifies the golden path: three years of returns,
// all assessments received, all balances paid, passport and address present.
func TestReadiness_AllMet(t *testing.T) {
	status := tcc.ComputeReadiness(tcc.ReadinessInput{
		Years:            threeFiledYears(),
		HasPassportPhoto: true,
		HasFullAddress:   true,
	})

	if !status.AllMet {
		for _, c := range status.Conditions {
			if !c.Met {
				t.Errorf("condition %s not met: %s", c.ID, c.Note)
			}
		}
		t.Fatal("expected AllMet=true")
	}

	if len(status.Conditions) != 6 {
		t.Errorf("expected 6 conditions, got %d", len(status.Conditions))
	}

	for _, c := range status.Conditions {
		if !c.Met {
			t.Errorf("%s: want Met=true, got false (note: %s)", c.ID, c.Note)
		}
		if c.Note != "" {
			t.Errorf("%s: Note should be empty when Met, got %q", c.ID, c.Note)
		}
	}
}

// TestReadiness_OnlyTwoYears checks that T1 fails when fewer than three years
// have been filed. T4 should also fail since the third year has no assessment.
func TestReadiness_OnlyTwoYears(t *testing.T) {
	years := []tcc.YearSummary{
		{YearOfAssessment: 2025, State: returns.StateAckUploaded, AssessmentServed: true, BalancePaid: true},
		{YearOfAssessment: 2026, State: returns.StateAckUploaded, AssessmentServed: true, BalancePaid: true},
	}

	status := tcc.ComputeReadiness(tcc.ReadinessInput{
		Years:            years,
		HasPassportPhoto: true,
		HasFullAddress:   true,
	})

	if status.AllMet {
		t.Fatal("expected AllMet=false with only two years filed")
	}

	t1 := findCondition(t, status, "T1")
	if t1.Met {
		t.Error("T1: want Met=false (only 2 years filed)")
	}
	if t1.Note == "" {
		t.Error("T1: Note must explain why condition is not met")
	}

	t4 := findCondition(t, status, "T4")
	if t4.Met {
		t.Error("T4: want Met=false (only 2 assessments served)")
	}
}

// TestReadiness_OneYearUnpaid checks that T5 fails when at least one year
// has an outstanding balance. All other conditions are met.
func TestReadiness_OneYearUnpaid(t *testing.T) {
	years := []tcc.YearSummary{
		{YearOfAssessment: 2024, State: returns.StateCleared, AssessmentServed: true, BalancePaid: true},
		{YearOfAssessment: 2025, State: returns.StateAckUploaded, AssessmentServed: true, BalancePaid: true},
		// 2026 is assessed but not yet paid.
		{YearOfAssessment: 2026, State: returns.StateAssessed, AssessmentServed: true, BalancePaid: false},
	}

	status := tcc.ComputeReadiness(tcc.ReadinessInput{
		Years:            years,
		HasPassportPhoto: true,
		HasFullAddress:   true,
	})

	if status.AllMet {
		t.Fatal("expected AllMet=false with one year unpaid")
	}

	t5 := findCondition(t, status, "T5")
	if t5.Met {
		t.Error("T5: want Met=false (2026 balance not paid)")
	}
	if t5.Note == "" {
		t.Error("T5: Note must explain the outstanding balance")
	}

	// T1 should still be met: 2024 (CLEARED) and 2025 (ACK_UPLOADED) qualify,
	// but 2026 is only ASSESSED — not filed. So T1 has only 2 qualifying years.
	t1 := findCondition(t, status, "T1")
	if t1.Met {
		t.Error("T1: want Met=false (2026 is ASSESSED, not ACK_UPLOADED or CLEARED)")
	}
}

// TestReadiness_MissingPassport checks that T2 fails when the taxpayer has
// not uploaded a passport photograph.
func TestReadiness_MissingPassport(t *testing.T) {
	status := tcc.ComputeReadiness(tcc.ReadinessInput{
		Years:            threeFiledYears(),
		HasPassportPhoto: false,
		HasFullAddress:   true,
	})

	if status.AllMet {
		t.Fatal("expected AllMet=false with no passport photograph")
	}

	t2 := findCondition(t, status, "T2")
	if t2.Met {
		t.Error("T2: want Met=false (no passport photograph)")
	}
	if t2.Note == "" {
		t.Error("T2: Note must explain what action is needed")
	}
}

// TestReadiness_MixedStates checks T1 when years are in mixed states.
// ACK_UPLOADED and CLEARED both qualify; ASSESSED does not.
func TestReadiness_MixedStates(t *testing.T) {
	years := []tcc.YearSummary{
		// Qualifies: has acknowledgement.
		{YearOfAssessment: 2024, State: returns.StateAckUploaded, AssessmentServed: true, BalancePaid: true},
		// Qualifies: cleared (no outstanding balance).
		{YearOfAssessment: 2025, State: returns.StateCleared, AssessmentServed: true, BalancePaid: true},
		// Does NOT qualify for T1: ASSESSED means the authority issued an
		// assessment but the taxpayer has not uploaded the acknowledgement.
		{YearOfAssessment: 2026, State: returns.StateAssessed, AssessmentServed: true, BalancePaid: false},
	}

	status := tcc.ComputeReadiness(tcc.ReadinessInput{
		Years:            years,
		HasPassportPhoto: true,
		HasFullAddress:   true,
	})

	if status.AllMet {
		t.Fatal("expected AllMet=false (2026 not yet filed/acknowledged)")
	}

	t1 := findCondition(t, status, "T1")
	if t1.Met {
		t.Error("T1: want Met=false — 2026 is ASSESSED, not ACK_UPLOADED or CLEARED")
	}

	t5 := findCondition(t, status, "T5")
	if t5.Met {
		t.Error("T5: want Met=false — 2026 has an outstanding balance")
	}
}
