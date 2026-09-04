package catalog

var muxOptions = []Option{
	opt(Field{Name: "unix_domains", Kind: List, Fields: UnixDomainFields, Default: []any{map[string]any{"name": "unix"}}, DefaultNote: "one default domain"}, "Multiplexing & Domains", "", nil, nil, ""),
	opt(Field{Name: "ssh_domains", Kind: List, Fields: SshDomainFields, DefaultNote: "derived from ~/.ssh/config"}, "Multiplexing & Domains", "", nil, nil, ""),
	opt(Field{Name: "ssh_backend", Kind: Enum, Enum: []string{"LibSsh", "Ssh2"}, Default: "LibSsh"}, "Multiplexing & Domains", "20211204-082213-a66c61ee9", nil, nil, ""),
	opt(Field{Name: "tls_clients", Kind: List, Fields: TlsClientFields, Default: []any{}}, "Multiplexing & Domains", "", nil, nil, ""),
	opt(Field{Name: "tls_servers", Kind: List, Fields: TlsServerFields, Default: []any{}}, "Multiplexing & Domains", "", nil, nil, ""),
	opt(Field{Name: "wsl_domains", Kind: List, Fields: WslDomainFields, DefaultNote: "derived from `wsl -l -v`"}, "Multiplexing & Domains", "20220319-142410-0fcdea07", []string{"Windows", "WSL"}, nil, ""),
	opt(Field{Name: "exec_domains", Kind: List, Fields: ExecDomainFields, Default: []any{}}, "Multiplexing & Domains", "", nil, nil, ""),
	opt(Field{Name: "serial_ports", Kind: List, Fields: SerialDomainFields, Default: []any{}}, "Multiplexing & Domains", "20230408-112425-69ae8472", nil, nil, ""),
	opt(Field{Name: "default_mux_server_domain", Kind: String, DefaultNote: "local"}, "Multiplexing & Domains", "20230712-072601-f4abf8fd", nil, nil, ""),
	opt(Field{Name: "daemon_options", Kind: Struct, Fields: DaemonOptionsFields, DefaultNote: "nil paths (runtime dir defaults)"}, "Multiplexing & Domains", "", []string{"Linux", "X11", "Wayland"}, nil, ""),
	opt(Field{Name: "mux_enable_ssh_agent", Kind: Bool, Default: true}, "Multiplexing & Domains", "nightly", nil, nil, ""),
	opt(Field{Name: "default_ssh_auth_sock", Kind: String, DefaultNote: "$SSH_AUTH_SOCK"}, "Multiplexing & Domains", "nightly", nil, nil, ""),
	opt(Field{Name: "mux_env_remove", Kind: StringList, Default: []any{"SSH_AUTH_SOCK", "SSH_CLIENT", "SSH_CONNECTION"}}, "Multiplexing & Domains", "20211204-082213-a66c61ee9", nil, nil, ""),
	opt(Field{Name: "mux_output_parser_buffer_size", Kind: Int, Min: ptr(1), Default: 131072}, "Multiplexing & Domains", "", nil, nil, ""),
	opt(Field{Name: "mux_output_parser_coalesce_delay_ms", Kind: Int, Min: ptr(0), Default: 3}, "Multiplexing & Domains", "", nil, nil, ""),
	opt(Field{Name: "ratelimit_mux_line_prefetches_per_second", Kind: Int, Min: ptr(1), Default: 50}, "Multiplexing & Domains", "", nil, nil, ""),
}
