package config_test

import (
	"testing"
	"time"

	"github.com/AlexMaron/baremetal-ccm-agent/internal/config"
	"github.com/stretchr/testify/require"
)

func TestMustLoad_Success(t *testing.T) {
    t.Setenv("ENV", "dev")
	t.Setenv("HTTP_ADDRESS", "localhost:8082")
	t.Setenv("HTTP_TIMEOUT", "4s")
	t.Setenv("HTTP_IDLE_TIMEOUT", "60s")
	t.Setenv("KUBECONFIG", "kubeconfig")
	t.Setenv("DATA_PLANE_HOSTS", "127.0.0.1,127.0.0.2")
	t.Setenv("USERNAME", "user")
	t.Setenv("PASSWORD", "pass")
	t.Setenv("DATA_PLANE_USERNAME", "haproxy")
	t.Setenv("DATA_PLANE_PASSWORD", "secret")

    cfg := config.MustLoad()

    require.Equal(t, "dev", cfg.Env)
    require.Equal(t, "localhost:8082", cfg.HTTPAddress)
    require.Equal(t, 4 * time.Second, cfg.HTTPTimeout)
    require.Equal(t, 60 * time.Second, cfg.HTTPIdleTimeout)
    require.Equal(t, "kubeconfig", cfg.Kubeconfig)
    require.Equal(t, []string{"127.0.0.1", "127.0.0.2"}, cfg.DataPlaneHosts)
    require.Equal(t, "user", cfg.Username)
    require.Equal(t, "pass", cfg.Password)
    require.Equal(t, "haproxy", cfg.DataPlaneUsername)
    require.Equal(t, "secret", cfg.DataPlanePassword)
}
