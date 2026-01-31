package agent

import (
	"strings"
	"testing"
)

// TestValidateKubectlCommand tests kubectl command validation
func TestValidateKubectlCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"valid get", []string{"get", "pods"}, false},
		{"valid describe", []string{"describe", "pod", "nginx"}, false},
		{"valid logs", []string{"logs", "nginx"}, false},
		{"valid apply", []string{"apply", "-f", "deploy.yaml"}, false},
		{"invalid empty", []string{}, true},
		{"invalid command", []string{"hacker-command"}, true},
		{"injection pipe", []string{"get", "pods", "|", "grep"}, true},
		{"injection semicolon", []string{"get", "pods;", "rm"}, true},
		{"injection &&", []string{"get", "pods", "&&", "echo"}, true},
		{"injection $()}", []string{"get", "$(whoami)"}, true},
		{"injection backticks", []string{"get", "`ls`"}, true},
		{"path traversal", []string{"get", "../../../etc/passwd"}, true},
		{"bulk delete all", []string{"delete", "pods", "--all"}, true},
		{"bulk delete wildcard", []string{"delete", "pods", "nginx-*"}, true},
		{"valid namespace", []string{"get", "pods", "-n", "default"}, false},
		{"invalid namespace", []string{"get", "pods", "-n", "Invalid_Name!"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKubectlCommand(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateKubectlCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIsValidKubernetesName tests Kubernetes name validation
func TestIsValidKubernetesName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid lowercase", "default", true},
		{"valid with dash", "kube-system", true},
		{"valid with dots", "my.namespace.com", true},
		{"valid alphanumeric", "namespace123", true},
		{"invalid uppercase", "MyNamespace", false},
		{"invalid underscore", "my_namespace", false},
		{"invalid special char", "namespace!", false},
		{"invalid start with dash", "-namespace", false},
		{"invalid end with dash", "namespace-", false},
		{"invalid empty", "", false},
		{"invalid too long", strings.Repeat("a", 254), false},
		{"valid max length", strings.Repeat("a", 253), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidKubernetesName(tt.input)
			if got != tt.want {
				t.Errorf("isValidKubernetesName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestSanitizeKubectlArgs tests argument sanitization
func TestSanitizeKubectlArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			"no dangerous args",
			[]string{"get", "pods", "-n", "default"},
			[]string{"get", "pods", "-n", "default"},
		},
		{
			"remove kubeconfig flag",
			[]string{"get", "pods", "--kubeconfig", "/path"},
			[]string{"get", "pods"},
		},
		{
			"remove token flag",
			[]string{"get", "pods", "--token", "secret"},
			[]string{"get", "pods"},
		},
		{
			"remove watch flag",
			[]string{"get", "pods", "-w"},
			[]string{"get", "pods"},
		},
		{
			"mixed safe and dangerous",
			[]string{"get", "pods", "--kubeconfig", "config", "-n", "default"},
			[]string{"get", "pods", "-n", "default"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeKubectlArgs(tt.args)
			if len(got) != len(tt.want) {
				t.Errorf("sanitizeKubectlArgs() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("sanitizeKubectlArgs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestAllowedCommands verifies the command whitelist
func TestAllowedCommands(t *testing.T) {
	readOnlyCommands := []string{"get", "describe", "logs", "explain", "top"}
	for _, cmd := range readOnlyCommands {
		if !allowedCommands[cmd] {
			t.Errorf("Read-only command %q should be in allowed list", cmd)
		}
	}

	writeCommands := []string{"apply", "create", "delete", "patch", "scale"}
	for _, cmd := range writeCommands {
		if !allowedCommands[cmd] {
			t.Errorf("Write command %q should be in allowed list", cmd)
		}
	}
}

// TestDangerousCommands verifies dangerous command detection
func TestDangerousCommands(t *testing.T) {
	dangerous := []string{"delete", "drain", "cordon", "taint", "scale"}
	for _, cmd := range dangerous {
		if !dangerousCommands[cmd] {
			t.Errorf("Command %q should be marked as dangerous", cmd)
		}
	}

	safe := []string{"get", "describe", "logs"}
	for _, cmd := range safe {
		if dangerousCommands[cmd] {
			t.Errorf("Command %q should not be marked as dangerous", cmd)
		}
	}
}

// BenchmarkValidateKubectlCommand benchmarks validation performance
func BenchmarkValidateKubectlCommand(b *testing.B) {
	args := []string{"get", "pods", "-n", "default", "-o", "json"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateKubectlCommand(args)
	}
}

// BenchmarkSanitizeKubectlArgs benchmarks sanitization performance
func BenchmarkSanitizeKubectlArgs(b *testing.B) {
	args := []string{"get", "pods", "--kubeconfig", "/path", "-n", "default", "--token", "secret", "-o", "json"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizeKubectlArgs(args)
	}
}
