package haproxy_test

import (
	"github.com/AlexMaron/baremetal-ccm-agent/internal/requests/haproxy"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_GetVersion(t *testing.T) {
	t.Parallel()

	var (
		versionCalled bool
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet ||
			r.URL.Path != "/v3/services/haproxy/configuration/version" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		versionCalled = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`1`))
	}))
	defer ts.Close()

	client := &haproxy.Client{
		BaseURL:    ts.URL,
		HTTPClient: ts.Client(),
	}

	version, err := client.GetConfigVersion(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(1), version)

	require.True(t, versionCalled, "version endpoint was not called")
}
