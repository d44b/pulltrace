package main

import (
	"context"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/d44b/pulltrace/internal/server"
	"github.com/d44b/pulltrace/web"
)

func main() {
	cfg := server.ConfigFromEnv()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		cancel()
	}()

	// Get embedded web UI filesystem
	var webFS fs.FS
	embedded, err := web.FS()
	if err != nil {
		log.Printf("warning: web UI not available: %v", err)
	} else {
		webFS = embedded
	}

	s := server.New(cfg, webFS)
	if err := s.Run(ctx); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
