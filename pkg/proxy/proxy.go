// Package proxy rewrites request before sending, and records states after receiving.
package proxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
)

type Option func(rp *httputil.ReverseProxy) error

func New(opts ...Option) (*httputil.ReverseProxy, error) {
	rp := &httputil.ReverseProxy{}
	for _, opt := range opts {
		if err := opt(rp); err != nil {
			return nil, err
		}
	}
	return rp, nil
}
func WithTransport(transport http.RoundTripper) Option {
	return func(rp *httputil.ReverseProxy) error {
		rp.Transport = transport
		return nil
	}
}

func WithModifyResponse(modifyResponse func(*http.Response) error) Option {
	return func(rp *httputil.ReverseProxy) error {
		rp.ModifyResponse = modifyResponse
		return nil
	}
}

func WithRewrites(rewrites ...func(*httputil.ProxyRequest)) Option {
	final := func(req *httputil.ProxyRequest) {
		for i := len(rewrites) - 1; i >= 0; i-- {
			rewrites[i](req)
		}
	}
	return func(rp *httputil.ReverseProxy) error {
		rp.Rewrite = final
		return nil
	}
}

func DebugRequest(req *httputil.ProxyRequest) {
	dump, err := httputil.DumpRequest(req.In, false)
	if err != nil {
		slog.Debug("failed to dump request", "error", err)
		return
	}
	slog.Debug("incoming request", "dump", string(dump))
	dump, err = httputil.DumpRequest(req.Out, false)
	if err != nil {
		slog.Debug("failed to dump request", "error", err)
		return
	}
	slog.Debug("outgoing request", "dump", string(dump))
}
