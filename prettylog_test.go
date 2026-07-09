package prettylog

import (
	"bytes"
	"context"
	"image/color"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

// capture returns a Handler wired to a bytes.Buffer so tests can inspect output.
func capture(opts ...Option) (*Handler, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	all := append([]Option{WithDestinationWriter(buf)}, opts...)
	return New(&slog.HandlerOptions{Level: slog.LevelDebug}, all...), buf
}

func TestNewHandler_Defaults(t *testing.T) {
	h := NewHandler(&slog.HandlerOptions{Level: slog.LevelInfo})
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.writer != os.Stdout {
		t.Error("expected default writer to be os.Stdout")
	}
	if !h.colorize {
		t.Error("expected colorize to be true by default")
	}
}

func TestEnabled(t *testing.T) {
	// Use Info threshold so we can test both sides of the gate
	h := New(&slog.HandlerOptions{Level: slog.LevelInfo}, WithDestinationWriter(io.Discard))
	ctx := context.Background()

	if h.Enabled(ctx, slog.LevelDebug) {
		t.Error("expected Debug to be disabled when threshold is Info")
	}
	if !h.Enabled(ctx, slog.LevelInfo) {
		t.Error("expected Info to be enabled")
	}
	if !h.Enabled(ctx, slog.LevelError) {
		t.Error("expected Error to be enabled")
	}
}

func TestHandle_NoColor(t *testing.T) {
	h, buf := capture() // no WithColor

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "hello world", 0)
	record.AddAttrs(slog.Int("count", 42))
	_ = h.Handle(context.Background(), record)

	out := buf.String()
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected message in output, got: %s", out)
	}
	if !strings.Contains(out, `"count": 42`) {
		t.Errorf("expected attribute in output, got: %s", out)
	}
	if strings.Contains(out, "\033[") {
		t.Error("expected no ANSI escape codes when colorize is false")
	}
}

func TestHandle_WithColor(t *testing.T) {
	h, buf := capture(WithColor())

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "colored msg", 0)
	_ = h.Handle(context.Background(), record)

	out := buf.String()
	if !strings.Contains(out, "\033[38;2;") {
		t.Errorf("expected ANSI true-color codes in output, got: %s", out)
	}
	if !strings.Contains(out, "colored msg") {
		t.Errorf("expected message in output, got: %s", out)
	}
}

func TestHandle_LevelColors(t *testing.T) {
	tests := []struct {
		level   slog.Level
		wantRGB string // partial ANSI sequence we expect
	}{
		{slog.LevelDebug, "38;2;211;211;211"}, // light grey
		{slog.LevelInfo, "38;2;0;255;255"},    // cyan
		{slog.LevelWarn, "38;2;255;215;0"},    // gold
		{slog.LevelError, "38;2;255;128;128"}, // light red
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			h, buf := capture(WithColor())
			record := slog.NewRecord(time.Now(), tt.level, "test", 0)
			_ = h.Handle(context.Background(), record)

			out := buf.String()
			if !strings.Contains(out, tt.wantRGB) {
				t.Errorf("expected RGB %s for level %s, got: %s", tt.wantRGB, tt.level, out)
			}
		})
	}
}

func TestHandle_CustomColor(t *testing.T) {
	custom := ColorMap{
		LevelInfo:  color.RGBA{255, 0, 0, 255},     // bright red
		Timestamp:  color.RGBA{0, 255, 0, 255},     // green
		Message:    color.RGBA{0, 0, 255, 255},     // blue
		Attributes: color.RGBA{255, 255, 255, 255}, // white
	}

	h, buf := capture(WithColor(), WithCustomColor(custom))
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "custom", 0)
	record.AddAttrs(slog.String("key", "val"))
	_ = h.Handle(context.Background(), record)

	out := buf.String()

	// LevelInfo should now be bright red
	if !strings.Contains(out, "38;2;255;0;0") {
		t.Errorf("expected custom red level color, got: %s", out)
	}
	// Timestamp should be green
	if !strings.Contains(out, "38;2;0;255;0") {
		t.Errorf("expected custom green timestamp, got: %s", out)
	}
	// Message should be blue
	if !strings.Contains(out, "38;2;0;0;255") {
		t.Errorf("expected custom blue message, got: %s", out)
	}
	// Attributes should be white
	if !strings.Contains(out, "38;2;255;255;255") {
		t.Errorf("expected custom white attributes, got: %s", out)
	}
}

func TestHandle_EmptyAttrsOmitted(t *testing.T) {
	h, buf := capture()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "no attrs", 0)
	_ = h.Handle(context.Background(), record)

	out := buf.String()
	if strings.Contains(out, "{}") {
		t.Error("expected empty attributes to be omitted by default")
	}
}

func TestHandle_EmptyAttrsForced(t *testing.T) {
	h, buf := capture(WithOutputEmptyAttrs())
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "no attrs", 0)
	_ = h.Handle(context.Background(), record)

	out := buf.String()
	if !strings.Contains(out, "{}") {
		t.Errorf("expected empty JSON object when WithOutputEmptyAttrs is set, got: %s", out)
	}
}

func TestWithAttrs_InheritsColorMap(t *testing.T) {
	custom := ColorMap{
		LevelInfo: color.RGBA{1, 2, 3, 255},
	}
	h, _ := capture(WithColor(), WithCustomColor(custom))
	child := h.WithAttrs([]slog.Attr{slog.String("app", "test")})

	if child == nil {
		t.Fatal("expected non-nil child handler")
	}

	ch := child.(*Handler)
	if r, g, b, _ := ch.colorMap.LevelInfo.RGBA(); r>>8 != 1 || g>>8 != 2 || b>>8 != 3 {
		t.Error("expected child handler to inherit custom color map")
	}
}

func TestWithGroup_InheritsColorMap(t *testing.T) {
	custom := ColorMap{
		Message: color.RGBA{10, 20, 30, 255},
	}
	h, _ := capture(WithCustomColor(custom))
	child := h.WithGroup("requests")

	if child == nil {
		t.Fatal("expected non-nil child handler")
	}

	ch := child.(*Handler)
	if r, g, b, _ := ch.colorMap.Message.RGBA(); r>>8 != 10 || g>>8 != 20 || b>>8 != 30 {
		t.Error("expected child handler to inherit custom color map")
	}
}

func TestResolveColor(t *testing.T) {
	fallback := color.RGBA{100, 100, 100, 255}

	// zero value should return fallback
	zero := color.RGBA{0, 0, 0, 0}
	got := resolveColor(zero, fallback)
	if got != fallback {
		t.Error("expected zero color to resolve to fallback")
	}

	// non-zero should return custom
	custom := color.RGBA{50, 60, 70, 255}
	got = resolveColor(custom, fallback)
	if got != custom {
		t.Error("expected non-zero color to resolve to itself")
	}
}

func TestMergedColorMap_PartialOverride(t *testing.T) {
	partial := ColorMap{
		LevelError: color.RGBA{255, 0, 0, 255},
		// everything else left zero
	}
	merged := mergedColorMap(partial)

	// overridden field
	if r, _, _, _ := merged.LevelError.RGBA(); r>>8 != 255 {
		t.Error("expected LevelError to be overridden")
	}

	// default field should still be present
	if r, g, b, _ := merged.LevelInfo.RGBA(); r>>8 != 0 || g>>8 != 255 || b>>8 != 255 {
		t.Error("expected LevelInfo to remain at default cyan")
	}
}

func TestSuppressDefaults(t *testing.T) {
	fn := suppressDefaults(nil)

	tests := []struct {
		key  string
		want bool // true = attr should be kept
	}{
		{slog.TimeKey, false},
		{slog.LevelKey, false},
		{slog.MessageKey, false},
		{"custom", true},
	}

	for _, tt := range tests {
		attr := slog.String(tt.key, "value")
		got := fn(nil, attr)
		if tt.want && got.Equal(slog.Attr{}) {
			t.Errorf("expected %s to be kept", tt.key)
		}
		if !tt.want && !got.Equal(slog.Attr{}) {
			t.Errorf("expected %s to be suppressed", tt.key)
		}
	}
}

func TestSuppressDefaults_ChainsNext(t *testing.T) {
	next := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == "drop" {
			return slog.Attr{}
		}
		return a
	}

	fn := suppressDefaults(next)

	// default suppression still works
	attr := slog.String(slog.TimeKey, "now")
	got := fn(nil, attr)
	if !got.Equal(slog.Attr{}) {
		t.Error("expected time to be suppressed")
	}

	// next function also runs
	attr = slog.String("drop", "me")
	got = fn(nil, attr)
	if !got.Equal(slog.Attr{}) {
		t.Error("expected chained next to drop 'drop' key")
	}
}

func TestColorizeString(t *testing.T) {
	c := color.RGBA{128, 64, 32, 255}
	got := colorizeString("hello", c)
	want := "\033[38;2;128;64;32mhello\033[0m"
	if got != want {
		t.Errorf("colorizeString mismatch:\ngot:  %s\nwant: %s", got, want)
	}
}
