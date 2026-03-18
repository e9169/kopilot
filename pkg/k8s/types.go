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

// SanitizeSeverity defines the severity level of a sanitize finding
type SanitizeSeverity string

const (
	// SanitizeCritical represents a critical security issue (penalty −10)
	SanitizeCritical SanitizeSeverity = "critical"
	// SanitizeMajor represents a major best-practice violation (penalty −5)
	SanitizeMajor SanitizeSeverity = "major"
	// SanitizeMinor represents a minor best-practice violation (penalty −2)
	SanitizeMinor SanitizeSeverity = "minor"
)

// SanitizeFinding represents a single linting finding for a workload or container
type SanitizeFinding struct {
	RuleID    string           `json:"rule_id"`
	Severity  SanitizeSeverity `json:"severity"`
	Workload  string           `json:"workload"`  // namespace/Kind/name
	Container string           `json:"container"` // container name; empty for pod-level rules
	Message   string           `json:"message"`
	Penalty   int              `json:"penalty"`
}

// NamespaceSanitizeScore holds the sanitization score for a single namespace
type NamespaceSanitizeScore struct {
	Namespace string            `json:"namespace"`
	Score     int               `json:"score"`
	Grade     string            `json:"grade"`
	Findings  []SanitizeFinding `json:"findings"`
}

// SanitizeResult holds the complete sanitization results for a cluster
type SanitizeResult struct {
	Context        string                   `json:"context"`
	Score          int                      `json:"score"`
	Grade          string                   `json:"grade"`
	TotalWorkloads int                      `json:"total_workloads"`
	TotalFindings  int                      `json:"total_findings"`
	CriticalCount  int                      `json:"critical_count"`
	MajorCount     int                      `json:"major_count"`
	MinorCount     int                      `json:"minor_count"`
	Namespaces     []NamespaceSanitizeScore `json:"namespaces"`
}
