package catalog

// Shared nested field schemas (Appendix B). Unions carry variant names in
// Enum and per-variant payload schemas in Fields (entry Name == variant name).

var modFlags = []string{"SHIFT", "CTRL", "ALT", "SUPER"}
var keyModFlags = []string{"SHIFT", "CTRL", "ALT", "SUPER", "LEADER"}

var FontAttributesFields = []Field{
	{Name: "family", Kind: String, Required: true},
	{Name: "weight", Kind: String, NumericString: true, Default: "Regular", Enum: []string{"Thin", "ExtraLight", "Light", "DemiLight", "Book", "Regular", "Medium", "DemiBold", "Bold", "ExtraBold", "Black", "ExtraBlack"}},
	{Name: "stretch", Kind: Enum, Enum: []string{"UltraCondensed", "ExtraCondensed", "Condensed", "SemiCondensed", "Normal", "SemiExpanded", "Expanded", "ExtraExpanded", "UltraExpanded"}, Default: "Normal"},
	{Name: "style", Kind: Enum, Enum: []string{"Normal", "Italic", "Oblique"}, Default: "Normal"},
	{Name: "harfbuzz_features", Kind: StringList},
	{Name: "freetype_load_target", Kind: Enum, Enum: []string{"Normal", "Light", "Mono", "HorizontalLcd", "VerticalLcd"}},
	{Name: "freetype_render_target", Kind: Enum, Enum: []string{"Normal", "Light", "Mono", "HorizontalLcd", "VerticalLcd"}},
	{Name: "freetype_load_flags", Kind: Flags, Enum: []string{"DEFAULT", "NO_HINTING", "NO_BITMAP", "FORCE_AUTOHINT", "MONOCHROME", "NO_AUTOHINT", "NO_SVG", "SVG_ONLY"}, EmptyToken: "DEFAULT"},
	{Name: "scale", Kind: Float, Min: ptr(0), MinExclusive: true},
	{Name: "assume_emoji_presentation", Kind: Bool},
}

var StyleRuleFields = []Field{
	{Name: "intensity", Kind: Enum, Enum: []string{"Normal", "Bold", "Half"}},
	{Name: "underline", Kind: Enum, Enum: []string{"None", "Single", "Double", "Curly", "Dotted", "Dashed"}},
	{Name: "italic", Kind: Bool},
	{Name: "blink", Kind: Enum, Enum: []string{"None", "Slow", "Rapid"}},
	{Name: "reverse", Kind: Bool},
	{Name: "strikethrough", Kind: Bool},
	{Name: "invisible", Kind: Bool},
	{Name: "font", Kind: Font, FontForeground: true, Required: true},
}

var HsbFields = []Field{
	{Name: "hue", Kind: Float, Default: 1.0},
	{Name: "saturation", Kind: Float, Default: 1.0},
	{Name: "brightness", Kind: Float, Default: 1.0},
}

var WindowPaddingFields = []Field{
	{Name: "left", Kind: Dimension, Default: "1cell"},
	{Name: "right", Kind: Dimension, Default: "1cell"},
	{Name: "top", Kind: Dimension, Default: "0.5cell"},
	{Name: "bottom", Kind: Dimension, Default: "0.5cell"},
}

var ContentAlignmentFields = []Field{
	{Name: "horizontal", Kind: Enum, Enum: []string{"Left", "Center", "Right"}, Default: "Left", Required: true},
	{Name: "vertical", Kind: Enum, Enum: []string{"Top", "Center", "Bottom"}, Default: "Top", Required: true},
}

var WindowFrameFields = []Field{
	{Name: "inactive_titlebar_bg", Kind: Color, Default: "#333333"},
	{Name: "active_titlebar_bg", Kind: Color, Default: "#333333"},
	{Name: "inactive_titlebar_fg", Kind: Color, Default: "#cccccc"},
	{Name: "active_titlebar_fg", Kind: Color, Default: "#ffffff"},
	{Name: "inactive_titlebar_border_bottom", Kind: Color, Default: "#2b2042"},
	{Name: "active_titlebar_border_bottom", Kind: Color, Default: "#2b2042"},
	{Name: "button_fg", Kind: Color, Default: "#cccccc"},
	{Name: "button_bg", Kind: Color, Default: "#333333"},
	{Name: "button_hover_fg", Kind: Color, Default: "#ffffff"},
	{Name: "button_hover_bg", Kind: Color, Default: "#1f1f1f"},
	{Name: "font", Kind: Font, DefaultNote: "Roboto"},
	{Name: "font_size", Kind: Float, DefaultNote: "10 on Windows, 12 elsewhere"},
	{Name: "border_left_width", Kind: Dimension, Default: "0px"},
	{Name: "border_right_width", Kind: Dimension, Default: "0px"},
	{Name: "border_top_height", Kind: Dimension, Default: "0px"},
	{Name: "border_bottom_height", Kind: Dimension, Default: "0px"},
	{Name: "border_left_color", Kind: Color},
	{Name: "border_right_color", Kind: Color},
	{Name: "border_top_color", Kind: Color},
	{Name: "border_bottom_color", Kind: Color},
}

var TabBarStyleFields = []Field{
	{Name: "new_tab", Kind: String, Default: " + "},
	{Name: "new_tab_hover", Kind: String, Default: " + "},
	{Name: "window_hide", Kind: String, Default: " . "},
	{Name: "window_hide_hover", Kind: String, Default: " . "},
	{Name: "window_maximize", Kind: String, Default: " - "},
	{Name: "window_maximize_hover", Kind: String, Default: " - "},
	{Name: "window_close", Kind: String, Default: " X "},
	{Name: "window_close_hover", Kind: String, Default: " X "},
}

var easingBezierFields = []Field{
	{Name: "x1", Kind: Float, Required: true},
	{Name: "y1", Kind: Float, Required: true},
	{Name: "x2", Kind: Float, Required: true},
	{Name: "y2", Kind: Float, Required: true},
}

var EasingNames = []string{"Linear", "Ease", "EaseIn", "EaseInOut", "EaseOut", "Constant", "CubicBezier"}
var EasingFields = []Field{
	{Name: "Linear", Kind: Union, Scalar: true},
	{Name: "Ease", Kind: Union, Scalar: true},
	{Name: "EaseIn", Kind: Union, Scalar: true},
	{Name: "EaseInOut", Kind: Union, Scalar: true},
	{Name: "EaseOut", Kind: Union, Scalar: true},
	{Name: "Constant", Kind: Union, Scalar: true},
	{Name: "CubicBezier", Kind: Union, Tuple: true, Fields: easingBezierFields},
}

var VisualBellFields = []Field{
	{Name: "fade_in_duration_ms", Kind: Int, Min: ptr(0), Default: 0.0},
	{Name: "fade_in_function", Kind: Union, Enum: EasingNames, Fields: EasingFields, Default: "Ease"},
	{Name: "fade_out_duration_ms", Kind: Int, Min: ptr(0), Default: 0.0},
	{Name: "fade_out_function", Kind: Union, Enum: EasingNames, Fields: EasingFields, Default: "Ease"},
	{Name: "target", Kind: Enum, Enum: []string{"BackgroundColor", "CursorColor"}, Default: "BackgroundColor"},
}

var gradientOrientationFields = []Field{
	{Name: "Linear", Kind: Union, Tuple: true, Fields: []Field{{Name: "angle", Kind: Float, Required: true}}},
	{Name: "Radial", Kind: Union, Tuple: true, Fields: []Field{
		{Name: "radius", Kind: Float, Required: true},
		{Name: "cx", Kind: Float, Required: true},
		{Name: "cy", Kind: Float, Required: true},
	}},
}

var GradientFields = []Field{
	{Name: "orientation", Kind: Union, Enum: []string{"Horizontal", "Vertical", "Linear", "Radial"}, Fields: gradientOrientationFields, Default: "Horizontal"},
	{Name: "colors", Kind: StringList},
	{Name: "preset", Kind: Enum, Enum: []string{"Blues", "BrBg", "BuGn", "BuPu", "Cividis", "Cool", "CubeHelixDefault", "GnBu", "Greens", "Greys", "Inferno", "Magma", "OrRd", "Oranges", "PiYg", "Plasma", "PrGn", "PuBu", "PuBuGn", "PuOr", "PuRd", "Purples", "Rainbow", "RdBu", "RdGy", "RdPu", "RdYlBu", "RdYlGn", "Reds", "Sinebow", "Spectral", "Turbo", "Viridis", "Warm", "YlGn", "YlGnBu", "YlOrBr", "YlOrRd"}},
	{Name: "interpolation", Kind: Enum, Enum: []string{"Linear", "Basis", "CatmullRom"}, Default: "Linear"},
	{Name: "blend", Kind: Enum, Enum: []string{"Rgb", "LinearRgb", "Hsv", "Oklab"}, Default: "Rgb"},
	{Name: "noise", Kind: Int, Min: ptr(0), DefaultNote: "64"},
	{Name: "segment_size", Kind: Int, Min: ptr(1)},
	{Name: "segment_smoothness", Kind: Float, Min: ptr(0), Max: ptr(1)},
}

var backgroundSourceFields = []Field{
	{Name: "File", Kind: Union, Fields: []Field{
		{Name: "path", Kind: Path, Required: true},
		{Name: "speed", Kind: Float},
	}},
	{Name: "Gradient", Kind: Union, Fields: GradientFields},
	{Name: "Color", Kind: Union, Scalar: true, Fields: []Field{{Name: "Color", Kind: Color}}},
}

var backgroundAttachmentFields = []Field{
	{Name: "Parallax", Kind: Union, Scalar: true, Fields: []Field{{Name: "Parallax", Kind: Float}}},
}

var BackgroundLayerFields = []Field{
	{Name: "source", Kind: Union, Enum: []string{"File", "Gradient", "Color"}, Fields: backgroundSourceFields, Required: true},
	{Name: "origin", Kind: Enum, Enum: []string{"BorderBox", "PaddingBox"}, Default: "BorderBox"},
	{Name: "attachment", Kind: Union, Enum: []string{"Fixed", "Scroll", "Parallax"}, Fields: backgroundAttachmentFields, Default: "Fixed"},
	{Name: "repeat_x", Kind: Enum, Enum: []string{"Repeat", "Mirror", "NoRepeat"}, Default: "Repeat"},
	{Name: "repeat_x_size", Kind: Dimension, Enum: []string{"Cover", "Contain"}},
	{Name: "repeat_y", Kind: Enum, Enum: []string{"Repeat", "Mirror", "NoRepeat"}, Default: "Repeat"},
	{Name: "repeat_y_size", Kind: Dimension, Enum: []string{"Cover", "Contain"}},
	{Name: "vertical_align", Kind: Enum, Enum: []string{"Top", "Middle", "Bottom"}, Default: "Top"},
	{Name: "vertical_offset", Kind: Dimension},
	{Name: "horizontal_align", Kind: Enum, Enum: []string{"Left", "Center", "Right"}, Default: "Left"},
	{Name: "horizontal_offset", Kind: Dimension},
	{Name: "opacity", Kind: Float01, Default: 1.0},
	{Name: "hsb", Kind: Struct, Fields: HsbFields},
	{Name: "width", Kind: Dimension, Enum: []string{"Cover", "Contain"}, Default: "Cover"},
	{Name: "height", Kind: Dimension, Enum: []string{"Cover", "Contain"}, Default: "Cover"},
}

var DaemonOptionsFields = []Field{
	{Name: "pid_file", Kind: Path},
	{Name: "stdout", Kind: Path},
	{Name: "stderr", Kind: Path},
}

var GpuInfoFields = []Field{
	{Name: "name", Kind: String, Required: true},
	{Name: "device_type", Kind: String, Required: true},
	{Name: "backend", Kind: String, Required: true},
	{Name: "driver", Kind: String},
	{Name: "driver_info", Kind: String},
	{Name: "vendor", Kind: Int},
	{Name: "device", Kind: Int},
}

var CellWidthFields = []Field{
	{Name: "first", Kind: Int, Min: ptr(0), Required: true},
	{Name: "last", Kind: Int, Min: ptr(0), Required: true},
	{Name: "width", Kind: Int, Min: ptr(1), Required: true},
}

var HyperlinkRuleFields = []Field{
	{Name: "regex", Kind: String, Required: true},
	{Name: "format", Kind: String, Required: true},
	{Name: "highlight", Kind: Int, Min: ptr(0), Default: 0.0},
}

var SpawnTabDomainNames = []string{"DefaultDomain", "CurrentPaneDomain", "DomainName", "DomainId"}
var spawnTabDomainFields = []Field{
	{Name: "DomainName", Kind: Union, Scalar: true, Fields: []Field{{Name: "DomainName", Kind: String}}},
	{Name: "DomainId", Kind: Union, Scalar: true, Fields: []Field{{Name: "DomainId", Kind: Int, Min: ptr(0)}}},
}

var guiPositionFields = []Field{
	{Name: "x", Kind: Dimension, Required: true},
	{Name: "y", Kind: Dimension, Required: true},
	{Name: "origin", Kind: Union, Enum: []string{"ScreenCoordinateSystem", "MainScreen", "ActiveScreen", "Named"}, Fields: []Field{
		{Name: "Named", Kind: Union, Scalar: true, Fields: []Field{{Name: "Named", Kind: String}}},
	}, Default: "ScreenCoordinateSystem"},
}

var SpawnCommandFields = []Field{
	{Name: "label", Kind: String},
	{Name: "args", Kind: StringList},
	{Name: "cwd", Kind: Dir},
	{Name: "set_environment_variables", Kind: StringMap},
	{Name: "domain", Kind: Union, Enum: SpawnTabDomainNames, Fields: spawnTabDomainFields, Default: "CurrentPaneDomain"},
	{Name: "position", Kind: Struct, Fields: guiPositionFields},
}

var SshDomainFields = []Field{
	{Name: "name", Kind: String, Required: true},
	{Name: "remote_address", Kind: String, Required: true},
	{Name: "username", Kind: String},
	{Name: "no_agent_auth", Kind: Bool, Default: false},
	{Name: "connect_automatically", Kind: Bool, Default: false},
	{Name: "timeout", Kind: Int, Default: 60.0},
	{Name: "remote_wezterm_path", Kind: String},
	{Name: "override_proxy_command", Kind: String},
	{Name: "ssh_backend", Kind: Enum, Enum: []string{"LibSsh", "Ssh2"}},
	{Name: "multiplexing", Kind: Enum, Enum: []string{"WezTerm", "None"}, Default: "WezTerm"},
	{Name: "ssh_option", Kind: StringMap},
	{Name: "default_prog", Kind: StringList},
	{Name: "assume_shell", Kind: Enum, Enum: []string{"Unknown", "Posix"}, Default: "Unknown"},
	{Name: "local_echo_threshold_ms", Kind: Int, Default: 100.0},
	{Name: "overlay_lag_indicator", Kind: Bool, Default: false},
}

var UnixDomainFields = []Field{
	{Name: "name", Kind: String, Required: true},
	{Name: "socket_path", Kind: Path},
	{Name: "connect_automatically", Kind: Bool, Default: false},
	{Name: "no_serve_automatically", Kind: Bool, Default: false},
	{Name: "serve_command", Kind: StringList},
	{Name: "proxy_command", Kind: StringList},
	{Name: "skip_permissions_check", Kind: Bool, Default: false},
	{Name: "read_timeout", Kind: Int, Default: 60.0},
	{Name: "write_timeout", Kind: Int, Default: 60.0},
	{Name: "local_echo_threshold_ms", Kind: Int},
	{Name: "overlay_lag_indicator", Kind: Bool, Default: false},
}

var TlsClientFields = []Field{
	{Name: "name", Kind: String, Required: true},
	{Name: "remote_address", Kind: String, Required: true},
	{Name: "bootstrap_via_ssh", Kind: String},
	{Name: "pem_private_key", Kind: Path},
	{Name: "pem_cert", Kind: Path},
	{Name: "pem_ca", Kind: Path},
	{Name: "pem_root_certs", Kind: StringList},
	{Name: "accept_invalid_hostnames", Kind: Bool, Default: false},
	{Name: "expected_cn", Kind: String},
	{Name: "connect_automatically", Kind: Bool, Default: false},
	{Name: "read_timeout", Kind: Int, Default: 60.0},
	{Name: "write_timeout", Kind: Int, Default: 60.0},
	{Name: "local_echo_threshold_ms", Kind: Int, Default: 100.0},
	{Name: "remote_wezterm_path", Kind: String},
	{Name: "overlay_lag_indicator", Kind: Bool, Default: false},
}

var TlsServerFields = []Field{
	{Name: "bind_address", Kind: String, Required: true},
	{Name: "pem_private_key", Kind: Path},
	{Name: "pem_cert", Kind: Path},
	{Name: "pem_ca", Kind: Path},
	{Name: "pem_root_certs", Kind: StringList},
}

var WslDomainFields = []Field{
	{Name: "name", Kind: String, Required: true},
	{Name: "distribution", Kind: String, Required: true},
	{Name: "username", Kind: String},
	{Name: "default_cwd", Kind: Dir},
	{Name: "default_prog", Kind: StringList},
}

var ExecDomainFields = []Field{
	{Name: "name", Kind: String, Required: true},
	{Name: "fixup", Kind: Lua, Required: true, Default: "function(cmd)\n  return cmd\nend"},
	{Name: "label", Kind: String},
}

var SerialDomainFields = []Field{
	{Name: "name", Kind: String, Required: true},
	{Name: "port", Kind: String, DefaultNote: "e.g. /dev/ttyUSB0 or COM3"},
	{Name: "baud", Kind: Int, Min: ptr(1), DefaultNote: "9600"},
}

var colorSpecNames = []string{"Default", "Color", "AnsiColor"}
var colorSpecFields = []Field{
	{Name: "Color", Kind: Union, Scalar: true, Fields: []Field{{Name: "Color", Kind: Color}}},
	{Name: "AnsiColor", Kind: Union, Scalar: true, Fields: []Field{{Name: "AnsiColor", Kind: Enum, Enum: []string{"Black", "Maroon", "Green", "Olive", "Navy", "Purple", "Teal", "Silver", "Grey", "Red", "Lime", "Yellow", "Blue", "Fuchsia", "Aqua", "White"}}}},
}

// ColorSpecField returns the Default|Color|AnsiColor union under the given key.
func ColorSpecField(name string) Field {
	return Field{Name: name, Kind: Union, Enum: colorSpecNames, Fields: colorSpecFields}
}

var TabBarColorFields = []Field{
	{Name: "bg_color", Kind: Color, Required: true},
	{Name: "fg_color", Kind: Color, Required: true},
	{Name: "intensity", Kind: Enum, Enum: []string{"Normal", "Bold", "Half"}, Default: "Normal"},
	{Name: "underline", Kind: Enum, Enum: []string{"None", "Single", "Double", "Curly", "Dotted", "Dashed"}, Default: "None"},
	{Name: "italic", Kind: Bool, Default: false},
	{Name: "strikethrough", Kind: Bool, Default: false},
}

var TabBarColorsFields = []Field{
	{Name: "background", Kind: Color, Default: "#333333"},
	{Name: "active_tab", Kind: Struct, Fields: TabBarColorFields},
	{Name: "inactive_tab", Kind: Struct, Fields: TabBarColorFields},
	{Name: "inactive_tab_hover", Kind: Struct, Fields: TabBarColorFields},
	{Name: "new_tab", Kind: Struct, Fields: TabBarColorFields},
	{Name: "new_tab_hover", Kind: Struct, Fields: TabBarColorFields},
	{Name: "inactive_tab_edge", Kind: Color, Default: "#575757"},
	{Name: "inactive_tab_edge_hover", Kind: Color, Default: "#363636"},
}

var PaletteFields = func() []Field {
	f := []Field{
		{Name: "foreground", Kind: Color},
		{Name: "background", Kind: Color},
		{Name: "cursor_fg", Kind: Color},
		{Name: "cursor_bg", Kind: Color},
		{Name: "cursor_border", Kind: Color},
		{Name: "selection_fg", Kind: Color},
		{Name: "selection_bg", Kind: Color},
		{Name: "ansi", Kind: List, ItemKind: Color, FixedLen: 8},
		{Name: "brights", Kind: List, ItemKind: Color, FixedLen: 8},
		{Name: "indexed", Kind: StringMap},
		{Name: "tab_bar", Kind: Struct, Fields: TabBarColorsFields},
		{Name: "scrollbar_thumb", Kind: Color},
		{Name: "split", Kind: Color},
		{Name: "visual_bell", Kind: Color},
		{Name: "compose_cursor", Kind: Color},
	}
	for _, n := range []string{"copy_mode_active_highlight_fg", "copy_mode_active_highlight_bg", "copy_mode_inactive_highlight_fg", "copy_mode_inactive_highlight_bg", "quick_select_label_fg", "quick_select_label_bg", "quick_select_match_fg", "quick_select_match_bg", "input_selector_label_fg", "input_selector_label_bg", "launcher_label_fg", "launcher_label_bg"} {
		f = append(f, ColorSpecField(n))
	}
	return f
}()

var KeyBindingFields = []Field{
	{Name: "key", Kind: String, Required: true},
	{Name: "mods", Kind: Flags, Enum: keyModFlags, EmptyToken: "NONE"},
	{Name: "action", Kind: Action, Required: true},
}

var LeaderFields = []Field{
	{Name: "key", Kind: String, Required: true},
	{Name: "mods", Kind: Flags, Enum: modFlags, EmptyToken: "NONE"},
	{Name: "timeout_milliseconds", Kind: Int, Min: ptr(0), Default: 1000.0},
}

var MouseButtonNames = []string{"Left", "Middle", "Right", "WheelUp", "WheelDown", "WheelLeft", "WheelRight"}
var mouseButtonFields = []Field{
	{Name: "WheelUp", Kind: Union, Scalar: true, Fields: []Field{{Name: "WheelUp", Kind: Int, Min: ptr(1)}}},
	{Name: "WheelDown", Kind: Union, Scalar: true, Fields: []Field{{Name: "WheelDown", Kind: Int, Min: ptr(1)}}},
	{Name: "WheelLeft", Kind: Union, Scalar: true, Fields: []Field{{Name: "WheelLeft", Kind: Int, Min: ptr(1)}}},
	{Name: "WheelRight", Kind: Union, Scalar: true, Fields: []Field{{Name: "WheelRight", Kind: Int, Min: ptr(1)}}},
}

var MouseBindingFields = []Field{
	{Name: "event", Kind: Struct, Required: true, Fields: []Field{
		{Name: "kind", Kind: Enum, Enum: []string{"Down", "Up", "Drag"}, Required: true},
		{Name: "streak", Kind: Int, Min: ptr(1), Default: 1.0},
		{Name: "button", Kind: Union, Enum: MouseButtonNames, Fields: mouseButtonFields, Required: true},
	}},
	{Name: "mods", Kind: Flags, Enum: keyModFlags, EmptyToken: "NONE"},
	{Name: "mouse_reporting", Kind: Bool, Default: false},
	{Name: "alt_screen", Kind: Enum, Enum: []string{"Any", "true", "false"}, Default: "Any"},
	{Name: "action", Kind: Action, Required: true},
}

var CopyModeNames = []string{
	"MoveToViewportBottom", "MoveToViewportTop", "MoveToViewportMiddle", "MoveToScrollbackTop", "MoveToScrollbackBottom",
	"ClearSelectionMode", "MoveToStartOfLineContent", "MoveToEndOfLineContent", "MoveToStartOfLine", "MoveToStartOfNextLine",
	"MoveToSelectionOtherEnd", "MoveToSelectionOtherEndHoriz", "MoveBackwardWord", "MoveForwardWord", "MoveForwardWordEnd",
	"MoveRight", "MoveLeft", "MoveUp", "MoveDown", "PageUp", "PageDown", "Close", "PriorMatch", "NextMatch",
	"PriorMatchPage", "NextMatchPage", "CycleMatchType", "ClearPattern", "EditPattern", "AcceptPattern",
	"MoveBackwardSemanticZone", "MoveForwardSemanticZone", "JumpAgain", "JumpReverse",
	"SetSelectionMode", "MoveByPage", "MoveBackwardZoneOfType", "MoveForwardZoneOfType", "JumpForward", "JumpBackward",
}

var CopyModeUnion = Field{
	Name: "CopyMode", Kind: Union, Enum: CopyModeNames, Fields: []Field{
		{Name: "SetSelectionMode", Kind: Union, Scalar: true, Fields: []Field{{Name: "SetSelectionMode", Kind: Enum, Enum: []string{"Cell", "Word", "Line", "SemanticZone", "Block"}}}},
		{Name: "MoveByPage", Kind: Union, Scalar: true, Fields: []Field{{Name: "MoveByPage", Kind: Float}}},
		{Name: "MoveBackwardZoneOfType", Kind: Union, Scalar: true, Fields: []Field{{Name: "MoveBackwardZoneOfType", Kind: Enum, Enum: []string{"Output", "Input", "Prompt"}}}},
		{Name: "MoveForwardZoneOfType", Kind: Union, Scalar: true, Fields: []Field{{Name: "MoveForwardZoneOfType", Kind: Enum, Enum: []string{"Output", "Input", "Prompt"}}}},
		{Name: "JumpForward", Kind: Union, Fields: []Field{{Name: "prev_char", Kind: Bool, Required: true}}},
		{Name: "JumpBackward", Kind: Union, Fields: []Field{{Name: "prev_char", Kind: Bool, Required: true}}},
	},
}
