package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/spetr/mcp-mail/types"
)

// Manager handles configuration loading and saving
type Manager struct {
	mu       sync.RWMutex
	config   *Config
	filePath string
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		config: DefaultConfig(),
	}
}

// Load loads configuration from a file
func (m *Manager) Load(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, use defaults
			m.filePath = filePath
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Merge with defaults
	m.config = mergeWithDefaults(&cfg)
	m.filePath = filePath

	return nil
}

// Save saves the current configuration to the file
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.filePath == "" {
		return fmt.Errorf("no config file path set")
	}

	return m.SaveTo(m.filePath)
}

// SaveTo saves the configuration to a specific file
func (m *Manager) SaveTo(filePath string) error {
	m.mu.RLock()
	cfg := m.config
	m.mu.RUnlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Get returns the current configuration (read-only copy)
func (m *Manager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.config
}

// GetAccounts returns all configured accounts
func (m *Manager) GetAccounts() []types.AccountConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	accounts := make([]types.AccountConfig, len(m.config.Accounts))
	copy(accounts, m.config.Accounts)
	return accounts
}

// GetAccount returns a specific account by ID
func (m *Manager) GetAccount(id string) (types.AccountConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, acc := range m.config.Accounts {
		if acc.ID == id {
			return acc, true
		}
	}
	return types.AccountConfig{}, false
}

// AddAccount adds a new account
func (m *Manager) AddAccount(account types.AccountConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate ID
	for _, acc := range m.config.Accounts {
		if acc.ID == account.ID {
			return fmt.Errorf("account with ID '%s' already exists", account.ID)
		}
	}

	m.config.Accounts = append(m.config.Accounts, account)
	return nil
}

// UpdateAccount updates an existing account
func (m *Manager) UpdateAccount(account types.AccountConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, acc := range m.config.Accounts {
		if acc.ID == account.ID {
			m.config.Accounts[i] = account
			return nil
		}
	}

	return fmt.Errorf("account with ID '%s' not found", account.ID)
}

// RemoveAccount removes an account by ID
func (m *Manager) RemoveAccount(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, acc := range m.config.Accounts {
		if acc.ID == id {
			m.config.Accounts = append(m.config.Accounts[:i], m.config.Accounts[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("account with ID '%s' not found", id)
}

// GetSettings returns the current settings
func (m *Manager) GetSettings() Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Settings
}

// GetServerConfig returns the server configuration
func (m *Manager) GetServerConfig() ServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Server
}

// mergeWithDefaults merges loaded config with default values
func mergeWithDefaults(cfg *Config) *Config {
	defaults := DefaultConfig()

	// Merge settings
	if cfg.Settings.IdleTimeout == 0 {
		cfg.Settings.IdleTimeout = defaults.Settings.IdleTimeout
	}
	if cfg.Settings.MaxMessageSize == 0 {
		cfg.Settings.MaxMessageSize = defaults.Settings.MaxMessageSize
	}

	// Merge server config
	if cfg.Server.Transport == "" {
		cfg.Server.Transport = defaults.Server.Transport
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = defaults.Server.Host
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = defaults.Server.Port
	}
	if cfg.Server.Auth.Type == "" {
		cfg.Server.Auth.Type = defaults.Server.Auth.Type
	}
	if cfg.Server.Auth.HeaderName == "" {
		cfg.Server.Auth.HeaderName = "X-API-Key"
	}

	// Merge cache config
	cfg.Cache.MergeDefaults()

	return cfg
}
