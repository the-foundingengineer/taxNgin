TASK-002 — Expand the golden corpus to the starter forty, and build the validation catalogue the engine is forbidden to own

## Context (read this first — you have no prior conversation history)

This repo is the backend for a Nigerian personal income tax prep product.
TASK-001 bootstraps the Go module, proves `taxengine.Compute` against five
hand-authored fixtures (G-001…G-005), and wires CI. That foundation is
necessary but not sufficient.

`tax/02-engine-and-employer.md` §4 names a **starter forty**: five fixtures
are written; the rest are specified by what they defend against (band
boundaries, exemption edges, rent-cap off-by-ones, relief interactions,
multi-source aggregation, credit/balance signs, and NTAA s.101 penalties).
TASK-001 explicitly forbids adding those cases. This task is that follow-on.

It also forbids mixing validation into `Compute`. Computation assumes
well-formed facts and errors on malformed ones; fitness-to-file is a
different job (`tax/02-engine-and-employer.md` §5). Several starter-forty
cases (rent claimed without evidence; unknown relief code; nil return) are
not `Compute` assertions — they are validation assertions. If you only add
YAML files and never build `Validate`, you have not executed the corpus.

Read in full before touching code:

- `tasks/TASK-001-bootstrap-and-verify-engine.md` (what already must be true)
- `tax/02-engine-and-employer.md` (§4 corpus, §5 validation catalogue)
- `tax/04-return-lifecycle-and-pack.md` (§5 penalty table)
- The five existing fixtures under `testdata/golden/G-00{1..5}-*.yaml`
- `rules/ng-2026.yaml` (do not change any figure)

## Prerequisite: TASK-001 must actually be done

If you find no `go.mod`, no `*_test.go`, no CI, a mixed-package `tax/`
directory, or `go test ./...` failing, **finish TASK-001 first** as a
prerequisite, then continue. Do not paper over a missing harness by
hard-coding numbers in ad-hoc tests. The starter forty rides the same
harness as G-001…G-005.

Known design bugs you may hit while the prerequisite tests land — fix them,
and say so in the report:

- `TaxFacts.Hash` shallow-copies the struct; `sort.Slice` mutates the
  caller's slices. Hashing must be a pure function.
- `SIGN` in `internal/returns/lifecycle.go` is documented as four independent
  guards (blocking errors, stale/missing assessment, unpublished rule set,
  `DeclaredByTaxpayer == false`). If the code implements fewer than four,
  that is a bug, not a simplification.
- `Compute` does not consult `ReliefClaim.HasEvidence`. That is correct.
  Evidence is a validation concern (`W101`). Do not silently skip rent
  relief inside the engine because a fixture lacks a PDF.

## What "done" looks like

### 1. One fixture file per case, practitioner-readable

Continue the existing format (`id`, `title`, `why_this_case`, `given`,
`expect`, `review`). Amounts in naira (two decimal places maximum, parsed
as exact kobo — never `float64` arithmetic). Every new file starts as
`review.status: UNVERIFIED`. Do not mark anything verified.

Add a `kind` field so the harness can dispatch:

| kind | Subject |
|---|---|
| `compute` (default) | `taxengine.Compute` vs `expect` |
| `error` | `Compute` must fail; `expect.error` is a substring |
| `penalty` | `taxengine.ComputePenalties` vs `expect` with an injected clock |
| `validate` | `internal/validate` vs `expect.issues` (code + severity) |

Do **not** edit `expect` numbers in G-001…G-005. If a new fixture's
hand-worked `expect` disagrees with the engine, report it — do not rewrite
the fixture to match the engine, and do not rewrite the engine to match a
guess. If you are sure the engine is wrong, fix the engine and explain.

### 2. The rest of the starter forty

Group IDs sequentially from G-006. Existing files keep their IDs. Where
the design lists a case that is already G-001…G-005, **do not duplicate
it** — point at the existing file in `why_this_case` of the nearest
sibling, and still hit the untested edge with a new file.

**Band boundaries (8).** Chargeable income at exactly ₦800,000 and
₦800,001; ₦3,000,000 and ₦3,000,001; ₦50,000,000 and ₦50,000,001; ₦0; ₦1.

The minimum-wage exemption fires on employment-only income at or below
₦840,000. Band-boundary cases that would otherwise be exempt **must use a
non-employment category** (e.g. `DIVIDEND` / `BUSINESS`) so you are
actually testing the band table. Document that choice. ₦0 of income will
likely trip the exemption short-circuit — that is a finding to record in
`why_this_case`, not a reason to skip the file.

**Exemption edges (5).** G-001 already covers a minimum-wage earner with
PAYE wrongly deducted. Still needed: exactly at threshold with no PAYE;
one kobo above; part-year employment that annualises above the threshold
but earns below it; ₦1 of non-employment side income breaking the
employment-only scope.

**Rent relief (5).** G-002 is rent of ₦2,500,000 (exactly on cap). Still
needed: ₦2,499,999; ₦2,500,001; zero rent declared; rent claimed with
`has_evidence: false` (this last one is `kind: validate`, warning `W101` —
Compute must still be willing to grant the relief; validation is what
stops the file).

**Relief interactions (6).** All reliefs defined in `ng-2026.yaml` claimed
at once (the design says "seven"; the file has eight — use the file, note
the discrepancy). Reliefs exceeding gross income, clamping chargeable
income at zero, never negative. Pension claimed with no employment income
(engine currently has no category gate — do not invent one; record
VERIFY). A relief code absent from the rule set (`kind: error`). A
`rate_of_base` relief whose base is missing (allowed amount 0, not an
error). One further interaction of your choosing that is not already
covered, with a real `why_this_case`.

**Multi-source (6).** G-005 is salary + freelance + rent. Still needed:
two employers, mid-year job change (sum, never overwrite); salary + rental
+ dividends; digital asset gain as the only income; foreign income with a
foreign tax credit **written as a visible v1.5 gap** (do not implement
FTC capping; do not let the fixture silently treat FTC like PAYE without
saying so); one further multi-source case.

**Credits and balances (4).** Over-withheld → refund; under-withheld →
payment; exactly withheld → zero balance; credits exceeding liability.
Use a non-exempt salary so these are credit tests, not exemption tests.
G-001 is an exemption+refund; it does not replace this group.

**Penalties (6).** Against the 31 March deadline in the rule set, with an
injected clock, never `time.Now()`: filed one day late; one month late;
six months late; filed on time but paid late (surcharge + interest; MPR
is an argument); nil return filed late (₦100,000 filing penalty on ₦0
tax); on-time filing (₦0). Worked numbers in
`tax/04-return-lifecycle-and-pack.md` §5 are the oracle for the filing
leg. Late-payment interest fixtures must state days, MPR and unpaid tax
explicitly.

### 3. `internal/validate` — the catalogue, as code

New package. Imports: standard library plus `taxengine` (types only).
No I/O, no clock unless injected, no YAML dependency.

Implement every code in the §5 table that can be decided from facts +
rule set + a small `FilingContext` (tax ID, NIN, nil-declaration flag,
accommodation type, employer-reported gross, prior-year income, years on
file, filing timestamp, deadline). Codes that need a database
(`E010` duplicate return) or a portal (`E004` SIRS resolution) must still
exist as functions that return a clear `ErrNotEnoughContext` rather than
being silently omitted — the catalogue is closed; absence is a bug.

Every issue is `{code, severity, field, message, citation}`. Tests for at
least: `E005`, `E006`, `E007`, `E008`, `W101`, `W103`, `W109`, `W110`.

### 4. Harness and CI

Extend `taxengine` golden tests (or a `internal/golden` helper used by
both `taxengine` and `internal/validate`) so one walk over
`testdata/golden/*.yaml` dispatches on `kind`. `go test ./...` stays green
for development even though every fixture is `UNVERIFIED`. The unsigned-
corpus **release** gate from TASK-001 must keep failing until a
practitioner signs the files — more UNVERIFIED files make that gate more
correct, not less.

Do not add the ~35 cases as Go table tests instead of YAML. The YAML is
the review artefact.

## Do not do

- Do not change any figure in `rules/ng-2026.yaml`.
- Do not flip `DRAFT` → `PUBLISHED` or fill `approved_by`.
- Do not mark any fixture `review.status` as verified.
- Do not implement foreign-tax-credit capping, pre-2026 rule sets, HTTP,
  a database, a portal pack renderer, or employer CSV ingest.
- Do not add third-party libraries to `taxengine`, `internal/returns`, or
  `internal/validate`. YAML belongs in tests / a loader helper.
- Do not use `float64` for money.

## Report back with

- Fixture inventory: ID, kind, group, pass/fail.
- Full detail on every disagreement (field, expected vs actual, engine bug
  vs fixture error vs known gap).
- The "seven vs eight reliefs" note, the ₦0-income/exemption finding, and
  the FTC gap, each in one paragraph.
- Any engine or lifecycle bug you fixed, and why it was a bug rather than
  a preference.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` output.
