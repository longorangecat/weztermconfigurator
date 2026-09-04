package catalog

var gpuOptions = []Option{
	opt(Field{Name: "front_end", Kind: Enum, Enum: []string{"OpenGL", "WebGpu", "Software"}, Default: "OpenGL"}, "Rendering & GPU", "", nil, nil, ""),
	opt(Field{Name: "webgpu_power_preference", Kind: Enum, Enum: []string{"LowPower", "HighPerformance"}, Default: "LowPower"}, "Rendering & GPU", "20221119-145034-49b9839f", nil, nil, ""),
	opt(Field{Name: "webgpu_force_fallback_adapter", Kind: Bool, Default: false}, "Rendering & GPU", "20221119-145034-49b9839f", nil, nil, ""),
	opt(Field{Name: "webgpu_preferred_adapter", Kind: Struct, Fields: GpuInfoFields}, "Rendering & GPU", "20221119-145034-49b9839f", nil, nil, ""),
	opt(Field{Name: "prefer_egl", Kind: Bool, DefaultNote: "false on Windows, true elsewhere"}, "Rendering & GPU", "", nil, map[string]any{"windows": false, "linux": true, "macos": true}, ""),
	opt(Field{Name: "enable_wayland", Kind: Bool, Default: true}, "Rendering & GPU", "", []string{"Linux"}, nil, ""),
	opt(Field{Name: "enable_zwlr_output_manager", Kind: Bool, Default: false}, "Rendering & GPU", "", []string{"Wayland"}, nil, ""),
	opt(Field{Name: "max_fps", Kind: Int, Min: ptr(1), Default: 60}, "Rendering & GPU", "", nil, nil, ""),
	opt(Field{Name: "animation_fps", Kind: Int, Min: ptr(1), Max: ptr(255), Default: 10}, "Rendering & GPU", "20220319-142410-0fcdea07", nil, nil, ""),
	opt(Field{Name: "shape_cache_size", Kind: Int, Min: ptr(1), Default: 1024}, "Rendering & GPU", "", nil, nil, ""),
	opt(Field{Name: "line_state_cache_size", Kind: Int, Min: ptr(1), Default: 1024}, "Rendering & GPU", "", nil, nil, ""),
	opt(Field{Name: "line_quad_cache_size", Kind: Int, Min: ptr(1), Default: 1024}, "Rendering & GPU", "", nil, nil, ""),
	opt(Field{Name: "line_to_ele_shape_cache_size", Kind: Int, Min: ptr(1), Default: 1024}, "Rendering & GPU", "", nil, nil, ""),
	opt(Field{Name: "glyph_cache_image_cache_size", Kind: Int, Min: ptr(1), Default: 256}, "Rendering & GPU", "", nil, nil, ""),
}
