// Package k8s provides Kubernetes cluster interaction capabilities.
// It handles kubeconfig parsing, cluster discovery, and status monitoring
// across multiple Kubernetes clusters with support for concurrent operations.
package k8s

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewProvider creates a new Kubernetes provider
func NewProvider(kubeconfigPath string) (*Provider, error) {
	// Load kubeconfig
	rawConfig, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Parse cluster information
	clusters := make(map[string]*ClusterInfo)
	currentContext := rawConfig.CurrentContext

	for contextName, contextInfo := range rawConfig.Contexts {
		clusterName := contextInfo.Cluster
		cluster, ok := rawConfig.Clusters[clusterName]
		if !ok {
			continue
		}

		clusters[contextName] = &ClusterInfo{
			Name:      clusterName,
			Server:    cluster.Server,
			Context:   contextName,
			User:      contextInfo.AuthInfo,
			Namespace: contextInfo.Namespace,
			IsCurrent: contextName == currentContext,
		}
	}

	return &Provider{
		kubeconfigPath: kubeconfigPath,
		rawConfig:      rawConfig,
		clusters:       clusters,
		currentContext: currentContext,
		cache:          make(map[string]*CachedClusterStatus),
		cacheTTL:       1 * time.Minute, // Default 1 minute cache
	}, nil
}

// GetClusters returns a list of all clusters in the kubeconfig
func (p *Provider) GetClusters() []*ClusterInfo {
	clusters := make([]*ClusterInfo, 0, len(p.clusters))
	for _, cluster := range p.clusters {
		clusters = append(clusters, cluster)
	}
	return clusters
}

// GetClusterByContext returns cluster information for a specific context
func (p *Provider) GetClusterByContext(contextName string) (*ClusterInfo, error) {
	cluster, ok := p.clusters[contextName]
	if !ok {
		return nil, fmt.Errorf("cluster context %q not found", contextName)
	}
	return cluster, nil
}

// GetClusterStatus returns detailed status information for a cluster
// createClientset creates a Kubernetes clientset for the given context
func (p *Provider) createClientset(contextName string) (kubernetes.Interface, *rest.Config, error) {
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: p.kubeconfigPath},
		&clientcmd.ConfigOverrides{CurrentContext: contextName},
	)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return clientset, restConfig, nil
}

func (p *Provider) GetClusterStatus(ctx context.Context, contextName string) (*ClusterStatus, error) {
	// Check cache first
	if cached := p.getCachedStatus(contextName); cached != nil {
		return cached, nil
	}

	clusterInfo, err := p.GetClusterByContext(contextName)
	if err != nil {
		return nil, err
	}

	status := &ClusterStatus{
		ClusterInfo: *clusterInfo,
	}

	// Create clientset for this specific context
	clientset, restConfig, err := p.createClientset(contextName)
	if err != nil {
		status.Error = err.Error()
		return status, nil
	}

	// Test connectivity with timeout
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Get cluster version
	version, err := getClusterVersion(queryCtx, clientset)
	if err != nil {
		status.Error = fmt.Sprintf("Failed to reach cluster: %v", err)
		status.IsReachable = false
		return status, nil
	}

	status.Version = version
	status.IsReachable = true
	status.APIServerURL = restConfig.Host

	// Collect node information
	nodeInfos, healthyNodes, err := collectNodeInfo(queryCtx, clientset)
	if err != nil {
		status.Error = fmt.Sprintf("Failed to list nodes: %v", err)
		return status, nil
	}
	status.Nodes = nodeInfos
	status.NodeCount = len(nodeInfos)
	status.HealthyNodes = healthyNodes

	// Collect namespace list
	namespaceList, err := collectNamespaceList(queryCtx, clientset)
	if err == nil {
		status.NamespaceList = namespaceList
	}

	// Collect pod health information
	totalPods, healthyPods, unhealthyPods, err := collectPodHealth(queryCtx, clientset)
	if err == nil {
		status.PodCount = totalPods
		status.HealthyPods = healthyPods
		status.UnhealthyPods = unhealthyPods
	}

	// Cache the result
	p.cacheStatus(contextName, status)
	return status, nil
}

// GetAllClusterStatuses returns status information for all clusters in parallel
func (p *Provider) GetAllClusterStatuses(ctx context.Context) []*ClusterStatus {
	clusters := p.GetClusters()
	statuses := make([]*ClusterStatus, len(clusters))

	var wg sync.WaitGroup
	for i, cluster := range clusters {
		wg.Add(1)
		go func(idx int, contextName string) {
			defer wg.Done()
			status, err := p.GetClusterStatus(ctx, contextName)
			if err != nil {
				// Create a status with error if GetClusterStatus fails
				statuses[idx] = &ClusterStatus{
					ClusterInfo: ClusterInfo{
						Context:     contextName,
						Name:        contextName,
						IsReachable: false,
					},
					Error: err.Error(),
				}
			} else {
				statuses[idx] = status
			}
		}(i, cluster.Context)
	}

	wg.Wait()
	return statuses
}

// GetCurrentContext returns the current context name
func (p *Provider) GetCurrentContext() string {
	return p.currentContext
}

// SetCurrentContext overrides the current context
func (p *Provider) SetCurrentContext(contextName string) error {
	if _, ok := p.clusters[contextName]; !ok {
		return fmt.Errorf("cluster context %q not found", contextName)
	}

	p.currentContext = contextName
	for _, cluster := range p.clusters {
		cluster.IsCurrent = cluster.Context == contextName
	}

	return nil
}
