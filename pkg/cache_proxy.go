// Package pkg provides a reverse proxy with Redis caching capabilities.
package pkg

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewCacheProxy(host string, redisServer map[string]string) http.Handler {
	domain, err := url.Parse(host)
	if err != nil {
		log.Fatalf("Failed to parse URL: %v", err)
		panic(err)
	}
	rp := httputil.NewSingleHostReverseProxy(domain)
	client, err := NewClient(
		ClientWithAdapter(NewRedisAdapter(&redis.RingOptions{
			Addrs: redisServer,
		})),
		// cache both GET and PUT methods
		ClientWithMethods([]string{http.MethodGet, http.MethodPost}),
		// cache responses for 24 hours
		ClientWithTTL(24*time.Hour),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
		panic(err)
	}
	return client.HTTPHandlerMiddleware(rp)
}
