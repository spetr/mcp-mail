package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spetr/mcp-mail/smtp"
	"github.com/spetr/mcp-mail/types"
)

func (r *Registry) registerSendTools() {
	// message_send - Send a new email
	r.addTool(
		mcp.NewTool("message_send",
			mcp.WithDescription("Send a new email message. Requires SMTP to be configured for the account."),
			requiredString("account_id", "Account ID"),
			requiredString("to", "Recipient email address(es), comma-separated for multiple"),
			requiredString("subject", "Email subject"),
			requiredString("body", "Email body (plain text)"),
			optionalString("cc", "CC recipients, comma-separated"),
			optionalString("bcc", "BCC recipients, comma-separated"),
			optionalString("html", "HTML body (optional, for rich text)"),
		),
		r.handleMessageSend,
	)

	// message_reply - Reply to an email
	r.addTool(
		mcp.NewTool("message_reply",
			mcp.WithDescription("Reply to an email. Automatically sets In-Reply-To and References headers for threading, and marks the original message as answered."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder containing the message"),
			requiredNumber("uid", "UID of the message to reply to"),
			requiredString("body", "Reply body (plain text)"),
			optionalBool("reply_all", "Reply to all recipients (default false)"),
			optionalString("html", "HTML body (optional)"),
		),
		r.handleMessageReply,
	)

	// message_forward - Forward an email
	r.addTool(
		mcp.NewTool("message_forward",
			mcp.WithDescription("Forward an email to another recipient."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder containing the message"),
			requiredNumber("uid", "UID of the message to forward"),
			requiredString("to", "Forward recipient(s), comma-separated"),
			optionalString("body", "Additional message to include (plain text)"),
			optionalString("html", "Additional message in HTML"),
		),
		r.handleMessageForward,
	)
}

func (r *Registry) handleMessageSend(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	to := request.GetString("to", "")
	subject := request.GetString("subject", "")
	body := request.GetString("body", "")
	cc := request.GetString("cc", "")
	bcc := request.GetString("bcc", "")
	html := request.GetString("html", "")

	if to == "" {
		return mcp.NewToolResultError("recipient (to) is required"), nil
	}
	if subject == "" {
		return mcp.NewToolResultError("subject is required"), nil
	}
	if body == "" && html == "" {
		return mcp.NewToolResultError("body is required"), nil
	}

	// Parse recipients
	toList := parseRecipients(to)
	ccList := parseRecipients(cc)
	bccList := parseRecipients(bcc)

	// Build message
	msg := &smtp.OutgoingMessage{
		To:      toList,
		Cc:      ccList,
		Bcc:     bccList,
		Subject: subject,
		Text:    body,
		HTML:    html,
	}

	// Send
	result, err := r.smtpMgr.Send(accountID, msg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to send email: %v", err)), nil
	}

	// Update statistics
	if r.memoryMgr != nil {
		r.memoryMgr.IncrementStat(accountID, "emails_sent", 1)
	}

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (r *Registry) handleMessageReply(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := uint32(request.GetInt("uid", 0))
	body := request.GetString("body", "")
	replyAll := request.GetBool("reply_all", false)
	html := request.GetString("html", "")

	if folder == "" {
		return mcp.NewToolResultError("folder is required"), nil
	}
	if uid == 0 {
		return mcp.NewToolResultError("uid is required"), nil
	}
	if body == "" && html == "" {
		return mcp.NewToolResultError("body is required"), nil
	}

	// Get original message
	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	original, err := client.GetMessage(folder, uid)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get original message: %v", err)), nil
	}

	// Get our own email address to filter out from CC
	cfg := r.configMgr.Get()
	var ownEmail string
	for _, acc := range cfg.Accounts {
		if acc.ID == accountID {
			ownEmail = strings.ToLower(acc.Username)
			if acc.FromAddress != "" {
				ownEmail = strings.ToLower(acc.FromAddress)
			}
			break
		}
	}

	// Build recipients
	var toList []string
	if len(original.From) > 0 {
		toList = []string{original.From[0].Address}
	}

	var ccList []string
	if replyAll {
		// Add To recipients (except ourselves)
		for _, addr := range original.To {
			if addr.Address != "" && strings.ToLower(addr.Address) != ownEmail {
				ccList = append(ccList, addr.Address)
			}
		}
		// Add CC recipients (except ourselves)
		for _, addr := range original.Cc {
			if addr.Address != "" && strings.ToLower(addr.Address) != ownEmail {
				ccList = append(ccList, addr.Address)
			}
		}
	}

	// Build subject
	subject := original.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}

	// Build References header
	var references []string
	references = append(references, original.References...)
	if original.MessageID != "" {
		references = append(references, original.MessageID)
	}

	// Build message
	msg := &smtp.OutgoingMessage{
		To:         toList,
		Cc:         ccList,
		Subject:    subject,
		Text:       body,
		HTML:       html,
		InReplyTo:  original.MessageID,
		References: references,
	}

	// Send
	result, err := r.smtpMgr.Send(accountID, msg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to send reply: %v", err)), nil
	}

	// Mark original as answered
	originalMarked := false
	if result.Success {
		markErr := client.MarkAnswered(folder, uid)
		originalMarked = markErr == nil

		// Update statistics and tracking
		if r.memoryMgr != nil {
			r.memoryMgr.IncrementStat(accountID, "emails_sent", 1)

			// Track that we replied to this sender
			if len(original.From) > 0 {
				r.memoryMgr.TrackMessageAnswered(accountID, original.From[0].Address)
			}
		}
	}

	resultJSON, _ := json.Marshal(map[string]interface{}{
		"success":          result.Success,
		"message_id":       result.MessageID,
		"original_marked":  originalMarked,
		"in_reply_to":      original.MessageID,
	})
	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (r *Registry) handleMessageForward(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := uint32(request.GetInt("uid", 0))
	to := request.GetString("to", "")
	body := request.GetString("body", "")
	html := request.GetString("html", "")

	if folder == "" {
		return mcp.NewToolResultError("folder is required"), nil
	}
	if uid == 0 {
		return mcp.NewToolResultError("uid is required"), nil
	}
	if to == "" {
		return mcp.NewToolResultError("recipient (to) is required"), nil
	}

	// Get original message
	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	original, err := client.GetMessage(folder, uid)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get original message: %v", err)), nil
	}

	// Build subject
	subject := original.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "fwd:") &&
		!strings.HasPrefix(strings.ToLower(subject), "fw:") {
		subject = "Fwd: " + subject
	}

	// Build forwarded body
	var forwardedBody strings.Builder
	if body != "" {
		forwardedBody.WriteString(body)
		forwardedBody.WriteString("\n\n")
	}
	forwardedBody.WriteString("---------- Forwarded message ----------\n")
	if len(original.From) > 0 {
		forwardedBody.WriteString(fmt.Sprintf("From: %s <%s>\n", original.From[0].Name, original.From[0].Address))
	}
	forwardedBody.WriteString(fmt.Sprintf("Date: %s\n", original.Date.Format("Mon, 02 Jan 2006 15:04:05 -0700")))
	forwardedBody.WriteString(fmt.Sprintf("Subject: %s\n", original.Subject))
	toStr := formatAddressesFromTypes(original.To)
	if toStr != "" {
		forwardedBody.WriteString(fmt.Sprintf("To: %s\n", toStr))
	}
	forwardedBody.WriteString("\n")
	forwardedBody.WriteString(original.Body.Text)

	// Build HTML body if original has HTML
	var forwardedHTML string
	if original.Body.HTML != "" || html != "" {
		var htmlBuilder strings.Builder
		if html != "" {
			htmlBuilder.WriteString(html)
			htmlBuilder.WriteString("<br><br>")
		}
		htmlBuilder.WriteString("<hr><b>---------- Forwarded message ----------</b><br>")
		if len(original.From) > 0 {
			htmlBuilder.WriteString(fmt.Sprintf("<b>From:</b> %s &lt;%s&gt;<br>", original.From[0].Name, original.From[0].Address))
		}
		htmlBuilder.WriteString(fmt.Sprintf("<b>Date:</b> %s<br>", original.Date.Format("Mon, 02 Jan 2006 15:04:05 -0700")))
		htmlBuilder.WriteString(fmt.Sprintf("<b>Subject:</b> %s<br>", original.Subject))
		if toStr != "" {
			htmlBuilder.WriteString(fmt.Sprintf("<b>To:</b> %s<br>", toStr))
		}
		htmlBuilder.WriteString("<br>")
		if original.Body.HTML != "" {
			htmlBuilder.WriteString(original.Body.HTML)
		} else {
			htmlBuilder.WriteString("<pre>")
			htmlBuilder.WriteString(original.Body.Text)
			htmlBuilder.WriteString("</pre>")
		}
		forwardedHTML = htmlBuilder.String()
	}

	// Build message
	msg := &smtp.OutgoingMessage{
		To:      parseRecipients(to),
		Subject: subject,
		Text:    forwardedBody.String(),
		HTML:    forwardedHTML,
	}

	// Send
	result, err := r.smtpMgr.Send(accountID, msg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to forward email: %v", err)), nil
	}

	// Update statistics
	if r.memoryMgr != nil && result.Success {
		r.memoryMgr.IncrementStat(accountID, "emails_sent", 1)
	}

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// Helper functions

func parseRecipients(recipients string) []string {
	if recipients == "" {
		return nil
	}
	parts := strings.Split(recipients, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func formatAddressesFromTypes(addrs []types.Address) string {
	var parts []string
	for _, addr := range addrs {
		if addr.Name != "" {
			parts = append(parts, fmt.Sprintf("%s <%s>", addr.Name, addr.Address))
		} else {
			parts = append(parts, addr.Address)
		}
	}
	return strings.Join(parts, ", ")
}
