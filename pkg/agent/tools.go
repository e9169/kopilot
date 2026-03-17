// Package agent provides the core Copilot agent functionality for Kubernetes cluster operations.
// This file contains tool definitions and related helpers.
package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/e9169/kopilot/pkg/k8s"
	copilot "github.com/github/copilot-sdk/go"
)

// defineTools creates all the Kubernetes-related tools for the agent
func defineTools(k8sProvider *k8s.Provider, state *agentState) []copilot.Tool {
	tools := []copilot.Tool{
		defineListClustersTool(k8sProvider, state),
		defineGetClusterStatusTool(k8sProvider, state),
		defineCompareClustersTool(k8sProvider, state),
		defineCheckAllClustersTool(k8sProvider, state),
		defineKubectlExecTool(k8sProvider, state),
		defineSanitizeClusterTool(k8sProvider, state),
		defineMCPListServersTool(state),
		defineMCPAddServerTool(state),
		defineMCPDeleteServerTool(state),
	}
	// Ensure all tool schemas are valid for all models/APIs.
	// Most LLM APIs (OpenAI, Anthropic, Google, etc.) require object schemas
	// to include a "properties" field even when there are no parameters.
	for i := range tools {
		tools[i] = fixEmptySchema(tools[i])
	}
	return tools
}

// fixEmptySchema ensures tools with no parameters have a valid JSON schema.
// The OpenAI API requires {"type":"object","properties":{}} for parameter-less tools.
// Without this, the SDK-generated schema for empty structs omits "properties",
// causing a 400 "object schema missing properties" error.
func fixEmptySchema(t copilot.Tool) copilot.Tool {
	if t.Parameters == nil {
		t.Parameters = map[string]any{}
	}
	if _, ok := t.Parameters["properties"]; !ok {
		t.Parameters["properties"] = map[string]any{}
	}
	return t
}

// ListClustersParams defines no parameters for list_clusters
type ListClustersParams struct{}

// ListClustersResult defines JSON output for list_clusters
type ListClustersResult struct {
	CurrentContext string             `json:"current_context"`
	Clusters       []*k8s.ClusterInfo `json:"clusters"`
}

func defineListClustersTool(k8sProvider *k8s.Provider, state *agentState) copilot.Tool {
	return copilot.DefineTool(
		toolListClusters,
		"List all Kubernetes clusters available in the kubeconfig file with basic information including context names, server URLs, and current context",
		func(params ListClustersParams, inv copilot.ToolInvocation) (any, error) {
			clusters := k8sProvider.GetClusters()
			currentContext := k8sProvider.GetCurrentContext()

			if isJSONOutput(state.outputFormat) {
				return ListClustersResult{
					CurrentContext: currentContext,
					Clusters:       clusters,
				}, nil
			}

			var result strings.Builder
			fmt.Fprintf(&result, "Found %d cluster(s):\n\n", len(clusters))

			for i, cluster := range clusters {
				marker := " "
				if cluster.Context == currentContext {
					marker = "*"
				}

				fmt.Fprintf(&result, "%s [%d] Context: %s\n", marker, i+1, cluster.Context)
				fmt.Fprintf(&result, "    Cluster: %s\n", cluster.Name)
				fmt.Fprintf(&result, "    Server: %s\n", cluster.Server)
				fmt.Fprintf(&result, "    User: %s\n", cluster.User)
				if cluster.Namespace != "" {
					fmt.Fprintf(&result, "    Default Namespace: %s\n", cluster.Namespace)
				}
				result.WriteString("\n")
			}

			fmt.Fprintf(&result, "\n* = Current context: %s\n", currentContext)

			return result.String(), nil
		},
	)
}

// GetClusterStatusParams defines parameters for get_cluster_status
type GetClusterStatusParams struct {
	Context string `json:"context" jsonschema:"The context name of the cluster to query (from list_clusters)"`
}

// writeUnreachableClusterStatus writes status for an unreachable cluster
func writeUnreachableClusterStatus(result *strings.Builder, status *k8s.ClusterStatus) {
	fmt.Fprintf(result, "❌ %s - DOWN (%s)\n", status.Context, status.Server)
	if status.Error != "" {
		fmt.Fprintf(result, "   Issue: %s\n", status.Error)
	}
}

// writeClusterInfo writes basic cluster information
func writeClusterInfo(result *strings.Builder, status *k8s.ClusterStatus) {
	result.WriteString("Cluster Information:\n")
	fmt.Fprintf(result, "  Context: %s\n", status.Context)
	fmt.Fprintf(result, "  API Server: %s\n", status.APIServerURL)
	fmt.Fprintf(result, "  Kubernetes Version: %s\n", status.Version)
	fmt.Fprintf(result, "  User: %s\n", status.User)
	if status.Namespace != "" {
		fmt.Fprintf(result, "  Default Namespace: %s\n", status.Namespace)
	}
	result.WriteString("\n")
}

// writeNodeInfo writes node information for a cluster
func writeNodeInfo(result *strings.Builder, status *k8s.ClusterStatus) {
	fmt.Fprintf(result, "Nodes: %d total, %d healthy\n", status.NodeCount, status.HealthyNodes)
	if len(status.Nodes) > 0 {
		result.WriteString("\nNode Details:\n")
		for _, node := range status.Nodes {
			statusIcon := "✅"
			if node.Status != "Ready" {
				statusIcon = "❌"
			}
			roles := strings.Join(node.Roles, ", ")
			fmt.Fprintf(result, "  %s %s\n", statusIcon, node.Name)
			fmt.Fprintf(result, "     Status: %s | Roles: %s | Age: %s\n", node.Status, roles, node.Age)
		}
	}
	result.WriteString("\n")
}

// writeNamespaceInfo writes namespace information for a cluster
func writeNamespaceInfo(result *strings.Builder, status *k8s.ClusterStatus) {
	if len(status.NamespaceList) > 0 {
		fmt.Fprintf(result, "Namespaces (%d):\n", len(status.NamespaceList))
		fmt.Fprintf(result, "  %s\n", strings.Join(status.NamespaceList, ", "))
	}

	if status.Error != "" {
		fmt.Fprintf(result, "\nWarning: %s\n", status.Error)
	}
}

func defineGetClusterStatusTool(k8sProvider *k8s.Provider, state *agentState) copilot.Tool {
	return copilot.DefineTool(
		toolGetClusterStatus,
		"Get detailed status information for a specific Kubernetes cluster including reachability, nodes, version, and health metrics. IMPORTANT: Present the tool output exactly as received - it contains visual card/box formatting. Do NOT convert it to a table.",
		func(params GetClusterStatusParams, inv copilot.ToolInvocation) (any, error) {
			ctx := context.Background()
			status, err := k8sProvider.GetClusterStatus(ctx, params.Context)
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster status: %w", err)
			}

			if isJSONOutput(state.outputFormat) {
				return status, nil
			}

			var result strings.Builder

			// Cluster header
			fmt.Fprintf(&result, "Cluster Status: %s\n", status.Name)
			result.WriteString(strings.Repeat("=", 80) + "\n\n")

			// Check if unreachable
			if !status.IsReachable {
				writeUnreachableClusterStatus(&result, status)
				return result.String(), nil
			}

			result.WriteString("✅ Status: REACHABLE\n\n")

			// Write cluster information
			writeClusterInfo(&result, status)
			writeNodeInfo(&result, status)
			writeNamespaceInfo(&result, status)

			return result.String(), nil
		},
	)
}

// CompareClusterParams defines parameters for compare_clusters
type CompareClusterParams struct {
	Contexts []string `json:"contexts" jsonschema:"Array of context names to compare (from list_clusters), minimum 2 required"`
}

// CompareClustersSummary defines summary JSON output for compare_clusters
type CompareClustersSummary struct {
	Total     int `json:"total"`
	Reachable int `json:"reachable"`
}

// CompareClustersResult defines JSON output for compare_clusters
type CompareClustersResult struct {
	Summary  CompareClustersSummary `json:"summary"`
	Clusters []ComparisonData       `json:"clusters"`
}

// ComparisonData holds comparison information for a cluster
type ComparisonData struct {
	Context      string
	Name         string
	Status       string
	Version      string
	Nodes        string
	HealthyNodes string
	APIServer    string
	Error        string
}

// buildComparisonData creates comparison data for a cluster
func buildComparisonData(k8sProvider *k8s.Provider, ctx context.Context, contextName string) ComparisonData {
	status, err := k8sProvider.GetClusterStatus(ctx, contextName)

	data := ComparisonData{
		Context: contextName,
	}

	if err != nil {
		data.Status = "ERROR"
		data.Error = err.Error()
		return data
	}

	data.Name = status.Name
	data.APIServer = status.APIServerURL

	if status.IsReachable {
		data.Status = "✅ Reachable"
		data.Version = status.Version
		data.Nodes = fmt.Sprintf("%d", status.NodeCount)
		data.HealthyNodes = fmt.Sprintf("%d", status.HealthyNodes)

		if status.HealthyNodes < status.NodeCount {
			data.Status = "⚠️  Degraded"
		}
	} else {
		data.Status = "❌ Unreachable"
		data.Error = status.Error
	}

	return data
}

// writeComparisonEntry writes a single comparison entry to the result
func writeComparisonEntry(result *strings.Builder, index int, comp ComparisonData) {
	fmt.Fprintf(result, "[%d] %s\n", index+1, comp.Context)
	fmt.Fprintf(result, "    Status: %s\n", comp.Status)

	if comp.Name != "" {
		fmt.Fprintf(result, "    Cluster: %s\n", comp.Name)
	}

	if comp.Version != "" {
		fmt.Fprintf(result, "    Version: %s\n", comp.Version)
	}

	if comp.Nodes != "" {
		fmt.Fprintf(result, "    Nodes: %s (Healthy: %s)\n", comp.Nodes, comp.HealthyNodes)
	}

	if comp.APIServer != "" {
		fmt.Fprintf(result, "    API Server: %s\n", comp.APIServer)
	}

	if comp.Error != "" {
		fmt.Fprintf(result, "    Error: %s\n", comp.Error)
	}

	result.WriteString("\n")
}

// countReachableClusters counts how many clusters are reachable
func countReachableClusters(comparisons []ComparisonData) int {
	count := 0
	for _, comp := range comparisons {
		if strings.Contains(comp.Status, "Reachable") || strings.Contains(comp.Status, "Degraded") {
			count++
		}
	}
	return count
}

func defineCompareClustersTool(k8sProvider *k8s.Provider, state *agentState) copilot.Tool {
	return copilot.DefineTool(
		toolCompareClusters,
		"Compare multiple Kubernetes clusters side by side, showing their versions, node counts, health status, and availability",
		func(params CompareClusterParams, inv copilot.ToolInvocation) (any, error) {
			if len(params.Contexts) == 0 {
				return nil, fmt.Errorf("at least one context must be provided")
			}

			ctx := context.Background()

			// Build comparison data for each cluster
			comparisons := make([]ComparisonData, 0, len(params.Contexts))
			for _, contextName := range params.Contexts {
				data := buildComparisonData(k8sProvider, ctx, contextName)
				comparisons = append(comparisons, data)
			}

			if isJSONOutput(state.outputFormat) {
				reachable := countReachableClusters(comparisons)
				return CompareClustersResult{
					Summary: CompareClustersSummary{
						Total:     len(comparisons),
						Reachable: reachable,
					},
					Clusters: comparisons,
				}, nil
			}

			var result strings.Builder
			fmt.Fprintf(&result, "Cluster Comparison (%d clusters)\n", len(params.Contexts))
			result.WriteString(strings.Repeat("=", 80) + "\n\n")

			// Write comparison entries
			for i, comp := range comparisons {
				writeComparisonEntry(&result, i, comp)
			}

			// Write summary
			reachable := countReachableClusters(comparisons)
			fmt.Fprintf(&result, "Summary: %d/%d clusters reachable\n", reachable, len(comparisons))

			return result.String(), nil
		},
	)
}

// CheckAllClustersParams defines no parameters for check_all_clusters
type CheckAllClustersParams struct{}

// CheckAllClustersSummary defines JSON output summary for check_all_clusters
type CheckAllClustersSummary struct {
	TotalClusters int `json:"total_clusters"`
	Reachable     int `json:"reachable"`
	FullyHealthy  int `json:"fully_healthy"`
	UnhealthyPods int `json:"unhealthy_pods"`
}

// CheckAllClustersResult defines JSON output for check_all_clusters
type CheckAllClustersResult struct {
	Summary  CheckAllClustersSummary `json:"summary"`
	Issues   []string                `json:"issues"`
	Clusters []*k8s.ClusterStatus    `json:"clusters"`
}

// clusterHealthSummary holds aggregated health metrics
type clusterHealthSummary struct {
	reachableCount     int
	healthyCount       int
	totalUnhealthyPods int
	issues             []string
}

// processReachableCluster processes health checks for a reachable cluster
func processReachableCluster(status *k8s.ClusterStatus, summary *clusterHealthSummary) {
	summary.reachableCount++
	hasIssues := false

	// Check node health
	if status.HealthyNodes < status.NodeCount && status.NodeCount > 0 {
		summary.issues = append(summary.issues, fmt.Sprintf("⚠️  %s: %d/%d nodes healthy", status.Context, status.HealthyNodes, status.NodeCount))
		hasIssues = true
	}

	// Check pod health
	if status.HealthyPods < status.PodCount && status.PodCount > 0 {
		unhealthyCount := status.PodCount - status.HealthyPods
		summary.totalUnhealthyPods += unhealthyCount
		summary.issues = append(summary.issues, fmt.Sprintf("⚠️  %s: %d/%d pods unhealthy", status.Context, unhealthyCount, status.PodCount))
		hasIssues = true
	}

	if !hasIssues && status.NodeCount > 0 {
		summary.healthyCount++
	}
}

// analyzeClusterHealth analyzes all cluster statuses and returns a summary
func analyzeClusterHealth(statuses []*k8s.ClusterStatus) clusterHealthSummary {
	summary := clusterHealthSummary{
		issues: []string{},
	}

	for _, status := range statuses {
		if status.IsReachable {
			processReachableCluster(status, &summary)
		} else {
			summary.issues = append(summary.issues, fmt.Sprintf("❌ %s: UNREACHABLE - %s", status.Context, status.Error))
		}
	}

	return summary
}

// writeCompactClusterStatus writes a single-line cluster status
func writeCompactClusterStatus(result *strings.Builder, status *k8s.ClusterStatus) {
	if !status.IsReachable {
		fmt.Fprintf(result, "❌ %s - DOWN (%s)\n", status.Context, status.Server)
	} else if status.HealthyNodes < status.NodeCount || status.HealthyPods < status.PodCount {
		fmt.Fprintf(result, "⚠️  %s - DEGRADED (nodes: %d/%d, pods: %d/%d)\n",
			status.Context, status.HealthyNodes, status.NodeCount, status.HealthyPods, status.PodCount)
	} else {
		fmt.Fprintf(result, "✅ %s - HEALTHY (nodes: %d, pods: %d, v%s)\n",
			status.Context, status.NodeCount, status.PodCount, status.Version)
	}
}

func defineCheckAllClustersTool(k8sProvider *k8s.Provider, state *agentState) copilot.Tool {
	return copilot.DefineTool(
		toolCheckAllClusters,
		"Check the status of ALL clusters in parallel for fast health monitoring. This is the most efficient way to get a complete overview of all clusters including their health status, node counts, version information, and any issues. Use this for initial health checks or when you need a full cluster overview. IMPORTANT: Present the tool output exactly as received - it already contains visual card formatting. Do NOT convert it to a table.",
		func(params CheckAllClustersParams, inv copilot.ToolInvocation) (any, error) {
			ctx := context.Background()
			statuses := k8sProvider.GetAllClusterStatuses(ctx)

			// Analyze cluster health
			summary := analyzeClusterHealth(statuses)

			if isJSONOutput(state.outputFormat) {
				return CheckAllClustersResult{
					Summary: CheckAllClustersSummary{
						TotalClusters: len(statuses),
						Reachable:     summary.reachableCount,
						FullyHealthy:  summary.healthyCount,
						UnhealthyPods: summary.totalUnhealthyPods,
					},
					Issues:   summary.issues,
					Clusters: statuses,
				}, nil
			}

			var result strings.Builder

			// Write compact cluster status
			for i, status := range statuses {
				if i > 0 {
					result.WriteString("\n")
				}
				writeCompactClusterStatus(&result, status)
			}

			// Write summary at the end
			result.WriteString("\n")
			fmt.Fprintf(&result, "📊 Summary: %d/%d reachable", summary.reachableCount, len(statuses))
			if summary.healthyCount > 0 {
				fmt.Fprintf(&result, ", %d healthy", summary.healthyCount)
			}
			if summary.totalUnhealthyPods > 0 {
				fmt.Fprintf(&result, ", %d unhealthy pods", summary.totalUnhealthyPods)
			}
			result.WriteString("\n")

			return result.String(), nil
		},
	)
}

// KubectlExecParams defines parameters for kubectl_exec
type KubectlExecParams struct {
	Context string   `json:"context" jsonschema:"The cluster context name to execute against (required)"`
	Args    []string `json:"args" jsonschema:"The kubectl command arguments (e.g., ['get', 'pods', '-n', 'default'])"`
}

// KubectlExecResult defines JSON output for kubectl_exec
type KubectlExecResult struct {
	Cluster  string `json:"cluster"`
	Context  string `json:"context"`
	Command  string `json:"command"`
	Output   string `json:"output"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Error    string `json:"error,omitempty"`
}

func defineKubectlExecTool(k8sProvider *k8s.Provider, state *agentState) copilot.Tool {
	return copilot.DefineTool(
		toolKubectlExec,
		"Execute kubectl commands against a specific Kubernetes cluster. Use this to perform operations like getting resources, scaling deployments, checking logs, describing resources, etc. Always specify the context and provide the kubectl arguments as an array.",
		func(params KubectlExecParams, inv copilot.ToolInvocation) (any, error) {
			return handleKubectlExec(k8sProvider, state, params)
		},
	)
}

func handleKubectlExec(k8sProvider *k8s.Provider, state *agentState, params KubectlExecParams) (any, error) {
	if err := validateKubectlExecParams(params); err != nil {
		return nil, err
	}

	cluster, err := getClusterForContext(k8sProvider, params.Context)
	if err != nil {
		return nil, err
	}
	clusterName := cluster.Name

	fullCommand, cmdArgs := buildKubectlCommand(params.Context, params.Args)
	isReadOnly := isReadOnlyCommand(params.Args)

	proceed, cancelResult, err := enforceExecutionMode(state, isReadOnly, clusterName, params.Context, fullCommand)
	if err != nil {
		return nil, err
	}
	if !proceed {
		return cancelResult, nil
	}

	printExecutionHeader(state, isReadOnly, fullCommand)

	output, execErr := runKubectlCommand(cmdArgs)
	if isJSONOutput(state.outputFormat) {
		return buildKubectlJSONResult(clusterName, params.Context, fullCommand, output, execErr)
	}
	return buildKubectlTextResult(clusterName, params.Context, fullCommand, output, execErr)
}

func validateKubectlExecParams(params KubectlExecParams) error {
	if params.Context == "" {
		return fmt.Errorf("context is required")
	}
	if len(params.Args) == 0 {
		return fmt.Errorf("kubectl arguments are required")
	}
	return nil
}

func getClusterForContext(k8sProvider *k8s.Provider, contextName string) (*k8s.ClusterInfo, error) {
	cluster, err := k8sProvider.GetClusterByContext(contextName)
	if err != nil {
		return nil, fmt.Errorf("cluster not found - context '%s' does not exist in your kubeconfig. Available contexts: %s",
			contextName, getAvailableContexts(k8sProvider))
	}
	return cluster, nil
}

func buildKubectlCommand(contextName string, args []string) (string, []string) {
	fullCommand := fmt.Sprintf("kubectl --context %s %s", contextName, strings.Join(args, " "))
	cmdArgs := append([]string{"--context", contextName}, args...)
	return fullCommand, cmdArgs
}

func enforceExecutionMode(state *agentState, isReadOnly bool, clusterName, contextName, fullCommand string) (bool, any, error) {
	if !isReadOnly && state.mode == ModeReadOnly {
		if isJSONOutput(state.outputFormat) {
			return false, fmt.Sprintf("write operation blocked in read-only mode. Cluster: %s (%s), Command: %s",
				clusterName, contextName, fullCommand), nil
		}
		// Offer the user a chance to switch to interactive mode rather than
		// surfacing the block as an error (which causes the model to retry or hallucinate).
		switched, err := offerModeSwitch(state, fullCommand)
		if err != nil {
			return false, nil, err
		}
		if !switched {
			return false, "Operation cancelled by user.", nil
		}
		// Fall through: mode is now ModeInteractive, so the block below will run.
	}

	if !isReadOnly && state.mode == ModeInteractive {
		proceed, err := confirmWriteOperation(state, fullCommand)
		if err != nil {
			return false, nil, err
		}
		if !proceed {
			return false, "Operation cancelled by user.", nil
		}
	}

	return true, nil, nil
}

// offerModeSwitch prompts the user to switch from read-only to interactive mode
// when a write operation is attempted. If the user agrees, state.mode is updated
// and true is returned so the caller can proceed with the normal confirmation flow.
func offerModeSwitch(state *agentState, fullCommand string) (bool, error) {
	resumeSpinner := pauseSpinner()
	defer resumeSpinner()

	fmt.Printf("\n%s🔒 Blocked:%s %s%s%s\n", colorRed, colorReset, colorBold, fullCommand, colorReset)
	fmt.Printf("%sThis write operation requires interactive mode.%s\n", colorYellow, colorReset)
	fmt.Print("Switch to interactive mode to proceed? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "yes" && response != "y" {
		fmt.Printf("\n%s❌ Operation cancelled by user%s\n\n", colorRed, colorReset)
		return false, nil
	}

	state.mode = ModeInteractive
	fmt.Printf("  %s●%s Switched to %s🔓 interactive%s mode\n\n", colorGreen, colorReset, colorGreen, colorReset)
	return true, nil
}

func confirmWriteOperation(state *agentState, fullCommand string) (bool, error) {
	resumeSpinner := pauseSpinner()
	defer resumeSpinner()

	if !isJSONOutput(state.outputFormat) {
		fmt.Printf("\n%s⚠️  Write Operation:%s %s%s%s\n", colorYellow, colorReset, colorBold, fullCommand, colorReset)
		fmt.Printf("%sThis will modify the cluster state.%s\n", colorYellow, colorReset)
	}
	fmt.Print("Do you want to proceed? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "yes" && response != "y" {
		if !isJSONOutput(state.outputFormat) {
			fmt.Printf("\n%s❌ Operation cancelled by user%s\n\n", colorRed, colorReset)
		}
		return false, nil
	}
	if !isJSONOutput(state.outputFormat) {
		fmt.Println()
	}

	return true, nil
}

func printExecutionHeader(state *agentState, isReadOnly bool, fullCommand string) {
	if isJSONOutput(state.outputFormat) {
		return
	}
	if isReadOnly {
		fmt.Printf("\r\033[K%s🔍 Executing:%s %s%s%s\n", colorCyan, colorReset, colorBold, fullCommand, colorReset)
	} else {
		fmt.Printf("\r\033[K%s⚡ Executing:%s %s%s%s\n", colorYellow, colorReset, colorBold, fullCommand, colorReset)
	}
}

func runKubectlCommand(cmdArgs []string) ([]byte, error) {
	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		return nil, fmt.Errorf("kubectl not found in PATH: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, kubectlPath, cmdArgs...)
	out, execErr := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("kubectl command timed out after 30s")
	}
	return out, execErr
}

func buildKubectlJSONResult(clusterName, contextName, fullCommand string, output []byte, execErr error) (any, error) {
	result := KubectlExecResult{
		Cluster: clusterName,
		Context: contextName,
		Command: fullCommand,
		Output:  string(output),
	}
	if execErr != nil {
		errMsg := execErr.Error()
		result.Error = errMsg
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			result.ExitCode = &exitCode
		}
		return result, fmt.Errorf("kubectl command failed on cluster %s (%s): %w", clusterName, contextName, execErr)
	}
	return result, nil
}

func buildKubectlTextResult(clusterName, contextName, fullCommand string, output []byte, execErr error) (string, error) {
	var result strings.Builder
	fmt.Fprintf(&result, "Cluster: %s (%s)\n", clusterName, contextName)
	fmt.Fprintf(&result, "Command: %s\n\n", fullCommand)

	if execErr != nil {
		fmt.Fprintf(&result, "❌ Error executing command on cluster %s:\n%v\n\n", clusterName, execErr)
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			fmt.Fprintf(&result, "Exit code: %d\n", exitErr.ExitCode())
		}
	}

	result.WriteString("Output:\n")
	result.WriteString(string(output))

	if execErr != nil {
		return result.String(), fmt.Errorf("kubectl command failed on cluster %s (%s): %w", clusterName, contextName, execErr)
	}

	return result.String(), nil
}

// SanitizeClusterParams defines parameters for sanitize_cluster
type SanitizeClusterParams struct {
	Context       string `json:"context" jsonschema:"The context name of the cluster to sanitize (from list_clusters)"`
	Namespace     string `json:"namespace,omitempty" jsonschema:"Optional: restrict the scan to a specific namespace; leave empty to scan all non-system namespaces"`
	IncludeSystem bool   `json:"include_system,omitempty" jsonschema:"If true, include system namespaces (kube-system, kube-public, kube-node-lease) in the scan"`
}

func defineSanitizeClusterTool(k8sProvider *k8s.Provider, state *agentState) copilot.Tool {
	return copilot.DefineTool(
		toolSanitizeCluster,
		"Lint all Deployments, StatefulSets, and DaemonSets in a cluster against Kubernetes best practices and security rules (CIS Benchmark, NSA/CISA guidelines). Returns a 0-100 score with an A-F grade, per-namespace breakdowns, and detailed findings per workload.",
		func(params SanitizeClusterParams, inv copilot.ToolInvocation) (any, error) {
			ctx := context.Background()
			report, err := k8sProvider.SanitizeCluster(ctx, params.Context, params.Namespace, params.IncludeSystem)
			if err != nil {
				return nil, fmt.Errorf("failed to sanitize cluster: %w", err)
			}

			if isJSONOutput(state.outputFormat) {
				return report, nil
			}

			return formatSanitizeResult(report), nil
		},
	)
}

// formatSanitizeResult formats a SanitizeResult as human-readable text
func formatSanitizeResult(report *k8s.SanitizeResult) string {
	var sb strings.Builder

	icon := sanitizeGradeIcon(report.Grade)
	fmt.Fprintf(&sb, "Cluster Sanitize Report: %s\n", report.Context)
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")
	fmt.Fprintf(&sb, "%s CLUSTER GRADE: %s  (score %d/100)\n", icon, report.Grade, report.Score)
	fmt.Fprintf(&sb, "   Scanned %d workload(s)  |  %d finding(s): %d critical, %d major, %d minor\n\n",
		report.TotalWorkloads, report.TotalFindings, report.CriticalCount, report.MajorCount, report.MinorCount)

	for _, ns := range report.Namespaces {
		nsIcon := sanitizeGradeIcon(ns.Grade)
		fmt.Fprintf(&sb, "%s namespace/%s  grade %s  score %d/100\n",
			nsIcon, ns.Namespace, ns.Grade, ns.Score)

		// Group findings by workload and sort worst-first
		byWorkload := groupFindingsByWorkload(ns.Findings)
		workloads := sortedWorkloadKeysByScore(byWorkload)
		for _, wl := range workloads {
			wf := byWorkload[wl]
			wScore := workloadScore(wf)
			wIcon := sanitizeScoreIcon(wScore)
			fmt.Fprintf(&sb, "  %s %s  score %d/100  (%d finding(s))\n",
				wIcon, stripNamespaceFromWorkload(wl), wScore, len(wf))
			writeSanitizeFindingGroup(&sb, "CRITICAL", filterSanitizeFindings(wf, k8s.SanitizeCritical))
			writeSanitizeFindingGroup(&sb, "MAJOR", filterSanitizeFindings(wf, k8s.SanitizeMajor))
			writeSanitizeFindingGroup(&sb, "MINOR", filterSanitizeFindings(wf, k8s.SanitizeMinor))
		}
		sb.WriteString("\n")
	}

	if report.TotalFindings == 0 {
		sb.WriteString("✅ No findings — all scanned workloads pass best-practice checks.\n")
	}
	return sb.String()
}

// sanitizeGradeIcon returns an emoji for a letter grade
func sanitizeGradeIcon(grade string) string {
	switch grade {
	case "A":
		return "🟢"
	case "B":
		return "🟡"
	case "C":
		return "🟠"
	case "D":
		return "🔴"
	default:
		return "💀"
	}
}

// sanitizeScoreIcon returns an emoji based on a raw score.
// ✅ is reserved for a perfect 100 (zero findings); anything less uses the grade icon.
func sanitizeScoreIcon(score int) string {
	if score == 100 {
		return "✅"
	}
	return sanitizeGradeIcon(workloadGrade(score))
}

// filterSanitizeFindings returns findings matching the given severity
func filterSanitizeFindings(findings []k8s.SanitizeFinding, sev k8s.SanitizeSeverity) []k8s.SanitizeFinding {
	out := make([]k8s.SanitizeFinding, 0)
	for _, f := range findings {
		if f.Severity == sev {
			out = append(out, f)
		}
	}
	return out
}

// groupFindingsByWorkload groups findings by their Workload ID
func groupFindingsByWorkload(findings []k8s.SanitizeFinding) map[string][]k8s.SanitizeFinding {
	m := make(map[string][]k8s.SanitizeFinding)
	for _, f := range findings {
		m[f.Workload] = append(m[f.Workload], f)
	}
	return m
}

// sortedWorkloadKeysByScore returns workload IDs sorted by score ascending (worst first), then alphabetically
func sortedWorkloadKeysByScore(m map[string][]k8s.SanitizeFinding) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		si := workloadScore(m[keys[i]])
		sj := workloadScore(m[keys[j]])
		if si != sj {
			return si < sj
		}
		return keys[i] < keys[j]
	})
	return keys
}

// workloadScore computes a 0-100 score from a workload's findings
func workloadScore(findings []k8s.SanitizeFinding) int {
	penalty := 0
	for _, f := range findings {
		penalty += f.Penalty
	}
	if penalty > 100 {
		return 0
	}
	return 100 - penalty
}

// workloadGrade returns a letter grade for a 0-100 score
func workloadGrade(score int) string {
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

// stripNamespaceFromWorkload converts "namespace/Kind/name" to "Kind/name"
func stripNamespaceFromWorkload(workload string) string {
	if idx := strings.Index(workload, "/"); idx >= 0 {
		return workload[idx+1:]
	}
	return workload
}

// writeSanitizeFindingGroup writes a labelled group of findings (workload already shown by caller)
func writeSanitizeFindingGroup(sb *strings.Builder, label string, findings []k8s.SanitizeFinding) {
	for _, f := range findings {
		containerPart := ""
		if f.Container != "" {
			containerPart = fmt.Sprintf(" [%s]", f.Container)
		}
		fmt.Fprintf(sb, "    [%s] %s%s: %s\n", label, f.RuleID, containerPart, f.Message)
	}
}

// ── MCP server management tools ─────────────────────────────────────────────

// MCPListServersParams defines no parameters for mcp_list_servers
type MCPListServersParams struct{}

// MCPListServersResult defines the output for mcp_list_servers
type MCPListServersResult struct {
	Servers []MCPServerConfig `json:"servers"`
}

func defineMCPListServersTool(state *agentState) copilot.Tool {
	return copilot.DefineTool(
		toolMCPListServers,
		"List all currently configured MCP (Model Context Protocol) servers",
		func(_ MCPListServersParams, _ copilot.ToolInvocation) (any, error) {
			servers, err := listMCPServers(state.mcpConfigPath)
			if err != nil {
				return nil, fmt.Errorf("failed to list MCP servers: %w", err)
			}
			if isJSONOutput(state.outputFormat) {
				return MCPListServersResult{Servers: servers}, nil
			}
			if len(servers) == 0 {
				return "No MCP servers configured. Use /mcp add <name> <url> or ask me to add one.", nil
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "Configured MCP servers (%d):\n\n", len(servers))
			for i, s := range servers {
				fmt.Fprintf(&sb, "  [%d] %s\n", i+1, s.Name)
				fmt.Fprintf(&sb, "      Type: %s\n", s.Type)
				fmt.Fprintf(&sb, "      URL:  %s\n", s.URL)
			}
			return sb.String(), nil
		},
	)
}

// MCPAddServerParams defines parameters for mcp_add_server
type MCPAddServerParams struct {
	Name string `json:"name" jsonschema:"Unique identifier for the MCP server (alphanumeric, hyphens, underscores)"`
	URL  string `json:"url"  jsonschema:"HTTP(S) endpoint URL of the MCP server"`
}

func defineMCPAddServerTool(state *agentState) copilot.Tool {
	return copilot.DefineTool(
		toolMCPAddServer,
		"Add or update an MCP (Model Context Protocol) server. The new server is persisted to the config file and will be active from the next session.",
		func(params MCPAddServerParams, _ copilot.ToolInvocation) (any, error) {
			entry := MCPServerConfig{Name: params.Name, Type: "http", URL: params.URL}
			if err := addMCPServer(state.mcpConfigPath, entry); err != nil {
				return nil, fmt.Errorf("failed to add MCP server: %w", err)
			}
			state.needsMCPReload = true
			return fmt.Sprintf("MCP server %q added (%s). The session will reload to connect to it.", params.Name, params.URL), nil
		},
	)
}

// MCPDeleteServerParams defines parameters for mcp_delete_server
type MCPDeleteServerParams struct {
	Name string `json:"name" jsonschema:"Name of the MCP server to remove"`
}

func defineMCPDeleteServerTool(state *agentState) copilot.Tool {
	return copilot.DefineTool(
		toolMCPDeleteServer,
		"Remove a configured MCP (Model Context Protocol) server by name",
		func(params MCPDeleteServerParams, _ copilot.ToolInvocation) (any, error) {
			if err := deleteMCPServer(state.mcpConfigPath, params.Name); err != nil {
				return nil, fmt.Errorf("failed to delete MCP server: %w", err)
			}
			state.needsMCPReload = true
			return fmt.Sprintf("MCP server %q removed. The session will reload.", params.Name), nil
		},
	)
}
