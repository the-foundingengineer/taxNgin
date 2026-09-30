// Package employer models the employer side of the pre-fill channel.
//
// Under Shape A the taxpayer declares their own income and carries the
// liability. An employer's payroll data is EVIDENCE, not a declaration. It
// becomes a declaration only when the taxpayer accepts it (EvPayrollAccepted
// with a TAXPAYER actor). This package models the input side: employer
// identity, import batches, and individual records.
//
// NDPA note — when an employer uploads payroll they are the data controller
// and this system is the processor. Once an employee creates an account and
// starts a return, this system becomes a controller for that person's data.
// A Data Processing Agreement with every employer is required before the first
// upload. ValidateImport enforces this gate. Do not remove or bypass it.
package employer

import (
	"time"

	"taxsoftware/taxengine"
)

// Employer is the aggregate root for a company that submits payroll data.
type Employer struct {
	ID          string
	CACNumber   string   // Corporate Affairs Commission registration number
	TaxID       string   // employer's tax identifier (FIRS TIN)
	LegalName   string
	Jurisdictions []string // states where this employer operates

	// DPASignedAt is nil until the employer has signed the Data Processing
	// Agreement required by the NDPA before any upload. ValidateImport refuses
	// to approve a batch when this is nil.
	DPASignedAt *time.Time
}

// EmploymentRelationship is the join between an employer and a taxpayer.
// TaxpayerID is empty until an employee claims a payroll record; until then
// the relationship exists on the employer side only.
type EmploymentRelationship struct {
	ID              string
	EmployerID      string
	TaxpayerID      string     // empty until claimed
	EmployeeRef     string     // employer's own staff number
	StartDate       time.Time
	EndDate         *time.Time // nil = current employment
	// MatchConfidence records how the link between employee and taxpayer was
	// established. Overlapping employment periods are legitimate (two concurrent
	// jobs) but are usually a date-entry error; flag them.
	MatchConfidence string // "NIN_EXACT" | "TAXID_EXACT" | "EMAIL" | "MANUAL"
}

// ImportStatus is the lifecycle state of a payroll batch.
type ImportStatus string

const (
	ImportReceived  ImportStatus = "RECEIVED"
	ImportParsed    ImportStatus = "PARSED"
	ImportValidated ImportStatus = "VALIDATED"
	ImportPublished ImportStatus = "PUBLISHED"
)

// PayrollImport is one batch of payroll records from one employer for one
// year of assessment. A batch must be fully VALIDATED before it can be
// PUBLISHED, and an employer must have a signed DPA before any batch is
// accepted.
type PayrollImport struct {
	ID               string
	EmployerID       string
	YearOfAssessment int
	Source           string       // "CSV" | "H1_EXPORT" | "API"
	UploadedBy       string       // staff or API credential ID
	UploadedAt       time.Time
	Status           ImportStatus
	Records          []PayrollRecord
}

// PayrollRecord is one employee's payroll data for the year. It is EVIDENCE,
// not a declaration. It becomes a declaration only when the taxpayer accepts it
// via EvPayrollAccepted (actored by TAXPAYER).
//
// Identifiers are stored as HMAC bytes (deterministic, keyed) so a leak of
// the payroll_records table yields salaries without identities. See hmac.go.
//
// BasicKobo, HousingKobo, and TransportKobo are stored separately so the
// pension-base ambiguity (8% of gross vs. basic+housing+transport) can be
// resolved later without going back to the employer for a re-upload.
type PayrollRecord struct {
	ID       string
	ImportID string

	EmployeeRef string

	// HMAC bytes — deterministic under a system-wide key. Used for matching
	// only; never displayed or logged. nil = identifier not provided by employer.
	NINHMACSuffix   []byte
	TaxIDHMACSuffix []byte
	EmailHMACSuffix []byte

	GrossKobo    taxengine.Kobo
	PAYEKobo     taxengine.Kobo
	PensionKobo  taxengine.Kobo
	NHFKobo      taxengine.Kobo
	NHISKobo     taxengine.Kobo

	// Components of the contested pension base.
	BasicKobo     taxengine.Kobo
	HousingKobo   taxengine.Kobo
	TransportKobo taxengine.Kobo

	// Populated once a taxpayer claims this record.
	ClaimedBy string
	ClaimedAt *time.Time
}
