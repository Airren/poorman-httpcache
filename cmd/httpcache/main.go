package main

import (
	"context"
	"fmt"
	"httpcache/pkg"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {

	jinaProxy := pkg.NewCacheProxy("https://r.jina.ai")
	// Initialize the reverse proxy and Redis middleware
	jinaServer := &http.Server{
		Addr:    ":8080",
		Handler: jinaProxy,
	}

	serperProxy := pkg.NewCacheProxy("https://google.serper.dev")

	serperServer := &http.Server{
		Addr:    ":8081",
		Handler: serperProxy,
	}

	// start the servers
	go func() {
		if err := jinaServer.ListenAndServe(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	go func() {
		if err := serperServer.ListenAndServe(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Wait for shutdown signal
	<-ctx.Done()
	fmt.Println("Received shutdown signal, shutting down servers...")

	// Create a context with a timeout for graceful shutdown
	shutdownCtx := context.Background()
	shutdownCtx, shutdownCancel := context.WithTimeout(shutdownCtx, 10*time.Second)
	defer shutdownCancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := jinaServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down jina server: %s\n", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := serperServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down serper server: %s\n", err)
		}
	}()
	wg.Wait()
}
