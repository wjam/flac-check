package errorutil

import (
	"fmt"
	"slices"
	"strings"

	"github.com/wjam/flac-check/internal/music/vorbis"
)

var _ error = NotSingleTagValueError{}

type NotSingleTagValueError struct {
	Tag    vorbis.Tag
	Values []string
}

func (e NotSingleTagValueError) Error() string {
	return fmt.Sprintf(
		"expected single value for %q, got %s",
		e.Tag,
		join(e.Values),
	)
}

func (e NotSingleTagValueError) Is(err error) bool {
	e2, ok := err.(NotSingleTagValueError)
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
