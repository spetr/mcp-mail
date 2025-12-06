package imap

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
)

// ChunkedResult contains results from a chunked operation
type ChunkedResult struct {
	TotalItems   int    `json:"total_items"`
	SuccessCount int    `json:"success_count"`
	FailCount    int    `json:"fail_count"`
	ChunksUsed   int    `json:"chunks_used"`
	Error        string `json:"error,omitempty"`
}

// DefaultChunkSize is the default number of UIDs to process in one batch
const DefaultChunkSize = 50

// chunkUIDs splits a slice of UIDs into chunks of the given size
func chunkUIDs(uids []uint32, chunkSize int) [][]uint32 {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	var chunks [][]uint32
	for i := 0; i < len(uids); i += chunkSize {
		end := i + chunkSize
		if end > len(uids) {
			end = len(uids)
		}
		chunks = append(chunks, uids[i:end])
	}
	return chunks
}

// bulkOperation is a function type for bulk operations on UIDs
type bulkOperation func(uids []uint32) (success, fail int, err error)

// executeChunked executes a bulk operation in chunks
func executeChunked(uids []uint32, chunkSize int, op bulkOperation) *ChunkedResult {
	chunks := chunkUIDs(uids, chunkSize)
	result := &ChunkedResult{
		TotalItems: len(uids),
		ChunksUsed: len(chunks),
	}

	for _, chunk := range chunks {
		success, fail, err := op(chunk)
		result.SuccessCount += success
		result.FailCount += fail
		if err != nil {
			result.Error = err.Error()
			return result
		}
	}

	return result
}

// DeleteMessages deletes multiple messages by UID
func (c *Client) DeleteMessages(folder string, uids []uint32, permanent bool) (successCount, failCount int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return 0, len(uids), err
	}

	if err := c.selectFolder(folder); err != nil {
		return 0, len(uids), err
	}

	// Create UID set
	var uidSet imap.UIDSet
	for _, uid := range uids {
		uidSet.AddNum(imap.UID(uid))
	}

	// Mark as deleted
	storeFlags := imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: true,
		Flags:  []imap.Flag{imap.FlagDeleted},
	}

	storeCmd := c.client.Store(uidSet, &storeFlags, nil)
	if err := storeCmd.Close(); err != nil {
		return 0, len(uids), fmt.Errorf("failed to mark messages as deleted: %w", err)
	}

	successCount = len(uids)

	// Expunge if permanent
	if permanent {
		expungeCmd := c.client.Expunge()
		if err := expungeCmd.Close(); err != nil {
			return successCount, 0, fmt.Errorf("failed to expunge: %w", err)
		}
	}

	// Invalidate folder cache
	if c.cache != nil {
		c.cache.InvalidateFolderContent(c.account.ID, folder)
	}

	return successCount, 0, nil
}

// MoveMessages moves multiple messages to another folder
func (c *Client) MoveMessages(folder string, uids []uint32, targetFolder string) (successCount, failCount int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return 0, len(uids), err
	}

	if err := c.selectFolder(folder); err != nil {
		return 0, len(uids), err
	}

	// Create UID set
	var uidSet imap.UIDSet
	for _, uid := range uids {
		uidSet.AddNum(imap.UID(uid))
	}

	// Check if MOVE is supported
	if c.client.Caps().Has(imap.CapMove) {
		if _, err := c.client.Move(uidSet, targetFolder).Wait(); err != nil {
			return 0, len(uids), fmt.Errorf("failed to move messages: %w", err)
		}
	} else {
		// Fallback: COPY + DELETE
		if _, err := c.client.Copy(uidSet, targetFolder).Wait(); err != nil {
			return 0, len(uids), fmt.Errorf("failed to copy messages: %w", err)
		}

		storeFlags := imap.StoreFlags{
			Op:     imap.StoreFlagsAdd,
			Silent: true,
			Flags:  []imap.Flag{imap.FlagDeleted},
		}

		storeCmd := c.client.Store(uidSet, &storeFlags, nil)
		if err := storeCmd.Close(); err != nil {
			return 0, len(uids), fmt.Errorf("failed to mark originals as deleted: %w", err)
		}

		expungeCmd := c.client.Expunge()
		if err := expungeCmd.Close(); err != nil {
			return len(uids), 0, fmt.Errorf("messages copied but failed to expunge originals: %w", err)
		}
	}

	// Invalidate cache for both folders
	if c.cache != nil {
		c.cache.InvalidateFolderContent(c.account.ID, folder)
		c.cache.InvalidateFolderContent(c.account.ID, targetFolder)
	}

	return len(uids), 0, nil
}

// CopyMessages copies multiple messages to another folder
func (c *Client) CopyMessages(folder string, uids []uint32, targetFolder string) (successCount, failCount int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return 0, len(uids), err
	}

	if err := c.selectFolder(folder); err != nil {
		return 0, len(uids), err
	}

	// Create UID set
	var uidSet imap.UIDSet
	for _, uid := range uids {
		uidSet.AddNum(imap.UID(uid))
	}

	if _, err := c.client.Copy(uidSet, targetFolder).Wait(); err != nil {
		return 0, len(uids), fmt.Errorf("failed to copy messages: %w", err)
	}

	// Invalidate cache for target folder
	if c.cache != nil {
		c.cache.InvalidateFolderContent(c.account.ID, targetFolder)
	}

	return len(uids), 0, nil
}

// SetFlagsBulk sets flags on multiple messages (replaces existing)
func (c *Client) SetFlagsBulk(folder string, uids []uint32, flags []string) (successCount, failCount int, err error) {
	return c.storeFlagsBulk(folder, uids, flags, imap.StoreFlagsSet)
}

// AddFlagsBulk adds flags to multiple messages
func (c *Client) AddFlagsBulk(folder string, uids []uint32, flags []string) (successCount, failCount int, err error) {
	return c.storeFlagsBulk(folder, uids, flags, imap.StoreFlagsAdd)
}

// RemoveFlagsBulk removes flags from multiple messages
func (c *Client) RemoveFlagsBulk(folder string, uids []uint32, flags []string) (successCount, failCount int, err error) {
	return c.storeFlagsBulk(folder, uids, flags, imap.StoreFlagsDel)
}

func (c *Client) storeFlagsBulk(folder string, uids []uint32, flags []string, op imap.StoreFlagsOp) (successCount, failCount int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return 0, len(uids), err
	}

	if err := c.selectFolder(folder); err != nil {
		return 0, len(uids), err
	}

	// Create UID set
	var uidSet imap.UIDSet
	for _, uid := range uids {
		uidSet.AddNum(imap.UID(uid))
	}

	// Convert flags
	imapFlags := make([]imap.Flag, len(flags))
	for i, f := range flags {
		imapFlags[i] = imap.Flag(f)
	}

	storeFlags := imap.StoreFlags{
		Op:     op,
		Silent: true,
		Flags:  imapFlags,
	}

	storeCmd := c.client.Store(uidSet, &storeFlags, nil)
	if err := storeCmd.Close(); err != nil {
		return 0, len(uids), fmt.Errorf("failed to store flags: %w", err)
	}

	return len(uids), 0, nil
}

// DeleteMessagesChunked deletes messages in chunks to avoid overwhelming the server
func (c *Client) DeleteMessagesChunked(folder string, uids []uint32, permanent bool, chunkSize int) *ChunkedResult {
	return executeChunked(uids, chunkSize, func(chunk []uint32) (int, int, error) {
		return c.DeleteMessages(folder, chunk, permanent)
	})
}

// MoveMessagesChunked moves messages in chunks
func (c *Client) MoveMessagesChunked(folder string, uids []uint32, targetFolder string, chunkSize int) *ChunkedResult {
	return executeChunked(uids, chunkSize, func(chunk []uint32) (int, int, error) {
		return c.MoveMessages(folder, chunk, targetFolder)
	})
}

// CopyMessagesChunked copies messages in chunks
func (c *Client) CopyMessagesChunked(folder string, uids []uint32, targetFolder string, chunkSize int) *ChunkedResult {
	return executeChunked(uids, chunkSize, func(chunk []uint32) (int, int, error) {
		return c.CopyMessages(folder, chunk, targetFolder)
	})
}

// SetFlagsBulkChunked sets flags on messages in chunks
func (c *Client) SetFlagsBulkChunked(folder string, uids []uint32, flags []string, chunkSize int) *ChunkedResult {
	return executeChunked(uids, chunkSize, func(chunk []uint32) (int, int, error) {
		return c.SetFlagsBulk(folder, chunk, flags)
	})
}

// AddFlagsBulkChunked adds flags to messages in chunks
func (c *Client) AddFlagsBulkChunked(folder string, uids []uint32, flags []string, chunkSize int) *ChunkedResult {
	return executeChunked(uids, chunkSize, func(chunk []uint32) (int, int, error) {
		return c.AddFlagsBulk(folder, chunk, flags)
	})
}

// RemoveFlagsBulkChunked removes flags from messages in chunks
func (c *Client) RemoveFlagsBulkChunked(folder string, uids []uint32, flags []string, chunkSize int) *ChunkedResult {
	return executeChunked(uids, chunkSize, func(chunk []uint32) (int, int, error) {
		return c.RemoveFlagsBulk(folder, chunk, flags)
	})
}

// GetTrashFolder attempts to find the trash folder from the server
func (c *Client) GetTrashFolder() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return ""
	}

	// Common trash folder names
	trashNames := []string{
		"Trash",
		"Deleted Items",
		"Deleted Messages",
		"Koš",           // Czech
		"Papierkorb",    // German
		"Corbeille",     // French
		"[Gmail]/Trash", // Gmail
	}

	// List folders and find one with \Trash attribute or matching name
	listCmd := c.client.List("", "*", nil)
	defer listCmd.Close()

	var trashFolder string
	for {
		mailbox := listCmd.Next()
		if mailbox == nil {
			break
		}

		// Check for \Trash attribute
		for _, attr := range mailbox.Attrs {
			if attr == "\\Trash" {
				return mailbox.Mailbox
			}
		}

		// Check name match
		for _, name := range trashNames {
			if mailbox.Mailbox == name {
				trashFolder = mailbox.Mailbox
			}
		}
	}

	return trashFolder
}
