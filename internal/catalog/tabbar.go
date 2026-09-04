package catalog

var tabBarOptions = []Option{
	opt(Field{Name: "enable_tab_bar", Kind: Bool, Default: true}, "Tab Bar", "", nil, nil, ""),
	opt(Field{Name: "use_fancy_tab_bar", Kind: Bool, Default: true}, "Tab Bar", "20220101-133340-7edc5b5a", nil, nil, ""),
	opt(Field{Name: "tab_bar_at_bottom", Kind: Bool, Default: false}, "Tab Bar", "20210502-130208-bff6815d", nil, nil, ""),
	opt(Field{Name: "hide_tab_bar_if_only_one_tab", Kind: Bool, Default: false}, "Tab Bar", "", nil, nil, ""),
	opt(Field{Name: "show_tab_index_in_tab_bar", Kind: Bool, Default: true}, "Tab Bar", "", nil, nil, ""),
	opt(Field{Name: "show_tabs_in_tab_bar", Kind: Bool, Default: true}, "Tab Bar", "20221119-145034-49b9839f", nil, nil, ""),
	opt(Field{Name: "show_new_tab_button_in_tab_bar", Kind: Bool, Default: true}, "Tab Bar", "20221119-145034-49b9839f", nil, nil, ""),
	opt(Field{Name: "show_close_tab_button_in_tabs", Kind: Bool, Default: true}, "Tab Bar", "nightly", nil, nil, ""),
	opt(Field{Name: "tab_and_split_indices_are_zero_based", Kind: Bool, Default: false}, "Tab Bar", "", nil, nil, ""),
	opt(Field{Name: "tab_max_width", Kind: Int, Min: ptr(1), Default: 16}, "Tab Bar", "", nil, nil, ""),
	opt(Field{Name: "tab_bar_style", Kind: Struct, Fields: TabBarStyleFields}, "Tab Bar", "20210314-114017-04b7cedd", nil, nil, ""),
	opt(Field{Name: "mouse_wheel_scrolls_tabs", Kind: Bool, Default: true}, "Tab Bar", "20230326-111934-3666303c", nil, nil, ""),
	opt(Field{Name: "switch_to_last_active_tab_when_closing_tab", Kind: Bool, Default: false}, "Tab Bar", "20220905-102802-7d4b8249", nil, nil, ""),
}
