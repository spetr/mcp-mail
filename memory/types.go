package memory

import (
	"strings"
	"time"
)

// AccountMemory represents persistent memory for an email account
type AccountMemory struct {
	// AccountID is the unique identifier for this account
	AccountID string `json:"account_id"`
	// LastUpdated is when the memory was last modified
	LastUpdated time.Time `json:"last_updated"`

	// ImportantContacts are people whose emails should be prioritized
	ImportantContacts []Contact `json:"important_contacts,omitempty"`

	// Topics contains categorized topics relevant to this account
	Topics Topics `json:"topics,omitempty"`

	// Unwanted contains rules for identifying unwanted/spam emails
	Unwanted UnwantedRules `json:"unwanted,omitempty"`

	// Preferences contains user preferences for this account
	Preferences Preferences `json:"preferences,omitempty"`

	// Notes are free-form notes about this account
	Notes []Note `json:"notes,omitempty"`

	// FolderPurposes describes what each folder is used for
	FolderPurposes map[string]string `json:"folder_purposes,omitempty"`

	// Statistics tracks usage statistics
	Statistics Statistics `json:"statistics,omitempty"`
}

// Contact represents an important contact
type Contact struct {
	Email        string    `json:"email"`
	Name         string    `json:"name,omitempty"`
	Role         string    `json:"role,omitempty"`        // e.g., "Boss", "Client", "Family"
	Priority     string    `json:"priority,omitempty"`    // "high", "normal", "low"
	Notes        string    `json:"notes,omitempty"`       // Additional notes about this contact
	Observations []string  `json:"observations,omitempty"` // Atomic facts learned about this contact
	AddedAt      time.Time `json:"added_at"`
	LastContact  time.Time `json:"last_contact,omitempty"`
}

// Topics contains categorized topics
type Topics struct {
	// Priority topics that should be flagged/highlighted
	Priority []string `json:"priority,omitempty"`
	// Projects are ongoing projects to track
	Projects []string `json:"projects,omitempty"`
	// WantedNewsletters are newsletters the user wants to receive
	WantedNewsletters []string `json:"wanted_newsletters,omitempty"`
	// CustomCategories allows user-defined categories
	CustomCategories map[string][]string `json:"custom_categories,omitempty"`
}

// UnwantedRules contains rules for identifying unwanted emails
type UnwantedRules struct {
	// Senders are email addresses or patterns to mark as unwanted
	// Supports wildcards: *@domain.com, newsletter@*
	Senders []string `json:"senders,omitempty"`
	// Subjects are subject line patterns to mark as unwanted
	Subjects []string `json:"subjects,omitempty"`
	// BodyKeywords are keywords in body that indicate unwanted mail
	BodyKeywords []string `json:"body_keywords,omitempty"`
	// AutoAction is what to do with matched emails: "mark_spam", "delete", "move_to_folder"
	AutoAction string `json:"auto_action,omitempty"`
	// AutoActionFolder is the folder to move to if auto_action is "move_to_folder"
	AutoActionFolder string `json:"auto_action_folder,omitempty"`
}

// Preferences contains user preferences
type Preferences struct {
	// Language preferred for responses (e.g., "cs", "en")
	Language string `json:"language,omitempty"`
	// SummaryStyle: "brief", "detailed", "bullet_points"
	SummaryStyle string `json:"summary_style,omitempty"`
	// Timezone for date/time formatting
	Timezone string `json:"timezone,omitempty"`
	// AutoMarkRead automatically mark emails as read when fetched
	AutoMarkRead bool `json:"auto_mark_read,omitempty"`
	// DefaultFolder to check first
	DefaultFolder string `json:"default_folder,omitempty"`
}

// Note is a free-form note about the account
type Note struct {
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Category  string    `json:"category,omitempty"` // e.g., "rule", "reminder", "info"
}

// Statistics tracks usage statistics
type Statistics struct {
	EmailsProcessed   int       `json:"emails_processed"`
	EmailsDeleted     int       `json:"emails_deleted"`
	EmailsMoved       int       `json:"emails_moved"`
	SpamDetected      int       `json:"spam_detected"`
	LastCleanup       time.Time `json:"last_cleanup,omitempty"`
	FirstUsed         time.Time `json:"first_used,omitempty"`
	TotalSessions     int       `json:"total_sessions"`
	LastSessionStart  time.Time `json:"last_session_start,omitempty"`
}

// NewAccountMemory creates a new empty memory for an account
func NewAccountMemory(accountID string) *AccountMemory {
	now := time.Now()
	return &AccountMemory{
		AccountID:         accountID,
		LastUpdated:       now,
		ImportantContacts: []Contact{},
		Topics: Topics{
			Priority:         []string{},
			Projects:         []string{},
			WantedNewsletters: []string{},
			CustomCategories: make(map[string][]string),
		},
		Unwanted: UnwantedRules{
			Senders:      []string{},
			Subjects:     []string{},
			BodyKeywords: []string{},
			AutoAction:   "mark_spam",
		},
		Preferences: Preferences{
			Language:      "cs",
			SummaryStyle:  "brief",
			DefaultFolder: "INBOX",
		},
		Notes:          []Note{},
		FolderPurposes: make(map[string]string),
		Statistics: Statistics{
			FirstUsed: now,
		},
	}
}

// MemorySummary is a brief summary of memory for AI context
type MemorySummary struct {
	AccountID            string   `json:"account_id"`
	ImportantContactsCount int    `json:"important_contacts_count"`
	ImportantEmails      []string `json:"important_emails,omitempty"`
	PriorityTopics       []string `json:"priority_topics,omitempty"`
	UnwantedSendersCount int      `json:"unwanted_senders_count"`
	NotesCount           int      `json:"notes_count"`
	RecentNotes          []string `json:"recent_notes,omitempty"`
	Preferences          Preferences `json:"preferences"`
	Hint                 string   `json:"hint"`
}

// SearchResult represents a search match in memory
type SearchResult struct {
	Type    string      `json:"type"`    // "contact", "note", "topic", "unwanted", "folder", "observation"
	Match   string      `json:"match"`   // The matched text
	Context string      `json:"context"` // Additional context
	Data    interface{} `json:"data,omitempty"` // Full data if needed
}

// SearchResults contains all search results
type SearchResults struct {
	Query   string         `json:"query"`
	Total   int            `json:"total"`
	Results []SearchResult `json:"results"`
}

// Search searches across all memory data for a query string
func (m *AccountMemory) Search(query string) *SearchResults {
	query = strings.ToLower(query)
	results := &SearchResults{
		Query:   query,
		Results: []SearchResult{},
	}

	// Search contacts
	for _, c := range m.ImportantContacts {
		if containsIgnoreCase(c.Email, query) || containsIgnoreCase(c.Name, query) ||
			containsIgnoreCase(c.Role, query) || containsIgnoreCase(c.Notes, query) {
			results.Results = append(results.Results, SearchResult{
				Type:    "contact",
				Match:   c.Email,
				Context: formatContact(c),
				Data:    c,
			})
		}
		// Search observations
		for _, obs := range c.Observations {
			if containsIgnoreCase(obs, query) {
				results.Results = append(results.Results, SearchResult{
					Type:    "observation",
					Match:   obs,
					Context: "Contact: " + c.Email,
				})
			}
		}
	}

	// Search notes
	for i, n := range m.Notes {
		if containsIgnoreCase(n.Content, query) || containsIgnoreCase(n.Category, query) {
			results.Results = append(results.Results, SearchResult{
				Type:    "note",
				Match:   n.Content,
				Context: "Category: " + n.Category,
				Data:    map[string]interface{}{"index": i, "note": n},
			})
		}
	}

	// Search topics
	for _, t := range m.Topics.Priority {
		if containsIgnoreCase(t, query) {
			results.Results = append(results.Results, SearchResult{
				Type:    "topic",
				Match:   t,
				Context: "Priority topic",
			})
		}
	}
	for _, t := range m.Topics.Projects {
		if containsIgnoreCase(t, query) {
			results.Results = append(results.Results, SearchResult{
				Type:    "topic",
				Match:   t,
				Context: "Project",
			})
		}
	}

	// Search unwanted senders
	for _, s := range m.Unwanted.Senders {
		if containsIgnoreCase(s, query) {
			results.Results = append(results.Results, SearchResult{
				Type:    "unwanted",
				Match:   s,
				Context: "Unwanted sender",
			})
		}
	}
	for _, s := range m.Unwanted.Subjects {
		if containsIgnoreCase(s, query) {
			results.Results = append(results.Results, SearchResult{
				Type:    "unwanted",
				Match:   s,
				Context: "Unwanted subject pattern",
			})
		}
	}

	// Search folder purposes
	for folder, purpose := range m.FolderPurposes {
		if containsIgnoreCase(folder, query) || containsIgnoreCase(purpose, query) {
			results.Results = append(results.Results, SearchResult{
				Type:    "folder",
				Match:   folder,
				Context: purpose,
			})
		}
	}

	results.Total = len(results.Results)
	return results
}

// helper functions
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func formatContact(c Contact) string {
	parts := []string{c.Email}
	if c.Name != "" {
		parts = append(parts, c.Name)
	}
	if c.Role != "" {
		parts = append(parts, "("+c.Role+")")
	}
	if c.Priority != "" && c.Priority != "normal" {
		parts = append(parts, "["+c.Priority+"]")
	}
	return strings.Join(parts, " ")
}

// GetSummary returns a brief summary for AI context
func (m *AccountMemory) GetSummary() *MemorySummary {
	summary := &MemorySummary{
		AccountID:              m.AccountID,
		ImportantContactsCount: len(m.ImportantContacts),
		PriorityTopics:         m.Topics.Priority,
		UnwantedSendersCount:   len(m.Unwanted.Senders),
		NotesCount:             len(m.Notes),
		Preferences:            m.Preferences,
		Hint:                   "You can update this memory using memory_* tools. Add important contacts, spam rules, or notes as you learn user preferences.",
	}

	// Add important contact emails
	for _, c := range m.ImportantContacts {
		summary.ImportantEmails = append(summary.ImportantEmails, c.Email)
	}

	// Add recent notes (last 5)
	for i := len(m.Notes) - 1; i >= 0 && len(summary.RecentNotes) < 5; i-- {
		summary.RecentNotes = append(summary.RecentNotes, m.Notes[i].Content)
	}

	return summary
}
