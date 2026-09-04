package catalog

var mouseOptions = []Option{
	opt(Field{Name: "mouse_bindings", Kind: Mouse, Default: []any{}}, "Mouse & Selection", "", nil, nil, ""),
	opt(Field{Name: "disable_default_mouse_bindings", Kind: Bool, Default: false}, "Mouse & Selection", "", nil, nil, ""),
	opt(Field{Name: "bypass_mouse_reporting_modifiers", Kind: Flags, Enum: []string{"SHIFT", "CTRL", "ALT", "SUPER"}, EmptyToken: "NONE", Default: "SHIFT"}, "Mouse & Selection", "20210814-124438-54e29167", nil, nil, ""),
	opt(Field{Name: "selection_word_boundary", Kind: String, Default: " \t\n{[}]()\"'`"}, "Mouse & Selection", "20210203-095643-70a364eb", nil, nil, ""),
	opt(Field{Name: "hyperlink_rules", Kind: List, Fields: HyperlinkRuleFields, DefaultNote: "6 built-in rules (raw-Lua default: wezterm.default_hyperlink_rules())"}, "Mouse & Selection", "", nil, nil, ""),
	opt(Field{Name: "quick_select_patterns", Kind: StringList, Default: []any{}}, "Mouse & Selection", "20210502-130208-bff6815d", nil, nil, ""),
	opt(Field{Name: "quick_select_alphabet", Kind: String, Default: "asdfqwerzxcvjklmiuopghtybn"}, "Mouse & Selection", "20210502-130208-bff6815d", nil, nil, ""),
	opt(Field{Name: "quick_select_remove_styling", Kind: Bool, Default: false}, "Mouse & Selection", "nightly", nil, nil, ""),
	opt(Field{Name: "disable_default_quick_select_patterns", Kind: Bool, Default: false}, "Mouse & Selection", "20210502-130208-bff6815d", nil, nil, ""),
}
