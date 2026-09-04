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
