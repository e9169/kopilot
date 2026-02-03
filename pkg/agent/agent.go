// Package agent provides the core Copilot agent functionality for Kubernetes cluster operations.
// It implements an interactive agent using the GitHub Copilot SDK that can monitor, query,
// and manage Kubernetes clusters through natural language interactions.
package agent

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/e9169/kopilot/pkg/k8s"
	copilot "github.com/github/copilot-sdk/go"
)

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
	colorReset     = "\033[0m"
	colorRed       = "\033[31m"
	colorGreen     = "\033[32m"
	colorYellow    = "\033[33m"
	colorMagenta   = "\033[35m"
	colorCyan      = "\033[36m"
	colorBold      = "\033[1m"
	colorDim       = "\033[2m"
	gradientColor1 = "\033[38;5;51m"  // Bright cyan
	gradientColor2 = "\033[38;5;45m"  // Cyan-blue
	gradientColor3 = "\033[38;5;39m"  // Blue
	gradientColor4 = "\033[38;5;69m"  // Blue-magenta
	gradientColor5 = "\033[38;5;99m"  // Magenta
	gradientColor6 = "\033[38;5;135m" // Light magenta

	cursorHide = "\033[?25l"
	cursorShow = "\033[?25h"
	clearLine  = "\033[2K"
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
	mode             ExecutionMode
	outputFormat     OutputFormat
	quotaPercentage  float64
	quotaDisplayed   bool
	isThinking       bool
	thinkingStopChan chan bool
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
	return `You are Kopilot, a helpful Kubernetes cluster operations assistant. Your role is to:

1. Monitor and manage Kubernetes clusters
2. Execute kubectl commands on the user's behalf
3. Check cluster health and report issues proactively
4. Answer questions about cluster resources
5. Perform operations like scaling, restarting pods, checking logs, etc.

When first started:
- Use check_all_clusters tool (NOT individual get_cluster_status calls) for fast parallel health checking
- Check BOTH node health AND pod health across all clusters
- Report any unhealthy pods found (Failed, Pending, CrashLoopBackOff, etc.)
- Provide a clear summary: either "All clusters are healthy ✓" or list specific issues found
- Ask "What would you like me to do?" to prompt user interaction

For presenting information:
- ALWAYS use Markdown tables for structured data (cluster comparisons, pod lists, node status, etc.)
- Keep table cells SHORT and CONCISE (max 20 characters per cell)
🚨 CRITICAL OUTPUT FORMATTING RULES 🚨

TOOL OUTPUT HANDLING:
- Present tool results VERBATIM - copy/paste without any changes
- NEVER create markdown tables (| | | format) in your responses
- NEVER reformat tool output into tables
- Do NOT summarize, restructure, or re-present tool results
- Tool output already has proper visual formatting (boxes, cards, emojis)
- Just show the tool output as-is and add brief commentary BELOW it

YOUR RESPONSE FORMAT:
- Show tool output first (unmodified)
- Add your analysis below using bullet points
- Use emojis + UPPERCASE for section headers (no markdown, no ANSI codes):
  * 🔵 STATUS:
  * ⚠️  POSSIBLE CAUSES:
  * ✅ NEXT STEPS:
  * 💡 SUGGESTION:
- Keep responses conversational and concise

For user requests:
- Use kubectl_exec tool to run kubectl commands when users ask for operations
- Always use the --context flag to specify which cluster
- Explain what you're doing before executing commands
- Show command output and interpret results for the user

Be conversational, proactive, and helpful!`
}

// startThinkingIndicator starts the animated gradient thinking line
func startThinkingIndicator(state *agentState) {
	if isJSONOutput(state.outputFormat) {
		return
	}

	state.isThinking = true
	state.thinkingStopChan = make(chan bool)

	go func() {
		colors := []string{
			gradientColor1, gradientColor2, gradientColor3,
			gradientColor4, gradientColor5, gradientColor6,
		}
		offset := 0
		barWidth := 60

		fmt.Print(cursorHide)       // Hide cursor during animation
		defer fmt.Print(cursorShow) // Show cursor when done

		for {
			select {
			case <-state.thinkingStopChan:
				fmt.Print("\r" + clearLine) // Clear the line
				return
			default:
				// Build gradient bar
				fmt.Print("\r")
				for i := 0; i < barWidth; i++ {
					colorIdx := (i + offset) % len(colors)
					fmt.Printf("%s━%s", colors[colorIdx], colorReset)
				}
				offset = (offset + 1) % len(colors)
				fmt.Print("\r") // Move cursor back to start

				// Sleep for animation frame
				// Use a select to allow immediate cancellation
				select {
				case <-state.thinkingStopChan:
					fmt.Print("\r" + clearLine)
					return
				case <-func() <-chan bool {
					ch := make(chan bool)
					go func() {
						// 80ms delay for smooth animation
						for i := 0; i < 8; i++ {
							select {
							case <-state.thinkingStopChan:
								close(ch)
								return
							default:
								// 10ms increments = 80ms total
								// This allows faster response to stop signal
								for j := 0; j < 1000000; j++ {
									// Busy wait for ~10ms
								}
							}
						}
						close(ch)
					}()
					return ch
				}():
				}
			}
		}
	}()
}

// stopThinkingIndicator stops the animated gradient thinking line
func stopThinkingIndicator(state *agentState) {
	if !state.isThinking {
		return
	}
	if state.thinkingStopChan != nil {
		close(state.thinkingStopChan)
		state.thinkingStopChan = nil
	}
	state.isThinking = false
}

// displayPersistentQuota shows the quota information persistently
func displayPersistentQuota(state *agentState) {
	if isJSONOutput(state.outputFormat) || state.quotaPercentage < 0 {
		return
	}

	statusIcon := getQuotaStatusIcon(state.quotaPercentage)
	quotaColor := getQuotaColor(state.quotaPercentage)
	fmt.Printf("%s[%s %s%.1f%% remaining%s]%s\n", colorDim, statusIcon, quotaColor, state.quotaPercentage, colorReset, colorReset)
}

// getQuotaStatusIcon returns the appropriate icon based on remaining percentage
func getQuotaStatusIcon(percentage float64) string {
	if percentage < 20 {
		return "🔴"
	} else if percentage < 50 {
		return "🟡"
	}
	return "🟢"
}

// getQuotaColor returns the appropriate color based on remaining percentage
func getQuotaColor(percentage float64) string {
	if percentage < 20 {
		return colorRed
	} else if percentage < 50 {
		return colorYellow
	}
	return colorGreen
}

// setupSessionEventHandler creates and returns an event handler for the session
func setupSessionEventHandler(session *copilot.Session, isIdlePtr *bool, state *agentState, messageBuffer *strings.Builder) {
	session.On(func(event copilot.SessionEvent) {
		switch event.Type {
		case "assistant.message_delta":
			// Stop thinking indicator on first content
			if event.Data.DeltaContent != nil && len(*event.Data.DeltaContent) > 0 {
				stopThinkingIndicator(state)
				// Show quota before first message if available
				if state.quotaDisplayed && messageBuffer.Len() == 0 {
					displayPersistentQuota(state)
				}
				fmt.Print(*event.Data.DeltaContent)
				messageBuffer.WriteString(*event.Data.DeltaContent)
			}
		case "assistant.message":
			stopThinkingIndicator(state)
			if messageBuffer.Len() == 0 && event.Data.Content != nil {
				// Show quota before message if available
				if state.quotaDisplayed {
					displayPersistentQuota(state)
				}
				fmt.Print(*event.Data.Content)
			}
			fmt.Println()
			messageBuffer.Reset()
		case "session.idle":
			stopThinkingIndicator(state)
			*isIdlePtr = true
		case "assistant.usage":
			handleUsageEvent(event, state)
		case "tool.execution_start":
			handleToolExecutionStart(event, state.outputFormat)
		}
	})
}

// handleUsageEvent processes usage events and displays quota information
func handleUsageEvent(event copilot.SessionEvent, state *agentState) {
	if event.Data.QuotaSnapshots == nil {
		return
	}

	if snapshot, exists := event.Data.QuotaSnapshots["premium_interactions"]; exists {
		if snapshot.RemainingPercentage >= 0 {
			state.quotaPercentage = snapshot.RemainingPercentage
			state.quotaDisplayed = true
		}
	}
}

// handleToolExecutionStart displays tool execution notifications
func handleToolExecutionStart(event copilot.SessionEvent, outputFormat OutputFormat) {
	// Tool execution happens silently - output is shown when complete
}

// Run starts the Copilot agent with Kubernetes cluster tools
func Run(k8sProvider *k8s.Provider, mode ExecutionMode, outputFormat OutputFormat) error {
	// Initialize agent state
	state := &agentState{
		mode:            mode,
		outputFormat:    outputFormat,
		quotaPercentage: -1,
		quotaDisplayed:  false,
		isThinking:      false,
	}

	// Create and start Copilot client
	client, err := createAndStartClient()
	if err != nil {
		return err
	}
	defer client.Stop()

	// Create initial session with cost-effective model
	session, err := createSessionWithModel(client, k8sProvider, state, modelCostEffective)
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
		// Display welcome message without consuming quota
		fmt.Printf("\n%s🚀 Kopilot - Kubernetes Cluster Assistant%s\n\n", colorBold+colorCyan, colorReset)

		// Display execution mode
		modeIcon := "🔒"
		if mode == ModeInteractive {
			modeIcon = "🔓"
		}
		fmt.Printf("%sExecution Mode:%s %s%s %s%s\n", colorDim, colorReset, modeIcon, colorBold, mode, colorReset)
		fmt.Printf("%sModel:%s %s%s%s\n\n", colorDim, colorReset, colorBold, modelCostEffective, colorReset)

		fmt.Printf("%sReady!%s\n\nI can help you with:\n", colorBold+colorGreen, colorReset)
		fmt.Printf("  %s•%s Cluster health checks (try: %scheck all clusters%s)\n", colorGreen, colorReset, colorCyan, colorReset)
		fmt.Printf("  %s•%s kubectl commands (try: %sshow me pods in default namespace%s)\n", colorGreen, colorReset, colorCyan, colorReset)
		fmt.Printf("  %s•%s Troubleshooting (try: %swhy is my pod failing?%s)\n", colorGreen, colorReset, colorCyan, colorReset)

		// Mode-specific instructions
		if mode == ModeReadOnly {
			fmt.Printf("\n%s⚠️  Read-only mode:%s Write operations are blocked. Use %s/interactive%s to enable writes.\n", colorYellow, colorReset, colorCyan, colorReset)
		} else {
			fmt.Printf("\n%s✓ Interactive mode:%s Write operations require confirmation. Use %s/readonly%s for read-only.\n", colorGreen, colorReset, colorCyan, colorReset)
		}

		fmt.Printf("\nType %s'exit'%s or %s'quit'%s to exit.\n\n", colorYellow, colorReset, colorYellow, colorReset)
	}

	// Mark as idle so user can start typing immediately
	isIdle = true

	// Interactive loop with session management
	return interactiveLoopWithModelSelection(client, k8sProvider, state, session, &isIdle, &messageBuffer)
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
func createAndStartClient() (*copilot.Client, error) {
	// Auto-detect Copilot CLI location
	cliPath, err := findCopilotCLI()
	if err != nil {
		return nil, err
	}

	log.Printf("Using Copilot CLI: %s", cliPath)

	// Verify the CLI is executable and authenticated
	if err := verifyCopilotCLI(cliPath); err != nil {
		return nil, fmt.Errorf("copilot CLI verification failed: %w\n\nPlease ensure GitHub Copilot is properly authenticated.\nYou may need to run: copilot auth login", err)
	}

	// Get current working directory for CLI context
	cwd, _ := os.Getwd()

	client := copilot.NewClient(&copilot.ClientOptions{
		CLIPath:     cliPath,
		Cwd:         cwd,
		UseStdio:    true,
		AutoStart:   boolPtr(true),
		AutoRestart: boolPtr(true),
		LogLevel:    "error",      // Reduce noise in logs
		Env:         os.Environ(), // Pass current environment
	})

	log.Println("Starting Copilot client...")
	if err := client.Start(); err != nil {
		return nil, fmt.Errorf("failed to start copilot client: %w\n\nTip: Ensure GitHub Copilot CLI is properly set up and authenticated", err)
	}

	log.Println("Copilot client started successfully")
	return client, nil
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

// verifyCopilotCLI checks if the Copilot CLI is accessible and working
func verifyCopilotCLI(cliPath string) error {
	// Check if file exists and is executable
	info, err := os.Stat(cliPath)
	if err != nil {
		return fmt.Errorf("CLI not accessible: %w", err)
	}

	// Check if it's executable (Unix-like systems)
	if info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("CLI is not executable: %s", cliPath)
	}

	// Try to run --version to verify it works
	cmd := exec.Command(cliPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("CLI execution failed: %w\nOutput: %s", err, string(output))
	}

	log.Printf("Copilot CLI version: %s", strings.TrimSpace(string(output)))
	return nil
}

// createSessionWithModel creates a new Copilot session with specified model
func createSessionWithModel(client *copilot.Client, k8sProvider *k8s.Provider, state *agentState, model string) (*copilot.Session, error) {
	tools := defineTools(k8sProvider, state)
	systemMessage := getSystemMessage()

	session, err := client.CreateSession(&copilot.SessionConfig{
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
	fmt.Print("\n> ")
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
			fmt.Printf("\n%sAlready in read-only mode%s\n", colorYellow, colorReset)
		} else {
			state.mode = ModeReadOnly
			fmt.Printf("\n%s🔒 Switched to read-only mode%s\n", colorGreen, colorReset)
			fmt.Printf("Write operations are now blocked. Use %s/interactive%s to enable writes.\n", colorCyan, colorReset)
		}
		return true

	case "/interactive", "/interactive on":
		if state.mode == ModeInteractive {
			fmt.Printf("\n%sAlready in interactive mode%s\n", colorYellow, colorReset)
		} else {
			state.mode = ModeInteractive
			fmt.Printf("\n%s🔓 Switched to interactive mode%s\n", colorGreen, colorReset)
			fmt.Printf("Write operations now require confirmation. Use %s/readonly%s for read-only mode.\n", colorCyan, colorReset)
		}
		return true

	case "/mode", "/status":
		modeIcon := "🔒"
		if state.mode == ModeInteractive {
			modeIcon = "🔓"
		}
		fmt.Printf("\n%sCurrent execution mode:%s %s%s %s%s\n", colorDim, colorReset, modeIcon, colorBold, state.mode, colorReset)
		return true
	}

	return false
}

// switchToModel switches to a different model by creating a new session
func switchToModel(
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
	newSession, err := createSessionWithModel(client, k8sProvider, state, newModel)
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
			fmt.Println("\nGoodbye! 👋")
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
			log.Printf("%sSwitching from %s to %s for query complexity%s", colorMagenta, currentModel, optimalModel, colorReset)
			currentSession, err = switchToModel(client, k8sProvider, state, currentSession, optimalModel, isIdle, messageBuffer)
			if err != nil {
				return err
			}
			currentModel = optimalModel
		}

		// Send user message
		*isIdle = false

		// Start thinking indicator
		startThinkingIndicator(state)

		_, err = currentSession.Send(copilot.MessageOptions{
			Prompt: input,
		})
		if err != nil {
			stopThinkingIndicator(state)
			return fmt.Errorf("failed to send message: %w", err)
		}
	}
}

// defineTools creates all the Kubernetes-related tools for the agent
