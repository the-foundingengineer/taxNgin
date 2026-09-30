// Package transport wires the domain packages to HTTP. All business logic
// stays in the domain packages; this package only translates HTTP requests
// into domain calls and domain results into HTTP responses.
//
// Routes use the Go 1.22 net/http pattern syntax: "METHOD /path/{param}".
// No external framework. No global state. The Server struct holds all
// dependencies; tests construct one directly without starting a real listener.
package transport

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"taxsoftware/internal/auth"
	"taxsoftware/internal/billing"
	"taxsoftware/internal/portalmap"
	"taxsoftware/internal/remita"
	"taxsoftware/internal/returns"
	"taxsoftware/internal/rulesregistry"
)

// Config bundles the dependencies the Server needs. Every field is required
// unless a sensible zero value exists (Clock defaults to time.Now).
type Config struct {
	Accounts  AccountStore
	Events    EventStore
	Sessions  SessionStore
	Recovery  RecoveryStore
	Billing   BillingStore
	Registry  *rulesregistry.Registry
	PortalMap portalmap.PortalMap
	// HMACKey is used to hash NIN/BVN before storage — same scheme as
	// internal/employer. Must be the same value across all processes and
	// deployments; rotate by recomputing all stored HMACs.
	HMACKey []byte
	// NINVerifier checks a NIN against the government identity database.
	// When nil, NIN verification is deferred and accounts are created as
	// PENDING_VERIFICATION. Set to auth.StubNINVerifier{} only in tests.
	NINVerifier auth.NINVerifier
	// PaymentGateway processes subscription fee payments via Paystack.
	// When nil, upgrade attempts return 503 Service Unavailable.
	PaymentGateway billing.PaymentGateway
	// RemitaClient generates RRR codes for LIRS tax payments.
	// When nil, Remita endpoints return 503 Service Unavailable.
	RemitaClient *remita.Client
	// Clock is injected for testing. Defaults to time.Now if nil.
	Clock func() time.Time
}

// Server is the HTTP handler for the tax product API. Create with New.
type Server struct {
	mux      *http.ServeMux
	accounts AccountStore
	events   EventStore
	sessions SessionStore
	recovery RecoveryStore
	billing  BillingStore
	registry *rulesregistry.Registry
	pm          portalmap.PortalMap
	hmacKey     []byte
	ninVerifier auth.NINVerifier
	clock       func() time.Time

	gateway      billing.PaymentGateway
	remitaClient *remita.Client
	authLimiters *limiterStore
	apiLimiters  *limiterStore
}

// New wires all routes and returns the server. The returned *Server implements
// http.Handler and can be passed directly to httptest.NewServer or http.ListenAndServe.
func New(cfg Config) *Server {
	clk := cfg.Clock
	if clk == nil {
		clk = time.Now
	}
	s := &Server{
		mux:      http.NewServeMux(),
		accounts: cfg.Accounts,
		events:   cfg.Events,
		sessions: cfg.Sessions,
		recovery: cfg.Recovery,
		billing:  cfg.Billing,
		registry: cfg.Registry,
		pm:           cfg.PortalMap,
		hmacKey:      cfg.HMACKey,
		ninVerifier:  cfg.NINVerifier,
		gateway:      cfg.PaymentGateway,
		remitaClient: cfg.RemitaClient,
		clock:        clk,
		authLimiters: newLimiterStore(authRate, authBurst),
		apiLimiters:  newLimiterStore(apiRate, apiBurst),
	}
	s.routes()
	return s
}

// ServeHTTP runs every request through the global middleware chain:
//   request ID → per-request Sentry hub → panic recovery/5xx capture → router
//
// Rate limiting is applied per-endpoint-group in routes(), not here, so that
// the /v1/onboarding/schema health-check path is never throttled.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	withRequestID(
		withSentryHub(
			withRecovery(s.mux),
		),
	).ServeHTTP(w, r)
}

func (s *Server) routes() {
	authRL := rateLimit(s.authLimiters, ipKey)
	apiRL  := rateLimit(s.apiLimiters, tokenKey)

	// Public — no rate limiting. Schema is static and doubles as a health check.
	s.mux.HandleFunc("GET /v1/onboarding/schema", s.handleGetOnboardingSchema)

	// Unauthenticated auth endpoints — strict IP-based rate limit.
	// Prevents credential stuffing, account enumeration, and OTP brute force.
	s.mux.Handle("POST /v1/accounts",              authRL(http.HandlerFunc(s.handleRegister)))
	s.mux.Handle("POST /v1/sessions",              authRL(http.HandlerFunc(s.handleLogin)))
	s.mux.Handle("POST /v1/recovery",              authRL(http.HandlerFunc(s.handleInitiateRecovery)))
	s.mux.Handle("POST /v1/recovery/{id}/otp",     authRL(http.HandlerFunc(s.handleSubmitOTP)))
	s.mux.Handle("POST /v1/recovery/{id}/complete", authRL(http.HandlerFunc(s.handleCompleteRecovery)))

	// Authenticated — all keyed by Bearer token, not IP.
	// This prevents one large corporate or carrier NAT from blocking other users.
	s.mux.Handle("DELETE /v1/sessions", apiRL(s.auth(s.handleLogout)))
	s.mux.Handle("POST /v1/accounts/verify",                              apiRL(s.auth(s.handleVerifyNIN)))

	s.mux.Handle("GET /v1/returns",                                       apiRL(s.auth(s.handleListReturns)))
	s.mux.Handle("POST /v1/returns",                                      apiRL(s.auth(s.handleOpenReturn)))
	s.mux.Handle("GET /v1/returns/{id}",                                  apiRL(s.auth(s.handleGetReturn)))
	s.mux.Handle("POST /v1/returns/{id}/income",                          apiRL(s.auth(s.handleDeclareIncome)))
	s.mux.Handle("DELETE /v1/returns/{id}/income/{income_id}",            apiRL(s.auth(s.handleRemoveIncome)))
	s.mux.Handle("POST /v1/returns/{id}/reliefs",                         apiRL(s.auth(s.handleClaimRelief)))
	s.mux.Handle("DELETE /v1/returns/{id}/reliefs/{code}",                apiRL(s.auth(s.handleRemoveRelief)))
	s.mux.Handle("POST /v1/returns/{id}/assess",                          apiRL(s.auth(s.handleAssess)))
	s.mux.Handle("POST /v1/returns/{id}/validate",                        apiRL(s.auth(s.handleValidate)))
	s.mux.Handle("POST /v1/returns/{id}/sign",                            apiRL(s.auth(s.handleSign)))
	s.mux.Handle("POST /v1/returns/{id}/pack",                            apiRL(s.auth(s.handleIssuePack)))
	s.mux.Handle("GET /v1/returns/{id}/pack",                             apiRL(s.auth(s.handleGetPack)))
	s.mux.Handle("POST /v1/returns/{id}/attest",                          apiRL(s.auth(s.handleAttest)))
	s.mux.Handle("POST /v1/returns/{id}/ack",                             apiRL(s.auth(s.handleUploadAck)))
	s.mux.Handle("POST /v1/returns/{id}/accommodation",                   apiRL(s.auth(s.handleDeclareAccommodation)))
	s.mux.Handle("POST /v1/returns/{id}/payroll/{record_id}/accept",      apiRL(s.auth(s.handleAcceptPayroll)))
	s.mux.Handle("POST /v1/returns/{id}/payroll/{record_id}/reject",      apiRL(s.auth(s.handleRejectPayroll)))
	s.mux.Handle("POST /v1/returns/{id}/ingest/csv",                      apiRL(s.auth(s.handleIngestCSV)))
	s.mux.Handle("GET /v1/entitlements",                                  apiRL(s.auth(s.handleGetEntitlements)))
	s.mux.Handle("POST /v1/billing/upgrade",                              apiRL(s.auth(s.handleUpgrade)))
	// Webhook and callback are public — Paystack signature / reference verify them.
	s.mux.HandleFunc("POST /v1/billing/webhook",                          s.handleBillingWebhook)
	
	// WhatsApp Webhooks
	s.mux.HandleFunc("GET /v1/whatsapp/webhook",                          s.handleVerifyWhatsAppWebhook)
	s.mux.HandleFunc("POST /v1/whatsapp/webhook",                         s.handleReceiveWhatsAppMessage)
	s.mux.HandleFunc("GET /v1/billing/callback",                          s.handleBillingCallback)
	s.mux.Handle("POST /v1/payment/remita/init",                          apiRL(s.auth(s.handleRemitaInit)))
	s.mux.Handle("POST /v1/payment/remita/verify",                        apiRL(s.auth(s.handleRemitaVerify)))
	s.mux.Handle("GET /v1/support/returns/{id}",                          apiRL(s.auth(s.handleSupportView)))

	// Static SPA — serve React app for all non-/v1/ paths. Opt-in via WEB_DIST
	// environment variable so the server stays API-only in tests and CI.
	if dist := os.Getenv("WEB_DIST"); dist != "" {
		s.mux.Handle("/", spaHandler(dist))
	}
}

// spaHandler serves static files from dir. For paths that don't match a file
// on disk it falls back to index.html so React Router handles client-side routes.
func spaHandler(dir string) http.Handler {
	fs := http.Dir(dir)
	fileServer := http.FileServer(fs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := fs.Open(filepath.Clean(r.URL.Path))
		if err != nil {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------------------
// Auth middleware
// ---------------------------------------------------------------------------

type ctxKey string

const ctxAccountID ctxKey = "account_id"

// auth wraps a handler function with Bearer-token session authentication.
func (s *Server) auth(fn http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing Authorization: Bearer <token> header")
			return
		}
		accountID, ok := s.sessions.AccountID(token)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid or expired session token")
			return
		}
		ctx := context.WithValue(r.Context(), ctxAccountID, accountID)
		fn(w, r.WithContext(ctx))
	})
}

func accountIDFromCtx(r *http.Request) string {
	id, _ := r.Context().Value(ctxAccountID).(string)
	return id
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}

// ---------------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// ---------------------------------------------------------------------------
// ID and OTP generation
// ---------------------------------------------------------------------------

func newID() string {
	b := make([]byte, 8)
	if _, err := cryptorand.Read(b); err != nil {
		panic("transport: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// generateOTP returns a random 6-digit numeric string. The plaintext is only
// ever returned to the test response body — in production it would be sent
// out-of-band via SMS or WhatsApp and never appear in any API response.
func generateOTP() string {
	var b [4]byte
	cryptorand.Read(b[:]) //nolint:errcheck — entropy failure panics in newID; tolerate here
	n := binary.BigEndian.Uint32(b[:]) % 1_000_000
	return fmt.Sprintf("%06d", n)
}

// appendRaw appends one event with the given seq directly, without replaying
// the projection afterwards. Use when appending multiple events in a loop
// where rebuilding the projection between each write is unnecessary.
func appendRaw(s *Server, accountID, returnID string, seq int, kind returns.EventKind, payload any, actor returns.Actor) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.events.Append(accountID, returns.Event{
		Seq:        seq,
		ReturnID:   returnID,
		Kind:       kind,
		Payload:    payloadBytes,
		Actor:      actor,
		OccurredAt: s.clock(),
	})
}
