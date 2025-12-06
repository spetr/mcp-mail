package imap

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
)

// SetFlags sets flags on a message (replaces existing flags)
func (c *Client) SetFlags(folder string, uid uint32, flags []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return err
	}

	if err := c.selectFolder(folder); err != nil {
		return err
	}

	uidSet := imap.UIDSetNum(imap.UID(uid))

	storeFlags := imap.StoreFlags{
		Op:     imap.StoreFlagsSet,
		Silent: true,
		Flags:  stringsToFlags(flags),
	}

	storeCmd := c.client.Store(uidSet, &storeFlags, nil)
	if err := storeCmd.Close(); err != nil {
		return fmt.Errorf("failed to set flags: %w", err)
	}

	return nil
}

// AddFlags adds flags to a message
func (c *Client) AddFlags(folder string, uid uint32, flags []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return err
	}

	if err := c.selectFolder(folder); err != nil {
		return err
	}

	uidSet := imap.UIDSetNum(imap.UID(uid))

	storeFlags := imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: true,
		Flags:  stringsToFlags(flags),
	}

	storeCmd := c.client.Store(uidSet, &storeFlags, nil)
	if err := storeCmd.Close(); err != nil {
		return fmt.Errorf("failed to add flags: %w", err)
	}

	return nil
}

// RemoveFlags removes flags from a message
func (c *Client) RemoveFlags(folder string, uid uint32, flags []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return err
	}

	if err := c.selectFolder(folder); err != nil {
		return err
	}

	uidSet := imap.UIDSetNum(imap.UID(uid))

	storeFlags := imap.StoreFlags{
		Op:     imap.StoreFlagsDel,
		Silent: true,
		Flags:  stringsToFlags(flags),
	}

	storeCmd := c.client.Store(uidSet, &storeFlags, nil)
	if err := storeCmd.Close(); err != nil {
		return fmt.Errorf("failed to remove flags: %w", err)
	}

	return nil
}

// MarkRead marks a message as read (adds \Seen flag)
func (c *Client) MarkRead(folder string, uid uint32) error {
	return c.AddFlags(folder, uid, []string{string(imap.FlagSeen)})
}

// MarkUnread marks a message as unread (removes \Seen flag)
func (c *Client) MarkUnread(folder string, uid uint32) error {
	return c.RemoveFlags(folder, uid, []string{string(imap.FlagSeen)})
}

// Flag marks a message with \Flagged (starred/important)
func (c *Client) Flag(folder string, uid uint32) error {
	return c.AddFlags(folder, uid, []string{string(imap.FlagFlagged)})
}

// Unflag removes the \Flagged flag from a message
func (c *Client) Unflag(folder string, uid uint32) error {
	return c.RemoveFlags(folder, uid, []string{string(imap.FlagFlagged)})
}

// MarkAnswered marks a message as answered
func (c *Client) MarkAnswered(folder string, uid uint32) error {
	return c.AddFlags(folder, uid, []string{string(imap.FlagAnswered)})
}

// stringsToFlags converts string slice to IMAP flags
func stringsToFlags(strs []string) []imap.Flag {
	flags := make([]imap.Flag, 0, len(strs))
	for _, s := range strs {
		flags = append(flags, imap.Flag(s))
	}
	return flags
}

// GetFlags returns the flags of a message
func (c *Client) GetFlags(folder string, uid uint32) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if err := c.selectFolder(folder); err != nil {
		return nil, err
	}

	uidSet := imap.UIDSetNum(imap.UID(uid))

	fetchOpts := &imap.FetchOptions{
		UID:   true,
		Flags: true,
	}

	fetchCmd := c.client.Fetch(uidSet, fetchOpts)
	defer fetchCmd.Close()

	msg := fetchCmd.Next()
	if msg == nil {
		return nil, fmt.Errorf("message with UID %d not found", uid)
	}

	buf, err := msg.Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to collect message data: %w", err)
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}

	return convertFlags(buf.Flags), nil
}
