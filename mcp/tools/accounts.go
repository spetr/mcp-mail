package tools

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spetr/mcp-mail/types"
)

// generateAccountSecret generates a random 32-byte secret for HMAC-based safe_id
func generateAccountSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

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

	// Build enriched response with memory state
	type AccountWithMemory struct {
		types.AccountStatus
		HasProfile     bool   `json:"has_profile"`
		ProfileType    string `json:"profile_type,omitempty"`
		LastAnalyzed   string `json:"last_analyzed,omitempty"`
		NeedsAnalysis  bool   `json:"needs_analysis"`
		SenderCount    int    `json:"sender_count,omitempty"`
		ContactCount   int    `json:"contact_count,omitempty"`
	}

	enrichedAccounts := make([]AccountWithMemory, 0, len(statuses))
	for _, status := range statuses {
		enriched := AccountWithMemory{
			AccountStatus: status,
			NeedsAnalysis: true,
		}

		// Add memory info if available
		if r.memoryMgr != nil {
			if mem, err := r.memoryMgr.GetOrLoad(status.ID); err == nil {
				enriched.HasProfile = !mem.Profile.LastAnalyzed.IsZero()
				enriched.ProfileType = mem.Profile.Type
				if !mem.Profile.LastAnalyzed.IsZero() {
					enriched.LastAnalyzed = mem.Profile.LastAnalyzed.Format("2006-01-02")
					enriched.NeedsAnalysis = false
				}
				enriched.SenderCount = len(mem.Profile.SenderProfiles)
				enriched.ContactCount = len(mem.ImportantContacts)
			}
		}

		enrichedAccounts = append(enrichedAccounts, enriched)
	}

	// Build response with instructions
	response := map[string]interface{}{
		"accounts": enrichedAccounts,
		"count":    len(enrichedAccounts),
	}

	// Add current account info
	if r.currentAccountID != "" {
		response["current_account"] = r.currentAccountID
	}

	// Add instructions for AI
	var instructions []string
	instructions = append(instructions, "📧 EMAIL ASSISTANT INSTRUCTIONS:")
	instructions = append(instructions, "1. Before reading emails, connect to an account using account_connect(id='...')")
	instructions = append(instructions, "2. If account has needs_analysis=true, run account_analyze to build sender profile")
	instructions = append(instructions, "3. Use memory_get to see saved contacts, preferences, and sender classifications")
	instructions = append(instructions, "4. When user says something is important/to ignore - ALWAYS save it using memory_set_sender or memory_add_contact")
	instructions = append(instructions, "5. Available memory tools: memory_set_sender, memory_add_contact, memory_add_note, memory_add_rule, memory_add_unwanted")

	// Add specific action hints
	var actionHints []string
	for _, acc := range enrichedAccounts {
		if acc.Connected && acc.NeedsAnalysis {
			actionHints = append(actionHints, fmt.Sprintf("⚠️ Account '%s' needs analysis: account_analyze(account_id='%s')", acc.ID, acc.ID))
		}
		if !acc.Connected {
			actionHints = append(actionHints, fmt.Sprintf("💡 Account '%s' is not connected: account_connect(id='%s')", acc.ID, acc.ID))
		}
	}

	response["instructions"] = instructions
	if len(actionHints) > 0 {
		response["action_hints"] = actionHints
	}

	result, err := json.Marshal(response)
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

	// Generate secret for HMAC-based safe_id
	secret, err := generateAccountSecret()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to generate account secret: %v", err)), nil
	}

	account := types.AccountConfig{
		ID:       id,
		Name:     name,
		Host:     host,
		Port:     port,
		TLS:      tls,
		Username: username,
		Password: password,
		Secret:   secret,
	}

	if err := r.configMgr.AddAccount(account); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add account: %v", err)), nil
	}

	// Save config to file
	if err := r.configMgr.Save(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("account added but failed to save config: %v", err)), nil
	}

	response := map[string]interface{}{
		"status":     "added",
		"account_id": id,
		"message":    fmt.Sprintf("Account '%s' added successfully", id),
		"next_steps": []string{
			fmt.Sprintf("Connect to the account: account_connect(id='%s')", id),
			fmt.Sprintf("After connecting, analyze the account to build a profile: account_analyze(account_id='%s')", id),
		},
		"hint": "After connecting and analyzing, you'll have a profile with sender statistics, importance patterns, and suggestions for classification.",
	}

	result, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(result)), nil
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

	// Detect context switch
	if r.currentAccountID != "" && r.currentAccountID != id {
		response["context_switch"] = map[string]interface{}{
			"previous_account": r.currentAccountID,
			"current_account":  id,
			"warning":          fmt.Sprintf("⚠️ CONTEXT SWITCH: You switched from '%s' to '%s'. Memory context has changed. All subsequent operations will use '%s' memory.", r.currentAccountID, id, id),
		}
	}
	r.currentAccountID = id

	// Load and include memory summary
	if r.memoryMgr != nil {
		mem, err := r.memoryMgr.GetOrLoad(id)
		if err == nil {
			// Start session and clear dedup cache
			r.memoryMgr.StartSession(id)
			r.memoryMgr.ClearSeenCache()

			summary := mem.GetSummary()
			response["memory"] = summary

			// Check if profile needs to be created/refreshed
			if mem.Profile.LastAnalyzed.IsZero() {
				response["action_required"] = map[string]interface{}{
					"action":      "account_analyze",
					"reason":      "Account has no profile yet. Run analysis to build sender statistics and importance patterns.",
					"command":     fmt.Sprintf("account_analyze(account_id='%s')", id),
					"description": "This will scan recent emails, detect newsletters/notifications, calculate response rates, and suggest important senders.",
				}
			}

			response["memory_hint"] = "Account memory loaded. Use memory_* tools to save important information (contacts, preferences, sender classifications). When user mentions something is important or should be ignored, remember to save it."
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
