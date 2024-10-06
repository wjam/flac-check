package errors

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var _ error = ErrNotSingleAlbumArtist{}

type ErrNotSingleAlbumArtist struct {
	Artists      []string
	AlbumArtists []string
}

func (e ErrNotSingleAlbumArtist) Error() string {
	return fmt.Sprintf(
		`expected single value for "ALBUMARTIST" when multiple "ARTIST", got %s & %s`,
		join(e.Artists),
		join(e.AlbumArtists),
	)
}

func (e ErrNotSingleAlbumArtist) Is(err error) bool {
	e2, ok := err.(ErrNotSingleAlbumArtist)
	if !ok {
		return false
	}
	return slices.Equal(e.AlbumArtists, e2.AlbumArtists) && slices.Equal(e.Artists, e2.Artists)
}

var _ error = ErrNotSingleTagValue{}

type ErrNotSingleTagValue struct {
	Tag    string
	Values []string
}

func (e ErrNotSingleTagValue) Error() string {
	return fmt.Sprintf(
		"expected single value for %q, got %s",
		e.Tag,
		join(e.Values),
	)
}

func (e ErrNotSingleTagValue) Is(err error) bool {
	e2, ok := err.(ErrNotSingleTagValue)
	if !ok {
		return false
	}
	return e.Tag == e2.Tag && slices.Equal(e.Values, e2.Values)
}

var _ error = ErrInvalidValue{}

type ErrInvalidValue struct {
	Tag    string
	Values []string
}

func (e ErrInvalidValue) Error() string {
	return fmt.Sprintf(
		"expected valid value for %q, got %s",
		e.Tag,
		join(e.Values),
	)
}

func (e ErrInvalidValue) Is(err error) bool {
	e2, ok := err.(ErrInvalidValue)
	if !ok {
		return false
	}
	return e.Tag == e2.Tag && slices.Equal(e.Values, e2.Values)
}

var ErrNoTagForPicture = errors.New("no tag to find image for track")

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
