package imap

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/spetr/mcp-mail/types"
)

// GetThread retrieves all messages in a thread starting from a given message UID
func (c *Client) GetThread(folder string, uid uint32) (*types.Thread, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if err := c.selectFolder(folder); err != nil {
		return nil, err
	}

	// First, get the starting message to find its Message-ID and References
	startMsg, err := c.fetchMessageEnvelope(uid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch starting message: %w", err)
	}

	// Collect all Message-IDs that belong to this thread
	threadIDs := make(map[string]bool)
	threadIDs[startMsg.MessageID] = true

	// Add References
	for _, ref := range startMsg.References {
		threadIDs[ref] = true
	}

	// Add In-Reply-To
	if startMsg.InReplyTo != "" {
		threadIDs[startMsg.InReplyTo] = true
	}

	// Now search for all messages that reference these IDs or have these IDs
	allMessages, err := c.fetchAllEnvelopes(folder)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages for thread building: %w", err)
	}

	// Build thread by finding related messages
	threadMessages := make([]internalEnvelope, 0)
	// Keep iterating until no new messages are found
	for {
		foundNew := false
		for _, msg := range allMessages {
			// Skip if already in thread
			if threadIDs[msg.MessageID] {
				continue
			}

			// Check if this message references any message in our thread
			if threadIDs[msg.InReplyTo] {
				threadIDs[msg.MessageID] = true
				foundNew = true
				continue
			}

			for _, ref := range msg.References {
				if threadIDs[ref] {
					threadIDs[msg.MessageID] = true
					foundNew = true
					break
				}
			}
		}
		if !foundNew {
			break
		}
	}

	// Collect all messages that are in the thread
	for _, msg := range allMessages {
		if threadIDs[msg.MessageID] {
			threadMessages = append(threadMessages, msg)
		}
	}

	// Sort by date (oldest first)
	sort.Slice(threadMessages, func(i, j int) bool {
		return threadMessages[i].Date.Before(threadMessages[j].Date)
	})

	// Build the Thread result
	thread := &types.Thread{
		Messages:     make([]types.ThreadMessage, 0, len(threadMessages)),
		TotalCount:   len(threadMessages),
		Participants: make([]types.Address, 0),
	}

	participantMap := make(map[string]types.Address)
	var rootMessageID string

	for i, msg := range threadMessages {
		isRoot := i == 0
		if isRoot {
			rootMessageID = msg.MessageID
			thread.Subject = normalizeSubject(msg.Subject)
		}

		// Count unread
		isUnread := true
		for _, flag := range msg.Flags {
			if flag == "\\Seen" {
				isUnread = false
				break
			}
		}
		if isUnread {
			thread.UnreadCount++
		}

		// Collect participants
		for _, addr := range msg.From {
			participantMap[strings.ToLower(addr.Address)] = addr
		}
		for _, addr := range msg.To {
			participantMap[strings.ToLower(addr.Address)] = addr
		}

		thread.Messages = append(thread.Messages, types.ThreadMessage{
			UID:       msg.UID,
			MessageID: msg.MessageID,
			Subject:   msg.Subject,
			From:      msg.From,
			To:        msg.To,
			Date:      msg.Date.Format(time.RFC3339),
			Flags:     msg.Flags,
			IsRoot:    isRoot,
		})
	}

	thread.ThreadID = rootMessageID
	if len(threadMessages) > 0 {
		thread.LastMessageDate = threadMessages[len(threadMessages)-1].Date.Format(time.RFC3339)
	}

	// Convert participant map to slice
	for _, addr := range participantMap {
		thread.Participants = append(thread.Participants, addr)
	}

	return thread, nil
}

// GetThreadByMessageID retrieves a thread by Message-ID
func (c *Client) GetThreadByMessageID(folder string, messageID string) (*types.Thread, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if err := c.selectFolder(folder); err != nil {
		return nil, err
	}

	// Search for the message with this Message-ID
	searchCriteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{
			{Key: "Message-ID", Value: messageID},
		},
	}

	searchOpts := &imap.SearchOptions{ReturnAll: true}
	searchCmd := c.client.UIDSearch(searchCriteria, searchOpts)
	searchData, err := searchCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	uids := searchData.AllUIDs()
	if len(uids) == 0 {
		return nil, fmt.Errorf("message with Message-ID '%s' not found", messageID)
	}

	// Unlock and call GetThread (which will re-lock)
	c.mu.Unlock()
	thread, err := c.GetThread(folder, uint32(uids[0]))
	c.mu.Lock()

	return thread, err
}

// ListThreads returns a list of thread summaries in a folder
func (c *Client) ListThreads(folder string, limit, offset int) ([]types.ThreadListItem, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if err := c.selectFolder(folder); err != nil {
		return nil, err
	}

	// Fetch all envelopes
	allMessages, err := c.fetchAllEnvelopes(folder)
	if err != nil {
		return nil, err
	}

	if len(allMessages) == 0 {
		return []types.ThreadListItem{}, nil
	}

	// Group messages into threads
	threads := c.groupIntoThreads(allMessages)

	// Sort threads by last message date (newest first)
	sort.Slice(threads, func(i, j int) bool {
		return threads[i].lastDate.After(threads[j].lastDate)
	})

	// Apply pagination
	if offset >= len(threads) {
		return []types.ThreadListItem{}, nil
	}

	end := offset + limit
	if end > len(threads) || limit <= 0 {
		end = len(threads)
	}

	threads = threads[offset:end]

	// Convert to ThreadListItem
	result := make([]types.ThreadListItem, 0, len(threads))
	for _, t := range threads {
		item := types.ThreadListItem{
			ThreadID:        t.rootMessageID,
			Subject:         t.subject,
			LastMessageDate: t.lastDate.Format(time.RFC3339),
			TotalCount:      t.totalCount,
			UnreadCount:     t.unreadCount,
			HasAttachments:  t.hasAttachments,
			Participants:    make([]types.Address, 0),
			LastMessageFrom: t.lastFrom,
		}

		for _, addr := range t.participants {
			item.Participants = append(item.Participants, addr)
		}

		result = append(result, item)
	}

	return result, nil
}

// Internal types for thread building
type internalEnvelope struct {
	UID        uint32
	MessageID  string
	InReplyTo  string
	References []string
	Subject    string
	From       []types.Address
	To         []types.Address
	Date       time.Time
	Flags      []string
	HasAttach  bool
}

type threadGroup struct {
	rootMessageID  string
	subject        string
	messages       []internalEnvelope
	participants   map[string]types.Address
	lastDate       time.Time
	lastFrom       []types.Address
	totalCount     int
	unreadCount    int
	hasAttachments bool
}

// fetchMessageEnvelope fetches envelope data for a single message
func (c *Client) fetchMessageEnvelope(uid uint32) (*internalEnvelope, error) {
	uidSet := imap.UIDSetNum(imap.UID(uid))

	fetchOpts := &imap.FetchOptions{
		UID:           true,
		Flags:         true,
		Envelope:      true,
		BodyStructure: &imap.FetchItemBodyStructure{},
	}

	fetchCmd := c.client.Fetch(uidSet, fetchOpts)
	defer fetchCmd.Close()

	msg := fetchCmd.Next()
	if msg == nil {
		return nil, fmt.Errorf("message with UID %d not found", uid)
	}

	buf, err := msg.Collect()
	if err != nil {
		return nil, err
	}

	env := &internalEnvelope{
		UID:       uint32(buf.UID),
		Flags:     convertFlags(buf.Flags),
		HasAttach: hasAttachments(buf.BodyStructure),
	}

	if buf.Envelope != nil {
		env.MessageID = buf.Envelope.MessageID
		env.Subject = buf.Envelope.Subject
		env.Date = buf.Envelope.Date
		env.From = convertAddresses(buf.Envelope.From)
		env.To = convertAddresses(buf.Envelope.To)
		if len(buf.Envelope.InReplyTo) > 0 {
			env.InReplyTo = buf.Envelope.InReplyTo[0]
		}
		// Parse References header - it's not directly available in Envelope
		// We would need to fetch the header separately for full References
	}

	return env, fetchCmd.Close()
}

// fetchAllEnvelopes fetches envelope data for all messages in current folder
func (c *Client) fetchAllEnvelopes(folder string) ([]internalEnvelope, error) {
	// Get message count
	selectData, err := c.client.Select(folder, nil).Wait()
	if err != nil {
		return nil, err
	}
	c.selected = folder

	if selectData.NumMessages == 0 {
		return []internalEnvelope{}, nil
	}

	// Fetch all messages
	var seqSet imap.SeqSet
	seqSet.AddRange(1, selectData.NumMessages)

	fetchOpts := &imap.FetchOptions{
		UID:           true,
		Flags:         true,
		Envelope:      true,
		BodyStructure: &imap.FetchItemBodyStructure{},
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierHeader, HeaderFields: []string{"References"}},
		},
	}

	fetchCmd := c.client.Fetch(seqSet, fetchOpts)
	defer fetchCmd.Close()

	messages := make([]internalEnvelope, 0, selectData.NumMessages)

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		buf, err := msg.Collect()
		if err != nil {
			continue
		}

		env := internalEnvelope{
			UID:       uint32(buf.UID),
			Flags:     convertFlags(buf.Flags),
			HasAttach: hasAttachments(buf.BodyStructure),
		}

		if buf.Envelope != nil {
			env.MessageID = buf.Envelope.MessageID
			env.Subject = buf.Envelope.Subject
			env.Date = buf.Envelope.Date
			env.From = convertAddresses(buf.Envelope.From)
			env.To = convertAddresses(buf.Envelope.To)
			if len(buf.Envelope.InReplyTo) > 0 {
				env.InReplyTo = buf.Envelope.InReplyTo[0]
			}
		}

		// Parse References header from body section
		for _, section := range buf.BodySection {
			if len(section.Bytes) > 0 {
				refs := parseReferencesHeader(string(section.Bytes))
				env.References = refs
			}
		}

		// Skip messages without Message-ID
		if env.MessageID != "" {
			messages = append(messages, env)
		}
	}

	return messages, fetchCmd.Close()
}

// groupIntoThreads groups messages into conversation threads
func (c *Client) groupIntoThreads(messages []internalEnvelope) []*threadGroup {
	// Map Message-ID to thread
	msgToThread := make(map[string]*threadGroup)
	threads := make([]*threadGroup, 0)

	// Sort by date first (oldest first) to process in order
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Date.Before(messages[j].Date)
	})

	for _, msg := range messages {
		var thread *threadGroup

		// Check if this message belongs to an existing thread
		// via In-Reply-To
		if msg.InReplyTo != "" {
			thread = msgToThread[msg.InReplyTo]
		}

		// Check via References
		if thread == nil {
			for _, ref := range msg.References {
				if t := msgToThread[ref]; t != nil {
					thread = t
					break
				}
			}
		}

		// Check by normalized subject (fallback)
		if thread == nil {
			normalizedSubj := normalizeSubject(msg.Subject)
			for _, t := range threads {
				if t.subject == normalizedSubj {
					thread = t
					break
				}
			}
		}

		if thread == nil {
			// Create new thread
			thread = &threadGroup{
				rootMessageID: msg.MessageID,
				subject:       normalizeSubject(msg.Subject),
				messages:      make([]internalEnvelope, 0),
				participants:  make(map[string]types.Address),
			}
			threads = append(threads, thread)
		}

		// Add message to thread
		thread.messages = append(thread.messages, msg)
		thread.totalCount++

		// Track unread
		isUnread := true
		for _, flag := range msg.Flags {
			if flag == "\\Seen" {
				isUnread = false
				break
			}
		}
		if isUnread {
			thread.unreadCount++
		}

		// Track attachments
		if msg.HasAttach {
			thread.hasAttachments = true
		}

		// Track participants
		for _, addr := range msg.From {
			thread.participants[strings.ToLower(addr.Address)] = addr
		}
		for _, addr := range msg.To {
			thread.participants[strings.ToLower(addr.Address)] = addr
		}

		// Update last message info
		if msg.Date.After(thread.lastDate) {
			thread.lastDate = msg.Date
			thread.lastFrom = msg.From
		}

		// Map this Message-ID to the thread
		msgToThread[msg.MessageID] = thread
	}

	return threads
}

// normalizeSubject removes Re:/Fwd:/etc. prefixes from subject
func normalizeSubject(subject string) string {
	// Pattern to match common reply/forward prefixes
	re := regexp.MustCompile(`(?i)^(re|fwd|fw|odp|sv|aw|r|rif|i|fs|vb|vs|tr|doorst|antw|res|enc|πρθ|回复|转发):\s*`)

	result := subject
	for {
		cleaned := re.ReplaceAllString(result, "")
		if cleaned == result {
			break
		}
		result = cleaned
	}

	return strings.TrimSpace(result)
}

// parseReferencesHeader parses References header into list of Message-IDs
func parseReferencesHeader(header string) []string {
	// Extract just the References value
	lines := strings.Split(header, "\n")
	var refsLine string
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "references:") {
			refsLine = strings.TrimPrefix(line, "References:")
			refsLine = strings.TrimPrefix(refsLine, "references:")
			break
		}
	}

	if refsLine == "" {
		return nil
	}

	// Extract Message-IDs (format: <...@...>)
	re := regexp.MustCompile(`<[^>]+>`)
	matches := re.FindAllString(refsLine, -1)

	return matches
}
