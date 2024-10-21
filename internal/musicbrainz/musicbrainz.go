// Package musicbrainz handles communication to MusicBrainz.
// https://musicbrainz.org/doc/MusicBrainz_API
package musicbrainz

import (
	"context"
	"net/http"

	"github.com/carlmjohnson/requests"
	"github.com/wjam/flac-check/internal/log"
)

type Client struct {
	configs []requests.Config
}

const BaseURL = "https://musicbrainz.org/ws/2/"

func New(opts ...requests.Config) *Client {
	return &Client{
		configs: append([]requests.Config{
			func(rb *requests.Builder) {
				rb.BaseURL(BaseURL).
					Transport(requests.LogTransport(
						&http.Transport{
							MaxConnsPerHost: 2,
						},
						log.HTTP,
					))
			},
		}, opts...),
	}
}

func (c Client) GetReleaseFromAlbumID(ctx context.Context, albumID string) (Release, error) {
	var release Release
	if err := requests.New(c.configs...).
		Pathf("./release/%s", albumID).
		ToJSON(&release).
		Fetch(ctx); err != nil {
		return Release{}, err
	}

	return release, nil
}

type Disc struct {
	Releases []Release `json:"releases"`
	Id       string    `json:"id"`
}

type Release struct {
	Status          string `json:"status"`
	Date            string `json:"date"`
	Title           string `json:"title"`
	Id              string `json:"id"`
	Quality         string `json:"quality"`
	CoverArtArchive struct {
		Count    int  `json:"count"`
		Artwork  bool `json:"artwork"`
		Front    bool `json:"front"`
		Darkened bool `json:"darkened"`
		Back     bool `json:"back"`
	} `json:"cover-art-archive"`
}
