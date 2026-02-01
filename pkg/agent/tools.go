// Package agent provides the core Copilot agent functionality for Kubernetes cluster operations.
// This file contains tool definitions and related helpers.
package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/e9169/kopilot/pkg/k8s"
	copilot "github.com/github/copilot-sdk/go"
)

// defineTools creates all the Kubernetes-related tools for the agent
func defineTools(k8sProvider *k8s.Provider, state *agentState) []copilot.Tool {
	return []copilot.Tool{
		defineListClustersTool(k8sProvider, state),
		defineGetClusterStatusTool(k8sProvider, state),
		defineCompareClustersTool(k8sProvider, state),
		defineCheckAllClustersTool(k8sProvider, state),
		defineKubectlExecTool(k8sProvider, state),
	}
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
	result.WriteString("⚠️  Status: UNREACHABLE\n")
	fmt.Fprintf(result, "Error: %s\n\n", status.Error)
	fmt.Fprintf(result, "Context: %s\n", status.Context)
	fmt.Fprintf(result, "Server: %s\n", status.Server)
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
		"Get detailed status information for a specific Kubernetes cluster including reachability, nodes, version, and health metrics",
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

// writeHealthSummary writes the summary section to the result
func writeHealthSummary(result *strings.Builder, summary clusterHealthSummary, totalClusters int) {
	fmt.Fprintf(result, "Summary: %d/%d clusters reachable, %d fully healthy", summary.reachableCount, totalClusters, summary.healthyCount)
	if summary.totalUnhealthyPods > 0 {
		fmt.Fprintf(result, ", %d unhealthy pods across all clusters", summary.totalUnhealthyPods)
	}
	result.WriteString("\n\n")
}

// writeIssues writes the issues section to the result
func writeIssues(result *strings.Builder, issues []string) {
	if len(issues) > 0 {
		result.WriteString("⚠️  Issues Found:\n")
		for _, issue := range issues {
			fmt.Fprintf(result, "  %s\n", issue)
		}
		result.WriteString("\n")
	} else {
		result.WriteString("✅ All clusters are healthy!\n\n")
	}
}

// getClusterStatusIcon returns the appropriate icon for a cluster status
func getClusterStatusIcon(status *k8s.ClusterStatus) string {
	if !status.IsReachable {
		return "❌"
	}
	if status.HealthyNodes < status.NodeCount || status.HealthyPods < status.PodCount {
		return "⚠️"
	}
	return "✅"
}

// writeClusterDetails writes detailed information for a single cluster
func writeClusterDetails(result *strings.Builder, status *k8s.ClusterStatus) {
	statusIcon := getClusterStatusIcon(status)
	fmt.Fprintf(result, "\n%s %s\n", statusIcon, status.Context)

	if !status.IsReachable {
		writeUnreachableClusterDetails(result, status)
	} else {
		writeReachableClusterDetails(result, status)
	}
}

// writeUnreachableClusterDetails writes details for an unreachable cluster
func writeUnreachableClusterDetails(result *strings.Builder, status *k8s.ClusterStatus) {
	result.WriteString("   Status: UNREACHABLE\n")
	fmt.Fprintf(result, "   Server: %s\n", status.Server)
	fmt.Fprintf(result, "   Error: %s\n", status.Error)
}

// writeReachableClusterDetails writes details for a reachable cluster
func writeReachableClusterDetails(result *strings.Builder, status *k8s.ClusterStatus) {
	healthStatus := "HEALTHY"
	if status.HealthyNodes < status.NodeCount || status.HealthyPods < status.PodCount {
		healthStatus = "DEGRADED"
	}
	fmt.Fprintf(result, "   Status: %s\n", healthStatus)
	fmt.Fprintf(result, "   Version: %s\n", status.Version)
	fmt.Fprintf(result, "   Nodes: %d total, %d healthy\n", status.NodeCount, status.HealthyNodes)
	fmt.Fprintf(result, "   Pods: %d total, %d healthy\n", status.PodCount, status.HealthyPods)
	fmt.Fprintf(result, "   Server: %s\n", status.APIServerURL)
	if status.Namespace != "" {
		fmt.Fprintf(result, "   Default Namespace: %s\n", status.Namespace)
	}

	writeUnhealthyPods(result, status.UnhealthyPods)
}

// writeUnhealthyPods writes the list of unhealthy pods
func writeUnhealthyPods(result *strings.Builder, pods []k8s.PodInfo) {
	if len(pods) == 0 {
		return
	}

	fmt.Fprintf(result, "   Unhealthy Pods (%d):\n", len(pods))
	for _, pod := range pods {
		fmt.Fprintf(result, "     - %s/%s: %s", pod.Namespace, pod.Name, pod.Status)
		if pod.Reason != "" {
			fmt.Fprintf(result, " (%s)", pod.Reason)
		}
		if pod.Restarts > 0 {
			fmt.Fprintf(result, " [%d restarts]", pod.Restarts)
		}
		result.WriteString("\n")
	}
}

func defineCheckAllClustersTool(k8sProvider *k8s.Provider, state *agentState) copilot.Tool {
	return copilot.DefineTool(
		toolCheckAllClusters,
		"Check the status of ALL clusters in parallel for fast health monitoring. This is the most efficient way to get a complete overview of all clusters including their health status, node counts, version information, and any issues. Use this for initial health checks or when you need a full cluster overview.",
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
			fmt.Fprintf(&result, "Health Check Results for %d Clusters\n", len(statuses))
			result.WriteString(strings.Repeat("=", 80) + "\n\n")

			// Write summary
			writeHealthSummary(&result, summary, len(statuses))

			// Write issues
			writeIssues(&result, summary.issues)

			// Write detailed status for each cluster
			result.WriteString("Cluster Details:\n")
			result.WriteString(strings.Repeat("-", 80) + "\n")

			for _, status := range statuses {
				writeClusterDetails(&result, status)
			}

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

	proceed, cancelResult, err := enforceExecutionMode(state, isReadOnly, clusterName, params.Context, fullCommand, params.Args[0])
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

func enforceExecutionMode(state *agentState, isReadOnly bool, clusterName, contextName, fullCommand, operation string) (bool, any, error) {
	if !isReadOnly && state.mode == ModeReadOnly {
		if !isJSONOutput(state.outputFormat) {
			fmt.Printf("\n%s🔒 Blocked:%s %s%s%s\n", colorRed, colorReset, colorBold, fullCommand, colorReset)
		}
		return false, nil, fmt.Errorf("write operation blocked in read-only mode.\n\nCluster: %s (%s)\nCommand: %s\nOperation: %s\n\nThis command would modify cluster state. Use /interactive to enable write operations with confirmation",
			clusterName, contextName, fullCommand, operation)
	}

	if !isReadOnly && state.mode == ModeInteractive {
		proceed, err := confirmWriteOperation(state, fullCommand)
		if err != nil {
			return false, nil, err
		}
		if !proceed {
			return false, "Operation cancelled by user", nil
		}
	}

	return true, nil, nil
}

func confirmWriteOperation(state *agentState, fullCommand string) (bool, error) {
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
		fmt.Printf("\n%s🔍 Executing:%s %s%s%s\n\n", colorCyan, colorReset, colorBold, fullCommand, colorReset)
		return
	}
	fmt.Printf("\n%s⚡ Executing:%s %s%s%s\n\n", colorYellow, colorReset, colorBold, fullCommand, colorReset)
}

func runKubectlCommand(cmdArgs []string) ([]byte, error) {
	cmd := exec.Command("kubectl", cmdArgs...)
	return cmd.CombinedOutput()
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
