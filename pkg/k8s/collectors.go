// Package k8s provides Kubernetes cluster interaction capabilities.
// This file contains helper functions for collecting cluster health data.
package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// DefaultAPITimeout is the default timeout for Kubernetes API calls
	DefaultAPITimeout = 30 * time.Second
	// DiscoveryTimeout is the timeout for discovery API calls (version checks)
	DiscoveryTimeout = 10 * time.Second
)

// getClusterVersion gets the Kubernetes version from the cluster
func getClusterVersion(ctx context.Context, clientset kubernetes.Interface) (string, error) {
	// Use a shorter timeout for version discovery
	discoveryCtx, cancel := context.WithTimeout(ctx, DiscoveryTimeout)
	defer cancel()

	// Note: ServerVersion doesn't accept context in current client-go version
	// but we still create the context for future compatibility
	_ = discoveryCtx
	versionInfo, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return versionInfo.GitVersion, nil
}

// collectNodeInfo collects node information from the cluster
func collectNodeInfo(ctx context.Context, clientset kubernetes.Interface) ([]NodeInfo, int, error) {
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, 0, err
	}

	nodeInfos := make([]NodeInfo, 0, len(nodes.Items))
	healthyCount := 0

	for _, node := range nodes.Items {
		nodeInfo := NodeInfo{
			Name:  node.Name,
			Roles: getNodeRoles(&node),
			Age:   time.Since(node.CreationTimestamp.Time).Round(time.Hour).String(),
		}

		// Determine node status
		nodeStatus := "Unknown"
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				if condition.Status == corev1.ConditionTrue {
					nodeStatus = "Ready"
					healthyCount++
				} else {
					nodeStatus = "NotReady"
				}
				break
			}
		}
		nodeInfo.Status = nodeStatus
		nodeInfos = append(nodeInfos, nodeInfo)
	}

	return nodeInfos, healthyCount, nil
}

// collectNamespaceList collects the list of namespaces from the cluster
func collectNamespaceList(ctx context.Context, clientset kubernetes.Interface) ([]string, error) {
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	namespaceList := make([]string, len(namespaces.Items))
	for i, ns := range namespaces.Items {
		namespaceList[i] = ns.Name
	}
	return namespaceList, nil
}

// isPodHealthy checks if a pod is healthy
func isPodHealthy(pod *corev1.Pod) bool {
	// Succeeded pods are completed jobs and should be considered healthy
	if pod.Status.Phase == corev1.PodSucceeded {
		return true
	}

	// For running pods, check if all containers are ready
	if pod.Status.Phase == corev1.PodRunning {
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				return false
			}
		}
		return true
	}

	// All other phases (Pending, Failed, Unknown) are unhealthy
	return false
}

// extractPodInfo extracts relevant information from an unhealthy pod
func extractPodInfo(pod *corev1.Pod) PodInfo {
	podInfo := PodInfo{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Status:    string(pod.Status.Phase),
	}

	// Get restart count
	for _, cs := range pod.Status.ContainerStatuses {
		podInfo.Restarts += cs.RestartCount
	}

	// Get reason for unhealthy state
	if pod.Status.Reason != "" {
		podInfo.Reason = pod.Status.Reason
	} else if len(pod.Status.ContainerStatuses) > 0 {
		cs := pod.Status.ContainerStatuses[0]
		if cs.State.Waiting != nil {
			podInfo.Reason = cs.State.Waiting.Reason
		} else if cs.State.Terminated != nil {
			podInfo.Reason = cs.State.Terminated.Reason
		} else if !cs.Ready {
			podInfo.Reason = "ContainerNotReady"
		}
	}

	return podInfo
}

// collectPodHealth collects pod health information from the cluster
func collectPodHealth(ctx context.Context, clientset kubernetes.Interface) (int, int, []PodInfo, error) {
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, 0, nil, err
	}

	totalPods := len(pods.Items)
	healthyPods := 0
	unhealthyPods := make([]PodInfo, 0)

	for _, pod := range pods.Items {
		if isPodHealthy(&pod) {
			healthyPods++
		} else {
			unhealthyPods = append(unhealthyPods, extractPodInfo(&pod))
		}
	}

	return totalPods, healthyPods, unhealthyPods, nil
}

// systemNamespaces contains Kubernetes-managed namespaces excluded from sanitization by default
var systemNamespaces = map[string]bool{
	"kube-system":     true,
	"kube-public":     true,
	"kube-node-lease": true,
}

const (
	sanitizePenaltyCritical = 10
	sanitizePenaltyMajor    = 5
	sanitizePenaltyMinor    = 2
)

// checkContainerSecurityContext evaluates CKS security-context rules (CKS-001, CKS-002, CKS-003).
func checkContainerSecurityContext(c corev1.Container, workload string, findings *[]SanitizeFinding) {
	// CKS-001: Privileged container
	if c.SecurityContext != nil &&
		c.SecurityContext.Privileged != nil &&
		*c.SecurityContext.Privileged {
		*findings = append(*findings, SanitizeFinding{
			RuleID:    "CKS-001",
			Severity:  SanitizeCritical,
			Workload:  workload,
			Container: c.Name,
			Message:   "container is running in privileged mode",
			Penalty:   sanitizePenaltyCritical,
		})
	}

	// CKS-002: Container may run as root
	runAsNonRoot := false
	if c.SecurityContext != nil {
		if c.SecurityContext.RunAsNonRoot != nil && *c.SecurityContext.RunAsNonRoot {
			runAsNonRoot = true
		}
		if c.SecurityContext.RunAsUser != nil && *c.SecurityContext.RunAsUser != 0 {
			runAsNonRoot = true
		}
	}
	if !runAsNonRoot {
		*findings = append(*findings, SanitizeFinding{
			RuleID:    "CKS-002",
			Severity:  SanitizeCritical,
			Workload:  workload,
			Container: c.Name,
			Message:   "container may run as root: runAsNonRoot is not true and no non-zero runAsUser is set",
			Penalty:   sanitizePenaltyCritical,
		})
	}

	// CKS-003: allowPrivilegeEscalation not explicitly false
	escalationBlocked := c.SecurityContext != nil &&
		c.SecurityContext.AllowPrivilegeEscalation != nil &&
		!*c.SecurityContext.AllowPrivilegeEscalation
	if !escalationBlocked {
		*findings = append(*findings, SanitizeFinding{
			RuleID:    "CKS-003",
			Severity:  SanitizeCritical,
			Workload:  workload,
			Container: c.Name,
			Message:   "allowPrivilegeEscalation is not explicitly set to false",
			Penalty:   sanitizePenaltyCritical,
		})
	}
}

// checkContainerFilesystemAndCaps evaluates filesystem and capability rules (BP-007, BP-008).
func checkContainerFilesystemAndCaps(c corev1.Container, workload string, findings *[]SanitizeFinding) {
	// BP-007: readOnlyRootFilesystem not true
	readOnly := c.SecurityContext != nil &&
		c.SecurityContext.ReadOnlyRootFilesystem != nil &&
		*c.SecurityContext.ReadOnlyRootFilesystem
	if !readOnly {
		*findings = append(*findings, SanitizeFinding{
			RuleID:    "BP-007",
			Severity:  SanitizeMinor,
			Workload:  workload,
			Container: c.Name,
			Message:   "readOnlyRootFilesystem is not set to true",
			Penalty:   sanitizePenaltyMinor,
		})
	}

	// BP-008: Capabilities not fully dropped
	allDropped := false
	if c.SecurityContext != nil && c.SecurityContext.Capabilities != nil {
		for _, cap := range c.SecurityContext.Capabilities.Drop {
			if string(cap) == "ALL" {
				allDropped = true
				break
			}
		}
	}
	if !allDropped {
		*findings = append(*findings, SanitizeFinding{
			RuleID:    "BP-008",
			Severity:  SanitizeMinor,
			Workload:  workload,
			Container: c.Name,
			Message:   "capabilities.drop does not include ALL",
			Penalty:   sanitizePenaltyMinor,
		})
	}
}

// checkContainerRules evaluates per-container linting rules and appends findings.
func checkContainerRules(c corev1.Container, workload string, findings *[]SanitizeFinding) {
	checkContainerSecurityContext(c, workload, findings)
	checkContainerFilesystemAndCaps(c, workload, findings)

	// BP-001: Missing livenessProbe
	if c.LivenessProbe == nil {
		*findings = append(*findings, SanitizeFinding{
			RuleID:    "BP-001",
			Severity:  SanitizeMajor,
			Workload:  workload,
			Container: c.Name,
			Message:   "no livenessProbe defined",
			Penalty:   sanitizePenaltyMajor,
		})
	}

	// BP-002: Missing readinessProbe
	if c.ReadinessProbe == nil {
		*findings = append(*findings, SanitizeFinding{
			RuleID:    "BP-002",
			Severity:  SanitizeMajor,
			Workload:  workload,
			Container: c.Name,
			Message:   "no readinessProbe defined",
			Penalty:   sanitizePenaltyMajor,
		})
	}

	// BP-003: Missing resource requests
	if len(c.Resources.Requests) == 0 {
		*findings = append(*findings, SanitizeFinding{
			RuleID:    "BP-003",
			Severity:  SanitizeMajor,
			Workload:  workload,
			Container: c.Name,
			Message:   "no resource requests defined",
			Penalty:   sanitizePenaltyMajor,
		})
	}

	// BP-004: Missing memory limit (OOM risk)
	memLimit := c.Resources.Limits[corev1.ResourceMemory]
	if memLimit.IsZero() {
		*findings = append(*findings, SanitizeFinding{
			RuleID:    "BP-004",
			Severity:  SanitizeMajor,
			Workload:  workload,
			Container: c.Name,
			Message:   "no memory limit defined (OOM risk)",
			Penalty:   sanitizePenaltyMajor,
		})
	}

	// BP-005: Image uses :latest or has no explicit tag (digest references are exempt)
	img := c.Image
	if !strings.Contains(img, "@") && (!strings.Contains(img, ":") || strings.HasSuffix(img, ":latest")) {
		*findings = append(*findings, SanitizeFinding{
			RuleID:    "BP-005",
			Severity:  SanitizeMajor,
			Workload:  workload,
			Container: c.Name,
			Message:   fmt.Sprintf("image %q uses :latest or has no explicit tag", img),
			Penalty:   sanitizePenaltyMajor,
		})
	}
}

// checkPodSpecRules evaluates pod-level security rules
func checkPodSpecRules(spec corev1.PodSpec, workload string, findings *[]SanitizeFinding) {
	// CKS-004: hostPID or hostNetwork enabled
	if spec.HostPID {
		*findings = append(*findings, SanitizeFinding{
			RuleID:   "CKS-004",
			Severity: SanitizeCritical,
			Workload: workload,
			Message:  "hostPID is enabled — shares host process namespace",
			Penalty:  sanitizePenaltyCritical,
		})
	}
	if spec.HostNetwork {
		*findings = append(*findings, SanitizeFinding{
			RuleID:   "CKS-004",
			Severity: SanitizeCritical,
			Workload: workload,
			Message:  "hostNetwork is enabled — exposes host network interfaces",
			Penalty:  sanitizePenaltyCritical,
		})
	}

	// CKS-005: hostPath volume present
	for _, vol := range spec.Volumes {
		if vol.HostPath != nil {
			*findings = append(*findings, SanitizeFinding{
				RuleID:   "CKS-005",
				Severity: SanitizeCritical,
				Workload: workload,
				Message:  fmt.Sprintf("hostPath volume %q mounts host path %q", vol.Name, vol.HostPath.Path),
				Penalty:  sanitizePenaltyCritical,
			})
		}
	}
}

// checkServiceRules evaluates linting rules for a Service resource
func checkServiceRules(svc corev1.ServiceSpec, workload string, findings *[]SanitizeFinding) {
	// SVC-001: NodePort exposes a port on every cluster node — unintended exposure risk
	if svc.Type == corev1.ServiceTypeNodePort {
		*findings = append(*findings, SanitizeFinding{
			RuleID:   "SVC-001",
			Severity: SanitizeMajor,
			Workload: workload,
			Message:  "Service type NodePort exposes a static port on every node — prefer LoadBalancer or Ingress",
			Penalty:  sanitizePenaltyMajor,
		})
	}
	// SVC-002: LoadBalancer with no load-balancer-source-ranges annotation — open to the internet
	if svc.Type == corev1.ServiceTypeLoadBalancer && len(svc.LoadBalancerSourceRanges) == 0 {
		*findings = append(*findings, SanitizeFinding{
			RuleID:   "SVC-002",
			Severity: SanitizeMajor,
			Workload: workload,
			Message:  "LoadBalancer Service has no loadBalancerSourceRanges — unrestricted internet access",
			Penalty:  sanitizePenaltyMajor,
		})
	}
	// SVC-003: Service has no selector — manual Endpoints management, easy to misconfigure
	if len(svc.Selector) == 0 && svc.Type != corev1.ServiceTypeExternalName {
		*findings = append(*findings, SanitizeFinding{
			RuleID:   "SVC-003",
			Severity: SanitizeMinor,
			Workload: workload,
			Message:  "Service has no selector — relies on manual Endpoints which may route to wrong or no pods",
			Penalty:  sanitizePenaltyMinor,
		})
	}
}

// checkIngressRules evaluates linting rules for an Ingress resource
func checkIngressRules(ing networkingv1.IngressSpec, workload string, findings *[]SanitizeFinding) {
	// ING-001: No TLS configured — HTTP traffic is unencrypted in transit
	if len(ing.TLS) == 0 {
		*findings = append(*findings, SanitizeFinding{
			RuleID:   "ING-001",
			Severity: SanitizeMajor,
			Workload: workload,
			Message:  "Ingress has no TLS configuration — traffic is unencrypted in transit",
			Penalty:  sanitizePenaltyMajor,
		})
	}
	// ING-002: Wildcard host rule
	for _, rule := range ing.Rules {
		if rule.Host == "" || strings.Contains(rule.Host, "*") {
			*findings = append(*findings, SanitizeFinding{
				RuleID:   "ING-002",
				Severity: SanitizeMinor,
				Workload: workload,
				Message:  fmt.Sprintf("Ingress rule has wildcard or empty host %q — broad exposure, hard to audit", rule.Host),
				Penalty:  sanitizePenaltyMinor,
			})
			break // one finding per Ingress is enough
		}
	}
}

// checkWorkloadMetadata evaluates workload-level label and annotation rules
func checkWorkloadMetadata(labels, annotations map[string]string, workload string, findings *[]SanitizeFinding) {
	// BP-009: Missing recommended "app" label
	if _, ok := labels["app"]; !ok {
		*findings = append(*findings, SanitizeFinding{
			RuleID:   "BP-009",
			Severity: SanitizeMinor,
			Workload: workload,
			Message:  "missing recommended label 'app'",
			Penalty:  sanitizePenaltyMinor,
		})
	}
	// BP-010: Missing recommended version annotation
	if _, ok := annotations["app.kubernetes.io/version"]; !ok {
		*findings = append(*findings, SanitizeFinding{
			RuleID:   "BP-010",
			Severity: SanitizeMinor,
			Workload: workload,
			Message:  "missing recommended annotation 'app.kubernetes.io/version'",
			Penalty:  sanitizePenaltyMinor,
		})
	}
}

// scoreFromFindings computes a 0–100 score by subtracting finding penalties from 100
func scoreFromFindings(findings []SanitizeFinding) int {
	penalty := 0
	for _, f := range findings {
		penalty += f.Penalty
	}
	if score := 100 - penalty; score > 0 {
		return score
	}
	return 0
}

// gradeFromScore converts a numerical score to a letter grade
func gradeFromScore(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 75:
		return "B"
	case score >= 60:
		return "C"
	case score >= 40:
		return "D"
	default:
		return "F"
	}
}

// shouldScanNamespace reports whether a namespace should be included in the sanitize scan.
func shouldScanNamespace(ns, targetNamespace string, includeSystem bool) bool {
	if targetNamespace != "" {
		return ns == targetNamespace
	}
	return includeSystem || !systemNamespaces[ns]
}

// podWorkloadRef identifies a single Kubernetes workload to be scanned.
type podWorkloadRef struct {
	namespace, kind, name string
	labels, annotations   map[string]string
}

// podScanAccumulator carries the shared filter config and output slices for a scan.
type podScanAccumulator struct {
	targetNamespace string
	includeSystem   bool
	findings        *[]SanitizeFinding
	workloads       *[]string
}

// scanPodWorkloadItem runs all pod-level rules against a single workload and appends results.
func scanPodWorkloadItem(ref podWorkloadRef, spec corev1.PodSpec, replicas int32, acc podScanAccumulator) {
	if !shouldScanNamespace(ref.namespace, acc.targetNamespace, acc.includeSystem) {
		return
	}
	workload := fmt.Sprintf("%s/%s/%s", ref.namespace, ref.kind, ref.name)
	*acc.workloads = append(*acc.workloads, workload)
	var findings []SanitizeFinding

	checkPodSpecRules(spec, workload, &findings)
	checkWorkloadMetadata(ref.labels, ref.annotations, workload, &findings)
	for _, c := range spec.Containers {
		checkContainerRules(c, workload, &findings)
	}

	// BP-006: single replica is a reliability risk for Deployments and StatefulSets
	if (ref.kind == "Deployment" || ref.kind == "StatefulSet") && replicas <= 1 {
		findings = append(findings, SanitizeFinding{
			RuleID:   "BP-006",
			Severity: SanitizeMajor,
			Workload: workload,
			Message:  fmt.Sprintf("%d replica(s) configured — single point of failure", replicas),
			Penalty:  sanitizePenaltyMajor,
		})
	}

	*acc.findings = append(*acc.findings, findings...)
}

// collectPodWorkloads scans Deployments, StatefulSets, DaemonSets, and CronJobs.
func collectPodWorkloads(ctx context.Context, clientset kubernetes.Interface, targetNamespace string, includeSystem bool, allFindings *[]SanitizeFinding, allWorkloads *[]string) error {
	acc := podScanAccumulator{
		targetNamespace: targetNamespace,
		includeSystem:   includeSystem,
		findings:        allFindings,
		workloads:       allWorkloads,
	}

	deploys, err := clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}
	for _, d := range deploys.Items {
		replicas := int32(1)
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}
		scanPodWorkloadItem(podWorkloadRef{d.Namespace, "Deployment", d.Name, d.Labels, d.Annotations}, d.Spec.Template.Spec, replicas, acc)
	}

	stss, err := clientset.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list statefulsets: %w", err)
	}
	for _, ss := range stss.Items {
		replicas := int32(1)
		if ss.Spec.Replicas != nil {
			replicas = *ss.Spec.Replicas
		}
		scanPodWorkloadItem(podWorkloadRef{ss.Namespace, "StatefulSet", ss.Name, ss.Labels, ss.Annotations}, ss.Spec.Template.Spec, replicas, acc)
	}

	// DaemonSets have no replica count; pass 2 to skip BP-006
	dss, err := clientset.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list daemonsets: %w", err)
	}
	for _, ds := range dss.Items {
		scanPodWorkloadItem(podWorkloadRef{ds.Namespace, "DaemonSet", ds.Name, ds.Labels, ds.Annotations}, ds.Spec.Template.Spec, 2, acc)
	}

	// CronJobs: pod-spec rules apply; BP-006 skipped via replicas=2
	cjs, err := clientset.BatchV1().CronJobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list cronjobs: %w", err)
	}
	for _, cj := range cjs.Items {
		scanPodWorkloadItem(podWorkloadRef{cj.Namespace, "CronJob", cj.Name, cj.Labels, cj.Annotations}, cj.Spec.JobTemplate.Spec.Template.Spec, 2, acc)
	}

	return nil
}

// collectNetworkResources scans Services and Ingresses.
func collectNetworkResources(ctx context.Context, clientset kubernetes.Interface, targetNamespace string, includeSystem bool, allFindings *[]SanitizeFinding, allWorkloads *[]string) error {
	svcs, err := clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}
	for _, svc := range svcs.Items {
		if !shouldScanNamespace(svc.Namespace, targetNamespace, includeSystem) {
			continue
		}
		// Skip the built-in kubernetes Service
		if svc.Namespace == "default" && svc.Name == "kubernetes" {
			continue
		}
		workload := fmt.Sprintf("%s/Service/%s", svc.Namespace, svc.Name)
		*allWorkloads = append(*allWorkloads, workload)
		var findings []SanitizeFinding
		checkServiceRules(svc.Spec, workload, &findings)
		*allFindings = append(*allFindings, findings...)
	}

	ings, err := clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list ingresses: %w", err)
	}
	for _, ing := range ings.Items {
		if !shouldScanNamespace(ing.Namespace, targetNamespace, includeSystem) {
			continue
		}
		workload := fmt.Sprintf("%s/Ingress/%s", ing.Namespace, ing.Name)
		*allWorkloads = append(*allWorkloads, workload)
		var findings []SanitizeFinding
		checkIngressRules(ing.Spec, workload, &findings)
		*allFindings = append(*allFindings, findings...)
	}

	return nil
}

// collectSanitizeFindings inspects Deployments, StatefulSets, DaemonSets, CronJobs, Services, and
// Ingresses in a cluster against best-practice and security rules. If targetNamespace is non-empty,
// only that namespace is scanned. If includeSystem is false, system namespaces are excluded.
// Returns the list of findings and the IDs of every scanned resource (ns/Kind/name).
func collectSanitizeFindings(ctx context.Context, clientset kubernetes.Interface, targetNamespace string, includeSystem bool) ([]SanitizeFinding, []string, error) {
	allFindings := make([]SanitizeFinding, 0)
	allWorkloads := make([]string, 0)

	if err := collectPodWorkloads(ctx, clientset, targetNamespace, includeSystem, &allFindings, &allWorkloads); err != nil {
		return nil, nil, err
	}
	if err := collectNetworkResources(ctx, clientset, targetNamespace, includeSystem, &allFindings, &allWorkloads); err != nil {
		return nil, nil, err
	}

	return allFindings, allWorkloads, nil
}

// averageScores returns the integer floor-average of a slice of scores, or 100 if empty.
func averageScores(scores []int) int {
	if len(scores) == 0 {
		return 100
	}
	sum := 0
	for _, s := range scores {
		sum += s
	}
	return sum / len(scores)
}

// buildSanitizeResult groups findings by namespace and computes scores by averaging per-workload
// scores rather than accumulating global penalties. This ensures that a large number of workloads
// does not unfairly collapse the cluster score to zero.
func buildSanitizeResult(contextName string, findings []SanitizeFinding, allWorkloads []string) *SanitizeResult {
	// Index findings by workload ID
	byWorkload := make(map[string][]SanitizeFinding)
	for _, f := range findings {
		byWorkload[f.Workload] = append(byWorkload[f.Workload], f)
	}

	// Group workload IDs by namespace
	byNS := make(map[string][]string)
	for _, w := range allWorkloads {
		ns := strings.SplitN(w, "/", 3)[0]
		byNS[ns] = append(byNS[ns], w)
	}

	// Compute per-namespace scores (mean of the per-workload scores in that namespace)
	nsList := make([]NamespaceSanitizeScore, 0, len(byNS))
	var allScores []int
	for ns, workloads := range byNS {
		var nsFindings []SanitizeFinding
		var nsScores []int
		for _, w := range workloads {
			wf := byWorkload[w]
			nsFindings = append(nsFindings, wf...)
			ws := scoreFromFindings(wf)
			nsScores = append(nsScores, ws)
		}
		allScores = append(allScores, nsScores...)
		nsScore := averageScores(nsScores)
		nsList = append(nsList, NamespaceSanitizeScore{
			Namespace: ns,
			Score:     nsScore,
			Grade:     gradeFromScore(nsScore),
			Findings:  nsFindings,
		})
	}

	// Sort namespaces worst score first; alphabetically as a tiebreaker
	sort.Slice(nsList, func(i, j int) bool {
		if nsList[i].Score != nsList[j].Score {
			return nsList[i].Score < nsList[j].Score
		}
		return nsList[i].Namespace < nsList[j].Namespace
	})

	// Cluster score = mean of every individual workload score
	clusterScore := averageScores(allScores)

	criticalCount, majorCount, minorCount := 0, 0, 0
	for _, f := range findings {
		switch f.Severity {
		case SanitizeCritical:
			criticalCount++
		case SanitizeMajor:
			majorCount++
		case SanitizeMinor:
			minorCount++
		}
	}

	return &SanitizeResult{
		Context:        contextName,
		Score:          clusterScore,
		Grade:          gradeFromScore(clusterScore),
		TotalWorkloads: len(allWorkloads),
		TotalFindings:  len(findings),
		CriticalCount:  criticalCount,
		MajorCount:     majorCount,
		MinorCount:     minorCount,
		Namespaces:     nsList,
	}
}

// getNodeRoles extracts roles from node labels
func getNodeRoles(node *corev1.Node) []string {
	roles := make([]string, 0)
	for label := range node.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	if len(roles) == 0 {
		roles = append(roles, "worker")
	}
	return roles
}
