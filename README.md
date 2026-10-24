# Guitarists

A lightweight, offline-first desktop toolkit for expert guitarists, built with Go and Wails v3.

## Features

- **Guitar tuner** — microphone pitch detection with note, frequency, cents deviation, and in-tune feedback. Supports auto-detect and standard tuning (E2 A2 D3 G3 B3 E4).
- **Metronome** — configurable BPM, time signature, subdivisions, accented downbeats, live tempo changes, and tap tempo.
- **Tab editor** — six-string tablature staff with keyboard entry (frets 0–24), slide notes, per-line pauses, cursor navigation, rests, lyrics above each staff, multiple or cloned song lines, tempo-based preview playback with live note highlighting, and local XML save/load.

Tabs are stored as XML in `guitarists-tabs` inside your Documents directory:

- Windows: `C:\Users\<user>\Documents\guitarists-tabs`
- Linux/macOS: `$XDG_DOCUMENTS_DIR` (or `~/Documents`)/`guitarists-tabs`

No network access, telemetry, or cloud services are used.

## Requirements

Common to all platforms:

- Go 1.26 or newer
- Node.js 20 or newer and npm 7 or newer
- [Wails v3 CLI](https://v3.wails.io/getting-started/installation/)
- [go-task](https://taskfile.dev/installation/) if you want to use the `Taskfile.yml` shortcuts
- A C compiler and `CGO_ENABLED=1`: microphone capture and metronome playback use the cgo-based `malgo`/miniaudio library

### Linux

- GTK4 and WebKitGTK 6.0 development packages
- GCC/build tools and `pkg-config`
- A working audio server (PipeWire, PulseAudio, or ALSA)

Example installs:

- Arch: `sudo pacman -S gtk4 webkitgtk-6.0 base-devel`
- Debian/Ubuntu: `sudo apt install build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev`
- Fedora: `sudo dnf install gcc-c++ pkgconf-pkg-config gtk4-devel webkitgtk6.0-devel`

### Windows

- Windows 10 (version 1809 or newer) or Windows 11
- Microsoft Edge WebView2 Runtime (preinstalled on Windows 11 and most Windows 10; otherwise install the Evergreen runtime)
- 64-bit MinGW-w64 GCC for cgo, for example via MSYS2 UCRT64: `pacman -S mingw-w64-ucrt-x86_64-gcc`, then add `C:\msys64\ucrt64\bin` to `PATH` and build with `CGO_ENABLED=1`
- Microphone access enabled for desktop apps under Settings → Privacy & security → Microphone
- Optional: NSIS, only if you add installer packaging

### macOS

- macOS 12 (Monterey) or newer
- Xcode Command Line Tools: `xcode-select --install`
- CoreAudio is built in; no extra audio packages are required
- A packaged `.app` must include `NSMicrophoneUsageDescription` in its `Info.plist`, and the app must be allowed under System Settings → Privacy & Security → Microphone

## Development

Install the Wails v3 CLI and the task runner:

```bash
# Wails v3 CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.34

# go-task (the `task` command); use one of the options below
go install github.com/go-task/task/v3/cmd/task@latest
# macOS:          brew install go-task
# Arch Linux:     sudo pacman -S go-task  # installs the binary as `go-task`
# Debian/Ubuntu:  sudo snap install task --classic
# Windows:        winget install Task.Task
```

With `go install`, make sure `$(go env GOPATH)/bin` is on your `PATH`. Task names are
case-sensitive and lowercase, so use `task build`, not `task Build`. Run `task --list` to
see all available tasks.

On Arch Linux, the `go-task` package installs its binary as `go-task` to avoid clashing
with the `task` binary from `taskwarrior`. Use `go-task build` / `go-task --list` if you
installed it that way. (`go install` always produces a `task` binary.)

```bash
# Run with hot reload
wails3 dev -config ./build/config.yml

# Or use the Taskfile: build the frontend and Go binary
task build
```

Run tests and static analysis:

```bash
go test ./...
go vet ./...
```

Frontend commands run from `frontend/`:

```bash
npm install
npm run build
```

## Project layout

See [`AGENTS.md`](AGENTS.md) for the full architecture, conventions, and tab XML schema.
