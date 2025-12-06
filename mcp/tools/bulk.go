package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spetr/mcp-mail/imap"
	"github.com/spetr/mcp-mail/types"
)

// BulkMessageRef represents a message reference with UID and safe_id for bulk operations
type BulkMessageRef struct {
	UID    uint32 `json:"uid"`
	SafeID string `json:"safe_id"`
}

func (r *Registry) registerBulkTools() {
	// messages_delete_bulk - Delete multiple messages (moves to trash)
	r.addTool(
		mcp.NewTool("messages_delete_bulk",
			mcp.WithDescription("Delete multiple messages by moving them to trash. Requires array of {uid, safe_id} objects from message_list. Messages are NOT permanently deleted."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			mcp.WithArray("messages",
				mcp.Required(),
				mcp.Description("Array of message references [{uid: number, safe_id: string}, ...] from message_list response"),
			),
		),
		r.handleMessagesDeleteBulk,
	)

	// messages_move_bulk - Move multiple messages
	r.addTool(
		mcp.NewTool("messages_move_bulk",
			mcp.WithDescription("Move multiple messages to another folder. Requires array of {uid, safe_id} objects from message_list."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Source folder name"),
			mcp.WithArray("messages",
				mcp.Required(),
				mcp.Description("Array of message references [{uid: number, safe_id: string}, ...] from message_list response"),
			),
			requiredString("target_folder", "Destination folder name"),
		),
		r.handleMessagesMovesBulk,
	)

	// messages_copy_bulk - Copy multiple messages
	r.addTool(
		mcp.NewTool("messages_copy_bulk",
			mcp.WithDescription("Copy multiple messages to another folder. Requires array of {uid, safe_id} objects from message_list."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Source folder name"),
			mcp.WithArray("messages",
				mcp.Required(),
				mcp.Description("Array of message references [{uid: number, safe_id: string}, ...] from message_list response"),
			),
			requiredString("target_folder", "Destination folder name"),
		),
		r.handleMessagesCopyBulk,
	)

	// messages_flag_bulk - Set flags on multiple messages (non-destructive, uses UIDs)
	r.addTool(
		mcp.NewTool("messages_flag_bulk",
			mcp.WithDescription("Set flags on multiple messages. This is a non-destructive operation so UIDs are accepted."),
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

// getMessageRefs extracts BulkMessageRef array from request
func getMessageRefs(request mcp.CallToolRequest) ([]BulkMessageRef, error) {
	args := request.GetArguments()
	messagesRaw, ok := args["messages"]
	if !ok {
		return nil, fmt.Errorf("messages parameter required")
	}

	messagesArray, ok := messagesRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("messages must be an array")
	}

	refs := make([]BulkMessageRef, 0, len(messagesArray))
	for i, item := range messagesArray {
		msgMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("message[%d]: must be an object with uid and safe_id", i)
		}

		uidRaw, ok := msgMap["uid"]
		if !ok {
			return nil, fmt.Errorf("message[%d]: uid is required", i)
		}
		uidFloat, ok := uidRaw.(float64)
		if !ok {
			return nil, fmt.Errorf("message[%d]: uid must be a number", i)
		}

		safeID, ok := msgMap["safe_id"].(string)
		if !ok || safeID == "" {
			return nil, fmt.Errorf("message[%d]: safe_id is required", i)
		}

		refs = append(refs, BulkMessageRef{
			UID:    uint32(uidFloat),
			SafeID: safeID,
		})
	}

	return refs, nil
}

func (r *Registry) handleMessagesDeleteBulk(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check read-only mode
	if err := r.protectMgr.CheckReadOnly(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")

	// Get message references
	messageRefs, err := getMessageRefs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(messageRefs) == 0 {
		return mcp.NewToolResultError("no messages provided"), nil
	}

	// Check bulk operation limit
	if len(messageRefs) > maxBulkOperations {
		return mcp.NewToolResultError(fmt.Sprintf("too many messages: %d (max %d)", len(messageRefs), maxBulkOperations)), nil
	}

	// Get account secret for HMAC validation
	secret, err := r.getAccountSecret(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get account secret: %v", err)), nil
	}

	// Validate all messages and extract UIDs
	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	uids, validationErrors := r.validateMessageRefsBulk(secret, client, folder, messageRefs)
	if len(validationErrors) > 0 && len(uids) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("all messages failed validation: %v", validationErrors[0])), nil
	}

	return r.executeDeleteBulkWithErrors(accountID, folder, uids, validationErrors)
}

func (r *Registry) handleMessagesMovesBulk(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check read-only mode
	if err := r.protectMgr.CheckReadOnly(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	targetFolder := request.GetString("target_folder", "")

	// Get message references
	messageRefs, err := getMessageRefs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(messageRefs) == 0 {
		return mcp.NewToolResultError("no messages provided"), nil
	}

	// Check bulk operation limit
	if len(messageRefs) > maxBulkOperations {
		return mcp.NewToolResultError(fmt.Sprintf("too many messages: %d (max %d)", len(messageRefs), maxBulkOperations)), nil
	}

	// Get account secret for HMAC validation
	secret, err := r.getAccountSecret(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get account secret: %v", err)), nil
	}

	// Validate all messages and extract UIDs
	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	uids, validationErrors := r.validateMessageRefsBulk(secret, client, folder, messageRefs)
	if len(validationErrors) > 0 && len(uids) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("all messages failed validation: %v", validationErrors[0])), nil
	}

	return r.executeMoveBulkWithErrors(accountID, folder, uids, targetFolder, validationErrors)
}

func (r *Registry) handleMessagesCopyBulk(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check read-only mode
	if err := r.protectMgr.CheckReadOnly(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	targetFolder := request.GetString("target_folder", "")

	// Get message references
	messageRefs, err := getMessageRefs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(messageRefs) == 0 {
		return mcp.NewToolResultError("no messages provided"), nil
	}

	// Check bulk operation limit
	if len(messageRefs) > maxBulkOperations {
		return mcp.NewToolResultError(fmt.Sprintf("too many messages: %d (max %d)", len(messageRefs), maxBulkOperations)), nil
	}

	// Get account secret for HMAC validation
	secret, err := r.getAccountSecret(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get account secret: %v", err)), nil
	}

	// Validate all messages and extract UIDs
	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	uids, validationErrors := r.validateMessageRefsBulk(secret, client, folder, messageRefs)
	if len(validationErrors) > 0 && len(uids) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("all messages failed validation: %v", validationErrors[0])), nil
	}

	return r.executeCopyBulkWithErrors(accountID, folder, uids, targetFolder, validationErrors)
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

// validateMessageRefsBulk validates multiple message references using HMAC and returns valid UIDs and errors
func (r *Registry) validateMessageRefsBulk(secret string, client *imap.Client, folder string, refs []BulkMessageRef) ([]uint32, []string) {
	var validUIDs []uint32
	var errors []string

	if len(refs) == 0 {
		return nil, errors
	}

	// Collect UIDs for batch header fetch
	uidsToFetch := make([]uint32, len(refs))
	for i, ref := range refs {
		uidsToFetch[i] = ref.UID
	}

	// Fetch headers for all UIDs in batch
	headers, err := client.GetMessageHeadersBatch(folder, uidsToFetch)
	if err != nil {
		errors = append(errors, fmt.Sprintf("failed to fetch message headers: %v", err))
		return nil, errors
	}

	// Create map of UID to headers for easy lookup
	headersByUID := make(map[uint32]*types.MessageEnvelope)
	for _, h := range headers {
		if h != nil {
			headersByUID[h.UID] = h
		}
	}

	// Validate each message reference
	for _, ref := range refs {
		header, exists := headersByUID[ref.UID]
		if !exists {
			errors = append(errors, fmt.Sprintf("message UID %d not found", ref.UID))
			continue
		}

		if err := validateSafeID(secret, ref.SafeID, ref.UID, header); err != nil {
			errors = append(errors, fmt.Sprintf("validation failed for UID %d: %v", ref.UID, err))
			continue
		}

		validUIDs = append(validUIDs, ref.UID)
	}

	return validUIDs, errors
}

// Execute functions that actually perform the operations
// All operations use chunking to avoid overwhelming the IMAP server

const defaultChunkSize = 50

func (r *Registry) executeDeleteBulkWithErrors(accountID, folder string, uids []uint32, validationErrors []string) (*mcp.CallToolResult, error) {
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
		"success":           chunkedResult.Error == "" && len(validationErrors) == 0,
		"action":            "moved_to_trash",
		"trash_folder":      trashFolder,
		"moved_count":       chunkedResult.SuccessCount,
		"failed_count":      chunkedResult.FailCount,
		"validation_errors": len(validationErrors),
		"total_requested":   len(uids) + len(validationErrors),
	}
	if chunkedResult.Error != "" {
		response["error"] = chunkedResult.Error
	}
	if len(validationErrors) > 0 {
		response["validation_error_details"] = validationErrors
	}

	result, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) executeMoveBulkWithErrors(accountID, folder string, uids []uint32, targetFolder string, validationErrors []string) (*mcp.CallToolResult, error) {
	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	chunkedResult := client.MoveMessagesChunked(folder, uids, targetFolder, defaultChunkSize)

	response := map[string]interface{}{
		"success":           chunkedResult.Error == "" && len(validationErrors) == 0,
		"action":            "moved",
		"target":            targetFolder,
		"moved_count":       chunkedResult.SuccessCount,
		"failed_count":      chunkedResult.FailCount,
		"validation_errors": len(validationErrors),
		"total_requested":   len(uids) + len(validationErrors),
	}
	if chunkedResult.Error != "" {
		response["error"] = chunkedResult.Error
	}
	if len(validationErrors) > 0 {
		response["validation_error_details"] = validationErrors
	}

	result, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) executeCopyBulkWithErrors(accountID, folder string, uids []uint32, targetFolder string, validationErrors []string) (*mcp.CallToolResult, error) {
	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	chunkedResult := client.CopyMessagesChunked(folder, uids, targetFolder, defaultChunkSize)

	response := map[string]interface{}{
		"success":           chunkedResult.Error == "" && len(validationErrors) == 0,
		"action":            "copied",
		"target":            targetFolder,
		"copied_count":      chunkedResult.SuccessCount,
		"failed_count":      chunkedResult.FailCount,
		"validation_errors": len(validationErrors),
		"total_requested":   len(uids) + len(validationErrors),
	}
	if chunkedResult.Error != "" {
		response["error"] = chunkedResult.Error
	}
	if len(validationErrors) > 0 {
		response["validation_error_details"] = validationErrors
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
