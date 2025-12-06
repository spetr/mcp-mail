package imap

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/spetr/mcp-mail/cache"
	"github.com/spetr/mcp-mail/types"
)

// Client wraps an IMAP client for a single account
type Client struct {
	mu       sync.Mutex
	account  types.AccountConfig
	client   *imapclient.Client
	selected string // currently selected mailbox
	cache    *cache.Manager
}

// NewClient creates a new IMAP client for the given account
func NewClient(account types.AccountConfig, cacheMgr *cache.Manager) *Client {
	return &Client{
		account: account,
		cache:   cacheMgr,
	}
}

// Connect establishes a connection to the IMAP server
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", c.account.Host, c.account.Port)

	var client *imapclient.Client
	var err error

	if c.account.TLS {
		// Connect with TLS
		client, err = imapclient.DialTLS(addr, &imapclient.Options{
			TLSConfig: &tls.Config{
				ServerName: c.account.Host,
			},
		})
	} else {
		// Connect without TLS (not recommended)
		client, err = imapclient.DialInsecure(addr, nil)
	}

	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Login
	if err := client.Login(c.account.Username, c.account.Password).Wait(); err != nil {
		client.Close()
		return fmt.Errorf("login failed: %w", err)
	}

	c.client = client
	return nil
}

// Close closes the IMAP connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return nil
	}

	// Try to logout gracefully
	_ = c.client.Logout().Wait()

	err := c.client.Close()
	c.client = nil
	c.selected = ""
	return err
}

// IsConnected checks if the client is connected
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return false
	}

	// Try a NOOP to verify connection
	err := c.client.Noop().Wait()
	return err == nil
}

// Capabilities returns the server capabilities
func (c *Client) Capabilities() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return nil
	}

	caps := c.client.Caps()
	result := make([]string, 0)
	for cap := range caps {
		result = append(result, string(cap))
	}
	return result
}

// Select selects a mailbox
func (c *Client) Select(mailbox string) (*imap.SelectData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	data, err := c.client.Select(mailbox, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select mailbox '%s': %w", mailbox, err)
	}

	c.selected = mailbox
	return data, nil
}

// SelectedMailbox returns the currently selected mailbox name
func (c *Client) SelectedMailbox() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.selected
}

// AccountID returns the account ID
func (c *Client) AccountID() string {
	return c.account.ID
}

// ensureConnected checks connection and returns error if not connected
func (c *Client) ensureConnected() error {
	if c.client == nil {
		return fmt.Errorf("not connected")
	}
	return nil
}

// selectFolder selects a mailbox if not already selected
// Must be called with lock held
func (c *Client) selectFolder(folder string) error {
	if c.selected == folder {
		return nil
	}

	_, err := c.client.Select(folder, nil).Wait()
	if err != nil {
		return fmt.Errorf("failed to select folder '%s': %w", folder, err)
	}
	c.selected = folder
	return nil
}
