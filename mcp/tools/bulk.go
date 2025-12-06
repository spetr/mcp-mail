package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spetr/mcp-mail/imap"
	"github.com/mark3labs/mcp-go/mcp"
)

func (r *Registry) registerBulkTools() {
	// messages_delete_bulk - Delete multiple messages (moves to trash)
	r.addTool(
		mcp.NewTool("messages_delete_bulk",
			mcp.WithDescription("Delete multiple messages by moving them to trash. Messages are NOT permanently deleted - they are moved to the trash folder. The user must manually empty trash to permanently delete."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			mcp.WithArray("uids",
				mcp.Required(),
				mcp.Description("List of message UIDs to delete"),
			),
		),
		r.handleMessagesDeleteBulk,
	)

	// messages_move_bulk - Move multiple messages
	r.addTool(
		mcp.NewTool("messages_move_bulk",
			mcp.WithDescription("Move multiple messages to another folder"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Source folder name"),
			mcp.WithArray("uids",
				mcp.Required(),
				mcp.Description("List of message UIDs to move"),
			),
			requiredString("target_folder", "Destination folder name"),
		),
		r.handleMessagesMovesBulk,
	)

	// messages_copy_bulk - Copy multiple messages
	r.addTool(
		mcp.NewTool("messages_copy_bulk",
			mcp.WithDescription("Copy multiple messages to another folder"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Source folder name"),
			mcp.WithArray("uids",
				mcp.Required(),
				mcp.Description("List of message UIDs to copy"),
			),
			requiredString("target_folder", "Destination folder name"),
		),
		r.handleMessagesCopyBulk,
	)

	// messages_flag_bulk - Set flags on multiple messages
	r.addTool(
		mcp.NewTool("messages_flag_bulk",
			mcp.WithDescription("Set flags on multiple messages"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			mcp.WithArray("uids",
				mcp.Required(),
				mcp.Description("List of message UIDs"),
			),
			mcp.WithArray("flags",
				mcp.Required(),
				mcp.Description("List of flags to set (e.g., \\Seen, \\Flagged)"),
			),
			optionalString("mode", "Flag operation mode: 'set' (replace), 'add', or 'remove' (default: add)"),
		),
		r.handleMessagesFlagBulk,
	)
}

func (r *Registry) handleMessagesDeleteBulk(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check read-only mode
	if err := r.protectMgr.CheckReadOnly(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")

	// Get UIDs (handles both int and string arrays)
	uids, err := getUIDs(request)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid UIDs: %v", err)), nil
	}

	if len(uids) == 0 {
		return mcp.NewToolResultError("no UIDs provided"), nil
	}

	// Check bulk operation limit
	if len(uids) > maxBulkOperations {
		return mcp.NewToolResultError(fmt.Sprintf("too many UIDs: %d (max %d)", len(uids), maxBulkOperations)), nil
	}

	return r.executeDeleteBulk(accountID, folder, uids)
}

func (r *Registry) handleMessagesMovesBulk(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check read-only mode
	if err := r.protectMgr.CheckReadOnly(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	targetFolder := request.GetString("target_folder", "")

	uids, err := getUIDs(request)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid UIDs: %v", err)), nil
	}

	if len(uids) == 0 {
		return mcp.NewToolResultError("no UIDs provided"), nil
	}

	// Check bulk operation limit
	if len(uids) > maxBulkOperations {
		return mcp.NewToolResultError(fmt.Sprintf("too many UIDs: %d (max %d)", len(uids), maxBulkOperations)), nil
	}

	return r.executeMoveBulk(accountID, folder, uids, targetFolder)
}

func (r *Registry) handleMessagesCopyBulk(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check read-only mode
	if err := r.protectMgr.CheckReadOnly(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	targetFolder := request.GetString("target_folder", "")

	uids, err := getUIDs(request)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid UIDs: %v", err)), nil
	}

	if len(uids) == 0 {
		return mcp.NewToolResultError("no UIDs provided"), nil
	}

	// Check bulk operation limit
	if len(uids) > maxBulkOperations {
		return mcp.NewToolResultError(fmt.Sprintf("too many UIDs: %d (max %d)", len(uids), maxBulkOperations)), nil
	}

	return r.executeCopyBulk(accountID, folder, uids, targetFolder)
}

func (r *Registry) handleMessagesFlagBulk(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check read-only mode
	if err := r.protectMgr.CheckReadOnly(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	flags := getStringArray(request, "flags")
	mode := request.GetString("mode", "add")

	uids, err := getUIDs(request)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid UIDs: %v", err)), nil
	}

	if len(uids) == 0 {
		return mcp.NewToolResultError("no UIDs provided"), nil
	}

	if len(flags) == 0 {
		return mcp.NewToolResultError("no flags provided"), nil
	}

	// Check bulk operation limit
	if len(uids) > maxBulkOperations {
		return mcp.NewToolResultError(fmt.Sprintf("too many UIDs: %d (max %d)", len(uids), maxBulkOperations)), nil
	}

	return r.executeFlagBulk(accountID, folder, uids, flags, mode)
}

// Execute functions that actually perform the operations
// All operations use chunking to avoid overwhelming the IMAP server

const defaultChunkSize = 50

func (r *Registry) executeDeleteBulk(accountID, folder string, uids []uint32) (*mcp.CallToolResult, error) {
	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	// Get trash folder - either from config or auto-detect
	trashFolder := r.protectMgr.GetTrashFolder()
	if trashFolder == "" {
		trashFolder = client.GetTrashFolder()
	}

	// Cannot delete if we can't find trash folder
	if trashFolder == "" {
		return mcp.NewToolResultError("cannot delete: trash folder not found. Configure trash_folder in security settings or ensure server has a trash folder."), nil
	}

	// Cannot delete from trash folder itself
	if folder == trashFolder {
		return mcp.NewToolResultError("cannot delete messages from trash folder. Empty the trash manually to permanently delete messages."), nil
	}

	// Move to trash (chunked)
	chunkedResult := client.MoveMessagesChunked(folder, uids, trashFolder, defaultChunkSize)

	response := map[string]interface{}{
		"success":      chunkedResult.Error == "",
		"action":       "moved_to_trash",
		"trash_folder": trashFolder,
		"moved_count":  chunkedResult.SuccessCount,
		"failed_count": chunkedResult.FailCount,
		"total":        chunkedResult.TotalItems,
		"chunks_used":  chunkedResult.ChunksUsed,
	}
	if chunkedResult.Error != "" {
		response["error"] = chunkedResult.Error
	}

	result, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) executeMoveBulk(accountID, folder string, uids []uint32, targetFolder string) (*mcp.CallToolResult, error) {
	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	chunkedResult := client.MoveMessagesChunked(folder, uids, targetFolder, defaultChunkSize)

	response := map[string]interface{}{
		"success":      chunkedResult.Error == "",
		"action":       "moved",
		"target":       targetFolder,
		"moved_count":  chunkedResult.SuccessCount,
		"failed_count": chunkedResult.FailCount,
		"total":        chunkedResult.TotalItems,
		"chunks_used":  chunkedResult.ChunksUsed,
	}
	if chunkedResult.Error != "" {
		response["error"] = chunkedResult.Error
	}

	result, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) executeCopyBulk(accountID, folder string, uids []uint32, targetFolder string) (*mcp.CallToolResult, error) {
	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	chunkedResult := client.CopyMessagesChunked(folder, uids, targetFolder, defaultChunkSize)

	response := map[string]interface{}{
		"success":      chunkedResult.Error == "",
		"action":       "copied",
		"target":       targetFolder,
		"copied_count": chunkedResult.SuccessCount,
		"failed_count": chunkedResult.FailCount,
		"total":        chunkedResult.TotalItems,
		"chunks_used":  chunkedResult.ChunksUsed,
	}
	if chunkedResult.Error != "" {
		response["error"] = chunkedResult.Error
	}

	result, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) executeFlagBulk(accountID, folder string, uids []uint32, flags []string, mode string) (*mcp.CallToolResult, error) {
	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	var chunkedResult *imap.ChunkedResult

	switch mode {
	case "set":
		chunkedResult = client.SetFlagsBulkChunked(folder, uids, flags, defaultChunkSize)
	case "remove":
		chunkedResult = client.RemoveFlagsBulkChunked(folder, uids, flags, defaultChunkSize)
	default: // "add"
		chunkedResult = client.AddFlagsBulkChunked(folder, uids, flags, defaultChunkSize)
	}

	response := map[string]interface{}{
		"success":       chunkedResult.Error == "",
		"action":        mode + "_flags",
		"flags":         flags,
		"success_count": chunkedResult.SuccessCount,
		"failed_count":  chunkedResult.FailCount,
		"total":         chunkedResult.TotalItems,
		"chunks_used":   chunkedResult.ChunksUsed,
	}
	if chunkedResult.Error != "" {
		response["error"] = chunkedResult.Error
	}

	result, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(result)), nil
}
