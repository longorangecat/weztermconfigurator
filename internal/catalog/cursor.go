package catalog

var cursorOptions = []Option{
	opt(Field{Name: "default_cursor_style", Kind: Enum, Enum: []string{"SteadyBlock", "BlinkingBlock", "SteadyUnderline", "BlinkingUnderline", "SteadyBar", "BlinkingBar"}, Default: "SteadyBlock"}, "Cursor & Blinking", "", nil, nil, ""),
	opt(Field{Name: "cursor_blink_rate", Kind: Int, Min: ptr(0), Default: 800}, "Cursor & Blinking", "", nil, nil, ""),
	opt(Field{Name: "cursor_blink_ease_in", Kind: Union, Enum: EasingNames, Fields: EasingFields, Default: "Linear"}, "Cursor & Blinking", "20220319-142410-0fcdea07", nil, nil, ""),
	opt(Field{Name: "cursor_blink_ease_out", Kind: Union, Enum: EasingNames, Fields: EasingFields, Default: "Linear"}, "Cursor & Blinking", "20220319-142410-0fcdea07", nil, nil, ""),
	opt(Field{Name: "text_blink_rate", Kind: Int, Min: ptr(0), Default: 500}, "Cursor & Blinking", "20210814-124438-54e29167", nil, nil, ""),
	opt(Field{Name: "text_blink_ease_in", Kind: Union, Enum: EasingNames, Fields: EasingFields, Default: "Linear"}, "Cursor & Blinking", "20220319-142410-0fcdea07", nil, nil, ""),
	opt(Field{Name: "text_blink_ease_out", Kind: Union, Enum: EasingNames, Fields: EasingFields, Default: "Linear"}, "Cursor & Blinking", "20220319-142410-0fcdea07", nil, nil, ""),
	opt(Field{Name: "text_blink_rate_rapid", Kind: Int, Min: ptr(0), Default: 250}, "Cursor & Blinking", "20210814-124438-54e29167", nil, nil, ""),
	opt(Field{Name: "text_blink_rapid_ease_in", Kind: Union, Enum: EasingNames, Fields: EasingFields, Default: "Linear"}, "Cursor & Blinking", "20220319-142410-0fcdea07", nil, nil, ""),
	opt(Field{Name: "text_blink_rapid_ease_out", Kind: Union, Enum: EasingNames, Fields: EasingFields, Default: "Linear"}, "Cursor & Blinking", "20220319-142410-0fcdea07", nil, nil, ""),
	opt(Field{Name: "hide_mouse_cursor_when_typing", Kind: Bool, Default: true}, "Cursor & Blinking", "20230320-124340-559cb7b0", nil, nil, ""),
}
