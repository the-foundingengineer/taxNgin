package taxengine

import (
	"errors"
	"math"
	"testing"
)

func TestNairaRoundTrip(t *testing.T) {
	cases := []int64{0, 1, 100, 840000, 12930000, -5000}
	for _, n := range cases {
		k := Naira(n)
		whole := int64(k) / KoboPerNaira
		if whole != n {
			t.Errorf("Naira(%d) -> %d kobo, whole naira = %d", n, k, whole)
		}
	}
}

func TestKoboString(t *testing.T) {
	tests := []struct {
		k    Kobo
		want string
	}{
		{Kobo(0), "0.00"},
		{Kobo(100), "1.00"},
		{Kobo(109104000), "1,091,040.00"},
		{Kobo(84000000), "840,000.00"},
		{Kobo(1293000000), "12,930,000.00"},
		{Kobo(-600000), "-6,000.00"},
		{Kobo(-1), "-0.01"},
		{Kobo(1), "0.01"},
		{Kobo(99), "0.99"},
	}
	for _, tt := range tests {
		if got := tt.k.String(); got != tt.want {
			t.Errorf("Kobo(%d).String() = %q, want %q", tt.k, got, tt.want)
		}
	}
}

func TestKoboFormat(t *testing.T) {
	tests := []struct {
		k    Kobo
		want string
	}{
		{Kobo(0), "\u20A60.00"},
		{Kobo(109104000), "\u20A61,091,040.00"},
		{Kobo(-600000), "\u20A6-6,000.00"},
	}
	for _, tt := range tests {
		if got := tt.k.Format(); got != tt.want {
			t.Errorf("Kobo(%d).Format() = %q, want %q", tt.k, got, tt.want)
		}
	}
}

func TestBasisPointsString(t *testing.T) {
	tests := []struct {
		bp   BasisPoints
		want string
	}{
		{BasisPoints(0), "0.00%"},
		{BasisPoints(1500), "15.00%"},
		{BasisPoints(1800), "18.00%"},
		{BasisPoints(2500), "25.00%"},
		{BasisPoints(2000), "20.00%"},
	}
	for _, tt := range tests {
		if got := tt.bp.String(); got != tt.want {
			t.Errorf("BasisPoints(%d).String() = %q, want %q", tt.bp, got, tt.want)
		}
	}
}

func TestDescaleHalfUp(t *testing.T) {
	tests := []struct {
		total int64
		want  Kobo
	}{
		// Exactly on boundary: 10000/10000 = 1.00, +half rounds to 1
		{10000, Kobo(1)},
		// 5000 is exactly half, rounds up to 1
		{5000, Kobo(1)},
		// 4999 rounds down to 0
		{4999, Kobo(0)},
		// 15000 / 10000 = 1.5 -> 2 with half-up
		{15000, Kobo(2)},
		// Negative: -5000 should round down (away from zero) to -1
		{-5000, Kobo(-1)},
		{-4999, Kobo(0)},
		{-10000, Kobo(-1)},
		{0, Kobo(0)},
	}
	for _, tt := range tests {
		got, err := descale(tt.total, RoundHalfUp)
		if err != nil {
			t.Errorf("descale(%d) error: %v", tt.total, err)
			continue
		}
		if got != tt.want {
			t.Errorf("descale(%d) = %d, want %d", tt.total, got, tt.want)
		}
	}
}

func TestDescaleDown(t *testing.T) {
	tests := []struct {
		total int64
		want  Kobo
	}{
		{9999, Kobo(0)},
		{10000, Kobo(1)},
		{15000, Kobo(1)},
		{-9999, Kobo(0)},
		{-10000, Kobo(-1)},
	}
	for _, tt := range tests {
		got, err := descale(tt.total, RoundDown)
		if err != nil {
			t.Errorf("descale(%d) error: %v", tt.total, err)
			continue
		}
		if got != tt.want {
			t.Errorf("descale(%d, Down) = %d, want %d", tt.total, got, tt.want)
		}
	}
}

func TestDescaleUnknownMode(t *testing.T) {
	_, err := descale(1000, "bogus")
	if err == nil {
		t.Fatal("expected error for unknown rounding mode")
	}
}

func TestScaledOverflow(t *testing.T) {
	max := int64(math.MaxInt64)
	_, err := scaled(Kobo(max), BasisPoints(2))
	if err == nil {
		t.Fatal("expected overflow error")
	}
	if !errors.Is(err, ErrOverflow) {
		t.Errorf("expected ErrOverflow, got %v", err)
	}
}

func TestScaledNoOverflow(t *testing.T) {
	_, err := scaled(Kobo(100000000000), BasisPoints(2500))
	if err != nil {
		t.Errorf("unexpected overflow: %v", err)
	}
}

func TestApplyRateHalfUp(t *testing.T) {
	// 20% of 2,500,000 naira = 500,000 naira
	got, err := applyRate(Naira(2500000), BasisPoints(2000), RoundHalfUp)
	if err != nil {
		t.Fatalf("applyRate error: %v", err)
	}
	if want := Naira(500000); got != want {
		t.Errorf("applyRate(2.5M, 20%%) = %d, want %d", got, want)
	}
}

func TestMinKobo(t *testing.T) {
	a := Naira(100)
	b := Naira(200)
	if got := minKobo(a, b); got != a {
		t.Errorf("minKobo(100, 200) = %d, want %d", got, a)
	}
	if got := minKobo(b, a); got != a {
		t.Errorf("minKobo(200, 100) = %d, want %d", got, a)
	}
	if got := minKobo(a, a); got != a {
		t.Errorf("minKobo(100, 100) = %d, want %d", got, a)
	}
}
