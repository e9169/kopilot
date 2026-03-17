package k8s

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestGetClusterVersionWithContext tests cluster version retrieval with context
func TestGetClusterVersionWithContext(t *testing.T) {
	clientset := fake.NewClientset()

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
	clientset := fake.NewClientset()

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

	clientset := fake.NewClientset(nodes)
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

	clientset := fake.NewClientset(namespaces)
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

	clientset := fake.NewClientset(pods)
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
			true,
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

	clientset := fake.NewClientset(nodes)
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

	clientset := fake.NewClientset(pods)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _ = collectPodHealth(ctx, clientset)
	}
}

// --- Sanitize tests ---

// TestScoreFromFindings tests the penalty-based scoring function
func TestScoreFromFindings(t *testing.T) {
	tests := []struct {
		name     string
		findings []SanitizeFinding
		want     int
	}{
		{"no findings", nil, 100},
		{"one critical (−10)", []SanitizeFinding{{Penalty: sanitizePenaltyCritical}}, 90},
		{"one major (−5)", []SanitizeFinding{{Penalty: sanitizePenaltyMajor}}, 95},
		{"one minor (−2)", []SanitizeFinding{{Penalty: sanitizePenaltyMinor}}, 98},
		{"many findings floor at 0", func() []SanitizeFinding {
			ff := make([]SanitizeFinding, 20)
			for i := range ff {
				ff[i] = SanitizeFinding{Penalty: sanitizePenaltyCritical}
			}
			return ff
		}(), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scoreFromFindings(tt.findings); got != tt.want {
				t.Errorf("scoreFromFindings() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestGradeFromScore tests grade letter assignment
func TestGradeFromScore(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{100, "A"}, {90, "A"}, {89, "B"}, {75, "B"},
		{74, "C"}, {60, "C"}, {59, "D"}, {40, "D"},
		{39, "F"}, {0, "F"},
	}
	for _, tt := range tests {
		if got := gradeFromScore(tt.score); got != tt.want {
			t.Errorf("gradeFromScore(%d) = %s, want %s", tt.score, got, tt.want)
		}
	}
}

// makeCompliantDeployment builds a Deployment that passes all sanitize rules
func makeCompliantDeployment(ns, name string, replicas int32) *appsv1.Deployment {
	trueVal := true
	falseVal := false
	rootUser := int64(1000)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    map[string]string{"app": name},
			Annotations: map[string]string{
				"app.kubernetes.io/version": "1.0.0",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "nginx:1.25.0",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
							LivenessProbe:  &corev1.Probe{},
							ReadinessProbe: &corev1.Probe{},
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &falseVal,
								AllowPrivilegeEscalation: &falseVal,
								RunAsNonRoot:             &trueVal,
								RunAsUser:                &rootUser,
								ReadOnlyRootFilesystem:   &trueVal,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
						},
					},
				},
			},
		},
	}
}

// TestCollectSanitizeFindingsCleanWorkload verifies that a fully compliant deployment produces no findings
func TestCollectSanitizeFindingsCleanWorkload(t *testing.T) {
	deploy := makeCompliantDeployment("default", "my-app", 2)
	clientset := fake.NewClientset(deploy)
	ctx := context.Background()

	findings, allWorkloads, err := collectSanitizeFindings(ctx, clientset, "", false)
	if err != nil {
		t.Fatalf("collectSanitizeFindings() returned error: %v", err)
	}

	if len(allWorkloads) != 1 {
		t.Errorf("total workloads = %d, want 1", len(allWorkloads))
	}
	if len(findings) != 0 {
		t.Errorf("got %d findings on compliant deployment, want 0:\n%v", len(findings), findings)
	}
}

// TestCollectSanitizeFindingsViolations verifies that a non-compliant deployment triggers expected rules
func TestCollectSanitizeFindingsViolations(t *testing.T) {
	replicas := int32(1) // BP-006
	privileged := true   // CKS-001
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bad-app",
			Namespace: "production",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "bad-app"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}},
				Spec: corev1.PodSpec{
					HostNetwork: true, // CKS-004
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "nginx:latest", // BP-005
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged, // CKS-001
							},
							// No probes, no resources — BP-001, BP-002, BP-003, BP-004
						},
					},
				},
			},
		},
	}

	clientset := fake.NewClientset(deploy)
	ctx := context.Background()

	findings, allWorkloads, err := collectSanitizeFindings(ctx, clientset, "", false)
	if err != nil {
		t.Fatalf("collectSanitizeFindings() returned error: %v", err)
	}

	if len(allWorkloads) != 1 {
		t.Errorf("total workloads = %d, want 1", len(allWorkloads))
	}

	ruleIDs := make(map[string]bool)
	for _, f := range findings {
		ruleIDs[f.RuleID] = true
	}

	expectedRules := []string{"CKS-001", "CKS-002", "CKS-003", "CKS-004", "BP-001", "BP-002", "BP-003", "BP-004", "BP-005", "BP-006", "BP-007", "BP-008", "BP-009", "BP-010"}
	for _, rule := range expectedRules {
		if !ruleIDs[rule] {
			t.Errorf("expected rule %s to be triggered, but it was not. Got rules: %v", rule, ruleIDs)
		}
	}
}

// TestCollectSanitizeFindingsSystemNamespaceExclusion verifies system namespaces are skipped by default
func TestCollectSanitizeFindingsSystemNamespaceExclusion(t *testing.T) {
	replicas := int32(1)
	sysWorkload := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "coredns", Namespace: "kube-system"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "coredns"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "coredns", Image: "coredns:latest"}}},
			},
		},
	}

	clientset := fake.NewClientset(sysWorkload)
	ctx := context.Background()

	// Default: system namespaces excluded
	_, excluded, err := collectSanitizeFindings(ctx, clientset, "", false)
	if err != nil {
		t.Fatalf("collectSanitizeFindings() returned error: %v", err)
	}
	if len(excluded) != 0 {
		t.Errorf("expected 0 workloads scanned (kube-system excluded), got %d", len(excluded))
	}

	// With includeSystem=true, kube-system should be scanned
	_, included, err := collectSanitizeFindings(ctx, clientset, "", true)
	if err != nil {
		t.Fatalf("collectSanitizeFindings(includeSystem=true) returned error: %v", err)
	}
	if len(included) != 1 {
		t.Errorf("expected 1 workload scanned with includeSystem=true, got %d", len(included))
	}
}

// TestCollectSanitizeFindingsNamespaceFilter verifies namespace filtering works correctly
func TestCollectSanitizeFindingsNamespaceFilter(t *testing.T) {
	replicas := int32(2)
	makeSimpleDeploy := func(ns, name string) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx:1.25.0"}}},
				},
			},
		}
	}

	clientset := fake.NewClientset(makeSimpleDeploy("staging", "app-a"), makeSimpleDeploy("production", "app-b"))
	ctx := context.Background()

	_, filtered, err := collectSanitizeFindings(ctx, clientset, "staging", false)
	if err != nil {
		t.Fatalf("collectSanitizeFindings() returned error: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("namespace filter: expected 1 workload, got %d", len(filtered))
	}
}

// TestBuildSanitizeResult verifies scoring and grouping logic
func TestBuildSanitizeResult(t *testing.T) {
	findings := []SanitizeFinding{
		{RuleID: "CKS-001", Severity: SanitizeCritical, Workload: "ns-a/Deployment/app", Container: "app", Penalty: sanitizePenaltyCritical},
		{RuleID: "BP-001", Severity: SanitizeMajor, Workload: "ns-a/Deployment/app", Container: "app", Penalty: sanitizePenaltyMajor},
		{RuleID: "BP-009", Severity: SanitizeMinor, Workload: "ns-b/Deployment/svc", Penalty: sanitizePenaltyMinor},
	}

	result := buildSanitizeResult("test-context", findings, []string{"ns-a/Deployment/app", "ns-b/Deployment/svc"})

	if result.Context != "test-context" {
		t.Errorf("Context = %q, want %q", result.Context, "test-context")
	}
	if result.TotalWorkloads != 2 {
		t.Errorf("TotalWorkloads = %d, want 2", result.TotalWorkloads)
	}
	if result.TotalFindings != 3 {
		t.Errorf("TotalFindings = %d, want 3", result.TotalFindings)
	}
	if result.CriticalCount != 1 {
		t.Errorf("CriticalCount = %d, want 1", result.CriticalCount)
	}
	if result.MajorCount != 1 {
		t.Errorf("MajorCount = %d, want 1", result.MajorCount)
	}
	if result.MinorCount != 1 {
		t.Errorf("MinorCount = %d, want 1", result.MinorCount)
	}
	if len(result.Namespaces) != 2 {
		t.Errorf("Namespaces count = %d, want 2", len(result.Namespaces))
	}

	// Per-workload: ns-a/Deployment/app penalty=15 → score 85; ns-b/Deployment/svc penalty=2 → score 98
	// Cluster average: (85+98)/2 = 91, grade A
	if result.Score != 91 {
		t.Errorf("cluster Score = %d, want 91", result.Score)
	}
	if result.Grade != "A" {
		t.Errorf("cluster Grade = %q, want A", result.Grade)
	}
}
