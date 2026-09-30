package auth_test

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"taxsoftware/internal/auth"
)

var t0 = time.Date(2027, 1, 1, 10, 0, 0, 0, time.UTC)

func activeAccount() auth.TaxpayerAccount {
	return auth.TaxpayerAccount{
		ID:          "acct-1",
		PhoneNumber: "+2348012345678",
		Status:      auth.AccountActive,
	}
}

func hashOTP(t *testing.T, otp string) []byte {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash OTP: %v", err)
	}
	return h
}

// ---------------------------------------------------------------------------
// InitiateRecovery
// ---------------------------------------------------------------------------

func TestInitiateRecovery_OTPMethod(t *testing.T) {
	acct := activeAccount()
	hash := hashOTP(t, "123456")
	sess, err := auth.InitiateRecovery(acct, auth.RecoveryOTPSMS, hash, t0)
	if err != nil {
		t.Fatalf("InitiateRecovery: %v", err)
	}
	if sess.State != auth.RecoveryOTPSent {
		t.Errorf("State: want OTP_SENT, got %s", sess.State)
	}
	if sess.AccountID != acct.ID {
		t.Errorf("AccountID: want %s, got %s", acct.ID, sess.AccountID)
	}
	// OTP expiry: 10 minutes
	wantExpiry := t0.Add(10 * time.Minute)
	if !sess.ExpiresAt.Equal(wantExpiry) {
		t.Errorf("ExpiresAt: want %v, got %v", wantExpiry, sess.ExpiresAt)
	}
}

func TestInitiateRecovery_BVNLongerExpiry(t *testing.T) {
	acct := activeAccount()
	hash := hashOTP(t, "654321")
	sess, err := auth.InitiateRecovery(acct, auth.RecoveryBVNMatch, hash, t0)
	if err != nil {
		t.Fatalf("InitiateRecovery: %v", err)
	}
	// BVN expiry: 30 minutes
	wantExpiry := t0.Add(30 * time.Minute)
	if !sess.ExpiresAt.Equal(wantExpiry) {
		t.Errorf("ExpiresAt: want %v (30min), got %v", wantExpiry, sess.ExpiresAt)
	}
}

func TestInitiateRecovery_SuspendedAccount(t *testing.T) {
	acct := activeAccount()
	acct.Status = auth.AccountSuspended
	_, err := auth.InitiateRecovery(acct, auth.RecoveryOTPSMS, nil, t0)
	if !errors.Is(err, auth.ErrAccountNotActive) {
		t.Errorf("want ErrAccountNotActive, got %v", err)
	}
}

func TestInitiateRecovery_PendingAccount(t *testing.T) {
	acct := activeAccount()
	acct.Status = auth.AccountPendingVerification
	hash := hashOTP(t, "123456")
	sess, err := auth.InitiateRecovery(acct, auth.RecoveryOTPSMS, hash, t0)
	if err != nil {
		t.Fatalf("PENDING_VERIFICATION accounts should be allowed to recover: %v", err)
	}
	if sess.State != auth.RecoveryOTPSent {
		t.Errorf("State: want OTP_SENT, got %s", sess.State)
	}
}

// ---------------------------------------------------------------------------
// SubmitOTP
// ---------------------------------------------------------------------------

func TestSubmitOTP_Correct(t *testing.T) {
	hash := hashOTP(t, "999888")
	sess, _ := auth.InitiateRecovery(activeAccount(), auth.RecoveryOTPSMS, hash, t0)

	sess, err := auth.SubmitOTP(sess, "999888", t0.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("SubmitOTP: %v", err)
	}
	if sess.State != auth.RecoveryVerified {
		t.Errorf("State: want VERIFIED, got %s", sess.State)
	}
	if sess.AttemptCount != 0 {
		t.Errorf("AttemptCount should be 0 on correct OTP, got %d", sess.AttemptCount)
	}
}

func TestSubmitOTP_Expired(t *testing.T) {
	hash := hashOTP(t, "111222")
	sess, _ := auth.InitiateRecovery(activeAccount(), auth.RecoveryOTPSMS, hash, t0)

	// Submit after expiry.
	after := t0.Add(11 * time.Minute)
	_, err := auth.SubmitOTP(sess, "111222", after)
	if !errors.Is(err, auth.ErrOTPExpired) {
		t.Errorf("want ErrOTPExpired, got %v", err)
	}
}

func TestSubmitOTP_WrongOTP(t *testing.T) {
	hash := hashOTP(t, "111222")
	sess, _ := auth.InitiateRecovery(activeAccount(), auth.RecoveryOTPSMS, hash, t0)

	sess, err := auth.SubmitOTP(sess, "000000", t0.Add(1*time.Minute))
	if !errors.Is(err, auth.ErrOTPInvalid) {
		t.Errorf("want ErrOTPInvalid, got %v", err)
	}
	if sess.AttemptCount != 1 {
		t.Errorf("AttemptCount: want 1, got %d", sess.AttemptCount)
	}
}

func TestSubmitOTP_LockoutAfterFiveFailures(t *testing.T) {
	hash := hashOTP(t, "correct")
	sess, _ := auth.InitiateRecovery(activeAccount(), auth.RecoveryOTPSMS, hash, t0)

	now := t0.Add(1 * time.Minute)
	for i := 0; i < 4; i++ {
		sess, _ = auth.SubmitOTP(sess, "wrong", now)
	}
	// Fifth failure triggers lockout.
	sess, err := auth.SubmitOTP(sess, "wrong", now)
	if !errors.Is(err, auth.ErrTooManyAttempts) {
		t.Errorf("5th failure: want ErrTooManyAttempts, got %v", err)
	}
	if sess.State != auth.RecoveryExpired {
		t.Errorf("State after lockout: want EXPIRED, got %s", sess.State)
	}
	// A correct OTP after lockout must still fail.
	_, err = auth.SubmitOTP(sess, "correct", now)
	if !errors.Is(err, auth.ErrOTPExpired) {
		t.Errorf("post-lockout submit: want ErrOTPExpired, got %v", err)
	}
}

func TestSubmitOTP_UsesConstantTimeComparison(t *testing.T) {
	// bcrypt.CompareHashAndPassword is the only acceptable comparison.
	// We can't test timing directly, but we can verify the function is called:
	// a raw bytes comparison would not fail here even with a wrong hash format.
	// The test passes as long as the function works correctly (bcrypt is used).
	hash := hashOTP(t, "secure")
	sess, _ := auth.InitiateRecovery(activeAccount(), auth.RecoveryOTPSMS, hash, t0)
	sess, err := auth.SubmitOTP(sess, "secure", t0.Add(1*time.Minute))
	if err != nil {
		t.Fatalf("correct OTP should succeed: %v", err)
	}
	if sess.State != auth.RecoveryVerified {
		t.Error("bcrypt comparison must accept the correct OTP")
	}
}

// ---------------------------------------------------------------------------
// CompleteRecovery
// ---------------------------------------------------------------------------

func TestCompleteRecovery_HappyPath(t *testing.T) {
	hash := hashOTP(t, "abc123")
	sess, _ := auth.InitiateRecovery(activeAccount(), auth.RecoveryOTPSMS, hash, t0)
	sess, _ = auth.SubmitOTP(sess, "abc123", t0.Add(1*time.Minute))

	newCred := []byte("new-password-hash")
	sess, err := auth.CompleteRecovery(sess, newCred, t0.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("CompleteRecovery: %v", err)
	}
	if sess.State != auth.RecoveryCompleted {
		t.Errorf("State: want COMPLETED, got %s", sess.State)
	}
	if sess.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestCompleteRecovery_NotVerified(t *testing.T) {
	hash := hashOTP(t, "abc123")
	sess, _ := auth.InitiateRecovery(activeAccount(), auth.RecoveryOTPSMS, hash, t0)
	// Do not call SubmitOTP — session is in OTP_SENT state.
	_, err := auth.CompleteRecovery(sess, nil, t0.Add(1*time.Minute))
	if !errors.Is(err, auth.ErrSessionNotVerified) {
		t.Errorf("want ErrSessionNotVerified, got %v", err)
	}
}

func TestCompleteRecovery_ExpiredAfterVerification(t *testing.T) {
	hash := hashOTP(t, "abc123")
	sess, _ := auth.InitiateRecovery(activeAccount(), auth.RecoveryOTPSMS, hash, t0)
	sess, _ = auth.SubmitOTP(sess, "abc123", t0.Add(1*time.Minute))

	// Complete after expiry.
	after := t0.Add(15 * time.Minute) // past the 10-minute OTP window
	_, err := auth.CompleteRecovery(sess, nil, after)
	if !errors.Is(err, auth.ErrSessionExpired) {
		t.Errorf("want ErrSessionExpired, got %v", err)
	}
}
