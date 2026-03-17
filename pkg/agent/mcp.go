package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// mcpHTTPType is the only supported MCP server transport type.
const mcpHTTPType = "http"

// mcpServerName is the allowed pattern for MCP server names.
var mcpServerNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// MCPServerConfig holds the configuration for a single MCP server.
type MCPServerConfig struct {
	// Name is the unique identifier used to reference this server.
	Name string `json:"name"`
	// Type is the transport type, currently only "http" is supported.
	Type string `json:"type"`
	// URL is the HTTP(S) endpoint of the MCP server.
	URL string `json:"url"`
}

// mcpConfig is the top-level structure stored in the MCP config file.
type mcpConfig struct {
	Servers []MCPServerConfig `json:"servers"`
}

// DefaultMCPConfigPath returns the default path for the MCP config file:
// $HOME/.kopilot/mcp.json, falling back to ".kopilot/mcp.json" on error.
func DefaultMCPConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".kopilot", "mcp.json")
	}
	return filepath.Join(home, ".kopilot", "mcp.json")
}

// loadMCPConfig reads the MCP config file from path.
// If the file does not exist an empty config is returned without error.
func loadMCPConfig(path string) (*mcpConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &mcpConfig{Servers: []MCPServerConfig{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading MCP config: %w", err)
	}

	var cfg mcpConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing MCP config: %w", err)
	}
	if cfg.Servers == nil {
		cfg.Servers = []MCPServerConfig{}
	}
	return &cfg, nil
}

// saveMCPConfig writes the config to path, creating parent directories as needed.
func saveMCPConfig(path string, cfg *mcpConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating MCP config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding MCP config: %w", err)
	}
	// Write atomically via temp file in the same directory.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("writing MCP config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("saving MCP config: %w", err)
	}
	return nil
}

// validateMCPServerName returns an error when name contains disallowed characters.
func validateMCPServerName(name string) error {
	if !mcpServerNameRe.MatchString(name) {
		return fmt.Errorf("invalid MCP server name %q: must match [a-zA-Z0-9_-] and be 1–64 characters", name)
	}
	return nil
}

// addMCPServer adds (or updates) a server entry in the config file at path.
func addMCPServer(path string, entry MCPServerConfig) error {
	if err := validateMCPServerName(entry.Name); err != nil {
		return err
	}
	if entry.URL == "" {
		return fmt.Errorf("URL must not be empty")
	}
	// Only "http" transport is supported by the SDK today.
	if entry.Type == "" {
		entry.Type = mcpHTTPType
	}
	if entry.Type != mcpHTTPType {
		return fmt.Errorf("unsupported MCP server type %q: only \"http\" is supported", entry.Type)
	}

	cfg, err := loadMCPConfig(path)
	if err != nil {
		return err
	}

	for i, s := range cfg.Servers {
		if s.Name == entry.Name {
			cfg.Servers[i] = entry
			return saveMCPConfig(path, cfg)
		}
	}
	cfg.Servers = append(cfg.Servers, entry)
	return saveMCPConfig(path, cfg)
}

// deleteMCPServer removes the named server from the config file at path.
// Returns an error if the name does not exist.
func deleteMCPServer(path, name string) error {
	cfg, err := loadMCPConfig(path)
	if err != nil {
		return err
	}

	newServers := cfg.Servers[:0]
	found := false
	for _, s := range cfg.Servers {
		if s.Name == name {
			found = true
			continue
		}
		newServers = append(newServers, s)
	}
	if !found {
		return fmt.Errorf("MCP server %q not found", name)
	}
	cfg.Servers = newServers
	return saveMCPConfig(path, cfg)
}

// listMCPServers returns all servers from the config file at path.
func listMCPServers(path string) ([]MCPServerConfig, error) {
	cfg, err := loadMCPConfig(path)
	if err != nil {
		return nil, err
	}
	return cfg.Servers, nil
}
