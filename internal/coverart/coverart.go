package coverart

import (
	"bytes"
	"context"

	"github.com/carlmjohnson/requests"
	"github.com/wjam/flac-check/internal/cache"
)

type Client struct {
	configs []requests.Config
}

const BaseURL = "https://coverartarchive.org/release/"

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

func (c Client) GetCoverArtFromMusicBrainzReleaseID(ctx context.Context, releaseID string) (string, error) {
	var images coverArts
	if err := requests.New(c.configs...).
		Pathf("./%s", releaseID).
		ToJSON(&images).
		Fetch(ctx); err != nil {
		return "", err
	}

	for _, img := range images.Images {
		if !img.Front || !img.Approved {
			continue
		}

		url := img.Image
		if v, ok := img.Thumbnails["large"]; ok {
			url = v
		}

		return url, nil
	}

	return "", nil
}

func (c Client) FetchImage(ctx context.Context, url string) ([]byte, error) {
	var img bytes.Buffer
	if err := requests.New(c.configs...).BaseURL(url).
		ToBytesBuffer(&img).Fetch(ctx); err != nil {
		return nil, err
	}

	return img.Bytes(), nil
}

type coverArts struct {
	Images []struct {
		Approved   bool              `json:"approved"`
		Back       bool              `json:"back"`
		Front      bool              `json:"front"`
		Image      string            `json:"image"`
		Thumbnails map[string]string `json:"thumbnails"`
	} `json:"images"`
}
