# Part 2 — Tax Engine and Employer Channel

**Decisions carried in from Part 1:** Shape A (self-preparation), Go, employer channel modelled from the start, no practitioner yet.

That last one is the biggest risk in the project, so it goes first.

---

## 1. The practitioner problem

You have a tax engine and no one qualified to say its numbers are right. Until that changes you have a calculator with opinions. Two things follow, one defensive and one practical.

### The defensive move: make it impossible to ship unsigned

`rules/ng-2026.yaml` carries `status: DRAFT` and an empty `approved_by`. `RuleSet.Validate` refuses to load anything that is not `PUBLISHED`, and refuses a `PUBLISHED` rule set with no approver on record. The only escape is `LoadOptions{AllowDraft: true}`, which exists for tests and the authoring tool.

Wire it so production never sets that flag — a constant in the config package, not an environment variable someone can flip at 2am during filing season. This is cheap now and impossible to retrofit after the first time an unsigned number reaches a taxpayer.

Every golden fixture likewise carries `review: {status: UNVERIFIED}`. Make CI fail the release build if any fixture in the corpus is still `UNVERIFIED`. Development runs fine; you simply cannot cut a release until the review is done.

### The practical move: get the sign-off cheaply

You do not need a retainer, a Big Four firm, or a partner. You need a fixed-scope review of a document.

**Who.** CITN maintains a members' register; the district societies in Lagos and Abuja are the practical entry point. What you want is an ACTI or FCTI in independent practice who does PAYE and individual returns day to day — not a transfer-pricing specialist, not a corporate tax partner. Someone who has personally filed a few hundred Form As knows where the portal and the statute disagree, which is knowledge you cannot get from a PDF.

**What you ask them to sign.** Not code. Two artefacts:

1. The rule set, printed as a one-page table of bands, reliefs, caps, exemptions and penalties with statutory citations. They tick or correct each line.
2. The golden fixtures, which is why those files are written in naira with the band walk spelled out. `testdata/golden/G-004-pwc-sample-25m.yaml` is deliberately readable by someone who has never opened a terminal.

**How to scope it.** A fixed fee for a first pass over roughly forty cases, plus a smaller annual renewal when the Finance Act moves the numbers, plus an hourly rate for questions during the year. Frame it as a recurring engagement, because it is — every rule set version needs a fresh signature, and that is a feature of the design, not an accident.

**What you can do before you find one.** Reconcile against published worked examples. The corpus already does this: `G-003` expects ₦12,930,000 on a ₦60,000,000 salary, which matches an independently published PAYE worked example exactly. `G-004` mirrors the fact pattern of PwC's published sample calculation, and the fixture carries an explicit action to pull PwC's stated figure and confirm it. Two unrelated parties reaching the same number is real corroboration. It is not sign-off, and the code will not let you pretend otherwise.

**One thing to ask them on day one:** whether the 2026 filing is year of assessment 2025 or 2026, and what the basis period is for employment versus business income. That ambiguity from Part 1 is still open and it is the one that corrupts your schema if you get it wrong.

---

## 2. Rule sets are data, and here is the proof

While researching this part I found a live example of exactly why. On the rent relief cap:

- PwC's Worldwide Tax Summaries: lower of ₦500,000 or 20% of annual rent
- TechCabal, BusinessDay, SmartSMSSolutions: same
- At least one other calculator site: **₦200,000** or actual rent, whichever is lower

The outlier is almost certainly reading an earlier draft of the Bill. But notice what happens in each architecture. If the cap is a Go constant, someone reads the wrong source, ships it, and every rent-relief claim is wrong by up to ₦300,000 of relief until a user complains. If the cap is a line in a signed YAML file with a `note` recording the conflict and a practitioner's name against it, the disagreement is visible, attributable, and correctable with a new version rather than a deploy.

The rule set file records that conflict verbatim. Read `rules/ng-2026.yaml`, the `RENT_RELIEF` entry.

A few design choices in that file worth defending:

**Rates are basis points, not floats.** `1500` means 15%. A YAML file cannot smuggle in `0.15000000000000002`, and every rate operation stays in integer arithmetic.

**Rounding is declared, and applied once.** `rounding_applied_at: final_liability` means the engine accumulates `slice × rate` across all six bands and divides by 10,000 exactly once. Rounding each band separately gives a different answer by a few kobo. Both are defensible; only one can be yours, and the user with a calculator deserves to know which.

**Pension takes the actual contribution, not a percentage.** The engine never computes 8% of anything, because the pensionable base is contested — PwC's own sample assumes 8% of gross income, while the Pension Reform Act basis is commonly stated as basic plus housing plus transport. That is payroll's problem. The engine takes the naira figure that was actually contributed and deducts it. One ambiguity, sidestepped rather than guessed.

**Only two relief evaluators exist.** `actual` and `rate_of_base`. Rent relief is the sole user of the second. Resist adding a third; the moment you need a formula language you have built a programming language with no debugger, and tax rules are not actually that varied.

---

## 3. The trace

`Compute` returns `(Assessment, Trace, error)`. The `Trace` is an ordered list of `Step`s, each with an operation, a label, an amount, a running total, a human detail string, and a citation.

For `G-005`, the multi-source case, it renders as:

```
ADD     Salary — Acme Ltd            9,000,000.00   →  9,000,000.00
ADD     Consulting fees              1,800,000.00   → 10,800,000.00
ADD     Rent from Yaba flat          1,200,000.00   → 12,000,000.00
TOTAL   Gross income                12,000,000.00

DEDUCT  Pension contribution           720,000.00   → 11,280,000.00
          actual ₦720,000
          NTA 2025 — national pension scheme contributions deductible
DEDUCT  Rent relief                    500,000.00   → 10,780,000.00
          min(20.00% × ₦3,600,000.00; ₦500,000.00) — cap applied
          NTA 2025 — rent relief (replaces Consolidated Relief Allowance)
TOTAL   Chargeable income           10,780,000.00

BAND    First ₦800,000                       0.00     0.00% of ₦800,000.00
BAND    ₦800,001 – ₦3,000,000          330,000.00    15.00% of ₦2,200,000.00
BAND    ₦3,000,001 – ₦12,000,000     1,400,400.00    18.00% of ₦7,780,000.00
TOTAL   Tax payable                  1,730,400.00

CREDIT  PAYE already deducted        1,600,000.00   →    130,400.00
TOTAL   Balance payable                130,400.00
```

That output is the product. It is what a support agent reads back to a confused user, what a tax officer accepts during an objection under the 30-day window, and the single clearest reason someone pays you instead of using a free band calculator. The line *"— cap applied"* is the difference between a user who thinks you made an error and a user who understands the law.

Three properties to preserve as the engine grows:

- **Every step carries its citation.** A user arguing with a number should end up arguing with the statute, not with your support queue.
- **Running totals are on every step.** People check arithmetic by following a column downward, not by trusting a final figure.
- **The trace is stored, not regenerated.** It goes in `assessments.trace` as JSONB alongside `facts_hash` and `rule_set_digest`. Regenerating it later against a newer rule set would produce a different explanation of the same historical number, which is worse than having none.

---

## 4. Golden corpus

### Format

One YAML file per case, in naira, with the band walk written out. The format exists so a practitioner can review it, so the review is a document markup task rather than a code review. Fields: `id`, `title`, `why_this_case`, `given`, `expect`, `review`.

`why_this_case` is not decoration. Six months from now someone will look at a failing case and need to know whether the expectation is load-bearing or arbitrary. `G-002` exists to catch an off-by-one in the rent cap comparison; that fact belongs in the file.

### The starter forty

Five are written. Here is the rest of the first pass, grouped by what they defend against:

**Band boundaries (8).** Chargeable income at exactly ₦800,000 and ₦800,001; at ₦3,000,000 and ₦3,000,001; at ₦50,000,000 and ₦50,000,001; ₦0; ₦1.

**Exemption edges (5).** Minimum wage exactly at threshold; one kobo above; part-year employment that annualises above the threshold but earns below it; minimum-wage earner with ₦1 of side income, which should break the exemption because it is scoped to employment-only; minimum-wage earner with PAYE wrongly deducted, which is `G-001` and produces a refund.

**Rent relief (5).** Rent at exactly ₦2,500,000, which is `G-002`; ₦2,499,999; ₦2,500,001; zero rent declared; rent declared but no evidence attached, which should validate as a warning rather than silently allow.

**Relief interactions (6).** All seven reliefs claimed at once; reliefs exceeding gross income, which must clamp to zero chargeable and not go negative; pension claimed with no employment income; a relief code not present in the rule set, which must error rather than be ignored.

**Multi-source (6).** Two employers, mid-year job change; salary plus freelance, which is `G-005`; salary plus rental plus dividends; digital asset gain as the only income; foreign income with a foreign tax credit, deferred to v1.5 but the fixture is written now so the gap is visible.

**Credits and balances (4).** Over-withheld producing a refund; under-withheld producing a payment; exactly withheld producing zero; credits exceeding liability.

**Penalties (6).** Filed one day late; one month late; six months late; filed on time but paid late; nil return filed late, which still attracts the ₦100,000 because the penalty is for failure to file, not failure to pay.

### The three test layers

**Golden.** Fixtures with signed expectations. Any change to an expected output requires a practitioner approval recorded in the pull request.

**Property.** Invariants over generated inputs. The four that matter, all verified against the band table already:

- Tax is monotonic in chargeable income.
- Take-home is monotonic too — one more naira of income never reduces net pay. This is the no-cliff property and it is what makes marginal rates trustworthy.
- Effective rate never exceeds the top marginal rate. Verified: it approaches but never reaches 25%.
- `Compute` is idempotent, and equal `TaxFacts` produce equal `Hash()`.

**Shadow replay.** Before publishing a new rule set version, replay every stored assessment against it and emit the exact list of taxpayers whose numbers would change and by how much. Never ship a rule change without that list in front of a human.

---

## 5. Validation catalogue

Validation is separate from computation. `Compute` assumes well-formed facts and errors loudly on malformed ones; `Validate` decides whether a return is fit to file. Two different jobs, two different packages.

Blocking errors prevent sign-off. Warnings appear but can be dismissed with an explicit acknowledgement, which is itself recorded as an event.

| Code | Severity | Rule | Why |
|---|---|---|---|
| `E001` | block | Tax ID present and verified | Filing without one attracts ₦50,000 + ₦25,000/month (NTAA s.100) |
| `E002` | block | NIN present for an individual | Tax ID for individuals derives from NIN |
| `E003` | block | Year of assessment and basis period are consistent | The Part 1 ambiguity; never let the two drift |
| `E004` | block | Relevant tax authority resolved from residency | Filing to the wrong SIRS is not filing |
| `E005` | block | At least one income item, or an explicit nil declaration | A nil return is a positive act, not an empty form |
| `E006` | block | No negative amounts anywhere | |
| `E007` | block | Every relief code exists in the loaded rule set | Catches a fixture written against an old version |
| `E008` | block | Rule set status is PUBLISHED | Belt and braces over the loader gate |
| `E009` | block | Declaration signed before submission | Shape A: the taxpayer signs, and the record must prove it |
| `E010` | block | No duplicate return for (taxpayer, YoA, jurisdiction) | Enforced in the schema too |
| `W101` | warn | Rent relief claimed without uploaded evidence | Relief granted only if rent paid is accurately declared and evidenced |
| `W102` | warn | Declared employment income differs from employer payroll record | Surface the difference; never silently pick a side |
| `W103` | warn | Pension claimed exceeds 8% of declared employment income | Usually a base-of-calculation error, occasionally correct |
| `W104` | warn | Rent relief claimed but accommodation recorded as owner-occupied | The portal asks for accommodation type separately |
| `W105` | warn | Prior year filed but this year's income fell by more than 50% | Not wrong, but the most common data-entry error |
| `W106` | warn | Fewer than three years of returns on file | Blocks the TCC; tell them now, not in April |
| `W107` | warn | Foreign income declared with no foreign tax credit claimed | Unilateral relief may be available and unclaimed |
| `W108` | warn | Digital asset gains declared as zero while exchange activity was imported | NTA 2025 s.4 names virtual asset gains explicitly |
| `W109` | warn | Filing after the deadline | Show the accrued penalty figure, not a generic caution |
| `W110` | warn | Business income declared with no expenses | Possible, but usually means expenses were not gathered |

Every validation result carries `{code, severity, field, message, citation}`. The client renders the message; the citation links to the statutory basis. A user told *"rent relief capped at ₦500,000 (NTA 2025)"* argues with the law rather than with you, and that is a support ticket you never receive.

---

## 6. Employer channel in the domain model

You chose to model this now, which is right, and it has one consequence that shapes everything else.

### The evidence-versus-declaration distinction

An employer telling you what they paid someone is **not** that person declaring their income. Under Shape A the taxpayer signs the return and carries the liability. So payroll data is *evidence*, and the taxpayer's confirmation is the *fact*.

Concretely: a `PayrollRecord` never becomes an `IncomeItem` by itself. It becomes a **proposal** that the taxpayer accepts, edits, or rejects, and the acceptance is the `IncomeDeclared` event that goes in the return's event log. If you collapse these two things into one table you will eventually file a return containing a number the taxpayer never saw, and under Shape A that is your problem in a way it would not be under Shape B.

This also gives you `W102` for free. Employer says ₦9,000,000, user's payslips say ₦9,400,000, both are recorded, the difference is surfaced.

### Aggregates

```
Employer (root)
  ├── cacNumber, taxID, legalName
  ├── jurisdictions[]          — employers often operate across states
  ├── PayrollContact[]
  └── DataProcessingAgreement  — see the NDPA note below

EmploymentRelationship        — the join that makes pre-fill possible
  ├── employerID, taxpayerID (nullable until claimed)
  ├── employeeRef             — the employer's own staff number
  ├── startDate, endDate      — nullable end = current
  └── matchConfidence         — how the employee was linked

PayrollImport (root)
  ├── employerID, yearOfAssessment
  ├── source                  — CSV | H1_EXPORT | API
  ├── uploadedBy, uploadedAt
  ├── status                  — RECEIVED → PARSED → VALIDATED → PUBLISHED
  └── PayrollRecord[]

PayrollRecord                 — EVIDENCE, not a declaration
  ├── employeeRef, nin?, taxID?, email?
  ├── grossKobo, payeRemittedKobo
  ├── pensionKobo, nhfKobo, nhisKobo
  ├── basicKobo, housingKobo, transportKobo   — the disputed pension base
  └── claimedByTaxpayerID?    — null until an employee claims it
```

Keeping `basic`, `housing` and `transport` separately is not fussiness. It is the only way to resolve the pension-base ambiguity later without going back to every employer for a re-upload.

### Schema

```sql
create table employers (
  id            uuid primary key,
  cac_number    text not null unique,
  tax_id        text,
  legal_name    text not null,
  created_at    timestamptz not null default now()
);

create table employment_relationships (
  id            uuid primary key,
  employer_id   uuid not null references employers(id),
  taxpayer_id   uuid references taxpayers(id),   -- null until claimed
  employee_ref  text not null,
  start_date    date not null,
  end_date      date,
  match_confidence text,                          -- NIN_EXACT | TAXID_EXACT | EMAIL | MANUAL
  unique (employer_id, employee_ref)
);

create table payroll_imports (
  id                 uuid primary key,
  employer_id        uuid not null references employers(id),
  year_of_assessment int not null,
  source             text not null,
  uploaded_by        uuid not null,
  uploaded_at        timestamptz not null default now(),
  status             text not null,
  row_count          int not null default 0,
  unique (employer_id, year_of_assessment, id)
);

create table payroll_records (
  id                    uuid primary key,
  import_id             uuid not null references payroll_imports(id),
  employee_ref          text not null,
  nin_hmac              bytea,          -- deterministic HMAC for matching only
  tax_id_hmac           bytea,
  email_hmac            bytea,
  gross_kobo            bigint not null,
  paye_remitted_kobo    bigint not null default 0,
  pension_kobo          bigint not null default 0,
  nhf_kobo              bigint not null default 0,
  nhis_kobo             bigint not null default 0,
  basic_kobo            bigint,
  housing_kobo          bigint,
  transport_kobo        bigint,
  claimed_by            uuid references taxpayers(id),
  claimed_at            timestamptz
);
create index on payroll_records (nin_hmac) where nin_hmac is not null;
create index on payroll_records (import_id);
```

Note the `_hmac` columns. You match employees to payroll records without ever storing a plaintext NIN in this table, using the same deterministic HMAC scheme as `taxpayer_identifiers` from Part 1. A leak of `payroll_records` then yields salaries without identities.

### The claim flow

1. Employer uploads payroll for the year of assessment. Status `RECEIVED`.
2. Parse and validate. Rows with no usable identifier are quarantined, not silently dropped.
3. Employer reviews a summary — row count, total gross, total PAYE — and publishes.
4. Employees are invited, or they sign up independently and get matched.
5. On match, the taxpayer sees a proposed pre-filled return with each figure attributed: *"Acme Ltd reported ₦9,000,000 gross and ₦1,600,000 PAYE for 2026."*
6. They accept, edit, or reject each line. Acceptance emits `IncomeDeclared`.

Step 6 is the one to get right. Never auto-accept.

### Multi-employer

A mid-year job change means two employers each report a partial year. The rules: sum, never overwrite; keep both `PayrollRecord`s; flag overlapping employment periods as a warning, since overlap is legitimate for someone with two jobs but is usually a date error. `EmploymentRelationship.startDate` and `endDate` exist so the return knows which months each employer covers.

### The NDPA consequence you cannot skip

When an employer uploads payroll, they are the data controller and **you are the processor** — they decided to send you their employees' personal data. Once an employee creates an account and starts a return, you become a controller for that person's data in your own right.

That split is not academic. It means you need a **data processing agreement with every employer** before the first upload, covering purpose limitation, retention, sub-processors and breach notification. It means quarantined and unclaimed payroll records need a retention clock, because holding a stranger's salary indefinitely on the theory that they might sign up one day is exactly the processing the NDPA constrains. And it strengthens the Part 1 conclusion that you should assume you are a data controller of major importance and register with the NDPC before launch rather than after.

Build the DPA into employer onboarding as a gate, the same way the rule-set signature gates the engine. Both are cheap now and expensive to retrofit.

---

## 7. What is in the repo

```
rules/ng-2026.yaml                      the tax law, as signed data (currently DRAFT)
taxengine/money.go                      Kobo, BasisPoints, exact integer arithmetic
taxengine/rules.go                      rule set types, loading, and the sign-off gate
taxengine/engine.go                     Compute, TaxFacts, Trace, Assessment
testdata/golden/G-00{1..5}-*.yaml       five reviewable golden cases
```

The engine imports nothing but the standard library. Enforce that with an import test in CI — `taxengine` reaching for `platform` or `net/http` should fail the build, because the moment it can do I/O it stops being replayable.

Arithmetic in all five fixtures has been independently verified. `G-003` additionally reproduces a published third-party worked example to the naira. None of it is signed off, and the code will not let you forget that.

---

## 8. Next

Part 3 covers the return lifecycle and the submission pack — the state machine including the `SELF_ATTESTED → ACK_UPLOADED` steps that close the filing loop under Shape A, the event log and its projections, and the field-by-field pack that carries a return across to the state portal in the portal's own order and labels.

Three things to line up in parallel, none of which are engineering:

1. **Email the JRB and NRS about Tax ID API access** — `smd@jrb.gov.ng` for individuals and business names, `tax.automation@nrs.gov.ng` for corporates. Long lead time, high value, costs nothing to start.
2. **Find the practitioner.** Everything else is blocked on this at release time, not at development time, so start now and keep building.
3. **Resolve the year-of-assessment question** before Part 3, because the return lifecycle and the pack both depend on it.
