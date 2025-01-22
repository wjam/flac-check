package music

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"slices"

	"github.com/wjam/flac-check/internal/log"
	"github.com/wjam/flac-check/internal/music/track"
	"github.com/wjam/flac-check/internal/util"
)

func (s *Scan) handleAlbum(ctx context.Context, root string, files []fs.DirEntry) error {
	ctx = log.WithAttrs(ctx, slog.String("path", root))
	log.Logger(ctx).DebugContext(ctx, "Processing album")
	album, err := readAllFlacTracks(ctx, root, files)
	if err != nil {
		return err
	}

	if len(album) == 0 {
		log.Logger(ctx).InfoContext(ctx, "Skipped album as it doesn't contain FLAC files")
		return nil
	}

	var errs []error

	for _, m := range album {
		ctx := log.WithAttrs(ctx, slog.String("track", m.String()))
		if err := s.handleTrack(ctx, m); err != nil {
			errs = append(errs, fmt.Errorf("failed to handle track %s: %w", m, err))
		}
	}

	errs = append(errs, album.validateTags()...)

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	for _, t := range album {
		ctx := log.WithAttrs(ctx, slog.String("track", t.String()))
		errs = append(errs, t.Save(ctx, s.opts.Write))
	}

	return errors.Join(errs...)
}

func (s *Scan) handleTrack(ctx context.Context, track *track.Track) error {
	track.CorrectTags()

	if err := s.addMusicBrainzAlbumID(ctx, track); err != nil {
		return err
	}

	if err := track.ValidateTags(); err != nil {
		return err
	}

	if !track.HasPicture() {
		if err := s.addFrontCoverToTrack(ctx, track); err != nil {
			return err
		}
	}

	if !track.HasGenre() {
		if err := s.addGenreTag(ctx, track); err != nil {
			return err
		}
	}

	if !track.HasLyrics() && s.opts.FetchLyrics {
		if err := s.addLyricsToTrack(ctx, track); err != nil {
			return err
		}
	}

	return nil
}

func (s *Scan) addMusicBrainzAlbumID(ctx context.Context, track *track.Track) error {
	if _, ok := track.GetMusicBrainzAlbumID(); ok {
		return nil
	}

	v, ok := track.GetMusicBrainzDiscID()
	if !ok || len(v) != 1 {
		return nil
	}

	rel, err := s.music.GetReleaseFromDiscID(ctx, v[0])
	if err != nil {
		return err
	}
	if rel == nil {
		log.Logger(ctx).InfoContext(ctx, "Unable to populate musicbrainz album ID")
		return nil
	}

	track.SetMusicBrainzAlbumID(rel.Id)

	return nil
}

func (s *Scan) addFrontCoverToTrack(ctx context.Context, track *track.Track) error {
	albumId, ok := track.GetMusicBrainzAlbumID()
	if !ok {
		return nil
	}

	rel, err := s.music.GetReleaseFromReleaseID(ctx, albumId[0])
	if err != nil {
		return err
	}

	var cover string
	if rel.CoverArtArchive.Count != 0 {
		var err error
		cover, err = s.art.GetCoverArtFromMusicBrainzReleaseID(ctx, rel.Id)
		if err != nil {
			return err
		}
	} else {
		var err error
		cover, err = s.wiki.GetCoverArtFromMusicBrainzReleaseID(ctx, rel.Id)
		if err != nil {
			return err
		}
	}

	if cover == "" {
		return fmt.Errorf("unable to find cover art")
	}

	data, err := s.art.FetchImage(ctx, cover)
	if err != nil {
		return err
	}
	return track.SetPicture(data, cover)
}

func (s *Scan) addLyricsToTrack(ctx context.Context, meta *track.Track) error {
	title, ok := meta.TagOk("TITLE")
	if !ok || len(title) != 1 {
		return nil
	}
	artist, ok := meta.TagOk("ARTIST")
	if !ok || len(artist) != 1 {
		return nil
	}
	album, ok := meta.TagOk("ALBUM")
	if !ok || len(album) != 1 {
		return nil
	}
	lyrics, err := s.lyrics.FindLyricsForTrack(ctx, title[0], artist[0], album[0])
	if err != nil {
		return err
	}

	if lyrics == nil || lyrics.Instrumental {
		log.Logger(ctx).DebugContext(ctx, "No lyrics found")
		return nil
	}

	international := slices.Contains(s.opts.InternationalArtists, artist[0])

	if lyrics.SyncedLyrics != "" {
		meta.SetSyncedLyrics(ctx, lyrics.SyncedLyrics, international)
		return nil
	} else if lyrics.PlainLyrics != "" {
		meta.SetUnsyncedLyrics(ctx, lyrics.PlainLyrics, international)
		return nil
	}

	return fmt.Errorf("lyrics was empty")
}

func (s *Scan) addGenreTag(ctx context.Context, track *track.Track) error {
	albumId, ok := track.GetMusicBrainzAlbumID()
	if !ok {
		return nil
	}

	rel, err := s.music.GetReleaseFromReleaseID(ctx, albumId[0])
	if err != nil {
		return err
	}

	genres := map[string]struct{}{}
	for _, genre := range rel.ReleaseGroup.Genres {
		genres[genre.Name] = struct{}{}
	}

	if len(genres) > 0 {
		genres := util.Keys(genres)
		slices.Sort(genres)
		track.SetGenres(genres)
	}

	return nil
}
