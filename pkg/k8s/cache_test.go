package k8s

import (
	"testing"
	"time"
)

// TestCacheBasicFunctionality tests basic cache operations
func TestCacheBasicFunctionality(t *testing.T) {
	kubeconfigPath, cleanup := createTempKubeconfig(t, 1)
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf("NewProvider() failed: %v", err)
	}

	contextName := "context-1"
	status := &ClusterStatus{
		ClusterInfo: ClusterInfo{
			Name:    "test-cluster",
			Context: contextName,
		},
		Version: "v1.28.0",
	}

	// Test caching a status
	provider.cacheStatus(contextName, status)

	// Test retrieving cached status
	cached := provider.getCachedStatus(contextName)
	if cached == nil {
		t.Error("Expected to retrieve cached status, got nil")
	}
	if cached.Version != status.Version {
		t.Errorf("Cached version = %s, want %s", cached.Version, status.Version)
	}
}

// TestCacheExpiration tests that cached entries expire after TTL
func TestCacheExpiration(t *testing.T) {
	kubeconfigPath, cleanup := createTempKubeconfig(t, 1)
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf("NewProvider() failed: %v", err)
	}

	// Set a very short TTL for testing
	provider.SetCacheTTL(10 * time.Millisecond)

	contextName := "context-1"
	status := &ClusterStatus{
		ClusterInfo: ClusterInfo{
			Name:    "test-cluster",
			Context: contextName,
		},
	}

	provider.cacheStatus(contextName, status)

	// Should be cached immediately
	if cached := provider.getCachedStatus(contextName); cached == nil {
		t.Error("Expected cached status immediately after caching")
	}

	// Wait for expiration
	time.Sleep(15 * time.Millisecond)

	// Should be nil after expiration
	if cached := provider.getCachedStatus(contextName); cached != nil {
		t.Error("Expected nil after cache expiration")
	}
}

// TestClearCache tests clearing the cache
func TestClearCache(t *testing.T) {
	kubeconfigPath, cleanup := createTempKubeconfig(t, 2)
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf("NewProvider() failed: %v", err)
	}

	// Cache two statuses
	provider.cacheStatus("context-1", &ClusterStatus{Version: "v1.28.0"})
	provider.cacheStatus("context-2", &ClusterStatus{Version: "v1.28.1"})

	// Verify both are cached
	if provider.getCachedStatus("context-1") == nil {
		t.Error("Expected context-1 to be cached")
	}
	if provider.getCachedStatus("context-2") == nil {
		t.Error("Expected context-2 to be cached")
	}

	// Clear cache
	provider.ClearCache()

	// Verify both are cleared
	if provider.getCachedStatus("context-1") != nil {
		t.Error("Expected context-1 to be cleared")
	}
	if provider.getCachedStatus("context-2") != nil {
		t.Error("Expected context-2 to be cleared")
	}
}

// TestSetCacheTTL tests modifying the cache TTL
func TestSetCacheTTL(t *testing.T) {
	kubeconfigPath, cleanup := createTempKubeconfig(t, 1)
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)
	if err != nil {
		t.Fatalf("NewProvider() failed: %v", err)
	}

	// Default TTL should be 1 minute
	defaultTTL := 1 * time.Minute
	if provider.cacheTTL != defaultTTL {
		t.Errorf("Default cacheTTL = %v, want %v", provider.cacheTTL, defaultTTL)
	}

	// Set new TTL
	newTTL := 5 * time.Minute
	provider.SetCacheTTL(newTTL)

	if provider.cacheTTL != newTTL {
		t.Errorf("After SetCacheTTL, cacheTTL = %v, want %v", provider.cacheTTL, newTTL)
	}
}

// BenchmarkCacheWrite benchmarks writing to the cache
func BenchmarkCacheWrite(b *testing.B) {
	kubeconfigPath, cleanup := createTempKubeconfig(&testing.T{}, 1)
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)
	if err != nil {
		b.Fatalf("NewProvider() failed: %v", err)
	}

	status := &ClusterStatus{Version: "v1.28.0"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.cacheStatus("context-1", status)
	}
}

// BenchmarkCacheRead benchmarks reading from the cache
func BenchmarkCacheRead(b *testing.B) {
	kubeconfigPath, cleanup := createTempKubeconfig(&testing.T{}, 1)
	defer cleanup()

	provider, err := NewProvider(kubeconfigPath)
	if err != nil {
		b.Fatalf("NewProvider() failed: %v", err)
	}

	status := &ClusterStatus{Version: "v1.28.0"}
	provider.cacheStatus("context-1", status)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.getCachedStatus("context-1")
	}
}
