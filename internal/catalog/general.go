package catalog

// opt builds a positional Option literal: Option{field, category, since, tags, byOS, deprecated}.
// The Field's Doc is filled from optionDocs (the option matrix help text).
func opt(f Field, category, since string, tags []string, byOS map[string]any, deprecated string) Option {
	f.Doc = optionDocs[f.Name]
	return Option{Field: f, Category: category, Since: since, Tags: tags, DefaultByOS: byOS, Deprecated: deprecated}
}

var generalOptions = []Option{
	opt(Field{Name: "default_prog", Kind: StringList, DefaultNote: "user's shell; Windows: %COMSPEC% or cmd.exe"}, "General & Startup", "", nil, nil, ""),
	opt(Field{Name: "default_cwd", Kind: Dir, DefaultNote: "home dir"}, "General & Startup", "20210203-095643-70a364eb", nil, nil, ""),
	opt(Field{Name: "default_domain", Kind: String, DefaultNote: "local"}, "General & Startup", "20220319-142410-0fcdea07", nil, nil, ""),
	opt(Field{Name: "default_workspace", Kind: String, DefaultNote: "default"}, "General & Startup", "20220319-142410-0fcdea07", nil, nil, ""),
	opt(Field{Name: "default_gui_startup_args", Kind: StringList, Default: []any{"start"}}, "General & Startup", "20220101-133340-7edc5b5a", nil, nil, ""),
	opt(Field{Name: "launch_menu", Kind: List, Fields: SpawnCommandFields}, "General & Startup", "20200503-171512-b13ef15f", nil, nil, ""),
	opt(Field{Name: "set_environment_variables", Kind: StringMap, Default: map[string]any{}}, "General & Startup", "", nil, nil, ""),
	opt(Field{Name: "term", Kind: String, Default: "xterm-256color"}, "General & Startup", "", nil, nil, ""),
	opt(Field{Name: "exit_behavior", Kind: Enum, Enum: []string{"Close", "CloseOnCleanExit", "Hold"}, Default: "Close"}, "General & Startup", "20210314-114017-04b7cedd", nil, nil, ""),
	opt(Field{Name: "exit_behavior_messaging", Kind: Enum, Enum: []string{"Verbose", "Brief", "Terse", "None"}, Default: "Verbose"}, "General & Startup", "20230712-072601-f4abf8fd", nil, nil, ""),
	opt(Field{Name: "clean_exit_codes", Kind: IntList, Default: []any{}, DefaultNote: "0 always clean"}, "General & Startup", "20220624-141144-bd1b7c5d", nil, nil, ""),
	opt(Field{Name: "skip_close_confirmation_for_processes_named", Kind: StringList, Default: []any{"bash", "sh", "zsh", "fish", "tmux", "nu", "nu.exe", "cmd.exe", "pwsh.exe", "powershell.exe"}}, "General & Startup", "20210404-112810-b63a949d", nil, nil, ""),
	opt(Field{Name: "window_close_confirmation", Kind: Enum, Enum: []string{"AlwaysPrompt", "NeverPrompt"}, Default: "AlwaysPrompt"}, "General & Startup", "", nil, nil, ""),
	opt(Field{Name: "quit_when_all_windows_are_closed", Kind: Bool, Default: true}, "General & Startup", "20230320-124340-559cb7b0", nil, nil, ""),
	opt(Field{Name: "prefer_to_spawn_tabs", Kind: Bool, Default: false}, "General & Startup", "20240203-110809-5046fc22", nil, nil, ""),
	opt(Field{Name: "detect_password_input", Kind: Bool, Default: true}, "General & Startup", "20220903-194523-3bb1ed61", []string{"Unix"}, nil, ""),
	opt(Field{Name: "enq_answerback", Kind: String, Default: ""}, "General & Startup", "", nil, nil, ""),
	opt(Field{Name: "automatically_reload_config", Kind: Bool, Default: true}, "General & Startup", "20201031-154415-9614e117", nil, nil, ""),
	opt(Field{Name: "check_for_updates", Kind: Bool, Default: true, DefaultNote: "false in distro builds"}, "General & Startup", "", nil, nil, ""),
	opt(Field{Name: "check_for_updates_interval_seconds", Kind: Int, Min: ptr(1), Default: 86400}, "General & Startup", "", nil, nil, ""),
	opt(Field{Name: "show_update_window", Kind: Bool, Default: false}, "General & Startup", "", nil, nil, "this option no longer does anything and will be removed in a future release"),
	opt(Field{Name: "periodic_stat_logging", Kind: Int, Min: ptr(0), Default: 0}, "General & Startup", "", nil, nil, ""),
	opt(Field{Name: "ulimit_nofile", Kind: Int, Min: ptr(1), Default: 2048}, "General & Startup", "20230408-112425-69ae8472", []string{"Linux", "macOS"}, nil, ""),
	opt(Field{Name: "ulimit_nproc", Kind: Int, Min: ptr(1), Default: 2048}, "General & Startup", "20230408-112425-69ae8472", []string{"Linux", "macOS"}, nil, ""),
}
