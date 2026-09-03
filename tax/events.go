package returns

import (
	"encoding/json"
	"fmt"
	"time"
)

// EventKind is the closed set of things that can happen to a return.
//
// Every one of these is a FACT that occurred, named in the past tense. None of
// them is a command. The distinction matters: commands can be rejected, facts
// cannot be un-happened, and an append-only log of facts is what lets you
// answer "what did this taxpayer declare, when, and on what evidence" two years
// later when the authority asks.
type EventKind string

const (
	EvReturnOpened      EventKind = "ReturnOpened"
	EvIncomeDeclared    EventKind = "IncomeDeclared"
	EvIncomeRevised     EventKind = "IncomeRevised"
	EvIncomeRemoved     EventKind = "IncomeRemoved"
	EvReliefClaimed     EventKind = "ReliefClaimed"
	EvReliefRevised     EventKind = "ReliefRevised"
	EvReliefRemoved     EventKind = "ReliefRemoved"
	EvBaseDeclared      EventKind = "BaseDeclared" // e.g. annual rent paid
	EvEvidenceAttached  EventKind = "EvidenceAttached"
	EvEvidenceRemoved   EventKind = "EvidenceRemoved"

	// Payroll data is EVIDENCE. It becomes a declaration only when the taxpayer
	// accepts it, and the acceptance is a separate event with its own actor.
	EvPayrollProposed   EventKind = "PayrollRecordProposed"
	EvPayrollAccepted   EventKind = "PayrollRecordAccepted"
	EvPayrollRejected   EventKind = "PayrollRecordRejected"

	EvAssessmentComputed EventKind = "AssessmentComputed"
	EvValidationRun      EventKind = "ValidationRun"
	EvWarningAcknowledged EventKind = "WarningAcknowledged"

	EvReturnSigned      EventKind = "ReturnSigned"
	EvReturnUnsigned    EventKind = "ReturnUnsigned"
	EvPackIssued        EventKind = "PackIssued"

	EvFilingAttested    EventKind = "FilingAttested"
	EvAcknowledgementUploaded EventKind = "AcknowledgementUploaded"

	EvAssessmentServed  EventKind = "AssessmentServed"
	EvObjectionLodged   EventKind = "ObjectionLodged"
	EvObjectionWithdrawn EventKind = "ObjectionWithdrawn"
	EvPaymentRecorded   EventKind = "PaymentRecorded"
	EvReturnCleared     EventKind = "ReturnCleared"

	EvAmendmentOpened   EventKind = "AmendmentOpened"
	EvReturnAbandoned   EventKind = "ReturnAbandoned"
)

type ActorKind string

const (
	ActorTaxpayer ActorKind = "TAXPAYER"
	ActorEmployer ActorKind = "EMPLOYER"
	ActorSystem   ActorKind = "SYSTEM"
	ActorStaff    ActorKind = "STAFF"
)

// Actor records who caused the fact. Under Shape A this matters more than it
// looks: the taxpayer carries the liability, so any figure in a filed return
// must trace to a TAXPAYER-actor event. A return whose income was declared
// entirely by SYSTEM or EMPLOYER actors is a bug, and it is detectable by
// walking the log.
type Actor struct {
	Kind ActorKind `json:"kind"`
	ID   string    `json:"id"`
}

type Event struct {
	Seq        int             `json:"seq"`
	ReturnID   string          `json:"return_id"`
	Kind       EventKind       `json:"kind"`
	Payload    json.RawMessage `json:"payload"`
	Actor      Actor           `json:"actor"`
	OccurredAt time.Time       `json:"occurred_at"`

	// IdempotencyKey deduplicates a retried request. Nigerian mobile networks
	// drop connections constantly and users hammer submit buttons. Without this
	// a flaky connection produces two IncomeDeclared events and a doubled salary.
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// ---------------------------------------------------------------------------
// Projection
// ---------------------------------------------------------------------------

// Projection is the current state of a return, rebuilt by folding its events.
// It is a cache, never a source of truth. Delete the whole projections table
// and it rebuilds from the log; delete the log and you have lost the return.
type Projection struct {
	ReturnID         string
	TaxpayerID       string
	YearOfAssessment int
	BasisPeriodStart time.Time
	BasisPeriodEnd   time.Time
	Jurisdiction     string

	State   State
	Version int // = last folded Seq. Optimistic concurrency token.

	Income  map[string]IncomeLine
	Reliefs map[string]ReliefLine
	Bases   map[string]int64 // kobo

	ProposedPayroll map[string]PayrollProposal

	Assessment       *AssessmentSnapshot
	AssessmentStale  bool
	RuleSetVersion   string
	RuleSetPublished bool
	PortalMapVersion string

	BlockingErrors []string
	Warnings       []string
	AcknowledgedWarnings map[string]bool

	SignedAt           *time.Time
	PackIssuedAt       *time.Time
	AttestedAt         *time.Time
	AckUploadedAt      *time.Time
	AssessmentServedAt *time.Time
	ObjectionDeadline  *time.Time
	PaidAt             *time.Time

	// DeclaredByTaxpayer is false if any income line entered the return without
	// a TAXPAYER-actor event behind it. Signing is blocked when it is false.
	DeclaredByTaxpayer bool
}

type IncomeLine struct {
	ID         string
	Category   string
	AmountKobo int64
	SourceRef  string
	Label      string
	DeclaredBy ActorKind
	EvidenceIDs []string
}

type ReliefLine struct {
	Code        string
	AmountKobo  int64
	EvidenceIDs []string
}

type PayrollProposal struct {
	RecordID   string
	EmployerID string
	GrossKobo  int64
	PayeKobo   int64
	Status     string // PROPOSED | ACCEPTED | REJECTED
}

type AssessmentSnapshot struct {
	FactsHash            string
	RuleSetVersion       string
	RuleSetDigest        string
	ChargeableIncomeKobo int64
	TaxPayableKobo       int64
	TotalCreditsKobo     int64
	BalanceKobo          int64
	ComputedAt           time.Time
}

// Fold applies one event. It is the only place a Projection changes.
//
// Two rules hold here and nowhere else in the codebase:
//   1. Fold never fails on an unknown event kind. A projection that panics on a
//      kind added by a newer deploy takes down every read during a rolling
//      release. Unknown kinds bump the version and are otherwise ignored.
//   2. Fold never performs I/O. Rebuilding a projection must be a pure walk over
//      the log, or replay is not replay.
func (p *Projection) Fold(e Event) error {
	if e.Seq != p.Version+1 {
		return fmt.Errorf("returns: out-of-order event: got seq %d, expected %d", e.Seq, p.Version+1)
	}
	p.Version = e.Seq

	switch e.Kind {
	case EvReturnOpened:
		var v struct {
			TaxpayerID       string    `json:"taxpayer_id"`
			YearOfAssessment int       `json:"year_of_assessment"`
			BasisPeriodStart time.Time `json:"basis_period_start"`
			BasisPeriodEnd   time.Time `json:"basis_period_end"`
			Jurisdiction     string    `json:"jurisdiction"`
		}
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		p.TaxpayerID = v.TaxpayerID
		p.YearOfAssessment = v.YearOfAssessment
		p.BasisPeriodStart = v.BasisPeriodStart
		p.BasisPeriodEnd = v.BasisPeriodEnd
		p.Jurisdiction = v.Jurisdiction
		p.State = StateDraft
		p.Income = map[string]IncomeLine{}
		p.Reliefs = map[string]ReliefLine{}
		p.Bases = map[string]int64{}
		p.ProposedPayroll = map[string]PayrollProposal{}
		p.AcknowledgedWarnings = map[string]bool{}
		p.DeclaredByTaxpayer = true // vacuously true until income arrives

	case EvIncomeDeclared, EvIncomeRevised:
		var v IncomeLine
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		v.DeclaredBy = e.Actor.Kind
		p.Income[v.ID] = v
		p.recomputeDeclaredBy()
		p.markStale()

	case EvIncomeRemoved:
		var v struct{ ID string `json:"id"` }
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		delete(p.Income, v.ID)
		p.recomputeDeclaredBy()
		p.markStale()

	case EvReliefClaimed, EvReliefRevised:
		var v ReliefLine
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		p.Reliefs[v.Code] = v
		p.markStale()

	case EvReliefRemoved:
		var v struct{ Code string `json:"code"` }
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		delete(p.Reliefs, v.Code)
		p.markStale()

	case EvBaseDeclared:
		var v struct {
			Name       string `json:"name"`
			AmountKobo int64  `json:"amount_kobo"`
		}
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		p.Bases[v.Name] = v.AmountKobo
		p.markStale()

	case EvPayrollProposed:
		var v PayrollProposal
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		v.Status = "PROPOSED"
		p.ProposedPayroll[v.RecordID] = v
		// Deliberately does NOT touch p.Income and does NOT mark the assessment
		// stale. An employer's assertion changes nothing about the return until
		// the taxpayer accepts it.

	case EvPayrollAccepted:
		var v struct {
			RecordID string     `json:"record_id"`
			Income   IncomeLine `json:"income"`
		}
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		if pr, ok := p.ProposedPayroll[v.RecordID]; ok {
			pr.Status = "ACCEPTED"
			p.ProposedPayroll[v.RecordID] = pr
		}
		v.Income.DeclaredBy = e.Actor.Kind // must be TAXPAYER
		p.Income[v.Income.ID] = v.Income
		p.recomputeDeclaredBy()
		p.markStale()

	case EvPayrollRejected:
		var v struct{ RecordID string `json:"record_id"` }
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		if pr, ok := p.ProposedPayroll[v.RecordID]; ok {
			pr.Status = "REJECTED"
			p.ProposedPayroll[v.RecordID] = pr
		}

	case EvEvidenceAttached:
		var v struct {
			Target     string `json:"target"` // "income:<id>" | "relief:<code>"
			EvidenceID string `json:"evidence_id"`
		}
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		p.attachEvidence(v.Target, v.EvidenceID)

	case EvAssessmentComputed:
		var v AssessmentSnapshot
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		p.Assessment = &v
		p.RuleSetVersion = v.RuleSetVersion
		p.AssessmentStale = false

	case EvValidationRun:
		var v struct {
			Blocking []string `json:"blocking"`
			Warnings []string `json:"warnings"`
		}
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		p.BlockingErrors = v.Blocking
		p.Warnings = v.Warnings
		if len(v.Blocking) == 0 && p.State == StateDraft {
			p.State = StateValidated
		}

	case EvWarningAcknowledged:
		var v struct{ Code string `json:"code"` }
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		p.AcknowledgedWarnings[v.Code] = true

	case EvReturnSigned:
		t := e.OccurredAt
		p.SignedAt = &t
		p.State = StateSigned

	case EvReturnUnsigned:
		p.SignedAt = nil
		p.State = StateValidated

	case EvPackIssued:
		var v struct{ PortalMapVersion string `json:"portal_map_version"` }
		_ = json.Unmarshal(e.Payload, &v)
		p.PortalMapVersion = v.PortalMapVersion
		t := e.OccurredAt
		p.PackIssuedAt = &t
		p.State = StatePackIssued

	case EvFilingAttested:
		t := e.OccurredAt
		p.AttestedAt = &t
		p.State = StateSelfAttested

	case EvAcknowledgementUploaded:
		t := e.OccurredAt
		p.AckUploadedAt = &t
		p.State = StateAckUploaded

	case EvAssessmentServed:
		var v struct {
			ServedAt   time.Time `json:"served_at"`
			WindowDays int       `json:"objection_window_days"`
		}
		if err := json.Unmarshal(e.Payload, &v); err != nil {
			return err
		}
		p.AssessmentServedAt = &v.ServedAt
		d := ObjectionDeadline(v.ServedAt, v.WindowDays)
		p.ObjectionDeadline = &d
		p.State = StateAssessed

	case EvObjectionLodged:
		p.State = StateObjected

	case EvObjectionWithdrawn:
		p.State = StateAssessed

	case EvPaymentRecorded:
		t := e.OccurredAt
		p.PaidAt = &t
		p.State = StatePaid

	case EvReturnCleared:
		p.State = StateCleared

	case EvAmendmentOpened:
		p.State = StateSuperseded

	case EvReturnAbandoned:
		p.State = StateAbandoned

	default:
		// Rule 1. Unknown kinds are tolerated, not fatal.
	}
	return nil
}

func (p *Projection) markStale() {
	if p.Assessment != nil {
		p.AssessmentStale = true
	}
}

func (p *Projection) recomputeDeclaredBy() {
	p.DeclaredByTaxpayer = true
	for _, l := range p.Income {
		if l.DeclaredBy != ActorTaxpayer {
			p.DeclaredByTaxpayer = false
			return
		}
	}
}

func (p *Projection) attachEvidence(target, evidenceID string) {
	const inc, rel = "income:", "relief:"
	switch {
	case len(target) > len(inc) && target[:len(inc)] == inc:
		id := target[len(inc):]
		if l, ok := p.Income[id]; ok {
			l.EvidenceIDs = append(l.EvidenceIDs, evidenceID)
			p.Income[id] = l
		}
	case len(target) > len(rel) && target[:len(rel)] == rel:
		code := target[len(rel):]
		if l, ok := p.Reliefs[code]; ok {
			l.EvidenceIDs = append(l.EvidenceIDs, evidenceID)
			p.Reliefs[code] = l
		}
	}
}

// Replay rebuilds a projection from scratch. Used on cold read, after a
// projection schema change, and in every test.
func Replay(events []Event) (*Projection, error) {
	p := &Projection{}
	for _, e := range events {
		if err := p.Fold(e); err != nil {
			return nil, fmt.Errorf("returns: replay failed at seq %d: %w", e.Seq, err)
		}
	}
	return p, nil
}
