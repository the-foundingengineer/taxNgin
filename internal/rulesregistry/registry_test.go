package rulesregistry_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"taxsoftware/internal/rulesregistry"
	"taxsoftware/internal/rulesyaml"
	"taxsoftware/taxengine"
)

// ---------------------------------------------------------------------------
// Infrastructure
// ---------------------------------------------------------------------------

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// minimalYAML returns a valid, self-contained rule set YAML string for the
// given jurisdiction and year. Used for in-memory unit tests that do not need
// a real rule set file.
func minimalYAML(jurisdiction string, year int, firstBandBP int) string {
	return `jurisdiction: ` + jurisdiction + `
year_of_assessment: ` + itoa(year) + `
version: "unit-test-fixture"
effective_from: 2000-01-01
status: DRAFT
approved_by:
  name: ""
  membership_no: ""
source:
  primary: "unit test"
arithmetic:
  unit: kobo
  rate_unit: basis_points
  rounding_mode: half_up
  rounding_applied_at: final_liability
bands:
  - { upper_kobo: 80000000, rate_bp: ` + itoa(firstBandBP) + `, label: "First band" }
  - { upper_kobo: null, rate_bp: 2500, label: "Top band" }
penalties:
  filing_deadline: 2000-03-31
  late_filing:
    first_month_kobo: 10000000
    per_subsequent_month_kobo: 5000000
    citation: "test"
  late_payment:
    surcharge_bp: 1000
    interest_basis: CBN_MPR
    citation: "test"
  objection_window_days: 30
`
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func loadOpts() taxengine.LoadOptions { return taxengine.LoadOptions{AllowDraft: true} }

// ---------------------------------------------------------------------------
// Unit tests (in-memory YAML)
// ---------------------------------------------------------------------------

func TestNew_RejectsInvalidYAML(t *testing.T) {
	garbage := []byte("%%% not valid yaml %%%")
	_, err := rulesregistry.New([][]byte{garbage}, rulesyaml.Unmarshal, loadOpts())
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestNew_RejectsDuplicateJurisdictionYear(t *testing.T) {
	raw := []byte(minimalYAML("NG", 2026, 0))
	_, err := rulesregistry.New([][]byte{raw, raw}, rulesyaml.Unmarshal, loadOpts())
	if err == nil {
		t.Fatal("expected error for duplicate (jurisdiction, year), got nil")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error should mention 'duplicate', got: %v", err)
	}
}

func TestGet_NotFound(t *testing.T) {
	raw := []byte(minimalYAML("NG", 2026, 0))
	reg, err := rulesregistry.New([][]byte{raw}, rulesyaml.Unmarshal, loadOpts())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = reg.Get("NG", 1900)
	if err == nil {
		t.Fatal("expected ErrRuleSetNotFound, got nil")
	}
	if !errors.Is(err, rulesregistry.ErrRuleSetNotFound) {
		t.Errorf("want ErrRuleSetNotFound, got: %v", err)
	}
}

func TestGet_Found(t *testing.T) {
	raw := []byte(minimalYAML("NG", 2026, 0))
	reg, err := rulesregistry.New([][]byte{raw}, rulesyaml.Unmarshal, loadOpts())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	rs, err := reg.Get("NG", 2026)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rs.Jurisdiction != "NG" || rs.YearOfAssessment != 2026 {
		t.Errorf("wrong rule set returned: %s/%d", rs.Jurisdiction, rs.YearOfAssessment)
	}
}

// TestGet_MultipleRuleSets proves that a registry with two rule sets returns
// the correct one for each (jurisdiction, year) — including verifying that the
// two rule sets produce different tax outputs from the same income, which is
// the architectural proof that the engine is purely data-driven.
func TestGet_MultipleRuleSets(t *testing.T) {
	// Two rule sets that differ only in the first-band rate.
	raw0 := []byte(minimalYAML("NG", 2026, 0))   // first band 0 bp → ₦0 tax on ₦600,000
	raw500 := []byte(minimalYAML("NG", 1, 500))   // first band 500 bp → ₦30,000 tax on ₦600,000

	reg, err := rulesregistry.New([][]byte{raw0, raw500}, rulesyaml.Unmarshal, loadOpts())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rs2026, err := reg.Get("NG", 2026)
	if err != nil {
		t.Fatalf("Get 2026: %v", err)
	}
	rs1, err := reg.Get("NG", 1)
	if err != nil {
		t.Fatalf("Get 1: %v", err)
	}

	// Same facts, two rule sets — different outcomes.
	facts := taxengine.TaxFacts{
		YearOfAssessment: 0,        // will be overridden per-call
		Jurisdiction:     "NG",
		IsResident:       true,
		Income: []taxengine.IncomeItem{
			{Category: taxengine.IncDividend, AmountKobo: 60_000_000}, // ₦600,000
		},
		DeclaredBases: map[string]taxengine.Kobo{},
	}

	facts.YearOfAssessment = 2026
	a2026, _, err := taxengine.Compute(facts, rs2026)
	if err != nil {
		t.Fatalf("Compute 2026: %v", err)
	}
	if a2026.TaxPayableKobo != 0 {
		t.Errorf("2026 first-band 0%%: want ₦0 tax, got %s", a2026.TaxPayableKobo.Format())
	}

	facts.YearOfAssessment = 1
	a1, _, err := taxengine.Compute(facts, rs1)
	if err != nil {
		t.Fatalf("Compute 1: %v", err)
	}
	// 5% × 60,000,000 kobo = 3,000,000 kobo = ₦30,000
	if a1.TaxPayableKobo != 3_000_000 {
		t.Errorf("fixture first-band 5%%: want ₦30,000 tax (3,000,000 kobo), got %s", a1.TaxPayableKobo.Format())
	}
}

// ---------------------------------------------------------------------------
// Integration tests (real files from the repo)
// ---------------------------------------------------------------------------

// TestRegistry_LoadsRealRuleSets loads the actual files from rules/ and
// testdata/rulesets/ and verifies Get("NG", 2026) returns a non-nil rule set.
func TestRegistry_LoadsRealRuleSets(t *testing.T) {
	reg := loadRepoRegistry(t)
	rs, err := reg.Get("NG", 2026)
	if err != nil {
		t.Fatalf("Get NG/2026: %v", err)
	}
	if rs.Jurisdiction != "NG" {
		t.Errorf("jurisdiction: want NG, got %s", rs.Jurisdiction)
	}
}

// TestRegistry_TestFixtureHasDifferentFirstBand verifies that the synthetic
// test fixture rule set (year_of_assessment: 1) has a non-zero first band rate,
// confirming it will produce different output from ng-2026 for the same income.
func TestRegistry_TestFixtureHasDifferentFirstBand(t *testing.T) {
	reg := loadRepoRegistry(t)
	rs1, err := reg.Get("NG", 1)
	if err != nil {
		t.Fatalf("Get NG/1: %v — is testdata/rulesets/ng-test-fixture.yaml present?", err)
	}
	if len(rs1.Bands) == 0 {
		t.Fatal("no bands in test fixture")
	}
	if rs1.Bands[0].RateBP == 0 {
		t.Error("test fixture first band should have a non-zero rate to prove data-driven dispatch")
	}
}

func loadRepoRegistry(t *testing.T) *rulesregistry.Registry {
	t.Helper()
	root := repoRoot()
	dirs := []string{
		filepath.Join(root, "rules"),
		filepath.Join(root, "testdata", "rulesets"),
	}
	var rawFiles [][]byte
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
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
