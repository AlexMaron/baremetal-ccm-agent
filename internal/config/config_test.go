package config_test

import (
	"github.com/AlexMaron/baremetal-ccm-agent/internal/config"
	"os"
	"testing"
	"time"
)

func TestMustLoad_Success(t *testing.T) {
	yaml := `
env: dev
kubeconfig: kubeconfig
externalIP:
  iface: eth0
auth:
  username: user
  password: pass
haproxy_auth:
  username: haproxy
  password: secret
haproxy_defaults:
  default_server:
    inter: 3
    fastinter: 1
    fall: 3
    rise: 2
    on-marked-down: shutdown-sessions
  balance:
    algorithm: roundrobin
  `

	file, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(file.Name())

	if _, err := file.WriteString(yaml); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	file.Close()

	// Переопределяем аргументы командной строки
	os.Args = []string{"cmd", "--config", file.Name()}

	cfg := config.MustLoad()

	if cfg.Env != "dev" {
		t.Errorf("expected Env=dev, got %s", cfg.Env)
	}

	if cfg.HTTPServer.Timeout != 4*time.Second {
		t.Errorf("expected Timeout=4s, got %s", cfg.HTTPServer.Timeout)
	}

	// Проверяем, что дефолтные значения установлены
	env := "dev"
	if cfg.Env != env {
		t.Errorf("expected default Env=%s, got %s", env, cfg.Env)
	}
	address := "localhost:8082"
	if cfg.HTTPServer.Address != address {
		t.Errorf("expected default Address=%s, got %s", address, cfg.HTTPServer.Address)
	}
	timeout := 4 * time.Second
	if cfg.HTTPServer.Timeout != timeout {
		t.Errorf("expected default Timeout=%s, got %s", timeout, cfg.HTTPServer.Timeout)
	}
}
