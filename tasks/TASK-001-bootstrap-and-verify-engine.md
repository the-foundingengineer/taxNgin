TASK-001 — Bootstrap the Go module and prove the tax engine actually compiles and computes correctly

## Context (read this first — you have no prior conversation history)

This repo is the backend for a Nigerian personal income tax prep product (a
TurboTax/FreeTaxUSA-style self-preparation tool for Nigeria). The domain
design and Go source in `tax/` were produced in a design conversation with an
LLM and **have never been compiled or run**. There is no `go.mod`, no test
files, no CI, and no git repository. This task turns that design into a real,
verified Go module. It is pure engineering — it does not require the legal
opinion, the CITN practitioner sign-off, or the year-of-assessment decision
that are still open elsewhere in the project (see "Do not do" below).

Read these two design docs in full before touching code — they explain *why*
the code is shaped the way it is, and several of your test cases come
directly from them:

- `tax/02-engine-and-employer.md`
- `tax/04-return-lifecycle-and-pack.md`

## Current state of `tax/`

Flat directory, no subfolders:

- `money.go`, `rules.go`, `engine.go`, `penalty.go` — `package taxengine`. Pure
  business logic: integer kobo arithmetic, versioned/signed rule sets, the
  `Compute` function producing an `Assessment` + `Trace`, penalty accrual.
  These files import **only the standard library** — that is a deliberate,
  load-bearing property (see §7 of `02-engine-and-employer.md`), not an
  accident.
- `events.go`, `lifecycle.go` — **`package returns`**, not `taxengine`. This
  is a concrete bug you will hit immediately: two different package names
  currently sit in the same directory, which Go will refuse to build. The
  design docs describe them living at `internal/returns/` — that's the fix,
  not a workaround.
- `ng-2026.yaml` — the NG 2026 rule set. `status: DRAFT`, `approved_by` empty.
  This is intentional and must stay that way (see "Do not do").
- `ng-la-etax.yaml` — the LIRS portal transcription map. Not exercised by the
  engine; leave it alone for this task.
- `G-001` … `G-005` `*.yaml` — five golden test fixtures, each with
  `given` (inputs), `expect` (outputs), and `review: {status: UNVERIFIED}`.
- Stray `*:Zone.Identifier` files — Windows download metadata, not part of
  the project. Do not add them to git.

## What "done" looks like

### 1. Real module layout

Reorganize `tax/` into the structure the docs already describe:

```
taxengine/          money.go, rules.go, engine.go, penalty.go
internal/returns/   events.go, lifecycle.go
rules/               ng-2026.yaml
portals/             ng-la-etax.yaml
testdata/golden/    G-00{1..5}-*.yaml
```

Initialize git at the repo root (`/home/theprinter/tax-software`), add a
`.gitignore` that excludes `*:Zone.Identifier` and normal Go build artifacts,
and make an initial commit before you start changing code, so the as-received
state is recoverable.

Add a `go.mod` at the module root (Go 1.22+). Pick a reasonable module path
(a placeholder like `tax-software` or a real eventual GitHub path is fine —
note in your final report which you chose and why).

### 2. It builds

`go build ./...` and `go vet ./...` succeed with zero errors. Fix whatever
you find — likely candidates: the package split above, YAML struct tag
mismatches, unused imports, the odd type mismatch. Where the fix is
non-obvious or changes documented behavior, say so explicitly in your report
rather than silently picking an interpretation.

`taxengine.Load` takes an injected `unmarshal func([]byte, any) error` —
that's deliberate, so the engine package itself never needs a YAML
dependency. Add `gopkg.in/yaml.v3` (or equivalent) only in test code / a
small loader helper outside `taxengine` and `internal/returns`.

### 3. Golden fixtures actually pass

Write a test harness (e.g. `taxengine/golden_test.go`) that, for each file in
`testdata/golden/`:

- Parses the fixture's `given` into `taxengine.TaxFacts`.
- Loads `rules/ng-2026.yaml` via `taxengine.Load(..., LoadOptions{AllowDraft: true})` — `AllowDraft` is correct here because this is a test, per the comment in `rules.go`.
- Calls `Compute` and asserts every field under `expect` matches: gross
  income, each relief by code, total reliefs, chargeable income, the
  band-by-band tax breakdown, tax payable, total credits, and the signed
  balance.

**If the engine's output disagrees with a fixture's `expect` block, that is a
bug to report, not something to paper over.** Do not edit the expected
numbers in a `G-00*.yaml` file to make a test pass — those numbers are what a
CITN practitioner will eventually sign off on, and silently changing them
would misrepresent what was reviewed. If you're confident the engine has a
genuine bug (not a fixture error), fix the engine and explain the fix; if
you're unsure which side is wrong, leave the test failing and flag it clearly
in your report.

### 4. Unit tests for the pieces the golden fixtures don't reach

- `money.go`: half-up rounding including negative amounts, overflow detection
  in the scaled-multiply path, `Naira`/`Kobo` round-tripping, `String()` /
  `Format()` output (comma grouping, ₦ prefix).
- `rules.go` `Validate`: rejects a DRAFT rule set unless `AllowDraft` is set;
  rejects a PUBLISHED rule set with no approver; rejects bands that are out
  of order or a final band with a non-nil upper bound; rejects a duplicate
  relief code; rejects a `rate_of_base` relief with no base or rate; rejects
  an unknown rounding mode or a unit that isn't `kobo`.
- `penalty.go`: reproduce the worked table in §5 of
  `04-return-lifecycle-and-pack.md` (₦0 on time, ₦100,000 at one day late,
  ₦350,000 at six months, ₦650,000 at twelve months) with an injected clock —
  never `time.Now()` in the test subject.
- `internal/returns/lifecycle.go`: the four `SIGN` guards each block
  independently — outstanding blocking validation errors, a stale or missing
  assessment, a non-PUBLISHED rule set, `DeclaredByTaxpayer == false`.
- `internal/returns/events.go` `Fold`: unknown event kinds are tolerated (bump
  version, no error), and folding is pure (no I/O) — same-input replay
  produces an identical projection.

### 5. Property tests

Table-driven or loop-generated (stdlib `testing` is enough — no need for
`testing/quick` or an external library) coverage of the four invariants named
explicitly in §4 of `02-engine-and-employer.md`:

- Tax payable is monotonic non-decreasing in chargeable income.
- Take-home (income − tax) never decreases as income increases by one naira —
  the no-cliff property.
- Effective rate never exceeds the top marginal band rate.
- `Compute` is idempotent, and equal `TaxFacts` produce equal `Hash()`.

### 6. The two structural CI gates the design calls for but nobody built yet

Both are named explicitly in the docs — implement them for real:

- **Import isolation** (§7 of `02-engine-and-employer.md`): an automated
  check that `taxengine` and `internal/returns` import nothing outside the
  Go standard library. `go list -deps` or parsing `go list -json` output both
  work; either a Go test or a small CI script is fine.
- **Unsigned-corpus gate** (§1 and §4 of `02-engine-and-employer.md`): CI must
  fail a *release* build if any fixture under `testdata/golden/` still has
  `review.status: UNVERIFIED`. Development/PR runs must **not** be blocked by
  this — right now all five fixtures are legitimately unverified, and that's
  correct until a practitioner signs them. Implement this as a separate
  script/job invoked only on a release workflow (e.g. a git tag push), not as
  part of the default `go test ./...`.

Wire both into a CI workflow (e.g. `.github/workflows/ci.yml`) that runs
`go build ./...`, `go vet ./...`, `go test ./... -race`, and the import
isolation check on every push/PR, plus a separate release job that adds the
unsigned-corpus gate.

## Do not do

- Do not change any figure in `rules/ng-2026.yaml` (bands, relief caps,
  thresholds, penalty amounts). Those are legal content, not code, and are
  out of scope here.
- Do not flip `status: DRAFT` to `PUBLISHED` or fill in `approved_by`. That
  requires an actual CITN-registered practitioner's sign-off, which hasn't
  happened yet — the loader is deliberately built to refuse this, and that
  refusal must keep working.
- Do not mark any `G-00*.yaml` fixture's `review.status` as verified.
  Same reason.
- Do not add the other ~35 planned golden fixtures listed in §4 of
  `02-engine-and-employer.md` ("The starter forty") — that's a follow-on task
  once this foundation is solid, and picking the right edge cases is a
  product/practitioner concern, not this task's.
- Do not touch `portals/ng-la-etax.yaml`, build a database schema, an HTTP
  API, auth, or any frontend. All out of scope for this task.
- Do not add dependencies to `taxengine` or `internal/returns` beyond the
  standard library, for any reason.

## Report back with

- The diff/PR, plus `go build ./...`, `go vet ./...`, and `go test ./... -v`
  output.
- Explicit pass/fail for each of the 5 golden fixtures, and full detail on
  any that fail (which field, expected vs. actual, your read on whether it's
  an engine bug or a fixture error — without editing the fixture to resolve
  it yourself).
- Any place you had to make a judgment call to get something to compile or
  pass, and why.
