package taxengine

import (
	"fmt"
	"time"
)

// Penalty accrual under NTAA 2025.
//
// This exists because "you are late, penalties may apply" is useless and
// "you owe ₦350,000 in penalties as at today, rising to ₦400,000 on 1 October"
// changes behaviour. A late filer who can see the number files today.
//
// Everything here is pure. The clock is an argument, never a call to time.Now(),
// so a penalty figure shown to a user in April can be reproduced exactly in a
// dispute in December.

type PenaltyAssessment struct {
	AsOf time.Time `json:"as_of"`

	MonthsInDefault    int  `json:"months_in_default"`
	LateFilingKobo     Kobo `json:"late_filing_kobo"`
	LatePaymentSurchargeKobo Kobo `json:"late_payment_surcharge_kobo"`
	LatePaymentInterestKobo  Kobo `json:"late_payment_interest_kobo"`
	TotalKobo          Kobo `json:"total_kobo"`

	// NextIncreaseOn is when the figure goes up, and by how much. Show both.
	NextIncreaseOn *time.Time `json:"next_increase_on,omitempty"`
	NextIncreaseKobo Kobo     `json:"next_increase_kobo,omitempty"`

	Trace Trace `json:"trace"`
}

// PenaltyInput is everything the calculation needs. No hidden dependencies.
type PenaltyInput struct {
	FilingDeadline time.Time
	// FiledOn is when the taxpayer actually filed. Zero means not yet filed,
	// in which case AsOf is used and the figure keeps climbing.
	FiledOn time.Time
	AsOf    time.Time

	// UnpaidKobo is tax assessed and unpaid. Zero for a nil return, which still
	// attracts the filing penalty — the penalty is for failing to file, not for
	// failing to pay.
	UnpaidKobo Kobo
	// PaymentDueOn and CBNMonetaryPolicyRateBP drive the interest leg. The MPR
	// is supplied by the caller; the engine does not fetch it, because a number
	// the engine went and looked up is a number you cannot reproduce later.
	PaymentDueOn            time.Time
	CBNMonetaryPolicyRateBP BasisPoints
}

// ComputePenalties applies the NTAA s.101 filing penalty and the late payment
// surcharge and interest.
func ComputePenalties(in PenaltyInput, r *RuleSet) (PenaltyAssessment, error) {
	var pa PenaltyAssessment
	pa.AsOf = in.AsOf

	if r == nil {
		return pa, fmt.Errorf("taxengine: nil rule set")
	}
	mode := r.Arithmetic.RoundingMode

	effective := in.FiledOn
	if effective.IsZero() {
		effective = in.AsOf
	}

	// --- Late filing: NTAA s.101 -------------------------------------------
	months := monthsCommenced(in.FilingDeadline, effective)
	pa.MonthsInDefault = months

	if months > 0 {
		lf := r.Penalties.LateFiling
		pa.LateFilingKobo = lf.FirstMonthKobo + Kobo(months-1)*lf.PerSubsequentMonthKobo
		pa.Trace.add(Step{
			Op:         "PENALTY",
			Label:      "Late filing",
			AmountKobo: pa.LateFilingKobo,
			Detail: fmt.Sprintf("%s for the first month + %s × %d subsequent month(s); deadline was %s",
				lf.FirstMonthKobo.Format(), lf.PerSubsequentMonthKobo.Format(),
				months-1, in.FilingDeadline.Format("2 January 2006")),
			Citation: lf.Citation,
		})

		if in.FiledOn.IsZero() {
			next := nextMonthBoundary(in.FilingDeadline, months)
			pa.NextIncreaseOn = &next
			pa.NextIncreaseKobo = lf.PerSubsequentMonthKobo
		}
	}

	// --- Late payment: surcharge + interest ---------------------------------
	if in.UnpaidKobo > 0 && !in.PaymentDueOn.IsZero() && in.AsOf.After(in.PaymentDueOn) {
		lp := r.Penalties.LatePayment

		sur, err := applyRate(in.UnpaidKobo, lp.SurchargeBP, mode)
		if err != nil {
			return pa, err
		}
		pa.LatePaymentSurchargeKobo = sur
		pa.Trace.add(Step{
			Op: "PENALTY", Label: "Late payment surcharge", AmountKobo: sur,
			Detail:   fmt.Sprintf("%s of unpaid tax %s", lp.SurchargeBP, in.UnpaidKobo.Format()),
			Citation: lp.Citation,
		})

		days := int64(in.AsOf.Sub(in.PaymentDueOn).Hours() / 24)
		if days > 0 {
			if in.CBNMonetaryPolicyRateBP <= 0 {
				return pa, fmt.Errorf(
					"taxengine: late payment interest requested but no CBN MPR supplied; " +
						"the engine will not guess a rate")
			}
			interest, err := simpleDailyInterest(in.UnpaidKobo, in.CBNMonetaryPolicyRateBP, days, mode)
			if err != nil {
				return pa, err
			}
			pa.LatePaymentInterestKobo = interest
			pa.Trace.add(Step{
				Op: "PENALTY", Label: "Interest on unpaid tax", AmountKobo: interest,
				Detail: fmt.Sprintf("%s per annum on %s for %d day(s)",
					in.CBNMonetaryPolicyRateBP, in.UnpaidKobo.Format(), days),
				Citation: lp.Citation,
			})
		}
	}

	pa.TotalKobo = pa.LateFilingKobo + pa.LatePaymentSurchargeKobo + pa.LatePaymentInterestKobo
	pa.Trace.add(Step{Op: "TOTAL", Label: "Total penalties and interest",
		AmountKobo: pa.TotalKobo, RunningKobo: pa.TotalKobo})

	return pa, nil
}

// monthsCommenced counts calendar months of default that have BEGUN. The day
// after the deadline starts month 1.
//
// VERIFY with the practitioner: NTAA s.101 says ₦100,000 "for the first month
// in which the default occurs" and ₦50,000 "for each subsequent month the
// failure continues". Whether a partial month counts is not settled on the face
// of it. This implements the conservative reading — a commenced month counts —
// which over-states rather than under-states, and an over-stated penalty makes
// a user file sooner. Change it here, and change the golden cases with it.
func monthsCommenced(deadline, filed time.Time) int {
	if !filed.After(deadline) {
		return 0
	}
	m := (filed.Year()-deadline.Year())*12 + int(filed.Month()) - int(deadline.Month())
	if filed.Day() > deadline.Day() {
		m++
	}
	if m < 1 {
		m = 1
	}
	return m
}

// nextMonthBoundary is the date the penalty next steps up, given that `months`
// have already commenced.
func nextMonthBoundary(deadline time.Time, months int) time.Time {
	return deadline.AddDate(0, months, 0)
}

// simpleDailyInterest computes principal × rate × days / 365 in integer
// arithmetic, with a single half-up rounding. Simple, not compound.
//
// VERIFY whether the NTAA contemplates simple or compound interest, and whether
// the MPR is applied as at the due date or as it moves. This assumes simple
// interest at a single supplied rate, which is the transparent reading and the
// one a user can check.
func simpleDailyInterest(principal Kobo, rateBP BasisPoints, days int64, mode RoundingMode) (Kobo, error) {
	if principal == 0 || rateBP == 0 || days == 0 {
		return 0, nil
	}
	p, r := int64(principal), int64(rateBP)
	if p != 0 && (p*r)/p != r {
		return 0, fmt.Errorf("%w: interest principal × rate", ErrOverflow)
	}
	num := p * r
	if num != 0 && (num*days)/num != days {
		return 0, fmt.Errorf("%w: interest × days", ErrOverflow)
	}
	num *= days

	den := int64(BasisPointScale) * 365
	switch mode {
	case RoundHalfUp:
		return Kobo((num + den/2) / den), nil
	case RoundDown:
		return Kobo(num / den), nil
	default:
		return 0, fmt.Errorf("taxengine: unknown rounding mode %q", mode)
	}
}
