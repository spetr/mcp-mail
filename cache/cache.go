package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spetr/mcp-mail/types"
	"github.com/spetr/fastcache"
)

// Config contains cache settings
type Config struct {
	Enabled              bool `json:"enabled"`
	FolderListTTLSec     int  `json:"folder_list_ttl_sec"`
	FolderInfoTTLSec     int  `json:"folder_info_ttl_sec"`
	MessageListTTLSec    int  `json:"message_list_ttl_sec"`
	SearchResultsTTLSec  int  `json:"search_results_ttl_sec"`
	TrashFolderTTLSec    int  `json:"trash_folder_ttl_sec"`
	MaxFolderInfoEntries int  `json:"max_folder_info_entries"`
	MaxMessageListEntries int `json:"max_message_list_entries"`
	MaxSearchResultEntries int `json:"max_search_result_entries"`
}

// DefaultConfig returns default cache configuration
func DefaultConfig() Config {
	return Config{
		Enabled:              true,
		FolderListTTLSec:     300, // 5 minutes
		FolderInfoTTLSec:     120, // 2 minutes
		MessageListTTLSec:    120, // 2 minutes
		SearchResultsTTLSec:  60,  // 1 minute
		TrashFolderTTLSec:    3600, // 1 hour
		MaxFolderInfoEntries: 100,
		MaxMessageListEntries: 50,
		MaxSearchResultEntries: 20,
	}
}

// MergeDefaults fills zero values with defaults
func (c *Config) MergeDefaults() {
	defaults := DefaultConfig()
	if c.FolderListTTLSec == 0 {
		c.FolderListTTLSec = defaults.FolderListTTLSec
	}
	if c.FolderInfoTTLSec == 0 {
		c.FolderInfoTTLSec = defaults.FolderInfoTTLSec
	}
	if c.MessageListTTLSec == 0 {
		c.MessageListTTLSec = defaults.MessageListTTLSec
	}
	if c.SearchResultsTTLSec == 0 {
		c.SearchResultsTTLSec = defaults.SearchResultsTTLSec
	}
	if c.TrashFolderTTLSec == 0 {
		c.TrashFolderTTLSec = defaults.TrashFolderTTLSec
	}
	if c.MaxFolderInfoEntries == 0 {
		c.MaxFolderInfoEntries = defaults.MaxFolderInfoEntries
	}
	if c.MaxMessageListEntries == 0 {
		c.MaxMessageListEntries = defaults.MaxMessageListEntries
	}
	if c.MaxSearchResultEntries == 0 {
		c.MaxSearchResultEntries = defaults.MaxSearchResultEntries
	}
}

// FolderState tracks UIDVALIDITY for cache invalidation
type FolderState struct {
	UIDValidity uint32
	UIDNext     uint32
	LastCheck   time.Time
}

// Manager manages all caches
type Manager struct {
	config Config
	ctx    context.Context

	// Cached data (all use LRU with TTL)
	folderStates  fastcache.Cache // key: "accountID:folder" -> *FolderState
	folderLists   fastcache.Cache // key: "accountID" -> []types.FolderListItem
	folderInfo    fastcache.Cache // key: "accountID:folder" -> *types.FolderInfo
	messageLists  fastcache.Cache // key: "accountID:folder:limit:offset" -> []types.MessageListItem
	searchResults fastcache.Cache // key: "accountID:folder:criteriaHash" -> []uint32
	trashFolders  fastcache.Cache // key: "accountID" -> string
}

// NewManager creates a new cache manager
func NewManager(cfg Config) *Manager {
	cfg.MergeDefaults()

	if !cfg.Enabled {
		return &Manager{config: cfg}
	}

	return &Manager{
		config: cfg,
		ctx:    context.Background(),

		folderStates: fastcache.New(100).LRU().
			Expiration(1 * time.Hour).Build(),

		folderLists: fastcache.New(10).LRU().
			Expiration(time.Duration(cfg.FolderListTTLSec) * time.Second).Build(),

		folderInfo: fastcache.New(cfg.MaxFolderInfoEntries).LRU().
			Expiration(time.Duration(cfg.FolderInfoTTLSec) * time.Second).Build(),

		messageLists: fastcache.New(cfg.MaxMessageListEntries).LRU().
			Expiration(time.Duration(cfg.MessageListTTLSec) * time.Second).Build(),

		searchResults: fastcache.New(cfg.MaxSearchResultEntries).LRU().
			Expiration(time.Duration(cfg.SearchResultsTTLSec) * time.Second).Build(),

		trashFolders: fastcache.New(10).LRU().
			Expiration(time.Duration(cfg.TrashFolderTTLSec) * time.Second).Build(),
	}
}

// IsEnabled returns true if caching is enabled
func (m *Manager) IsEnabled() bool {
	return m.config.Enabled
}

// === GET METHODS ===

// GetFolderList returns cached folder list or nil
func (m *Manager) GetFolderList(accountID string) []types.FolderListItem {
	if !m.config.Enabled {
		return nil
	}
	val, err := m.folderLists.GetIFPresent(m.ctx, accountID)
	if err != nil {
		return nil
	}
	if list, ok := val.([]types.FolderListItem); ok {
		return list
	}
	return nil
}

// GetFolderInfo returns cached folder info or nil
func (m *Manager) GetFolderInfo(accountID, folder string) *types.FolderInfo {
	if !m.config.Enabled {
		return nil
	}
	key := fmt.Sprintf("%s:%s", accountID, folder)
	val, err := m.folderInfo.GetIFPresent(m.ctx, key)
	if err != nil {
		return nil
	}
	if info, ok := val.(*types.FolderInfo); ok {
		return info
	}
	return nil
}

// GetMessageList returns cached message list or nil
func (m *Manager) GetMessageList(accountID, folder string, limit, offset int) []types.MessageListItem {
	if !m.config.Enabled {
		return nil
	}
	key := fmt.Sprintf("%s:%s:%d:%d", accountID, folder, limit, offset)
	val, err := m.messageLists.GetIFPresent(m.ctx, key)
	if err != nil {
		return nil
	}
	if list, ok := val.([]types.MessageListItem); ok {
		return list
	}
	return nil
}

// GetSearchResults returns cached search UIDs or nil
func (m *Manager) GetSearchResults(accountID, folder, criteriaHash string) []uint32 {
	if !m.config.Enabled {
		return nil
	}
	key := fmt.Sprintf("%s:%s:%s", accountID, folder, criteriaHash)
	val, err := m.searchResults.GetIFPresent(m.ctx, key)
	if err != nil {
		return nil
	}
	if uids, ok := val.([]uint32); ok {
		return uids
	}
	return nil
}

// GetTrashFolder returns cached trash folder name or ""
func (m *Manager) GetTrashFolder(accountID string) string {
	if !m.config.Enabled {
		return ""
	}
	val, err := m.trashFolders.GetIFPresent(m.ctx, accountID)
	if err != nil {
		return ""
	}
	if name, ok := val.(string); ok {
		return name
	}
	return ""
}

// === SET METHODS ===

// SetFolderList stores folder list in cache
func (m *Manager) SetFolderList(accountID string, folders []types.FolderListItem) {
	if m.config.Enabled && m.folderLists != nil {
		m.folderLists.SetWithExpire(accountID, folders,
			time.Duration(m.config.FolderListTTLSec)*time.Second)
	}
}

// SetFolderInfo stores folder info in cache
func (m *Manager) SetFolderInfo(accountID, folder string, info *types.FolderInfo) {
	if m.config.Enabled && m.folderInfo != nil {
		key := fmt.Sprintf("%s:%s", accountID, folder)
		m.folderInfo.SetWithExpire(key, info,
			time.Duration(m.config.FolderInfoTTLSec)*time.Second)
	}
}

// SetMessageList stores message list in cache
func (m *Manager) SetMessageList(accountID, folder string, limit, offset int, messages []types.MessageListItem) {
	if m.config.Enabled && m.messageLists != nil {
		key := fmt.Sprintf("%s:%s:%d:%d", accountID, folder, limit, offset)
		m.messageLists.SetWithExpire(key, messages,
			time.Duration(m.config.MessageListTTLSec)*time.Second)
	}
}

// SetSearchResults stores search UIDs in cache
func (m *Manager) SetSearchResults(accountID, folder, criteriaHash string, uids []uint32) {
	if m.config.Enabled && m.searchResults != nil {
		key := fmt.Sprintf("%s:%s:%s", accountID, folder, criteriaHash)
		m.searchResults.SetWithExpire(key, uids,
			time.Duration(m.config.SearchResultsTTLSec)*time.Second)
	}
}

// SetTrashFolder stores trash folder name in cache
func (m *Manager) SetTrashFolder(accountID, trashFolder string) {
	if m.config.Enabled && m.trashFolders != nil {
		m.trashFolders.SetWithExpire(accountID, trashFolder,
			time.Duration(m.config.TrashFolderTTLSec)*time.Second)
	}
}

// === INVALIDATION METHODS ===

// InvalidateFolderList removes cached folder list for account
func (m *Manager) InvalidateFolderList(accountID string) {
	if m.config.Enabled && m.folderLists != nil {
		m.folderLists.Remove(accountID)
	}
}

// InvalidateFolder removes all cache for a folder (after UIDVALIDITY change)
func (m *Manager) InvalidateFolder(accountID, folder string) {
	if !m.config.Enabled {
		return
	}
	key := fmt.Sprintf("%s:%s", accountID, folder)
	if m.folderInfo != nil {
		m.folderInfo.Remove(key)
	}
	if m.folderStates != nil {
		m.folderStates.Remove(key)
	}

	// Remove message lists and search results for this folder
	prefix := fmt.Sprintf("%s:%s:", accountID, folder)
	m.invalidateByPrefix(m.messageLists, prefix)
	m.invalidateByPrefix(m.searchResults, prefix)
}

// InvalidateFolderContent removes message lists and search for folder (after message changes)
func (m *Manager) InvalidateFolderContent(accountID, folder string) {
	if !m.config.Enabled {
		return
	}
	prefix := fmt.Sprintf("%s:%s:", accountID, folder)
	m.invalidateByPrefix(m.messageLists, prefix)
	m.invalidateByPrefix(m.searchResults, prefix)

	// Folder info too - message count changed
	key := fmt.Sprintf("%s:%s", accountID, folder)
	if m.folderInfo != nil {
		m.folderInfo.Remove(key)
	}
}

// InvalidateAccount removes all cache for account
func (m *Manager) InvalidateAccount(accountID string) {
	if !m.config.Enabled {
		return
	}
	if m.folderLists != nil {
		m.folderLists.Remove(accountID)
	}
	if m.trashFolders != nil {
		m.trashFolders.Remove(accountID)
	}
	prefix := accountID + ":"
	m.invalidateByPrefix(m.folderInfo, prefix)
	m.invalidateByPrefix(m.folderStates, prefix)
	m.invalidateByPrefix(m.messageLists, prefix)
	m.invalidateByPrefix(m.searchResults, prefix)
}

// === UIDVALIDITY TRACKING ===

// CheckUIDValidity checks UIDVALIDITY and returns true if cache is valid
// If UIDVALIDITY changed, automatically invalidates cache for folder
func (m *Manager) CheckUIDValidity(accountID, folder string, newUIDValidity uint32) bool {
	if !m.config.Enabled || m.folderStates == nil {
		return true
	}

	key := fmt.Sprintf("%s:%s", accountID, folder)

	val, err := m.folderStates.GetIFPresent(m.ctx, key)
	if err != nil {
		// No state - store new one
		m.folderStates.Set(key, &FolderState{
			UIDValidity: newUIDValidity,
			LastCheck:   time.Now(),
		})
		return true // First access - cache is empty, consider valid
	}

	state, ok := val.(*FolderState)
	if !ok {
		return false
	}

	if state.UIDValidity != newUIDValidity {
		// UIDVALIDITY changed - invalidate everything for this folder
		m.InvalidateFolder(accountID, folder)

		// Store new state
		m.folderStates.Set(key, &FolderState{
			UIDValidity: newUIDValidity,
			LastCheck:   time.Now(),
		})
		return false // Cache was invalidated
	}

	// UIDVALIDITY is same - cache is valid
	state.LastCheck = time.Now()
	return true
}

// === HELPER METHODS ===

func (m *Manager) invalidateByPrefix(cache fastcache.Cache, prefix string) {
	if cache == nil {
		return
	}
	keys := cache.Keys(true) // true = check expired
	for _, k := range keys {
		if key, ok := k.(string); ok {
			if strings.HasPrefix(key, prefix) {
				cache.Remove(k)
			}
		}
	}
}

// Stats returns cache statistics
func (m *Manager) Stats() map[string]int {
	if !m.config.Enabled {
		return map[string]int{"enabled": 0}
	}
	return map[string]int{
		"enabled":        1,
		"folder_lists":   m.folderLists.Len(true),
		"folder_info":    m.folderInfo.Len(true),
		"message_lists":  m.messageLists.Len(true),
		"search_results": m.searchResults.Len(true),
		"folder_states":  m.folderStates.Len(true),
	}
}
