package music

import (
	"fmt"
	"math"
	"slices"
	"strconv"

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

// Tags that should be consistent across tracks in an album.
func (a album) validateTags(silenceTracks map[string][]int) []error {
	var errs []error

	for tag, invalid := range map[string][]string{"ALBUM": {""}, "DATE": {"", "0001-01-01"}} {
		values := a.getTag(tag)
		if len(values) != 1 {
			errs = append(errs, errors.NotSingleTagValueError{
				Tag:    tag,
				Values: values,
			})
			continue
		}
		if slices.Contains(invalid, values[0]) {
			errs = append(errs, errors.InvalidValueError{
				Tag:         tag,
				Values:      values,
				Expectation: "valid",
			})
		}
	}

	// Either ARTIST or ALBUMARTIST should be consistent across tracks
	artists := a.getTag("ARTIST")
	if len(artists) != 1 {
		albumArtists := a.getTag("ALBUMARTIST")
		if len(albumArtists) != 1 {
			errs = append(errs, errors.NotSingleAlbumArtistError{
				Artists:      artists,
				AlbumArtists: albumArtists,
			})
		}
	}

	if err := a.validateConsistentGenre(); err != nil {
		errs = append(errs, err)
	}

	if err := a.validateMusicBrainzTags(); err != nil {
		errs = append(errs, err)
	}

	errs = append(errs, a.validateDiscNumbers()...)
	errs = append(errs, a.validateTrackNumbers(silenceTracks)...)

	return errs
}

func (a album) validateDiscNumbers() []error {
	discNumbers := map[int]struct{}{}
	lowest := math.MaxInt32
	highest := math.MinInt32

	for _, t := range a {
		vees, ok := t.GetDiscNumber()
		if !ok {
			continue
		}
		disc, err := strconv.Atoi(vees[0])
		if err != nil {
			continue
		}

		if disc < lowest {
			lowest = disc
		}
		if disc > highest {
			highest = disc
		}
		discNumbers[disc] = struct{}{}
	}

	if len(discNumbers) == 0 {
		// requirement enforced by track.go
		return nil
	}

	var errs []error

	if lowest != 0 && lowest != 1 {
		errs = append(errs, errors.InvalidStartingDiscNumberError{Lowest: lowest})
	}

	for i := lowest; i <= highest; i++ {
		if _, ok := discNumbers[i]; !ok {
			errs = append(errs, errors.MissingDiscNumberError{DiscNumber: i})
		}
	}

	return errs
}

func (a album) validateTrackNumbers(silenceTracks map[string][]int) []error {
	discTracks := a.readDiscTrackNumberCounts()

	var albumName string
	if v, ok := a[0].TagOk("ALBUM"); ok {
		albumName = v[0]
	}
	var artist string
	if v, ok := a[0].TagOk("ALBUMARTIST"); ok {
		artist = v[0]
	} else if v, ok := a[0].TagOk("ARTIST"); ok {
		artist = v[0]
	}

	silenceTracksForAlbum := silenceTracks[fmt.Sprintf("%s/%s", artist, albumName)]

	var errs []error
	for disk, tracks := range discTracks {
		lowest := 1
		highest := math.MinInt32
		for trackNumber, count := range tracks {
			if count > 1 {
				errs = append(errs, errors.DiscTrackNumberCollisionError{
					DiscNumber:  disk,
					TrackNumber: trackNumber,
					Count:       count,
				})
			}
			if trackNumber > highest {
				highest = trackNumber
			}
		}

		for i := lowest; i <= highest; i++ {
			if _, ok := tracks[i]; !ok && !slices.Contains(silenceTracksForAlbum, i) {
				errs = append(errs, errors.MissingTrackNumberError{TrackNumber: i})
			}
		}
	}

	return errs
}

func (a album) readDiscTrackNumberCounts() map[int]map[int]int {
	discTracks := map[int]map[int]int{}
	for _, t := range a {
		vees, ok := t.GetDiscNumber()
		if !ok {
			continue
		}
		disc, err := strconv.Atoi(vees[0])
		if err != nil {
			continue
		}

		vees, ok = t.GetTrackNumber()
		if !ok {
			continue
		}
		trackNumber, err := strconv.Atoi(vees[0])
		if err != nil {
			continue
		}

		if _, ok := discTracks[disc]; !ok {
			discTracks[disc] = map[int]int{}
		}
		discTracks[disc][trackNumber]++
	}
	return discTracks
}

func (a album) validateConsistentGenre() error {
	genres, _ := a[0].GetGenres()

	for _, t := range a {
		other, _ := t.GetGenres()
		if !slices.Equal(genres, other) {
			return errors.InvalidGenreTagError{
				Values: a.getTag("GENRE"),
			}
		}
	}

	return nil
}

func (a album) validateMusicBrainzTags() error {
	albums := map[string]struct{}{}

	for _, t := range a {
		v, ok := t.GetMusicBrainzAlbumID()
		if !ok {
			continue
		}

		for _, s := range v {
			albums[s] = struct{}{}
		}
	}

	if len(albums) > 1 {
		return errors.InvalidValueError{
			Tag:         "MUSICBRAINZ_ALBUMID",
			Values:      util.Keys(albums),
			Expectation: "single",
		}
	}

	return nil
}
