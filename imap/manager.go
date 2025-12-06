package imap

import (
	"fmt"
	"sync"

	"github.com/spetr/mcp-mail/cache"
	"github.com/spetr/mcp-mail/types"
)

// Manager handles multiple IMAP account connections
type Manager struct {
	mu      sync.RWMutex
	clients map[string]*Client
	cache   *cache.Manager
}

// NewManager creates a new IMAP connection manager
func NewManager(cacheMgr *cache.Manager) *Manager {
	return &Manager{
		clients: make(map[string]*Client),
		cache:   cacheMgr,
	}
}

// GetCache returns the cache manager
func (m *Manager) GetCache() *cache.Manager {
	return m.cache
}

// Connect connects to an IMAP account
func (m *Manager) Connect(account types.AccountConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already connected
	if client, exists := m.clients[account.ID]; exists {
		if client.IsConnected() {
			return nil // Already connected
		}
		// Clean up old connection
		client.Close()
		delete(m.clients, account.ID)
	}

	// Create and connect new client
	client := NewClient(account, m.cache)
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to %s: %w", account.ID, err)
	}

	m.clients[account.ID] = client
	return nil
}

// Disconnect disconnects from an IMAP account
func (m *Manager) Disconnect(accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[accountID]
	if !exists {
		return fmt.Errorf("account '%s' is not connected", accountID)
	}

	if err := client.Close(); err != nil {
		return fmt.Errorf("failed to disconnect from %s: %w", accountID, err)
	}

	delete(m.clients, accountID)
	return nil
}

// DisconnectAll disconnects from all accounts
func (m *Manager) DisconnectAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, client := range m.clients {
		client.Close()
		delete(m.clients, id)
	}
}

// GetClient returns a connected client for the given account ID
func (m *Manager) GetClient(accountID string) (*Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[accountID]
	if !exists {
		return nil, fmt.Errorf("account '%s' is not connected", accountID)
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("account '%s' connection lost", accountID)
	}

	return client, nil
}

// IsConnected checks if an account is connected
func (m *Manager) IsConnected(accountID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[accountID]
	if !exists {
		return false
	}

	return client.IsConnected()
}

// GetStatus returns the status of all accounts
func (m *Manager) GetStatus(accounts []types.AccountConfig) []types.AccountStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]types.AccountStatus, 0, len(accounts))
	for _, acc := range accounts {
		status := types.AccountStatus{
			ID:        acc.ID,
			Name:      acc.Name,
			Connected: false,
		}

		if client, exists := m.clients[acc.ID]; exists {
			status.Connected = client.IsConnected()
			if status.Connected {
				status.Capabilities = client.Capabilities()
			}
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// GetConnectedAccounts returns IDs of all connected accounts
func (m *Manager) GetConnectedAccounts() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.clients))
	for id, client := range m.clients {
		if client.IsConnected() {
			ids = append(ids, id)
		}
	}

	return ids
}
