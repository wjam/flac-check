// Package musicbrainz handles communication to MusicBrainz.
// https://musicbrainz.org/doc/MusicBrainz_API
package musicbrainz

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/carlmjohnson/requests"
	"github.com/wjam/flac-check/internal/cache"
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
				rb.BaseURL(BaseURL)
			},
			cache.TransportCache(),
		}, opts...),
	}
}

func (c *Client) GetReleaseGroupFromReleaseGroupID(ctx context.Context, id string) (ReleaseGroup, error) {
	var group ReleaseGroup
	if err := requests.New(c.configs...).
		Pathf("./release-group/%s", id).
		Param("inc", "url-rels+annotation").
		Accept("application/json").
		ToJSON(&group).
		Fetch(ctx); err != nil {
		return ReleaseGroup{}, err
	}
	return group, nil
}

func (c *Client) GetReleaseFromReleaseID(ctx context.Context, albumID string) (Release, error) {
	var release Release
	if err := requests.New(c.configs...).
		Pathf("./release/%s", albumID).
		Param("inc", "release-groups").
		Accept("application/json").
		ToJSON(&release).
		Fetch(ctx); err != nil {
		return Release{}, err
	}

	return release, nil
}

func (c *Client) GetReleaseFromDiscID(ctx context.Context, discID string) (*Release, error) {
	var discs struct {
		Releases []Release `json:"releases"`
	}
	if err := requests.New(c.configs...).
		Pathf("./discid/%s", discID).
		Accept("application/json").
		ToJSON(&discs).
		Fetch(ctx); err != nil {
		if requests.HasStatusErr(err, http.StatusNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if len(discs.Releases) == 0 {
		return nil, errors.New("no releases found")
	}
	if len(discs.Releases) == 1 {
		return &discs.Releases[0], nil
	}

	var validReleases []Release
	for _, release := range discs.Releases {
		if release.Country != "XE" && release.Country != "XW" && release.Country != "GB" {
			log.Logger(ctx).DebugContext(ctx, "Skipping release as incorrect country", slog.String("country", release.Country))
			continue
		}
		if len(release.Media) != 1 {
			log.Logger(ctx).DebugContext(ctx, "Skipping release as no media")
			continue
		}
		if release.Media[0].Format != "CD" {
			log.Logger(ctx).DebugContext(ctx, "Skipping release as incorrect media format", slog.String("format", release.Media[0].Format))
			continue
		}
		validReleases = append(validReleases, release)
	}

	if len(validReleases) != 1 {
		return nil, errors.New("could not find one release for disc")
	}

	return &validReleases[0], nil
}

type Release struct {
	Status          string `json:"status"`
	Date            string `json:"date"`
	Title           string `json:"title"`
	Id              string `json:"id"`
	Quality         string `json:"quality"`
	Country         string `json:"country"`
	CoverArtArchive struct {
		Count    int  `json:"count"`
		Artwork  bool `json:"artwork"`
		Front    bool `json:"front"`
		Darkened bool `json:"darkened"`
		Back     bool `json:"back"`
	} `json:"cover-art-archive"`
	Media []struct {
		Format string `json:"format"`
	} `json:"media"`
	ReleaseGroup struct {
		Id string `json:"id"`
	} `json:"release-group"`
}

type ReleaseGroup struct {
	Relations []struct {
		Url struct {
			Resource string `json:"resource"`
			Id       string `json:"id"`
		} `json:"url"`
		Type string `json:"type"`
	} `json:"relations"`
}

func (g ReleaseGroup) GetURLForType(relType string) string {
	for _, rel := range g.Relations {
		if rel.Type != relType {
			continue
		}

		return rel.Url.Resource
	}

	return ""
}
