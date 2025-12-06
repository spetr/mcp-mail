package imap

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/spetr/mcp-mail/types"
)

// ListMessages returns messages in a folder
func (c *Client) ListMessages(folder string, limit, offset int) ([]types.MessageListItem, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	// Select the folder - this returns message count, no need for separate Status call
	selectData, err := c.client.Select(folder, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select folder '%s': %w", folder, err)
	}
	c.selected = folder

	if selectData.NumMessages == 0 {
		return []types.MessageListItem{}, nil
	}

	total := int(selectData.NumMessages)

	if limit <= 0 {
		limit = 50 // default limit
	}

	// Calculate range (fetch newest messages first)
	end := total - offset
	if end <= 0 {
		return []types.MessageListItem{}, nil
	}

	start := end - limit + 1
	if start < 1 {
		start = 1
	}

	// Create sequence set with range
	var seqSet imap.SeqSet
	seqSet.AddRange(uint32(start), uint32(end))

	// Fetch options
	fetchOpts := &imap.FetchOptions{
		UID:           true,
		Flags:         true,
		Envelope:      true,
		BodyStructure: &imap.FetchItemBodyStructure{},
	}

	fetchCmd := c.client.Fetch(seqSet, fetchOpts)
	defer fetchCmd.Close()

	messages := make([]types.MessageListItem, 0, limit)
	var fetchErrors int
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		buf, err := msg.Collect()
		if err != nil {
			fetchErrors++
			log.Printf("Warning: Failed to collect message data: %v", err)
			continue
		}

		item := types.MessageListItem{
			UID:            uint32(buf.UID),
			Flags:          convertFlags(buf.Flags),
			HasAttachments: hasAttachments(buf.BodyStructure),
		}

		if buf.Envelope != nil {
			item.Subject = buf.Envelope.Subject
			item.Date = buf.Envelope.Date
			item.From = convertAddresses(buf.Envelope.From)
		}

		messages = append(messages, item)
	}

	if fetchErrors > 0 {
		log.Printf("Warning: %d message(s) failed to load in folder %s", fetchErrors, folder)
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}

	// Reverse to get newest first
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetMessage returns a complete message by UID
func (c *Client) GetMessage(folder string, uid uint32) (*types.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if err := c.selectFolder(folder); err != nil {
		return nil, err
	}

	// Fetch the message
	uidSet := imap.UIDSetNum(imap.UID(uid))

	fetchOpts := &imap.FetchOptions{
		UID:           true,
		Flags:         true,
		Envelope:      true,
		BodyStructure: &imap.FetchItemBodyStructure{},
		BodySection:   []*imap.FetchItemBodySection{{}}, // Fetch full body
	}

	fetchCmd := c.client.Fetch(uidSet, fetchOpts)
	defer fetchCmd.Close()

	msg := fetchCmd.Next()
	if msg == nil {
		return nil, fmt.Errorf("message with UID %d not found", uid)
	}

	buf, err := msg.Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to collect message data: %w", err)
	}

	result := &types.Message{
		MessageEnvelope: types.MessageEnvelope{
			UID:   uint32(buf.UID),
			Flags: convertFlags(buf.Flags),
		},
	}

	if buf.Envelope != nil {
		result.Subject = buf.Envelope.Subject
		result.Date = buf.Envelope.Date
		result.MessageID = buf.Envelope.MessageID
		if len(buf.Envelope.InReplyTo) > 0 {
			result.InReplyTo = buf.Envelope.InReplyTo[0]
		}
		result.From = convertAddresses(buf.Envelope.From)
		result.To = convertAddresses(buf.Envelope.To)
		result.Cc = convertAddresses(buf.Envelope.Cc)
		result.Bcc = convertAddresses(buf.Envelope.Bcc)
		result.ReplyTo = convertAddresses(buf.Envelope.ReplyTo)
	}

	// Parse body sections
	var parseErrors []string
	for _, literal := range buf.BodySection {
		if len(literal.Bytes) == 0 {
			continue
		}

		body, err := parseMessageBody(bytes.NewReader(literal.Bytes))
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("body parse: %v", err))
			continue
		}
		result.Body = *body
		result.Attachments = parseAttachments(bytes.NewReader(literal.Bytes))
	}

	if len(parseErrors) > 0 {
		log.Printf("Warning: Message UID %d had parse errors: %v", uid, parseErrors)
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}

	return result, nil
}

// GetMessageHeaders returns only the headers of a message
func (c *Client) GetMessageHeaders(folder string, uid uint32) (*types.MessageEnvelope, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if err := c.selectFolder(folder); err != nil {
		return nil, err
	}

	// Fetch only envelope and flags
	uidSet := imap.UIDSetNum(imap.UID(uid))

	fetchOpts := &imap.FetchOptions{
		UID:        true,
		Flags:      true,
		Envelope:   true,
		RFC822Size: true,
	}

	fetchCmd := c.client.Fetch(uidSet, fetchOpts)
	defer fetchCmd.Close()

	msg := fetchCmd.Next()
	if msg == nil {
		return nil, fmt.Errorf("message with UID %d not found", uid)
	}

	buf, err := msg.Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to collect message data: %w", err)
	}

	result := &types.MessageEnvelope{
		UID:   uint32(buf.UID),
		Flags: convertFlags(buf.Flags),
		Size:  uint32(buf.RFC822Size),
	}

	if buf.Envelope != nil {
		result.Subject = buf.Envelope.Subject
		result.Date = buf.Envelope.Date
		result.MessageID = buf.Envelope.MessageID
		if len(buf.Envelope.InReplyTo) > 0 {
			result.InReplyTo = buf.Envelope.InReplyTo[0]
		}
		result.From = convertAddresses(buf.Envelope.From)
		result.To = convertAddresses(buf.Envelope.To)
		result.Cc = convertAddresses(buf.Envelope.Cc)
		result.Bcc = convertAddresses(buf.Envelope.Bcc)
		result.ReplyTo = convertAddresses(buf.Envelope.ReplyTo)
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}

	return result, nil
}

// DeleteMessage deletes a message by UID
func (c *Client) DeleteMessage(folder string, uid uint32, permanent bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return err
	}

	if err := c.selectFolder(folder); err != nil {
		return err
	}

	uidSet := imap.UIDSetNum(imap.UID(uid))

	// Mark as deleted
	storeFlags := imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: true,
		Flags:  []imap.Flag{imap.FlagDeleted},
	}

	storeCmd := c.client.Store(uidSet, &storeFlags, nil)
	if err := storeCmd.Close(); err != nil {
		return fmt.Errorf("failed to mark message as deleted: %w", err)
	}

	// Expunge if permanent
	if permanent {
		expungeCmd := c.client.Expunge()
		if err := expungeCmd.Close(); err != nil {
			return fmt.Errorf("failed to expunge: %w", err)
		}
	}

	// Invalidate folder info cache (message count changed)
	if c.cache != nil {
		c.cache.InvalidateFolderContent(c.account.ID, folder)
	}

	return nil
}

// MoveMessage moves a message to another folder
func (c *Client) MoveMessage(folder string, uid uint32, targetFolder string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return err
	}

	if err := c.selectFolder(folder); err != nil {
		return err
	}

	uidSet := imap.UIDSetNum(imap.UID(uid))

	// Check if MOVE is supported
	if c.client.Caps().Has(imap.CapMove) {
		if _, err := c.client.Move(uidSet, targetFolder).Wait(); err != nil {
			return fmt.Errorf("failed to move message: %w", err)
		}
	} else {
		// Fallback: COPY + DELETE
		if _, err := c.client.Copy(uidSet, targetFolder).Wait(); err != nil {
			return fmt.Errorf("failed to copy message: %w", err)
		}

		storeFlags := imap.StoreFlags{
			Op:     imap.StoreFlagsAdd,
			Silent: true,
			Flags:  []imap.Flag{imap.FlagDeleted},
		}

		storeCmd := c.client.Store(uidSet, &storeFlags, nil)
		if err := storeCmd.Close(); err != nil {
			return fmt.Errorf("failed to mark original as deleted: %w", err)
		}

		expungeCmd := c.client.Expunge()
		if err := expungeCmd.Close(); err != nil {
			return fmt.Errorf("failed to expunge: %w", err)
		}
	}

	// Invalidate cache for both folders
	if c.cache != nil {
		c.cache.InvalidateFolderContent(c.account.ID, folder)
		c.cache.InvalidateFolderContent(c.account.ID, targetFolder)
	}

	return nil
}

// CopyMessage copies a message to another folder
func (c *Client) CopyMessage(folder string, uid uint32, targetFolder string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnected(); err != nil {
		return err
	}

	if err := c.selectFolder(folder); err != nil {
		return err
	}

	uidSet := imap.UIDSetNum(imap.UID(uid))

	if _, err := c.client.Copy(uidSet, targetFolder).Wait(); err != nil {
		return fmt.Errorf("failed to copy message: %w", err)
	}

	// Invalidate cache for target folder
	if c.cache != nil {
		c.cache.InvalidateFolderContent(c.account.ID, targetFolder)
	}

	return nil
}

// convertFlags converts IMAP flags to string slice
func convertFlags(flags []imap.Flag) []string {
	result := make([]string, 0, len(flags))
	for _, flag := range flags {
		result = append(result, string(flag))
	}
	return result
}

// convertAddresses converts IMAP addresses to our Address type
func convertAddresses(addrs []imap.Address) []types.Address {
	result := make([]types.Address, 0, len(addrs))
	for _, addr := range addrs {
		result = append(result, types.Address{
			Name:    addr.Name,
			Address: addr.Addr(),
		})
	}
	return result
}

// hasAttachments checks if body structure contains attachments
func hasAttachments(bs imap.BodyStructure) bool {
	if bs == nil {
		return false
	}

	switch b := bs.(type) {
	case *imap.BodyStructureSinglePart:
		disp := b.Disposition()
		if disp != nil && strings.EqualFold(disp.Value, "attachment") {
			return true
		}
	case *imap.BodyStructureMultiPart:
		for _, child := range b.Children {
			if hasAttachments(child) {
				return true
			}
		}
	}

	return false
}

// parseMessageBody parses the message body from raw bytes
func parseMessageBody(r io.Reader) (*types.MessageBody, error) {
	entity, err := message.Read(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read message entity: %w", err)
	}

	body := &types.MessageBody{}

	if mr := entity.MultipartReader(); mr != nil {
		// Multipart message
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Warning: Failed to read multipart: %v", err)
				break
			}

			contentType, _, err := part.Header.ContentType()
			if err != nil {
				log.Printf("Warning: Failed to get content type: %v", err)
				continue
			}

			partBody, err := io.ReadAll(part.Body)
			if err != nil {
				log.Printf("Warning: Failed to read part body: %v", err)
				continue
			}

			switch {
			case strings.HasPrefix(contentType, "text/plain"):
				if body.Text == "" {
					body.Text = string(partBody)
				}
			case strings.HasPrefix(contentType, "text/html"):
				if body.HTML == "" {
					body.HTML = string(partBody)
				}
			case strings.HasPrefix(contentType, "multipart/"):
				// Recursively handle nested multipart
				subBody, err := parseMessageBody(bytes.NewReader(partBody))
				if err != nil {
					log.Printf("Warning: Failed to parse nested multipart: %v", err)
					continue
				}
				if subBody != nil {
					if body.Text == "" {
						body.Text = subBody.Text
					}
					if body.HTML == "" {
						body.HTML = subBody.HTML
					}
				}
			}
		}
	} else {
		// Single part message
		contentType, _, err := entity.Header.ContentType()
		if err != nil {
			contentType = "text/plain" // default
		}

		partBody, err := io.ReadAll(entity.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read message body: %w", err)
		}

		switch {
		case strings.HasPrefix(contentType, "text/plain"):
			body.Text = string(partBody)
		case strings.HasPrefix(contentType, "text/html"):
			body.HTML = string(partBody)
		default:
			body.Text = string(partBody)
		}
	}

	return body, nil
}

// parseAttachments extracts attachment info from the message
func parseAttachments(r io.Reader) []types.MessageAttachment {
	attachments := make([]types.MessageAttachment, 0)

	entity, err := message.Read(r)
	if err != nil {
		log.Printf("Warning: Failed to read entity for attachments: %v", err)
		return attachments
	}

	reader := mail.NewReader(entity)
	partNum := 0

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Warning: Failed to read part for attachments: %v", err)
			break
		}

		partNum++

		// Check if this is an attachment
		if attachHeader, ok := part.Header.(*mail.AttachmentHeader); ok {
			filename, err := attachHeader.Filename()
			if err != nil {
				log.Printf("Warning: Failed to get attachment filename: %v", err)
				filename = fmt.Sprintf("attachment_%d", partNum)
			}
			if filename == "" {
				filename = fmt.Sprintf("attachment_%d", partNum)
			}

			contentType, _, err := attachHeader.ContentType()
			if err != nil {
				contentType = "application/octet-stream"
			}

			// Read to get size
			data, err := io.ReadAll(part.Body)
			if err != nil {
				log.Printf("Warning: Failed to read attachment body: %v", err)
				continue
			}

			attachments = append(attachments, types.MessageAttachment{
				PartID:      fmt.Sprintf("%d", partNum),
				Filename:    filename,
				ContentType: contentType,
				Size:        uint32(len(data)),
				Inline:      false,
			})
		}
	}

	return attachments
}
