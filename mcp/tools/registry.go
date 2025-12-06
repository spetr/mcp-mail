package tools

import (
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spetr/mcp-mail/config"
	"github.com/spetr/mcp-mail/imap"
	"github.com/spetr/mcp-mail/memory"
	"github.com/spetr/mcp-mail/security"
	"github.com/spetr/mcp-mail/smtp"
)

// maxBulkOperations is the maximum number of UIDs allowed per bulk operation
const maxBulkOperations = 1000

// Tool represents a registered MCP tool
type Tool struct {
	Definition mcp.Tool
	Handler    server.ToolHandlerFunc
}

// Registry holds all registered tools and their dependencies
type Registry struct {
	configMgr  *config.Manager
	imapMgr    *imap.Manager
	memoryMgr  *memory.Manager
	smtpMgr    *smtp.Manager
	protectMgr *security.ProtectionManager
	tools      []Tool

	// Track current active account for context switching detection
	currentAccountID string
}

// NewRegistry creates a new tool registry
func NewRegistry(configMgr *config.Manager, imapMgr *imap.Manager) *Registry {
	cfg := configMgr.Get()

	// Ensure security config has defaults
	cfg.Security.MergeDefaults()

	// Create memory manager
	memMgr, err := memory.NewManager("")
	if err != nil {
		log.Printf("Warning: Failed to initialize memory manager: %v (memory tools disabled)", err)
		memMgr = nil
	}

	// Create SMTP manager
	smtpMgr := smtp.NewManager(configMgr)

	r := &Registry{
		configMgr:  configMgr,
		imapMgr:    imapMgr,
		memoryMgr:  memMgr,
		smtpMgr:    smtpMgr,
		protectMgr: security.NewProtectionManager(&cfg.Security),
		tools:      make([]Tool, 0),
	}

	// Register all tools
	r.registerInstructionTools()
	r.registerAccountTools()
	r.registerFolderTools()
	r.registerMessageTools()
	r.registerThreadTools()
	r.registerFlagTools()
	r.registerSearchTools()
	r.registerBulkTools()
	r.registerSendTools()
	if memMgr != nil {
		r.registerMemoryTools()
		r.registerAnalyzeTools()
	}

	return r
}

// GetAllTools returns all registered tools
func (r *Registry) GetAllTools() []Tool {
	return r.tools
}

// addTool adds a tool to the registry
func (r *Registry) addTool(definition mcp.Tool, handler server.ToolHandlerFunc) {
	r.tools = append(r.tools, Tool{
		Definition: definition,
		Handler:    handler,
	})
}

// Helper function to create a required string parameter
func requiredString(name, description string) mcp.ToolOption {
	return mcp.WithString(name, mcp.Required(), mcp.Description(description))
}

// Helper function to create an optional string parameter
func optionalString(name, description string) mcp.ToolOption {
	return mcp.WithString(name, mcp.Description(description))
}

// Helper function to create an optional number parameter
func optionalNumber(name, description string) mcp.ToolOption {
	return mcp.WithNumber(name, mcp.Description(description))
}

// Helper function to create a required number parameter
func requiredNumber(name, description string) mcp.ToolOption {
	return mcp.WithNumber(name, mcp.Required(), mcp.Description(description))
}

// Helper function to create an optional boolean parameter
func optionalBool(name, description string) mcp.ToolOption {
	return mcp.WithBoolean(name, mcp.Description(description))
}

// getAccountSecret returns the HMAC secret for an account
// If account has no secret (legacy), generates and saves one
func (r *Registry) getAccountSecret(accountID string) (string, error) {
	account, ok := r.configMgr.GetAccount(accountID)
	if !ok {
		return "", fmt.Errorf("account '%s' not found", accountID)
	}

	// If account has no secret (legacy account), generate one
	if account.Secret == "" {
		secret, err := generateAccountSecret()
		if err != nil {
			return "", fmt.Errorf("failed to generate secret: %v", err)
		}
		account.Secret = secret

		// Update account with new secret
		if err := r.configMgr.UpdateAccount(account); err != nil {
			return "", fmt.Errorf("failed to save secret: %v", err)
		}
		if err := r.configMgr.Save(); err != nil {
			log.Printf("Warning: Failed to persist account secret: %v", err)
		}
	}

	return account.Secret, nil
}
