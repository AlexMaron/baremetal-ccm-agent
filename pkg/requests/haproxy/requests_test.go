package haproxy_test

import (
	"github.com/AlexMaron/baremetal-ccm-agent/pkg/requests/haproxy"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type haproxyMockCalls struct {
	GetVersion           bool
	GetBackend           bool
	GetBackendNames      bool
	PostBackend          bool
	PutBackend           bool
	DeleteBackend        bool
	GetBackendServers    bool
	PostBackendServers   bool
	PutBackendServers    bool
	DeleteBackendServers bool
	ForcePostConflict    bool
	PostFrontend         bool
	PutFrontend          bool
	PostFrontendBinds    bool
	DeleteFrontend       bool
}

const backendName = "test-backend"
const serverName = "test-server"

func testLogger() *slog.Logger {
	return slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}),
	)
}

func newHaproxyTestClient(t *testing.T, calls *haproxyMockCalls) *haproxy.Client {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {

		// GET version test
		case r.Method == http.MethodGet &&
			r.URL.Path == "/v3/services/haproxy/configuration/version":

			calls.GetVersion = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`1`))
			return

		// POST backend test
		case r.Method == http.MethodPost &&
			r.URL.Path == "/v3/services/haproxy/configuration/backends":

			if calls.ForcePostConflict {
				calls.PostBackend = false
				w.WriteHeader(http.StatusConflict)
			} else {
				calls.PostBackend = true
				w.WriteHeader(http.StatusOK)
			}
			require.Equal(t, "1", r.URL.Query().Get("version"))

			var body haproxy.BackendRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

			require.Equal(t, backendName, body.Name)
			require.Equal(t, "tcp", body.Mode)

			return

		// PUT backend
		case r.Method == http.MethodPut &&
			r.URL.Path == "/v3/services/haproxy/configuration/backends/"+backendName:

			calls.PutBackend = true
			require.Equal(t, "1", r.URL.Query().Get("version"))

			w.WriteHeader(http.StatusOK)
			return

		case r.Method == http.MethodDelete &&
			r.URL.Path == "/v3/services/haproxy/configuration/backends/"+backendName:

			calls.DeleteBackend = true
			require.Equal(t, "1", r.URL.Query().Get("version"))

			w.WriteHeader(http.StatusOK)
			return

		case r.Method == http.MethodGet &&
			r.URL.Path == "/v3/services/haproxy/configuration/backends/test-backend/servers":

			calls.GetBackendServers = true
			servers := []haproxy.ServerRequest{
				{
					Name:    "server-1",
					Address: "10.0.0.1",
					Port:    80,
					Check:   haproxy.ServerCheckEnabled,
				},
				{
					Name:    "server-2",
					Address: "10.0.0.2",
					Port:    8080,
					Check:   haproxy.ServerCheckDisabled,
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			require.NoError(t, json.NewEncoder(w).Encode(servers))
			return

		// PUT backend servers
		case r.Method == http.MethodPost &&
			r.URL.Path == "/v3/services/haproxy/configuration/backends/test-backend/servers":

			calls.PostBackendServers = true
			require.Equal(t, "1", r.URL.Query().Get("version"))
			w.WriteHeader(http.StatusOK)
			return

		// DELETE backend servers
		case r.Method == http.MethodDelete &&
			r.URL.Path == "/v3/services/haproxy/configuration/backends/test-backend/servers/"+serverName:

			calls.DeleteBackendServers = true
			require.Equal(t, "1", r.URL.Query().Get("version"))
			w.WriteHeader(http.StatusOK)
			return

		// GET backend names
		case r.Method == http.MethodGet &&
			r.URL.Path == "/v3/services/haproxy/configuration/backends":

			backendNames := []haproxy.BackendRequest{
				{
					Name: "server-1",
				},
			}

			calls.GetBackendNames = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			require.NoError(t, json.NewEncoder(w).Encode(backendNames))
			return

		// POST frontend
		case r.Method == http.MethodPost &&
			r.URL.Path == "/v3/services/haproxy/configuration/frontends":

			if calls.ForcePostConflict {
				calls.PostFrontend = false
				w.WriteHeader(http.StatusConflict)
			} else {
				calls.PostFrontend = true
				w.WriteHeader(http.StatusOK)
			}
			require.Equal(t, "1", r.URL.Query().Get("version"))

			var body haproxy.FrontendRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

			require.Equal(t, backendName, body.Name)
			require.Equal(t, "tcp", body.Mode)
			return

		// PUT frontend
		case r.Method == http.MethodPut &&
			r.URL.Path == "/v3/services/haproxy/configuration/frontends/"+backendName:

			calls.PutFrontend = true
			require.Equal(t, "1", r.URL.Query().Get("version"))

			w.WriteHeader(http.StatusOK)
			return

		// POST frontend binds
		case r.Method == http.MethodPost &&
			r.URL.Path == "/v3/services/haproxy/configuration/frontends/"+backendName+"/binds":

			calls.PostFrontendBinds = true
			require.Equal(t, "1", r.URL.Query().Get("version"))

			w.WriteHeader(http.StatusOK)
			return

		// DELETE frontend
		case r.Method == http.MethodDelete &&
			r.URL.Path == "/v3/services/haproxy/configuration/frontends/"+backendName:

			calls.DeleteFrontend = true
			require.Equal(t, "1", r.URL.Query().Get("version"))

			w.WriteHeader(http.StatusOK)
			return
		}

		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))

	t.Cleanup(ts.Close)

	return &haproxy.Client{
		BaseURL:    ts.URL,
		HTTPClient: ts.Client(),
		Log:        testLogger(),
	}
}

func TestClient_CreateBackend(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	calls.ForcePostConflict = false
	client := newHaproxyTestClient(t, calls)

	request := &haproxy.BackendRequest{
		Name: backendName,
		Mode: "tcp",
	}

	err := client.CreateBackend(context.Background(), *request)

	require.NoError(t, err)

	require.True(t, calls.GetVersion, "GET version was not called")
	require.True(t, calls.PostBackend, "POST backend was not called")
	require.False(t, calls.PutBackend)
}

func TestClient_BackendHTTPValidationFail(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	calls.ForcePostConflict = false
	client := newHaproxyTestClient(t, calls)

	request := &haproxy.BackendRequest{
		Name: backendName,
		Mode: "http",
		DefaultServer: haproxy.DefaultServer{
			SendProxy: "enabled",
		},
	}

	err := client.CreateBackend(context.Background(), *request)

	// Проверяем что валидация рабоает
	var vErr *haproxy.ValidationError
	require.ErrorAs(t, err, &vErr, "Validation error expected.")
	require.Equal(t, "UnsupportedCombination", vErr.Reason)
	require.Equal(t, "send-proxy is not allowed in http mode", vErr.Message)
	require.Equal(t, "DefaultServer.SendProxy", vErr.Field)
}

func TestClient_BackendTCPAdvCheckValidationFail(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	calls.ForcePostConflict = false
	client := newHaproxyTestClient(t, calls)

	request := &haproxy.BackendRequest{
		Name:     backendName,
		AdvCheck: "httpchk",
	}

	err := client.CreateBackend(context.Background(), *request)

	// Проверяем что валидация рабоает
	var vErr *haproxy.ValidationError
	require.ErrorAs(t, err, &vErr, "Validation error expected.")
	require.Equal(t, "UnsupportedCombination", vErr.Reason)
	require.Equal(t, "httpchk is not allowed in tcp mode", vErr.Message)
}

func TestClient_BackendTCPBalanceValidationFail(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	calls.ForcePostConflict = false
	client := newHaproxyTestClient(t, calls)

	request := &haproxy.BackendRequest{
		Name: backendName,
		Balance: haproxy.Balance{
			Algorithm: "uri",
		},
	}

	err := client.CreateBackend(context.Background(), *request)

	// Проверяем что валидация рабоает
	var vErr *haproxy.ValidationError
	require.ErrorAs(t, err, &vErr, "Validation error expected.")
	require.Equal(t, haproxy.ReasonUnsupportedCombination, vErr.Reason)
	require.Equal(t, haproxy.MsgHTTPBalanceInTCP, vErr.Message)
}

func TestClient_CreateBackendIfExists(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	calls.ForcePostConflict = true
	client := newHaproxyTestClient(t, calls)

	request := haproxy.BackendRequest{
		Name:           backendName,
		Mode:           "tcp",
		ConnectTimeout: "5s",
	}

	err := client.CreateBackend(context.Background(), request)
	require.NoError(t, err)

	require.True(t, calls.GetVersion, "version endpoint was not called")
	require.False(t, calls.PostBackend, "POST backend was not called")
	require.True(t, calls.PutBackend, "PUT backend was not called")
}

func TestClient_DeleteBackend(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	client := newHaproxyTestClient(t, calls)

	err := client.DeleteBackend(context.Background(), backendName)
	require.NoError(t, err)

	require.True(t, calls.GetVersion, "version endpoint was not called")
	require.True(t, calls.DeleteBackend, "DELETE backend was not called")
}

func TestClient_GetBackendServers(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	client := newHaproxyTestClient(t, calls)

	servers, err := client.GetBackendServers(context.Background(), backendName)
	require.NoError(t, err)
	require.Len(t, servers, 2)
	require.Equal(t, "server-1", servers[0].Name)
	require.Equal(t, "10.0.0.1", servers[0].Address)
	require.Equal(t, "server-2", servers[1].Name)
	require.Equal(t, "10.0.0.2", servers[1].Address)

	require.True(t, calls.GetBackendServers, "GET backend was not called")
}

func TestClient_PostBackendServers(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	client := newHaproxyTestClient(t, calls)

	servers := haproxy.ServerRequest{
		Name:    "server-1",
		Address: "10.0.0.1",
		Port:    80,
		Check:   haproxy.ServerCheckEnabled,
	}
	err := client.AddBackendServer(context.Background(), backendName, servers)
	require.NoError(t, err)
	require.Equal(t, "server-1", servers.Name)
	require.Equal(t, "10.0.0.1", servers.Address)

	require.True(t, calls.GetVersion, "version endpoint was not called")
	require.True(t, calls.PostBackendServers, "POST backend was not called")
}

func TestClient_DeleteBackendServers(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	client := newHaproxyTestClient(t, calls)

	err := client.DeleteBackendServer(context.Background(), backendName, serverName)
	require.NoError(t, err)

	require.True(t, calls.DeleteBackendServers, "DELETE backend was not called")
}

func TestClient_GetBackendNames(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	client := newHaproxyTestClient(t, calls)

	names, err := client.GetBackendNames(context.Background())
	require.NoError(t, err)
	require.Len(t, names, 1)
	require.Equal(t, "server-1", names[0])

	require.True(t, calls.GetBackendNames, "GET backend was not called")
}

func TestClient_AddBackendOptions(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	client := newHaproxyTestClient(t, calls)

	balance := haproxy.Balance{
			Algorithm: "leastconn",
    }
	defaultServer := haproxy.DefaultServer{
			Inter:        3000,
			Fastinter:    1000,
			Fall:         3,
			Rise:         4,
			OnMarkedDown: "session-shutdown",
    }

	optsBody := &haproxy.BackendRequest{
		Name:          backendName,
		AdvCheck:      "tcp-check",
		Balance:       balance,
		DefaultServer: defaultServer,
		Mode:          "http",
	}

	err := client.AddBackendOptions(context.Background(), backendName, optsBody)
	require.NoError(t, err)

	require.True(t, calls.PutBackend, "PUT backend was not called")
}

func TestClient_CreateFrontend(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	calls.ForcePostConflict = false
	client := newHaproxyTestClient(t, calls)

	body := &haproxy.FrontendRequest{
		Name:           backendName,
		DefaultBackend: backendName,
		Mode:           "tcp",
	}

	err := client.CreateFrontend(context.Background(), body)
	require.NoError(t, err)

	require.True(t, calls.GetVersion, "version endpoint was not called")
	require.True(t, calls.PostFrontend, "POST backend was not called")
	require.False(t, calls.PutFrontend, "PUT backend was not called")
}

func TestClient_CreateFrontendFail(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	calls.ForcePostConflict = false
	client := newHaproxyTestClient(t, calls)

	body := &haproxy.FrontendRequest{
		Name:           backendName,
		DefaultBackend: "",
		Mode:           "tcp",
	}

	err := client.CreateFrontend(context.Background(), body)
	vErr := &haproxy.ValidationError{}
	require.ErrorAs(t, err, &vErr, "frontend validation error expected")
	require.Equal(t, body.DefaultBackend, "", "fronend validation DefaultBackend not empty")
}

func TestClient_CreateFrontendIfExists(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	calls.ForcePostConflict = true
	client := newHaproxyTestClient(t, calls)

	body := &haproxy.FrontendRequest{
		Name:           backendName,
		DefaultBackend: backendName,
		Mode:           "tcp",
	}

	err := client.CreateFrontend(context.Background(), body)
	require.NoError(t, err)

	require.True(t, calls.GetVersion, "version endpoint was not called")
	require.False(t, calls.PostFrontend, "POST backend was not called")
	require.True(t, calls.PutFrontend, "PUT backend was not called")
}

func TestClient_AddFrontendBinds(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	client := newHaproxyTestClient(t, calls)

	body := haproxy.FrontendBindRequest{
		Name:    backendName,
		Address: "10.0.0.1",
		Port:    80,
	}

	err := client.AddFrontendBinds(context.Background(), backendName, body)
	require.NoError(t, err)

	require.True(t, calls.GetVersion, "version endpoint was not called")
	require.True(t, calls.PostFrontendBinds, "POST backend was not called")
}

func TestClient_DeleteFrontend(t *testing.T) {
	t.Parallel()

	calls := &haproxyMockCalls{}
	client := newHaproxyTestClient(t, calls)

	err := client.DeleteFrontend(context.Background(), backendName)
	require.NoError(t, err)

	require.True(t, calls.GetVersion, "version endpoint was not called")
	require.True(t, calls.DeleteFrontend, "DELETE backend was not called")
}
