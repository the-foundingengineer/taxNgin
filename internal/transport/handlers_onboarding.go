package transport

import (
	"net/http"
	"taxsoftware/internal/onboarding"
)

func (s *Server) handleGetOnboardingSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, onboarding.GetSchema())
}
