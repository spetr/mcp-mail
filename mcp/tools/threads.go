package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spetr/mcp-mail/types"
)

// ThreadMessageWithSafeID extends ThreadMessage with safe_id for destructive operations
type ThreadMessageWithSafeID struct {
	types.ThreadMessage
	SafeID string `json:"safe_id"`
}

// ThreadWithSafeID extends Thread with safe_ids for each message
type ThreadWithSafeID struct {
	ThreadID        string                    `json:"thread_id"`
	Subject         string                    `json:"subject"`
	Messages        []ThreadMessageWithSafeID `json:"messages"`
	TotalCount      int                       `json:"total_count"`
	UnreadCount     int                       `json:"unread_count"`
	Participants    []types.Address           `json:"participants"`
	LastMessageDate string                    `json:"last_message_date"`
}

// addSafeIDsToThread converts a Thread to ThreadWithSafeID
func addSafeIDsToThread(secret string, thread *types.Thread) *ThreadWithSafeID {
	result := &ThreadWithSafeID{
		ThreadID:        thread.ThreadID,
		Subject:         thread.Subject,
		TotalCount:      thread.TotalCount,
		UnreadCount:     thread.UnreadCount,
		Participants:    thread.Participants,
		LastMessageDate: thread.LastMessageDate,
		Messages:        make([]ThreadMessageWithSafeID, len(thread.Messages)),
	}

	for i, msg := range thread.Messages {
		fromAddr := ""
		if len(msg.From) > 0 {
			fromAddr = msg.From[0].Address
		}

		result.Messages[i] = ThreadMessageWithSafeID{
			ThreadMessage: msg,
			SafeID:        generateSafeID(secret, msg.UID, fromAddr, msg.Subject),
		}
	}

	return result
}

func (r *Registry) registerThreadTools() {
	// thread_list - List threads in a folder
	r.addTool(
		mcp.NewTool("thread_list",
			mcp.WithDescription("List email conversation threads in a folder. Groups related messages together based on Message-ID, In-Reply-To, and References headers. Returns thread summaries with participant info, message counts, and last message date."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name (e.g., INBOX)"),
			optionalNumber("limit", "Maximum number of threads to return (default: 20)"),
			optionalNumber("offset", "Number of threads to skip for pagination (default: 0)"),
		),
		r.handleThreadList,
	)

	// thread_get - Get a complete thread
	r.addTool(
		mcp.NewTool("thread_get",
			mcp.WithDescription("Get all messages in a conversation thread starting from a specific message UID. Returns the full thread with all related messages in chronological order, participants, and unread count."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID to get thread for"),
		),
		r.handleThreadGet,
	)

	// thread_get_by_id - Get thread by Message-ID
	r.addTool(
		mcp.NewTool("thread_get_by_id",
			mcp.WithDescription("Get a conversation thread by Message-ID header. Useful when you have a reference to a message but not its UID."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredString("message_id", "Message-ID header value (e.g., '<abc123@example.com>')"),
		),
		r.handleThreadGetByID,
	)
}

func (r *Registry) handleThreadList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	limit := getNumber(request, "limit", 20)
	offset := getNumber(request, "offset", 0)

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	threads, err := client.ListThreads(folder, limit, offset)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list threads: %v", err)), nil
	}

	result, err := json.Marshal(threads)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleThreadGet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)

	// Get account secret for HMAC-based safe_id
	secret, err := r.getAccountSecret(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get account secret: %v", err)), nil
	}

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	thread, err := client.GetThread(folder, uid)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get thread: %v", err)), nil
	}

	// Add safe_id to each message in the thread
	threadWithSafeIDs := addSafeIDsToThread(secret, thread)

	result, err := json.Marshal(threadWithSafeIDs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleThreadGetByID(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	messageID := request.GetString("message_id", "")

	// Get account secret for HMAC-based safe_id
	secret, err := r.getAccountSecret(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get account secret: %v", err)), nil
	}

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	thread, err := client.GetThreadByMessageID(folder, messageID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get thread: %v", err)), nil
	}

	// Add safe_id to each message in the thread
	threadWithSafeIDs := addSafeIDsToThread(secret, thread)

	result, err := json.Marshal(threadWithSafeIDs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}
