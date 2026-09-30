package employer

import (
	"encoding/json"
	"fmt"
	"time"

	"taxsoftware/internal/returns"
)

// ProposalEvent builds the EvPayrollProposed event to append to a taxpayer's
// return log when a payroll record is matched. The actor is set to
// ActorEmployer — the proposal is never a taxpayer declaration, which is
// exactly what keeps DeclaredByTaxpayer true until the taxpayer explicitly
// accepts.
//
// The payload carries only the fields Fold needs (record_id, employer_id,
// gross_kobo, paye_kobo). The full PayrollRecord stays on the employer's
// import batch, not in the return's event log.
func ProposalEvent(rec PayrollRecord, emp Employer, returnID string, seq int, occurredAt time.Time) returns.Event {
	// Marshal a returns.PayrollProposal struct (not a snake_case map) so the
	// key names produced match the field names that Fold's json.Unmarshal expects.
	// PayrollProposal has no json tags, so Go uses capital-case field names as keys.
	payload, _ := json.Marshal(returns.PayrollProposal{
		RecordID:   rec.ID,
		EmployerID: emp.ID,
		GrossKobo:  int64(rec.GrossKobo),
		PayeKobo:   int64(rec.PAYEKobo),
	})
	return returns.Event{
		Seq:        seq,
		ReturnID:   returnID,
		Kind:       returns.EvPayrollProposed,
		Payload:    json.RawMessage(payload),
		Actor:      returns.Actor{Kind: returns.ActorEmployer, ID: emp.ID},
		OccurredAt: occurredAt,
	}
}

// AcceptanceEvent builds the EvPayrollAccepted event. The caller constructs
// the IncomeLine (choosing the Label, mapping GrossKobo to the declared amount)
// and provides the actor that is performing the acceptance.
//
// AcceptanceEvent returns an error if actor.Kind is not ActorTaxpayer. This is
// a structural guardrail: it makes it impossible to write an auto-accept path
// that silently bypasses the DeclaredByTaxpayer guard. The moment the actor
// is not TAXPAYER, signing will be blocked — but it is better to catch it here
// before any event is emitted.
func AcceptanceEvent(
	recID string,
	line returns.IncomeLine,
	returnID string,
	seq int,
	actor returns.Actor,
	occurredAt time.Time,
) (returns.Event, error) {
	if actor.Kind != returns.ActorTaxpayer {
		return returns.Event{}, fmt.Errorf(
			"employer: AcceptanceEvent requires %s actor; got %s — "+
				"auto-accept bypasses the DeclaredByTaxpayer signing guard and is not permitted",
			returns.ActorTaxpayer, actor.Kind)
	}
	payload, err := json.Marshal(struct {
		RecordID string             `json:"record_id"`
		Income   returns.IncomeLine `json:"income"`
	}{RecordID: recID, Income: line})
	if err != nil {
		return returns.Event{}, fmt.Errorf("employer: marshal acceptance payload: %w", err)
	}
	return returns.Event{
		Seq:        seq,
		ReturnID:   returnID,
		Kind:       returns.EvPayrollAccepted,
		Payload:    json.RawMessage(payload),
		Actor:      actor,
		OccurredAt: occurredAt,
	}, nil
}
