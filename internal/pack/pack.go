// Package pack generates the submission pack: the transcription script a
// taxpayer uses when filing their return on the LIRS eTax portal under Shape A.
//
// The pack maps a Projection's fields to the portal's own labels, in the
// portal's own order. It is pure: no I/O, no clock, no network. The caller
// supplies the Projection and the PortalMap; Generate produces a Pack.
package pack

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"taxsoftware/internal/portalmap"
	"taxsoftware/internal/returns"
	"taxsoftware/taxengine"
)

// PackField is one data-entry instruction: what to type and where.
type PackField struct {
	PortalLabel string
	Value       string // formatted value to enter in the portal
	Note        string // human note from the portal map or resolver
	Upload      string // "Bring: X" when the field requires a document
	Missing     bool   // true when the projection has no value for this field
}

// PackRow wraps one or more PackFields. In practice each row holds one field;
// the slice exists so future rendering can group related fields visually.
type PackRow struct {
	Fields []PackField
}

// UnmappedIncomeItem is an income category that the portal has no field for.
// The pack surfaces it explicitly so the user knows to ask the practitioner.
type UnmappedIncomeItem struct {
	Category string
	Amount   taxengine.Kobo
	Note     string // from the portal map's unmapped_income.note
}

// PreconditionResult reports whether a portal precondition is met.
// BlocksPack / BlocksTCC come from the portal map; Met comes from the caller.
// Generate reports precondition state but never aborts on an unmet condition —
// the caller decides whether to show a blocking warning before presenting the pack.
type PreconditionResult struct {
	ID         string
	Label      string
	Detail     string
	BlocksPack bool
	BlocksTCC  bool
	Met        bool
}

// PackPage is one portal screen, with its fields resolved to concrete values.
type PackPage struct {
	ID    string
	Title string
	Note  string
	Rows  []PackRow
}

// Pack is the complete transcription script for one return filing.
type Pack struct {
	ReturnID         string
	YearOfAssessment int
	PortalMapVersion string
	Pages            []PackPage
	UnmappedIncome   []UnmappedIncomeItem
	Preconditions    []PreconditionResult
	// AssessmentTrace is the explanation of the assessment, included as an
	// appendix so the user has the reasoning next to the numbers.
	AssessmentTrace []taxengine.Step
}

// GenerateInput bundles everything Generate needs.
type GenerateInput struct {
	Projection *returns.Projection
	PortalMap  portalmap.PortalMap
	// PreconditionsMet maps a precondition ID (e.g. "PC01") to whether the
	// caller has confirmed it is met. Absent keys default to false (not met).
	PreconditionsMet map[string]bool
}

// Generate produces a Pack from the projection and the portal map.
//
// It returns an error only if the projection is in a state too early for a
// pack (DRAFT or VALIDATED). All other conditions — unmet preconditions,
// missing accommodation, zero amounts — are reported inside the Pack without
// returning an error, because the caller must decide what to do.
func Generate(in GenerateInput) (Pack, error) {
	p := in.Projection

	switch p.State {
	case returns.StateDraft, returns.StateValidated:
		return Pack{}, fmt.Errorf(
			"pack: cannot generate pack for return in state %s; "+
				"the return must be at least %s", p.State, returns.StateSigned)
	}

	pack := Pack{
		ReturnID:         p.ReturnID,
		YearOfAssessment: p.YearOfAssessment,
		PortalMapVersion: in.PortalMap.Portal.MapVersion,
	}

	// Preconditions — report state, never abort.
	for _, pc := range in.PortalMap.Preconditions {
		pack.Preconditions = append(pack.Preconditions, PreconditionResult{
			ID:         pc.ID,
			Label:      pc.Label,
			Detail:     pc.Detail,
			BlocksPack: pc.BlocksPack,
			BlocksTCC:  pc.BlocksTCC,
			Met:        in.PreconditionsMet[pc.ID],
		})
	}

	// Pages.
	for _, page := range in.PortalMap.Pages {
		pp := PackPage{ID: page.ID, Title: page.Title, Note: page.Note}
		for _, field := range page.Fields {
			if !evaluateConditional(field.ConditionalOn, p) {
				continue
			}
			pp.Rows = append(pp.Rows, resolveField(field, p)...)
		}
		pack.Pages = append(pack.Pages, pp)

		// Unmapped income — only pages that declare the spec carry one.
		if page.UnmappedIncome != nil {
			for _, cat := range page.UnmappedIncome.Categories {
				// GRATUITY is mapped on Page 3 via income.GRATUITY; do not
				// flag it here. See TASK-004 §6 for the portal map finding.
				if cat == "GRATUITY" {
					continue
				}
				total := sumIncomeByCategory(p, cat)
				if total > 0 {
					pack.UnmappedIncome = append(pack.UnmappedIncome, UnmappedIncomeItem{
						Category: cat,
						Amount:   total,
						Note:     page.UnmappedIncome.Note,
					})
				}
			}
		}
	}

	// Assessment trace — deserialise the raw JSON stored in the snapshot.
	if p.Assessment != nil && len(p.Assessment.Trace) > 0 {
		var steps []taxengine.Step
		if err := json.Unmarshal(p.Assessment.Trace, &steps); err == nil {
			pack.AssessmentTrace = steps
		}
	}

	return pack, nil
}

// ---------------------------------------------------------------------------
// Conditional evaluation
// ---------------------------------------------------------------------------

// evaluateConditional interprets a conditional_on expression from the portal
// map. Unknown expressions default to true (show the field) so that new
// conditions in an updated map degrade gracefully on an older generator.
func evaluateConditional(cond string, p *returns.Projection) bool {
	if cond == "" {
		return true
	}
	switch cond {
	case "accommodation.type == Rent":
		return p.AccommodationType == "Rent"
	}
	return true
}

// ---------------------------------------------------------------------------
// Source path resolver
// ---------------------------------------------------------------------------

// resolveField maps a PageField's source path to one or more PackRows.
// The design rule: every source path either resolves to a value (possibly ₦0)
// or sets Missing: true with an explanatory note. Nothing is silently dropped.
func resolveField(field portalmap.PageField, p *returns.Projection) []PackRow {
	src := field.Source

	// Repeatable EMPLOYMENT fields: income.EMPLOYMENT[].source_name / .amount
	if strings.HasPrefix(src, "income.EMPLOYMENT[].") {
		sub := strings.TrimPrefix(src, "income.EMPLOYMENT[].")
		return resolveEmploymentRepeatable(field, sub, p)
	}

	// Compound income: income.BUSINESS + income.PROFESSIONAL
	if strings.Contains(src, " + ") {
		return resolveCompound(field, p)
	}

	// Simple income category: income.CATEGORY
	if strings.HasPrefix(src, "income.") {
		cat := strings.TrimPrefix(src, "income.")
		return []PackRow{oneRow(field, formatKobo(sumIncomeByCategory(p, cat), field.Format), field.Note, field.Upload, false)}
	}

	// Relief: reliefs.CODE — absent means ₦0, not Missing.
	if strings.HasPrefix(src, "reliefs.") {
		code := strings.TrimPrefix(src, "reliefs.")
		var amount taxengine.Kobo
		if line, ok := p.Reliefs[code]; ok {
			amount = taxengine.Kobo(line.AmountKobo)
		}
		return []PackRow{oneRow(field, formatKobo(amount, field.Format), field.Note, field.Upload, false)}
	}

	// Declared base: declared_bases.KEY — absent means Missing.
	if strings.HasPrefix(src, "declared_bases.") {
		key := strings.TrimPrefix(src, "declared_bases.")
		if v, ok := p.Bases[key]; ok {
			return []PackRow{oneRow(field, formatKobo(taxengine.Kobo(v), field.Format), field.Note, field.Upload, false)}
		}
		return []PackRow{missingRow(field, "Not declared — emit EvBaseDeclared")}
	}

	// Accommodation fields.
	if strings.HasPrefix(src, "accommodation.") {
		sub := strings.TrimPrefix(src, "accommodation.")
		return resolveAccommodation(field, sub, p)
	}

	// Profile fields — not in the projection; always Missing.
	if strings.HasPrefix(src, "profile.") {
		return []PackRow{missingRow(field, "Enter from your eTax profile")}
	}

	// Unknown source path — surface it rather than silently dropping the field.
	return []PackRow{missingRow(field, fmt.Sprintf("source path %q not resolved by this generator version", src))}
}

// resolveEmploymentRepeatable expands EMPLOYMENT income lines into N rows,
// sorted by IncomeLine.ID for a stable, deterministic output order.
func resolveEmploymentRepeatable(field portalmap.PageField, sub string, p *returns.Projection) []PackRow {
	var lines []returns.IncomeLine
	for _, line := range p.Income {
		if line.Category == string(taxengine.IncEmployment) {
			lines = append(lines, line)
		}
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].ID < lines[j].ID })

	if len(lines) == 0 {
		return []PackRow{missingRow(field, "No employment income declared")}
	}

	var rows []PackRow
	for _, line := range lines {
		var value string
		switch sub {
		case "source_name":
			// Label carries the employer name (e.g. "Acme Ltd"). Fall back to
			// SourceRef (employer Tax ID) and then to a placeholder.
			value = line.Label
			if value == "" {
				value = line.SourceRef
			}
			if value == "" {
				value = "Unknown employer"
			}
		case "amount":
			value = formatKobo(taxengine.Kobo(line.AmountKobo), field.Format)
		default:
			value = ""
		}
		rows = append(rows, oneRow(field, value, field.Note, field.Upload, false))
	}
	return rows
}

// resolveCompound handles source expressions like "income.BUSINESS + income.PROFESSIONAL".
// It sums both categories and appends a note listing non-zero components so
// the user can check our arithmetic.
func resolveCompound(field portalmap.PageField, p *returns.Projection) []PackRow {
	parts := strings.Split(field.Source, " + ")
	var total taxengine.Kobo
	var components []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		cat := strings.TrimPrefix(part, "income.")
		sum := sumIncomeByCategory(p, cat)
		total += sum
		if sum > 0 {
			components = append(components, fmt.Sprintf("%s: %s", cat, sum.Format()))
		}
	}
	note := field.Note
	if len(components) > 0 {
		comp := strings.Join(components, ", ")
		if note != "" {
			note = note + " (" + comp + ")"
		} else {
			note = comp
		}
	}
	return []PackRow{oneRow(field, formatKobo(total, field.Format), note, field.Upload, false)}
}

// resolveAccommodation handles the accommodation.* family of source paths.
func resolveAccommodation(field portalmap.PageField, sub string, p *returns.Projection) []PackRow {
	switch sub {
	case "type":
		if p.AccommodationType == "" {
			return []PackRow{missingRow(field, "Declare accommodation via EvAccommodationDeclared")}
		}
		return []PackRow{oneRow(field, p.AccommodationType, field.Note, field.Upload, false)}

	case "ownership":
		if p.OwnershipType == "" {
			return []PackRow{missingRow(field, "Declare accommodation via EvAccommodationDeclared")}
		}
		return []PackRow{oneRow(field, p.OwnershipType, field.Note, field.Upload, false)}

	case "landlord":
		var parts []string
		if p.LandlordName != "" {
			parts = append(parts, p.LandlordName)
		}
		if p.LandlordAddress != "" {
			parts = append(parts, p.LandlordAddress)
		}
		if len(parts) == 0 {
			return []PackRow{missingRow(field, "Declare landlord details via EvAccommodationDeclared")}
		}
		return []PackRow{oneRow(field, strings.Join(parts, ", "), field.Note, field.Upload, false)}
	}

	return []PackRow{missingRow(field, fmt.Sprintf("unknown accommodation field %q", sub))}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func oneRow(field portalmap.PageField, value, note, upload string, missing bool) PackRow {
	return PackRow{Fields: []PackField{{
		PortalLabel: field.PortalLabel,
		Value:       value,
		Note:        note,
		Upload:      upload,
		Missing:     missing,
	}}}
}

func missingRow(field portalmap.PageField, note string) PackRow {
	return PackRow{Fields: []PackField{{
		PortalLabel: field.PortalLabel,
		Note:        note,
		Missing:     true,
	}}}
}

// sumIncomeByCategory sums all income lines whose Category matches the given
// string. Returns 0 if no matching lines exist — this is not Missing.
func sumIncomeByCategory(p *returns.Projection, category string) taxengine.Kobo {
	var total taxengine.Kobo
	for _, line := range p.Income {
		if line.Category == category {
			total += taxengine.Kobo(line.AmountKobo)
		}
	}
	return total
}

// formatKobo formats a Kobo value for display. The portal always wants
// naira with two decimal places, so format is effectively always naira_2dp.
func formatKobo(k taxengine.Kobo, _ string) string {
	return k.Format()
}
