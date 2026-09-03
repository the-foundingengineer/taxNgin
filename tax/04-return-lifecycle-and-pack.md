# Part 3 — Return Lifecycle and the Submission Pack

Part 1 chose the architecture. Part 2 built the engine that produces a number. Part 3 is about everything that happens to that number afterwards, which under Shape A is where the product actually lives or dies.

The reason is blunt. You compute a correct return and then hand it to a user who must retype it into `etax.lirs.net` themselves. Every field they hunt for is a field they mistype. Every step they are unsure about is a support ticket. And when they finish, nothing tells you whether it worked. That gap is the design problem for this part.

---

## 1. The lifecycle

```
                          ┌──────────────── INVALIDATE ────────────────┐
                          │                                            │
   DRAFT ──VALIDATE──▶ VALIDATED ──SIGN──▶ SIGNED ──ISSUE_PACK──▶ PACK_ISSUED
     │                     ▲   │             │                         │
     │                     └───┘             └────UNSIGN───────────────┘
     │                  (edit after                                    │
  ABANDON                validation)                      ┌────────────┴────────────┐
     │                                                    │                         │
     ▼                                          ATTEST_FILED              UPLOAD_ACK
 ABANDONED                                                │                         │
                                                          ▼                         │
                                                   SELF_ATTESTED ──UPLOAD_ACK──▶ ACK_UPLOADED
                                                          │                         │
                                                          └──────RECORD_ASSESSMENT──┤
                                                                                    ▼
                                                                               ASSESSED
                                                                            ┌───────┼───────┐
                                                                        OBJECT   PAY    CLEAR
                                                                            │       │       │
                                                                            ▼       ▼       ▼
                                                                       OBJECTED   PAID  CLEARED
                                                                                    │
                                                                                  CLEAR
```

The full machine is in `internal/returns/lifecycle.go`. Two states in it are load-bearing and worth defending.

### SELF_ATTESTED and ACK_UPLOADED are not the same state

`SELF_ATTESTED` means the user told you they filed. `ACK_UPLOADED` means they handed you the portal's acknowledgement receipt and you have evidence.

The temptation is to collapse them, because they feel like the same thing and the second one requires nagging the user for a document. Do not. The gap between them is a real thing that happens to real people: they intend to file, the portal times out or they get interrupted, and later they tell you they did it. If those are one state you cannot tell an attested return from a filed one, and every downstream feature that depends on knowing — TCC readiness, deadline reminders, multi-year catch-up — is built on a guess.

This is the specific cost of Shape A. Name it in the model rather than papering over it.

### Signing has real guards

Under Shape A the taxpayer carries the liability, so `SIGN` is the moment your software's number becomes their declaration. Four guards, all in the code:

- No blocking validation errors outstanding.
- An assessment exists, and it is not stale — `AssessmentStale` flips true the moment any income, relief or base changes after computation, so nobody signs a figure that no longer matches the facts.
- The rule set is `PUBLISHED`, not `DRAFT`. The Part 2 sign-off gate reaches all the way here.
- `DeclaredByTaxpayer` is true.

That last one deserves its own paragraph.

---

## 2. Actors, and why the log records who

Every event carries an `Actor` with a kind: `TAXPAYER`, `EMPLOYER`, `SYSTEM`, or `STAFF`.

This is not audit theatre. Under Shape A, a figure in a filed return must trace to something the taxpayer themselves declared. So the projection walks the income lines and sets `DeclaredByTaxpayer` false if any of them arrived without a `TAXPAYER`-actor event behind it. Signing is blocked when it is false.

Concretely, this makes the Part 2 rule structural rather than a convention. An employer uploading payroll emits `PayrollRecordProposed`, which deliberately does **not** touch `Projection.Income` and does **not** mark the assessment stale — an employer's assertion changes nothing about the return. Only `PayrollRecordAccepted`, actored by the taxpayer, creates the income line. If someone later writes a well-meaning auto-accept path, `DeclaredByTaxpayer` goes false and signing fails. The architecture refuses rather than relying on everyone remembering.

---

## 3. The event log

`internal/returns/events.go` has the closed catalogue, the `Projection`, and `Fold`.

Two rules hold in `Fold` and nowhere else:

**Unknown event kinds are tolerated, not fatal.** During a rolling deploy, new instances write kinds old instances have never seen. A projection that errors on an unknown kind takes down every read for the duration of the release. Unknown kinds bump the version and are otherwise ignored.

**Fold performs no I/O.** Rebuilding a projection is a pure walk over the log. Break this and replay stops being replay, which costs you the ability to rebuild projections after a schema change and the ability to answer a query about a 2026 return in 2031.

### Concurrency

`Projection.Version` is the sequence number of the last folded event, and it doubles as the optimistic concurrency token. Writes append at `expected_version + 1`; the unique constraint on `(return_id, seq)` from Part 1 rejects the loser. Retry by re-reading and re-validating, never by forcing.

### Idempotency

Every event carries an optional `IdempotencyKey`. Nigerian mobile connections drop mid-request constantly and people hammer submit buttons on a spinner. Without deduplication, one flaky POST becomes two `IncomeDeclared` events and a doubled salary. Key on `(return_id, idempotency_key)` with a unique index and return the original result on a repeat.

### Snapshots

Do not build them yet. A personal return generates tens of events, not thousands, and replay of a few dozen JSON blobs is microseconds. Add snapshotting when a p99 read actually shows it, and put the threshold in config so you can find out cheaply.

---

## 4. The submission pack

This is the centrepiece of Part 3 and the thing users will actually judge you on.

The pack is not a pretty PDF of a tax return. It is a **transcription script**: every field the portal will ask for, in the portal's order, with the portal's own label text, and the value to type. Generated from `portals/ng-la-etax.yaml`.

### Portal maps are versioned data, exactly like rule sets

Same argument, same shape. LIRS reorders a page and you publish `0.2.0`; you do not deploy code. Packs generated last April remain reproducible because each `PackIssued` event records the `portal_map_version` used.

The map I have written mirrors the observed 2026 LIRS eTax flow: three pages, in order.

- **Page 1, Statement of Income** — Employer Name, Salary, Commission, Trade/Business Income, Allowances, Pension Income, Interest, Rent Received, Dividends, Foreign Income.
- **Page 2, Mandatory Disclosure of Accommodation** — residential address, accommodation type, ownership type, rent amounts, landlord details.
- **Page 3, Other Disclosure for Reliefs** — life assurance premiums, NHF, gratuities received, NHIS, pension contributions, with uploads.

### Three things the map catches that a naive PDF would not

**Field names that mean the opposite of what a user assumes.** "Pension Income" on Page 1 is pension *received*. "Pension Contributions" on Page 3 is pension *paid*. "Rent Received" on Page 1 is rent you collect as a landlord; the rent you pay goes on Page 2. Users transpose these constantly. The map carries a `note` on each and the pack prints it.

**Our categories that do not map one-to-one.** We model `BUSINESS` and `PROFESSIONAL` separately; the portal has a single Trade/Business Income field. The map sums them and the pack shows the components underneath, so the user can check our arithmetic rather than trust it.

**Income the portal has no field for.** NTA 2025 s.4 taxes digital and virtual asset gains, prizes, honoraria, grants, awards, discounts and rebates. The observed Page 1 has no dedicated field for several of these. The map marks them `unmapped_income` and the pack refuses to silently fold them into an adjacent field. Flag it to the user, and ask the practitioner where LIRS expects them declared. **This is an open item, not a solved one.**

### Preconditions run before generation, not after

`PC01` through `PC04` are checked first. Discovering a missing Payer ID after transcribing thirty fields is the worst possible moment. The passport photograph is the sharp one: not required to file, required for the TCC, so it blocks `blocks_tcc` rather than `blocks_pack` and gets flagged early.

`PC04` is the employer cross-check — the portal's own RETURNS → EMPLOYER FILED RETURNS screen. The user should confirm their employer filed for the relevant years, because if a year is missing no amount of correct filing on their side fixes it; they have to go to HR. That check generates warning `W102` from Part 2 when the employer's figure and the user's declared figure disagree.

### Pack format

Generate one artefact in two renderings from the same data:

- **On screen**: a step-by-step wizard mirroring the three portal pages, each field with a copy button. This is the primary experience. Nobody wants to look at a PDF on one half of a phone screen and a portal on the other.
- **PDF**: for people who print it, or sit with an accountant, or file from a different device.

Both carry the assessment trace from Part 2 as an appendix, so the user has the reasoning next to the numbers when something is queried.

---

## 5. Penalties

`taxengine/penalty.go`. Pure, clock injected, no rate lookups.

"You are late, penalties may apply" is useless. "You owe ₦350,000 as at today, rising to ₦400,000 on 1 October" changes behaviour. The second one is a product feature.

Under NTAA s.101, ₦100,000 for the first month of default plus ₦50,000 for each subsequent month. Worked through, against a 31 March deadline:

| Filed | Months in default | Penalty |
|---|---|---|
| 31 March | 0 | ₦0 |
| 1 April | 1 | ₦100,000 |
| 30 April | 1 | ₦100,000 |
| 1 May | 2 | ₦150,000 |
| 30 September | 6 | ₦350,000 |
| 31 March following year | 12 | ₦650,000 |

Late payment adds a 10% surcharge on unpaid tax plus interest at the CBN Monetary Policy Rate. On ₦1,730,400 unpaid at a 27.5% MPR: ₦173,040 surcharge, plus ₦39,111.78 interest at 30 days, ₦117,335.34 at 90 days, ₦475,860 at a year.

Three deliberate choices:

**The MPR is an argument, not a lookup.** A number the engine went and fetched is a number you cannot reproduce in a dispute. The caller supplies it and it is stored with the result. If it is missing, `ComputePenalties` errors rather than guessing.

**Partial months count.** NTAA s.101 says ₦100,000 "for the first month in which the default occurs" and ₦50,000 "for each subsequent month the failure continues", and whether a partial month counts is not settled on the face of it. The code implements the conservative reading, which over-states rather than under-states. An over-stated penalty makes a user file sooner; an under-stated one makes them relaxed and then surprised. **[VERIFY]** with the practitioner and change the constant with the golden cases.

**Interest is simple, not compound.** The transparent reading, and the one a user can check on a calculator. **[VERIFY]**, along with whether the MPR is fixed at the due date or floats.

One rule that catches people: the filing penalty applies to a **nil return** too. It is a penalty for failing to file, not for failing to pay. Someone below the ₦800,000 threshold who skips filing owes ₦100,000 while owing ₦0 in tax. That is the single most useful thing you can tell a low-income user, and it should be on your landing page.

---

## 6. TCC readiness

The Tax Clearance Certificate is what people actually want. Six conditions, all in `portals/ng-la-etax.yaml`, all trackable:

| | Condition | Where we know it |
|---|---|---|
| T1 | Returns filed for the last three years, employer-filed and individual | `ACK_UPLOADED` state across three years of assessment |
| T2 | Passport photograph on the eTax profile | User confirmation, `PC03` |
| T3 | Full address updated | User confirmation, `PC02` |
| T4 | Assessments received for the last three years | `AssessmentServed` events |
| T5 | All taxes due fully paid | `PaymentRecorded` events |
| T6 | Profile complete | `PC02` |

`TCCReadiness` is a projection across a taxpayer's returns, not within one, and it is the best retention surface you have. Rather than a nag, it is a checklist that tells someone exactly what is missing and what to do about it. Someone two years into filing with you is looking at a green checklist and will not leave.

Note the dependency chain: T1 needs three years of returns, which is why multi-year catch-up filing is both a paid tier and the on-ramp to the thing users came for. Sell the certificate, deliver the filing.

---

## 7. Multi-year filing

A new user who has never filed typically needs three years at once. Design notes:

**One return per year, never a combined one.** Each is a separate aggregate with its own year of assessment, basis period, rule set version, and portal map version. A 2024 return computes under 2024's rules, not today's. This is the entire reason rule sets are versioned and dated.

**You do not have those rule sets yet.** `rules/ng-2026.yaml` is the only one written. Filing for years before 2026 means encoding the pre-reform regime — Consolidated Relief Allowance of ₦200,000 plus 20% of gross income, the old 7%–24% bands, the 1% minimum tax. That is a second rule set and a second golden corpus, needing the same practitioner sign-off. Scope it explicitly; it is not free.

**Order matters.** The portal walkthrough says to start with the earliest year. The pack sequence should follow.

**The penalty stack is the honest bad news.** Three years unfiled means three separate s.101 accruals. Someone catching up may owe more in penalties than in tax. Show the total up front rather than revealing it a year at a time. People forgive a large number they were warned about.

---

## 8. Amendments and objections

**Amendment.** A new return supersedes the old one; the old moves to `SUPERSEDED` and is never mutated. Because both are event logs, the diff between them is derivable rather than reconstructed, which is exactly what the authority will ask for.

**Objection.** NTAA s.34 gives 30 days from service of an assessment, after which it becomes binding even where it overstates the liability. `ObjectionDeadline` computes the date from the served date and the window in the rule set. Surface it as a countdown, because 30 days is short and the consequence of missing it is that a wrong number becomes legally correct.

Under Shape A you cannot lodge the objection for them. What you can do is compute the difference between the authority's assessment and yours, produce the supporting schedule, and tell them the deadline. That is worth paying for.

---

## 9. What is in the repo now

```
rules/ng-2026.yaml                     tax law as signed data (DRAFT)
portals/ng-la-etax.yaml                LIRS eTax transcription map (DRAFT)
taxengine/money.go                     exact integer money
taxengine/rules.go                     rule set loading + sign-off gate
taxengine/engine.go                    Compute → Assessment + Trace
taxengine/penalty.go                   NTAA s.101 accrual, clock injected
internal/returns/lifecycle.go          state machine, transitions, guards
internal/returns/events.go             event catalogue, Projection, Fold, Replay
testdata/golden/G-00{1..5}-*.yaml      five reviewable golden cases
```

Penalty arithmetic in section 5 has been independently verified. The Go has not been compiled — no toolchain available here — so expect to fix an import or a type name on first `go build`.

---

## 10. Open items from this part

1. **Walk a real return through etax.lirs.net with `portals/ng-la-etax.yaml` open** and correct every label. The map is built from published walkthroughs, not a logged-in session. Half a day, and it is the highest-value half day in the project.
2. **Where does LIRS expect digital asset gains, prizes and honoraria declared** on Page 1? The map marks them unmapped and the pack will not guess.
3. **Do partial months count** under NTAA s.101, and is late payment interest simple or compound at a fixed or floating MPR?
4. **The year of assessment question**, still open from Part 1, now blocking: it determines what the pack tells the user to select on the portal.
5. **Pre-2026 rule sets** for multi-year catch-up, with their own corpus and sign-off.

Items 2, 3 and 4 go to the practitioner. Item 1 you can do yourself this week.

Part 4 covers ingestion: payslip and H1 extraction, bank statement parsing, the employer bulk import pipeline, and the `IncomeSource` port with its aggregator and open-banking adapters.
