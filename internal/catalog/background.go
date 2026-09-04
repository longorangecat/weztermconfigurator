package catalog

var backgroundOptions = []Option{
	opt(Field{Name: "window_background_opacity", Kind: Float01, Default: 1.0}, "Background & Transparency", "20201031-154415-9614e117", nil, nil, ""),
	opt(Field{Name: "window_background_image", Kind: Path}, "Background & Transparency", "20201031-154415-9614e117", nil, nil, ""),
	opt(Field{Name: "window_background_image_hsb", Kind: Struct, Fields: HsbFields}, "Background & Transparency", "20201031-154415-9614e117", nil, nil, ""),
	opt(Field{Name: "window_background_gradient", Kind: Struct, Fields: GradientFields}, "Background & Transparency", "20210814-124438-54e29167", nil, nil, ""),
	opt(Field{Name: "background", Kind: List, Fields: BackgroundLayerFields, Default: []any{}}, "Background & Transparency", "20220624-141144-bd1b7c5d", nil, nil, ""),
	opt(Field{Name: "macos_window_background_blur", Kind: Int, Min: ptr(0), Default: 0}, "Background & Transparency", "20230326-111934-3666303c", []string{"macOS"}, nil, ""),
	opt(Field{Name: "kde_window_background_blur", Kind: Bool, Default: false}, "Background & Transparency", "nightly", []string{"Linux", "KDE", "Wayland"}, nil, "this option has been replaced with `wayland_window_background_blur` and will be removed in a future release"),
	opt(Field{Name: "wayland_window_background_blur", Kind: Bool, Default: false}, "Background & Transparency", "", []string{"Wayland"}, nil, ""),
	opt(Field{Name: "win32_system_backdrop", Kind: Enum, Enum: []string{"Auto", "Disable", "Acrylic", "Mica", "Tabbed"}, Default: "Auto"}, "Background & Transparency", "20230712-072601-f4abf8fd", []string{"Windows"}, nil, ""),
	opt(Field{Name: "win32_acrylic_accent_color", Kind: Color, Default: "rgba(40,40,40,0.004)"}, "Background & Transparency", "20230712-072601-f4abf8fd", []string{"Windows"}, nil, ""),
}
