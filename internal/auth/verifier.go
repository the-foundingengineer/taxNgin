package auth

import "errors"

// ErrVerificationUnavailable is returned by verifiers when the external
// service is not reachable. The caller should treat this as a soft failure
// and create the account as PENDING_VERIFICATION for later retry.
var ErrVerificationUnavailable = errors.New("auth: verification service unavailable")

// NINVerificationRequest carries the fields required for a cross-check
// verification. The provider confirms that the NIN belongs to a person
// whose name and date of birth match what the taxpayer has supplied.
type NINVerificationRequest struct {
	NIN       string
	FirstName string
	LastName  string
	BirthDate string // YYYY-MM-DD
	Gender    string // "Male" or "Female"
	Phone     string
}

// NINRecord is the minimal data returned by a successful NIN verification.
type NINRecord struct {
	NIN      string
	Verified bool
}

// NINVerifier cross-checks a NIN against the government identity database.
// Implement with Interswitch Data Service (64 NGN/call).
type NINVerifier interface {
	Verify(req NINVerificationRequest) (NINRecord, error)
}

// StubNINVerifier always returns Verified: true. Use in tests and staging.
// MUST NOT be used in production — it bypasses identity verification.
type StubNINVerifier struct{}

func (s StubNINVerifier) Verify(req NINVerificationRequest) (NINRecord, error) {
	return NINRecord{NIN: req.NIN, Verified: true}, nil
}
