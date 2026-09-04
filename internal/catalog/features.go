package catalog

import (
	"fmt"
	"strings"
)

// Feature is a one-click community Lua feature. An enabled feature emits a
// Lua block (generated from Params) into wezterm.lua before Custom Lua.
type Feature struct {
	ID     string
	Name   string
	Doc    string
	Source string
	Params []FeatureParam
	Emit   func(p map[string]string) string
}

// FeatureParam is one editable field on a feature card.
type FeatureParam struct {
	Name    string
	Label   string
	Kind    string   // "string" | "bool" | "text" | "select"
	Options []string // select only
	Default string
}

// IsOn reports whether the feature is enabled: true when any param differs
// from its default, or the explicit "__on" marker is set (parameterless use).
func (f *Feature) IsOn(p map[string]string) bool {
	if p == nil {
		return false
	}
	if p["__on"] == "1" {
		return true
	}
	for _, prm := range f.Params {
		if p[prm.Name] != "" && p[prm.Name] != prm.Default {
			return true
		}
	}
	return false
}

// fq renders a Lua single-quoted string literal.
func fq(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return `'` + s + `'`
}

// fs returns the param value or def when blank.
func fs(p map[string]string, k, def string) string {
	if v := strings.TrimSpace(p[k]); v != "" {
		return v
	}
	return def
}

// fb parses a truthy param.
func fb(p map[string]string, k string) bool {
	switch strings.TrimSpace(strings.ToLower(p[k])) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// fkeys renders a keybinding param ("CTRL|SHIFT|T" style) as a
// key/mods pair plus the event emit helper binding, or "" when unset.
func fkeys(p map[string]string, k, event string) string {
	spec := strings.TrimSpace(p[k])
	if spec == "" {
		return ""
	}
	parts := strings.Split(spec, "|")
	key := strings.ToUpper(parts[len(parts)-1])
	mods := parts[:len(parts)-1]
	for i, m := range mods {
		mods[i] = strings.ToUpper(strings.TrimSpace(m))
	}
	m := strings.Join(mods, "|")
	if m == "" {
		m = "NONE"
	}
	return fmt.Sprintf("wezterm.on('format-event-%s', function() end)\nconfig.keys = { table.unpack(config.keys or {}), { key = %s, mods = %s, action = wezterm.action.EmitEvent(%s) } }\n",
		event, fq(key), fq(m), fq(event))
}

// keyEmit is the canonical pattern: an EmitEvent binding + handler pair.
func keyBind(p map[string]string, key, event string) string {
	spec := strings.TrimSpace(p[key])
	if spec == "" {
		return ""
	}
	parts := strings.Split(spec, "|")
	k := strings.ToUpper(parts[len(parts)-1])
	mods := make([]string, 0, len(parts))
	for _, m := range parts[:len(parts)-1] {
		if m = strings.ToUpper(strings.TrimSpace(m)); m != "" {
			mods = append(mods, m)
		}
	}
	mm := "NONE"
	if len(mods) > 0 {
		mm = strings.Join(mods, "|")
	}
	return fmt.Sprintf("config.keys = config.keys or {}\ntable.insert(config.keys, { key = %s, mods = %s, action = wezterm.action.EmitEvent(%s) })\n",
		fq(k), fq(mm), fq(event))
}

var Features = []Feature{
	{
		ID:     "tab_title_powerline",
		Name:   "Powerline tab titles",
		Doc:    "Powerline separators, per-process nerd-font icons, tab index, truncation, hover/active colors.",
		Source: "https://github.com/KevinSilvester/wezterm-config/blob/master/events/tab-title.lua",
		Params: []FeatureParam{
			{Name: "max_width", Label: "Max width (cells)", Kind: "string", Default: "40"},
			{Name: "icons", Label: "Process icons", Kind: "bool", Default: "1"},
			{Name: "index", Label: "Tab index prefix", Kind: "bool", Default: "1"},
		},
		Emit: func(p map[string]string) string {
			useIcons := fb(p, "icons")
			useIndex := fb(p, "index")
			maxw := fs(p, "max_width", "40")
			var b strings.Builder
			b.WriteString("local function cfg_tab_title(tab)\n")
			b.WriteString("  local title = tab.tab_title\n")
			b.WriteString("  if title and #title > 0 then return title end\n")
			b.WriteString("  local pane = tab.active_pane\n")
			b.WriteString("  local proc = pane.foreground_process_name or ''\n")
			b.WriteString("  proc = proc:gsub('^.*/', '')\n")
			if useIcons {
				b.WriteString("  local icons = { bash = '\\u{f489} ', zsh = '\\u{f489} ', fish = '\\u{f489} ', nvim = '\\u{e62b} ', vim = '\\u{e62b} ', ssh = '\\u{f489} ', top = '\\u{f13d} ', htop = '\\u{f13d} ', btm = '\\u{f13d} ', python = '\\u{e606} ', node = '\\u{e718} ' }\n")
				b.WriteString("  local icon = icons[proc] or ''\n")
			}
			b.WriteString("  local idx = ''\n")
			if useIndex {
				b.WriteString("  idx = tostring(tab.tab_index + 1) .. ' '\n")
			}
			b.WriteString("  local t = icon .. idx .. (pane.title ~= '' and pane.title or proc)\n")
			b.WriteString("  return wezterm.truncate_right(t, " + maxw + ")\n")
			b.WriteString("end\n\n")
			b.WriteString("wezterm.on('format-tab-title', function(tab, tabs, panes, _config, hover, max_width)\n")
			b.WriteString("  local edge = '#333333'\n  local bg = '#4a4a4a'\n  local fg = '#c0c0c0'\n")
			b.WriteString("  if tab.is_active then bg = '#2b2042'; fg = '#ffffff'\n  elseif hover then bg = '#5a5a5a' end\n")
			b.WriteString("  local title = cfg_tab_title(tab)\n")
			b.WriteString("  if tab.is_active and #title > 0 then title = '\\u{25b6} ' .. title end\n")
			b.WriteString("  title = ' ' .. title .. ' '\n")
			b.WriteString("  if wezterm.column_width(title) > max_width then title = wezterm.truncate_right(title, max_width - 1) .. '\\u{2026}' end\n")
			b.WriteString("  return {{ { Background = { Color = edge } }, { Text = '\\u{e0b6}' },\n")
			b.WriteString("    { Background = { Color = bg } }, { Foreground = { Color = fg } }, { Text = title },\n")
			b.WriteString("    { Background = { Color = edge } }, { Foreground = { Color = bg } }, { Text = '\\u{e0b4}' } }}\n")
			b.WriteString("end)\n")
			return b.String()
		},
	},
	{
		ID:     "tab_unseen_output",
		Name:   "Unseen-output tab badge",
		Doc:    "Colored dot badge on background tabs that produced output since last view.",
		Source: "https://github.com/KevinSilvester/wezterm-config/blob/master/events/tab-title.lua",
		Params: []FeatureParam{
			{Name: "color", Label: "Badge color", Kind: "string", Default: "#d08770"},
		},
		Emit: func(p map[string]string) string {
			c := fs(p, "color", "#d08770")
			return fmt.Sprintf(`wezterm.on('format-tab-title', function(tab, tabs, panes, config, hover, max_width)
  local dot = ''
  if tab.has_unseen_output and not tab.is_active then
    dot = { { Foreground = { Color = %s } }, { Text = ' \\u{25cf}' } }
  end
  local title = ' ' .. (tab.tab_title ~= '' and tab.tab_title or tab.active_pane.title) .. ' '
  return {{ { Text = title }, dot }}
end)`, fq(c))
		},
	},
	{
		ID:     "tab_cwd_title",
		Name:   "Cwd + hostname tab titles",
		Doc:    "tmux-style tab titles showing the working directory (and optionally hostname).",
		Source: "https://github.com/yutkat/dotfiles/blob/main/.config/wezterm/on.lua",
		Params: []FeatureParam{
			{Name: "depth", Label: "Path depth (components)", Kind: "string", Default: "1"},
			{Name: "hostname", Label: "Show hostname", Kind: "bool", Default: "1"},
		},
		Emit: func(p map[string]string) string {
			host := fb(p, "hostname")
			h := ""
			if host {
				h = "    host = uri.host or ''\n    if host ~= '' then parts[#parts+1] = host end\n"
			}
			return fmt.Sprintf(`local function basename(s)
  return string.gsub(s or '', '(.*[/\\])(.*)', '%%2')
end

wezterm.on('format-tab-title', function(tab, tabs, panes, config, hover, max_width)
  local pane = tab.active_pane
  local uri = pane.current_working_dir and pane.current_working_dir.file_path or ''
  local parts = {}
  local depth = %s
  local comps = {}
  for comp in string.gmatch(uri, '[^/\\]+') do comps[#comps+1] = comp end
  local start = math.max(1, #comps - depth + 1)
  for i = start, #comps do parts[#parts+1] = comps[i] end
%s  local title = table.concat(parts, '/')
  if title == '' then title = pane.title end
  title = ' ' .. tostring(tab.tab_index + 1) .. ': ' .. title .. ' '
  return {{ { Text = title } }}
end)`, fs(p, "depth", "1"), h)
		},
	},
	{
		ID:     "right_status_powerline",
		Name:   "Powerline right-status bar",
		Doc:    "Right status with powerline segments: clock, battery, and current working directory.",
		Source: "https://github.com/KevinSilvester/wezterm-config/blob/master/events/right-status.lua",
		Params: []FeatureParam{
			{Name: "date_format", Label: "Date format (strftime)", Kind: "string", Default: "%a %H:%M"},
			{Name: "battery", Label: "Show battery", Kind: "bool", Default: "1"},
			{Name: "cwd", Label: "Show cwd", Kind: "bool", Default: "1"},
		},
		Emit: func(p map[string]string) string {
			var segs strings.Builder
			segs.WriteString("  table.insert(cells, { Text = wezterm.strftime('" + fs(p, "date_format", "%a %H:%M") + "') })\n")
			if fb(p, "battery") {
				segs.WriteString("  for _, b in ipairs(wezterm.battery_info()) do\n")
				segs.WriteString("    local icon = b.state == 'Charging' and '\\u{f0e7}' or '\\u{f240}'\n")
				segs.WriteString("    table.insert(cells, { Text = string.format('%%s %%%%d%%%%', icon, math.floor(b.state_of_charge * 100)) })\n")
				segs.WriteString("  end\n")
			}
			cwd := ""
			if fb(p, "cwd") {
				cwd = "  local cwd = pane.current_working_dir and pane.current_working_dir.file_path or ''\n  if cwd ~= '' then table.insert(cells, { Text = cwd:gsub('^.*/', '') }) end\n"
			}
			return fmt.Sprintf(`wezterm.on('update-status', function(window, pane)
  local cells = {}
%s%s  local bg = '#2b2042'
  local fmt_cells = {}
  for _, c in ipairs(cells) do
    table.insert(fmt_cells, { Background = { Color = bg } })
    table.insert(fmt_cells, { Foreground = { Color = '#ffffff' } })
    table.insert(fmt_cells, { Text = ' ' .. c.Text .. ' ' })
    table.insert(fmt_cells, { Background = { Color = '#333333' } })
    table.insert(fmt_cells, { Text = ' ' })
  end
  window:set_right_status(wezterm.format(fmt_cells))
end)`, segs.String(), cwd)
		},
	},
	{
		ID:     "left_status_workspace",
		Name:   "Left status (workspace + mode)",
		Doc:    "Left edge of the tab bar shows the workspace name and any active key-table mode.",
		Source: "https://github.com/KevinSilvester/wezterm-config/blob/master/events/left-status.lua",
		Params: []FeatureParam{
			{Name: "hide_default", Label: "Hide when workspace is 'default'", Kind: "bool", Default: "1"},
		},
		Emit: func(p map[string]string) string {
			hide := "true"
			if !fb(p, "hide_default") {
				hide = "false"
			}
			return fmt.Sprintf(`wezterm.on('update-status', function(window, pane)
  local ws = window:active_workspace()
  local mode = window:active_key_table()
  local left = ''
  if not (%s and ws == 'default') then
    left = ' \\u{f2d0} ' .. ws
  end
  if mode then left = left .. ' \\u{f12c6} ' .. mode end
  window:set_left_status(wezterm.format({
    { Foreground = { Color = '#2b2042' }, Text = left },
  }))
end)`, hide)
		},
	},
	{
		ID:     "ssh_badge",
		Name:   "SSH / remote-domain badge",
		Doc:    "Shows the multiplexing domain name in the right status when the pane is remote.",
		Source: "https://github.com/yutkat/dotfiles/blob/main/.config/wezterm/on.lua",
		Params: []FeatureParam{
			{Name: "hide", Label: "Hide for domains (comma-sep)", Kind: "string", Default: "local"},
			{Name: "icon", Label: "Icon", Kind: "string", Default: "\\u{f489}"},
		},
		Emit: func(p map[string]string) string {
			hide := fs(p, "hide", "local")
			icon := fs(p, "icon", "\\u{f489}")
			return fmt.Sprintf(`wezterm.on('update-right-status', function(window, pane)
  local dom = pane:get_domain_name()
  local hidden = { %s }
  for _, h in ipairs(hidden) do
    if dom == h then return end
  end
  window:set_right_status(wezterm.format({
    { Foreground = { Color = '#d08770' }, Text = ' %s ' .. dom .. ' ' },
  }))
end)`, hide, icon)
		},
	},
	{
		ID:     "zoom_indicator",
		Name:   "Zoomed-pane indicator",
		Doc:    "Shows a label in the right status when the active pane is zoomed.",
		Source: "https://github.com/yutkat/dotfiles/blob/main/.config/wezterm/on.lua",
		Params: []FeatureParam{
			{Name: "label", Label: "Label", Kind: "string", Default: "ZOOM"},
		},
		Emit: func(p map[string]string) string {
			l := fs(p, "label", "ZOOM")
			return fmt.Sprintf(`wezterm.on('update-right-status', function(window, pane)
  for _, info in ipairs(window:active_tab():panes_with_info()) do
    if info.is_zoomed and info.pane_id == pane:pane_id() then
      window:set_right_status(' %s ')
      return
    end
  end
  window:set_right_status('')
end)`, fq(l))
		},
	},
	{
		ID:     "keytable_indicator",
		Name:   "Active key-table indicator",
		Doc:    "Shows which named key table is active (e.g. resize mode) in the right status.",
		Source: "https://wezterm.org/config/key-tables.html",
		Params: []FeatureParam{
			{Name: "prefix", Label: "Prefix", Kind: "string", Default: "TABLE:"},
		},
		Emit: func(p map[string]string) string {
			pre := fs(p, "prefix", "TABLE:")
			return fmt.Sprintf(`wezterm.on('update-right-status', function(window, pane)
  local kt = window:active_key_table()
  if kt then
    window:set_right_status(wezterm.format({
      { Foreground = { Color = '#a3be8c' }, Text = ' %s %s ' },
    }))
  else
    window:set_right_status('')
  end
end)`, fq(pre), "' .. kt .. '")
		},
	},
	{
		ID:     "opacity_toggle",
		Name:   "Opacity runtime toggle",
		Doc:    "A keybinding that cycles window transparency through your chosen levels without a reload.",
		Source: "https://github.com/wezterm/wezterm/discussions/628",
		Params: []FeatureParam{
			{Name: "levels", Label: "Opacity levels (comma-sep)", Kind: "string", Default: "1.0,0.75,0.5"},
			{Name: "key", Label: "Keybinding (MODS|KEY)", Kind: "string", Default: "CTRL|SHIFT|O"},
		},
		Emit: func(p map[string]string) string {
			lv := fs(p, "levels", "1.0,0.75,0.5")
			var lua strings.Builder
			lua.WriteString("local opacity_levels = { ")
			for _, part := range strings.Split(lv, ",") {
				lua.WriteString(strings.TrimSpace(part) + ", ")
			}
			lua.WriteString("}\nlocal opacity_idx = 1\n")
			lua.WriteString("wezterm.on('toggle-opacity', function(window, pane)\n")
			lua.WriteString("  local overrides = window:get_config_overrides() or {}\n")
			lua.WriteString("  opacity_idx = (opacity_idx % #opacity_levels) + 1\n")
			lua.WriteString("  overrides.window_background_opacity = opacity_levels[opacity_idx]\n")
			lua.WriteString("  window:set_config_overrides(overrides)\nend)\n")
			lua.WriteString(keyBind(p, "key", "toggle-opacity"))
			return lua.String()
		},
	},
	{
		ID:     "backdrop_cycler",
		Name:   "Background image cycler",
		Doc:    "Keybindings to cycle or randomize the window background image from a folder.",
		Source: "https://github.com/KevinSilvester/wezterm-config/blob/master/utils/backdrops.lua",
		Params: []FeatureParam{
			{Name: "dir", Label: "Images directory", Kind: "string", Default: ""},
			{Name: "next_key", Label: "Next image (MODS|KEY)", Kind: "string", Default: "CTRL|SHIFT|N"},
			{Name: "random_key", Label: "Random image (MODS|KEY)", Kind: "string", Default: "CTRL|SHIFT|R"},
		},
		Emit: func(p map[string]string) string {
			dir := fs(p, "dir", "")
			if dir == "" {
				return "-- backdrop cycler: set an images directory to enable"
			}
			var b strings.Builder
			b.WriteString("local backdrop_dir = " + fq(dir) + "\n")
			b.WriteString("local backdrops = wezterm.glob(backdrop_dir .. '*.jpg')\n")
			b.WriteString("for _, f in ipairs(wezterm.glob(backdrop_dir .. '*.png')) do table.insert(backdrops, f) end\n")
			b.WriteString("table.sort(backdrops)\n")
			b.WriteString("local backdrop_idx = 1\n")
			b.WriteString("local function set_backdrop(window, path)\n")
			b.WriteString("  window:set_config_overrides({ background = { { source = { File = path }, horizontal_align = 'Center', vertical_align = 'Middle' }, { source = { Color = '#101010' }, opacity = 0.85, width = '100%%', height = '100%%' } } })\n")
			b.WriteString("end\n")
			b.WriteString("wezterm.on('backdrop-next', function(window, pane)\n")
			b.WriteString("  if #backdrops == 0 then return end\n")
			b.WriteString("  backdrop_idx = (backdrop_idx % #backdrops) + 1\n")
			b.WriteString("  set_backdrop(window, backdrops[backdrop_idx])\nend)\n")
			b.WriteString("wezterm.on('backdrop-random', function(window, pane)\n")
			b.WriteString("  if #backdrops == 0 then return end\n")
			b.WriteString("  set_backdrop(window, backdrops[math.random(#backdrops)])\nend)\n")
			b.WriteString(keyBind(p, "next_key", "backdrop-next"))
			b.WriteString(keyBind(p, "random_key", "backdrop-random"))
			return b.String()
		},
	},
	{
		ID:     "random_gradient",
		Name:   "Random gradient background",
		Doc:    "Generates a fresh random hwb() gradient on every config load.",
		Source: "https://github.com/wezterm/wezterm/discussions/628",
		Params: []FeatureParam{
			{Name: "orientation", Label: "Orientation", Kind: "select", Options: []string{"Linear", "Radial", "Horizontal", "Vertical"}, Default: "Linear"},
		},
		Emit: func(p map[string]string) string {
			o := fs(p, "orientation", "Linear")
			var orient string
			switch o {
			case "Radial":
				orient = "{ Radial = { radius = 0.8, cx = 0.5, cy = 0.5 } }"
			case "Horizontal":
				orient = "'Horizontal'"
			case "Vertical":
				orient = "'Vertical'"
			default:
				orient = "{ Linear = { angle = math.random(0, 359) } }"
			}
			return fmt.Sprintf(`local function random_gradient_colors()
  local function hue() return math.random(0, 359) end
  return { string.format('hwb(%%d, 20%%%%, 30%%%%)', hue()),
           string.format('hwb(%%d, 20%%%%, 30%%%%)', (hue() + 180) %% 360) }
end
config.window_background_gradient = {
  orientation = %s,
  colors = random_gradient_colors(),
  blend = 'Oklab',
  interpolation = 'Linear',
}`, orient)
		},
	},
	{
		ID:     "theme_rotator",
		Name:   "Theme rotator",
		Doc:    "Keybindings to cycle/randomize all built-in color schemes, with a toast notification.",
		Source: "https://github.com/koh-sh/wezterm-theme-rotator",
		Params: []FeatureParam{
			{Name: "next_key", Label: "Next scheme (MODS|KEY)", Kind: "string", Default: "CTRL|SHIFT|T"},
			{Name: "random_key", Label: "Random scheme (MODS|KEY)", Kind: "string", Default: "CTRL|SHIFT|Y"},
		},
		Emit: func(p map[string]string) string {
			var b strings.Builder
			b.WriteString("local all_schemes = {}\n")
			b.WriteString("for name, _ in pairs(wezterm.color.get_builtin_schemes()) do table.insert(all_schemes, name) end\n")
			b.WriteString("table.sort(all_schemes)\n")
			b.WriteString("local function scheme_idx(window)\n")
			b.WriteString("  local cur = window:effective_config().color_scheme\n")
			b.WriteString("  for i, s in ipairs(all_schemes) do if s == cur then return i end end\n")
			b.WriteString("  return 0\nend\n")
			b.WriteString("wezterm.on('scheme-next', function(window, pane)\n")
			b.WriteString("  local i = (scheme_idx(window) % #all_schemes) + 1\n")
			b.WriteString("  window:set_config_overrides({ color_scheme = all_schemes[i] })\n")
			b.WriteString("  window:toast_notification('wezterm', all_schemes[i] .. ' (' .. i .. '/' .. #all_schemes .. ')', nil, 2000)\nend)\n")
			b.WriteString("wezterm.on('scheme-random', function(window, pane)\n")
			b.WriteString("  local s = all_schemes[math.random(#all_schemes)]\n")
			b.WriteString("  window:set_config_overrides({ color_scheme = s })\n")
			b.WriteString("  window:toast_notification('wezterm', s, nil, 2000)\nend)\n")
			b.WriteString(keyBind(p, "next_key", "scheme-next"))
			b.WriteString(keyBind(p, "random_key", "scheme-random"))
			return b.String()
		},
	},
	{
		ID:     "ssh_domains_auto",
		Name:   "SSH domains from ~/.ssh/config",
		Doc:    "Generates a multiplexing ssh_domain for every host in your ssh config at load time.",
		Source: "https://github.com/yutkat/dotfiles/blob/main/.config/wezterm/wezterm.lua",
		Params: []FeatureParam{
			{Name: "multiplexing", Label: "Multiplexing", Kind: "select", Options: []string{"WezTerm", "None"}, Default: "None"},
			{Name: "assume_shell", Label: "Assume shell", Kind: "select", Options: []string{"Posix", "Unknown"}, Default: "Posix"},
		},
		Emit: func(p map[string]string) string {
			mux := fs(p, "multiplexing", "None")
			sh := fs(p, "assume_shell", "Posix")
			return fmt.Sprintf(`config.ssh_domains = (function()
  local doms = {}
  for _, host in ipairs(wezterm.enumerate_ssh_hosts()) do
    table.insert(doms, {
      name = host.host,
      remote_address = host.host .. ':' .. (host.port or 22),
      username = host.user,
      multiplexing = %s,
      assume_shell = %s,
    })
  end
  return doms
end)()`, fq(mux), fq(sh))
		},
	},
	{
		ID:     "gui_startup_layout",
		Name:   "GUI-startup workspace layout",
		Doc:    "Opens a named workspace with predefined tabs and splits running your startup commands.",
		Source: "https://github.com/KevinSilvester/wezterm-config/blob/master/events/gui-startup.lua",
		Params: []FeatureParam{
			{Name: "workspace", Label: "Workspace name", Kind: "string", Default: "main"},
			{Name: "commands", Label: "Startup commands (shell command per line)", Kind: "text", Default: ""},
		},
		Emit: func(p map[string]string) string {
			ws := fs(p, "workspace", "main")
			var tabs strings.Builder
			first := true
			for _, line := range strings.Split(strings.TrimSpace(p["commands"]), "\n") {
				cmd := strings.TrimSpace(line)
				if cmd == "" {
					continue
				}
				args := strings.Fields(cmd)
				var quoted []string
				for _, a := range args {
					quoted = append(quoted, fq(a))
				}
				if first {
					tabs.WriteString("  local tab, pane, win = mux.spawn_window{ workspace = " + fq(ws) + ", args = { " + strings.Join(quoted, ", ") + " } }\n")
					first = false
				} else {
					tabs.WriteString("  tab = win:spawn_tab{ args = { " + strings.Join(quoted, ", ") + " } }\n")
				}
			}
			if first {
				tabs.WriteString("  local tab, pane, win = mux.spawn_window{ workspace = " + fq(ws) + " }\n")
			}
			return fmt.Sprintf(`wezterm.on('gui-startup', function(cmd)
  local tab, pane, win
  local mux = wezterm.mux
%s  mux.set_default_workspace(%s)
end)`, tabs.String(), fq(ws))
		},
	},
	{
		ID:     "scrollback_editor",
		Name:   "Scrollback to editor",
		Doc:    "Keybinding that dumps the visible scrollback into a temp file and opens it in your editor.",
		Source: "https://wezterm.org/config/lua/wezterm.on.html",
		Params: []FeatureParam{
			{Name: "editor", Label: "Editor command", Kind: "string", Default: "nvim"},
			{Name: "key", Label: "Keybinding (MODS|KEY)", Kind: "string", Default: "CTRL|SHIFT|E"},
		},
		Emit: func(p map[string]string) string {
			ed := fs(p, "editor", "nvim")
			return fmt.Sprintf(`wezterm.on('scrollback-to-editor', function(window, pane)
  local text = pane:get_lines_as_text(pane:get_dimensions().scrollback_rows)
  local name = os.tmpname()
  local f = io.open(name, 'w+')
  f:write(text)
  f:flush()
  f:close()
  window:perform_action(wezterm.action.SpawnCommandInNewTab{ args = { %s, name } }, pane)
  wezterm.sleep_ms(1000)
  os.remove(name)
end)
%s`, fq(ed), keyBind(p, "key", "scrollback-to-editor"))
		},
	},
	{
		ID:     "open_uri_handler",
		Name:   "open-uri handler",
		Doc:    "Intercepts clicked links (file:// etc.) and opens them with a command you choose.",
		Source: "https://github.com/wezterm/wezterm/discussions/628",
		Params: []FeatureParam{
			{Name: "scheme", Label: "Scheme to intercept", Kind: "select", Options: []string{"file", "mailto"}, Default: "file"},
			{Name: "command", Label: "Command (URI appended)", Kind: "string", Default: "xdg-open"},
		},
		Emit: func(p map[string]string) string {
			scheme := fs(p, "scheme", "file")
			cmd := fs(p, "command", "xdg-open")
			return fmt.Sprintf(`wezterm.on('open-uri', function(window, pane, uri)
  local prefix = %s .. '://'
  if uri:sub(1, #prefix) == prefix then
    local target = uri:sub(#prefix + 1)
    window:perform_action(wezterm.action.SpawnCommandInNewTab{ args = { %s, target } }, pane)
    return false
  end
end)`, fq(scheme), fq(cmd))
		},
	},
	{
		ID:     "quick_select_pack",
		Name:   "Quick-select pattern pack",
		Doc:    "Adds extra quick-select regex presets: git SHA, hex color, IPv4, semver, k8s pod name.",
		Source: "https://wezterm.org/config/lua/config/quick_select_patterns.html",
		Params: []FeatureParam{
			{Name: "git_sha", Label: "Git SHA", Kind: "bool", Default: "1"},
			{Name: "hex_color", Label: "Hex color", Kind: "bool", Default: "1"},
			{Name: "ipv4", Label: "IPv4 address", Kind: "bool", Default: "1"},
			{Name: "semver", Label: "Semver", Kind: "bool", Default: "1"},
			{Name: "pod", Label: "K8s pod name", Kind: "bool", Default: "1"},
		},
		Emit: func(p map[string]string) string {
			var pats []string
			if fb(p, "git_sha") {
				pats = append(pats, "'[0-9a-f]{7,40}'")
			}
			if fb(p, "hex_color") {
				pats = append(pats, "'#[0-9a-fA-F]{6}'")
			}
			if fb(p, "ipv4") {
				pats = append(pats, `'\\d+\\.\\d+\\.\\d+\\.\\d+'`)
			}
			if fb(p, "semver") {
				pats = append(pats, "'\\d+\\.\\d+\\.\\d+'")
			}
			if fb(p, "pod") {
				pats = append(pats, "'[a-z0-9]+-[a-z0-9]+-[a-z0-9]{5}'")
			}
			if len(pats) == 0 {
				return ""
			}
			return "config.quick_select_patterns = (function()\n  local pats = {}\n  for _, v in ipairs(config.quick_select_patterns or {}) do table.insert(pats, v) end\n" +
				"  for _, v in ipairs({ " + strings.Join(pats, ", ") + " }) do table.insert(pats, v) end\n  return pats\nend)()"
		},
	},
	{
		ID:     "fullscreen_padding",
		Name:   "Fullscreen padding auto-recompute",
		Doc:    "Expands window padding in fullscreen (screen split into thirds) and restores it on exit.",
		Source: "https://github.com/wezterm/wezterm/discussions/628",
		Params: []FeatureParam{},
		Emit: func(p map[string]string) string {
			return `wezterm.on('window-resized', function(window, pane)
  local dims = window:get_dimensions()
  local overrides = window:get_config_overrides() or {}
  if dims.is_full_screen then
    overrides.window_padding = { left = tostring(math.floor(dims.pixel_width / 3)) .. 'px', right = '0', top = '0', bottom = '0' }
  else
    overrides.window_padding = nil
  end
  window:set_config_overrides(overrides)
end)
wezterm.on('trigger-fs-pad', function(window, pane)
  local overrides = window:get_config_overrides() or {}
  overrides.window_padding = nil
  window:set_config_overrides(overrides)
end)`
		},
	},
	{
		ID:     "gpu_adapter_select",
		Name:   "GPU adapter auto-selector",
		Doc:    "Scores wezterm.gui.enumerate_gpus() and pins the best WebGpu adapter (Discrete > Vulkan > Dx12).",
		Source: "https://github.com/KevinSilvester/wezterm-config/blob/master/utils/gpu-adapter.lua",
		Params: []FeatureParam{
			{Name: "backend", Label: "Preferred backend", Kind: "select", Options: []string{"Vulkan", "Dx12", "Gl", "Metal"}, Default: "Vulkan"},
		},
		Emit: func(p map[string]string) string {
			backend := fs(p, "backend", "Vulkan")
			return fmt.Sprintf(`config.front_end = 'WebGpu'
config.webgpu_preferred_adapter = (function()
  local best, best_score = nil, -1
  for _, gpu in ipairs(wezterm.gui.enumerate_gpus()) do
    local score = 0
    if gpu.device_type == 'DiscreteGpu' then score = score + 10 end
    if gpu.backend == %s then score = score + 5 end
    if score > best_score then best, best_score = gpu, score end
  end
  return best
end)()`, fq(backend))
		},
	},
	{
		ID:     "workspace_indicator",
		Name:   "Workspace indicator",
		Doc:    "Right-status shows the active workspace name, hidden while on the 'default' workspace.",
		Source: "https://github.com/yutkat/dotfiles/blob/main/.config/wezterm/on.lua",
		Params: []FeatureParam{
			{Name: "icon", Label: "Icon", Kind: "string", Default: "\\u{f2d0}"},
		},
		Emit: func(p map[string]string) string {
			icon := fs(p, "icon", "\\u{f2d0}")
			return fmt.Sprintf(`wezterm.on('update-right-status', function(window, pane)
  local ws = window:active_workspace()
  if ws == 'default' then ws = '' end
  window:set_right_status(' %s ' .. ws .. ' ')
end)`, icon)
		},
	},
}

// FindFeature returns the feature with the given ID, or nil.
func FindFeature(id string) *Feature {
	for i := range Features {
		if Features[i].ID == id {
			return &Features[i]
		}
	}
	return nil
}
