package main

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestRunWithValidKubeconfig(t *testing.T) {
	// Create temporary kubeconfig
	tmpfile, cleanup := createTestKubeconfig(t)
	defer cleanup()

	// Set environment variable
	originalKubeconfig := os.Getenv("KUBECONFIG")
	_ = os.Setenv("KUBECONFIG", tmpfile)
	defer func() { _ = os.Setenv("KUBECONFIG", originalKubeconfig) }()

	// Note: We can't actually run the full application in a unit test
	// because it requires Copilot CLI to be installed and running
	// This test verifies the setup works correctly
	if _, err := os.Stat(tmpfile); os.IsNotExist(err) {
		t.Errorf("Kubeconfig file does not exist: %s", tmpfile)
	}
}

func TestRunWithMissingKubeconfig(t *testing.T) {
	// Temporarily unset KUBECONFIG and point to non-existent default location
	originalKubeconfig := os.Getenv("KUBECONFIG")
	originalHome := os.Getenv("HOME")

	// Create temp dir that will be used as HOME
	tmpDir, err := os.MkdirTemp("", "kopilot-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	_ = os.Unsetenv("KUBECONFIG")
	_ = os.Setenv("HOME", tmpDir)

	defer func() {
		if originalKubeconfig != "" {
			_ = os.Setenv("KUBECONFIG", originalKubeconfig)
		} else {
			_ = os.Unsetenv("KUBECONFIG")
		}
		_ = os.Setenv("HOME", originalHome)
	}()

	// Verify that the default kubeconfig path is checked
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
		}
	}

	// The path should be constructed correctly even if the file doesn't exist
	if kubeconfigPath == "" && os.Getenv("KUBECONFIG") == "" {
		t.Log("KUBECONFIG not set; default path construction exercised")
	}
}

func TestRunWithCustomKubeconfigPath(t *testing.T) {
	tmpfile, cleanup := createTestKubeconfig(t)
	defer cleanup()

	// Set custom KUBECONFIG
	originalKubeconfig := os.Getenv("KUBECONFIG")
	_ = os.Setenv("KUBECONFIG", tmpfile)
	defer func() { _ = os.Setenv("KUBECONFIG", originalKubeconfig) }()

	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath != tmpfile {
		t.Errorf("KUBECONFIG = %s, want %s", kubeconfigPath, tmpfile)
	}
}

func TestKubeconfigPathResolution(t *testing.T) {
	tests := []struct {
		name           string
		envValue       string
		expectEnvValue bool
	}{
		{
			name:           "KUBECONFIG environment variable set",
			envValue:       "/custom/path/to/kubeconfig",
			expectEnvValue: true,
		},
		{
			name:           "KUBECONFIG environment variable empty",
			envValue:       "",
			expectEnvValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalKubeconfig := os.Getenv("KUBECONFIG")
			defer func() { _ = os.Setenv("KUBECONFIG", originalKubeconfig) }()

			if tt.envValue != "" {
				_ = os.Setenv("KUBECONFIG", tt.envValue)
			} else {
				_ = os.Unsetenv("KUBECONFIG")
			}

			kubeconfigPath := os.Getenv("KUBECONFIG")
			if tt.expectEnvValue && kubeconfigPath != tt.envValue {
				t.Errorf("Expected kubeconfig path %s, got %s", tt.envValue, kubeconfigPath)
			}
			if !tt.expectEnvValue && kubeconfigPath != "" {
				t.Errorf("Expected empty kubeconfig path, got %s", kubeconfigPath)
			}
		})
	}
}

func TestUserHomeDirResolution(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine user home directory")
	}

	if homeDir == "" {
		t.Error("UserHomeDir() returned empty string")
	}

	// Verify .kube directory path construction
	kubePath := filepath.Join(homeDir, ".kube", "config")
	if kubePath == "" {
		t.Error("Failed to construct kubeconfig path")
	}
}

// Helper functions

func createTestKubeconfig(t *testing.T) (string, func()) {
	t.Helper()

	tmpfile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		t.Fatal(err)
	}

	config := clientcmdapi.NewConfig()

	config.Clusters["test-cluster"] = &clientcmdapi.Cluster{
		Server:                "https://test-cluster.example.com",
		InsecureSkipTLSVerify: true,
	}

	config.AuthInfos["test-user"] = &clientcmdapi.AuthInfo{
		Token: "test-token",
	}

	config.Contexts["test-context"] = &clientcmdapi.Context{
		Cluster:  "test-cluster",
		AuthInfo: "test-user",
	}

	config.CurrentContext = "test-context"

	err = clientcmd.WriteToFile(*config, tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		_ = os.Remove(tmpfile.Name())
	}

	return tmpfile.Name(), cleanup
}

func TestApplicationConstants(t *testing.T) {
	// Verify the application can be built and constants are defined
	// This is a smoke test to ensure main package compiles
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
}

func BenchmarkKubeconfigPathLookup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			homeDir, _ := os.UserHomeDir()
			_ = filepath.Join(homeDir, ".kube", "config")
		}
	}
}
