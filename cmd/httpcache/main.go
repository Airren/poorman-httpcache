package main

import (
	"context"
	"fmt"
	"httpcache/pkg"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	env "github.com/caarlos0/env/v11"
)

type Config struct {
	RedisServer  string `env:"REDIS_SERVER" envDefault:"localhost:6379"`
	SerperAPIKey string `env:"SERPER_API_KEY"`
	JinaAPIKey   string `env:"JINA_API_KEY"`
	Port         int    `env:"PORT" envDefault:"3000"`
}

func main() {

	// parse with generics
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		slog.Error("Failed to parse config", "error", err)
		os.Exit(1)
	}

	config := map[string]string{
		"server0": cfg.RedisServer,
	}
	jinaProxy := pkg.NewCacheProxy("https://r.jina.ai", config)
	serperProxy := pkg.NewCacheProxy("https://google.serper.dev", config)

	// Create a single HTTP server with path-based routing
	mux := http.NewServeMux()

	// Route /jina/* requests to jinaProxy
	mux.HandleFunc("/jina/", func(w http.ResponseWriter, r *http.Request) {
		// Strip the /jina prefix from the request path
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/jina")
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		jinaProxy.ServeHTTP(w, r)
	})

	// Route /serper/* requests to serperProxy
	mux.HandleFunc("/serper/", func(w http.ResponseWriter, r *http.Request) {
		// Strip the /serper prefix from the request path
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/serper")
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		serperProxy.ServeHTTP(w, r)
	})

	// Default route for root path
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("HTTP Cache Proxy\n\nAvailable endpoints:\n- /jina/* -> Jina AI Proxy\n- /serper/* -> Serper Proxy"))
	})

	// Single server listening on port 8080
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	// Start the single server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("Received shutdown signal, shutting down server...")

	// Create a context with a timeout for graceful shutdown
	shutdownCtx := context.Background()
	shutdownCtx, shutdownCancel := context.WithTimeout(shutdownCtx, 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Error shutting down server", "error", err)
	}
}
