package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/d44b/pulltrace/internal/agent"
)

func main() {
	cfg := agent.ConfigFromEnv()
	if cfg.NodeName == "" {
		log.Fatal("PULLTRACE_NODE_NAME is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		cancel()
	}()

	a := agent.New(cfg)
	if err := a.Run(ctx); err != nil {
		log.Fatalf("agent failed: %v", err)
	}
}
