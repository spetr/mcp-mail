package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	if mgr.GetDirectory() != tempDir {
		t.Errorf("expected directory %s, got %s", tempDir, mgr.GetDirectory())
	}
}

func TestNewManagerDefaultDirectory(t *testing.T) {
	// Test with empty directory - should use default
	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("failed to create manager with default dir: %v", err)
	}

	home, _ := os.UserHomeDir()
	expectedDir := filepath.Join(home, ".mcp-imap", "memory")
	if mgr.GetDirectory() != expectedDir {
		t.Errorf("expected directory %s, got %s", expectedDir, mgr.GetDirectory())
	}

	// Cleanup
	os.RemoveAll(expectedDir)
}

func TestNewAccountMemory(t *testing.T) {
	mem := NewAccountMemory("test-account")

	if mem.AccountID != "test-account" {
		t.Errorf("expected AccountID 'test-account', got %s", mem.AccountID)
	}
	if mem.Preferences.Language != "cs" {
		t.Errorf("expected default language 'cs', got %s", mem.Preferences.Language)
	}
	if mem.Preferences.DefaultFolder != "INBOX" {
		t.Errorf("expected default folder 'INBOX', got %s", mem.Preferences.DefaultFolder)
	}
	if mem.Unwanted.AutoAction != "mark_spam" {
		t.Errorf("expected auto_action 'mark_spam', got %s", mem.Unwanted.AutoAction)
	}
}

func TestLoadAndSave(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	// Load creates new memory if not exists
	mem, err := mgr.Load(accountID)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	if mem.AccountID != accountID {
		t.Errorf("expected AccountID %s, got %s", accountID, mem.AccountID)
	}

	// Modify and save immediately (using SaveNow for tests)
	mem.Preferences.Language = "en"
	if err := mgr.SaveNow(accountID); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(tempDir, accountID+".json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("expected memory file to exist")
	}

	// Create new manager and load
	mgr2, _ := NewManager(tempDir)
	mem2, err := mgr2.Load(accountID)
	if err != nil {
		t.Fatalf("failed to load in new manager: %v", err)
	}
	if mem2.Preferences.Language != "en" {
		t.Errorf("expected language 'en', got %s", mem2.Preferences.Language)
	}
}

func TestGetOrLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	// First call should create new
	mem1, err := mgr.GetOrLoad(accountID)
	if err != nil {
		t.Fatalf("failed to get or load: %v", err)
	}

	// Second call should return same instance
	mem2, err := mgr.GetOrLoad(accountID)
	if err != nil {
		t.Fatalf("failed to get or load: %v", err)
	}

	if mem1 != mem2 {
		t.Error("expected same instance on second call")
	}
}

func TestAddContact(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	contact := Contact{
		Email:    "boss@company.com",
		Name:     "My Boss",
		Role:     "Boss",
		Priority: "high",
	}

	if err := mgr.AddContact(accountID, contact); err != nil {
		t.Fatalf("failed to add contact: %v", err)
	}

	mem, _ := mgr.GetOrLoad(accountID)
	if len(mem.ImportantContacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(mem.ImportantContacts))
	}
	if mem.ImportantContacts[0].Email != "boss@company.com" {
		t.Errorf("expected email 'boss@company.com', got %s", mem.ImportantContacts[0].Email)
	}

	// Update existing contact
	contact.Name = "My Updated Boss"
	if err := mgr.AddContact(accountID, contact); err != nil {
		t.Fatalf("failed to update contact: %v", err)
	}

	mem, _ = mgr.GetOrLoad(accountID)
	if len(mem.ImportantContacts) != 1 {
		t.Errorf("expected still 1 contact after update, got %d", len(mem.ImportantContacts))
	}
	if mem.ImportantContacts[0].Name != "My Updated Boss" {
		t.Errorf("expected updated name, got %s", mem.ImportantContacts[0].Name)
	}
}

func TestRemoveContact(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	// Add two contacts
	mgr.AddContact(accountID, Contact{Email: "a@test.com"})
	mgr.AddContact(accountID, Contact{Email: "b@test.com"})

	// Remove first
	if err := mgr.RemoveContact(accountID, "a@test.com"); err != nil {
		t.Fatalf("failed to remove contact: %v", err)
	}

	mem, _ := mgr.GetOrLoad(accountID)
	if len(mem.ImportantContacts) != 1 {
		t.Errorf("expected 1 contact after removal, got %d", len(mem.ImportantContacts))
	}
	if mem.ImportantContacts[0].Email != "b@test.com" {
		t.Errorf("expected remaining contact 'b@test.com', got %s", mem.ImportantContacts[0].Email)
	}
}

func TestAddUnwantedSender(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	if err := mgr.AddUnwantedSender(accountID, "spam@example.com"); err != nil {
		t.Fatalf("failed to add unwanted sender: %v", err)
	}

	mem, _ := mgr.GetOrLoad(accountID)
	if len(mem.Unwanted.Senders) != 1 {
		t.Fatalf("expected 1 unwanted sender, got %d", len(mem.Unwanted.Senders))
	}

	// Adding duplicate should not create duplicate
	if err := mgr.AddUnwantedSender(accountID, "spam@example.com"); err != nil {
		t.Fatalf("failed to add duplicate: %v", err)
	}

	mem, _ = mgr.GetOrLoad(accountID)
	if len(mem.Unwanted.Senders) != 1 {
		t.Errorf("expected still 1 sender after duplicate add, got %d", len(mem.Unwanted.Senders))
	}
}

func TestRemoveUnwantedSender(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	mgr.AddUnwantedSender(accountID, "spam1@example.com")
	mgr.AddUnwantedSender(accountID, "spam2@example.com")

	if err := mgr.RemoveUnwantedSender(accountID, "spam1@example.com"); err != nil {
		t.Fatalf("failed to remove unwanted sender: %v", err)
	}

	mem, _ := mgr.GetOrLoad(accountID)
	if len(mem.Unwanted.Senders) != 1 {
		t.Errorf("expected 1 sender after removal, got %d", len(mem.Unwanted.Senders))
	}
}

func TestAddNote(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	if err := mgr.AddNote(accountID, "Important rule: always archive work emails", "rule"); err != nil {
		t.Fatalf("failed to add note: %v", err)
	}

	mem, _ := mgr.GetOrLoad(accountID)
	if len(mem.Notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(mem.Notes))
	}
	if mem.Notes[0].Category != "rule" {
		t.Errorf("expected category 'rule', got %s", mem.Notes[0].Category)
	}
}

func TestRemoveNote(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	mgr.AddNote(accountID, "Note 1", "info")
	mgr.AddNote(accountID, "Note 2", "info")

	if err := mgr.RemoveNote(accountID, 0); err != nil {
		t.Fatalf("failed to remove note: %v", err)
	}

	mem, _ := mgr.GetOrLoad(accountID)
	if len(mem.Notes) != 1 {
		t.Errorf("expected 1 note after removal, got %d", len(mem.Notes))
	}
	if mem.Notes[0].Content != "Note 2" {
		t.Errorf("expected remaining note 'Note 2', got %s", mem.Notes[0].Content)
	}
}

func TestAddPriorityTopic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	if err := mgr.AddPriorityTopic(accountID, "urgent"); err != nil {
		t.Fatalf("failed to add topic: %v", err)
	}

	mem, _ := mgr.GetOrLoad(accountID)
	if len(mem.Topics.Priority) != 1 {
		t.Fatalf("expected 1 priority topic, got %d", len(mem.Topics.Priority))
	}

	// Duplicate should not be added
	mgr.AddPriorityTopic(accountID, "urgent")
	mem, _ = mgr.GetOrLoad(accountID)
	if len(mem.Topics.Priority) != 1 {
		t.Errorf("expected still 1 topic, got %d", len(mem.Topics.Priority))
	}
}

func TestSetFolderPurpose(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	if err := mgr.SetFolderPurpose(accountID, "Archive", "Old emails to keep"); err != nil {
		t.Fatalf("failed to set folder purpose: %v", err)
	}

	mem, _ := mgr.GetOrLoad(accountID)
	if mem.FolderPurposes["Archive"] != "Old emails to keep" {
		t.Errorf("expected folder purpose, got %s", mem.FolderPurposes["Archive"])
	}
}

func TestUpdatePreferences(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	prefs := Preferences{
		Language:      "en",
		SummaryStyle:  "detailed",
		Timezone:      "Europe/Prague",
		DefaultFolder: "Work",
	}

	if err := mgr.UpdatePreferences(accountID, prefs); err != nil {
		t.Fatalf("failed to update preferences: %v", err)
	}

	mem, _ := mgr.GetOrLoad(accountID)
	if mem.Preferences.Language != "en" {
		t.Errorf("expected language 'en', got %s", mem.Preferences.Language)
	}
	if mem.Preferences.SummaryStyle != "detailed" {
		t.Errorf("expected summary style 'detailed', got %s", mem.Preferences.SummaryStyle)
	}
}

func TestIncrementStat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	mgr.IncrementStat(accountID, "emails_processed", 5)
	mgr.IncrementStat(accountID, "emails_deleted", 2)

	mem, _ := mgr.GetOrLoad(accountID)
	if mem.Statistics.EmailsProcessed != 5 {
		t.Errorf("expected EmailsProcessed 5, got %d", mem.Statistics.EmailsProcessed)
	}
	if mem.Statistics.EmailsDeleted != 2 {
		t.Errorf("expected EmailsDeleted 2, got %d", mem.Statistics.EmailsDeleted)
	}
}

func TestStartSession(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	before := time.Now()
	mgr.StartSession(accountID)
	after := time.Now()

	mem, _ := mgr.GetOrLoad(accountID)
	if mem.Statistics.TotalSessions != 1 {
		t.Errorf("expected TotalSessions 1, got %d", mem.Statistics.TotalSessions)
	}
	if mem.Statistics.LastSessionStart.Before(before) || mem.Statistics.LastSessionStart.After(after) {
		t.Error("LastSessionStart not in expected time range")
	}
}

func TestListAccounts(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)

	// Create some accounts (use SaveNow for immediate persistence in tests)
	mgr.Load("account1")
	mgr.SaveNow("account1")
	mgr.Load("account2")
	mgr.SaveNow("account2")

	accounts, err := mgr.List()
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(accounts))
	}
}

func TestDelete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	// Create and save immediately (use SaveNow for tests)
	mgr.Load(accountID)
	mgr.SaveNow(accountID)

	// Verify file exists
	filePath := filepath.Join(tempDir, accountID+".json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("expected memory file to exist before delete")
	}

	// Delete
	if err := mgr.Delete(accountID); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("expected memory file to be deleted")
	}

	// Memory should not be in cache
	_, ok := mgr.Get(accountID)
	if ok {
		t.Error("expected memory to be removed from cache")
	}
}

func TestSearch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	// Add some data
	mgr.AddContact(accountID, Contact{Email: "john@example.com", Name: "John Doe", Role: "Developer"})
	mgr.AddNote(accountID, "Remember to check john's emails daily", "reminder")
	mgr.AddPriorityTopic(accountID, "project-alpha")
	mgr.AddUnwantedSender(accountID, "spam@badsite.com")
	mgr.SetFolderPurpose(accountID, "Work", "Work related emails")

	// Search for "john"
	results, err := mgr.Search(accountID, "john")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if results.Total < 2 {
		t.Errorf("expected at least 2 results for 'john', got %d", results.Total)
	}

	// Search for "spam"
	results, err = mgr.Search(accountID, "spam")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if results.Total < 1 {
		t.Errorf("expected at least 1 result for 'spam', got %d", results.Total)
	}

	// Search for "work"
	results, err = mgr.Search(accountID, "work")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if results.Total < 1 {
		t.Errorf("expected at least 1 result for 'work', got %d", results.Total)
	}
}

func TestAddObservation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	// First add a contact
	mgr.AddContact(accountID, Contact{Email: "boss@company.com", Name: "Boss"})

	// Add observation
	if err := mgr.AddObservation(accountID, "boss@company.com", "Prefers short emails"); err != nil {
		t.Fatalf("failed to add observation: %v", err)
	}

	contact, err := mgr.GetContact(accountID, "boss@company.com")
	if err != nil {
		t.Fatalf("failed to get contact: %v", err)
	}
	if len(contact.Observations) != 1 {
		t.Errorf("expected 1 observation, got %d", len(contact.Observations))
	}

	// Adding duplicate should not create duplicate
	mgr.AddObservation(accountID, "boss@company.com", "Prefers short emails")
	contact, _ = mgr.GetContact(accountID, "boss@company.com")
	if len(contact.Observations) != 1 {
		t.Errorf("expected still 1 observation, got %d", len(contact.Observations))
	}
}

func TestRemoveObservation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	mgr.AddContact(accountID, Contact{Email: "boss@company.com"})
	mgr.AddObservation(accountID, "boss@company.com", "Observation 1")
	mgr.AddObservation(accountID, "boss@company.com", "Observation 2")

	if err := mgr.RemoveObservation(accountID, "boss@company.com", "Observation 1"); err != nil {
		t.Fatalf("failed to remove observation: %v", err)
	}

	contact, _ := mgr.GetContact(accountID, "boss@company.com")
	if len(contact.Observations) != 1 {
		t.Errorf("expected 1 observation after removal, got %d", len(contact.Observations))
	}
	if contact.Observations[0] != "Observation 2" {
		t.Errorf("expected 'Observation 2', got %s", contact.Observations[0])
	}
}

func TestGetContact(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, _ := NewManager(tempDir)
	accountID := "test-account"

	mgr.AddContact(accountID, Contact{Email: "test@example.com", Name: "Test User"})

	contact, err := mgr.GetContact(accountID, "test@example.com")
	if err != nil {
		t.Fatalf("failed to get contact: %v", err)
	}
	if contact.Name != "Test User" {
		t.Errorf("expected name 'Test User', got %s", contact.Name)
	}

	// Non-existent contact
	_, err = mgr.GetContact(accountID, "nonexistent@example.com")
	if err == nil {
		t.Error("expected error for non-existent contact")
	}
}

func TestGetSummary(t *testing.T) {
	mem := NewAccountMemory("test-account")
	mem.ImportantContacts = []Contact{
		{Email: "a@test.com"},
		{Email: "b@test.com"},
	}
	mem.Topics.Priority = []string{"urgent", "important"}
	mem.Unwanted.Senders = []string{"spam1@test.com", "spam2@test.com"}
	mem.Notes = []Note{
		{Content: "Note 1"},
		{Content: "Note 2"},
		{Content: "Note 3"},
	}

	summary := mem.GetSummary()

	if summary.ImportantContactsCount != 2 {
		t.Errorf("expected 2 important contacts, got %d", summary.ImportantContactsCount)
	}
	if len(summary.ImportantEmails) != 2 {
		t.Errorf("expected 2 important emails, got %d", len(summary.ImportantEmails))
	}
	if len(summary.PriorityTopics) != 2 {
		t.Errorf("expected 2 priority topics, got %d", len(summary.PriorityTopics))
	}
	if summary.UnwantedSendersCount != 2 {
		t.Errorf("expected 2 unwanted senders, got %d", summary.UnwantedSendersCount)
	}
	if summary.NotesCount != 3 {
		t.Errorf("expected 3 notes, got %d", summary.NotesCount)
	}
}
