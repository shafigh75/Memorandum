package main

import (
	"Memorandum/config"
	"Memorandum/server/db"
	httpHandler "Memorandum/server/http"
	rpcHandler "Memorandum/server/rpc"
	Logger "Memorandum/utils/logger"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func printBanner(name string) {
	// Define the banner style
	border := strings.Repeat("=", len(name)+4)
	starBorder := strings.Repeat("*", len(name)+4)

	// Print the banner
	fmt.Println(starBorder)
	fmt.Printf("* %s *\n", name)
	fmt.Println(border)
	fmt.Println("In-memory database for efficient data management.")
	fmt.Println(starBorder)
}

func main() {
	printBanner("Memorandum")
	// Load configuration
	confPath := "config/config.json"
	config, err := config.LoadConfig(confPath)
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	store, err := db.LoadConfigAndCreateStore(confPath)
	if err != nil {
		fmt.Println("Error creating Store:", err)
		return
	}

	httpLogPath := config.HttpLogPath
	httpLogger, err := Logger.NewLogger(httpLogPath)
	if err != nil {
		fmt.Println("Error creating Logger:", err)
		return
	}

	// Start the cleanup routine based on the config
	store.StartCleanupRoutine(time.Duration(config.CleanupInterval) * time.Second)

	// Create a new HTTP server
	httpServer := &http.Server{
		Addr:    config.HTTPPort,
		Handler: httpHandler.NewHandler(store, httpLogger), // Use the handler created from the store
	}

	// Start the HTTP server in a goroutine
	go func() {
		fmt.Println("Starting HTTP server on", config.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Error starting HTTP server:", err)
		}
	}()

	// Start the RPC server in a goroutine
	rpcLogPath := config.RPCLogPath
	rpcLogger, err := Logger.NewLogger(rpcLogPath)
	go rpcHandler.StartRPCServer(store, config.RPCPort, rpcLogger)

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

	// close the store gracefully
	store.Close()
	fmt.Println("Shutdown complete.")
}
