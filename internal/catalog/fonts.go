package catalog

var fontsOptions = []Option{
	opt(Field{Name: "font", Kind: Font, DefaultNote: "JetBrains Mono (bundled)"}, "Fonts", "", nil, nil, ""),
	opt(Field{Name: "font_size", Kind: Float, Min: ptr(0), MinExclusive: true, Default: 12.0}, "Fonts", "", nil, nil, ""),
	opt(Field{Name: "line_height", Kind: Float, Min: ptr(0), MinExclusive: true, Default: 1.0}, "Fonts", "", nil, nil, ""),
	opt(Field{Name: "cell_width", Kind: Float, Min: ptr(0), MinExclusive: true, Default: 1.0}, "Fonts", "20220624-141144-bd1b7c5d", nil, nil, ""),
	opt(Field{Name: "font_rules", Kind: List, Fields: StyleRuleFields, Default: []any{}, DefaultNote: "auto-generated bold/italic/dim variants"}, "Fonts", "", nil, nil, ""),
	opt(Field{Name: "font_dirs", Kind: StringList, Default: []any{}}, "Fonts", "", nil, nil, ""),
	opt(Field{Name: "font_locator", Kind: Enum, Enum: []string{"FontConfig", "Gdi", "CoreText", "ConfigDirsOnly"}, DefaultNote: "FontConfig on Linux, Gdi on Windows, CoreText on macOS"}, "Fonts", "", nil, map[string]any{"linux": "FontConfig", "windows": "Gdi", "macos": "CoreText"}, ""),
	opt(Field{Name: "search_font_dirs_for_fallback", Kind: Bool, Default: false}, "Fonts", "", nil, nil, ""),
	opt(Field{Name: "sort_fallback_fonts_by_coverage", Kind: Bool, Default: false}, "Fonts", "", nil, nil, ""),
	opt(Field{Name: "use_cap_height_to_scale_fallback_fonts", Kind: Bool, Default: false}, "Fonts", "20210502-130208-bff6815d", nil, nil, ""),
	opt(Field{Name: "warn_about_missing_glyphs", Kind: Bool, Default: true}, "Fonts", "20210502-130208-bff6815d", nil, nil, ""),
	opt(Field{Name: "ignore_svg_fonts", Kind: Bool, Default: false}, "Fonts", "", nil, nil, ""),
	opt(Field{Name: "dpi", Kind: Float, Min: ptr(0), MinExclusive: true, DefaultNote: "auto-detected"}, "Fonts", "", nil, nil, ""),
	opt(Field{Name: "dpi_by_screen", Kind: FloatMap, Default: map[string]any{}}, "Fonts", "", nil, nil, ""),
}
