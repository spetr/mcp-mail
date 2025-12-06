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
			mcp.WithDescription("Update user preferences for this account."),
			requiredString("account_id", "Account ID"),
			optionalString("language", "Preferred language (e.g., 'cs', 'en')"),
			optionalString("summary_style", "Summary style: 'brief', 'detailed', 'bullet_points'"),
			optionalString("timezone", "Timezone for dates"),
			optionalString("default_folder", "Default folder to check"),
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

	prefs := memory.Preferences{
		Language:      language,
		SummaryStyle:  summaryStyle,
		Timezone:      timezone,
		DefaultFolder: defaultFolder,
	}

	if err := r.memoryMgr.UpdatePreferences(accountID, prefs); err != nil {
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
