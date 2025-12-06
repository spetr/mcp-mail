package types

// AccountConfig represents IMAP account configuration
type AccountConfig struct {
	// IMAP configuration
	ID       string `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	TLS      bool   `json:"tls"`
	Username string `json:"username"`
	Password string `json:"password"`

	// SMTP configuration (optional - for sending emails)
	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`     // default 587 (STARTTLS) or 465 (TLS)
	SMTPTLS      bool   `json:"smtp_tls,omitempty"`      // use direct TLS (port 465)
	SMTPUsername string `json:"smtp_username,omitempty"` // defaults to Username
	SMTPPassword string `json:"smtp_password,omitempty"` // defaults to Password
	FromAddress  string `json:"from_address,omitempty"`  // defaults to Username

	// Security - secret for HMAC-based safe_id generation (auto-generated)
	Secret string `json:"secret,omitempty"`
}

// AccountStatus represents the connection status of an account
type AccountStatus struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
	Error     string `json:"error,omitempty"`
	// Server capabilities when connected
	Capabilities []string `json:"capabilities,omitempty"`
}

// AccountInfo contains detailed information about a connected account
type AccountInfo struct {
	AccountStatus
	TotalFolders  int    `json:"total_folders"`
	TotalMessages int    `json:"total_messages"`
	UnreadCount   int    `json:"unread_count"`
	CurrentFolder string `json:"current_folder,omitempty"`
}
