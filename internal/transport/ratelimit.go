package transport

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Rate limit tiers. Calibrated for a Nigerian tax product with a hard March 31
// filing deadline that concentrates 70% of annual traffic into two weeks.
//
// Auth tier (unauthenticated endpoints):
//   5 requests/minute, burst 5 — stops credential stuffing and account
//   enumeration while allowing a human who mistyped their password to retry.
//
// API tier (authenticated endpoints):
//   120 requests/minute, burst 30 — a complete return filing takes ~50 calls.
//   This allows a user to file and re-assess several times in a session without
//   hitting limits, while still throttling runaway clients.
//
// Keys:
//   Auth endpoints → client IP (unauthenticated, no session to key on).
//   API endpoints  → Bearer token (avoids punishing thousands of users who
//                    share a single corporate or mobile-carrier NAT IP).
const (
	authRate  = rate.Limit(5.0 / 60.0)
	authBurst = 5

	apiRate  = rate.Limit(2.0) // 2 req/s = 120 req/min
	apiBurst = 30
)

// visitor pairs a token-bucket limiter with a last-seen timestamp so idle
// entries can be evicted without a fixed-size LRU structure.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// limiterStore is a concurrency-safe map of key → token-bucket limiter with
// TTL-based eviction. Memory is bounded: at most one entry per active client,
// evicted after 10 minutes of inactivity.
type limiterStore struct {
	mu    sync.Mutex
	m     map[string]*visitor
	r     rate.Limit
	burst int
}

func newLimiterStore(r rate.Limit, burst int) *limiterStore {
	s := &limiterStore{
		m:     make(map[string]*visitor),
		r:     r,
		burst: burst,
	}
	go s.evict()
	return s
}

func (s *limiterStore) get(key string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[key]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(s.r, s.burst)}
		s.m[key] = v
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// evict removes entries idle for more than 10 minutes. Runs every 5 minutes.
// This is intentionally simple: no LRU heap, no size cap. At 100 bytes/entry
// and 50,000 concurrent sessions the map is ~5 MB — negligible.
func (s *limiterStore) evict() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for range t.C {
		cutoff := time.Now().Add(-10 * time.Minute)
		s.mu.Lock()
		for k, v := range s.m {
			if v.lastSeen.Before(cutoff) {
				delete(s.m, k)
			}
		}
		s.mu.Unlock()
	}
}

// rateLimit returns middleware that enforces the given store's limits, keying
// each request by keyFn. Exceeded requests receive 429 with Retry-After.
func rateLimit(store *limiterStore, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limiter := store.get(keyFn(r))
			res := limiter.Reserve()
			if !res.OK() {
				// Burst size exceeded even with infinite time — shouldn't happen
				// with a well-configured limiter, but handle defensively.
				res.Cancel()
				tooManyRequests(w, 60)
				return
			}
			delay := res.Delay()
			if delay > 0 {
				// Token not available yet. Cancel the reservation so the token
				// is returned to the bucket — we're not going to wait.
				res.Cancel()
				tooManyRequests(w, int(math.Ceil(delay.Seconds())))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func tooManyRequests(w http.ResponseWriter, retryAfterSecs int) {
	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSecs))
	writeError(w, http.StatusTooManyRequests,
		"rate limit exceeded — please wait "+strconv.Itoa(retryAfterSecs)+" second(s) and retry")
}

// clientIP extracts the real client IP. Behind Railway's reverse proxy the
// true client IP is the first (leftmost) value in X-Forwarded-For because
// Railway prepends it. Falls back to X-Real-IP, then RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the leftmost entry — the client's IP before any proxies.
		if i := strings.IndexByte(xff, ','); i != -1 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// tokenKey returns the Bearer token for per-session API rate limiting.
// Falls back to IP so unauthenticated requests to authenticated endpoints
// are still bounded (unlikely in normal use, but protects the endpoint).
func tokenKey(r *http.Request) string {
	if tok := bearerToken(r); tok != "" {
		return "tok:" + tok
	}
	return "ip:" + clientIP(r)
}

// ipKey returns the client IP as a rate-limit key.
func ipKey(r *http.Request) string {
	return "ip:" + clientIP(r)
}
