package imap

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/spetr/mcp-mail/types"
)

// ListFolders returns all folders for the account
func (c *Client) ListFolders() ([]types.FolderListItem, error) {
	// Check cache first
	if c.cache != nil {
		if cached := c.cache.GetFolderList(c.account.ID); cached != nil {
			return cached, nil
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	// List all mailboxes
	listCmd := c.client.List("", "*", nil)
	mailboxes, err := listCmd.Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	folders := make([]types.FolderListItem, 0, len(mailboxes))
	for _, mb := range mailboxes {
		folder := types.FolderListItem{
			Name:       mb.Mailbox,
			Delimiter:  string(mb.Delim),
			Attributes: convertMailboxAttrs(mb.Attrs),
		}

		// Detect special use
		folder.IsInbox = strings.EqualFold(mb.Mailbox, "INBOX")
		for _, attr := range mb.Attrs {
			switch attr {
			case imap.MailboxAttrSent:
				folder.IsSent = true
			case imap.MailboxAttrDrafts:
				folder.IsDrafts = true
			case imap.MailboxAttrTrash:
				folder.IsTrash = true
			case imap.MailboxAttrJunk:
				folder.IsJunk = true
			case imap.MailboxAttrArchive:
				folder.IsArchive = true
			}
		}

		folders = append(folders, folder)
	}

	// Cache the result
	if c.cache != nil {
		c.cache.SetFolderList(c.account.ID, folders)
	}

	return folders, nil
}

// GetFolderInfo returns detailed information about a folder
func (c *Client) GetFolderInfo(name string) (*types.FolderInfo, error) {
	// Check cache first
	if c.cache != nil {
		if cached := c.cache.GetFolderInfo(c.account.ID, name); cached != nil {
			return cached, nil
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	// Get STATUS of the mailbox
	statusOpts := &imap.StatusOptions{
		NumMessages: true,
		NumUnseen:   true,
		UIDNext:     true,
		UIDValidity: true,
	}

	statusCmd := c.client.Status(name, statusOpts)
	data, err := statusCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to get folder status: %w", err)
	}

	info := &types.FolderInfo{
		Name:        name,
		UIDValidity: data.UIDValidity,
		UIDNext:     uint32(data.UIDNext),
	}
	if data.NumMessages != nil {
		info.Messages = *data.NumMessages
	}
	if data.NumUnseen != nil {
		info.Unseen = *data.NumUnseen
	}

	// Get delimiter from LIST
	listCmd := c.client.List("", name, nil)
	mailboxes, err := listCmd.Collect()
	if err == nil && len(mailboxes) > 0 {
		info.Delimiter = string(mailboxes[0].Delim)
		info.Attributes = convertMailboxAttrs(mailboxes[0].Attrs)
	}

	// Cache the result and check UIDVALIDITY
	if c.cache != nil {
		c.cache.CheckUIDValidity(c.account.ID, name, data.UIDValidity)
		c.cache.SetFolderInfo(c.account.ID, name, info)
	}

	return info, nil
}

// CreateFolder creates a new mailbox folder
func (c *Client) CreateFolder(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return err
	}

	if err := c.client.Create(name, nil).Wait(); err != nil {
		return fmt.Errorf("failed to create folder '%s': %w", name, err)
	}

	// Invalidate folder list cache
	if c.cache != nil {
		c.cache.InvalidateFolderList(c.account.ID)
	}

	return nil
}

// RenameFolder renames a mailbox folder
func (c *Client) RenameFolder(oldName, newName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return err
	}

	if err := c.client.Rename(oldName, newName, nil).Wait(); err != nil {
		return fmt.Errorf("failed to rename folder from '%s' to '%s': %w", oldName, newName, err)
	}

	// Update selected if we renamed the current mailbox
	if c.selected == oldName {
		c.selected = newName
	}

	// Invalidate cache for both old and new folder
	if c.cache != nil {
		c.cache.InvalidateFolderList(c.account.ID)
		c.cache.InvalidateFolder(c.account.ID, oldName)
		c.cache.InvalidateFolder(c.account.ID, newName)
	}

	return nil
}

// DeleteFolder deletes a mailbox folder
func (c *Client) DeleteFolder(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return err
	}

	// Cannot delete selected mailbox, so unselect first if needed
	if c.selected == name {
		// Select INBOX to unselect current
		if _, err := c.client.Select("INBOX", nil).Wait(); err != nil {
			return fmt.Errorf("failed to unselect mailbox before delete: %w", err)
		}
		c.selected = "INBOX"
	}

	if err := c.client.Delete(name).Wait(); err != nil {
		return fmt.Errorf("failed to delete folder '%s': %w", name, err)
	}

	// Invalidate cache
	if c.cache != nil {
		c.cache.InvalidateFolderList(c.account.ID)
		c.cache.InvalidateFolder(c.account.ID, name)
	}

	return nil
}

// SubscribeFolder subscribes to a folder
func (c *Client) SubscribeFolder(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return err
	}

	if err := c.client.Subscribe(name).Wait(); err != nil {
		return fmt.Errorf("failed to subscribe to folder '%s': %w", name, err)
	}

	return nil
}

// UnsubscribeFolder unsubscribes from a folder
func (c *Client) UnsubscribeFolder(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return err
	}

	if err := c.client.Unsubscribe(name).Wait(); err != nil {
		return fmt.Errorf("failed to unsubscribe from folder '%s': %w", name, err)
	}

	return nil
}

// convertMailboxAttrs converts IMAP mailbox attributes to string slice
func convertMailboxAttrs(attrs []imap.MailboxAttr) []string {
	result := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		result = append(result, string(attr))
	}
	return result
}

// FolderExists checks if a folder exists
func (c *Client) FolderExists(name string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return false, err
	}

	// List the specific folder
	listCmd := c.client.List("", name, nil)
	mailboxes, err := listCmd.Collect()
	if err != nil {
		return false, fmt.Errorf("failed to check folder existence: %w", err)
	}

	// Check if the exact folder was found
	for _, mb := range mailboxes {
		if mb.Mailbox == name {
			return true, nil
		}
	}

	return false, nil
}

// GetDelimiter returns the folder delimiter for the server (usually "/" or ".")
func (c *Client) GetDelimiter() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return "", err
	}

	// List root to get delimiter
	listCmd := c.client.List("", "", nil)
	mailboxes, err := listCmd.Collect()
	if err != nil {
		return "", fmt.Errorf("failed to get delimiter: %w", err)
	}

	if len(mailboxes) > 0 {
		return string(mailboxes[0].Delim), nil
	}

	// Default to "/" if we can't determine
	return "/", nil
}

// GetOrCreateFolder creates a folder if it doesn't exist, returns info about the folder
func (c *Client) GetOrCreateFolder(name string) (*types.FolderInfo, bool, error) {
	// Check if exists first (without holding lock for too long)
	exists, err := c.FolderExists(name)
	if err != nil {
		return nil, false, err
	}

	created := false
	if !exists {
		// Create the folder
		if err := c.CreateFolder(name); err != nil {
			return nil, false, err
		}
		created = true
	}

	// Get folder info
	info, err := c.GetFolderInfo(name)
	if err != nil {
		return nil, created, err
	}

	return info, created, nil
}

// EnsureFolderPath creates all folders in a path, like "INBOX/Projects/2024"
// Returns info about the final folder and list of created folders
func (c *Client) EnsureFolderPath(path string) (*types.FolderInfo, []string, error) {
	// Get delimiter
	delim, err := c.GetDelimiter()
	if err != nil {
		return nil, nil, err
	}

	// Split path into parts
	parts := strings.Split(path, delim)
	if len(parts) == 0 {
		return nil, nil, fmt.Errorf("invalid path: %s", path)
	}

	created := []string{}
	currentPath := ""

	// Create each level of the hierarchy
	for i, part := range parts {
		if part == "" {
			continue
		}

		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + delim + part
		}

		// Check if this level exists
		exists, err := c.FolderExists(currentPath)
		if err != nil {
			return nil, created, fmt.Errorf("failed to check folder '%s': %w", currentPath, err)
		}

		if !exists {
			// Create this level
			if err := c.CreateFolder(currentPath); err != nil {
				// Try to continue - some servers auto-create parent folders
				if i < len(parts)-1 {
					continue
				}
				return nil, created, fmt.Errorf("failed to create folder '%s': %w", currentPath, err)
			}
			created = append(created, currentPath)
		}
	}

	// Get info about the final folder
	info, err := c.GetFolderInfo(path)
	if err != nil {
		return nil, created, err
	}

	return info, created, nil
}
