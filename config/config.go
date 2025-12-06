package config

import (
	"github.com/spetr/mcp-mail/cache"
	"github.com/spetr/mcp-mail/types"
)

// Config represents the application configuration
type Config struct {
	Accounts []types.AccountConfig `json:"accounts"`
	Settings Settings              `json:"settings"`
	Server   ServerConfig          `json:"server"`
	Security SecurityConfig        `json:"security"`
	Cache    cache.Config          `json:"cache"`
}

// Settings contains general application settings
type Settings struct {
	// IdleTimeout in seconds for IMAP IDLE
	IdleTimeout int `json:"idle_timeout,omitempty"`
	// MaxMessageSize in bytes (default 10MB)
	MaxMessageSize int64 `json:"max_message_size,omitempty"`
	// AutoConnect automatically connect to accounts on startup
	AutoConnect bool `json:"auto_connect,omitempty"`
}

// ServerConfig contains MCP server settings
type ServerConfig struct {
	// Transport: "stdio" or "sse"
	Transport string `json:"transport,omitempty"`
	// Host for SSE server (default "localhost")
	Host string `json:"host,omitempty"`
	// Port for SSE server (default 8080)
	Port int `json:"port,omitempty"`
	// Auth configuration for SSE
	Auth AuthConfig `json:"auth,omitempty"`
}

// AuthConfig contains authentication settings for SSE transport
type AuthConfig struct {
	// Type: "none", "bearer", "basic", "api_key"
	Type string `json:"type,omitempty"`
	// Token for bearer auth
	Token string `json:"token,omitempty"`
	// Username for basic auth
	Username string `json:"username,omitempty"`
	// Password for basic auth
	Password string `json:"password,omitempty"`
	// HeaderName for api_key auth (default "X-API-Key")
	HeaderName string `json:"header_name,omitempty"`
}

// SecurityConfig contains security settings for operations
type SecurityConfig struct {
	// MaxBulkOperations - maximum messages per single bulk operation (default 1000)
	MaxBulkOperations int `json:"max_bulk_operations,omitempty"`
	// ProtectedFolders - folders that cannot be deleted
	ProtectedFolders []string `json:"protected_folders,omitempty"`
	// TrashFolder - folder to use for delete (default auto-detected from server)
	TrashFolder string `json:"trash_folder,omitempty"`
	// ReadOnly - disable all write operations (default false)
	ReadOnly bool `json:"read_only,omitempty"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Accounts: []types.AccountConfig{},
		Settings: Settings{
			IdleTimeout:    300,
			MaxMessageSize: 10 * 1024 * 1024, // 10MB
			AutoConnect:    false,
		},
		Server: ServerConfig{
			Transport: "stdio",
			Host:      "localhost",
			Port:      8080,
			Auth: AuthConfig{
				Type: "none",
			},
		},
		Security: SecurityConfig{
			MaxBulkOperations: 1000,
			ProtectedFolders:  []string{"INBOX"},
			ReadOnly:          false,
		},
		Cache: cache.DefaultConfig(),
	}
}

// DefaultSecurityConfig returns default security settings
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		MaxBulkOperations: 1000,
		ProtectedFolders:  []string{"INBOX"},
		ReadOnly:          false,
	}
}

// MergeDefaults fills in zero values with defaults
// Note: ReadOnly defaults to false which is the zero value, so it doesn't need merging
func (s *SecurityConfig) MergeDefaults() {
	defaults := DefaultSecurityConfig()
	if s.MaxBulkOperations == 0 {
		s.MaxBulkOperations = defaults.MaxBulkOperations
	}
	if len(s.ProtectedFolders) == 0 {
		s.ProtectedFolders = defaults.ProtectedFolders
	}
}
