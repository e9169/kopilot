package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	errNewProviderFailed    = "NewProvider() failed: %v"
	testContext1            = "context-1"
	testContext2            = "context-2"
	kubeconfigFilePattern   = "kubeconfig-*.yaml"
	testNode1               = "node-1"
	testNode2               = "node-2"
	testNamespaceDefault    = "default"
	testNamespaceKubeSystem = "kube-system"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() (string, func())
		wantErr   bool
		wantCount int
	}{
		{
			name: "valid kubeconfig with single cluster",
			setupFunc: func() (string, func()) {
				return createTempKubeconfig(t, 1)
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "valid kubeconfig with multiple clusters",
			setupFunc: func() (string, func()) {
				return createTempKubeconfig(t, 3)
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name: "non-existent kubeconfig",
			setupFunc: func() (string, func()) {
				return "/non/existent/path", func() {
					// No cleanup needed for non-existent path
				}
			},
			wantErr: true,
		},
		{
			name: "invalid kubeconfig format",
			setupFunc: func() (string, func()) {
				tmpfile, err := os.CreateTemp("", kubeconfigFilePattern)
				if err != nil {
					t.Fatal(err)
				}
				_, _ = tmpfile.WriteString("invalid: yaml: content: [[[")
				_ = tmpfile.Close()
				return tmpfile.Name(), func() { _ = os.Remove(tmpfile.Name()) }
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runNewProviderTest(t, tt.setupFunc, tt.wantErr, tt.wantCount)
		})
	}
}

func runNewProviderTest(t *testing.T, setupFunc func() (string, func()), wantErr bool, wantCount int) {
	t.Helper()
	kubeconfigPath, cleanup := setupFunc()
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)

	if wantErr {
		if err == nil {
			t.Errorf("NewProvider() expected error, got nil")
		}
		return
	}

	if err != nil {
		t.Errorf("NewProvider() unexpected error: %v", err)
		return
	}

	if provider == nil {
		t.Fatal("NewProvider() returned nil provider")
	}

	clusters := provider.GetClusters()
	if len(clusters) != wantCount {
		t.Errorf("GetClusters() got %d clusters, want %d", len(clusters), wantCount)
	}
}

func TestGetClusters(t *testing.T) {
	kubeconfigPath, cleanup := createTempKubeconfig(t, 2)
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf(errNewProviderFailed, err)
	}

	clusters := provider.GetClusters()

	if len(clusters) != 2 {
		t.Errorf("GetClusters() got %d clusters, want 2", len(clusters))
	}

	// Verify cluster structure
	for _, cluster := range clusters {
		if cluster.Context == "" {
			t.Error("Cluster has empty context")
		}
		if cluster.Name == "" {
			t.Error("Cluster has empty name")
		}
		if cluster.Server == "" {
			t.Error("Cluster has empty server")
		}
	}
}

func TestGetClusterByContext(t *testing.T) {
	kubeconfigPath, cleanup := createTempKubeconfig(t, 2)
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf(errNewProviderFailed, err)
	}

	tests := []struct {
		name        string
		contextName string
		wantErr     bool
	}{
		{
			name:        "existing context",
			contextName: testContext1,
			wantErr:     false,
		},
		{
			name:        "non-existent context",
			contextName: "non-existent",
			wantErr:     true,
		},
		{
			name:        "empty context name",
			contextName: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster, err := provider.GetClusterByContext(tt.contextName)

			if tt.wantErr {
				if err == nil {
					t.Error("GetClusterByContext() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetClusterByContext() unexpected error: %v", err)
				return
			}

			if cluster.Context != tt.contextName {
				t.Errorf("GetClusterByContext() got context %s, want %s", cluster.Context, tt.contextName)
			}
		})
	}
}

func TestGetCurrentContext(t *testing.T) {
	kubeconfigPath, cleanup := createTempKubeconfig(t, 2)
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf(errNewProviderFailed, err)
	}

	currentContext := provider.GetCurrentContext()
	if currentContext == "" {
		t.Error("GetCurrentContext() returned empty string")
	}

	if currentContext != testContext1 {
		t.Errorf("GetCurrentContext() got %s, want %s", currentContext, testContext1)
	}
}

func TestSetCurrentContext(t *testing.T) {
	kubeconfigPath, cleanup := createTempKubeconfig(t, 2)
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf(errNewProviderFailed, err)
	}

	if err := provider.SetCurrentContext(testContext2); err != nil {
		t.Fatalf("SetCurrentContext() unexpected error: %v", err)
	}

	if got := provider.GetCurrentContext(); got != testContext2 {
		t.Errorf("GetCurrentContext() got %s, want %s", got, testContext2)
	}

	clusters := provider.GetClusters()
	for _, cluster := range clusters {
		if cluster.Context == testContext2 && !cluster.IsCurrent {
			t.Errorf("SetCurrentContext() did not mark %s as current", testContext2)
		}
		if cluster.Context != testContext2 && cluster.IsCurrent {
			t.Errorf("SetCurrentContext() incorrectly marked %s as current", cluster.Context)
		}
	}

	if err := provider.SetCurrentContext("does-not-exist"); err == nil {
		t.Error("SetCurrentContext() expected error for invalid context, got nil")
	}
}

func TestGetClusterStatusInvalidContext(t *testing.T) {
	kubeconfigPath, cleanup := createTempKubeconfig(t, 1)
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf(errNewProviderFailed, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = provider.GetClusterStatus(ctx, "non-existent-context")
	if err == nil {
		t.Error("GetClusterStatus() expected error for non-existent context, got nil")
	}
}

// createTempKubeconfig creates a temporary kubeconfig file for testing
func createTempKubeconfig(t *testing.T, numClusters int) (string, func()) {
	t.Helper()

	tmpfile, err := os.CreateTemp("", kubeconfigFilePattern)
	if err != nil {
		t.Fatal(err)
	}

	config := clientcmdapi.NewConfig()

	for i := 1; i <= numClusters; i++ {
		clusterName := fmt.Sprintf("cluster-%d", i)
		contextName := fmt.Sprintf("context-%d", i)
		userName := fmt.Sprintf("user-%d", i)

		config.Clusters[clusterName] = &clientcmdapi.Cluster{
			Server:                fmt.Sprintf("https://cluster-%d.example.com", i),
			InsecureSkipTLSVerify: true,
		}

		config.AuthInfos[userName] = &clientcmdapi.AuthInfo{
			Token: "test-token",
		}

		config.Contexts[contextName] = &clientcmdapi.Context{
			Cluster:  clusterName,
			AuthInfo: userName,
		}
	}

	// Set first context as current
	if numClusters > 0 {
		config.CurrentContext = testContext1
	}

	err = clientcmd.WriteToFile(*config, tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		_ = os.Remove(tmpfile.Name())
	}

	return tmpfile.Name(), cleanup
}

func TestProviderConcurrency(t *testing.T) {
	kubeconfigPath, cleanup := createTempKubeconfig(t, 3)
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf(errNewProviderFailed, err)
	}

	// Test concurrent access to GetClusters
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			clusters := provider.GetClusters()
			if len(clusters) != 3 {
				t.Errorf("Concurrent GetClusters() got %d clusters, want 3", len(clusters))
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestKubeconfigInvalidPath(t *testing.T) {
	invalidPaths := []string{
		"",
		"/dev/null/invalid",
		string([]byte{0x00}), // null byte
	}

	for _, path := range invalidPaths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			_, err := NewProvider(path)
			if err == nil {
				t.Errorf("NewProvider(%q) expected error, got nil", path)
			}
		})
	}
}

func TestGetNodeRoles(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   []string
	}{
		{
			name: "control-plane node",
			labels: map[string]string{
				"node-role.kubernetes.io/control-plane": "",
			},
			want: []string{"control-plane"},
		},
		{
			name: "worker node (no role labels)",
			labels: map[string]string{
				"kubernetes.io/hostname": testNode1,
			},
			want: []string{"worker"},
		},
		{
			name: "multiple roles",
			labels: map[string]string{
				"node-role.kubernetes.io/master": "",
				"node-role.kubernetes.io/worker": "",
			},
			want: []string{"master", "worker"},
		},
		{
			name: "empty role label",
			labels: map[string]string{
				"node-role.kubernetes.io/": "",
			},
			want: []string{"worker"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: tt.labels,
				},
			}
			roles := getNodeRoles(node)
			if len(roles) != len(tt.want) {
				t.Errorf("getNodeRoles() got %d roles, want %d", len(roles), len(tt.want))
				return
			}
			// Check all expected roles are present
			roleMap := make(map[string]bool)
			for _, role := range roles {
				roleMap[role] = true
			}
			for _, wantRole := range tt.want {
				if !roleMap[wantRole] {
					t.Errorf("getNodeRoles() missing role %s, got %v", wantRole, roles)
				}
			}
		})
	}
}

func TestGetAllClusterStatuses(t *testing.T) {
	kubeconfigPath, cleanup := createTempKubeconfig(t, 3)
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf(errNewProviderFailed, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	statuses := provider.GetAllClusterStatuses(ctx)

	if len(statuses) != 3 {
		t.Errorf("GetAllClusterStatuses() returned %d statuses, want 3", len(statuses))
	}

	// All should have errors since these are mock clusters
	for i, status := range statuses {
		if status == nil {
			t.Errorf("Status %d is nil", i)
			continue
		}
		if status.Context == "" {
			t.Errorf("Status %d has empty context", i)
		}
	}
}

func TestNewProviderMissingCluster(t *testing.T) {
	tmpfile, err := os.CreateTemp("", kubeconfigFilePattern)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	config := clientcmdapi.NewConfig()

	// Create context without cluster
	config.Contexts["orphan-context"] = &clientcmdapi.Context{
		Cluster:  "missing-cluster",
		AuthInfo: "user-1",
	}
	config.AuthInfos["user-1"] = &clientcmdapi.AuthInfo{
		Token: "test-token",
	}
	config.CurrentContext = "orphan-context"

	err = clientcmd.WriteToFile(*config, tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	provider, err := NewProvider(tmpfile.Name())
	if err != nil {
		t.Fatalf(errNewProviderFailed, err)
	}

	// Should have 0 clusters since the context references a missing cluster
	clusters := provider.GetClusters()
	if len(clusters) != 0 {
		t.Errorf("Expected 0 clusters for orphan context, got %d", len(clusters))
	}
}
func TestCollectNodeInfo(t *testing.T) {
	ctx := context.Background()

	// Create a fake clientset with test nodes
	clientset := fake.NewClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNode1,
				Labels: map[string]string{
					"node-role.kubernetes.io/control-plane": "",
				},
				CreationTimestamp: metav1.NewTime(time.Now().Add(-24 * time.Hour)),
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:              testNode2,
				CreationTimestamp: metav1.NewTime(time.Now().Add(-48 * time.Hour)),
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		},
	)

	nodeInfos, healthyCount, err := collectNodeInfo(ctx, clientset)
	if err != nil {
		t.Fatalf("collectNodeInfo() error = %v", err)
	}

	if len(nodeInfos) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodeInfos))
	}

	if healthyCount != 1 {
		t.Errorf("Expected 1 healthy node, got %d", healthyCount)
	}

	// Check first node
	if nodeInfos[0].Name != testNode1 {
		t.Errorf("Expected node name %s, got %s", testNode1, nodeInfos[0].Name)
	}
	if nodeInfos[0].Status != "Ready" {
		t.Errorf("Expected node status Ready, got %s", nodeInfos[0].Status)
	}
	if len(nodeInfos[0].Roles) == 0 {
		t.Error("Expected node to have roles")
	}

	// Check second node
	if nodeInfos[1].Status != "NotReady" {
		t.Errorf("Expected node status NotReady, got %s", nodeInfos[1].Status)
	}
}

func TestCollectNamespaceList(t *testing.T) {
	ctx := context.Background()

	clientset := fake.NewClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespaceKubeSystem,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-public",
			},
		},
	)

	namespaces, err := collectNamespaceList(ctx, clientset)
	if err != nil {
		t.Fatalf("collectNamespaceList() error = %v", err)
	}

	if len(namespaces) != 3 {
		t.Errorf("Expected 3 namespaces, got %d", len(namespaces))
	}

	expectedNamespaces := map[string]bool{
		testNamespaceDefault:    true,
		testNamespaceKubeSystem: true,
		"kube-public":           true,
	}

	for _, ns := range namespaces {
		if !expectedNamespaces[ns] {
			t.Errorf("Unexpected namespace: %s", ns)
		}
	}
}

func TestCollectPodHealth(t *testing.T) {
	ctx := context.Background()

	clientset := fake.NewClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "healthy-pod",
				Namespace: testNamespaceDefault,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  "container-1",
						Ready: true,
					},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "unhealthy-pod",
				Namespace: testNamespaceDefault,
			},
			Status: corev1.PodStatus{
				Phase:  corev1.PodPending,
				Reason: "ContainerCreating",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failed-pod",
				Namespace: testNamespaceKubeSystem,
			},
			Status: corev1.PodStatus{
				Phase:  corev1.PodFailed,
				Reason: "Error",
			},
		},
	)

	totalPods, healthyPods, unhealthyPods, err := collectPodHealth(ctx, clientset)
	if err != nil {
		t.Fatalf("collectPodHealth() error = %v", err)
	}

	if totalPods != 3 {
		t.Errorf("Expected 3 total pods, got %d", totalPods)
	}

	if healthyPods != 1 {
		t.Errorf("Expected 1 healthy pod, got %d", healthyPods)
	}

	if len(unhealthyPods) != 2 {
		t.Errorf("Expected 2 unhealthy pods, got %d", len(unhealthyPods))
	}
}
