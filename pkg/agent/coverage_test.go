package agent

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/e9169/kopilot/pkg/k8s"
)

// ── agent.go helpers ──────────────────────────────────────────────────────────

const (
	testVersion      = "v1.28.0"
	testNamespace    = "kube-system"
	testCmdGetPods   = "kubectl get pods"
	testCmdDeletePod = "kubectl delete pod x"
	testServerURL    = "https://test-server.example.com"
)

func TestParseAgentType(t *testing.T) {
	validCases := []struct {
		input    string
		expected AgentType
	}{
		{"default", AgentDefault},
		{"debugger", AgentDebugger},
		{"security", AgentSecurity},
		{"optimizer", AgentOptimizer},
		{"gitops", AgentGitOps},
		{"DEFAULT", AgentDefault},
		{"Debugger", AgentDebugger},
	}
	for _, tc := range validCases {
		got, err := ParseAgentType(tc.input)
		if err != nil {
			t.Errorf("ParseAgentType(%q) unexpected error: %v", tc.input, err)
		}
		if got != tc.expected {
			t.Errorf("ParseAgentType(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}

	_, err := ParseAgentType("unknown")
	if err == nil {
		t.Error("ParseAgentType(\"unknown\") expected error, got nil")
	}
	if !strings.Contains(err.Error(), "valid agents") {
		t.Errorf("error message should list valid agents, got: %s", err.Error())
	}
}

func TestAllAgentNames(t *testing.T) {
	names := allAgentNames()
	want := []string{"default", "debugger", "security", "optimizer", "gitops"}
	if len(names) != len(want) {
		t.Fatalf("allAgentNames() returned %d names, want %d", len(names), len(want))
	}
	for i, n := range want {
		if names[i] != n {
			t.Errorf("allAgentNames()[%d] = %q, want %q", i, names[i], n)
		}
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	const key = "KOPILOT_TEST_GETENVKEY"
	t.Cleanup(func() { os.Unsetenv(key) })

	// Not set → default
	if got := getEnvOrDefault(key, "fallback"); got != "fallback" {
		t.Errorf("getEnvOrDefault unset = %q, want %q", got, "fallback")
	}

	// Set → env value
	os.Setenv(key, "custom")
	if got := getEnvOrDefault(key, "fallback"); got != "custom" {
		t.Errorf("getEnvOrDefault set = %q, want %q", got, "custom")
	}
}

func TestExecutionModeString(t *testing.T) {
	cases := []struct {
		mode ExecutionMode
		want string
	}{
		{ModeReadOnly, "read-only"},
		{ModeInteractive, "interactive"},
		{ExecutionMode(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.mode.String(); got != tc.want {
			t.Errorf("ExecutionMode(%q).String() = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

func TestGetSystemMessage(t *testing.T) {
	msg := getSystemMessage()
	if msg == "" {
		t.Fatal("getSystemMessage() returned empty string")
	}
	for _, keyword := range []string{"Kopilot", "kubectl", "cluster"} {
		if !strings.Contains(msg, keyword) {
			t.Errorf("system message should contain %q", keyword)
		}
	}
}

func TestIsExitCommand(t *testing.T) {
	exits := []string{"exit", "EXIT", "Exit", "quit", "QUIT", "Quit"}
	for _, s := range exits {
		if !isExitCommand(s) {
			t.Errorf("isExitCommand(%q) should be true", s)
		}
	}
	notExits := []string{"", "help", "exitnow", "quitnow", "/exit"}
	for _, s := range notExits {
		if isExitCommand(s) {
			t.Errorf("isExitCommand(%q) should be false", s)
		}
	}
}

// TestHandleModeSwitchExtended covers /readonly on, /interactive on and /help variants
// not already covered by TestHandleModeSwitch in agent_test.go.
func TestHandleModeSwitchExtended(t *testing.T) {
	cases := []struct {
		input         string
		initialMode   ExecutionMode
		wantHandled   bool
		wantFinalMode ExecutionMode
	}{
		{"/readonly on", ModeInteractive, true, ModeReadOnly},
		{"/interactive on", ModeReadOnly, true, ModeInteractive},
		{"/help", ModeReadOnly, true, ModeReadOnly},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			state := &agentState{mode: tc.initialMode}
			got := handleModeSwitch(tc.input, state)
			if got != tc.wantHandled {
				t.Errorf("handleModeSwitch(%q) handled = %v, want %v", tc.input, got, tc.wantHandled)
			}
			if state.mode != tc.wantFinalMode {
				t.Errorf("handleModeSwitch(%q) mode = %q, want %q", tc.input, state.mode, tc.wantFinalMode)
			}
		})
	}
}

func TestFormatAgentSwitchMessage(t *testing.T) {
	msg := formatAgentSwitchMessage(AgentDefault)
	if !strings.Contains(msg, "default") {
		t.Errorf("switch-to-default message should contain 'default', got: %s", msg)
	}

	for _, at := range []AgentType{AgentDebugger, AgentSecurity, AgentOptimizer, AgentGitOps} {
		msg = formatAgentSwitchMessage(at)
		def := agentDefinitions[at]
		if !strings.Contains(msg, def.DisplayName) {
			t.Errorf("switch message for %q should contain %q, got: %s", at, def.DisplayName, msg)
		}
	}
}

func TestFormatAlreadyUsingAgent(t *testing.T) {
	msg := formatAlreadyUsingAgent(AgentDefault)
	if !strings.Contains(msg, "default") {
		t.Errorf("already-using-default message should contain 'default', got: %s", msg)
	}

	for _, at := range []AgentType{AgentDebugger, AgentSecurity, AgentOptimizer, AgentGitOps} {
		msg = formatAlreadyUsingAgent(at)
		def := agentDefinitions[at]
		if !strings.Contains(msg, def.DisplayName) {
			t.Errorf("already-using message for %q should contain %q, got: %s", at, def.DisplayName, msg)
		}
	}
}

func TestHandleAgentCommand(t *testing.T) {
	state := &agentState{selectedAgent: AgentDefault}

	// Not an /agent command
	isCmd, _, err := handleAgentCommand("hello world", state)
	if isCmd || err != nil {
		t.Errorf("non-agent command: isCmd=%v err=%v, want false, nil", isCmd, err)
	}

	// "/agent" alone → show list, no agent change
	isCmd, agent, err := handleAgentCommand("/agent", state)
	if !isCmd || err != nil {
		t.Errorf("/agent: isCmd=%v err=%v, want true, nil", isCmd, err)
	}
	if agent != AgentDefault {
		t.Errorf("/agent agent = %q, want %q", agent, AgentDefault)
	}

	// "/agent list"
	isCmd, _, err = handleAgentCommand("/agent list", state)
	if !isCmd || err != nil {
		t.Errorf("/agent list: isCmd=%v err=%v, want true, nil", isCmd, err)
	}

	// "/agent debugger" → switch
	isCmd, agent, err = handleAgentCommand("/agent debugger", state)
	if !isCmd || err != nil || agent != AgentDebugger {
		t.Errorf("/agent debugger: isCmd=%v agent=%q err=%v, want true, debugger, nil", isCmd, agent, err)
	}

	// "/agent default" when already default → already using
	isCmd, agent, err = handleAgentCommand("/agent default", state)
	if !isCmd || err != nil || agent != AgentDefault {
		t.Errorf("/agent default (same): isCmd=%v agent=%q err=%v", isCmd, agent, err)
	}

	// "/agent invalid"
	isCmd, _, err = handleAgentCommand("/agent invalid", state)
	if !isCmd || err == nil {
		t.Errorf("/agent invalid: isCmd=%v err=%v, want true, non-nil error", isCmd, err)
	}

	// "/agent too many args"
	isCmd, _, err = handleAgentCommand("/agent one two three", state)
	if !isCmd || err == nil {
		t.Errorf("/agent too many args: isCmd=%v err=%v, want true, error", isCmd, err)
	}
}

func TestGetAvailableContexts(t *testing.T) {
	provider := createMockProvider(t)
	ctx := getAvailableContexts(provider)
	if ctx == "none" {
		t.Error("expected at least one cluster context, got 'none'")
	}
	// Mock provider creates 2 clusters
	if !strings.Contains(ctx, ",") {
		t.Errorf("expected multiple contexts separated by comma, got: %s", ctx)
	}
}

// ── tools.go helpers ──────────────────────────────────────────────────────────

func TestWriteUnreachableClusterStatus(t *testing.T) {
	var b strings.Builder
	status := &k8s.ClusterStatus{
		ClusterInfo: k8s.ClusterInfo{Context: "test-ctx", Server: testServerURL},
		Error:       "connection refused",
	}
	writeUnreachableClusterStatus(&b, status)
	out := b.String()
	if !strings.Contains(out, "test-ctx") {
		t.Error("output should contain context name")
	}
	if !strings.Contains(out, "connection refused") {
		t.Error("output should contain error message")
	}

	// No error field
	var b2 strings.Builder
	writeUnreachableClusterStatus(&b2, &k8s.ClusterStatus{ClusterInfo: k8s.ClusterInfo{Context: "c"}})
	if strings.Contains(b2.String(), "Issue:") {
		t.Error("output should not have Issue line when error is empty")
	}
}

func TestWriteClusterInfo(t *testing.T) {
	var b strings.Builder
	status := &k8s.ClusterStatus{
		ClusterInfo:  k8s.ClusterInfo{Context: "ctx1", User: "admin", Namespace: testNamespace},
		APIServerURL: "https://api.example.com",
		Version:      testVersion,
	}
	writeClusterInfo(&b, status)
	out := b.String()
	for _, want := range []string{"ctx1", "admin", testNamespace, "https://api.example.com", testVersion} {
		if !strings.Contains(out, want) {
			t.Errorf("clusterInfo output missing %q", want)
		}
	}

	// Without namespace
	var b2 strings.Builder
	writeClusterInfo(&b2, &k8s.ClusterStatus{})
	if strings.Contains(b2.String(), "Default Namespace") {
		t.Error("should not print namespace line when empty")
	}
}

func TestWriteNodeInfo(t *testing.T) {
	var b strings.Builder
	status := &k8s.ClusterStatus{
		ClusterInfo:  k8s.ClusterInfo{},
		NodeCount:    2,
		HealthyNodes: 1,
		Nodes: []k8s.NodeInfo{
			{Name: "node-a", Status: "Ready", Roles: []string{"control-plane"}, Age: "10d"},
			{Name: "node-b", Status: "NotReady", Roles: []string{"worker"}, Age: "5d"},
		},
	}
	writeNodeInfo(&b, status)
	out := b.String()
	if !strings.Contains(out, "2 total") {
		t.Error("output should show total node count")
	}
	if !strings.Contains(out, "node-a") || !strings.Contains(out, "node-b") {
		t.Error("output should list node names")
	}
	if !strings.Contains(out, "✅") || !strings.Contains(out, "❌") {
		t.Error("output should show ready/not-ready icons")
	}

	// No nodes
	var b2 strings.Builder
	writeNodeInfo(&b2, &k8s.ClusterStatus{})
	if strings.Contains(b2.String(), "Node Details") {
		t.Error("should not print node details when no nodes")
	}
}

func TestWriteNamespaceInfo(t *testing.T) {
	var b strings.Builder
	status := &k8s.ClusterStatus{
		NamespaceList: []string{"default", testNamespace},
		Error:         "partial error",
	}
	writeNamespaceInfo(&b, status)
	out := b.String()
	if !strings.Contains(out, "default") {
		t.Error("output should list namespaces")
	}
	if !strings.Contains(out, "partial error") {
		t.Error("output should show warning when error present")
	}

	// Empty
	var b2 strings.Builder
	writeNamespaceInfo(&b2, &k8s.ClusterStatus{})
	if b2.String() != "" {
		t.Errorf("empty status should produce empty output, got: %q", b2.String())
	}
}

func TestWriteComparisonEntry(t *testing.T) {
	var b strings.Builder
	comp := ComparisonData{
		Context:      "prod",
		Name:         "production",
		Status:       "✅ Reachable",
		Version:      testVersion,
		Nodes:        "3",
		HealthyNodes: "3",
		APIServer:    "https://api.prod",
	}
	writeComparisonEntry(&b, 0, comp)
	out := b.String()
	for _, want := range []string{"[1]", "prod", "production", testVersion, "3", "https://api.prod"} {
		if !strings.Contains(out, want) {
			t.Errorf("comparison entry missing %q", want)
		}
	}

	// Minimal (only Context and Status)
	var b2 strings.Builder
	writeComparisonEntry(&b2, 1, ComparisonData{Context: "c", Status: "❌ Unreachable", Error: "timeout"})
	if !strings.Contains(b2.String(), "timeout") {
		t.Error("error should appear in minimal entry")
	}
}

func TestCountReachableClusters(t *testing.T) {
	comparisons := []ComparisonData{
		{Status: "✅ Reachable"},
		{Status: "⚠️  Degraded"},
		{Status: "❌ Unreachable"},
		{Status: "ERROR"},
	}
	if got := countReachableClusters(comparisons); got != 2 {
		t.Errorf("countReachableClusters = %d, want 2", got)
	}

	if got := countReachableClusters(nil); got != 0 {
		t.Errorf("countReachableClusters(nil) = %d, want 0", got)
	}
}

func TestAnalyzeClusterHealth(t *testing.T) {
	statuses := []*k8s.ClusterStatus{
		{
			ClusterInfo: k8s.ClusterInfo{Context: "healthy", IsReachable: true},
			NodeCount:   2, HealthyNodes: 2,
			PodCount: 10, HealthyPods: 10,
		},
		{
			ClusterInfo: k8s.ClusterInfo{Context: "degraded-nodes", IsReachable: true},
			NodeCount:   3, HealthyNodes: 2,
			PodCount: 5, HealthyPods: 5,
		},
		{
			ClusterInfo: k8s.ClusterInfo{Context: "unhealthy-pods", IsReachable: true},
			NodeCount:   2, HealthyNodes: 2,
			PodCount: 10, HealthyPods: 7,
		},
		{
			ClusterInfo: k8s.ClusterInfo{Context: "down", IsReachable: false, Server: "https://gone"},
			Error:       "timeout",
		},
	}
	summary := analyzeClusterHealth(statuses)

	if summary.reachableCount != 3 {
		t.Errorf("reachableCount = %d, want 3", summary.reachableCount)
	}
	if summary.healthyCount != 1 {
		t.Errorf("healthyCount = %d, want 1", summary.healthyCount)
	}
	if summary.totalUnhealthyPods != 3 {
		t.Errorf("totalUnhealthyPods = %d, want 3", summary.totalUnhealthyPods)
	}
	if len(summary.issues) != 3 { // degraded-nodes, unhealthy-pods, down
		t.Errorf("issues count = %d, want 3: %v", len(summary.issues), summary.issues)
	}
}

func TestWriteCompactClusterStatus(t *testing.T) {
	cases := []struct {
		name    string
		status  k8s.ClusterStatus
		wantStr string
	}{
		{
			name: "healthy",
			status: k8s.ClusterStatus{
				ClusterInfo: k8s.ClusterInfo{Context: "prod", IsReachable: true},
				NodeCount:   3, HealthyNodes: 3, PodCount: 20, HealthyPods: 20, Version: "1.28.0",
			},
			wantStr: "✅",
		},
		{
			name: "degraded nodes",
			status: k8s.ClusterStatus{
				ClusterInfo: k8s.ClusterInfo{Context: "stg", IsReachable: true},
				NodeCount:   3, HealthyNodes: 2, PodCount: 10, HealthyPods: 10,
			},
			wantStr: "⚠️",
		},
		{
			name: "degraded pods",
			status: k8s.ClusterStatus{
				ClusterInfo: k8s.ClusterInfo{Context: "dev", IsReachable: true},
				NodeCount:   2, HealthyNodes: 2, PodCount: 10, HealthyPods: 8,
			},
			wantStr: "⚠️",
		},
		{
			name:    "down",
			status:  k8s.ClusterStatus{ClusterInfo: k8s.ClusterInfo{Context: "old", IsReachable: false}},
			wantStr: "❌",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b strings.Builder
			writeCompactClusterStatus(&b, &tc.status)
			if !strings.Contains(b.String(), tc.wantStr) {
				t.Errorf("writeCompactClusterStatus(%s) missing %q, got: %s", tc.name, tc.wantStr, b.String())
			}
		})
	}
}

func TestValidateKubectlExecParams(t *testing.T) {
	if err := validateKubectlExecParams(KubectlExecParams{Context: "", Args: []string{"get", "pods"}}); err == nil {
		t.Error("empty context should return error")
	}
	if err := validateKubectlExecParams(KubectlExecParams{Context: "ctx", Args: []string{}}); err == nil {
		t.Error("empty args should return error")
	}
	if err := validateKubectlExecParams(KubectlExecParams{Context: "ctx", Args: []string{"get", "pods"}}); err != nil {
		t.Errorf("valid params should not return error: %v", err)
	}
}

func TestBuildKubectlCommand(t *testing.T) {
	full, cmdArgs := buildKubectlCommand("my-ctx", []string{"get", "pods", "-n", "default"})
	if !strings.Contains(full, "kubectl") {
		t.Error("full command should contain 'kubectl'")
	}
	if !strings.Contains(full, "--context my-ctx") {
		t.Error("full command should contain context flag")
	}
	if !strings.Contains(full, "get pods") {
		t.Error("full command should contain kubectl args")
	}
	if cmdArgs[0] != "--context" || cmdArgs[1] != "my-ctx" {
		t.Errorf("cmdArgs should start with --context my-ctx, got: %v", cmdArgs)
	}
	if cmdArgs[2] != "get" {
		t.Errorf("cmdArgs[2] should be 'get', got: %q", cmdArgs[2])
	}
}

func TestEnforceExecutionModeReadOnly(t *testing.T) {
	// Write op in read-only mode → blocked
	state := &agentState{mode: ModeReadOnly, outputFormat: OutputText}
	proceed, result, err := enforceExecutionMode(state, false, "prod", "ctx", testCmdDeletePod, "delete")
	if proceed || result != nil || err == nil {
		t.Errorf("write op in read-only should be blocked: proceed=%v result=%v err=%v", proceed, result, err)
	}
	if !strings.Contains(err.Error(), "read-only") {
		t.Errorf("error should mention read-only, got: %s", err.Error())
	}

	// Read op in read-only mode → allowed
	proceed, _, err = enforceExecutionMode(state, true, "prod", "ctx", testCmdGetPods, "get")
	if !proceed || err != nil {
		t.Errorf("read op in read-only should be allowed: proceed=%v err=%v", proceed, err)
	}
}

func TestBuildKubectlJSONResult(t *testing.T) {
	// Success case
	result, err := buildKubectlJSONResult("prod", "ctx", testCmdGetPods, []byte("output"), nil)
	if err != nil {
		t.Errorf("success case should not return error: %v", err)
	}
	r, ok := result.(KubectlExecResult)
	if !ok {
		t.Fatalf("result should be KubectlExecResult, got %T", result)
	}
	if r.Cluster != "prod" || r.Output != "output" || r.Error != "" {
		t.Errorf("unexpected result fields: %+v", r)
	}

	// Error case (non-exit error)
	result, err = buildKubectlJSONResult("prod", "ctx", testCmdDeletePod, []byte(""), fmt.Errorf("some error"))
	if err == nil {
		t.Error("error case should return an error")
	}
	r, ok = result.(KubectlExecResult)
	if !ok {
		t.Fatalf("result should be KubectlExecResult on error, got %T", result)
	}
	if r.Error == "" {
		t.Error("KubectlExecResult.Error should be set on error")
	}
}

func TestBuildKubectlTextResult(t *testing.T) {
	// Success case
	out, err := buildKubectlTextResult("prod", "ctx", testCmdGetPods, []byte("NAME\npod-1"), nil)
	if err != nil {
		t.Errorf("success case should not return error: %v", err)
	}
	for _, want := range []string{"prod", "ctx", testCmdGetPods, "pod-1"} {
		if !strings.Contains(out, want) {
			t.Errorf("text result missing %q", want)
		}
	}

	// Error case
	out, err = buildKubectlTextResult("prod", "ctx", testCmdDeletePod, []byte("forbidden"), fmt.Errorf("exit status 1"))
	if err == nil {
		t.Error("error case should return an error")
	}
	if !strings.Contains(out, "❌") {
		t.Error("error output should contain error icon")
	}
	if !strings.Contains(out, "forbidden") {
		t.Error("error output should contain command output")
	}
}
