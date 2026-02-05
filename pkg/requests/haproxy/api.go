package haproxy

import "context"

type API interface {
	// reader
	GetBackendNames(ctx context.Context) ([]string, error)
	GetBackendServers(ctx context.Context, backendName string) (*[]ServerRequest, error)

	// writer
	CreateBackend(ctx context.Context, request *BackendRequest) error
	AddBackendServer(ctx context.Context, backendName string, body *ServerRequest) error
	DeleteBackendServer(ctx context.Context, backendName, serverName string) error
	DeleteBackend(ctx context.Context, name string) error
	AddBackendOptions(ctx context.Context, backendName string, opts any) error
	CreateFrontend(ctx context.Context, body *FrontendRequest) error
	AddFrontendBinds(ctx context.Context, frontendName string, body *FrontendBindRequest) error
	DeleteFrontend(ctx context.Context, name string) error
}

var _ API = (*Client)(nil)
