package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func (r *Registry) registerFolderTools() {
	// folder_list - List all folders
	r.addTool(
		mcp.NewTool("folder_list",
			mcp.WithDescription("List all mailbox folders for an account"),
			requiredString("account_id", "Account ID to list folders from"),
		),
		r.handleFolderList,
	)

	// folder_info - Get folder information
	r.addTool(
		mcp.NewTool("folder_info",
			mcp.WithDescription("Get detailed information about a folder"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
		),
		r.handleFolderInfo,
	)

	// folder_create - Create a new folder
	r.addTool(
		mcp.NewTool("folder_create",
			mcp.WithDescription("Create a new mailbox folder"),
			requiredString("account_id", "Account ID"),
			requiredString("name", "Folder name to create"),
		),
		r.handleFolderCreate,
	)

	// folder_rename - Rename a folder
	r.addTool(
		mcp.NewTool("folder_rename",
			mcp.WithDescription("Rename a mailbox folder"),
			requiredString("account_id", "Account ID"),
			requiredString("old_name", "Current folder name"),
			requiredString("new_name", "New folder name"),
		),
		r.handleFolderRename,
	)

	// folder_delete - Delete a folder
	r.addTool(
		mcp.NewTool("folder_delete",
			mcp.WithDescription("Delete a mailbox folder"),
			requiredString("account_id", "Account ID"),
			requiredString("name", "Folder name to delete"),
		),
		r.handleFolderDelete,
	)

	// folder_subscribe - Subscribe to a folder
	r.addTool(
		mcp.NewTool("folder_subscribe",
			mcp.WithDescription("Subscribe to a mailbox folder"),
			requiredString("account_id", "Account ID"),
			requiredString("name", "Folder name to subscribe to"),
		),
		r.handleFolderSubscribe,
	)

	// folder_unsubscribe - Unsubscribe from a folder
	r.addTool(
		mcp.NewTool("folder_unsubscribe",
			mcp.WithDescription("Unsubscribe from a mailbox folder"),
			requiredString("account_id", "Account ID"),
			requiredString("name", "Folder name to unsubscribe from"),
		),
		r.handleFolderUnsubscribe,
	)

	// folder_get_or_create - Get or create folder (idempotent)
	r.addTool(
		mcp.NewTool("folder_get_or_create",
			mcp.WithDescription("Get a folder, creating it if it doesn't exist. Idempotent - safe to call multiple times."),
			requiredString("account_id", "Account ID"),
			requiredString("name", "Folder name to get or create"),
		),
		r.handleFolderGetOrCreate,
	)

	// folder_ensure_path - Create nested folder path
	r.addTool(
		mcp.NewTool("folder_ensure_path",
			mcp.WithDescription("Ensure a nested folder path exists, creating all intermediate folders as needed. E.g., 'INBOX/Projects/2024' will create INBOX, INBOX/Projects, and INBOX/Projects/2024 if they don't exist."),
			requiredString("account_id", "Account ID"),
			requiredString("path", "Full folder path to ensure (e.g., 'INBOX/Projects/2024')"),
		),
		r.handleFolderEnsurePath,
	)

	// folder_exists - Check if folder exists
	r.addTool(
		mcp.NewTool("folder_exists",
			mcp.WithDescription("Check if a folder exists"),
			requiredString("account_id", "Account ID"),
			requiredString("name", "Folder name to check"),
		),
		r.handleFolderExists,
	)
}

func (r *Registry) handleFolderList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	folders, err := client.ListFolders()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list folders: %v", err)), nil
	}

	result, err := json.Marshal(folders)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleFolderInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	info, err := client.GetFolderInfo(folder)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get folder info: %v", err)), nil
	}

	result, err := json.Marshal(info)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleFolderCreate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	name := request.GetString("name", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.CreateFolder(name); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create folder: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Folder '%s' created successfully", name)), nil
}

func (r *Registry) handleFolderRename(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	oldName := request.GetString("old_name", "")
	newName := request.GetString("new_name", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.RenameFolder(oldName, newName); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to rename folder: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Folder renamed from '%s' to '%s'", oldName, newName)), nil
}

func (r *Registry) handleFolderDelete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	name := request.GetString("name", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.DeleteFolder(name); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete folder: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Folder '%s' deleted successfully", name)), nil
}

func (r *Registry) handleFolderSubscribe(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	name := request.GetString("name", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.SubscribeFolder(name); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to subscribe to folder: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Subscribed to folder '%s'", name)), nil
}

func (r *Registry) handleFolderUnsubscribe(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	name := request.GetString("name", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	if err := client.UnsubscribeFolder(name); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to unsubscribe from folder: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Unsubscribed from folder '%s'", name)), nil
}

func (r *Registry) handleFolderGetOrCreate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	name := request.GetString("name", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	info, created, err := client.GetOrCreateFolder(name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get or create folder: %v", err)), nil
	}

	response := map[string]interface{}{
		"folder":  info,
		"created": created,
	}
	if created {
		response["message"] = fmt.Sprintf("Folder '%s' created successfully", name)
	} else {
		response["message"] = fmt.Sprintf("Folder '%s' already exists", name)
	}

	result, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleFolderEnsurePath(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	path := request.GetString("path", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	info, created, err := client.EnsureFolderPath(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to ensure folder path: %v", err)), nil
	}

	response := map[string]interface{}{
		"folder":          info,
		"created_folders": created,
		"created_count":   len(created),
	}
	if len(created) > 0 {
		response["message"] = fmt.Sprintf("Created %d folder(s): %v", len(created), created)
	} else {
		response["message"] = fmt.Sprintf("Path '%s' already exists", path)
	}

	result, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleFolderExists(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	name := request.GetString("name", "")

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	exists, err := client.FolderExists(name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to check folder existence: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"folder": name,
		"exists": exists,
	})
	return mcp.NewToolResultText(string(result)), nil
}
