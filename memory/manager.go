package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Manager handles loading and saving account memories
type Manager struct {
	mu        sync.RWMutex
	memories  map[string]*AccountMemory
	directory string

	// Deduplication: track seen message UIDs per session to avoid duplicate counts
	// Key is "accountID:folder:uid"
	seenMessages map[string]bool
	seenMu       sync.RWMutex

	// Async save: track dirty accounts
	dirtyAccounts map[string]bool
	dirtyMu       sync.Mutex
	saveChan      chan struct{}
}

// NewManager creates a new memory manager
func NewManager(directory string) (*Manager, error) {
	// Expand ~ to home directory
	if directory == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		directory = filepath.Join(home, ".mcp-imap", "memory")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, fmt.Errorf("failed to create memory directory: %w", err)
	}

	m := &Manager{
		memories:      make(map[string]*AccountMemory),
		directory:     directory,
		seenMessages:  make(map[string]bool),
		dirtyAccounts: make(map[string]bool),
		saveChan:      make(chan struct{}, 1),
	}

	// Start background save goroutine
	go m.backgroundSaver()

	return m, nil
}

// backgroundSaver handles async saves
func (m *Manager) backgroundSaver() {
	for range m.saveChan {
		m.Flush()
	}
}

// markDirty marks an account as needing to be saved
func (m *Manager) markDirty(accountID string) {
	m.dirtyMu.Lock()
	m.dirtyAccounts[accountID] = true
	m.dirtyMu.Unlock()

	// Trigger save (non-blocking)
	select {
	case m.saveChan <- struct{}{}:
	default:
		// Already scheduled
	}
}

// GetDirectory returns the memory storage directory
func (m *Manager) GetDirectory() string {
	return m.directory
}

// Load loads memory for an account from disk
func (m *Manager) Load(accountID string) (*AccountMemory, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already loaded
	if mem, ok := m.memories[accountID]; ok {
		return mem, nil
	}

	filePath := m.getFilePath(accountID)

	// Check if file exists
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new memory
			mem := NewAccountMemory(accountID)
			m.memories[accountID] = mem
			return mem, nil
		}
		return nil, fmt.Errorf("failed to read memory file: %w", err)
	}

	var mem AccountMemory
	if err := json.Unmarshal(data, &mem); err != nil {
		return nil, fmt.Errorf("failed to parse memory file: %w", err)
	}

	m.memories[accountID] = &mem
	return &mem, nil
}

// Save saves memory for an account to disk (async)
func (m *Manager) Save(accountID string) error {
	m.mu.RLock()
	_, ok := m.memories[accountID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("memory for account '%s' not loaded", accountID)
	}

	m.markDirty(accountID)
	return nil
}

// SaveNow saves memory immediately (use for critical updates)
func (m *Manager) SaveNow(accountID string) error {
	m.mu.RLock()
	mem, ok := m.memories[accountID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("memory for account '%s' not loaded", accountID)
	}

	return m.writeToDisk(accountID, mem)
}

// Flush saves all dirty accounts immediately
func (m *Manager) Flush() {
	m.dirtyMu.Lock()
	toSave := m.dirtyAccounts
	m.dirtyAccounts = make(map[string]bool)
	m.dirtyMu.Unlock()

	for accountID := range toSave {
		m.mu.RLock()
		mem, ok := m.memories[accountID]
		m.mu.RUnlock()
		if ok {
			m.writeToDisk(accountID, mem)
		}
	}
}

// writeToDisk writes memory to disk
func (m *Manager) writeToDisk(accountID string, mem *AccountMemory) error {
	mem.LastUpdated = time.Now()

	data, err := json.MarshalIndent(mem, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal memory: %w", err)
	}

	filePath := m.getFilePath(accountID)
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write memory file: %w", err)
	}

	return nil
}

// Get returns the memory for an account (must be loaded first)
func (m *Manager) Get(accountID string) (*AccountMemory, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mem, ok := m.memories[accountID]
	return mem, ok
}

// GetOrLoad returns the memory for an account, loading it if necessary
func (m *Manager) GetOrLoad(accountID string) (*AccountMemory, error) {
	m.mu.RLock()
	mem, ok := m.memories[accountID]
	m.mu.RUnlock()

	if ok {
		return mem, nil
	}

	return m.Load(accountID)
}

// Update updates the memory for an account and saves it
func (m *Manager) Update(accountID string, updateFn func(*AccountMemory)) error {
	mem, err := m.GetOrLoad(accountID)
	if err != nil {
		return err
	}

	m.mu.Lock()
	updateFn(mem)
	m.mu.Unlock()

	return m.Save(accountID)
}

// AddContact adds an important contact
func (m *Manager) AddContact(accountID string, contact Contact) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		contact.AddedAt = time.Now()

		// Check if contact already exists and update
		for i, c := range mem.ImportantContacts {
			if c.Email == contact.Email {
				mem.ImportantContacts[i] = contact
				return
			}
		}

		mem.ImportantContacts = append(mem.ImportantContacts, contact)
	})
}

// RemoveContact removes a contact by email
func (m *Manager) RemoveContact(accountID, email string) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		for i, c := range mem.ImportantContacts {
			if c.Email == email {
				mem.ImportantContacts = append(mem.ImportantContacts[:i], mem.ImportantContacts[i+1:]...)
				return
			}
		}
	})
}

// AddUnwantedSender adds a sender to the unwanted list
func (m *Manager) AddUnwantedSender(accountID, sender string) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		// Check if already exists
		for _, s := range mem.Unwanted.Senders {
			if s == sender {
				return
			}
		}
		mem.Unwanted.Senders = append(mem.Unwanted.Senders, sender)
	})
}

// RemoveUnwantedSender removes a sender from the unwanted list
func (m *Manager) RemoveUnwantedSender(accountID, sender string) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		for i, s := range mem.Unwanted.Senders {
			if s == sender {
				mem.Unwanted.Senders = append(mem.Unwanted.Senders[:i], mem.Unwanted.Senders[i+1:]...)
				return
			}
		}
	})
}

// AddNote adds a note
func (m *Manager) AddNote(accountID, content, category string) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		note := Note{
			Content:   content,
			CreatedAt: time.Now(),
			Category:  category,
		}
		mem.Notes = append(mem.Notes, note)
	})
}

// RemoveNote removes a note by index
func (m *Manager) RemoveNote(accountID string, index int) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		if index >= 0 && index < len(mem.Notes) {
			mem.Notes = append(mem.Notes[:index], mem.Notes[index+1:]...)
		}
	})
}

// AddPriorityTopic adds a priority topic
func (m *Manager) AddPriorityTopic(accountID, topic string) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		for _, t := range mem.Topics.Priority {
			if t == topic {
				return
			}
		}
		mem.Topics.Priority = append(mem.Topics.Priority, topic)
	})
}

// SetFolderPurpose sets the purpose description for a folder
func (m *Manager) SetFolderPurpose(accountID, folder, purpose string) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		if mem.FolderPurposes == nil {
			mem.FolderPurposes = make(map[string]string)
		}
		mem.FolderPurposes[folder] = purpose
	})
}

// UpdatePreferences updates preferences
func (m *Manager) UpdatePreferences(accountID string, prefs Preferences) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		if prefs.Language != "" {
			mem.Preferences.Language = prefs.Language
		}
		if prefs.SummaryStyle != "" {
			mem.Preferences.SummaryStyle = prefs.SummaryStyle
		}
		if prefs.Timezone != "" {
			mem.Preferences.Timezone = prefs.Timezone
		}
		if prefs.DefaultFolder != "" {
			mem.Preferences.DefaultFolder = prefs.DefaultFolder
		}
		mem.Preferences.AutoMarkRead = prefs.AutoMarkRead
	})
}

// IncrementStat increments a statistic
func (m *Manager) IncrementStat(accountID string, stat string, value int) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		switch stat {
		case "emails_processed":
			mem.Statistics.EmailsProcessed += value
		case "emails_deleted":
			mem.Statistics.EmailsDeleted += value
		case "emails_moved":
			mem.Statistics.EmailsMoved += value
		case "spam_detected":
			mem.Statistics.SpamDetected += value
		}
	})
}

// StartSession marks the start of a new session
func (m *Manager) StartSession(accountID string) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		mem.Statistics.TotalSessions++
		mem.Statistics.LastSessionStart = time.Now()
	})
}

// List returns all account IDs that have memory files
func (m *Manager) List() ([]string, error) {
	entries, err := os.ReadDir(m.directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read memory directory: %w", err)
	}

	var accounts []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			name := entry.Name()
			accounts = append(accounts, name[:len(name)-5]) // Remove .json
		}
	}

	return accounts, nil
}

// Delete deletes memory for an account
func (m *Manager) Delete(accountID string) error {
	m.mu.Lock()
	delete(m.memories, accountID)
	m.mu.Unlock()

	filePath := m.getFilePath(accountID)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete memory file: %w", err)
	}

	return nil
}

func (m *Manager) getFilePath(accountID string) string {
	return filepath.Join(m.directory, accountID+".json")
}

// Search searches memory for a query string
func (m *Manager) Search(accountID, query string) (*SearchResults, error) {
	mem, err := m.GetOrLoad(accountID)
	if err != nil {
		return nil, err
	}
	return mem.Search(query), nil
}

// AddObservation adds an observation to a contact
func (m *Manager) AddObservation(accountID, email, observation string) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		for i, c := range mem.ImportantContacts {
			if c.Email == email {
				// Check if observation already exists
				for _, obs := range c.Observations {
					if obs == observation {
						return
					}
				}
				mem.ImportantContacts[i].Observations = append(mem.ImportantContacts[i].Observations, observation)
				return
			}
		}
	})
}

// RemoveObservation removes an observation from a contact
func (m *Manager) RemoveObservation(accountID, email, observation string) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		for i, c := range mem.ImportantContacts {
			if c.Email == email {
				for j, obs := range c.Observations {
					if obs == observation {
						mem.ImportantContacts[i].Observations = append(
							mem.ImportantContacts[i].Observations[:j],
							mem.ImportantContacts[i].Observations[j+1:]...,
						)
						return
					}
				}
				return
			}
		}
	})
}

// GetContact returns a contact by email
func (m *Manager) GetContact(accountID, email string) (*Contact, error) {
	mem, err := m.GetOrLoad(accountID)
	if err != nil {
		return nil, err
	}
	for _, c := range mem.ImportantContacts {
		if c.Email == email {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("contact '%s' not found", email)
}

// ============================================================================
// Profile Management
// ============================================================================

// TrackMessageSeen records that a message from a sender was seen
// This is called passively during message_list, message_get, search operations
// Uses deduplication to avoid counting same message multiple times
func (m *Manager) TrackMessageSeen(accountID, from string, flags []string) error {
	if from == "" {
		return nil
	}

	// Extract email from "Name <email>" format
	email := extractEmail(from)
	if email == "" {
		return nil
	}

	return m.Update(accountID, func(mem *AccountMemory) {
		if mem.Profile.SenderProfiles == nil {
			mem.Profile.SenderProfiles = make(map[string]*SenderProfile)
		}

		sp, exists := mem.Profile.SenderProfiles[email]
		if !exists {
			sp = &SenderProfile{
				Email: email,
				Name:  extractName(from),
				Stats: SenderStats{
					FirstSeen: time.Now(),
				},
			}
			mem.Profile.SenderProfiles[email] = sp
		}

		sp.Stats.TotalSeen++
		sp.Stats.LastSeen = time.Now()

		// Check for flags
		for _, flag := range flags {
			switch flag {
			case "\\Answered":
				sp.Stats.TotalAnswered++
			case "\\Flagged":
				sp.Stats.TotalFlagged++
			}
		}
	})
}

// TrackMessageSeenWithUID records message seen with deduplication by UID
// This prevents counting the same message multiple times in a session
func (m *Manager) TrackMessageSeenWithUID(accountID, folder string, uid uint32, from string, flags []string) error {
	if from == "" {
		return nil
	}

	// Check for deduplication
	dedupKey := fmt.Sprintf("%s:%s:%d", accountID, folder, uid)
	m.seenMu.RLock()
	alreadySeen := m.seenMessages[dedupKey]
	m.seenMu.RUnlock()

	if alreadySeen {
		// Already tracked this message, skip
		return nil
	}

	// Mark as seen
	m.seenMu.Lock()
	m.seenMessages[dedupKey] = true
	m.seenMu.Unlock()

	// Extract email from "Name <email>" format
	email := extractEmail(from)
	if email == "" {
		return nil
	}

	return m.Update(accountID, func(mem *AccountMemory) {
		if mem.Profile.SenderProfiles == nil {
			mem.Profile.SenderProfiles = make(map[string]*SenderProfile)
		}

		sp, exists := mem.Profile.SenderProfiles[email]
		if !exists {
			sp = &SenderProfile{
				Email: email,
				Name:  extractName(from),
				Stats: SenderStats{
					FirstSeen: time.Now(),
				},
			}
			mem.Profile.SenderProfiles[email] = sp
		}

		sp.Stats.TotalSeen++
		sp.Stats.LastSeen = time.Now()

		// Check for flags - only count once per message
		for _, flag := range flags {
			switch flag {
			case "\\Answered":
				sp.Stats.TotalAnswered++
			case "\\Flagged":
				sp.Stats.TotalFlagged++
			}
		}
	})
}

// ClearSeenCache clears the deduplication cache (call at session start)
func (m *Manager) ClearSeenCache() {
	m.seenMu.Lock()
	m.seenMessages = make(map[string]bool)
	m.seenMu.Unlock()
}

// TrackMessageAnswered records that user replied to a message from sender
// This is called when message_reply is used
func (m *Manager) TrackMessageAnswered(accountID, from string) error {
	if from == "" {
		return nil
	}

	email := extractEmail(from)
	if email == "" {
		return nil
	}

	return m.Update(accountID, func(mem *AccountMemory) {
		if mem.Profile.SenderProfiles == nil {
			mem.Profile.SenderProfiles = make(map[string]*SenderProfile)
		}

		sp, exists := mem.Profile.SenderProfiles[email]
		if !exists {
			sp = &SenderProfile{
				Email: email,
				Name:  extractName(from),
				Stats: SenderStats{
					FirstSeen: time.Now(),
				},
			}
			mem.Profile.SenderProfiles[email] = sp
		}

		sp.Stats.TotalAnswered++
		sp.Stats.LastSeen = time.Now()
	})
}

// GetSenderProfile returns profile for a specific sender
func (m *Manager) GetSenderProfile(accountID, email string) (*SenderProfile, error) {
	mem, err := m.GetOrLoad(accountID)
	if err != nil {
		return nil, err
	}

	if mem.Profile.SenderProfiles == nil {
		return nil, fmt.Errorf("sender '%s' not found", email)
	}

	sp, exists := mem.Profile.SenderProfiles[email]
	if !exists {
		return nil, fmt.Errorf("sender '%s' not found", email)
	}

	return sp, nil
}

// SetSenderProfile sets or updates profile for a sender
func (m *Manager) SetSenderProfile(accountID string, profile *SenderProfile) error {
	if profile.Email == "" {
		return fmt.Errorf("email is required")
	}

	return m.Update(accountID, func(mem *AccountMemory) {
		if mem.Profile.SenderProfiles == nil {
			mem.Profile.SenderProfiles = make(map[string]*SenderProfile)
		}

		existing, exists := mem.Profile.SenderProfiles[profile.Email]
		if exists {
			// Preserve stats when updating
			if profile.Name != "" {
				existing.Name = profile.Name
			}
			if profile.Type != "" {
				existing.Type = profile.Type
			}
			if profile.Importance != "" {
				existing.Importance = profile.Importance
			}
			if profile.Relationship != "" {
				existing.Relationship = profile.Relationship
			}
			if profile.Notes != "" {
				existing.Notes = profile.Notes
			}
		} else {
			if profile.Stats.FirstSeen.IsZero() {
				profile.Stats.FirstSeen = time.Now()
			}
			mem.Profile.SenderProfiles[profile.Email] = profile
		}
	})
}

// RemoveSenderProfile removes a sender profile
func (m *Manager) RemoveSenderProfile(accountID, email string) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		if mem.Profile.SenderProfiles != nil {
			delete(mem.Profile.SenderProfiles, email)
		}
	})
}

// AddImportanceRule adds or updates an importance rule
func (m *Manager) AddImportanceRule(accountID string, rule ImportanceRule) error {
	if rule.Pattern == "" {
		return fmt.Errorf("pattern is required")
	}
	if rule.Action == "" {
		return fmt.Errorf("action is required")
	}

	return m.Update(accountID, func(mem *AccountMemory) {
		// Check if rule with same pattern exists
		for i, r := range mem.Profile.ImportanceRules {
			if r.Pattern == rule.Pattern {
				mem.Profile.ImportanceRules[i] = rule
				return
			}
		}
		mem.Profile.ImportanceRules = append(mem.Profile.ImportanceRules, rule)
	})
}

// RemoveImportanceRule removes a rule by pattern
func (m *Manager) RemoveImportanceRule(accountID, pattern string) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		for i, r := range mem.Profile.ImportanceRules {
			if r.Pattern == pattern {
				mem.Profile.ImportanceRules = append(
					mem.Profile.ImportanceRules[:i],
					mem.Profile.ImportanceRules[i+1:]...,
				)
				return
			}
		}
	})
}

// UpdateProfile updates the account profile metadata
func (m *Manager) UpdateProfile(accountID string, profileType, description, activitySummary string) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		if profileType != "" {
			mem.Profile.Type = profileType
		}
		if description != "" {
			mem.Profile.Description = description
		}
		if activitySummary != "" {
			mem.Profile.ActivitySummary = activitySummary
		}
	})
}

// SetProfileAnalyzed marks the profile as analyzed
func (m *Manager) SetProfileAnalyzed(accountID string, count int) error {
	return m.Update(accountID, func(mem *AccountMemory) {
		mem.Profile.LastAnalyzed = time.Now()
		mem.Profile.AnalyzedCount = count
	})
}

// GetAnalysisPreferences returns analysis preferences with defaults
func (m *Manager) GetAnalysisPreferences(accountID string) (depth int, period int, err error) {
	mem, err := m.GetOrLoad(accountID)
	if err != nil {
		return 1000, 180, err
	}

	depth = mem.Preferences.AnalysisDepth
	if depth <= 0 {
		depth = 1000
	}

	period = mem.Preferences.AnalysisPeriod
	if period <= 0 {
		period = 180
	}

	return depth, period, nil
}

// Helper functions

func extractEmail(from string) string {
	// Handle "Name <email@domain.com>" format
	if start := strings.Index(from, "<"); start != -1 {
		if end := strings.Index(from, ">"); end > start {
			return strings.TrimSpace(from[start+1 : end])
		}
	}
	// Assume it's just an email
	return strings.TrimSpace(from)
}

func extractName(from string) string {
	// Handle "Name <email@domain.com>" format
	if start := strings.Index(from, "<"); start > 0 {
		name := strings.TrimSpace(from[:start])
		// Remove quotes if present
		name = strings.Trim(name, "\"'")
		return name
	}
	return ""
}
