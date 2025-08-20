// Package proxy rewrites request before sending, and records states after receiving.
package proxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Option func(rp *httputil.ReverseProxy) error

// New creates a new ReverseProxy with the given options.
func New(opts ...Option) (*httputil.ReverseProxy, error) {
	rp := &httputil.ReverseProxy{}
	for _, opt := range opts {
		if err := opt(rp); err != nil {
			return nil, err
		}
	}
	return rp, nil
}

// WithTransport sets the transport for the ReverseProxy.
func WithTransport(transport http.RoundTripper) Option {
	return func(rp *httputil.ReverseProxy) error {
		rp.Transport = transport
		return nil
	}
}

// WithModifyResponse sets the modify response function for the ReverseProxy.
func WithModifyResponse(modifyResponse func(*http.Response) error) Option {
	return func(rp *httputil.ReverseProxy) error {
		rp.ModifyResponse = modifyResponse
		return nil
	}
}

// WithRewrites sets the rewrites for the ReverseProxy.
func WithRewrites(rewrites ...func(*httputil.ProxyRequest)) Option {
	final := func(req *httputil.ProxyRequest) {
		for _, rewrite := range rewrites {
			rewrite(req)
		}
	}
	return func(rp *httputil.ReverseProxy) error {
		rp.Rewrite = final
		return nil
	}
}

// DebugRequest dumps the request and response for debugging.
func DebugRequest(logger *slog.Logger) func(req *httputil.ProxyRequest) {
	return func(req *httputil.ProxyRequest) {
		dump, err := httputil.DumpRequest(req.In, false)
		if err != nil {
			logger.Debug("failed to dump request", "error", err)
			return
		}
		logger.Debug("incoming request", "dump", string(dump))
		dump, err = httputil.DumpRequest(req.Out, false)
		if err != nil {
			logger.Debug("failed to dump request", "error", err)
			return
		}
		logger.Debug("outgoing request", "dump", string(dump))
	}
}

// ProxyTransport creates a new transport for the ReverseProxy.
func ProxyTransport(enablePorxy bool, outboundURL string, logger *slog.Logger) *http.Transport {
	transport := &http.Transport{}
	if enablePorxy {
		proxyURL, err := url.Parse(outboundURL)
		if err != nil {
			logger.Debug("Error parsing outbound proxy URL", "url", outboundURL, "error", err)
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
			logger.Debug("Using outbound proxy", "proxy", proxyURL.String())
		}
	} else {
		// If outboundProxyURL is empty or invalid, transport.Proxy will remain nil,
		// and http.DefaultTransport (which uses environment variables like HTTP_PROXY) will be effectively used by default by the ReverseProxy.
		// To ensure our explicit proxy setting (or lack thereof) is used, we always set the transport.
		// If no proxy is set, it will use a new transport with no proxy, overriding env vars.
		// If you want to respect HTTP_PROXY, HTTPS_PROXY, NO_PROXY env vars when outboundProxyURL is not set, use http.DefaultTransport.
		// For this specific feature, we want to explicitly control the proxy via the command-line flag or have no proxy if not specified there.
		logger.Debug("No outbound proxy URL provided. Using direct connection.")
		// Ensure no proxy is used if not specified, effectively overriding environment variables.
		transport.Proxy = nil // Explicitly set to nil to override environment proxy settings
	} // else, transport.Proxy is already set if outboundProxyURL was valid, or nil if parsing failed (with a log message)
	return transport
}
