TASK-004 — Build the submission pack generator and TCC readiness projection

## Context (read this first — you have no prior conversation history)

This repo is the backend for a Nigerian personal income tax prep product (Shape A:
self-preparation). TASKs 001–003 built the engine, the golden corpus, the validation
catalogue, and the return lifecycle integration. What exists now:

- `taxengine.Compute` → `Assessment` + `Trace` (pure, no I/O)
- `internal/returns.Projection` — event-sourced state of a single return
- `internal/bridge.ProjectionToFacts` — bridges Projection to TaxFacts
- `internal/validate.Validate` — fitness-to-file catalogue
- `internal/returns.Apply` — state machine with four SIGN guards

What none of those tasks built is **the thing users actually interact with when filing**.
Under Shape A the taxpayer transcribes their return into the LIRS eTax portal by hand.
Every field they have to hunt for is a field they mistype. The pack is a transcription
script: every field the portal asks for, in the portal's order, with the portal's own
label text and the value to type.

The portal map lives at `portals/ng-la-etax.yaml`. It is versioned data for the same
reason the rule set is: LIRS reorders a page → publish a new map, not a deploy. The
pack generator reads the portal map and a `Projection`, and produces a `Pack` struct.
Pure Go, no I/O, clock and portal map injected.

This task also builds the `TCCReadiness` projection — the six-condition checklist
that tells a taxpayer exactly what they are missing before a Tax Clearance Certificate
can be issued. It is the best retention surface in the product.

Read in full before touching code:

- `tasks/TASK-003-return-lifecycle-integration.md` (what the foundation provides)
- `tax/04-return-lifecycle-and-pack.md` (§4 pack, §6 TCC, §8 amendments/objections)
- `portals/ng-la-etax.yaml` (the portal map — DRAFT, field structure, source paths)
- `internal/returns/events.go` (Projection fields, Fold, event catalogue)
- `internal/returns/lifecycle.go` (States, transitions, ObjectionDeadline)
- `internal/bridge/bridge.go` (existing projection→facts mapping, for context)
- `internal/rulesyaml/yaml.go` (the pattern to follow for the portal map loader)

## Prerequisite: TASK-003 must be done

`go test ./... -count=1` must pass including the TASK-003 integration tests. If it
does not, finish TASK-003 first. Do not run a pack generator against a harness that
cannot pass its own lifecycle tests.

## What "done" looks like

### 1. Portal map loader — `internal/portalmap`

New package. Mirrors `internal/rulesyaml`: YAML parsing lives here, not in the
generator. The generator takes a `PortalMap` value, never a file path.

**Types** (derive the rest from the YAML structure):

```go
type PortalMap struct {
    Portal       PortalMeta
    Preconditions []Precondition
    Navigation   []NavStep
    Pages        []Page
    Uploads      []UploadRequirement
    PostSubmission []PostStep
    TCC          TCCSection
}

type PortalMeta struct {
    Code           string
    Jurisdiction   string
    Authority      string
    MapVersion     string
    ObservedSeason int
    Status         string // "DRAFT" or "PUBLISHED"
}

type Precondition struct {
    ID          string
    Label       string
    Detail      string
    BlocksPack  bool
    BlocksTCC   bool
    GeneratesWarning string
}

type Page struct {
    ID     string
    Title  string
    Fields []PageField
    UnmappedIncome *UnmappedIncomeSpec
    Next   string
    Submit string
}

type PageField struct {
    PortalLabel   string
    Source        string // path expression, e.g. "income.EMPLOYMENT[].amount"
    Repeatable    bool
    Format        string // "naira_2dp" or empty
    Note          string
    ConditionalOn string
    Upload        string
    Enum          []string
}

type UnmappedIncomeSpec struct {
    Note       string
    Categories []string
}
```

**Loader:**

```go
type LoadOptions struct {
    AllowDraft bool
}

func Load(data []byte, unmarshal func([]byte, any) error, opts LoadOptions) (PortalMap, error)
```

Reject any portal map whose `status != "PUBLISHED"` unless `AllowDraft: true` is set.
The current `portals/ng-la-etax.yaml` has `status: DRAFT` — all tests must pass
`AllowDraft: true`. The production caller never sets it, same as the rule set gate.

Test: `Load` without `AllowDraft` rejects a DRAFT map. `Load` with `AllowDraft` returns
a valid `PortalMap`.

### 2. Accommodation declaration — new event in `internal/returns`

The portal's Page 2 ("Mandatory Disclosure of Accommodation") asks for accommodation
type, ownership type, annual rent paid, and landlord details. Annual rent paid is
already in `Projection.Bases["annual_rent_paid"]` via `EvBaseDeclared`. The rest is
not in the projection at all.

Add to `internal/returns/events.go`:

```go
EvAccommodationDeclared EventKind = "AccommodationDeclared"
```

Payload:

```go
// resolved from json in Fold
struct {
    Type          string `json:"type"`            // "Rent" | "Self-owned"
    OwnershipType string `json:"ownership_type"`  // "Tenant" | "Owner"
    LandlordName  string `json:"landlord_name"`
    LandlordAddress string `json:"landlord_address"`
}
```

Add to `Projection`:

```go
AccommodationType    string
OwnershipType        string
LandlordName         string
LandlordAddress      string
```

`Fold` handles `EvAccommodationDeclared` by setting those four fields. It does **not**
mark the assessment stale — accommodation disclosure is a portal compliance requirement,
not a change to the taxable facts. Do not add `markStale()` here.

Test: emit `EvAccommodationDeclared`, replay, assert all four fields on the Projection.
Also confirm the assessment (if any) is not marked stale.

### 3. Assessment trace in `AssessmentSnapshot`

The design doc (§3 of `02-engine-and-employer.md`) is explicit: "The trace is stored,
not regenerated." A pack regenerated later against a newer rule set would produce a
different explanation of the same historical number, which is worse than having none.

Currently `AssessmentSnapshot` has no trace field. Add:

```go
Trace []taxengine.Step `json:"trace,omitempty"`
```

Update `Fold`'s handling of `EvAssessmentComputed` to parse and store `Trace` from the
payload. Update every place that constructs an `EvAssessmentComputed` event (in tests
and the integration layer) to include the trace in the payload. The pack generator
populates `Pack.AssessmentTrace` from `Projection.Assessment.Trace`.

If the projection's assessment has no trace (e.g., replaying a legacy event that
predates this field), `Pack.AssessmentTrace` is nil — not an error.

### 4. Pack generator — `internal/pack`

New package. Imports: `internal/portalmap`, `internal/returns`, `taxengine`. No I/O,
no YAML dependency.

**Core types:**

```go
type PackField struct {
    PortalLabel string
    Value       string
    Note        string
    Upload      string  // "Bring: X" if the field carries an upload requirement
    Missing     bool    // true if the projection has no value for this field
}

// A repeatable field expands to multiple rows (e.g. two employers → two rows).
type PackRow struct {
    Fields []PackField
}

type UnmappedIncomeItem struct {
    Category string
    Amount   taxengine.Kobo
    Note     string
}

type PreconditionResult struct {
    ID         string
    Label      string
    Detail     string
    BlocksPack bool
    BlocksTCC  bool
    Met        bool
}

type PackPage struct {
    ID     string
    Title  string
    Note   string
    Rows   []PackRow
}

type Pack struct {
    ReturnID         string
    YearOfAssessment int
    PortalMapVersion string
    Pages            []PackPage
    UnmappedIncome   []UnmappedIncomeItem
    Preconditions    []PreconditionResult
    AssessmentTrace  []taxengine.Step
}
```

**Generation input and function:**

```go
type GenerateInput struct {
    Projection       *returns.Projection
    PortalMap        portalmap.PortalMap
    // Caller-provided precondition state (profile, photograph, etc.).
    // Key is the precondition ID ("PC01", "PC02", ...).
    // If a key is absent, the corresponding PreconditionResult has Met: false.
    PreconditionsMet map[string]bool
}

func Generate(in GenerateInput) (Pack, error)
```

`Generate` must:

1. **State check.** Reject projections whose state is earlier than `SIGNED`
   (`StateDraft`, `StateValidated`). A pack cannot be generated for an unsigned return.
   Any state from `StateSigned` onward is accepted.

2. **Preconditions.** Build one `PreconditionResult` per precondition in the portal
   map, marking `Met` from `in.PreconditionsMet`. A precondition absent from the map
   is not an error. The pack is returned regardless of precondition state — the caller
   decides whether to surface a blocking warning. `Generate` reports, it does not abort.

3. **Page and field mapping.** Walk each page and field in the portal map. For each
   field, resolve the source path against the projection using the resolver (§5 below).
   Non-repeatable fields produce a single `PackRow` with one `PackField`. Repeatable
   fields produce one `PackRow` per resolved value.

4. **Unmapped income.** After mapping all pages, collect any income category listed in
   `page.UnmappedIncome.Categories` that has a non-zero total in the projection. Exclude
   `GRATUITY` from the unmapped list — it is mapped on Page 3 (see §6 below for the
   finding to report). Include the remaining categories in `Pack.UnmappedIncome`.

5. **Assessment trace.** Copy `Projection.Assessment.Trace` into `Pack.AssessmentTrace`
   if non-nil. If `Projection.Assessment` is nil, leave `AssessmentTrace` nil.

6. Return the `Pack`.

`Generate` does **not** validate the return. Blocking errors, missing reliefs, and
other fitness-to-file concerns are `internal/validate`'s job. The generator maps what
is there; it does not fail because something is missing from the projection.

### 5. Source path resolver

Write an unexported `resolve` function (or method) that maps a portal map source path
to one or more (label, value) pairs from the projection.

Required mappings — derive the rest from the portal map's source fields:

| Portal map source | Resolution |
|---|---|
| `income.EMPLOYMENT[].source_name` | One entry per EMPLOYMENT income line; value is `IncomeLine.Label`; repeatable |
| `income.EMPLOYMENT[].amount` | One entry per EMPLOYMENT income line; value is `IncomeLine.AmountKobo` formatted as naira |
| `income.COMMISSION` | Sum of all COMMISSION income lines |
| `income.BUSINESS + income.PROFESSIONAL` | Sum of both categories; note shows components |
| `income.ALLOWANCES` | Sum of ALLOWANCES income lines |
| `income.PENSION_RECEIVED` | Sum of PENSION_RECEIVED income lines (pension *received*, not contributed) |
| `income.INTEREST` | Sum of INTEREST income lines |
| `income.RENT_RECEIVED` | Sum of RENT_RECEIVED income lines (rent received as landlord, not rent paid) |
| `income.DIVIDEND` | Sum of DIVIDEND income lines |
| `income.FOREIGN` | Sum of FOREIGN income lines |
| `income.GRATUITY` | Sum of GRATUITY income lines |
| `profile.residential_address` | Not in Projection → `Missing: true`, note: "Enter from your profile" |
| `accommodation.type` | `Projection.AccommodationType`; if empty → `Missing: true` |
| `accommodation.ownership` | `Projection.OwnershipType`; if empty → `Missing: true` |
| `accommodation.landlord` | `Projection.LandlordName + ", " + Projection.LandlordAddress`; if both empty → `Missing: true` |
| `declared_bases.annual_rent_paid` | `Projection.Bases["annual_rent_paid"]` formatted as naira; if absent → `Missing: true` |
| `reliefs.PENSION` | `Projection.Reliefs["PENSION"].AmountKobo` formatted as naira; if absent → zero (not Missing) |
| `reliefs.NHF` | `Projection.Reliefs["NHF"].AmountKobo`; absent → zero |
| `reliefs.NHIS` | `Projection.Reliefs["NHIS"].AmountKobo`; absent → zero |
| `reliefs.LIFE_ASSURANCE` | `Projection.Reliefs["LIFE_ASSURANCE"].AmountKobo`; absent → zero |

Money format: naira with two decimal places, comma-grouped, ₦ prefix. Use the same
format as `taxengine.Kobo.String()` (or a helper derived from it). Never `float64`.

Conditional fields: if a field carries `conditional_on` and the condition is false
(e.g. `accommodation.type != "Rent"`), skip it — do not produce a `PackField` at all.
"Rent Received" and "Annual Rent Paid" are different fields; do not conflate them.

Fields with no matching source path in the resolver must produce `Missing: true` with
a note explaining why, not be silently dropped.

### 6. GRATUITY ambiguity — resolve and report

`portals/ng-la-etax.yaml` Page 1 lists `GRATUITY` in `unmapped_income.categories`.
Page 3 has a mapped field with `source: income.GRATUITY` and a note that "It appears
on the reliefs page but it is income." These are contradictory.

Resolution: treat `GRATUITY` as mapped (it appears on Page 3). Do not flag it in
`Pack.UnmappedIncome`. The unmapped categories for Page 1 are therefore `DIGITAL_ASSET_GAIN`,
`CAPITAL_GAIN`, and `OTHER` only.

Report this in the "Report back with" section as a portal map finding. Do not edit
`portals/ng-la-etax.yaml`.

### 7. TCC readiness — `internal/tcc`

New package. Pure: no I/O, no database. Takes a summary slice and profile flags,
returns a `TCCStatus`. Imports only `internal/returns` (for the `State` type) and
the standard library.

```go
type YearSummary struct {
    YearOfAssessment int
    State            returns.State
    AssessmentServed bool // Projection.AssessmentServedAt != nil
    BalancePaid      bool // Projection.PaidAt != nil, or State is CLEARED/PAID
}

type ReadinessInput struct {
    // Three most recent years of assessment, in any order.
    // If fewer than three are provided, T1 and T4 will not be met.
    Years            []YearSummary
    HasPassportPhoto bool // T2: passport photograph on eTax profile
    HasFullAddress   bool // T3 and T6: address and profile complete
}

type Condition struct {
    ID   string // "T1" through "T6"
    Label string
    Met  bool
    Note string // empty when Met is true; explanation when false
}

type TCCStatus struct {
    Conditions []Condition // always six, in T1–T6 order
    AllMet     bool
}

func ComputeReadiness(in ReadinessInput) TCCStatus
```

Conditions and rules:

| ID | Label | Met when |
|---|---|---|
| T1 | Annual returns filed for the last three years | Three `YearSummary` entries with `State == StateAckUploaded` or `State == StateCleared` |
| T2 | Passport photograph uploaded | `in.HasPassportPhoto` |
| T3 | Full address updated | `in.HasFullAddress` |
| T4 | Tax assessments received for the last three years | Three entries with `AssessmentServed: true` |
| T5 | All taxes due fully paid | All provided entries have `BalancePaid: true` |
| T6 | Profile complete | `in.HasFullAddress` (same signal as T3; profile and address are the same gate) |

`AllMet` is true only if all six `Condition.Met` values are true. Populate `Note`
with a short, user-facing explanation for every condition that is not met.

### 8. Test suite

**`internal/portalmap/portalmap_test.go`:**

- `TestLoad_DraftRejected` — call `Load` on the real `portals/ng-la-etax.yaml` without
  `AllowDraft`; expect an error containing "DRAFT".
- `TestLoad_AllowDraft` — same file with `AllowDraft: true`; expect no error and
  `PortalMap.Portal.MapVersion == "0.1.0-draft"`.

**`internal/returns/events_test.go` (extend or add):**

- `TestFold_AccommodationDeclared` — emit `EvAccommodationDeclared`, replay, assert
  all four accommodation fields on the projection.
- `TestFold_AccommodationDoesNotMarkStale` — emit `EvAssessmentComputed` then
  `EvAccommodationDeclared`; assert `AssessmentStale` is still false.

**`internal/pack/pack_test.go`:**

Fixtures: build projections in-memory using `Replay`, never reach in mid-fold. Each
scenario uses `EvReturnOpened` → relevant domain events → `EvReturnSigned` (and
`EvAssessmentComputed` where the pack's trace appendix is needed).

- `TestGenerate_HappyPath` — employment income, pension relief, declared rent base,
  accommodation declared as Rent. Assert: Page 1 has Employer Name and Salary rows,
  Page 3 has pension, Page 2 has accommodation type and annual rent paid. No `Missing:
  true` on any resolvable field.

- `TestGenerate_MultiEmployer` — two EMPLOYMENT income items (different source refs).
  Assert Page 1 produces two Employer Name rows and two Salary rows (repeatable field
  expansion).

- `TestGenerate_UnmappedIncome` — projection with `DIGITAL_ASSET_GAIN` income.
  Assert `Pack.UnmappedIncome` has exactly one entry with the correct category and
  formatted amount. Assert `GRATUITY` income (if present) is NOT in UnmappedIncome.

- `TestGenerate_MissingAccommodation` — employment income, rent relief claimed, but
  `EvAccommodationDeclared` never emitted. Assert Page 2 accommodation fields have
  `Missing: true`.

- `TestGenerate_TooEarlyState` — projection in `DRAFT` state. `Generate` must return
  an error. Projection in `VALIDATED` state must also return an error.

- `TestGenerate_SignedStateAccepted` — projection in `SIGNED` state. `Generate` must
  succeed (pack can be previewed before `ISSUE_PACK` transition fires).

- `TestGenerate_PreconditionsMet` — call with `PreconditionsMet{"PC01": true, "PC03": false}`.
  Assert `Pack.Preconditions` contains PC01 with `BlocksPack: false, Met: true`, PC03
  with `BlocksPack: false, BlocksTCC: true, Met: false`. Assert the pack is still
  generated (preconditions do not abort generation).

- `TestGenerate_PC01NotMet` — `PreconditionsMet` has no entry for PC01. Assert PC01
  result has `BlocksPack: true, Met: false`.

- `TestGenerate_AssessmentTrace` — project includes `EvAssessmentComputed` with a
  non-empty trace. Assert `Pack.AssessmentTrace` is populated.

- `TestGenerate_ConditionalField_SelfOwned` — `AccommodationType: "Self-owned"`. Assert
  "Annual Rent Paid" and "Landlord Details" fields are absent from the pack (conditional
  on Rent).

**`internal/tcc/tcc_test.go`:**

- `TestReadiness_AllMet` — three years, all `ACK_UPLOADED`, all assessments served, all
  paid, passport and address present. `AllMet: true`.

- `TestReadiness_OnlyTwoYears` — two `YearSummary` entries. T1 not met; note mentions
  three years required.

- `TestReadiness_OneYearUnpaid` — three years filed and assessed, but one `BalancePaid:
  false`. T5 not met.

- `TestReadiness_MissingPassport` — `HasPassportPhoto: false`. T2 not met.

- `TestReadiness_MixedStates` — one year `ACK_UPLOADED`, one year `CLEARED`, one year
  `ASSESSED` (not yet uploaded). T1 not met because the third year is not filed.

### 9. Harness and CI

`go test ./... -count=1` stays green. No new external dependencies in `internal/pack`,
`internal/tcc`, or `internal/portalmap`. YAML parsing belongs in `internal/portalmap`
only — the generator and TCC packages have no YAML dependency.

The import isolation test from TASK-001 (`taxengine` and `internal/returns` import
nothing outside the standard library) must still pass. `internal/pack` importing
`taxengine` for types is fine; `taxengine` importing `internal/pack` is not.

## Do not do

- Do not change any figure in `rules/ng-2026.yaml`.
- Do not change any field, label, or value in `portals/ng-la-etax.yaml`. The GRATUITY
  ambiguity is a finding to report, not a licence to edit the map.
- Do not flip any `status` to `PUBLISHED` or any fixture `review.status` to verified.
- Do not build an HTTP server, PDF renderer, or frontend wizard. The pack is a Go
  struct, not a document.
- Do not build the multi-year filing UI, the objection lodgement flow, or pre-2026
  rule sets. Those are later tasks.
- Do not implement foreign-tax-credit capping.
- Do not use `float64` for money anywhere.
- Do not call `time.Now()` in any test subject. Inject the clock where needed.
- Do not add external dependencies to `taxengine`, `internal/returns`, `internal/pack`,
  `internal/tcc`, or `internal/portalmap`.
- Do not add a database, a file system dependency, or HTTP to any package in `internal/`.

## Report back with

- **Package inventory:** what was created or modified, what it imports, one-line purpose.
- **Source path mapping table:** every portal map source path, the Projection field or
  expression it resolves to, and whether it is `Missing: true` by design.
- **GRATUITY finding:** one paragraph explaining the conflict in the portal map and
  the resolution chosen.
- **Accommodation event:** payload schema, how Fold handles it, confirmation it does
  not mark the assessment stale.
- **AssessmentSnapshot.Trace:** what changed in the payload, how existing tests were
  updated, and the behaviour when replaying a legacy event without a trace.
- **TCC test results:** pass/fail for all five required scenarios and which T-conditions
  each exercises.
- **Any other portal map ambiguity** you hit while implementing the resolver (source
  paths that were under-specified, conditional logic that needed interpretation), and
  how you resolved it.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` output.
