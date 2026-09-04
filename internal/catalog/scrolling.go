package catalog

var scrollingOptions = []Option{
	opt(Field{Name: "scrollback_lines", Kind: Int, Min: ptr(0), Max: ptr(999999999), Default: 3500}, "Scrolling & Panes", "", nil, nil, ""),
	opt(Field{Name: "enable_scroll_bar", Kind: Bool, Default: false}, "Scrolling & Panes", "", nil, nil, ""),
	opt(Field{Name: "min_scroll_bar_height", Kind: Dimension, Default: "0.5cell"}, "Scrolling & Panes", "20220624-141144-bd1b7c5d", nil, nil, ""),
	opt(Field{Name: "scroll_to_bottom_on_input", Kind: Bool, Default: true}, "Scrolling & Panes", "", nil, nil, ""),
	opt(Field{Name: "alternate_buffer_wheel_scroll_speed", Kind: Int, Min: ptr(0), Max: ptr(255), Default: 3}, "Scrolling & Panes", "20210203-095643-70a364eb", nil, nil, ""),
	opt(Field{Name: "pane_focus_follows_mouse", Kind: Bool, Default: false}, "Scrolling & Panes", "20210502-130208-bff6815d", nil, nil, ""),
	opt(Field{Name: "swallow_mouse_click_on_pane_focus", Kind: Bool, Default: false}, "Scrolling & Panes", "20210502-130208-bff6815d", nil, nil, ""),
	opt(Field{Name: "swallow_mouse_click_on_window_focus", Kind: Bool}, "Scrolling & Panes", "20220319-142410-0fcdea07", nil, map[string]any{"macos": true, "linux": false, "windows": false}, ""),
	opt(Field{Name: "unzoom_on_switch_pane", Kind: Bool, Default: true}, "Scrolling & Panes", "20211204-082213-a66c61ee9", nil, nil, ""),
}
