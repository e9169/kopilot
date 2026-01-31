// Package k8s provides Kubernetes cluster interaction capabilities.
// This file contains helper functions for collecting cluster health data.
package k8s

import (
	"context"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
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
	if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded {
		return false
	}

	// Check if all containers are ready
	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready {
			return false
		}
	}

	return pod.Status.Phase == corev1.PodRunning
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
