package employer

import (
	"crypto/hmac"
	"crypto/sha256"
)

// ComputeHMAC returns a deterministic HMAC-SHA256 of value under key.
//
// The same key must be used consistently across the entire system. If the key
// rotates, all stored HMACs must be recomputed — store the key in config or a
// secrets manager, never in code. This function is safe to call concurrently.
func ComputeHMAC(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(value))
	return mac.Sum(nil)
}

// MatchRecord reports whether a taxpayer with the given identifiers matches rec.
//
// If all three identifiers are empty, returns false unconditionally — never
// auto-match on the absence of information. This prevents a programming error
// (forgotten NIN) from accidentally matching every record.
//
// For each non-empty identifier, the candidate value is HMACed under hmacKey
// and compared with the record's stored HMAC using hmac.Equal (constant-time
// comparison, preventing timing attacks). If the record has no HMAC for a
// provided identifier (nil slice), that identifier does not match.
//
// Logic: all provided, non-empty identifiers must match (AND semantics).
// Providing a superset of identifiers that the record has (e.g. providing both
// NIN and TaxID when the record only stores NIN) returns false — the record
// cannot confirm the second identifier.
func MatchRecord(rec PayrollRecord, hmacKey []byte, nin, taxID, email string) bool {
	if nin == "" && taxID == "" && email == "" {
		return false
	}
	if nin != "" {
		if len(rec.NINHMACSuffix) == 0 || !hmac.Equal(ComputeHMAC(hmacKey, nin), rec.NINHMACSuffix) {
			return false
		}
	}
	if taxID != "" {
		if len(rec.TaxIDHMACSuffix) == 0 || !hmac.Equal(ComputeHMAC(hmacKey, taxID), rec.TaxIDHMACSuffix) {
			return false
		}
	}
	if email != "" {
		if len(rec.EmailHMACSuffix) == 0 || !hmac.Equal(ComputeHMAC(hmacKey, email), rec.EmailHMACSuffix) {
			return false
		}
	}
	return true
}
