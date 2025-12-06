package smtp

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Client represents an SMTP client for sending emails
type Client struct {
	config SMTPConfig
	from   string // From address for this account
}

// NewClient creates a new SMTP client
func NewClient(config SMTPConfig, from string) *Client {
	// Set defaults
	if config.Port == 0 {
		if config.TLS {
			config.Port = 465 // SMTPS
		} else {
			config.Port = 587 // STARTTLS
		}
	}

	return &Client{
		config: config,
		from:   from,
	}
}

// Send sends an email message
func (c *Client) Send(msg *OutgoingMessage) (*SendResult, error) {
	// Set From if not set
	if msg.From == "" {
		msg.From = c.from
	}

	// Generate Message-ID if not set
	if msg.MessageID == "" {
		domain := extractDomain(msg.From)
		msg.MessageID = fmt.Sprintf("<%s@%s>", uuid.New().String(), domain)
	}

	// Set Date if not set
	if msg.Date.IsZero() {
		msg.Date = time.Now()
	}

	// Build email content
	content := c.buildEmail(msg)

	// Collect all recipients
	recipients := append([]string{}, msg.To...)
	recipients = append(recipients, msg.Cc...)
	recipients = append(recipients, msg.Bcc...)

	if len(recipients) == 0 {
		return &SendResult{Success: false, Error: "no recipients"}, fmt.Errorf("no recipients")
	}

	// Send email
	var err error
	if c.config.TLS && c.config.Port == 465 {
		err = c.sendTLS(msg.From, recipients, content)
	} else {
		err = c.sendStartTLS(msg.From, recipients, content)
	}

	if err != nil {
		return &SendResult{Success: false, Error: err.Error()}, err
	}

	return &SendResult{
		Success:   true,
		MessageID: msg.MessageID,
	}, nil
}

// sendStartTLS sends email using STARTTLS (port 587)
func (c *Client) sendStartTLS(from string, to []string, content []byte) error {
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	// Create auth
	auth := smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)

	// Send mail
	return smtp.SendMail(addr, auth, from, to, content)
}

// sendTLS sends email using direct TLS (port 465)
func (c *Client) sendTLS(from string, to []string, content []byte) error {
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	// Connect with TLS
	tlsConfig := &tls.Config{
		ServerName: c.config.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, c.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Authenticate
	auth := smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", rcpt, err)
		}
	}

	// Send data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start data: %w", err)
	}

	_, err = w.Write(content)
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data: %w", err)
	}

	return client.Quit()
}

// buildEmail builds the email content with headers and body
func (c *Client) buildEmail(msg *OutgoingMessage) []byte {
	var b strings.Builder

	// Required headers
	b.WriteString(fmt.Sprintf("From: %s\r\n", msg.From))
	b.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	if len(msg.Cc) > 0 {
		b.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.Cc, ", ")))
	}
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", encodeSubject(msg.Subject)))
	b.WriteString(fmt.Sprintf("Date: %s\r\n", msg.Date.Format("Mon, 02 Jan 2006 15:04:05 -0700")))
	b.WriteString(fmt.Sprintf("Message-ID: %s\r\n", msg.MessageID))

	// Threading headers
	if msg.InReplyTo != "" {
		b.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", msg.InReplyTo))
	}
	if len(msg.References) > 0 {
		b.WriteString(fmt.Sprintf("References: %s\r\n", strings.Join(msg.References, " ")))
	}

	// MIME headers
	b.WriteString("MIME-Version: 1.0\r\n")

	if msg.HTML != "" && msg.Text != "" {
		// Multipart message
		boundary := "boundary-" + uuid.New().String()[:8]
		b.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		b.WriteString("\r\n")

		// Plain text part - use base64 for UTF-8 safety
		b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		b.WriteString("Content-Transfer-Encoding: base64\r\n")
		b.WriteString("\r\n")
		b.WriteString(encodeBase64Body(msg.Text))
		b.WriteString("\r\n")

		// HTML part - use base64 for UTF-8 safety
		b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		b.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		b.WriteString("Content-Transfer-Encoding: base64\r\n")
		b.WriteString("\r\n")
		b.WriteString(encodeBase64Body(msg.HTML))
		b.WriteString("\r\n")

		b.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if msg.HTML != "" {
		// HTML only - use base64 for UTF-8 safety
		b.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		b.WriteString("Content-Transfer-Encoding: base64\r\n")
		b.WriteString("\r\n")
		b.WriteString(encodeBase64Body(msg.HTML))
	} else {
		// Plain text only - use base64 for UTF-8 safety
		b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		b.WriteString("Content-Transfer-Encoding: base64\r\n")
		b.WriteString("\r\n")
		b.WriteString(encodeBase64Body(msg.Text))
	}

	return []byte(b.String())
}

// Helper functions

func extractDomain(email string) string {
	// Handle "Name <email@domain.com>" format
	if start := strings.Index(email, "<"); start != -1 {
		if end := strings.Index(email, ">"); end > start {
			email = email[start+1 : end]
		}
	}

	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return "localhost"
}

func encodeSubject(subject string) string {
	// Check if subject needs encoding (contains non-ASCII)
	needsEncoding := false
	for _, r := range subject {
		if r > 127 {
			needsEncoding = true
			break
		}
	}

	if !needsEncoding {
		return subject
	}

	// Use RFC 2047 encoding with standard library
	return fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
}

// encodeBase64Body encodes body text as base64 with proper line wrapping (76 chars per line)
func encodeBase64Body(text string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))

	// Wrap at 76 characters per RFC 2045
	var wrapped strings.Builder
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		wrapped.WriteString(encoded[i:end])
		if end < len(encoded) {
			wrapped.WriteString("\r\n")
		}
	}

	return wrapped.String()
}
