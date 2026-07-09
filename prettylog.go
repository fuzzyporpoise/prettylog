// Package prettylog provides a colorful, human-readable slog.Handler.
package prettylog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

const timeFormat = "[15:04:05.000]"

// ColorMap holds the semantic colors used by the handler.
// Any zero-value color.Color field falls back to the default.
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

// defaultColorMap is the out-of-the-box color scheme.
var defaultColorMap = ColorMap{
	LevelDebug:  color.RGBA{211, 211, 211, 255}, // light grey
	LevelInfo:   color.RGBA{0, 255, 255, 255},   // cyan
	LevelNotice: color.RGBA{255, 255, 0, 255},   // yellow
	LevelWarn:   color.RGBA{255, 215, 0, 255},   // gold
	LevelError:  color.RGBA{255, 128, 128, 255}, // light red
	LevelFatal:  color.RGBA{255, 128, 255, 255}, // light magenta
	Timestamp:   color.RGBA{0, 0, 255, 255},     // blue
	Message:     color.RGBA{255, 192, 203, 255}, // pink
	Attributes:  color.RGBA{128, 128, 128, 255}, // dark grey
}

// colorizeString wraps a string with an ANSI 24-bit true-color escape sequence.
// It extracts the 8-bit RGB components from c using the standard color.Color
// interface and returns the input string wrapped in \033[38;2;R;G;Bm...\033[0m.
func colorizeString(v string, c color.Color) string {
	r, g, b, _ := c.RGBA()
	// RGBA returns values in range [0, 65535]; shift down to [0, 255].
	return fmt.Sprintf("\033[38;2;%d;%d;%dm%s\033[0m", r>>8, g>>8, b>>8, v)
}

// Handler is a slog.Handler that prints human-readable, colorized log output.
// It wraps a JSON handler to extract attributes, then formats them as a
// single line with optional ANSI colors.
type Handler struct {
	h                slog.Handler
	r                func([]string, slog.Attr) slog.Attr
	b                *bytes.Buffer
	m                *sync.Mutex
	writer           io.Writer
	colorize         bool
	outputEmptyAttrs bool
	colorMap         ColorMap
}

// Enabled reports whether the handler handles records at the given level.
// The handler ignores records whose level is lower.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
// The Handler preserves the same color map and writer.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		h: h.h.WithAttrs(attrs), b: h.b, r: h.r, m: h.m,
		writer: h.writer, colorize: h.colorize, colorMap: h.colorMap,
	}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
// The Handler preserves the same color map and writer.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		h: h.h.WithGroup(name), b: h.b, r: h.r, m: h.m,
		writer: h.writer, colorize: h.colorize, colorMap: h.colorMap,
	}
}

// computeAttrs extracts the structured attributes from a slog.Record by
// delegating to the inner JSON handler, then unmarshaling the result.
// The method is synchronized using h.m because the inner handler writes
// into a shared buffer.
func (h *Handler) computeAttrs(
	ctx context.Context,
	r slog.Record,
) (map[string]any, error) {
	h.m.Lock()
	defer func() {
		h.b.Reset()
		h.m.Unlock()
	}()
	if err := h.h.Handle(ctx, r); err != nil {
		return nil, fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}

	var attrs map[string]any
	err := json.Unmarshal(h.b.Bytes(), &attrs)
	if err != nil {
		return nil, fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}
	return attrs, nil
}

// resolveColor returns custom if it is non-nil and non-zero, otherwise fallback.
func resolveColor(custom, fallback color.Color) color.Color {
	if custom == nil {
		return fallback
	}
	r, g, b, a := custom.RGBA()
	if r == 0 && g == 0 && b == 0 && a == 0 {
		return fallback
	}
	return custom
}

// mergedColorMap overlays custom values onto the defaults.
// Only non-zero fields in custom are applied; zero-value fields fall back
// to the corresponding default.
func mergedColorMap(custom ColorMap) ColorMap {
	return ColorMap{
		LevelDebug:  resolveColor(custom.LevelDebug, defaultColorMap.LevelDebug),
		LevelInfo:   resolveColor(custom.LevelInfo, defaultColorMap.LevelInfo),
		LevelNotice: resolveColor(custom.LevelNotice, defaultColorMap.LevelNotice),
		LevelWarn:   resolveColor(custom.LevelWarn, defaultColorMap.LevelWarn),
		LevelError:  resolveColor(custom.LevelError, defaultColorMap.LevelError),
		LevelFatal:  resolveColor(custom.LevelFatal, defaultColorMap.LevelFatal),
		Timestamp:   resolveColor(custom.Timestamp, defaultColorMap.Timestamp),
		Message:     resolveColor(custom.Message, defaultColorMap.Message),
		Attributes:  resolveColor(custom.Attributes, defaultColorMap.Attributes),
	}
}

// Handle formats a single slog.Record and writes it to h.writer.
// The output format is:
//
//	[timestamp] [level:] [message] [attributes]
//
// Each component is colorized according to h.colorMap when h.colorize is true.
// Attributes are rendered as indented JSON.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	var colorize func(string, color.Color) string
	if h.colorize {
		colorize = colorizeString
	} else {
		colorize = func(s string, _ color.Color) string { return s }
	}

	cm := h.colorMap

	var level string
	levelAttr := slog.Attr{
		Key:   slog.LevelKey,
		Value: slog.AnyValue(r.Level),
	}
	if h.r != nil {
		levelAttr = h.r([]string{}, levelAttr)
	}

	if !levelAttr.Equal(slog.Attr{}) {
		level = levelAttr.Value.String() + ":"

		if r.Level <= slog.LevelDebug {
			level = colorize(level, cm.LevelDebug)
		} else if r.Level <= slog.LevelInfo {
			level = colorize(level, cm.LevelInfo)
		} else if r.Level < slog.LevelWarn {
			level = colorize(level, cm.LevelNotice)
		} else if r.Level < slog.LevelError {
			level = colorize(level, cm.LevelWarn)
		} else if r.Level <= slog.LevelError+1 {
			level = colorize(level, cm.LevelError)
		} else if r.Level > slog.LevelError+1 {
			level = colorize(level, cm.LevelFatal)
		}
	}

	var timestamp string
	timeAttr := slog.Attr{
		Key:   slog.TimeKey,
		Value: slog.StringValue(r.Time.Format(timeFormat)),
	}
	if h.r != nil {
		timeAttr = h.r([]string{}, timeAttr)
	}
	if !timeAttr.Equal(slog.Attr{}) {
		timestamp = colorize(timeAttr.Value.String(), cm.Timestamp)
	}

	var msg string
	msgAttr := slog.Attr{
		Key:   slog.MessageKey,
		Value: slog.StringValue(r.Message),
	}
	if h.r != nil {
		msgAttr = h.r([]string{}, msgAttr)
	}
	if !msgAttr.Equal(slog.Attr{}) {
		msg = colorize(msgAttr.Value.String(), cm.Message)
	}

	attrs, err := h.computeAttrs(ctx, r)
	if err != nil {
		return err
	}

	var attrsAsBytes []byte
	if h.outputEmptyAttrs || len(attrs) > 0 {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(attrs); err != nil {
			return fmt.Errorf("error when marshaling attrs: %w", err)
		}
		attrsAsBytes = bytes.TrimRight(buf.Bytes(), "\n")
	}

	out := strings.Builder{}
	if len(timestamp) > 0 {
		out.WriteString(timestamp)
		out.WriteString(" ")
	}
	if len(level) > 0 {
		out.WriteString(level)
		out.WriteString(" ")
	}
	if len(msg) > 0 {
		out.WriteString(msg)
		out.WriteString(" ")
	}
	if len(attrsAsBytes) > 0 {
		out.WriteString(colorize(string(attrsAsBytes), cm.Attributes))
	}

	_, err = io.WriteString(h.writer, out.String()+"\n")
	if err != nil {
		return err
	}

	return nil
}

// suppressDefaults returns a ReplaceAttr function that strips the default
// top-level attributes (time, level, message) so they can be formatted
// manually. If next is non-nil, it is called after the default suppression.
func suppressDefaults(
	next func([]string, slog.Attr) slog.Attr,
) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey ||
			a.Key == slog.LevelKey ||
			a.Key == slog.MessageKey {
			return slog.Attr{}
		}
		if next == nil {
			return a
		}
		return next(groups, a)
	}
}

// New creates a new Handler with the given options.
// The handler writes to an internal buffer during attribute extraction,
// then renders the final output to the configured writer.
// If handlerOptions is nil, default options are used.
func New(handlerOptions *slog.HandlerOptions, options ...Option) *Handler {
	if handlerOptions == nil {
		handlerOptions = &slog.HandlerOptions{}
	}

	buf := &bytes.Buffer{}
	handler := &Handler{
		b: buf,
		h: slog.NewJSONHandler(buf, &slog.HandlerOptions{
			Level:       handlerOptions.Level,
			AddSource:   handlerOptions.AddSource,
			ReplaceAttr: suppressDefaults(handlerOptions.ReplaceAttr),
		}),
		r:        handlerOptions.ReplaceAttr,
		m:        &sync.Mutex{},
		colorMap: defaultColorMap,
	}

	for _, opt := range options {
		opt(handler)
	}

	return handler
}

// NewHandler is a convenience constructor that returns a Handler
// configured to write colorized output to os.Stdout.
func NewHandler(opts *slog.HandlerOptions) *Handler {
	return New(opts, WithDestinationWriter(os.Stdout), WithColor())
}

// Option configures a Handler.
type Option func(h *Handler)

// WithDestinationWriter sets the output writer for the handler.
// The default writer is os.Stdout when using NewHandler.
func WithDestinationWriter(writer io.Writer) Option {
	return func(h *Handler) {
		h.writer = writer
	}
}

// WithColor enables ANSI color output.
// Colors are drawn from the handler's ColorMap (default or custom).
func WithColor() Option {
	return func(h *Handler) {
		h.colorize = true
	}
}

// WithCustomColor overlays the provided ColorMap onto the defaults.
// Only non-zero colors in the map are applied; zero-value colors fall back
// to the built-in defaults. This allows partial customization of the palette.
func WithCustomColor(cm ColorMap) Option {
	return func(h *Handler) {
		h.colorMap = mergedColorMap(cm)
	}
}

// WithOutputEmptyAttrs forces the handler to print an empty JSON object
// ({}) when a record has no attributes. By default empty attributes are omitted.
func WithOutputEmptyAttrs() Option {
	return func(h *Handler) {
		h.outputEmptyAttrs = true
	}
}
