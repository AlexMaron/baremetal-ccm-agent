package loadbalancer

import (
	"cccm-agent/internal/requests/haproxy"
	"context"
)

type HAProxyAPI interface {
	CreateBackend(ctx context.Context, request haproxy.BackendRequest) error
	AddBackendServer(ctx context.Context, backendName string, body haproxy.ServerRequest) error
	AddBackendOptions(ctx context.Context, backendName string, opts any) error
	CreateFrontend(ctx context.Context, body *haproxy.FrontendRequest) error
	AddFrontendBinds(ctx context.Context, frontendName string, body haproxy.FrontendBindRequest) error
	DeleteBackend(ctx context.Context, name string) error
	DeleteFrontend(ctx context.Context, name string) error
}
