package music

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"slices"

	interrors "github.com/wjam/music_check/internal/errors"
	"github.com/wjam/music_check/internal/log"
	"github.com/wjam/music_check/internal/music/track"
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

	errs = append(errs, album.validateTags()...)

	for _, m := range album {
		ctx := log.WithAttrs(ctx, slog.String("track", m.String()))
		if err := s.handleTrack(ctx, m); err != nil {
			errs = append(errs, fmt.Errorf("failed to handle track %s: %w", m, err))
		}
	}

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
	if err := track.ValidateTags(); err != nil {
		return err
	}

	if !track.HasPicture() {
		if err := s.addFrontCoverToTrack(ctx, track); err != nil {
			return err
		}
	}

	if !track.HasLyrics() {
		if err := s.addLyricsToTrack(ctx, track); err != nil {
			return err
		}
	}

	return nil
}

func (s *Scan) addFrontCoverToTrack(ctx context.Context, track *track.Track) error {
	if v, ok := track.TagOk("MUSICBRAINZ_ALBUMID"); ok {
		rel, err := s.music.GetReleaseFromAlbumID(ctx, v[0])
		if err != nil {
			return err
		}
		cover, err := s.art.GetCoverArtFromMusicBrainzReleaseID(ctx, rel.Id)
		if err != nil {
			return err
		}

		for _, img := range cover.Images {
			if !img.Front || !img.Approved {
				continue
			}

			url := img.Image
			if v, ok := img.Thumbnails["large"]; ok {
				url = v
			}

			data, err := s.art.FetchImage(ctx, url)
			if err != nil {
				return err
			}
			return track.SetPicture(data, url)
		}
	}

	return interrors.ErrNoTagForPicture
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
		log.Logger(ctx).InfoContext(ctx, "No lyrics found")
		return nil
	}

	if lyrics.Instrumental {
		log.Logger(ctx).InfoContext(ctx, "No lyrics for instrumental track")
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
