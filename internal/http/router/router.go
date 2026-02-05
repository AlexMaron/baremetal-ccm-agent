package router

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/AlexMaron/baremetal-ccm-agent/internal/config"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/fanout"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/http/handlers/loadbalancer"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/nodewatcher"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func BuildRouter(
	ctx context.Context,
	log *slog.Logger,
	externalIP net.IP,
	nodeStore *nodewatcher.Store,
	haproxyClient *fanout.FanoutClient,
	cfg *config.Config,
) http.Handler {
	router := chi.NewRouter()

	router.Use(
		middleware.RequestID,
		middleware.Logger,
		middleware.Recoverer,
		middleware.URLFormat,
	)

	authMiddleware := middleware.BasicAuth("github.com/AlexMaron/baremetal-ccm-agent", map[string]string{
		cfg.Username: cfg.Password,
	})

	router.With(authMiddleware).Post(
		"/api/v1/loadbalancer/add",
		loadbalancer.NewLoadBalancerCreate(ctx, log, externalIP, nodeStore, haproxyClient),
	)
	router.With(authMiddleware).Delete(
		"/api/v1/loadbalancer/delete",
		loadbalancer.LoadBalancerDelete(ctx, log, externalIP, nodeStore, haproxyClient),
	)
	router.With(authMiddleware).Get(
		"/api/v1/loadbalancer/get",
		loadbalancer.LoadBalancerGetIP(ctx, log, externalIP),
	)

	return router
}
