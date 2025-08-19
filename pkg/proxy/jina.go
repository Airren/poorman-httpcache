package proxy

import (
	"fmt"
	"math/rand"
	"net/http/httputil"
	"net/url"
	"strings"
)

// RewriteJinaKey rewrites the header to use a random key from the list.
//
//	curl "https://r.jina.ai/https://www.example.com" \
//	 -H "Authorization: Bearer jina_xxx"
func RewriteJinaKey(keys []string) func(*httputil.ProxyRequest) {
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
func RewriteJinaPath(targetURL string) func(*httputil.ProxyRequest) {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic(err)
	}
	return func(req *httputil.ProxyRequest) {
		req.SetURL(target)
		req.Out.URL.Path = strings.TrimPrefix(req.Out.URL.Path, "/jina")
		req.Out.URL.Scheme = "https"
	}
}

// func NewJina(keys []string) *httputil.ReverseProxy {
// 	proxy, _ := New(
// 		WithRewrites(
// 			RewriteJinaPath("https://api.jina.ai"),
// 			RewriteJinaHeader(keys),
// 		),
// 	)
// 	return proxy
// }
