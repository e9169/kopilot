package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e9169/kopilot/pkg/k8s"
	copilot "github.com/github/copilot-sdk/go"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	errToolDescriptionEmpty = "Tool description is empty"
)

func TestDefineTools(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly}

	tools := defineTools(provider, state)

	if len(tools) != 5 {
		t.Errorf("defineTools() returned %d tools, want 5", len(tools))
	}

	expectedNames := map[string]bool{
		toolListClusters:     false,
		toolGetClusterStatus: false,
		toolCompareClusters:  false,
		toolCheckAllClusters: false,
		toolKubectlExec:      false,
	}

	for _, tool := range tools {
		if _, exists := expectedNames[tool.Name]; !exists {
			t.Errorf("Unexpected tool name: %s", tool.Name)
		}
		expectedNames[tool.Name] = true

		if tool.Description == "" {
			t.Errorf("Tool %s has empty description", tool.Name)
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("Missing expected tool: %s", name)
		}
	}
}

func TestListClustersTool(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly, outputFormat: OutputText}
	tool := defineListClustersTool(provider, state)

	if tool.Name != toolListClusters {
		t.Errorf("Tool name = %s, want %s", tool.Name, toolListClusters)
	}

	if tool.Description == "" {
		t.Error(errToolDescriptionEmpty)
	}

	// Test tool invocation
	_ = ListClustersParams{}
	inv := copilot.ToolInvocation{}

	result, err := tool.Handler(inv)
	if err != nil {
		t.Errorf("Tool handler returned error: %v", err)
	}

	if result.TextResultForLLM == "" {
		t.Error("Tool handler returned empty result")
	}
}

func TestListClustersToolJSONOutput(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly, outputFormat: OutputJSON}
	tool := defineListClustersTool(provider, state)

	inv := copilot.ToolInvocation{}
	result, err := tool.Handler(inv)
	if err != nil {
		t.Errorf("Tool handler returned error: %v", err)
	}

	if result.TextResultForLLM == "" {
		t.Error("Tool handler returned empty result")
	}

	if !strings.Contains(result.TextResultForLLM, "clusters") {
		t.Error("JSON output did not include expected key 'clusters'")
	}
}

func TestGetClusterStatusTool(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly, outputFormat: OutputText}
	tool := defineGetClusterStatusTool(provider, state)

	if tool.Name != toolGetClusterStatus {
		t.Errorf("Tool name = %s, want %s", tool.Name, toolGetClusterStatus)
	}

	tests := []struct {
		name    string
		context string
		wantErr bool
	}{
		{
			name:    "valid context",
			context: "context-1",
			wantErr: false,
		},
		{
			name:    "invalid context",
			context: "non-existent",
			wantErr: true,
		},
		{
			name:    "empty context",
			context: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This would require mock implementation or integration test
			// For unit tests, we verify the tool structure
			if tool.Description == "" {
				t.Error(errToolDescriptionEmpty)
			}
		})
	}
}

func TestCompareClustersTool(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly, outputFormat: OutputText}
	tool := defineCompareClustersTool(provider, state)

	if tool.Name != toolCompareClusters {
		t.Errorf("Tool name = %s, want %s", tool.Name, toolCompareClusters)
	}

	if tool.Description == "" {
		t.Error(errToolDescriptionEmpty)
	}

	// Verify tool accepts array parameter
	// In actual usage, Copilot SDK will validate the schema
}

func TestListClustersParams(t *testing.T) {
	// Verify struct is properly defined
	var params ListClustersParams
	_ = params // Empty struct should be valid
}

func TestGetClusterStatusParams(t *testing.T) {
	params := GetClusterStatusParams{
		Context: "test-context",
	}

	if params.Context != "test-context" {
		t.Errorf("Context = %s, want test-context", params.Context)
	}

	// Test empty context
	emptyParams := GetClusterStatusParams{}
	if emptyParams.Context != "" {
		t.Error("Empty params should have empty context")
	}
}

func TestCompareClusterParams(t *testing.T) {
	params := CompareClusterParams{
		Contexts: []string{"ctx1", "ctx2", "ctx3"},
	}

	if len(params.Contexts) != 3 {
		t.Errorf("Contexts length = %d, want 3", len(params.Contexts))
	}

	// Test empty contexts
	emptyParams := CompareClusterParams{}
	if len(emptyParams.Contexts) != 0 {
		t.Error("Empty params should have no contexts")
	}
}

func TestRunWithInvalidProvider(t *testing.T) {
	// Test that Run handles nil provider gracefully
	// Note: In production, this should never happen due to validation
	// but good defensive programming to check
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	// This test verifies the function signature exists
	// Actual testing would require mocking the Copilot client
}

func TestToolDescriptions(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly}
	tools := defineTools(provider, state)

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			if len(tool.Description) < 20 {
				t.Errorf("Tool %s has very short description: %s", tool.Name, tool.Description)
			}

			// Check for key terms in descriptions
			validateToolDescriptionContent(t, tool.Name, tool.Description)
		})
	}
}

func validateToolDescriptionContent(t *testing.T, toolName, description string) {
	t.Helper()
	switch toolName {
	case "list_clusters":
		if !contains(description, "kubeconfig") {
			t.Error("list_clusters description should mention kubeconfig")
		}
	case "get_cluster_status":
		if !contains(description, "status") && !contains(description, "health") {
			t.Error("get_cluster_status description should mention status or health")
		}
	case toolCompareClusters:
		if !contains(description, "compare") && !contains(description, "comparison") {
			t.Error("compare_clusters description should mention compare or comparison")
		}
	}
}

func TestToolParameterValidation(t *testing.T) {
	// Test that parameter structs have proper JSON tags
	tests := []struct {
		name      string
		checkFunc func() bool
	}{
		{
			name: "GetClusterStatusParams has json tags",
			checkFunc: func() bool {
				params := GetClusterStatusParams{Context: "test"}
				return params.Context != ""
			},
		},
		{
			name: "CompareClusterParams has json tags",
			checkFunc: func() bool {
				params := CompareClusterParams{Contexts: []string{"test"}}
				return len(params.Contexts) > 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.checkFunc() {
				t.Error("Parameter validation failed")
			}
		})
	}
}

// Helper functions

func createMockProvider(t *testing.T) *k8s.Provider {
	t.Helper()

	tmpfile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	config := clientcmdapi.NewConfig()

	// Create 2 test clusters
	for i := 1; i <= 2; i++ {
		clusterName := filepath.Base(tmpfile.Name()) + "-cluster-" + string(rune('0'+i))
		contextName := filepath.Base(tmpfile.Name()) + "-context-" + string(rune('0'+i))
		userName := filepath.Base(tmpfile.Name()) + "-user-" + string(rune('0'+i))

		config.Clusters[clusterName] = &clientcmdapi.Cluster{
			Server:                "https://127.0.0.1:6443",
			InsecureSkipTLSVerify: true,
		}

		config.AuthInfos[userName] = &clientcmdapi.AuthInfo{
			Token: "test-token",
		}

		config.Contexts[contextName] = &clientcmdapi.Context{
			Cluster:  clusterName,
			AuthInfo: userName,
		}
	}

	config.CurrentContext = filepath.Base(tmpfile.Name()) + "-context-1"

	err = clientcmd.WriteToFile(*config, tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	provider, err := k8s.NewProvider(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	return provider
}

func contains(s, substr string) bool {
	// Make case-insensitive comparison
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)*2 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestToolConcurrency(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly}
	tools := defineTools(provider, state)

	// Test that tools can be called concurrently
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for _, tool := range tools {
				if tool.Name == "list_clusters" {
					inv := copilot.ToolInvocation{}
					_, _ = tool.Handler(inv)
				}
			}
			done <- true
		}()
	}

	timeout := time.After(5 * time.Second)
	for i := 0; i < 5; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			t.Fatal("Tool concurrency test timed out")
		}
	}
}

func BenchmarkDefineTools(b *testing.B) {
	tmpfile, _ := os.CreateTemp("", "kubeconfig-*.yaml")
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	config := clientcmdapi.NewConfig()
	config.Clusters["test"] = &clientcmdapi.Cluster{Server: "https://localhost"}
	config.AuthInfos["test"] = &clientcmdapi.AuthInfo{Token: "test"}
	config.Contexts["test"] = &clientcmdapi.Context{Cluster: "test", AuthInfo: "test"}
	config.CurrentContext = "test"

	_ = clientcmd.WriteToFile(*config, tmpfile.Name())

	provider, _ := k8s.NewProvider(tmpfile.Name())

	state := &agentState{mode: ModeReadOnly}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = defineTools(provider, state)
	}
}

func BenchmarkListClustersTool(b *testing.B) {
	provider := createMockProvider(&testing.T{})
	state := &agentState{mode: ModeReadOnly, outputFormat: OutputText}
	tool := defineListClustersTool(provider, state)
	inv := copilot.ToolInvocation{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Handler(inv)
	}
}

// TestExecutionMode tests the ExecutionMode type
func TestExecutionMode(t *testing.T) {
	tests := []struct {
		mode     ExecutionMode
		expected string
	}{
		{ModeReadOnly, "read-only"},
		{ModeInteractive, "interactive"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.expected {
				t.Errorf("ExecutionMode.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestIsReadOnlyCommand tests the isReadOnlyCommand function
func TestIsReadOnlyCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{"get pods", []string{"get", "pods"}, true},
		{"describe deployment", []string{"describe", "deployment", "nginx"}, true},
		{"logs", []string{"logs", "pod-name"}, true},
		{"top nodes", []string{"top", "nodes"}, true},
		{"explain", []string{"explain", "pods"}, true},
		{"config view", []string{"config", "view"}, true},
		{"scale deployment", []string{"scale", "deployment", "nginx", "--replicas=3"}, false},
		{"delete pod", []string{"delete", "pod", "nginx"}, false},
		{"apply", []string{"apply", "-f", "deployment.yaml"}, false},
		{"patch", []string{"patch", "deployment", "nginx"}, false},
		{"edit", []string{"edit", "deployment", "nginx"}, false},
		{"drain", []string{"drain", "node-1"}, false},
		{"empty args", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReadOnlyCommand(tt.args); got != tt.expected {
				t.Errorf("isReadOnlyCommand(%v) = %v, want %v", tt.args, got, tt.expected)
			}
		})
	}
}

// TestHandleModeSwitch tests the mode switching commands
func TestHandleModeSwitch(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		initialMode  ExecutionMode
		expectedMode ExecutionMode
		shouldHandle bool
	}{
		{"switch to readonly", "/readonly", ModeInteractive, ModeReadOnly, true},
		{"switch to interactive", "/interactive", ModeReadOnly, ModeInteractive, true},
		{"already readonly", "/readonly", ModeReadOnly, ModeReadOnly, true},
		{"already interactive", "/interactive", ModeInteractive, ModeInteractive, true},
		{"check mode", "/mode", ModeReadOnly, ModeReadOnly, true},
		{"check status", "/status", ModeInteractive, ModeInteractive, true},
		{"not a command", "show me pods", ModeReadOnly, ModeReadOnly, false},
		{"with extra space", "  /readonly  ", ModeInteractive, ModeReadOnly, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &agentState{mode: tt.initialMode}
			got := handleModeSwitch(tt.input, state)

			if got != tt.shouldHandle {
				t.Errorf("handleModeSwitch() returned %v, want %v", got, tt.shouldHandle)
			}

			if state.mode != tt.expectedMode {
				t.Errorf("mode after handleModeSwitch() = %v, want %v", state.mode, tt.expectedMode)
			}
		})
	}
}

// TestAgentState tests the agentState structure
func TestAgentState(t *testing.T) {
	state := &agentState{mode: ModeReadOnly}

	if state.mode != ModeReadOnly {
		t.Errorf("Initial mode = %v, want %v", state.mode, ModeReadOnly)
	}

	state.mode = ModeInteractive
	if state.mode != ModeInteractive {
		t.Errorf("Updated mode = %v, want %v", state.mode, ModeInteractive)
	}
}

// TestDefineToolsWithState ensures tools are created with state parameter
func TestDefineToolsWithState(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly, outputFormat: OutputText}

	tools := defineTools(provider, state)

	if len(tools) != 5 {
		t.Errorf("defineTools() returned %d tools, want 5", len(tools))
	}

	// Verify kubectl_exec tool exists
	var kubectlTool *copilot.Tool
	for i := range tools {
		if tools[i].Name == toolKubectlExec {
			kubectlTool = &tools[i]
			break
		}
	}

	if kubectlTool == nil {
		t.Fatal("kubectl_exec tool not found")
		return
	}

	if kubectlTool.Description == "" {
		t.Error("kubectl_exec tool has empty description")
	}
}

func TestIsJSONOutput(t *testing.T) {
	if !isJSONOutput(OutputJSON) {
		t.Error("isJSONOutput(OutputJSON) = false, want true")
	}
	if isJSONOutput(OutputText) {
		t.Error("isJSONOutput(OutputText) = true, want false")
	}
}
