// Package pkg provides a reverse proxy with cache functionality.
package pkg

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"httpcache/pkg/cache"

	"github.com/redis/go-redis/v9"
)

// NewCacheProxy creates a new cache proxy with the given host and redis server.
func NewCacheProxy(host string, redisServer map[string]string) http.Handler {
	domain, err := url.Parse(host)
	if err != nil {
		slog.Error("Failed to parse URL", "host", host, "error", err)
		os.Exit(1)
	}
	rp := httputil.NewSingleHostReverseProxy(domain)
	cache, err := cache.New(
		cache.WithAdapter(cache.NewRedisAdapter(&redis.RingOptions{
			Addrs: redisServer,
		})),
		// cache both GET and PUT methods
		cache.WithMethods([]string{http.MethodGet, http.MethodPost}),
		// cache responses for 24 hours
		cache.WithTTL(24*time.Hour),
	)
	if err != nil {
		slog.Error("Failed to create client", "error", err)
		os.Exit(1)
	}
	return cache.HTTPHandlerMiddleware(rp)
}
