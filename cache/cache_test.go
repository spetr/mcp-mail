package cache

import (
	"testing"
	"time"

	"github.com/spetr/mcp-mail/types"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Enabled {
		t.Error("expected Enabled to be true by default")
	}
	if cfg.FolderListTTLSec != 300 {
		t.Errorf("expected FolderListTTLSec 300, got %d", cfg.FolderListTTLSec)
	}
	if cfg.FolderInfoTTLSec != 120 {
		t.Errorf("expected FolderInfoTTLSec 120, got %d", cfg.FolderInfoTTLSec)
	}
	if cfg.MessageListTTLSec != 120 {
		t.Errorf("expected MessageListTTLSec 120, got %d", cfg.MessageListTTLSec)
	}
	if cfg.SearchResultsTTLSec != 60 {
		t.Errorf("expected SearchResultsTTLSec 60, got %d", cfg.SearchResultsTTLSec)
	}
	if cfg.TrashFolderTTLSec != 3600 {
		t.Errorf("expected TrashFolderTTLSec 3600, got %d", cfg.TrashFolderTTLSec)
	}
}

func TestConfigMergeDefaults(t *testing.T) {
	cfg := Config{Enabled: true}
	cfg.MergeDefaults()

	if cfg.FolderListTTLSec != 300 {
		t.Errorf("expected FolderListTTLSec 300 after merge, got %d", cfg.FolderListTTLSec)
	}

	// Test that existing values are not overwritten
	cfg2 := Config{Enabled: true, FolderListTTLSec: 600}
	cfg2.MergeDefaults()

	if cfg2.FolderListTTLSec != 600 {
		t.Errorf("expected FolderListTTLSec 600 (not merged), got %d", cfg2.FolderListTTLSec)
	}
}

func TestNewManagerEnabled(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	if !mgr.IsEnabled() {
		t.Error("expected manager to be enabled")
	}
}

func TestNewManagerDisabled(t *testing.T) {
	cfg := Config{Enabled: false}
	mgr := NewManager(cfg)

	if mgr.IsEnabled() {
		t.Error("expected manager to be disabled")
	}

	// Operations on disabled cache should return nil/empty
	if mgr.GetFolderList("acc1") != nil {
		t.Error("expected nil from disabled cache")
	}
}

func TestFolderListCache(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FolderListTTLSec = 1 // Short TTL for testing
	mgr := NewManager(cfg)

	accountID := "test-account"
	folders := []types.FolderListItem{
		{Name: "INBOX", Delimiter: "/"},
		{Name: "Sent", Delimiter: "/"},
	}

	// Should be nil initially
	if mgr.GetFolderList(accountID) != nil {
		t.Error("expected nil before setting")
	}

	// Set and get
	mgr.SetFolderList(accountID, folders)
	result := mgr.GetFolderList(accountID)
	if result == nil {
		t.Fatal("expected non-nil after setting")
	}
	if len(result) != 2 {
		t.Errorf("expected 2 folders, got %d", len(result))
	}
	if result[0].Name != "INBOX" {
		t.Errorf("expected first folder 'INBOX', got %s", result[0].Name)
	}

	// Invalidate
	mgr.InvalidateFolderList(accountID)
	if mgr.GetFolderList(accountID) != nil {
		t.Error("expected nil after invalidation")
	}
}

func TestFolderInfoCache(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	accountID := "test-account"
	folder := "INBOX"
	info := &types.FolderInfo{
		Name:        "INBOX",
		Messages:    100,
		Unseen:      5,
		UIDValidity: 12345,
	}

	// Set and get
	mgr.SetFolderInfo(accountID, folder, info)
	result := mgr.GetFolderInfo(accountID, folder)
	if result == nil {
		t.Fatal("expected non-nil after setting")
	}
	if result.Messages != 100 {
		t.Errorf("expected Messages 100, got %d", result.Messages)
	}
	if result.Unseen != 5 {
		t.Errorf("expected Unseen 5, got %d", result.Unseen)
	}

	// Different folder should be nil
	if mgr.GetFolderInfo(accountID, "Sent") != nil {
		t.Error("expected nil for different folder")
	}
}

func TestMessageListCache(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	accountID := "test-account"
	folder := "INBOX"
	messages := []types.MessageListItem{
		{UID: 1, Subject: "Test 1"},
		{UID: 2, Subject: "Test 2"},
	}

	// Set and get with same limit/offset
	mgr.SetMessageList(accountID, folder, 10, 0, messages)
	result := mgr.GetMessageList(accountID, folder, 10, 0)
	if result == nil {
		t.Fatal("expected non-nil after setting")
	}
	if len(result) != 2 {
		t.Errorf("expected 2 messages, got %d", len(result))
	}

	// Different limit/offset should be nil
	if mgr.GetMessageList(accountID, folder, 20, 0) != nil {
		t.Error("expected nil for different limit")
	}
	if mgr.GetMessageList(accountID, folder, 10, 10) != nil {
		t.Error("expected nil for different offset")
	}
}

func TestSearchResultsCache(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	accountID := "test-account"
	folder := "INBOX"
	criteriaHash := "abc123"
	uids := []uint32{1, 2, 3, 4, 5}

	// Set and get
	mgr.SetSearchResults(accountID, folder, criteriaHash, uids)
	result := mgr.GetSearchResults(accountID, folder, criteriaHash)
	if result == nil {
		t.Fatal("expected non-nil after setting")
	}
	if len(result) != 5 {
		t.Errorf("expected 5 UIDs, got %d", len(result))
	}

	// Different criteria should be nil
	if mgr.GetSearchResults(accountID, folder, "different") != nil {
		t.Error("expected nil for different criteria")
	}
}

func TestTrashFolderCache(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	accountID := "test-account"
	trashFolder := "[Gmail]/Trash"

	// Set and get
	mgr.SetTrashFolder(accountID, trashFolder)
	result := mgr.GetTrashFolder(accountID)
	if result != trashFolder {
		t.Errorf("expected '%s', got '%s'", trashFolder, result)
	}

	// Different account should be empty
	if mgr.GetTrashFolder("other-account") != "" {
		t.Error("expected empty for different account")
	}
}

func TestInvalidateFolder(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	accountID := "test-account"
	folder := "INBOX"

	// Set some data
	mgr.SetFolderInfo(accountID, folder, &types.FolderInfo{Name: folder, Messages: 10})
	mgr.SetMessageList(accountID, folder, 10, 0, []types.MessageListItem{{UID: 1}})
	mgr.SetSearchResults(accountID, folder, "hash", []uint32{1, 2})

	// Verify data exists
	if mgr.GetFolderInfo(accountID, folder) == nil {
		t.Error("expected folder info to exist")
	}

	// Invalidate folder
	mgr.InvalidateFolder(accountID, folder)

	// Verify data is gone
	if mgr.GetFolderInfo(accountID, folder) != nil {
		t.Error("expected folder info to be invalidated")
	}
}

func TestInvalidateFolderContent(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	accountID := "test-account"
	folder := "INBOX"

	// Set some data
	mgr.SetFolderInfo(accountID, folder, &types.FolderInfo{Name: folder, Messages: 10})
	mgr.SetMessageList(accountID, folder, 10, 0, []types.MessageListItem{{UID: 1}})

	// Invalidate folder content
	mgr.InvalidateFolderContent(accountID, folder)

	// Folder info should also be invalidated (message count changed)
	if mgr.GetFolderInfo(accountID, folder) != nil {
		t.Error("expected folder info to be invalidated")
	}
	if mgr.GetMessageList(accountID, folder, 10, 0) != nil {
		t.Error("expected message list to be invalidated")
	}
}

func TestInvalidateAccount(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	accountID := "test-account"

	// Set various data
	mgr.SetFolderList(accountID, []types.FolderListItem{{Name: "INBOX"}})
	mgr.SetFolderInfo(accountID, "INBOX", &types.FolderInfo{Name: "INBOX"})
	mgr.SetTrashFolder(accountID, "Trash")

	// Verify exists
	if mgr.GetFolderList(accountID) == nil {
		t.Error("expected folder list to exist")
	}

	// Invalidate account
	mgr.InvalidateAccount(accountID)

	// All should be gone
	if mgr.GetFolderList(accountID) != nil {
		t.Error("expected folder list to be invalidated")
	}
	if mgr.GetFolderInfo(accountID, "INBOX") != nil {
		t.Error("expected folder info to be invalidated")
	}
	if mgr.GetTrashFolder(accountID) != "" {
		t.Error("expected trash folder to be invalidated")
	}
}

func TestCheckUIDValidity(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	accountID := "test-account"
	folder := "INBOX"

	// First check - should return true (no previous state)
	if !mgr.CheckUIDValidity(accountID, folder, 12345) {
		t.Error("expected true on first check")
	}

	// Same UIDVALIDITY - should return true
	if !mgr.CheckUIDValidity(accountID, folder, 12345) {
		t.Error("expected true with same UIDVALIDITY")
	}

	// Different UIDVALIDITY - should return false and invalidate
	if mgr.CheckUIDValidity(accountID, folder, 99999) {
		t.Error("expected false with different UIDVALIDITY")
	}
}

func TestStats(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	stats := mgr.Stats()
	if stats["enabled"] != 1 {
		t.Error("expected enabled=1 in stats")
	}

	// Add some data
	mgr.SetFolderList("acc1", []types.FolderListItem{{Name: "INBOX"}})
	mgr.SetFolderList("acc2", []types.FolderListItem{{Name: "INBOX"}})

	stats = mgr.Stats()
	if stats["folder_lists"] != 2 {
		t.Errorf("expected folder_lists=2, got %d", stats["folder_lists"])
	}
}

func TestDisabledCacheStats(t *testing.T) {
	cfg := Config{Enabled: false}
	mgr := NewManager(cfg)

	stats := mgr.Stats()
	if stats["enabled"] != 0 {
		t.Error("expected enabled=0 for disabled cache")
	}
}

func TestCacheTTLExpiration(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FolderListTTLSec = 1 // 1 second TTL
	mgr := NewManager(cfg)

	accountID := "test-account"
	folders := []types.FolderListItem{{Name: "INBOX"}}

	mgr.SetFolderList(accountID, folders)

	// Should exist immediately
	if mgr.GetFolderList(accountID) == nil {
		t.Error("expected folder list to exist immediately")
	}

	// Wait for expiration
	time.Sleep(1500 * time.Millisecond)

	// Should be expired
	if mgr.GetFolderList(accountID) != nil {
		t.Error("expected folder list to be expired after TTL")
	}
}
