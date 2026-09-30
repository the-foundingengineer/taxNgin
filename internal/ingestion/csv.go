package ingestion

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"taxsoftware/taxengine"
)

// ColumnMapping maps zero-based column indices to the fields of PayslipRecord.
// Use -1 for a field that is absent from the file.
type ColumnMapping struct {
	SourceName  int // employer or income source name
	GrossKobo   int // annual gross pay (naira)
	PAYEKobo    int // PAYE deducted (naira)
	PensionKobo int
	NHFKobo     int
	NHISKobo    int // -1 if column not present
	// SkipRows is the number of header rows before data begins.
	SkipRows int
}

// DefaultH1Mapping returns the column mapping for a standard LIRS H1 payroll
// schedule CSV. Row 0 is a header and skipped. Columns:
//
//	0: Source / employer name
//	1: Annual gross pay (naira, integer or naira.kobo)
//	2: PAYE deducted (naira)
//	3: Pension contributions (naira)
//	4: NHF contributions (naira)
//	5: NHIS contributions (naira, optional)
func DefaultH1Mapping() ColumnMapping {
	return ColumnMapping{
		SkipRows:    1,
		SourceName:  0,
		GrossKobo:   1,
		PAYEKobo:    2,
		PensionKobo: 3,
		NHFKobo:     4,
		NHISKobo:    5,
	}
}

// CSVIngester implements IncomeSource for CSV payroll files. It is the primary
// path for the employer bulk import pipeline and the LIRS H1 schedule.
type CSVIngester struct {
	Columns ColumnMapping
}

func (c *CSVIngester) Name() string { return "csv-ingester" }

// Extract parses data as CSV and returns one PayslipRecord per data row.
// Rows with unparseable amounts are quarantined (GrossKobo=0, Notes set)
// rather than causing an error. Only a completely malformed CSV returns an error.
func (c *CSVIngester) Extract(data []byte) ([]PayslipRecord, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1 // allow ragged rows
	all, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ingestion: parse CSV: %w", err)
	}

	var records []PayslipRecord
	for rowIdx, row := range all {
		if rowIdx < c.Columns.SkipRows {
			continue
		}
		rec := c.parseRow(row, rowIdx)
		records = append(records, rec)
	}
	return records, nil
}

func (c *CSVIngester) parseRow(row []string, rowIdx int) PayslipRecord {
	var rec PayslipRecord
	var quarantine []string

	rec.SourceRef = c.strCol(row, c.Columns.SourceName)

	var err error

	rec.GrossKobo, err = c.koboCol(row, c.Columns.GrossKobo)
	if err != nil {
		quarantine = append(quarantine, fmt.Sprintf("row %d: gross_kobo: %v", rowIdx, err))
	}

	rec.TaxDeductedKobo, err = c.koboCol(row, c.Columns.PAYEKobo)
	if err != nil {
		quarantine = append(quarantine, fmt.Sprintf("row %d: paye_kobo: %v", rowIdx, err))
	}

	rec.PensionKobo, err = c.koboCol(row, c.Columns.PensionKobo)
	if err != nil {
		quarantine = append(quarantine, fmt.Sprintf("row %d: pension_kobo: %v", rowIdx, err))
	}

	rec.NHFKobo, err = c.koboCol(row, c.Columns.NHFKobo)
	if err != nil {
		quarantine = append(quarantine, fmt.Sprintf("row %d: nhf_kobo: %v", rowIdx, err))
	}

	if c.Columns.NHISKobo >= 0 {
		rec.NHISKobo, err = c.koboCol(row, c.Columns.NHISKobo)
		if err != nil {
			// NHIS column is optional — log but do not quarantine for this alone.
			rec.NHISKobo = 0
		}
	}

	if len(quarantine) > 0 {
		rec.GrossKobo = 0 // canonical quarantine signal
		rec.Notes = strings.Join(quarantine, "; ")
	}
	return rec
}

func (c *CSVIngester) strCol(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func (c *CSVIngester) koboCol(row []string, idx int) (taxengine.Kobo, error) {
	if idx < 0 {
		return 0, nil
	}
	if idx >= len(row) {
		return 0, fmt.Errorf("column %d missing (row has %d columns)", idx, len(row))
	}
	return parseKobo(strings.TrimSpace(row[idx]))
}

// parseKobo parses a naira string (integer or naira.kobo with ≤2 decimal places)
// into kobo without ever using float64. Comma separators are stripped.
// An empty string returns 0, not an error — absent amounts are treated as zero.
func parseKobo(s string) (taxengine.Kobo, error) {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}

	wholeStr, fracStr, hasDot := strings.Cut(s, ".")

	var whole int64
	for _, c := range wholeStr {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a valid naira amount: %q", s)
		}
		whole = whole*10 + int64(c-'0')
	}

	var frac int64
	if hasDot {
		if len(fracStr) > 2 {
			return 0, fmt.Errorf("more than two decimal places in %q", s)
		}
		for len(fracStr) < 2 {
			fracStr += "0"
		}
		for _, c := range fracStr {
			if c < '0' || c > '9' {
				return 0, fmt.Errorf("not a valid naira amount: %q", s)
			}
			frac = frac*10 + int64(c-'0')
		}
	}

	k := whole*100 + frac
	if neg {
		k = -k
	}
	return taxengine.Kobo(k), nil
}

// AggregateBySource collapses multiple PayslipRecords from the same SourceRef
// into a single annual record by summing all Kobo fields. Records from
// different sources (mid-year job change) remain separate.
//
// Quarantined records (IsQuarantined()==true) are excluded from aggregation
// and returned unchanged so the caller can surface them to the user.
func AggregateBySource(records []PayslipRecord) []PayslipRecord {
	type key = string
	order := []key{}
	buckets := map[key]*PayslipRecord{}

	var quarantined []PayslipRecord

	for _, r := range records {
		if r.IsQuarantined() {
			quarantined = append(quarantined, r)
			continue
		}
		k := r.SourceRef
		if _, seen := buckets[k]; !seen {
			order = append(order, k)
			copy := r
			copy.GrossKobo = 0
			copy.TaxDeductedKobo = 0
			copy.PensionKobo = 0
			copy.NHFKobo = 0
			copy.NHISKobo = 0
			buckets[k] = &copy
		}
		b := buckets[k]
		b.GrossKobo += r.GrossKobo
		b.TaxDeductedKobo += r.TaxDeductedKobo
		b.PensionKobo += r.PensionKobo
		b.NHFKobo += r.NHFKobo
		b.NHISKobo += r.NHISKobo
	}

	out := make([]PayslipRecord, 0, len(order)+len(quarantined))
	for _, k := range order {
		out = append(out, *buckets[k])
	}
	return append(out, quarantined...)
}
