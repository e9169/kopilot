# Specialist Agent Personas

## Overview

Kopilot includes a set of **specialist agent personas** that focus the AI on specific operational domains. Each agent has a tailored system prompt, domain-specific examples, and always uses the premium model to ensure the highest reasoning quality for its specialized tasks.

Switch agents at startup with the `--agent` flag, or at runtime using the `/agent` command.

## Available Agents

### `default` — Standard Kopilot

The default generalist persona. Handles cluster monitoring, health checks, kubectl execution, comparisons, and general Q&A. Uses intelligent model selection (cost-effective for simple queries, premium for complex ones).

**Example prompts:**

- "Show me all clusters"
- "What's the status of my production cluster?"
- "Compare dev and prod resource usage"
- "Check health of all clusters"

---

### `debugger` — 🔍 K8s Debugger

Root cause analysis, log correlation, and pod failure diagnosis.

The Debugger agent starts with events and recent changes before diving into logs. It correlates pod status, events, and logs to identify root causes, traces failure chains, and suggests targeted remediation steps — not just restarts. Focuses on the **WHY before the HOW TO FIX**.

**Best for:**

- CrashLoopBackOff and OOMKilled diagnosis
- Deployment rollout failures
- Service connectivity and 5xx errors
- Multi-pod correlation analysis

**Example prompts:**

- "Why is my pod in CrashLoopBackOff?"
- "What caused this deployment rollout to fail?"
- "Diagnose why my service is returning 503 errors"
- "My pod keeps OOMKilled, investigate it"
- "Show all recent events for broken pods"

---

### `security` — 🛡️ K8s Security

RBAC auditing, privilege escalation detection, and network policy review.

The Security agent audits your cluster for misconfigurations and security risks. It identifies overprivileged service accounts, containers running as root, unintended network exposure, and Pod Security Admission violations. All findings are reported with severity levels (CRITICAL/HIGH/MEDIUM/LOW) and remediation steps.

**Best for:**

- RBAC and permission audits
- Privilege escalation detection
- Network policy gap analysis
- Secret exposure review
- PSA/PSP compliance checks

**Example prompts:**

- "Audit RBAC roles for overprivileged accounts"
- "Find pods running as root or privileged"
- "Check network policies for exposed services"
- "Review secret usage across namespaces"
- "Are there any PSA violations in this cluster?"

---

### `optimizer` — ⚡ K8s Optimizer

Resource right-sizing, HPA/VPA recommendations, and cost optimization.

The Optimizer agent identifies over-provisioned workloads, missing resource limits, and unused deployments. It recommends HPA/VPA configurations, analyzes node bin-packing efficiency, and highlights namespace quota issues. Findings are presented with estimated savings and priority (HIGH/MEDIUM/LOW impact).

**Best for:**

- Right-sizing CPU and memory requests/limits
- HPA and VPA setup recommendations
- Idle workload detection
- Node utilization and consolidation analysis
- Namespace quota tuning

**Example prompts:**

- "Which pods have no resource limits set?"
- "Find over-provisioned workloads in production"
- "Show node CPU and memory utilization"
- "Which deployments would benefit from HPA?"
- "Identify idle or low-traffic services"

---

### `gitops` — 🔄 K8s GitOps

Flux and ArgoCD sync status, drift detection, and reconciliation diagnostics.

The GitOps agent monitors the state of your GitOps tooling across Flux Kustomizations, HelmReleases, and ArgoCD Applications. It detects drift (resources modified outside of Git), diagnoses reconciliation failures, and tracks image automation policies. Always distinguishes between **desired state (Git)** and **actual state (cluster)**.

**Best for:**

- Flux Kustomization and HelmRelease health
- ArgoCD sync and out-of-sync detection
- Reconciliation failure diagnosis
- Configuration drift identification
- Image automation and update policy review

**Example prompts:**

- "Are all Flux Kustomizations synced?"
- "Show ArgoCD apps that are out of sync"
- "Why is this HelmRelease failing to reconcile?"
- "Find resources modified outside of GitOps"
- "Check Flux image automation status"

---

## Using Agents

### Start with a specialist at launch

```bash
# Start with the debugger agent
kopilot --agent debugger

# Start with the security auditor
kopilot --agent security

# Start with the resource optimizer
kopilot --agent optimizer

# Start with the GitOps specialist
kopilot --agent gitops
```

### Switch agents at runtime

You can switch between agents during a session without restarting:

```text
# Show current agent and available roster
❯ /agent
❯ /agent list

# Switch to a specialist
❯ /agent debugger
❯ /agent security
❯ /agent optimizer
❯ /agent gitops

# Return to the default persona
❯ /agent default
```

### Check available agents

```bash
kopilot --help
# Specialist Agents section lists: default, debugger, security, optimizer, gitops
```

---

## Model Selection and Agents

All specialist agents (`debugger`, `security`, `optimizer`, `gitops`) always use the **premium model** (`claude-sonnet-4.6` by default), regardless of how simple the query appears. This is because specialist reasoning benefits from higher model capacity — even a simple "list all pods" query issued through the Debugger agent may require deep context to give a meaningful diagnosis.

The `default` agent uses intelligent model selection: cost-effective (`gpt-4.1`) for simple queries and premium for troubleshooting and complex operations. See [MODEL_SELECTION.md](MODEL_SELECTION.md) for details.

| Agent | Model strategy |
| ------- | --------------- |
| `default` | Dynamic — cost-effective or premium based on query |
| `debugger` | Always premium |
| `security` | Always premium |
| `optimizer` | Always premium |
| `gitops` | Always premium |

---

## Execution Mode Interaction

Agents and execution modes are **fully independent** layers:

- The **agent** controls the AI's system prompt and reasoning focus
- The **execution mode** controls whether kubectl write operations are allowed

A `security` agent session still enforces read-only mode by default. Switching to `/interactive` enables write operations regardless of which agent is active.

```bash
# Security audit in read-only mode (default) — safe, can't accidentally change anything
kopilot --agent security

# Optimizer with interactive mode — allows scaling recommendations to be applied
kopilot --agent optimizer --interactive
```

See [EXECUTION_MODES.md](EXECUTION_MODES.md) for details on mode behavior.

---

## Summary Table

| Agent | CLI flag | Icon | Premium model | Focus area |
| ------- | ---------- | ------ | :---: | ---------- |
| `default` | _(omit flag)_ | — | Dynamic | General Kubernetes operations |
| `debugger` | `--agent debugger` | 🔍 | ✅ Always | Root cause analysis & log correlation |
| `security` | `--agent security` | 🛡️ | ✅ Always | RBAC, privileges & network policies |
| `optimizer` | `--agent optimizer` | ⚡ | ✅ Always | Resource right-sizing & cost savings |
| `gitops` | `--agent gitops` | 🔄 | ✅ Always | Flux/ArgoCD sync & drift detection |
