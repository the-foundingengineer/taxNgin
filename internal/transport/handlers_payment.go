package transport

import (
	"net/http"

	"taxsoftware/internal/remita"
)

// POST /v1/payment/remita/init — generate a Remita RRR for a LIRS tax payment.
//
// Request:  { "amount": <naira float>, "order_id": "<lirs efile ref>" }
// Response: { "rrr": "280007648597", "amount": 60000.00 }
//
// The RRR is valid for payment at any Nigerian bank for 3 banking days.
// amount is in Nigerian Naira — the frontend sends PIT estimates in Naira,
// not kobo, so no conversion is done here.
func (s *Server) handleRemitaInit(w http.ResponseWriter, r *http.Request) {
	if s.remitaClient == nil {
		writeError(w, http.StatusServiceUnavailable, "Remita gateway not configured — credentials pending KYC approval")
		return
	}

	accountID := accountIDFromCtx(r)

	var req struct {
		Amount  float64 `json:"amount"`
		OrderID string  `json:"order_id"`
	}
	if err := readJSON(r, &req); err != nil || req.Amount <= 0 || req.OrderID == "" {
		writeError(w, http.StatusBadRequest, "amount (naira) and order_id are required")
		return
	}

	account, err := s.accounts.GetByID(accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load account")
		return
	}

	email := account.EmailAddress
	if email == "" {
		email = accountID + "@noreply.taxsoftware.ng"
	}

	rrr, err := s.remitaClient.GenerateRRR(remita.GenerateRRRRequest{
		AmountNaira: req.Amount,
		OrderID:     req.OrderID,
		PayerName:   "Taxpayer",
		PayerEmail:  email,
		PayerPhone:  account.PhoneNumber,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate RRR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"rrr":    rrr,
		"amount": req.Amount,
	})
}

// POST /v1/payment/remita/verify — check whether an RRR has been settled.
//
// Request:  { "rrr": "280007648597" }
// Response: { "paid": true }
func (s *Server) handleRemitaVerify(w http.ResponseWriter, r *http.Request) {
	if s.remitaClient == nil {
		writeError(w, http.StatusServiceUnavailable, "Remita gateway not configured — credentials pending KYC approval")
		return
	}

	var req struct {
		RRR string `json:"rrr"`
	}
	if err := readJSON(r, &req); err != nil || req.RRR == "" {
		writeError(w, http.StatusBadRequest, "rrr is required")
		return
	}

	paid, err := s.remitaClient.IsPaid(req.RRR)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify RRR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"paid": paid})
}
