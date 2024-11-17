package main

import (
	"context"
	"log/slog"
	"math"
	"os/signal"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/wjam/flac-check/internal/coverart"
	"github.com/wjam/flac-check/internal/log"
	"github.com/wjam/flac-check/internal/lrclib"
	"github.com/wjam/flac-check/internal/music"
	"github.com/wjam/flac-check/internal/musicbrainz"
	"github.com/wjam/flac-check/internal/wikidata"
	"github.com/wjam/flac-check/internal/wikipedia"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), shutdownSignals...)
	defer cancel()

	if err := root().ExecuteContext(ctx); err != nil {
		panic(err)
	}
}

func root() *cobra.Command {
	var removeLogAttrs []string
	logLevel := &logLevelFlag{level: slog.LevelInfo}

	var opts music.ScanOptions

	cmd := &cobra.Command{
		Short:        "check all FLAC music files",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := log.ContextWithLogger(cmd.Context(), slog.New(log.WithAttrsFromContextHandler{
				Parent:            slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: logLevel.level}),
				IgnoredAttributes: removeLogAttrs,
			}))

			cmd.SetContext(ctx)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			work := music.NewScan(args[0], opts)
			return work.Run(cmd.Context())
		},
	}

	const (
		removeLogAttr = "remove-log-attr"
	)
	cmd.Flags().StringSliceVar(
		&removeLogAttrs,
		removeLogAttr,
		[]string{},
		"Remove log attributes from the output - for testing purposes",
	)
	cmd.Flags().Var(logLevel, "log-level", "Level to log at")
	if err := cmd.Flags().MarkHidden(removeLogAttr); err != nil {
		panic(err)
	}

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

	const (
		coverartBaseUrl    = "coverart-baseurl"
		lrclibBaseUrl      = "lrclib-baseurl"
		musicbrainzBaseUrl = "musicbrainz-baseurl"
		wikipediaBaseUrl   = "wikipedia-baseurl"
		wikidataBaseUrl    = "wikidata-baseurl"
	)
	cmd.Flags().StringVar(&opts.CoverartBaseUrl, coverartBaseUrl, coverart.BaseURL, "")
	cmd.Flags().StringVar(&opts.LrclibBaseUrl, lrclibBaseUrl, lrclib.BaseURL, "")
	cmd.Flags().StringVar(&opts.MusicbrainzBaseUrl, musicbrainzBaseUrl, musicbrainz.BaseURL, "")
	cmd.Flags().StringVar(&opts.WikipediaBaseUrl, wikipediaBaseUrl, wikipedia.BaseURL, "")
	cmd.Flags().StringVar(&opts.WikidataBaseUrl, wikidataBaseUrl, wikidata.BaseURL, "")

	for _, s := range []string{coverartBaseUrl, lrclibBaseUrl, musicbrainzBaseUrl, lrclibBaseUrl, musicbrainzBaseUrl, wikipediaBaseUrl, wikidataBaseUrl} {
		if err := cmd.Flags().MarkHidden(s); err != nil {
			panic(err)
		}
	}

	return cmd
}
