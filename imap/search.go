package imap

import (
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/spetr/mcp-mail/types"
)

// Search searches for messages matching the criteria
func (c *Client) Search(folder string, criteria *types.SearchCriteria) (*types.SearchResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if err := c.selectFolder(folder); err != nil {
		return nil, err
	}

	// Build IMAP search criteria
	imapCriteria := buildSearchCriteria(criteria)

	// Execute search
	searchOpts := &imap.SearchOptions{
		ReturnAll:   true,
		ReturnCount: true,
	}

	searchCmd := c.client.UIDSearch(imapCriteria, searchOpts)
	data, err := searchCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert UIDs
	uids := make([]uint32, len(data.AllUIDs()))
	for i, uid := range data.AllUIDs() {
		uids[i] = uint32(uid)
	}

	return &types.SearchResult{
		UIDs:  uids,
		Total: len(uids),
	}, nil
}

// SearchUnread searches for unread messages
func (c *Client) SearchUnread(folder string) (*types.SearchResult, error) {
	seen := false
	return c.Search(folder, &types.SearchCriteria{
		Seen: &seen,
	})
}

// SearchFlagged searches for flagged/starred messages
func (c *Client) SearchFlagged(folder string) (*types.SearchResult, error) {
	flagged := true
	return c.Search(folder, &types.SearchCriteria{
		Flagged: &flagged,
	})
}

// SearchFrom searches for messages from a specific sender
func (c *Client) SearchFrom(folder string, from string) (*types.SearchResult, error) {
	return c.Search(folder, &types.SearchCriteria{
		From: from,
	})
}

// SearchSubject searches for messages with specific subject text
func (c *Client) SearchSubject(folder string, subject string) (*types.SearchResult, error) {
	return c.Search(folder, &types.SearchCriteria{
		Subject: subject,
	})
}

// SearchBody searches for messages with specific body text
func (c *Client) SearchBody(folder string, body string) (*types.SearchResult, error) {
	return c.Search(folder, &types.SearchCriteria{
		Body: body,
	})
}

// SearchDateRange searches for messages within a date range
func (c *Client) SearchDateRange(folder string, since, before *time.Time) (*types.SearchResult, error) {
	return c.Search(folder, &types.SearchCriteria{
		Since:  since,
		Before: before,
	})
}

// SearchToday searches for messages received today
func (c *Client) SearchToday(folder string) (*types.SearchResult, error) {
	today := time.Now().Truncate(24 * time.Hour)
	return c.Search(folder, &types.SearchCriteria{
		Since: &today,
	})
}

// buildSearchCriteria converts our criteria to IMAP search criteria
func buildSearchCriteria(criteria *types.SearchCriteria) *imap.SearchCriteria {
	if criteria == nil {
		return &imap.SearchCriteria{}
	}

	sc := &imap.SearchCriteria{}

	// Text searches
	if criteria.From != "" {
		sc.Header = append(sc.Header, imap.SearchCriteriaHeaderField{
			Key:   "From",
			Value: criteria.From,
		})
	}

	if criteria.To != "" {
		sc.Header = append(sc.Header, imap.SearchCriteriaHeaderField{
			Key:   "To",
			Value: criteria.To,
		})
	}

	if criteria.Subject != "" {
		sc.Header = append(sc.Header, imap.SearchCriteriaHeaderField{
			Key:   "Subject",
			Value: criteria.Subject,
		})
	}

	if criteria.Body != "" {
		sc.Body = []string{criteria.Body}
	}

	if criteria.Text != "" {
		sc.Text = []string{criteria.Text}
	}

	// Date filters
	if criteria.Since != nil {
		sc.Since = *criteria.Since
	}

	if criteria.Before != nil {
		sc.Before = *criteria.Before
	}

	// Note: "On" date filter is handled by combining Since and Before
	// since go-imap/v2 doesn't have a direct On field
	if criteria.On != nil {
		sc.Since = *criteria.On
		nextDay := criteria.On.AddDate(0, 0, 1)
		sc.Before = nextDay
	}

	// Flag filters
	if criteria.Seen != nil {
		if *criteria.Seen {
			sc.Flag = append(sc.Flag, imap.FlagSeen)
		} else {
			sc.NotFlag = append(sc.NotFlag, imap.FlagSeen)
		}
	}

	if criteria.Answered != nil {
		if *criteria.Answered {
			sc.Flag = append(sc.Flag, imap.FlagAnswered)
		} else {
			sc.NotFlag = append(sc.NotFlag, imap.FlagAnswered)
		}
	}

	if criteria.Flagged != nil {
		if *criteria.Flagged {
			sc.Flag = append(sc.Flag, imap.FlagFlagged)
		} else {
			sc.NotFlag = append(sc.NotFlag, imap.FlagFlagged)
		}
	}

	if criteria.Deleted != nil {
		if *criteria.Deleted {
			sc.Flag = append(sc.Flag, imap.FlagDeleted)
		} else {
			sc.NotFlag = append(sc.NotFlag, imap.FlagDeleted)
		}
	}

	if criteria.Draft != nil {
		if *criteria.Draft {
			sc.Flag = append(sc.Flag, imap.FlagDraft)
		} else {
			sc.NotFlag = append(sc.NotFlag, imap.FlagDraft)
		}
	}

	// Size filters
	if criteria.Larger > 0 {
		sc.Larger = int64(criteria.Larger)
	}

	if criteria.Smaller > 0 {
		sc.Smaller = int64(criteria.Smaller)
	}

	// UID range
	if criteria.UIDFrom > 0 || criteria.UIDto > 0 {
		from := imap.UID(criteria.UIDFrom)
		if from == 0 {
			from = 1
		}
		to := imap.UID(criteria.UIDto)
		if to == 0 {
			to = imap.UID(^uint32(0)) // Max UID
		}

		sc.UID = []imap.UIDSet{imap.UIDSetNum(from, to)}
	}

	// OR criteria
	if len(criteria.Or) > 0 {
		for _, orCriteria := range criteria.Or {
			orSc := buildSearchCriteria(&orCriteria)
			sc.Or = append(sc.Or, [2]imap.SearchCriteria{*sc, *orSc})
		}
	}

	// NOT criteria
	if criteria.Not != nil {
		notSc := buildSearchCriteria(criteria.Not)
		sc.Not = append(sc.Not, *notSc)
	}

	return sc
}
