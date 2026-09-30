TASK-008 — Platform Foundations: Ingestion, Auth, Billing, Notifications, Support Console, and the April Spike

## Context (read this first — you have no prior conversation history)

This repo is the backend for a Nigerian personal income tax prep product (Shape A:
self-preparation). TASKs 001–007 built the tax engine, the return lifecycle, the
validation catalogue, the submission pack, the TCC readiness projection, the multi-year
registry, the filing coordinator, the employer channel, shadow replay, and the
amendment delta. The engine is correct and the lifecycle is complete.

What does not yet exist is everything that makes the engine a product a real person
can use. Six areas are wholly undesigned:

1. **Ingestion** (Part 4 of the design doc) — how taxpayer income evidence gets into
   the return. The employer channel is modelled (TASK-006) but the IncomeSource port
   that unifies payslip, H1, CSV, and bank statement inputs does not exist. Without
   it, every income source is a separate bespoke integration.

2. **Auth and account recovery** — NIN-anchored identity is genuinely different from
   a standard email/password flow. Taxpayers cannot "reset via email" when identity
   is anchored on NIN and BVN. Getting recovery wrong locks people out during
   filing season or lets bad actors in.

3. **Billing and entitlements** — the product has tiers. Without a model of what each
   tier permits, every feature is implicitly free and there is no gating surface to
   wire a payment gateway to.

4. **Notifications** — in Nigeria, WhatsApp reaches more taxpayers than email. A
   deadline reminder sent to WhatsApp at 14 days, 7 days, 3 days, and the morning of
   March 31 is a product feature. A generic email is not.

5. **Support console** — a support agent who can see a return cannot be a support
   agent who has access to every salary in the system. The NDPA makes this a legal
   requirement, not a nice-to-have. Building the redacted view now prevents the
   pattern where engineers eventually grep the production database to answer support
   tickets.

6. **April spike runbook** — the LIRS filing deadline is March 31. At 50,000 users
   with 70% filing in the final 2 weeks, the engineering answers to "will it hold?"
   need to exist before March, not during it.

Each area gets a minimal but complete treatment: interfaces defined, core logic
implemented, external dependencies mocked, tests written. Nothing here requires a
live NIN API, a Paystack account, a WhatsApp Business approval, or a database. Every
deliverable is pure Go or an interface that pure-Go stubs can satisfy.

Read in full before touching code:

- `tasks/TASK-006-employer-channel.md` (employer HMAC scheme, DPA gate, IncomeSource context)
- `tasks/TASK-007-shadow-replay-evidence-and-amendments.md` (amendment and evidence context)
- `tax/02-engine-and-employer.md` (§6 employer channel design, NDPA note, claim flow)
- `tax/04-return-lifecycle-and-pack.md` (§6 TCC readiness, §8 objection support)
- `internal/employer/types.go` (PayrollRecord — PayslipRecord must be consistent with it)
- `internal/employer/hmac.go` (HMAC scheme — auth MUST use the same approach for NIN/BVN)
- `internal/returns/events.go` (Projection fields available to the support view)
- `internal/validate/validate.go` (FilingContext — entitlements gate which validations run)

## Prerequisite: TASK-007 must be done

`go test ./... -count=1` must pass including the shadow replay and amendment tests. If
`internal/shadowreplay` and `internal/amendment` do not exist, finish TASK-007 first.

## What "done" looks like

---

### 1. Ingestion — `internal/ingestion`

The IncomeSource port is the unification point for everything that puts income evidence
into a return. Adapters (CSV, PDF extractor, open banking) implement the port. The
port emits `returns.Event`s, specifically `EvPayrollProposed` or `EvIncomeDeclared`
depending on the evidence source.

#### 1a. Types

```go
// PayslipRecord is the normalised output of any income source adapter. All
// adapters — CSV, PDF, bank statement, open banking — produce this. The
// caller converts it to return events via the IncomeSource port.
type PayslipRecord struct {
    SourceRef       string         // employer name, bank name, trading platform
    PeriodStart     time.Time      // first day of pay period (monthly or annual)
    PeriodEnd       time.Time
    GrossKobo       taxengine.Kobo
    TaxDeductedKobo taxengine.Kobo // PAYE or WHT deducted at source
    PensionKobo     taxengine.Kobo
    NHFKobo         taxengine.Kobo
    NHISKobo        taxengine.Kobo
    Notes           string         // e.g. "month 9 payslip missing — gap flagged"
}

// IncomeSource is the port that all income adapters implement.
type IncomeSource interface {
    Name() string
    // Extract parses raw bytes in the adapter's native format and returns
    // normalised PayslipRecords. It must not perform I/O; data is pre-loaded
    // and passed in. Extract never calls time.Now().
    Extract(data []byte) ([]PayslipRecord, error)
}

// DocumentExtractor extends IncomeSource for document-based formats (PDF,
// Excel). The MimeType lets the dispatcher route the document to the right
// adapter before calling Extract.
type DocumentExtractor interface {
    IncomeSource
    MimeType() string
}
```

#### 1b. CSV ingester (implement fully)

```go
type ColumnMapping struct {
    SourceName  int // zero-based column index for employer / source name
    GrossKobo   int
    PAYEKobo    int
    PensionKobo int
    NHFKobo     int
    NHISKobo    int
    // SkipRows is the number of header rows to skip before data rows begin.
    SkipRows    int
}

// DefaultH1Mapping returns the column mapping for the standard LIRS H1 payroll
// schedule CSV export. Column indices are 0-based.
func DefaultH1Mapping() ColumnMapping

// CSVIngester implements IncomeSource for CSV payroll files.
// It is the primary path for the employer bulk import pipeline.
type CSVIngester struct{ Columns ColumnMapping }

func (c *CSVIngester) Name() string
func (c *CSVIngester) Extract(data []byte) ([]PayslipRecord, error)
```

The CSV ingester must:
- Skip `SkipRows` header rows.
- Parse amounts as integer naira (no decimal point) OR as naira.kobo with two decimal
  places — the same `golden.ParseNaira` logic. Never `float64`.
- Quarantine rows with missing or unparseable amounts in `Notes`, not silently drop them.
  A quarantined row produces a `PayslipRecord` with `GrossKobo == 0` and a `Notes`
  entry naming the column and row number. The caller decides what to do with it.
- Return an error only if the entire file cannot be parsed (malformed CSV). Individual
  bad rows are quarantined, not errors.

#### 1c. Aggregator

```go
// AggregateBySource collapses multiple PayslipRecords from the same SourceRef
// into a single annual record, summing all Kobo fields. Used when monthly
// payslips are ingested instead of an annual H1 summary.
//
// Records with different SourceRefs are kept separate — a mid-year job change
// must not be summed into one record.
func AggregateBySource(records []PayslipRecord) []PayslipRecord
```

#### 1d. Stub adapters for other sources

These are interfaces only. No implementation is required — stubs are enough to prove
the port works and to give the next engineer a clear target.

```go
// PDFPayslipExtractor extracts PayslipRecords from PDF payslips using an
// external document processing service. Implement with an injected HTTP client
// when the external service is chosen.
type PDFPayslipExtractor interface {
    DocumentExtractor
}

// OpenBankingAdapter fetches transaction data via the Nigerian Open Banking API
// and categorises income transactions. Implement when NIBSS Open Banking access
// is approved.
type OpenBankingAdapter interface {
    IncomeSource
    // FetchTransactions fetches transaction data for the given account reference
    // and period. The account reference is consent-gated; never stored.
    FetchTransactions(accountRef string, from, to time.Time) ([]PayslipRecord, error)
}
```

Define `StubPDFExtractor` and `StubOpenBankingAdapter` (both return
`errors.New("not implemented")` from `Extract`). These are the injection points
for real adapters when external services are contracted.

#### 1e. Required tests

- `TestCSVIngester_HappyPath` — parse a 3-row CSV with known values; assert three
  `PayslipRecord`s with correct `GrossKobo` values.
- `TestCSVIngester_QuarantinesBadRow` — one row has a non-numeric gross; assert the
  quarantined record has `GrossKobo == 0` and a non-empty `Notes`.
- `TestCSVIngester_SkipsHeaderRows` — `SkipRows: 2`; assert data starts from row 3.
- `TestCSVIngester_NoFloat64` — all amounts are integers; assert the ingester never
  produces a value that differs from its parsed naira × 100.
- `TestAggregateBySource_SameSource` — three monthly records from "Acme Ltd"; assert
  one aggregate with summed gross.
- `TestAggregateBySource_DifferentSources` — two records from "Acme Ltd", one from
  "Beta Corp"; assert two aggregates.
- `TestAggregateBySource_MidYearJobChange` — five months at Acme, seven at Beta;
  assert separate aggregates, correct sums.

---

### 2. Auth — `internal/auth`

NIN/BVN-anchored identity is not an email/password flow with a different label. The
differences are architectural:

- You cannot "reset via email" because the identity proof is NIN, not email.
- NIN verification requires the JRB API (long lead time — interface it, do not
  implement it). Until the API is available, verification is deferred, not skipped.
- BVN verification goes through NIBSS via a bank partner (also interfaced only).
- OTP delivery goes to a registered mobile number, not email.
- Sessions must be revocable: if a NIN is compromised, invalidate all sessions.

#### 2a. Account model

```go
type AccountStatus string
const (
    AccountPendingVerification AccountStatus = "PENDING_VERIFICATION"
    AccountActive              AccountStatus = "ACTIVE"
    AccountSuspended           AccountStatus = "SUSPENDED"
)

// TaxpayerAccount is the identity record for one individual taxpayer.
// NIN and BVN are never stored in plaintext — the same HMAC scheme as
// internal/employer/hmac.go. A leak of the accounts table yields
// account metadata without usable identifiers.
type TaxpayerAccount struct {
    ID              string
    NINHMACSuffix   []byte        // from employer.ComputeHMAC — same key, same scheme
    BVNHMACSuffix   []byte        // optional; nil until taxpayer provides BVN
    PhoneNumber     string        // required; primary OTP delivery channel
    EmailAddress    string        // optional; secondary notification channel
    CredentialHash  []byte        // bcrypt of password/PIN
    Status          AccountStatus
    CreatedAt       time.Time
    LastLoginAt     *time.Time
    VerifiedAt      *time.Time    // nil until NIN is verified via JRB
}
```

#### 2b. Recovery model

```go
type RecoveryMethod string
const (
    RecoveryOTPSMS    RecoveryMethod = "OTP_SMS"    // OTP to registered phone
    RecoveryBVNMatch  RecoveryMethod = "BVN_MATCH"  // verify they own a BVN-linked account
)

type RecoveryState string
const (
    RecoveryInitiated RecoveryState = "INITIATED"
    RecoveryOTPSent   RecoveryState = "OTP_SENT"
    RecoveryVerified  RecoveryState = "VERIFIED"
    RecoveryCompleted RecoveryState = "COMPLETED"
    RecoveryExpired   RecoveryState = "EXPIRED"
)

type RecoverySession struct {
    ID          string
    AccountID   string
    Method      RecoveryMethod
    State       RecoveryState
    OTPHash     []byte        // bcrypt of the OTP; never stored in plaintext
    AttemptCount int          // lock out after 5 failed OTP attempts
    ExpiresAt   time.Time     // 10 minutes for OTP, 30 minutes for BVN match
    CompletedAt *time.Time
}
```

#### 2c. Recovery state machine (implement fully)

All functions are pure — external verification is injected via interfaces.

```go
// InitiateRecovery starts a recovery session for the given account.
// Returns ErrAccountNotActive if the account is suspended or pending.
// The OTP is generated by the caller and passed in already hashed; this
// function never generates randomness (not its job) and never calls time.Now().
func InitiateRecovery(account TaxpayerAccount, method RecoveryMethod, otpHash []byte, now time.Time) (RecoverySession, error)

// SubmitOTP verifies the OTP and advances the session to VERIFIED.
// Returns ErrOTPExpired if now is past session.ExpiresAt.
// Returns ErrOTPInvalid with incremented AttemptCount.
// Returns ErrTooManyAttempts (and sets State=EXPIRED) after 5 failures.
// The comparison uses bcrypt.CompareHashAndPassword — never == on raw bytes.
func SubmitOTP(session RecoverySession, providedOTP string, now time.Time) (RecoverySession, error)

// CompleteRecovery resets the credential and closes the session.
// Returns ErrSessionNotVerified if State != VERIFIED.
// Returns ErrSessionExpired if now > ExpiresAt.
func CompleteRecovery(session RecoverySession, newCredentialHash []byte, now time.Time) (RecoverySession, error)
```

Errors must be typed sentinels (`var ErrOTPInvalid = errors.New(...)`) so callers can
distinguish them with `errors.Is`.

#### 2d. External verifier interfaces (define, do not implement)

```go
// NINVerifier checks whether a NIN is registered and returns basic record
// data for cross-checking (date of birth, name fragment). The JRB API
// endpoint is smd@jrb.gov.ng — request access; long lead time.
type NINVerifier interface {
    Verify(nin string) (NINRecord, error)
}

// BVNVerifier checks whether a BVN is linked to a bank account and returns
// a name for cross-checking. Goes through NIBSS via a bank partner.
type BVNVerifier interface {
    Verify(bvn string) (BVNRecord, error)
}

type NINRecord struct {
    NIN            string
    NameFragment   string // first 3 letters of surname — for cross-check only
    DateOfBirth    time.Time
    Verified       bool
}

type BVNRecord struct {
    BVN          string
    NameFragment string
    Verified     bool
}

// StubNINVerifier always returns Verified: true. Use in tests and
// pre-API-approval staging; never in production.
type StubNINVerifier struct{}
func (s StubNINVerifier) Verify(nin string) (NINRecord, error)

type StubBVNVerifier struct{}
func (s StubBVNVerifier) Verify(bvn string) (BVNRecord, error)
```

#### 2e. Required tests

- `TestInitiateRecovery_OTPMethod` — active account, OTP_SMS method; assert session is
  INITIATED with OTP_SENT state and 10-minute expiry.
- `TestInitiateRecovery_SuspendedAccount` — ErrAccountNotActive returned.
- `TestSubmitOTP_Correct` — correct OTP within expiry; assert state == VERIFIED.
- `TestSubmitOTP_Expired` — `now` is past `ExpiresAt`; assert ErrOTPExpired.
- `TestSubmitOTP_WrongOTP` — wrong OTP; assert ErrOTPInvalid and AttemptCount == 1.
- `TestSubmitOTP_LockoutAfterFiveFailures` — submit wrong OTP 5 times; assert
  ErrTooManyAttempts and State == EXPIRED.
- `TestCompleteRecovery_HappyPath` — VERIFIED session, valid now; assert COMPLETED.
- `TestCompleteRecovery_NotVerified` — INITIATED session; assert ErrSessionNotVerified.

---

### 3. Billing and entitlements — `internal/billing`

The model has three tiers. The entitlements determine what calls succeed upstream —
the billing package does not call the engine; it answers "can this account do X?"

```go
type Tier string
const (
    TierFree     Tier = "FREE"     // one return, current year only
    TierStandard Tier = "STANDARD" // current + one prior year, employer pre-fill
    TierPremium  Tier = "PREMIUM"  // unlimited years, TCC tracking, priority support
)

type Entitlements struct {
    MaxYearsOfAssessment int    // 1, 2, or math.MaxInt
    EmployerPreFill      bool
    TCCTracking          bool
    MultiYearCatchUp     bool
    PrioritySupport      bool
}

type Action string
const (
    ActionFileReturn       Action = "file_return"
    ActionAccessPriorYear  Action = "access_prior_year"  // more than 1 year ago
    ActionEmployerPreFill  Action = "employer_prefill"
    ActionViewTCCReadiness Action = "view_tcc_readiness"
    ActionMultiYearCatchUp Action = "multi_year_catchup"
)

func EntitlementsFor(tier Tier) Entitlements
func (e Entitlements) Can(action Action) bool
func (e Entitlements) CanAccessYear(yearOfAssessment, currentYear int) bool
```

#### 3a. Billing event log (define types, no payment implementation)

```go
type BillingEventKind string
const (
    BillingSubscriptionStarted BillingEventKind = "SUBSCRIPTION_STARTED"
    BillingSubscriptionRenewed BillingEventKind = "SUBSCRIPTION_RENEWED"
    BillingSubscriptionExpired BillingEventKind = "SUBSCRIPTION_EXPIRED"
    BillingPaymentFailed       BillingEventKind = "PAYMENT_FAILED"
    BillingDowngraded          BillingEventKind = "DOWNGRADED"
)

type BillingEvent struct {
    ID          string
    AccountID   string
    Kind        BillingEventKind
    Tier        Tier
    OccurredAt  time.Time
    Reference   string // provider payment reference; empty for system events
}

// CurrentTier derives the active tier from an account's billing event log.
// The most recent SUBSCRIPTION_STARTED or SUBSCRIPTION_RENEWED defines the
// tier. If no events exist, returns TierFree.
func CurrentTier(events []BillingEvent, now time.Time) Tier

// PaymentGateway is the interface for Nigerian payment processors (Paystack,
// Flutterwave). Implement when a gateway is contracted.
type PaymentGateway interface {
    InitiatePayment(accountID string, tier Tier, callbackURL string) (string, error) // returns redirect URL
    VerifyPayment(reference string) (BillingEvent, error)
}

type StubPaymentGateway struct{ AlwaysSucceeds bool }
func (s StubPaymentGateway) InitiatePayment(accountID string, tier Tier, callbackURL string) (string, error)
func (s StubPaymentGateway) VerifyPayment(reference string) (BillingEvent, error)
```

#### 3b. Required tests

- `TestEntitlementsFor_Free` — TierFree: MaxYears==1, EmployerPreFill==false,
  TCCTracking==false.
- `TestEntitlementsFor_Premium` — all flags true.
- `TestCan_FreeCannotAccessPriorYear` — TierFree, ActionAccessPriorYear → false.
- `TestCan_StandardCanAccessOnePriorYear` — TierStandard, year N-1 → true;
  year N-2 → false.
- `TestCurrentTier_NoEvents` — empty log → TierFree.
- `TestCurrentTier_AfterExpiry` — SUBSCRIPTION_STARTED for STANDARD + SUBSCRIPTION_EXPIRED
  after 30 days; `now` is after expiry → TierFree.
- `TestCurrentTier_Renewal` — STARTED + RENEWED; tier derived from RENEWED event.

---

### 4. Notifications — `internal/notify`

The filing deadline is March 31. A user who receives a WhatsApp message on March 17
("14 days until your filing deadline — your return is 60% complete") and another on
March 28 ("3 days — your estimated penalty if you miss the deadline is ₦100,000") is
a user who files. The notification model must be able to express both.

#### 4a. Types

```go
type Channel string
const (
    ChannelWhatsApp Channel = "WHATSAPP" // primary — highest open rate in Nigeria
    ChannelSMS      Channel = "SMS"      // fallback when WhatsApp is unavailable
    ChannelEmail    Channel = "EMAIL"    // legal notices and receipts; do not rely on it for deadlines
)

type NotifKind string
const (
    NotifDeadlineApproaching  NotifKind = "deadline_approaching"  // 14d, 7d, 3d, 1d
    NotifPenaltyAccruing      NotifKind = "penalty_accruing"      // return is already late
    NotifAssessmentServed     NotifKind = "assessment_served"
    NotifObjectionDeadline    NotifKind = "objection_deadline"    // 7d, 3d before NTAA s.34 window closes
    NotifReturnCleared        NotifKind = "return_cleared"
    NotifTCCAvailable         NotifKind = "tcc_available"
    NotifTCCMissingCondition  NotifKind = "tcc_missing_condition" // T1–T6 not met
    NotifLoginDetected        NotifKind = "login_detected"        // new device or location
    NotifRecoveryInitiated    NotifKind = "recovery_initiated"
    NotifPaymentConfirmed     NotifKind = "payment_confirmed"
)

type NotifStatus string
const (
    NotifPending   NotifStatus = "PENDING"
    NotifSent      NotifStatus = "SENT"
    NotifFailed    NotifStatus = "FAILED"
    NotifCancelled NotifStatus = "CANCELLED"
)

type Notification struct {
    ID          string
    AccountID   string
    Kind        NotifKind
    Channel     Channel
    To          string      // phone number ("234XXXXXXXXXX") or email
    Body        string      // rendered, channel-appropriate text
    ScheduledAt time.Time
    SentAt      *time.Time
    Status      NotifStatus
}

// Notifier is the interface that channel providers implement.
// Twilio, Africa's Talking, and direct WhatsApp Business API all satisfy it.
type Notifier interface {
    Send(n Notification) error
}

type StubNotifier struct {
    Sent []Notification
}
func (s *StubNotifier) Send(n Notification) error
```

#### 4b. Notification builder (implement fully)

```go
// NotifData carries the domain-specific fields needed to render a notification.
// Fields are zero-valued when not relevant to the NotifKind.
type NotifData struct {
    YearOfAssessment    int
    FilingDeadline      time.Time
    DaysRemaining       int
    AccruedPenaltyKobo  taxengine.Kobo
    AuthorityRef        string        // assessment reference number
    TCCConditions       []string      // T-conditions that are not met
    LoginLocation       string        // city or "unknown location"
}

// ChannelPreference defines the order in which channels are tried.
// Build from the account's registered phone and email.
type ChannelPreference struct {
    Primary    Channel
    Fallbacks  []Channel
    PhoneNumber string
    Email       string
}

func DefaultChannelPreference(hasWhatsApp bool, phoneNumber, email string) ChannelPreference

// BuildNotifications returns one Notification per channel in the preference order.
// The Body is rendered in an appropriate register for each channel:
//   - WhatsApp: short, conversational, ₦ symbol, uses "you"
//   - SMS:      very short (160 chars), no special characters
//   - Email:    full sentence, formal, statutory citation included
//
// The clock (now) is injected — never time.Now(). ScheduledAt is set to now for
// immediate delivery or to a future time when advance scheduling is needed.
func BuildNotifications(kind NotifKind, data NotifData, prefs ChannelPreference, now time.Time) []Notification
```

WhatsApp body length must not exceed 1,024 characters. SMS body must not exceed 160
characters. Truncation must not cut a ₦ amount mid-number — better to omit the
amount than to show a truncated one.

#### 4c. Advance scheduling

```go
// DeadlineSchedule returns the set of NotifKind=NotifDeadlineApproaching
// notifications to schedule for a given filing deadline. The schedule is
// 14 days, 7 days, 3 days, and 1 day before the deadline.
// Notifications for dates that have already passed (now > scheduledAt) are omitted.
func DeadlineSchedule(deadline time.Time, prefs ChannelPreference, now time.Time) []Notification
```

#### 4d. Required tests

- `TestBuildNotifications_DeadlineApproaching_WhatsApp` — 14 days out, WhatsApp channel;
  assert body contains the day count and is ≤ 1,024 characters.
- `TestBuildNotifications_PenaltyAccruing_SMS` — ₦100,000 penalty; SMS body ≤ 160
  characters; penalty amount present.
- `TestBuildNotifications_FallbackChannels` — `Primary: WhatsApp`, Fallbacks: [SMS];
  assert two notifications returned (one per channel).
- `TestDeadlineSchedule_AllFuture` — deadline 20 days out; assert 4 notifications
  (14d, 7d, 3d, 1d) all have ScheduledAt in the future.
- `TestDeadlineSchedule_SomePassed` — deadline 5 days out; assert only 3d and 1d
  notifications (14d and 7d have passed).
- `TestDeadlineSchedule_DeadlinePassed` — `now` is after deadline; assert empty slice.
- `TestBuildNotifications_ObjectionDeadline` — 3 days before NTAA s.34 window closes;
  assert body contains "30 days" or "objection" and the statutory reference.

---

### 5. Support console — `internal/support`

The data minimisation principle from the NDPA: a support agent must be able to
understand what is wrong with a return without seeing the taxpayer's financial data.

Concretely, the support view must answer:
- What state is this return in?
- What validation errors are blocking it?
- Has a pack been issued? Has the taxpayer attested to filing?
- Are there payroll proposals? Have they been accepted?
- What is the timeline of state transitions?

The support view must NOT reveal:
- Any Kobo amount (income, relief, tax, balance)
- NIN, BVN, Tax ID
- Employer names or Tax IDs
- Evidence document IDs

```go
// SupportView is the redacted view of a return projection. It contains enough
// to diagnose and explain a return's status to a support agent without exposing
// any financial data or personal identifiers.
//
// This type must never grow a money field. Any request to add one must be
// rejected and escalated — a support agent who sees salary data is an NDPA
// liability and an insider-threat surface.
type SupportView struct {
    ReturnID         string
    YearOfAssessment int
    Jurisdiction     string
    State            returns.State
    Version          int            // event log sequence number
    AssessmentStale  bool
    RuleSetPublished bool

    // Income is described by category only, with no amounts.
    IncomeCategoryCount map[string]int // e.g. {"EMPLOYMENT": 2, "DIVIDEND": 1}
    HasCreditClaimed    bool           // PAYE or other credit exists, no amount
    HasReliefClaimed    bool           // at least one relief, no code or amount

    // Validation — codes and severities only.
    BlockingIssueCodes []string // ["E001", "E005"]
    WarningCodes       []string // ["W101", "W109"]

    // Payroll proposals — status only, no amounts or employer identifiers.
    PayrollProposalCount  int
    PayrollAcceptedCount  int
    PayrollRejectedCount  int

    // Lifecycle flags — what has happened.
    PackIssued    bool
    Filed         bool // AttestedAt != nil
    AckUploaded   bool
    Assessed      bool
    Paid          bool

    // Timeline — state + timestamp, no amounts.
    Timeline []SupportTimelineEntry
}

type SupportTimelineEntry struct {
    State       returns.State
    OccurredAt  time.Time
}

// BuildSupportView constructs the redacted view. It takes the Projection and
// the BlockingIssueCodes and WarningCodes from the most recent ValidationRun
// (which are already on the Projection as strings — no financial data).
func BuildSupportView(p *returns.Projection) SupportView
```

**Redaction audit:** every field on `SupportView` must have a comment explaining why
it is safe to show an agent. If you cannot write that comment, the field should not
be there. This is not decoration — it is the audit trail that satisfies the data
minimisation review.

#### 5b. Required tests

- `TestBuildSupportView_NeverExposesAmounts` — project has employment income of ₦9m
  and PAYE credit of ₦1.6m; assert no string in any field of `SupportView` contains
  "9" followed by digits that could reconstruct the amount. (A simple substring check
  is sufficient — the point is that no amount leaks as a formatted string.)
- `TestBuildSupportView_IncomeCategoriesCorrect` — two EMPLOYMENT lines and one
  DIVIDEND line; assert `IncomeCategoryCount["EMPLOYMENT"] == 2`.
- `TestBuildSupportView_BlockingCodesPresent` — projection has blocking issues
  `["E001", "E005"]`; assert they appear in `BlockingIssueCodes`.
- `TestBuildSupportView_PayrollProposalCounts` — two proposals: one ACCEPTED, one
  REJECTED; assert counts match.
- `TestBuildSupportView_TimelineOrdered` — fold events in ascending seq order; assert
  `Timeline` is also ascending by OccurredAt.
- `TestBuildSupportView_NeverExposesNIN` — construct an account that, if the support
  view were to reflect it, would show a NIN fragment; assert no NIN-shaped string
  (11 contiguous digits) appears anywhere in the view's string fields.

---

### 6. April spike runbook

#### 6a. Capacity planning document — `docs/runbook-april-spike.md`

A practitioner-readable document covering:

**Traffic model** — derive from realistic assumptions, show the arithmetic:
- Target user base year 1: 50,000 accounts
- Historical LIRS filing pattern: 70% of annual filings in the final 14 days of March
- Peak period: March 18–31 (14 days)
- Peak submissions in that window: 35,000
- Daily peak: assume 20% on March 31 itself = 7,000 returns filed on deadline day
- Hourly peak (March 31, 10am–2pm Lagos time): ~2,000 returns/hour = ~33/minute
- Engine compute time: <1ms (measured in the property tests)
- Pack generation: <5ms (all in-memory)

**Bottlenecks by layer** — each with a concrete mitigation:

| Layer | What spikes | Mitigation |
|---|---|---|
| Tax engine (Compute) | CPU; trivially parallelisable | Goroutine pool; 1 core handles >10,000/second |
| Pack generator | Memory (portal map in each request) | Singleton PortalMap per process startup |
| Registry | Read-heavy; rule sets are immutable | Singleton Registry; no per-request load |
| Notification delivery | WhatsApp/SMS rate limits | Queue + exponential backoff; shed load gracefully |
| NIN verification (if enabled) | JRB API rate limit | Cache positive verifications for 30 days; never re-verify in the same session |
| LIRS portal (etax.lirs.net) | The portal itself may slow down | The pack is a transcription script; if the portal is down, the pack is still valid and the user retries manually |

**Degradation plan** — ordered by severity:
1. NIN verification latency → accept submission, queue verification, notify when complete
2. WhatsApp API rate limit → fall back to SMS; never drop notifications silently
3. JRB API unavailable → disable NIN verification entirely; log for retry; do not block filing
4. Database write latency → the event log is append-only; backpressure at the HTTP layer

**Pre-spike checklist** (2 weeks before March 31):
- [ ] Shadow replay all stored assessments against the current rule set (TASK-007)
- [ ] Notification schedule deployed: March 17, March 24, March 28, March 30
- [ ] Stub NIN verifier is NOT deployed to production
- [ ] `StubPaymentGateway.AlwaysSucceeds` is NOT set to true in production config
- [ ] Registry loaded and smoke-tested with `reg.Get("NG", 2026)`
- [ ] Pack generator smoke-tested with a known good projection
- [ ] Support console access verified for all support agents

#### 6b. Capacity assertions — `internal/capacity/capacity_test.go`

New package. Pure in-memory. No external calls, no time.Now().

```go
// TestEngineComputeUnder1ms — run Compute 1,000 times with a representative
// TaxFacts; assert all complete in under 1ms each. Uses testing.B's timer
// if run as a benchmark, or a simple loop with time.Since if run as a test.

// TestPackGeneratorUnder5ms — run Generate 1,000 times with a pre-loaded
// PortalMap (loaded once, singleton) and a representative Projection; assert
// all complete in under 5ms each.

// TestRegistryLookupUnder100µs — run reg.Get("NG", 2026) 10,000 times;
// assert all complete in under 100µs each.

// TestNotificationBuildUnder1ms — run BuildNotifications 1,000 times with
// NotifDeadlineApproaching and a realistic NotifData; assert all under 1ms.
```

These tests are not load tests — they run in CI and fail if a refactor accidentally
makes a hot path 10× slower. They are not benchmarks; they are correctness tests for
performance contracts.

Additionally, add a `TestNoSingletonInEngine` test that confirms `taxengine.Compute`
uses no package-level mutable state. This test creates 100 goroutines, each calling
`Compute` simultaneously with different facts, and asserts no race condition. Run with
`-race`.

---

### 7. Harness and CI

`go test ./... -count=1` stays green. Import graph for the new packages:

| Package | Allowed imports |
|---|---|
| `internal/ingestion` | `taxengine`, stdlib |
| `internal/auth` | `golang.org/x/crypto/bcrypt`, stdlib |
| `internal/billing` | stdlib only |
| `internal/notify` | `taxengine`, stdlib |
| `internal/support` | `internal/returns`, stdlib |
| `internal/capacity` | `taxengine`, `internal/pack`, `internal/rulesregistry`, stdlib |

`internal/auth` requires `golang.org/x/crypto/bcrypt` — this is the only new external
dependency this task adds to `go.mod`. Add it.

The isolation test (`taxengine` and `internal/returns` import nothing outside stdlib)
must still pass.

## Do not do

- Do not implement an HTTP API, REST or otherwise. All types are Go structs. The
  transport layer is a separate task.
- Do not integrate with any external service: JRB, NRS, NIBSS, Paystack, Flutterwave,
  Twilio, Africa's Talking, WhatsApp Business API, or any open banking provider. Every
  external dependency in this task is an interface with a stub.
- Do not implement PDF parsing, Excel parsing, or OCR. The `PDFPayslipExtractor` is an
  interface only.
- Do not add salary amounts or financial data to `SupportView`. If a field looks like
  it might leak a Kobo amount, it does not belong on the struct.
- Do not use `float64` for money anywhere in the ingestion pipeline. The CSV parser must
  use `golden.ParseNaira` or equivalent integer arithmetic.
- Do not call `time.Now()` in any test subject. All clocks are injected.
- Do not change any figure in `rules/ng-2026.yaml` or any golden fixture.
- Do not build the frontend wizard, PDF renderer, or WhatsApp message template UI.
  Bodies are plain strings; rendering is the caller's concern.
- Do not implement a multi-tenancy model or employer portal. One account, one taxpayer.

## Report back with

- **Ingestion**: `TestCSVIngester_QuarantinesBadRow` walkthrough — exact input CSV,
  the quarantine record produced, the `Notes` content.
- **Auth**: the recovery state machine as a state-transition table (State × Event →
  Next State × Error). Confirmation that OTP comparison uses `bcrypt.CompareHashAndPassword`
  and not `bytes.Equal` or `==`.
- **Billing**: the full `Entitlements` table for each tier, and the `CanAccessYear`
  logic for STANDARD tier with an example: year 2024 with currentYear=2026.
- **Notify**: the rendered WhatsApp body and SMS body for `NotifDeadlineApproaching`
  with 3 days remaining and a ₦100,000 accrued penalty. Confirm SMS ≤ 160 chars.
- **Support console**: the complete `SupportView` field list with the redaction
  comment for each field. Confirmation that `TestBuildSupportView_NeverExposesAmounts`
  passes, including the specific regex or substring check used.
- **April spike**: the traffic model arithmetic (show the workings), the complete
  pre-spike checklist, and `TestEngineComputeUnder1ms` pass/fail and actual p99 time.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1 -race` output.
