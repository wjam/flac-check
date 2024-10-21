package music

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/carlmjohnson/requests"
	"github.com/sourcegraph/conc/pool"
	"github.com/wjam/flac-check/internal/coverart"
	"github.com/wjam/flac-check/internal/lrclib"
	"github.com/wjam/flac-check/internal/music/track"
	"github.com/wjam/flac-check/internal/musicbrainz"
	"github.com/wjam/flac-check/internal/util"
)

type ScanOptions struct {
	Write                bool
	InternationalArtists []string
	Parallelism          uint16

	CoverartBaseUrl    string
	LrclibBaseUrl      string
	MusicbrainzBaseUrl string
}

func (s ScanOptions) artClient() *coverart.Client {
	return coverart.New(func(rb *requests.Builder) {
		rb.BaseURL(s.CoverartBaseUrl)
	})
}

func (s ScanOptions) lrcLibClient() *lrclib.Client {
	return lrclib.New(func(rb *requests.Builder) {
		rb.BaseURL(s.LrclibBaseUrl)
	})
}

func (s ScanOptions) musicBrainzClient() *musicbrainz.Client {
	return musicbrainz.New(func(rb *requests.Builder) {
		rb.BaseURL(s.MusicbrainzBaseUrl)
	})
}

type Scan struct {
	path   string
	opts   ScanOptions
	art    *coverart.Client
	lyrics *lrclib.Client
	music  *musicbrainz.Client
}

func NewScan(path string, opts ScanOptions) *Scan {
	return &Scan{
		path:   path,
		opts:   opts,
		art:    opts.artClient(),
		lyrics: opts.lrcLibClient(),
		music:  opts.musicBrainzClient(),
	}
}

func (s *Scan) Run(ctx context.Context) error {
	group := pool.New().WithErrors().WithMaxGoroutines(int(s.opts.Parallelism)).WithContext(ctx)
	for e, err := range util.WalkDirIter(s.path) {
		e := e
		err := err
		if err != nil {
			group.Go(func(ctx context.Context) error {
				return err
			})
			continue
		}

		if !e.Entry.IsDir() {
			continue
		}

		entries, err := os.ReadDir(e.Path)
		if err != nil {
			group.Go(func(ctx context.Context) error {
				return err
			})
			continue
		}

		files := filesOnly(entries)

		if len(files) != len(entries) {
			continue
		}

		group.Go(func(ctx context.Context) error {
			err := s.handleAlbum(ctx, e.Path, files)
			if err == nil {
				return nil
			}

			return fmt.Errorf("album %s: %w", e.Path, err)
		})
	}

	return group.Wait()
}

func filesOnly(entries []fs.DirEntry) []fs.DirEntry {
	var files []fs.DirEntry
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e)
		}
	}
	return files
}

func readAllFlacTracks(ctx context.Context, root string, files []fs.DirEntry) (album, error) {
	var tracks []*track.Track
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".flac" {
			continue
		}

		if ctx.Err() != nil {
			// Check the context hasn't been cancelled before doing expensive parsing
			return nil, ctx.Err()
		}

		path := filepath.Join(root, file.Name())

		t, err := track.NewTrack(path)
		if err != nil {
			return nil, err
		}

		tracks = append(tracks, t)
	}
	return tracks, nil
}
