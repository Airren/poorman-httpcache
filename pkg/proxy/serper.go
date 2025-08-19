package proxy

import (
	"math/rand"
	"net/http/httputil"
	"net/url"
	"strings"
)

// LoadBalanceSerperKey rewrites the header to use a random key from the list.
//
//	curl --location 'https://google.serper.dev/search' \
//	--header 'X-API-KEY: xxx' \
//	--header 'Content-Type: application/json' \
//	--data '{"q":"apple inc"}'
func LoadBalanceSerperKey(keys []string) func(*httputil.ProxyRequest) {
	return func(req *httputil.ProxyRequest) {
		randomKey := keys[rand.Intn(len(keys))]
		req.Out.Header.Set("X-API-KEY", randomKey)
	}
}

func ReplaceSerperKey(key string) func(*httputil.ProxyRequest) {
	return func(req *httputil.ProxyRequest) {
		req.Out.Header.Set("X-API-KEY", key)
	}
}

func RewriteSerperPath(targetURL string) func(*httputil.ProxyRequest) {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic(err)
	}
	return func(req *httputil.ProxyRequest) {
		req.SetURL(target)
		req.Out.URL.Path = strings.TrimPrefix(req.Out.URL.Path, "/serper")
		req.Out.URL.Scheme = "https"
	}
}
