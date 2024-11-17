package log

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"
)

func HTTP(req *http.Request, res *http.Response, err error, duration time.Duration) {
	attrs := []any{
		slog.Duration("duration", duration),
		slog.Int("status", res.StatusCode),
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	Logger(req.Context()).DebugContext(req.Context(), fmt.Sprintf("%s %s", req.Method, req.URL.String()), attrs...)
}

type attrsKey struct{}
type loggerKey struct{}

func WithAttrs(ctx context.Context, attr ...slog.Attr) context.Context {
	var existing []slog.Attr
	if v := ctx.Value(attrsKey{}); v != nil {
		existing = v.([]slog.Attr)
	}
	return context.WithValue(ctx, attrsKey{}, append(existing, attr...))
}

func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func Logger(ctx context.Context) *slog.Logger {
	return ctx.Value(loggerKey{}).(*slog.Logger)
}

var _ slog.Handler = WithAttrsFromContextHandler{}

type WithAttrsFromContextHandler struct {
	Parent            slog.Handler
	IgnoredAttributes []string
}

func (w WithAttrsFromContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return w.Parent.Enabled(ctx, level)
}

func (w WithAttrsFromContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if v := ctx.Value(attrsKey{}); v != nil {
		record.AddAttrs(v.([]slog.Attr)...)
	}

	newRecord := slog.Record{
		Time:    record.Time,
		Message: record.Message,
		Level:   record.Level,
		PC:      record.PC,
	}

	if slices.Contains(w.IgnoredAttributes, "time") {
		newRecord.Time = time.Time{}
	}

	record.Attrs(func(a slog.Attr) bool {
		if slices.Contains(w.IgnoredAttributes, a.Key) {
			return true
		}

		newRecord.AddAttrs(a)
		return true
	})

	return w.Parent.Handle(ctx, newRecord)
}

func (w WithAttrsFromContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return w.Parent.WithAttrs(attrs)
}

func (w WithAttrsFromContextHandler) WithGroup(name string) slog.Handler {
	return w.Parent.WithGroup(name)
}
