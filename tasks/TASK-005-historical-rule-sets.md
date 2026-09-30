TASK-005 — Historical Rule Sets and Multi-Year Coordination

## Context (read this first — you have no prior conversation history)

This repo is the backend for a Nigerian personal income tax prep product (Shape A:
self-preparation). TASKs 001–004 built the engine, the golden corpus, the lifecycle
integration, the submission pack, and the TCC readiness projection. 

What exists now is an engine natively aware only of the `ng-2026.yaml` rule set.
However, the TCC readiness checklist built in TASK-004 explicitly demands three
consecutive years of filed and paid assessments (`T1` and `T4` conditions). A user
cannot achieve TCC readiness unless the backend calculates 2025, 2024, and 2023
returns.

If the engine's data-driven design holds, it should support historical years entirely
by authoring new YAML files and loading them, absolutely without version-sniffing
(`if year == 2023`) inside the pure `taxengine` logic. In addition, the validation
catalogue must become multi-year aware. Specifically, implementing the W105 warning
("income fell by more than 50% from prior year") requires a coordination layer that
extracts prior year data from an adjacent projection and correctly feeds it into the
filing context.

Read in full before touching code:

- `tasks/TASK-004-submission-pack-and-tcc-readiness.md` (what TCC expects)
- `rules/ng-2026.yaml` (the baseline schema)
- `taxengine/engine.go` (assert its pure, stateless, rule-set-driven nature)
- `internal/validate/validate.go` (the `FilingContext` structure)

## Prerequisite: TASK-004 must be done

`go test ./... -count=1` must pass, and `internal/pack`, `internal/tcc`, and
`internal/portalmap` must exist. If they do not, finish TASK-004 first. Do not
build multi-year logic on a foundation that lacks the TCC readiness projection
it is meant to satisfy.

## What "done" looks like

### 1. Authoring Historical Rule Sets

Create `rules/ng-2025.yaml`, `rules/ng-2024.yaml`, and `rules/ng-2023.yaml` by
using `ng-2026.yaml` as a baseline. Update `year_of_assessment` in each.
To prove the engine behaves differently based purely on data, you must encode the
following historical realities:

- **2023 Exemption Threshold**: Prior to the late-2024 minimum wage adjustment,
  the exemption threshold was `360_000_000` kobo (₦360,000 annualized). Update 
  the `full_exemption_if_at_or_below` threshold in `ng-2023.yaml` to enforce this.
- **Rule Set Versions**: Set the version string accurately reflecting the historical
  nature, e.g., `2023.1.0`. Keep the status as `DRAFT` for all of them.

### 2. Multi-Year Rule Set Registry

Update `internal/golden/fixture.go` (or a dedicated `internal/rulesyaml` registry)
to load an entire directory of rules, exposing a `Registry` interface.

```go
type Registry interface {
    Get(jurisdiction string, year int) (*taxengine.RuleSet, error)
}
```

The system must load all `.yaml` files in `rules/`, parse them, validate them, and 
store them in memory. Requesting a missing jurisdiction or year must return an explicit 
`ErrRuleSetNotFound`. Tests must prove that `Get("NG", 2023)` returns a rule set that
caps exemptions at ₦360,000, while `Get("NG", 2026)` caps it at the 2026 rate.

### 3. Historical Validation Coordination (The W105 Implementation)

`internal/validate.FilingContext` possesses a `PriorYearGrossKobo` pointer. Up until
this point, it has been mocked or nil.

Create a robust coordination function in `internal/integration` (or a new coordinator 
layer) that takes a slice of all known `Projection`s for a taxpayer.

```go
func BuildFilingContext(target *returns.Projection, history []*returns.Projection) validate.FilingContext
```

This function must:
1. Scan `history` for the projection where `YearOfAssessment == target.YearOfAssessment - 1`.
2. Ensure that historically selected projection is at least in `StateAssessed` (meaning it 
   has a valid `AssessmentSnapshot` and `ChargeableIncomeKobo` or `GrossIncomeKobo` stored).
3. Populate `target`'s `FilingContext.PriorYearGrossKobo` with the historic gross sum.
4. Pass it down to `validate.Validate`.

### 4. Golden Test Expansion

Expand the golden corpus (`testdata/golden/`) to prove multi-year engine behaviour without
altering engine code.

- **`G-043-historical-2023-exemption.yaml`**: A taxpayer earning ₦370,000 strictly from
  employment in 2023. Under 2026 rules, they would be globally exempt. Under 2023 rules, 
  they exceed the ₦360,000 threshold and must pay PAYE on the excess. Assert the tax
  payable strictly matches the historical bands.
- **`G-044-validation-w105-income-drop.yaml`**: Prove that the coordinator layer correctly
  triggers the `W105` warning when Year N is filed with ₦1M gross, while Year N-1 had 
  ₦3M gross. 

## Do not do

- Do not add `if year == 2023` anywhere inside `taxengine/engine.go` or `penalty.go`. If 
  you touch `.go` files in `taxengine` to change calculation math, you have failed the 
  architecture invariant.
- Do not implement database interfaces. Continue to rely on in-memory slices of `*Projection`.
- Do not implement cross-year loss relief (e.g. carrying forward a business loss). That 
  requires a deeper rethink of the `TaxFacts` boundary.
- Do not alter `Walkthrough.md` from TASK-004 until instructed.

## Report back with

- A summary of exactly how `Registry` was implemented and injected to the testing harnessed.
- Demonstration that `engine.go` successfully processed historical rule changes without 
  code alterations.
- An explanation of how `BuildFilingContext` safely handles scenarios where the exact 
  N-1 year was never filed (e.g., skips W105 seamlessly).
- Test execution proof (`go test -v ./... -count=1`) highlighting `G-043` and `G-044` passing.
