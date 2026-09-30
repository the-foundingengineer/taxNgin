// Package notify models the notification system. In Nigeria, WhatsApp reaches
// more taxpayers than email — a deadline reminder sent to WhatsApp three days
// before March 31 changes behaviour. Email alone does not.
//
// Key design decisions:
//   - Channel preference is explicit and ordered; fallback channels are tried
//     automatically when the primary fails.
//   - Bodies are rendered per-channel: WhatsApp allows ≤1,024 characters and
//     the ₦ symbol; SMS requires ≤160 ASCII characters (avoid ₦ which triggers
//     UCS-2 encoding and halves the limit).
//   - The clock is always injected — never time.Now() in this package.
package notify

import (
	"fmt"
	"strings"
	"time"

	"taxsoftware/taxengine"
)

// Channel is a delivery mechanism for notifications.
type Channel string

const (
	// ChannelWhatsApp is the primary channel. Highest open rate in Nigeria.
	// Requires a verified WhatsApp Business account (Twilio or Africa's Talking).
	ChannelWhatsApp Channel = "WHATSAPP"
	// ChannelSMS is the fallback. Universal reach; 160-character limit in GSM-7.
	ChannelSMS Channel = "SMS"
	// ChannelEmail is for legal notices and receipts. Do not rely on it for
	// time-sensitive deadline reminders — open rates are too low.
	ChannelEmail Channel = "EMAIL"
)

// NotifKind names the domain event that triggered the notification.
type NotifKind string

const (
	NotifDeadlineApproaching NotifKind = "deadline_approaching" // 14d, 7d, 3d, 1d
	NotifPenaltyAccruing     NotifKind = "penalty_accruing"     // already late
	NotifAssessmentServed    NotifKind = "assessment_served"
	NotifObjectionDeadline   NotifKind = "objection_deadline" // 7d, 3d before window closes
	NotifReturnCleared       NotifKind = "return_cleared"
	NotifTCCAvailable        NotifKind = "tcc_available"
	NotifTCCMissingCondition NotifKind = "tcc_missing_condition"
	NotifLoginDetected       NotifKind = "login_detected"
	NotifRecoveryInitiated   NotifKind = "recovery_initiated"
	NotifPaymentConfirmed    NotifKind = "payment_confirmed"
)

// NotifStatus is the delivery lifecycle of one Notification.
type NotifStatus string

const (
	NotifPending   NotifStatus = "PENDING"
	NotifSent      NotifStatus = "SENT"
	NotifFailed    NotifStatus = "FAILED"
	NotifCancelled NotifStatus = "CANCELLED"
)

// Notification is one message to one account on one channel.
type Notification struct {
	ID          string
	AccountID   string
	Kind        NotifKind
	Channel     Channel
	To          string // phone in E.164 ("+234XXXXXXXXXX") or email address
	Body        string // rendered, channel-appropriate text
	ScheduledAt time.Time
	SentAt      *time.Time
	Status      NotifStatus
}

// Notifier is the interface that channel providers implement.
// Implement with Twilio, Africa's Talking, or the WhatsApp Business API.
type Notifier interface {
	Send(n Notification) error
}

// StubNotifier records sent notifications. Use in tests.
type StubNotifier struct {
	Sent []Notification
}

func (s *StubNotifier) Send(n Notification) error {
	s.Sent = append(s.Sent, n)
	return nil
}

// ChannelPreference defines the delivery order for notifications.
type ChannelPreference struct {
	Primary     Channel
	Fallbacks   []Channel
	PhoneNumber string // E.164 format
	Email       string
}

// DefaultChannelPreference returns WhatsApp as primary when available, SMS as
// the first fallback, and Email as the last resort.
func DefaultChannelPreference(hasWhatsApp bool, phoneNumber, email string) ChannelPreference {
	primary := ChannelSMS
	var fallbacks []Channel
	if hasWhatsApp {
		primary = ChannelWhatsApp
		fallbacks = append(fallbacks, ChannelSMS)
	}
	if email != "" {
		fallbacks = append(fallbacks, ChannelEmail)
	}
	return ChannelPreference{
		Primary:     primary,
		Fallbacks:   fallbacks,
		PhoneNumber: phoneNumber,
		Email:       email,
	}
}

// to returns the delivery address for the given channel.
func (cp ChannelPreference) to(ch Channel) string {
	switch ch {
	case ChannelWhatsApp, ChannelSMS:
		return cp.PhoneNumber
	case ChannelEmail:
		return cp.Email
	}
	return ""
}

// NotifData carries the domain-specific fields needed to render a notification.
// Fields are zero-valued when not relevant to the NotifKind.
type NotifData struct {
	YearOfAssessment   int
	FilingDeadline     time.Time
	DaysRemaining      int
	AccruedPenaltyKobo taxengine.Kobo
	AuthorityRef       string   // assessment reference number
	TCCConditions      []string // T-condition IDs that are not met
	LoginLocation      string   // city or "an unknown location"
}

// BuildNotifications returns one Notification per channel in the preference
// order (primary first, then fallbacks). ScheduledAt is set to now for
// immediate delivery.
func BuildNotifications(kind NotifKind, data NotifData, prefs ChannelPreference, now time.Time) []Notification {
	channels := append([]Channel{prefs.Primary}, prefs.Fallbacks...)
	out := make([]Notification, 0, len(channels))
	for _, ch := range channels {
		to := prefs.to(ch)
		if to == "" {
			continue
		}
		out = append(out, Notification{
			Kind:        kind,
			Channel:     ch,
			To:          to,
			Body:        renderBody(kind, data, ch),
			ScheduledAt: now,
			Status:      NotifPending,
		})
	}
	return out
}

// DeadlineSchedule returns advance NotifDeadlineApproaching notifications
// scheduled 14, 7, 3, and 1 day before the deadline.
// Notifications whose ScheduledAt is before or equal to now are omitted —
// there is no point scheduling a reminder for a moment that has already passed.
func DeadlineSchedule(deadline time.Time, prefs ChannelPreference, now time.Time) []Notification {
	intervals := []int{14, 7, 3, 1}
	var out []Notification
	for _, days := range intervals {
		scheduledAt := deadline.AddDate(0, 0, -days)
		if !scheduledAt.After(now) {
			continue
		}
		remaining := days
		data := NotifData{
			FilingDeadline: deadline,
			DaysRemaining:  remaining,
		}
		for _, n := range BuildNotifications(NotifDeadlineApproaching, data, prefs, scheduledAt) {
			out = append(out, n)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Body rendering — channel-specific templates
// ---------------------------------------------------------------------------

const (
	maxWhatsAppChars = 1024
	maxSMSChars      = 160
)

func renderBody(kind NotifKind, data NotifData, ch Channel) string {
	var body string
	switch ch {
	case ChannelWhatsApp:
		body = renderWhatsApp(kind, data)
	case ChannelSMS:
		body = renderSMS(kind, data)
	case ChannelEmail:
		body = renderEmail(kind, data)
	}
	return body
}

func renderWhatsApp(kind NotifKind, d NotifData) string {
	switch kind {
	case NotifDeadlineApproaching:
		msg := fmt.Sprintf(
			"Your %d tax return is due in %d day(s), by %s.",
			d.YearOfAssessment, d.DaysRemaining, d.FilingDeadline.Format("2 Jan 2006"),
		)
		if d.AccruedPenaltyKobo > 0 {
			msg += " " + fmt.Sprintf("Late penalty: %s.", d.AccruedPenaltyKobo.Format())
		}
		msg += " Open the app to complete your return."
		return truncate(msg, maxWhatsAppChars)

	case NotifPenaltyAccruing:
		return truncate(fmt.Sprintf(
			"Your %d return is overdue. Penalty so far: %s and rising. File now to stop it.",
			d.YearOfAssessment, d.AccruedPenaltyKobo.Format(),
		), maxWhatsAppChars)

	case NotifAssessmentServed:
		msg := "LIRS has issued your tax assessment."
		if d.AuthorityRef != "" {
			msg += " Ref: " + d.AuthorityRef + "."
		}
		msg += " Log in to review. You have 30 days to object if the figure is wrong."
		return truncate(msg, maxWhatsAppChars)

	case NotifObjectionDeadline:
		return truncate(fmt.Sprintf(
			"Objection deadline in %d day(s). After that, the LIRS assessment becomes legally binding even if it overstates your liability. Open the app to review.",
			d.DaysRemaining,
		), maxWhatsAppChars)

	case NotifReturnCleared:
		return fmt.Sprintf("Your %d tax return has been cleared. Well done!", d.YearOfAssessment)

	case NotifTCCAvailable:
		return "Your Tax Clearance Certificate is now available. Log in to download it."

	case NotifTCCMissingCondition:
		conds := strings.Join(d.TCCConditions, ", ")
		return truncate(
			"Your TCC is not yet available. Missing: "+conds+". Open the app to see what is needed.",
			maxWhatsAppChars,
		)

	case NotifLoginDetected:
		return fmt.Sprintf(
			"New login to your account from %s. If this was not you, change your password immediately.",
			d.LoginLocation,
		)

	case NotifRecoveryInitiated:
		return "A password recovery was started for your account. If you did not request this, call us immediately."

	case NotifPaymentConfirmed:
		return "Payment confirmed. Your subscription is now active."
	}
	return ""
}

func renderSMS(kind NotifKind, d NotifData) string {
	// SMS: 160 chars, GSM-7 only (avoid ₦ — use "N" instead, ASCII).
	nairaStr := func(k taxengine.Kobo) string {
		// Use k.String() which gives "1,000,000.00" and prefix with "N".
		return "N" + k.String()
	}
	switch kind {
	case NotifDeadlineApproaching:
		msg := fmt.Sprintf("Tax due in %dd (%s).",
			d.DaysRemaining, d.FilingDeadline.Format("2 Jan"))
		if d.AccruedPenaltyKobo > 0 {
			msg += " Penalty: " + nairaStr(d.AccruedPenaltyKobo) + "."
		}
		msg += " File now."
		return truncate(msg, maxSMSChars)

	case NotifPenaltyAccruing:
		return truncate(fmt.Sprintf(
			"Tax overdue. Penalty: %s. File now to stop it.", nairaStr(d.AccruedPenaltyKobo),
		), maxSMSChars)

	case NotifAssessmentServed:
		msg := "LIRS assessment issued."
		if d.AuthorityRef != "" {
			msg += " Ref:" + d.AuthorityRef + "."
		}
		msg += " 30 days to object. Log in."
		return truncate(msg, maxSMSChars)

	case NotifObjectionDeadline:
		return truncate(fmt.Sprintf(
			"Objection deadline in %dd. NTAA s.34. Log in.", d.DaysRemaining,
		), maxSMSChars)

	case NotifReturnCleared:
		return fmt.Sprintf("%d return cleared.", d.YearOfAssessment)

	case NotifTCCAvailable:
		return "TCC ready. Log in to download."

	case NotifLoginDetected:
		return "New login. Not you? Change password now."

	case NotifRecoveryInitiated:
		return "Password recovery started. Not you? Call us."

	case NotifPaymentConfirmed:
		return "Payment confirmed. Subscription active."
	}
	return ""
}

func renderEmail(kind NotifKind, d NotifData) string {
	switch kind {
	case NotifDeadlineApproaching:
		body := fmt.Sprintf(
			"Dear taxpayer,\n\nYour %d personal income tax return is due in %d day(s), by %s.\n",
			d.YearOfAssessment, d.DaysRemaining, d.FilingDeadline.Format("2 January 2006"),
		)
		if d.AccruedPenaltyKobo > 0 {
			body += fmt.Sprintf("\nA penalty of %s is already accruing.\n", d.AccruedPenaltyKobo.Format())
		}
		body += "\nPlease log in to complete your return.\n"
		return body

	case NotifObjectionDeadline:
		return fmt.Sprintf(
			"Dear taxpayer,\n\nYou have %d day(s) remaining to object to the LIRS assessment "+
				"of your %d personal income tax return. After this window, the assessment "+
				"becomes legally binding under NTAA s.34, even where it overstates the liability.\n\n"+
				"Please log in to review the assessment and prepare an objection schedule.\n",
			d.DaysRemaining, d.YearOfAssessment,
		)

	case NotifAssessmentServed:
		return fmt.Sprintf(
			"Dear taxpayer,\n\nLIRS has issued a tax assessment for your %d return"+
				" (Reference: %s). You have 30 days to object under NTAA s.34.\n\n"+
				"Log in to compare the LIRS figure against our computed assessment.\n",
			d.YearOfAssessment, d.AuthorityRef,
		)
	}
	// For brevity, other kinds fall through to a generic email.
	return renderWhatsApp(kind, d)
}

// truncate clips body to maxLen characters without splitting a currency amount.
// If truncation is needed, a trailing "..." is appended.
func truncate(body string, maxLen int) string {
	runes := []rune(body)
	if len(runes) <= maxLen {
		return body
	}
	// Back off from maxLen to avoid cutting mid-number.
	cutAt := maxLen - 3
	for cutAt > 0 {
		c := runes[cutAt]
		if c < '0' || c > '9' {
			break
		}
		cutAt--
	}
	return string(runes[:cutAt]) + "..."
}
