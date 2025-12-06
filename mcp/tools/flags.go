package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *Registry) registerFlagTools() {
	// message_set_flags - Set flags on a message
	r.addTool(
		mcp.NewTool("message_set_flags",
			mcp.WithDescription("Set flags on a message (replaces existing flags)"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
			mcp.WithArray("flags",
				mcp.Required(),
				mcp.Description("List of flags to set (e.g., \\Seen, \\Flagged, \\Answered)"),
			),
		),
		r.handleMessageSetFlags,
	)

	// message_add_flags - Add flags to a message
	r.addTool(
		mcp.NewTool("message_add_flags",
			mcp.WithDescription("Add flags to a message"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
			mcp.WithArray("flags",
				mcp.Required(),
				mcp.Description("List of flags to add"),
			),
		),
		r.handleMessageAddFlags,
	)

	// message_remove_flags - Remove flags from a message
	r.addTool(
		mcp.NewTool("message_remove_flags",
			mcp.WithDescription("Remove flags from a message"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
			mcp.WithArray("flags",
				mcp.Required(),
				mcp.Description("List of flags to remove"),
			),
		),
		r.handleMessageRemoveFlags,
	)

	// message_mark_read - Mark a message as read
	r.addTool(
		mcp.NewTool("message_mark_read",
			mcp.WithDescription("Mark a message as read"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
		),
		r.handleMessageMarkRead,
	)

	// message_mark_unread - Mark a message as unread
	r.addTool(
		mcp.NewTool("message_mark_unread",
			mcp.WithDescription("Mark a message as unread"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
		),
		r.handleMessageMarkUnread,
	)

	// message_flag - Flag/star a message
	r.addTool(
		mcp.NewTool("message_flag",
			mcp.WithDescription("Flag/star a message (mark as important)"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
		),
		r.handleMessageFlag,
	)

	// message_unflag - Unflag/unstar a message
	r.addTool(
		mcp.NewTool("message_unflag",
			mcp.WithDescription("Unflag/unstar a message"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
		),
		r.handleMessageUnflag,
	)

	// message_get_flags - Get flags of a message
	r.addTool(
		mcp.NewTool("message_get_flags",
			mcp.WithDescription("Get the current flags of a message"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
		),
		r.handleMessageGetFlags,
	)
}

func (r *Registry) handleMessageSetFlags(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)
	flags := getStringArray(request, "flags")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.SetFlags(folder, uid, flags); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to set flags: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Flags set on message UID %d", uid)), nil
}

func (r *Registry) handleMessageAddFlags(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)
	flags := getStringArray(request, "flags")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.AddFlags(folder, uid, flags); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add flags: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Flags added to message UID %d", uid)), nil
}

func (r *Registry) handleMessageRemoveFlags(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)
	flags := getStringArray(request, "flags")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.RemoveFlags(folder, uid, flags); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove flags: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Flags removed from message UID %d", uid)), nil
}

func (r *Registry) handleMessageMarkRead(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.MarkRead(folder, uid); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to mark as read: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message UID %d marked as read", uid)), nil
}

func (r *Registry) handleMessageMarkUnread(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.MarkUnread(folder, uid); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to mark as unread: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message UID %d marked as unread", uid)), nil
}

func (r *Registry) handleMessageFlag(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.Flag(folder, uid); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to flag message: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message UID %d flagged", uid)), nil
}

func (r *Registry) handleMessageUnflag(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.Unflag(folder, uid); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to unflag message: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message UID %d unflagged", uid)), nil
}

func (r *Registry) handleMessageGetFlags(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	flags, err := client.GetFlags(folder, uid)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get flags: %v", err)), nil
	}

	result, err := json.Marshal(map[string]interface{}{
		"uid":   uid,
		"flags": flags,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}
