package haproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

func (c *Client) bodyJSONLog(ctx context.Context, body any) {
	if c.Log != nil && c.Log.Enabled(ctx, slog.LevelDebug) {
		b, err := json.MarshalIndent(body, "", "  ")
		if err != nil {
			c.Log.Error("failed to marshal backend body", slog.Any("err", err))
		} else {
			fmt.Println(string(b))
		}
	}
}

func (b *BackendRequest) RequiresBackendHTTP() bool {
	if b == nil {
		return false
	}

	switch b.Balance.Algorithm {
	case "hdr", "uri", "uri_param":
		return true
	default:
		return false
	}
}

func ValidateFronend(request *FrontendRequest) error {
	if request.DefaultBackend == "" {
		return fmt.Errorf("DefaultBackend is empty.")
	}
	return nil
}

func DefaultFrontendModeRule(f *FrontendRequest) error {
	if f.Mode == "" {
		f.Mode = "tcp"
	}
	return nil
}

type BackendRule func(*BackendRequest) error

type BackendNormalizer struct {
	rules []BackendRule
}

func (n *BackendNormalizer) Normalize(b *BackendRequest) error {
	for _, rule := range n.rules {
		if err := rule(b); err != nil {
			return err
		}
	}
	return nil
}

type FrontendRule func(*FrontendRequest) error

type FrontendNormalizer struct {
	rules []FrontendRule
}

func (n *FrontendNormalizer) Normalize(b *FrontendRequest) error {
	for _, rule := range n.rules {
		if err := rule(b); err != nil {
			return err
		}
	}
	return nil
}

func DefaultBackendModeRule(b *BackendRequest) error {
	if b.Mode == "" {
		b.Mode = "tcp"
	}
	return nil
}

func TCPBalanceRule(b *BackendRequest) error {
	if b.Mode == "tcp" && b.Balance.Algorithm == "" {
		b.Balance.Algorithm = "leastconn"
	}
	return nil
}

func AdvCheckValidationRule(b *BackendRequest) error {
	if b.Mode == "http" {
		b.AdvCheck = "httpchk"
	} else {
		b.AdvCheck = "tcp-check"
	}
	return nil
}

func DefaultServerValidateRule(b *BackendRequest) error {
	if b.DefaultServer.Inter == 0 {
		b.DefaultServer.Inter = 3000
	}
	if b.DefaultServer.Fastinter == 0 {
		b.DefaultServer.Fastinter = 1000
	}
	if b.DefaultServer.Fall == 0 {
		b.DefaultServer.Fall = 4
	}
	if b.DefaultServer.Rise == 0 {
		b.DefaultServer.Rise = 3
	}
	if b.DefaultServer.OnMarkedDown == "" {
		b.DefaultServer.OnMarkedDown = "shutdown-sessions"
	}
	return nil
}
