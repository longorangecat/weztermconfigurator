package catalog

var keyboardOptions = []Option{
	opt(Field{Name: "enable_csi_u_key_encoding", Kind: Bool, Default: false}, "Keyboard & Input", "", nil, nil, ""),
	opt(Field{Name: "enable_kitty_keyboard", Kind: Bool, Default: false}, "Keyboard & Input", "20220624-141144-bd1b7c5d", nil, nil, ""),
	opt(Field{Name: "allow_win32_input_mode", Kind: Bool, Default: true}, "Keyboard & Input", "20220319-142410-0fcdea07", []string{"Windows"}, nil, ""),
	opt(Field{Name: "use_dead_keys", Kind: Bool, Default: true}, "Keyboard & Input", "", nil, nil, ""),
	opt(Field{Name: "send_composed_key_when_left_alt_is_pressed", Kind: Bool, Default: false}, "Keyboard & Input", "", nil, nil, ""),
	opt(Field{Name: "send_composed_key_when_right_alt_is_pressed", Kind: Bool, Default: true}, "Keyboard & Input", "", nil, nil, ""),
	opt(Field{Name: "treat_left_ctrlalt_as_altgr", Kind: Bool, Default: false}, "Keyboard & Input", "20210314-114017-04b7cedd", nil, nil, ""),
	opt(Field{Name: "swap_backspace_and_delete", Kind: Bool, Default: false}, "Keyboard & Input", "", nil, nil, ""),
	opt(Field{Name: "debug_key_events", Kind: Bool, Default: false}, "Keyboard & Input", "", nil, nil, ""),
	opt(Field{Name: "use_ime", Kind: Bool, Default: true}, "Keyboard & Input", "", nil, nil, ""),
	opt(Field{Name: "xim_im_name", Kind: String}, "Keyboard & Input", "20220101-133340-7edc5b5a", []string{"X11"}, nil, ""),
	opt(Field{Name: "ime_preedit_rendering", Kind: Enum, Enum: []string{"Builtin", "System"}, Default: "Builtin"}, "Keyboard & Input", "20220624-141144-bd1b7c5d", []string{"Linux", "Windows"}, nil, ""),
	opt(Field{Name: "macos_forward_to_ime_modifier_mask", Kind: Flags, Enum: []string{"SHIFT", "CTRL", "ALT", "SUPER"}, EmptyToken: "NONE", Default: "SHIFT"}, "Keyboard & Input", "20230408-112425-69ae8472", []string{"macOS"}, nil, ""),
	opt(Field{Name: "canonicalize_pasted_newlines", Kind: Enum, Enum: []string{"None", "LineFeed", "CarriageReturn", "CarriageReturnAndLineFeed"}}, "Keyboard & Input", "20211204-082213-a66c61ee9", nil, map[string]any{"windows": "CarriageReturnAndLineFeed", "linux": "CarriageReturn", "macos": "CarriageReturn"}, ""),
	opt(Field{Name: "quote_dropped_files", Kind: Enum, Enum: []string{"None", "SpacesOnly", "Posix", "Windows", "WindowsAlwaysQuoted"}}, "Keyboard & Input", "20220624-141144-bd1b7c5d", nil, map[string]any{"windows": "Windows", "linux": "SpacesOnly", "macos": "SpacesOnly"}, ""),
}
