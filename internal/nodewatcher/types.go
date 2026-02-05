package nodewatcher

import (
	"context"
	"log/slog"
	"sync"

	"github.com/AlexMaron/baremetal-ccm-agent/internal/fanout"
	"github.com/AlexMaron/baremetal-ccm-agent/pkg/requests/haproxy"
	v1 "k8s.io/api/core/v1"
)

type HAProxyReader interface {
	GetBackendNames(ctx context.Context) ([]string, error)
	GetBackendServers(ctx context.Context, backendName string) (*[]haproxy.ServerRequest, error)
}

type HAProxyWriter interface {
	CreateBackend(ctx context.Context, req *haproxy.BackendRequest) error
	AddBackendServer(ctx context.Context, backendName string, body *haproxy.ServerRequest) error
	DeleteBackend(ctx context.Context, name string) error
	DeleteBackendServer(ctx context.Context, backendName, serverName string) error
	CreateFrontend(ctx context.Context, body *haproxy.FrontendRequest) error
	AddFrontendBinds(ctx context.Context, frontendName string, body *haproxy.FrontendBindRequest) error
}

type NodeCache interface {
	GetNodeInfo(func(NodeInfo) bool)
}

type NodeInfo struct {
	Hostname string
	IP       string
}

type Store struct {
	cache  sync.Map
	log    *slog.Logger
	reader haproxy.API
	writer *fanout.FanoutClient
}

type NodeHandlerFunc func(ctx context.Context, hostname, ip string, node *v1.Node, haproxyReader haproxy.API, haproxyWriter HAProxyWriter)
