package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/haoyuan1313/go-inventory-sync-demo/internal/inventory"
)

func main() {
	port := flag.Int("port", 8080, "local HTTP port")
	host := flag.String("host", "127.0.0.1", "listen host; use 0.0.0.0 only inside a host-restricted Docker demo")
	statePath := flag.String("state", "data/inventory-state.json", "path to persisted demo state")
	threshold := flag.Int("threshold", 10, "low stock threshold (strictly below)")
	flag.Parse()
	if *port < 1 || *port > 65535 || *threshold < 1 {
		slog.Error("invalid port or threshold")
		os.Exit(2)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	store, err := inventory.OpenStore(*statePath)
	if err != nil {
		logger.Error("failed to open state", "error", err)
		os.Exit(1)
	}
	mock := inventory.NewMockSupplier()
	base := "http://127.0.0.1:" + strconv.Itoa(*port)
	engine := &inventory.Engine{Source: inventory.SupplierClient{URL: base + "/mock/supplier/items", Attempts: 3}, Store: store, Threshold: *threshold, Logger: logger}
	server := &http.Server{Addr: *host + ":" + strconv.Itoa(*port), Handler: inventory.NewHandler(engine, mock), ReadHeaderTimeout: 5 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	logger.Info("inventory sync demo ready", "url", base, "state", *statePath, "threshold", *threshold)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
