// Package k8s provides Kubernetes cluster interaction capabilities.
// This file defines shared types used by the provider and collectors.
package k8s

import (
	"sync"
	"time"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// ClusterInfo represents information about a Kubernetes cluster
type ClusterInfo struct {
	Name        string
	Server      string
	Context     string
	User        string
	Namespace   string
	IsCurrent   bool
	IsReachable bool
}

// ClusterStatus represents detailed status information about a cluster
type ClusterStatus struct {
	ClusterInfo
	Version       string
	NodeCount     int
	HealthyNodes  int
	Nodes         []NodeInfo
	NamespaceList []string
	APIServerURL  string
	Error         string
	PodCount      int
	HealthyPods   int
	UnhealthyPods []PodInfo
}

// NodeInfo represents information about a Kubernetes node
type NodeInfo struct {
	Name   string
	Status string
	Roles  []string
	Age    string
}

// PodInfo represents information about an unhealthy pod
type PodInfo struct {
	Name      string
	Namespace string
	Status    string
	Reason    string
	Restarts  int32
}

// CachedClusterStatus holds a cached cluster status with expiration
type CachedClusterStatus struct {
	Status    *ClusterStatus
	ExpiresAt time.Time
}

// Provider manages Kubernetes cluster information and operations
type Provider struct {
	kubeconfigPath string
	rawConfig      *clientcmdapi.Config
	clusters       map[string]*ClusterInfo
	currentContext string

	// Caching support
	cacheMutex sync.RWMutex
	cache      map[string]*CachedClusterStatus
	cacheTTL   time.Duration
}
