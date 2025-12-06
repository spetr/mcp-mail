package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *Registry) registerInstructionTools() {
	// get_instructions - Get initial context and instructions
	r.addTool(
		mcp.NewTool("get_instructions",
			mcp.WithDescription("Get initial instructions and context for the email assistant. Call this at the start of a conversation to understand available accounts, their status, and how to use the email tools effectively."),
		),
		r.handleGetInstructions,
	)
}

func (r *Registry) handleGetInstructions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accounts := r.configMgr.GetAccounts()
	statuses := r.imapMgr.GetStatus(accounts)

	// Build account summaries with memory state
	type AccountSummary struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Connected     bool   `json:"connected"`
		HasProfile    bool   `json:"has_profile"`
		NeedsAnalysis bool   `json:"needs_analysis"`
		SenderCount   int    `json:"sender_count,omitempty"`
		ContactCount  int    `json:"contact_count,omitempty"`
	}

	accountSummaries := make([]AccountSummary, 0, len(statuses))
	for _, status := range statuses {
		summary := AccountSummary{
			ID:            status.ID,
			Name:          status.Name,
			Connected:     status.Connected,
			NeedsAnalysis: true,
		}

		if r.memoryMgr != nil {
			if mem, err := r.memoryMgr.GetOrLoad(status.ID); err == nil {
				summary.HasProfile = !mem.Profile.LastAnalyzed.IsZero()
				if summary.HasProfile {
					summary.NeedsAnalysis = false
				}
				summary.SenderCount = len(mem.Profile.SenderProfiles)
				summary.ContactCount = len(mem.ImportantContacts)
			}
		}

		accountSummaries = append(accountSummaries, summary)
	}

	// Build comprehensive instructions
	instructions := []string{
		"📧 EMAIL ASSISTANT - QUICK START GUIDE",
		"",
		"🔧 BASIC WORKFLOW:",
		"1. List accounts: account_list",
		"2. Connect: account_connect(id='account_id')",
		"3. If needs_analysis=true: account_analyze(account_id='...')",
		"4. List folders: folder_list(account_id='...')",
		"5. List messages: message_list(account_id='...', folder='INBOX')",
		"6. Read message: message_get(account_id='...', folder='...', uid=123)",
		"",
		"💾 MEMORY SYSTEM - VERY IMPORTANT:",
		"- When user says an email/sender is IMPORTANT → memory_set_sender(..., importance='high')",
		"- When user says to IGNORE a sender → memory_set_sender(..., importance='ignore')",
		"- When you learn a contact is important → memory_add_contact(...)",
		"- To add notes about preferences → memory_add_note(...)",
		"- To mark unwanted senders → memory_add_unwanted(sender='*@spam.com')",
		"- ALWAYS save user preferences to memory!",
		"",
		"📊 AVAILABLE MEMORY TOOLS:",
		"- memory_get(account_id) - Get memory summary",
		"- memory_set_sender(account_id, email, importance, type) - Classify sender",
		"- memory_add_contact(account_id, email, name, role, priority) - Add important contact",
		"- memory_add_note(account_id, content, category) - Save notes/preferences",
		"- memory_add_unwanted(account_id, sender/subject/keyword) - Mark as spam",
		"- memory_add_rule(account_id, pattern, action) - Add filtering rule",
		"",
		"📧 MESSAGE OPERATIONS:",
		"- message_list - List messages with pagination",
		"- message_get - Read full message with body",
		"- message_delete - Move to trash",
		"- message_move - Move to folder",
		"- message_search - Search messages",
		"- thread_get - Get full conversation thread",
		"",
		"✉️ SENDING EMAIL:",
		"- send_email - Compose and send new email",
		"- send_reply - Reply to a message",
		"- send_reply_all - Reply to all recipients",
		"- send_forward - Forward a message",
	}

	// Build suggested actions based on account state
	var suggestedActions []string

	if len(accountSummaries) == 0 {
		suggestedActions = append(suggestedActions, "⚠️ No accounts configured. Use account_add to add an IMAP account.")
	} else {
		for _, acc := range accountSummaries {
			if !acc.Connected {
				suggestedActions = append(suggestedActions, fmt.Sprintf("💡 Connect to '%s': account_connect(id='%s')", acc.ID, acc.ID))
			} else if acc.NeedsAnalysis {
				suggestedActions = append(suggestedActions, fmt.Sprintf("⚠️ Account '%s' needs analysis: account_analyze(account_id='%s')", acc.ID, acc.ID))
			}
		}
	}

	if len(suggestedActions) == 0 && len(accountSummaries) > 0 {
		suggestedActions = append(suggestedActions, "✅ All accounts ready! Ask the user what they'd like to do with their email.")
	}

	response := map[string]interface{}{
		"accounts":          accountSummaries,
		"account_count":     len(accountSummaries),
		"instructions":      instructions,
		"suggested_actions": suggestedActions,
	}

	// Add current account if set
	if r.currentAccountID != "" {
		response["current_account"] = r.currentAccountID
	}

	result, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}
