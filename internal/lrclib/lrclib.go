package lrclib

import (
	"context"

	"github.com/carlmjohnson/requests"
	"github.com/wjam/flac-check/internal/cache"
)

type Client struct {
	configs []requests.Config
}

const BaseURL = "https://lrclib.net/api/"

func New(opts ...requests.Config) *Client {
	return &Client{
		configs: append([]requests.Config{
			func(rb *requests.Builder) {
				rb.BaseURL(BaseURL)
			},
			cache.TransportCache(),
		}, opts...),
	}
}

func (c Client) FindLyricsForTrack(ctx context.Context, track, artist, album string) (*Lyrics, error) {
	var lyrics Lyrics
	if err := requests.New(c.configs...).
		Pathf("./get").
		Param("artist_name", artist).
		Param("album_name", album).
		Param("track_name", track).
		ToJSON(&lyrics).
		Fetch(ctx); err != nil {
		if requests.HasStatusErr(err, 404) {
			return nil, nil
		}
		return nil, err
	}

	return &lyrics, nil
}

type Lyrics struct {
	Instrumental bool   `json:"instrumental"`
	PlainLyrics  string `json:"plainLyrics"`
	SyncedLyrics string `json:"syncedLyrics"`
}
