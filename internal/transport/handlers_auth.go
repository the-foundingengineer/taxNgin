package transport

import (
	"bytes"
	"errors"
	"net/http"

	"taxsoftware/internal/auth"
	"taxsoftware/internal/employer"

	"golang.org/x/crypto/bcrypt"
)

// POST /v1/accounts — register a new taxpayer account.
//
// Request:  { "nin": "...", "phone": "...", "email": "...", "password": "..." }
// Response: { "account_id": "..." }
//
// NIN is HMACed immediately and never stored in plaintext. The plaintext is
// held in memory only for the duration of this request.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NIN       string `json:"nin"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		BirthDate string `json:"date_of_birth"` // YYYY-MM-DD
		Gender    string `json:"gender"`         // "Male" or "Female"
		Phone     string `json:"phone"`
		Email     string `json:"email"`
		Password  string `json:"password"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.NIN == "" || req.Phone == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "nin, phone, and password are required")
		return
	}

	ninHMAC := employer.ComputeHMAC(s.hmacKey, req.NIN)

	if _, err := s.accounts.FindByNINHMAC(ninHMAC); err == nil {
		writeError(w, http.StatusConflict, "an account already exists for this NIN")
		return
	}

	// All accounts begin as pending verification to defer KYC costs
	// until the user is ready to file a return or purchase a tier.
	status := auth.AccountPendingVerification

	credHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash credential")
		return
	}

	acct := auth.TaxpayerAccount{
		ID:             "acct-" + newID(),
		NINHMACSuffix:  ninHMAC,
		PhoneNumber:    req.Phone,
		EmailAddress:   req.Email,
		CredentialHash: credHash,
		Status:         status,
		CreatedAt:      s.clock(),
	}
	if err := s.accounts.Save(acct); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save account")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"account_id": acct.ID})
}

// POST /v1/sessions — authenticate and create a session.
//
// Request:  { "nin": "...", "password": "..." }
// Response: { "token": "..." }
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NIN      string `json:"nin"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	ninHMAC := employer.ComputeHMAC(s.hmacKey, req.NIN)
	acct, err := s.accounts.FindByNINHMAC(ninHMAC)
	if err != nil {
		// Don't reveal whether the NIN exists.
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword(acct.CredentialHash, []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if acct.Status != auth.AccountActive && acct.Status != auth.AccountPendingVerification {
		writeError(w, http.StatusForbidden, "account is suspended")
		return
	}

	now := s.clock()
	acct.LastLoginAt = &now
	s.accounts.Save(acct) //nolint:errcheck — best-effort timestamp update

	token := "sess-" + newID()
	if err := s.sessions.Create(token, acct.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// DELETE /v1/sessions — revoke the current session.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.sessions.Delete(bearerToken(r)) //nolint:errcheck — best-effort
	w.WriteHeader(http.StatusNoContent)
}

// POST /v1/recovery — initiate account recovery via OTP.
//
// Request:  { "nin": "...", "method": "OTP_SMS" }
// Response: { "session_id": "...", "otp": "..." }
//
// In production the OTP is delivered out-of-band (SMS/WhatsApp) and never
// returned in the response. It is included here only to make the recovery
// flow testable without a real SMS gateway.
func (s *Server) handleInitiateRecovery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NIN    string `json:"nin"`
		Method string `json:"method"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	ninHMAC := employer.ComputeHMAC(s.hmacKey, req.NIN)
	acct, err := s.accounts.FindByNINHMAC(ninHMAC)
	if err != nil {
		// Prevent account enumeration — always respond identically.
		writeJSON(w, http.StatusAccepted, map[string]string{
			"message": "if an account exists for this NIN, an OTP has been dispatched",
		})
		return
	}

	method := auth.RecoveryMethod(req.Method)
	if method == "" {
		method = auth.RecoveryOTPSMS
	}

	otp := generateOTP()
	otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.MinCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate OTP")
		return
	}

	session, err := auth.InitiateRecovery(acct, method, otpHash, s.clock())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	session.ID = "rec-" + newID()

	if err := s.recovery.Save(session); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store recovery session")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"session_id": session.ID,
		// WARNING: OTP returned in response body for testability only.
		// In production: delete this field and deliver via SMS/WhatsApp.
		"otp": otp,
	})
}

// POST /v1/recovery/{id}/otp — submit an OTP to advance a recovery session.
//
// Request:  { "otp": "123456" }
// Response: { "state": "VERIFIED" }
func (s *Server) handleSubmitOTP(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	var req struct {
		OTP string `json:"otp"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	session, err := s.recovery.Get(sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "recovery session not found")
		return
	}

	updated, err := auth.SubmitOTP(session, req.OTP, s.clock())
	s.recovery.Save(updated) //nolint:errcheck — persist attempt count even on failure
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"state": string(updated.State)})
}

// POST /v1/recovery/{id}/complete — set a new password after OTP verification.
//
// Request:  { "new_password": "..." }
// Response: { "state": "COMPLETED" }
func (s *Server) handleCompleteRecovery(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "new_password is required")
		return
	}

	session, err := s.recovery.Get(sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "recovery session not found")
		return
	}

	newCredHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash new credential")
		return
	}

	updated, err := auth.CompleteRecovery(session, newCredHash, s.clock())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	acct, err := s.accounts.GetByID(updated.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "account not found")
		return
	}
	acct.CredentialHash = newCredHash
	s.accounts.Save(acct)    //nolint:errcheck
	s.recovery.Save(updated) //nolint:errcheck

	writeJSON(w, http.StatusOK, map[string]string{"state": string(updated.State)})
}

// POST /v1/accounts/verify — perform KYC evaluation on a deferred basis.
//
// Request: { "first_name": "...", "last_name": "...", "date_of_birth": "...", "gender": "..." }
//
// Evaluates the logged-in user's stored NINHMAC against the Dojah API. If successful, upgrades state to ACTIVE.
func (s *Server) handleVerifyNIN(w http.ResponseWriter, r *http.Request) {
	// Not all test suites provide a verifier; if missing, fail soft
	if s.ninVerifier == nil {
		writeError(w, http.StatusServiceUnavailable, "KYC verification service is not configured")
		return
	}

	acctID := accountIDFromCtx(r)
	acct, err := s.accounts.GetByID(acctID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "account not found")
		return
	}

	if acct.Status == auth.AccountActive {
		writeJSON(w, http.StatusOK, map[string]string{"status": string(acct.Status)})
		return
	}

	var req struct {
		NIN       string `json:"nin"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		BirthDate string `json:"date_of_birth"`
		Gender    string `json:"gender"`
		Phone     string `json:"phone"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.NIN == "" {
		writeError(w, http.StatusBadRequest, "nin is required")
		return
	}
	
	// Double check that the provided NIN hashes to the same suffix stored for this account
	ninHMAC := employer.ComputeHMAC(s.hmacKey, req.NIN)
	if !bytes.Equal(ninHMAC, acct.NINHMACSuffix) {
		writeError(w, http.StatusForbidden, "provided NIN does not match the registered account")
		return
	}

	record, err := s.ninVerifier.Verify(auth.NINVerificationRequest{
		NIN:       req.NIN,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		BirthDate: req.BirthDate,
		Gender:    req.Gender,
		Phone:     req.Phone,
	})

	if errors.Is(err, auth.ErrVerificationUnavailable) {
		writeError(w, http.StatusServiceUnavailable, "verification service unavailable, try again later")
		return
	} else if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "NIN could not be verified: "+err.Error())
		return
	} else if !record.Verified {
		writeError(w, http.StatusUnprocessableEntity, "NIN is not registered")
		return
	}

	// Update account status to ACTIVE
	acct.Status = auth.AccountActive
	now := s.clock()
	acct.VerifiedAt = &now
	if err := s.accounts.Save(acct); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update account status")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": string(acct.Status)})
}
