// Package rulesregistry provides an in-memory index of loaded rule sets,
// keyed by (jurisdiction, year_of_assessment). It is the production answer to
// "give me the rule set for NG 2026" and the mechanism that will serve
// historical years once practitioners have signed them.
//
// The package accepts raw YAML bytes and an injected unmarshal function so it
// never imports a YAML library directly — the same pattern as taxengine.Load.
package rulesregistry

import (
	"errors"
	"fmt"
	"strconv"

	"taxsoftware/taxengine"
)

// ErrRuleSetNotFound is returned when no rule set matches the requested
// (jurisdiction, year_of_assessment) pair.
var ErrRuleSetNotFound = errors.New("rulesregistry: rule set not found")

// Registry holds parsed rule sets indexed by (jurisdiction, year_of_assessment).
// It is read-only after construction; all mutation happens in New.
type Registry struct {
	sets map[string]*taxengine.RuleSet
}

// New parses each element of rawFiles as a rule set YAML, validates it via
// taxengine.Load, and returns a Registry indexed by (jurisdiction, year).
//
// Returns an error on the first file that fails to parse or validate, and on
// any duplicate (jurisdiction, year) pair. A Registry with duplicate keys is
// corrupt data — the caller must ensure each directory contains exactly one
// rule set per (jurisdiction, year).
func New(rawFiles [][]byte, unmarshal func([]byte, any) error, opts taxengine.LoadOptions) (*Registry, error) {
	r := &Registry{sets: make(map[string]*taxengine.RuleSet, len(rawFiles))}
	for i, raw := range rawFiles {
		rs, err := taxengine.Load(raw, unmarshal, opts)
		if err != nil {
			return nil, fmt.Errorf("rulesregistry: file index %d: %w", i, err)
		}
		k := indexKey(rs.Jurisdiction, rs.YearOfAssessment)
		if _, exists := r.sets[k]; exists {
			return nil, fmt.Errorf("rulesregistry: duplicate rule set for (%s, %d)",
				rs.Jurisdiction, rs.YearOfAssessment)
		}
		r.sets[k] = rs
	}
	return r, nil
}

// Get retrieves the rule set for the given jurisdiction and year of assessment.
// Returns ErrRuleSetNotFound when no matching rule set has been loaded.
func (r *Registry) Get(jurisdiction string, year int) (*taxengine.RuleSet, error) {
	rs, ok := r.sets[indexKey(jurisdiction, year)]
	if !ok {
		return nil, fmt.Errorf("%w: (%s, %d)", ErrRuleSetNotFound, jurisdiction, year)
	}
	return rs, nil
}

func indexKey(jurisdiction string, year int) string {
	return jurisdiction + "/" + strconv.Itoa(year)
}
