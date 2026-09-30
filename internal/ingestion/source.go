// Package ingestion defines the IncomeSource port through which all income
// evidence enters the return lifecycle. Adapters (CSV, PDF, open banking)
// implement the port and normalise their native formats into PayslipRecord.
// The return-event bridge then converts PayslipRecords into EvPayrollProposed
// or EvIncomeDeclared events.
//
// Design rule: Extract never performs I/O. Data is pre-loaded by the caller
// and passed in as bytes. This keeps adapters testable without network access
// and keeps replay of existing events pure.
package ingestion

import (
	"errors"
	"time"

	"taxsoftware/taxengine"
)

// ErrNotImplemented is returned by stub adapters that have not yet been wired
// to an external service.
var ErrNotImplemented = errors.New("ingestion: adapter not implemented")

// PayslipRecord is the normalised output of any income source adapter. All
// adapters — CSV, PDF, bank statement, open banking — produce this. The
// caller converts it to return events.
//
// All Kobo fields must be non-negative. Negative values indicate a parsing
// error; callers should treat them as quarantined and inspect Notes.
type PayslipRecord struct {
	SourceRef       string         // employer name, bank name, or trading platform
	PeriodStart     time.Time      // first day of the pay period
	PeriodEnd       time.Time      // last day of the pay period
	GrossKobo       taxengine.Kobo
	TaxDeductedKobo taxengine.Kobo // PAYE or withholding tax deducted at source
	PensionKobo     taxengine.Kobo
	NHFKobo         taxengine.Kobo
	NHISKobo        taxengine.Kobo
	// Notes carries warnings and quarantine reasons. Non-empty means the
	// caller should review this record before accepting it into a return.
	Notes string
}

// IsQuarantined reports whether this record has been flagged for review.
// Quarantined records have GrossKobo == 0 and a non-empty Notes field.
// They are returned from Extract rather than dropped so the caller can
// surface the problem to the user.
func (r PayslipRecord) IsQuarantined() bool {
	return r.GrossKobo == 0 && r.Notes != ""
}

// IncomeSource is the port that all income adapters implement. Each adapter
// converts its native format into PayslipRecords and must not perform I/O.
type IncomeSource interface {
	// Name is a human-readable identifier for log messages and UI attribution.
	Name() string
	// Extract parses data and returns PayslipRecords. Individual bad rows are
	// quarantined (returned with IsQuarantined()==true), not errors. Extract
	// returns a non-nil error only when the entire file cannot be parsed.
	Extract(data []byte) ([]PayslipRecord, error)
}

// DocumentExtractor extends IncomeSource for document-based formats (PDF,
// Excel). MimeType lets the dispatcher route the document before calling Extract.
type DocumentExtractor interface {
	IncomeSource
	MimeType() string
}

// OpenBankingAdapter extends IncomeSource with a method to fetch transaction
// data for a specific period. Implement when NIBSS Open Banking access is
// approved.
type OpenBankingAdapter interface {
	IncomeSource
	// FetchTransactions fetches transaction data for the given consent-gated
	// account reference. The accountRef must never be stored.
	FetchTransactions(accountRef string, from, to time.Time) ([]PayslipRecord, error)
}

// ---------------------------------------------------------------------------
// Stubs — injection points for real adapters when external services are ready
// ---------------------------------------------------------------------------

// StubPDFExtractor satisfies DocumentExtractor. Replace with a real
// implementation when a document processing service is contracted.
type StubPDFExtractor struct{}

func (s StubPDFExtractor) Name() string                         { return "pdf-payslip-stub" }
func (s StubPDFExtractor) MimeType() string                     { return "application/pdf" }
func (s StubPDFExtractor) Extract(_ []byte) ([]PayslipRecord, error) {
	return nil, ErrNotImplemented
}

// StubOpenBankingAdapter satisfies OpenBankingAdapter. Replace with a real
// implementation when NIBSS Open Banking approval is received.
type StubOpenBankingAdapter struct{}

func (s StubOpenBankingAdapter) Name() string { return "open-banking-stub" }
func (s StubOpenBankingAdapter) Extract(_ []byte) ([]PayslipRecord, error) {
	return nil, ErrNotImplemented
}
func (s StubOpenBankingAdapter) FetchTransactions(_ string, _, _ time.Time) ([]PayslipRecord, error) {
	return nil, ErrNotImplemented
}
