// Package kyc provides NIN verification via Interswitch Data Service.
//
// Interswitch uses OAuth 2.0 client credentials. Access tokens are cached
// and refreshed automatically before expiry.
//
// Required environment variables:
//
//	INTERSWITCH_CLIENT_ID     — from the Interswitch developer console
//	INTERSWITCH_CLIENT_SECRET — from the Interswitch developer console
//	INTERSWITCH_ENV           — "sandbox" (default) or "production"
//
// When credentials are absent, New returns (nil, nil) and the caller falls
// back to creating accounts as PENDING_VERIFICATION.
//
// Pricing: ₦64 per NIN lookup (production).
// Sandbox test NIN: 11111111111
package kyc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"taxsoftware/internal/auth"
)

const (
	iswSandboxPassport = "https://sandbox.interswitchng.com/passport/oauth/token"
	iswProdPassport    = "https://passport.interswitchng.com/passport/oauth/token"

	iswSandboxVerify = "https://sandbox.interswitchng.com/isw-data-service/api/v1/request/verification"
	iswProdVerify    = "https://api-gateway.interswitchng.com/isw-data-service/api/v1/request/verification"
)

// Client implements auth.NINVerifier via Interswitch Data Service.
type Client struct {
	clientID     string
	clientSecret string
	passportURL  string
	verifyURL    string
	http         *http.Client

	mu          sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

// New returns an Interswitch client. Returns (nil, nil) when credentials are
// absent — the caller should treat nil as "verification not configured".
func New(clientID, clientSecret, env string) (*Client, error) {
	if clientID == "" || clientSecret == "" {
		return nil, nil
	}
	passportURL := iswSandboxPassport
	verifyURL := iswSandboxVerify
	if env == "production" {
		passportURL = iswProdPassport
		verifyURL = iswProdVerify
	}
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		passportURL:  passportURL,
		verifyURL:    verifyURL,
		http:         &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// Verify cross-checks a NIN against the Interswitch identity database.
func (c *Client) Verify(req auth.NINVerificationRequest) (auth.NINRecord, error) {
	token, err := c.accessToken()
	if err != nil {
		return auth.NINRecord{}, auth.ErrVerificationUnavailable
	}

	body, err := json.Marshal(map[string]string{
		"firstname":      req.FirstName,
		"lastname":       req.LastName,
		"birthDate":      req.BirthDate,
		"gender":         req.Gender,
		"phone":          req.Phone,
		"validationType": "NIN",
		"validationId":   req.NIN,
	})
	if err != nil {
		return auth.NINRecord{}, fmt.Errorf("kyc: marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.verifyURL, strings.NewReader(string(body)))
	if err != nil {
		return auth.NINRecord{}, fmt.Errorf("kyc: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(httpReq)
	if err != nil {
		return auth.NINRecord{}, auth.ErrVerificationUnavailable
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return auth.NINRecord{}, auth.ErrVerificationUnavailable
	}

	var resp struct {
		ResponseCode    string `json:"responseCode"`
		ResponseMessage string `json:"responseMessage"`
		Data            struct {
			Status         string `json:"status"`
			IdentityNumber string `json:"identityNumber"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return auth.NINRecord{}, fmt.Errorf("kyc: parse response: %w", err)
	}

	if res.StatusCode == http.StatusUnauthorized || res.StatusCode >= 500 {
		return auth.NINRecord{}, auth.ErrVerificationUnavailable
	}
	if resp.ResponseCode != "00" {
		return auth.NINRecord{}, fmt.Errorf("kyc: NIN not verified (%s: %s)", resp.ResponseCode, resp.ResponseMessage)
	}

	return auth.NINRecord{
		NIN:      resp.Data.IdentityNumber,
		Verified: resp.Data.Status == "VERIFIED",
	}, nil
}

// accessToken returns a cached OAuth token, fetching a new one if needed.
func (c *Client) accessToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cachedToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.cachedToken, nil
	}

	creds := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	form := url.Values{"grant_type": {"client_credentials"}}

	req, err := http.NewRequest(http.MethodPost, c.passportURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+creds)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("kyc: token fetch: %w", err)
	}
	defer res.Body.Close()

	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(res.Body).Decode(&tok); err != nil {
		return "", fmt.Errorf("kyc: token parse: %w", err)
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("kyc: empty access token")
	}

	c.cachedToken = tok.AccessToken
	// Expire the cache 60s before actual expiry to avoid using a token that
	// just ticked over on the server side.
	c.tokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn-60) * time.Second)
	return c.cachedToken, nil
}
