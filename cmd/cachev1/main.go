package main

import (
	"context"
	"fmt"
	"httpcache/pkg"
	"httpcache/pkg/cache"
	"httpcache/pkg/proxy"
	"httpcache/pkg/tollgate"
	"httpcache/pkg/tollgate/adapter"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/redis/go-redis/v9"
)

func NewCache(cfg pkg.Config, logger *slog.Logger) (*cache.Cache, error) {
	cache, err := cache.New(
		cache.WithAdapter(cache.NewRedisAdapter(&redis.RingOptions{
			Addrs:    map[string]string{"server0": fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort)},
			Username: cfg.RedisUsername,
			Password: cfg.RedisPassword,
		}, logger)),
		// cache both GET and PUT methods
		cache.WithMethods([]string{http.MethodGet, http.MethodPost}),
		// cache responses for 24 hours
		cache.WithTTL(24*time.Hour),
		cache.WithLogger(logger),
	)
	if err != nil {
		logger.Error("Failed to create cache", "error", err)
		return nil, err
	}
	return cache, nil
}

func NewJinaProxy(cache *cache.Cache, cfg pkg.Config, logger *slog.Logger) (http.Handler, error) {
	target, err := url.Parse("https://r.jina.ai")
	if err != nil {
		logger.Error("Failed to parse Jina target URL", "error", err)
		return nil, err
	}

	rp, err := proxy.New(
		proxy.WithRewrites(
			proxy.RewriteJinaPath(target),
			proxy.ReplaceJinaKey(cfg.JinaAPIKey),
			proxy.DebugRequest(logger),
		),
	)
	if err != nil {
		logger.Error("Failed to create Jina proxy", "error", err)
		return nil, err
	}

	skAdapter := adapter.NewSecretKey(cfg.InternalKey, "jina")
	secretKeyExtract := func(r *http.Request) string {
		return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	tollgate := tollgate.New(skAdapter, secretKeyExtract)

	return tollgate.HTTPHandlerMiddleware(cache.HTTPHandlerMiddleware(rp)), nil
}

func NewSerperProxy(cache *cache.Cache, cfg pkg.Config, logger *slog.Logger) (http.Handler, error) {
	target, err := url.Parse("https://google.serper.dev")
	if err != nil {
		logger.Error("Failed to parse Serper target URL", "error", err)
		return nil, err
	}

	rp, err := proxy.New(
		proxy.WithRewrites(
			proxy.RewriteSerperPath(target),
			proxy.ReplaceSerperKey(cfg.SerperAPIKey),
			proxy.DebugRequest(logger),
		),
	)
	if err != nil {
		logger.Error("Failed to create Serper proxy", "error", err)
		return nil, err
	}
	skAdapter := adapter.NewSecretKey(cfg.InternalKey, "serper")
	secretKeyExtract := func(r *http.Request) string {
		return r.Header.Get("X-API-KEY")
	}
	tollgate := tollgate.New(skAdapter, secretKeyExtract)

	return tollgate.HTTPHandlerMiddleware(cache.HTTPHandlerMiddleware(rp)), nil
}

func run(ctx context.Context, cfg pkg.Config, logger *slog.Logger) error {
	cache, err := NewCache(cfg, logger)
	if err != nil {
		return fmt.Errorf("NewCache: %w", err)
	}
	jinaProxy, err := NewJinaProxy(cache, cfg, logger)
	if err != nil {
		return fmt.Errorf("NewJinaProxy: %w", err)
	}
	serperProxy, err := NewSerperProxy(cache, cfg, logger)
	if err != nil {
		return fmt.Errorf("NewSerperProxy: %w", err)
	}

	// Create a single HTTP server with path-based routing
	mux := http.NewServeMux()

	mux.HandleFunc("/jina/", func(w http.ResponseWriter, r *http.Request) {
		jinaProxy.ServeHTTP(w, r)
	})
	mux.HandleFunc("/serper/", func(w http.ResponseWriter, r *http.Request) {
		serperProxy.ServeHTTP(w, r)
	})

	var h http.Handler = mux
	h = pkg.GetLoggerMiddleware(logger)(h)
	h = middleware.Recoverer(h)

	// Single server listening on port 8080
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: h,
	}

	// Start the single server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", "error", err)
			return
		}
	}()

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("Received shutdown signal, shutting down server...")

	// Create a context with a timeout for graceful shutdown
	shutdownCtx := context.Background()
	shutdownCtx, shutdownCancel := context.WithTimeout(shutdownCtx, 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error shutting down server", "error", err)
		return err
	}
	return nil
}

func main() {
	// parse with generics
	cfg, err := pkg.GetConfig()
	if err != nil {
		// Can't use logger here since it hasn't been created yet
		slog.Error("Failed to parse config", "error", err)
		os.Exit(1)
	}
	logger := pkg.GetLogger(cfg.LogLevel)
	logger.Info("Config", "cfg", cfg)
	ctx := context.Background()

	if err := run(ctx, cfg, logger); err != nil {
		logger.Error("Failed to run", "error", err)
		os.Exit(1)
	}
}
