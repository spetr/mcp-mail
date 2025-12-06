package smtp

import "time"

// OutgoingMessage represents an email to be sent
type OutgoingMessage struct {
	// Recipients
	To  []string `json:"to"`
	Cc  []string `json:"cc,omitempty"`
	Bcc []string `json:"bcc,omitempty"`

	// Headers
	Subject    string   `json:"subject"`
	InReplyTo  string   `json:"in_reply_to,omitempty"`  // Message-ID for threading
	References []string `json:"references,omitempty"`   // For threading chain

	// Body
	Text string `json:"text,omitempty"` // Plain text body
	HTML string `json:"html,omitempty"` // HTML body (optional)

	// Metadata (set by client)
	From      string    `json:"from,omitempty"` // Set by client from account config
	MessageID string    `json:"message_id,omitempty"`
	Date      time.Time `json:"date,omitempty"`
}

// SendResult contains the result of sending an email
type SendResult struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SMTPConfig contains SMTP server configuration
type SMTPConfig struct {
	Host     string `json:"smtp_host"`
	Port     int    `json:"smtp_port"`
	TLS      bool   `json:"smtp_tls"`
	Username string `json:"smtp_username"`
	Password string `json:"smtp_password"`
}
