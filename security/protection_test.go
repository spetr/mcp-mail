package security

import (
	"testing"

	"github.com/spetr/mcp-mail/config"
)

func TestNewProtectionManager(t *testing.T) {
	cfg := &config.SecurityConfig{
		MaxBulkOperations: 500,
		ProtectedFolders:  []string{"INBOX", "Sent"},
		TrashFolder:       "Trash",
		ReadOnly:          false,
	}

	pm := NewProtectionManager(cfg)
	if pm == nil {
		t.Fatal("expected non-nil ProtectionManager")
	}
}

func TestCheckReadOnly(t *testing.T) {
	tests := []struct {
		name      string
		readOnly  bool
		wantError bool
	}{
		{"read-write mode", false, false},
		{"read-only mode", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.SecurityConfig{ReadOnly: tt.readOnly}
			pm := NewProtectionManager(cfg)

			err := pm.CheckReadOnly()
			if tt.wantError && err == nil {
				t.Error("expected error for read-only mode")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestIsReadOnly(t *testing.T) {
	cfg := &config.SecurityConfig{ReadOnly: true}
	pm := NewProtectionManager(cfg)

	if !pm.IsReadOnly() {
		t.Error("expected IsReadOnly to return true")
	}

	cfg2 := &config.SecurityConfig{ReadOnly: false}
	pm2 := NewProtectionManager(cfg2)

	if pm2.IsReadOnly() {
		t.Error("expected IsReadOnly to return false")
	}
}

func TestIsFolderProtected(t *testing.T) {
	cfg := &config.SecurityConfig{
		ProtectedFolders: []string{"INBOX", "Sent", "Important"},
	}
	pm := NewProtectionManager(cfg)

	tests := []struct {
		folder    string
		protected bool
	}{
		{"INBOX", true},
		{"inbox", true}, // case insensitive
		{"INBOX/Subfolder", true}, // subfolder of protected
		{"Sent", true},
		{"Important", true},
		{"Trash", false},
		{"Archive", false},
		{"Drafts", false},
	}

	for _, tt := range tests {
		t.Run(tt.folder, func(t *testing.T) {
			result := pm.IsFolderProtected(tt.folder)
			if result != tt.protected {
				t.Errorf("IsFolderProtected(%s) = %v, want %v", tt.folder, result, tt.protected)
			}
		})
	}
}

func TestIsFolderProtectedSubfolders(t *testing.T) {
	cfg := &config.SecurityConfig{
		ProtectedFolders: []string{"INBOX"},
	}
	pm := NewProtectionManager(cfg)

	// Subfolders of protected folders should also be protected
	if !pm.IsFolderProtected("INBOX/Work") {
		t.Error("expected INBOX/Work to be protected")
	}
	if !pm.IsFolderProtected("inbox/personal") {
		t.Error("expected inbox/personal to be protected (case insensitive)")
	}

	// But folders that just start with INBOX should not be protected
	// (e.g., "INBOXES" is not a subfolder of "INBOX")
	if pm.IsFolderProtected("INBOXES") {
		t.Error("expected INBOXES to NOT be protected")
	}
}

func TestCheckFolderDeletion(t *testing.T) {
	tests := []struct {
		name      string
		readOnly  bool
		protected []string
		folder    string
		wantError bool
	}{
		{
			name:      "normal folder in read-write mode",
			readOnly:  false,
			protected: []string{"INBOX"},
			folder:    "Trash",
			wantError: false,
		},
		{
			name:      "protected folder in read-write mode",
			readOnly:  false,
			protected: []string{"INBOX"},
			folder:    "INBOX",
			wantError: true,
		},
		{
			name:      "normal folder in read-only mode",
			readOnly:  true,
			protected: []string{"INBOX"},
			folder:    "Trash",
			wantError: true,
		},
		{
			name:      "protected subfolder",
			readOnly:  false,
			protected: []string{"INBOX"},
			folder:    "INBOX/Important",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.SecurityConfig{
				ReadOnly:         tt.readOnly,
				ProtectedFolders: tt.protected,
			}
			pm := NewProtectionManager(cfg)

			err := pm.CheckFolderDeletion(tt.folder)
			if tt.wantError && err == nil {
				t.Error("expected error")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetTrashFolder(t *testing.T) {
	// With configured trash folder
	cfg := &config.SecurityConfig{TrashFolder: "[Gmail]/Trash"}
	pm := NewProtectionManager(cfg)

	if pm.GetTrashFolder() != "[Gmail]/Trash" {
		t.Errorf("expected '[Gmail]/Trash', got '%s'", pm.GetTrashFolder())
	}

	// Without configured trash folder
	cfg2 := &config.SecurityConfig{}
	pm2 := NewProtectionManager(cfg2)

	if pm2.GetTrashFolder() != "" {
		t.Errorf("expected empty string, got '%s'", pm2.GetTrashFolder())
	}
}

func TestGetProtectedFolders(t *testing.T) {
	cfg := &config.SecurityConfig{
		ProtectedFolders: []string{"INBOX", "Sent", "Important"},
	}
	pm := NewProtectionManager(cfg)

	folders := pm.GetProtectedFolders()
	if len(folders) != 3 {
		t.Errorf("expected 3 folders, got %d", len(folders))
	}

	// Check that all folders are present
	expected := map[string]bool{"INBOX": true, "Sent": true, "Important": true}
	for _, f := range folders {
		if !expected[f] {
			t.Errorf("unexpected folder in list: %s", f)
		}
	}
}

func TestEmptyProtectedFolders(t *testing.T) {
	cfg := &config.SecurityConfig{
		ProtectedFolders: []string{},
	}
	pm := NewProtectionManager(cfg)

	// Nothing should be protected
	if pm.IsFolderProtected("INBOX") {
		t.Error("expected INBOX to NOT be protected with empty list")
	}
	if pm.IsFolderProtected("Sent") {
		t.Error("expected Sent to NOT be protected with empty list")
	}

	// Deletion should be allowed
	if err := pm.CheckFolderDeletion("INBOX"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}
