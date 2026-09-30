// Package tcc computes Tax Clearance Certificate readiness across a
// taxpayer's returns. It is pure: no I/O, no database, no clock. The caller
// supplies year summaries and profile flags; ComputeReadiness returns the
// six T1–T6 conditions that the LIRS requires before issuing a TCC.
//
// The TCC is the thing taxpayers actually want. Showing them a live checklist
// of what is missing — not a generic "come back later" — is the best retention
// surface in the product. A user two years into filing with you is looking at
// a checklist that only needs one more line. They will not leave.
package tcc

import (
	"fmt"

	"taxsoftware/internal/returns"
)

// YearSummary condenses one year of assessment to the fields TCC readiness
// needs. The caller populates it from wherever the returns are stored.
type YearSummary struct {
	YearOfAssessment int
	// State is the current lifecycle state of the return for this year.
	State returns.State
	// AssessmentServed is true when the authority has issued an assessment
	// for this year (Projection.AssessmentServedAt != nil).
	AssessmentServed bool
	// BalancePaid is true when the balance for this year has been settled
	// (Projection.PaidAt != nil, or State is PAID or CLEARED).
	BalancePaid bool
}

// ReadinessInput is everything ComputeReadiness needs to evaluate T1–T6.
type ReadinessInput struct {
	// Years holds the available year summaries, in any order. TCC requires
	// three consecutive years; if fewer than three are provided, T1 and T4
	// will reflect that.
	Years []YearSummary
	// HasPassportPhoto is true when the taxpayer has uploaded a recent passport
	// photograph to their eTax profile. Drives T2.
	HasPassportPhoto bool
	// HasFullAddress is true when the taxpayer's full residential address is
	// present on the eTax profile. Drives T3 and T6.
	HasFullAddress bool
}

// Condition is one of the six TCC requirements.
type Condition struct {
	ID    string // "T1" through "T6"
	Label string
	Met   bool
	// Note is empty when Met is true. When Met is false it contains a brief,
	// user-facing explanation of what is missing.
	Note string
}

// TCCStatus is the result of a readiness check.
type TCCStatus struct {
	// Conditions always has six entries in T1–T6 order.
	Conditions []Condition
	AllMet     bool
}

// ComputeReadiness evaluates the six TCC conditions and returns the status.
// It never returns an error: missing data surfaces as a condition with Met=false.
func ComputeReadiness(in ReadinessInput) TCCStatus {
	var status TCCStatus

	// T1 — Annual returns filed for the last three years.
	// "Filed" means the taxpayer has uploaded the acknowledgement receipt
	// (ACK_UPLOADED) or the return has been fully cleared (CLEARED). A return
	// sitting in SELF_ATTESTED is a claim, not evidence.
	filedCount := 0
	for _, y := range in.Years {
		if y.State == returns.StateAckUploaded || y.State == returns.StateCleared {
			filedCount++
		}
	}
	status.Conditions = append(status.Conditions, condition(
		"T1",
		"Annual returns filed for the last three years, both employer-filed and individual-filed",
		filedCount >= 3,
		fmt.Sprintf("%d of 3 required years have acknowledgement receipts uploaded", filedCount),
	))

	// T2 — Passport photograph uploaded to the eTax profile.
	status.Conditions = append(status.Conditions, condition(
		"T2",
		"Recent passport photograph uploaded to the eTax profile",
		in.HasPassportPhoto,
		"Upload a recent passport photograph via the eTax profile settings",
	))

	// T3 — Full address updated on the eTax profile.
	status.Conditions = append(status.Conditions, condition(
		"T3",
		"Full address updated",
		in.HasFullAddress,
		"Update your full residential address on the eTax profile",
	))

	// T4 — Tax assessments received for the last three years.
	servedCount := 0
	for _, y := range in.Years {
		if y.AssessmentServed {
			servedCount++
		}
	}
	status.Conditions = append(status.Conditions, condition(
		"T4",
		"Tax assessments received for the last three years",
		servedCount >= 3,
		fmt.Sprintf("%d of 3 required years have received assessments from LIRS", servedCount),
	))

	// T5 — All taxes due fully paid.
	// Met when every provided year has BalancePaid=true. A zero-year input
	// is not met — there is nothing to clear against.
	allPaid := len(in.Years) > 0
	for _, y := range in.Years {
		if !y.BalancePaid {
			allPaid = false
			break
		}
	}
	status.Conditions = append(status.Conditions, condition(
		"T5",
		"All taxes due fully paid",
		allPaid,
		"One or more years have an outstanding balance; pay through the LIRS eTax portal",
	))

	// T6 — Profile complete. The portal equates this with a full address,
	// matching the T3 signal. They are listed separately in the TCC checklist
	// but track the same user action.
	status.Conditions = append(status.Conditions, condition(
		"T6",
		"Profile complete",
		in.HasFullAddress,
		"Complete your eTax profile: full residential address, phone number, and email",
	))

	// AllMet is true only when every condition is satisfied.
	status.AllMet = true
	for _, c := range status.Conditions {
		if !c.Met {
			status.AllMet = false
			break
		}
	}

	return status
}

func condition(id, label string, met bool, failNote string) Condition {
	c := Condition{ID: id, Label: label, Met: met}
	if !met {
		c.Note = failNote
	}
	return c
}
