package track

import (
	"fmt"
	"slices"
	"strings"

	"github.com/wjam/flac-check/internal/music/vorbis"
)

var _ error = InvalidTagValueError{}

type InvalidTagValueError struct {
	Tag     vorbis.Tag
	Pattern string
	Value   string
}

func (e InvalidTagValueError) Error() string {
	return fmt.Sprintf(
		"expected value for %q matching %q, got %q",
		e.Tag,
		e.Pattern,
		e.Value,
	)
}

func (e InvalidTagValueError) Is(err error) bool {
	e2, ok := err.(InvalidTagValueError)
	if !ok {
		return false
	}
	return e.Tag == e2.Tag && e.Pattern == e2.Pattern && e.Value == e2.Value
}

var _ error = InvalidIntTagError{}

type InvalidIntTagError struct {
	Tag    vorbis.Tag
	Values []string
}

func (e InvalidIntTagError) Error() string {
	return fmt.Sprintf(
		"expected integer value for %q, got %s",
		e.Tag,
		join(e.Values),
	)
}

func (e InvalidIntTagError) Is(err error) bool {
	e2, ok := err.(InvalidIntTagError)
	if !ok {
		return false
	}
	return e.Tag == e2.Tag && slices.Equal(e.Values, e2.Values)
}

func join(s []string) string {
	if s == nil {
		return "<nil>"
	}
	if len(s) == 0 {
		return "<empty>"
	}

	slices.Sort(s)

	return strings.Join(s, ",")
}
