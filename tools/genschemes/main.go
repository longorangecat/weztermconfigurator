// Command genschemes converts wezterm's docs/colorschemes/data.json into the
// embedded catalog scheme list.
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
	var in []scheme
	if err := json.Unmarshal(raw, &in); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sort.Slice(in, func(i, j int) bool {
		return strings.ToLower(in[i].Name) < strings.ToLower(in[j].Name)
	})
	out, err := json.MarshalIndent(in, "", " ")
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
