package luaimport

import (
	"fmt"
	"os"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// rt is the per-import sandbox: the Lua state plus the bookkeeping that lets
// convert.go tell a value the stub made up from one the config built.
type rt struct {
	L        *lua.LState
	actions  map[*lua.LTable]string // bare act.X reference -> action name
	proxies  map[*lua.LTable]string // catch-all proxy -> readable path
	events   int                    // wezterm.on registrations seen
	nodes    int                    // conversion budget left
	started  time.Time
	deadline time.Time
}

// builderNoops are config_builder methods that take arguments and do nothing.
// Every other missing member stays nil so a typo still reads as a typo.
var builderNoops = map[string]bool{
	"set_strict_mode":        true,
	"set_strict_mode_ignore": true,
	"add_glob_domain":        true,
	"apply_to_config":        true,
}

// newState builds a Lua state with only the safe libraries. io is never
// opened, os is a four-function stand-in, the file and code loaders the base
// library brings along are removed and require only serves preloaded modules,
// so an imported file can neither read the disk nor run anything else.
func newState(r *rt) *lua.LState {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	r.L = L
	for _, open := range []lua.LGFunction{lua.OpenBase, lua.OpenTable, lua.OpenString, lua.OpenMath} {
		open(L)
		L.Pop(1)
	}
	openPackage(L)
	L.SetGlobal("os", r.osTable())
	for _, name := range []string{"dofile", "load", "loadfile", "collectgarbage"} {
		L.SetGlobal(name, lua.LNil)
	}
	L.PreloadModule("wezterm", func(L *lua.LState) int {
		L.Push(r.wezterm())
		return 1
	})
	return L
}

// expire raises a Lua error once the import deadline has passed, so a file that
// keeps calling into the stub stops cleanly instead of only being cut off from
// the outside.
func (r *rt) expire() {
	if time.Now().After(r.deadline) {
		r.L.RaiseError("import timed out")
	}
}

// openPackage installs package.preload for require 'wezterm' and replaces the
// loader chain with a preload-only one so require can never reach the disk.
func openPackage(L *lua.LState) {
	lua.OpenPackage(L)
	L.Pop(1)
	pkg := L.GetGlobal("package")
	// Without a search path the stock loaders find nothing even if they are
	// reached, and loaders is not part of the sandbox's surface.
	L.SetField(pkg, "path", lua.LString(""))
	L.SetField(pkg, "cpath", lua.LString(""))
	L.SetField(pkg, "loaders", lua.LNil)
	loaders := L.NewTable()
	L.RawSetInt(loaders, 1, L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		preload := L.GetField(pkg, "preload")
		if fn, ok := L.GetField(preload, name).(*lua.LFunction); ok {
			L.Push(fn)
			return 1
		}
		L.Push(lua.LString("no preloaded module '" + name + "'"))
		return 1
	}))
	L.SetField(L.Get(lua.RegistryIndex), "_LOADERS", loaders)
}

// osTable is the whole os an imported file sees: getenv, time, clock and date.
func (r *rt) osTable() *lua.LTable {
	L := r.L
	t := L.NewTable()
	L.SetField(t, "getenv", L.NewFunction(func(L *lua.LState) int {
		if L.GetTop() < 1 {
			L.Push(lua.LNil)
			return 1
		}
		if v, ok := os.LookupEnv(L.ToString(1)); ok {
			L.Push(lua.LString(v))
		} else {
			L.Push(lua.LNil)
		}
		return 1
	}))
	L.SetField(t, "time", L.NewFunction(func(L *lua.LState) int {
		if L.GetTop() >= 1 {
			L.Push(L.CheckNumber(1))
			return 1
		}
		L.Push(lua.LNumber(time.Now().Unix()))
		return 1
	}))
	L.SetField(t, "clock", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(time.Since(r.started).Seconds()))
		return 1
	}))
	L.SetField(t, "date", L.NewFunction(func(L *lua.LState) int {
		format := "%Y-%m-%d %H:%M:%S"
		if L.GetTop() >= 1 {
			s, ok := L.Get(1).(lua.LString)
			if !ok {
				L.RaiseError("bad argument #1 to 'date' (string expected, got %s)", L.Get(1).Type().String())
			}
			format = string(s)
		}
		when := time.Now()
		if L.GetTop() >= 2 {
			when = time.Unix(int64(L.CheckNumber(2)), 0)
		}
		L.Push(lua.LString(strftime(format, when)))
		return 1
	}))
	return t
}

// strftime expands the specifiers a config could plausibly use.
// ponytail: not full strftime and no "*t" table; add them if a real config
// ever needs them.
func strftime(format string, t time.Time) string {
	var b strings.Builder
	skip := false
	for i := range len(format) {
		if skip {
			skip = false
			continue
		}
		if format[i] != '%' || i+1 >= len(format) {
			b.WriteByte(format[i])
			continue
		}
		skip = true
		switch format[i+1] {
		case 'Y':
			fmt.Fprintf(&b, "%04d", t.Year())
		case 'm':
			fmt.Fprintf(&b, "%02d", int(t.Month()))
		case 'd':
			fmt.Fprintf(&b, "%02d", t.Day())
		case 'H':
			fmt.Fprintf(&b, "%02d", t.Hour())
		case 'M':
			fmt.Fprintf(&b, "%02d", t.Minute())
		case 'S':
			fmt.Fprintf(&b, "%02d", t.Second())
		case '%':
			b.WriteByte('%')
		default:
			b.WriteByte('%')
			b.WriteByte(format[i+1])
		}
	}
	return b.String()
}

// wezterm builds the permissive stub. Everything a config is likely to touch
// resolves to something harmless, and anything unknown resolves to a proxy
// that keeps absorbing calls instead of failing the whole import.
func (r *rt) wezterm() *lua.LTable {
	L := r.L
	m := L.NewTable()
	L.SetField(m, "config_builder", L.NewFunction(func(L *lua.LState) int {
		L.Push(r.builder())
		return 1
	}))
	L.SetField(m, "font", L.NewFunction(r.font))
	L.SetField(m, "font_with_fallback", L.NewFunction(r.font))
	L.SetField(m, "action_callback", L.NewFunction(func(L *lua.LState) int {
		if _, ok := L.Get(1).(*lua.LFunction); !ok {
			L.RaiseError("wezterm.action_callback expects a function")
		}
		// Hand the function back untouched: convert.go reports it as a value
		// with no JSON shape rather than inventing one.
		L.Push(L.Get(1))
		return 1
	}))
	L.SetField(m, "on", L.NewFunction(func(L *lua.LState) int {
		r.expire()
		if _, ok := L.Get(1).(lua.LString); ok {
			r.events++
		}
		return 0
	}))
	L.SetField(m, "action", r.actionTable())
	L.SetMetatable(m, r.moduleMeta("wezterm"))
	return m
}

// builder is the config table config_builder() hands back: a plain table whose
// documented no-op methods can still be called.
func (r *rt) builder() *lua.LTable {
	L := r.L
	t := L.NewTable()
	meta := L.NewTable()
	L.SetField(meta, "__index", L.NewFunction(func(L *lua.LState) int {
		if s, ok := L.Get(2).(lua.LString); ok && builderNoops[string(s)] {
			L.Push(L.NewFunction(func(L *lua.LState) int { return 0 }))
			return 1
		}
		L.Push(lua.LNil)
		return 1
	}))
	L.SetMetatable(t, meta)
	return t
}

// font serves both wezterm.font and wezterm.font_with_fallback and returns the
// shape the catalog's Font kind uses: {"font":[{family=..}, ..]}.
func (r *rt) font(L *lua.LState) int {
	out := L.NewTable()
	switch arg := L.Get(1).(type) {
	case lua.LString: // wezterm.font('Family', {weight='Bold'})
		entry := L.NewTable()
		L.SetField(entry, "family", arg)
		if opts, ok := L.Get(2).(*lua.LTable); ok {
			opts.ForEach(func(k, v lua.LValue) {
				if _, isIndex := k.(lua.LNumber); !isIndex {
					L.SetField(entry, k.String(), v)
				}
			})
		}
		list := L.NewTable()
		L.RawSetInt(list, 1, entry)
		L.SetField(out, "font", list)
	case *lua.LTable: // wezterm.font_with_fallback{ {family=..}, .. }
		L.SetField(out, "font", arg)
	default:
		L.RaiseError("wezterm.font expects a font family or a list of font attributes")
	}
	L.Push(out)
	return 1
}

// actionTable makes wezterm.action a proxy: act.Name is a bare reference and
// act.Name(arg) is that action carrying its argument.
func (r *rt) actionTable() *lua.LTable {
	L := r.L
	act := L.NewTable()
	meta := L.NewTable()
	L.SetField(meta, "__index", L.NewFunction(func(L *lua.LState) int {
		name, ok := L.Get(2).(lua.LString)
		if !ok {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(r.actionProxy(string(name)))
		return 1
	}))
	L.SetMetatable(act, meta)
	return act
}

// actionProxy is one act.Name reference. Stored on its own it is a bare action;
// called with an argument it becomes that name mapped to the argument.
func (r *rt) actionProxy(name string) *lua.LTable {
	L := r.L
	t := L.NewTable()
	r.actions[t] = name
	meta := L.NewTable()
	L.SetField(meta, "__call", L.NewFunction(func(L *lua.LState) int {
		// Called with no argument: the bare reference already renders as
		// act.Name, so keep the same value.
		if L.GetTop() < 2 {
			L.Push(t)
			return 1
		}
		out := L.NewTable()
		L.SetField(out, name, L.Get(2))
		L.Push(out)
		return 1
	}))
	L.SetMetatable(t, meta)
	return t
}

// moduleMeta answers any member a real wezterm module has and this stub does
// not with a proxy, so unknown API use degrades to one skipped value instead
// of a nil index error.
func (r *rt) moduleMeta(prefix string) *lua.LTable {
	L := r.L
	meta := L.NewTable()
	L.SetField(meta, "__index", L.NewFunction(func(L *lua.LState) int {
		key, _ := L.Get(2).(lua.LString)
		L.Push(r.proxy(prefix + "." + string(key)))
		return 1
	}))
	return meta
}

// proxy stands for a value of unknown shape. It indexes and calls to more
// proxies, so chained API use keeps working and only fails once the value
// reaches the config, where it becomes a Skip naming the call.
func (r *rt) proxy(path string) *lua.LTable {
	L := r.L
	t := L.NewTable()
	r.proxies[t] = path
	meta := L.NewTable()
	L.SetField(meta, "__index", L.NewFunction(func(L *lua.LState) int {
		key, _ := L.Get(2).(lua.LString)
		L.Push(r.proxy(path + "." + string(key)))
		return 1
	}))
	L.SetField(meta, "__call", L.NewFunction(func(L *lua.LState) int {
		L.Push(r.proxy(path + "()"))
		return 1
	}))
	L.SetMetatable(t, meta)
	return t
}
