package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spetr/mcp-mail/types"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test default settings
	if cfg.Settings.IdleTimeout != 300 {
		t.Errorf("expected IdleTimeout 300, got %d", cfg.Settings.IdleTimeout)
	}
	if cfg.Settings.MaxMessageSize != 10*1024*1024 {
		t.Errorf("expected MaxMessageSize 10MB, got %d", cfg.Settings.MaxMessageSize)
	}
	if cfg.Settings.AutoConnect != false {
		t.Errorf("expected AutoConnect false, got %v", cfg.Settings.AutoConnect)
	}

	// Test default server config
	if cfg.Server.Transport != "stdio" {
		t.Errorf("expected Transport 'stdio', got %s", cfg.Server.Transport)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("expected Host 'localhost', got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected Port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Auth.Type != "none" {
		t.Errorf("expected Auth.Type 'none', got %s", cfg.Server.Auth.Type)
	}

	// Test default security config
	if cfg.Security.MaxBulkOperations != 1000 {
		t.Errorf("expected MaxBulkOperations 1000, got %d", cfg.Security.MaxBulkOperations)
	}
	if len(cfg.Security.ProtectedFolders) != 1 || cfg.Security.ProtectedFolders[0] != "INBOX" {
		t.Errorf("expected ProtectedFolders ['INBOX'], got %v", cfg.Security.ProtectedFolders)
	}
	if cfg.Security.ReadOnly != false {
		t.Errorf("expected ReadOnly false, got %v", cfg.Security.ReadOnly)
	}
}

func TestDefaultSecurityConfig(t *testing.T) {
	cfg := DefaultSecurityConfig()

	if cfg.MaxBulkOperations != 1000 {
		t.Errorf("expected MaxBulkOperations 1000, got %d", cfg.MaxBulkOperations)
	}
	if len(cfg.ProtectedFolders) != 1 || cfg.ProtectedFolders[0] != "INBOX" {
		t.Errorf("expected ProtectedFolders ['INBOX'], got %v", cfg.ProtectedFolders)
	}
}

func TestSecurityConfigMergeDefaults(t *testing.T) {
	// Test with zero values
	cfg := SecurityConfig{}
	cfg.MergeDefaults()

	if cfg.MaxBulkOperations != 1000 {
		t.Errorf("expected MaxBulkOperations 1000 after merge, got %d", cfg.MaxBulkOperations)
	}
	if len(cfg.ProtectedFolders) != 1 {
		t.Errorf("expected 1 ProtectedFolder after merge, got %d", len(cfg.ProtectedFolders))
	}

	// Test with existing values (should not be overwritten)
	cfg2 := SecurityConfig{
		MaxBulkOperations: 500,
		ProtectedFolders:  []string{"INBOX", "Sent"},
	}
	cfg2.MergeDefaults()

	if cfg2.MaxBulkOperations != 500 {
		t.Errorf("expected MaxBulkOperations 500 (not merged), got %d", cfg2.MaxBulkOperations)
	}
	if len(cfg2.ProtectedFolders) != 2 {
		t.Errorf("expected 2 ProtectedFolders (not merged), got %d", len(cfg2.ProtectedFolders))
	}
}

func TestManagerLoadAndSave(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test-config.json")

	// Create manager and add account
	mgr := NewManager()
	acc := types.AccountConfig{
		ID:       "test",
		Name:     "Test Account",
		Host:     "imap.test.com",
		Port:     993,
		TLS:      true,
		Username: "user@test.com",
		Password: "secret",
	}
	mgr.AddAccount(acc)

	// Save config
	if err := mgr.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load in new manager
	mgr2 := NewManager()
	if err := mgr2.Load(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	accounts := mgr2.GetAccounts()
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account after load, got %d", len(accounts))
	}
	if accounts[0].ID != "test" {
		t.Errorf("expected account ID 'test', got %s", accounts[0].ID)
	}
	if accounts[0].Host != "imap.test.com" {
		t.Errorf("expected host 'imap.test.com', got %s", accounts[0].Host)
	}
}

func TestManagerLoadNonExistent(t *testing.T) {
	mgr := NewManager()
	// Loading non-existent file should not return error - it uses defaults
	err := mgr.Load("/nonexistent/path/config.json")
	// The behavior is: if file doesn't exist, it's not an error, just use defaults
	// But if the path's directory doesn't exist, it will fail to set up later saves
	// Let's test with a valid directory but non-existent file
	tempDir, _ := os.MkdirTemp("", "config-test-*")
	defer os.RemoveAll(tempDir)

	err = mgr.Load(filepath.Join(tempDir, "nonexistent.json"))
	if err != nil {
		t.Errorf("expected no error for non-existent file (should use defaults), got: %v", err)
	}
}

func TestManagerAddRemoveAccount(t *testing.T) {
	mgr := NewManager()

	// Add account
	acc := types.AccountConfig{
		ID:   "gmail",
		Name: "Gmail",
		Host: "imap.gmail.com",
		Port: 993,
		TLS:  true,
	}

	if err := mgr.AddAccount(acc); err != nil {
		t.Fatalf("failed to add account: %v", err)
	}

	// Get account
	retrieved, ok := mgr.GetAccount("gmail")
	if !ok {
		t.Fatal("account not found after adding")
	}
	if retrieved.Host != "imap.gmail.com" {
		t.Errorf("expected host 'imap.gmail.com', got %s", retrieved.Host)
	}

	// Try adding duplicate
	if err := mgr.AddAccount(acc); err == nil {
		t.Error("expected error adding duplicate account")
	}

	// Remove account
	if err := mgr.RemoveAccount("gmail"); err != nil {
		t.Fatalf("failed to remove account: %v", err)
	}

	// Verify removed
	_, ok = mgr.GetAccount("gmail")
	if ok {
		t.Error("account still found after removal")
	}

	// Remove non-existent
	if err := mgr.RemoveAccount("nonexistent"); err == nil {
		t.Error("expected error removing non-existent account")
	}
}

func TestManagerGetAccounts(t *testing.T) {
	mgr := NewManager()

	acc1 := types.AccountConfig{ID: "acc1", Name: "Account 1", Host: "imap1.test.com", Port: 993, TLS: true}
	acc2 := types.AccountConfig{ID: "acc2", Name: "Account 2", Host: "imap2.test.com", Port: 993, TLS: true}

	mgr.AddAccount(acc1)
	mgr.AddAccount(acc2)

	accounts := mgr.GetAccounts()
	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(accounts))
	}
}

func TestManagerGetServerConfig(t *testing.T) {
	mgr := NewManager()

	// Get returns a copy, so we need to modify the internal config differently
	// The Get() method returns a copy, so modifications don't affect internal state
	// We need to test GetServerConfig returns the defaults

	serverCfg := mgr.GetServerConfig()
	if serverCfg.Transport != "stdio" {
		t.Errorf("expected default transport 'stdio', got %s", serverCfg.Transport)
	}
	if serverCfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", serverCfg.Port)
	}
}

func TestManagerLoadWithOverrides(t *testing.T) {
	// Create temp config file with custom values
	tempDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"server": {
			"transport": "sse",
			"port": 9000
		},
		"accounts": [
			{
				"id": "test",
				"name": "Test",
				"host": "imap.test.com",
				"port": 993,
				"tls": true
			}
		]
	}`
	os.WriteFile(configPath, []byte(configContent), 0600)

	mgr := NewManager()
	if err := mgr.Load(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	serverCfg := mgr.GetServerConfig()
	if serverCfg.Transport != "sse" {
		t.Errorf("expected transport 'sse', got %s", serverCfg.Transport)
	}
	if serverCfg.Port != 9000 {
		t.Errorf("expected port 9000, got %d", serverCfg.Port)
	}

	accounts := mgr.GetAccounts()
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}
}

func TestManagerUpdateAccount(t *testing.T) {
	mgr := NewManager()

	acc := types.AccountConfig{
		ID:       "test",
		Name:     "Test Account",
		Host:     "old.host.com",
		Port:     993,
		TLS:      true,
		Username: "user@test.com",
	}
	mgr.AddAccount(acc)

	// Update account
	acc.Host = "new.host.com"
	if err := mgr.UpdateAccount(acc); err != nil {
		t.Fatalf("failed to update account: %v", err)
	}

	// Verify update
	retrieved, _ := mgr.GetAccount("test")
	if retrieved.Host != "new.host.com" {
		t.Errorf("expected host 'new.host.com', got %s", retrieved.Host)
	}

	// Update non-existent account
	nonExistent := types.AccountConfig{ID: "nonexistent"}
	if err := mgr.UpdateAccount(nonExistent); err == nil {
		t.Error("expected error updating non-existent account")
	}
}

func TestManagerGetSettings(t *testing.T) {
	mgr := NewManager()
	settings := mgr.GetSettings()

	// Should return defaults
	if settings.IdleTimeout != 300 {
		t.Errorf("expected IdleTimeout 300, got %d", settings.IdleTimeout)
	}
	if settings.MaxMessageSize != 10*1024*1024 {
		t.Errorf("expected MaxMessageSize 10MB, got %d", settings.MaxMessageSize)
	}
}

func TestManagerSaveWithoutPath(t *testing.T) {
	mgr := NewManager()
	// Save without a path should fail
	err := mgr.Save()
	if err == nil {
		t.Error("expected error when saving without path")
	}
}

func TestMergeWithDefaults(t *testing.T) {
	// Test that merging fills in defaults for missing values
	cfg := &Config{
		Server: ServerConfig{
			Transport: "sse", // Custom value
			// Host and Port should get defaults
		},
	}

	merged := mergeWithDefaults(cfg)

	if merged.Server.Transport != "sse" {
		t.Error("custom transport should be preserved")
	}
	if merged.Server.Host != "localhost" {
		t.Errorf("expected default host 'localhost', got %s", merged.Server.Host)
	}
	if merged.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", merged.Server.Port)
	}
	if merged.Settings.IdleTimeout != 300 {
		t.Errorf("expected default IdleTimeout 300, got %d", merged.Settings.IdleTimeout)
	}
}
