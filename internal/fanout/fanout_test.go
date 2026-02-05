package fanout

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AlexMaron/baremetal-ccm-agent/pkg/requests/haproxy"
	"github.com/stretchr/testify/require"
)

type mockHaproxyAPI struct {
	failures map[string]int
	calls    map[string]int
}

func newMockHaproxyAPI() *mockHaproxyAPI {
	return &mockHaproxyAPI{
		failures: make(map[string]int),
		calls:    make(map[string]int),
	}
}

func (m *mockHaproxyAPI) GetBackendNames(ctx context.Context) ([]string, error) {
	m.calls["GetBackendNames"]++
	if m.failures["GetBackendNames"] > 0 {
		m.failures["GetBackendNames"]--
		return nil, errors.New("fail")
	}
	return []string{"backend1", "backend2"}, nil
}

func (m *mockHaproxyAPI) GetBackendServers(ctx context.Context, backendName string) ([]haproxy.ServerRequest, error) {
	m.calls["GetBackendServers"]++
	if m.failures["GetBackendServers"] > 0 {
		m.failures["GetBackendServers"]--
		return nil, errors.New("fail")
	}
	return []haproxy.ServerRequest{{Name: "server1"}}, nil
}

func (m *mockHaproxyAPI) CreateBackend(ctx context.Context, req haproxy.BackendRequest) error {
	m.calls["CreateBackend"]++
	if m.failures["CreateBackend"] > 0 {
		m.failures["CreateBackend"]--
		return errors.New("fail")
	}
	return nil
}

func (m *mockHaproxyAPI) AddBackendServer(ctx context.Context, backendName string, body haproxy.ServerRequest) error {
	m.calls["AddBackendServer"]++
	if m.failures["AddBackendServer"] > 0 {
		m.failures["AddBackendServer"]--
		return errors.New("fail")
	}
	return nil
}
func (m *mockHaproxyAPI) AddBackendOptions(ctx context.Context, backendName string, opts any) error {
	m.calls["AddBackendOptions"]++
	if m.failures["AddBackendOptions"] > 0 {
		m.failures["AddBackendOptions"]--
		return errors.New("fail")
	}
	return nil
}

func (m *mockHaproxyAPI) CreateFrontend(ctx context.Context, body *haproxy.FrontendRequest) error {
	m.calls["CreateFrontend"]++
	if m.failures["CreateFrontend"] > 0 {
		m.failures["CreateFrontend"]--
		return errors.New("fail")
	}
	return nil
}

func (m *mockHaproxyAPI) AddFrontendBinds(ctx context.Context, frontendName string, body haproxy.FrontendBindRequest) error {
	m.calls["AddFrontendBinds"]++
	if m.failures["AddFrontendBinds"] > 0 {
		m.failures["AddFrontendBinds"]--
		return errors.New("fail")
	}
	return nil
}

func (m *mockHaproxyAPI) DeleteBackend(ctx context.Context, name string) error {
	m.calls["DeleteBackend"]++
	if m.failures["DeleteBackend"] > 0 {
		m.failures["DeleteBackend"]--
		return errors.New("fail")
	}
	return nil
}

func (m *mockHaproxyAPI) DeleteFrontend(ctx context.Context, name string) error {
	m.calls["DeleteFrontend"]++
	if m.failures["DeleteFrontend"] > 0 {
		m.failures["DeleteFrontend"]--
		return errors.New("fail")
	}
	return nil
}

func (m *mockHaproxyAPI) DeleteBackendServer(ctx context.Context, backendName, serverName string) error {
	m.calls["DeleteBackendServer"]++
	if m.failures["DeleteBackendServer"] > 0 {
		m.failures["DeleteBackendServer"]--
		return errors.New("fail")
	}
	return nil
}

// --------------------- Тесты ---------------------

func TestFanoutClient_Success(t *testing.T) {
	ctx := context.Background()
	m1 := newMockHaproxyAPI()
	m2 := newMockHaproxyAPI()

	client := NewFanoutClient([]haproxy.API{m1, m2}, 2, 1*time.Millisecond)

	err := client.CreateBackend(ctx, haproxy.BackendRequest{Name: "test"})

	require.NoError(t, err)
	require.Equal(t, 1, m1.calls["CreateBackend"])
	require.Equal(t, 1, m2.calls["CreateBackend"])
}

func TestFanoutClient_Retry(t *testing.T) {
	ctx := context.Background()
	m := newMockHaproxyAPI()
	m.failures["CreateBackend"] = 2

	client := NewFanoutClient([]haproxy.API{m}, 3, 1*time.Millisecond)

	err := client.CreateBackend(ctx, haproxy.BackendRequest{Name: "retry-test"})

	require.NoError(t, err)
	require.Equal(t, 3, m.calls["CreateBackend"])
}

func TestFanoutClient_FailureAfterRetries(t *testing.T) {
	ctx := context.Background()
	m := newMockHaproxyAPI()
	m.failures["CreateBackend"] = 5

	client := NewFanoutClient([]haproxy.API{m}, 3, 1*time.Millisecond)

	err := client.CreateBackend(ctx, haproxy.BackendRequest{Name: "fail-test"})

	require.Error(t, err)
	require.Equal(t, 4, m.calls["CreateBackend"])
}

func TestFanoutClient_MultipleClientRetry(t *testing.T) {
	ctx := context.Background()
	m1 := newMockHaproxyAPI()
	m2 := newMockHaproxyAPI()
	m1.failures["CreateBackend"] = 1
	m2.failures["CreateBackend"] = 2

	client := NewFanoutClient([]haproxy.API{m1, m2}, 2, 1*time.Millisecond)

	err := client.CreateBackend(ctx, haproxy.BackendRequest{Name: "multi-client"})

	require.NoError(t, err)
	require.Equal(t, 2, m1.calls["CreateBackend"])
	require.Equal(t, 3, m2.calls["CreateBackend"])
}
