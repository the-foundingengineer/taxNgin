TASK-007 — Shadow Replay, Evidence Integrity, and Amendment Tracking

## Context (read this first — you have no prior conversation history)

This repo is the backend for a Nigerian personal income tax prep product (Shape A:
self-preparation). TASKs 001–006 built the engine, golden corpus, lifecycle
integration, submission pack, TCC readiness, multi-year registry, filing coordinator,
and employer channel.

Three data-integrity gaps remain that must be closed before the product can be
trusted in production:

**Gap 1 — `EvEvidenceRemoved` is a silent no-op.**
`internal/returns/events.go` defines `EvEvidenceRemoved EventKind = "EvidenceRemoved"`
(line 28) but `Fold`'s switch statement has no case for it. When that event is
replayed, it falls through to the `default` branch (unknown kinds are tolerated —
correct design) but produces no effect. A user who removes their rent-receipt upload
still has `EvidenceIDs` containing the receipt, so `HasEvidence` stays true and W101
never fires. That is a correctness bug: the validation catalogue's evidence check is
bypassed.

**Gap 2 — Shadow replay is not implemented.**
The design doc (`tax/02-engine-and-employer.md` §4) is explicit:
> "Before publishing a new rule set version, replay every stored assessment against it
> and emit the exact list of taxpayers whose numbers would change and by how much.
> Never ship a rule change without that list in front of a human."
Nothing in the repo does this. The Registry from TASK-005 makes it achievable in
pure Go. This is a blocking prerequisite for the practitioner sign-off workflow: the
practitioner corrects numbers → engineer updates YAML → shadow replay confirms only
the expected assessments change → practitioner approves → publish.

**Gap 3 — Amendment tracking has no forward link.**
`EvAmendmentOpened` in `Fold` sets `p.State = StateSuperseded` but the event payload
carries no reference to the amendment return that replaced it. The design doc
(Part 3 §8) says:
> "Because both are event logs, the diff between them is derivable rather than
> reconstructed, which is exactly what the authority will ask for."
Without a link between the original and the amendment, the diff cannot be computed.
Additionally, the amendment return has no way to declare it amends a specific prior
return, so the objection schedule ("our number vs. the authority's number, supporting
schedule attached") cannot be generated.

Read in full before touching code:

- `tasks/TASK-006-employer-channel.md` (what the projection contains)
- `tax/02-engine-and-employer.md` (§4 shadow replay, §3 trace)
- `tax/04-return-lifecycle-and-pack.md` (§8 amendments and objections)
- `internal/returns/events.go` (Fold, EvEvidenceAttached, EvEvidenceRemoved)
- `internal/bridge/bridge.go` (ProjectionToFacts — feeds the shadow replay engine)
- `taxengine/engine.go` (Compute — the function shadow replay calls)
- `internal/rulesregistry/registry.go` (Registry.Get — how to fetch a rule set)

## Prerequisite: TASK-005 and TASK-006 must be done

`go test ./... -count=1` must pass. `internal/rulesregistry`, `internal/coordinator`,
and `internal/employer` must exist. Shadow replay depends on the Registry to look up
the correct rule set by (jurisdiction, year). If those packages do not exist, finish
TASK-005 first.

## What "done" looks like

### 1. Bug fix — `EvEvidenceRemoved` in Fold

Add a case to `Fold`'s switch statement in `internal/returns/events.go`.

The `EvEvidenceRemoved` payload must mirror `EvEvidenceAttached` exactly, so the
two events are symmetric:

```json
{"target": "relief:RENT_RELIEF", "evidence_id": "ev-123"}
```

where `target` is `"income:<id>"` or `"relief:<code>"`.

Add a `removeEvidence(target, evidenceID string)` method on `*Projection` alongside
the existing `attachEvidence`. Implementation: remove exactly one occurrence of
`evidenceID` from the relevant `EvidenceIDs` slice. If the ID is not present, do
nothing — idempotent. Do not error on an unknown target; log-replay tolerance
applies here too.

**Required tests in `internal/returns/events_test.go`:**

- `TestFold_EvidenceRemoved_Relief` — attach an evidence ID to `RENT_RELIEF`, then
  remove it. Assert `Reliefs["RENT_RELIEF"].EvidenceIDs` is empty. Assert
  `bridge.ProjectionToFacts(p).Reliefs[0].HasEvidence == false`, confirming W101
  would fire after removal.
- `TestFold_EvidenceRemoved_Income` — attach an evidence ID to an income line, then
  remove it. Assert `Income[id].EvidenceIDs` is empty.
- `TestFold_EvidenceRemoved_IdempotentOnMissing` — remove an evidence ID that was
  never attached. Assert no panic, no error, projection unchanged.
- `TestFold_EvidenceRemoved_PreservesOtherIds` — attach two evidence IDs, remove
  one. Assert the other remains.
- `TestFold_EvidenceReplay_Roundtrip` — attach, remove, re-attach. The final
  `EvidenceIDs` must contain exactly the re-attached ID, proving Fold is pure (same
  event sequence → same projection).

### 2. Shadow replay — `internal/shadowreplay`

New package. Imports: `internal/bridge`, `internal/returns`, `taxengine`. No I/O,
no clock, no YAML dependency.

```go
// Delta describes one assessment whose computed tax would change under the new
// rule set. DiffKobo is positive when the taxpayer would owe more.
type Delta struct {
    ReturnID           string
    YearOfAssessment   int
    Jurisdiction       string
    OldTaxPayableKobo  taxengine.Kobo
    NewTaxPayableKobo  taxengine.Kobo
    DiffKobo           taxengine.Kobo // NewTaxPayableKobo - OldTaxPayableKobo
    OldChargeableKobo  taxengine.Kobo
    NewChargeableKobo  taxengine.Kobo
    // AssessmentWasStale is true when the stored snapshot no longer reflects
    // the projection's current facts. The replay still runs so the delta is
    // visible, but the caller should treat this delta with caution.
    AssessmentWasStale bool
}

// Report is the output of one shadow replay run.
type Report struct {
    NewRuleSetVersion string
    NewRuleSetDigest  string
    Replayed          int     // assessments successfully re-run
    Changed           []Delta // assessments whose tax payable changed
    Skipped           int     // projections with no assessment or wrong year
}

// Replay re-runs the most recent assessment for every projection in
// projections against newRS. Only projections whose Jurisdiction and
// YearOfAssessment match newRS are replayed; others contribute to
// Report.Skipped. Projections with Assessment == nil are also skipped.
//
// Replay never mutates projections. It calls bridge.ProjectionToFacts to
// reconstruct the facts and taxengine.Compute to get the new assessment.
// The original snapshot is compared against the new assessment; differences
// populate Report.Changed.
//
// Returns an error only if Compute returns an error (which means the facts
// themselves are malformed — a bug, not a normal condition).
func Replay(projections []*returns.Projection, newRS *taxengine.RuleSet) (Report, error)
```

**Required tests in `internal/shadowreplay/shadowreplay_test.go`:**

Build projections in-memory with `returns.Replay` (event sequences), not by
manually constructing struct fields. Use two rule sets: `ruleSetA` (first band
0 bp — produces ₦0 on ₦600,000 DIVIDEND) and `ruleSetB` (first band 500 bp —
produces ₦30,000 on the same income). Load both from the YAML bytes of the actual
rule sets in the repo (ng-2026 for year 2026 and the test fixture for year 1).

- `TestReplay_NoDelta` — one projection with a snapshot computed under `ruleSetA`.
  Shadow replay against `ruleSetA` → `Report.Changed` is empty, `Report.Replayed == 1`.
  This is the steady-state check: publishing the exact same numbers must never show
  a delta.

- `TestReplay_Delta` — one projection with a snapshot computed under `ruleSetA`
  (₦0 tax on ₦600,000). Shadow replay against `ruleSetB` (5% first band) →
  `Report.Changed` has one `Delta` with `OldTaxPayableKobo == 0`,
  `NewTaxPayableKobo == 3_000_000`, `DiffKobo == 3_000_000`.

- `TestReplay_SkipsNoAssessment` — projection with `Assessment == nil` (never
  computed). → `Report.Skipped == 1`, `Report.Replayed == 0`.

- `TestReplay_SkipsWrongYear` — projection with `YearOfAssessment == 2026`, new RS
  with `YearOfAssessment == 1`. → `Report.Skipped == 1`.

- `TestReplay_StaleAssessmentFlagged` — projection with `AssessmentStale == true`.
  → Replayed (not skipped), `Delta.AssessmentWasStale == true` if the numbers differ.
  If the numbers happen to match despite staleness, no delta entry, but the replay
  still counts in `Report.Replayed`.

- `TestReplay_MultipleProjections` — three projections under `ruleSetA`: two with
  snapshots (one matches, one differs from ruleSetB), one with no assessment. Shadow
  replay against `ruleSetB` → `Report.Changed` has one entry, `Report.Skipped == 1`,
  `Report.Replayed == 2`.

- `TestReplay_GoldenCorpusNoDelta` — load all `compute` and `validate` fixtures from
  `testdata/golden/`, build a minimal projection for each (emit `EvReturnOpened` +
  `EvIncomeDeclared` events for each income item, fold in a fake `EvAssessmentComputed`
  built from the fixture's `expect` values), run shadow replay against the same rule
  set. Assert `Report.Changed` is empty. This is the regression gate: if any of the
  42 existing fixtures shows a delta against their own rule set, something is wrong
  with the bridge or the replay logic.

### 3. Amendment tracking — forward link and delta

**3a. Extend `EvReturnOpened` payload (backward-compatible)**

Add an optional `amends_return_id` field to the `EvReturnOpened` payload struct in
`Fold`. Extend `Projection` with:

```go
AmendsReturnID string // empty unless this return is an amendment of another
```

Fold handles it in `EvReturnOpened`:

```go
// existing fields...
AmendsReturnID string `json:"amends_return_id"`
```

This field is empty for all existing events (backward compatible — `json.Unmarshal`
leaves unrecognised fields at zero values).

**3b. Extend `EvAmendmentOpened` payload**

`EvAmendmentOpened` is emitted on the ORIGINAL return when an amendment is opened.
Add a payload to it:

```json
{"amendment_return_id": "R-AMEND-001"}
```

Extend `Projection`:

```go
AmendmentReturnID string // set on the superseded return; ID of its replacement
```

Fold handles it in `EvAmendmentOpened`:

```go
case EvAmendmentOpened:
    var v struct{ AmendmentReturnID string `json:"amendment_return_id"` }
    _ = json.Unmarshal(e.Payload, &v)
    p.AmendmentReturnID = v.AmendmentReturnID
    p.State = StateSuperseded
```

This is backward-compatible: existing `EvAmendmentOpened` events without a payload
produce an empty `AmendmentReturnID`, which is acceptable — the link is lost for
historical events but new ones carry it.

**3c. `internal/amendment` — delta computation**

New package. Imports: `taxengine`, `internal/returns`. No I/O.

```go
// IncomeDelta describes one income line that was added, removed, or changed.
type IncomeDelta struct {
    ID           string
    Category     string
    OldAmountKobo taxengine.Kobo // 0 if line was added in amendment
    NewAmountKobo taxengine.Kobo // 0 if line was removed in amendment
    DiffKobo      taxengine.Kobo
}

// ReliefDelta describes one relief claim that was added, removed, or changed.
type ReliefDelta struct {
    Code          string
    OldAmountKobo taxengine.Kobo
    NewAmountKobo taxengine.Kobo
    DiffKobo      taxengine.Kobo
}

// AssessmentDelta is the diff between an original return and its amendment.
// It is what the taxpayer submits alongside their objection under NTAA s.34.
type AssessmentDelta struct {
    OriginalReturnID  string
    AmendmentReturnID string
    YearOfAssessment  int
    Jurisdiction      string

    OldTaxPayableKobo taxengine.Kobo
    NewTaxPayableKobo taxengine.Kobo
    TaxDiffKobo       taxengine.Kobo // positive = amendment increases liability

    OldChargeableKobo taxengine.Kobo
    NewChargeableKobo taxengine.Kobo

    ChangedIncome  []IncomeDelta
    ChangedReliefs []ReliefDelta
}

// ComputeDelta derives the assessment difference between original and amendment.
// Both projections must have non-nil Assessments. Returns an error if either
// is missing an assessment — an unevaluated return cannot be diffed.
//
// The delta is computed from the projections' AssessmentSnapshot fields (stored
// numbers, not re-computed). Computing the delta from stored snapshots rather
// than re-running the engine is intentional: the delta must describe the numbers
// as they were signed, not what a later rule set would produce.
func ComputeDelta(original, amendment *returns.Projection) (AssessmentDelta, error)
```

**Required tests in `internal/amendment/amendment_test.go`:**

- `TestComputeDelta_HappyPath` — original return has ₦9,000,000 income, ₦50,000 tax.
  Amendment corrects income to ₦10,000,000 (one new income line), tax to ₦75,000.
  Assert `TaxDiffKobo == 25,000 kobo`, `len(ChangedIncome) == 1`,
  `ChangedIncome[0].DiffKobo > 0`.

- `TestComputeDelta_DecreasedLiability` — amendment reduces income (e.g. employer
  correction). `TaxDiffKobo` is negative (taxpayer overpaid). Assert the sign is correct.

- `TestComputeDelta_MissingAssessment` — original has `Assessment == nil` → error.
  Amendment has `Assessment == nil` → error.

- `TestComputeDelta_AddedIncomeLine` — amendment adds a completely new income category.
  Assert new line appears in `ChangedIncome` with `OldAmountKobo == 0`.

- `TestComputeDelta_RemovedIncomeLine` — amendment removes an income line that was
  on the original. Assert it appears in `ChangedIncome` with `NewAmountKobo == 0`.

- `TestComputeDelta_UnchangedLinesNotReported` — original and amendment share two
  income lines; only one differs. Assert `len(ChangedIncome) == 1`.

### 4. Objection schedule

Extend `internal/amendment` with a pure `ObjectionSchedule`:

```go
// ObjectionSchedule is the document a taxpayer submits to challenge an authority
// assessment that differs from the product's computed assessment.
type ObjectionSchedule struct {
    ReturnID          string
    YearOfAssessment  int
    Jurisdiction      string
    ObjectionDeadline time.Time

    // Our assessment (from the projection's AssessmentSnapshot).
    OurTaxPayableKobo taxengine.Kobo
    OurTraceSteps     []taxengine.Step // deserialised from AssessmentSnapshot.Trace

    // The authority's assessment (supplied by the caller after it is served).
    AuthorityTaxPayableKobo taxengine.Kobo

    // The difference the taxpayer is objecting to. Positive = authority is
    // overcharging; negative = authority is undercharging (rare, still valid).
    DisputedAmountKobo taxengine.Kobo

    // DaysRemaining is the number of days left to file the objection, computed
    // from ObjectionDeadline and the clock the caller provides. Negative means
    // the window has closed (the assessment is binding).
    DaysRemaining int
}

// BuildObjectionSchedule constructs the schedule from the projection and the
// authority's served assessment amount. The clock is injected; never time.Now().
func BuildObjectionSchedule(
    p *returns.Projection,
    authorityTaxPayableKobo taxengine.Kobo,
    now time.Time,
) (ObjectionSchedule, error)
```

Returns an error if:
- `p.Assessment == nil` — no computed assessment to compare against
- `p.ObjectionDeadline == nil` — no assessment has been served yet

`OurTraceSteps` is deserialised from `p.Assessment.Trace` (the `json.RawMessage`).
If the trace is nil (legacy event), `OurTraceSteps` is nil — not an error.

**Required tests:**

- `TestBuildObjectionSchedule_HappyPath` — projection with assessment (₦50,000 tax),
  objection deadline set (30 days from service), authority claims ₦80,000. Assert
  `DisputedAmountKobo == 30,000 kobo`, `DaysRemaining > 0`.
- `TestBuildObjectionSchedule_DeadlinePassed` — `now` is after the deadline. Assert
  `DaysRemaining < 0`.
- `TestBuildObjectionSchedule_NoAssessment` — `p.Assessment == nil` → error.
- `TestBuildObjectionSchedule_NoDeadline` — `p.ObjectionDeadline == nil` → error.
- `TestBuildObjectionSchedule_TraceDeserialised` — snapshot has a non-empty
  `Trace json.RawMessage`. Assert `OurTraceSteps` is populated.

### 5. Integration scenario — full amendment cycle

Add a test to `internal/integration/lifecycle_test.go` (or a new file there):

**S-AMEND-001: Original → SUPERSEDED → amendment assessed → delta computed.**

1. Open return R-ORIG. Declare ₦9,000,000 EMPLOYMENT income. Compute, validate, sign,
   issue pack, attest, upload ACK, record assessment (₦1,000,000 tax, served now,
   30-day window). State: ASSESSED.
2. Open amendment return R-AMEND with `amends_return_id: "R-ORIG"` in the payload.
3. Emit `EvAmendmentOpened` on R-ORIG with `amendment_return_id: "R-AMEND"`.
   Assert R-ORIG is SUPERSEDED. Assert `R-ORIG.AmendmentReturnID == "R-AMEND"`.
4. Assert `R-AMEND.AmendsReturnID == "R-ORIG"`.
5. Declare revised income on R-AMEND (₦10,000,000 — added consulting fees).
   Compute, validate, sign. Record a new assessment (₦1,500,000 tax).
6. Call `amendment.ComputeDelta(original, amend)`. Assert `TaxDiffKobo == 500,000 kobo`.
7. Call `amendment.BuildObjectionSchedule(original, authorityKobo, now)` with the
   authority's served assessment (₦1,200,000 tax). Assert `DisputedAmountKobo` is the
   expected difference. Assert `DaysRemaining` is within the 30-day window.

### 6. Harness and CI

`go test ./... -count=1` stays green. The import isolation test must still pass:
`taxengine` and `internal/returns` import nothing outside stdlib.

New packages and their allowed imports:
- `internal/shadowreplay` → `internal/bridge`, `internal/returns`, `taxengine`
- `internal/amendment` → `internal/returns`, `taxengine`

Neither imports a YAML library, a database driver, or `net/http`.

## Do not do

- Do not change any figure in `rules/ng-2026.yaml` or `testdata/rulesets/ng-test-fixture.yaml`.
- Do not mark any golden fixture `review.status` as verified.
- Do not build a database, an HTTP API, or a persistence layer.
- Do not implement foreign tax credit capping, cross-year loss carryforward, or pre-2026
  rule sets. None of those have practitioner sign-off.
- Do not use `float64` for money.
- Do not call `time.Now()` in any test subject. The objection schedule function takes
  `now time.Time` — inject it from the test.
- Do not build a PDF renderer or formatted output for the objection schedule. The
  schedule is a Go struct; rendering is a UI concern.
- Do not automatically send an objection. Under Shape A the product cannot lodge the
  objection for the taxpayer; it can only produce the schedule.

## Report back with

- **Evidence-removal fix**: the exact `removeEvidence` implementation, and a
  `TestFold_EvidenceRemoved_Relief` walkthrough confirming `HasEvidence` transitions
  from true to false after removal.
- **Shadow replay**: the five required test scenarios, pass/fail, and the specific
  `DiffKobo` values in `TestReplay_Delta`. Confirm `TestReplay_GoldenCorpusNoDelta`
  passes (no existing fixture shows a delta against its own rule set).
- **Amendment tracking**: the two new Projection fields, the payload extensions for
  `EvReturnOpened` and `EvAmendmentOpened`, and confirmation of backward compatibility
  (existing S-001…S-010 integration tests unaffected).
- **S-AMEND-001**: event sequence, `TaxDiffKobo`, `DisputedAmountKobo`,
  `DaysRemaining` values at each checkpoint.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` output.
