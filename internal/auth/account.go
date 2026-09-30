// Package auth models NIN-anchored taxpayer identity. It is fundamentally
// different from a standard email/password flow:
//
//   - NIN (National Identification Number) is the primary anchor, not email.
//     Identity verification goes through the JRB API. Until that API is
//     available, verification is deferred, not skipped.
//   - Account recovery without email: OTP to the registered phone number is
//     the primary path; BVN match is the secondary path.
//   - NIN and BVN are never stored in plaintext. The same HMAC scheme used in
//     internal/employer/hmac.go is applied here: a keyed HMAC-SHA256 stored as
//     bytes. A leak of the accounts table yields metadata without usable identifiers.
//   - Sessions are revocable: if a NIN is flagged as compromised, all sessions
//     for that account must be invalidable without rebuilding the account.
package auth

import "time"

// AccountStatus is the lifecycle state of a taxpayer account.
type AccountStatus string

const (
	// AccountPendingVerification means the account exists but the NIN has not
	// yet been confirmed via the JRB API. The account can be used but the
	// practitioner sign-off gate and certain validation checks will flag it.
	AccountPendingVerification AccountStatus = "PENDING_VERIFICATION"
	AccountActive              AccountStatus = "ACTIVE"
	// AccountSuspended means the account has been locked by a staff action.
	// Recovery is not possible until an admin unsuspends it.
	AccountSuspended AccountStatus = "SUSPENDED"
)

// TaxpayerAccount is the identity record for one individual taxpayer.
//
// NIN and BVN are stored as HMAC-SHA256 bytes under a system-wide secret key
// (see internal/employer/hmac.go for the scheme). A database leak reveals
// account metadata without yielding usable national identifiers.
type TaxpayerAccount struct {
	ID  string
	// NINHMACSuffix is the HMAC of the taxpayer's NIN under the system key.
	// It is used to link this account to PayrollRecords (same scheme).
	NINHMACSuffix  []byte
	// BVNHMACSuffix is optional. Nil until the taxpayer registers their BVN.
	BVNHMACSuffix  []byte
	// PhoneNumber is the registered mobile number in E.164 format (e.g. "+234XXXXXXXXXX").
	// Primary OTP delivery channel.
	PhoneNumber    string
	// EmailAddress is optional and used only for secondary notifications and
	// legal receipts. Never used as the primary recovery channel.
	EmailAddress   string
	// CredentialHash is a bcrypt hash of the account password or PIN.
	CredentialHash []byte
	Status         AccountStatus
	CreatedAt      time.Time
	LastLoginAt    *time.Time
	// VerifiedAt is set when the NIN is confirmed via the JRB API.
	// Nil means the account is in PENDING_VERIFICATION state.
	VerifiedAt     *time.Time
}
