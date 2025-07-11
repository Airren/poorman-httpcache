// Package pkg provides a reverse proxy with Redis caching capabilities.
package pkg

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type Middleware func(next http.RoundTripper) http.RoundTripper

type CacheProxy struct {
	Client *Client
	Proxy  *httputil.ReverseProxy
}

func NewCacheProxy(host string) *CacheProxy {
	domain, err := url.Parse(host)
	if err != nil {
		log.Fatalf("Failed to parse URL: %v", err)
	}
	rp := httputil.NewSingleHostReverseProxy(domain)
	client, err := NewClient(
		// TODO: put redis options here
		ClientWithAdapter(NewRedisAdapter(nil)),
		ClientWithMethods([]string{http.MethodGet, http.MethodPut}),
		ClientWithTTL(24*time.Hour),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	rp.Transport = client.RoundTripperMiddleware(http.DefaultTransport)
	return &CacheProxy{
		Client: client,
		Proxy:  rp,
	}
}

func (cp *CacheProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cp.Proxy.ServeHTTP(w, r)
}
