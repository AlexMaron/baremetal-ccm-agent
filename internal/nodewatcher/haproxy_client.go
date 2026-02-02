package nodewatcher

import (
	"cccm-agent/internal/requests/haproxy"
	"context"
)

type HAProxyAPI interface {
	GetBackendNames(ctx context.Context) ([]string, error)
	GetBackendServers(ctx context.Context, backendName string) ([]haproxy.ServerRequest, error)
	DeleteBackendServer(ctx context.Context, backendName, serverName string) error
	AddBackendServer(ctx context.Context, backendName string, body haproxy.ServerRequest) error
}
