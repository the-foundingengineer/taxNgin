package taxengine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type RoundingMode string

const (
	RoundHalfUp RoundingMode = "half_up"
	RoundDown   RoundingMode = "down"
)

type RuleStatus string

const (
	StatusDraft     RuleStatus = "DRAFT"
	StatusPublished RuleStatus = "PUBLISHED"
	StatusSuperseded RuleStatus = "SUPERSEDED"
)

// RuleSet is the tax law for one jurisdiction and one year of assessment,
// as data. It is immutable once published: to change a number you publish a
// new version, because every stored assessment references the version that
// produced it and must remain reproducible for as long as the tax authority
// can query it.
type RuleSet struct {
	Jurisdiction     string     `yaml:"jurisdiction"`
	YearOfAssessment int        `yaml:"year_of_assessment"`
	Version          string     `yaml:"version"`
	EffectiveFrom    time.Time  `yaml:"effective_from"`
	EffectiveTo      *time.Time `yaml:"effective_to"`
	Status           RuleStatus `yaml:"status"`

	ApprovedBy Approval   `yaml:"approved_by"`
	Source     Source     `yaml:"source"`
	Arithmetic Arithmetic `yaml:"arithmetic"`

	Bands      []Band      `yaml:"bands"`
	Reliefs    []ReliefDef `yaml:"reliefs"`
	Exemptions []Exemption `yaml:"exemptions"`
	Credits    []CreditDef `yaml:"credits"`
	Penalties  Penalties   `yaml:"penalties"`

	// digest is computed at load time and recorded on every assessment, so a
	// file edited in place after publication is detectable.
	digest string
}

// Approval records the accredited practitioner who signed off these numbers.
// An unsigned rule set is a spreadsheet, not tax law.
type Approval struct {
	Name         string     `yaml:"name"`
	MembershipNo string     `yaml:"membership_no"`
	ApprovedAt   *time.Time `yaml:"approved_at"`
	Scope        string     `yaml:"scope"`
}

type Source struct {
	Primary   string   `yaml:"primary"`
	Secondary []string `yaml:"secondary"`
}

type Arithmetic struct {
	Unit              string       `yaml:"unit"`
	RateUnit          string       `yaml:"rate_unit"`
	RoundingMode      RoundingMode `yaml:"rounding_mode"`
	RoundingAppliedAt string       `yaml:"rounding_applied_at"`
}

// Band is one slice of the progressive scale. UpperKobo is the top of the
// slice, not its width. The final band has UpperKobo nil.
type Band struct {
	UpperKobo *Kobo       `yaml:"upper_kobo"`
	RateBP    BasisPoints `yaml:"rate_bp"`
	Label     string      `yaml:"label"`
}

type ReliefKind string

const (
	// ReliefActual deducts the amount actually incurred, optionally capped.
	ReliefActual ReliefKind = "actual"
	// ReliefRateOfBase deducts RateBP of a declared base figure, optionally capped.
	// Rent relief is the only current user: 20% of annual rent, capped at ₦500,000.
	ReliefRateOfBase ReliefKind = "rate_of_base"
)

type ReliefDef struct {
	Code             string      `yaml:"code"`
	Label            string      `yaml:"label"`
	Kind             ReliefKind  `yaml:"kind"`
	Base             string      `yaml:"base"`     // for rate_of_base
	RateBP           BasisPoints `yaml:"rate_bp"`  // for rate_of_base
	CapKobo          *Kobo       `yaml:"cap_kobo"` // nil = uncapped
	RequiresEvidence bool        `yaml:"requires_evidence"`
	Citation         string      `yaml:"citation"`
	Note             string      `yaml:"note"`
}

type Exemption struct {
	Code                  string   `yaml:"code"`
	Label                 string   `yaml:"label"`
	Kind                  string   `yaml:"kind"`
	Subject               string   `yaml:"subject"`
	ThresholdKobo         Kobo     `yaml:"threshold_kobo"`
	AppliesWhenOnlyIncome []string `yaml:"applies_when_only_income_is"`
	Citation              string   `yaml:"citation"`
	Note                  string   `yaml:"note"`
}

type CreditDef struct {
	Code     string `yaml:"code"`
	Label    string `yaml:"label"`
	Order    int    `yaml:"order"`
	Citation string `yaml:"citation"`
	Note     string `yaml:"note"`
}

type Penalties struct {
	FilingDeadline time.Time `yaml:"filing_deadline"`
	LateFiling     struct {
		FirstMonthKobo         Kobo   `yaml:"first_month_kobo"`
		PerSubsequentMonthKobo Kobo   `yaml:"per_subsequent_month_kobo"`
		Citation               string `yaml:"citation"`
	} `yaml:"late_filing"`
	LatePayment struct {
		SurchargeBP   BasisPoints `yaml:"surcharge_bp"`
		InterestBasis string      `yaml:"interest_basis"`
		Citation      string      `yaml:"citation"`
	} `yaml:"late_payment"`
	ObjectionWindowDays int `yaml:"objection_window_days"`
}

// Digest is the SHA-256 of the rule set file as loaded. Recorded on every
// assessment. If someone edits a published file in place, every subsequent
// assessment carries a different digest and the drift is visible.
func (r *RuleSet) Digest() string { return r.digest }

// LoadOptions controls how strict loading is.
type LoadOptions struct {
	// AllowDraft permits loading a rule set that no practitioner has signed.
	// Production callers must leave this false. Set it in tests and in the
	// authoring tool, nowhere else.
	AllowDraft bool
}

// Load parses and validates a rule set. It refuses anything that would produce
// a wrong number silently.
func Load(raw []byte, unmarshal func([]byte, any) error, opts LoadOptions) (*RuleSet, error) {
	var rs RuleSet
	if err := unmarshal(raw, &rs); err != nil {
		return nil, fmt.Errorf("taxengine: parsing rule set: %w", err)
	}
	sum := sha256.Sum256(raw)
	rs.digest = hex.EncodeToString(sum[:])

	if err := rs.Validate(opts); err != nil {
		return nil, err
	}
	return &rs, nil
}

// Validate enforces the invariants that make the engine safe. Every one of
// these has a plausible failure mode that would otherwise ship a wrong number
// to a real taxpayer.
func (r *RuleSet) Validate(opts LoadOptions) error {
	if r.Jurisdiction == "" {
		return fmt.Errorf("taxengine: rule set has no jurisdiction")
	}
	if r.YearOfAssessment == 0 {
		return fmt.Errorf("taxengine: rule set has no year_of_assessment")
	}
	if r.Version == "" {
		return fmt.Errorf("taxengine: rule set has no version")
	}

	// The gate that matters. A rule set nobody has signed cannot compute a
	// number a taxpayer relies on.
	if r.Status != StatusPublished && !opts.AllowDraft {
		return fmt.Errorf(
			"taxengine: rule set %s/%d v%s has status %q and is not signed off; "+
				"refusing to load (set LoadOptions.AllowDraft only in tests)",
			r.Jurisdiction, r.YearOfAssessment, r.Version, r.Status)
	}
	if r.Status == StatusPublished && (r.ApprovedBy.Name == "" || r.ApprovedBy.ApprovedAt == nil) {
		return fmt.Errorf("taxengine: rule set %s v%s is PUBLISHED but has no approver on record",
			r.Jurisdiction, r.Version)
	}

	switch r.Arithmetic.RoundingMode {
	case RoundHalfUp, RoundDown:
	default:
		return fmt.Errorf("taxengine: unknown rounding_mode %q", r.Arithmetic.RoundingMode)
	}
	if r.Arithmetic.Unit != "kobo" {
		return fmt.Errorf("taxengine: unit must be kobo, got %q", r.Arithmetic.Unit)
	}

	if len(r.Bands) == 0 {
		return fmt.Errorf("taxengine: rule set has no bands")
	}
	var prev Kobo = 0
	for i, b := range r.Bands {
		last := i == len(r.Bands)-1
		if last {
			if b.UpperKobo != nil {
				return fmt.Errorf("taxengine: final band must have upper_kobo: null")
			}
			continue
		}
		if b.UpperKobo == nil {
			return fmt.Errorf("taxengine: band %d has null upper_kobo but is not final", i)
		}
		if *b.UpperKobo <= prev {
			return fmt.Errorf("taxengine: band %d upper_kobo %s does not exceed previous %s",
				i, b.UpperKobo.String(), prev.String())
		}
		prev = *b.UpperKobo
	}
	for i, b := range r.Bands {
		if b.RateBP < 0 || b.RateBP > BasisPointScale {
			return fmt.Errorf("taxengine: band %d rate %s outside 0–100%%", i, b.RateBP)
		}
	}

	seen := map[string]bool{}
	for _, rd := range r.Reliefs {
		if seen[rd.Code] {
			return fmt.Errorf("taxengine: duplicate relief code %q", rd.Code)
		}
		seen[rd.Code] = true
		switch rd.Kind {
		case ReliefActual:
		case ReliefRateOfBase:
			if rd.Base == "" {
				return fmt.Errorf("taxengine: relief %q is rate_of_base but names no base", rd.Code)
			}
			if rd.RateBP <= 0 {
				return fmt.Errorf("taxengine: relief %q is rate_of_base but has no rate", rd.Code)
			}
		default:
			return fmt.Errorf("taxengine: relief %q has unknown kind %q", rd.Code, rd.Kind)
		}
	}
	return nil
}
