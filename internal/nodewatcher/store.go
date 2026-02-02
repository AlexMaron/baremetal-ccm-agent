package nodewatcher

import (
	"github.com/AlexMaron/baremetal-ccm-agent/internal/lib/logger/sl"
	"github.com/AlexMaron/baremetal-ccm-agent/pkg/requests/haproxy"
	"context"
	"fmt"
	"log/slog"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func NewStore(log *slog.Logger) *Store {
    s := &Store{
        log: log,
    }
    return s
}

func (s *Store) HandleNode(ctx context.Context, hostname, ip string, node *v1.Node, haproxyClient HAProxyAPI) {
    if s.shouldSkipNode(hostname, ip) {
        return
    }

    if s.handleUnschedulable(ctx, hostname, node, haproxyClient) {
        return
    }

    if s.handleNotReady(ctx, hostname, node, haproxyClient) {
        return
    }

    s.upsertNode(hostname, ip)
	s.SyncNodeToBackends(ctx, haproxyClient)
}

func (s *Store) shouldSkipNode(hostname, ip string) bool {
    if ip == "" {
        s.log.Info("Skipping node with no IP", slog.String("hostname", hostname))
        return true
    }
    return false
}

func (s *Store) handleUnschedulable(ctx context.Context, hostname string, node *v1.Node, haproxyClient HAProxyAPI) bool {
    if !node.Spec.Unschedulable {
        return false
    }

    s.removeNode(hostname, "Unschedulable")
    s.removeNodeFromBackends(ctx, hostname, haproxyClient)
    return true
}

func (s *Store) handleNotReady(ctx context.Context, hostname string, node *v1.Node, haproxyClient HAProxyAPI) bool {
    if s.isNodeReady(node) {
        return false
    }

    s.removeNode(hostname, "NotReady")
    s.removeNodeFromBackends(ctx, hostname, haproxyClient)
    return true
}

func (s *Store) removeNode(hostname, reason string) {
    s.cache.Delete(hostname)
    klog.InfoS("Node removed from cache", "hostname", hostname, "reason", reason)
}

func (s *Store) isNodeReady(node *v1.Node) bool {
    for _, cond := range node.Status.Conditions {
        if cond.Type == v1.NodeReady {
            return cond.Status == v1.ConditionTrue
        }
    }
    s.log.Warn("Node has no NodeReady condition", slog.String("node", node.Name))
    return false
}

func (s *Store) listBackends(ctx context.Context, haproxyClient HAProxyAPI) ([]string, error) {
    backends, err := haproxyClient.GetBackendNames(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to  get backends: %v", err)
    }
    return backends, nil
}

func (s *Store) removeNodeFromBackends(ctx context.Context, hostname string, haproxyClient HAProxyAPI) {
    backends, err := s.listBackends(ctx, haproxyClient)
    if err != nil {
        return
    }

    for _, backend := range backends {
        haproxyClient.DeleteBackendServer(ctx, backend, hostname)
    }
}

func (s *Store) upsertNode(hostname, ip string) {
    s.cache.Store(hostname, NodeInfo{
        Hostname: hostname,
        IP: ip,
    })
}

func (s *Store) SyncNodeToBackends(ctx context.Context, haproxyClient HAProxyAPI) {
    backends, err := s.listBackends(ctx, haproxyClient)
    if err != nil {
        return
    }

    for _, backend := range backends {
        s.syncBackend(ctx, haproxyClient, backend)
    }
}


func (s *Store) syncBackend(ctx context.Context, haproxyClient HAProxyAPI, backend string) {
    servers, err := s.getBackendServers(ctx, haproxyClient, backend)
    if err != nil {
        return
    }

    port, existingMap, ok := s.prepareBackendState(servers, backend)
    if !ok {
        return
    }

    s.syncNodesToBackend(ctx, haproxyClient, backend, port, existingMap)
}

func (s *Store) getBackendServers(ctx context.Context, haproxyClient HAProxyAPI, backend string) ([]haproxy.ServerRequest, error) {
    servers, err := haproxyClient.GetBackendServers(ctx, backend)
    if err != nil {
        s.log.Error(
            "failed to get backend servers port",
            sl.Err(err),
            slog.String("backend", backend),
            )
        return nil, err
    }
    return servers, nil
}

func (s *Store) prepareBackendState(
    servers []haproxy.ServerRequest,
    backend string,
) (port int32, existing map[string]int32, ok bool) {
    if len(servers) == 0 {
        s.log.Info("backend has no servers, skipping", slog.String("backend", backend))
        return 0, nil, false
    }

    existing = make(map[string]int32, len(servers))
    for _, srv := range servers {
        existing[srv.Name] = srv.Port
    }

    return servers[0].Port, existing, true
}

func (s *Store) syncNodesToBackend(
    ctx context.Context,
    haproxyClient HAProxyAPI,
    backend string,
    port int32,
    existing map[string]int32,
) {
    s.cache.Range(func(_, value any) bool {
        node, ok := value.(NodeInfo)
        if !ok {
            return true
        }

        if s.nodeAlreadyExists(node, existing, port) {
            return true
        }

        s.addNodeToBackend(ctx, haproxyClient, backend, node, port)
        return true
    })
}

func (s *Store) nodeAlreadyExists (node NodeInfo, existing map[string]int32, port int32) bool {
    existingPort, exists := existing[node.Hostname]
    return exists && existingPort == port
}

func (s *Store) addNodeToBackend(
    ctx context.Context,
    haproxyClient HAProxyAPI,
    backend string,
    node NodeInfo,
    port int32,
) {
    serverBody := haproxy.ServerRequest{
        Name: node.Hostname,
        Address: node.IP,
        Port: port,
        Check: haproxy.ServerCheckEnabled,
    }

    if err := haproxyClient.AddBackendServer(ctx, backend, serverBody); err != nil {
        s.log.Info(
            "failed to add node to backend",
            sl.Err(err),
            slog.String("backend", backend),
            slog.String("node", node.Hostname),
            )
        return
    }

    s.log.Info(
		"node added to backend",
		slog.String("backend", backend),
		slog.String("node", node.Hostname),
		slog.Int64("port", int64(port)),
        )
}

func (s *Store) GetNodeInfo(fn func(NodeInfo) bool) {
	s.cache.Range(func(_, value any) bool {
		node, ok := value.(NodeInfo)
		if !ok {
			return true
		}
		return fn(node)
	})
}
