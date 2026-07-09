# Contributing to prettylog

Thank you for considering a contribution! This is a pure-Go `slog.Handler` with zero external dependencies. Keeping it lightweight, predictable, and strictly stdlib is the top priority.

## Quick Start

```bash
git clone <your-fork>
cd prettylog
go build ./...
go test ./...
go vet ./...
```

## Project Structure

All source lives in the root package (`prettylog`). There is no `main` package.

| File | What it does |
|------|-------------|
| `prettylog.go` | Core `Handler` implementation and public API |
| `prettylog_test.go` | Table-driven tests for formatting, colors, and options |

## Conventions

### Go Version

Target **Go 1.26.4**. Avoid language features you aren't certain exist in this version. When in doubt, check [go.dev/doc/go1.26](https://go.dev/doc/go1.26).

### No External Dependencies

The module uses **stdlib only**. Do not add third-party packages to `go.mod`. If a dependency seems necessary, open an issue to discuss alternatives first. The `image/color` package is the single source of truth for color representation.

### Error Handling

Return errors rather than silently falling back. `Handler.Handle()` and `computeAttrs()` propagate errors to the caller. Malformed JSON during attribute extraction is a failure, not a warning.

### Functional Options

`Handler` uses the functional options pattern:

```go
func New(handlerOptions *slog.HandlerOptions, options ...Option) *Handler
```

If you add a new option, follow the existing `WithXxx` naming and apply it in `New` before the final validation step. Options that modify `ColorMap` should merge rather than replace, preserving zero-value fallbacks.

### Color Spelling

Use American spelling: `Color`, `ColorMap`, `resolveColor` — not `Colour`.

### `strings.Builder`

Prefer `strings.Builder` with `WriteString`/`WriteByte` over `fmt.Fprintf` for string assembly in hot paths. The `Handle` method builds output this way; keep it consistent.

## Code Quality

Run the full check before pushing:

```bash
make check          # fmt, vet, lint, test
```

Or manually:

```bash
go fmt ./...
go vet ./...
golangci-lint run   # config: .golangci.yml
go test ./...
```

### Linting

We use `golangci-lint` with a custom config (`.golangci.yml`). Key enabled linters: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `misspell`.

### Tests

New features or bug fixes **should** include tests. Prefer table-driven tests. Target files for coverage:

- `Handle` in `prettylog.go` (formatting, color output, level routing)
- `resolveColor` / `mergedColorMap` in `prettylog.go` (zero-value fallback logic)
- `suppressDefaults` in `prettylog.go` (attribute filtering)
- `colorizeString` in `prettylog.go` (ANSI sequence generation)

## Colors

### Default Palette

Built-in colors are defined in `defaultColorMap` as `color.RGBA` values with inline comments describing the human-readable color name. If you adjust a default, update the comment and the `README.md` table.

### Custom Colors

Users override via `WithCustomColor(cm ColorMap)`. The merge logic in `mergedColorMap` treats zero-value `color.Color` (RGBA all zero) as "use default." This is intentional — do not change the zero-value sentinel without a breaking-change discussion.

### ANSI Output

`colorizeString` generates 24-bit true-color escape sequences (`\033[38;2;R;G;Bm`). It extracts components via the standard `color.Color` interface and shifts from 16-bit to 8-bit. Keep this helper self-contained; it is the only place ANSI codes are produced.

## Pull Request Process

1. **Open an issue first** for significant changes (new API surface, breaking changes, new dependencies).
2. **Fork and branch**: `git checkout -b fix/description` or `feature/description`.
3. **Write tests** for any new behavior or bug fix.
4. **Run `make check`** and ensure everything passes.
5. **Update docs** if you change the public API (`Handler`, `ColorMap`, `Option` functions).
6. **Squash** logically related commits if the history is noisy.
7. **Reference the issue** in your PR description.

## Release Notes

This project uses [GoReleaser](https://goreleaser.com/) and conventional commit grouping for changelogs. While we don't enforce commit message format in PRs, clean history helps. The changelog groups commits by:

- `feat:` → "New!"
- `fix:` → "Fixed"
- `docs:` → "Docs"
- `(deps)` → "Deps"
- everything else → "Other stuff"

## Questions?

Open a [Discussion](https://github.com/yourname/prettylog/discussions) or issue. For bug reports, include:

- Go version (`go version`)
- Handler initialization code (options used)
- Expected vs actual output
- Terminal emulator (if color rendering is relevant)
