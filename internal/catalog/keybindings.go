package catalog

var keyBindingOptions = []Option{
	opt(Field{Name: "keys", Kind: Keys, Default: []any{}}, "Key Bindings", "", nil, nil, ""),
	opt(Field{Name: "key_tables", Kind: KeyTables, Default: map[string]any{}}, "Key Bindings", "", nil, nil, ""),
	opt(Field{Name: "leader", Kind: Leader}, "Key Bindings", "", nil, nil, ""),
	opt(Field{Name: "disable_default_key_bindings", Kind: Bool, Default: false}, "Key Bindings", "", nil, nil, ""),
	opt(Field{Name: "key_map_preference", Kind: Enum, Enum: []string{"Mapped", "Physical"}, Default: "Mapped"}, "Key Bindings", "20220408-101518-b908e2dd", nil, nil, ""),
}
