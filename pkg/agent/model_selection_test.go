package agent

import (
	"fmt"
	"testing"
)

// TestSelectModelForQuery tests model selection for AgentDefault with various query types.
// Specialist agent behaviour is covered by TestSpecialistAgentsAlwaysUsePremiumModel.
func TestSelectModelForQuery(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		expectedModel string
	}{
		// Simple queries - should use cost-effective model
		{"list clusters", "list all clusters", modelCostEffective},
		{"show status", "show me the status of my pods", modelCostEffective},
		{"get resources", "get all namespaces", modelCostEffective},
		{"check health", "check the health of the cluster", modelCostEffective},
		{"what query", "what pods are running?", modelCostEffective},
		{"describe resource", "describe the deployment", modelCostEffective},
		{"ambiguous query", "tell me about kubernetes", modelCostEffective},
		{"general question", "how are things looking?", modelCostEffective},

		// Troubleshooting queries - should use premium model
		{"why question", "why is my pod not starting?", modelPremium},
		{"troubleshoot issue", "troubleshoot the connection problem", modelPremium},
		{"debug error", "debug this error", modelPremium},
		{"investigate failure", "investigate why the service failed", modelPremium},
		{"fix problem", "help me fix this broken deployment", modelPremium},
		{"explain issue", "explain why this is not working", modelPremium},
		{"diagnose crash", "diagnose the crash loop", modelPremium},
		{"analyze problem", "analyze this issue for me", modelPremium},

		// Complex kubectl operations - should use premium model
		{"scale deployment", "scale the deployment to 5 replicas", modelPremium},
		{"restart pods", "restart the pods in default namespace", modelPremium},
		{"delete resource", "delete the failed pod", modelPremium},
		{"apply manifest", "apply the new configuration", modelPremium},
		{"patch deployment", "patch the deployment with new image", modelPremium},
		{"rollback", "rollback the last deployment", modelPremium},
		{"drain node", "drain the worker node", modelPremium},
		{"cordon node", "cordon the node for maintenance", modelPremium},

		// Mixed keywords - troubleshooting takes precedence over simple keywords
		{"list with error", "list pods with error status", modelPremium},
		{"show failed", "show me all failed deployments", modelPremium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectModelForQuery(tt.query, AgentDefault)
			if result != tt.expectedModel {
				t.Errorf("selectModelForQuery(%q, AgentDefault) = %q, want %q", tt.query, result, tt.expectedModel)
			}
		})
	}
}

// TestSpecialistAgentsAlwaysUsePremiumModel verifies that all specialist agent
// types always select the premium model regardless of query complexity.
func TestSpecialistAgentsAlwaysUsePremiumModel(t *testing.T) {
	specialistAgents := []AgentType{AgentDebugger, AgentSecurity, AgentOptimizer, AgentGitOps}

	// Representative queries spanning simple, ambiguous, and complex prompts.
	queries := []string{
		"list events",
		"show me all pods",
		"get service accounts",
		"how are things looking?",
		"check network policies",
		"show resource usage",
		"status of all kustomizations",
		"list helm releases",
	}

	for _, agent := range specialistAgents {
		for _, query := range queries {
			t.Run(fmt.Sprintf("%s/%s", agent, query), func(t *testing.T) {
				result := selectModelForQuery(query, agent)
				if result != modelPremium {
					t.Errorf("selectModelForQuery(%q, %q) = %q, want %q", query, agent, result, modelPremium)
				}
			})
		}
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
