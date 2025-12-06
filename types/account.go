package types

// AccountConfig represents IMAP account configuration
type AccountConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	TLS      bool   `json:"tls"`
	Username string `json:"username"`
	Password string `json:"password"`
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
