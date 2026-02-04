package nodewatcher

import (
	"context"
	"fmt"
	"github.com/AlexMaron/baremetal-ccm-agent/pkg/requests/haproxy"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testHostname = "test-node-01"

type MockHAProxyAPI struct {
	mock.Mock
	AddedServers []haproxy.ServerRequest
}

func (m *MockHAProxyAPI) CreateBackend(ctx context.Context, request haproxy.BackendRequest) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

func (m *MockHAProxyAPI) CreateFrontend(ctx context.Context, body *haproxy.FrontendRequest) error {
	args := m.Called(ctx, body)
	return args.Error(0)
}

func (m *MockHAProxyAPI) AddServer(ctx context.Context, backend string, server haproxy.ServerRequest) error {
	m.AddedServers = append(m.AddedServers, server)
	return nil
}

func (m *MockHAProxyAPI) DeleteBackend(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockHAProxyAPI) DeleteFrontend(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockHAProxyAPI) DeleteBackendServer(ctx context.Context, backend, server string) error {
	args := m.Called(ctx, backend, server)
	return args.Error(0)
}

func (m *MockHAProxyAPI) AddBackendServer(ctx context.Context, backend string, body haproxy.ServerRequest) error {
	args := m.Called(ctx, backend, body)
	return args.Error(0)
}

func (m *MockHAProxyAPI) GetBackendNames(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockHAProxyAPI) AddBackendOptions(ctx context.Context, backendName string, opts any) error {
	args := m.Called(ctx, backendName, opts)
	return args.Error(0)
}

func (m *MockHAProxyAPI) AddFrontendBinds(ctx context.Context, frontendName string, body haproxy.FrontendBindRequest) error {
	args := m.Called(ctx, frontendName, body)
	return args.Error(0)
}

func (m *MockHAProxyAPI) GetBackendServers(ctx context.Context, backend string) ([]haproxy.ServerRequest, error) {
	args := m.Called(ctx, backend)
	return args.Get(0).([]haproxy.ServerRequest), args.Error(1)
}

func mockServersData() ([]haproxy.ServerRequest, []haproxy.ServerRequest, map[string]int32) {
	expectedServers := []haproxy.ServerRequest{
		{
			Name:    "test-node-01",
			Address: "10.0.0.1",
			Port:    8080,
			Check:   haproxy.ServerCheckEnabled,
		},
		{
			Name:    "test-node-02",
			Address: "10.0.0.2",
			Port:    8080,
			Check:   haproxy.ServerCheckEnabled,
		},
	}
	existingServers := []haproxy.ServerRequest{
		{
			Name:    "test-node-03",
			Address: "10.0.0.3",
			Port:    8080,
			Check:   haproxy.ServerCheckEnabled,
		},
		{
			Name:    "test-node-04",
			Address: "10.0.0.4",
			Port:    8080,
			Check:   haproxy.ServerCheckEnabled,
		},
	}
	serversMap := make(map[string]int32, len(expectedServers))
	for _, srv := range expectedServers {
		serversMap[srv.Name] = srv.Port
	}
	return expectedServers, existingServers, serversMap
}

func NewTestNode_Ready(name string, ip string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"node-role.kubernetes.io/test": "true"},
		},
		Status: v1.NodeStatus{
			Addresses: []v1.NodeAddress{
				{
					Type:    v1.NodeInternalIP,
					Address: ip,
				},
				{
					Type:    v1.NodeHostName,
					Address: name,
				},
			},
			Conditions: []v1.NodeCondition{
				{
					Type:   v1.NodeReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}
}

func NewTestNode_Unschedulable(name string, ip string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"node-role.kubernetes.io/test": "true"},
		},
		Spec: v1.NodeSpec{
			Unschedulable: true,
		},
		Status: v1.NodeStatus{
			Addresses: []v1.NodeAddress{
				{
					Type:    v1.NodeInternalIP,
					Address: ip,
				},
				{
					Type:    v1.NodeHostName,
					Address: name,
				},
			},
			Conditions: []v1.NodeCondition{
				{
					Type:   v1.NodeReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}
}

func NewTestNode_NotReady(name string, ip string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"node-role.kubernetes.io/test": "true"},
		},
		Status: v1.NodeStatus{
			Addresses: []v1.NodeAddress{
				{
					Type:    v1.NodeInternalIP,
					Address: ip,
				},
				{
					Type:    v1.NodeHostName,
					Address: name,
				},
			},
			Conditions: []v1.NodeCondition{
				{
					Type:   v1.NodeReady,
					Status: v1.ConditionFalse,
				},
			},
		},
	}
}

func TestHandleNode_Ready(t *testing.T) {
	ctx := context.Background()
	s := &Store{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	expectedServers, existingServers, _ := mockServersData()
	hostname := expectedServers[0].Name
	ip := expectedServers[0].Address
	for _, server := range expectedServers {
		s.cache.Store(server.Name, NodeInfo{
			Hostname: server.Name,
			IP:       server.Address,
		})
	}
	node := NewTestNode_Ready(expectedServers[0].Name, expectedServers[0].Address)

	haproxyReader := &MockHAProxyAPI{}
	haproxyWriter := &MockHAProxyAPI{}
	haproxyWriter.On("DeleteBackendServer", ctx, "test-backend-01", hostname).Return(nil)
	haproxyReader.On("GetBackendNames", ctx).Return([]string{"test-backend-01"}, nil)
	haproxyReader.On("GetBackendServers", ctx, "test-backend-01").Return(existingServers, nil)
	for _, server := range expectedServers {
		haproxyWriter.On("AddBackendServer", ctx, "test-backend-01", server).Return(nil)
	}

	s.HandleNode(ctx, hostname, ip, node, haproxyReader, haproxyWriter)

	haproxyWriter.AssertNotCalled(t, "DeleteBackendServer", ctx, "test-backend-01", hostname)
	haproxyReader.AssertCalled(t, "GetBackendServers", ctx, "test-backend-01")
	haproxyWriter.AssertCalled(t, "AddBackendServer", ctx, "test-backend-01", expectedServers[0])
}

func TestHandleNode_NoReady(t *testing.T) {
	ctx := context.Background()
	s := &Store{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	hostname := "test-node-01"
	ip := "10.0.0.1"
	node := NewTestNode_NotReady(hostname, ip)

	haproxyReader := &MockHAProxyAPI{}
	haproxyWriter := &MockHAProxyAPI{}
	haproxyWriter.On("DeleteBackendServer", ctx, "test-backend-01", hostname).Return(nil)
	haproxyReader.On("GetBackendNames", ctx).Return([]string{"test-backend-01"}, nil)
	haproxyReader.On("GetBackendServers", ctx, "test-backend-01").Return([]haproxy.ServerRequest{}, fmt.Errorf("force error")).Maybe()
	haproxyWriter.On("AddBackendServer", ctx, "test-backend-01", hostname).Return(fmt.Errorf("force error")).Maybe()

	s.HandleNode(ctx, hostname, ip, node, haproxyReader, haproxyWriter)

	haproxyWriter.AssertCalled(t, "DeleteBackendServer", ctx, "test-backend-01", hostname)
	haproxyReader.AssertNotCalled(t, "GetBackendServers", ctx, "test-backend-01")
	haproxyReader.AssertNotCalled(t, "AddBackendServer", ctx, "test-backend-01", hostname)
}

func TestHandleNode_Unschedulable(t *testing.T) {
	ctx := context.Background()
	s := &Store{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	hostname := "test-node-01"
	ip := "10.0.0.1"
	node := NewTestNode_Unschedulable(hostname, ip)

	haproxyReader := &MockHAProxyAPI{}
	haproxyWriter := &MockHAProxyAPI{}
	haproxyWriter.On("DeleteBackendServer", ctx, "test-backend-01", hostname).Return(nil).Maybe()
	haproxyReader.On("GetBackendNames", ctx).Return([]string{"test-backend-01"}, nil)
	haproxyReader.On("GetBackendServers", ctx, "test-backend-01").Return([]haproxy.ServerRequest{}, fmt.Errorf("force error")).Maybe()
	haproxyWriter.On("AddBackendServer", ctx, "test-backend-01", hostname).Return(fmt.Errorf("force error")).Maybe()

	s.HandleNode(ctx, hostname, ip, node, haproxyReader, haproxyWriter)

	haproxyWriter.AssertCalled(t, "DeleteBackendServer", ctx, "test-backend-01", hostname)
	haproxyReader.AssertCalled(t, "GetBackendNames", ctx)
	haproxyReader.AssertNotCalled(t, "GetBackendServers", ctx, "test-backend-01")
	haproxyWriter.AssertNotCalled(t, "AddBackendServer", ctx, "test-backend-01", hostname)
}

func TestHandleNode_SkipNode(t *testing.T) {
	ctx := context.Background()
	s := &Store{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	hostname := "test-node-01"
	ip := ""
	node := NewTestNode_Unschedulable(hostname, ip)

	haproxyReader := &MockHAProxyAPI{}
	haproxyWriter := &MockHAProxyAPI{}
	haproxyWriter.On("DeleteBackendServer", ctx, "test-backend-01", hostname).Return(nil).Maybe()
	haproxyReader.On("GetBackendNames", ctx).Return([]string{"test-backend-01"}, nil).Maybe()
	haproxyReader.On("GetBackendServers", ctx, "test-backend-01").Return([]haproxy.ServerRequest{}, fmt.Errorf("force error")).Maybe()
	haproxyWriter.On("AddBackendServer", ctx, "test-backend-01", hostname).Return(fmt.Errorf("force error")).Maybe()

	s.HandleNode(ctx, hostname, ip, node, haproxyReader, haproxyWriter)

	haproxyWriter.AssertNotCalled(t, "DeleteBackendServer", ctx, "test-backend-01", hostname)
	haproxyReader.AssertNotCalled(t, "GetBackendServers", ctx, "test-backend-01")
	haproxyReader.AssertNotCalled(t, "GetBackendNames", ctx)
	haproxyWriter.AssertNotCalled(t, "AddBackendServer", ctx, "test-backend-01", hostname)
}

func Test_shoudSkipNode(t *testing.T) {
	s := &Store{log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	ip := ""

	ok := s.shouldSkipNode("test-node-01", ip)

	require.True(t, ok)
}

func Test_handleUnschedulable(t *testing.T) {
	ctx := context.Background()
	s := &Store{}
	node := NewTestNode_Unschedulable("test-node-01", "10.0.0.1")

	haproxyReader := &MockHAProxyAPI{}
	haproxyWriter := &MockHAProxyAPI{}

	haproxyReader.On("GetBackendNames", ctx).Return([]string{"test-backend-01", "test-backend-02"}, nil)
	haproxyWriter.On("DeleteBackendServer", ctx, "test-backend-01", "test-node-01").Return(nil)
	haproxyWriter.On("DeleteBackendServer", ctx, "test-backend-02", "test-node-01").Return(nil)

	ok := s.handleUnschedulable(ctx, "test-node-01", node, haproxyReader, haproxyWriter)
	require.True(t, ok)

	haproxyReader.AssertExpectations(t)
	haproxyWriter.AssertExpectations(t)
}

func Test_handleNotReady(t *testing.T) {
	ctx := context.Background()
	s := &Store{}
	node := NewTestNode_NotReady("test-node-01", "10.0.0.1")

	haproxyReader := &MockHAProxyAPI{}
	haproxyWriter := &MockHAProxyAPI{}

	haproxyReader.On("GetBackendNames", ctx).Return([]string{"test-backend-01", "test-backend-02"}, nil)
	haproxyWriter.On("DeleteBackendServer", ctx, "test-backend-01", "test-node-01").Return(nil)
	haproxyWriter.On("DeleteBackendServer", ctx, "test-backend-02", "test-node-01").Return(nil)

	ok := s.handleNotReady(ctx, "test-node-01", node, haproxyReader, haproxyWriter)
	require.True(t, ok)

	haproxyReader.AssertExpectations(t)
	haproxyWriter.AssertExpectations(t)
}

func TestRemoveNode(t *testing.T) {
	s := &Store{}
	ip := "10.0.0.1"
	s.cache.Store(testHostname, ip)
	s.cache.Store("test-node-02", "10.0.0.1")

	_, ok := s.cache.Load(testHostname)
	require.True(t, ok)

	s.removeNode(testHostname, "testRemoveNode")

	_, ok = s.cache.Load(testHostname)
	require.False(t, ok)
	_, ok = s.cache.Load("test-node-02")
	require.True(t, ok)
}

func Test_removeNodeFromBackend(t *testing.T) {
	ctx := context.Background()
	s := &Store{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	haproxyReader := &MockHAProxyAPI{}
	haproxyWriter := &MockHAProxyAPI{}

	backendNames := []string{
		"test-backend-01",
	}

	haproxyReader.On("GetBackendNames", ctx).Return(backendNames, nil)
	haproxyWriter.On("DeleteBackendServer", ctx, "test-backend-01", "node-01").Return(nil)

	s.removeNodeFromBackends(ctx, "node-01", haproxyReader, haproxyWriter)

	haproxyReader.AssertExpectations(t)
	haproxyWriter.AssertExpectations(t)
}

func Test_removeNodeFromBackend_listBackends_Fail(t *testing.T) {
	ctx := context.Background()
	s := &Store{}

	haproxyReader := &MockHAProxyAPI{}
	haproxyWriter := &MockHAProxyAPI{}

	haproxyReader.On("GetBackendNames", ctx).Return([]string{}, fmt.Errorf("failed to  get backends"))
	haproxyWriter.On("DeleteBackendServer", ctx, "test-backend-01", "node-01").Return(nil).Maybe()

	s.removeNodeFromBackends(ctx, "node-01", haproxyReader, haproxyWriter)

	haproxyWriter.AssertNotCalled(t, "DeleteBackendServer", ctx, "test-backend-01", "node-01")
	haproxyReader.AssertExpectations(t)
	haproxyWriter.AssertExpectations(t)
}

func TestUpsertNode(t *testing.T) {
	s := &Store{}

	s.upsertNode(testHostname, "10.0.0.1")

	v, ok := s.cache.Load(testHostname)
	require.True(t, ok)

	node := v.(NodeInfo)
	require.Equal(t, testHostname, node.Hostname)
	require.Equal(t, "10.0.0.1", node.IP)
}

func TestSyncBackend(t *testing.T) {
	ctx := context.Background()
	s := &Store{}
	s.log = slog.New(slog.NewTextHandler(os.Stdout, nil))

	haproxyReader := &MockHAProxyAPI{}
	haproxyWriter := &MockHAProxyAPI{}

	expectedServers, existingServers, _ := mockServersData()
	for _, server := range expectedServers {
		s.cache.Store(server.Name, NodeInfo{
			Hostname: server.Name,
			IP:       server.Address,
		})
	}

	// Мокаем GetBackendServers
	haproxyReader.On("GetBackendServers", ctx, "test-backend-01").Return(existingServers, nil)

	// Ловим любые вызовы AddBackendServer и возвращаем nil (иначе panic)
	for _, server := range expectedServers {
		haproxyWriter.On("AddBackendServer", ctx, "test-backend-01", server).Return(nil)
	}

	s.syncBackend(ctx, haproxyReader, haproxyWriter, "test-backend-01")

	haproxyReader.AssertExpectations(t)
	haproxyWriter.AssertExpectations(t)
}

func TestSyncBackend_GetBackendServersReturn(t *testing.T) {
	ctx := context.Background()
	s := &Store{}
	s.log = slog.New(slog.NewTextHandler(os.Stdout, nil))

	haproxyReader := &MockHAProxyAPI{}
	haproxyWriter := &MockHAProxyAPI{}

	// Мокаем GetBackendServers
	haproxyReader.On("GetBackendServers", mock.Anything, "backend-error").Return([]haproxy.ServerRequest{}, fmt.Errorf("TEST GetBackendServers error"))
	haproxyWriter.On("AddBackendServer", ctx, "backend-error").
		Return(nil).Maybe()

	s.syncBackend(ctx, haproxyReader, haproxyWriter, "backend-error")

	haproxyReader.AssertCalled(t, "GetBackendServers", mock.Anything, mock.Anything, mock.Anything)
	haproxyWriter.AssertNotCalled(t, "AddBackendServer", ctx, "backend-error")
}

func TestSyncBackend_PrepareBackendStateReturn(t *testing.T) {
	ctx := context.Background()
	s := &Store{}
	s.log = slog.New(slog.NewTextHandler(os.Stdout, nil))

	haproxyReader := &MockHAProxyAPI{}
	haproxyWriter := &MockHAProxyAPI{}

	// Мокаем GetBackendServers
	haproxyReader.On("GetBackendServers", mock.Anything, "backend-error").Return([]haproxy.ServerRequest{}, nil)
	haproxyWriter.On("AddBackendServer", ctx, "backend-error").
		Return(nil).Maybe()

	s.syncBackend(ctx, haproxyReader, haproxyWriter, "backend-error")

	haproxyReader.AssertCalled(t, "GetBackendServers", mock.Anything, mock.Anything, mock.Anything)
	haproxyWriter.AssertNotCalled(t, "AddBackendServer", ctx, "backend-error")
}

func TestGetBackendServers(t *testing.T) {
	ctx := context.Background()

	s := &Store{}
	s.log = slog.New(slog.NewTextHandler(os.Stdout, nil))

	haproxyReader := &MockHAProxyAPI{}

	expectedServers, _, _ := mockServersData()
	haproxyReader.On("GetBackendServers", ctx, "test-backend-01").Return(expectedServers, nil)

	servers, err := s.getBackendServers(ctx, haproxyReader, "test-backend-01")
	require.NoError(t, err)
	require.Equal(t, expectedServers, servers)
}

func TestPrepareBackendState(t *testing.T) {
	s := &Store{}
	s.log = slog.New(slog.NewTextHandler(os.Stdout, nil))

	expectedServers, _, expectedMap := mockServersData()
	port, existingMap, ok := s.prepareBackendState(expectedServers, "test-backend-01")
	require.Equal(t, port, int32(8080))
	require.Equal(t, existingMap, expectedMap)
	require.True(t, ok)

}

func TestSyncNodeToBackeds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	s := &Store{}
	s.log = slog.New(slog.NewTextHandler(os.Stdout, nil))
	haproxyReader := &MockHAProxyAPI{}
	haproxyWriter := &MockHAProxyAPI{}

	haproxyReader.On("GetBackendServers", mock.Anything, "test-backend-01").
		Return([]haproxy.ServerRequest{
			{Name: "existing-test", Port: 8080},
		}, nil)

	haproxyReader.On("GetBackendServers", mock.Anything, "test-backend-02").
		Return([]haproxy.ServerRequest{
			{Name: "existing-test", Port: 8080},
		}, nil)

	haproxyWriter.On("AddBackendServer", mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	s.cache.Store("test-node-01", NodeInfo{
		Hostname: "test-node-01",
		IP:       "10.0.0.1",
	})

	haproxyReader.On("GetBackendNames", ctx).Return([]string{"test-backend-01", "test-backend-02"}, nil)

	s.SyncNodeToBackends(ctx, haproxyReader, haproxyWriter)
	haproxyReader.AssertCalled(t, "GetBackendServers", mock.Anything, "test-backend-01")
	haproxyReader.AssertCalled(t, "GetBackendServers", mock.Anything, "test-backend-02")
	haproxyWriter.AssertNumberOfCalls(t, "AddBackendServer", 2)
}

func TestSyncNodeToBackends_ListBackendsError(t *testing.T) {
	ctx := context.Background()
	s := &Store{}
	haproxyReader := &MockHAProxyAPI{}
	haproxyWriter := &MockHAProxyAPI{}

	haproxyReader.On("GetBackendServers", mock.Anything, "test-backend-01").
		Return([]haproxy.ServerRequest{
			{Name: "existing-test", Port: 8080},
		}, nil)

	haproxyReader.On("GetBackendNames", ctx).Return([]string{}, fmt.Errorf("failed to  get backends"))
	s.SyncNodeToBackends(ctx, haproxyReader, haproxyWriter)

	haproxyWriter.AssertNotCalled(t, "AddBackendServer", mock.Anything, mock.Anything, mock.Anything)
}

func Test_syncNodesToBackend(t *testing.T) {
	ctx := context.Background()

	s := &Store{}
	s.log = slog.New(slog.NewTextHandler(os.Stdout, nil))

	haproxyWriter := &MockHAProxyAPI{}

	expectedServers, _, serversMap := mockServersData()
	for _, server := range expectedServers {
		s.cache.Store(server.Name, NodeInfo{
			Hostname: server.Name,
			IP:       server.Address,
		})
		haproxyWriter.AddServer(ctx, "test-backend-01", server)
	}
	s.syncNodesToBackend(ctx, haproxyWriter, "test-backend-01", expectedServers[0].Port, serversMap)
	_, ok1 := s.cache.Load("test-node-01")
	_, ok2 := s.cache.Load("test-node-02")
	require.True(t, ok1)
	require.True(t, ok2)
	require.Len(t, haproxyWriter.AddedServers, len(expectedServers))
	for i, server := range expectedServers {
		require.Equal(t, server, haproxyWriter.AddedServers[i])
	}
}

func Test_nodeAlreadyExists(t *testing.T) {
	s := &Store{}

	_, existingServers, _ := mockServersData()
	for _, server := range existingServers {
		s.cache.Store(server.Name, NodeInfo{
			Hostname: server.Name,
			IP:       server.Address,
		})
	}
	port, existingMap, _ := s.prepareBackendState(existingServers, "test-backend-01")
	s.cache.Range(func(_, value any) bool {
		node, ok := value.(NodeInfo)
		if !ok {
			return true
		}
		ok = s.nodeAlreadyExists(node, existingMap, port)
		require.True(t, ok)
		return true
	})
}

func Test_GetNodeInfo(t *testing.T) {
	s := &Store{}

	existingServers, _, _ := mockServersData()

	for _, server := range existingServers {
		s.cache.Store(server.Name, NodeInfo{
			Hostname: server.Name,
			IP:       server.Address,
		})
	}
	s.GetNodeInfo(func(ni NodeInfo) bool {
		servers := &[]haproxy.ServerRequest{
			{
				Name:    ni.Hostname,
				Address: ni.IP,
			},
		}
		for _, server := range *servers {
			_, ok := s.cache.Load(server.Name)
			require.True(t, ok)
		}
		return true
	})
}
