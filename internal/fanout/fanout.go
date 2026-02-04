package fanout

import (
	"context"
	"errors"
	"time"

	"github.com/AlexMaron/baremetal-ccm-agent/pkg/requests/haproxy"
)

type FanoutClient struct {
    clients []haproxy.API
    retries int
    delay   time.Duration
}

func NewFanoutClient(clients []haproxy.API, retries int, delay time.Duration) *FanoutClient {
    return &FanoutClient{
        clients: clients,
        retries: retries,
        delay: delay,
    }
}

func (f *FanoutClient) invoke(fn func(haproxy.API) error) error {
    var finalErr error
    for _, c := range f.clients {
        var err error
        for i := 0; i <= f.retries; i++ {
            err = fn(c)
            if err == nil {
            break
            }
            time.Sleep(f.delay)
        }
        if err != nil {
            finalErr = errors.Join(finalErr, err)
        }
    }

    return finalErr
}

func (f *FanoutClient) CreateBackend(ctx context.Context, request haproxy.BackendRequest) error {
	return f.invoke(func(c haproxy.API) error {
		return c.CreateBackend(ctx, request)
	})
}

func (f *FanoutClient) AddBackendServer(ctx context.Context, backendName string, body haproxy.ServerRequest) error {
	return f.invoke(func(c haproxy.API) error {
		return c.AddBackendServer(ctx, backendName, body)
	})
}

func (f *FanoutClient) AddBackendOptions(ctx context.Context, backendName string, opts any) error {
	return f.invoke(func(c haproxy.API) error {
		return c.AddBackendOptions(ctx, backendName, opts)
	})
}

func (f *FanoutClient) CreateFrontend(ctx context.Context, body *haproxy.FrontendRequest) error {
	return f.invoke(func(c haproxy.API) error {
		return c.CreateFrontend(ctx, body)
	})
}

func (f *FanoutClient) AddFrontendBinds(ctx context.Context, frontendName string, body haproxy.FrontendBindRequest) error {
	return f.invoke(func(c haproxy.API) error {
		return c.AddFrontendBinds(ctx, frontendName, body)
	})
}

func (f *FanoutClient) DeleteBackend(ctx context.Context, name string) error {
	return f.invoke(func(c haproxy.API) error {
		return c.DeleteBackend(ctx, name)
	})
}

func (f *FanoutClient) DeleteFrontend(ctx context.Context, name string) error {
	return f.invoke(func(c haproxy.API) error {
		return c.DeleteFrontend(ctx, name)
	})
}

func (f *FanoutClient) DeleteBackendServer(ctx context.Context, backendName, serverName string) error {
	return f.invoke(func(c haproxy.API) error {
		return c.DeleteBackendServer(ctx, backendName, serverName)
	})
}

func (f *FanoutClient) GetBackendNames(ctx context.Context) ([]string, error) {
    return nil, errors.New("fanout client does not support read operations")
}

