TASK-006 — Employer Channel: Payroll Import, HMAC Matching, and Claim Flow

## Context (read this first — you have no prior conversation history)

This repo is the backend for a Nigerian personal income tax prep product (Shape A:
self-preparation). TASKs 001–005 built the engine, the golden corpus, the lifecycle
integration, the submission pack, the TCC readiness projection, and the multi-year
registry and filing coordinator.

The employer channel is the most important pre-fill mechanism in the product and the
one that drives W102. Under Shape A, the taxpayer declares their own income and
carries the liability — but most employed Nigerians receive a payslip and have PAYE
remitted by their employer. If the product can show the taxpayer their employer's
pre-filled figures and let them accept, edit, or reject each line, the filing
experience is dramatically better than a blank form.

The event catalogue (`internal/returns/events.go`) already models the outcome side:
`EvPayrollProposed`, `EvPayrollAccepted`, `EvPayrollRejected`. What does not exist
is the **input side**: the types that represent an employer, an import batch, and an
individual payroll record, plus the matching logic that links a payroll record to a
taxpayer's return.

Read in full before touching code:

- `tasks/TASK-005-registry-coordinator-and-w105.md` (W102 integration in coordinator)
- `tax/02-engine-and-employer.md` (§6 in full — employer channel design)
- `internal/returns/events.go` (EvPayrollProposed, EvPayrollAccepted, EvPayrollRejected,
  PayrollProposal, Projection.ProposedPayroll)
- `internal/returns/lifecycle.go` (DeclaredByTaxpayer guard — the architectural stake)
- `internal/validate/validate.go` (W102 — what BuildFilingContext feeds it)

## Prerequisite: TASK-005 must be done

`go test ./... -count=1` must pass including the coordinator tests. The employer
channel produces payroll events that the lifecycle integration already tests; the
coordinator (from TASK-005) drives the W102 wire-up. If those packages do not exist,
finish TASK-005 first.

## What "done" looks like

### 1. `internal/employer/types.go` — Domain types

New package. Imports only the standard library plus `taxengine` (for `Kobo` and
`IncomeCategory`). No I/O, no YAML dependency.

```go
// Employer is the aggregate root for a company that submits payroll data.
type Employer struct {
    ID          string    // UUID, assigned at onboarding
    CACNumber   string    // Corporate Affairs Commission registration
    TaxID       string    // employer's own tax identifier
    LegalName   string
    Jurisdictions []string  // states where this employer operates
    DPASignedAt *time.Time // nil until the Data Processing Agreement is signed
}

// EmploymentRelationship is the join between an employer and a taxpayer.
// Until a taxpayer claims a payroll record, TaxpayerID is empty.
type EmploymentRelationship struct {
    ID              string
    EmployerID      string
    TaxpayerID      string    // empty until claimed
    EmployeeRef     string    // employer's own staff number
    StartDate       time.Time
    EndDate         *time.Time // nil = current employment
    MatchConfidence string    // "NIN_EXACT" | "TAXID_EXACT" | "EMAIL" | "MANUAL"
}

// ImportStatus is the lifecycle of a payroll batch.
type ImportStatus string

const (
    ImportReceived  ImportStatus = "RECEIVED"
    ImportParsed    ImportStatus = "PARSED"
    ImportValidated ImportStatus = "VALIDATED"
    ImportPublished ImportStatus = "PUBLISHED"
)

// PayrollImport is one batch of payroll data from one employer for one year.
type PayrollImport struct {
    ID                string
    EmployerID        string
    YearOfAssessment  int
    Source            string // "CSV" | "H1_EXPORT" | "API"
    UploadedBy        string // staff or API credential ID
    UploadedAt        time.Time
    Status            ImportStatus
    Records           []PayrollRecord
}

// PayrollRecord is EVIDENCE, not a declaration. It becomes a declaration only
// when the taxpayer accepts it (EvPayrollAccepted with TAXPAYER actor). Storing
// identifiers as HMAC bytes lets the match work without persisting plaintext NINs.
type PayrollRecord struct {
    ID              string
    ImportID        string
    EmployeeRef     string
    NINHMACSuffix   []byte // deterministic HMAC — only used for matching, never for display
    TaxIDHMACSuffix []byte
    EmailHMACSuffix []byte
    GrossKobo       taxengine.Kobo
    PAYEKobo        taxengine.Kobo
    PensionKobo     taxengine.Kobo
    NHFKobo         taxengine.Kobo
    NHISKobo        taxengine.Kobo
    BasicKobo       taxengine.Kobo // component of disputed pension base
    HousingKobo     taxengine.Kobo
    TransportKobo   taxengine.Kobo
    ClaimedBy       string    // taxpayer ID, empty until claimed
    ClaimedAt       *time.Time
}
```

Keep `BasicKobo`, `HousingKobo`, `TransportKobo` separately: the pension base
ambiguity (gross vs. basic + housing + transport) cannot be resolved later if these
are collapsed. The design doc is explicit on this.

### 2. `internal/employer/hmac.go` — Identifier matching

The NDPA prohibits holding strangers' NINs in plaintext in the payroll table.
Matching by NIN is done by comparing HMACs of the candidate NIN against the stored
HMAC suffix.

```go
// ComputeHMAC returns a deterministic HMAC-SHA256 of value under key.
// The same key must be used consistently across the entire system — if the key
// rotates, all stored HMACs must be recomputed. Store the key in config, never
// in code.
func ComputeHMAC(key []byte, value string) []byte

// MatchRecord reports whether a taxpayer with the given identifiers matches a
// PayrollRecord. Any non-empty identifier that produces a matching HMAC is a hit.
// All provided identifiers must pass (AND logic) if more than one is given.
// If no identifiers are provided, returns false — never auto-match on absence.
func MatchRecord(rec PayrollRecord, hmacKey []byte, nin, taxID, email string) bool
```

Implementation notes:
- Use `crypto/hmac` and `crypto/sha256` from the standard library only.
- `ComputeHMAC` is deterministic (same key + same value → same bytes every time).
  The HMAC must be computed in constant time (`hmac.Equal` for comparison) to
  prevent timing attacks.
- `MatchRecord` treats empty strings as "not provided" and skips their check.
  A record with a nil `NINHMACSuffix` cannot match on NIN (nil ≠ any HMAC).

Tests:
- `TestComputeHMAC_Deterministic` — same inputs produce same output.
- `TestComputeHMAC_DifferentKeys` — different keys produce different output.
- `TestMatchRecord_HitOnNIN` — NIN matches, tax ID and email empty.
- `TestMatchRecord_HitOnEmail` — email matches.
- `TestMatchRecord_NoMatchOnWrongNIN` — NIN does not match.
- `TestMatchRecord_EmptyIdentifiers` — all identifiers empty → false (never auto-match).
- `TestMatchRecord_TimingSafe` — call `MatchRecord` on matching and non-matching inputs;
  assert `hmac.Equal` is used (the standard library's constant-time compare).
  If you cannot introspect this directly, document it as a code-review invariant.

### 3. `internal/employer/propose.go` — Proposal event builder

```go
// ProposalEvent builds the EvPayrollProposed event that is appended to a
// taxpayer's return log when a payroll record is matched. The caller provides
// the return ID and the next sequence number.
//
// The event carries only GrossKobo and PAYEKobo on the proposal — the full
// record is not in the event log because it belongs to the employer's import.
// The return only records that a proposal was received and from whom.
func ProposalEvent(rec PayrollRecord, emp Employer, returnID string, seq int, occurredAt time.Time) returns.Event
```

The payload must match the `EvPayrollProposed` shape that `Fold` already handles:
```json
{"record_id":"...","employer_id":"...","gross_kobo":...,"paye_kobo":...}
```

The `Actor` on the event is `ActorEmployer` with `ID = emp.ID`. This is the
architectural stake: the proposal is not a TAXPAYER event, so `DeclaredByTaxpayer`
stays true only if the taxpayer later accepts it.

```go
// AcceptanceEvent builds the EvPayrollAccepted event. The taxpayer actor and the
// income line they are accepting must be provided by the caller. This function
// enforces that the Actor.Kind is TAXPAYER; it returns an error if it is not.
//
// The caller constructs the IncomeLine from the record fields (converting
// GrossKobo to the declared amount, choosing the Label from the employer name).
func AcceptanceEvent(
    recID string,
    line returns.IncomeLine,
    returnID string,
    seq int,
    actor returns.Actor,
    occurredAt time.Time,
) (returns.Event, error)
```

The error case: if `actor.Kind != returns.ActorTaxpayer`, return an error. This is
a structural guardrail — it makes it impossible to write an auto-accept path that
bypasses the `DeclaredByTaxpayer` guard. The design doc (§2 of Part 3 and §6 of
Part 2) describes this refusal as the point of the architecture.

Tests in `internal/employer/propose_test.go`:
- `TestProposalEvent_Shape` — the resulting event has Kind=EvPayrollProposed,
  Actor.Kind=ActorEmployer, correct payload fields.
- `TestProposalEvent_DoesNotTouchIncome` — fold the event; assert no income line
  is created and assessment is not stale.
- `TestAcceptanceEvent_TaxpayerActor` — a TAXPAYER actor succeeds.
- `TestAcceptanceEvent_NonTaxpayerActorRejected` — an EMPLOYER actor returns error.
- `TestAcceptanceEvent_CreatesDeclaredByTaxpayer` — fold a ProposalEvent then the
  AcceptanceEvent; assert DeclaredByTaxpayer is true.
- `TestAcceptanceEvent_AutoAcceptPatternDetected` — attempting to accept with a
  SYSTEM actor returns an error with a message naming the guard.

### 4. `internal/employer/validation.go` — Import preconditions

```go
// ValidateImport checks that a PayrollImport is ready to be published. It is
// pure (no I/O) and returns a list of blocking issues.
//
// Checks:
//   - Employer has signed the Data Processing Agreement (DPASignedAt != nil).
//   - All records have a non-zero GrossKobo.
//   - No record has a negative amount in any Kobo field.
//   - EmployerID is not empty.
//   - YearOfAssessment is not zero.
func ValidateImport(imp PayrollImport, emp Employer) []ImportIssue

type ImportIssue struct {
    RecordID string // empty for batch-level issues
    Field    string
    Message  string
}
```

The DPA gate mirrors the rule-set sign-off gate: both are cheap now and impossible
to retrofit once employer data is in the system. A `ValidateImport` that returns no
issues is the precondition for `Status = ImportPublished`.

Tests:
- `TestValidateImport_NoDPA` — `DPASignedAt` nil → issue with empty RecordID.
- `TestValidateImport_ZeroGross` — one record with GrossKobo=0 → issue with that RecordID.
- `TestValidateImport_NegativeAmount` — one record with negative PAYEKobo → issue.
- `TestValidateImport_Clean` — fully valid import → empty issues slice.

### 5. Multi-employer integration scenario

Add a scenario to the integration test suite (either in `internal/integration/` or
a new `internal/employer/integration_test.go`):

**S-EMPLOYER-001: Mid-year job change — two employers, income summed.**

1. Open a return.
2. Employer A proposes a payroll record (₦6,000,000 gross, ₦800,000 PAYE) — fold
   the event. Assert Income is still empty. Assert assessment is not stale.
3. Employer B proposes a payroll record (₦4,000,000 gross, ₦400,000 PAYE) — fold.
   Same assertions.
4. Taxpayer accepts Employer A's record (TAXPAYER actor). Fold. Assert Income has
   one line with ₦6,000,000.
5. Taxpayer accepts Employer B's record (TAXPAYER actor). Fold. Assert Income has
   two lines (total ₦10,000,000). Assert DeclaredByTaxpayer is true.
6. Assert that attempting to sign before computing an assessment fails with the
   stale/missing assessment guard.
7. Assert that if either proposal is accepted by a non-TAXPAYER actor (simulate by
   using AcceptanceEvent with ActorSystem), the function returns an error before
   any event is emitted.

**S-EMPLOYER-002: Partial acceptance — one employer accepted, one rejected.**

1. Open return, propose two records (A and B).
2. Accept A (TAXPAYER). Reject B (TAXPAYER, `EvPayrollRejected`).
3. Assert Income has only one line (from A).
4. Assert ProposedPayroll has A with Status="ACCEPTED" and B with Status="REJECTED".
5. Assert DeclaredByTaxpayer is true (only TAXPAYER-actor income is present).

**S-EMPLOYER-003: W102 wire-up via coordinator.**

1. Open return. Propose payroll with GrossKobo=₦9,000,000.
2. Accept payroll. Income is ₦9,000,000 from TAXPAYER actor.
3. Revise income to ₦8,500,000 (taxpayer's payslips disagree with employer record).
4. Call `coordinator.BuildFilingContext` on this projection with an appropriate base.
5. Assert `FilingContext.EmployerGrossKobo` == ₦9,000,000 (from accepted proposal).
6. Assert W102 fires in `validate.Validate` output (₦9m employer vs. ₦8.5m declared).

### 6. DPA note

The NDPA consequence is structural, not optional. Include a package-level comment
in `internal/employer/types.go` that states:

> When an employer uploads payroll, they are the data controller and this system is
> the processor. Once an employee creates an account and starts a return, this system
> becomes a controller for that person's data. A Data Processing Agreement with every
> employer is required before the first upload. `ValidateImport` enforces this gate;
> do not remove or bypass it.

This ensures the next engineer to touch this code knows why the gate exists.

### 7. Harness and CI

`go test ./... -count=1` stays green. `internal/employer` imports only the standard
library, `taxengine` (for Kobo/IncomeCategory types), and `internal/returns` (for
event types). It must not import a YAML library, a database driver, or `net/http`.

The import isolation test must still pass for `taxengine` and `internal/returns`.
`internal/employer` importing both is fine; the check is one-directional.

## Do not do

- Do not build a database schema, SQL queries, or a migration tool. All types are
  in-memory structs. The persistence layer is out of scope until TASK-007 or later.
- Do not build an HTTP API, a CSV parser, or an employer onboarding UI. The employer
  channel's input side is this task; the transport layer is a separate concern.
- Do not implement employer CSV ingest or H1 export parsing. The `Source` field
  records where the data came from; parsing those formats is a later task.
- Do not auto-accept a payroll record. `AcceptanceEvent` must return an error if
  the actor is not TAXPAYER. This is the point. If someone adds a convenient
  auto-accept helper, the architecture fails silently.
- Do not build foreign-tax-credit capping, cross-year loss carryforward, or
  any other cross-year computation.
- Do not change any figure in `rules/ng-2026.yaml`.
- Do not use `float64` for money.
- Do not call `time.Now()` in any test subject.

## Report back with

- Package inventory: what was created, what it imports, one-line purpose.
- The HMAC design: key lifecycle, where the key comes from in tests, and why
  `hmac.Equal` is required for comparison.
- The `AcceptanceEvent` guard: exact error message, and which scenario would cause
  it to fire silently if it were absent.
- S-EMPLOYER-001 walkthrough: which events were emitted, which assertions passed,
  and the `DeclaredByTaxpayer` value at each checkpoint.
- S-EMPLOYER-003 walkthrough: the `EmployerGrossKobo` value passed to
  `FilingContext`, the W102 issue produced, and the declared vs. employer amount.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` output.
