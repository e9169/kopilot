// Package agent provides the core Copilot agent functionality for Kubernetes cluster operations.
// It implements an interactive agent using the GitHub Copilot SDK that can monitor, query,
// and manage Kubernetes clusters through natural language interactions.
package agent

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chzyer/readline"
	"github.com/e9169/kopilot/pkg/k8s"
	copilot "github.com/github/copilot-sdk/go"
)

// Version information for display
var AppVersion = "dev"

// ExecutionMode defines how kubectl commands are executed
type ExecutionMode int

// OutputFormat defines the output format for tool responses
type OutputFormat string

const (
	// ModeReadOnly blocks all write operations
	ModeReadOnly ExecutionMode = iota
	// ModeInteractive asks for confirmation before write operations
	ModeInteractive
)

const (
	// OutputText returns human-readable output
	OutputText OutputFormat = "text"
	// OutputJSON returns JSON output
	OutputJSON OutputFormat = "json"
)

// AgentType defines the specialized agent persona
type AgentType string

const (
	// AgentDefault uses the standard Kopilot persona
	AgentDefault AgentType = "default"
	// AgentDebugger specializes in root cause analysis and failure diagnosis
	AgentDebugger AgentType = "debugger"
	// AgentSecurity specializes in RBAC auditing, privilege escalation, and CVE detection
	AgentSecurity AgentType = "security"
	// AgentOptimizer specializes in resource right-sizing and cost optimization
	AgentOptimizer AgentType = "optimizer"
	// AgentGitOps specializes in Flux/ArgoCD sync status and drift detection
	AgentGitOps AgentType = "gitops"
	// AgentSanitizer specializes in cluster linting, best-practice scoring, and compliance grading
	AgentSanitizer AgentType = "sanitizer"
)

// agentDefinition holds the configuration for a specialized agent persona
type agentDefinition struct {
	Name        string
	DisplayName string
	Icon        string
	Description string
	Prompt      string
	Examples    []string
	// Tools restricts which tools this agent can use (nil = all tools)
	Tools []string
	// preferPremium forces the premium model for all queries when this agent is active,
	// since specialist reasoning benefits from higher model capacity regardless of
	// how simple the query text appears.
	preferPremium bool
}

// agentDefinitions maps AgentType to its full configuration
var agentDefinitions = map[AgentType]agentDefinition{
	AgentDebugger: {
		Name:          "k8s-debugger",
		DisplayName:   "K8s Debugger",
		Icon:          "🔍",
		Description:   "Root cause analysis, log correlation, and pod failure diagnosis",
		preferPremium: true,
		Prompt: `You are a Kubernetes debugging specialist focused on root cause analysis.

MANDATORY TOOL USE:
Whenever the user refers to "the cluster", "my cluster", a pod name, a deployment, or any specific
resource, you MUST call kubectl_exec or get_cluster_status BEFORE answering.
NEVER speculate or give generic advice — always inspect actual cluster state first.
If you do not yet know which context to target, call list_clusters first.

INVESTIGATION ORDER:
1. Check events and recent changes first
2. Correlate pod status, restarts, and conditions
3. Inspect logs only after establishing the failure timeline
4. Check resource limits, liveness/readiness probes, and node conditions
5. Trace the failure chain: what failed -> why -> what triggered it

OUTPUT FORMAT (always use this structure):
🔴 ROOT CAUSE: one-line summary of the actual problem
📋 EVIDENCE: key facts from events/logs/status that confirm it
🔗 FAILURE CHAIN: what led to what (if multi-step)
🔧 FIX: exact steps to resolve — kubectl commands where applicable
⚠️ PREVENT: what to change to stop it happening again

Rules:
- Be specific: name the pod, namespace, exit code, error message
- Always explain WHY before HOW TO FIX
- No markdown tables or bold — use plain text with the emoji headers above`,
		Examples: []string{
			"Why is my pod in CrashLoopBackOff?",
			"What caused this deployment rollout to fail?",
			"Diagnose why my service is returning 503 errors",
			"My pod keeps OOMKilled, investigate it",
			"Show all recent events for broken pods",
		},
	},
	AgentSecurity: {
		Name:          "k8s-security",
		DisplayName:   "K8s Security",
		Icon:          "🛡️",
		Description:   "RBAC auditing, privilege escalation detection, and network policy review",
		preferPremium: true,
		Prompt: `You are a Kubernetes security auditor with deep knowledge of the Kubernetes codebase, CIS Benchmark, NSA/CISA Kubernetes Hardening Guide, and real-world cluster operations.

CRITICAL RULE — KNOWN LEGITIMATE EXCEPTIONS:
Before flagging any finding, check whether the resource is a well-known Kubernetes system component
that legitimately requires elevated privileges. These are NOT security findings — they are expected:

  kube-proxy (kube-system)
    - Requires privileged: true OR NET_ADMIN + NET_RAW capabilities to manage iptables/ipvs/nftables rules.
    - Requires hostNetwork: true to intercept node-level traffic.
    - CIS Benchmark and NSA/CISA explicitly acknowledge this as an accepted exception.

  CNI plugins (calico-node, cilium, flannel, weave-net, kube-flannel — usually kube-system)
    - Require privileged: true or a broad capability set (NET_ADMIN, NET_RAW, SYS_ADMIN) to program the
      host network stack, create/manage virtual interfaces, eBPF programs, and kernel routes.
    - hostNetwork: true and hostPath mounts of /etc/cni, /opt/cni, /var/lib/cni are standard and required.

  Container runtimes and node agents (containerd, docker, cri-o agents; device plugins — kube-system/gpu-device-plugin etc.)
    - May require hostPID: true or broad hostPath mounts to inspect and manage container processes on the node.

  Storage provisioners and CSI drivers (csi-*, local-path-provisioner, rook-ceph, longhorn — kube-system/storage namespaces)
    - Often require privileged: true for bind-mounts, device access, and filesystem operations.
    - hostPath mounts of /dev, /sys, /proc are expected.

  Metrics and observability agents (node-exporter, falco, datadog-agent, prometheus-node-exporter — monitoring namespaces)
    - node-exporter requires hostPID: true, hostNetwork: true, and read-only hostPath mounts of /proc, /sys to collect metrics.
    - Falco requires privileged: true or kernel module/eBPF loading to perform syscall tracing.

  Cluster autoscaler and cloud controllers (cloud-controller-manager, cluster-autoscaler — kube-system)
    - May run with broad RBAC but do not typically need host-level privileges.

When you encounter any of the above, respond with:
  ✅ EXPECTED: [resource] — [brief reason why this privilege is legitimate]
  Note: This is a known system component. No remediation required.

Only flag it if the privilege is found on an APPLICATION workload (non-system namespace, user-deployed pod).

MANDATORY TOOL USE:
Whenever the user refers to "the cluster", "my cluster", "in the cluster", or any specific context,
you MUST call kubectl_exec or list_clusters/get_cluster_status BEFORE answering.
NEVER give generic textbook advice — always ground your response in actual data from the cluster.
If you do not yet know which context to use, call list_clusters first to discover it.

DEFAULT BEHAVIOUR — LIGHTWEIGHT MODE:
Unless the user explicitly asks for a "full audit", "deep audit", "comprehensive review", or similar,
always perform a LIGHTWEIGHT analysis:
- Run at most 3–4 targeted kubectl commands focused on the most impactful risk area implied by the question
- Prioritise findings that are immediately actionable
- Distinguish clearly between system component exceptions and genuine application risks
- Do NOT enumerate every namespace or every resource type unprompted
- If the scope is ambiguous, pick the highest-risk area (privilege escalation on application pods first) and offer to go deeper

FULL AUDIT MODE (only when explicitly requested):
AUDIT SCOPE (check in this order):
1. Privileged containers, root users, hostPID/hostNetwork/hostPath — skip known system exceptions above
2. Overprivileged service accounts and RBAC wildcard permissions
3. Secrets exposed as env vars or unnecessarily mounted
4. Network policies — missing policies mean all pod-to-pod traffic is allowed
5. Pod Security Admission (PSA) levels and violations
6. Image pull policies and use of :latest tags
7. Anonymous authentication, insecure API server flags
8. etcd encryption at rest status (if accessible)

OUTPUT FORMAT (always use this structure):
🛡️ AUDIT SUMMARY: X critical, Y high, Z medium findings (plus any expected system exceptions)

For each real finding:
🔴 CRITICAL / 🟠 HIGH / 🟡 MEDIUM / 🔵 LOW — FINDING TITLE
  Resource: namespace/kind/name (container if relevant)
  Risk: what an attacker can do with this misconfiguration
  Fix: exact remediation step (kubectl patch command or minimal YAML diff)

For each known system exception:
✅ EXPECTED: namespace/kind/name — reason (e.g. "kube-proxy requires privileged for iptables management")

✅ CLEAN: list areas with no findings

After a lightweight check, always end with:
💡 TIP: Ask for a "full audit" to check all security areas in depth.

Rules:
- Always include the resource name, namespace, and kind
- Never flag system DaemonSets in kube-system for privileges they are architecturally required to hold
- Prioritise: application privilege escalation > secret exposure > network exposure > system misconfig
- When unsure if a privilege is legitimate, explain the trade-off rather than blanket-flagging it
- No markdown tables or bold — use plain text with the emoji headers above`,
		Examples: []string{
			"Audit RBAC roles for overprivileged accounts",
			"Find application pods running as root or privileged",
			"Check network policies for exposed services",
			"Review secret usage across namespaces",
			"Are there any PSA violations in this cluster?",
			"Is kube-proxy supposed to be privileged?",
		},
	},
	AgentOptimizer: {
		Name:          "k8s-optimizer",
		DisplayName:   "K8s Optimizer",
		Icon:          "⚡",
		Description:   "Resource right-sizing, HPA/VPA recommendations, and cost optimization",
		preferPremium: true,
		Prompt: `You are a Kubernetes resource optimization specialist. Identify waste, risk, and right-sizing opportunities.

MANDATORY TOOL USE:
Whenever the user refers to "the cluster", "my cluster", or any specific workload/namespace,
you MUST call kubectl_exec or get_cluster_status BEFORE answering.
NEVER give generic textbook recommendations — always base findings on actual resource data from the cluster.
If you do not yet know which context to target, call list_clusters first.

ANALYSIS SCOPE (check in this order):
1. Pods with no resource requests or limits (node stability risk)
2. Containers where requests >> actual usage (over-provisioned)
3. Containers where usage approaches limits (under-provisioned, risk of OOM/throttle)
4. Deployments with replicas but near-zero traffic (idle workloads)
5. Node utilization and bin-packing efficiency
6. Missing HPA on variable-traffic deployments

OUTPUT FORMAT (always use this structure):
⚡ OPTIMIZATION SUMMARY: X high, Y medium, Z low impact findings

For each finding:
🔴 HIGH / 🟡 MEDIUM / 🔵 LOW IMPACT — FINDING TITLE
  Workload: namespace/name (container)
  Current: requests=X limits=Y actual usage=Z
  Recommendation: specific change with values
  Estimated saving: CPU/memory freed or risk reduced

📊 NODE EFFICIENCY: overall utilization snapshot

Rules:
- Always quote current values and recommended values side by side
- Separate waste findings (cost) from risk findings (stability)
- No markdown tables or bold — use plain text with the emoji headers above`,
		Examples: []string{
			"Which pods have no resource limits set?",
			"Find over-provisioned workloads in production",
			"Show node CPU and memory utilization",
			"Which deployments would benefit from HPA?",
			"Identify idle or low-traffic services",
		},
	},
	AgentGitOps: {
		Name:          "k8s-gitops",
		DisplayName:   "K8s GitOps",
		Icon:          "🔄",
		Description:   "Flux and ArgoCD sync status, drift detection, and reconciliation diagnostics",
		preferPremium: true,
		Prompt: `You are a GitOps operations specialist for Kubernetes. You monitor sync health and detect drift between Git and the cluster.

MANDATORY TOOL USE:
Whenever the user refers to "the cluster", "my cluster", or any Flux/ArgoCD resource,
you MUST call kubectl_exec or get_cluster_status BEFORE answering.
NEVER assume sync status — always inspect the actual reconciler state from the cluster.
If you do not yet know which context to target, call list_clusters first.

INVESTIGATION ORDER:
1. Check overall sync status for all Flux Kustomizations / ArgoCD Applications
2. For anything not Synced/Healthy: get the exact error and last reconciliation time
3. Identify resources modified outside GitOps (drift)
4. Check suspended or paused reconcilers
5. Review image automation and update policies

OUTPUT FORMAT (always use this structure):
🔄 GITOPS SUMMARY: X synced, Y out-of-sync, Z suspended

For each out-of-sync or failed resource:
🔴 FAILED / 🟡 OUT-OF-SYNC / ⏸️ SUSPENDED — NAME (namespace)
  Type: Kustomization / HelmRelease / Application
  Last sync: timestamp
  Error: exact error message
  Fix: specific reconciliation command or config change

✅ SYNCED: list of healthy resources (one per line)

🔀 DRIFT DETECTED (if any):
  Resource modified outside GitOps with diff summary

Rules:
- Always distinguish desired state (Git) from actual state (cluster)
- Include the last sync timestamp for every resource
- No markdown tables or bold — use plain text with the emoji headers above`,
		Examples: []string{
			"Are all Flux Kustomizations synced?",
			"Show ArgoCD apps that are out of sync",
			"Why is this HelmRelease failing to reconcile?",
			"Find resources modified outside of GitOps",
			"Check Flux image automation status",
		},
	},
	AgentSanitizer: {
		Name:          "k8s-sanitizer",
		DisplayName:   "K8s Sanitizer",
		Icon:          "🧹",
		Description:   "Workload linting, best-practice scoring, and cluster health grading (probes, resource limits, image tags, replicas, container hygiene)",
		preferPremium: true,
		Prompt: `You are a Kubernetes cluster sanitizer and workload linter. You score workloads against best-practice rules and give actionable grades.

ANALYSIS APPROACH:
1. Call sanitize_cluster to get the full findings and score for the target cluster
2. Present the overall cluster grade and score prominently at the top
3. Show per-namespace breakdowns, worst namespaces first
4. Group findings by severity: Critical (CKS-* misconfigurations) first, Major (reliability and operational risks), then Minor (hygiene)
5. Conclude with a prioritised remediation plan

OUTPUT FORMAT (always use this structure):
🧹 SANITIZE REPORT: context-name
Overall: score/100 — GRADE
Scanned N workload(s) | N finding(s): N critical, N major, N minor

For each namespace (worst grade first):
  GRADE namespace/name  score X/100  (N findings: C critical, M major, N minor)

  For each workload within the namespace (worst score first):
    WGRADE Kind/name  score X/100  (N finding(s))
    [CRITICAL] RULE-ID [container]: message
    [MAJOR]    RULE-ID [container]: message
    [MINOR]    RULE-ID [container]: message

🔧 REMEDIATION PRIORITY:
  1. Fix CKS-* findings on APPLICATION workloads first — these are real misconfigurations (privileged, root, host network/PID/path)
  2. Add health probes (BP-001, BP-002) — prevents traffic to broken pods
  3. Set resource limits (BP-003, BP-004) — prevents OOM and node instability
  4. Pin image tags (BP-005) — ensures reproducible deployments
  5. Raise replica counts (BP-006) — improves availability
  6. Harden container filesystem (BP-007, BP-008) — reduces blast radius

Rules:
- IMPORTANT: Show every individual workload with its findings — never collapse, summarise, or omit resources
- Present the tool output content faithfully; do not replace resource details with counts or summaries
- Sort namespaces worst-to-best (lowest score first); within each namespace sort workloads worst-to-best
- For namespaces with no findings, show: ✅ namespace/name — A (0 findings)
- Always explain what each rule checks and why it matters for reliability or operations
- For CKS-* findings on known system components (kube-proxy, CNI plugins like calico-node/cilium/flannel,
  CSI drivers, node-exporter, falco, cloud-controller-manager in kube-system), note that the privilege is
  architecturally required and not a remediation target — suggest verifying no application pods share the pattern.
- No markdown tables or bold — use plain text with the emoji headers above`,
		Examples: []string{
			"Sanitize my cluster and give me a grade",
			"What is the score for the production namespace?",
			"Which workloads are missing health probes?",
			"Show me all BP-006 violations (single-replica deployments)",
			"How do I improve my cluster score from F to B?",
		},
	},
}

// allAgentNames returns a sorted slice of all valid agent names for help text
func allAgentNames() []string {
	return []string{
		string(AgentDefault),
		string(AgentDebugger),
		string(AgentSecurity),
		string(AgentOptimizer),
		string(AgentGitOps),
		string(AgentSanitizer),
	}
}

// ParseAgentType converts a string to an AgentType, returning an error for unknown values
func ParseAgentType(s string) (AgentType, error) {
	switch AgentType(strings.ToLower(s)) {
	case AgentDefault, AgentDebugger, AgentSecurity, AgentOptimizer, AgentGitOps, AgentSanitizer:
		return AgentType(strings.ToLower(s)), nil
	default:
		return AgentDefault, fmt.Errorf("unknown agent %q — valid agents: %s", s, strings.Join(allAgentNames(), ", "))
	}
}

const (
	// Default model selection constants
	defaultModelCostEffective = "gpt-4o-mini" // Cost-effective model for simple queries
	defaultModelPremium       = "gpt-4o"      // Premium model for complex tasks

	// ANSI color codes
	colorReset     = "\033[0m"
	colorRed       = "\033[31m"
	colorGreen     = "\033[32m"
	colorYellow    = "\033[33m"
	colorCyan      = "\033[36m"
	colorBold      = "\033[1m"
	colorDim       = "\033[2m"
	colorUserInput = "\033[38;2;6;182;212m" // Cyan (#06b6d4) for user input, matching kopilot website

	// Spinner animation label
	spinnerLabel = "thinking"

	// fmtErrorBullet is the format string for inline error messages
	fmtErrorBullet = "  %s●%s Error: %v\n"
)

const (
	toolListClusters     = "list_clusters"
	toolGetClusterStatus = "get_cluster_status"
	toolCompareClusters  = "compare_clusters"
	toolCheckAllClusters = "check_all_clusters"
	toolKubectlExec      = "kubectl_exec"
	toolSanitizeCluster  = "sanitize_cluster"
	toolMCPListServers   = "mcp_list_servers"
	toolMCPAddServer     = "mcp_add_server"
	toolMCPDeleteServer  = "mcp_delete_server"
)

// Model configuration - can be overridden by environment variables
var (
	modelCostEffective = getEnvOrDefault("KOPILOT_MODEL_COST_EFFECTIVE", defaultModelCostEffective)
	modelPremium       = getEnvOrDefault("KOPILOT_MODEL_PREMIUM", defaultModelPremium)
)

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// readOnlyCommands lists kubectl commands that are safe in read-only mode
var readOnlyCommands = []string{
	"get", "describe", "logs", "top", "explain",
	"api-resources", "api-versions", "cluster-info",
	"version", "config", "diff", "auth",
}

// agentState holds the runtime state of the agent
type agentState struct {
	mode            ExecutionMode
	outputFormat    OutputFormat
	quotaPercentage float64
	quotaUnlimited  bool
	quotaUsed       float64
	quotaTotal      float64
	selectedAgent   AgentType
	mcpConfigPath   string
	needsMCPReload  bool
	// denyWritesUntilNextPrompt blocks additional write tool calls after the
	// user declines a write confirmation. It is reset when a new prompt arrives.
	denyWritesUntilNextPrompt bool
	// abortCurrentTurn cancels the in-flight model response. It is set just before
	// sending a user prompt and cleared when the turn ends.
	abortCurrentTurn func()
	abortMu          sync.Mutex
}

// setAbortCurrentTurn installs (or clears) the active-turn abort callback.
func (s *agentState) setAbortCurrentTurn(fn func()) {
	s.abortMu.Lock()
	defer s.abortMu.Unlock()
	s.abortCurrentTurn = fn
}

// abortTurnIfActive aborts the currently running model turn, if any.
func (s *agentState) abortTurnIfActive() {
	s.abortMu.Lock()
	fn := s.abortCurrentTurn
	s.abortMu.Unlock()
	if fn != nil {
		fn()
	}
}

// loopDeps groups the immutable runtime dependencies shared across the interactive session loop.
type loopDeps struct {
	ctx         context.Context
	client      *copilot.Client
	k8sProvider *k8s.Provider
	state       *agentState
	isIdle      *bool
}

func isJSONOutput(format OutputFormat) bool {
	return format == OutputJSON
}

// String returns a human-readable name for the execution mode
func (m ExecutionMode) String() string {
	switch m {
	case ModeReadOnly:
		return "read-only"
	case ModeInteractive:
		return "interactive"
	default:
		return "unknown"
	}
}

// isReadOnlyCommand checks if a kubectl command is read-only
func isReadOnlyCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}

	cmd := args[0]
	if cmd == "rollout" {
		if len(args) < 2 {
			return false
		}
		subCmd := args[1]
		return subCmd == "status" || subCmd == "history"
	}

	for _, readOnlyCmd := range readOnlyCommands {
		if cmd == readOnlyCmd {
			return true
		}
	}
	return false
}

// getAvailableContexts returns a comma-separated list of available cluster contexts
func getAvailableContexts(k8sProvider *k8s.Provider) string {
	clusters := k8sProvider.GetClusters()
	contexts := make([]string, len(clusters))
	for i, cluster := range clusters {
		contexts[i] = cluster.Context
	}
	if len(contexts) == 0 {
		return "none"
	}
	return strings.Join(contexts, ", ")
}

// getSystemMessage returns the system message for the Copilot session
func getSystemMessage() string {
	return `You are Kopilot, a Kubernetes cluster operations assistant.

You help users:
- Monitor and manage Kubernetes clusters
- Execute kubectl commands
- Check cluster health and diagnose issues
- Answer questions about cluster resources

When presenting information:
- Use clear, concise language in plain text format
- DO NOT use markdown formatting (no **bold**, no tables, no *** patterns)
- Show tool output directly without reformatting
- Use emoji + uppercase for section headers (e.g., 🔵 STATUS:, ⚠️ POSSIBLE CAUSES:, ✅ NEXT STEPS:)
- Add brief analysis or next steps when helpful

For kubectl operations:
- Always specify the cluster context with --context flag
- Explain what you're doing before executing commands
- Interpret command output for the user

Cluster targeting:
- ALWAYS assume the current cluster context for any operation unless the user explicitly names a different cluster or the request clearly involves multiple clusters (e.g. comparisons, "all clusters", cross-cluster checks).
- Never ask the user which cluster to use when a single-cluster operation is implied — just use the current context.
- Use list_clusters to discover the current context when needed, then proceed immediately.
- "the cluster", "my cluster", "this cluster", "the current cluster", "cluster status", "status of the cluster" → single-cluster operation, use get_cluster_status with the CURRENT context only.
- "all clusters", "every cluster", "all my clusters", "check all", "across clusters", "compare" → multi-cluster operation, use check_all_clusters or compare_clusters.
- When in doubt between single and multi, default to single (current context).

Be helpful, clear, and conversational.`
}

// onMessageEvent handles the final complete assistant message.
func onMessageEvent(event copilot.SessionEvent) {
	if event.Data.Content != nil && *event.Data.Content != "" {
		fmt.Println()
		fmt.Println(*event.Data.Content)
	}
}

// onSessionErrorEvent prints errors from session.error events to the user.
func onSessionErrorEvent(event copilot.SessionEvent) {
	d := event.Data
	msg := "(unknown session error)"
	if d.Message != nil && *d.Message != "" {
		msg = *d.Message
	}
	code := ""
	if d.StatusCode != nil {
		code = fmt.Sprintf(" [status %d]", *d.StatusCode)
	}
	errType := ""
	if d.ErrorType != nil && *d.ErrorType != "" {
		errType = fmt.Sprintf(" [%s]", *d.ErrorType)
	}
	reason := ""
	if d.ErrorReason != nil && *d.ErrorReason != "" {
		reason = fmt.Sprintf(" (reason: %s)", *d.ErrorReason)
	}
	fmt.Fprintf(os.Stderr, "Error%s%s: %s%s\n", code, errType, msg, reason)
}

// onUsageEvent records quota information from usage snapshots.
func onUsageEvent(event copilot.SessionEvent, state *agentState) {
	if event.Data.QuotaSnapshots == nil {
		return
	}
	snapshot, exists := event.Data.QuotaSnapshots["premium_interactions"]
	if exists && snapshot.RemainingPercentage >= 0 {
		state.quotaPercentage = snapshot.RemainingPercentage
		state.quotaUnlimited = snapshot.IsUnlimitedEntitlement
		state.quotaUsed = snapshot.UsedRequests
		state.quotaTotal = snapshot.EntitlementRequests
	}
}

// setupSessionEventHandler creates and returns an event handler for the session.
func setupSessionEventHandler(session *copilot.Session, isIdlePtr *bool, state *agentState) {
	session.On(func(event copilot.SessionEvent) {
		switch event.Type {
		case "assistant.message":
			onMessageEvent(event)
		case "session.error":
			onSessionErrorEvent(event)
		case "session.idle":
			*isIdlePtr = true
			state.setAbortCurrentTurn(nil)
		case "assistant.usage":
			onUsageEvent(event, state)
		}
	})
}

// defaultExamples is the pool of general-purpose example prompts shown at startup.
var defaultExamples = []string{
	"Show me all my clusters",
	"What's the health status of all clusters?",
	"List all pods in the default namespace",
	"Compare production and staging clusters",
	"Check if all nodes are ready",
	"Show me failing pods",
	"Get status of cluster production",
	"How many pods are running?",
	"List all namespaces",
	"Show me pod resource usage",
	"Check health of all clusters in parallel",
	"What version of Kubernetes am I running?",
	"Show me recent events",
	"List all services",
	"Check node capacity",
	"Show deployments in kube-system",
}

// getRandomExamples returns a random selection of example prompts.
func getRandomExamples(count int) []string {
	shuffled := make([]string, len(defaultExamples))
	copy(shuffled, defaultExamples)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	if count > len(shuffled) {
		count = len(shuffled)
	}
	return shuffled[:count]
}

// Run starts the Copilot agent with Kubernetes cluster tools.
// mcpConfigPath is the path to the JSON file storing MCP server configurations;
// pass an empty string to use the default (~/.kopilot/mcp.json).
func Run(k8sProvider *k8s.Provider, mode ExecutionMode, outputFormat OutputFormat, agentType AgentType, mcpConfigPath string) error {
	// Configure logging to stderr to avoid interfering with stdio-based JSON-RPC
	log.SetOutput(os.Stderr)

	if mcpConfigPath == "" {
		mcpConfigPath = DefaultMCPConfigPath()
	}

	// Initialize agent state
	state := &agentState{
		mode:            mode,
		outputFormat:    outputFormat,
		quotaPercentage: -1,
		selectedAgent:   agentType,
		mcpConfigPath:   mcpConfigPath,
	}

	// Create a cancellable context for the entire agent lifecycle
	// This allows graceful shutdown on Ctrl+C or other signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create and start Copilot client
	client, err := createAndStartClient(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Stop(); err != nil {
			log.Printf("Warning: error stopping Copilot client: %v", err)
		}
	}()

	// Create initial session with cost-effective model
	session, err := createSessionWithModel(ctx, client, k8sProvider, state, modelCostEffective)
	if err != nil {
		return err
	}
	defer func() {
		if destroyErr := session.Destroy(); destroyErr != nil {
			log.Printf("Warning: failed to destroy session: %v", destroyErr)
		}
	}()

	// Set up event handling
	var isIdle bool
	setupSessionEventHandler(session, &isIdle, state)

	if !isJSONOutput(outputFormat) {
		printBanner(k8sProvider, mode, agentType, mcpConfigPath)
	}

	// Mark as idle so user can start typing immediately
	isIdle = true

	// Interactive loop with session management
	deps := &loopDeps{
		ctx:         ctx,
		client:      client,
		k8sProvider: k8sProvider,
		state:       state,
		isIdle:      &isIdle,
	}
	return interactiveLoopWithModelSelection(deps, session)
}

// printBanner prints the ASCII art logo and startup status to stdout.
func printBanner(k8sProvider *k8s.Provider, mode ExecutionMode, agentType AgentType, mcpConfigPath string) {
	fmt.Println()
	fmt.Printf("%s  $    $$                       $     \"\"$$               $$           %s\n", colorCyan, colorReset)
	fmt.Printf("%s  $  $$     #$$$    $ $$$     $$$       $$      $$$1   $$$$$$$   %s[))%s  \n", colorCyan, colorRed, colorReset)
	fmt.Printf("%s  $$$$     $    $   $d   $      $       $$     $    $    $$      %s)))%s  \n", colorCyan, colorRed, colorReset)
	fmt.Printf("%s  $   $    $    $[  $    $;     $       $$    $$    $    B$      %s)))%s  \n", colorCyan, colorRed, colorReset)
	fmt.Printf("%s  $    $$   $$j$$   $$$|$$   $$$$$$$     $$$   $$\\$$      $$$$   %s)))%s  \n", colorCyan, colorRed, colorReset)
	fmt.Printf("%s                    $                                            %s[))%s  \n", colorCyan, colorRed, colorReset)
	fmt.Println()
	fmt.Printf("               %sKubernetes Operations Assistant%s\n", colorDim, colorReset)
	fmt.Printf("                         %s%s%s\n", colorDim, AppVersion, colorReset)
	fmt.Println()

	clusters := k8sProvider.GetClusters()
	currentCtx := k8sProvider.GetCurrentContext()
	fmt.Printf("  %s●%s Connected to %d cluster(s)\n", colorGreen, colorReset, len(clusters))
	if currentCtx != "" {
		fmt.Printf("  %s●%s Active context: %s%s%s\n", colorCyan, colorReset, colorCyan, currentCtx, colorReset)
	}

	printBannerMode(mode)
	printBannerAgent(agentType)
	printBannerMCP(mcpConfigPath)
	printBannerExamples(agentType)
}

// printBannerMode prints the current execution mode line.
func printBannerMode(mode ExecutionMode) {
	modeIcon, modeColor, modeText := "🔒", colorYellow, "read-only"
	if mode == ModeInteractive {
		modeIcon, modeColor, modeText = "🔓", colorGreen, "interactive"
	}
	fmt.Printf("  %s●%s Mode: %s%s %s%s\n", modeColor, colorReset, modeIcon, modeColor, modeText, colorReset)
}

// printBannerAgent prints the active specialist agent line, if one is selected.
func printBannerAgent(agentType AgentType) {
	if agentType == AgentDefault {
		return
	}
	def := agentDefinitions[agentType]
	fmt.Printf("  %s●%s Agent: %s%s %s%s — %s\n", colorCyan, colorReset, colorCyan, def.Icon, def.DisplayName, colorReset, def.Description)
}

// printBannerMCP prints a summary of configured MCP servers at startup.
func printBannerMCP(mcpConfigPath string) {
	servers, err := listMCPServers(mcpConfigPath)
	if err != nil || len(servers) == 0 {
		return
	}
	fmt.Printf("  %s●%s MCP servers: %s", colorCyan, colorReset, colorDim)
	names := make([]string, len(servers))
	for i, s := range servers {
		names[i] = s.Name
	}
	fmt.Printf("%s%s\n", strings.Join(names, ", "), colorReset)
}

// printBannerExamples prints the "Try asking" prompt examples.
func printBannerExamples(agentType AgentType) {
	examples := getRandomExamples(3)
	if agentType != AgentDefault {
		def := agentDefinitions[agentType]
		examples = def.Examples
		if len(examples) > 3 {
			examples = examples[:3]
		}
	}
	fmt.Println()
	fmt.Printf("  %sTry asking:%s\n", colorDim, colorReset)
	for _, example := range examples {
		fmt.Printf("    %s•%s %s\"%s\"%s\n", colorCyan, colorReset, colorDim, example, colorReset)
	}
	fmt.Println()
	fmt.Printf("  %sType your request to get started. Enter 'exit' to quit.%s\n", colorDim, colorReset)
	fmt.Printf("  %sHint: /help to see all commands or /agent list to see specialist agents.%s\n", colorDim, colorReset)
	fmt.Println()
}

// createAndStartClient creates and starts the Copilot client.
// The SDK uses the embedded CLI binary (bundled via `go tool bundler`)
// or falls back to the `copilot` CLI in PATH.
func createAndStartClient(ctx context.Context) (*copilot.Client, error) {
	// Get current working directory for CLI context
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Let the SDK auto-detect the CLI binary.
	// The embedded CLI (from `go tool bundler`) takes priority,
	// then COPILOT_CLI_PATH env var, then `copilot` in PATH.
	client := copilot.NewClient(&copilot.ClientOptions{
		Cwd:      cwd,
		LogLevel: "error", // Reduce noise in logs
	})

	log.Println("Starting Copilot client...")
	if err := client.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start copilot client: %w\n\nTip: Ensure GitHub Copilot CLI is properly set up and authenticated.\nRun 'go tool bundler' to embed the correct CLI version", err)
	}

	log.Println("Copilot client started successfully")
	return client, nil
}

// buildCustomAgents converts all agentDefinitions into SDK CustomAgentConfig entries
func buildCustomAgents() []copilot.CustomAgentConfig {
	configs := make([]copilot.CustomAgentConfig, 0, len(agentDefinitions))
	for _, def := range agentDefinitions {
		configs = append(configs, copilot.CustomAgentConfig{
			Name:        def.Name,
			DisplayName: def.DisplayName,
			Description: def.Description,
			Prompt:      def.Prompt,
			Tools:       def.Tools, // nil means all tools are available
		})
	}
	return configs
}

// buildSystemMessage composes the full system message, optionally including the
// specialist prompt for the currently selected agent persona.
//
// When a specialist is active, a bridging directive is inserted between the base
// and the specialist prompt. This ensures the model applies the specialist lens
// to ALL requests — including generic ones like "analyze the cluster" or "check
// the current cluster" — rather than falling back to the generalist persona.
func buildSystemMessage(agentType AgentType) string {
	base := getSystemMessage()
	if agentType == AgentDefault {
		return base
	}
	def, ok := agentDefinitions[agentType]
	if !ok {
		return base
	}

	// The bridge directive is intentionally placed between the base and the
	// specialist prompt so the model understands the active mode before reading
	// the specialist's detailed instructions.
	bridge := fmt.Sprintf(
		"ACTIVE SPECIALIST: %s %s\n"+
			"You are currently operating in %s specialist mode. "+
			"For ALL user requests — including broad or generic ones such as \"analyze\", "+
			"\"check\", \"review\", \"assess\", \"what do you think about my cluster\", "+
			"or any request that does not specify a domain — you MUST interpret and answer "+
			"exclusively through the %s specialist lens defined below. "+
			"Never give a generic Kubernetes overview when a specialist agent is active; "+
			"always frame every analysis, finding, and recommendation through that specialist's domain.",
		def.Icon, def.DisplayName,
		def.DisplayName,
		def.DisplayName,
	)

	return base + "\n\n" + bridge + "\n\n" + def.Prompt
}

// loadMCPServersForSession reads the MCP config and converts it to the SDK map type.
// Returns nil when no servers are configured so that MCPServers is omitted from the session.
func loadMCPServersForSession(cfgPath string) map[string]copilot.MCPServerConfig {
	servers, err := listMCPServers(cfgPath)
	if err != nil || len(servers) == 0 {
		return nil
	}
	m := make(map[string]copilot.MCPServerConfig, len(servers))
	for _, s := range servers {
		m[s.Name] = copilot.MCPServerConfig{
			"type": s.Type,
			"url":  s.URL,
		}
	}
	return m
}

// createSessionWithModel creates a new Copilot session with specified model
func createSessionWithModel(ctx context.Context, client *copilot.Client, k8sProvider *k8s.Provider, state *agentState, model string) (*copilot.Session, error) {
	tools := defineTools(k8sProvider, state)
	systemMessage := buildSystemMessage(state.selectedAgent)
	mcpServers := loadMCPServersForSession(state.mcpConfigPath)

	session, err := client.CreateSession(ctx, &copilot.SessionConfig{
		Model:               model,
		Streaming:           false,
		Tools:               tools,
		CustomAgents:        buildCustomAgents(),
		OnPermissionRequest: copilot.PermissionHandler.ApproveAll,
		SystemMessage: &copilot.SystemMessageConfig{
			Mode:    "replace",
			Content: systemMessage,
		},
		MCPServers: mcpServers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	agentLabel := string(state.selectedAgent)
	log.Printf("Session created with model: %s, agent: %s", model, agentLabel)
	return session, nil
}

// selectModelForQuery determines the best model based on query complexity, intent, and active agent.
// Specialist agents always use the premium model — their reasoning tasks benefit from higher
// model capacity regardless of how simple the query text appears.
func selectModelForQuery(query string, agentType AgentType) string {
	// Specialist agents always warrant the premium model
	if def, ok := agentDefinitions[agentType]; ok && def.preferPremium {
		return modelPremium
	}
	lowerQuery := strings.ToLower(query)

	// High-priority/complex tasks - use premium model
	troubleshootingKeywords := []string{
		"why", "troubleshoot", "debug", "investigate", "error", "fail",
		"crash", "not working", "broken", "issue", "problem", "wrong",
		"fix", "solve", "diagnose", "analyze", "explain", "understand",
	}

	for _, keyword := range troubleshootingKeywords {
		if strings.Contains(lowerQuery, keyword) {
			return modelPremium // Premium model for troubleshooting
		}
	}

	// Simple queries and read-only kubectl operations - use cost-effective model first
	simpleKeywords := []string{
		"list", "show", "get", "describe", "status", "health",
		"what", "how many", "check", "logs", "log", "exec", "top",
		"view", "display", "see", "kubectl get", "kubectl describe", "kubectl logs",
	}

	for _, keyword := range simpleKeywords {
		if strings.Contains(lowerQuery, keyword) {
			return modelCostEffective // Cost-effective for simple queries
		}
	}

	// Complex kubectl operations that modify state - use premium model
	kubectlComplexKeywords := []string{
		"scale", "restart", "delete", "apply", "patch", "edit",
		"rollback", "drain", "cordon", "taint", "create", "update",
	}

	for _, keyword := range kubectlComplexKeywords {
		if strings.Contains(lowerQuery, keyword) {
			return modelPremium // Better reasoning for operations
		}
	}

	// Default to cost-effective model
	return modelCostEffective
}

// spinnerFrames are the braille dot animation frames shown while the AI is thinking.
var spinnerFrames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

// spinnerPaused is set to 1 while a tool needs exclusive terminal access (e.g. interactive confirmation prompt).
var spinnerPaused atomic.Int32

// pauseSpinner suppresses the spinner and clears any in-progress spinner line.
// The caller must call the returned resume function when done.
func pauseSpinner() func() {
	spinnerPaused.Store(1)
	time.Sleep(120 * time.Millisecond) // let any in-flight tick finish
	fmt.Printf("\r\033[K")             // erase whatever the spinner last drew
	return func() { spinnerPaused.Store(0) }
}

// startSpinner launches an animated spinner in a goroutine and returns a stop function.
// The caller must call the returned function to stop the spinner and erase the line.
func startSpinner() func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-done:
				fmt.Printf("\r\033[K") // erase spinner line
				return
			case <-ticker.C:
				if spinnerPaused.Load() == 0 {
					fmt.Printf("\r  %s%s%s %s...", colorCyan, spinnerFrames[i%len(spinnerFrames)], colorReset, spinnerLabel)
				}
				i++
			}
		}
	}()
	return func() {
		close(done)
		time.Sleep(20 * time.Millisecond) // give the goroutine time to erase the line
	}
}

// waitForIdle waits until the session is idle, with a ceiling of 5 minutes
// to prevent an infinite hang if session.idle never fires (e.g. on SDK error).
func waitForIdle(isIdle *bool) {
	deadline := time.Now().Add(5 * time.Minute)
	for !*isIdle {
		if time.Now().After(deadline) {
			*isIdle = true // unblock; the session is assumed dead
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// waitForIdleWithSpinner waits for the session to become idle, showing an animated
// spinner if the session is not already idle (i.e. the AI is still responding).
func waitForIdleWithSpinner(isIdle *bool) {
	if *isIdle {
		return
	}
	stop := startSpinner()
	deadline := time.Now().Add(5 * time.Minute)
	for !*isIdle {
		if time.Now().After(deadline) {
			*isIdle = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	stop()
}

// cyanPainter implements readline.Painter to colour typed input text cyan
// without affecting the prompt itself.
type cyanPainter struct{}

func (p *cyanPainter) Paint(line []rune, _ int) []rune {
	if len(line) == 0 {
		return line
	}
	prefix := []rune(colorUserInput)
	suffix := []rune(colorReset)
	result := make([]rune, 0, len(prefix)+len(line)+len(suffix))
	result = append(result, prefix...)
	result = append(result, line...)
	result = append(result, suffix...)
	return result
}

// rlPromptString returns a readline-compatible prompt string.
// ANSI escape sequences are wrapped in \x01…\x02 so readline does not count
// them toward the visible line length, preventing cursor misalignment.
// Input text colouring is handled by cyanPainter, not the prompt string.
func rlPromptString(state *agentState) string {
	wrap := func(seq string) string { return "\x01" + seq + "\x02" }
	if isJSONOutput(state.outputFormat) || state.quotaUnlimited || state.quotaPercentage < 0 {
		return "❯ "
	}
	pct := state.quotaPercentage
	var col, indicator string
	switch {
	case pct <= 5:
		col = colorRed
		indicator = fmt.Sprintf("[⚠ %.0f%%]", pct)
	case pct <= 20:
		col = colorYellow
		indicator = fmt.Sprintf("[%.0f%%]", pct)
	default:
		col = colorDim
		indicator = fmt.Sprintf("[%.0f%%]", pct)
	}
	return wrap(col) + indicator + wrap(colorReset) + " ❯ "
}

// newReadlineInstance creates a readline instance with in-session history.
func newReadlineInstance() (*readline.Instance, error) {
	return readline.NewEx(&readline.Config{
		HistoryLimit:           500,
		InterruptPrompt:        "^C",
		EOFPrompt:              "exit",
		HistorySearchFold:      true,
		DisableAutoSaveHistory: false,
		Painter:                &cyanPainter{},
	})
}

// readUserInput reads and trims user input via the readline instance.
// Ctrl+C clears the current line and continues; Ctrl+D exits gracefully.
func readUserInput(rl *readline.Instance, state *agentState) (string, error) {
	rl.SetPrompt(rlPromptString(state))
	input, err := rl.Readline()
	fmt.Print(colorReset) // reset typed-text colour
	if err == readline.ErrInterrupt {
		// Ctrl+C — cancel current line, continue loop
		return "", nil
	}
	if err == io.EOF {
		// Ctrl+D — treat as exit
		return "exit", nil
	}
	if err != nil {
		return "", fmt.Errorf("error reading input: %w", err)
	}
	return strings.TrimSpace(input), nil
}

// isExitCommand checks if the input is an exit command
func isExitCommand(input string) bool {
	lower := strings.ToLower(input)
	return lower == "exit" || lower == "quit"
}

// isUnknownSlashCommand returns true when the input starts with '/' but is not
// handled by any known command dispatcher (handleModeSwitch, handleAgentCommand, handleMCPCommand).
func isUnknownSlashCommand(input string) bool {
	if !strings.HasPrefix(input, "/") {
		return false
	}
	lower := strings.TrimSpace(strings.ToLower(input))
	// Known prefixes — keep in sync with handleModeSwitch, handleAgentCommand, handleMCPCommand.
	known := []string{"/help", "/mode", "/status", "/readonly", "/interactive", "/agent", "/mcp"}
	for _, prefix := range known {
		if lower == prefix || strings.HasPrefix(lower, prefix+" ") {
			return false
		}
	}
	return true
}

// printHelpMessage displays all available runtime commands.
func printHelpMessage(state *agentState) {
	agentNames := strings.Join(allAgentNames(), " | ")
	fmt.Println()
	fmt.Printf("  %s━━ Kopilot Commands ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorCyan, colorReset)
	fmt.Println()
	fmt.Printf("  %sSession%s\n", colorDim, colorReset)
	fmt.Printf("    %s/help%s             show this help message\n", colorCyan, colorReset)
	fmt.Printf("    %sexit%s, %squit%s        exit Kopilot\n", colorCyan, colorReset, colorCyan, colorReset)
	fmt.Println()
	fmt.Printf("  %sExecution Mode%s\n", colorDim, colorReset)
	fmt.Printf("    %s/mode%s, %s/status%s       show current execution mode\n", colorCyan, colorReset, colorCyan, colorReset)
	fmt.Printf("    %s/readonly%s [on]       switch to 🔒 read-only mode (blocks write operations)\n", colorCyan, colorReset)
	fmt.Printf("    %s/interactive%s [on]    switch to 🔓 interactive mode (prompts before writes)\n", colorCyan, colorReset)
	fmt.Println()
	fmt.Printf("  %sSpecialist Agents%s\n", colorDim, colorReset)
	fmt.Printf("    %s/agent%s             show active agent and available roster\n", colorCyan, colorReset)
	fmt.Printf("    %s/agent list%s        same as /agent\n", colorCyan, colorReset)
	fmt.Printf("    %s/agent <name>%s      switch agent  [ %s ]\n", colorCyan, colorReset, agentNames)
	fmt.Println()
	fmt.Printf("  %sMCP Servers%s\n", colorDim, colorReset)
	fmt.Printf("    %s/mcp list%s              list configured MCP servers\n", colorCyan, colorReset)
	fmt.Printf("    %s/mcp add <name> <url>%s  add or update an MCP server\n", colorCyan, colorReset)
	fmt.Printf("    %s/mcp delete <name>%s     remove an MCP server\n", colorCyan, colorReset)
	fmt.Println()
	fmt.Printf("  %sCurrent mode: %s  |  Active agent: %s%s\n", colorDim, state.mode, state.selectedAgent, colorReset)
	fmt.Println()
}

// handleModeSwitch handles runtime mode switching commands
// Returns true if the input was a mode switch command
func handleModeSwitch(input string, state *agentState) bool {
	lower := strings.TrimSpace(strings.ToLower(input))

	switch lower {
	case "/help":
		printHelpMessage(state)
		return true

	case "/readonly", "/readonly on":
		if state.mode == ModeReadOnly {
			fmt.Printf("  %s●%s Already in read-only mode\n", colorYellow, colorReset)
		} else {
			state.mode = ModeReadOnly
			fmt.Printf("  %s●%s Switched to %s🔒 read-only%s mode\n", colorGreen, colorReset, colorYellow, colorReset)
		}
		return true

	case "/interactive", "/interactive on":
		if state.mode == ModeInteractive {
			fmt.Printf("  %s●%s Already in interactive mode\n", colorYellow, colorReset)
		} else {
			state.mode = ModeInteractive
			fmt.Printf("  %s●%s Switched to %s🔓 interactive%s mode\n", colorGreen, colorReset, colorGreen, colorReset)
		}
		return true

	case "/mode", "/status":
		modeIcon := "🔒"
		modeColor := colorYellow
		if state.mode == ModeInteractive {
			modeIcon = "🔓"
			modeColor = colorGreen
		}
		fmt.Printf("  %s●%s Current mode: %s%s %s%s\n", modeColor, colorReset, modeIcon, modeColor, state.mode, colorReset)
		return true
	}

	return false
}

// printAgentList displays the active agent and the available agent roster.
func printAgentList(state *agentState) {
	currentDef, isSpecialist := agentDefinitions[state.selectedAgent]
	if state.selectedAgent == AgentDefault || !isSpecialist {
		fmt.Printf("  %s●%s Active agent: %sdefault%s (standard Kopilot persona)\n", colorCyan, colorReset, colorCyan, colorReset)
	} else {
		fmt.Printf("  %s●%s Active agent: %s%s %s%s\n", colorCyan, colorReset, colorCyan, currentDef.Icon, currentDef.DisplayName, colorReset)
	}
	fmt.Printf("\n  %sAvailable agents:%s\n", colorDim, colorReset)
	fmt.Printf("    %s•%s %sdefault%s  — standard Kopilot persona\n", colorCyan, colorReset, colorDim, colorReset)
	for _, at := range []AgentType{AgentDebugger, AgentSecurity, AgentOptimizer, AgentGitOps, AgentSanitizer} {
		def := agentDefinitions[at]
		marker := " "
		if state.selectedAgent == at {
			marker = "*"
		}
		fmt.Printf("    %s%s%s %s%-10s%s — %s\n", colorCyan, marker, colorReset, colorDim, string(at), colorReset, def.Description)
	}
	fmt.Println()
}

// formatAgentSwitchMessage returns the confirmation line shown after switching to an agent.
func formatAgentSwitchMessage(newAgent AgentType) string {
	if newAgent == AgentDefault {
		return fmt.Sprintf("  %s●%s Switched to %sdefault%s agent", colorGreen, colorReset, colorCyan, colorReset)
	}
	def := agentDefinitions[newAgent]
	return fmt.Sprintf("  %s●%s Switched to %s%s %s%s", colorGreen, colorReset, colorCyan, def.Icon, def.DisplayName, colorReset)
}

// formatAlreadyUsingAgent returns the message shown when the requested agent is already active.
func formatAlreadyUsingAgent(at AgentType) string {
	if at == AgentDefault {
		return fmt.Sprintf("  %s●%s Already using the default agent", colorYellow, colorReset)
	}
	def := agentDefinitions[at]
	return fmt.Sprintf("  %s●%s Already using %s%s %s%s", colorYellow, colorReset, colorCyan, def.Icon, def.DisplayName, colorReset)
}

// handleAgentCommand checks whether the input is an /agent command.
// Returns (isCommand, newAgentType, error). newAgentType is only valid when
// isCommand is true and error is nil and the agent actually changed.
func handleAgentCommand(input string, state *agentState) (isCommand bool, newAgent AgentType, err error) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(strings.ToLower(trimmed), "/agent") {
		return false, AgentDefault, nil
	}

	parts := strings.Fields(trimmed)

	// "/agent" or "/agent list" → show status and roster
	if len(parts) == 1 || (len(parts) == 2 && strings.ToLower(parts[1]) == "list") {
		printAgentList(state)
		return true, state.selectedAgent, nil
	}

	// "/agent <name>" → validate and return the target agent
	if len(parts) == 2 {
		return resolveSwitchTarget(parts[1], state)
	}

	return true, AgentDefault, fmt.Errorf("usage: /agent [list | %s]", strings.Join(allAgentNames(), " | "))
}

// resolveSwitchTarget parses and validates the target agent name for /agent <name>.
func resolveSwitchTarget(name string, state *agentState) (bool, AgentType, error) {
	parsed, err := ParseAgentType(name)
	if err != nil {
		return true, AgentDefault, err
	}
	if parsed == state.selectedAgent {
		fmt.Println(formatAlreadyUsingAgent(parsed))
		return true, parsed, nil
	}
	return true, parsed, nil
}

// switchToModel replaces the current session with a new one using the given model.
// All runtime dependencies are supplied via deps.
func switchToModel(deps *loopDeps, oldSession *copilot.Session, newModel string) (*copilot.Session, error) {
	if err := oldSession.Destroy(); err != nil {
		log.Printf("Warning: failed to destroy old session: %v", err)
	}

	newSession, err := createSessionWithModel(deps.ctx, deps.client, deps.k8sProvider, deps.state, newModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create new session: %w", err)
	}

	setupSessionEventHandler(newSession, deps.isIdle, deps.state)
	waitForIdle(deps.isIdle)

	return newSession, nil
}

// turnState holds the mutable per-iteration state of the interactive loop.
type turnState struct {
	session *copilot.Session
	model   string
}

// processTurn handles a single interactive turn: read input, dispatch commands, send to model.
// Returns (exit=true) when the user has chosen to quit, or an error on failure.
func processTurn(deps *loopDeps, rl *readline.Instance, ts *turnState) (exit bool, err error) {
	if isJSONOutput(deps.state.outputFormat) {
		waitForIdle(deps.isIdle)
	} else {
		waitForIdleWithSpinner(deps.isIdle)
	}

	input, err := readUserInput(rl, deps.state)
	if err != nil {
		return false, err
	}

	if input == "" {
		return false, nil
	}

	if isExitCommand(input) {
		fmt.Println("")
		return true, nil
	}

	// A new user prompt starts a new decision cycle; allow write confirmations again.
	deps.state.denyWritesUntilNextPrompt = false

	if handleModeSwitch(input, deps.state) {
		return false, nil
	}

	if handled, err := dispatchAgentCommand(deps, input, ts); handled {
		return false, err
	}

	if handled, err := dispatchMCPCommand(deps, input, ts); handled {
		return false, err
	}

	// Reject unrecognised slash commands rather than forwarding them to the model.
	if isUnknownSlashCommand(input) {
		cmd := strings.Fields(input)[0]
		fmt.Printf("  %s●%s Unknown command %s%s%s — type %s/help%s to see available commands\n",
			colorRed, colorReset, colorBold, cmd, colorReset, colorCyan, colorReset)
		return false, nil
	}

	if err := sendToModel(deps, ts, input); err != nil {
		deps.state.setAbortCurrentTurn(nil)
		return false, err
	}

	// If an MCP tool modified the config during this turn, reload the session so
	// the next turn connects to the updated set of MCP servers.
	if deps.state.needsMCPReload {
		deps.state.needsMCPReload = false
		fmt.Printf("  %s●%s Reloading session with updated MCP servers...%s\n", colorCyan, colorReset, "")
		newSession, err := switchToModel(deps, ts.session, ts.model)
		if err != nil {
			return false, err
		}
		ts.session = newSession
	}
	return false, nil
}

// dispatchMCPCommand processes /mcp commands.
// Returns (true, err) when the input was an MCP command (whether it succeeded or not).
func dispatchMCPCommand(deps *loopDeps, input string, ts *turnState) (bool, error) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(strings.ToLower(trimmed), "/mcp") {
		return false, nil
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 1 || (len(parts) == 2 && strings.ToLower(parts[1]) == "list") {
		printMCPList(deps.state.mcpConfigPath)
		return true, nil
	}

	subCmd := strings.ToLower(parts[1])
	switch subCmd {
	case "add":
		return true, mcpHandleAdd(deps, parts, ts)
	case "delete", "remove":
		return true, mcpHandleDeleteRemove(deps, parts, ts)
	default:
		fmt.Printf("  %s●%s Unknown /mcp sub-command %s%q%s — use: list, add, delete\n",
			colorRed, colorReset, colorBold, subCmd, colorReset)
		return true, nil
	}
}

// mcpHandleAdd processes the "/mcp add <name> <url>" sub-command.
func mcpHandleAdd(deps *loopDeps, parts []string, ts *turnState) error {
	if len(parts) < 4 {
		fmt.Printf("  %s●%s Usage: /mcp add <name> <url>\n", colorRed, colorReset)
		return nil
	}
	name, url := parts[2], parts[3]
	if err := addMCPServer(deps.state.mcpConfigPath, MCPServerConfig{Name: name, Type: mcpHTTPType, URL: url}); err != nil {
		fmt.Printf(fmtErrorBullet, colorRed, colorReset, err)
		return nil
	}
	fmt.Printf("  %s●%s Added MCP server %s%s%s — reloading session...\n", colorGreen, colorReset, colorCyan, name, colorReset)
	newSession, err := switchToModel(deps, ts.session, ts.model)
	if err != nil {
		return err
	}
	ts.session = newSession
	return nil
}

// mcpHandleDeleteRemove processes the "/mcp delete <name>" sub-command.
func mcpHandleDeleteRemove(deps *loopDeps, parts []string, ts *turnState) error {
	if len(parts) < 3 {
		fmt.Printf("  %s●%s Usage: /mcp delete <name>\n", colorRed, colorReset)
		return nil
	}
	name := parts[2]
	if err := deleteMCPServer(deps.state.mcpConfigPath, name); err != nil {
		fmt.Printf(fmtErrorBullet, colorRed, colorReset, err)
		return nil
	}
	fmt.Printf("  %s●%s Removed MCP server %s%s%s — reloading session...\n", colorGreen, colorReset, colorCyan, name, colorReset)
	newSession, err := switchToModel(deps, ts.session, ts.model)
	if err != nil {
		return err
	}
	ts.session = newSession
	return nil
}

// printMCPList prints all configured MCP servers to stdout.
func printMCPList(cfgPath string) {
	servers, err := listMCPServers(cfgPath)
	if err != nil {
		fmt.Printf("  %s●%s Error reading MCP config: %v\n", colorRed, colorReset, err)
		return
	}
	fmt.Println()
	fmt.Printf("  %s━━ MCP Servers (%s) ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorCyan, cfgPath, colorReset)
	fmt.Println()
	if len(servers) == 0 {
		fmt.Printf("  %sNo MCP servers configured.%s\n", colorDim, colorReset)
		fmt.Printf("  %sUse /mcp add <name> <url> to add one.%s\n", colorDim, colorReset)
	} else {
		for i, s := range servers {
			fmt.Printf("  %s[%d]%s %s%s%s\n", colorCyan, i+1, colorReset, colorCyan, s.Name, colorReset)
			fmt.Printf("      Type: %s\n", s.Type)
			fmt.Printf("      URL:  %s\n", s.URL)
		}
	}
	fmt.Println()
}

// dispatchAgentCommand processes /agent commands.
// Returns (true, err) when the input was an agent command (whether it succeeded or not).
func dispatchAgentCommand(deps *loopDeps, input string, ts *turnState) (bool, error) {
	isAgentCmd, newAgent, agentErr := handleAgentCommand(input, deps.state)
	if !isAgentCmd {
		return false, nil
	}
	if agentErr != nil {
		fmt.Printf(fmtErrorBullet, colorRed, colorReset, agentErr)
		return true, nil
	}
	newSession, err := applyAgentSwitch(deps, newAgent, ts.session, ts.model)
	if err != nil {
		return true, err
	}
	ts.session = newSession
	return true, nil
}

// longRunningKeywords are phrases that signal a broad, multi-step analysis
// that will invoke many tool calls and take significantly longer than a simple query.
var longRunningKeywords = []string{
	"full", "complete", "all", "entire", "everything", "thorough", "comprehensive",
	"audit", "analyze", "analyse", "analysis", "review", "assess", "assessment",
	"scan", "check everything", "check all", "deep dive", "deep-dive",
	"top to bottom", "end to end",
}

// isLongRunningQuery returns true when the prompt looks like a broad analysis
// that will fan out into many sequential tool calls.
func isLongRunningQuery(input string, agentType AgentType) bool {
	// Specialist agents doing any form of analysis are inherently multi-step.
	if agentType != AgentDefault {
		lower := strings.ToLower(input)
		for _, kw := range longRunningKeywords {
			if strings.Contains(lower, kw) {
				return true
			}
		}
	}
	return false
}

// printLongRunningWarning shows a one-line heads-up before a query that will take a while.
func printLongRunningWarning(agentType AgentType) {
	label := "analysis"
	if agentType != AgentDefault {
		def := agentDefinitions[agentType]
		label = def.DisplayName + " analysis"
	}
	fmt.Printf("  %s⏳ This %s may take a while — running multiple checks against your cluster...%s\n",
		colorDim, label, colorReset)
}

// sendToModel selects the best model for the query and sends it, updating ts as needed.
func sendToModel(deps *loopDeps, ts *turnState, input string) error {
	if optimalModel := selectModelForQuery(input, deps.state.selectedAgent); optimalModel != ts.model {
		newSession, err := switchToModel(deps, ts.session, optimalModel)
		if err != nil {
			return err
		}
		ts.session = newSession
		ts.model = optimalModel
	}
	if !isJSONOutput(deps.state.outputFormat) && isLongRunningQuery(input, deps.state.selectedAgent) {
		printLongRunningWarning(deps.state.selectedAgent)
	}
	*deps.isIdle = false
	deps.state.setAbortCurrentTurn(func() {
		if abortErr := ts.session.Abort(deps.ctx); abortErr != nil {
			log.Printf("Warning: failed to abort current turn: %v", abortErr)
		}
	})
	_, err := ts.session.Send(deps.ctx, copilot.MessageOptions{Prompt: input})
	if err != nil {
		deps.state.setAbortCurrentTurn(nil)
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

// interactiveLoopWithModelSelection handles interactive conversation with dynamic model selection.
func interactiveLoopWithModelSelection(deps *loopDeps, initialSession *copilot.Session) error {
	rl, err := newReadlineInstance()
	if err != nil {
		return fmt.Errorf("failed to initialise readline: %w", err)
	}
	defer func() {
		if closeErr := rl.Close(); closeErr != nil {
			log.Printf("Warning: failed to close readline: %v", closeErr)
		}
	}()

	ts := &turnState{session: initialSession, model: modelCostEffective}
	for {
		exit, err := processTurn(deps, rl, ts)
		if err != nil {
			return err
		}
		if exit {
			return nil
		}
	}
}

// applyAgentSwitch applies a validated /agent switch command, recreating the session if needed.
// Returns the (possibly new) current session.
func applyAgentSwitch(deps *loopDeps, newAgent AgentType, currentSession *copilot.Session, currentModel string) (*copilot.Session, error) {
	if newAgent == deps.state.selectedAgent {
		return currentSession, nil
	}
	deps.state.selectedAgent = newAgent
	newSession, err := switchToModel(deps, currentSession, currentModel)
	if err != nil {
		return currentSession, err
	}
	fmt.Println(formatAgentSwitchMessage(newAgent))
	return newSession, nil
}

// defineTools creates all the Kubernetes-related tools for the agent
