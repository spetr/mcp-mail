package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spetr/mcp-mail/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

func (r *Registry) registerMemoryTools() {
	// memory_get - Get memory for an account
	r.addTool(
		mcp.NewTool("memory_get",
			mcp.WithDescription("Get the stored memory/context for an email account. Contains important contacts, topics, spam rules, and notes learned from previous interactions."),
			requiredString("account_id", "Account ID"),
		),
		r.handleMemoryGet,
	)

	// memory_get_full - Get full memory details
	r.addTool(
		mcp.NewTool("memory_get_full",
			mcp.WithDescription("Get the complete memory for an account with all details (contacts, rules, notes, statistics)."),
			requiredString("account_id", "Account ID"),
		),
		r.handleMemoryGetFull,
	)

	// memory_add_contact - Add an important contact
	r.addTool(
		mcp.NewTool("memory_add_contact",
			mcp.WithDescription("Add or update an important contact. Use this when you learn someone is important to the user."),
			requiredString("account_id", "Account ID"),
			requiredString("email", "Contact email address"),
			optionalString("name", "Contact name"),
			optionalString("role", "Contact role (e.g., 'Boss', 'Client', 'Family')"),
			optionalString("priority", "Priority level: 'high', 'normal', 'low'"),
			optionalString("notes", "Additional notes about this contact"),
		),
		r.handleMemoryAddContact,
	)

	// memory_remove_contact - Remove a contact
	r.addTool(
		mcp.NewTool("memory_remove_contact",
			mcp.WithDescription("Remove a contact from important contacts list."),
			requiredString("account_id", "Account ID"),
			requiredString("email", "Contact email to remove"),
		),
		r.handleMemoryRemoveContact,
	)

	// memory_add_unwanted - Add unwanted sender/subject
	r.addTool(
		mcp.NewTool("memory_add_unwanted",
			mcp.WithDescription("Add a sender or subject pattern to the unwanted/spam list. Use wildcards: *@domain.com, newsletter@*"),
			requiredString("account_id", "Account ID"),
			optionalString("sender", "Sender email or pattern to mark as unwanted"),
			optionalString("subject", "Subject pattern to mark as unwanted"),
			optionalString("keyword", "Body keyword to mark as unwanted"),
		),
		r.handleMemoryAddUnwanted,
	)

	// memory_remove_unwanted - Remove from unwanted list
	r.addTool(
		mcp.NewTool("memory_remove_unwanted",
			mcp.WithDescription("Remove a sender or subject from the unwanted list."),
			requiredString("account_id", "Account ID"),
			optionalString("sender", "Sender to remove from unwanted"),
			optionalString("subject", "Subject to remove from unwanted"),
		),
		r.handleMemoryRemoveUnwanted,
	)

	// memory_add_note - Add a note
	r.addTool(
		mcp.NewTool("memory_add_note",
			mcp.WithDescription("Add a note about this account. Use this to remember user preferences, rules, or important information."),
			requiredString("account_id", "Account ID"),
			requiredString("content", "Note content"),
			optionalString("category", "Note category: 'rule', 'reminder', 'preference', 'info'"),
		),
		r.handleMemoryAddNote,
	)

	// memory_add_topic - Add a priority topic
	r.addTool(
		mcp.NewTool("memory_add_topic",
			mcp.WithDescription("Add a priority topic to watch for in emails."),
			requiredString("account_id", "Account ID"),
			requiredString("topic", "Topic keyword or phrase"),
		),
		r.handleMemoryAddTopic,
	)

	// memory_set_folder_purpose - Set folder purpose
	r.addTool(
		mcp.NewTool("memory_set_folder_purpose",
			mcp.WithDescription("Record what a folder is used for."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredString("purpose", "Description of folder purpose"),
		),
		r.handleMemorySetFolderPurpose,
	)

	// memory_update_preferences - Update preferences
	r.addTool(
		mcp.NewTool("memory_update_preferences",
			mcp.WithDescription("Update user preferences for this account including analysis settings."),
			requiredString("account_id", "Account ID"),
			optionalString("language", "Preferred language (e.g., 'cs', 'en')"),
			optionalString("summary_style", "Summary style: 'brief', 'detailed', 'bullet_points'"),
			optionalString("timezone", "Timezone for dates"),
			optionalString("default_folder", "Default folder to check"),
			optionalNumber("analysis_depth", "Max messages to analyze (default 1000)"),
			optionalNumber("analysis_period", "Days to look back for analysis (default 180)"),
			optionalBool("hints_enabled", "Enable/disable hints in responses (default true)"),
		),
		r.handleMemoryUpdatePreferences,
	)

	// memory_search - Search across all memory data
	r.addTool(
		mcp.NewTool("memory_search",
			mcp.WithDescription("Search across all memory data (contacts, notes, topics, unwanted rules, folder purposes). Returns all matches."),
			requiredString("account_id", "Account ID"),
			requiredString("query", "Search query - searches in emails, names, roles, notes, topics, etc."),
		),
		r.handleMemorySearch,
	)

	// memory_add_observation - Add an observation/fact about a contact
	r.addTool(
		mcp.NewTool("memory_add_observation",
			mcp.WithDescription("Add an atomic observation (fact) about a contact. Use this to record specific facts learned about a person, e.g., 'Prefers communication in Czech', 'Usually responds within 24h', 'Works at Company X'."),
			requiredString("account_id", "Account ID"),
			requiredString("email", "Contact email address"),
			requiredString("observation", "The fact/observation to record"),
		),
		r.handleMemoryAddObservation,
	)

	// memory_remove_observation - Remove an observation from a contact
	r.addTool(
		mcp.NewTool("memory_remove_observation",
			mcp.WithDescription("Remove a specific observation from a contact."),
			requiredString("account_id", "Account ID"),
			requiredString("email", "Contact email address"),
			requiredString("observation", "The observation to remove (must match exactly)"),
		),
		r.handleMemoryRemoveObservation,
	)

	// memory_get_contact - Get detailed info about a specific contact
	r.addTool(
		mcp.NewTool("memory_get_contact",
			mcp.WithDescription("Get all stored information about a specific contact including observations."),
			requiredString("account_id", "Account ID"),
			requiredString("email", "Contact email address"),
		),
		r.handleMemoryGetContact,
	)

	// =========================================================================
	// Profile Tools
	// =========================================================================

	// memory_update_profile - Update account profile
	r.addTool(
		mcp.NewTool("memory_update_profile",
			mcp.WithDescription("Update the account profile. Use this to describe the account type, purpose, and activity patterns as you learn about the user's email habits."),
			requiredString("account_id", "Account ID"),
			optionalString("type", "Account type: 'personal', 'work', 'mixed', 'newsletters'"),
			optionalString("description", "AI-generated description of the account and its purpose"),
			optionalString("activity_summary", "Summary of user's email activity patterns (e.g., 'Aktivní večer, odpovídá do 24h')"),
		),
		r.handleMemoryUpdateProfile,
	)

	// memory_set_sender - Set sender profile/classification
	r.addTool(
		mcp.NewTool("memory_set_sender",
			mcp.WithDescription("Classify a sender. Use this to mark senders as important, newsletters, or to ignore. The system tracks statistics (response rate, frequency) automatically."),
			requiredString("account_id", "Account ID"),
			requiredString("email", "Sender email address"),
			optionalString("name", "Sender name"),
			optionalString("type", "Sender type: 'person', 'newsletter', 'notification', 'service'"),
			optionalString("importance", "Importance level: 'high', 'normal', 'low', 'ignore'"),
			optionalString("relationship", "Relationship: 'colleague', 'family', 'friend', 'vendor', 'support'"),
			optionalString("notes", "Notes about this sender"),
		),
		r.handleMemorySetSender,
	)

	// memory_get_sender - Get sender profile with stats
	r.addTool(
		mcp.NewTool("memory_get_sender",
			mcp.WithDescription("Get profile and statistics for a specific sender. Shows classification, response rate, and email frequency."),
			requiredString("account_id", "Account ID"),
			requiredString("email", "Sender email address"),
		),
		r.handleMemoryGetSender,
	)

	// memory_remove_sender - Remove sender profile
	r.addTool(
		mcp.NewTool("memory_remove_sender",
			mcp.WithDescription("Remove a sender profile (classification and stats)."),
			requiredString("account_id", "Account ID"),
			requiredString("email", "Sender email address"),
		),
		r.handleMemoryRemoveSender,
	)

	// memory_add_rule - Add importance rule
	r.addTool(
		mcp.NewTool("memory_add_rule",
			mcp.WithDescription("Add a rule for determining email importance. Rules are patterns that match emails and assign actions."),
			requiredString("account_id", "Account ID"),
			requiredString("pattern", "Match pattern: 'from:*@company.com', 'subject:urgent', 'to:team@*'"),
			requiredString("action", "Action: 'high_priority', 'needs_response', 'normal', 'low_priority', 'ignore'"),
			optionalString("reason", "Explanation why this rule exists"),
			optionalNumber("confidence", "Confidence level 0-100 (default 80)"),
		),
		r.handleMemoryAddRule,
	)

	// memory_remove_rule - Remove importance rule
	r.addTool(
		mcp.NewTool("memory_remove_rule",
			mcp.WithDescription("Remove an importance rule by its pattern."),
			requiredString("account_id", "Account ID"),
			requiredString("pattern", "Pattern to remove"),
		),
		r.handleMemoryRemoveRule,
	)

	// memory_prune - Clean up stale memory data
	r.addTool(
		mcp.NewTool("memory_prune",
			mcp.WithDescription("Clean up stale/unimportant data from memory. Removes sender profiles that have no classification, low activity (<3 emails), and haven't been seen in 90+ days. Use this when memory gets too large."),
			requiredString("account_id", "Account ID"),
		),
		r.handleMemoryPrune,
	)

	// memory_stats - Get memory statistics
	r.addTool(
		mcp.NewTool("memory_stats",
			mcp.WithDescription("Get statistics about memory size (sender count, notes, rules, etc). Use this to check if memory needs pruning."),
			requiredString("account_id", "Account ID"),
		),
		r.handleMemoryStats,
	)
}

func (r *Registry) handleMemoryGet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")

	mem, err := r.memoryMgr.GetOrLoad(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory: %v", err)), nil
	}

	summary := mem.GetSummary()
	result, _ := json.Marshal(summary)
	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleMemoryGetFull(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")

	mem, err := r.memoryMgr.GetOrLoad(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory: %v", err)), nil
	}

	result, _ := json.Marshal(mem)
	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleMemoryAddContact(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	email := request.GetString("email", "")
	name := request.GetString("name", "")
	role := request.GetString("role", "")
	priority := request.GetString("priority", "normal")
	notes := request.GetString("notes", "")

	contact := memory.Contact{
		Email:    email,
		Name:     name,
		Role:     role,
		Priority: priority,
		Notes:    notes,
		AddedAt:  time.Now(),
	}

	if err := r.memoryMgr.AddContact(accountID, contact); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add contact: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Contact '%s' added to important contacts for account '%s'", email, accountID)), nil
}

func (r *Registry) handleMemoryRemoveContact(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	email := request.GetString("email", "")

	if err := r.memoryMgr.RemoveContact(accountID, email); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove contact: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Contact '%s' removed from important contacts", email)), nil
}

func (r *Registry) handleMemoryAddUnwanted(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	sender := request.GetString("sender", "")
	subject := request.GetString("subject", "")
	keyword := request.GetString("keyword", "")

	var added []string

	if sender != "" {
		if err := r.memoryMgr.AddUnwantedSender(accountID, sender); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to add unwanted sender: %v", err)), nil
		}
		added = append(added, fmt.Sprintf("sender '%s'", sender))
	}

	if subject != "" {
		if err := r.memoryMgr.Update(accountID, func(mem *memory.AccountMemory) {
			for _, s := range mem.Unwanted.Subjects {
				if s == subject {
					return
				}
			}
			mem.Unwanted.Subjects = append(mem.Unwanted.Subjects, subject)
		}); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to add unwanted subject: %v", err)), nil
		}
		added = append(added, fmt.Sprintf("subject '%s'", subject))
	}

	if keyword != "" {
		if err := r.memoryMgr.Update(accountID, func(mem *memory.AccountMemory) {
			for _, k := range mem.Unwanted.BodyKeywords {
				if k == keyword {
					return
				}
			}
			mem.Unwanted.BodyKeywords = append(mem.Unwanted.BodyKeywords, keyword)
		}); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to add unwanted keyword: %v", err)), nil
		}
		added = append(added, fmt.Sprintf("keyword '%s'", keyword))
	}

	if len(added) == 0 {
		return mcp.NewToolResultError("no sender, subject, or keyword provided"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Added to unwanted list: %v", added)), nil
}

func (r *Registry) handleMemoryRemoveUnwanted(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	sender := request.GetString("sender", "")
	subject := request.GetString("subject", "")

	var removed []string

	if sender != "" {
		if err := r.memoryMgr.RemoveUnwantedSender(accountID, sender); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to remove unwanted sender: %v", err)), nil
		}
		removed = append(removed, fmt.Sprintf("sender '%s'", sender))
	}

	if subject != "" {
		if err := r.memoryMgr.Update(accountID, func(mem *memory.AccountMemory) {
			for i, s := range mem.Unwanted.Subjects {
				if s == subject {
					mem.Unwanted.Subjects = append(mem.Unwanted.Subjects[:i], mem.Unwanted.Subjects[i+1:]...)
					return
				}
			}
		}); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to remove unwanted subject: %v", err)), nil
		}
		removed = append(removed, fmt.Sprintf("subject '%s'", subject))
	}

	if len(removed) == 0 {
		return mcp.NewToolResultError("no sender or subject provided"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Removed from unwanted list: %v", removed)), nil
}

func (r *Registry) handleMemoryAddNote(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	content := request.GetString("content", "")
	category := request.GetString("category", "info")

	if err := r.memoryMgr.AddNote(accountID, content, category); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add note: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Note added to account '%s'", accountID)), nil
}

func (r *Registry) handleMemoryAddTopic(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	topic := request.GetString("topic", "")

	if err := r.memoryMgr.AddPriorityTopic(accountID, topic); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add topic: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Priority topic '%s' added", topic)), nil
}

func (r *Registry) handleMemorySetFolderPurpose(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	purpose := request.GetString("purpose", "")

	if err := r.memoryMgr.SetFolderPurpose(accountID, folder, purpose); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to set folder purpose: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Folder '%s' purpose set to: %s", folder, purpose)), nil
}

func (r *Registry) handleMemoryUpdatePreferences(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	language := request.GetString("language", "")
	summaryStyle := request.GetString("summary_style", "")
	timezone := request.GetString("timezone", "")
	defaultFolder := request.GetString("default_folder", "")
	analysisDepth := request.GetInt("analysis_depth", 0)
	analysisPeriod := request.GetInt("analysis_period", 0)

	// Check if hints_enabled was explicitly provided
	var hintsEnabled *bool
	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if _, exists := args["hints_enabled"]; exists {
			val := request.GetBool("hints_enabled", true)
			hintsEnabled = &val
		}
	}

	err := r.memoryMgr.Update(accountID, func(mem *memory.AccountMemory) {
		if language != "" {
			mem.Preferences.Language = language
		}
		if summaryStyle != "" {
			mem.Preferences.SummaryStyle = summaryStyle
		}
		if timezone != "" {
			mem.Preferences.Timezone = timezone
		}
		if defaultFolder != "" {
			mem.Preferences.DefaultFolder = defaultFolder
		}
		if analysisDepth > 0 {
			mem.Preferences.AnalysisDepth = analysisDepth
		}
		if analysisPeriod > 0 {
			mem.Preferences.AnalysisPeriod = analysisPeriod
		}
		if hintsEnabled != nil {
			mem.Preferences.HintsEnabled = hintsEnabled
		}
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update preferences: %v", err)), nil
	}

	return mcp.NewToolResultText("Preferences updated"), nil
}

func (r *Registry) handleMemorySearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	query := request.GetString("query", "")

	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	results, err := r.memoryMgr.Search(accountID, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search memory: %v", err)), nil
	}

	resultJSON, _ := json.Marshal(results)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (r *Registry) handleMemoryAddObservation(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	email := request.GetString("email", "")
	observation := request.GetString("observation", "")

	if observation == "" {
		return mcp.NewToolResultError("observation is required"), nil
	}

	if err := r.memoryMgr.AddObservation(accountID, email, observation); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add observation: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Observation added to contact '%s'", email)), nil
}

func (r *Registry) handleMemoryRemoveObservation(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	email := request.GetString("email", "")
	observation := request.GetString("observation", "")

	if err := r.memoryMgr.RemoveObservation(accountID, email, observation); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove observation: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Observation removed from contact '%s'", email)), nil
}

func (r *Registry) handleMemoryGetContact(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	email := request.GetString("email", "")

	contact, err := r.memoryMgr.GetContact(accountID, email)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get contact: %v", err)), nil
	}

	resultJSON, _ := json.Marshal(contact)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// ============================================================================
// Profile Tools
// ============================================================================

func (r *Registry) handleMemoryUpdateProfile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	profileType := request.GetString("type", "")
	description := request.GetString("description", "")
	activitySummary := request.GetString("activity_summary", "")

	if profileType == "" && description == "" && activitySummary == "" {
		return mcp.NewToolResultError("at least one of type, description, or activity_summary is required"), nil
	}

	if err := r.memoryMgr.UpdateProfile(accountID, profileType, description, activitySummary); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update profile: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Profile updated for account '%s'", accountID)), nil
}

func (r *Registry) handleMemorySetSender(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	email := request.GetString("email", "")
	name := request.GetString("name", "")
	senderType := request.GetString("type", "")
	importance := request.GetString("importance", "")
	relationship := request.GetString("relationship", "")
	notes := request.GetString("notes", "")

	if email == "" {
		return mcp.NewToolResultError("email is required"), nil
	}

	profile := &memory.SenderProfile{
		Email:        email,
		Name:         name,
		Type:         senderType,
		Importance:   importance,
		Relationship: relationship,
		Notes:        notes,
	}

	if err := r.memoryMgr.SetSenderProfile(accountID, profile); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to set sender profile: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Sender profile updated for '%s'", email)), nil
}

func (r *Registry) handleMemoryGetSender(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	email := request.GetString("email", "")

	if email == "" {
		return mcp.NewToolResultError("email is required"), nil
	}

	profile, err := r.memoryMgr.GetSenderProfile(accountID, email)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("sender not found: %v", err)), nil
	}

	resultJSON, _ := json.Marshal(profile)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (r *Registry) handleMemoryRemoveSender(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	email := request.GetString("email", "")

	if email == "" {
		return mcp.NewToolResultError("email is required"), nil
	}

	if err := r.memoryMgr.RemoveSenderProfile(accountID, email); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove sender: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Sender profile removed for '%s'", email)), nil
}

func (r *Registry) handleMemoryAddRule(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	pattern := request.GetString("pattern", "")
	action := request.GetString("action", "")
	reason := request.GetString("reason", "")
	confidence := request.GetInt("confidence", 80)

	if pattern == "" {
		return mcp.NewToolResultError("pattern is required (e.g., 'from:*@company.com', 'subject:urgent')"), nil
	}
	if action == "" {
		return mcp.NewToolResultError("action is required (high_priority, needs_response, normal, low_priority, ignore)"), nil
	}

	rule := memory.ImportanceRule{
		Pattern:     pattern,
		Action:      action,
		Reason:      reason,
		Confidence:  confidence,
		LearnedFrom: "manual",
	}

	if err := r.memoryMgr.AddImportanceRule(accountID, rule); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add rule: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Importance rule added: %s → %s", pattern, action)), nil
}

func (r *Registry) handleMemoryRemoveRule(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	pattern := request.GetString("pattern", "")

	if pattern == "" {
		return mcp.NewToolResultError("pattern is required"), nil
	}

	if err := r.memoryMgr.RemoveImportanceRule(accountID, pattern); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove rule: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Importance rule removed: %s", pattern)), nil
}

func (r *Registry) handleMemoryPrune(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")

	var pruneResult memory.PruneResult

	err := r.memoryMgr.Update(accountID, func(mem *memory.AccountMemory) {
		pruneResult = mem.Prune()
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to prune memory: %v", err)), nil
	}

	result, _ := json.Marshal(pruneResult)
	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleMemoryStats(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")

	mem, err := r.memoryMgr.GetOrLoad(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory: %v", err)), nil
	}

	stats := mem.GetMemoryStats()
	result, _ := json.Marshal(stats)
	return mcp.NewToolResultText(string(result)), nil
}
