package ingestion_test

import (
	"strings"
	"testing"

	"taxsoftware/internal/ingestion"
	"taxsoftware/taxengine"
)

// ---------------------------------------------------------------------------
// CSVIngester tests
// ---------------------------------------------------------------------------

var defaultCSV = `Source,Gross,PAYE,Pension,NHF
Acme Ltd,9000000,1600000,720000,135000
Beta Corp,3000000,300000,240000,45000
Tech Inc,5000000,700000,400000,75000`

func TestCSVIngester_HappyPath(t *testing.T) {
	ing := &ingestion.CSVIngester{Columns: ingestion.DefaultH1Mapping()}
	records, err := ing.Extract([]byte(defaultCSV))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("want 3 records, got %d", len(records))
	}

	assertRecord(t, records[0], "Acme Ltd", 900_000_000, 160_000_000, 72_000_000, 13_500_000)
	assertRecord(t, records[1], "Beta Corp", 300_000_000, 30_000_000, 24_000_000, 4_500_000)
	assertRecord(t, records[2], "Tech Inc", 500_000_000, 70_000_000, 40_000_000, 7_500_000)
}

func assertRecord(t *testing.T, r ingestion.PayslipRecord, source string, gross, paye, pension, nhf int64) {
	t.Helper()
	if r.SourceRef != source {
		t.Errorf("SourceRef: want %q, got %q", source, r.SourceRef)
	}
	if int64(r.GrossKobo) != gross {
		t.Errorf("%s GrossKobo: want %d, got %d", source, gross, r.GrossKobo)
	}
	if int64(r.TaxDeductedKobo) != paye {
		t.Errorf("%s TaxDeductedKobo: want %d, got %d", source, paye, r.TaxDeductedKobo)
	}
	if int64(r.PensionKobo) != pension {
		t.Errorf("%s PensionKobo: want %d, got %d", source, pension, r.PensionKobo)
	}
	if int64(r.NHFKobo) != nhf {
		t.Errorf("%s NHFKobo: want %d, got %d", source, nhf, r.NHFKobo)
	}
}

func TestCSVIngester_QuarantinesBadRow(t *testing.T) {
	csv := `Source,Gross,PAYE,Pension,NHF
Acme Ltd,9000000,1600000,720000,135000
Bad Corp,INVALID,300000,240000,45000
Tech Inc,5000000,700000,400000,75000`

	ing := &ingestion.CSVIngester{Columns: ingestion.DefaultH1Mapping()}
	records, err := ing.Extract([]byte(csv))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("want 3 records (including quarantined), got %d", len(records))
	}

	// Bad Corp record must be quarantined: GrossKobo == 0 and Notes non-empty.
	bad := records[1]
	if !bad.IsQuarantined() {
		t.Errorf("Bad Corp: expected quarantine (GrossKobo=0 and Notes set); got GrossKobo=%d Notes=%q",
			bad.GrossKobo, bad.Notes)
	}
	if !strings.Contains(bad.Notes, "gross_kobo") {
		t.Errorf("Notes should mention gross_kobo field; got %q", bad.Notes)
	}

	// Good rows must parse correctly.
	if records[0].GrossKobo != 900_000_000 {
		t.Errorf("Acme GrossKobo: want 900000000, got %d", records[0].GrossKobo)
	}
	if records[2].GrossKobo != 500_000_000 {
		t.Errorf("Tech Inc GrossKobo: want 500000000, got %d", records[2].GrossKobo)
	}
}

func TestCSVIngester_SkipsHeaderRows(t *testing.T) {
	csv := `Report: LIRS H1 2026
Source,Gross,PAYE,Pension,NHF
Acme Ltd,9000000,1600000,720000,135000`

	m := ingestion.DefaultH1Mapping()
	m.SkipRows = 2
	ing := &ingestion.CSVIngester{Columns: m}
	records, err := ing.Extract([]byte(csv))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("want 1 record after skipping 2 header rows, got %d", len(records))
	}
	if records[0].SourceRef != "Acme Ltd" {
		t.Errorf("SourceRef: want Acme Ltd, got %q", records[0].SourceRef)
	}
}

func TestCSVIngester_NoFloat64(t *testing.T) {
	// Amounts that would lose precision if converted through float64.
	// 100000001 naira = 10000000100 kobo. float64 would round this.
	csv := `Source,Gross,PAYE,Pension,NHF
Precise,100000001,0,0,0`

	ing := &ingestion.CSVIngester{Columns: ingestion.DefaultH1Mapping()}
	records, err := ing.Extract([]byte(csv))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// ₦100,000,001 = 10,000,000,100 kobo
	want := taxengine.Kobo(100_000_001 * 100)
	if records[0].GrossKobo != want {
		t.Errorf("precision: want %d kobo, got %d", want, records[0].GrossKobo)
	}
}

func TestCSVIngester_CommaThousandSeparators(t *testing.T) {
	csv := `Source,Gross,PAYE,Pension,NHF
Acme Ltd,"9,000,000","1,600,000",720000,135000`

	ing := &ingestion.CSVIngester{Columns: ingestion.DefaultH1Mapping()}
	records, err := ing.Extract([]byte(csv))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if records[0].GrossKobo != 900_000_000 {
		t.Errorf("comma-separated: want 900000000, got %d", records[0].GrossKobo)
	}
}

func TestCSVIngester_MalformedCSVError(t *testing.T) {
	// A CSV with mismatched quotes is malformed at the format level.
	malformed := `"unclosed quote`
	ing := &ingestion.CSVIngester{Columns: ingestion.DefaultH1Mapping()}
	_, err := ing.Extract([]byte(malformed))
	if err == nil {
		t.Fatal("expected error for malformed CSV, got nil")
	}
}

// ---------------------------------------------------------------------------
// AggregateBySource tests
// ---------------------------------------------------------------------------

func TestAggregateBySource_SameSource(t *testing.T) {
	records := []ingestion.PayslipRecord{
		{SourceRef: "Acme Ltd", GrossKobo: 300_000_000, TaxDeductedKobo: 30_000_000},
		{SourceRef: "Acme Ltd", GrossKobo: 400_000_000, TaxDeductedKobo: 50_000_000},
		{SourceRef: "Acme Ltd", GrossKobo: 200_000_000, TaxDeductedKobo: 20_000_000},
	}
	out := ingestion.AggregateBySource(records)
	if len(out) != 1 {
		t.Fatalf("want 1 aggregate, got %d", len(out))
	}
	if out[0].GrossKobo != 900_000_000 {
		t.Errorf("GrossKobo: want 900000000, got %d", out[0].GrossKobo)
	}
	if out[0].TaxDeductedKobo != 100_000_000 {
		t.Errorf("TaxDeductedKobo: want 100000000, got %d", out[0].TaxDeductedKobo)
	}
}

func TestAggregateBySource_DifferentSources(t *testing.T) {
	records := []ingestion.PayslipRecord{
		{SourceRef: "Acme Ltd", GrossKobo: 300_000_000},
		{SourceRef: "Acme Ltd", GrossKobo: 400_000_000},
		{SourceRef: "Beta Corp", GrossKobo: 200_000_000},
	}
	out := ingestion.AggregateBySource(records)
	if len(out) != 2 {
		t.Fatalf("want 2 aggregates (Acme + Beta), got %d", len(out))
	}
	bySource := map[string]taxengine.Kobo{}
	for _, r := range out {
		bySource[r.SourceRef] = r.GrossKobo
	}
	if bySource["Acme Ltd"] != 700_000_000 {
		t.Errorf("Acme gross: want 700000000, got %d", bySource["Acme Ltd"])
	}
	if bySource["Beta Corp"] != 200_000_000 {
		t.Errorf("Beta gross: want 200000000, got %d", bySource["Beta Corp"])
	}
}

func TestAggregateBySource_MidYearJobChange(t *testing.T) {
	// 5 months at Acme (₦3m gross each) + 7 months at Beta (₦2m gross each).
	var records []ingestion.PayslipRecord
	for i := 0; i < 5; i++ {
		records = append(records, ingestion.PayslipRecord{SourceRef: "Acme Ltd", GrossKobo: 300_000_000})
	}
	for i := 0; i < 7; i++ {
		records = append(records, ingestion.PayslipRecord{SourceRef: "Beta Corp", GrossKobo: 200_000_000})
	}

	out := ingestion.AggregateBySource(records)
	if len(out) != 2 {
		t.Fatalf("want 2 aggregates, got %d", len(out))
	}
	bySource := map[string]taxengine.Kobo{}
	for _, r := range out {
		bySource[r.SourceRef] = r.GrossKobo
	}
	// Acme: 5 × ₦3,000,000 = ₦15,000,000 = 1,500,000,000 kobo
	if bySource["Acme Ltd"] != 1_500_000_000 {
		t.Errorf("Acme: want 1500000000, got %d", bySource["Acme Ltd"])
	}
	// Beta: 7 × ₦2,000,000 = ₦14,000,000 = 1,400,000,000 kobo
	if bySource["Beta Corp"] != 1_400_000_000 {
		t.Errorf("Beta: want 1400000000, got %d", bySource["Beta Corp"])
	}
}

func TestAggregateBySource_QuarantinedExcluded(t *testing.T) {
	records := []ingestion.PayslipRecord{
		{SourceRef: "Acme Ltd", GrossKobo: 900_000_000},
		{SourceRef: "Bad Corp", GrossKobo: 0, Notes: "row 2: gross_kobo: not a valid naira amount"},
	}
	out := ingestion.AggregateBySource(records)
	// Quarantined record returned unchanged; Acme aggregate separate.
	if len(out) != 2 {
		t.Fatalf("want 2 (1 aggregate + 1 quarantined), got %d", len(out))
	}
	var foundQuarantined bool
	for _, r := range out {
		if r.IsQuarantined() {
			foundQuarantined = true
		}
	}
	if !foundQuarantined {
		t.Error("quarantined record should appear in output unchanged")
	}
}
