package transport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"taxsoftware/internal/bridge"
	"taxsoftware/internal/coordinator"
	"taxsoftware/internal/pack"
	"taxsoftware/internal/returns"
	"taxsoftware/internal/validate"
	"taxsoftware/taxengine"
)

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// loadOwned loads and replays the event log for a return, verifying that
// accountID is the owner. On failure it writes the error response and returns nil.
func (s *Server) loadOwned(w http.ResponseWriter, returnID, accountID string) *returns.Projection {
	owner, ok := s.events.OwnerOf(returnID)
	if !ok {
		writeError(w, http.StatusNotFound, "return not found")
		return nil
	}
	if owner != accountID {
		writeError(w, http.StatusForbidden, "this return belongs to another account")
		return nil
	}
	evts, err := s.events.Load(returnID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load return")
		return nil
	}
	p, err := returns.Replay(evts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to replay return: "+err.Error())
		return nil
	}
	p.ReturnID = returnID
	return p
}

// nextSeq returns the sequence number for the next event on a return.
func (s *Server) nextSeq(returnID string) (int, error) {
	evts, err := s.events.Load(returnID)
	if err != nil {
		return 0, err
	}
	return len(evts) + 1, nil
}

// emit appends one event to the store and returns the updated projection.
func (s *Server) emit(w http.ResponseWriter, accountID, returnID string, kind returns.EventKind, payload any, actor returns.Actor) *returns.Projection {
	seq, err := s.nextSeq(returnID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to determine sequence")
		return nil
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to marshal event payload")
		return nil
	}
	e := returns.Event{
		Seq:        seq,
		ReturnID:   returnID,
		Kind:       kind,
		Payload:    payloadBytes,
		Actor:      actor,
		OccurredAt: s.clock(),
	}
	if err := s.events.Append(accountID, e); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to append event")
		return nil
	}
	// Reload and replay to get the authoritative projection after the new event.
	evts, err := s.events.Load(returnID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reload return")
		return nil
	}
	p, err := returns.Replay(evts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to replay after emit")
		return nil
	}
	p.ReturnID = returnID
	return p
}

// ---------------------------------------------------------------------------
// Open / read
// ---------------------------------------------------------------------------

// GET /v1/returns — list all return IDs for the authenticated account.
//
// Response: { "return_ids": ["ret-...", ...] }
func (s *Server) handleListReturns(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	ids := s.events.ListForAccount(accountID)
	if ids == nil {
		ids = []string{}
	}
	writeJSON(w, http.StatusOK, map[string][]string{"return_ids": ids})
}

// POST /v1/returns — open a new tax return.
//
// Request:  { "year_of_assessment": 2026, "jurisdiction": "NG" }
// Response: { "return_id": "..." }
func (s *Server) handleOpenReturn(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	var req struct {
		YearOfAssessment int    `json:"year_of_assessment"`
		Jurisdiction     string `json:"jurisdiction"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.YearOfAssessment == 0 || req.Jurisdiction == "" {
		writeError(w, http.StatusBadRequest, "year_of_assessment and jurisdiction are required")
		return
	}

	returnID := "ret-" + newID()
	payload := map[string]any{
		"taxpayer_id":        accountID,
		"year_of_assessment": req.YearOfAssessment,
		"basis_period_start": time.Date(req.YearOfAssessment, 1, 1, 0, 0, 0, 0, time.UTC),
		"basis_period_end":   time.Date(req.YearOfAssessment, 12, 31, 0, 0, 0, 0, time.UTC),
		"jurisdiction":       req.Jurisdiction,
	}
	payloadBytes, _ := json.Marshal(payload)
	e := returns.Event{
		Seq:        1,
		ReturnID:   returnID,
		Kind:       returns.EvReturnOpened,
		Payload:    payloadBytes,
		Actor:      returns.Actor{Kind: returns.ActorTaxpayer, ID: accountID},
		OccurredAt: s.clock(),
	}
	if err := s.events.Append(accountID, e); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to open return")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"return_id": returnID})
}

// GET /v1/returns/{id} — get the current return projection.
func (s *Server) handleGetReturn(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	p := s.loadOwned(w, r.PathValue("id"), accountID)
	if p == nil {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ---------------------------------------------------------------------------
// Income
// ---------------------------------------------------------------------------

// POST /v1/returns/{id}/income — declare an income line.
//
// Request:  { "id": "...", "category": "EMPLOYMENT", "amount_kobo": 600000000, "source_ref": "Acme Ltd", "label": "Annual salary" }
// Response: projection
func (s *Server) handleDeclareIncome(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")

	if s.loadOwned(w, returnID, accountID) == nil {
		return // ownership/existence check; error already written
	}

	var req struct {
		ID        string `json:"id"`
		Category  string `json:"category"`
		AmountKobo int64  `json:"amount_kobo"`
		SourceRef string `json:"source_ref"`
		Label     string `json:"label"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Category == "" || req.AmountKobo <= 0 {
		writeError(w, http.StatusBadRequest, "category and amount_kobo (> 0) are required")
		return
	}
	if req.ID == "" {
		req.ID = "inc-" + newID()
	}

	line := returns.IncomeLine{
		ID:         req.ID,
		Category:   req.Category,
		AmountKobo: req.AmountKobo,
		SourceRef:  req.SourceRef,
		Label:      req.Label,
	}
	p := s.emit(w, accountID, returnID, returns.EvIncomeDeclared, line,
		returns.Actor{Kind: returns.ActorTaxpayer, ID: accountID})
	if p == nil {
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// DELETE /v1/returns/{id}/income/{income_id} — remove an income line.
func (s *Server) handleRemoveIncome(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")
	incomeID := r.PathValue("income_id")

	if s.loadOwned(w, returnID, accountID) == nil {
		return
	}

	p := s.emit(w, accountID, returnID, returns.EvIncomeRemoved,
		map[string]string{"id": incomeID},
		returns.Actor{Kind: returns.ActorTaxpayer, ID: accountID})
	if p == nil {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ---------------------------------------------------------------------------
// Reliefs
// ---------------------------------------------------------------------------

// POST /v1/returns/{id}/reliefs — claim a relief.
//
// Request:  { "code": "PENSION", "amount_kobo": 48000000, "has_evidence": false }
// Response: projection
func (s *Server) handleClaimRelief(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")

	if s.loadOwned(w, returnID, accountID) == nil {
		return
	}

	var req struct {
		Code        string `json:"code"`
		AmountKobo  int64  `json:"amount_kobo"`
		HasEvidence bool   `json:"has_evidence"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	line := returns.ReliefLine{
		Code:       req.Code,
		AmountKobo: req.AmountKobo,
	}
	p := s.emit(w, accountID, returnID, returns.EvReliefClaimed, line,
		returns.Actor{Kind: returns.ActorTaxpayer, ID: accountID})
	if p == nil {
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// DELETE /v1/returns/{id}/reliefs/{code} — remove a relief.
func (s *Server) handleRemoveRelief(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")
	code := r.PathValue("code")

	if s.loadOwned(w, returnID, accountID) == nil {
		return
	}

	p := s.emit(w, accountID, returnID, returns.EvReliefRemoved,
		map[string]string{"code": code},
		returns.Actor{Kind: returns.ActorTaxpayer, ID: accountID})
	if p == nil {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ---------------------------------------------------------------------------
// Assessment (tax computation)
// ---------------------------------------------------------------------------

// POST /v1/returns/{id}/assess — run the tax engine and record the result.
//
// Response: projection with Assessment populated.
func (s *Server) handleAssess(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")

	proj := s.loadOwned(w, returnID, accountID)
	if proj == nil {
		return
	}

	rs, err := s.registry.Get(proj.Jurisdiction, proj.YearOfAssessment)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity,
			fmt.Sprintf("no rule set for (%s, %d): %v", proj.Jurisdiction, proj.YearOfAssessment, err))
		return
	}

	facts := bridge.ProjectionToFacts(proj)
	assessment, trace, err := taxengine.Compute(facts, rs)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "engine error: "+err.Error())
		return
	}

	traceBytes, _ := json.Marshal(trace)
	snap := returns.AssessmentSnapshot{
		FactsHash:            assessment.FactsHash,
		RuleSetVersion:       assessment.RuleSetVersion,
		RuleSetDigest:        assessment.RuleSetDigest,
		RuleSetPublished:     rs.Status == taxengine.StatusPublished,
		ChargeableIncomeKobo: int64(assessment.ChargeableIncomeKobo),
		TaxPayableKobo:       int64(assessment.TaxPayableKobo),
		TotalCreditsKobo:     int64(assessment.TotalCreditsKobo),
		BalanceKobo:          int64(assessment.BalanceKobo),
		ComputedAt:           s.clock(),
		Trace:                traceBytes,
	}

	p := s.emit(w, accountID, returnID, returns.EvAssessmentComputed, snap,
		returns.Actor{Kind: returns.ActorSystem, ID: "engine"})
	if p == nil {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// POST /v1/returns/{id}/validate — run the validation catalogue.
//
// Request:
//
//	{
//	  "nin": "...",
//	  "tax_id": "...",
//	  "tax_id_verified": false,
//	  "nil_declaration": false,
//	  "duplicate_check_ok": true
//	}
//
// NIN is used only to satisfy E002; it is not stored by this handler.
// Response: projection (with blocking_errors and warnings updated).
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")

	proj := s.loadOwned(w, returnID, accountID)
	if proj == nil {
		return
	}

	var req struct {
		NIN             string `json:"nin"`
		TaxID           string `json:"tax_id"`
		TaxIDVerified   bool   `json:"tax_id_verified"`
		NilDeclaration  bool   `json:"nil_declaration"`
		DuplicateCheckOK bool  `json:"duplicate_check_ok"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	rs, err := s.registry.Get(proj.Jurisdiction, proj.YearOfAssessment)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "no rule set: "+err.Error())
		return
	}

	base := validate.FilingContext{
		Facts:            bridge.ProjectionToFacts(proj),
		NIN:              req.NIN,
		TaxID:            req.TaxID,
		TaxIDVerified:    req.TaxIDVerified,
		NilDeclaration:   req.NilDeclaration,
		Signed:           stateAtLeastSigned(proj.State),
		DeclaredByTaxpayer: proj.DeclaredByTaxpayer,
		AccommodationType: proj.AccommodationType,
		RelevantAuthority: jurisdictionAuthority(proj.Jurisdiction),
		DuplicateKnown:   true,
		DuplicateCheckOK: req.DuplicateCheckOK,
	}
	ctx := coordinator.BuildFilingContext(proj, nil, base)

	result, err := validate.Validate(ctx, rs)
	if err != nil && !errors.Is(err, validate.ErrNotEnoughContext) {
		writeError(w, http.StatusUnprocessableEntity, "validation error: "+err.Error())
		return
	}

	var blocking, warnings []string
	for _, issue := range result.Issues {
		if issue.Severity == validate.SeverityBlock {
			blocking = append(blocking, issue.Code)
		} else {
			warnings = append(warnings, issue.Code)
		}
	}
	if blocking == nil {
		blocking = []string{}
	}
	if warnings == nil {
		warnings = []string{}
	}

	payload := map[string][]string{"blocking": blocking, "warnings": warnings}
	p := s.emit(w, accountID, returnID, returns.EvValidationRun, payload,
		returns.Actor{Kind: returns.ActorSystem, ID: "validator"})
	if p == nil {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// stateAtLeastSigned returns true for any lifecycle state at or beyond SIGNED.
func stateAtLeastSigned(state returns.State) bool {
	switch state {
	case returns.StateSigned, returns.StatePackIssued, returns.StateSelfAttested,
		returns.StateAckUploaded, returns.StateAssessed, returns.StateObjected,
		returns.StatePaid, returns.StateCleared:
		return true
	}
	return false
}

// jurisdictionAuthority maps a jurisdiction code to the relevant tax authority.
func jurisdictionAuthority(jurisdiction string) string {
	switch jurisdiction {
	case "NG", "LAGOS":
		return "LIRS"
	case "FCT", "ABUJA":
		return "FCT-IRS"
	default:
		if jurisdiction != "" {
			return jurisdiction // passthrough for other states
		}
		return ""
	}
}

// ---------------------------------------------------------------------------
// Sign
// ---------------------------------------------------------------------------

// POST /v1/returns/{id}/sign — sign the return declaration.
//
// The return must be VALIDATED (no blocking issues). Signing is the taxpayer's
// attestation that all declared information is accurate.
func (s *Server) handleSign(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")

	proj := s.loadOwned(w, returnID, accountID)
	if proj == nil {
		return
	}
	if proj.State == returns.StateDraft {
		writeError(w, http.StatusBadRequest, "return must be validated before signing — run /validate first")
		return
	}
	if !proj.DeclaredByTaxpayer {
		writeError(w, http.StatusBadRequest, "all income must be declared by the taxpayer before signing")
		return
	}

	p := s.emit(w, accountID, returnID, returns.EvReturnSigned, struct{}{},
		returns.Actor{Kind: returns.ActorTaxpayer, ID: accountID})
	if p == nil {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ---------------------------------------------------------------------------
// Pack
// ---------------------------------------------------------------------------

// POST /v1/returns/{id}/pack — generate and issue the submission pack.
//
// The return must be at least SIGNED. Issues EvPackIssued and returns the
// pack as JSON — each row is a portal data-entry instruction.
func (s *Server) handleIssuePack(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")

	proj := s.loadOwned(w, returnID, accountID)
	if proj == nil {
		return
	}

	generated, err := pack.Generate(pack.GenerateInput{
		Projection:       proj,
		PortalMap:        s.pm,
		PreconditionsMet: map[string]bool{},
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot generate pack: "+err.Error())
		return
	}

	payload := map[string]string{"portal_map_version": s.pm.Portal.MapVersion}
	p := s.emit(w, accountID, returnID, returns.EvPackIssued, payload,
		returns.Actor{Kind: returns.ActorSystem, ID: "pack-generator"})
	if p == nil {
		return
	}

	writeJSON(w, http.StatusCreated, generated)
}

// GET /v1/returns/{id}/pack — download the submission pack.
//
// Regenerates the pack from the current projection on every call (pack
// generation is <5ms and the portal map is a singleton).
func (s *Server) handleGetPack(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")

	proj := s.loadOwned(w, returnID, accountID)
	if proj == nil {
		return
	}
	if proj.PackIssuedAt == nil {
		writeError(w, http.StatusBadRequest, "pack has not been issued yet — POST /pack first")
		return
	}

	generated, err := pack.Generate(pack.GenerateInput{
		Projection:       proj,
		PortalMap:        s.pm,
		PreconditionsMet: map[string]bool{},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "pack generation failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, generated)
}

// ---------------------------------------------------------------------------
// Post-filing events
// ---------------------------------------------------------------------------

// POST /v1/returns/{id}/attest — record that the taxpayer attested to filing.
func (s *Server) handleAttest(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")

	proj := s.loadOwned(w, returnID, accountID)
	if proj == nil {
		return
	}
	if proj.PackIssuedAt == nil {
		writeError(w, http.StatusBadRequest, "a submission pack must be issued before attesting")
		return
	}

	p := s.emit(w, accountID, returnID, returns.EvFilingAttested, struct{}{},
		returns.Actor{Kind: returns.ActorTaxpayer, ID: accountID})
	if p == nil {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// POST /v1/returns/{id}/ack — record the portal acknowledgement reference.
//
// Request:  { "reference": "LIRS-2026-ABC123" }
func (s *Server) handleUploadAck(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")

	proj := s.loadOwned(w, returnID, accountID)
	if proj == nil {
		return
	}
	if proj.AttestedAt == nil {
		writeError(w, http.StatusBadRequest, "filing must be attested before uploading the acknowledgement")
		return
	}

	var req struct {
		Reference string `json:"reference"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	p := s.emit(w, accountID, returnID, returns.EvAcknowledgementUploaded,
		map[string]string{"reference": req.Reference},
		returns.Actor{Kind: returns.ActorTaxpayer, ID: accountID})
	if p == nil {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ---------------------------------------------------------------------------
// Accommodation
// ---------------------------------------------------------------------------

// POST /v1/returns/{id}/accommodation — declare the taxpayer's accommodation.
//
// Required for portal pack Page 2 (LIRS eTax accommodation section).
// Does NOT mark the assessment stale — accommodation is a portal compliance
// requirement, not a change to taxable facts.
//
// Request: { "type": "Rented", "ownership_type": "Tenant",
//            "landlord_name": "...", "landlord_address": "..." }
func (s *Server) handleDeclareAccommodation(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")

	if s.loadOwned(w, returnID, accountID) == nil {
		return
	}

	var req struct {
		Type            string `json:"type"`
		OwnershipType   string `json:"ownership_type"`
		LandlordName    string `json:"landlord_name"`
		LandlordAddress string `json:"landlord_address"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}

	payload := map[string]string{
		"type":             req.Type,
		"ownership_type":   req.OwnershipType,
		"landlord_name":    req.LandlordName,
		"landlord_address": req.LandlordAddress,
	}
	p := s.emit(w, accountID, returnID, returns.EvAccommodationDeclared, payload,
		returns.Actor{Kind: returns.ActorTaxpayer, ID: accountID})
	if p == nil {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ---------------------------------------------------------------------------
// Payroll proposal accept / reject
// ---------------------------------------------------------------------------

// POST /v1/returns/{id}/payroll/{record_id}/accept — accept an employer
// payroll proposal and convert it to a TAXPAYER-actor income declaration.
//
// Request: { "income_id": "...", "category": "EMPLOYMENT", "label": "..." }
func (s *Server) handleAcceptPayroll(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")
	recordID := r.PathValue("record_id")

	proj := s.loadOwned(w, returnID, accountID)
	if proj == nil {
		return
	}

	proposal, ok := proj.ProposedPayroll[recordID]
	if !ok {
		writeError(w, http.StatusNotFound, "payroll proposal not found")
		return
	}
	if proposal.Status != "PROPOSED" {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("proposal is already %s", proposal.Status))
		return
	}

	var req struct {
		IncomeID string `json:"income_id"`
		Category string `json:"category"`
		Label    string `json:"label"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.IncomeID == "" {
		req.IncomeID = "inc-" + newID()
	}
	if req.Category == "" {
		req.Category = "EMPLOYMENT"
	}

	payload := map[string]any{
		"record_id": recordID,
		"income": returns.IncomeLine{
			ID:         req.IncomeID,
			Category:   req.Category,
			AmountKobo: proposal.GrossKobo,
			SourceRef:  proposal.EmployerID,
			Label:      req.Label,
		},
	}
	p := s.emit(w, accountID, returnID, returns.EvPayrollAccepted, payload,
		returns.Actor{Kind: returns.ActorTaxpayer, ID: accountID})
	if p == nil {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// POST /v1/returns/{id}/payroll/{record_id}/reject — reject an employer
// payroll proposal. The proposal remains visible but cannot be re-accepted.
func (s *Server) handleRejectPayroll(w http.ResponseWriter, r *http.Request) {
	accountID := accountIDFromCtx(r)
	returnID := r.PathValue("id")
	recordID := r.PathValue("record_id")

	proj := s.loadOwned(w, returnID, accountID)
	if proj == nil {
		return
	}
	if _, ok := proj.ProposedPayroll[recordID]; !ok {
		writeError(w, http.StatusNotFound, "payroll proposal not found")
		return
	}

	p := s.emit(w, accountID, returnID, returns.EvPayrollRejected,
		map[string]string{"record_id": recordID},
		returns.Actor{Kind: returns.ActorTaxpayer, ID: accountID})
	if p == nil {
		return
	}
	writeJSON(w, http.StatusOK, p)
}
