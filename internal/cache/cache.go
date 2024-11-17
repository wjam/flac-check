package cache

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/carlmjohnson/requests"
	"github.com/wjam/flac-check/internal/log"
)

var cache = &cacheTripper{
	parent: requests.LogTransport(&http.Transport{
		MaxConnsPerHost: 2,
	}, log.HTTP),
}

func TransportCache() requests.Config {
	return func(rb *requests.Builder) {
		rb.Transport(cache)
	}
}

var _ http.RoundTripper = &cacheTripper{}

type cacheTripper struct {
	parent http.RoundTripper
	cache  sync.Map
}

func (c *cacheTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	key := key(request)
	res, ok := c.cache.Load(key)
	if ok {
		return http.ReadResponse(bufio.NewReader(bytes.NewReader(res.([]byte))), request)
	}

	response, err := c.parent.RoundTrip(request)
	if err != nil {
		return nil, err
	}

	responseContent, err := httputil.DumpResponse(response, true)
	if err != nil {
		return nil, err
	}
	c.cache.Store(key, responseContent)

	return response, nil
}

func key(req *http.Request) string {
	return fmt.Sprintf("%s %s", req.Method, req.URL)
}
