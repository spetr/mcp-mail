package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spetr/mcp-mail/types"
	"github.com/mark3labs/mcp-go/mcp"
)

func (r *Registry) registerAccountTools() {
	// account_list - List all configured accounts
	r.addTool(
		mcp.NewTool("account_list",
			mcp.WithDescription("List all configured IMAP accounts and their connection status"),
		),
		r.handleAccountList,
	)

	// account_add - Add a new account
	r.addTool(
		mcp.NewTool("account_add",
			mcp.WithDescription("Add a new IMAP account configuration"),
			requiredString("id", "Unique identifier for the account"),
			requiredString("name", "Display name for the account"),
			requiredString("host", "IMAP server hostname"),
			requiredNumber("port", "IMAP server port (usually 993 for TLS)"),
			optionalBool("tls", "Use TLS connection (default: true)"),
			requiredString("username", "IMAP username/email"),
			requiredString("password", "IMAP password or app password"),
		),
		r.handleAccountAdd,
	)

	// account_remove - Remove an account
	r.addTool(
		mcp.NewTool("account_remove",
			mcp.WithDescription("Remove an IMAP account configuration"),
			requiredString("id", "Account ID to remove"),
		),
		r.handleAccountRemove,
	)

	// account_connect - Connect to an account
	r.addTool(
		mcp.NewTool("account_connect",
			mcp.WithDescription("Connect to an IMAP account"),
			requiredString("id", "Account ID to connect to"),
		),
		r.handleAccountConnect,
	)

	// account_disconnect - Disconnect from an account
	r.addTool(
		mcp.NewTool("account_disconnect",
			mcp.WithDescription("Disconnect from an IMAP account"),
			requiredString("id", "Account ID to disconnect from"),
		),
		r.handleAccountDisconnect,
	)

	// account_status - Get account status
	r.addTool(
		mcp.NewTool("account_status",
			mcp.WithDescription("Get the connection status of an account"),
			requiredString("id", "Account ID to check"),
		),
		r.handleAccountStatus,
	)
}

func (r *Registry) handleAccountList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accounts := r.configMgr.GetAccounts()
	statuses := r.imapMgr.GetStatus(accounts)

	result, err := json.Marshal(statuses)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleAccountAdd(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetString("id", "")
	name := request.GetString("name", "")
	host := request.GetString("host", "")
	port := getNumber(request, "port", 993)
	tls := request.GetBool("tls", true)
	username := request.GetString("username", "")
	password := request.GetString("password", "")

	account := types.AccountConfig{
		ID:       id,
		Name:     name,
		Host:     host,
		Port:     port,
		TLS:      tls,
		Username: username,
		Password: password,
	}

	if err := r.configMgr.AddAccount(account); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add account: %v", err)), nil
	}

	// Save config to file
	if err := r.configMgr.Save(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("account added but failed to save config: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Account '%s' added successfully", id)), nil
}

func (r *Registry) handleAccountRemove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetString("id", "")

	// Disconnect if connected
	if r.imapMgr.IsConnected(id) {
		r.imapMgr.Disconnect(id)
	}

	if err := r.configMgr.RemoveAccount(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove account: %v", err)), nil
	}

	// Save config to file
	if err := r.configMgr.Save(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("account removed but failed to save config: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Account '%s' removed successfully", id)), nil
}

func (r *Registry) handleAccountConnect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetString("id", "")

	account, ok := r.configMgr.GetAccount(id)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("account '%s' not found", id)), nil
	}

	if err := r.imapMgr.Connect(account); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to connect: %v", err)), nil
	}

	// Build response with memory info if available
	response := map[string]interface{}{
		"status":     "connected",
		"account_id": id,
		"message":    fmt.Sprintf("Connected to account '%s' successfully", id),
	}

	// Load and include memory summary
	if r.memoryMgr != nil {
		mem, err := r.memoryMgr.GetOrLoad(id)
		if err == nil {
			// Start session
			r.memoryMgr.StartSession(id)

			summary := mem.GetSummary()
			response["memory"] = summary
			response["memory_hint"] = "Account memory loaded. You can update it using memory_* tools as you learn user preferences. Important contacts and spam rules are available in the memory."
		}
	}

	result, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleAccountDisconnect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetString("id", "")

	if err := r.imapMgr.Disconnect(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to disconnect: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Disconnected from account '%s'", id)), nil
}

func (r *Registry) handleAccountStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetString("id", "")

	account, ok := r.configMgr.GetAccount(id)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("account '%s' not found", id)), nil
	}

	status := types.AccountStatus{
		ID:        account.ID,
		Name:      account.Name,
		Connected: r.imapMgr.IsConnected(id),
	}

	if status.Connected {
		client, err := r.imapMgr.GetClient(id)
		if err == nil {
			status.Capabilities = client.Capabilities()
		}
	}

	result, err := json.Marshal(status)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}
