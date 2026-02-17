// Package agent provides the core Copilot agent functionality for Kubernetes cluster operations.
// It implements an interactive agent using the GitHub Copilot SDK that can monitor, query,
// and manage Kubernetes clusters through natural language interactions.
package agent

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

const (
	// Default model selection constants
	defaultModelCostEffective = "gpt-4o-mini" // Cost-effective model for simple queries
	defaultModelPremium       = "gpt-4o"      // Premium model for complex tasks

	// ANSI color codes
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

const (
	toolListClusters     = "list_clusters"
	toolGetClusterStatus = "get_cluster_status"
	toolCompareClusters  = "compare_clusters"
	toolCheckAllClusters = "check_all_clusters"
	toolKubectlExec      = "kubectl_exec"
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
	quotaDisplayed  bool
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
- Use clear, concise language
- Show tool output directly without reformatting
- Use markdown tables for structured data when appropriate
- Add brief analysis or next steps when helpful

For kubectl operations:
- Always specify the cluster context with --context flag
- Explain what you're doing before executing commands
- Interpret command output for the user

Be helpful, clear, and conversational.`
}

// displayQuota shows quota information in GitHub Copilot CLI style
func displayQuota(state *agentState) {
	if isJSONOutput(state.outputFormat) || state.quotaPercentage < 0 {
		return
	}
	fmt.Printf("%.0f%% of quota remaining\n", state.quotaPercentage)
}

// setupSessionEventHandler creates and returns an event handler for the session
func setupSessionEventHandler(session *copilot.Session, isIdlePtr *bool, state *agentState, messageBuffer *strings.Builder) {
	session.On(func(event copilot.SessionEvent) {
		switch event.Type {
		case "assistant.message_delta":
			if event.Data.DeltaContent != nil && len(*event.Data.DeltaContent) > 0 {
				// Display quota before first content if available
				if state.quotaDisplayed && messageBuffer.Len() == 0 {
					displayQuota(state)
					state.quotaDisplayed = false // Only show once per response
				}
				fmt.Print(*event.Data.DeltaContent)
				messageBuffer.WriteString(*event.Data.DeltaContent)
			}
		case "assistant.message":
			if messageBuffer.Len() == 0 && event.Data.Content != nil {
				// Display quota before message if available
				if state.quotaDisplayed {
					displayQuota(state)
					state.quotaDisplayed = false
				}
				fmt.Print(*event.Data.Content)
			}
			fmt.Println()
			messageBuffer.Reset()
		case "session.idle":
			*isIdlePtr = true
		case "assistant.usage":
			if event.Data.QuotaSnapshots != nil {
				if snapshot, exists := event.Data.QuotaSnapshots["premium_interactions"]; exists {
					if snapshot.RemainingPercentage >= 0 {
						state.quotaPercentage = snapshot.RemainingPercentage
						state.quotaDisplayed = true
					}
				}
			}
		}
	})
}

// getRandomExamples returns a random selection of example prompts
func getRandomExamples(count int) []string {
	examples := []string{
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

	// Shuffle and select (Go 1.20+ automatically seeds the global RNG)
	shuffled := make([]string, len(examples))
	copy(shuffled, examples)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	if count > len(shuffled) {
		count = len(shuffled)
	}

	return shuffled[:count]
}

// Run starts the Copilot agent with Kubernetes cluster tools
func Run(k8sProvider *k8s.Provider, mode ExecutionMode, outputFormat OutputFormat) error {
	// Configure logging to stderr to avoid interfering with stdio-based JSON-RPC
	log.SetOutput(os.Stderr)

	// Initialize agent state
	state := &agentState{
		mode:            mode,
		outputFormat:    outputFormat,
		quotaPercentage: -1,
		quotaDisplayed:  false,
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
	messageBuffer := strings.Builder{}
	var isIdle bool
	setupSessionEventHandler(session, &isIdle, state, &messageBuffer)

	if !isJSONOutput(outputFormat) {
		// ASCII art logo
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

		// Status messages with icons
		clusters := k8sProvider.GetClusters()
		currentCtx := k8sProvider.GetCurrentContext()
		fmt.Printf("  %s●%s Connected to %d cluster(s)\n", colorGreen, colorReset, len(clusters))
		if currentCtx != "" {
			fmt.Printf("  %s●%s Active context: %s%s%s\n", colorCyan, colorReset, colorCyan, currentCtx, colorReset)
		}

		// Execution mode with icon
		modeIcon := "🔒"
		modeColor := colorYellow
		modeText := "read-only"
		if mode == ModeInteractive {
			modeIcon = "🔓"
			modeColor = colorGreen
			modeText = "interactive"
		}
		fmt.Printf("  %s●%s Mode: %s%s %s%s\n", modeColor, colorReset, modeIcon, modeColor, modeText, colorReset)

		// Show random example prompts
		fmt.Println()
		fmt.Printf("  %sTry asking:%s\n", colorDim, colorReset)
		examples := getRandomExamples(3)
		for _, example := range examples {
			fmt.Printf("    %s•%s %s\"%s\"%s\n", colorCyan, colorReset, colorDim, example, colorReset)
		}

		fmt.Println()
		fmt.Printf("  %sType your request to get started. Enter 'exit' to quit.%s\n", colorDim, colorReset)
		fmt.Println()
	}

	// Mark as idle so user can start typing immediately
	isIdle = true

	// Interactive loop with session management
	return interactiveLoopWithModelSelection(ctx, client, k8sProvider, state, session, &isIdle, &messageBuffer)
}

// findCopilotCLI attempts to locate the Copilot CLI executable
func findCopilotCLI() (string, error) {
	// 1. Check if 'copilot' is in PATH
	if path, err := exec.LookPath("copilot"); err == nil {
		log.Printf("Found copilot CLI in PATH: %s", path)
		return path, nil
	}

	// 2. Check VS Code global storage (common location)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		vscodeLocations := []string{
			filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "globalStorage", "github.copilot-chat", "copilotCli", "copilot"),
			filepath.Join(homeDir, ".vscode", "extensions", "github.copilot-chat*", "copilotCli", "copilot"),
			filepath.Join(homeDir, "AppData", "Roaming", "Code", "User", "globalStorage", "github.copilot-chat", "copilotCli", "copilot.exe"), // Windows
		}

		for _, location := range vscodeLocations {
			// Handle glob patterns
			if strings.Contains(location, "*") {
				matches, _ := filepath.Glob(location)
				for _, match := range matches {
					if _, err := os.Stat(match); err == nil {
						log.Printf("Found copilot CLI at: %s", match)
						return match, nil
					}
				}
			} else {
				if _, err := os.Stat(location); err == nil {
					log.Printf("Found copilot CLI at: %s", location)
					return location, nil
				}
			}
		}
	}

	// 3. Check common installation paths
	commonPaths := []string{
		"/usr/local/bin/copilot",
		"/opt/homebrew/bin/copilot",
		"/usr/bin/copilot",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			log.Printf("Found copilot CLI at: %s", path)
			return path, nil
		}
	}

	// Build helpful error message with installation instructions
	return "", buildCLINotFoundError()
}

// buildCLINotFoundError creates a detailed error message for missing CLI
func buildCLINotFoundError() error {
	msg := `GitHub Copilot CLI not found.

To install GitHub Copilot CLI, choose one of the following methods:

1. Using npm (recommended):
   npm install -g @githubnext/github-copilot-cli

2. Using Homebrew (macOS/Linux):
   brew install github/gh-copilot/gh-copilot

3. Using GitHub CLI extension:
   gh extension install github/gh-copilot

4. Download from VS Code:
   Install the GitHub Copilot extension in VS Code - the CLI will be bundled.

After installation, authenticate with:
   copilot auth login

For more information, visit:
   https://docs.github.com/en/copilot/github-copilot-in-the-cli

Searched locations:
   - PATH environment variable
   - VS Code global storage directories
   - /usr/local/bin/copilot
   - /opt/homebrew/bin/copilot
   - /usr/bin/copilot`

	return fmt.Errorf("%s", msg)
}

// createAndStartClient creates and starts the Copilot client
func createAndStartClient(ctx context.Context) (*copilot.Client, error) {
	// Auto-detect Copilot CLI location from user's system
	cliPath, err := findCopilotCLI()
	if err != nil {
		return nil, err
	}

	log.Printf("Using Copilot CLI: %s", cliPath)

	// Verify CLI version compatibility
	if err := verifyCLIVersion(cliPath); err != nil {
		log.Printf("Warning: %v", err)
		// Continue anyway - user might have a compatible build
	}

	// Get current working directory for CLI context
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Let SDK choose the best communication mode (defaults to stdio)
	// Don't specify UseStdio - let SDK handle it
	autoStart := true
	autoRestart := true

	client := copilot.NewClient(&copilot.ClientOptions{
		CLIPath:     cliPath,
		Cwd:         cwd,
		AutoStart:   &autoStart,
		AutoRestart: &autoRestart,
		LogLevel:    "error",      // Reduce noise in logs
		Env:         os.Environ(), // Pass current environment
	})

	log.Println("Starting Copilot client...")
	if err := client.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start copilot client: %w\n\nTip: Ensure GitHub Copilot CLI is properly set up and authenticated.\nFor best compatibility, use CLI version 0.0.410 (SDK v0.1.23 requirement)", err)
	}

	log.Println("Copilot client started successfully")
	return client, nil
}

// verifyCLIVersion checks if the CLI version is compatible with the SDK
func verifyCLIVersion(cliPath string) error {
	cmd := exec.Command(cliPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to check CLI version: %w", err)
	}

	version := strings.TrimSpace(string(output))
	log.Printf("Copilot CLI version: %s", version)

	// SDK v0.1.23 works with CLI v0.0.410
	// Just log a warning for informational purposes, don't block startup
	if !strings.Contains(version, "0.0.410") && !strings.Contains(version, "0.0.409") {
		log.Printf("Warning: CLI version may not be fully tested with SDK v0.1.23. Found: %s", version)
		log.Printf("Recommended versions: CLI v0.0.410 (current) or v0.0.409")
	}

	return nil
}

// createSessionWithModel creates a new Copilot session with specified model
func createSessionWithModel(ctx context.Context, client *copilot.Client, k8sProvider *k8s.Provider, state *agentState, model string) (*copilot.Session, error) {
	tools := defineTools(k8sProvider, state)
	systemMessage := getSystemMessage()

	session, err := client.CreateSession(ctx, &copilot.SessionConfig{
		Model: model,
		Tools: tools,
		SystemMessage: &copilot.SystemMessageConfig{
			Mode:    "append",
			Content: systemMessage,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	log.Printf("Session created with model: %s", model)
	return session, nil
}

// selectModelForQuery determines the best model based on query complexity and intent
func selectModelForQuery(query string) string {
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

// waitForIdle waits until the session is idle
func waitForIdle(isIdle *bool) {
	for !*isIdle {
		continue
	}
}

// readUserInput reads and trims user input from stdin
func readUserInput(reader *bufio.Reader) (string, error) {
	fmt.Print("❯ ")
	input, err := reader.ReadString('\n')
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

// handleModeSwitch handles runtime mode switching commands
// Returns true if the input was a mode switch command
func handleModeSwitch(input string, state *agentState) bool {
	lower := strings.TrimSpace(strings.ToLower(input))

	switch lower {
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

// switchToModel switches to a different model by creating a new session
func switchToModel(
	ctx context.Context,
	client *copilot.Client,
	k8sProvider *k8s.Provider,
	state *agentState,
	oldSession *copilot.Session,
	newModel string,
	isIdle *bool,
	messageBuffer *strings.Builder,
) (*copilot.Session, error) {
	// Destroy old session
	if err := oldSession.Destroy(); err != nil {
		log.Printf("Warning: failed to destroy old session: %v", err)
	}

	// Create new session with optimal model
	newSession, err := createSessionWithModel(ctx, client, k8sProvider, state, newModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create new session: %w", err)
	}

	// Set up event handling for new session
	setupSessionEventHandler(newSession, isIdle, state, messageBuffer)

	// Wait for new session to be ready
	waitForIdle(isIdle)

	return newSession, nil
}

// interactiveLoopWithModelSelection handles interactive conversation with dynamic model selection
func interactiveLoopWithModelSelection(
	ctx context.Context,
	client *copilot.Client,
	k8sProvider *k8s.Provider,
	state *agentState,
	initialSession *copilot.Session,
	isIdle *bool,
	messageBuffer *strings.Builder,
) error {
	reader := bufio.NewReader(os.Stdin)
	currentSession := initialSession
	currentModel := modelCostEffective

	for {
		waitForIdle(isIdle)
		messageBuffer.Reset()

		input, err := readUserInput(reader)
		if err != nil {
			return err
		}

		if input == "" {
			continue
		}

		if isExitCommand(input) {
			fmt.Println("")
			return nil
		}

		// Handle runtime mode switch commands
		if handleModeSwitch(input, state) {
			continue
		}

		// Determine optimal model for this query
		optimalModel := selectModelForQuery(input)

		// Switch model if needed
		if optimalModel != currentModel {
			currentSession, err = switchToModel(ctx, client, k8sProvider, state, currentSession, optimalModel, isIdle, messageBuffer)
			if err != nil {
				return err
			}
			currentModel = optimalModel
		}

		// Send user message with timeout context
		// Use a per-request timeout to prevent indefinite blocking
		*isIdle = false

		_, err = currentSession.Send(ctx, copilot.MessageOptions{
			Prompt: input,
		})
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}
}

// defineTools creates all the Kubernetes-related tools for the agent
