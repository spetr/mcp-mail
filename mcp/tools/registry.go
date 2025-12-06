package tools

import (
	"log"

	"github.com/spetr/mcp-mail/config"
	"github.com/spetr/mcp-mail/imap"
	"github.com/spetr/mcp-mail/memory"
	"github.com/spetr/mcp-mail/security"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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
	protectMgr *security.ProtectionManager
	tools      []Tool
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

	r := &Registry{
		configMgr:  configMgr,
		imapMgr:    imapMgr,
		memoryMgr:  memMgr,
		protectMgr: security.NewProtectionManager(&cfg.Security),
		tools:      make([]Tool, 0),
	}

	// Register all tools
	r.registerAccountTools()
	r.registerFolderTools()
	r.registerMessageTools()
	r.registerThreadTools()
	r.registerFlagTools()
	r.registerSearchTools()
	r.registerBulkTools()
	if memMgr != nil {
		r.registerMemoryTools()
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
