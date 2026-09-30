// Package billing models product tiers and the entitlements they grant.
// Entitlements gate which product features are available to a given account.
// Billing events record the history of tier changes; the current tier is
// derived from that log rather than stored as a single mutable field.
package billing

import (
	"errors"
	"math"
	"sort"
	"time"
)

// Tier is the subscription level for a taxpayer account.
type Tier string

const (
	// TierFree allows filing one return for the current year only.
	// No employer pre-fill, no TCC tracking.
	TierFree Tier = "FREE"

	// TierStandard adds employer pre-fill and access to one prior year.
	TierStandard Tier = "STANDARD"

	// TierPremium gives unlimited year access, TCC tracking, multi-year
	// catch-up filing, and priority support.
	TierPremium Tier = "PREMIUM"
)

// Action is a product feature that may or may not be permitted by a tier.
type Action string

const (
	ActionFileReturn       Action = "file_return"
	ActionAccessPriorYear  Action = "access_prior_year"
	ActionEmployerPreFill  Action = "employer_prefill"
	ActionViewTCCReadiness Action = "view_tcc_readiness"
	ActionMultiYearCatchUp Action = "multi_year_catchup"
)

// Entitlements describes what a given tier permits.
type Entitlements struct {
	// MaxYearsOfAssessment is the number of distinct years an account may
	// access. 1 = current year only; math.MaxInt = unlimited.
	MaxYearsOfAssessment int
	EmployerPreFill      bool
	TCCTracking          bool
	MultiYearCatchUp     bool
	PrioritySupport      bool
}

// EntitlementsFor returns the entitlements for the given tier.
// Unknown tiers are treated as TierFree.
func EntitlementsFor(tier Tier) Entitlements {
	switch tier {
	case TierStandard:
		return Entitlements{
			MaxYearsOfAssessment: 2,
			EmployerPreFill:      true,
		}
	case TierPremium:
		return Entitlements{
			MaxYearsOfAssessment: math.MaxInt,
			EmployerPreFill:      true,
			TCCTracking:          true,
			MultiYearCatchUp:     true,
			PrioritySupport:      true,
		}
	default: // TierFree and unknown tiers
		return Entitlements{MaxYearsOfAssessment: 1}
	}
}

// Can reports whether the entitlements permit the requested action.
func (e Entitlements) Can(action Action) bool {
	switch action {
	case ActionFileReturn:
		return true // all tiers can file at least one return
	case ActionAccessPriorYear:
		return e.MaxYearsOfAssessment > 1
	case ActionEmployerPreFill:
		return e.EmployerPreFill
	case ActionViewTCCReadiness:
		return e.TCCTracking
	case ActionMultiYearCatchUp:
		return e.MultiYearCatchUp
	}
	return false
}

// CanAccessYear reports whether the entitlements allow access to the given
// year of assessment. yearsBack = currentYear - yearOfAssessment.
// A future year (yearsBack < 0) is never permitted.
func (e Entitlements) CanAccessYear(yearOfAssessment, currentYear int) bool {
	yearsBack := currentYear - yearOfAssessment
	if yearsBack < 0 {
		return false
	}
	return yearsBack < e.MaxYearsOfAssessment
}

// ---------------------------------------------------------------------------
// Billing event log
// ---------------------------------------------------------------------------

// BillingEventKind is a fact in the account's billing history.
type BillingEventKind string

const (
	BillingSubscriptionStarted BillingEventKind = "SUBSCRIPTION_STARTED"
	BillingSubscriptionRenewed BillingEventKind = "SUBSCRIPTION_RENEWED"
	BillingSubscriptionExpired BillingEventKind = "SUBSCRIPTION_EXPIRED"
	BillingPaymentFailed       BillingEventKind = "PAYMENT_FAILED"
	BillingDowngraded          BillingEventKind = "DOWNGRADED"
)

// BillingEvent is one fact in the billing history of an account. Events are
// never deleted; the current state is derived by replaying them in order.
type BillingEvent struct {
	ID         string
	AccountID  string
	Kind       BillingEventKind
	Tier       Tier
	OccurredAt time.Time
	// Reference is the payment provider's transaction reference.
	// Empty for system-generated events (e.g. SUBSCRIPTION_EXPIRED).
	Reference string
}

// CurrentTier derives the active tier from the billing event log.
// Events are sorted by OccurredAt before processing. If no events exist,
// returns TierFree.
func CurrentTier(events []BillingEvent, _ time.Time) Tier {
	if len(events) == 0 {
		return TierFree
	}
	sorted := make([]BillingEvent, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].OccurredAt.Before(sorted[j].OccurredAt)
	})

	tier := TierFree
	for _, ev := range sorted {
		switch ev.Kind {
		case BillingSubscriptionStarted, BillingSubscriptionRenewed:
			tier = ev.Tier
		case BillingSubscriptionExpired, BillingDowngraded:
			tier = TierFree
		}
	}
	return tier
}

// ---------------------------------------------------------------------------
// Payment gateway interface
// ---------------------------------------------------------------------------

// ErrPaymentFailed is returned when a gateway confirms a failed payment.
var ErrPaymentFailed = errors.New("billing: payment failed")

// PaymentGateway is the interface for Nigerian payment processors.
// Implement with Paystack or Flutterwave when a gateway is contracted.
type PaymentGateway interface {
	// InitiatePayment returns a redirect URL and payment reference the user
	// follows to complete payment. The callbackURL receives the browser
	// redirect when payment is done.
	InitiatePayment(accountID string, tier Tier, callbackURL string) (redirectURL, reference string, err error)
	// VerifyPayment confirms a payment using the provider's reference and
	// returns a BillingEvent to append to the account's history.
	VerifyPayment(reference string) (BillingEvent, error)
}

// WebhookVerifier verifies an inbound webhook signature from a payment provider.
type WebhookVerifier interface {
	VerifyWebhook(body []byte, signature string) bool
}

// StubPaymentGateway satisfies PaymentGateway for tests. AlwaysSucceeds
// controls whether VerifyPayment returns a successful event or an error.
// MUST NOT be deployed to production.
type StubPaymentGateway struct {
	AlwaysSucceeds bool
}

func (s StubPaymentGateway) InitiatePayment(accountID string, tier Tier, _ string) (string, string, error) {
	ref := "stub-ref-" + accountID
	return "https://stub-gateway/pay?account=" + accountID + "&tier=" + string(tier), ref, nil
}

func (s StubPaymentGateway) VerifyPayment(reference string) (BillingEvent, error) {
	if !s.AlwaysSucceeds {
		return BillingEvent{}, ErrPaymentFailed
	}
	return BillingEvent{
		Kind:      BillingSubscriptionStarted,
		Tier:      TierStandard,
		Reference: reference,
	}, nil
}
