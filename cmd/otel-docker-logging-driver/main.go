package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/go-plugins-helpers/sdk"

	"github.com/moritzloewenstein/otel-docker-logging-driver/internal/config"
	"github.com/moritzloewenstein/otel-docker-logging-driver/internal/driver"
	"github.com/moritzloewenstein/otel-docker-logging-driver/internal/otelx"
)

func main() {
	cfg := config.FromEnv()

	exp, provider, err := otelx.SetupProvider(context.Background(), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to setup otlp exporter: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = provider.Shutdown(ctx)
		_ = exp.Shutdown(ctx)
	}()

	drv := driver.New(cfg, provider)

	h := sdk.NewHandler(`{"Implements": ["LoggingDriver"]}`)
	driver.RegisterHandlers(&h, drv)

	// Graceful shutdown of the HTTP server on unix socket is handled by Docker.
	go func() {
		if err := h.ServeUnix("otel-logs", 0); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "serve error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for SIGINT/SIGTERM to exit cleanly
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
