package haproxy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) CreateBackend(ctx context.Context, request *BackendRequest) error {
	// Get Haprxoy config version
	version, err := c.GetConfigVersion(ctx)
	if err != nil {
		return err
	}

	normalizer := BackendNormalizer{
		rules: []BackendRule{
			DefaultBackendModeRule,
			TCPBalanceRule,
			AdvCheckValidationRule,
			DefaultServerValidateRule,
		},
	}
	if err := normalizer.Normalize(request); err != nil {
		return err
	}

	log.Printf("After normalize: backend.Balance.Algorithm=%q", request.Balance.Algorithm)

	body := request

	c.bodyJSONLog(ctx, body)

	q := url.Values{}
	q.Set("version", strconv.FormatInt(version, 10))

	resp, err := c.doRequest(
		ctx,
		http.MethodPost,
		pathBackends,
		q,
		body,
		nil,
	)
	if resp == nil {
		return err
	}
	if resp.StatusCode == 409 {
		updateBackend := fmt.Sprintf(pathReplaceBackend, body.Name)
		resp, err = c.doRequest(
			ctx,
			http.MethodPut,
			updateBackend,
			q,
			body,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to create or update backend: %w", err)
		}
		if resp.StatusCode >= 300 && resp.StatusCode != 409 {
			return fmt.Errorf("failed to create or update backend: status=%d", resp.StatusCode)
		}
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) DeleteBackend(ctx context.Context, name string) error {
	// Get Haprxoy config version
	version, err := c.GetConfigVersion(ctx)
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("version", strconv.FormatInt(version, 10))
	path := fmt.Sprintf(pathReplaceBackend, name)

	resp, err := c.doRequest(
		ctx,
		http.MethodDelete,
		path,
		q,
		nil,
		nil,
	)
	if resp == nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) GetBackendServers(ctx context.Context, backendName string) (*[]ServerRequest, error) {
	var servers *[]ServerRequest
	path := fmt.Sprintf(pathServers, backendName)
	resp, err := c.doRequest(
		ctx,
		http.MethodGet,
		path,
		nil,
		nil,
		&servers,
	)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return servers, nil
}

func (c *Client) AddBackendServer(ctx context.Context, backendName string, body *ServerRequest) error {
	// Get Haprxoy config version
	version, err := c.GetConfigVersion(ctx)
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("version", strconv.FormatInt(version, 10))

	path := fmt.Sprintf(pathServers, backendName)
	resp, err := c.doRequest(
		ctx,
		http.MethodPost,
		path,
		q,
		body,
		nil,
	)
	if resp == nil {
		return err
	}
	if resp.StatusCode == 409 {
		updatePath := fmt.Sprintf(pathReplaceServer, backendName, body.Name)
		resp, err = c.doRequest(
			ctx,
			http.MethodPut,
			updatePath,
			q,
			body,
			nil,
		)
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) DeleteBackendServer(ctx context.Context, backendName, serverName string) error {
	// Get Haprxoy config version
	version, err := c.GetConfigVersion(ctx)
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("version", strconv.FormatInt(version, 10))

	path := fmt.Sprintf(pathDeleteServer, backendName, serverName)
	resp, err := c.doRequest(
		ctx,
		http.MethodDelete,
		path,
		q,
		nil,
		nil,
	)
	if resp == nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) GetBackendNames(ctx context.Context) ([]string, error) {
	var backends []BackendRequest
	path := fmt.Sprint(pathBackends)
	resp, err := c.doRequest(
		ctx,
		http.MethodGet,
		path,
		nil,
		nil,
		&backends,
	)
	if err != nil {
		return nil, err
	}

	backendNames := make([]string, 0, len(backends))
	for _, b := range backends {
		backendNames = append(backendNames, b.Name)
	}
	defer resp.Body.Close()

	return backendNames, nil
}

func (c *Client) AddBackendOptions(ctx context.Context, backendName string, opts any) error {
	// Get Haprxoy config version
	version, err := c.GetConfigVersion(ctx)
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("version", strconv.FormatInt(version, 10))

	path := fmt.Sprintf(pathReplaceBackend, backendName)
	resp, err := c.doRequest(
		ctx,
		http.MethodPut,
		path,
		q,
		opts,
		nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) CreateFrontend(ctx context.Context, body *FrontendRequest) error {
	// Get Haprxoy config version
	version, err := c.GetConfigVersion(ctx)
	if err != nil {
		return err
	}

	normalizer := FrontendNormalizer{
		rules: []FrontendRule{
			DefaultFrontendModeRule,
			ValidateFronend,
		},
	}
	if err := normalizer.Normalize(body); err != nil {
		return err
	}

	c.bodyJSONLog(ctx, body)

	q := url.Values{}
	q.Set("version", strconv.FormatInt(version, 10))

	resp, err := c.doRequest(
		ctx,
		http.MethodPost,
		pathAddFrontends,
		q,
		body,
		nil,
	)
	if resp == nil {
		return err
	}
	if resp.StatusCode == 409 {
		updatePath := fmt.Sprintf(pathReplaceFrontend, body.Name)
		resp, err = c.doRequest(
			ctx,
			http.MethodPut,
			updatePath,
			q,
			body,
			nil,
		)
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) AddFrontendBinds(ctx context.Context, frontendName string, body *FrontendBindRequest) error {
	// Get Haprxoy config version
	version, err := c.GetConfigVersion(ctx)
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("version", strconv.FormatInt(version, 10))

	path := fmt.Sprintf(pathAddFrontendBinds, body.Name)
	resp, err := c.doRequest(
		ctx,
		http.MethodPost,
		path,
		q,
		body,
		nil,
	)
	if resp == nil {
		return err
	}
	if resp.StatusCode == 409 {
		updatePath := fmt.Sprintf(pathReplaceFrontendBinds, frontendName, body.Name)
		resp, err = c.doRequest(
			ctx,
			http.MethodPut,
			updatePath,
			q,
			body,
			nil,
		)
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) DeleteFrontend(ctx context.Context, name string) error {
	// Get Haprxoy config version
	version, err := c.GetConfigVersion(ctx)
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("version", strconv.FormatInt(version, 10))
	path := fmt.Sprintf(pathReplaceFrontend, name)

	resp, err := c.doRequest(
		ctx,
		http.MethodDelete,
		path,
		q,
		nil,
		nil,
	)
	if resp == nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
