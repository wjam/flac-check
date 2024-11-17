package wikidata

import (
	"context"

	"github.com/carlmjohnson/requests"
	"github.com/wjam/flac-check/internal/cache"
)

type Client struct {
	configs []requests.Config
}

const BaseURL = "https://www.wikidata.org/w/rest.php/wikibase/v1/entities/items/"

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

func (c Client) GetLinksForItem(ctx context.Context, id string) (WikiData, error) {
	var data WikiData
	if err := requests.New(c.configs...).
		Pathf("./%s", id).
		Param("_fields", "sitelinks").
		Accept("application/json").
		ToJSON(&data).
		Fetch(ctx); err != nil {
		return WikiData{}, err
	}

	return data, nil
}

type WikiData struct {
	SiteLinks struct {
		EnWiki struct {
			Title string `json:"title"`
		} `json:"enwiki"`
	} `json:"sitelinks"`
}
