# April Spike Runbook — LIRS Filing Season

**Filing deadline: March 31, 23:59 WAT**  
**Peak window: March 18–31 (14 days)**  
**Owner: Engineering on-call**

---

## Traffic Model

Target user base (Year 1): **50,000 accounts**

Historical LIRS pattern: 70% of annual filings in the final 14 days of March.

| Metric | Calculation | Result |
|---|---|---|
| Peak-window submissions | 50,000 × 70% | 35,000 returns |
| March 31 share | 35,000 × 20% | 7,000 returns filed on deadline day |
| Hourly peak (10am–2pm WAT) | 7,000 ÷ 4 hours | 1,750 returns/hour |
| Per-minute peak | 1,750 ÷ 60 | ~29 returns/minute |
| Engine compute time | measured (property tests) | < 1 ms per return |
| Pack generation time | measured (capacity tests) | < 5 ms per return |

**Conclusion:** the computation layer is not the bottleneck. At 29 returns/minute, a single CPU core running the engine is using approximately 30ms of compute per minute — a 0.05% utilisation. The bottlenecks are I/O and external service rate limits (see below).

---

## Bottlenecks by Layer

| Layer | What spikes | Mitigation |
|---|---|---|
| Tax engine (Compute) | CPU; trivially parallelisable | Goroutine pool; 1 core handles >10,000 computes/second at <1ms each |
| Pack generator | Memory (PortalMap per request) | Singleton PortalMap loaded at process startup; never reload on hot path |
| Registry | Read-heavy; rule sets are immutable | Singleton Registry; no per-request disk or network access |
| Notification delivery | WhatsApp/SMS provider rate limits | Queue with exponential backoff; fall back to SMS when WhatsApp throttles; never drop silently |
| NIN verification | JRB API rate limit (unknown SLA) | Cache positive verifications for 30 days; never re-verify in the same session; disable entirely if JRB is unreachable |
| LIRS eTax portal | The portal itself may slow or go down | The pack is a transcription script; if the portal is unavailable, the pack is still valid and the user retries manually. Log the portal status separately from return status. |
| Database writes | Event log is append-only | Apply backpressure at the HTTP layer; return HTTP 429 with Retry-After before the event log write queue exceeds threshold |

---

## Degradation Plan

Ordered by severity. Apply in sequence; higher-severity degradations subsume lower ones.

### Level 1 — NIN verification latency

**Trigger:** JRB API p95 > 2 s, or error rate > 5%.

**Action:**
1. Accept the submission optimistically.
2. Enqueue NIN verification as a background job.
3. Notify taxpayer via preferred channel when verification completes or fails.
4. Do not block filing. The submission pack is still valid.

**Rollback:** when JRB p95 returns below 500 ms for 5 consecutive minutes, drain the backlog queue.

### Level 2 — WhatsApp API rate limit

**Trigger:** WhatsApp provider returns HTTP 429 or error rate > 10%.

**Action:**
1. Immediately fall back to SMS for all queued notifications.
2. Do not drop notifications silently — every queued notification must be delivered to at least one channel.
3. Log the fallback event per notification ID.

**Rollback:** when WhatsApp API error rate returns below 1% for 10 consecutive minutes, resume WhatsApp delivery for new notifications. Backlogged notifications already sent via SMS are not re-sent.

### Level 3 — JRB API unavailable

**Trigger:** JRB API returns no response or consistent 5xx for > 60 seconds.

**Action:**
1. Disable NIN verification entirely. Set feature flag `nin_verification_enabled = false`.
2. Log every skipped verification with taxpayer account ID and timestamp for retry.
3. Do not block filing.
4. Alert the on-call engineer.

**Rollback:** when JRB API is confirmed healthy, set `nin_verification_enabled = true` and replay skipped verifications from the log.

### Level 4 — Database write latency

**Trigger:** event log write p95 > 500 ms, or write error rate > 1%.

**Action:**
1. Apply HTTP 429 backpressure at the API gateway before accepting new return mutations.
2. Return `Retry-After: 30` to clients.
3. Continue read requests (projection rebuilds, pack downloads, support console) normally — they do not write.
4. Alert the on-call database engineer.

**Rollback:** when p95 returns below 100 ms for 5 consecutive minutes, remove backpressure.

---

## Pre-Spike Checklist

Run this checklist two weeks before March 31 (i.e., by March 17).

### Engine and rule set

- [ ] `go test ./... -count=1 -race` passes with zero failures
- [ ] `TestEngineComputeUnder1ms` passes in CI — confirms engine is within performance contract
- [ ] `TestPackGeneratorUnder5ms` passes in CI — confirms pack generator is within contract
- [ ] `TestRegistryLookupUnder100µs` passes in CI — confirms registry lookup is within contract
- [ ] Shadow replay all stored assessments against the current rule set (`TASK-007`)
- [ ] Registry loaded and smoke-tested: `reg.Get("NG", 2026)` returns the correct rule set
- [ ] Pack generator smoke-tested with a known good projection

### Notifications

- [ ] Deadline notification schedule deployed for the following dates:
  - March 17 (14 days before deadline)
  - March 24 (7 days before deadline)
  - March 28 (3 days before deadline)
  - March 30 (1 day before deadline)
- [ ] WhatsApp → SMS fallback confirmed with a test notification to a staging account
- [ ] SMS delivery confirmed with Africa's Talking or equivalent provider

### Auth and billing

- [ ] `StubNINVerifier` is **NOT** deployed to production — confirm by checking the production binary's injected verifier type
- [ ] `StubPaymentGateway.AlwaysSucceeds` is **NOT** set to `true` in the production config
- [ ] Account recovery OTP flow tested end-to-end in staging

### Support console

- [ ] Support console access verified for all support agents (role assignment confirmed)
- [ ] `TestBuildSupportView_NeverExposesAmounts` passes — NDPA redaction audit
- [ ] Support agents briefed on the redacted view: they will not see salary figures

### Infrastructure

- [ ] Singleton PortalMap confirmed loaded at process startup (check startup log: `portal map loaded: version=...`)
- [ ] Singleton Registry confirmed loaded at process startup (check startup log: `registry loaded: 2026`)
- [ ] Database connection pool sized for peak write throughput (at least 2× normal)
- [ ] HTTP 429 backpressure threshold configured and tested in staging
- [ ] JRB API health check endpoint added to monitoring; alert set at > 5% error rate

---

## Day-of Runbook (March 31)

**08:00 WAT** — On-call engineer confirms all pre-spike checklist items are green.

**08:30 WAT** — Send final "1 day" deadline notification to all accounts that have not filed.

**10:00–14:00 WAT** — Peak window. Monitor:
- HTTP error rate (target: < 0.1%)
- Engine compute p95 (target: < 1ms)
- Pack generation p95 (target: < 5ms)
- WhatsApp delivery success rate (target: > 95%)
- Database write p95 (target: < 100ms)

**14:00 WAT** — If any metric is in degradation Level 3 or 4, engage on-call database or infrastructure engineer.

**23:59 WAT** — Filing deadline passes. Accept any in-flight requests that were initiated before midnight.

**24:00 WAT** — Switch penalty accrual mode on in the engine for late filers. Verify `NotifPenaltyAccruing` notifications are queued for the first business day of April.

---

## What This Runbook Does Not Cover

- PDF rendering or portal session automation — the pack is a transcription script; the taxpayer manually enters data
- Multi-tenancy or employer portal load — employer bulk imports are rate-limited by HMAC validation, not the engine
- Frontend wizard performance — out of scope for this backend runbook
