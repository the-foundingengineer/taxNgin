package bridge

import (
	"taxsoftware/internal/returns"
	"taxsoftware/taxengine"
)

// ProjectionToFacts bridges the event-sourced Projection state into the
// pure data struct required by the taxengine.
//
// Mappings and assumptions:
// - IsResident: Defaulted to true, as non-resident handling is out of scope
//   for v1 and projection does not capture residency.
// - MonthsEmployed: Defaulted to 12 (full year) since the projection
//   basis period is not explicitly converted to months here yet.
// - EvidenceIDs: Dropped (lossy mapping). Evidence verification is isolated
//   in the validation layer (internal/validate). The engine only needs to know
//   if evidence _exists_ (HasEvidence bool) to permit/restrict relief claims.
// - Credits: Not currently populated from projection. The event log does not
//   capture EvCreditClaimed events yet (PAYE credits from payroll records would
//   eventually flow here).
func ProjectionToFacts(p *returns.Projection) taxengine.TaxFacts {
	facts := taxengine.TaxFacts{
		YearOfAssessment: p.YearOfAssessment,
		Jurisdiction:     p.Jurisdiction,
		IsResident:       true,
		MonthsEmployed:   12,
		DeclaredBases:    make(map[string]taxengine.Kobo, len(p.Bases)),
	}

	for _, inc := range p.Income {
		facts.Income = append(facts.Income, taxengine.IncomeItem{
			Category:   taxengine.IncomeCategory(inc.Category),
			AmountKobo: taxengine.Kobo(inc.AmountKobo),
			SourceRef:  inc.SourceRef,
			Label:      inc.Label,
		})
	}

	for _, rel := range p.Reliefs {
		facts.Reliefs = append(facts.Reliefs, taxengine.ReliefClaim{
			Code:        rel.Code,
			AmountKobo:  taxengine.Kobo(rel.AmountKobo),
			HasEvidence: len(rel.EvidenceIDs) > 0,
		})
	}

	for k, v := range p.Bases {
		facts.DeclaredBases[k] = taxengine.Kobo(v)
	}

	return facts
}
