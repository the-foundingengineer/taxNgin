package transport

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
)

type ctxKeyReqID struct{}

// withRequestID generates a unique request ID, stores it in the context, and
// echoes it in the X-Request-ID response header. If the inbound request
// already carries X-Request-ID (set by Railway's load balancer or a client),
// that value is preserved so end-to-end tracing works across services.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			b := make([]byte, 8)
			cryptorand.Read(b) //nolint:errcheck — entropy failure is catastrophic; let it panic
			id = hex.EncodeToString(b)
		}
		ctx := context.WithValue(r.Context(), ctxKeyReqID{}, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// withSentryHub clones the global Sentry hub for each request so that scope
// mutations (SetUser, SetTag) are isolated per goroutine and don't bleed
// across concurrent requests.
func withSentryHub(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetRequest(r)
		if id, ok := r.Context().Value(ctxKeyReqID{}).(string); ok {
			hub.Scope().SetTag("request_id", id)
		}
		ctx := sentry.SetHubOnContext(r.Context(), hub)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// withRecovery wraps a handler to:
//  1. Catch panics, report them to Sentry, and return 500 — the server stays up.
//  2. Capture non-panic 5xx responses as Sentry messages so silent bugs
//     (writeError called with 500 but no panic) are still observable.
//
// 4xx errors are intentionally NOT captured — they are client mistakes, not
// server bugs. Capturing them would create noise and inflate Sentry quotas.
func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		defer func() {
			hub := sentry.GetHubFromContext(r.Context())

			if p := recover(); p != nil {
				if hub != nil {
					hub.RecoverWithContext(r.Context(), p)
					hub.Flush(2 * time.Second)
				}
				// Only write the 500 if the handler hasn't started a response yet.
				if !sr.written {
					writeError(w, http.StatusInternalServerError, "internal server error")
				}
				return
			}

			// Non-panic server error — capture as a message so it's findable.
			if sr.status >= 500 && hub != nil {
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetTag("http.method", r.Method)
					scope.SetTag("http.path", r.URL.Path)
					scope.SetTag("http.status", strconv.Itoa(sr.status))
					if acctID := accountIDFromCtx(r); acctID != "" {
						scope.SetUser(sentry.User{ID: acctID})
					}
					hub.CaptureMessage(r.Method + " " + r.URL.Path +
						" returned " + strconv.Itoa(sr.status))
				})
			}
		}()

		next.ServeHTTP(sr, r)
	})
}

// statusRecorder wraps http.ResponseWriter to capture the status code written
// by a handler. Required for withRecovery's non-panic 5xx capture.
type statusRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

func (sr *statusRecorder) WriteHeader(code int) {
	if !sr.written {
		sr.status = code
		sr.written = true
	}
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	if !sr.written {
		sr.status = http.StatusOK
		sr.written = true
	}
	return sr.ResponseWriter.Write(b)
}
