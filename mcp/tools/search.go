package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spetr/mcp-mail/types"
	"github.com/mark3labs/mcp-go/mcp"
)

func (r *Registry) registerSearchTools() {
	// search - Full IMAP search
	r.addTool(
		mcp.NewTool("search",
			mcp.WithDescription("Search for messages using IMAP search criteria"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder to search in"),
			optionalString("from", "Search in From header"),
			optionalString("to", "Search in To header"),
			optionalString("subject", "Search in Subject header"),
			optionalString("body", "Search in message body"),
			optionalString("text", "Search in entire message (headers + body)"),
			optionalString("since", "Messages since date (RFC3339 format, e.g., 2024-01-01)"),
			optionalString("before", "Messages before date (RFC3339 format)"),
			optionalBool("seen", "Filter by seen/unseen status"),
			optionalBool("flagged", "Filter by flagged status"),
			optionalNumber("larger", "Messages larger than N bytes"),
			optionalNumber("smaller", "Messages smaller than N bytes"),
		),
		r.handleSearch,
	)

	// search_unread - Quick search for unread messages
	r.addTool(
		mcp.NewTool("search_unread",
			mcp.WithDescription("Search for unread messages in a folder"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder to search in"),
		),
		r.handleSearchUnread,
	)

	// search_flagged - Quick search for flagged messages
	r.addTool(
		mcp.NewTool("search_flagged",
			mcp.WithDescription("Search for flagged/starred messages in a folder"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder to search in"),
		),
		r.handleSearchFlagged,
	)

	// search_from - Quick search by sender
	r.addTool(
		mcp.NewTool("search_from",
			mcp.WithDescription("Search for messages from a specific sender"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder to search in"),
			requiredString("from", "Sender email or name to search for"),
		),
		r.handleSearchFrom,
	)

	// search_subject - Quick search by subject
	r.addTool(
		mcp.NewTool("search_subject",
			mcp.WithDescription("Search for messages with specific subject text"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder to search in"),
			requiredString("subject", "Subject text to search for"),
		),
		r.handleSearchSubject,
	)

	// search_today - Quick search for today's messages
	r.addTool(
		mcp.NewTool("search_today",
			mcp.WithDescription("Search for messages received today"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder to search in"),
		),
		r.handleSearchToday,
	)

	// search_date_range - Search by date range
	r.addTool(
		mcp.NewTool("search_date_range",
			mcp.WithDescription("Search for messages within a date range"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder to search in"),
			optionalString("since", "Start date (RFC3339 format, e.g., 2024-01-01)"),
			optionalString("before", "End date (RFC3339 format)"),
		),
		r.handleSearchDateRange,
	)
}

func (r *Registry) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")

	// Build search criteria from arguments
	criteria := &types.SearchCriteria{}

	if from := request.GetString("from", ""); from != "" {
		criteria.From = from
	}
	if to := request.GetString("to", ""); to != "" {
		criteria.To = to
	}
	if subject := request.GetString("subject", ""); subject != "" {
		criteria.Subject = subject
	}
	if body := request.GetString("body", ""); body != "" {
		criteria.Body = body
	}
	if text := request.GetString("text", ""); text != "" {
		criteria.Text = text
	}

	// Parse dates
	if sinceStr := request.GetString("since", ""); sinceStr != "" {
		t, err := parseDate(sinceStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid since date: %v", err)), nil
		}
		criteria.Since = &t
	}
	if beforeStr := request.GetString("before", ""); beforeStr != "" {
		t, err := parseDate(beforeStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid before date: %v", err)), nil
		}
		criteria.Before = &t
	}

	// Boolean filters - check if they were provided
	args := request.GetArguments()
	if _, ok := args["seen"]; ok {
		seen := request.GetBool("seen", false)
		criteria.Seen = &seen
	}
	if _, ok := args["flagged"]; ok {
		flagged := request.GetBool("flagged", false)
		criteria.Flagged = &flagged
	}

	// Size filters
	if larger := getNumber(request, "larger", 0); larger > 0 {
		criteria.Larger = uint32(larger)
	}
	if smaller := getNumber(request, "smaller", 0); smaller > 0 {
		criteria.Smaller = uint32(smaller)
	}

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	result, err := client.Search(folder, criteria)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (r *Registry) handleSearchUnread(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	result, err := client.SearchUnread(folder)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (r *Registry) handleSearchFlagged(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	result, err := client.SearchFlagged(folder)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (r *Registry) handleSearchFrom(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	from := request.GetString("from", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	result, err := client.SearchFrom(folder, from)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (r *Registry) handleSearchSubject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	subject := request.GetString("subject", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	result, err := client.SearchSubject(folder, subject)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (r *Registry) handleSearchToday(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	result, err := client.SearchToday(folder)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (r *Registry) handleSearchDateRange(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	sinceStr := request.GetString("since", "")
	beforeStr := request.GetString("before", "")

	var since, before *time.Time

	if sinceStr != "" {
		t, err := parseDate(sinceStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid since date: %v", err)), nil
		}
		since = &t
	}

	if beforeStr != "" {
		t, err := parseDate(beforeStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid before date: %v", err)), nil
		}
		before = &t
	}

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	result, err := client.SearchDateRange(folder, since, before)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// parseDate parses a date string in various formats
func parseDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
		"02.01.2006",
		"01/02/2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse date: %s", s)
}
