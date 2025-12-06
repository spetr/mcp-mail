package types

import "time"

// SearchCriteria represents IMAP search criteria
type SearchCriteria struct {
	// Text searches
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Subject string `json:"subject,omitempty"`
	Body    string `json:"body,omitempty"`
	Text    string `json:"text,omitempty"` // Search in headers and body

	// Date filters
	Since  *time.Time `json:"since,omitempty"`
	Before *time.Time `json:"before,omitempty"`
	On     *time.Time `json:"on,omitempty"`

	// Flag filters
	Seen     *bool `json:"seen,omitempty"`
	Answered *bool `json:"answered,omitempty"`
	Flagged  *bool `json:"flagged,omitempty"`
	Deleted  *bool `json:"deleted,omitempty"`
	Draft    *bool `json:"draft,omitempty"`

	// Size filters (in bytes)
	Larger  uint32 `json:"larger,omitempty"`
	Smaller uint32 `json:"smaller,omitempty"`

	// UID range
	UIDFrom uint32 `json:"uid_from,omitempty"`
	UIDto   uint32 `json:"uid_to,omitempty"`

	// Combine with OR (default is AND)
	Or []SearchCriteria `json:"or,omitempty"`

	// Negate
	Not *SearchCriteria `json:"not,omitempty"`
}

// SearchResult contains search results
type SearchResult struct {
	UIDs  []uint32 `json:"uids"`
	Total int      `json:"total"`
}

// QuickSearch types
type QuickSearchType string

const (
	QuickSearchUnread    QuickSearchType = "unread"
	QuickSearchFlagged   QuickSearchType = "flagged"
	QuickSearchToday     QuickSearchType = "today"
	QuickSearchThisWeek  QuickSearchType = "this_week"
	QuickSearchWithAttachments QuickSearchType = "with_attachments"
)
