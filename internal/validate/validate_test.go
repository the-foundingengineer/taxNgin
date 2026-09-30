package validate

import (
	"errors"
	"strings"
	"testing"
	"time"

	"taxsoftware/internal/golden"
	"taxsoftware/taxengine"
)

func TestCatalogueRequiredCodes(t *testing.T) {
	rs, err := golden.LoadRuleSet()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("E005 empty without nil declaration", func(t *testing.T) {
		got, err := Validate(completeCtx(taxengine.TaxFacts{YearOfAssessment: 2026}), rs)
		if err != nil {
			t.Fatal(err)
		}
		assertCode(t, got, "E005", SeverityBlock)
	})
	t.Run("E006 negative", func(t *testing.T) {
		ctx := completeCtx(taxengine.TaxFacts{
			YearOfAssessment: 2026,
			Income:           []taxengine.IncomeItem{{Category: taxengine.IncBusiness, AmountKobo: -1}},
		})
		got, _ := Validate(ctx, rs)
		assertCode(t, got, "E006", SeverityBlock)
	})
	t.Run("E007 unknown relief", func(t *testing.T) {
		ctx := completeCtx(taxengine.TaxFacts{
			YearOfAssessment: 2026,
			Income:           []taxengine.IncomeItem{{Category: taxengine.IncBusiness, AmountKobo: taxengine.Naira(1)}},
			Reliefs:          []taxengine.ReliefClaim{{Code: "UNICORN", AmountKobo: 1}},
		})
		got, _ := Validate(ctx, rs)
		assertCode(t, got, "E007", SeverityBlock)
	})
	t.Run("E008 draft rule set", func(t *testing.T) {
		ctx := completeCtx(taxengine.TaxFacts{
			YearOfAssessment: 2026,
			Income:           []taxengine.IncomeItem{{Category: taxengine.IncBusiness, AmountKobo: taxengine.Naira(1)}},
		})
		got, _ := Validate(ctx, rs)
		assertCode(t, got, "E008", SeverityBlock)
	})
	t.Run("W101 rent without evidence", func(t *testing.T) {
		ctx := completeCtx(taxengine.TaxFacts{
			YearOfAssessment: 2026,
			Income:           []taxengine.IncomeItem{{Category: taxengine.IncEmployment, AmountKobo: taxengine.Naira(9_000_000)}},
			Reliefs:          []taxengine.ReliefClaim{{Code: "RENT_RELIEF", HasEvidence: false}},
			DeclaredBases:    map[string]taxengine.Kobo{"annual_rent_paid": taxengine.Naira(1_000_000)},
		})
		got, _ := Validate(ctx, rs)
		assertCode(t, got, "W101", SeverityWarn)
	})
	t.Run("W103 pension over 8 percent", func(t *testing.T) {
		ctx := completeCtx(taxengine.TaxFacts{
			YearOfAssessment: 2026,
			Income:           []taxengine.IncomeItem{{Category: taxengine.IncEmployment, AmountKobo: taxengine.Naira(1_000_000)}},
			Reliefs:          []taxengine.ReliefClaim{{Code: "PENSION", AmountKobo: taxengine.Naira(90_000), HasEvidence: true}},
		})
		got, _ := Validate(ctx, rs)
		assertCode(t, got, "W103", SeverityWarn)
	})
	t.Run("W109 late filing names the penalty", func(t *testing.T) {
		ctx := completeCtx(taxengine.TaxFacts{
			YearOfAssessment: 2026,
			Income:           []taxengine.IncomeItem{{Category: taxengine.IncBusiness, AmountKobo: taxengine.Naira(1)}},
		})
		ctx.FilingDeadline = time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)
		ctx.FiledAt = time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC)
		ctx.AccruedPenaltyKobo = taxengine.Naira(100_000)
		got, _ := Validate(ctx, rs)
		assertCode(t, got, "W109", SeverityWarn)
		found := false
		for _, i := range got.Issues {
			if i.Code == "W109" && strings.Contains(i.Message, "100,000") {
				found = true
			}
		}
		if !found {
			t.Fatal("W109 must carry the accrued penalty figure")
		}
	})
	t.Run("W110 business without expenses", func(t *testing.T) {
		ctx := completeCtx(taxengine.TaxFacts{
			YearOfAssessment: 2026,
			Income:           []taxengine.IncomeItem{{Category: taxengine.IncBusiness, AmountKobo: taxengine.Naira(500_000)}},
		})
		got, _ := Validate(ctx, rs)
		assertCode(t, got, "W110", SeverityWarn)
	})
}

func TestE010RequiresContext(t *testing.T) {
	_, err := DuplicateReturn(false, false)
	if !errors.Is(err, ErrNotEnoughContext) {
		t.Fatalf("got %v", err)
	}
	iss, err := DuplicateReturn(true, true)
	if err != nil || iss.Code != "E010" {
		t.Fatalf("%v %+v", err, iss)
	}
}

func completeCtx(f taxengine.TaxFacts) FilingContext {
	return FilingContext{
		Facts:              f,
		TaxID:              "TIN-1",
		TaxIDVerified:      true,
		NIN:                "12345678901",
		Signed:             true,
		DeclaredByTaxpayer: true,
		RelevantAuthority:  "LIRS",
		YearsOnFile:        3,
	}
}

func assertCode(t *testing.T, r Result, code string, sev Severity) {
	t.Helper()
	for _, i := range r.Issues {
		if i.Code == code && i.Severity == sev {
			return
		}
	}
	t.Fatalf("missing %s (%s) in %+v", code, sev, r.Issues)
}
