package catalog

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed schemes.json
var schemesJSON []byte

// Scheme is one built-in WezTerm color scheme.
type Scheme struct {
	Name    string         `json:"name"`
	Aliases []string       `json:"aliases"`
	Colors  map[string]any `json:"colors"`
}

var (
	schemesOnce sync.Once
	schemes     []Scheme
)

func loadSchemes() {
	var raw []Scheme
	if err := json.Unmarshal(schemesJSON, &raw); err != nil {
		return
	}
	schemes = raw
}

// Schemes returns all built-in color schemes (parsed once).
func Schemes() []Scheme {
	schemesOnce.Do(loadSchemes)
	return schemes
}

// SchemeNames returns the built-in scheme names in file order.
func SchemeNames() []string {
	ss := Schemes()
	names := make([]string, len(ss))
	for i, s := range ss {
		names[i] = s.Name
	}
	return names
}
