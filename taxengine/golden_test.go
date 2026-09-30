package taxengine_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"taxsoftware/internal/golden"
	"taxsoftware/internal/rulesregistry"
	"taxsoftware/internal/rulesyaml"
	"taxsoftware/internal/validate"
	"taxsoftware/taxengine"
)

// loadRegistry builds a Registry from the real rule sets in rules/ plus any
// test fixtures in testdata/rulesets/. AllowDraft is set because every rule
// set in this project is still in DRAFT status.
func loadRegistry(t *testing.T) *rulesregistry.Registry {
	t.Helper()
	root := golden.RepoRoot()
	dirs := []string{
		filepath.Join(root, "rules"),
		filepath.Join(root, "testdata", "rulesets"),
	}
	var rawFiles [][]byte
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue // testdata/rulesets/ may not exist in early checkouts
			}
			t.Fatalf("read dir %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatalf("read %s: %v", e.Name(), err)
			}
			rawFiles = append(rawFiles, raw)
		}
	}
	reg, err := rulesregistry.New(rawFiles, rulesyaml.Unmarshal, taxengine.LoadOptions{AllowDraft: true})
	if err != nil {
		t.Fatalf("build registry: %v", err)
	}
	return reg
}

func TestGoldenCorpus(t *testing.T) {
	reg := loadRegistry(t)

	fx, paths, err := golden.LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(fx) < 5 {
		t.Fatalf("expected at least the original 5 fixtures, got %d", len(fx))
	}

	for i, f := range fx {
		f, path := f, paths[i]
		t.Run(f.ID+"/"+f.Title, func(t *testing.T) {
			if f.Review.Status == "" {
				t.Errorf("%s: missing review.status", path)
			}

			// Resolve the rule set for this fixture. Fixtures that predate
			// multi-year support omit jurisdiction and year; fall back to NG/2026.
			jur := f.Jurisdiction
			if jur == "" {
				jur = "NG"
			}
			year := f.YearOfAssessment
			if year == 0 {
				year = 2026
			}
			rs, err := reg.Get(jur, year)
			if err != nil {
				t.Fatalf("no rule set for (%s, %d) — add a YAML file or check the fixture fields: %v",
					jur, year, err)
			}

			switch f.Kind {
			case golden.KindCompute, "":
				assertCompute(t, rs, f)
			case golden.KindError:
				assertError(t, rs, f)
			case golden.KindPenalty:
				assertPenalty(t, rs, f)
			case golden.KindValidate:
				assertValidate(t, rs, f)
			default:
				t.Fatalf("unknown kind %q", f.Kind)
			}
		})
	}
}

func assertCompute(t *testing.T, rs *taxengine.RuleSet, f golden.Fixture) {
	t.Helper()
	facts, err := f.Facts()
	if err != nil {
		t.Fatal(err)
	}
	a, tr, err := taxengine.Compute(facts, rs)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if f.Expect.Gap != "" {
		t.Logf("GAP %s: %s (engine still produced tax %s)", f.ID, f.Expect.Gap, a.TaxPayableKobo.Format())
	}
	cmp := func(name string, want *golden.Naira, got taxengine.Kobo) {
		t.Helper()
		if want == nil {
			return
		}
		if want.Kobo != got {
			t.Errorf("%s: want %s got %s", name, want.Kobo.Format(), got.Format())
		}
	}
	cmp("gross_income", f.Expect.GrossIncome, a.GrossIncomeKobo)
	cmp("total_reliefs", f.Expect.TotalReliefs, a.TotalReliefsKobo)
	cmp("chargeable_income", f.Expect.ChargeableIncome, a.ChargeableIncomeKobo)
	cmp("tax_payable", f.Expect.TaxPayable, a.TaxPayableKobo)
	cmp("total_credits", f.Expect.TotalCredits, a.TotalCreditsKobo)
	cmp("balance", f.Expect.Balance, a.BalanceKobo)
	if f.Expect.ExemptionApplied != nil {
		want := strings.TrimSpace(*f.Expect.ExemptionApplied)
		if want == "null" {
			want = ""
		}
		if a.ExemptionApplied != want {
			t.Errorf("exemption_applied: want %q got %q", want, a.ExemptionApplied)
		}
	}
	gotRelief := golden.ReliefsFromTrace(tr)
	for code, want := range f.Expect.Reliefs {
		if gotRelief[code] != want.Kobo {
			t.Errorf("relief %s: want %s got %s", code, want.Kobo.Format(), gotRelief[code].Format())
		}
	}
	if len(f.Expect.Bands) > 0 {
		gotBands := golden.BandTaxes(tr)
		if len(gotBands) != len(f.Expect.Bands) {
			t.Errorf("bands: want %d steps got %d", len(f.Expect.Bands), len(gotBands))
		} else {
			for i, b := range f.Expect.Bands {
				if gotBands[i] != b.Tax.Kobo {
					t.Errorf("band[%d] tax: want %s got %s", i, b.Tax.Kobo.Format(), gotBands[i].Format())
				}
			}
		}
	}
}

func assertError(t *testing.T, rs *taxengine.RuleSet, f golden.Fixture) {
	t.Helper()
	facts, err := f.Facts()
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = taxengine.Compute(facts, rs)
	if err == nil {
		t.Fatal("expected Compute to fail")
	}
	if f.Expect.ErrorContains != "" && !strings.Contains(err.Error(), f.Expect.ErrorContains) {
		t.Fatalf("error %q does not contain %q", err.Error(), f.Expect.ErrorContains)
	}
}

func assertPenalty(t *testing.T, rs *taxengine.RuleSet, f golden.Fixture) {
	t.Helper()
	in := taxengine.PenaltyInput{
		FilingDeadline:          rs.Penalties.FilingDeadline,
		CBNMonetaryPolicyRateBP: taxengine.BasisPoints(f.Given.MPRBP),
	}
	if f.Given.FiledOn != nil {
		in.FiledOn = *f.Given.FiledOn
	}
	if f.Given.AsOf != nil {
		in.AsOf = *f.Given.AsOf
	}
	if f.Given.Unpaid != nil {
		in.UnpaidKobo = f.Given.Unpaid.Kobo
	}
	if f.Given.PaymentDueOn != nil {
		in.PaymentDueOn = *f.Given.PaymentDueOn
	}
	got, err := taxengine.ComputePenalties(in, rs)
	if err != nil {
		t.Fatal(err)
	}
	if f.Expect.LateFiling != nil && got.LateFilingKobo != f.Expect.LateFiling.Kobo {
		t.Errorf("late_filing: want %s got %s", f.Expect.LateFiling.Kobo.Format(), got.LateFilingKobo.Format())
	}
	if f.Expect.Surcharge != nil && got.LatePaymentSurchargeKobo != f.Expect.Surcharge.Kobo {
		t.Errorf("surcharge: want %s got %s", f.Expect.Surcharge.Kobo.Format(), got.LatePaymentSurchargeKobo.Format())
	}
	if f.Expect.Interest != nil && got.LatePaymentInterestKobo != f.Expect.Interest.Kobo {
		t.Errorf("interest: want %s got %s", f.Expect.Interest.Kobo.Format(), got.LatePaymentInterestKobo.Format())
	}
	if f.Expect.PenaltyTotal != nil && got.TotalKobo != f.Expect.PenaltyTotal.Kobo {
		t.Errorf("penalty_total: want %s got %s", f.Expect.PenaltyTotal.Kobo.Format(), got.TotalKobo.Format())
	}
	if f.Expect.MonthsInDefault != nil && got.MonthsInDefault != *f.Expect.MonthsInDefault {
		t.Errorf("months_in_default: want %d got %d", *f.Expect.MonthsInDefault, got.MonthsInDefault)
	}
}

func assertValidate(t *testing.T, rs *taxengine.RuleSet, f golden.Fixture) {
	t.Helper()
	facts, err := f.Facts()
	if err != nil {
		t.Fatal(err)
	}
	ctx := validate.FilingContext{
		Facts:              facts,
		TaxID:              "TIN-GOLDEN",
		TaxIDVerified:      true,
		NIN:                "11111111111",
		NilDeclaration:     f.Given.NilDeclaration,
		Signed:             true,
		DeclaredByTaxpayer: true,
		RelevantAuthority:  "LIRS",
		YearsOnFile:        3,
		AccommodationType:  f.Given.Accommodation,
	}
	got, err := validate.Validate(ctx, rs)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range f.Expect.Issues {
		found := false
		for _, iss := range got.Issues {
			if iss.Code == want.Code && string(iss.Severity) == want.Severity {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing issue %s (%s); have %s", want.Code, want.Severity, issueCodes(got))
		}
	}
}

func issueCodes(r validate.Result) string {
	var b []string
	for _, i := range r.Issues {
		b = append(b, fmt.Sprintf("%s:%s", i.Code, i.Severity))
	}
	return strings.Join(b, ",")
}
