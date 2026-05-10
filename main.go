package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/e9169/kopilot/pkg/agent"
	"github.com/e9169/kopilot/pkg/k8s"
)

var (
	// Version information - set via ldflags during build
	// Example: go build -ldflags "-X main.version=1.2.3"
	version   = "dev"     // Default to "dev" for development builds
	buildDate = "unknown" // Set by build process
	gitCommit = "unknown" // Set by build process
)

func main() {
	// Parse command-line flags
	showVersion := flag.Bool("version", false, "Show version information")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	interactive := flag.Bool("interactive", false, "Enable interactive mode (asks before write operations)")
	defaultKubeconfig := os.Getenv("KUBECONFIG")
	if defaultKubeconfig == "" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			defaultKubeconfig = filepath.Join(homeDir, ".kube", "config")
		}
	}
	kubeconfig := flag.String("kubeconfig", defaultKubeconfig, "Path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	contextName := flag.String("context", "", "Override kubeconfig context")
	outputFormat := flag.String("output", string(agent.OutputText), "Output format: text or json")
	agentName := flag.String("agent", string(agent.AgentDefault), "Specialist agent persona: default, debugger, security, optimizer, gitops")
	mcpConfig := flag.String("mcp-config", "", "Path to MCP server config file (default: ~/.kopilot/mcp.json)")
	aiProvider := flag.String("ai-provider", "copilot", "AI provider to use: copilot, openai, gemini")
	flag.BoolVar(verbose, "v", false, "Enable verbose logging (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Kopilot - Kubernetes Cluster Status Agent\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  kopilot [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExecution Modes:\n")
		fmt.Fprintf(os.Stderr, "  Read-only (default): Blocks all write operations for safety\n")
		fmt.Fprintf(os.Stderr, "  Interactive (--interactive): Asks for confirmation before write operations\n")
		fmt.Fprintf(os.Stderr, "\nSpecialist Agents:\n")
		fmt.Fprintf(os.Stderr, "  default    Standard Kopilot persona\n")
		fmt.Fprintf(os.Stderr, "  debugger   Root cause analysis and pod failure diagnosis\n")
		fmt.Fprintf(os.Stderr, "  security   RBAC auditing and privilege escalation detection\n")
		fmt.Fprintf(os.Stderr, "  optimizer  Resource right-sizing and cost optimization\n")
		fmt.Fprintf(os.Stderr, "  gitops     Flux/ArgoCD sync status and drift detection\n")
		fmt.Fprintf(os.Stderr, "  sanitizer  Cluster linting, best-practice scoring, and compliance grading\n")
		fmt.Fprintf(os.Stderr, "\nRuntime Commands:\n")
		fmt.Fprintf(os.Stderr, "  /readonly          Switch to read-only mode\n")
		fmt.Fprintf(os.Stderr, "  /interactive       Switch to interactive mode\n")
		fmt.Fprintf(os.Stderr, "  /mode              Show current execution mode\n")
		fmt.Fprintf(os.Stderr, "  /agent             Show active agent and available agents\n")
		fmt.Fprintf(os.Stderr, "  /agent <name>      Switch to a different specialist agent\n")
		fmt.Fprintf(os.Stderr, "  /mcp list          List configured MCP servers\n")
		fmt.Fprintf(os.Stderr, "  /mcp add <n> <url> Add an MCP server\n")
		fmt.Fprintf(os.Stderr, "  /mcp delete <name> Remove an MCP server\n")
		fmt.Fprintf(os.Stderr, "\nAI Providers:\n")
		fmt.Fprintf(os.Stderr, "  copilot   GitHub Copilot (default)\n")
		fmt.Fprintf(os.Stderr, "              Auth: Copilot CLI token (automatic when gh copilot is installed)\n")
		fmt.Fprintf(os.Stderr, "              No extra environment variables required.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  openai    OpenAI or any OpenAI-compatible API (Ollama, Azure OpenAI, Anthropic…)\n")
		fmt.Fprintf(os.Stderr, "              OPENAI_API_KEY    API key (required; use 'none' for local models)\n")
		fmt.Fprintf(os.Stderr, "              OPENAI_BASE_URL   Custom base URL (optional, for non-OpenAI backends)\n")
		fmt.Fprintf(os.Stderr, "              Examples:\n")
		fmt.Fprintf(os.Stderr, "                OPENAI_API_KEY=sk-... kopilot --ai-provider=openai\n")
		fmt.Fprintf(os.Stderr, "                OPENAI_BASE_URL=http://localhost:11434/v1 kopilot --ai-provider=openai  # Ollama\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  gemini    Google Gemini (Gemini 1.5 / 2.0 / 2.5)\n")
		fmt.Fprintf(os.Stderr, "              GEMINI_API_KEY    API key from https://ai.google.dev (recommended)\n")
		fmt.Fprintf(os.Stderr, "              Alternatively, authenticate with Application Default Credentials:\n")
		fmt.Fprintf(os.Stderr, "                gcloud auth application-default login\n")
		fmt.Fprintf(os.Stderr, "              Example:\n")
		fmt.Fprintf(os.Stderr, "                GEMINI_API_KEY=AIza... kopilot --ai-provider=gemini\n")
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  KUBECONFIG        Path to kubeconfig file (default: ~/.kube/config)\n")
		fmt.Fprintf(os.Stderr, "  OPENAI_API_KEY    API key for --ai-provider=openai\n")
		fmt.Fprintf(os.Stderr, "  OPENAI_BASE_URL   Custom API base URL for OpenAI-compatible backends\n")
		fmt.Fprintf(os.Stderr, "  GEMINI_API_KEY    API key for --ai-provider=gemini\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  kopilot                                           # GitHub Copilot, read-only\n")
		fmt.Fprintf(os.Stderr, "  kopilot --interactive                             # interactive mode\n")
		fmt.Fprintf(os.Stderr, "  kopilot --agent debugger                         # debugging specialist\n")
		fmt.Fprintf(os.Stderr, "  kopilot --agent security                         # security auditor\n")
		fmt.Fprintf(os.Stderr, "  kopilot --agent optimizer                        # optimization specialist\n")
		fmt.Fprintf(os.Stderr, "  kopilot --agent gitops                           # GitOps specialist\n")
		fmt.Fprintf(os.Stderr, "  kopilot --agent sanitizer                        # cluster sanitizer\n")
		fmt.Fprintf(os.Stderr, "  OPENAI_API_KEY=sk-... kopilot --ai-provider=openai\n")
		fmt.Fprintf(os.Stderr, "  OPENAI_BASE_URL=http://localhost:11434/v1 kopilot --ai-provider=openai  # Ollama\n")
		fmt.Fprintf(os.Stderr, "  GEMINI_API_KEY=AIza... kopilot --ai-provider=gemini\n")
		fmt.Fprintf(os.Stderr, "  kopilot --mcp-config ./mcp.json                  # custom MCP server config\n")
		fmt.Fprintf(os.Stderr, "  kopilot -v                                        # verbose logging\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("kopilot version %s\n", version)
		fmt.Printf("  build date: %s\n", buildDate)
		fmt.Printf("  git commit: %s\n", gitCommit)
		os.Exit(0)
	}

	// Set log output based on verbose flag
	if !*verbose {
		log.SetFlags(0) // Remove timestamp for cleaner output
	}

	// Determine execution mode
	mode := agent.ModeReadOnly
	if *interactive {
		mode = agent.ModeInteractive
	}

	format := agent.OutputFormat(*outputFormat)
	if format != agent.OutputText && format != agent.OutputJSON {
		log.Fatalf("Invalid --output value: %s (use 'text' or 'json')", *outputFormat)
	}

	agentType, agentErr := agent.ParseAgentType(*agentName)
	if agentErr != nil {
		log.Fatalf("Invalid --agent value: %v", agentErr)
	}

	if err := run(mode, *kubeconfig, *contextName, format, agentType, *mcpConfig, *aiProvider); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run(mode agent.ExecutionMode, kubeconfigPath string, contextName string, outputFormat agent.OutputFormat, agentType agent.AgentType, mcpConfigPath string, providerName string) error {
	// Set version in agent package for display
	agent.AppVersion = version

	// Verify kubeconfig exists. The path comes from a CLI flag/env that the
	// operator controls; os.Stat only checks existence and does not expose content.
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) { // #nosec G703
		return fmt.Errorf("kubeconfig not found at %s: %w", kubeconfigPath, err)
	}

	log.Printf("Using kubeconfig: %s", kubeconfigPath)

	// Initialize Kubernetes provider
	k8sProvider, err := k8s.NewProvider(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to initialize kubernetes provider: %w", err)
	}

	if contextName != "" {
		if err := k8sProvider.SetCurrentContext(contextName); err != nil {
			return fmt.Errorf("failed to set context: %w", err)
		}
		log.Printf("Using context override: %s", contextName)
	}

	log.Printf("Successfully loaded %d cluster(s) from kubeconfig", len(k8sProvider.GetClusters()))

	// Initialize LLM provider
	provider, err := agent.NewProviderByName(providerName)
	if err != nil {
		return err
	}

	// Initialize and run the agent
	log.Println("Starting kopilot agent...")
	if err := agent.Run(k8sProvider, mode, outputFormat, agentType, mcpConfigPath, provider); err != nil {
		return fmt.Errorf("failed to run agent: %w", err)
	}

	return nil
}
