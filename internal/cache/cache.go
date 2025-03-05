package cache

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/wjam/flac-check/internal/log"
	"golang.org/x/time/rate"
)

const musicBrainzConcurrentLimit = 2

func TransportCache() requests.Config {
	cache := &cacheTripper{
		parent: requests.LogTransport(&http.Transport{
			MaxConnsPerHost: musicBrainzConcurrentLimit,
		}, log.HTTP),
	}
	return func(rb *requests.Builder) {
		rb.Transport(cache)
	}
}

var _ http.RoundTripper = &cacheTripper{}

type cacheTripper struct {
	parent http.RoundTripper
	cache  sync.Map
	limit  sync.Map
}

func (c *cacheTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	key := key(request)
	res, ok := c.cache.Load(key)
	if ok {
		return http.ReadResponse(bufio.NewReader(bytes.NewReader(res.([]byte))), request)
	}

	// limit requests to a host to 1 request per second - the musicbrainz API rate limit
	limit, _ := c.limit.LoadOrStore(request.URL.Host, rate.NewLimiter(rate.Every(1*time.Second), 1))
	if err := limit.(*rate.Limiter).Wait(request.Context()); err != nil {
		return nil, err
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
