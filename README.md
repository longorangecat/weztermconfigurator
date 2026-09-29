# WezTerm Configurator

Desktop GUI (Go + Fyne) that configures WezTerm without hand-writing Lua.
The app owns `wezterm.lua`: it generates the whole file from its own JSON
state (saved beside it as `wezterm_configurator.json`) and never merges
hand-written Lua. An existing non-generated `wezterm.lua` is backed up once
(`wezterm.lua.bak-<timestamp>`) before the first overwrite.

## Linux prerequisites

```sh
sudo apt-get install gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev
```

## Run

```sh
go run .
```

## Build (Linux)

```sh
go build -o dist/weztermconfigurator .
```

## Cross-build (Windows, from Linux)

```sh
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
  go build -ldflags="-H=windowsgui -s -w" -o dist/weztermconfigurator.exe .
```

## Build (native Windows)

Requires a gcc such as MSYS2 mingw-w64:

```sh
go build -ldflags="-H=windowsgui" .
```

## Files the app writes

- `<config dir>/wezterm.lua` — generated config (overwrite on save).
- `<config dir>/wezterm_configurator.json` — app state (option values, raw
  Lua overrides, target platform, custom Lua block).
- `<config>.bak-<timestamp>` — one-time backup of a pre-existing
  non-generated config.
- `<config dir>/backups/<timestamp>/` — `wezterm.lua` + state copied before
  every save; the newest 5 are kept (File → Restore backup…).
- `<config dir>/profiles/<name>.json` — named snapshots of your option values
  (File → Profiles…).

## Features

- Quick Settings page with your own pinned options (★ on any option row).
- Live preview pane (fake terminal with your scheme, font size, opacity,
  cursor, padding, tab bar).
- Colour scheme picker with swatches and filtering.
- Review the diff against the current `wezterm.lua` before every save
  (can be switched off), plus a key-binding conflict warning.
- Undo / redo (Ctrl+Z / Ctrl+Y), Ctrl+F search (also matches choice names),
  per-choice `?` help, Docs link per option, F1 shortcut list.
- File → Import existing wezterm.lua…: runs your hand-written config in a
  sandbox and imports every option it can represent; the rest is listed.
