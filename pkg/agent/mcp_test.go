package agent

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	testMCPConfigFile   = "mcp.json"
	testMCPServerName   = "my-server"
	testMCPServerURL    = "http://localhost:8080"
	testMCPServerName2  = "test-server"
	testMCPServerURL2   = "http://localhost:9090"
	testMCPServerNewURL = "http://new-url:9090"
	errListMCPServers   = "listMCPServers() error: %v"
)

// TestDefaultMCPConfigPath verifies it returns a non-empty path ending in mcp.json.
func TestDefaultMCPConfigPath(t *testing.T) {
	path := DefaultMCPConfigPath()
	if path == "" {
		t.Fatal("DefaultMCPConfigPath() returned empty string")
	}
	if filepath.Base(path) != testMCPConfigFile {
		t.Errorf("DefaultMCPConfigPath() = %q, want path ending with mcp.json", path)
	}
}

// TestLoadMCPConfigNonExistent verifies that a missing config file returns an empty config.
func TestLoadMCPConfigNonExistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), testMCPConfigFile)
	cfg, err := loadMCPConfig(path)
	if err != nil {
		t.Fatalf("loadMCPConfig() on nonexistent file returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("loadMCPConfig() returned nil config for nonexistent file")
	}
	if len(cfg.Servers) != 0 {
		t.Errorf("loadMCPConfig() on nonexistent file: got %d servers, want 0", len(cfg.Servers))
	}
}

// TestLoadMCPConfigInvalidJSON verifies that malformed JSON returns an error.
func TestLoadMCPConfigInvalidJSON(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mcp-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("this is not json {{{"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	_, err = loadMCPConfig(f.Name())
	if err == nil {
		t.Error("loadMCPConfig() with invalid JSON should return an error")
	}
}

// TestSaveMCPConfigAndLoad verifies a round-trip save and load of config.
func TestSaveMCPConfigAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subdir", testMCPConfigFile)
	cfg := &mcpConfig{
		Servers: []MCPServerConfig{
			{Name: testMCPServerName, Type: "http", URL: testMCPServerURL},
		},
	}

	if err := saveMCPConfig(path, cfg); err != nil {
		t.Fatalf("saveMCPConfig() error: %v", err)
	}

	loaded, err := loadMCPConfig(path)
	if err != nil {
		t.Fatalf("loadMCPConfig() after save returned error: %v", err)
	}
	if len(loaded.Servers) != 1 {
		t.Errorf("got %d servers after round-trip, want 1", len(loaded.Servers))
	}
	if loaded.Servers[0].Name != testMCPServerName {
		t.Errorf("server name = %q, want %q", loaded.Servers[0].Name, testMCPServerName)
	}
	if loaded.Servers[0].URL != testMCPServerURL {
		t.Errorf("server URL = %q, want %q", loaded.Servers[0].URL, testMCPServerURL)
	}
}

// TestSaveMCPConfigFileMode verifies the saved file has mode 0600.
func TestSaveMCPConfigFileMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), testMCPConfigFile)
	if err := saveMCPConfig(path, &mcpConfig{}); err != nil {
		t.Fatalf("saveMCPConfig() error: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat after save: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 0600", perm)
	}
}

// TestValidateMCPServerName verifies the name validation regex.
func TestValidateMCPServerName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"alphanumeric", "myserver", false},
		{"with dash", "my-server", false},
		{"with underscore", "my_server", false},
		{"with numbers", "server123", false},
		{"mixed", "my-server_123", false},
		{"single char", "a", false},
		{"max length 64", "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789__", false},
		{"empty", "", true},
		{"too long 65", "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789___", true},
		{"with space", "my server", true},
		{"with slash", "my/server", true},
		{"with dot", "my.server", true},
		{"with at", "my@server", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMCPServerName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMCPServerName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestAddMCPServerNew verifies adding a new server populates the config file.
func TestAddMCPServerNew(t *testing.T) {
	path := filepath.Join(t.TempDir(), testMCPConfigFile)

	entry := MCPServerConfig{Name: testMCPServerName2, URL: testMCPServerURL2}
	if err := addMCPServer(path, entry); err != nil {
		t.Fatalf("addMCPServer() error: %v", err)
	}

	servers, err := listMCPServers(path)
	if err != nil {
		t.Fatalf(errListMCPServers, err)
	}
	if len(servers) != 1 {
		t.Errorf("got %d servers, want 1", len(servers))
	}
	if servers[0].Name != testMCPServerName2 {
		t.Errorf("server name = %q, want %q", servers[0].Name, testMCPServerName2)
	}
	// Type should default to "http" when not set
	if servers[0].Type != "http" {
		t.Errorf("server type = %q, want %q", servers[0].Type, "http")
	}
}

// TestAddMCPServerUpdate verifies that adding a server with an existing name updates it.
func TestAddMCPServerUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), testMCPConfigFile)

	_ = addMCPServer(path, MCPServerConfig{Name: "server1", URL: "http://old-url:8080"})
	_ = addMCPServer(path, MCPServerConfig{Name: "server2", URL: "http://other:8081"})

	// Update server1 to a new URL
	if err := addMCPServer(path, MCPServerConfig{Name: "server1", URL: testMCPServerNewURL}); err != nil {
		t.Fatalf("addMCPServer() update error: %v", err)
	}

	servers, err := listMCPServers(path)
	if err != nil {
		t.Fatalf(errListMCPServers, err)
	}
	if len(servers) != 2 {
		t.Errorf("got %d servers after update, want 2", len(servers))
	}
	found := false
	for _, s := range servers {
		if s.Name == "server1" {
			found = true
			if s.URL != testMCPServerNewURL {
				t.Errorf("server1 URL = %q, want %q", s.URL, testMCPServerNewURL)
			}
		}
	}
	if !found {
		t.Error("server1 not found after update")
	}
}

// TestAddMCPServerValidation verifies validation errors are returned.
func TestAddMCPServerValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), testMCPConfigFile)

	// Invalid name
	err := addMCPServer(path, MCPServerConfig{Name: "bad name!", URL: "http://example.com"})
	if err == nil {
		t.Error("addMCPServer() with invalid name should return an error")
	}

	// Empty URL
	err = addMCPServer(path, MCPServerConfig{Name: "validname", URL: ""})
	if err == nil {
		t.Error("addMCPServer() with empty URL should return an error")
	}

	// Unsupported transport type
	err = addMCPServer(path, MCPServerConfig{Name: "validname", Type: "grpc", URL: "http://example.com"})
	if err == nil {
		t.Error("addMCPServer() with unsupported type should return an error")
	}
}

// TestDeleteMCPServerExisting verifies removing a known server.
func TestDeleteMCPServerExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), testMCPConfigFile)

	_ = addMCPServer(path, MCPServerConfig{Name: "server1", URL: "http://host1:8080"})
	_ = addMCPServer(path, MCPServerConfig{Name: "server2", URL: "http://host2:8080"})

	if err := deleteMCPServer(path, "server1"); err != nil {
		t.Fatalf("deleteMCPServer() error: %v", err)
	}

	servers, err := listMCPServers(path)
	if err != nil {
		t.Fatalf(errListMCPServers, err)
	}
	if len(servers) != 1 {
		t.Errorf("got %d servers after delete, want 1", len(servers))
	}
	if servers[0].Name != "server2" {
		t.Errorf("remaining server = %q, want %q", servers[0].Name, "server2")
	}
}

// TestDeleteMCPServerNotFound verifies that deleting non-existent server returns an error.
func TestDeleteMCPServerNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), testMCPConfigFile)

	err := deleteMCPServer(path, "nonexistent")
	if err == nil {
		t.Error("deleteMCPServer() on nonexistent server should return an error")
	}
}

// TestListMCPServersEmpty verifies listing an empty (nonexistent) config returns an empty slice.
func TestListMCPServersEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), testMCPConfigFile)

	servers, err := listMCPServers(path)
	if err != nil {
		t.Fatalf("listMCPServers() on missing config returned error: %v", err)
	}
	if len(servers) != 0 {
		t.Errorf("got %d servers from empty config, want 0", len(servers))
	}
}

// TestListMCPServersMultiple verifies listing several servers added in sequence.
func TestListMCPServersMultiple(t *testing.T) {
	path := filepath.Join(t.TempDir(), testMCPConfigFile)

	_ = addMCPServer(path, MCPServerConfig{Name: "s1", URL: "http://host1:8080"})
	_ = addMCPServer(path, MCPServerConfig{Name: "s2", URL: "http://host2:8080"})
	_ = addMCPServer(path, MCPServerConfig{Name: "s3", URL: "http://host3:8080"})

	servers, err := listMCPServers(path)
	if err != nil {
		t.Fatalf(errListMCPServers, err)
	}
	if len(servers) != 3 {
		t.Errorf("got %d servers, want 3", len(servers))
	}
}
