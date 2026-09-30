<p align="center">
  <source media="(prefers-color-scheme: dark)" srcset="images/slog-dark.png">
  <source media="(prefers-color-scheme: light)" srcset="images/slog-light.png">
  <img alt="Project Logo" src="images/slog-dark.png" width="128">
</p>

# prettylog

[![Go Version](https://img.shields.io/github/go-mod/go-version/fuzzyporpoise/prettylog)](https://go.dev/) [![License](https://img.shields.io/github/license/fuzzyporpoise/prettylog)](./LICENSE) [![Build Status](https://img.shields.io/github/actions/workflow/status/fuzzyporpoise/prettylog/ci.yml?branch=main)](https://github.com/fuzzyporpoise/prettylog/actions)

A human-readable, colorized `slog.Handler` for Go.  
It formats log records as a single line with a timestamp, level, message, and indented JSON attributes, using **24-bit ANSI true-color** via the standard `image/color` package.

## Features

- **Standard library only** — uses `image/color` and `color.Color`; no external color dependencies.
- **Customizable palette** — override any or all colors with `WithCustomColor`.
- **Semantic color names** — colors are mapped to their usage (`LevelDebug`, `Timestamp`, `Message`, etc.) rather than raw color names.
- **Zero-value fallback** — when customizing, only the colors you specify are changed; the rest stay at their defaults.

## Installation

```bash
go get go.fuzzyporpoise.dev/prettylog
```

## Quick start

```go
package main

import (
 "log/slog"
 "os"

 "go.fuzzyporpoise.dev/prettylog"
)

func main() {
 handler := prettylog.NewHandler(&slog.HandlerOptions{Level: slog.LevelDebug})
 logger := slog.New(handler)

 logger.Info("server started", "port", 8080)
 logger.Error("connection failed", "addr", "10.0.0.1", "retry", true)
}
```

Output (colors rendered in a compatible terminal):

```sh
[14:32:01.123] INFO: server started {
  "port": 8080
}
[14:32:01.124] ERROR: connection failed {
  "addr": "10.0.0.1",
  "retry": true
}
```

## Default color scheme

| Field        | Default color | Description |
|--------------|---------------|-------------|
| `LevelDebug` | light grey    | Levels ≤ `slog.LevelDebug` |
| `LevelInfo`  | cyan          | Levels ≤ `slog.LevelInfo` |
| `LevelNotice`| yellow        | Custom levels between Info and Warn |
| `LevelWarn`  | gold          | `slog.LevelWarn` and levels between Warn and Error |
| `LevelError` | light red     | `slog.LevelError` and levels up to `Error+1` |
| `LevelFatal` | light magenta | Levels > `Error+1` |
| `Timestamp`  | blue          | Record timestamp |
| `Message`    | pink          | Record message text |
| `Attributes` | dark grey     | JSON-encoded attributes |

## Custom colors

Use `WithCustomColor` to override any part of the palette. Unchanged fields fall back to the defaults automatically.

```go
package main

import (
 "image/color"
 "log/slog"
 "os"

 "go.fuzzyporpoise.dev/prettylog"
)

func main() {
 handler := prettylog.New(
  &slog.HandlerOptions{Level: slog.LevelDebug},
  prettylog.WithColor(),
  prettylog.WithDestinationWriter(os.Stdout),
  prettylog.WithCustomColor(prettylog.ColorMap{
   LevelError: color.RGBA{255, 0, 0, 255},   // bright red
   LevelFatal: color.RGBA{148, 0, 211, 255}, // violet
   Message:    color.RGBA{255, 255, 255, 255}, // white
  }),
 )

 logger := slog.New(handler)
 logger.Error("something went wrong")
}
```

## Options

| Option | Description |
|--------|-------------|
| `WithDestinationWriter(w io.Writer)` | Set the output writer (default `os.Stdout` when using `NewHandler`). |
| `WithColor()` | Enable ANSI color output. |
| `WithCustomColor(cm ColorMap)` | Overlay a custom color map onto the defaults. |
| `WithOutputEmptyAttrs()` | Print `{}` when a record has no attributes instead of omitting them. |

## API

### `func New(handlerOptions *slog.HandlerOptions, options ...Option) *Handler`

Creates a new `Handler`. If `handlerOptions` is `nil`, default options are used.

### `func NewHandler(opts *slog.HandlerOptions) *Handler`

Convenience constructor equivalent to:

```go
New(opts, WithDestinationWriter(os.Stdout), WithColor())
```

### `type ColorMap`

```go
type ColorMap struct {
 LevelDebug  color.Color // custom levels at or below slog.LevelDebug
 LevelInfo   color.Color // custom levels at or below slog.LevelInfo
 LevelNotice color.Color // custom levels between Info and Warn
 LevelWarn   color.Color // slog.LevelWarn and custom levels between Warn and Error
 LevelError  color.Color // slog.LevelError and custom levels up to Error+1
 LevelFatal  color.Color // custom levels above Error+1
 Timestamp   color.Color // record timestamp
 Message     color.Color // record message text
 Attributes  color.Color // JSON-encoded record attributes
}
```

Any field left as the zero value (`RGBA{0,0,0,0}`) falls back to the built-in default.

## License

[MIT](LICENSE)
