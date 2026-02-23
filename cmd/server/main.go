package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"sift/internal/database"
	"sift/internal/server"
)

func main() {
	dbPath := "sift.db"
	if envPath := os.Getenv("SIFT_DB_PATH"); envPath != "" {
		dbPath = envPath
	}

	dbProvider, err := database.NewSQLiteProvider(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbProvider.Close()

	if err := database.Migrate(dbProvider); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	factory := server.NewFactory(dbProvider)
	mcpServer := factory.CreateMCPServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "Shutting down...")
		cancel()
	}()

	fmt.Fprintln(os.Stderr, "Sift MCP server starting (stdio transport)")

	if err := mcpServer.Start(ctx); err != nil {
		if err != context.Canceled {
			log.Fatalf("MCP server error: %v", err)
		}
	}
}
