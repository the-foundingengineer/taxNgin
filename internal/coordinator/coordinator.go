// Package coordinator derives cross-year fields for a validate.FilingContext
// from a taxpayer's projection history. It is the bridge between the event-
// sourced single-return view (internal/returns) and the multi-return signals
// that the validation catalogue needs (W105 prior-year income drop, W102
// employer vs. taxpayer gross mismatch, W106 years-on-file count).
//
// All derivations are pure: no I/O, no clock, no database.
package coordinator

import (
	"taxsoftware/internal/returns"
	"taxsoftware/internal/validate"
	"taxsoftware/taxengine"
)

// BuildFilingContext augments base with cross-year fields derived from target
// and history. The caller populates base with identity fields (TaxID, NIN,
// RelevantAuthority, Signed, Facts, etc.); BuildFilingContext fills in:
//
//   - PriorYearGrossKobo  (W105 — income-drop warning)
//   - EmployerGrossKobo   (W102 — employer/taxpayer mismatch)
//   - YearsOnFile         (W106 — TCC prerequisite counter)
//
// history may be nil or empty; the coordinator degrades gracefully, leaving
// the corresponding fields at their zero values so the warnings stay silent.
func BuildFilingContext(
	target *returns.Projection,
	history []*returns.Projection,
	base validate.FilingContext,
) validate.FilingContext {
	ctx := base

	// W105 — prior year gross.
	// Search history for the year immediately before the target. If found and
	// it had non-zero income, populate PriorYearGrossKobo. A nil-return prior
	// year (zero income) deliberately leaves PriorYearGrossKobo nil: going from
	// ₦0 to any income is not a suspicious drop and must not fire W105.
	priorYoA := target.YearOfAssessment - 1
	for _, h := range history {
		if h.YearOfAssessment != priorYoA {
			continue
		}
		var gross taxengine.Kobo
		for _, line := range h.Income {
			gross += taxengine.Kobo(line.AmountKobo)
		}
		if gross > 0 {
			ctx.PriorYearGrossKobo = &gross
		}
		break
	}

	// W102 — employer gross from accepted payroll proposals.
	// Sum GrossKobo across every proposal whose Status is "ACCEPTED". The
	// proposal retains the employer's originally-reported gross even if the
	// taxpayer later revises their declared income, which is exactly what
	// creates the W102 discrepancy.
	var employerGross taxengine.Kobo
	for _, pr := range target.ProposedPayroll {
		if pr.Status == "ACCEPTED" {
			employerGross += taxengine.Kobo(pr.GrossKobo)
		}
	}
	if employerGross > 0 {
		ctx.EmployerGrossKobo = &employerGross
	}

	// W106 — years on file.
	// Count history projections in a filed state (ACK_UPLOADED or CLEARED),
	// plus the target year if it is in one of those states.
	years := 0
	for _, h := range history {
		if isFiled(h.State) {
			years++
		}
	}
	if isFiled(target.State) {
		years++
	}
	ctx.YearsOnFile = years

	return ctx
}

func isFiled(s returns.State) bool {
	return s == returns.StateAckUploaded || s == returns.StateCleared
}
