package transport

import (
	"io"
	"net/http"

	"taxsoftware/internal/ingestion"
	"taxsoftware/internal/returns"
)

// POST /v1/returns/{id}/ingest/csv — parse a CSV payroll file and create
// payroll proposals on the return.
//
// The request body must be a raw CSV file in the LIRS H1 format. Each row
// becomes one EvPayrollProposed event. The taxpayer must then accept or reject
// each proposal via the payroll acceptance flow.
//
// Content-Type: text/csv (or omit — the body is read as raw bytes regardless).
func (s *Server) handleIngestCSV(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")

	if s.loadOwned(w, returnID, accountID) == nil {
		return
	}

	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	if len(data) == 0 {
		writeError(w, http.StatusBadRequest, "request body is empty")
		return
	}

	ing := &ingestion.CSVIngester{Columns: ingestion.DefaultH1Mapping()}
	records, err := ing.Extract(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, "CSV parse error: "+err.Error())
		return
	}

	records = ingestion.AggregateBySource(records)

	evts, err := s.events.Load(returnID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load return")
		return
	}
	nextSeq := len(evts) + 1

	for _, rec := range records {
		proposal := returns.PayrollProposal{
			RecordID:   "prop-" + newID(),
			EmployerID: rec.SourceRef,
			GrossKobo:  int64(rec.GrossKobo),
			PayeKobo:   int64(rec.TaxDeductedKobo),
		}
		if err := appendRaw(s, accountID, returnID, nextSeq, returns.EvPayrollProposed, proposal,
			returns.Actor{Kind: returns.ActorSystem, ID: "csv-ingester"}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save payroll proposal")
			return
		}
		nextSeq++
	}

	writeJSON(w, http.StatusCreated, map[string]int{"proposals_created": len(records)})
}
