package nodewatcher

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type handlerCall struct {
    called bool
    hostname string
    ip string
    node *v1.Node
}

var dummyHandler NodeHandlerFunc = func(
    _ context.Context,
    _ string,
    _ string,
    _ *v1.Node,
    _ HAProxyAPI,
) {
    // intentionally empty
}

type handlerSpy struct {
    mu sync.Mutex
    calls int
    last string
}

func (h *handlerSpy) Handler(_ context.Context, _, _ string, node *v1.Node, _ HAProxyAPI) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.calls++
    if node != nil {
        h.last = node.Name
    }
}

func (h *handlerSpy) Calls() int {
    h.mu.Lock()
    defer h.mu.Unlock()
    return h.calls
}

func startNodeInformerForTest(t *testing.T, handler NodeHandlerFunc) (clientset kubernetes.Interface, stopCh chan struct{}) {
    t.Helper()

    clientset = fake.NewClientset()
    stopCh = make(chan struct{})

    createNodeInformer(
        context.Background(),
        clientset,
        stopCh,
        handler,
        &MockHAProxyAPI{},
        )

    return clientset, stopCh
}

func testHandler(call *handlerCall) NodeHandlerFunc {
    return func(ctx context.Context, hostname, ip string, node *v1.Node, haproxyClient HAProxyAPI) {
        call.called = true
        call.hostname = hostname
        call.ip = ip
        call.node = node
    }
}

func TestHandleNode_normalNode(t *testing.T) {
    call := &handlerCall{}

    node := &v1.Node{
        ObjectMeta: metav1.ObjectMeta{
            Name: "worker-1",
        },
        Status: v1.NodeStatus{
            Addresses: []v1.NodeAddress{
                {
                    Type: v1.NodeInternalIP,
                    Address: "10.0.0.1",
                },
            },
        },
    }

    handleNode(context.Background(), node, testHandler(call), nil)
    require.True(t, call.called)
    require.Equal(t, node.ObjectMeta.Name, call.hostname)
    require.Equal(t, "10.0.0.1", call.ip)
    require.Equal(t, node, call.node)
}

func TestHandleNode_NoInternalIP(t *testing.T) {
    call := &handlerCall{}

    node := &v1.Node{
        ObjectMeta: metav1.ObjectMeta{
            Name: "worker-1",
        },
        Status: v1.NodeStatus{
            Addresses: []v1.NodeAddress{
                {
                    Type: v1.NodeHostName,
                    Address: "worker-2.local",
                },
            },
        },
    }

    handleNode(context.Background(), node, testHandler(call), nil)
    require.True(t, call.called)
    require.Equal(t, "", call.ip)
}

func TestStartWatcherWithClient_ReturnStopChannel(t *testing.T) {
    clientset := fake.NewClientset()
    stopCh := StartWatcherWithClient(context.Background(), clientset, dummyHandler, &MockHAProxyAPI{})

    if stopCh == nil {
        t.Fatal("stopCh is nil")
    }
}

func TestCreateNodeInformer_Add(t *testing.T) {
    spy := &handlerSpy{}

    clientset, stopCh := startNodeInformerForTest(t, spy.Handler)
    defer close(stopCh)

    node := &v1.Node{
        ObjectMeta: metav1.ObjectMeta{
            Name: "node-add",
        },
    }

    _, err := clientset.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})
    require.NoError(t, err)

    require.Eventually(t, func() bool {
        return spy.Calls() == 1
    }, time.Second, 10*time.Millisecond)
}

func TestCreateNodeInformer_Update(t *testing.T) {
    ctx := context.Background()
    spy := &handlerSpy{}

    clientset, stopCh := startNodeInformerForTest(t, spy.Handler)
    defer close(stopCh)

    node := &v1.Node{
        ObjectMeta: metav1.ObjectMeta{
            Name: "node-update",
        },
    }

    created, err := clientset.CoreV1().Nodes().Create(ctx, node, metav1.CreateOptions{})
    require.NoError(t, err)

    created.Labels = map[string]string{"updated": "true"}

    _, err = clientset.CoreV1().Nodes().Update(ctx, created, metav1.UpdateOptions{})
    require.NoError(t, err)

    require.Eventually(t, func() bool {
        return spy.Calls() == 2
    }, time.Second, 10*time.Millisecond)
}

func TestCreateNodeInformer_Delete(t *testing.T) {
    ctx := context.Background()
    spy := &handlerSpy{}

    clientset, stopCh := startNodeInformerForTest(t, spy.Handler)
    defer close(stopCh)

    node := &v1.Node{
        ObjectMeta: metav1.ObjectMeta{
            Name: "node-delete",
        },
    }

    _, err := clientset.CoreV1().Nodes().Create(ctx, node, metav1.CreateOptions{})
    require.NoError(t, err)


    err = clientset.CoreV1().Nodes().Delete(ctx, node.Name, metav1.DeleteOptions{})
    require.NoError(t, err)

    require.Eventually(t, func() bool {
        return spy.Calls() == 2
    }, time.Second, 10*time.Millisecond)
}
