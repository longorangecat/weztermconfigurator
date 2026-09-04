package catalog

var overlayOptions = []Option{
	opt(Field{Name: "command_palette_font", Kind: Font, DefaultNote: "= window_frame.font"}, "Overlays & Palettes", "nightly", nil, nil, ""),
	opt(Field{Name: "command_palette_font_size", Kind: Float, Min: ptr(0), MinExclusive: true, Default: 14.0}, "Overlays & Palettes", "20230320-124340-559cb7b0", nil, nil, ""),
	opt(Field{Name: "command_palette_line_height", Kind: Float, Min: ptr(0), MinExclusive: true, Default: 1.0}, "Overlays & Palettes", "nightly", nil, nil, ""),
	opt(Field{Name: "command_palette_rows", Kind: Int, Min: ptr(1), DefaultNote: "computed"}, "Overlays & Palettes", "20240127-113634-bbcac864", nil, nil, ""),
	opt(Field{Name: "command_palette_fg_color", Kind: Color, Default: "#BFBFBF"}, "Overlays & Palettes", "20230320-124340-559cb7b0", nil, nil, ""),
	opt(Field{Name: "command_palette_bg_color", Kind: Color, Default: "#333333"}, "Overlays & Palettes", "20230320-124340-559cb7b0", nil, nil, ""),
	opt(Field{Name: "char_select_font", Kind: Font, DefaultNote: "= window_frame.font"}, "Overlays & Palettes", "nightly", nil, nil, ""),
	opt(Field{Name: "char_select_font_size", Kind: Float, Min: ptr(0), MinExclusive: true, Default: 18.0}, "Overlays & Palettes", "20220903-194523-3bb1ed61", nil, nil, ""),
	opt(Field{Name: "char_select_fg_color", Kind: Color, Default: "#BFBFBF"}, "Overlays & Palettes", "20230712-072601-f4abf8fd", nil, nil, ""),
	opt(Field{Name: "char_select_bg_color", Kind: Color, Default: "#333333"}, "Overlays & Palettes", "20230712-072601-f4abf8fd", nil, nil, ""),
	opt(Field{Name: "pane_select_font", Kind: Font, DefaultNote: "= window_frame.font"}, "Overlays & Palettes", "nightly", nil, nil, ""),
	opt(Field{Name: "pane_select_font_size", Kind: Float, Min: ptr(0), MinExclusive: true, Default: 36.0}, "Overlays & Palettes", "", nil, nil, ""),
	opt(Field{Name: "pane_select_fg_color", Kind: Color, Default: "#BFBFBF"}, "Overlays & Palettes", "", nil, nil, ""),
	opt(Field{Name: "pane_select_bg_color", Kind: Color, Default: "rgba(0,0,0,0.5)"}, "Overlays & Palettes", "", nil, nil, ""),
	opt(Field{Name: "launcher_alphabet", Kind: String, Default: "1234567890abcdefghilmnopqrstuvwxyz"}, "Overlays & Palettes", "nightly", nil, nil, ""),
	opt(Field{Name: "ui_key_cap_rendering", Kind: Enum, Enum: []string{"UnixLong", "Emacs", "AppleSymbols", "WindowsLong", "WindowsSymbols"}}, "Overlays & Palettes", "20240203-110809-5046fc22", nil, map[string]any{"linux": "UnixLong", "windows": "WindowsSymbols", "macos": "AppleSymbols"}, ""),
	opt(Field{Name: "palette_max_key_assigments_for_action", Kind: Int, Min: ptr(0), Default: 1}, "Overlays & Palettes", "", nil, nil, ""),
}
