package transport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"taxsoftware/internal/billing"
)

// GET /v1/entitlements — return the current tier and entitlements for the
// logged-in account.
func (s *Server) handleGetEntitlements(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)

	events, err := s.billing.Events(accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load billing events")
		return
	}

	tier := billing.CurrentTier(events, s.clock())
	ent := billing.EntitlementsFor(tier)

	writeJSON(w, http.StatusOK, map[string]any{
		"tier":         string(tier),
		"entitlements": ent,
	})
}

// POST /v1/billing/upgrade — initiate a Paystack subscription payment.
// Returns {"authorization_url": "...", "reference": "..."}.
// The caller should redirect the browser to authorization_url.
func (s *Server) handleUpgrade(w http.ResponseWriter, r *http.Request) {
	if s.gateway == nil {
		writeError(w, http.StatusServiceUnavailable, "payment gateway not configured")
		return
	}

	accountID := accountIDFromCtx(r)

	var req struct {
		Tier        billing.Tier `json:"tier"`
		CallbackURL string       `json:"callback_url"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Tier == billing.TierFree || req.Tier == "" {
		writeError(w, http.StatusBadRequest, "tier must be STANDARD or PREMIUM")
		return
	}
	if req.CallbackURL == "" {
		writeError(w, http.StatusBadRequest, "callback_url is required")
		return
	}

	redirectURL, reference, err := s.gateway.InitiatePayment(accountID, req.Tier, req.CallbackURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to initiate payment")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"authorization_url": redirectURL,
		"reference":         reference,
	})
}

// POST /v1/billing/webhook — receives Paystack charge.success events.
// Paystack retries on any non-2xx response, so we return 200 for all
// terminal outcomes (invalid signature, payment failed, unknown event).
func (s *Server) handleBillingWebhook(w http.ResponseWriter, r *http.Request) {
	if s.gateway == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if v, ok := s.gateway.(billing.WebhookVerifier); ok {
		sig := r.Header.Get("x-paystack-signature")
		if !v.VerifyWebhook(body, sig) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	var event struct {
		Event string `json:"event"`
		Data  struct {
			Reference string `json:"reference"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if event.Event != "charge.success" {
		w.WriteHeader(http.StatusOK)
		return
	}

	billingEvent, err := s.gateway.VerifyPayment(event.Data.Reference)
	if err != nil {
		if errors.Is(err, billing.ErrPaymentFailed) {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	billingEvent.ID = newID()
	billingEvent.OccurredAt = s.clock()
	if err := s.billing.Append(billingEvent); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GET /v1/billing/callback — browser redirect target after Paystack checkout.
// Verifies the reference server-side then redirects to the SPA pricing page.
// The webhook is authoritative for storing the billing event; this handler
// is for UX only.
func (s *Server) handleBillingCallback(w http.ResponseWriter, r *http.Request) {
	if s.gateway == nil {
		http.Redirect(w, r, "/pricing?billing=error", http.StatusSeeOther)
		return
	}

	reference := r.URL.Query().Get("reference")
	if reference == "" {
		http.Redirect(w, r, "/pricing?billing=error", http.StatusSeeOther)
		return
	}

	if _, err := s.gateway.VerifyPayment(reference); err != nil {
		http.Redirect(w, r, "/pricing?billing=error", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/pricing?billing=success", http.StatusSeeOther)
}
