package billing

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const paystackBase = "https://api.paystack.co"

// tierAmountKobo maps a subscription tier to its price in kobo (₦ × 100).
func tierAmountKobo(t Tier) int64 {
	switch t {
	case TierStandard:
		return 500_000 // ₦5,000
	case TierPremium:
		return 1_500_000 // ₦15,000
	default:
		return 0
	}
}

// PaystackGateway implements PaymentGateway and WebhookVerifier using the
// Paystack API. Create with NewPaystackGateway; the zero value is not usable.
type PaystackGateway struct {
	secretKey string
	client    *http.Client
}

// NewPaystackGateway returns a gateway authenticated with secretKey.
// secretKey is the live or test secret key from the Paystack dashboard.
func NewPaystackGateway(secretKey string) *PaystackGateway {
	return &PaystackGateway{
		secretKey: secretKey,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

// InitiatePayment calls Paystack /transaction/initialize and returns the
// checkout URL and reference the caller should redirect the browser to.
func (g *PaystackGateway) InitiatePayment(accountID string, tier Tier, callbackURL string) (string, string, error) {
	amountKobo := tierAmountKobo(tier)
	if amountKobo == 0 {
		return "", "", fmt.Errorf("billing: tier %q does not require payment", tier)
	}

	body, _ := json.Marshal(map[string]any{
		"email":        fmt.Sprintf("%s@noreply.taxsoftware.ng", accountID),
		"amount":       amountKobo,
		"callback_url": callbackURL,
		"metadata": map[string]string{
			"account_id": accountID,
			"tier":       string(tier),
		},
	})

	req, err := http.NewRequest(http.MethodPost, paystackBase+"/transaction/initialize", strings.NewReader(string(body)))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+g.secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("billing: paystack initialize: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			AuthorizationURL string `json:"authorization_url"`
			Reference        string `json:"reference"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", fmt.Errorf("billing: paystack initialize decode: %w", err)
	}
	if !out.Status {
		return "", "", fmt.Errorf("billing: paystack initialize: %s", out.Message)
	}
	return out.Data.AuthorizationURL, out.Data.Reference, nil
}

// VerifyPayment calls Paystack /transaction/verify/{reference} and returns a
// BillingEvent ready to be stored. Returns ErrPaymentFailed when the
// transaction exists but was not successful.
func (g *PaystackGateway) VerifyPayment(reference string) (BillingEvent, error) {
	req, err := http.NewRequest(http.MethodGet, paystackBase+"/transaction/verify/"+reference, nil)
	if err != nil {
		return BillingEvent{}, err
	}
	req.Header.Set("Authorization", "Bearer "+g.secretKey)

	resp, err := g.client.Do(req)
	if err != nil {
		return BillingEvent{}, fmt.Errorf("billing: paystack verify: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Status    string `json:"status"`
			Reference string `json:"reference"`
			Metadata  struct {
				AccountID string `json:"account_id"`
				Tier      string `json:"tier"`
			} `json:"metadata"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return BillingEvent{}, fmt.Errorf("billing: paystack verify decode: %w", err)
	}
	if !out.Status {
		return BillingEvent{}, fmt.Errorf("billing: paystack verify: %s", out.Message)
	}
	if out.Data.Status != "success" {
		return BillingEvent{}, ErrPaymentFailed
	}
	return BillingEvent{
		AccountID:  out.Data.Metadata.AccountID,
		Kind:       BillingSubscriptionStarted,
		Tier:       Tier(out.Data.Metadata.Tier),
		Reference:  out.Data.Reference,
		OccurredAt: time.Now(),
	}, nil
}

// VerifyWebhook returns true when signature is the valid Paystack HMAC-SHA512
// of body using the gateway's secret key. Call this before trusting any
// inbound webhook payload.
func (g *PaystackGateway) VerifyWebhook(body []byte, signature string) bool {
	mac := hmac.New(sha512.New, []byte(g.secretKey))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
