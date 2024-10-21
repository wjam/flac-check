package music

import (
	"slices"

	"github.com/wjam/flac-check/internal/errors"
	"github.com/wjam/flac-check/internal/music/track"
	"github.com/wjam/flac-check/internal/util"
)

type album []*track.Track

func (a album) getTag(name string) []string {
	values := map[string]struct{}{}

	for _, m := range a {
		for _, s := range m.Tag(name) {
			values[s] = struct{}{}
		}
	}

	return util.Keys(values)
}

// Tags that should be consistent across tracks in an album
func (a album) validateTags() []error {
	var errs []error

	for tag, invalid := range map[string][]string{"ALBUM": {""}, "DATE": {"", "0001-01-01"}} {
		values := a.getTag(tag)
		if len(values) != 1 {
			if tag == "DATE" &&
				slices.Contains(a[0].Tag("ARTIST"), "King Size Slim") &&
				slices.Contains(a[0].Tag("ALBUM"), "Only My Good Self to Blame") {
				// TODO between 2003 & 2008
				continue
			}
			errs = append(errs, errors.ErrNotSingleTagValue{
				Tag:    tag,
				Values: values,
			})
			continue
		}
		if slices.Contains(invalid, values[0]) {
			errs = append(errs, errors.ErrInvalidValue{
				Tag:    tag,
				Values: values,
			})
		}
	}

	// Either ARTIST or ALBUMARTIST should be consistent across tracks
	artists := a.getTag("ARTIST")
	if len(artists) != 1 {
		albumArtists := a.getTag("ALBUMARTIST")
		if len(albumArtists) != 1 {
			errs = append(errs, errors.ErrNotSingleAlbumArtist{
				Artists:      artists,
				AlbumArtists: albumArtists,
			})
		}
	}

	return errs
}
