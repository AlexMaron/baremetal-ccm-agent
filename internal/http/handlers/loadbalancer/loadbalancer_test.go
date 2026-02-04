package loadbalancer_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/http/handlers/loadbalancer"
	"github.com/AlexMaron/baremetal-ccm-agent/internal/nodewatcher"
	"github.com/AlexMaron/baremetal-ccm-agent/pkg/requests/haproxy"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type haproxyMock struct {
	CreateBackendCalled  bool
	AddServerCalled      int
	AddOptionsCalled     bool
	CreateFrontendCalled bool
	AddBindCalled        bool

	FailOn   string
	FailWith error
}

var _ loadbalancer.HAProxyWriter = (*haproxyMock)(nil)

func (m *haproxyMock) CreateBackend(ctx context.Context, request haproxy.BackendRequest) error {
	m.CreateBackendCalled = true
	if m.FailOn == "CreateBackend" {
		if m.FailWith != nil {
			return m.FailWith
		}
		return errors.New("fail")
	}
	return nil
}

func (m *haproxyMock) AddBackendServer(ctx context.Context, _ string, _ haproxy.ServerRequest) error {
	m.AddServerCalled++
	if m.FailOn == "AddBackendServer" {
		return errors.New("fail")
	}
	return nil
}

func (m *haproxyMock) AddBackendOptions(ctx context.Context, _ string, _ any) error {
	m.AddOptionsCalled = true
	return nil
}

func (m *haproxyMock) CreateFrontend(ctx context.Context, _ *haproxy.FrontendRequest) error {
	m.CreateFrontendCalled = true
	return nil
}

func (m *haproxyMock) AddFrontendBinds(ctx context.Context, _ string, _ haproxy.FrontendBindRequest) error {
	m.AddBindCalled = true
	return nil
}

func (m *haproxyMock) DeleteBackend(ctx context.Context, name string) error {
	return nil
}

func (m *haproxyMock) DeleteFrontend(ctx context.Context, name string) error {
	return nil
}

func (m *haproxyMock) DeleteBackendServer(ctx context.Context, backendName, serverName string) error {
	return nil
}

type nodeCacheMock struct {
	nodes []nodewatcher.NodeInfo
}

func (n *nodeCacheMock) GetNodeInfo(fn func(nodewatcher.NodeInfo) bool) {
	for _, node := range n.nodes {
		if !fn(node) {
			return
		}
	}
}

func TestRenderError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	status := http.StatusBadRequest
	msg := "something went wrong"

	loadbalancer.RenderError(w, r, status, msg)

	res := w.Result()
	defer res.Body.Close()

	require.Equal(t, status, res.StatusCode)

	var body struct {
		Error string `json:"error"`
	}

	err := json.NewDecoder(res.Body).Decode(&body)
	require.NoError(t, err)
	require.Equal(t, msg, body.Error)
}

func TestLoadBalancerCreate_OK(t *testing.T) {
	haproxyMock := &haproxyMock{}

	nodes := &nodeCacheMock{
		nodes: []nodewatcher.NodeInfo{
			{Hostname: "node-1", IP: "10.0.0.1"},
			{Hostname: "node-2", IP: "10.0.0.2"},
		},
	}

	handler := loadbalancer.NewLoadBalancerCreate(context.Background(), slog.Default(), net.ParseIP("1.2.3.4"), nodes, haproxyMock)

	body := `{
      "loadbalancer": {
        "name": "test-backend",
        "mode": "tcp",
        "port": 8080,
        "node_port": 31600,
        "backend": {
        },
        "frontend": {
        }
      }
    }`

	req := httptest.NewRequest(http.MethodPost, "/loadbalancer/add", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	handler(rr, req)

	// HTTP-level проверки
	require.Equal(t, http.StatusOK, rr.Code)

	// Поведение
	require.True(t, haproxyMock.CreateBackendCalled)
	require.Equal(t, 2, haproxyMock.AddServerCalled)
	require.True(t, haproxyMock.AddOptionsCalled)
	require.True(t, haproxyMock.CreateFrontendCalled)
	require.True(t, haproxyMock.AddBindCalled)
}

func TestLoadBalancerCreate_ValidationError(t *testing.T) {
	haproxyMock := &haproxyMock{
		FailOn: "CreateBackend",
	}

	handler := loadbalancer.NewLoadBalancerCreate(context.Background(), slog.Default(), net.ParseIP("1.2.3.4"), &nodeCacheMock{}, haproxyMock)

	req := httptest.NewRequest(http.MethodPost, "/loadbalancer/add", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	handler(w, req)

	require.NotEqual(t, http.StatusOK, w.Result().StatusCode)

	resp := w.Result()
	resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, string(respBody), "required")
}

func TestLoadBalancerCreate_BadJSON(t *testing.T) {
	haproxyMock := &haproxyMock{
		FailOn: "CreateBackend",
	}

	handler := loadbalancer.NewLoadBalancerCreate(context.Background(), slog.Default(), net.ParseIP("1.2.3.4"), &nodeCacheMock{}, haproxyMock)

	reqBody := `{
    BAD JSON
    }`

	req := httptest.NewRequest(http.MethodPost, "/loadbalancer/add", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	handler(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestLoadBalancerCreate_ValidateCreateBackendOptions(t *testing.T) {
	haproxyMock := &haproxyMock{
		FailOn: "CreateBackend",
	}

	body := `{
    "loadbalancer": {
      "name": "test-backend",
      "mode": "tcp",
      "port": 8080,
      "node_port": 31600,
      "adv_check": "httpchk"
      }
    }`

	handler := loadbalancer.NewLoadBalancerCreate(context.Background(), slog.Default(), net.ParseIP("1.2.3.4"), &nodeCacheMock{}, haproxyMock)

	req := httptest.NewRequest(http.MethodPost, "/loadbalancer/add", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	require.Equal(t, http.StatusUnprocessableEntity, w.Result().StatusCode)
	require.Contains(t, w.Body.String(), "fail")
}

func TestLoadBalancerDelete(t *testing.T) {
	haproxyMock := &haproxyMock{}

	handler := loadbalancer.LoadBalancerDelete(context.Background(), slog.Default(), net.ParseIP("1.2.3.4"), &nodeCacheMock{}, haproxyMock)

	body := `{
    "loadbalancer": {
      "name": "test-backend"
      }
    }`

	req := httptest.NewRequest(http.MethodDelete, "/loadbalancer/delete", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	require.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestLoadBalancerDelete_BadJSON(t *testing.T) {
	haproxyMock := &haproxyMock{}

	handler := loadbalancer.LoadBalancerDelete(context.Background(), slog.Default(), net.ParseIP("1.2.3.4"), &nodeCacheMock{}, haproxyMock)

	body := `{
    BAD JSON
    }`

	req := httptest.NewRequest(http.MethodDelete, "/loadbalancer/delete", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestLoadBalancerGetIP(t *testing.T) {
	t.Parallel()

	expectedIP := net.ParseIP("1.2.3.4")
	handler := loadbalancer.LoadBalancerGetIP(context.Background(), slog.Default(), expectedIP)

	req := httptest.NewRequest(http.MethodDelete, "/loadbalancer/get", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	res := w.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, w.Result().StatusCode)
	require.Equal(t, "application/json", res.Header.Get("Content-Type"))

	var body struct {
		ExternalIP string `json:"external_ip"`
	}
	err := json.NewDecoder(res.Body).Decode(&body)
	require.NoError(t, err)
	require.Equal(t, expectedIP.String(), body.ExternalIP)
}
