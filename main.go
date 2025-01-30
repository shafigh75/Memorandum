package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/shafigh75/Memorandum/cluster"

	"github.com/shafigh75/Memorandum/config"
	"github.com/shafigh75/Memorandum/server/db"
	httpHandler "github.com/shafigh75/Memorandum/server/http"
	rpcHandler "github.com/shafigh75/Memorandum/server/rpc"
	Logger "github.com/shafigh75/Memorandum/utils/logger"
)

const (
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Reset  = "\033[0m"
)

func printBanner(name string) {
	// Define the banner style
	border := strings.Repeat("=", len(name)+4)
	starBorder := strings.Repeat("*", len(name)+4)

	// Print the banner
	fmt.Println(Green + starBorder)
	fmt.Printf("* %s *\n", name)
	fmt.Println(border)
	fmt.Println("In-memory database for efficient data management.")
	fmt.Println(starBorder + Reset)
}

func main() {
	printBanner("Memorandum")
	// Load configuration
	confPath := "config/config.json"
	config, err := config.LoadConfig(confPath)
	if err != nil {
		fmt.Println(Red+"Error loading config:"+Reset, err)
		return
	}

	store, err := db.LoadConfigAndCreateStore(confPath)
	if err != nil {
		fmt.Println(Red+"Error creating Store:"+Reset, err)
		return
	}

	httpLogPath := config.HttpLogPath
	httpLogger, err := Logger.NewLogger(httpLogPath)
	if err != nil {
		fmt.Println(Red+"Error creating Logger:"+Reset, err)
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
			fmt.Println(Red+"Error starting HTTP server:"+Reset, err)
		}
	}()

	// Start the RPC server in a goroutine
	rpcLogPath := config.RPCLogPath
	rpcLogger, err := Logger.NewLogger(rpcLogPath)
	if err != nil {
		fmt.Println(Yellow + "logger is disabled ..." + Reset)
	}
	go rpcHandler.StartRPCServer(store, config.RPCPort, rpcLogger)

	isClustered := config.ClusterEnabled
	if isClustered {
		fmt.Println(Red + "Running in cluster Mode, starting server ..." + Reset)
		go cluster.StartHTTPServer(config.ClusterPort)
	} else {
		fmt.Println(Red + "Running as standalone server ... " + Reset)
	}

	// Channel to listen for shutdown signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a shutdown signal
	<-signalChan
	fmt.Println(Blue + "Shutdown signal received. Initiating graceful shutdown..." + Reset)

	// Create a context with a timeout for the shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown the HTTP server gracefully
	if err := httpServer.Shutdown(ctx); err != nil {
		fmt.Println(Red+"Error shutting down HTTP server:"+Reset, err)
	}

	// close the store gracefully
	store.Close()
	fmt.Println(Green + "Shutdown complete." + Reset)
}
