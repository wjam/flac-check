package wikipedia

import (
	"context"
	"strings"

	"github.com/wjam/flac-check/internal/cache"
	"github.com/wjam/flac-check/internal/musicbrainz"
	"github.com/wjam/flac-check/internal/wikidata"

	"github.com/carlmjohnson/requests"
)

type Client struct {
	brainz  *musicbrainz.Client
	data    *wikidata.Client
	configs []requests.Config
}

const BaseURL = "https://en.wikipedia.org/w/api.php"

func New(brainz *musicbrainz.Client, data *wikidata.Client, opts ...requests.Config) *Client {
	return &Client{
		brainz: brainz,
		data:   data,
		configs: append([]requests.Config{
			func(rb *requests.Builder) {
				rb.BaseURL(BaseURL)
			},
			cache.TransportCache(),
		}, opts...),
	}
}

func (c Client) GetCoverArtFromMusicBrainzReleaseID(ctx context.Context, releaseID string) (string, error) {
	release, err := c.brainz.GetReleaseFromReleaseID(ctx, releaseID)
	if err != nil {
		return "", err
	}

	group, err := c.brainz.GetReleaseGroupFromReleaseGroupID(ctx, release.ReleaseGroup.ID)
	if err != nil {
		return "", err
	}

	wikiDataURL := group.GetURLForType("wikidata")
	if wikiDataURL == "" {
		return "", nil
	}

	wikiDataURLParts := strings.Split(wikiDataURL, "/")
	wikiDataID := wikiDataURLParts[len(wikiDataURLParts)-1]

	data, err := c.data.GetLinksForItem(ctx, wikiDataID)
	if err != nil {
		return "", err
	}

	var wikiped wikipedia
	if err := requests.New(c.configs...).
		Param("action", "query").
		Param("format", "json").
		Param("prop", "pageimages|categories").
		Param("titles", data.SiteLinks.EnWiki.Title).
		Param("generator", "images").
		Param("formatversion", "2").
		Param("piprop", "original").
		ToJSON(&wikiped).
		Fetch(ctx); err != nil {
		return "", err
	}

	for _, page := range wikiped.Query.Pages {
		if page.PageID == 0 {
			continue
		}
		for _, cat := range page.Categories {
			if cat.Title != "Category:Album covers" {
				continue
			}

			return page.Original.Source, nil
		}
	}

	return "", nil
}

type wikipedia struct {
	Continue struct {
		Clcontinue string `json:"clcontinue"`
		Continue   string `json:"continue"`
	} `json:"continue"`
	Query struct {
		Pages []struct {
			Title    string `json:"title"`
			Original struct {
				Source string `json:"source"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"original"`
			PageID     int `json:"pageid,omitempty"`
			Categories []struct {
				Ns    int    `json:"ns"`
				Title string `json:"title"`
			} `json:"categories,omitempty"`
		} `json:"pages"`
	} `json:"query"`
}
