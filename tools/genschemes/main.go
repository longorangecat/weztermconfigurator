// Command genschemes converts wezterm's docs/colorschemes/data.json into the
// embedded catalog scheme list.
//
// Regeneration command:
//
//	curl -sL https://raw.githubusercontent.com/wezterm/wezterm/b09b56c29c1e367e598b60ca266e2cc9038751e0/docs/colorschemes/data.json -o /tmp/wezterm-data.json && go run ./tools/genschemes /tmp/wezterm-data.json internal/catalog/schemes.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type scheme struct {
	Name    string         `json:"name"`
	Aliases []string       `json:"aliases"`
	Colors  map[string]any `json:"colors"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: genschemes <data.json> <out.json>")
		os.Exit(2)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var in []struct {
		Colors   map[string]any `json:"colors"`
		Metadata struct {
			Name    string   `json:"name"`
			Aliases []string `json:"aliases"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	schemes := make([]scheme, len(in))
	for i, entry := range in {
		if entry.Metadata.Name == "" {
			fmt.Fprintf(os.Stderr, "entry %d has no metadata.name\n", i)
			os.Exit(1)
		}
		schemes[i] = scheme{
			Name:    entry.Metadata.Name,
			Aliases: entry.Metadata.Aliases,
			Colors:  entry.Colors,
		}
	}
	sort.Slice(schemes, func(i, j int) bool {
		li, lj := strings.ToLower(schemes[i].Name), strings.ToLower(schemes[j].Name)
		if li == lj {
			return schemes[i].Name < schemes[j].Name
		}
		return li < lj
	})
	out, err := json.MarshalIndent(schemes, "", " ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(os.Args[2], out, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d schemes to %s\n", len(in), os.Args[2])
}
