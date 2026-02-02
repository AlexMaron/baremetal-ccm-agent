package main

import (
	"github.com/AlexMaron/baremetal-ccm-agent/internal/config"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/http/router"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/lib/externalip"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/lib/logger/sl"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/nodewatcher"
	"github.com/AlexMaron/baremetal-ccm-agent/pkg/requests/haproxy"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

func main() {
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)
	log.Info("Starting github.com/AlexMaron/baremetal-ccm-agent", slog.String("env", cfg.Env))

	haproxyClient := &haproxy.Client{
		BaseURL: "http://localhost:5555",
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		Username: cfg.HaproxyAuth.Username,
		Password: cfg.HaproxyAuth.Password,
		Log:      setupLogger(cfg.Env),
	}

    if err := run(context.Background(), cfg, log, haproxyClient); err != nil {
        log.Error("Agent stopped with error", sl.Err(err))
        os.Exit(1)
    }
}

func run(ctx context.Context, cfg *config.Config, log *slog.Logger, haproxyClient *haproxy.Client) error {
	externalIP, err := externalip.GetExternalIP()
    if err != nil {
        return fmt.Errorf("failed to get external IP: %w", err)
    }

	nodeStore := nodewatcher.NewStore(log)
	stopCh := nodewatcher.StartWatcher(ctx, cfg.Kubeconfig, nodeStore.HandleNode, haproxyClient)

    router := router.BuildRouter(ctx, log, externalIP, nodeStore, haproxyClient, cfg)

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.Idle_timeout,
	}

	go func() {
		log.Info("HTTP server starting", slog.String("addr", cfg.HTTPServer.Address))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Error("HTTP server failed", sl.Err(err))
		}
	}()

    select {
    case <-ctx.Done():
    case <-stopCh:
    }

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown failed", sl.Err(err))
	}

	log.Info("Agent stopped gracefully")
    return nil
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envDev:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}
	return log
}
