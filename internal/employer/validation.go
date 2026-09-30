package employer

import "fmt"

// ImportIssue describes one blocking problem found during import validation.
// RecordID is empty for batch-level issues (missing DPA, empty employer ID,
// missing year of assessment).
type ImportIssue struct {
	RecordID string
	Field    string
	Message  string
}

// ValidateImport checks that imp is ready to be published. It returns every
// blocking issue found; an empty slice means the import is clean.
//
// ValidateImport is pure: no I/O, no clock. The DPA gate mirrors the rule-set
// sign-off gate — both are cheap now and structurally impossible to retrofit
// once real data is in the system.
func ValidateImport(imp PayrollImport, emp Employer) []ImportIssue {
	var issues []ImportIssue

	// Batch-level: NDPA Data Processing Agreement gate.
	if emp.DPASignedAt == nil {
		issues = append(issues, ImportIssue{
			Field: "data_processing_agreement",
			Message: "Employer has not signed the Data Processing Agreement; " +
				"upload is not permitted under the Nigeria Data Protection Act",
		})
	}

	if emp.ID == "" {
		issues = append(issues, ImportIssue{
			Field:   "employer_id",
			Message: "Employer ID must not be empty",
		})
	}

	if imp.YearOfAssessment == 0 {
		issues = append(issues, ImportIssue{
			Field:   "year_of_assessment",
			Message: "Year of assessment must not be zero",
		})
	}

	// Record-level checks.
	for _, rec := range imp.Records {
		if rec.GrossKobo == 0 {
			issues = append(issues, ImportIssue{
				RecordID: rec.ID,
				Field:    "gross_kobo",
				Message:  fmt.Sprintf("record %s: gross must be greater than zero", rec.ID),
			})
		}
		type koboCheck struct {
			field string
			value interface{ isNeg() bool }
		}
		negFields := []struct {
			field string
			kobo  int64
		}{
			{"gross_kobo", int64(rec.GrossKobo)},
			{"paye_kobo", int64(rec.PAYEKobo)},
			{"pension_kobo", int64(rec.PensionKobo)},
			{"nhf_kobo", int64(rec.NHFKobo)},
			{"nhis_kobo", int64(rec.NHISKobo)},
			{"basic_kobo", int64(rec.BasicKobo)},
			{"housing_kobo", int64(rec.HousingKobo)},
			{"transport_kobo", int64(rec.TransportKobo)},
		}
		for _, f := range negFields {
			if f.kobo < 0 {
				issues = append(issues, ImportIssue{
					RecordID: rec.ID,
					Field:    f.field,
					Message:  fmt.Sprintf("record %s: %s must not be negative", rec.ID, f.field),
				})
			}
		}
	}

	return issues
}
