package catalog

var bellOptions = []Option{
	opt(Field{Name: "audible_bell", Kind: Enum, Enum: []string{"SystemBeep", "Disabled"}, Default: "SystemBeep"}, "Bell & Notifications", "20211204-082213-a66c61ee9", nil, nil, ""),
	opt(Field{Name: "visual_bell", Kind: Struct, Fields: VisualBellFields, Default: map[string]any{"fade_in_duration_ms": 0.0, "fade_out_duration_ms": 0.0, "fade_in_function": "Ease", "fade_out_function": "Ease", "target": "BackgroundColor"}}, "Bell & Notifications", "20211204-082213-a66c61ee9", nil, nil, ""),
	opt(Field{Name: "notification_handling", Kind: Enum, Enum: []string{"AlwaysShow", "NeverShow", "SuppressFromFocusedPane", "SuppressFromFocusedTab", "SuppressFromFocusedWindow"}, Default: "AlwaysShow"}, "Bell & Notifications", "20240127-113634-bbcac864", nil, nil, ""),
}
