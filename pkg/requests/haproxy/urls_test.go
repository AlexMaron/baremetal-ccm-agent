package haproxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDoRequest_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": "ok"}`))
	}))
	defer ts.Close()

	client := &Client{
		BaseURL:    ts.URL,
		HTTPClient: ts.Client(),
	}

	var out map[string]string

	resp, err := client.doRequest(context.Background(), http.MethodPost, "/test", nil, map[string]string{"a": "b"}, &out)
	require.NoError(t, err)
	require.Equal(t, "ok", out["result"])
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
