package proxy

import (
	"fmt"
	"math/rand"
	"net/http/httputil"
	"net/url"
	"strings"
)

// LoadBalanceJinaKey rewrites the header to use a random key from the list.
//
//	curl "https://r.jina.ai/https://www.example.com" \
//	 -H "Authorization: Bearer jina_xxx"
func LoadBalanceJinaKey(keys []string) func(*httputil.ProxyRequest) {
	return func(req *httputil.ProxyRequest) {
		// select a random key from the list
		randomKey := keys[rand.Intn(len(keys))]
		req.Out.Header.Set("Authorization", fmt.Sprintf("Bearer %s", randomKey))
	}
}

// ReplaceJinaKey replaces the header to use the given key.
//
//	curl "https://r.jina.ai/https://www.example.com" \
//	 -H "Authorization: Bearer jina_xxx"
func ReplaceJinaKey(key string) func(*httputil.ProxyRequest) {
	return func(req *httputil.ProxyRequest) {
		req.Out.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	}
}

// RewriteJinaPath rewrites the path to the target URL.
//
//	curl "https://r.jina.ai/https://www.example.com" \
//	 -H "Authorization: Bearer jina_xxx"
func RewriteJinaPath(target *url.URL) func(*httputil.ProxyRequest) {
	return func(req *httputil.ProxyRequest) {
		req.SetURL(target)
		req.Out.URL.Path = strings.TrimPrefix(req.Out.URL.Path, "/jina")
		req.Out.URL.Scheme = target.Scheme
	}
}
