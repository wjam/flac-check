package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log/slog"
	"strconv"
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
	return l.level.UnmarshalText([]byte(s))
}

func (l *logLevelFlag) Type() string {
	return fmt.Sprintf("%s|%s|%s|%s", slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError)
}

var _ pflag.Value = &stringToIntSliceFlag{}

func newStringToIntSliceValue(val map[string][]int, p *map[string][]int) *stringToIntSliceFlag {
	ssv := new(stringToIntSliceFlag)
	ssv.value = p
	*ssv.value = val
	return ssv
}

type stringToIntSliceFlag struct {
	value   *map[string][]int
	changed bool
}

func (s *stringToIntSliceFlag) String() string {
	records := make([]string, 0, len(*s.value)>>1)
	for k, eyes := range *s.value {
		var v []string
		for _, i := range eyes {
			v = append(v, strconv.Itoa(i))
		}
		records = append(records, k+"="+strings.Join(v, ","))
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(records); err != nil {
		panic(err)
	}
	w.Flush()
	return "[" + strings.TrimSpace(buf.String()) + "]"
}

func (s *stringToIntSliceFlag) Set(val string) error {
	const keyValuePairLength = 2
	parts := strings.SplitN(val, "=", keyValuePairLength)
	if len(parts) != keyValuePairLength {
		return fmt.Errorf("invalid value %q", val)
	}

	var vals []int
	for _, v := range strings.Split(parts[1], ",") {
		i, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		vals = append(vals, i)
	}

	if !s.changed {
		*s.value = make(map[string][]int)
		s.changed = true
	}

	(*s.value)[parts[0]] = append((*s.value)[parts[0]], vals...)

	return nil
}

func (s *stringToIntSliceFlag) Type() string {
	return "stringToIntSlice"
}
