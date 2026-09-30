// Package support provides the redacted view of a return that is safe to show
// a support agent without exposing financial data or personal identifiers.
//
// Every field on SupportView carries a comment explaining why it is safe for
// an agent to see. This is the audit trail for NDPA data minimisation review.
// A support agent must be able to diagnose a return's status without seeing
// any salary, credit, or relief amount, and without seeing NIN, BVN, Tax ID,
// or employer names.
package support

import (
	"sort"
	"time"

	"taxsoftware/internal/returns"
)

// SupportView is the redacted view of a return projection. It contains enough
// to diagnose and explain a return's status to a support agent without exposing
// any financial data or personal identifiers.
//
// This type must never grow a money field. Any request to add one must be
// rejected and escalated — a support agent who sees salary data is an NDPA
// liability and an insider-threat surface.
type SupportView struct {
	// ReturnID identifies the specific case. It is a system-generated opaque
	// string; no financial data or identity information is encoded in it.
	ReturnID string

	// YearOfAssessment is the tax year being assessed (e.g. 2026). It is a
	// calendar year integer, not an income amount.
	YearOfAssessment int

	// Jurisdiction is the tax authority code (e.g. "NG" for LIRS). It tells
	// the agent which portal and procedures apply; contains no taxpayer data.
	Jurisdiction string

	// State is the current lifecycle label (e.g. DRAFT, SIGNED, CLEARED). It
	// is a domain code string that contains no amounts or identity fields.
	State returns.State

	// Version is the event log sequence number of the last folded event. It
	// helps agents and engineers diagnose replay consistency issues.
	Version int

	// AssessmentStale is true when income or reliefs changed after the last
	// assessment was computed. A flag only; no amount is implied or present.
	AssessmentStale bool

	// RuleSetPublished is true when the engine rule set used for the most
	// recent assessment has been formally published by the authority. A flag
	// only; no version string or financial content.
	RuleSetPublished bool

	// IncomeCategoryCount records how many income lines exist per category
	// (e.g. {"EMPLOYMENT": 2, "DIVIDEND": 1}). Keys are domain code strings
	// (not employer names); values are counts (not amounts).
	IncomeCategoryCount map[string]int

	// HasCreditClaimed is true when at least one tax credit (e.g. PAYE) is
	// recorded on the latest assessment. The credit amount is withheld.
	HasCreditClaimed bool

	// HasReliefClaimed is true when at least one relief line exists in the
	// return. The relief codes and amounts are withheld.
	HasReliefClaimed bool

	// BlockingIssueCodes are the validation error codes (e.g. "E001", "E005")
	// that prevent the return from progressing. Codes are domain identifiers;
	// they contain no amounts, income figures, or taxpayer identifiers.
	BlockingIssueCodes []string

	// WarningCodes are advisory validation codes. Same safety rationale as
	// BlockingIssueCodes: codes only, no amounts.
	WarningCodes []string

	// PayrollProposalCount is the total number of employer payroll proposals
	// attached to this return. Count only; no employer names or gross amounts.
	PayrollProposalCount int

	// PayrollAcceptedCount is the number of proposals the taxpayer accepted.
	// Count only; no amounts or employer identifiers.
	PayrollAcceptedCount int

	// PayrollRejectedCount is the number of proposals the taxpayer rejected.
	// Count only; no amounts or employer identifiers.
	PayrollRejectedCount int

	// PackIssued is true when the submission pack has been generated for this
	// return. A boolean flag; the pack content is not included.
	PackIssued bool

	// Filed is true when the taxpayer has attested that they submitted the
	// return on the state portal. A boolean flag; no date or amount exposed.
	Filed bool

	// AckUploaded is true when the taxpayer uploaded the portal acknowledgement
	// receipt. A boolean flag; the receipt content is not included.
	AckUploaded bool

	// Assessed is true when the authority has served a formal assessment. A
	// boolean flag; the assessed amount is withheld.
	Assessed bool

	// Paid is true when a payment has been recorded against the return. A
	// boolean flag; the paid amount is withheld.
	Paid bool

	// Timeline records the state transitions in ascending chronological order.
	// Each entry carries a state label and a timestamp only; no amounts or
	// identity fields.
	Timeline []SupportTimelineEntry
}

// SupportTimelineEntry is one observed state transition in the return's history.
type SupportTimelineEntry struct {
	// State is the lifecycle state reached at this point in time.
	State returns.State
	// OccurredAt is when the transition was recorded.
	OccurredAt time.Time
}

// BuildSupportView constructs the redacted view from a Projection. The
// BlockingIssueCodes and WarningCodes are taken directly from the Projection
// where they already exist as plain code strings with no financial content.
func BuildSupportView(p *returns.Projection) SupportView {
	sv := SupportView{
		ReturnID:         p.ReturnID,
		YearOfAssessment: p.YearOfAssessment,
		Jurisdiction:     p.Jurisdiction,
		State:            p.State,
		Version:          p.Version,
		AssessmentStale:  p.AssessmentStale,
		RuleSetPublished: p.RuleSetPublished,
	}

	sv.IncomeCategoryCount = make(map[string]int)
	for _, line := range p.Income {
		sv.IncomeCategoryCount[line.Category]++
	}

	if p.Assessment != nil {
		sv.HasCreditClaimed = p.Assessment.TotalCreditsKobo != 0
	}
	sv.HasReliefClaimed = len(p.Reliefs) > 0

	sv.BlockingIssueCodes = append([]string(nil), p.BlockingErrors...)
	sv.WarningCodes = append([]string(nil), p.Warnings...)

	for _, pr := range p.ProposedPayroll {
		sv.PayrollProposalCount++
		switch pr.Status {
		case "ACCEPTED":
			sv.PayrollAcceptedCount++
		case "REJECTED":
			sv.PayrollRejectedCount++
		}
	}

	sv.PackIssued = p.PackIssuedAt != nil
	sv.Filed = p.AttestedAt != nil
	sv.AckUploaded = p.AckUploadedAt != nil
	sv.Assessed = p.AssessmentServedAt != nil
	sv.Paid = p.PaidAt != nil

	var entries []SupportTimelineEntry
	addEntry := func(state returns.State, t *time.Time) {
		if t != nil {
			entries = append(entries, SupportTimelineEntry{State: state, OccurredAt: *t})
		}
	}
	addEntry(returns.StateSigned, p.SignedAt)
	addEntry(returns.StatePackIssued, p.PackIssuedAt)
	addEntry(returns.StateSelfAttested, p.AttestedAt)
	addEntry(returns.StateAckUploaded, p.AckUploadedAt)
	addEntry(returns.StateAssessed, p.AssessmentServedAt)
	addEntry(returns.StatePaid, p.PaidAt)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].OccurredAt.Before(entries[j].OccurredAt)
	})
	sv.Timeline = entries

	return sv
}
