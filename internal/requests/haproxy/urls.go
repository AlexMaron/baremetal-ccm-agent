package haproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	apiBase = "/v3/services/haproxy/configuration"

	pathVersion              = apiBase + "/version"
	pathBackends             = apiBase + "/backends"
	pathReplaceBackend       = apiBase + "/backends/%s"
	pathServers              = apiBase + "/backends/%s/servers"
	pathReplaceServer        = apiBase + "/backends/%s/servers/%s"
	pathDeleteServer         = apiBase + "/backends/%s/servers/%s"
	pathAddFrontends         = apiBase + "/frontends"
	pathReplaceFrontend      = apiBase + "/frontends/%s"
	pathAddFrontendBinds     = apiBase + "/frontends/%s/binds"
	pathReplaceFrontendBinds = apiBase + "/frontends/%s/binds/%s"
	pathReplaceBinds         = apiBase + "/frontends/binds/%s"
)

func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values, body, out any) (*http.Response, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	u.Path = path
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		if resp.StatusCode == 409 {
			return resp, nil
		}
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error: %s (%d)", b, resp.StatusCode)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return nil, fmt.Errorf("failed to decode respose body: %w", err)
		}
	}

	return resp, nil
}
