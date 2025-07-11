// Package pkg provides a reverse proxy with Redis caching capabilities.
package pkg

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/go-redis/cache/v9"
	"golang.org/x/sync/singleflight"
)

type Middleware func(next http.RoundTripper) http.RoundTripper

type CacheProxy struct {
	RedisSF    *singleflight.Group
	RedisCache *cache.Cache
	Proxy      *httputil.ReverseProxy
}

func NewCacheProxy(host string) *CacheProxy {
	domain, err := url.Parse(host)
	if err != nil {
		log.Fatalf("Failed to parse URL: %v", err)
	}
	rp := httputil.NewSingleHostReverseProxy(domain)
	// TODO: configure cache
	cache := cache.New(&cache.Options{})
	redisCacheMiddleware := NewRedisMiddleware(cache)
	rp.Transport = redisCacheMiddleware(http.DefaultTransport)
	return &CacheProxy{
		RedisSF:    &singleflight.Group{},
		RedisCache: cache,
		Proxy:      rp,
	}
}

func (cp *CacheProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cp.Proxy.ServeHTTP(w, r)
}
