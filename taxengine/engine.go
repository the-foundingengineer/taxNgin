package taxengine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// ---------------------------------------------------------------------------
// Inputs
// ---------------------------------------------------------------------------

type IncomeCategory string

const (
	IncEmployment   IncomeCategory = "EMPLOYMENT"
	IncBusiness     IncomeCategory = "BUSINESS"
	IncProfessional IncomeCategory = "PROFESSIONAL"
	IncRent         IncomeCategory = "RENT_RECEIVED"
	IncDividend     IncomeCategory = "DIVIDEND"
	IncInterest     IncomeCategory = "INTEREST"
	IncRoyalty      IncomeCategory = "ROYALTY"
	IncPension      IncomeCategory = "PENSION_RECEIVED"
	IncGratuity     IncomeCategory = "GRATUITY"
	IncDigitalAsset IncomeCategory = "DIGITAL_ASSET_GAIN"
	IncCapitalGain  IncomeCategory = "CAPITAL_GAIN"
	IncOther        IncomeCategory = "OTHER"
	IncForeign      IncomeCategory = "FOREIGN"
)

// IncomeItem is one declared line of income. Amounts are already converted to
// naira; the engine does not do FX. Conversion happens upstream at a declared
// rate and the rate is stored as evidence, because an exchange rate is a fact
// about a date, not a computation.
type IncomeItem struct {
	Category  IncomeCategory `json:"category"`
	AmountKobo Kobo          `json:"amount_kobo"`
	SourceRef string         `json:"source_ref"` // employer Tax ID, platform, tenant
	Label     string         `json:"label"`
}

// ReliefClaim is what the taxpayer says they are entitled to. The engine
// decides what is actually allowed, applying caps and formulas.
type ReliefClaim struct {
	Code       string `json:"code"`
	AmountKobo Kobo   `json:"amount_kobo"`   // for kind=actual
	HasEvidence bool  `json:"has_evidence"`
}

type CreditClaim struct {
	Code       string `json:"code"`
	AmountKobo Kobo   `json:"amount_kobo"`
}

// TaxFacts is the complete input. If two TaxFacts are equal, the output is
// equal — that is the whole contract of this package.
type TaxFacts struct {
	YearOfAssessment int    `json:"year_of_assessment"`
	Jurisdiction     string `json:"jurisdiction"`
	IsResident       bool   `json:"is_resident"`

	Income  []IncomeItem  `json:"income"`
	Reliefs []ReliefClaim `json:"reliefs"`
	Credits []CreditClaim `json:"credits"`

	// Declared bases referenced by rate_of_base reliefs, e.g.
	// {"annual_rent_paid": 300000000}. Kept separate from reliefs because the
	// base is a fact about the taxpayer's life, not a claim about tax.
	DeclaredBases map[string]Kobo `json:"declared_bases"`

	// MonthsEmployed lets the minimum-wage exemption annualise a part-year
	// employment. 12 for a full year.
	MonthsEmployed int `json:"months_employed"`
}

// Hash is the canonical fingerprint of the inputs. Stored alongside every
// assessment so you can prove which facts produced which number, and skip
// recomputation when nothing material changed.
func (f TaxFacts) Hash() (string, error) {
	c := f
	sort.Slice(c.Income, func(i, j int) bool {
		if c.Income[i].Category != c.Income[j].Category {
			return c.Income[i].Category < c.Income[j].Category
		}
		if c.Income[i].SourceRef != c.Income[j].SourceRef {
			return c.Income[i].SourceRef < c.Income[j].SourceRef
		}
		return c.Income[i].AmountKobo < c.Income[j].AmountKobo
	})
	sort.Slice(c.Reliefs, func(i, j int) bool { return c.Reliefs[i].Code < c.Reliefs[j].Code })
	sort.Slice(c.Credits, func(i, j int) bool { return c.Credits[i].Code < c.Credits[j].Code })

	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// ---------------------------------------------------------------------------
// Outputs
// ---------------------------------------------------------------------------

// Step is one line of the explanation. Every number the user sees traces back
// to one of these, and every one carries the authority it rests on.
type Step struct {
	Op       string `json:"op"`       // ADD | DEDUCT | BAND | EXEMPT | CREDIT | TOTAL
	Label    string `json:"label"`
	AmountKobo Kobo `json:"amount_kobo"`
	RunningKobo Kobo `json:"running_kobo"`
	Detail   string `json:"detail"`   // "min(20% × ₦3,000,000; ₦500,000) — cap applied"
	Citation string `json:"citation"`
}

type Trace struct {
	Steps []Step `json:"steps"`
}

func (t *Trace) add(s Step) { t.Steps = append(t.Steps, s) }

type Assessment struct {
	RuleSetVersion string `json:"rule_set_version"`
	RuleSetDigest  string `json:"rule_set_digest"`
	FactsHash      string `json:"facts_hash"`

	GrossIncomeKobo      Kobo `json:"gross_income_kobo"`
	TotalReliefsKobo     Kobo `json:"total_reliefs_kobo"`
	ChargeableIncomeKobo Kobo `json:"chargeable_income_kobo"`
	TaxPayableKobo       Kobo `json:"tax_payable_kobo"`
	TotalCreditsKobo     Kobo `json:"total_credits_kobo"`

	// BalanceKobo is positive when the taxpayer owes, negative when a refund
	// is due. One signed field, not two, so the sign can never disagree.
	BalanceKobo Kobo `json:"balance_kobo"`

	ExemptionApplied string `json:"exemption_applied,omitempty"`
}

// ---------------------------------------------------------------------------
// The computation
// ---------------------------------------------------------------------------

// Compute turns facts plus a rule set into an assessment and its explanation.
//
// It performs no I/O, reads no clock, and consults no randomness. Given the
// same arguments it returns byte-identical output forever. That property is
// what lets you replay ten million historical assessments in CI, and what lets
// you answer a query about a 2026 return in 2031.
func Compute(f TaxFacts, r *RuleSet) (Assessment, Trace, error) {
	var tr Trace
	var a Assessment

	if r == nil {
		return a, tr, fmt.Errorf("taxengine: nil rule set")
	}
	if f.YearOfAssessment != r.YearOfAssessment {
		return a, tr, fmt.Errorf(
			"taxengine: facts are for year of assessment %d but rule set is for %d",
			f.YearOfAssessment, r.YearOfAssessment)
	}

	hash, err := f.Hash()
	if err != nil {
		return a, tr, err
	}
	a.FactsHash = hash
	a.RuleSetVersion = r.Version
	a.RuleSetDigest = r.digest

	mode := r.Arithmetic.RoundingMode

	// --- 1. Gross income -----------------------------------------------------
	var gross Kobo
	byCategory := map[IncomeCategory]Kobo{}
	for _, it := range f.Income {
		if it.AmountKobo < 0 {
			return a, tr, fmt.Errorf("taxengine: negative income amount for %s", it.Category)
		}
		gross += it.AmountKobo
		byCategory[it.Category] += it.AmountKobo
		label := it.Label
		if label == "" {
			label = string(it.Category)
		}
		tr.add(Step{Op: "ADD", Label: label, AmountKobo: it.AmountKobo,
			RunningKobo: gross, Detail: string(it.Category)})
	}
	a.GrossIncomeKobo = gross
	tr.add(Step{Op: "TOTAL", Label: "Gross income", AmountKobo: gross, RunningKobo: gross})

	// --- 2. Full exemptions, before anything else ----------------------------
	for _, ex := range r.Exemptions {
		ok, why := exemptionApplies(ex, f, byCategory)
		if !ok {
			continue
		}
		a.ChargeableIncomeKobo = 0
		a.TaxPayableKobo = 0
		a.ExemptionApplied = ex.Code
		for _, c := range f.Credits {
			a.TotalCreditsKobo += c.AmountKobo
		}
		a.BalanceKobo = -a.TotalCreditsKobo // any PAYE already paid is refundable
		tr.add(Step{Op: "EXEMPT", Label: ex.Label, AmountKobo: 0, RunningKobo: 0,
			Detail: why, Citation: ex.Citation})
		return a, tr, nil
	}

	// --- 3. Reliefs ----------------------------------------------------------
	defs := map[string]ReliefDef{}
	for _, d := range r.Reliefs {
		defs[d.Code] = d
	}

	claims := append([]ReliefClaim(nil), f.Reliefs...)
	sort.Slice(claims, func(i, j int) bool { return claims[i].Code < claims[j].Code })

	var totalRelief Kobo
	for _, cl := range claims {
		def, ok := defs[cl.Code]
		if !ok {
			return a, tr, fmt.Errorf("taxengine: relief %q is not defined in rule set %s v%s",
				cl.Code, r.Jurisdiction, r.Version)
		}
		allowed, detail, err := evaluateRelief(def, cl, f, mode)
		if err != nil {
			return a, tr, err
		}
		if allowed == 0 {
			continue
		}
		totalRelief += allowed
		tr.add(Step{Op: "DEDUCT", Label: def.Label, AmountKobo: allowed,
			RunningKobo: gross - totalRelief, Detail: detail, Citation: def.Citation})
	}

	// Reliefs cannot create negative income.
	if totalRelief > gross {
		totalRelief = gross
		tr.add(Step{Op: "DEDUCT", Label: "Reliefs restricted",
			Detail: "total reliefs cannot exceed gross income", AmountKobo: 0, RunningKobo: 0})
	}
	a.TotalReliefsKobo = totalRelief

	chargeable := gross - totalRelief
	a.ChargeableIncomeKobo = chargeable
	tr.add(Step{Op: "TOTAL", Label: "Chargeable income",
		AmountKobo: chargeable, RunningKobo: chargeable})

	// --- 4. Bands ------------------------------------------------------------
	// Accumulate (slice × rate) across every band and divide exactly once.
	// This is arithmetic.rounding_applied_at: final_liability.
	var accumulated int64
	var lower Kobo
	for _, b := range r.Bands {
		if chargeable <= lower {
			break
		}
		top := chargeable
		if b.UpperKobo != nil && *b.UpperKobo < chargeable {
			top = *b.UpperKobo
		}
		slice := top - lower
		if slice > 0 {
			s, err := scaled(slice, b.RateBP)
			if err != nil {
				return a, tr, err
			}
			accumulated += s
			bandTax, err := descale(s, mode)
			if err != nil {
				return a, tr, err
			}
			tr.add(Step{
				Op:     "BAND",
				Label:  b.Label,
				AmountKobo: bandTax,
				Detail: fmt.Sprintf("%s of %s", b.RateBP, slice.Format()),
			})
		}
		if b.UpperKobo == nil {
			break
		}
		lower = *b.UpperKobo
	}

	taxPayable, err := descale(accumulated, mode)
	if err != nil {
		return a, tr, err
	}
	a.TaxPayableKobo = taxPayable
	tr.add(Step{Op: "TOTAL", Label: "Tax payable",
		AmountKobo: taxPayable, RunningKobo: taxPayable})

	// --- 5. Credits ----------------------------------------------------------
	order := map[string]int{}
	for _, c := range r.Credits {
		order[c.Code] = c.Order
	}
	creds := append([]CreditClaim(nil), f.Credits...)
	sort.Slice(creds, func(i, j int) bool {
		if order[creds[i].Code] != order[creds[j].Code] {
			return order[creds[i].Code] < order[creds[j].Code]
		}
		return creds[i].Code < creds[j].Code
	})

	running := taxPayable
	for _, c := range creds {
		if c.AmountKobo < 0 {
			return a, tr, fmt.Errorf("taxengine: negative credit %s", c.Code)
		}
		a.TotalCreditsKobo += c.AmountKobo
		running -= c.AmountKobo
		label := c.Code
		for _, d := range r.Credits {
			if d.Code == c.Code {
				label = d.Label
			}
		}
		tr.add(Step{Op: "CREDIT", Label: label, AmountKobo: c.AmountKobo, RunningKobo: running})
	}

	a.BalanceKobo = taxPayable - a.TotalCreditsKobo
	final := "Balance payable"
	if a.BalanceKobo < 0 {
		final = "Refund due"
	}
	tr.add(Step{Op: "TOTAL", Label: final, AmountKobo: a.BalanceKobo, RunningKobo: a.BalanceKobo})

	return a, tr, nil
}

func exemptionApplies(ex Exemption, f TaxFacts, byCategory map[IncomeCategory]Kobo) (bool, string) {
	if ex.Kind != "full_exemption_if_at_or_below" {
		return false, ""
	}
	// The exemption is scoped: it only applies when the taxpayer has no income
	// outside the listed categories.
	allowed := map[IncomeCategory]bool{}
	for _, c := range ex.AppliesWhenOnlyIncome {
		allowed[IncomeCategory(c)] = true
	}
	for cat, amt := range byCategory {
		if amt > 0 && !allowed[cat] {
			return false, ""
		}
	}

	var subject Kobo
	switch ex.Subject {
	case "employment_income_annualised":
		earned := byCategory[IncEmployment]
		months := f.MonthsEmployed
		if months <= 0 || months > 12 {
			months = 12
		}
		subject = earned * Kobo(12) / Kobo(months)
	default:
		return false, ""
	}

	if subject <= ex.ThresholdKobo {
		return true, fmt.Sprintf("annualised employment income %s is at or below the threshold %s",
			subject.Format(), ex.ThresholdKobo.Format())
	}
	return false, ""
}

func evaluateRelief(def ReliefDef, cl ReliefClaim, f TaxFacts, mode RoundingMode) (Kobo, string, error) {
	switch def.Kind {
	case ReliefActual:
		if cl.AmountKobo < 0 {
			return 0, "", fmt.Errorf("taxengine: negative relief claim %s", def.Code)
		}
		amt := cl.AmountKobo
		if def.CapKobo != nil && amt > *def.CapKobo {
			return *def.CapKobo, fmt.Sprintf("claimed %s, capped at %s",
				amt.Format(), def.CapKobo.Format()), nil
		}
		return amt, fmt.Sprintf("actual %s", amt.Format()), nil

	case ReliefRateOfBase:
		base, ok := f.DeclaredBases[def.Base]
		if !ok || base <= 0 {
			return 0, "", nil // nothing declared, nothing allowed
		}
		computed, err := applyRate(base, def.RateBP, mode)
		if err != nil {
			return 0, "", err
		}
		if def.CapKobo == nil {
			return computed, fmt.Sprintf("%s of %s", def.RateBP, base.Format()), nil
		}
		allowed := minKobo(computed, *def.CapKobo)
		detail := fmt.Sprintf("min(%s × %s; %s)", def.RateBP, base.Format(), def.CapKobo.Format())
		if allowed == *def.CapKobo && computed > *def.CapKobo {
			detail += " — cap applied"
		}
		return allowed, detail, nil

	default:
		return 0, "", fmt.Errorf("taxengine: unknown relief kind %q for %s", def.Kind, def.Code)
	}
}
