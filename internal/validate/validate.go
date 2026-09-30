package validate

import (
	"errors"
	"fmt"
	"time"

	"taxsoftware/taxengine"
)

var ErrNotEnoughContext = errors.New("validate: not enough context")

type Severity string

const (
	SeverityBlock Severity = "block"
	SeverityWarn  Severity = "warn"
)

type Issue struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Field    string   `json:"field"`
	Message  string   `json:"message"`
	Citation string   `json:"citation"`
}

type FilingContext struct {
	Facts taxengine.TaxFacts

	TaxID         string
	TaxIDVerified bool
	NIN           string

	NilDeclaration bool
	Signed         bool
	DeclaredByTaxpayer bool

	AccommodationType string
	EmployerGrossKobo *taxengine.Kobo
	PriorYearGrossKobo *taxengine.Kobo
	YearsOnFile       int

	FiledAt        time.Time
	FilingDeadline time.Time
	AccruedPenaltyKobo taxengine.Kobo

	ExchangeActivityImported bool
	DigitalAssetDeclaredZero bool

	DuplicateKnown bool
	DuplicateCheckOK bool

	RelevantAuthority string
}

type Result struct {
	Issues []Issue
}

func (r Result) Blocking() []Issue {
	var out []Issue
	for _, i := range r.Issues {
		if i.Severity == SeverityBlock {
			out = append(out, i)
		}
	}
	return out
}

func Validate(ctx FilingContext, rs *taxengine.RuleSet) (Result, error) {
	var out Result
	if rs == nil {
		return out, fmt.Errorf("validate: nil rule set")
	}

	if ctx.TaxID == "" || !ctx.TaxIDVerified {
		out.Issues = append(out.Issues, Issue{
			Code: "E001", Severity: SeverityBlock, Field: "tax_id",
			Message:  "Tax ID must be present and verified before filing",
			Citation: "NTAA s.100",
		})
	}
	if ctx.NIN == "" {
		out.Issues = append(out.Issues, Issue{
			Code: "E002", Severity: SeverityBlock, Field: "nin",
			Message:  "NIN is required for an individual; Tax ID derives from it",
			Citation: "NTA 2025",
		})
	}
	if ctx.Facts.YearOfAssessment != rs.YearOfAssessment {
		out.Issues = append(out.Issues, Issue{
			Code: "E003", Severity: SeverityBlock, Field: "year_of_assessment",
			Message:  "Year of assessment is inconsistent with the loaded rule set",
			Citation: "return schema",
		})
	}
	if ctx.RelevantAuthority == "" {
		out.Issues = append(out.Issues, Issue{
			Code: "E004", Severity: SeverityBlock, Field: "jurisdiction",
			Message:  "Relevant tax authority is not resolved from residency",
			Citation: "PIT jurisdiction rules",
		})
	}

	hasIncome := false
	for _, in := range ctx.Facts.Income {
		if in.AmountKobo > 0 {
			hasIncome = true
		}
		if in.AmountKobo < 0 {
			out.Issues = append(out.Issues, Issue{
				Code: "E006", Severity: SeverityBlock, Field: string(in.Category),
				Message: "Negative amounts are not permitted",
			})
		}
	}
	if !hasIncome && !ctx.NilDeclaration {
		out.Issues = append(out.Issues, Issue{
			Code: "E005", Severity: SeverityBlock, Field: "income",
			Message:  "A return needs at least one income item, or an explicit nil declaration",
			Citation: "nil return is a positive act",
		})
	}

	defs := map[string]taxengine.ReliefDef{}
	for _, d := range rs.Reliefs {
		defs[d.Code] = d
	}
	var employment, business, pensionClaim, rentClaim taxengine.Kobo
	var rentEvidence bool
	var businessExpense bool
	for _, in := range ctx.Facts.Income {
		switch in.Category {
		case taxengine.IncEmployment:
			employment += in.AmountKobo
		case taxengine.IncBusiness, taxengine.IncProfessional:
			business += in.AmountKobo
		}
	}
	for _, cl := range ctx.Facts.Reliefs {
		if _, ok := defs[cl.Code]; !ok {
			out.Issues = append(out.Issues, Issue{
				Code: "E007", Severity: SeverityBlock, Field: cl.Code,
				Message:  fmt.Sprintf("Relief %q is not defined in rule set %s v%s", cl.Code, rs.Jurisdiction, rs.Version),
				Citation: "rule set relief catalogue",
			})
			continue
		}
		if cl.AmountKobo < 0 {
			out.Issues = append(out.Issues, Issue{
				Code: "E006", Severity: SeverityBlock, Field: cl.Code,
				Message: "Negative amounts are not permitted",
			})
		}
		switch cl.Code {
		case "PENSION":
			pensionClaim += cl.AmountKobo
		case "RENT_RELIEF":
			rentClaim++
			rentEvidence = cl.HasEvidence
		case "BUSINESS_EXPENSES":
			if cl.AmountKobo > 0 {
				businessExpense = true
			}
		}
	}
	for _, c := range ctx.Facts.Credits {
		if c.AmountKobo < 0 {
			out.Issues = append(out.Issues, Issue{
				Code: "E006", Severity: SeverityBlock, Field: c.Code,
				Message: "Negative amounts are not permitted",
			})
		}
	}

	if rs.Status != taxengine.StatusPublished {
		out.Issues = append(out.Issues, Issue{
			Code: "E008", Severity: SeverityBlock, Field: "rule_set",
			Message:  "Rule set status is not PUBLISHED",
			Citation: "sign-off gate",
		})
	}
	if !ctx.Signed {
		out.Issues = append(out.Issues, Issue{
			Code: "E009", Severity: SeverityBlock, Field: "declaration",
			Message:  "Declaration must be signed before submission",
			Citation: "Shape A: taxpayer signs",
		})
	}
	if ctx.DuplicateKnown {
		if !ctx.DuplicateCheckOK {
			return out, fmt.Errorf("%w for E010 duplicate-return check", ErrNotEnoughContext)
		}
	}

	if rentClaim > 0 && !rentEvidence {
		out.Issues = append(out.Issues, Issue{
			Code: "W101", Severity: SeverityWarn, Field: "RENT_RELIEF",
			Message:  "Rent relief claimed without uploaded evidence",
			Citation: "NTA 2025 — rent relief requires accurate declaration and evidence",
		})
	}
	if ctx.EmployerGrossKobo != nil && employment != *ctx.EmployerGrossKobo {
		out.Issues = append(out.Issues, Issue{
			Code: "W102", Severity: SeverityWarn, Field: "income.EMPLOYMENT",
			Message: "Declared employment income differs from employer payroll record",
		})
	}
	if pensionClaim > 0 && employment > 0 {
		cap := eightPercent(employment)
		if pensionClaim > cap {
			out.Issues = append(out.Issues, Issue{
				Code: "W103", Severity: SeverityWarn, Field: "PENSION",
				Message:  "Pension claimed exceeds 8% of declared employment income",
				Citation: "Pension Reform Act common base check",
			})
		}
	}
	if rentClaim > 0 && ctx.AccommodationType == "owner-occupied" {
		out.Issues = append(out.Issues, Issue{
			Code: "W104", Severity: SeverityWarn, Field: "accommodation",
			Message: "Rent relief claimed but accommodation recorded as owner-occupied",
		})
	}
	if ctx.PriorYearGrossKobo != nil && *ctx.PriorYearGrossKobo > 0 {
		half := *ctx.PriorYearGrossKobo / 2
		gross := sumGross(ctx.Facts)
		if gross < half {
			out.Issues = append(out.Issues, Issue{
				Code: "W105", Severity: SeverityWarn, Field: "income",
				Message: "Prior year filed but this year's income fell by more than 50%",
			})
		}
	}
	if ctx.YearsOnFile < 3 {
		out.Issues = append(out.Issues, Issue{
			Code: "W106", Severity: SeverityWarn, Field: "history",
			Message: "Fewer than three years of returns on file — this blocks a TCC",
		})
	}
	hasForeign := false
	hasFTC := false
	for _, in := range ctx.Facts.Income {
		if in.Category == taxengine.IncForeign && in.AmountKobo > 0 {
			hasForeign = true
		}
	}
	for _, c := range ctx.Facts.Credits {
		if c.Code == "FOREIGN_TAX_CREDIT" && c.AmountKobo > 0 {
			hasFTC = true
		}
	}
	if hasForeign && !hasFTC {
		out.Issues = append(out.Issues, Issue{
			Code: "W107", Severity: SeverityWarn, Field: "FOREIGN_TAX_CREDIT",
			Message:  "Foreign income declared with no foreign tax credit claimed",
			Citation: "NTA 2025 — unilateral relief may be available",
		})
	}
	if ctx.ExchangeActivityImported && ctx.DigitalAssetDeclaredZero {
		out.Issues = append(out.Issues, Issue{
			Code: "W108", Severity: SeverityWarn, Field: "DIGITAL_ASSET_GAIN",
			Message:  "Digital asset gains declared as zero while exchange activity was imported",
			Citation: "NTA 2025 s.4",
		})
	}
	if !ctx.FiledAt.IsZero() && !ctx.FilingDeadline.IsZero() && ctx.FiledAt.After(ctx.FilingDeadline) {
		out.Issues = append(out.Issues, Issue{
			Code: "W109", Severity: SeverityWarn, Field: "filed_at",
			Message:  fmt.Sprintf("Filing after the deadline; accrued penalty %s", ctx.AccruedPenaltyKobo.Format()),
			Citation: "NTAA 2025 s.101",
		})
	}
	if business > 0 && !businessExpense {
		out.Issues = append(out.Issues, Issue{
			Code: "W110", Severity: SeverityWarn, Field: "BUSINESS_EXPENSES",
			Message: "Business income declared with no expenses",
		})
	}

	_ = ctx.DeclaredByTaxpayer
	return out, nil
}

func sumGross(f taxengine.TaxFacts) taxengine.Kobo {
	var g taxengine.Kobo
	for _, in := range f.Income {
		g += in.AmountKobo
	}
	return g
}

func eightPercent(employment taxengine.Kobo) taxengine.Kobo {
	prod := int64(employment) * 800
	return taxengine.Kobo((prod + 5000) / 10000)
}

func DuplicateReturn(checked, alreadyFiled bool) (Issue, error) {
	if !checked {
		return Issue{}, fmt.Errorf("%w: E010 duplicate-return check", ErrNotEnoughContext)
	}
	if alreadyFiled {
		return Issue{
			Code: "E010", Severity: SeverityBlock, Field: "return",
			Message: "A return already exists for this taxpayer, year of assessment, and jurisdiction",
		}, nil
	}
	return Issue{}, nil
}
