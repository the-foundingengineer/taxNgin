package employer_test

import (
	"testing"
	"time"

	"taxsoftware/internal/employer"
	"taxsoftware/taxengine"
)

func signedEmp() employer.Employer {
	ts := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	return employer.Employer{
		ID:          "EMP-1",
		LegalName:   "Acme Ltd",
		DPASignedAt: &ts,
	}
}

func cleanImport() employer.PayrollImport {
	return employer.PayrollImport{
		ID:               "IMP-1",
		EmployerID:       "EMP-1",
		YearOfAssessment: 2026,
		Records: []employer.PayrollRecord{
			{ID: "REC-1", GrossKobo: taxengine.Kobo(500_000_000), PAYEKobo: taxengine.Kobo(50_000_000)},
		},
	}
}

func hasIssueField(issues []employer.ImportIssue, field string) bool {
	for _, iss := range issues {
		if iss.Field == field {
			return true
		}
	}
	return false
}

func hasIssueRecord(issues []employer.ImportIssue, recID string) bool {
	for _, iss := range issues {
		if iss.RecordID == recID {
			return true
		}
	}
	return false
}

func TestValidateImport_Clean(t *testing.T) {
	issues := employer.ValidateImport(cleanImport(), signedEmp())
	if len(issues) != 0 {
		t.Errorf("expected no issues for a clean import, got: %v", issues)
	}
}

func TestValidateImport_NoDPA(t *testing.T) {
	emp := signedEmp()
	emp.DPASignedAt = nil
	issues := employer.ValidateImport(cleanImport(), emp)
	if !hasIssueField(issues, "data_processing_agreement") {
		t.Errorf("expected DPA issue; got: %v", issues)
	}
}

func TestValidateImport_ZeroGross(t *testing.T) {
	imp := cleanImport()
	imp.Records[0].GrossKobo = 0
	issues := employer.ValidateImport(imp, signedEmp())
	if !hasIssueRecord(issues, "REC-1") {
		t.Errorf("expected issue for REC-1 (zero gross); got: %v", issues)
	}
	if !hasIssueField(issues, "gross_kobo") {
		t.Errorf("expected issue field gross_kobo; got: %v", issues)
	}
}

func TestValidateImport_NegativeAmount(t *testing.T) {
	imp := cleanImport()
	imp.Records[0].PAYEKobo = taxengine.Kobo(-1)
	issues := employer.ValidateImport(imp, signedEmp())
	if !hasIssueRecord(issues, "REC-1") {
		t.Errorf("expected issue for REC-1 (negative PAYE); got: %v", issues)
	}
	if !hasIssueField(issues, "paye_kobo") {
		t.Errorf("expected issue field paye_kobo; got: %v", issues)
	}
}

func TestValidateImport_MissingYear(t *testing.T) {
	imp := cleanImport()
	imp.YearOfAssessment = 0
	issues := employer.ValidateImport(imp, signedEmp())
	if !hasIssueField(issues, "year_of_assessment") {
		t.Errorf("expected year_of_assessment issue; got: %v", issues)
	}
}

func TestValidateImport_MultipleIssues(t *testing.T) {
	emp := signedEmp()
	emp.DPASignedAt = nil
	imp := cleanImport()
	imp.Records[0].GrossKobo = 0
	issues := employer.ValidateImport(imp, emp)
	if len(issues) < 2 {
		t.Errorf("expected at least 2 issues (DPA + zero gross), got %d: %v", len(issues), issues)
	}
}
