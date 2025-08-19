// Package pkg provides a reverse proxy with cache functionality.
package pkg

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewCacheProxy creates a new cache proxy with the given host and redis server.
func NewCacheProxy(host string, redisServer map[string]string) http.Handler {
	domain, err := url.Parse(host)
	if err != nil {
		log.Fatalf("Failed to parse URL: %v", err)
	}
	rp := httputil.NewSingleHostReverseProxy(domain)
	cache, err := NewCache(
		CacheWithAdapter(NewRedisAdapter(&redis.RingOptions{
			Addrs: redisServer,
		})),
		// cache both GET and PUT methods
		CacheWithMethods([]string{http.MethodGet, http.MethodPost}),
		// cache responses for 24 hours
		CacheWithTTL(24*time.Hour),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return cache.HTTPHandlerMiddleware(rp)
}
