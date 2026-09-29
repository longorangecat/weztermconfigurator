// Package luaimport turns a hand-written wezterm.lua into app state.
//
// The file runs inside a sandboxed Lua state with only the safe libraries, and
// whatever it returns is converted to the JSON shape the catalog describes and
// validated with the emitter that will later write it back out. Anything that
// cannot be represented is reported instead of guessed at.
package luaimport

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
	"weztermconfigurator/internal/catalog"
	"weztermconfigurator/internal/luagen"
	"weztermconfigurator/internal/state"
)

// importTimeout bounds how long a file may run.
var importTimeout = 5 * time.Second

// maxSrcBytes rejects a file no editor would hold before parsing it.
const maxSrcBytes = 2 << 20

// chunkName is the name errors report, so the line numbers in a failure point
// at the user's file rather than at this program.
const chunkName = "wezterm.lua"

// Skip is one piece of the source that could not be taken over.
type Skip struct {
	Name   string // the config key, or the path inside it
	Reason string
}

// Result is what one import produced.
type Result struct {
	State    *state.State // Values filled in, TargetOS = target, no Raw/CustomLua
	Imported []string     // option names taken over, sorted
	Skipped  []Skip       // config keys that could not be represented
	Notes    []string     // things worth telling the user about the file
}

// Import runs a wezterm.lua and converts the config table it returns into
// state for the given target platform. The error is reserved for a file that
// cannot be run at all: a syntax or runtime error, a run that outlives the
// deadline, or a chunk that returns no table.
func Import(src, target string) (*Result, error) {
	if len(src) > maxSrcBytes {
		return nil, fmt.Errorf("wezterm.lua is too large to import: %d bytes, the limit is %d", len(src), maxSrcBytes)
	}
	done := make(chan outcome, 1)
	go func() { done <- run(src, target) }()
	select {
	case o := <-done:
		return o.res, o.err
	case <-time.After(importTimeout):
		// ponytail: gopher-lua has no interrupt hook, so a file that loops
		// without end keeps one CPU-bound goroutine until the process exits.
		return nil, fmt.Errorf("importing wezterm.lua timed out after %s: it must be a plain config, without an endless loop", importTimeout)
	}
}

type outcome struct {
	res *Result
	err error
}

// run does the whole job inside one goroutine so the result never outlives the
// Lua state it was read from.
func run(src, target string) outcome {
	r := &rt{
		actions:  map[*lua.LTable]string{},
		proxies:  map[*lua.LTable]string{},
		nodes:    maxNodes,
		started:  time.Now(),
		deadline: time.Now().Add(importTimeout),
	}
	L := newState(r)
	defer L.Close()

	fn, err := L.Load(strings.NewReader(src), chunkName)
	if err != nil {
		return outcome{err: fmt.Errorf("could not read %s: %s", chunkName, oneLine(err))}
	}
	L.Push(fn)
	if err := L.PCall(0, lua.MultRet, nil); err != nil {
		return outcome{err: fmt.Errorf("could not run %s: %s", chunkName, oneLine(err))}
	}
	if L.GetTop() < 1 {
		return outcome{err: fmt.Errorf("%s must end with \"return config\" so there is a configuration to import", chunkName)}
	}
	cfg, ok := L.Get(-1).(*lua.LTable)
	if !ok {
		return outcome{err: fmt.Errorf("%s returned %s, not a configuration table: it must end with \"return config\"",
			chunkName, L.Get(-1).Type().String())}
	}
	return outcome{res: collect(r, cfg, target)}
}

// collect walks the returned config table and fills a fresh state.
func collect(r *rt, cfg *lua.LTable, target string) *Result {
	st := state.New(target)
	res := &Result{State: st}
	es, err := r.entries(cfg, "config")
	if err != nil {
		res.Notes = append(res.Notes, err.Error())
		return res
	}
	for _, e := range es {
		path := "config." + e.key
		o := catalog.Find(e.key)
		if o == nil {
			res.Skipped = append(res.Skipped, Skip{e.key, "not a known WezTerm option"})
			continue
		}
		v, skipped, err := r.value(e.val, &o.Field, path, 0)
		if err != nil {
			res.Skipped = append(res.Skipped, Skip{e.key, reason(path, err)})
			continue
		}
		if err := check(st, o.Name, v); err != nil {
			res.Skipped = append(res.Skipped, Skip{e.key, err.Error()})
			continue
		}
		st.Values[o.Name] = v
		res.Imported = append(res.Imported, o.Name)
		res.Skipped = append(res.Skipped, skipped...)
	}
	sort.Strings(res.Imported)
	sort.SliceStable(res.Skipped, func(i, j int) bool { return res.Skipped[i].Name < res.Skipped[j].Name })
	if r.events > 0 {
		noun := "handlers"
		if r.events == 1 {
			noun = "handler"
		}
		res.Notes = append(res.Notes, fmt.Sprintf("%d wezterm.on() event %s were ignored: paste them into Custom Lua", r.events, noun))
	}
	return res
}

// check validates one converted option by emitting it on its own, so a value
// only lands in the state when the emitter can write it back out.
func check(st *state.State, name string, v any) error {
	probe := state.New(st.TargetOS)
	probe.Values[name] = v
	_, err := luagen.Emit(probe)
	return err
}

// reason drops the value path from a conversion error: the Skip already names
// the option the user wrote.
func reason(path string, err error) string {
	var se *skipErr
	if errors.As(err, &se) {
		return se.reason
	}
	return strings.TrimPrefix(err.Error(), path+": ")
}

// oneLine reduces a gopher-lua error to its message, dropping the traceback.
func oneLine(err error) string {
	var ae *lua.ApiError
	msg := err.Error()
	if errors.As(err, &ae) {
		msg = ae.Object.String()
	}
	if i := strings.Index(msg, "\nstack traceback:"); i >= 0 {
		msg = msg[:i]
	}
	return msg
}
