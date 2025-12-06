package security

import (
	"fmt"
	"strings"

	"github.com/spetr/mcp-mail/config"
)

// ProtectionManager handles folder protection and read-only mode
type ProtectionManager struct {
	config *config.SecurityConfig
}

// NewProtectionManager creates a new protection manager
func NewProtectionManager(cfg *config.SecurityConfig) *ProtectionManager {
	return &ProtectionManager{
		config: cfg,
	}
}

// CheckReadOnly returns an error if read-only mode is enabled
func (pm *ProtectionManager) CheckReadOnly() error {
	if pm.config.ReadOnly {
		return fmt.Errorf("server is in read-only mode, write operations are disabled")
	}
	return nil
}

// IsReadOnly returns true if read-only mode is enabled
func (pm *ProtectionManager) IsReadOnly() bool {
	return pm.config.ReadOnly
}

// IsFolderProtected checks if a folder is in the protected list
func (pm *ProtectionManager) IsFolderProtected(folder string) bool {
	folderLower := strings.ToLower(folder)
	for _, protected := range pm.config.ProtectedFolders {
		if strings.ToLower(protected) == folderLower {
			return true
		}
		// Also check if folder starts with protected folder path
		// e.g., "INBOX/Subfolder" is protected if "INBOX" is protected
		if strings.HasPrefix(folderLower, strings.ToLower(protected)+"/") {
			return true
		}
	}
	return false
}

// CheckFolderDeletion returns an error if folder cannot be deleted
func (pm *ProtectionManager) CheckFolderDeletion(folder string) error {
	if err := pm.CheckReadOnly(); err != nil {
		return err
	}

	if pm.IsFolderProtected(folder) {
		return fmt.Errorf("folder '%s' is protected and cannot be deleted", folder)
	}

	return nil
}

// GetTrashFolder returns the configured trash folder name
func (pm *ProtectionManager) GetTrashFolder() string {
	if pm.config.TrashFolder != "" {
		return pm.config.TrashFolder
	}
	return "" // Will be auto-detected from IMAP server
}

// GetProtectedFolders returns the list of protected folders
func (pm *ProtectionManager) GetProtectedFolders() []string {
	return pm.config.ProtectedFolders
}
