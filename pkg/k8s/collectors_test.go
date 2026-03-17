package k8s

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testNsKubeSystem           = "kube-system"
	testWorkloadNsADeployApp   = "ns-a/Deployment/app"
	testBuildSanitizeContext   = "test-context"
	testIngressHost            = "myapp.example.com"
	testLBSourceRange          = "10.0.0.0/8" // NOSONAR - test data for LoadBalancer source range validation
	errCollectSanitizeFindings = "collectSanitizeFindings() returned error: %v"
	errCollectNetworkResources = "collectNetworkResources() error: %v"
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
			{ObjectMeta: metav1.ObjectMeta{Name: testNsKubeSystem}},
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
		t.Fatalf(errCollectSanitizeFindings, err)
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
		t.Fatalf(errCollectSanitizeFindings, err)
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
		ObjectMeta: metav1.ObjectMeta{Name: "coredns", Namespace: testNsKubeSystem},
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
		t.Fatalf(errCollectSanitizeFindings, err)
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
		t.Fatalf(errCollectSanitizeFindings, err)
	}
	if len(filtered) != 1 {
		t.Errorf("namespace filter: expected 1 workload, got %d", len(filtered))
	}
}

// TestBuildSanitizeResult verifies scoring and grouping logic
func TestBuildSanitizeResult(t *testing.T) {
	findings := []SanitizeFinding{
		{RuleID: "CKS-001", Severity: SanitizeCritical, Workload: testWorkloadNsADeployApp, Container: "app", Penalty: sanitizePenaltyCritical},
		{RuleID: "BP-001", Severity: SanitizeMajor, Workload: testWorkloadNsADeployApp, Container: "app", Penalty: sanitizePenaltyMajor},
		{RuleID: "BP-009", Severity: SanitizeMinor, Workload: "ns-b/Deployment/svc", Penalty: sanitizePenaltyMinor},
	}

	result := buildSanitizeResult(testBuildSanitizeContext, findings, []string{testWorkloadNsADeployApp, "ns-b/Deployment/svc"})

	if result.Context != testBuildSanitizeContext {
		t.Errorf("Context = %q, want %q", result.Context, testBuildSanitizeContext)
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

// ── checkServiceRules ─────────────────────────────────────────────────────────

// TestCheckServiceRulesNodePort verifies SVC-001 fires for NodePort services.
func TestCheckServiceRulesNodePort(t *testing.T) {
	spec := corev1.ServiceSpec{Type: corev1.ServiceTypeNodePort}
	var findings []SanitizeFinding
	checkServiceRules(spec, "default/Service/my-svc", &findings)

	if len(findings) == 0 {
		t.Fatal("expected at least one finding for NodePort service, got none")
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "SVC-001" {
			found = true
		}
	}
	if !found {
		t.Errorf("SVC-001 not found in findings: %v", findings)
	}
}

// TestCheckServiceRulesLoadBalancerNoSourceRanges verifies SVC-002 fires.
func TestCheckServiceRulesLoadBalancerNoSourceRanges(t *testing.T) {
	spec := corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer}
	var findings []SanitizeFinding
	checkServiceRules(spec, "default/Service/lb-svc", &findings)

	found := false
	for _, f := range findings {
		if f.RuleID == "SVC-002" {
			found = true
		}
	}
	if !found {
		t.Errorf("SVC-002 not found in findings: %v", findings)
	}
}

// TestCheckServiceRulesLoadBalancerWithSourceRanges verifies SVC-002 does NOT fire.
func TestCheckServiceRulesLoadBalancerWithSourceRanges(t *testing.T) {
	spec := corev1.ServiceSpec{
		Type:                     corev1.ServiceTypeLoadBalancer,
		LoadBalancerSourceRanges: []string{testLBSourceRange},
		Selector:                 map[string]string{"app": "myapp"},
	}
	var findings []SanitizeFinding
	checkServiceRules(spec, "default/Service/lb-restricted", &findings)

	for _, f := range findings {
		if f.RuleID == "SVC-002" {
			t.Errorf("SVC-002 should not fire when loadBalancerSourceRanges is set")
		}
	}
}

// TestCheckServiceRulesNoSelector verifies SVC-003 fires for selector-less ClusterIP services.
func TestCheckServiceRulesNoSelector(t *testing.T) {
	spec := corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP}
	var findings []SanitizeFinding
	checkServiceRules(spec, "default/Service/headless", &findings)

	found := false
	for _, f := range findings {
		if f.RuleID == "SVC-003" {
			found = true
		}
	}
	if !found {
		t.Errorf("SVC-003 not found in findings: %v", findings)
	}
}

// TestCheckServiceRulesExternalNameNoSVC003 verifies ExternalName services skip SVC-003.
func TestCheckServiceRulesExternalNameNoSVC003(t *testing.T) {
	spec := corev1.ServiceSpec{Type: corev1.ServiceTypeExternalName}
	var findings []SanitizeFinding
	checkServiceRules(spec, "default/Service/ext", &findings)

	for _, f := range findings {
		if f.RuleID == "SVC-003" {
			t.Error("SVC-003 should not fire for ExternalName services")
		}
	}
}

// TestCheckServiceRulesCleanService verifies no findings for a well-configured ClusterIP.
func TestCheckServiceRulesCleanService(t *testing.T) {
	spec := corev1.ServiceSpec{
		Type:     corev1.ServiceTypeClusterIP,
		Selector: map[string]string{"app": "myapp"},
	}
	var findings []SanitizeFinding
	checkServiceRules(spec, "default/Service/clean", &findings)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for clean service, got %d: %v", len(findings), findings)
	}
}

// ── checkIngressRules ─────────────────────────────────────────────────────────

// TestCheckIngressRulesNoTLS verifies ING-001 fires when TLS is absent.
func TestCheckIngressRulesNoTLS(t *testing.T) {
	spec := networkingv1.IngressSpec{
		Rules: []networkingv1.IngressRule{
			{Host: testIngressHost},
		},
	}
	var findings []SanitizeFinding
	checkIngressRules(spec, "default/Ingress/my-ing", &findings)

	found := false
	for _, f := range findings {
		if f.RuleID == "ING-001" {
			found = true
		}
	}
	if !found {
		t.Errorf("ING-001 not found for Ingress without TLS: %v", findings)
	}
}

// TestCheckIngressRulesWildcardHost verifies ING-002 fires for wildcard/empty hosts.
func TestCheckIngressRulesWildcardHost(t *testing.T) {
	spec := networkingv1.IngressSpec{
		TLS: []networkingv1.IngressTLS{{Hosts: []string{"*.example.com"}}},
		Rules: []networkingv1.IngressRule{
			{Host: "*.example.com"},
		},
	}
	var findings []SanitizeFinding
	checkIngressRules(spec, "default/Ingress/wildcard-ing", &findings)

	found := false
	for _, f := range findings {
		if f.RuleID == "ING-002" {
			found = true
		}
	}
	if !found {
		t.Errorf("ING-002 not found for wildcard host Ingress: %v", findings)
	}
}

// TestCheckIngressRulesEmptyHost verifies ING-002 fires for empty host entries.
func TestCheckIngressRulesEmptyHost(t *testing.T) {
	spec := networkingv1.IngressSpec{
		TLS:   []networkingv1.IngressTLS{{Hosts: []string{testIngressHost}}},
		Rules: []networkingv1.IngressRule{{Host: ""}},
	}
	var findings []SanitizeFinding
	checkIngressRules(spec, "default/Ingress/empty-host-ing", &findings)

	found := false
	for _, f := range findings {
		if f.RuleID == "ING-002" {
			found = true
		}
	}
	if !found {
		t.Errorf("ING-002 not found for Ingress with empty host: %v", findings)
	}
}

// TestCheckIngressRulesClean verifies no findings for a properly configured Ingress.
func TestCheckIngressRulesClean(t *testing.T) {
	spec := networkingv1.IngressSpec{
		TLS: []networkingv1.IngressTLS{
			{Hosts: []string{testIngressHost}, SecretName: "tls-secret"},
		},
		Rules: []networkingv1.IngressRule{
			{Host: testIngressHost},
		},
	}
	var findings []SanitizeFinding
	checkIngressRules(spec, "default/Ingress/clean-ing", &findings)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for clean Ingress, got %d: %v", len(findings), findings)
	}
}

// ── collectNetworkResources ───────────────────────────────────────────────────

// TestCollectNetworkResourcesWithService verifies Services are scanned and findings reported.
func TestCollectNetworkResourcesWithService(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "my-svc", Namespace: "production"},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort, // triggers SVC-001
		},
	}
	clientset := fake.NewClientset(svc)
	ctx := context.Background()

	var allFindings []SanitizeFinding
	var allWorkloads []string
	err := collectNetworkResources(ctx, clientset, "", false, &allFindings, &allWorkloads)
	if err != nil {
		t.Fatalf(errCollectNetworkResources, err)
	}

	if len(allWorkloads) != 1 {
		t.Errorf("got %d workloads, want 1", len(allWorkloads))
	}
	found := false
	for _, f := range allFindings {
		if f.RuleID == "SVC-001" {
			found = true
		}
	}
	if !found {
		t.Errorf("SVC-001 not found in network resource findings: %v", allFindings)
	}
}

// TestCollectNetworkResourcesSkipsKubernetesService verifies the built-in kubernetes Service is skipped.
func TestCollectNetworkResourcesSkipsKubernetesService(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "kubernetes", Namespace: "default"},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP},
	}
	clientset := fake.NewClientset(svc)
	ctx := context.Background()

	var allFindings []SanitizeFinding
	var allWorkloads []string
	if err := collectNetworkResources(ctx, clientset, "", false, &allFindings, &allWorkloads); err != nil {
		t.Fatalf(errCollectNetworkResources, err)
	}

	if len(allWorkloads) != 0 {
		t.Errorf("kubernetes/default Service should be skipped but got %d workloads", len(allWorkloads))
	}
}

// TestCollectNetworkResourcesWithIngress verifies Ingresses are scanned.
func TestCollectNetworkResourcesWithIngress(t *testing.T) {
	pathType := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "my-ing", Namespace: "production"},
		Spec: networkingv1.IngressSpec{
			// No TLS → ING-001
			Rules: []networkingv1.IngressRule{
				{
					Host: testIngressHost,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{Path: "/", PathType: &pathType, Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "my-svc",
										Port: networkingv1.ServiceBackendPort{Number: 80},
									},
								}},
							},
						},
					},
				},
			},
		},
	}
	clientset := fake.NewClientset(ing)
	ctx := context.Background()

	var allFindings []SanitizeFinding
	var allWorkloads []string
	if err := collectNetworkResources(ctx, clientset, "", false, &allFindings, &allWorkloads); err != nil {
		t.Fatalf(errCollectNetworkResources, err)
	}

	if len(allWorkloads) != 1 {
		t.Errorf("got %d workloads, want 1", len(allWorkloads))
	}
	found := false
	for _, f := range allFindings {
		if f.RuleID == "ING-001" {
			found = true
		}
	}
	if !found {
		t.Errorf("ING-001 not found in ingress findings: %v", allFindings)
	}
}

// TestCollectNetworkResourcesSystemNamespaceExclusion verifies system namespace filtering applies.
func TestCollectNetworkResourcesSystemNamespaceExclusion(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "coredns", Namespace: testNsKubeSystem},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, Selector: map[string]string{"app": "coredns"}},
	}
	clientset := fake.NewClientset(svc)
	ctx := context.Background()

	var allFindings []SanitizeFinding
	var allWorkloads []string
	if err := collectNetworkResources(ctx, clientset, "", false, &allFindings, &allWorkloads); err != nil {
		t.Fatalf(errCollectNetworkResources, err)
	}

	if len(allWorkloads) != 0 {
		t.Errorf("kube-system service should be excluded, got %d workloads", len(allWorkloads))
	}
}

// TestCollectNetworkResourcesNamespaceFilter verifies targetNamespace scoping works.
func TestCollectNetworkResourcesNamespaceFilter(t *testing.T) {
	svc1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "svc-prod", Namespace: "production"},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, Selector: map[string]string{"app": "x"}},
	}
	svc2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "svc-staging", Namespace: "staging"},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, Selector: map[string]string{"app": "y"}},
	}
	clientset := fake.NewClientset(svc1, svc2)
	ctx := context.Background()

	var allFindings []SanitizeFinding
	var allWorkloads []string
	if err := collectNetworkResources(ctx, clientset, "production", false, &allFindings, &allWorkloads); err != nil {
		t.Fatalf(errCollectNetworkResources, err)
	}

	if len(allWorkloads) != 1 {
		t.Errorf("got %d workloads with targetNamespace=production, want 1", len(allWorkloads))
	}
}
