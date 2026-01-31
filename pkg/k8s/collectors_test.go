package k8s

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestGetClusterVersionWithContext tests cluster version retrieval with context
func TestGetClusterVersionWithContext(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	version, err := getClusterVersion(ctx, clientset)
	if err != nil {
		t.Fatalf("getClusterVersion() failed: %v", err)
	}

	// Fake clientset returns a default version
	if version == "" {
		t.Error("Expected non-empty version")
	}
}

// TestGetClusterVersionWithTimeout tests context timeout handling
func TestGetClusterVersionWithTimeout(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait a moment to ensure timeout
	time.Sleep(10 * time.Millisecond)

	_, err := getClusterVersion(ctx, clientset)
	// With fake clientset, this might not timeout, but it shouldn't panic
	if err != nil {
		t.Logf("Expected timeout or success, got error: %v", err)
	}
}

// TestCollectNodeInfoWithContext tests node collection with context
func TestCollectNodeInfoWithContext(t *testing.T) {
	// Create fake clientset with test nodes
	nodes := &corev1.NodeList{
		Items: []corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "node-2"},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(nodes)
	ctx := context.Background()

	nodeList, readyCount, err := collectNodeInfo(ctx, clientset)
	if err != nil {
		t.Fatalf("collectNodeInfo() failed: %v", err)
	}

	if len(nodeList) != 2 {
		t.Errorf("Got %d nodes, want 2", len(nodeList))
	}

	if readyCount != 1 {
		t.Errorf("ReadyNodes = %d, want 1", readyCount)
	}
}

// TestCollectNamespaceListWithContext tests namespace collection with context
func TestCollectNamespaceListWithContext(t *testing.T) {
	namespaces := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "my-app"}},
		},
	}

	clientset := fake.NewSimpleClientset(namespaces)
	ctx := context.Background()

	nsList, err := collectNamespaceList(ctx, clientset)
	if err != nil {
		t.Fatalf("collectNamespaceList() failed: %v", err)
	}

	if len(nsList) != 3 {
		t.Errorf("Got %d namespaces, want 3", len(nsList))
	}
}

// TestCollectPodHealthWithContext tests pod health collection with context
func TestCollectPodHealthWithContext(t *testing.T) {
	pods := &corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "healthy-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{Ready: true},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pending-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failed-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase:  corev1.PodFailed,
					Reason: "Error",
					ContainerStatuses: []corev1.ContainerStatus{
						{RestartCount: 5},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(pods)
	ctx := context.Background()

	totalPods, healthyPods, unhealthyPods, err := collectPodHealth(ctx, clientset)
	if err != nil {
		t.Fatalf("collectPodHealth() failed: %v", err)
	}

	if totalPods != 3 {
		t.Errorf("TotalPods = %d, want 3", totalPods)
	}

	if healthyPods != 1 {
		t.Errorf("HealthyPods = %d, want 1", healthyPods)
	}

	if len(unhealthyPods) != 2 {
		t.Errorf("Got %d unhealthy pods, want 2", len(unhealthyPods))
	}
}

// TestContextTimeoutConstants tests that timeout constants are reasonable
func TestContextTimeoutConstants(t *testing.T) {
	if DefaultAPITimeout < 1*time.Second {
		t.Errorf("DefaultAPITimeout = %v, should be at least 1 second", DefaultAPITimeout)
	}

	if DiscoveryTimeout < 1*time.Second {
		t.Errorf("DiscoveryTimeout = %v, should be at least 1 second", DiscoveryTimeout)
	}

	if DefaultAPITimeout > 2*time.Minute {
		t.Errorf("DefaultAPITimeout = %v, should not exceed 2 minutes", DefaultAPITimeout)
	}
}

// TestIsPodHealthy tests pod health determination
func TestIsPodHealthy(t *testing.T) {
	tests := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{
			"running with ready containers",
			&corev1.Pod{
				Status: corev1.PodStatus{
					Phase:             corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{{Ready: true}},
				},
			},
			true,
		},
		{
			"running with not ready containers",
			&corev1.Pod{
				Status: corev1.PodStatus{
					Phase:             corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{{Ready: false}},
				},
			},
			false,
		},
		{
			"pending pod",
			&corev1.Pod{
				Status: corev1.PodStatus{Phase: corev1.PodPending},
			},
			false,
		},
		{
			"failed pod",
			&corev1.Pod{
				Status: corev1.PodStatus{Phase: corev1.PodFailed},
			},
			false,
		},
		{
			"succeeded pod",
			&corev1.Pod{
				Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPodHealthy(tt.pod)
			if got != tt.want {
				t.Errorf("isPodHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

// BenchmarkCollectNodeInfo benchmarks node collection
func BenchmarkCollectNodeInfo(b *testing.B) {
	nodes := &corev1.NodeList{
		Items: make([]corev1.Node, 100),
	}
	for i := range nodes.Items {
		nodes.Items[i].Status.Conditions = []corev1.NodeCondition{
			{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
		}
	}

	clientset := fake.NewSimpleClientset(nodes)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = collectNodeInfo(ctx, clientset)
	}
}

// BenchmarkCollectPodHealth benchmarks pod health collection
func BenchmarkCollectPodHealth(b *testing.B) {
	pods := &corev1.PodList{
		Items: make([]corev1.Pod, 100),
	}
	for i := range pods.Items {
		pods.Items[i].Status.Phase = corev1.PodRunning
		pods.Items[i].Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: true}}
	}

	clientset := fake.NewSimpleClientset(pods)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _ = collectPodHealth(ctx, clientset)
	}
}
