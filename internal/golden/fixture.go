package golden

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"taxsoftware/internal/rulesyaml"
	"taxsoftware/taxengine"
)

type Kind string

const (
	KindCompute  Kind = "compute"
	KindError    Kind = "error"
	KindPenalty  Kind = "penalty"
	KindValidate Kind = "validate"
)

type Fixture struct {
	ID               string `yaml:"id"`
	Title            string `yaml:"title"`
	Kind             Kind   `yaml:"kind"`
	YearOfAssessment int    `yaml:"year_of_assessment"`
	Jurisdiction     string `yaml:"jurisdiction"`
	RuleSetVersion   string `yaml:"rule_set_version"`
	WhyThisCase      string `yaml:"why_this_case"`
	EngineStatus     string `yaml:"engine_status"`
	Given            Given  `yaml:"given"`
	Expect           Expect `yaml:"expect"`
	Review           struct {
		Status string `yaml:"status"`
	} `yaml:"review"`
}

type Given struct {
	Residency      string         `yaml:"residency"`
	MonthsEmployed int            `yaml:"months_employed"`
	Income         []IncomeYAML   `yaml:"income"`
	DeclaredBases  map[string]Naira `yaml:"declared_bases"`
	Reliefs        []ReliefYAML   `yaml:"reliefs"`
	Credits        []CreditYAML   `yaml:"credits"`
	NilDeclaration bool           `yaml:"nil_declaration"`
	Accommodation  string         `yaml:"accommodation_type"`

	FiledOn      *time.Time `yaml:"filed_on"`
	AsOf         *time.Time `yaml:"as_of"`
	Unpaid       *Naira     `yaml:"unpaid_naira"`
	PaymentDueOn *time.Time `yaml:"payment_due_on"`
	MPRBP        int        `yaml:"mpr_bp"`
}

type IncomeYAML struct {
	Category string `yaml:"category"`
	Label    string `yaml:"label"`
	Naira    Naira  `yaml:"naira"`
	Source   string `yaml:"source_ref"`
}

type ReliefYAML struct {
	Code     string `yaml:"code"`
	Naira    *Naira `yaml:"naira"`
	Evidence *bool  `yaml:"evidence"`
}

type CreditYAML struct {
	Code  string `yaml:"code"`
	Naira Naira  `yaml:"naira"`
}

type Expect struct {
	GrossIncome      *Naira           `yaml:"gross_income"`
	Reliefs          map[string]Naira `yaml:"reliefs"`
	TotalReliefs     *Naira           `yaml:"total_reliefs"`
	ChargeableIncome *Naira           `yaml:"chargeable_income"`
	Bands            []BandYAML       `yaml:"bands"`
	TaxPayable       *Naira           `yaml:"tax_payable"`
	TotalCredits     *Naira           `yaml:"total_credits"`
	Balance          *Naira           `yaml:"balance"`
	ExemptionApplied *string          `yaml:"exemption_applied"`
	ErrorContains    string           `yaml:"error"`
	Gap              string           `yaml:"gap"`
	Issues           []IssueYAML      `yaml:"issues"`
	LateFiling       *Naira           `yaml:"late_filing"`
	Surcharge        *Naira           `yaml:"late_payment_surcharge"`
	Interest         *Naira           `yaml:"late_payment_interest"`
	PenaltyTotal     *Naira           `yaml:"penalty_total"`
	MonthsInDefault  *int             `yaml:"months_in_default"`
}

type BandYAML struct {
	Slice  string `yaml:"slice"`
	Amount Naira  `yaml:"amount"`
	Rate   string `yaml:"rate"`
	Tax    Naira  `yaml:"tax"`
}

type IssueYAML struct {
	Code     string `yaml:"code"`
	Severity string `yaml:"severity"`
}

func RepoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func LoadRuleSet() (*taxengine.RuleSet, error) {
	path := filepath.Join(RepoRoot(), "rules", "ng-2026.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return taxengine.Load(raw, rulesyaml.Unmarshal, taxengine.LoadOptions{AllowDraft: true})
}

func LoadAll() ([]Fixture, []string, error) {
	dir := filepath.Join(RepoRoot(), "testdata", "golden")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	var out []Fixture
	var paths []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil, nil, err
		}
		var f Fixture
		if err := rulesyaml.Unmarshal(raw, &f); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if f.Kind == "" {
			f.Kind = KindCompute
		}
		out = append(out, f)
		paths = append(paths, p)
	}
	return out, paths, nil
}

func (f Fixture) Facts() (taxengine.TaxFacts, error) {
	facts := taxengine.TaxFacts{
		YearOfAssessment: f.YearOfAssessment,
		Jurisdiction:     f.Jurisdiction,
		IsResident:       f.Given.Residency != "nonresident",
		MonthsEmployed:   f.Given.MonthsEmployed,
		DeclaredBases:    map[string]taxengine.Kobo{},
	}
	for _, in := range f.Given.Income {
		facts.Income = append(facts.Income, taxengine.IncomeItem{
			Category:   taxengine.IncomeCategory(in.Category),
			AmountKobo: in.Naira.Kobo,
			SourceRef:  in.Source,
			Label:      in.Label,
		})
	}
	for k, v := range f.Given.DeclaredBases {
		facts.DeclaredBases[k] = v.Kobo
	}
	for _, r := range f.Given.Reliefs {
		cl := taxengine.ReliefClaim{Code: r.Code}
		if r.Naira != nil {
			cl.AmountKobo = r.Naira.Kobo
		}
		if r.Evidence != nil {
			cl.HasEvidence = *r.Evidence
		}
		facts.Reliefs = append(facts.Reliefs, cl)
	}
	for _, c := range f.Given.Credits {
		facts.Credits = append(facts.Credits, taxengine.CreditClaim{
			Code:       c.Code,
			AmountKobo: c.Naira.Kobo,
		})
	}
	return facts, nil
}

func ReliefsFromTrace(tr taxengine.Trace) map[string]taxengine.Kobo {
	out := map[string]taxengine.Kobo{}
	for _, s := range tr.Steps {
		if s.Op != "DEDUCT" || s.Label == "Reliefs restricted" {
			continue
		}
		code := codeFromReliefLabel(s.Label)
		out[code] += s.AmountKobo
	}
	return out
}

func codeFromReliefLabel(label string) string {
	switch label {
	case "Rent relief":
		return "RENT_RELIEF"
	case "Pension contribution (employee)":
		return "PENSION"
	case "National Housing Fund contribution":
		return "NHF"
	case "National Health Insurance contribution":
		return "NHIS"
	case "Life assurance premium (self and spouse)":
		return "LIFE_ASSURANCE"
	case "Medical expenses and insurance premiums":
		return "MEDICAL"
	case "Interest on loan for owner-occupied residence":
		return "HOUSING_LOAN_INTEREST"
	case "Allowable business expenses":
		return "BUSINESS_EXPENSES"
	default:
		return label
	}
}

func BandTaxes(tr taxengine.Trace) []taxengine.Kobo {
	var out []taxengine.Kobo
	for _, s := range tr.Steps {
		if s.Op == "BAND" {
			out = append(out, s.AmountKobo)
		}
	}
	return out
}
