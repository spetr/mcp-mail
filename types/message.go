package types

import "time"

// MessageEnvelope contains message header information
type MessageEnvelope struct {
	UID         uint32    `json:"uid"`
	MessageID   string    `json:"message_id,omitempty"`
	Subject     string    `json:"subject"`
	From        []Address `json:"from"`
	To          []Address `json:"to"`
	Cc          []Address `json:"cc,omitempty"`
	Bcc         []Address `json:"bcc,omitempty"`
	ReplyTo     []Address `json:"reply_to,omitempty"`
	Date        time.Time `json:"date"`
	Size        uint32    `json:"size"`
	Flags       []string  `json:"flags"`
	InReplyTo   string    `json:"in_reply_to,omitempty"`
	References  []string  `json:"references,omitempty"`
}

// Address represents an email address
type Address struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address"`
}

// MessageBody contains the message body content
type MessageBody struct {
	Text string `json:"text,omitempty"`
	HTML string `json:"html,omitempty"`
}

// MessageAttachment represents an email attachment
type MessageAttachment struct {
	PartID      string `json:"part_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        uint32 `json:"size"`
	Inline      bool   `json:"inline,omitempty"`
}

// Message represents a complete email message
type Message struct {
	MessageEnvelope
	Body        MessageBody         `json:"body"`
	Attachments []MessageAttachment `json:"attachments,omitempty"`
}

// MessageListItem is a lighter version for listing messages
type MessageListItem struct {
	UID     uint32    `json:"uid"`
	Subject string    `json:"subject"`
	From    []Address `json:"from"`
	Date    time.Time `json:"date"`
	Size    uint32    `json:"size"`
	Flags   []string  `json:"flags"`
	HasAttachments bool `json:"has_attachments"`
}

// MessageListOptions specifies options for listing messages
type MessageListOptions struct {
	Folder string `json:"folder"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	// Sort options
	SortBy    string `json:"sort_by,omitempty"`    // date, from, subject, size
	SortOrder string `json:"sort_order,omitempty"` // asc, desc
}

// Common IMAP flags
const (
	FlagSeen     = "\\Seen"
	FlagAnswered = "\\Answered"
	FlagFlagged  = "\\Flagged"
	FlagDeleted  = "\\Deleted"
	FlagDraft    = "\\Draft"
	FlagRecent   = "\\Recent"
)

// Thread represents an email conversation/thread
type Thread struct {
	// ThreadID is typically the Message-ID of the root message
	ThreadID string `json:"thread_id"`
	// Subject is the normalized subject (without Re:/Fwd: prefixes)
	Subject string `json:"subject"`
	// Messages in chronological order (oldest first)
	Messages []ThreadMessage `json:"messages"`
	// TotalCount is the number of messages in the thread
	TotalCount int `json:"total_count"`
	// UnreadCount is the number of unread messages
	UnreadCount int `json:"unread_count"`
	// Participants are all unique senders/recipients in the thread
	Participants []Address `json:"participants"`
	// LastMessageDate is the date of the most recent message
	LastMessageDate string `json:"last_message_date"`
}

// ThreadMessage represents a message within a thread
type ThreadMessage struct {
	UID       uint32    `json:"uid"`
	MessageID string    `json:"message_id"`
	Subject   string    `json:"subject"`
	From      []Address `json:"from"`
	To        []Address `json:"to"`
	Date      string    `json:"date"`
	Flags     []string  `json:"flags"`
	// Snippet is a short preview of the message body
	Snippet string `json:"snippet,omitempty"`
	// IsRoot indicates if this is the first message in the thread
	IsRoot bool `json:"is_root,omitempty"`
}

// ThreadListItem is a summary of a thread for listing
type ThreadListItem struct {
	ThreadID        string    `json:"thread_id"`
	Subject         string    `json:"subject"`
	Participants    []Address `json:"participants"`
	LastMessageDate string    `json:"last_message_date"`
	TotalCount      int       `json:"total_count"`
	UnreadCount     int       `json:"unread_count"`
	// LastMessageFrom is the sender of the most recent message
	LastMessageFrom []Address `json:"last_message_from"`
	// HasAttachments indicates if any message in thread has attachments
	HasAttachments bool `json:"has_attachments"`
}
