package main

import (
	"context"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/wjam/flac-check/internal/coverart"
	"github.com/wjam/flac-check/internal/log"
	"github.com/wjam/flac-check/internal/lrclib"
	"github.com/wjam/flac-check/internal/music"
	"github.com/wjam/flac-check/internal/musicbrainz"
	"github.com/wjam/flac-check/internal/wikidata"
	"github.com/wjam/flac-check/internal/wikipedia"

	"github.com/spf13/cobra"
)

func main() {
	if err := run(); err != nil {
		// cobra will print out the error
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return root().ExecuteContext(ctx)
}

func root() *cobra.Command {
	var removeLogAttrs []string
	logLevel := &logLevelFlag{level: slog.LevelInfo}

	var opts music.ScanOptions

	cmd := &cobra.Command{
		Short:        "check all FLAC music files",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			ctx := log.ContextWithLogger(cmd.Context(), slog.New(log.WithAttrsFromContextHandler{
				Parent: slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{
					Level:       logLevel.level,
					ReplaceAttr: log.FilterAttributesFromLog(removeLogAttrs),
				}),
			}))

			cmd.SetContext(ctx)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			work := music.NewScan(args[0], opts)
			return work.Run(cmd.Context())
		},
	}

	cmd.Flags().Var(logLevel, "log-level", "Level to log at")

	cmd.Flags().BoolVar(&opts.FetchLyrics, "fetch-lyrics", true, "whether to fetch missing lyrics")
	cmd.Flags().BoolVar(&opts.Write, "write", false, "write changes to disc rather than log them")
	cmd.Flags().StringSliceVar(
		&opts.InternationalArtists, "international-artists", []string{"BABYMETAL"},
		"artists which are expected to have lyrics with non-ascii characters",
	)
	cmd.Flags().Uint16Var(
		&opts.Parallelism, "parallelism", uint16(math.Max(1, float64(runtime.NumCPU()-1))),
		"number of albums to process in parallel",
	)

	// Flags to aid testing

	const (
		coverartBaseURL    = "coverart-baseurl"
		lrclibBaseURL      = "lrclib-baseurl"
		musicbrainzBaseURL = "musicbrainz-baseurl"
		wikipediaBaseURL   = "wikipedia-baseurl"
		wikidataBaseURL    = "wikidata-baseurl"
		removeLogAttr      = "remove-log-attr"
	)
	cmd.Flags().StringVar(&opts.CoverartBaseURL, coverartBaseURL, coverart.BaseURL, "")
	cmd.Flags().StringVar(&opts.LrclibBaseURL, lrclibBaseURL, lrclib.BaseURL, "")
	cmd.Flags().StringVar(&opts.MusicbrainzBaseURL, musicbrainzBaseURL, musicbrainz.BaseURL, "")
	cmd.Flags().StringVar(&opts.WikipediaBaseURL, wikipediaBaseURL, wikipedia.BaseURL, "")
	cmd.Flags().StringVar(&opts.WikidataBaseURL, wikidataBaseURL, wikidata.BaseURL, "")
	cmd.Flags().StringSliceVar(&removeLogAttrs, removeLogAttr, []string{}, "")

	for _, s := range []string{
		coverartBaseURL, lrclibBaseURL, musicbrainzBaseURL, lrclibBaseURL,
		musicbrainzBaseURL, wikipediaBaseURL, wikidataBaseURL, removeLogAttr,
	} {
		if err := cmd.Flags().MarkHidden(s); err != nil {
			panic(err)
		}
	}

	return cmd
}
