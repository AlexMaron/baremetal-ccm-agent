package loadbalancer

import (
	"github.com/AlexMaron/baremetal-ccm-agent/internal/lib/api/response"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/lib/logger/sl"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/nodewatcher"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/requests/haproxy"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type NodeSource interface {
	GetNodeInfo(func(hostname string, ip string) bool)
}

type LoadBalancerCreateRequest struct {
    Name     string                        `json:"name" validate:"required"`
	Port     int32                         `json:"port" validate:"required"`
	NodePort int32                         `json:"node_port" validate:"required"`
	Backend  haproxy.BackendRequest        `json:"backend" validate:"required"`
	Frontend haproxy.FrontendRequest       `json:"frontend" validate:"required"`
}

type CreateRequest struct {
	LB LoadBalancerCreateRequest `json:"loadbalancer"`
}

type LoadBalancerDeleteRequest struct {
    Name     string                        `json:"name" validate:"required"`
}

type DeleteRequest struct {
	LB LoadBalancerDeleteRequest `json:"loadbalancer"`
}

type Response struct {
	response.Response
	ExternalIP net.IP `json:"external_ip"`
}

func RenderError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	w.WriteHeader(status)
	render.JSON(w, r, response.Error(msg))
}

func NewLoadBalancerCreate(ctx context.Context, log *slog.Logger, externalIP net.IP, nodesCache nodewatcher.NodeCache, haproxyClient HAProxyAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.loadbalancer.NewLoadBalancerCreate"
		log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req CreateRequest

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("Failed to decode request body", sl.Err(err))
			RenderError(w, r, http.StatusInternalServerError, err.Error())

			return
		}

		log.Info("Request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))

            render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.ValidateError(validateErr))

			return
		}

		if err := haproxyClient.CreateBackend(ctx, req.LB.Backend); err != nil {
			log.Info("HAProxy error", sl.Err(err))
			var vErr *haproxy.ValidationError
			if errors.As(err, &vErr) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				if err := json.NewEncoder(w).Encode(vErr); err != nil {
					log.Error("failed to encode response", sl.Err(err))
				}
                return
			}
			RenderError(w, r, http.StatusInternalServerError, err.Error())
			return
		}

		nodesCache.GetNodeInfo(func(node nodewatcher.NodeInfo) bool {
			serverBody := &haproxy.ServerRequest{
				Name:    node.Hostname,
				Address: node.IP,
				Port:    req.LB.NodePort,
				Check:   haproxy.ServerCheckEnabled,
			}
			if err := haproxyClient.AddBackendServer(ctx, req.LB.Name, *serverBody); err != nil {
				log.Info("HAProxy error", sl.Err(err))
				RenderError(w, r, http.StatusInternalServerError, err.Error())
				return false
			}
			return true
		})

		optsBody := &req.LB.Backend
		if err := haproxyClient.AddBackendOptions(ctx, req.LB.Name, *optsBody); err != nil {
			log.Info("HAProxy error", sl.Err(err))
			RenderError(w, r, http.StatusInternalServerError, err.Error())
			return
		}

		frontendBody := &req.LB.Frontend
		if err := haproxyClient.CreateFrontend(ctx, frontendBody); err != nil {
			log.Info("HAProxy error", sl.Err(err))
			RenderError(w, r, http.StatusInternalServerError, err.Error())
			return
		}

		bindBody := &haproxy.FrontendBindRequest{
			Name:    req.LB.Name,
			Address: "*",
			Port:    req.LB.Port,
		}
		if err := haproxyClient.AddFrontendBinds(ctx, req.LB.Name, *bindBody); err != nil {
			log.Info("HAProxy error", sl.Err(err))
			RenderError(w, r, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("Load balancer successfully created")

		render.JSON(w, r, Response{
			Response:   response.OK(),
			ExternalIP: externalIP,
		})
	}
}

func LoadBalancerDelete(ctx context.Context, log *slog.Logger, externalIP net.IP, nodesCache nodewatcher.NodeCache, haproxyClient HAProxyAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.loadbalancer.LoadBalancerDelete"
		log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req DeleteRequest

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("Failed to decode request body", sl.Err(err))
			RenderError(w, r, http.StatusInternalServerError, err.Error())

			return
		}

		log.Info("Request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))

            render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.ValidateError(validateErr))

			return
		}

		if err := haproxyClient.DeleteFrontend(ctx, req.LB.Name); err != nil {
			log.Info("HAProxy error", sl.Err(err))
			RenderError(w, r, http.StatusInternalServerError, err.Error())
			return
		}

		if err := haproxyClient.DeleteBackend(ctx, req.LB.Name); err != nil {
			log.Info("HAProxy error", sl.Err(err))
			RenderError(w, r, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("Load balancer successfully created")

		render.JSON(w, r, Response{
			Response:   response.OK(),
			ExternalIP: externalIP,
		})
	}
}

func LoadBalancerGetIP(ctx context.Context, log *slog.Logger, externalIP net.IP) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.loadbalancer.LoadBalancerGet"
		log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		log.Info("Load balancer ip has been successfully assigned")

		render.JSON(w, r, Response{
			Response:   response.OK(),
			ExternalIP: externalIP,
		})
	}
}
