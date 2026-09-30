package golden

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	"taxsoftware/taxengine"
)

// Naira is a YAML scalar in naira, stored as kobo. At most two decimal places.
// Unmarshalling never uses float64.
type Naira struct {
	Kobo taxengine.Kobo
}

func (n *Naira) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("golden: naira amount must be a scalar, got %s", value.Tag)
	}
	k, err := ParseNaira(value.Value)
	if err != nil {
		return err
	}
	n.Kobo = k
	return nil
}

func ParseNaira(s string) (taxengine.Kobo, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" || s == "~" {
		return 0, fmt.Errorf("golden: empty naira amount")
	}
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	wholeStr, fracStr, ok := strings.Cut(s, ".")
	if !ok {
		wholeStr, fracStr = s, ""
	}
	if wholeStr == "" {
		wholeStr = "0"
	}
	var whole int64
	for _, c := range wholeStr {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("golden: invalid naira amount %q", s)
		}
		whole = whole*10 + int64(c-'0')
	}
	if len(fracStr) > 2 {
		return 0, fmt.Errorf("golden: naira amount %q has more than two decimal places", s)
	}
	for len(fracStr) < 2 {
		fracStr += "0"
	}
	var frac int64
	for _, c := range fracStr {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("golden: invalid naira amount %q", s)
		}
		frac = frac*10 + int64(c-'0')
	}
	k := whole*int64(taxengine.KoboPerNaira) + frac
	if neg {
		k = -k
	}
	return taxengine.Kobo(k), nil
}
