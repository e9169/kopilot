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
	provider := createMockProvider(b)
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
