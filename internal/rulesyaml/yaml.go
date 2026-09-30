// Package rulesyaml unmarshals rule-set and golden YAML outside taxengine.
// The engine accepts an injected unmarshal function so it never depends on a YAML library.
package rulesyaml

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Unmarshal decodes YAML into v. Date-only scalars (YYYY-MM-DD) decode as midnight UTC.
func Unmarshal(in []byte, v any) error {
	var node yaml.Node
	if err := yaml.Unmarshal(in, &node); err != nil {
		return err
	}
	promoteDates(&node)
	if err := node.Decode(v); err != nil {
		return fmt.Errorf("rulesyaml: %w", err)
	}
	return nil
}

func promoteDates(n *yaml.Node) {
	if n == nil {
		return
	}
	if n.Kind == yaml.ScalarNode && looksLikeDate(n.Value) {
		if _, err := time.Parse("2006-01-02", n.Value); err == nil {
			n.Value = n.Value + "T00:00:00Z"
			n.Tag = "!!timestamp"
		}
	}
	for i := range n.Content {
		promoteDates(n.Content[i])
	}
}

func looksLikeDate(s string) bool {
	if len(s) != 10 {
		return false
	}
	if s[4] != '-' || s[7] != '-' {
		return false
	}
	for i, c := range s {
		if i == 4 || i == 7 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
