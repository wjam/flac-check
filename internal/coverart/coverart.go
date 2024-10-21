package coverart

import (
	"bytes"
	"context"
	"net/http"

	"github.com/carlmjohnson/requests"
	"github.com/wjam/flac-check/internal/log"
)

type Client struct {
	configs []requests.Config
}

const BaseURL = "https://coverartarchive.org/release/"

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

func (c Client) GetCoverArtFromMusicBrainzReleaseID(ctx context.Context, releaseID string) (CoverArts, error) {
	var images CoverArts
	if err := requests.New(c.configs...).
		Pathf("./%s", releaseID).
		ToJSON(&images).
		Fetch(ctx); err != nil {
		return CoverArts{}, err
	}

	return images, nil
}

func (c Client) FetchImage(ctx context.Context, url string) ([]byte, error) {
	var img bytes.Buffer
	if err := requests.New(c.configs...).BaseURL(url).
		ToBytesBuffer(&img).Fetch(ctx); err != nil {
		return nil, err
	}

	return img.Bytes(), nil
}

type CoverArts struct {
	Images []struct {
		Approved   bool              `json:"approved"`
		Back       bool              `json:"back"`
		Front      bool              `json:"front"`
		Image      string            `json:"image"`
		Thumbnails map[string]string `json:"thumbnails"`
	} `json:"images"`
}
