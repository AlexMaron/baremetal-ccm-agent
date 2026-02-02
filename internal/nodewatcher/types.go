package nodewatcher

import (
	"context"
	"log/slog"
	"sync"

	v1 "k8s.io/api/core/v1"
)

type NodeCache interface {
	GetNodeInfo(func(NodeInfo) bool)
}

type NodeInfo struct {
	Hostname string
	IP       string
}

type Store struct {
	cache sync.Map
	log   *slog.Logger
}

type NodeHandlerFunc func(ctx context.Context, hostname, ip string, node *v1.Node, haproxyClient HAProxyAPI)
