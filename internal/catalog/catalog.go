// Package catalog holds the complete WezTerm option catalog: every user-facing
// Config field with its type, schema, defaults and documentation.
package catalog

// Kind identifies the value shape of a Field and drives both the UI widget
// choice and the Lua emitter.
type Kind uint8

const (
	Bool Kind = iota + 1
	Int
	Float
	Float01
	String
	Enum
	Flags
	Color
	Path
	Dir
	Dimension
	StringList
	IntList
	StringMap
	FloatMap
	Font   // TextStyle: JSON {"font":[FontAttributes...],"foreground":"..."}; fixed schema FontAttributesFields
	Struct // Fields = named members; JSON object
	List   // Fields = item schema (Struct-like) unless ItemKind set; JSON array
	Union  // Enum = variant names; Fields = payload schema per variant (entry Name == variant); JSON string | {"Variant":payload}
	Palette
	NamedPalettes
	Keys
	KeyTables
	Mouse
	Leader
	Action
	SchemeName
	Lua // verbatim Lua expression (multi-line); JSON string
)

// Field describes one option or one nested schema member.
type Field struct {
	Name           string // exact Lua key (or Union variant name for payload entries)
	Kind           Kind
	Doc            string
	Default        any      // JSON-shaped effective default shown as placeholder; nil = WezTerm nil/omitted
	DefaultNote    string   // human note shown after "Default:" when Default is nil or behavioral
	Enum           []string // Enum/Flags tokens; Union variant names; Dimension: extra literal tokens (Cover|Contain); StringList: allowed line values
	Fields         []Field  // Struct members | List item members | Union payloads
	ItemKind       Kind     // List of scalars/unions (e.g. List of Action, List of Color); zero = struct items
	FixedLen       int      // List: exactly N items (ansi/brights = 8); 0 = variable
	Tuple          bool     // Union payload / action arg emitted positionally as an array
	Scalar         bool     // Union payload is a bare scalar rather than an object
	Required       bool     // must be set whenever the enclosing value is emitted
	Min, Max       *float64 // inclusive bounds for Int/Float
	MinExclusive   bool     // Min is exclusive (Float[>0])
	EmptyToken     string   // Flags: emitted when nothing selected ("NONE" | "DEFAULT" | "")
	NumericString  bool     // String: emit as a number when the text is all digits (font weight)
	FontForeground bool     // Font: expose the `foreground` row (font_rules only)
}

// Option is a top-level WezTerm config option.
//
// The data files use positional literals:
//
//	Option{Field{...}, "Category", "Since", nil, nil, ""}
//
// so a misplaced Doc string cannot compile.

type Option struct {
	Field
	Category    string
	Since       string
	Tags        []string       // platform tags from the matrix; empty = all
	DefaultByOS map[string]any // "linux"|"windows"|"macos" → overrides Default
	Deprecated  string         // upstream message, shown as badge
}

// Options holds all 219 config options in matrix order, populated by the
// per-category data files' init functions.
var Options []Option

func init() {
	Options = append(Options,
		generalOptions...,
	)
	Options = append(Options, fontsOptions...)
	Options = append(Options, textRenderOptions...)
	Options = append(Options, colorsOptions...)
	Options = append(Options, windowOptions...)
	Options = append(Options, backgroundOptions...)
	Options = append(Options, tabBarOptions...)
	Options = append(Options, cursorOptions...)
	Options = append(Options, scrollingOptions...)
	Options = append(Options, keyBindingOptions...)
	Options = append(Options, keyboardOptions...)
	Options = append(Options, mouseOptions...)
	Options = append(Options, overlayOptions...)
	Options = append(Options, bellOptions...)
	Options = append(Options, muxOptions...)
	Options = append(Options, gpuOptions...)
	Options = append(Options, terminalOptions...)
}

// Categories lists the nav order; every Option.Category must be one of these.
var Categories = []string{
	"General & Startup",
	"Fonts",
	"Text Rendering",
	"Colors",
	"Window",
	"Background & Transparency",
	"Tab Bar",
	"Cursor & Blinking",
	"Scrolling & Panes",
	"Key Bindings",
	"Keyboard & Input",
	"Mouse & Selection",
	"Overlays & Palettes",
	"Bell & Notifications",
	"Multiplexing & Domains",
	"Rendering & GPU",
	"Terminal & Security",
}

// CustomLuaCategory is a pseudo-category rendered by the UI, not an Option.
const CustomLuaCategory = "Custom Lua"

// PluginsCategory is a pseudo-category rendered by the UI, not an Option.
const PluginsCategory = "Plugins"

// ptr returns a pointer to f (for Min/Max literals).
func ptr(f float64) *float64 { return &f }

// ByCategory returns pointers to the options of one category, in catalog order.
func ByCategory(cat string) []*Option {
	var out []*Option
	for i := range Options {
		if Options[i].Category == cat {
			out = append(out, &Options[i])
		}
	}
	return out
}

// Find returns the option with the given Lua name, or nil.
func Find(name string) *Option {
	for i := range Options {
		if Options[i].Name == name {
			return &Options[i]
		}
	}
	return nil
}

var targetTagSets = map[string]map[string]bool{
	"linux":   {"Linux": true, "X11": true, "Wayland": true, "KDE": true, "Unix": true},
	"windows": {"Windows": true, "WSL": true},
	"macos":   {"macOS": true, "Unix": true},
}

// RelevantTo reports whether an option with the given platform tags applies
// to the target OS. Empty tags apply everywhere.
func RelevantTo(tags []string, target string) bool {
	if len(tags) == 0 {
		return true
	}
	set := targetTagSets[target]
	if set == nil {
		return false
	}
	for _, t := range tags {
		if set[t] {
			return true
		}
	}
	return false
}

// DefaultFor returns the per-target default of an option.
func DefaultFor(o *Option, target string) any {
	if v, ok := o.DefaultByOS[target]; ok {
		return v
	}
	return o.Default
}
