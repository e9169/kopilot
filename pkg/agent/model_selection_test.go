package agent

import (
	"testing"
)

func TestSelectModelForQuery(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		agentType     AgentType
		expectedModel string
	}{
		// Simple queries with default agent - should use cost-effective model
		{
			name:          "list clusters",
			query:         "list all clusters",
			agentType:     AgentDefault,
			expectedModel: modelCostEffective,
		},
		{
			name:          "show status",
			query:         "show me the status of my pods",
			agentType:     AgentDefault,
			expectedModel: modelCostEffective,
		},
		{
			name:          "get resources",
			query:         "get all namespaces",
			agentType:     AgentDefault,
			expectedModel: modelCostEffective,
		},
		{
			name:          "check health",
			query:         "check the health of the cluster",
			agentType:     AgentDefault,
			expectedModel: modelCostEffective,
		},
		{
			name:          "what query",
			query:         "what pods are running?",
			agentType:     AgentDefault,
			expectedModel: modelCostEffective,
		},
		{
			name:          "describe resource",
			query:         "describe the deployment",
			agentType:     AgentDefault,
			expectedModel: modelCostEffective,
		},

		// Troubleshooting queries with default agent - should use premium model
		{
			name:          "why question",
			query:         "why is my pod not starting?",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "troubleshoot issue",
			query:         "troubleshoot the connection problem",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "debug error",
			query:         "debug this error",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "investigate failure",
			query:         "investigate why the service failed",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "fix problem",
			query:         "help me fix this broken deployment",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "explain issue",
			query:         "explain why this is not working",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "diagnose crash",
			query:         "diagnose the crash loop",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "analyze problem",
			query:         "analyze this issue for me",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},

		// Complex kubectl operations with default agent - should use premium model
		{
			name:          "scale deployment",
			query:         "scale the deployment to 5 replicas",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "restart pods",
			query:         "restart the pods in default namespace",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "delete resource",
			query:         "delete the failed pod",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "apply manifest",
			query:         "apply the new configuration",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "patch deployment",
			query:         "patch the deployment with new image",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "rollback",
			query:         "rollback the last deployment",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "drain node",
			query:         "drain the worker node",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "cordon node",
			query:         "cordon the node for maintenance",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},

		// Default case with default agent - ambiguous queries should use cost-effective
		{
			name:          "ambiguous query",
			query:         "tell me about kubernetes",
			agentType:     AgentDefault,
			expectedModel: modelCostEffective,
		},
		{
			name:          "general question",
			query:         "how are things looking?",
			agentType:     AgentDefault,
			expectedModel: modelCostEffective,
		},

		// Mixed keywords with default agent - troubleshooting takes precedence
		{
			name:          "list with error",
			query:         "list pods with error status",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},
		{
			name:          "show failed",
			query:         "show me all failed deployments",
			agentType:     AgentDefault,
			expectedModel: modelPremium,
		},

		// Specialist agents always use premium model, regardless of query text
		{
			name:          "debugger simple query",
			query:         "show me all pods",
			agentType:     AgentDebugger,
			expectedModel: modelPremium,
		},
		{
			name:          "debugger list query",
			query:         "list events",
			agentType:     AgentDebugger,
			expectedModel: modelPremium,
		},
		{
			name:          "security simple query",
			query:         "get service accounts",
			agentType:     AgentSecurity,
			expectedModel: modelPremium,
		},
		{
			name:          "security check query",
			query:         "check network policies",
			agentType:     AgentSecurity,
			expectedModel: modelPremium,
		},
		{
			name:          "optimizer show query",
			query:         "show resource usage",
			agentType:     AgentOptimizer,
			expectedModel: modelPremium,
		},
		{
			name:          "optimizer ambiguous query",
			query:         "how are things looking?",
			agentType:     AgentOptimizer,
			expectedModel: modelPremium,
		},
		{
			name:          "gitops status query",
			query:         "status of all kustomizations",
			agentType:     AgentGitOps,
			expectedModel: modelPremium,
		},
		{
			name:          "gitops list query",
			query:         "list helm releases",
			agentType:     AgentGitOps,
			expectedModel: modelPremium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectModelForQuery(tt.query, tt.agentType)
			if result != tt.expectedModel {
				t.Errorf("selectModelForQuery(%q, %q) = %q, want %q", tt.query, tt.agentType, result, tt.expectedModel)
			}
		})
	}
}

func TestModelConstants(t *testing.T) {
	// Verify the constants have expected values
	if modelCostEffective != "gpt-4o-mini" {
		t.Errorf("modelCostEffective = %q, want %q", modelCostEffective, "gpt-4o-mini")
	}

	if modelPremium != "gpt-4o" {
		t.Errorf("modelPremium = %q, want %q", modelPremium, "gpt-4o")
	}
}
