package types

// FolderInfo represents information about a mailbox folder
type FolderInfo struct {
	Name        string   `json:"name"`
	Delimiter   string   `json:"delimiter"`
	Attributes  []string `json:"attributes,omitempty"`
	Messages    uint32   `json:"messages"`
	Recent      uint32   `json:"recent"`
	Unseen      uint32   `json:"unseen"`
	UIDValidity uint32   `json:"uid_validity"`
	UIDNext     uint32   `json:"uid_next"`
}

// FolderListItem represents a folder in the list (without full status)
type FolderListItem struct {
	Name       string   `json:"name"`
	Delimiter  string   `json:"delimiter"`
	Attributes []string `json:"attributes,omitempty"`
	// Special use flags
	IsInbox   bool `json:"is_inbox,omitempty"`
	IsSent    bool `json:"is_sent,omitempty"`
	IsDrafts  bool `json:"is_drafts,omitempty"`
	IsTrash   bool `json:"is_trash,omitempty"`
	IsJunk    bool `json:"is_junk,omitempty"`
	IsArchive bool `json:"is_archive,omitempty"`
}

// Common folder attribute constants
const (
	AttrNoSelect   = "\\Noselect"
	AttrNoInferior = "\\Noinferiors"
	AttrMarked     = "\\Marked"
	AttrUnmarked   = "\\Unmarked"
	AttrHasChildren = "\\HasChildren"
	AttrHasNoChildren = "\\HasNoChildren"

	// Special-use attributes (RFC 6154)
	AttrAll      = "\\All"
	AttrArchive  = "\\Archive"
	AttrDrafts   = "\\Drafts"
	AttrFlagged  = "\\Flagged"
	AttrJunk     = "\\Junk"
	AttrSent     = "\\Sent"
	AttrTrash    = "\\Trash"
	AttrImportant = "\\Important"
)
