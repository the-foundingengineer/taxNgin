TASK-005 — Multi-Year Rule Set Registry, Filing Coordinator, and W105 Integration

## Context (read this first — you have no prior conversation history)

This repo is the backend for a Nigerian personal income tax prep product (Shape A:
self-preparation). TASKs 001–004 built the engine, the golden corpus, the lifecycle
integration, the submission pack, and the TCC readiness projection.

Two structural gaps remain before multi-year filing can work:

**Gap 1 — No Registry.** Every test that needs a rule set calls `golden.LoadRuleSet()`
directly. There is no production pattern for "give me the rule set for jurisdiction NG,
year of assessment 2026" — or any other year. When practitioners eventually sign
historical rule sets (they must; see Part 3 §7 of `tax/04-return-lifecycle-and-pack.md`),
nothing exists to load them by (jurisdiction, year) lookup.

**Gap 2 — No filing coordinator.** `validate.FilingContext` has `PriorYearGrossKobo
*Kobo` and `YearsOnFile int`. Both are always nil/0 in tests because nothing derives
them from a prior year's projection. The W105 warning ("income fell by more than 50%
from prior year") never fires in a realistic scenario. Similarly, `EmployerGrossKobo`
is always nil even though accepted payroll proposals are already on the Projection.

This task closes both gaps. It explicitly does **not** author historical tax law. The
pre-2026 regime (Consolidated Relief Allowance, old 7–24% bands, 1% minimum tax)
requires the same CITN practitioner sign-off as `ng-2026.yaml` — which has not
happened. A synthetic test-fixture rule set, clearly labeled as not historical law,
proves the data-driven architecture instead.

Read in full before touching code:

- `tasks/TASK-004-submission-pack-and-tcc-readiness.md` (TCC requirements)
- `tax/04-return-lifecycle-and-pack.md` (§7 multi-year, §4 shadow replay)
- `tax/02-engine-and-employer.md` (§4 three test layers, shadow replay)
- `internal/validate/validate.go` (FilingContext, W105, W102)
- `taxengine/rules.go` (Load, LoadOptions, RuleSet.Validate)
- `internal/golden/fixture.go` (existing LoadRuleSet and harness to understand what changes)
- `taxengine/golden_test.go` (the test harness to update)

## Prerequisite: TASK-004 must be done

`go test ./... -count=1` must pass. `internal/pack`, `internal/tcc`, and
`internal/portalmap` must exist. If not, finish TASK-004 first.

## What "done" looks like

### 1. `internal/rulesregistry` — Rule Set Registry

New package. Not in `internal/golden` (test helper) and not in `internal/rulesyaml`
(YAML parsing utility). Its own package with a clear production responsibility.

```go
var ErrRuleSetNotFound = errors.New("rulesregistry: rule set not found")

// Registry is an in-memory index of rule sets keyed by (jurisdiction, year_of_assessment).
type Registry struct { ... }

// New parses each YAML file in rawFiles into a RuleSet, validates it, and
// returns a Registry indexed by (jurisdiction, year_of_assessment). It fails
// fast: any file that fails taxengine.Load is a hard error.
//
// The caller provides the unmarshal function and LoadOptions so this package
// never imports a YAML library (same pattern as taxengine.Load). In production,
// AllowDraft is false. In tests, AllowDraft is true because all our rule sets
// are still DRAFT.
func New(rawFiles [][]byte, unmarshal func([]byte, any) error, opts taxengine.LoadOptions) (*Registry, error)

// Get retrieves a rule set. Returns ErrRuleSetNotFound if no rule set for
// (jurisdiction, year) has been loaded.
func (r *Registry) Get(jurisdiction string, year int) (*taxengine.RuleSet, error)
```

The implementation: parse each file with `taxengine.Load`, store in a map keyed
`jurisdiction + "/" + strconv.Itoa(year)`. No ambiguity, no fuzzy matching.

Tests:
- `TestNew_RejectsInvalidYAML` — a garbled YAML file makes `New` return an error.
- `TestNew_RejectsDuplicateJurisdictionYear` — two files with the same
  (jurisdiction, year) make `New` return an error. A Registry with duplicate keys
  is corrupt data, not an ambiguity to silently resolve.
- `TestGet_Found` — `Get("NG", 2026)` returns the ng-2026 rule set.
- `TestGet_NotFound` — `Get("NG", 1900)` returns `ErrRuleSetNotFound`.
- `TestGet_MultipleRuleSets` — load two rule sets; `Get` returns the right one
  for each year; the two rule sets produce different outputs for the same income.

### 2. Synthetic test-fixture rule set

New file: `testdata/rulesets/ng-test-fixture.yaml`.

This file is **not** historical tax law. Its sole purpose is to prove that two
different YAML files produce two different engine outputs from the same `TaxFacts`,
without any `if year == X` branch in the engine. Choose values that make the
difference unmistakable:

- Keep the same structure as `ng-2026.yaml` (bands, reliefs, exemptions, penalties).
- Set `jurisdiction: "NG"` and `year_of_assessment: 1` (year 1 is
  unambiguously synthetic; no real tax year is ever 1).
- Change the first band rate from 0 bp to a non-zero value (e.g. 500 bp = 5%) so
  that any income below ₦800,000 produces visible tax rather than zero. This makes
  the registry dispatch test trivially verifiable: the same ₦600,000 income
  produces ₦0 under ng-2026 (first band 0%, then possibly exempt) and produces a
  clear positive number under the fixture.
- Set `status: DRAFT` and leave `approved_by` empty.
- Top-of-file comment: "TEST FIXTURE — not historical tax law. Change any value
  freely; none of these numbers represent any real year's statutory rates."
- `version: "test-fixture"`, `penalties.filing_deadline:` some synthetic date.

Do not put this file in `rules/`. Only practitioner-reviewed rule sets belong there.

### 3. Golden test harness update

`taxengine/golden_test.go` currently calls `golden.LoadRuleSet()` for every fixture,
always loading ng-2026. Change it to:

1. Build a `Registry` at test startup by scanning two directories: `rules/` (real
   rule sets) and `testdata/rulesets/` (test fixtures). Load both with `AllowDraft: true`.
2. For each fixture, look up `reg.Get(fixture.Jurisdiction, fixture.YearOfAssessment)`.
   Fallback: if `Jurisdiction` is empty, default to `"NG"`; if `YearOfAssessment` is
   zero, default to `2026`. This maintains backward compatibility with fixtures that
   predate multi-year support and do not set these fields.
3. If `Get` returns `ErrRuleSetNotFound`, fail the test immediately with a clear message
   naming the fixture and the (jurisdiction, year) that was looked up.

Existing fixtures that already carry `year_of_assessment: 2026` (most of them) work
unchanged. Fixtures with a missing or zero year fall back to 2026.

### 4. G-043 — proof of data-driven multi-year dispatch

New fixture: `testdata/golden/G-043-registry-dispatch-proof.yaml`.

- `kind: compute`
- `year_of_assessment: 1` (the synthetic fixture year)
- `jurisdiction: "NG"`
- Income that falls in the first band: employment income that the 2026 rule set
  would exempt (e.g., income ≤ ₦840,000) but the test fixture would tax (exemption
  threshold is also changed, or the income is above the fixture's threshold).
  Alternatively, use income in the first band where the rate difference (0 bp vs.
  500 bp) produces clearly different expected tax.
- `why_this_case`: "Proves the engine dispatches on YAML data, not on year constants
  in code. The fixture rule set (year_of_assessment: 1) has a non-zero first band
  rate. The same income produces ₦0 under ng-2026 and a positive tax payable here."
- `expect.tax_payable`: the hand-computed correct amount under the fixture's rate.
- `review.status: UNVERIFIED`.

Do **not** edit `expect` in any existing fixture to make this work. If the harness
breaks an existing fixture after the Registry change, that is a bug to fix, not a
fixture to adjust.

### 5. `internal/coordinator` — Filing Context Builder

New package. Imports `internal/returns`, `internal/validate`, `taxengine`. No I/O.

```go
// BuildFilingContext constructs a validate.FilingContext for the target
// projection by deriving cross-year fields from the history slice. The caller
// provides a base context (TaxID, NIN, Signed, RelevantAuthority, etc.); the
// coordinator populates the fields that require projection history.
//
// history may be nil or empty — the coordinator degrades gracefully.
func BuildFilingContext(
    target *returns.Projection,
    history []*returns.Projection,
    base validate.FilingContext,
) validate.FilingContext
```

**W105 — prior year gross (`PriorYearGrossKobo`)**

1. Search `history` for a projection where
   `p.YearOfAssessment == target.YearOfAssessment - 1`.
2. If found, sum `AmountKobo` across all of that projection's `Income` map.
3. If the sum is > 0, set `ctx.PriorYearGrossKobo = &sum`. If the sum is zero
   (nil-return year — no income), leave `PriorYearGrossKobo` nil. This is correct:
   W105's existing logic only fires when `PriorYearGrossKobo != nil && > 0`. A
   nil-return prior year does not trigger the warning.
4. If no matching prior year is found (first-time filer, or the year was never in
   the slice), leave `PriorYearGrossKobo` nil. Do not guess, do not default to zero.

**W102 — employer gross (`EmployerGrossKobo`)**

The Projection already holds accepted payroll proposals in `ProposedPayroll`. Sum
`GrossKobo` from proposals with `Status == "ACCEPTED"`. If the total is > 0, set
`ctx.EmployerGrossKobo = &total`. W102 then fires (inside `validate.Validate`) when
the employer-reported gross differs from the taxpayer's declared employment income.

Neither W105 nor W102 require any external service. Both are derivable purely from
the in-memory projection and its history.

**YearsOnFile**

Count projections in `history` where `State == StateAckUploaded || State ==
StateCleared`. Include the target year in the count if it is in one of those states.
This is what `validate.Validate`'s W106 check ("fewer than three years on file —
blocks TCC") needs.

**Required tests in `internal/coordinator/coordinator_test.go`:**

- `TestBuildFilingContext_W105_FiresOnDrop` — target year gross ₦1m, prior year
  gross ₦3m (>50% drop). W105 present in Validate output.
- `TestBuildFilingContext_W105_NoFireOnSmallDrop` — target ₦2m, prior ₦3m (<50%
  drop). W105 absent.
- `TestBuildFilingContext_W105_NoPriorYear` — no prior year in history.
  `PriorYearGrossKobo` nil. W105 absent.
- `TestBuildFilingContext_W105_NilReturnPriorYear` — prior year projection has
  empty Income map (nil return). `PriorYearGrossKobo` nil. W105 absent.
- `TestBuildFilingContext_W102_FiresOnMismatch` — accepted payroll reports
  ₦9m; taxpayer declared ₦8.5m. W102 present.
- `TestBuildFilingContext_W102_NoFireOnMatch` — accepted payroll and declared
  income match exactly. W102 absent.
- `TestBuildFilingContext_W102_NoProposals` — no payroll proposals. W102 absent.
- `TestBuildFilingContext_YearsOnFile` — history contains 2 ACK_UPLOADED and 1
  CLEARED; target is DRAFT. `YearsOnFile` = 3.

Each test builds Projections using `returns.Replay` from an event sequence. Never
reach into the projection mid-construction.

### 6. Harness and CI

`go test ./... -count=1` stays green. The Registry and coordinator packages must not
import a YAML library; they take the unmarshal function as an argument. The golden
test harness change must not break any of G-001…G-042.

## Do not do

- Do not create `rules/ng-2025.yaml`, `rules/ng-2024.yaml`, or `rules/ng-2023.yaml`.
  Those files require a CITN practitioner to verify the pre-reform regime (CRA,
  old bands, 1% minimum tax). That engagement is not complete. Fabricating
  historical legal content defeats the entire design of the sign-off gate.
- Do not add `if year == X` anywhere in `taxengine/engine.go` or `penalty.go`.
  The point of this task is to prove that is never necessary.
- Do not implement cross-year loss relief or FTC carryforward. Both require a
  deeper rethink of the `TaxFacts` boundary.
- Do not build a database, an HTTP server, or a migration tool.
- Do not change any figure in `rules/ng-2026.yaml`.
- Do not mark any fixture `review.status` as verified.
- Do not use `float64` for money.
- Do not call `time.Now()` in any test subject.

## Report back with

- Package inventory: what was created, what it imports, one-line purpose.
- Demonstration that `G-043` passes under the test-fixture rule set and would fail
  (different expected tax) if the harness mistakenly loaded ng-2026 for year 1.
- `BuildFilingContext` mapping table: which Projection fields feed which
  `FilingContext` fields, and the precise rule for each nil/zero edge case.
- W105 test results: all eight required scenarios, pass/fail, and the specific
  `PriorYearGrossKobo` value used in each.
- W102 test results: all three required scenarios.
- Confirmation that the golden harness change leaves G-001…G-042 unaffected.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` output.
