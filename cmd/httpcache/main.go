package main

import (
	"context"
	"httpcache/pkg"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {

	config := map[string]string{
		"test": "localhost:6379",
	}
	jinaProxy := pkg.NewCacheProxy("https://r.jina.ai", config)
	// Initialize the reverse proxy and Redis middleware
	jinaServer := &http.Server{
		Addr:    ":8080",
		Handler: jinaProxy,
	}

	serperProxy := pkg.NewCacheProxy("https://google.serper.dev", config)

	serperServer := &http.Server{
		Addr:    ":8081",
		Handler: serperProxy,
	}

	// start the servers
	go func() {
		if err := jinaServer.ListenAndServe(); err != nil {
			slog.Error("Jina server failed", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		if err := serperServer.ListenAndServe(); err != nil {
			slog.Error("Serper server failed", "error", err)
			os.Exit(1)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("Received shutdown signal, shutting down servers...")

	// Create a context with a timeout for graceful shutdown
	shutdownCtx := context.Background()
	shutdownCtx, shutdownCancel := context.WithTimeout(shutdownCtx, 10*time.Second)
	defer shutdownCancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := jinaServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("Error shutting down jina server", "error", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := serperServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("Error shutting down serper server", "error", err)
		}
	}()
	wg.Wait()
}
