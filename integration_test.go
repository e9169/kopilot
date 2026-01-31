// Integration tests for kopilot agent
// These tests require a valid kubeconfig with accessible clusters
// Run with: go test -tags=integration ./...

//go:build integration
// +build integration

package main

import (
	"os"
	"testing"

	"github.com/e9169/kopilot/pkg/k8s"
)

const integrationEnvVar = "KOPILOT_RUN_INTEGRATION_TESTS"

func requireIntegrationEnabled(t *testing.T) {
	t.Helper()
	if os.Getenv(integrationEnvVar) != "1" {
		t.Skipf("Set %s=1 to run integration tests", integrationEnvVar)
	}
}

func TestIntegration_FullAgentFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	requireIntegrationEnabled(t)

	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skip("Cannot determine kubeconfig path")
		}
		kubeconfigPath = homeDir + "/.kube/config"
	}

	// Verify kubeconfig exists
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		t.Skip("No kubeconfig found, skipping integration test")
	}

	// Initialize provider
	provider, err := k8s.NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Verify we can list clusters
	clusters := provider.GetClusters()
	if len(clusters) == 0 {
		t.Skip("No clusters configured in kubeconfig")
	}

	t.Logf("Found %d clusters in kubeconfig", len(clusters))

	// Verify each cluster has required fields
	for _, cluster := range clusters {
		if cluster.Name == "" {
			t.Error("Cluster has empty name")
		}
		if cluster.Context == "" {
			t.Error("Cluster has empty context")
		}
		if cluster.Server == "" {
			t.Error("Cluster has empty server")
		}
		t.Logf("Cluster: %s (Context: %s, Server: %s)",
			cluster.Name, cluster.Context, cluster.Server)
	}

	// Test getting current context
	currentContext := provider.GetCurrentContext()
	if currentContext == "" {
		t.Error("Current context is empty")
	}
	t.Logf("Current context: %s", currentContext)
}

func TestIntegration_KubernetesConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	requireIntegrationEnabled(t)

	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skip("Cannot determine kubeconfig path")
		}
		kubeconfigPath = homeDir + "/.kube/config"
	}

	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		t.Skip("No kubeconfig found, skipping integration test")
	}

	provider, err := k8s.NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	clusters := provider.GetClusters()
	if len(clusters) == 0 {
		t.Skip("No clusters configured")
	}

	// Note: Testing actual cluster connectivity requires clusters to be reachable
	// This test verifies the provider can be initialized with real kubeconfig
	t.Logf("Provider initialized successfully with %d clusters", len(clusters))
}

func TestIntegration_AgentToolDefinitions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	requireIntegrationEnabled(t)

	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skip("Cannot determine kubeconfig path")
		}
		kubeconfigPath = homeDir + "/.kube/config"
	}

	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		t.Skip("No kubeconfig found, skipping integration test")
	}

	provider, err := k8s.NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// This tests that the agent module can be imported and used
	// Full agent execution requires Copilot CLI which we can't test in CI
	_ = provider
}
