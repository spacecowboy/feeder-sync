package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spacecowboy/feeder-sync/internal/server"
)

func main() {
	conn := os.Getenv("FEEDER_SYNC_POSTGRES_CONN")

	if conn == "" {
		log.Fatal("FEEDER_SYNC_POSTGRES_CONN environment variable not set")
	}

	router, err := server.NewServerWithPostgres(conn)

	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if err := router.Close(); err != nil {
			fmt.Printf("Failed to close server: %v", err)
			return
		}
	}()

	srv := &http.Server{
		Addr:    ":34217",
		Handler: router,
	}

	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		log.Println("Serving on %q...", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Println("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
