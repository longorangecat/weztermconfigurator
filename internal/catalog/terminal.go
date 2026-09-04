package catalog

var terminalOptions = []Option{
	opt(Field{Name: "enable_kitty_graphics", Kind: Bool, Default: true}, "Terminal & Security", "", nil, nil, ""),
	opt(Field{Name: "enable_title_reporting", Kind: Bool, Default: false}, "Terminal & Security", "", nil, nil, ""),
	opt(Field{Name: "enable_checksum_rectangular_area", Kind: Bool, Default: false}, "Terminal & Security", "", nil, nil, ""),
	opt(Field{Name: "allow_download_protocols", Kind: Bool, Default: true}, "Terminal & Security", "", nil, nil, ""),
	opt(Field{Name: "log_unknown_escape_sequences", Kind: Bool, Default: false}, "Terminal & Security", "20230320-124340-559cb7b0", nil, nil, ""),
	opt(Field{Name: "status_update_interval", Kind: Int, Min: ptr(1), Default: 1000}, "Terminal & Security", "20210314-114017-04b7cedd", nil, nil, ""),
}
