package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/AlexMaron/baremetal-ccm-agent/internal/config"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/fanout"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/http/router"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/lib/externalip"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/lib/logger/sl"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/nodewatcher"
	"github.com/AlexMaron/baremetal-ccm-agent/pkg/requests/haproxy"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

func main() {
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)
	log.Info("Starting github.com/AlexMaron/baremetal-ccm-agent", slog.String("env", cfg.Env))

	if err := run(context.Background(), cfg, log); err != nil {
		log.Error("Agent stopped with error", sl.Err(err))
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	externalIP, err := externalip.GetExternalIP()
	if err != nil {
		return fmt.Errorf("failed to get external IP: %w", err)
	}

	reader := haproxyClient(cfg.DataPlaneHosts[0], *cfg)
	var writers []haproxy.API
	for _, host := range cfg.DataPlaneHosts {
		c := haproxyClient(host, *cfg)
		writers = append(writers, c)
	}

	fanout := fanout.NewFanoutClient(writers, 3, time.Second)

	nodeStore := nodewatcher.NewStore(log, reader, fanout)
	stopCh := nodewatcher.StartWatcher(ctx, cfg.Kubeconfig, nodeStore.HandleNode, reader, fanout)

	router := router.BuildRouter(ctx, log, externalIP, nodeStore, fanout, cfg)

	srv := &http.Server{
		Addr:         cfg.HTTPAddress,
		Handler:      router,
		ReadTimeout:  cfg.HTTPTimeout,
		WriteTimeout: cfg.HTTPTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}

	go func() {
		log.Info("HTTP server starting", slog.String("addr", cfg.HTTPAddress))
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

func haproxyClient(baseURL string, cfg config.Config) *haproxy.Client {
	return &haproxy.Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		Username: cfg.DataPlaneUsername,
		Password: cfg.DataPlanePassword,
		Log:      setupLogger(cfg.Env),
	}

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
