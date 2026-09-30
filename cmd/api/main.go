// cmd/api is the HTTP server entry point for the tax product API.
//
// All business logic lives in the internal packages; this file wires them
// together and starts the listener. Configuration is environment-based —
// no config file is parsed here.
//
// Required environment variables:
//
//	HMAC_KEY    — 32+ random bytes, hex-encoded. Used to hash NIN/BVN at rest.
//	              Generate with: openssl rand -hex 32
//	RULES_DIR   — path to the directory containing rule-set YAML files (default: rules/)
//	PORTALS_DIR — path to the directory containing portal-map YAML files (default: rules/portals/)
//	ADDR        — listen address (default: :PORT or :8080)
//
// Optional:
//
//	DATABASE_URL         — PostgreSQL connection string. When set, persistent PostgreSQL
//	                       stores are used. When absent, in-memory stores are used (data
//	                       lost on restart — development only).
//	PAYSTACK_SECRET_KEY  — Paystack live/test secret key. When set, POST /v1/billing/upgrade
//	                       and webhook verification are active. When absent, upgrade
//	                       attempts return 503 Service Unavailable.
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"

	"taxsoftware/internal/billing"
	"taxsoftware/internal/kyc"
	"taxsoftware/internal/pgstore"
	"taxsoftware/internal/portalmap"
	"taxsoftware/internal/remita"
	"taxsoftware/internal/rulesregistry"
	"taxsoftware/internal/rulesyaml"
	"taxsoftware/internal/transport"
	"taxsoftware/taxengine"
)

func main() {
	initSentry()
	defer sentry.Flush(2 * time.Second)

	hmacKey := mustHMACKey()
	registry := mustRegistry()
	pm := mustPortalMap()
	addr := env("ADDR", ":"+env("PORT", "8080"))

	ninVerifier, err := kyc.NewDojah(
		os.Getenv("DOJAH_APP_ID"),
		os.Getenv("DOJAH_AUTHORIZATION_KEY"),
	)
	if err != nil {
		log.Fatalf("api: kyc (dojah): %v", err)
	}
	if ninVerifier != nil {
		log.Printf("api: NIN verification enabled via Dojah")
	} else {
		log.Printf("api: NIN verification disabled — accounts created as PENDING_VERIFICATION")
	}

	cfg := transport.Config{
		Registry:    registry,
		PortalMap:   pm,
		HMACKey:     hmacKey,
		NINVerifier: ninVerifier,
		Clock:       time.Now,
	}

	if paystackKey := os.Getenv("PAYSTACK_SECRET_KEY"); paystackKey != "" {
		cfg.PaymentGateway = billing.NewPaystackGateway(paystackKey)
		log.Printf("api: Paystack payment gateway enabled")
	} else {
		log.Printf("api: payment gateway disabled (PAYSTACK_SECRET_KEY not set)")
	}

	if mid := os.Getenv("REMITA_MERCHANT_ID"); mid != "" {
		cfg.RemitaClient = remita.New(remita.Config{
			MerchantID:    mid,
			ServiceTypeID: os.Getenv("REMITA_SERVICE_TYPE_ID"),
			APIKey:        os.Getenv("REMITA_API_KEY"),
			Env:           os.Getenv("REMITA_ENV"),
		})
		log.Printf("api: Remita gateway enabled (env=%s)", env("REMITA_ENV", "live"))
	} else {
		log.Printf("api: Remita gateway disabled (REMITA_MERCHANT_ID not set — pending KYC)")
	}

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		db, err := pgstore.Open(context.Background(), dbURL)
		if err != nil {
			log.Fatalf("api: postgres: %v", err)
		}
		defer db.Close()
		cfg.Accounts = db.Accounts()
		cfg.Events = db.Events()
		cfg.Sessions = db.Sessions()
		cfg.Recovery = db.Recovery()
		cfg.Billing = db.Billing()
		log.Printf("api: using PostgreSQL store")
	} else {
		cfg.Accounts = transport.NewMemAccountStore()
		cfg.Events = transport.NewMemEventStore()
		cfg.Sessions = transport.NewMemSessionStore()
		cfg.Recovery = transport.NewMemRecoveryStore()
		cfg.Billing = transport.NewMemBillingStore()
		log.Printf("api: using in-memory store (data will be lost on restart)")
	}

	srv := transport.New(cfg)
	log.Printf("api: listening on %s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("api: server error: %v", err)
	}
}

func mustHMACKey() []byte {
	raw := os.Getenv("HMAC_KEY")
	if raw == "" {
		log.Fatal("HMAC_KEY environment variable is required (generate: openssl rand -hex 32)")
	}
	key, err := hex.DecodeString(raw)
	if err != nil {
		log.Fatalf("HMAC_KEY is not valid hex: %v", err)
	}
	if len(key) < 32 {
		log.Fatalf("HMAC_KEY must be at least 32 bytes (64 hex chars); got %d bytes", len(key))
	}
	return key
}

func mustRegistry() *rulesregistry.Registry {
	dir := env("RULES_DIR", "rules")
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("cannot read rules dir %q: %v", dir, err)
	}
	var rawFiles [][]byte
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			log.Fatalf("read rule set %s: %v", e.Name(), err)
		}
		rawFiles = append(rawFiles, raw)
	}
	if len(rawFiles) == 0 {
		log.Fatalf("no rule-set YAML files found in %q", dir)
	}
	allowDraft := os.Getenv("ALLOW_DRAFT") == "true"
	reg, err := rulesregistry.New(rawFiles, rulesyaml.Unmarshal, taxengine.LoadOptions{AllowDraft: allowDraft})
	if err != nil {
		log.Fatalf("build registry: %v", err)
	}
	log.Printf("api: loaded %d rule set(s) from %s", len(rawFiles), dir)
	return reg
}

func mustPortalMap() portalmap.PortalMap {
	dir := env("PORTALS_DIR", filepath.Join("rules", "portals"))
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("cannot read portals dir %q: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			log.Fatalf("read portal map %s: %v", e.Name(), err)
		}
		pm, err := portalmap.Load(raw, rulesyaml.Unmarshal, portalmap.LoadOptions{AllowDraft: os.Getenv("ALLOW_DRAFT") == "true"})
		if err != nil {
			log.Fatalf("load portal map %s: %v", e.Name(), err)
		}
		log.Printf("api: loaded portal map %s (version %s)", e.Name(), pm.Portal.MapVersion)
		return pm
	}
	log.Fatalf("no portal-map YAML files found in %q", dir)
	panic("unreachable")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var _ = fmt.Sprintf

func initSentry() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		log.Printf("api: error monitoring disabled (SENTRY_DSN not set)")
		return
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		Environment: env("RAILWAY_ENVIRONMENT", "development"),
		// Sample 10% of transactions for performance monitoring.
		// At 50,000 users on March 31 this keeps Sentry quotas manageable
		// while still giving representative latency data.
		TracesSampleRate: 0.1,
		// Capture the full request body on errors (NIN is already HMAC'd
		// before it reaches any handler, so no PII risk here).
		SendDefaultPII: false,
	})
	if err != nil {
		log.Printf("api: sentry init failed: %v", err)
		return
	}
	log.Printf("api: error monitoring enabled via Sentry (env=%s)", env("RAILWAY_ENVIRONMENT", "development"))
}
