package employer_test

import (
	"bytes"
	"testing"

	"taxsoftware/internal/employer"
)

var testKey = []byte("test-hmac-key-do-not-use-in-production")

func recWithNIN(nin string) employer.PayrollRecord {
	return employer.PayrollRecord{
		ID:            "rec-1",
		NINHMACSuffix: employer.ComputeHMAC(testKey, nin),
	}
}

func recWithEmail(email string) employer.PayrollRecord {
	return employer.PayrollRecord{
		ID:              "rec-2",
		EmailHMACSuffix: employer.ComputeHMAC(testKey, email),
	}
}

func TestComputeHMAC_Deterministic(t *testing.T) {
	a := employer.ComputeHMAC(testKey, "12345678901")
	b := employer.ComputeHMAC(testKey, "12345678901")
	if !bytes.Equal(a, b) {
		t.Error("same inputs must produce identical HMAC bytes")
	}
}

func TestComputeHMAC_DifferentKeys(t *testing.T) {
	key1 := []byte("key-one")
	key2 := []byte("key-two")
	a := employer.ComputeHMAC(key1, "12345678901")
	b := employer.ComputeHMAC(key2, "12345678901")
	if bytes.Equal(a, b) {
		t.Error("different keys must produce different HMACs")
	}
}

func TestComputeHMAC_DifferentValues(t *testing.T) {
	a := employer.ComputeHMAC(testKey, "11111111111")
	b := employer.ComputeHMAC(testKey, "22222222222")
	if bytes.Equal(a, b) {
		t.Error("different values must produce different HMACs")
	}
}

func TestMatchRecord_HitOnNIN(t *testing.T) {
	rec := recWithNIN("11111111111")
	if !employer.MatchRecord(rec, testKey, "11111111111", "", "") {
		t.Error("expected match on NIN")
	}
}

func TestMatchRecord_HitOnEmail(t *testing.T) {
	rec := recWithEmail("user@example.com")
	if !employer.MatchRecord(rec, testKey, "", "", "user@example.com") {
		t.Error("expected match on email")
	}
}

func TestMatchRecord_NoMatchOnWrongNIN(t *testing.T) {
	rec := recWithNIN("11111111111")
	if employer.MatchRecord(rec, testKey, "99999999999", "", "") {
		t.Error("expected no match on wrong NIN")
	}
}

func TestMatchRecord_NoMatchWhenRecordLacksField(t *testing.T) {
	// Record has no TaxID HMAC stored. Providing a taxID must fail.
	rec := recWithNIN("11111111111")
	if employer.MatchRecord(rec, testKey, "11111111111", "TIN-001", "") {
		t.Error("expected no match: record has no TaxID HMAC, cannot confirm TaxID")
	}
}

func TestMatchRecord_EmptyIdentifiers(t *testing.T) {
	rec := recWithNIN("11111111111")
	if employer.MatchRecord(rec, testKey, "", "", "") {
		t.Error("empty identifiers must never match — prevents accidental wildcard matching")
	}
}

func TestMatchRecord_ANDLogic(t *testing.T) {
	// Record has NIN stored. Providing NIN (correct) AND TaxID (but record has no TaxID) must fail.
	rec := recWithNIN("11111111111")
	if employer.MatchRecord(rec, testKey, "11111111111", "TIN-999", "") {
		t.Error("AND logic: if any provided identifier cannot be confirmed, result must be false")
	}
}
