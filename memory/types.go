package memory

import (
	"fmt"
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

	// Profile contains AI-learned profile of the account
	Profile AccountProfile `json:"profile,omitempty"`
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

	// Analysis preferences
	AnalysisDepth  int  `json:"analysis_depth,omitempty"`  // max messages to analyze (default 1000)
	AnalysisPeriod int  `json:"analysis_period,omitempty"` // days to look back (default 180)
	AutoAnalyze    bool `json:"auto_analyze,omitempty"`    // auto-analyze on connect
	HintsEnabled   *bool `json:"hints_enabled,omitempty"`  // show hints (default true)
}

// Note is a free-form note about the account
type Note struct {
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Category  string    `json:"category,omitempty"` // e.g., "rule", "reminder", "info"
}

// Statistics tracks usage statistics
type Statistics struct {
	EmailsProcessed  int       `json:"emails_processed"`
	EmailsDeleted    int       `json:"emails_deleted"`
	EmailsMoved      int       `json:"emails_moved"`
	EmailsSent       int       `json:"emails_sent"`
	SpamDetected     int       `json:"spam_detected"`
	LastCleanup      time.Time `json:"last_cleanup,omitempty"`
	FirstUsed        time.Time `json:"first_used,omitempty"`
	TotalSessions    int       `json:"total_sessions"`
	LastSessionStart time.Time `json:"last_session_start,omitempty"`
}

// AccountProfile contains AI-learned profile of the account
type AccountProfile struct {
	// Type of account: "personal", "work", "mixed", "newsletters"
	Type string `json:"type,omitempty"`
	// Description is AI-generated description of the account
	Description string `json:"description,omitempty"`
	// SenderProfiles contains learned information about senders
	SenderProfiles map[string]*SenderProfile `json:"sender_profiles,omitempty"`
	// ImportanceRules are learned rules for email importance
	ImportanceRules []ImportanceRule `json:"importance_rules,omitempty"`
	// ActivitySummary describes user's email activity patterns
	ActivitySummary string `json:"activity_summary,omitempty"`
	// LastAnalyzed is when deep analysis was last performed
	LastAnalyzed time.Time `json:"last_analyzed,omitempty"`
	// AnalyzedCount is how many messages were analyzed
	AnalyzedCount int `json:"analyzed_count,omitempty"`
}

// SenderProfile contains learned information about a sender
type SenderProfile struct {
	Email        string      `json:"email"`
	Name         string      `json:"name,omitempty"`
	Type         string      `json:"type,omitempty"`         // "person", "newsletter", "notification", "service", "unknown"
	Importance   string      `json:"importance,omitempty"`   // "high", "normal", "low", "ignore"
	Relationship string      `json:"relationship,omitempty"` // "colleague", "family", "friend", "vendor", "support"
	Notes        string      `json:"notes,omitempty"`
	Stats        SenderStats `json:"stats"`
}

// SenderStats contains statistics about a sender (passively collected)
type SenderStats struct {
	TotalSeen     int       `json:"total_seen"`
	TotalAnswered int       `json:"total_answered"`
	TotalFlagged  int       `json:"total_flagged"`
	LastSeen      time.Time `json:"last_seen,omitempty"`
	FirstSeen     time.Time `json:"first_seen,omitempty"`
}

// ImportanceRule defines a rule for determining email importance
type ImportanceRule struct {
	// Pattern is a match pattern: "from:*@company.com", "subject:urgent", "to:team@*"
	Pattern string `json:"pattern"`
	// Action: "high_priority", "needs_response", "normal", "low_priority", "ignore"
	Action string `json:"action"`
	// Reason explains why this rule exists
	Reason string `json:"reason,omitempty"`
	// Confidence is 0-100 indicating how confident the rule is
	Confidence int `json:"confidence,omitempty"`
	// LearnedFrom: "response_pattern", "flag_pattern", "manual", "analysis"
	LearnedFrom string `json:"learned_from,omitempty"`
}

// NewAccountMemory creates a new empty memory for an account
func NewAccountMemory(accountID string) *AccountMemory {
	now := time.Now()
	hintsEnabled := true
	return &AccountMemory{
		AccountID:         accountID,
		LastUpdated:       now,
		ImportantContacts: []Contact{},
		Topics: Topics{
			Priority:          []string{},
			Projects:          []string{},
			WantedNewsletters: []string{},
			CustomCategories:  make(map[string][]string),
		},
		Unwanted: UnwantedRules{
			Senders:      []string{},
			Subjects:     []string{},
			BodyKeywords: []string{},
			AutoAction:   "mark_spam",
		},
		Preferences: Preferences{
			Language:       "cs",
			SummaryStyle:   "brief",
			DefaultFolder:  "INBOX",
			AnalysisDepth:  1000,
			AnalysisPeriod: 180,
			HintsEnabled:   &hintsEnabled,
		},
		Notes:          []Note{},
		FolderPurposes: make(map[string]string),
		Statistics: Statistics{
			FirstUsed: now,
		},
		Profile: AccountProfile{
			SenderProfiles:  make(map[string]*SenderProfile),
			ImportanceRules: []ImportanceRule{},
		},
	}
}

// MemorySummary is a brief summary of memory for AI context
type MemorySummary struct {
	AccountID              string          `json:"account_id"`
	ImportantContactsCount int             `json:"important_contacts_count"`
	ImportantEmails        []string        `json:"important_emails,omitempty"`
	PriorityTopics         []string        `json:"priority_topics,omitempty"`
	UnwantedSendersCount   int             `json:"unwanted_senders_count"`
	NotesCount             int             `json:"notes_count"`
	RecentNotes            []string        `json:"recent_notes,omitempty"`
	Preferences            Preferences     `json:"preferences"`
	Profile                *ProfileSummary `json:"profile,omitempty"`
	Hints                  []string        `json:"hints,omitempty"`
}

// ProfileSummary is a condensed profile for AI context
type ProfileSummary struct {
	Type            string              `json:"type,omitempty"`
	Description     string              `json:"description,omitempty"`
	KeySenders      map[string][]string `json:"key_senders,omitempty"`      // importance -> []emails
	TopSenders      []string            `json:"top_senders,omitempty"`      // most frequent senders
	Rules           []string            `json:"rules,omitempty"`            // human-readable rules
	ActivitySummary string              `json:"activity_summary,omitempty"`
	LastAnalyzed    string              `json:"last_analyzed,omitempty"`
	NeedsRefresh    bool                `json:"needs_refresh,omitempty"`
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
// This is optimized to minimize context size while providing useful information
func (m *AccountMemory) GetSummary() *MemorySummary {
	summary := &MemorySummary{
		AccountID:              m.AccountID,
		ImportantContactsCount: len(m.ImportantContacts),
		UnwantedSendersCount:   len(m.Unwanted.Senders),
		NotesCount:             len(m.Notes),
		Preferences:            m.Preferences,
	}

	// Add only top 5 priority topics
	if len(m.Topics.Priority) > 5 {
		summary.PriorityTopics = m.Topics.Priority[:5]
	} else {
		summary.PriorityTopics = m.Topics.Priority
	}

	// Add important contact emails (max 10)
	for i, c := range m.ImportantContacts {
		if i >= 10 {
			break
		}
		summary.ImportantEmails = append(summary.ImportantEmails, c.Email)
	}

	// Add recent notes (last 3, truncated)
	for i := len(m.Notes) - 1; i >= 0 && len(summary.RecentNotes) < 3; i-- {
		note := m.Notes[i].Content
		if len(note) > 100 {
			note = note[:100] + "..."
		}
		summary.RecentNotes = append(summary.RecentNotes, note)
	}

	// Add profile summary if profile exists
	if m.Profile.Type != "" || m.Profile.Description != "" || len(m.Profile.SenderProfiles) > 0 {
		summary.Profile = m.GetProfileSummary()
	}

	// Generate hints (limited)
	summary.Hints = m.GenerateHints()

	return summary
}

// GetProfileSummary returns a condensed profile for AI context
// Optimized to return only the most important data
func (m *AccountMemory) GetProfileSummary() *ProfileSummary {
	ps := &ProfileSummary{
		Type:            m.Profile.Type,
		Description:     m.Profile.Description,
		ActivitySummary: m.Profile.ActivitySummary,
		KeySenders:      make(map[string][]string),
	}

	// Group senders by importance - only high and ignore are useful for context
	// Limit each category to 5 entries
	for email, sp := range m.Profile.SenderProfiles {
		if sp.Importance == "high" || sp.Importance == "ignore" {
			if len(ps.KeySenders[sp.Importance]) < 5 {
				ps.KeySenders[sp.Importance] = append(ps.KeySenders[sp.Importance], email)
			}
		}
	}

	// Top 5 senders by volume (only if they have significant activity)
	type senderCount struct {
		email string
		count int
	}
	var topSenders []senderCount
	for email, sp := range m.Profile.SenderProfiles {
		if sp.Stats.TotalSeen >= 3 {
			topSenders = append(topSenders, senderCount{email, sp.Stats.TotalSeen})
		}
	}
	// Simple sort
	for i := 0; i < len(topSenders); i++ {
		for j := i + 1; j < len(topSenders); j++ {
			if topSenders[j].count > topSenders[i].count {
				topSenders[i], topSenders[j] = topSenders[j], topSenders[i]
			}
		}
	}
	for i := 0; i < len(topSenders) && i < 5; i++ {
		ps.TopSenders = append(ps.TopSenders, topSenders[i].email)
	}

	// Convert rules to human-readable format (max 5)
	for i, rule := range m.Profile.ImportanceRules {
		if i >= 5 {
			break
		}
		ruleStr := rule.Pattern + " → " + rule.Action
		ps.Rules = append(ps.Rules, ruleStr)
	}

	// Check if refresh needed
	if !m.Profile.LastAnalyzed.IsZero() {
		ps.LastAnalyzed = m.Profile.LastAnalyzed.Format("2006-01-02")
		daysSinceAnalysis := int(time.Since(m.Profile.LastAnalyzed).Hours() / 24)
		ps.NeedsRefresh = daysSinceAnalysis > 30
	} else {
		ps.NeedsRefresh = true
	}

	return ps
}

// GenerateHints generates contextual hints for AI
func (m *AccountMemory) GenerateHints() []string {
	// Check if hints are disabled
	if m.Preferences.HintsEnabled != nil && !*m.Preferences.HintsEnabled {
		return nil
	}

	var hints []string

	// =========================================================================
	// Profile and Analysis Hints
	// =========================================================================

	// Profile hints
	if m.Profile.LastAnalyzed.IsZero() {
		hints = append(hints, "⚠ PROFILE: Account has no profile. Run account_analyze to create one.")
	} else {
		daysSinceAnalysis := int(time.Since(m.Profile.LastAnalyzed).Hours() / 24)
		if daysSinceAnalysis > 30 {
			hints = append(hints, "💡 PROFILE: Profile is older than 30 days. Consider running account_analyze to refresh.")
		}
	}

	// High response rate senders without classification
	unclassifiedHighResponse := 0
	for _, sp := range m.Profile.SenderProfiles {
		if sp.Importance == "" && sp.Stats.TotalSeen >= 5 {
			if sp.Stats.TotalSeen > 0 {
				rate := float64(sp.Stats.TotalAnswered) / float64(sp.Stats.TotalSeen)
				if rate >= 0.7 {
					unclassifiedHighResponse++
				}
			}
		}
	}
	if unclassifiedHighResponse > 0 {
		hints = append(hints, fmt.Sprintf("💡 CLASSIFY: %d senders have high response rate but are not classified. Use memory_set_sender to classify them.", unclassifiedHighResponse))
	}

	// Memory tools hint if no profile
	if m.Profile.Type == "" && m.Profile.Description == "" {
		hints = append(hints, "ℹ REMEMBER: You can update profile using memory_update_profile, memory_set_sender and memory_add_rule as you learn about the user's email habits.")
	}

	// =========================================================================
	// Important Information Reminder
	// =========================================================================

	// Remind AI to remember important information
	hints = append(hints, "📝 IMPORTANT: When user says an email/sender is important, remember to keep, or needs attention - use memory_set_sender(importance='high') or memory_add_contact to save this. When user says to ignore something - use memory_set_sender(importance='ignore') or memory_add_unwanted.")

	// =========================================================================
	// Memory Size Check - Rationalization Hints
	// =========================================================================

	memoryStats := m.GetMemoryStats()

	// Check if memory is getting large and needs rationalization
	if memoryStats.TotalSenderProfiles > 500 {
		lowActivityCount := 0
		for _, sp := range m.Profile.SenderProfiles {
			// Senders with no activity in 90+ days and low importance
			if sp.Stats.TotalSeen < 3 && sp.Importance == "" {
				if time.Since(sp.Stats.LastSeen) > 90*24*time.Hour {
					lowActivityCount++
				}
			}
		}
		if lowActivityCount > 100 {
			hints = append(hints, fmt.Sprintf("🧹 CLEANUP: Memory has %d sender profiles, %d are inactive (no activity 90+ days, <3 emails). Consider removing stale profiles using memory_remove_sender.", memoryStats.TotalSenderProfiles, lowActivityCount))
		}
	}

	if memoryStats.TotalNotes > 50 {
		hints = append(hints, fmt.Sprintf("🧹 CLEANUP: Memory has %d notes. Consider consolidating or removing outdated notes.", memoryStats.TotalNotes))
	}

	if memoryStats.TotalRules > 30 {
		hints = append(hints, fmt.Sprintf("🧹 CLEANUP: Memory has %d importance rules. Consider reviewing and consolidating overlapping rules.", memoryStats.TotalRules))
	}

	// Overall memory size warning
	if memoryStats.TotalSenderProfiles > 1000 || memoryStats.TotalContacts > 200 {
		hints = append(hints, "⚠ MEMORY: Memory is getting large. Review and rationalize: remove inactive senders, consolidate rules, archive old notes.")
	}

	return hints
}

// MemoryStats contains statistics about memory size
type MemoryStats struct {
	TotalContacts       int `json:"total_contacts"`
	TotalNotes          int `json:"total_notes"`
	TotalUnwantedRules  int `json:"total_unwanted_rules"`
	TotalSenderProfiles int `json:"total_sender_profiles"`
	TotalRules          int `json:"total_importance_rules"`
	TotalFolderPurposes int `json:"total_folder_purposes"`
}

// GetMemoryStats returns statistics about memory size
func (m *AccountMemory) GetMemoryStats() MemoryStats {
	return MemoryStats{
		TotalContacts:       len(m.ImportantContacts),
		TotalNotes:          len(m.Notes),
		TotalUnwantedRules:  len(m.Unwanted.Senders) + len(m.Unwanted.Subjects) + len(m.Unwanted.BodyKeywords),
		TotalSenderProfiles: len(m.Profile.SenderProfiles),
		TotalRules:          len(m.Profile.ImportanceRules),
		TotalFolderPurposes: len(m.FolderPurposes),
	}
}

// PruneResult contains the result of pruning operation
type PruneResult struct {
	RemovedSenders int `json:"removed_senders"`
	RemovedNotes   int `json:"removed_notes"`
	TotalBefore    int `json:"total_before"`
	TotalAfter     int `json:"total_after"`
}

// Prune removes stale/unimportant data from memory
// Removes sender profiles that:
// - Have no classification (importance, relationship, notes)
// - Have low activity (< 3 emails)
// - Haven't been seen in 90+ days
func (m *AccountMemory) Prune() PruneResult {
	result := PruneResult{
		TotalBefore: len(m.Profile.SenderProfiles),
	}

	cutoff := time.Now().AddDate(0, 0, -90) // 90 days ago

	// Prune sender profiles
	for email, sp := range m.Profile.SenderProfiles {
		// Keep if has user classification
		if sp.Importance != "" || sp.Relationship != "" || sp.Notes != "" {
			continue
		}

		// Keep if has significant activity
		if sp.Stats.TotalSeen >= 3 || sp.Stats.TotalAnswered > 0 || sp.Stats.TotalFlagged > 0 {
			continue
		}

		// Remove if last seen is before cutoff
		if sp.Stats.LastSeen.Before(cutoff) {
			delete(m.Profile.SenderProfiles, email)
			result.RemovedSenders++
		}
	}

	result.TotalAfter = len(m.Profile.SenderProfiles)
	return result
}
