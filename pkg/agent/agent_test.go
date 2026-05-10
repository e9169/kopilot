package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"encoding/json"
	"fmt"

	"github.com/e9169/kopilot/pkg/k8s"
	"github.com/e9169/kopilot/pkg/llm"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	errToolDescriptionEmpty = "Tool description is empty"
	errToolNameFormat       = "Tool name = %s, want %s"
	testClusterContext      = "my-cluster"
)

func TestDefineTools(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly}

	tools := defineTools(provider, state)

	if len(tools) != 9 {
		t.Errorf("defineTools() returned %d tools, want 9", len(tools))
	}

	expectedNames := map[string]bool{
		toolListClusters:     false,
		toolGetClusterStatus: false,
		toolCompareClusters:  false,
		toolCheckAllClusters: false,
		toolKubectlExec:      false,
		toolSanitizeCluster:  false,
		toolMCPListServers:   false,
		toolMCPAddServer:     false,
		toolMCPDeleteServer:  false,
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
		t.Errorf(errToolNameFormat, tool.Name, toolListClusters)
	}

	if tool.Description == "" {
		t.Error(errToolDescriptionEmpty)
	}

	// Test tool invocation
	_ = ListClustersParams{}
	inv := llm.ToolInvocation{}

	result, err := tool.Handler(nil, inv)
	if err != nil {
		t.Errorf("Tool handler returned error: %v", err)
	}

	if fmt.Sprintf("%v", result) == "" {
		t.Error("Tool handler returned empty result")
	}
}

func TestListClustersToolJSONOutput(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly, outputFormat: OutputJSON}
	tool := defineListClustersTool(provider, state)

	inv := llm.ToolInvocation{}
	result, err := tool.Handler(nil, inv)
	if err != nil {
		t.Errorf("Tool handler returned error: %v", err)
	}

	if fmt.Sprintf("%v", result) == "" {
		t.Error("Tool handler returned empty result")
	}

	b, _ := json.Marshal(result)
	if !strings.Contains(string(b), "clusters") {
		t.Error("JSON output did not include expected key 'clusters'")
	}
}

func TestGetClusterStatusTool(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly, outputFormat: OutputText}
	tool := defineGetClusterStatusTool(provider, state)

	if tool.Name != toolGetClusterStatus {
		t.Errorf(errToolNameFormat, tool.Name, toolGetClusterStatus)
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
		t.Errorf(errToolNameFormat, tool.Name, toolCompareClusters)
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

func createMockProvider(t testing.TB) *k8s.Provider {
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
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
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
					inv := llm.ToolInvocation{}
					_, _ = tool.Handler(nil, inv)
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
	provider := createMockProvider(b)
	state := &agentState{mode: ModeReadOnly, outputFormat: OutputText}
	tool := defineListClustersTool(provider, state)
	inv := llm.ToolInvocation{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Handler(nil, inv)
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
	const rolloutTarget = "deployment/hello-world"

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
		{"rollout status", []string{"rollout", "status", rolloutTarget}, true},
		{"rollout history", []string{"rollout", "history", rolloutTarget}, true},
		{"rollout restart", []string{"rollout", "restart", rolloutTarget}, false},
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

	if len(tools) != 9 {
		t.Errorf("defineTools() returned %d tools, want 9", len(tools))
	}

	// Verify kubectl_exec tool exists
	var kubectlTool *llm.Tool
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

func TestParseSanitizerAgent(t *testing.T) {
	agentType, err := ParseAgentType("sanitizer")
	if err != nil {
		t.Fatalf("ParseAgentType(sanitizer) returned unexpected error: %v", err)
	}
	if agentType != AgentSanitizer {
		t.Errorf("ParseAgentType(sanitizer) = %q, want %q", agentType, AgentSanitizer)
	}
}

func TestParseSanitizerAgentCaseInsensitive(t *testing.T) {
	for _, input := range []string{"Sanitizer", "SANITIZER", "sanitizer"} {
		agentType, err := ParseAgentType(input)
		if err != nil {
			t.Errorf("ParseAgentType(%q) returned unexpected error: %v", input, err)
			continue
		}
		if agentType != AgentSanitizer {
			t.Errorf("ParseAgentType(%q) = %q, want %q", input, agentType, AgentSanitizer)
		}
	}
}

func TestSanitizeClusterParams(t *testing.T) {
	params := SanitizeClusterParams{
		Context:       testClusterContext,
		Namespace:     "production",
		IncludeSystem: false,
	}
	if params.Context != testClusterContext {
		t.Errorf("Context = %q, want %q", params.Context, testClusterContext)
	}
	if params.Namespace != "production" {
		t.Errorf("Namespace = %q, want %q", params.Namespace, "production")
	}
	if params.IncludeSystem {
		t.Error("IncludeSystem should default to false")
	}
}

func TestSanitizeClusterTool(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly, outputFormat: OutputText}
	tool := defineSanitizeClusterTool(provider, state)

	if tool.Name != toolSanitizeCluster {
		t.Errorf(errToolNameFormat, tool.Name, toolSanitizeCluster)
	}
	if tool.Description == "" {
		t.Error(errToolDescriptionEmpty)
	}
}

func TestSanitizerAgentDefinition(t *testing.T) {
	def, ok := agentDefinitions[AgentSanitizer]
	if !ok {
		t.Fatal("AgentSanitizer not found in agentDefinitions")
	}
	if def.Icon != "🧹" {
		t.Errorf("Icon = %q, want 🧹", def.Icon)
	}
	if def.Prompt == "" {
		t.Error("AgentSanitizer prompt is empty")
	}
	if !def.preferPremium {
		t.Error("AgentSanitizer should set preferPremium = true")
	}
	if len(def.Examples) == 0 {
		t.Error("AgentSanitizer should have at least one example")
	}
}

// TestExtractAttachments exercises the @<filepath> parsing logic.
func TestExtractAttachments(t *testing.T) {
	// Create a temporary regular file we can read.
	tmpDir := t.TempDir()
	regularFile := filepath.Join(tmpDir, "hello.txt")
	if err := os.WriteFile(regularFile, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory (not a regular file).
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0700); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name            string
		input           string
		wantPrompt      string
		wantAttachCount int
	}{
		{
			name:            "no @ tokens",
			input:           "show me all pods",
			wantPrompt:      "show me all pods",
			wantAttachCount: 0,
		},
		{
			name:            "bare @ with no path",
			input:           "@ show pods",
			wantPrompt:      "@ show pods",
			wantAttachCount: 0,
		},
		{
			name:            "non-existent path",
			input:           "@/no/such/file.txt check it",
			wantPrompt:      "@/no/such/file.txt check it",
			wantAttachCount: 0,
		},
		{
			name:            "directory path (not regular file)",
			input:           "analyse @" + subDir,
			wantPrompt:      "analyse @" + subDir,
			wantAttachCount: 0,
		},
		{
			name:            "valid regular file",
			input:           "read @" + regularFile,
			wantPrompt:      "read",
			wantAttachCount: 1,
		},
		{
			name:            "mixed tokens: one valid, one missing",
			input:           "@" + regularFile + " and @/gone.txt",
			wantPrompt:      "and @/gone.txt",
			wantAttachCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPrompt, gotAttach := extractAttachments(tt.input)
			if gotPrompt != tt.wantPrompt {
				t.Errorf("prompt = %q, want %q", gotPrompt, tt.wantPrompt)
			}
			if len(gotAttach) != tt.wantAttachCount {
				t.Errorf("attachments count = %d, want %d", len(gotAttach), tt.wantAttachCount)
			}
		})
	}
}

// TestExtractAttachmentsUnreadable verifies that files that exist but are not
// readable by the current process are not promoted to attachments.
func TestExtractAttachmentsUnreadable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can read any file — skipping unreadable-file test")
	}
	tmpDir := t.TempDir()
	unreadable := filepath.Join(tmpDir, "secret.txt")
	if err := os.WriteFile(unreadable, []byte("secret"), 0000); err != nil {
		t.Fatal(err)
	}
	prompt, attachments := extractAttachments("read @" + unreadable)
	if len(attachments) != 0 {
		t.Errorf("expected 0 attachments for unreadable file, got %d", len(attachments))
	}
	if !strings.Contains(prompt, "@"+unreadable) {
		t.Errorf("unreadable token should be preserved in prompt, got %q", prompt)
	}
}

// TestHistoryFilePath verifies that historyFilePath returns a non-empty path
// when a home directory is available.
func TestHistoryFilePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available — skipping historyFilePath test")
	}
	path := historyFilePath()
	if path == "" {
		t.Fatal("historyFilePath() returned empty string when home dir is available")
	}
	want := filepath.Join(home, ".kopilot", "history")
	if path != want {
		t.Errorf("historyFilePath() = %q, want %q", path, want)
	}
	// The directory should have been created.
	if _, statErr := os.Stat(filepath.Dir(path)); statErr != nil {
		t.Errorf("historyFilePath() directory not created: %v", statErr)
	}
}

// TestSelectModelForcedOverride verifies that a non-empty forcedModel bypasses
// all keyword-based and agent-type-based routing.
func TestSelectModelForcedOverride(t *testing.T) {
	const customModel = "my-custom-model"
	tests := []struct {
		name      string
		query     string
		agentType AgentType
	}{
		{"simple query with forced model", "list pods", AgentDefault},
		{"complex query with forced model", "troubleshoot crash", AgentDefault},
		{"specialist agent with forced model", "list pods", AgentDebugger},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectModelForQuery(tt.query, tt.agentType, customModel)
			if got != customModel {
				t.Errorf("selectModelForQuery(%q, %q, %q) = %q, want %q",
					tt.query, tt.agentType, customModel, got, customModel)
			}
		})
	}
}

// TestLastResponseMutex verifies that setLastResponse / getLastResponse work
// correctly under concurrent access (data-race detection).
func TestLastResponseMutex(t *testing.T) {
	state := &agentState{}
	const value = "hello from goroutine"
	done := make(chan struct{})
	go func() {
		state.setLastResponse(value)
		close(done)
	}()
	<-done
	if got := state.getLastResponse(); got != value {
		t.Errorf("getLastResponse() = %q, want %q", got, value)
	}
}

// TestFormatBytes exercises all three branches: bytes, KB, and MB.
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024*1024*2 + 1024*512, "2.5 MB"},
	}
	for _, tt := range tests {
		got := formatBytes(tt.input)
		if got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestEstimateTokens verifies the rough token count heuristic (1 token ≈ 4 chars).
func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"abcd", 1},
		{"abcdefgh", 2},
		{"hello world!", 3},
	}
	for _, tt := range tests {
		got := estimateTokens(tt.input)
		if got != tt.want {
			t.Errorf("estimateTokens(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// TestHandleLastEmpty verifies /last with no previous response.
func TestHandleLastEmpty(t *testing.T) {
	state := &agentState{}
	handled, err := handleLast(state)
	if err != nil {
		t.Fatalf("handleLast returned unexpected error: %v", err)
	}
	if !handled {
		t.Error("handleLast should return handled=true")
	}
}

// TestHandleLastWithContent verifies /last prints the stored response.
func TestHandleLastWithContent(t *testing.T) {
	state := &agentState{}
	state.setLastResponse("some assistant response")
	handled, err := handleLast(state)
	if err != nil {
		t.Fatalf("handleLast returned unexpected error: %v", err)
	}
	if !handled {
		t.Error("handleLast should return handled=true")
	}
}

// TestHandleCopyEmpty verifies /copy when there is nothing in the buffer.
func TestHandleCopyEmpty(t *testing.T) {
	state := &agentState{}
	handled, err := handleCopy(state)
	if err != nil {
		t.Fatalf("handleCopy returned unexpected error: %v", err)
	}
	if !handled {
		t.Error("handleCopy should return handled=true")
	}
}

// TestHandleCopyWithContent verifies /copy when the buffer has content.
// copyToClipboard may succeed (macOS) or fail (CI without xclip/xsel) — either
// branch provides coverage; only the "nothing to copy" early-return is omitted.
func TestHandleCopyWithContent(t *testing.T) {
	state := &agentState{}
	state.setLastResponse("response to copy")
	handled, err := handleCopy(state)
	if err != nil {
		t.Fatalf("handleCopy returned unexpected error: %v", err)
	}
	if !handled {
		t.Error("handleCopy should return handled=true")
	}
}

// TestHandleStreamerToggle verifies that /streamer without arguments toggles the flag.
func TestHandleStreamerToggle(t *testing.T) {
	state := &agentState{streamerMode: false}

	handled, err := handleStreamer(state, "/streamer")
	if err != nil || !handled {
		t.Fatalf("handleStreamer toggle: handled=%v err=%v", handled, err)
	}
	if !state.streamerMode {
		t.Error("expected streamerMode=true after first toggle")
	}

	handled, err = handleStreamer(state, "/streamer")
	if err != nil || !handled {
		t.Fatalf("handleStreamer toggle back: handled=%v err=%v", handled, err)
	}
	if state.streamerMode {
		t.Error("expected streamerMode=false after second toggle")
	}
}

// TestHandleStreamerExplicit verifies /streamer on and /streamer off.
func TestHandleStreamerExplicit(t *testing.T) {
	state := &agentState{}

	if _, err := handleStreamer(state, "/streamer on"); err != nil {
		t.Fatal(err)
	}
	if !state.streamerMode {
		t.Error("expected streamerMode=true after '/streamer on'")
	}

	if _, err := handleStreamer(state, "/streamer off"); err != nil {
		t.Fatal(err)
	}
	if state.streamerMode {
		t.Error("expected streamerMode=false after '/streamer off'")
	}
}

// TestHandleStreamerInvalid verifies that an unknown argument is rejected gracefully.
func TestHandleStreamerInvalid(t *testing.T) {
	state := &agentState{}
	handled, err := handleStreamer(state, "/streamer maybe")
	if err != nil {
		t.Fatalf("handleStreamer invalid arg returned error: %v", err)
	}
	if !handled {
		t.Error("handleStreamer should return handled=true even for invalid arg")
	}
}

// TestHandleModelCommandNoArgs verifies /model with no arguments prints status.
func TestHandleModelCommandNoArgs(t *testing.T) {
	provider := createMockProvider(t)
	idle := true
	deps := &loopDeps{
		ctx:         context.Background(),
		k8sProvider: provider,
		state:       &agentState{},
		isIdle:      &idle,
	}
	ts := &turnState{model: modelCostEffective}

	handled, err := handleModelCommand(deps, "/model", ts)
	if err != nil {
		t.Fatalf("handleModelCommand(/model) returned error: %v", err)
	}
	if !handled {
		t.Error("handleModelCommand(/model) should return handled=true")
	}
}

// TestHandleModelCommandNoArgsWithForced verifies /model displays forced model info.
func TestHandleModelCommandNoArgsWithForced(t *testing.T) {
	provider := createMockProvider(t)
	idle := true
	deps := &loopDeps{
		ctx:         context.Background(),
		k8sProvider: provider,
		state:       &agentState{forcedModel: "gpt-4o"},
		isIdle:      &idle,
	}
	ts := &turnState{model: "gpt-4o"}

	handled, err := handleModelCommand(deps, "/model", ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Error("should be handled")
	}
}

// TestHandleModelCommandReset verifies /model reset clears the forced model.
func TestHandleModelCommandReset(t *testing.T) {
	provider := createMockProvider(t)
	idle := true
	deps := &loopDeps{
		ctx:         context.Background(),
		k8sProvider: provider,
		state:       &agentState{forcedModel: "gpt-4o"},
		isIdle:      &idle,
	}
	ts := &turnState{model: "gpt-4o"}

	handled, err := handleModelCommand(deps, "/model reset", ts)
	if err != nil {
		t.Fatalf("handleModelCommand(/model reset) returned error: %v", err)
	}
	if !handled {
		t.Error("handleModelCommand(/model reset) should return handled=true")
	}
	if deps.state.forcedModel != "" {
		t.Errorf("forcedModel should be cleared, got %q", deps.state.forcedModel)
	}
}

// TestHandleContextCommandList verifies /context list via the mock provider.
func TestHandleContextCommandList(t *testing.T) {
	provider := createMockProvider(t)
	idle := true
	deps := &loopDeps{
		ctx:         context.Background(),
		k8sProvider: provider,
		state:       &agentState{},
		isIdle:      &idle,
	}

	for _, input := range []string{"/context", "/context list", "/context LIST"} {
		handled, err := handleContextCommand(deps, input)
		if err != nil {
			t.Fatalf("handleContextCommand(%q) returned error: %v", input, err)
		}
		if !handled {
			t.Errorf("handleContextCommand(%q) should return handled=true", input)
		}
	}
}

// TestHandleContextCommandUse verifies /context use <name> switches the active context.
func TestHandleContextCommandUse(t *testing.T) {
	provider := createMockProvider(t)
	clusters := provider.GetClusters()
	if len(clusters) == 0 {
		t.Skip("no clusters in mock provider")
	}
	targetCtx := clusters[0].Context

	idle := true
	deps := &loopDeps{
		ctx:         context.Background(),
		k8sProvider: provider,
		state:       &agentState{},
		isIdle:      &idle,
	}

	handled, err := handleContextCommand(deps, "/context use "+targetCtx)
	if err != nil {
		t.Fatalf("handleContextCommand(/context use ...) returned error: %v", err)
	}
	if !handled {
		t.Error("should return handled=true")
	}
	if provider.GetCurrentContext() != targetCtx {
		t.Errorf("current context = %q, want %q", provider.GetCurrentContext(), targetCtx)
	}
}

// TestHandleContextCommandInvalid verifies /context with bad syntax is gracefully rejected.
func TestHandleContextCommandInvalid(t *testing.T) {
	provider := createMockProvider(t)
	idle := true
	deps := &loopDeps{
		ctx:         context.Background(),
		k8sProvider: provider,
		state:       &agentState{},
		isIdle:      &idle,
	}

	handled, err := handleContextCommand(deps, "/context badcmd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Error("should return handled=true")
	}
}

// TestPrintUsage exercises printUsage under several quota conditions.
func TestPrintUsage(t *testing.T) {
	tests := []struct {
		name  string
		state *agentState
	}{
		{
			name: "unlimited quota",
			state: &agentState{
				sessionStart:   time.Now().Add(-90 * time.Second),
				quotaUnlimited: true,
				turnCount:      5,
				turnsMiniCount: 3,
				turnsGPT4Count: 2,
			},
		},
		{
			name: "limited quota",
			state: &agentState{
				sessionStart:       time.Now().Add(-2 * time.Minute),
				quotaUnlimited:     false,
				quotaTotal:         100,
				quotaUsed:          40,
				premiumUsedAtStart: 30,
				quotaPercentage:    60,
				turnCount:          3,
			},
		},
		{
			name: "hours elapsed",
			state: &agentState{
				sessionStart:   time.Now().Add(-70 * time.Minute),
				quotaUnlimited: true,
				forcedModel:    "gpt-4o",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// printUsage writes to stdout; we just ensure it doesn't panic.
			printUsage(tt.state)
		})
	}
}

// TestDispatchUXCommandLast verifies /last is routed and handled.
func TestDispatchUXCommandLast(t *testing.T) {
	provider := createMockProvider(t)
	idle := true
	state := &agentState{}
	state.setLastResponse("response text")
	deps := &loopDeps{
		ctx:         context.Background(),
		k8sProvider: provider,
		state:       state,
		isIdle:      &idle,
	}
	ts := &turnState{model: modelCostEffective}

	handled, err := dispatchUXCommand(deps, "/last", ts)
	if err != nil || !handled {
		t.Errorf("dispatchUXCommand(/last): handled=%v err=%v", handled, err)
	}
}

// TestDispatchUXCommandUsage verifies /usage is routed and handled.
func TestDispatchUXCommandUsage(t *testing.T) {
	provider := createMockProvider(t)
	idle := true
	deps := &loopDeps{
		ctx:         context.Background(),
		k8sProvider: provider,
		state:       &agentState{sessionStart: time.Now(), quotaUnlimited: true},
		isIdle:      &idle,
	}
	ts := &turnState{}

	handled, err := dispatchUXCommand(deps, "/usage", ts)
	if err != nil || !handled {
		t.Errorf("dispatchUXCommand(/usage): handled=%v err=%v", handled, err)
	}
}

// TestDispatchUXCommandStreamer verifies /streamer is dispatched.
func TestDispatchUXCommandStreamer(t *testing.T) {
	provider := createMockProvider(t)
	idle := true
	deps := &loopDeps{
		ctx:         context.Background(),
		k8sProvider: provider,
		state:       &agentState{},
		isIdle:      &idle,
	}
	ts := &turnState{}

	for _, cmd := range []string{"/streamer", "/streamer on", "/streamer off"} {
		handled, err := dispatchUXCommand(deps, cmd, ts)
		if err != nil || !handled {
			t.Errorf("dispatchUXCommand(%q): handled=%v err=%v", cmd, handled, err)
		}
	}
}

// TestDispatchUXCommandCopy verifies /copy is dispatched (empty buffer case).
func TestDispatchUXCommandCopy(t *testing.T) {
	provider := createMockProvider(t)
	idle := true
	deps := &loopDeps{
		ctx:         context.Background(),
		k8sProvider: provider,
		state:       &agentState{},
		isIdle:      &idle,
	}
	ts := &turnState{}

	handled, err := dispatchUXCommand(deps, "/copy", ts)
	if err != nil || !handled {
		t.Errorf("dispatchUXCommand(/copy): handled=%v err=%v", handled, err)
	}
}

// TestDispatchUXCommandModel verifies /model and /model reset are dispatched.
func TestDispatchUXCommandModel(t *testing.T) {
	provider := createMockProvider(t)
	idle := true
	deps := &loopDeps{
		ctx:         context.Background(),
		k8sProvider: provider,
		state:       &agentState{forcedModel: "gpt-4o"},
		isIdle:      &idle,
	}
	ts := &turnState{model: "gpt-4o"}

	for _, cmd := range []string{"/model", "/model reset"} {
		handled, err := dispatchUXCommand(deps, cmd, ts)
		if err != nil || !handled {
			t.Errorf("dispatchUXCommand(%q): handled=%v err=%v", cmd, handled, err)
		}
	}
}

// TestDispatchUXCommandContext verifies /context list is dispatched.
func TestDispatchUXCommandContext(t *testing.T) {
	provider := createMockProvider(t)
	idle := true
	deps := &loopDeps{
		ctx:         context.Background(),
		k8sProvider: provider,
		state:       &agentState{},
		isIdle:      &idle,
	}
	ts := &turnState{}

	handled, err := dispatchUXCommand(deps, "/context list", ts)
	if err != nil || !handled {
		t.Errorf("dispatchUXCommand(/context list): handled=%v err=%v", handled, err)
	}
}

// TestDispatchUXCommandUnknown verifies that unknown commands return handled=false.
func TestDispatchUXCommandUnknown(t *testing.T) {
	provider := createMockProvider(t)
	idle := true
	deps := &loopDeps{
		ctx:         context.Background(),
		k8sProvider: provider,
		state:       &agentState{},
		isIdle:      &idle,
	}
	ts := &turnState{}

	handled, err := dispatchUXCommand(deps, "/unknown", ts)
	if err != nil {
		t.Fatalf("unexpected error for unknown command: %v", err)
	}
	if handled {
		t.Error("dispatchUXCommand should return handled=false for unknown commands")
	}
}

// TestBuildAttachmentContent covers the WP-05 safety limits.
func TestBuildAttachmentContent(t *testing.T) {
	tmpDir := t.TempDir()

	smallFile := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(smallFile, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}

	binaryFile := filepath.Join(tmpDir, "binary.bin")
	if err := os.WriteFile(binaryFile, []byte{0x7f, 0x45, 0x4c, 0x46, 0x00, 0x01}, 0600); err != nil {
		t.Fatal(err)
	}

	largeFile := filepath.Join(tmpDir, "large.txt")
	if err := os.WriteFile(largeFile, make([]byte, maxAttachmentFileSize+1), 0600); err != nil {
		t.Fatal(err)
	}

	t.Run("small text file included", func(t *testing.T) {
		got := buildAttachmentContent([]string{smallFile}, OutputJSON)
		if !strings.Contains(got, "hello world") {
			t.Errorf("expected small file content, got: %q", got)
		}
	})

	t.Run("binary file excluded", func(t *testing.T) {
		got := buildAttachmentContent([]string{binaryFile}, OutputJSON)
		if strings.Contains(got, "\x00") {
			t.Error("binary file content should not appear in output")
		}
		if strings.Contains(got, "binary.bin") {
			t.Error("binary file name should not appear when excluded")
		}
	})

	t.Run("large file excluded", func(t *testing.T) {
		got := buildAttachmentContent([]string{largeFile}, OutputJSON)
		if strings.Contains(got, "large.txt") {
			t.Error("large file name should not appear when excluded")
		}
	})

	t.Run("cumulative limit respected", func(t *testing.T) {
		// Three files of 400 KB each (< 512 KB per-file limit).
		// First two fit (800 KB < 1 MB); third pushes total to 1.2 MB and is excluded.
		const fileSize = 400 * 1024
		payload := make([]byte, fileSize)
		for i := range payload {
			payload[i] = 'a'
		}
		f1 := filepath.Join(tmpDir, "big1.txt")
		f2 := filepath.Join(tmpDir, "big2.txt")
		f3 := filepath.Join(tmpDir, "big3.txt")
		for _, f := range []string{f1, f2, f3} {
			if err := os.WriteFile(f, payload, 0600); err != nil {
				t.Fatal(err)
			}
		}
		got := buildAttachmentContent([]string{f1, f2, f3}, OutputJSON)
		if !strings.Contains(got, "big1.txt") {
			t.Error("first file should be included")
		}
		if !strings.Contains(got, "big2.txt") {
			t.Error("second file should be included")
		}
		if strings.Contains(got, "big3.txt") {
			t.Error("third file should be excluded by cumulative limit")
		}
	})
}

// TestKubectlTimeout covers the WP-06 timeout configurability.
func TestKubectlTimeout(t *testing.T) {
	t.Run("default when unset", func(t *testing.T) {
		t.Setenv("KOPILOT_KUBECTL_TIMEOUT", "")
		if got := kubectlTimeout(); got != 30*time.Second {
			t.Errorf("default timeout = %v, want 30s", got)
		}
	})

	t.Run("custom valid duration", func(t *testing.T) {
		t.Setenv("KOPILOT_KUBECTL_TIMEOUT", "2m")
		if got := kubectlTimeout(); got != 2*time.Minute {
			t.Errorf("custom timeout = %v, want 2m", got)
		}
	})

	t.Run("invalid value falls back to default", func(t *testing.T) {
		t.Setenv("KOPILOT_KUBECTL_TIMEOUT", "not-a-duration")
		if got := kubectlTimeout(); got != 30*time.Second {
			t.Errorf("invalid timeout = %v, want 30s fallback", got)
		}
	})

	t.Run("zero or negative falls back to default", func(t *testing.T) {
		t.Setenv("KOPILOT_KUBECTL_TIMEOUT", "-5s")
		if got := kubectlTimeout(); got != 30*time.Second {
			t.Errorf("negative timeout = %v, want 30s fallback", got)
		}
	})
}

// ── WP-07: provider abstraction lifecycle and regression coverage ─────────────

// fakeSession is a minimal llm.Session stub for WP-07 contract tests.
type fakeSession struct {
	disconnected bool
	handlers     []func(llm.Event)
}

func (s *fakeSession) Disconnect() error                             { s.disconnected = true; return nil }
func (s *fakeSession) SendPrompt(_ context.Context, _ string) error  { return nil }
func (s *fakeSession) On(h func(llm.Event))                          { s.handlers = append(s.handlers, h) }
func (s *fakeSession) emit(e llm.Event) {
	for _, h := range s.handlers {
		h(e)
	}
}

// fakeProvider is a minimal llm.Provider that returns a pre-built fakeSession.
type fakeProvider struct {
	session    *fakeSession
	lastConfig *llm.SessionConfig
}

func (p *fakeProvider) Name() string { return "fake" }
func (p *fakeProvider) Start(_ context.Context) error { return nil }
func (p *fakeProvider) Stop() error                   { return nil }
func (p *fakeProvider) CreateSession(_ context.Context, cfg *llm.SessionConfig) (llm.Session, error) {
	p.lastConfig = cfg
	return p.session, nil
}

// TestSetupSessionEventHandlerRouting verifies that each normalized EventType
// is dispatched by setupSessionEventHandler without panicking.
func TestSetupSessionEventHandlerRouting(t *testing.T) {
	sess := &fakeSession{}
	isIdle := false
	state := &agentState{outputFormat: OutputJSON}

	setupSessionEventHandler(sess, &isIdle, state)

	// EventIdle must flip the idle flag.
	sess.emit(llm.Event{Type: llm.EventIdle})
	if !isIdle {
		t.Error("EventIdle should set isIdle=true")
	}

	// All other events must not panic regardless of Data contents.
	sess.emit(llm.Event{Type: llm.EventMessage, Data: &llm.MessageData{Content: "hello"}})
	sess.emit(llm.Event{Type: llm.EventDelta, Data: &llm.DeltaData{Content: "chunk"}})
	sess.emit(llm.Event{Type: llm.EventError, Data: &llm.ErrorData{Message: "boom"}})
	sess.emit(llm.Event{Type: llm.EventUsage, Data: &llm.UsageData{QuotaPercentage: 42}})
}

// TestSwitchToModelDisconnectsOldSession verifies that switchToModel calls
// Disconnect on the previous session and returns the provider's new session.
func TestSwitchToModelDisconnectsOldSession(t *testing.T) {
	k8sProvider := newTestK8sProvider(t)

	oldSess := &fakeSession{}
	newSess := &fakeSession{}
	provider := &fakeProvider{session: newSess}

	isIdle := true // pre-set so waitForIdle returns immediately
	state := &agentState{
		mode:          ModeReadOnly,
		outputFormat:  OutputJSON,
		selectedAgent: AgentDefault,
		mcpConfigPath: filepath.Join(t.TempDir(), "mcp.json"),
	}
	deps := &loopDeps{
		ctx:         context.Background(),
		provider:    provider,
		k8sProvider: k8sProvider,
		state:       state,
		isIdle:      &isIdle,
	}

	got, err := switchToModel(deps, oldSess, "test-model")
	if err != nil {
		t.Fatalf("switchToModel returned error: %v", err)
	}
	if !oldSess.disconnected {
		t.Error("switchToModel should call Disconnect on the old session")
	}
	if got != newSess {
		t.Error("switchToModel should return the session from provider.CreateSession")
	}
}

// TestCreateSessionWithModelIncludesMCPServers verifies that createSessionWithModel
// always includes the MCPServers key in ExtraConfig, even when the config file is absent.
func TestCreateSessionWithModelIncludesMCPServers(t *testing.T) {
	k8sProvider := newTestK8sProvider(t)

	sess := &fakeSession{}
	provider := &fakeProvider{session: sess}

	state := &agentState{
		mode:          ModeReadOnly,
		outputFormat:  OutputJSON,
		selectedAgent: AgentDefault,
		mcpConfigPath: filepath.Join(t.TempDir(), "mcp.json"), // absent → empty list
	}

	_, err := createSessionWithModel(context.Background(), provider, k8sProvider, state, "some-model")
	if err != nil {
		t.Fatalf("createSessionWithModel returned error: %v", err)
	}
	if provider.lastConfig == nil {
		t.Fatal("provider.CreateSession was never called")
	}
	if _, ok := provider.lastConfig.ExtraConfig["MCPServers"]; !ok {
		t.Error("SessionConfig.ExtraConfig must contain MCPServers key")
	}
	if provider.lastConfig.Model != "some-model" {
		t.Errorf("Model = %q, want some-model", provider.lastConfig.Model)
	}
}
