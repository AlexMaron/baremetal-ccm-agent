package externalip_test

import (
	"github.com/AlexMaron/baremetal-ccm-agent/internal/lib/externalip"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetExternalIP_Error(t *testing.T) {
	addrs := []net.Addr{
		&net.IPNet{IP: net.ParseIP("127.0.0.1")},
		&net.IPNet{IP: net.ParseIP("192.168.1.10")},
	}

	ip, err := externalip.GetExternalIPFromAddrs(addrs)
	require.Error(t, err)
	require.Nil(t, ip)
}

func TestGetExternalIP_Ok(t *testing.T) {
	addrs := []net.Addr{
		&net.IPNet{IP: net.ParseIP("192.168.1.10")},
		&net.IPNet{IP: net.ParseIP("8.8.8.8")},
	}

	ip, err := externalip.GetExternalIPFromAddrs(addrs)
	require.NoError(t, err)
	require.Equal(t, net.ParseIP("8.8.8.8"), ip)
}
