package agent

import (
	"testing"
)

func TestSelectModelForQuery(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		expectedModel string
	}{
		// Simple queries - should use cost-effective model
		{
			name:          "list clusters",
			query:         "list all clusters",
			expectedModel: modelCostEffective,
		},
		{
			name:          "show status",
			query:         "show me the status of my pods",
			expectedModel: modelCostEffective,
		},
		{
			name:          "get resources",
			query:         "get all namespaces",
			expectedModel: modelCostEffective,
		},
		{
			name:          "check health",
			query:         "check the health of the cluster",
			expectedModel: modelCostEffective,
		},
		{
			name:          "what query",
			query:         "what pods are running?",
			expectedModel: modelCostEffective,
		},
		{
			name:          "describe resource",
			query:         "describe the deployment",
			expectedModel: modelCostEffective,
		},

		// Troubleshooting queries - should use premium model
		{
			name:          "why question",
			query:         "why is my pod not starting?",
			expectedModel: modelPremium,
		},
		{
			name:          "troubleshoot issue",
			query:         "troubleshoot the connection problem",
			expectedModel: modelPremium,
		},
		{
			name:          "debug error",
			query:         "debug this error",
			expectedModel: modelPremium,
		},
		{
			name:          "investigate failure",
			query:         "investigate why the service failed",
			expectedModel: modelPremium,
		},
		{
			name:          "fix problem",
			query:         "help me fix this broken deployment",
			expectedModel: modelPremium,
		},
		{
			name:          "explain issue",
			query:         "explain why this is not working",
			expectedModel: modelPremium,
		},
		{
			name:          "diagnose crash",
			query:         "diagnose the crash loop",
			expectedModel: modelPremium,
		},
		{
			name:          "analyze problem",
			query:         "analyze this issue for me",
			expectedModel: modelPremium,
		},

		// Complex kubectl operations - should use premium model
		{
			name:          "scale deployment",
			query:         "scale the deployment to 5 replicas",
			expectedModel: modelPremium,
		},
		{
			name:          "restart pods",
			query:         "restart the pods in default namespace",
			expectedModel: modelPremium,
		},
		{
			name:          "delete resource",
			query:         "delete the failed pod",
			expectedModel: modelPremium,
		},
		{
			name:          "apply manifest",
			query:         "apply the new configuration",
			expectedModel: modelPremium,
		},
		{
			name:          "patch deployment",
			query:         "patch the deployment with new image",
			expectedModel: modelPremium,
		},
		{
			name:          "rollback",
			query:         "rollback the last deployment",
			expectedModel: modelPremium,
		},
		{
			name:          "drain node",
			query:         "drain the worker node",
			expectedModel: modelPremium,
		},
		{
			name:          "cordon node",
			query:         "cordon the node for maintenance",
			expectedModel: modelPremium,
		},

		// Default case - ambiguous queries should use cost-effective
		{
			name:          "ambiguous query",
			query:         "tell me about kubernetes",
			expectedModel: modelCostEffective,
		},
		{
			name:          "general question",
			query:         "how are things looking?",
			expectedModel: modelCostEffective,
		},

		// Mixed keywords - troubleshooting takes precedence
		{
			name:          "list with error",
			query:         "list pods with error status",
			expectedModel: modelPremium,
		},
		{
			name:          "show failed",
			query:         "show me all failed deployments",
			expectedModel: modelPremium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectModelForQuery(tt.query)
			if result != tt.expectedModel {
				t.Errorf("selectModelForQuery(%q) = %q, want %q", tt.query, result, tt.expectedModel)
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
