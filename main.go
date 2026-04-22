package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"boot.dev/linko/internal/store"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	httpPort := flag.Int("port", 8899, "port to listen on")
	dataDir := flag.String("data", "./data", "directory to store data")
	flag.Parse()

	status := run(ctx, cancel, *httpPort, *dataDir)
	cancel()
	os.Exit(status)
}

func run(ctx context.Context, cancel context.CancelFunc, httpPort int, dataDir string) int {
	f, err := os.OpenFile("linko.access.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0776)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	var accessLogger = log.New(f, "INFO: ", log.LstdFlags)
	var stdLogger = log.New(os.Stderr, "DEBUG: ", log.LstdFlags)

	stdLogger.Printf("Linko is running on http://localhost:%d", httpPort)
	st, err := store.New(dataDir, stdLogger)
	if err != nil {
		stdLogger.Printf("failed to create store: %v", err)
		return 1
	}
	s := newServer(*st, httpPort, cancel, accessLogger)
	var serverErr error
	go func() {
		serverErr = s.start()
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.shutdown(shutdownCtx); err != nil {
		stdLogger.Printf("failed to shutdown server: %v", err)
		return 1
	}
	if serverErr != nil {
		stdLogger.Printf("server error: %v", serverErr)
		return 1
	}
	stdLogger.Printf("Linko is shutting down")
	return 0
}
