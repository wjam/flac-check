package errors

import (
	"fmt"
	"slices"
	"strings"
)

var _ error = NotSingleAlbumArtistError{}

type NotSingleAlbumArtistError struct {
	Artists      []string
	AlbumArtists []string
}

func (e NotSingleAlbumArtistError) Error() string {
	return fmt.Sprintf(
		`expected single value for "ALBUMARTIST" when multiple "ARTIST", got %s & %s`,
		join(e.Artists),
		join(e.AlbumArtists),
	)
}

func (e NotSingleAlbumArtistError) Is(err error) bool {
	e2, ok := err.(NotSingleAlbumArtistError)
	if !ok {
		return false
	}
	return slices.Equal(e.AlbumArtists, e2.AlbumArtists) && slices.Equal(e.Artists, e2.Artists)
}

var _ error = NotSingleTagValueError{}

type NotSingleTagValueError struct {
	Tag    string
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

var _ error = InvalidValueError{}

type InvalidValueError struct {
	Tag    string
	Values []string
}

func (e InvalidValueError) Error() string {
	return fmt.Sprintf(
		"expected valid value for %q, got %s",
		e.Tag,
		join(e.Values),
	)
}

func (e InvalidValueError) Is(err error) bool {
	e2, ok := err.(InvalidValueError)
	if !ok {
		return false
	}
	return e.Tag == e2.Tag && slices.Equal(e.Values, e2.Values)
}

var _ error = InvalidIntTagError{}

type InvalidIntTagError struct {
	Tag    string
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

var _ error = InvalidStartingDiscNumberError{}

type InvalidStartingDiscNumberError struct {
	Lowest int
}

func (e InvalidStartingDiscNumberError) Error() string {
	return fmt.Sprintf("album disc number must start at either 0 or 1 rather than %d", e.Lowest)
}

func (e InvalidStartingDiscNumberError) Is(err error) bool {
	e2, ok := err.(InvalidStartingDiscNumberError)
	if !ok {
		return false
	}
	return e.Lowest == e2.Lowest
}

var _ error = MissingDiscNumberError{}

type MissingDiscNumberError struct {
	DiscNumber int
}

func (e MissingDiscNumberError) Error() string {
	return fmt.Sprintf("album disc number %d is missing", e.DiscNumber)
}

func (e MissingDiscNumberError) Is(err error) bool {
	e2, ok := err.(MissingDiscNumberError)
	if !ok {
		return false
	}
	return e.DiscNumber == e2.DiscNumber
}

var _ error = DiscTrackNumberCollisionError{}

type DiscTrackNumberCollisionError struct {
	DiscNumber  int
	TrackNumber int
	Count       int
}

func (e DiscTrackNumberCollisionError) Error() string {
	return fmt.Sprintf("expected 1 track for disc %d track %d but found %d", e.DiscNumber, e.TrackNumber, e.Count)
}

func (e DiscTrackNumberCollisionError) Is(err error) bool {
	e2, ok := err.(DiscTrackNumberCollisionError)
	if !ok {
		return false
	}
	return e.DiscNumber == e2.DiscNumber && e.TrackNumber == e2.TrackNumber && e.Count == e2.Count
}

var _ error = InvalidGenreTagError{}

type InvalidGenreTagError struct {
	Values []string
}

func (e InvalidGenreTagError) Error() string {
	return fmt.Sprintf(
		"expected consistent value for genre, got %s",
		join(e.Values),
	)
}

func (e InvalidGenreTagError) Is(err error) bool {
	e2, ok := err.(InvalidGenreTagError)
	if !ok {
		return false
	}
	return slices.Equal(e.Values, e2.Values)
}

var _ error = InvalidTagValueError{}

type InvalidTagValueError struct {
	Tag     string
	Pattern string
	Value   string
}

func (e InvalidTagValueError) Error() string {
	return fmt.Sprintf(
		"expected value for %q matching %q, got %s",
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
