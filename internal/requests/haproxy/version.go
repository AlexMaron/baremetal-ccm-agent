package haproxy

import (
	"context"
	"net/http"
)

func (c *Client) GetConfigVersion(ctx context.Context) (int64, error) {
	var resp int64
	_, err := c.doRequest(ctx, http.MethodGet, pathVersion, nil, nil, &resp)
	if err != nil {
		return 0, err
	}

	return resp, nil
}
