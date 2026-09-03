package returns

import (
	"fmt"
	"time"
)

// State is where a return sits in its life.
//
// The path from SUBMITTED onwards is the Shape A tax. Because the taxpayer
// files personally on the state portal, nothing tells us it worked. There is no
// callback, no acknowledgement webhook, no agent session we can read. So the
// return passes through SELF_ATTESTED — the user says they filed — and then
// ACK_UPLOADED, where they hand us the portal's receipt and we have evidence
// rather than a claim.
//
// Resist the temptation to collapse those two. The gap between them is a real
// thing that happens to real users: they mean to file, get interrupted, and
// tell you they did. If those are the same state you cannot tell an attested
// return from a filed one, and TCC readiness becomes a guess.
type State string

const (
	StateDraft        State = "DRAFT"
	StateValidated    State = "VALIDATED"
	StateSigned       State = "SIGNED"
	StatePackIssued   State = "PACK_ISSUED"
	StateSelfAttested State = "SELF_ATTESTED"
	StateAckUploaded  State = "ACK_UPLOADED"
	StateAssessed     State = "ASSESSED"
	StatePaid         State = "PAID"
	StateCleared      State = "CLEARED"

	StateObjected  State = "OBJECTED"
	StateSuperseded State = "SUPERSEDED" // replaced by an amendment
	StateAbandoned State = "ABANDONED"
)

// Transition names the edge, not just the destination, so the event log records
// what the user did rather than only where they ended up.
type Transition string

const (
	TrValidate   Transition = "VALIDATE"
	TrInvalidate Transition = "INVALIDATE" // edited after validation
	TrSign       Transition = "SIGN"
	TrUnsign     Transition = "UNSIGN"
	TrIssuePack  Transition = "ISSUE_PACK"
	TrAttest     Transition = "ATTEST_FILED"
	TrUploadAck  Transition = "UPLOAD_ACK"
	TrRecordAssessment Transition = "RECORD_ASSESSMENT"
	TrRecordPayment    Transition = "RECORD_PAYMENT"
	TrClear      Transition = "CLEAR"
	TrObject     Transition = "OBJECT"
	TrWithdrawObjection Transition = "WITHDRAW_OBJECTION"
	TrAmend      Transition = "AMEND"
	TrAbandon    Transition = "ABANDON"
)

type edge struct {
	from Transition
	src  State
	dst  State
}

// transitions is the whole machine. If an edge is not here it cannot happen,
// and Apply returns an error naming both states. A tax return that can reach an
// arbitrary state is a tax return you cannot reason about.
var transitions = []edge{
	{TrValidate, StateDraft, StateValidated},
	{TrInvalidate, StateValidated, StateDraft},
	{TrInvalidate, StateSigned, StateDraft},

	{TrSign, StateValidated, StateSigned},
	{TrUnsign, StateSigned, StateValidated},

	{TrIssuePack, StateSigned, StatePackIssued},
	{TrInvalidate, StatePackIssued, StateDraft},

	{TrAttest, StatePackIssued, StateSelfAttested},
	{TrUploadAck, StateSelfAttested, StateAckUploaded},
	// A user who kept their receipt can skip straight past attestation.
	{TrUploadAck, StatePackIssued, StateAckUploaded},

	{TrRecordAssessment, StateSelfAttested, StateAssessed},
	{TrRecordAssessment, StateAckUploaded, StateAssessed},

	{TrObject, StateAssessed, StateObjected},
	{TrWithdrawObjection, StateObjected, StateAssessed},

	{TrRecordPayment, StateAssessed, StatePaid},
	{TrClear, StatePaid, StateCleared},
	// A nil or fully-credited return owes nothing and clears without payment.
	{TrClear, StateAssessed, StateCleared},

	{TrAmend, StateAckUploaded, StateSuperseded},
	{TrAmend, StateAssessed, StateSuperseded},
	{TrAmend, StatePaid, StateSuperseded},
	{TrAmend, StateCleared, StateSuperseded},

	{TrAbandon, StateDraft, StateAbandoned},
	{TrAbandon, StateValidated, StateAbandoned},
}

// Guard is a precondition on a transition. Guards read the projection; they do
// not perform I/O, so the machine stays testable.
type Guard func(r *Projection) error

var guards = map[Transition]Guard{
	TrValidate: func(r *Projection) error {
		if len(r.BlockingErrors) > 0 {
			return fmt.Errorf("returns: %d blocking validation errors remain (first: %s)",
				len(r.BlockingErrors), r.BlockingErrors[0])
		}
		return nil
	},
	TrSign: func(r *Projection) error {
		if r.Assessment == nil {
			return fmt.Errorf("returns: cannot sign a return with no computed assessment")
		}
		if r.AssessmentStale {
			return fmt.Errorf("returns: facts changed since the assessment was computed; recompute before signing")
		}
		if !r.RuleSetPublished {
			return fmt.Errorf("returns: cannot sign against an unpublished rule set")
		}
		return nil
	},
	TrIssuePack: func(r *Projection) error {
		if r.SignedAt == nil {
			return fmt.Errorf("returns: cannot issue a pack for an unsigned return")
		}
		if r.PortalMapVersion == "" {
			return fmt.Errorf("returns: no portal map loaded for jurisdiction %s", r.Jurisdiction)
		}
		return nil
	},
	TrAttest: func(r *Projection) error {
		if r.PackIssuedAt == nil {
			return fmt.Errorf("returns: cannot attest to filing before a pack was issued")
		}
		return nil
	},
	TrObject: func(r *Projection) error {
		if r.AssessmentServedAt == nil {
			return fmt.Errorf("returns: no assessment has been served")
		}
		return nil
	},
}

// Apply validates a transition against the machine and its guard. It does not
// mutate anything — the caller emits the corresponding event, and the
// projection advances when the event is folded in. Keeping validation and
// mutation apart is what lets you replay the log.
func Apply(cur State, t Transition, p *Projection) (State, error) {
	var dst State
	found := false
	for _, e := range transitions {
		if e.from == t && e.src == cur {
			dst, found = e.dst, true
			break
		}
	}
	if !found {
		return cur, fmt.Errorf("returns: transition %s is not permitted from state %s", t, cur)
	}
	if g, ok := guards[t]; ok {
		if err := g(p); err != nil {
			return cur, err
		}
	}
	return dst, nil
}

// ObjectionDeadline returns the last day to object to a served assessment.
// The window is 30 days under NTAA s.34, after which the assessment becomes
// binding even where it overstates the liability.
//
// The window is passed in from the rule set rather than hardcoded, and the
// clock is passed in rather than read, so this stays pure.
func ObjectionDeadline(servedAt time.Time, windowDays int) time.Time {
	return servedAt.AddDate(0, 0, windowDays)
}

// IsTerminal reports whether a return can still change.
func (s State) IsTerminal() bool {
	switch s {
	case StateCleared, StateSuperseded, StateAbandoned:
		return true
	}
	return false
}
