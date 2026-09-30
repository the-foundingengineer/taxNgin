package billing_test

import (
	"math"
	"testing"
	"time"

	"taxsoftware/internal/billing"
)

var now = time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC)

// ---------------------------------------------------------------------------
// Entitlements
// ---------------------------------------------------------------------------

func TestEntitlementsFor_Free(t *testing.T) {
	e := billing.EntitlementsFor(billing.TierFree)
	if e.MaxYearsOfAssessment != 1 {
		t.Errorf("Free MaxYears: want 1, got %d", e.MaxYearsOfAssessment)
	}
	if e.EmployerPreFill || e.TCCTracking || e.MultiYearCatchUp || e.PrioritySupport {
		t.Error("Free tier must have all feature flags false")
	}
}

func TestEntitlementsFor_Standard(t *testing.T) {
	e := billing.EntitlementsFor(billing.TierStandard)
	if e.MaxYearsOfAssessment != 2 {
		t.Errorf("Standard MaxYears: want 2, got %d", e.MaxYearsOfAssessment)
	}
	if !e.EmployerPreFill {
		t.Error("Standard must have EmployerPreFill=true")
	}
	if e.TCCTracking || e.MultiYearCatchUp || e.PrioritySupport {
		t.Error("Standard must not have Premium-only features")
	}
}

func TestEntitlementsFor_Premium(t *testing.T) {
	e := billing.EntitlementsFor(billing.TierPremium)
	if e.MaxYearsOfAssessment != math.MaxInt {
		t.Errorf("Premium MaxYears: want MaxInt, got %d", e.MaxYearsOfAssessment)
	}
	if !e.EmployerPreFill || !e.TCCTracking || !e.MultiYearCatchUp || !e.PrioritySupport {
		t.Error("Premium must have all feature flags true")
	}
}

func TestEntitlementsFor_UnknownTierFallsBackToFree(t *testing.T) {
	e := billing.EntitlementsFor(billing.Tier("ENTERPRISE"))
	if e.MaxYearsOfAssessment != 1 {
		t.Errorf("Unknown tier: want Free (MaxYears=1), got %d", e.MaxYearsOfAssessment)
	}
}

func TestCan_AllTiersCanFileReturn(t *testing.T) {
	for _, tier := range []billing.Tier{billing.TierFree, billing.TierStandard, billing.TierPremium} {
		if !billing.EntitlementsFor(tier).Can(billing.ActionFileReturn) {
			t.Errorf("Tier %s: must be able to file a return", tier)
		}
	}
}

func TestCan_FreeCannotAccessPriorYear(t *testing.T) {
	e := billing.EntitlementsFor(billing.TierFree)
	if e.Can(billing.ActionAccessPriorYear) {
		t.Error("Free tier must not access prior years")
	}
}

func TestCan_StandardCanAccessOnePriorYear(t *testing.T) {
	e := billing.EntitlementsFor(billing.TierStandard)
	// year N-1 is accessible
	if !e.CanAccessYear(2025, 2026) {
		t.Error("Standard: year N-1 (2025) should be accessible when current is 2026")
	}
	// year N-2 is not
	if e.CanAccessYear(2024, 2026) {
		t.Error("Standard: year N-2 (2024) should NOT be accessible when current is 2026")
	}
}

func TestCan_PremiumCanAccessAllPriorYears(t *testing.T) {
	e := billing.EntitlementsFor(billing.TierPremium)
	for _, year := range []int{2026, 2025, 2024, 2023, 2020, 2010} {
		if !e.CanAccessYear(year, 2026) {
			t.Errorf("Premium: year %d should be accessible", year)
		}
	}
}

func TestCanAccessYear_FutureYearNeverPermitted(t *testing.T) {
	e := billing.EntitlementsFor(billing.TierPremium)
	if e.CanAccessYear(2027, 2026) {
		t.Error("future year (2027) should never be permitted")
	}
}

func TestCan_TCCTrackingOnlyForPremium(t *testing.T) {
	if billing.EntitlementsFor(billing.TierFree).Can(billing.ActionViewTCCReadiness) {
		t.Error("Free: TCC tracking not permitted")
	}
	if billing.EntitlementsFor(billing.TierStandard).Can(billing.ActionViewTCCReadiness) {
		t.Error("Standard: TCC tracking not permitted")
	}
	if !billing.EntitlementsFor(billing.TierPremium).Can(billing.ActionViewTCCReadiness) {
		t.Error("Premium: TCC tracking must be permitted")
	}
}

// ---------------------------------------------------------------------------
// CurrentTier
// ---------------------------------------------------------------------------

func TestCurrentTier_NoEvents(t *testing.T) {
	tier := billing.CurrentTier(nil, now)
	if tier != billing.TierFree {
		t.Errorf("no events: want Free, got %s", tier)
	}
}

func TestCurrentTier_AfterExpiry(t *testing.T) {
	events := []billing.BillingEvent{
		{Kind: billing.BillingSubscriptionStarted, Tier: billing.TierStandard,
			OccurredAt: now.AddDate(0, -1, 0)},
		{Kind: billing.BillingSubscriptionExpired, Tier: billing.TierFree,
			OccurredAt: now.AddDate(0, 0, -1)},
	}
	tier := billing.CurrentTier(events, now)
	if tier != billing.TierFree {
		t.Errorf("after expiry: want Free, got %s", tier)
	}
}

func TestCurrentTier_Renewal(t *testing.T) {
	events := []billing.BillingEvent{
		{Kind: billing.BillingSubscriptionStarted, Tier: billing.TierStandard,
			OccurredAt: now.AddDate(-1, 0, 0)},
		{Kind: billing.BillingSubscriptionRenewed, Tier: billing.TierPremium,
			OccurredAt: now.AddDate(0, -1, 0)},
	}
	tier := billing.CurrentTier(events, now)
	if tier != billing.TierPremium {
		t.Errorf("after renewal to Premium: want Premium, got %s", tier)
	}
}

func TestCurrentTier_OutOfOrderEventsHandled(t *testing.T) {
	// Events provided out of chronological order — sort must fix this.
	events := []billing.BillingEvent{
		{Kind: billing.BillingSubscriptionExpired, Tier: billing.TierFree,
			OccurredAt: now.AddDate(0, 0, -1)},
		{Kind: billing.BillingSubscriptionStarted, Tier: billing.TierStandard,
			OccurredAt: now.AddDate(0, -2, 0)},
	}
	tier := billing.CurrentTier(events, now)
	if tier != billing.TierFree {
		t.Errorf("expired after start: want Free, got %s", tier)
	}
}
