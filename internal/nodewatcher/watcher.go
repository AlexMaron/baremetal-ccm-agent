package nodewatcher

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func StartWatcherWithClient(ctx context.Context, clientset kubernetes.Interface, handler NodeHandlerFunc, haproxyClient HAProxyAPI) <-chan struct{} {
    stopCh := setupSignalHandler()
    createNodeInformer(ctx, clientset, stopCh, handler, haproxyClient)
    return stopCh
}

func StartWatcher(ctx context.Context, kubeconfigPath string, handler NodeHandlerFunc, haproxyClient HAProxyAPI) <-chan struct{} {
	clientset := createClientSet(kubeconfigPath)
	return StartWatcherWithClient(ctx, clientset, handler, haproxyClient)
}

func createClientSet(kubeconfigPath string) kubernetes.Interface {
    var (
        restConfig *rest.Config
        err error
    )

    if kubeconfigPath != "" {
        restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
    } else {
        restConfig, err = rest.InClusterConfig()
    }

	if err != nil {
		klog.Fatalf("failed to get rest config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		klog.Fatalf("failed to create clientset: %v", err)
	}

	return clientset
}

func handleNode(ctx context.Context, obj any, handler NodeHandlerFunc, haproxyClient HAProxyAPI) {
    node, ok := obj.(*v1.Node)
    if !ok {
        return
    }

    if _, isControlPlane := node.Labels["node-role.kubernetes.io/control-plane"]; isControlPlane {
        return
    }

    ip := ""
    for _, addr := range node.Status.Addresses {
        if addr.Type == v1.NodeInternalIP {
            ip = addr.Address
            break
        }
    }
	handler(ctx, node.Name, ip, node, haproxyClient)
}

func createNodeInformer(ctx context.Context, clientset kubernetes.Interface, stopCh <-chan struct{}, handler NodeHandlerFunc, haproxyClient HAProxyAPI) cache.SharedIndexInformer {
	informerFactory := informers.NewSharedInformerFactory(clientset, 30*time.Second)
	nodeInformer := informerFactory.Core().V1().Nodes().Informer()

	nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
            handleNode(ctx, obj, handler, haproxyClient)
			logNode("ADD", obj)
		},
		UpdateFunc: func(oldObj, newObj any) {
            handleNode(ctx, newObj, handler, haproxyClient)
			logNodeUpdate(oldObj, newObj)
		},
		DeleteFunc: func(obj any) {
            handleNode(ctx, obj, handler, haproxyClient)
			logNode("DELETE", obj)
		},
	})

	informerFactory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, nodeInformer.HasSynced) {
		klog.Fatalf("failed to sync node informer")
	}
	return nodeInformer
}

func logNode(event string, obj any) {
	node, ok := obj.(*v1.Node)
	if !ok {
		klog.Error("object is not a Node")
		return
	}

	var readyStatus v1.ConditionStatus
	for _, cond := range node.Status.Conditions {
		if cond.Type == v1.NodeReady {
			readyStatus = cond.Status
			break
		}
	}

	var internalIP string
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			internalIP = addr.Address
		}
	}

	klog.InfoS(
		"Node event",
		"event", event,
		"name", node.Name,
		"ready", readyStatus,
		"unschedulable", node.Spec.Unschedulable,
		"InternalIP", internalIP,
		"labels", node.Labels,
	)
}

func logNodeUpdate(oldObj, newObj any) {
	oldNode, ok1 := oldObj.(*v1.Node)
	newNode, ok2 := newObj.(*v1.Node)
	if !ok1 || !ok2 {
		klog.Error("object is not a Node")
		return
	}

	if oldNode.ResourceVersion == newNode.ResourceVersion {
		return
	}

	logNode("UPDATE", newNode)
}

func setupSignalHandler() <-chan struct{} {
	stopCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		close(stopCh)
	}()

	return stopCh
}
