// Package remita wraps the Remita e-Channels API used to generate RRR codes
// for LIRS tax payments. An RRR (Remita Retrieval Reference) is a 12-digit
// code the taxpayer uses to pay at any Nigerian bank branch, internet banking
// portal, or the Remita payment portal.
//
// API reference: https://remita.net/developers
package remita

import (
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	baseSandbox = "https://remitademo.net/remita/exapp/api/v1/send/api/echannelsvc"
	baseLive    = "https://login.remita.net/remita/exapp/api/v1/send/api/echannelsvc"
)

// Config holds the credentials Remita assigns after account approval.
// All three fields are required; Env defaults to "live" when blank.
type Config struct {
	MerchantID    string
	ServiceTypeID string
	APIKey        string
	// Env is "demo" for sandbox testing, "live" (or blank) for production.
	Env string
}

// Client wraps the Remita e-Channels API. Create with New.
type Client struct {
	cfg  Config
	base string
	http *http.Client
}

// New returns a Client ready to make API calls.
func New(cfg Config) *Client {
	base := baseLive
	if cfg.Env == "demo" {
		base = baseSandbox
	}
	return &Client{
		cfg:  cfg,
		base: base,
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// sha512hex returns the lower-case hex SHA-512 of the concatenated parts.
// Remita uses this for all request authentication hashes.
func sha512hex(parts ...string) string {
	h := sha512.New()
	for _, p := range parts {
		h.Write([]byte(p))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// authHeader builds the Authorization header value Remita expects.
func (c *Client) authHeader(token string) string {
	return fmt.Sprintf("remitaConsumerKey=%s,remitaConsumerToken=%s", c.cfg.MerchantID, token)
}

// GenerateRRRRequest carries payer details for one LIRS tax payment.
type GenerateRRRRequest struct {
	// AmountNaira is the tax amount in Nigerian Naira (not kobo).
	AmountNaira float64
	// OrderID is your unique reference for this payment (e.g. the LIRS e-file ref).
	// Must be unique per transaction; Remita rejects duplicate order IDs.
	OrderID    string
	PayerName  string
	PayerEmail string
	PayerPhone string
}

// GenerateRRR calls the Remita payment-init endpoint and returns a 12-digit RRR.
// The RRR is valid for payment at any Nigerian bank for 3 banking days.
func (c *Client) GenerateRRR(req GenerateRRRRequest) (string, error) {
	amountStr := fmt.Sprintf("%.2f", req.AmountNaira)
	h := sha512hex(c.cfg.MerchantID, c.cfg.ServiceTypeID, req.OrderID, amountStr, c.cfg.APIKey)

	body, _ := json.Marshal(map[string]string{
		"serviceTypeId": c.cfg.ServiceTypeID,
		"amount":        amountStr,
		"orderId":       req.OrderID,
		"payerName":     req.PayerName,
		"payerEmail":    req.PayerEmail,
		"payerPhone":    req.PayerPhone,
		"description":   "LIRS Annual Tax Payment",
	})

	httpReq, err := http.NewRequest(
		http.MethodPost,
		c.base+"/merchant/api/paymentinit",
		strings.NewReader(string(body)),
	)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json;charset=UTF-8")
	httpReq.Header.Set("Authorization", c.authHeader(h))

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("remita: generate RRR: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		StatusCode string `json:"statuscode"`
		RRR        string `json:"RRR"`
		Status     string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("remita: generate RRR decode: %w", err)
	}
	// "025" is the Remita success status for RRR generation.
	if out.StatusCode != "025" {
		return "", fmt.Errorf("remita: generate RRR: %s (code %s)", out.Status, out.StatusCode)
	}
	return out.RRR, nil
}

// IsPaid checks whether a previously generated RRR has been settled.
// Returns true only when Remita confirms a successful payment (status "01").
func (c *Client) IsPaid(rrr string) (bool, error) {
	h := sha512hex(c.cfg.MerchantID, rrr, c.cfg.APIKey)
	url := fmt.Sprintf("%s/%s/%s/%s/status.reg", c.base, c.cfg.MerchantID, rrr, h)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", c.authHeader(h))

	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("remita: verify RRR: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, fmt.Errorf("remita: verify RRR decode: %w", err)
	}
	return out.Status == "01", nil
}
