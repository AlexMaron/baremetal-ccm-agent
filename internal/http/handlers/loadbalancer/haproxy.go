package loadbalancer

import (
	"context"
	"github.com/AlexMaron/baremetal-ccm-agent/pkg/requests/haproxy"
)

type HAProxyWriter interface {
	CreateBackend(ctx context.Context, request *haproxy.BackendRequest) error
	AddBackendServer(ctx context.Context, backendName string, body *haproxy.ServerRequest) error
	AddBackendOptions(ctx context.Context, backendName string, opts any) error
	CreateFrontend(ctx context.Context, body *haproxy.FrontendRequest) error
	AddFrontendBinds(ctx context.Context, frontendName string, body *haproxy.FrontendBindRequest) error
	DeleteBackend(ctx context.Context, name string) error
	DeleteFrontend(ctx context.Context, name string) error
	DeleteBackendServer(ctx context.Context, backendName, serverName string) error
}

type HAProxyReader interface {
	GetBackendNames(ctx context.Context) ([]string, error)
}
