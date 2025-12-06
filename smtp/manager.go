package smtp

import (
	"fmt"
	"sync"

	"github.com/spetr/mcp-mail/config"
	"github.com/spetr/mcp-mail/types"
)

// Manager manages SMTP clients for multiple accounts
type Manager struct {
	mu        sync.RWMutex
	clients   map[string]*Client
	configMgr *config.Manager
}

// NewManager creates a new SMTP manager
func NewManager(configMgr *config.Manager) *Manager {
	return &Manager{
		clients:   make(map[string]*Client),
		configMgr: configMgr,
	}
}

// GetClient returns an SMTP client for the specified account
func (m *Manager) GetClient(accountID string) (*Client, error) {
	m.mu.RLock()
	client, exists := m.clients[accountID]
	m.mu.RUnlock()

	if exists {
		return client, nil
	}

	// Create new client
	return m.createClient(accountID)
}

// createClient creates a new SMTP client for an account
func (m *Manager) createClient(accountID string) (*Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring lock
	if client, exists := m.clients[accountID]; exists {
		return client, nil
	}

	// Get account config
	cfg := m.configMgr.Get()
	var accountCfg *types.AccountConfig
	for i := range cfg.Accounts {
		if cfg.Accounts[i].ID == accountID {
			accountCfg = &cfg.Accounts[i]
			break
		}
	}

	if accountCfg == nil {
		return nil, fmt.Errorf("account '%s' not found", accountID)
	}

	// Check if SMTP is configured - try to derive if not
	smtpHost := accountCfg.SMTPHost
	if smtpHost == "" {
		smtpHost = deriveSmtpHost(accountCfg.Host)
		if smtpHost == "" {
			return nil, fmt.Errorf("SMTP not configured for account '%s'", accountID)
		}
	}

	// Build SMTP config
	smtpConfig := SMTPConfig{
		Host:     smtpHost,
		Port:     accountCfg.SMTPPort,
		TLS:      accountCfg.SMTPTLS,
		Username: accountCfg.SMTPUsername,
		Password: accountCfg.SMTPPassword,
	}

	// Use IMAP credentials as fallback
	if smtpConfig.Username == "" {
		smtpConfig.Username = accountCfg.Username
	}
	if smtpConfig.Password == "" {
		smtpConfig.Password = accountCfg.Password
	}

	// Determine From address
	from := accountCfg.Username
	if accountCfg.FromAddress != "" {
		from = accountCfg.FromAddress
	}

	// Create client
	client := NewClient(smtpConfig, from)
	m.clients[accountID] = client

	return client, nil
}

// Send sends an email using the specified account
func (m *Manager) Send(accountID string, msg *OutgoingMessage) (*SendResult, error) {
	client, err := m.GetClient(accountID)
	if err != nil {
		return nil, err
	}

	return client.Send(msg)
}

// ClearClient removes a client from the cache (e.g., on config change)
func (m *Manager) ClearClient(accountID string) {
	m.mu.Lock()
	delete(m.clients, accountID)
	m.mu.Unlock()
}

// deriveSmtpHost tries to derive SMTP host from IMAP host
func deriveSmtpHost(imapHost string) string {
	// Common patterns
	patterns := map[string]string{
		"imap.gmail.com":     "smtp.gmail.com",
		"imap.mail.yahoo.com": "smtp.mail.yahoo.com",
		"imap-mail.outlook.com": "smtp-mail.outlook.com",
		"outlook.office365.com": "smtp.office365.com",
		"imap.seznam.cz":     "smtp.seznam.cz",
		"imap.centrum.cz":    "smtp.centrum.cz",
	}

	if smtpHost, ok := patterns[imapHost]; ok {
		return smtpHost
	}

	// Try replacing imap with smtp
	if len(imapHost) >= 5 && imapHost[:5] == "imap." {
		return "smtp." + imapHost[5:]
	}

	return ""
}
