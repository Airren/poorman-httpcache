package pkg

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// CacheOption is used to set Client settings.
type CacheOption func(c *Cache) error

// NewCache initializes the cache HTTP middleware client with the given
// options.
func NewCache(opts ...CacheOption) (*Cache, error) {
	c := &Cache{}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	if c.adapter == nil {
		return nil, errors.New("cache client adapter is not set")
	}
	if int64(c.ttl) < 1 {
		return nil, errors.New("cache client ttl is not set")
	}
	if c.methods == nil {
		c.methods = []string{http.MethodGet}
	}

	return c, nil
}

// CacheWithAdapter sets the adapter type for the HTTP cache
// middleware client.
func CacheWithAdapter(a Adapter) CacheOption {
	return func(c *Cache) error {
		c.adapter = a
		return nil
	}
}

// CacheWithTTL sets how long each response is going to be cached.
func CacheWithTTL(ttl time.Duration) CacheOption {
	return func(c *Cache) error {
		if int64(ttl) < 1 {
			return fmt.Errorf("cache client ttl %v is invalid", ttl)
		}

		c.ttl = ttl

		return nil
	}
}

// CacheWithRefreshKey sets the parameter key used to free a request
// cached response. Optional setting.
func CacheWithRefreshKey(refreshKey string) CacheOption {
	return func(c *Cache) error {
		c.refreshKey = refreshKey
		return nil
	}
}

// CacheWithMethods sets the acceptable HTTP methods to be cached.
// Optional setting. If not set, default is "GET".
func CacheWithMethods(methods []string) CacheOption {
	return func(c *Cache) error {
		for _, method := range methods {
			if method != http.MethodGet && method != http.MethodPost {
				return fmt.Errorf("invalid method %s", method)
			}
		}
		c.methods = methods
		return nil
	}
}

// CacheWithExpiresHeader enables middleware to add an Expires header to responses.
// Optional setting. If not set, default is false.
func CacheWithExpiresHeader() CacheOption {
	return func(c *Cache) error {
		c.writeExpiresHeader = true
		return nil
	}
}
