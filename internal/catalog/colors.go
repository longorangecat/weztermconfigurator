package catalog

var colorsOptions = []Option{
	opt(Field{Name: "color_scheme", Kind: SchemeName, DefaultNote: "default palette"}, "Colors", "", nil, nil, ""),
	opt(Field{Name: "colors", Kind: Palette}, "Colors", "", nil, nil, ""),
	opt(Field{Name: "color_schemes", Kind: NamedPalettes, Default: map[string]any{}}, "Colors", "", nil, nil, ""),
	opt(Field{Name: "color_scheme_dirs", Kind: StringList, Default: []any{}}, "Colors", "", nil, nil, ""),
	opt(Field{Name: "bold_brightens_ansi_colors", Kind: Enum, Enum: []string{"BrightAndBold", "BrightOnly", "No"}, Default: "BrightAndBold"}, "Colors", "", nil, nil, ""),
	opt(Field{Name: "foreground_text_hsb", Kind: Struct, Fields: HsbFields, Default: map[string]any{"hue": 1.0, "saturation": 1.0, "brightness": 1.0}}, "Colors", "20210314-114017-04b7cedd", nil, nil, ""),
	opt(Field{Name: "inactive_pane_hsb", Kind: Struct, Fields: HsbFields, Default: map[string]any{"hue": 1.0, "saturation": 0.9, "brightness": 0.8}}, "Colors", "", nil, nil, ""),
	opt(Field{Name: "text_min_contrast_ratio", Kind: Float, Min: ptr(1)}, "Colors", "nightly", nil, nil, ""),
	opt(Field{Name: "text_background_opacity", Kind: Float01, Default: 1.0}, "Colors", "", nil, nil, ""),
	opt(Field{Name: "force_reverse_video_cursor", Kind: Bool, Default: false}, "Colors", "20210502-130208-bff6815d", nil, nil, ""),
	opt(Field{Name: "reverse_video_cursor_min_contrast", Kind: Float, Min: ptr(1), Default: 2.5}, "Colors", "nightly", nil, nil, ""),
}
