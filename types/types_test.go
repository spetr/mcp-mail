package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAccountConfigJSON(t *testing.T) {
	cfg := AccountConfig{
		ID:       "gmail",
		Name:     "My Gmail",
		Host:     "imap.gmail.com",
		Port:     993,
		TLS:      true,
		Username: "user@gmail.com",
		Password: "secret",
	}

	// Marshal
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal
	var cfg2 AccountConfig
	if err := json.Unmarshal(data, &cfg2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if cfg2.ID != cfg.ID {
		t.Errorf("expected ID %s, got %s", cfg.ID, cfg2.ID)
	}
	if cfg2.Host != cfg.Host {
		t.Errorf("expected Host %s, got %s", cfg.Host, cfg2.Host)
	}
	if cfg2.Port != cfg.Port {
		t.Errorf("expected Port %d, got %d", cfg.Port, cfg2.Port)
	}
	if cfg2.TLS != cfg.TLS {
		t.Errorf("expected TLS %v, got %v", cfg.TLS, cfg2.TLS)
	}
}

func TestAccountStatusJSON(t *testing.T) {
	status := AccountStatus{
		ID:           "gmail",
		Name:         "My Gmail",
		Connected:    true,
		Capabilities: []string{"IMAP4rev1", "IDLE", "MOVE"},
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var status2 AccountStatus
	if err := json.Unmarshal(data, &status2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if status2.Connected != status.Connected {
		t.Errorf("expected Connected %v, got %v", status.Connected, status2.Connected)
	}
	if len(status2.Capabilities) != len(status.Capabilities) {
		t.Errorf("expected %d capabilities, got %d", len(status.Capabilities), len(status2.Capabilities))
	}
}

func TestAccountStatusWithError(t *testing.T) {
	status := AccountStatus{
		ID:        "gmail",
		Name:      "My Gmail",
		Connected: false,
		Error:     "connection refused",
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify error is included in JSON
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if m["error"] != "connection refused" {
		t.Error("expected error field in JSON")
	}
}

func TestAccountInfoEmbedding(t *testing.T) {
	info := AccountInfo{
		AccountStatus: AccountStatus{
			ID:        "gmail",
			Connected: true,
		},
		TotalFolders:  10,
		TotalMessages: 1000,
		UnreadCount:   25,
		CurrentFolder: "INBOX",
	}

	// Verify embedded fields are accessible
	if info.ID != "gmail" {
		t.Errorf("expected ID 'gmail', got %s", info.ID)
	}
	if !info.Connected {
		t.Error("expected Connected to be true")
	}
	if info.TotalFolders != 10 {
		t.Errorf("expected TotalFolders 10, got %d", info.TotalFolders)
	}
}

func TestSearchCriteriaJSON(t *testing.T) {
	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	seen := true

	criteria := SearchCriteria{
		From:    "boss@company.com",
		Subject: "Important",
		Since:   &since,
		Seen:    &seen,
		Larger:  1024,
	}

	data, err := json.Marshal(criteria)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var criteria2 SearchCriteria
	if err := json.Unmarshal(data, &criteria2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if criteria2.From != criteria.From {
		t.Errorf("expected From %s, got %s", criteria.From, criteria2.From)
	}
	if criteria2.Subject != criteria.Subject {
		t.Errorf("expected Subject %s, got %s", criteria.Subject, criteria2.Subject)
	}
	if criteria2.Larger != criteria.Larger {
		t.Errorf("expected Larger %d, got %d", criteria.Larger, criteria2.Larger)
	}
}

func TestSearchCriteriaOmitEmpty(t *testing.T) {
	// Empty criteria should produce minimal JSON
	criteria := SearchCriteria{}

	data, err := json.Marshal(criteria)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Should be mostly empty
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	// Only zero-value fields that aren't omitempty should be present
	if _, ok := m["from"]; ok {
		t.Error("expected 'from' to be omitted when empty")
	}
	if _, ok := m["subject"]; ok {
		t.Error("expected 'subject' to be omitted when empty")
	}
}

func TestSearchCriteriaWithOr(t *testing.T) {
	criteria := SearchCriteria{
		From: "user1@test.com",
		Or: []SearchCriteria{
			{From: "user2@test.com"},
			{From: "user3@test.com"},
		},
	}

	data, err := json.Marshal(criteria)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var criteria2 SearchCriteria
	if err := json.Unmarshal(data, &criteria2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(criteria2.Or) != 2 {
		t.Errorf("expected 2 OR criteria, got %d", len(criteria2.Or))
	}
	if criteria2.Or[0].From != "user2@test.com" {
		t.Errorf("expected first OR from 'user2@test.com', got %s", criteria2.Or[0].From)
	}
}

func TestSearchCriteriaWithNot(t *testing.T) {
	notCriteria := SearchCriteria{From: "spam@test.com"}
	criteria := SearchCriteria{
		From: "user@test.com",
		Not:  &notCriteria,
	}

	data, err := json.Marshal(criteria)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var criteria2 SearchCriteria
	if err := json.Unmarshal(data, &criteria2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if criteria2.Not == nil {
		t.Fatal("expected NOT criteria to be present")
	}
	if criteria2.Not.From != "spam@test.com" {
		t.Errorf("expected NOT from 'spam@test.com', got %s", criteria2.Not.From)
	}
}

func TestSearchResult(t *testing.T) {
	result := SearchResult{
		UIDs:  []uint32{1, 2, 3, 4, 5},
		Total: 5,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result2 SearchResult
	if err := json.Unmarshal(data, &result2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result2.Total != 5 {
		t.Errorf("expected Total 5, got %d", result2.Total)
	}
	if len(result2.UIDs) != 5 {
		t.Errorf("expected 5 UIDs, got %d", len(result2.UIDs))
	}
}

func TestQuickSearchTypes(t *testing.T) {
	// Verify constants are defined correctly
	tests := []struct {
		searchType QuickSearchType
		expected   string
	}{
		{QuickSearchUnread, "unread"},
		{QuickSearchFlagged, "flagged"},
		{QuickSearchToday, "today"},
		{QuickSearchThisWeek, "this_week"},
		{QuickSearchWithAttachments, "with_attachments"},
	}

	for _, tt := range tests {
		t.Run(string(tt.searchType), func(t *testing.T) {
			if string(tt.searchType) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.searchType))
			}
		})
	}
}

func TestSearchCriteriaDateRange(t *testing.T) {
	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	criteria := SearchCriteria{
		Since:  &since,
		Before: &before,
	}

	if criteria.Since.Year() != 2024 || criteria.Since.Month() != 1 {
		t.Error("Since date not set correctly")
	}
	if criteria.Before.Year() != 2024 || criteria.Before.Month() != 12 {
		t.Error("Before date not set correctly")
	}
}

func TestSearchCriteriaUIDRange(t *testing.T) {
	criteria := SearchCriteria{
		UIDFrom: 100,
		UIDto:   200,
	}

	if criteria.UIDFrom != 100 {
		t.Errorf("expected UIDFrom 100, got %d", criteria.UIDFrom)
	}
	if criteria.UIDto != 200 {
		t.Errorf("expected UIDto 200, got %d", criteria.UIDto)
	}
}

func TestSearchCriteriaSizeFilters(t *testing.T) {
	criteria := SearchCriteria{
		Larger:  1024 * 1024,    // 1MB
		Smaller: 10 * 1024 * 1024, // 10MB
	}

	if criteria.Larger != 1024*1024 {
		t.Errorf("expected Larger 1MB, got %d", criteria.Larger)
	}
	if criteria.Smaller != 10*1024*1024 {
		t.Errorf("expected Smaller 10MB, got %d", criteria.Smaller)
	}
}

func TestSearchCriteriaAllFlags(t *testing.T) {
	seen := true
	answered := true
	flagged := false
	deleted := false
	draft := true

	criteria := SearchCriteria{
		Seen:     &seen,
		Answered: &answered,
		Flagged:  &flagged,
		Deleted:  &deleted,
		Draft:    &draft,
	}

	if *criteria.Seen != true {
		t.Error("expected Seen true")
	}
	if *criteria.Answered != true {
		t.Error("expected Answered true")
	}
	if *criteria.Flagged != false {
		t.Error("expected Flagged false")
	}
	if *criteria.Deleted != false {
		t.Error("expected Deleted false")
	}
	if *criteria.Draft != true {
		t.Error("expected Draft true")
	}
}
