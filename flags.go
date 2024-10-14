package main

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/pflag"
)

var _ pflag.Value = &logLevelFlag{}

type logLevelFlag struct {
	level slog.Level
}

func (l *logLevelFlag) String() string {
	return l.level.String()
}

func (l *logLevelFlag) Set(s string) error {
	levels := map[string]slog.Level{
		"warn":  slog.LevelWarn,
		"info":  slog.LevelInfo,
		"debug": slog.LevelDebug,
	}

	level, ok := levels[strings.ToLower(s)]
	if !ok {
		return fmt.Errorf("unknown log level: %q", s)
	}

	l.level = level
	return nil
}

func (l *logLevelFlag) Type() string {
	return "log-level"
}
