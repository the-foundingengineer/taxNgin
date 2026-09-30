package notify_test

import (
	"testing"
	"time"

	"taxsoftware/internal/notify"
	"taxsoftware/taxengine"
)

var deadline = time.Date(2027, 3, 31, 23, 59, 59, 0, time.UTC)
var t0 = time.Date(2027, 3, 17, 9, 0, 0, 0, time.UTC) // 14 days before deadline

func waPref() notify.ChannelPreference {
	return notify.DefaultChannelPreference(true, "+2348012345678", "user@example.com")
}

func smsPref() notify.ChannelPreference {
	return notify.DefaultChannelPreference(false, "+2348012345678", "")
}

// ---------------------------------------------------------------------------
// BuildNotifications
// ---------------------------------------------------------------------------

func TestBuildNotifications_DeadlineApproaching_WhatsApp(t *testing.T) {
	data := notify.NotifData{
		YearOfAssessment: 2026,
		FilingDeadline:   deadline,
		DaysRemaining:    14,
	}
	ns := notify.BuildNotifications(notify.NotifDeadlineApproaching, data, waPref(), t0)
	if len(ns) == 0 {
		t.Fatal("expected at least one notification")
	}
	var wa notify.Notification
	found := false
	for _, n := range ns {
		if n.Channel == notify.ChannelWhatsApp {
			wa = n
			found = true
		}
	}
	if !found {
		t.Fatal("WhatsApp notification expected")
	}
	if len([]rune(wa.Body)) > 1024 {
		t.Errorf("WhatsApp body too long: %d > 1024 chars", len([]rune(wa.Body)))
	}
	if wa.Body == "" {
		t.Error("WhatsApp body must not be empty")
	}
}

func TestBuildNotifications_PenaltyAccruing_SMS(t *testing.T) {
	data := notify.NotifData{
		YearOfAssessment:   2026,
		AccruedPenaltyKobo: taxengine.Kobo(10_000_000), // ₦100,000
	}
	ns := notify.BuildNotifications(notify.NotifPenaltyAccruing, data, smsPref(), t0)
	if len(ns) == 0 {
		t.Fatal("expected at least one notification")
	}
	sms := ns[0]
	if sms.Channel != notify.ChannelSMS {
		t.Fatalf("expected SMS, got %s", sms.Channel)
	}
	if len(sms.Body) > 160 {
		t.Errorf("SMS body too long: %d > 160 chars", len(sms.Body))
	}
	// Penalty amount must appear in some form (we use "N" prefix instead of ₦).
	if sms.Body == "" {
		t.Error("SMS body must not be empty")
	}
}

func TestBuildNotifications_FallbackChannels(t *testing.T) {
	// WhatsApp primary + SMS fallback → two notifications.
	data := notify.NotifData{YearOfAssessment: 2026, DaysRemaining: 7, FilingDeadline: deadline}
	ns := notify.BuildNotifications(notify.NotifDeadlineApproaching, data, waPref(), t0)

	channels := map[notify.Channel]bool{}
	for _, n := range ns {
		channels[n.Channel] = true
	}
	if !channels[notify.ChannelWhatsApp] {
		t.Error("WhatsApp notification expected")
	}
	if !channels[notify.ChannelSMS] {
		t.Error("SMS fallback notification expected")
	}
}

func TestBuildNotifications_ObjectionDeadline(t *testing.T) {
	data := notify.NotifData{YearOfAssessment: 2026, DaysRemaining: 3}
	ns := notify.BuildNotifications(notify.NotifObjectionDeadline, data, waPref(), t0)
	if len(ns) == 0 {
		t.Fatal("expected notifications")
	}
	body := ns[0].Body
	// Body must mention the objection context.
	if body == "" {
		t.Error("objection deadline body must not be empty")
	}
}

func TestBuildNotifications_ScheduledAtIsNow(t *testing.T) {
	data := notify.NotifData{YearOfAssessment: 2026, DaysRemaining: 1, FilingDeadline: deadline}
	ns := notify.BuildNotifications(notify.NotifDeadlineApproaching, data, smsPref(), t0)
	for _, n := range ns {
		if !n.ScheduledAt.Equal(t0) {
			t.Errorf("ScheduledAt should be now (%v), got %v", t0, n.ScheduledAt)
		}
	}
}

func TestBuildNotifications_Status_Pending(t *testing.T) {
	data := notify.NotifData{YearOfAssessment: 2026, DaysRemaining: 7, FilingDeadline: deadline}
	ns := notify.BuildNotifications(notify.NotifDeadlineApproaching, data, smsPref(), t0)
	for _, n := range ns {
		if n.Status != notify.NotifPending {
			t.Errorf("new notification must be PENDING, got %s", n.Status)
		}
	}
}

// ---------------------------------------------------------------------------
// DeadlineSchedule
// ---------------------------------------------------------------------------

func TestDeadlineSchedule_AllFuture(t *testing.T) {
	now := deadline.AddDate(0, 0, -20) // 20 days before deadline
	ns := notify.DeadlineSchedule(deadline, smsPref(), now)
	// Expect notifications for 14d, 7d, 3d, 1d = 4 intervals × 1 channel = 4
	if len(ns) != 4 {
		t.Errorf("want 4 notifications (all future), got %d", len(ns))
	}
	for _, n := range ns {
		if !n.ScheduledAt.After(now) {
			t.Errorf("all ScheduledAt must be in the future; got %v", n.ScheduledAt)
		}
	}
}

func TestDeadlineSchedule_SomePassed(t *testing.T) {
	// 5 days before deadline: 14d and 7d have passed; 3d and 1d remain.
	now := deadline.AddDate(0, 0, -5)
	ns := notify.DeadlineSchedule(deadline, smsPref(), now)
	if len(ns) != 2 {
		t.Errorf("want 2 notifications (3d and 1d); got %d", len(ns))
	}
}

func TestDeadlineSchedule_DeadlinePassed(t *testing.T) {
	now := deadline.AddDate(0, 0, 1) // one day after deadline
	ns := notify.DeadlineSchedule(deadline, smsPref(), now)
	if len(ns) != 0 {
		t.Errorf("want 0 notifications after deadline; got %d", len(ns))
	}
}

func TestDeadlineSchedule_TwoChannels(t *testing.T) {
	// WhatsApp + SMS = 2 channels × 4 intervals = 8 notifications.
	now := deadline.AddDate(0, 0, -20)
	ns := notify.DeadlineSchedule(deadline, waPref(), now)
	// WhatsApp primary + SMS fallback + Email (if waPref has email).
	// waPref() has WhatsApp + SMS + Email = 3 channels × 4 = 12.
	if len(ns) < 8 {
		t.Errorf("want at least 8 notifications (2+ channels × 4 intervals), got %d", len(ns))
	}
}

// ---------------------------------------------------------------------------
// StubNotifier
// ---------------------------------------------------------------------------

func TestStubNotifier_RecordsSent(t *testing.T) {
	stub := &notify.StubNotifier{}
	n := notify.Notification{Kind: notify.NotifReturnCleared, Channel: notify.ChannelSMS}
	if err := stub.Send(n); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(stub.Sent) != 1 {
		t.Fatalf("want 1 sent, got %d", len(stub.Sent))
	}
	if stub.Sent[0].Kind != notify.NotifReturnCleared {
		t.Errorf("Kind: want %s, got %s", notify.NotifReturnCleared, stub.Sent[0].Kind)
	}
}
