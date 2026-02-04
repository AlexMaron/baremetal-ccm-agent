package externalip

import (
	"fmt"
	"net"
)

func GetExternalIPFromAddrs(addrs []net.Addr) (net.IP, error) {
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip == nil {
			continue
		}

		if ip.To4() != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("no external IPv4")
}

func GetExternalIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		if ip, err := GetExternalIPFromAddrs(addrs); err != nil {
			return ip, nil
		}
	}

	return nil, fmt.Errorf("no global unicast IPv4 found")
}
