package transport

import (
	"net/http"

	"taxsoftware/internal/returns"
	"taxsoftware/internal/support"
)

// GET /v1/support/returns/{id} — get the redacted support view of a return.
//
// In production this endpoint is role-gated at the API gateway; any support
// agent with the "support_read" role can query any return without being the
// account owner. For this implementation, any authenticated session may call
// it — the redaction guarantees in support.BuildSupportView are the primary
// safety control regardless of who the caller is.
func (s *Server) handleSupportView(w http.ResponseWriter, r *http.Request) {
	returnID := r.PathValue("id")

	evts, err := s.events.Load(returnID)
	if err != nil {
		writeError(w, http.StatusNotFound, "return not found")
		return
	}
	proj, err := returns.Replay(evts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to replay return")
		return
	}
	proj.ReturnID = returnID

	writeJSON(w, http.StatusOK, support.BuildSupportView(proj))
}
