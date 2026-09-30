TASK-003 — Wire the return lifecycle end-to-end and prove it against the event log, the state machine, and the engine

## Context (read this first — you have no prior conversation history)

This repo is the backend for a Nigerian personal income tax prep product.
TASK-001 bootstrapped the Go module, proved `taxengine.Compute` against five
hand-authored golden fixtures, unit-tested the lifecycle state machine and
`Fold` in isolation, and wired CI. TASK-002 expanded the golden corpus to
forty-two fixtures, verified the validation catalogue (`internal/validate`),
and audited the design invariants (pure hash, four SIGN guards, evidence
separation).

What neither task did is **drive the return through its lifecycle as a
cohesive flow**. The state machine exists. The event log exists. `Compute`,
`Validate`, and `ComputePenalties` exist. But nobody has yet built the
integration layer that turns taxpayer actions into events, folds those events
into projections, invokes the engine on the projection's facts, runs
validation, and proves the guards fire — or don't — exactly when the design
says they should.

That is this task. It is the first time these three packages are tested
together rather than in isolation, and it is where the seams between them
break.

Read in full before touching code:

- `tasks/TASK-001-bootstrap-and-verify-engine.md` (what the foundation expects)
- `tasks/TASK-002-starter-forty-corpus-and-validation.md` (what the corpus covers)
- `tax/02-engine-and-employer.md` (§3 trace, §5 validation, §6 employer channel)
- `tax/04-return-lifecycle-and-pack.md` (§1 lifecycle, §2 actors, §3 event log, §4 pack)
- `internal/returns/lifecycle.go` (the state machine)
- `internal/returns/events.go` (the event catalogue and `Fold`)
- `internal/validate/validate.go` (the validation catalogue)
- `taxengine/engine.go` (the Compute function)

## Prerequisite: TASK-001 and TASK-002 must be done

`go test ./...` must pass. All forty-two golden fixtures must exist under
`testdata/golden/` and be dispatched by kind. If that is not the case,
**finish the earlier tasks first**. Do not run integration tests against a
harness that cannot pass its own unit tests.

## What "done" looks like

### 1. `internal/returns` integration test suite

New file: `internal/returns/integration_test.go`. This package currently has
unit tests for individual transitions and Fold steps. The integration tests
drive multi-step scenarios end-to-end.

Each scenario begins with `ReturnOpened`, walks the return through a series
of domain actions, and asserts the projection state, the transition outcomes,
and the engine results at each checkpoint. Use `Replay` to rebuild the
projection at every assertion point — never reach into the projection
mid-construction and then keep folding; that papers over ordering bugs.

**Required scenarios (minimum — add more if you find a gap):**

**S-001: Happy path — draft to cleared.** Open a return. Declare income
(two employment lines, one EMPLOYMENT actor TAXPAYER, one EMPLOYMENT actor
EMPLOYER which blocks signing). Accept the employer payroll record so both
lines are TAXPAYER-declared. Declare a relief (PENSION). Declare a base
(annual_rent_paid). Claim RENT_RELIEF. Compute the assessment — assert the
assessment snapshot matches `taxengine.Compute` called with the same facts.
Run validation — assert the issues list is what `internal/validate.Validate`
would produce. Transition VALIDATE → SIGN → ISSUE_PACK → ATTEST_FILED →
UPLOAD_ACK → RECORD_ASSESSMENT → RECORD_PAYMENT → CLEAR. At each transition
assert the state is correct and no guard rejects illegitimately.

**S-002: SIGN guard — stale assessment.** Open a return, declare income,
compute an assessment. Then revise income (triggering `markStale`). Attempt
SIGN. It must fail with "facts changed since the assessment was computed".
Recompute, then SIGN succeeds.

**S-003: SIGN guard — unpublished rule set.** Open a return, declare
income, compute an assessment. Set `RuleSetPublished = false` on the
projection. Attempt SIGN. It must fail with "unpublished rule set". This
verifies the belt-and-braces gate from §1 of `04-return-lifecycle-and-pack.md`.

**S-004: SIGN guard — DeclaredByTaxpayer false.** Open a return. Emit
`PayrollRecordProposed` (EMPLOYER actor). Do not accept it. Income arrives
with EMPLOYER actor. Attempt SIGN. It must fail with "not all income was
declared by the taxpayer actor". Accept the payroll record (TAXPAYER actor).
SIGN succeeds.

**S-005: SIGN guard — blocking validation errors.** Open a return. Run
validation producing blocking errors (e.g. no income, no nil declaration).
Attempt VALIDATE → SIGN. VALIDATE fails because blocking errors remain.
Add income, revalidate, VALIDATE succeeds, SIGN succeeds.

**S-006: Edit after signing invalidates.** Open a return, declare income,
compute, validate, sign. Then revise income. Assert state returns to DRAFT
(via INVALIDATE). Recompute, revalidate, re-sign. This is the cycle the
design calls out in the lifecycle diagram.

**S-007: Payroll proposal does not touch income.** Emit
`PayrollRecordProposed`. Assert projection Income is empty and the
assessment is NOT stale. This is the Part 2 rule: employer data is evidence,
not a declaration.

**S-008: Objection deadline.** Transition to ASSESSED, then
RECORD_ASSESSMENT with a served date and window. Assert ObjectionDeadline is
set correctly (served + 30 days per the rule set). Transition to OBJECTED
then WITHDRAW, asserting state roundtrips.

**S-009: Idempotency.** Emit two events with the same `IdempotencyKey`.
The test must demonstrate that the second event is either rejected or
produces an identical projection, depending on how the deduplication layer
is implemented. If the current code has no deduplication logic, **implement
it in `Fold`** — the design document (§3 of `04-return-lifecycle-and-pack.md`)
is explicit that idempotency is required, and its absence is a bug.

**S-010: Unknown event kind tolerance.** Emit an event with kind
`"FutureFeatureAdded"` (not in the catalogue). Assert Fold does not error,
version advances, and the projection is otherwise unchanged. This is the
rolling-deploy safeguard from §3 of `04-return-lifecycle-and-pack.md`, and
TASK-001 already unit-tests it. Repeat it in integration to confirm it
survives a full replay.

### 2. Projection → TaxFacts bridge

New function: `internal/returns.ProjectionToFacts(p *Projection) taxengine.TaxFacts`

This is the missing glue. The projection stores income, reliefs, bases,
and credits. `Compute` takes `TaxFacts`. The bridge maps one to the other.
It must be tested: given a projection with known income and relief lines,
the TaxFacts it produces must, when fed to `Compute`, yield the same
assessment stored in the projection snapshot.

If a mapping is lossy (e.g. projection stores evidence IDs but TaxFacts
does not carry them), document it explicitly. If the mapping is ambiguous
(e.g. the projection uses `int64` kobo but `TaxFacts` uses `taxengine.Kobo`),
resolve it with a type assertion, not a cast.

### 3. Assessment consistency invariant

New test: compute via the bridge, compare to the snapshot. For every
scenario in §1 where an assessment is computed, assert that the projection's
`AssessmentSnapshot` fields (chargeable income, tax payable, total credits,
balance) are byte-identical to a fresh `Compute` call with
`ProjectionToFacts(p)`. If they differ, that is a roundtrip bug at the seam
between the event log and the engine, and it is the most dangerous kind
because it means the number the user signed is not the number the engine
would produce again.

### 4. Validation integration

New test: run `internal/validate.Validate` using facts from the bridge and
the loaded rule set. Assert the issues match what the projection recorded
from the `ValidationRun` event. If the projection stores a simplified list
of codes and `Validate` returns full `Issue` structs, the test asserts on
codes and severities, not messages.

### 5. `RuleSetPublished` flag management

`Projection.RuleSetPublished` is read by the SIGN guard, but nothing in
the event log currently sets it. If `EvAssessmentComputed` carries the rule
set version and the rule set status, Fold should set
`RuleSetPublished = (status == "PUBLISHED")`. If the event does not carry
the status, the guard must obtain it another way — document the decision.

If this flag is currently always `false` (because nothing sets it), every
SIGN attempt against a DRAFT rule set fails for the right reason and every
attempt against a PUBLISHED rule set fails for the wrong reason. That is a
bug TASK-001 and TASK-002 never hit because they tested the guard in
isolation with a hand-set projection. Fix it, and say so in the report.

### 6. Harness and CI

`go test ./... -count=1` stays green. The integration tests must live in
`internal/returns/` (they test the package's public surface) or in a
dedicated `internal/integration/` package that imports both `returns` and
`taxengine`. Either is acceptable. Do not put them in `taxengine` — the
engine must not know about returns.

Add a build tag `//go:build integration` only if the tests are slow (>5s).
If they run in under a second, which they should because everything is
in-memory and pure, keep them in the default test set.

## Do not do

- Do not change any figure in `rules/ng-2026.yaml`.
- Do not flip `DRAFT` → `PUBLISHED` or fill `approved_by`.
- Do not mark any fixture `review.status` as verified.
- Do not edit `expect` numbers in any `G-*.yaml` file.
- Do not add a database, an HTTP server, auth, a portal pack renderer,
  employer CSV ingest, or any external dependency to `taxengine` or
  `internal/returns`.
- Do not build the submission pack or the TCC readiness projection.
  Those require the portal map, which is out of scope until a logged-in
  walk-through of etax.lirs.net corrects every label.
- Do not use `float64` for money.
- Do not use `time.Now()` anywhere in the test subject. Inject the clock.

## Report back with

- Scenario inventory: ID, description, pass/fail, and the specific
  transitions exercised.
- Full detail on every guard assertion: which guard, what was expected,
  what happened, whether it is a bug fix or a finding.
- The `ProjectionToFacts` mapping, including any lossy or ambiguous fields,
  and the resolution.
- The `RuleSetPublished` fix, with before/after behavior.
- Any event log or Fold bug you fixed, and why it was a bug rather than a
  design choice.
- Any scenario you added beyond the ten required, and why.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` output.
