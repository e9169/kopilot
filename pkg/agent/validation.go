// Package agent provides enhanced kubectl command validation.
// This file contains security and safety checks for kubectl command execution.
package agent

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// dangerousCommands lists kubectl commands that should be extra careful with
	dangerousCommands = map[string]bool{
		"delete":      true,
		"drain":       true,
		"cordon":      true,
		"uncordon":    true,
		"taint":       true,
		"label":       true,
		"annotate":    true,
		"patch":       true,
		"replace":     true,
		"scale":       true,
		"set":         true,
		"rollout":     true,
		"certificate": true,
	}

	// Command injection patterns to block
	injectionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`[;&|]`),      // Shell operators
		regexp.MustCompile(`\$\(`),       // Command substitution
		regexp.MustCompile("`"),          // Backticks
		regexp.MustCompile(`>\s*/`),      // File redirect to root
		regexp.MustCompile(`<\s*/`),      // File read from root
		regexp.MustCompile(`\.\./`),      // Directory traversal
		regexp.MustCompile(`~[^/\s]*\/`), // Home directory expansion
		regexp.MustCompile(`\$\{`),       // Variable expansion
		regexp.MustCompile(`\x00`),       // Null bytes
	}

	// Allowed kubectl subcommands (whitelist approach)
	allowedCommands = map[string]bool{
		// Read-only commands
		"get":           true,
		"describe":      true,
		"logs":          true,
		"explain":       true,
		"api-resources": true,
		"api-versions":  true,
		"cluster-info":  true,
		"version":       true,
		"config":        true,
		"top":           true,
		"diff":          true,
		"auth":          true,
		// Write commands (will be blocked in read-only mode)
		"create":       true,
		"apply":        true,
		"edit":         true,
		"delete":       true,
		"replace":      true,
		"patch":        true,
		"scale":        true,
		"autoscale":    true,
		"rollout":      true,
		"set":          true,
		"label":        true,
		"annotate":     true,
		"expose":       true,
		"run":          true,
		"port-forward": true,
		"proxy":        true,
		"attach":       true,
		"exec":         true,
		"cp":           true,
		"drain":        true,
		"cordon":       true,
		"uncordon":     true,
		"taint":        true,
		"certificate":  true,
		"wait":         true,
	}
)

// validateKubectlCommand performs comprehensive validation on kubectl commands
func validateKubectlCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no kubectl command provided")
	}

	command := args[0]

	// Check if command is in allowed list
	if !allowedCommands[command] {
		return fmt.Errorf("kubectl command '%s' is not allowed. Use standard kubectl commands only", command)
	}

	// Check all arguments for injection patterns
	if err := checkInjectionPatterns(args); err != nil {
		return err
	}

	// Validate namespace flags if present
	if err := validateNamespaceFlags(args); err != nil {
		return err
	}

	// Additional validation for dangerous commands
	if err := validateDangerousCommands(command, args); err != nil {
		return err
	}

	return nil
}

// checkInjectionPatterns checks for command injection patterns
func checkInjectionPatterns(args []string) error {
	fullCommand := strings.Join(args, " ")
	for _, pattern := range injectionPatterns {
		if pattern.MatchString(fullCommand) {
			return fmt.Errorf("potential command injection detected in arguments: %s", pattern.String())
		}
	}
	return nil
}

// validateNamespaceFlags validates namespace flag values
func validateNamespaceFlags(args []string) error {
	for i, arg := range args {
		if (arg == "-n" || arg == "--namespace") && i+1 < len(args) {
			ns := args[i+1]
			if !isValidKubernetesName(ns) {
				return fmt.Errorf("invalid namespace name: %s", ns)
			}
		}
		if strings.HasPrefix(arg, "--namespace=") {
			ns := strings.TrimPrefix(arg, "--namespace=")
			if !isValidKubernetesName(ns) {
				return fmt.Errorf("invalid namespace name: %s", ns)
			}
		}
	}
	return nil
}

// validateDangerousCommands performs additional validation for dangerous commands
func validateDangerousCommands(command string, args []string) error {
	if !dangerousCommands[command] {
		return nil
	}

	// Ensure no wildcard deletions
	if command == "delete" {
		for _, arg := range args {
			if arg == "--all" || strings.Contains(arg, "*") {
				return fmt.Errorf("bulk delete operations with --all or wildcards require extra caution")
			}
		}
	}

	return nil
}

// isValidKubernetesName checks if a string is a valid Kubernetes resource name
func isValidKubernetesName(name string) bool {
	// Kubernetes names must be lowercase alphanumeric, -, or .
	// and must start and end with alphanumeric
	if len(name) == 0 || len(name) > 253 {
		return false
	}

	match, _ := regexp.MatchString(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`, name)
	return match
}

// sanitizeKubectlArgs removes potentially dangerous flags and arguments
func sanitizeKubectlArgs(args []string) []string {
	sanitized := make([]string, 0, len(args))
	skipNext := false

	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		// Skip dangerous global flags
		if strings.HasPrefix(arg, "--kubeconfig=") ||
			strings.HasPrefix(arg, "--token=") ||
			strings.HasPrefix(arg, "--certificate-authority=") ||
			strings.HasPrefix(arg, "--client-certificate=") ||
			strings.HasPrefix(arg, "--client-key=") {
			continue
		}

		if arg == "--kubeconfig" || arg == "--token" ||
			arg == "--certificate-authority" ||
			arg == "--client-certificate" || arg == "--client-key" {
			// Skip this flag and its value
			skipNext = true
			continue
		}

		// Remove excessive output flags that could cause issues
		if arg == "-w" || arg == "--watch" {
			// Watch could cause issues in non-interactive mode
			continue
		}

		sanitized = append(sanitized, arg)
	}

	return sanitized
}
