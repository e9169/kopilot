//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/e9169/kopilot/pkg/k8s"
)

func main() {
	homeDir, _ := os.UserHomeDir()
	kubeconfigPath := filepath.Join(homeDir, ".kube", "config")

	provider, err := k8s.NewProvider(kubeconfigPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	clusters := provider.GetClusters()
	fmt.Printf("Testing with %d clusters\n\n", len(clusters))

	ctx := context.Background()

	// Test 1: Parallel execution
	fmt.Println("Test 1: Parallel execution with GetAllClusterStatuses()")
	start := time.Now()
	statuses := provider.GetAllClusterStatuses(ctx)
	parallelTime := time.Since(start)
	fmt.Printf("  Checked %d clusters in %v\n", len(statuses), parallelTime)
	reachable := 0
	for _, s := range statuses {
		if s.IsReachable {
			reachable++
		}
	}
	fmt.Printf("  Result: %d/%d clusters reachable\n\n", reachable, len(statuses))

	// Test 2: Sequential execution (for comparison)
	fmt.Println("Test 2: Sequential execution with individual GetClusterStatus() calls")
	start = time.Now()
	sequentialCount := 0
	sequentialReachable := 0
	for _, cluster := range clusters {
		status, _ := provider.GetClusterStatus(ctx, cluster.Context)
		if status != nil && status.IsReachable {
			sequentialReachable++
		}
		sequentialCount++
	}
	sequentialTime := time.Since(start)
	fmt.Printf("  Checked %d clusters in %v\n", sequentialCount, sequentialTime)
	fmt.Printf("  Result: %d/%d clusters reachable\n\n", sequentialReachable, sequentialCount)

	// Performance comparison
	fmt.Println("Performance Comparison:")
	fmt.Printf("  Parallel:   %v\n", parallelTime)
	fmt.Printf("  Sequential: %v\n", sequentialTime)
	if sequentialTime > parallelTime {
		speedup := float64(sequentialTime) / float64(parallelTime)
		fmt.Printf("  Speedup:    %.2fx faster 🚀\n", speedup)
	}
}
