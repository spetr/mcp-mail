package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *Registry) registerMessageTools() {
	// message_list - List messages in a folder
	r.addTool(
		mcp.NewTool("message_list",
			mcp.WithDescription("List messages in a folder with pagination"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name (e.g., INBOX)"),
			optionalNumber("limit", "Maximum number of messages to return (default: 50)"),
			optionalNumber("offset", "Number of messages to skip (default: 0)"),
		),
		r.handleMessageList,
	)

	// message_get - Get a complete message
	r.addTool(
		mcp.NewTool("message_get",
			mcp.WithDescription("Get a complete message including body and attachments info"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
		),
		r.handleMessageGet,
	)

	// message_get_headers - Get message headers only
	r.addTool(
		mcp.NewTool("message_get_headers",
			mcp.WithDescription("Get message headers without body (faster)"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
		),
		r.handleMessageGetHeaders,
	)

	// message_delete - Delete a message (moves to trash)
	r.addTool(
		mcp.NewTool("message_delete",
			mcp.WithDescription("Delete a message by moving it to trash. Messages are NOT permanently deleted."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
		),
		r.handleMessageDelete,
	)

	// message_move - Move a message to another folder
	r.addTool(
		mcp.NewTool("message_move",
			mcp.WithDescription("Move a message to another folder"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Source folder name"),
			requiredNumber("uid", "Message UID"),
			requiredString("target_folder", "Destination folder name"),
		),
		r.handleMessageMove,
	)

	// message_copy - Copy a message to another folder
	r.addTool(
		mcp.NewTool("message_copy",
			mcp.WithDescription("Copy a message to another folder"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Source folder name"),
			requiredNumber("uid", "Message UID"),
			requiredString("target_folder", "Destination folder name"),
		),
		r.handleMessageCopy,
	)
}

func (r *Registry) handleMessageList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	limit := getNumber(request, "limit", 50)
	offset := getNumber(request, "offset", 0)

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	messages, err := client.ListMessages(folder, limit, offset)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list messages: %v", err)), nil
	}

	result, err := json.Marshal(messages)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleMessageGet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	message, err := client.GetMessage(folder, uid)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get message: %v", err)), nil
	}

	result, err := json.Marshal(message)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleMessageGetHeaders(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	headers, err := client.GetMessageHeaders(folder, uid)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get message headers: %v", err)), nil
	}

	result, err := json.Marshal(headers)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleMessageDelete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	// Get trash folder - either from config or auto-detect
	trashFolder := r.protectMgr.GetTrashFolder()
	if trashFolder == "" {
		trashFolder = client.GetTrashFolder()
	}

	if trashFolder == "" {
		return mcp.NewToolResultError("cannot delete: trash folder not found"), nil
	}

	if folder == trashFolder {
		return mcp.NewToolResultError("cannot delete messages from trash folder"), nil
	}

	// Move to trash
	if err := client.MoveMessage(folder, uid, trashFolder); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to move message to trash: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message UID %d moved to trash (%s)", uid, trashFolder)), nil
}

func (r *Registry) handleMessageMove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)
	targetFolder := request.GetString("target_folder", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.MoveMessage(folder, uid, targetFolder); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to move message: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message UID %d moved to '%s'", uid, targetFolder)), nil
}

func (r *Registry) handleMessageCopy(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)
	targetFolder := request.GetString("target_folder", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.CopyMessage(folder, uid, targetFolder); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to copy message: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message UID %d copied to '%s'", uid, targetFolder)), nil
}
