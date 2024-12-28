package main

import (
	"Memorandum/config"
	"Memorandum/server/db"
	httpHandler "Memorandum/server/http"
	rpcHandler "Memorandum/server/rpc"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Load configuration
	config, err := config.LoadConfig("config/config.json")
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	store := db.NewShardedInMemoryStore(config.NumShards)

	// Start the cleanup routine based on the config
	store.StartCleanupRoutine(time.Duration(config.CleanupInterval) * time.Second)

	// Create a new HTTP server
	httpServer := &http.Server{
		Addr:    config.HTTPPort,
		Handler: httpHandler.NewHandler(store), // Use the handler created from the store
	}

	// Start the HTTP server in a goroutine
	go func() {
		fmt.Println("Starting HTTP server on", config.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Error starting HTTP server:", err)
		}
	}()

	// Start the RPC server in a goroutine
	go rpcHandler.StartRPCServer(store, config.RPCPort)

	// Channel to listen for shutdown signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a shutdown signal
	<-signalChan
	fmt.Println("Shutdown signal received. Initiating graceful shutdown...")

	// Create a context with a timeout for the shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown the HTTP server gracefully
	if err := httpServer.Shutdown(ctx); err != nil {
		fmt.Println("Error shutting down HTTP server:", err)
	}

	// cleaup the store gracefully
	store.Cleanup()
	fmt.Println("Shutdown complete.")
}
