// Package portalmap loads and represents a portal transcription map.
//
// A portal map is versioned data, exactly like a rule set: when LIRS reorders
// a page you publish a new map version, not a deploy. Packs generated against
// an old map remain reproducible because each PackIssued event records the
// map version used.
//
// The loader follows the same contract as taxengine.Load: the caller provides
// the raw bytes and an injected unmarshal function so this package never
// imports a YAML library directly.
package portalmap

import "fmt"

// PortalMap is the parsed representation of a portal transcription file.
type PortalMap struct {
	Portal         PortalMeta          `yaml:"portal"`
	Preconditions  []Precondition      `yaml:"preconditions"`
	Navigation     []NavStep           `yaml:"navigation"`
	Pages          []Page              `yaml:"pages"`
	Uploads        []UploadRequirement `yaml:"uploads"`
	PostSubmission []PostStep          `yaml:"post_submission"`
	TCC            TCCSection          `yaml:"tcc"`
}

// PortalMeta identifies the portal and its version.
type PortalMeta struct {
	Code           string `yaml:"code"`
	Jurisdiction   string `yaml:"jurisdiction"`
	Authority      string `yaml:"authority"`
	URL            string `yaml:"url"`
	MapVersion     string `yaml:"map_version"`
	ObservedSeason int    `yaml:"observed_season"`
	// Status must be "PUBLISHED" for production use. "DRAFT" is permitted only
	// when LoadOptions.AllowDraft is true.
	Status string `yaml:"status"`
}

// Precondition is a check that must be satisfied before the pack is useful.
// BlocksPack means the portal cannot be started without it (e.g. Payer ID).
// BlocksTCC means it does not block filing but blocks the TCC (e.g. photograph).
type Precondition struct {
	ID               string `yaml:"id"`
	Label            string `yaml:"label"`
	Detail           string `yaml:"detail"`
	BlocksPack       bool   `yaml:"blocks_pack"`
	BlocksTCC        bool   `yaml:"blocks_tcc"`
	GeneratesWarning string `yaml:"generates_warning"`
}

// NavStep is one step in the portal navigation, reproduced so the pack can
// tell the user where they are.
type NavStep struct {
	Step   string `yaml:"step"`
	Detail string `yaml:"detail"`
	Field  string `yaml:"field"`
}

// Page is one screen of the portal form.
type Page struct {
	ID             string              `yaml:"id"`
	Title          string              `yaml:"title"`
	Note           string              `yaml:"note"`
	Fields         []PageField         `yaml:"fields"`
	UnmappedIncome *UnmappedIncomeSpec `yaml:"unmapped_income"`
	Next           string              `yaml:"next"`
	Submit         string              `yaml:"submit"`
}

// PageField is one data-entry field inside a portal page.
type PageField struct {
	PortalLabel   string   `yaml:"portal_label"`
	Source        string   `yaml:"source"`
	Repeatable    bool     `yaml:"repeatable"`
	Format        string   `yaml:"format"`        // "naira_2dp" or empty
	Note          string   `yaml:"note"`
	ConditionalOn string   `yaml:"conditional_on"` // e.g. "accommodation.type == Rent"
	Upload        string   `yaml:"upload"`
	Enum          []string `yaml:"enum"`
}

// UnmappedIncomeSpec names income categories that have no portal field.
// The pack generator flags any non-zero amounts for user attention.
type UnmappedIncomeSpec struct {
	Note       string   `yaml:"note"`
	Categories []string `yaml:"categories"`
}

// UploadRequirement lists a document the user needs before starting.
type UploadRequirement struct {
	Label        string `yaml:"label"`
	RequiredWhen string `yaml:"required_when"`
	Note         string `yaml:"note"`
}

// PostStep is an instruction for after the portal submission.
type PostStep struct {
	Step      string `yaml:"step"`
	Detail    string `yaml:"detail"`
	OurAction string `yaml:"our_action"`
}

// TCCSection contains the Tax Clearance Certificate route and conditions.
type TCCSection struct {
	Route             string         `yaml:"route"`
	Verification      string         `yaml:"verification"`
	StatutoryTimeline string         `yaml:"statutory_timeline"`
	Conditions        []TCCCondition `yaml:"conditions"`
}

// TCCCondition is one of the six T1–T6 conditions required for a TCC.
type TCCCondition struct {
	ID    string `yaml:"id"`
	Label string `yaml:"label"`
}

// LoadOptions mirrors taxengine.LoadOptions: only one flag for now.
type LoadOptions struct {
	// AllowDraft permits loading a portal map that has not been verified
	// against a live portal session. Set this only in tests and the authoring
	// tool — production callers must leave it false.
	AllowDraft bool
}

// Load parses raw YAML bytes into a PortalMap using the injected unmarshal
// function. It rejects DRAFT maps unless AllowDraft is set, for the same
// reason the rule set loader does: a map nobody has verified can guide a
// user to the wrong field.
func Load(data []byte, unmarshal func([]byte, any) error, opts LoadOptions) (PortalMap, error) {
	var pm PortalMap
	if err := unmarshal(data, &pm); err != nil {
		return pm, fmt.Errorf("portalmap: %w", err)
	}
	if pm.Portal.Status != "PUBLISHED" && !opts.AllowDraft {
		return pm, fmt.Errorf(
			"portalmap: map %s (version %s) has status %q and has not been verified "+
				"against a live portal session; set LoadOptions.AllowDraft only in tests",
			pm.Portal.Code, pm.Portal.MapVersion, pm.Portal.Status)
	}
	return pm, nil
}
