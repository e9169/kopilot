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
		fmt.Fprintf(os.Stderr, "\nRuntime Commands:\n")
		fmt.Fprintf(os.Stderr, "  /readonly          Switch to read-only mode\n")
		fmt.Fprintf(os.Stderr, "  /interactive       Switch to interactive mode\n")
		fmt.Fprintf(os.Stderr, "  /mode              Show current execution mode\n")
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  KUBECONFIG    Path to kubeconfig file (default: ~/.kube/config)\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  kopilot                          # Start in read-only mode\n")
		fmt.Fprintf(os.Stderr, "  kopilot --interactive            # Start in interactive mode\n")
		fmt.Fprintf(os.Stderr, "  kopilot -v                       # Start with verbose logging\n")
		fmt.Fprintf(os.Stderr, "  KUBECONFIG=./custom kopilot      # Use custom kubeconfig\n")
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

	if err := run(mode, *kubeconfig, *contextName, format); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run(mode agent.ExecutionMode, kubeconfigPath string, contextName string, outputFormat agent.OutputFormat) error {
	// Verify kubeconfig exists
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
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

	// Initialize and run the agent
	log.Println("Starting kopilot agent...")
	if err := agent.Run(k8sProvider, mode, outputFormat); err != nil {
		return fmt.Errorf("failed to run agent: %w", err)
	}

	return nil
}
