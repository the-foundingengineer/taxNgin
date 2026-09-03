package taxengine

import (
	"errors"
	"fmt"
	"strings"
)

// Kobo is an amount of money in Nigerian kobo. 1 naira = 100 kobo.
//
// Every monetary value in this package is a Kobo. There is no float64 anywhere
// in the engine and there must never be one: a tax figure that disagrees with
// the user's calculator by a fraction of a kobo destroys trust in every other
// number on the screen.
//
// int64 spans roughly ±92 quadrillion kobo (±₦920 trillion), which is beyond
// any conceivable personal return by many orders of magnitude.
type Kobo int64

const KoboPerNaira = 100

// ErrOverflow is returned when an intermediate product would exceed int64.
var ErrOverflow = errors.New("taxengine: arithmetic overflow")

// Naira builds a Kobo from whole naira. Use only in tests and fixtures.
func Naira(n int64) Kobo { return Kobo(n * KoboPerNaira) }

// String renders as a grouped naira figure: 1091040.00 kobo -> "10,910.40".
func (k Kobo) String() string {
	neg := k < 0
	if neg {
		k = -k
	}
	whole := int64(k) / KoboPerNaira
	frac := int64(k) % KoboPerNaira

	s := fmt.Sprintf("%d", whole)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	out := fmt.Sprintf("%s.%02d", b.String(), frac)
	if neg {
		return "-" + out
	}
	return out
}

// Format renders with the currency symbol, for traces shown to users.
func (k Kobo) Format() string { return "₦" + k.String() }

// BasisPoints is a rate where 10000 bp = 100%. 1500 bp = 15%.
//
// Rates are integers so that rate arithmetic is exact and a rule set file
// cannot smuggle in a value like 0.15000000000000002.
type BasisPoints int32

const BasisPointScale = 10000

func (bp BasisPoints) String() string {
	return fmt.Sprintf("%d.%02d%%", int32(bp)/100, int32(bp)%100)
}

// scaled returns amount*bp WITHOUT dividing. Callers accumulate scaled values
// across bands and divide exactly once, which is what
// arithmetic.rounding_applied_at: final_liability means.
func scaled(amount Kobo, bp BasisPoints) (int64, error) {
	if amount == 0 || bp == 0 {
		return 0, nil
	}
	a, b := int64(amount), int64(bp)
	prod := a * b
	// Detect overflow by inverting the multiplication.
	if prod/b != a {
		return 0, fmt.Errorf("%w: %d * %d", ErrOverflow, a, b)
	}
	return prod, nil
}

// descale divides an accumulated scaled value back to Kobo under the given
// rounding mode. This is the single rounding point for the whole computation.
func descale(total int64, mode RoundingMode) (Kobo, error) {
	switch mode {
	case RoundHalfUp:
		if total >= 0 {
			return Kobo((total + BasisPointScale/2) / BasisPointScale), nil
		}
		return Kobo((total - BasisPointScale/2) / BasisPointScale), nil
	case RoundDown:
		return Kobo(total / BasisPointScale), nil
	default:
		return 0, fmt.Errorf("taxengine: unknown rounding mode %q", mode)
	}
}

// applyRate is the convenience path for a one-shot rate application, e.g. a
// relief computed as 20% of declared rent. It rounds immediately, which is
// correct here because the relief itself is a stated figure, not an
// intermediate of the band walk.
func applyRate(amount Kobo, bp BasisPoints, mode RoundingMode) (Kobo, error) {
	s, err := scaled(amount, bp)
	if err != nil {
		return 0, err
	}
	return descale(s, mode)
}

// min returns the smaller of two amounts. Used for capped reliefs.
func minKobo(a, b Kobo) Kobo {
	if a < b {
		return a
	}
	return b
}
