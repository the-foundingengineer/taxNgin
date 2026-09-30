package kyc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"taxsoftware/internal/auth"
)

const dojahBaseURL = "https://api.dojah.io/api/v1"

// DojahClient implements auth.NINVerifier via the Dojah API.
type DojahClient struct {
	appID    string
	authKey  string
	baseURL  string
	http     *http.Client
}

// NewDojah returns a Dojah client. Returns (nil, nil) if credentials are absent.
func NewDojah(appID, authKey string) (*DojahClient, error) {
	if appID == "" || authKey == "" {
		return nil, nil
	}
	return &DojahClient{
		appID:   appID,
		authKey: authKey,
		baseURL: dojahBaseURL,
		http:    &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// Verify cross-checks a NIN against the Dojah identity database.
func (c *DojahClient) Verify(req auth.NINVerificationRequest) (auth.NINRecord, error) {
	url := fmt.Sprintf("%s/kyc/nin?nin=%s", c.baseURL, req.NIN)

	httpReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return auth.NINRecord{}, fmt.Errorf("kyc: build request: %w", err)
	}
	httpReq.Header.Set("AppId", c.appID)
	httpReq.Header.Set("Authorization", c.authKey)

	res, err := c.http.Do(httpReq)
	if err != nil {
		return auth.NINRecord{}, auth.ErrVerificationUnavailable
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized || res.StatusCode >= 500 {
		return auth.NINRecord{}, auth.ErrVerificationUnavailable
	}

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return auth.NINRecord{}, auth.ErrVerificationUnavailable
	}

	var resp struct {
		Entity struct {
			NIN string `json:"nin"`
		} `json:"entity"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return auth.NINRecord{}, fmt.Errorf("kyc: parse response: %w", err)
	}

	if resp.Error != "" {
		return auth.NINRecord{}, fmt.Errorf("kyc: NIN not verified (%s)", resp.Error)
	}
	
	// A successful HTTP 200 with no error indicates the NIN exists.
	// Dojah typically returns the NIN payload in the Entity field.
	if resp.Entity.NIN == "" {
		return auth.NINRecord{Verified: false}, nil
	}

	return auth.NINRecord{
		NIN:      resp.Entity.NIN,
		Verified: true,
	}, nil
}
