package catalog

var windowOptions = []Option{
	opt(Field{Name: "initial_cols", Kind: Int, Min: ptr(1), Max: ptr(65535), Default: 80}, "Window", "", nil, nil, ""),
	opt(Field{Name: "initial_rows", Kind: Int, Min: ptr(1), Max: ptr(65535), Default: 24}, "Window", "", nil, nil, ""),
	opt(Field{Name: "window_decorations", Kind: Flags, Enum: []string{"TITLE", "RESIZE", "INTEGRATED_BUTTONS", "MACOS_FORCE_DISABLE_SHADOW", "MACOS_FORCE_ENABLE_SHADOW", "MACOS_FORCE_SQUARE_CORNERS", "MACOS_USE_BACKGROUND_COLOR_AS_TITLEBAR_COLOR"}, EmptyToken: "NONE", Default: "TITLE|RESIZE"}, "Window", "20210314-114017-04b7cedd", nil, nil, ""),
	opt(Field{Name: "integrated_title_buttons", Kind: StringList, Enum: []string{"Hide", "Maximize", "Close"}, Default: []any{"Hide", "Maximize", "Close"}}, "Window", "20230408-112425-69ae8472", nil, nil, ""),
	opt(Field{Name: "integrated_title_button_alignment", Kind: Enum, Enum: []string{"Right", "Left"}, Default: "Right"}, "Window", "20230408-112425-69ae8472", nil, nil, ""),
	opt(Field{Name: "integrated_title_button_style", Kind: Enum, Enum: []string{"Windows", "Gnome", "MacOsNative"}}, "Window", "20230408-112425-69ae8472", nil, map[string]any{"macos": "MacOsNative", "linux": "Windows", "windows": "Windows"}, ""),
	opt(Field{Name: "integrated_title_button_color", Kind: Color, Default: "Auto"}, "Window", "20230408-112425-69ae8472", nil, nil, ""),
	opt(Field{Name: "window_frame", Kind: Struct, Fields: WindowFrameFields}, "Window", "20210814-124438-54e29167", nil, nil, ""),
	opt(Field{Name: "window_padding", Kind: Struct, Fields: WindowPaddingFields, Default: map[string]any{"left": "1cell", "right": "1cell", "top": "0.5cell", "bottom": "0.5cell"}}, "Window", "20211204-082213-a66c61ee9", nil, nil, ""),
	opt(Field{Name: "window_content_alignment", Kind: Struct, Fields: ContentAlignmentFields, Default: map[string]any{"horizontal": "Left", "vertical": "Top"}}, "Window", "nightly", nil, nil, ""),
	opt(Field{Name: "adjust_window_size_when_changing_font_size", Kind: Bool, DefaultNote: "auto: false on tiling DEs"}, "Window", "20210203-095643-70a364eb", nil, nil, ""),
	opt(Field{Name: "use_resize_increments", Kind: Bool, Default: false}, "Window", "20211204-082213-a66c61ee9", []string{"Linux", "macOS"}, nil, ""),
	opt(Field{Name: "tiling_desktop_environments", Kind: StringList, Default: []any{"X11 LG3D", "X11 Qtile", "X11 awesome", "X11 bspwm", "X11 dwm", "X11 i3", "X11 xmonad"}}, "Window", "20230712-072601-f4abf8fd", []string{"Linux"}, nil, ""),
	opt(Field{Name: "native_macos_fullscreen_mode", Kind: Bool, Default: false}, "Window", "", []string{"macOS"}, nil, ""),
	opt(Field{Name: "macos_fullscreen_extend_behind_notch", Kind: Bool, Default: false}, "Window", "nightly", []string{"macOS"}, nil, ""),
	opt(Field{Name: "xcursor_theme", Kind: String}, "Window", "", []string{"X11", "Wayland"}, nil, ""),
	opt(Field{Name: "xcursor_size", Kind: Int, Min: ptr(1)}, "Window", "", []string{"X11", "Wayland"}, nil, ""),
}
