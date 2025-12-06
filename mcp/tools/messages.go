package tools

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spetr/mcp-mail/types"
)

// generateSafeID creates an HMAC-based identifier for safe destructive operations
// Format: hmac_hash (12 chars) - cryptographically bound to UID, from, subject, and account secret
// This prevents AI from creating valid safe_ids without knowing the secret
func generateSafeID(secret string, uid uint32, from string, subject string) string {
	// Create message to sign: uid|from|subject
	message := fmt.Sprintf("%d|%s|%s", uid, strings.ToLower(from), subject)

	// Generate HMAC-SHA256
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	hash := h.Sum(nil)

	// Return first 12 hex chars (48 bits of entropy - sufficient for validation)
	return hex.EncodeToString(hash)[:12]
}

// validateSafeID verifies that safe_id matches the actual message using HMAC
func validateSafeID(secret string, safeID string, uid uint32, msg *types.MessageEnvelope) error {
	if uid != msg.UID {
		return fmt.Errorf("UID mismatch: provided %d, message has %d", uid, msg.UID)
	}

	// Get from address
	fromAddr := ""
	if len(msg.From) > 0 {
		fromAddr = msg.From[0].Address
	}

	// Generate expected safe_id
	expectedSafeID := generateSafeID(secret, msg.UID, fromAddr, msg.Subject)

	if safeID != expectedSafeID {
		return fmt.Errorf("safe_id mismatch: the provided safe_id does not match this message")
	}

	return nil
}


// truncate shortens a string to max length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func (r *Registry) registerMessageTools() {
	// message_list - List messages in a folder
	r.addTool(
		mcp.NewTool("message_list",
			mcp.WithDescription("List messages in a folder with pagination"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name (e.g., INBOX)"),
			optionalNumber("limit", "Maximum number of messages to return (default: 50)"),
			optionalNumber("offset", "Number of messages to skip (default: 0)"),
		),
		r.handleMessageList,
	)

	// message_get - Get a complete message
	r.addTool(
		mcp.NewTool("message_get",
			mcp.WithDescription("Get a complete message including body and attachments info"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
		),
		r.handleMessageGet,
	)

	// message_get_headers - Get message headers only
	r.addTool(
		mcp.NewTool("message_get_headers",
			mcp.WithDescription("Get message headers without body (faster)"),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
		),
		r.handleMessageGetHeaders,
	)

	// message_delete - Delete a message (moves to trash)
	r.addTool(
		mcp.NewTool("message_delete",
			mcp.WithDescription("Delete a message by moving it to trash. Requires both uid AND safe_id from message_list/message_get to prevent accidental deletion. Messages are NOT permanently deleted."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Folder name"),
			requiredNumber("uid", "Message UID"),
			requiredString("safe_id", "Safe message identifier (HMAC token from message_list/message_get response)"),
		),
		r.handleMessageDelete,
	)

	// message_move - Move a message to another folder
	r.addTool(
		mcp.NewTool("message_move",
			mcp.WithDescription("Move a message to another folder. Requires both uid AND safe_id to prevent accidental moves."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Source folder name"),
			requiredNumber("uid", "Message UID"),
			requiredString("safe_id", "Safe message identifier (HMAC token from message_list/message_get response)"),
			requiredString("target_folder", "Destination folder name"),
		),
		r.handleMessageMove,
	)

	// message_copy - Copy a message to another folder
	r.addTool(
		mcp.NewTool("message_copy",
			mcp.WithDescription("Copy a message to another folder. Requires both uid AND safe_id to ensure correct message is copied."),
			requiredString("account_id", "Account ID"),
			requiredString("folder", "Source folder name"),
			requiredNumber("uid", "Message UID"),
			requiredString("safe_id", "Safe message identifier (HMAC token from message_list/message_get response)"),
			requiredString("target_folder", "Destination folder name"),
		),
		r.handleMessageCopy,
	)
}

func (r *Registry) handleMessageList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	limit := getNumber(request, "limit", 50)
	offset := getNumber(request, "offset", 0)

	// Get account secret for HMAC-based safe_id
	secret, err := r.getAccountSecret(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get account secret: %v", err)), nil
	}

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	messages, err := client.ListMessages(folder, limit, offset)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list messages: %v", err)), nil
	}

	// Build enriched messages with safe_id for destructive operations
	type MessageWithSafeID struct {
		types.MessageListItem
		SafeID string `json:"safe_id"`
	}

	enrichedMessages := make([]MessageWithSafeID, len(messages))
	for i, msg := range messages {
		fromAddr := ""
		if len(msg.From) > 0 {
			fromAddr = msg.From[0].Address
		}
		enrichedMessages[i] = MessageWithSafeID{
			MessageListItem: msg,
			SafeID:          generateSafeID(secret, msg.UID, fromAddr, msg.Subject),
		}
	}

	// Build enriched response with memory context
	response := map[string]interface{}{
		"messages":   enrichedMessages,
		"account_id": accountID,
		"folder":     folder,
		"count":      len(enrichedMessages),
	}

	// Add memory context if available
	if r.memoryMgr != nil {
		mem, err := r.memoryMgr.GetOrLoad(accountID)
		if err == nil {
			// Find important/known senders in this list
			var importantSenders []string
			var knownSenders []map[string]interface{}

			for _, msg := range enrichedMessages {
				if len(msg.From) > 0 {
					email := strings.ToLower(msg.From[0].Address)

					// Check if sender is in important contacts
					for _, contact := range mem.ImportantContacts {
						if strings.ToLower(contact.Email) == email {
							importantSenders = append(importantSenders, fmt.Sprintf("%s (%s)", email, contact.Role))
							break
						}
					}

					// Check sender profile
					if sp, exists := mem.Profile.SenderProfiles[email]; exists {
						if sp.Importance == "high" || sp.Importance == "low" || sp.Importance == "ignore" {
							knownSenders = append(knownSenders, map[string]interface{}{
								"email":      email,
								"importance": sp.Importance,
								"type":       sp.Type,
							})
						}
					}
				}
			}

			if len(importantSenders) > 0 {
				response["important_senders_in_list"] = importantSenders
			}
			if len(knownSenders) > 0 {
				response["known_senders_in_list"] = knownSenders
			}

			// Add contextual hint
			response["memory_hint"] = "📝 When you read an email and learn something important about a sender (e.g., user says 'this is important' or 'ignore these'), use memory_set_sender to save it."
		}
	}

	result, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleMessageGet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)

	// Get account secret for HMAC-based safe_id
	secret, err := r.getAccountSecret(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get account secret: %v", err)), nil
	}

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	message, err := client.GetMessage(folder, uid)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get message: %v", err)), nil
	}

	// Generate safe_id for destructive operations
	fromAddr := ""
	if len(message.From) > 0 {
		fromAddr = message.From[0].Address
	}
	safeID := generateSafeID(secret, message.UID, fromAddr, message.Subject)

	// Build enriched response
	response := map[string]interface{}{
		"message":    message,
		"safe_id":    safeID,
		"account_id": accountID,
		"folder":     folder,
	}

	// Add sender context from memory - only include useful info
	if r.memoryMgr != nil && len(message.From) > 0 {
		senderEmail := strings.ToLower(message.From[0].Address)

		mem, err := r.memoryMgr.GetOrLoad(accountID)
		if err == nil {
			var senderContext map[string]interface{}
			var importance string

			// Check sender profile - only include if has useful classification
			if sp, exists := mem.Profile.SenderProfiles[senderEmail]; exists {
				importance = sp.Importance
				// Only include context if sender is classified
				if sp.Importance != "" || sp.Relationship != "" || sp.Notes != "" {
					senderContext = map[string]interface{}{
						"importance": sp.Importance,
					}
					if sp.Type != "" {
						senderContext["type"] = sp.Type
					}
					if sp.Relationship != "" {
						senderContext["relationship"] = sp.Relationship
					}
					if sp.Notes != "" {
						senderContext["notes"] = sp.Notes
					}
				}
			}

			// Check if in important contacts
			for _, contact := range mem.ImportantContacts {
				if strings.ToLower(contact.Email) == senderEmail {
					if senderContext == nil {
						senderContext = make(map[string]interface{})
					}
					senderContext["is_important_contact"] = true
					senderContext["contact_role"] = contact.Role
					importance = "high"
					break
				}
			}

			// Check unwanted rules
			for _, unwanted := range mem.Unwanted.Senders {
				if matchesPattern(unwanted, senderEmail) {
					if senderContext == nil {
						senderContext = make(map[string]interface{})
					}
					senderContext["is_unwanted"] = true
					importance = "ignore"
					break
				}
			}

			if senderContext != nil {
				response["sender_context"] = senderContext
			}

			// Add concise hint based on context
			if importance == "high" {
				response["memory_hint"] = "⭐ HIGH importance sender"
			} else if importance == "ignore" {
				response["memory_hint"] = "🔕 IGNORE sender"
			}
			// Don't add hints for unknown senders - saves context
		}
	}

	result, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

// calculateResponseRate calculates response rate as percentage
func calculateResponseRate(answered, seen int) float64 {
	if seen == 0 {
		return 0
	}
	return float64(answered) / float64(seen) * 100
}

// matchesPattern checks if email matches a wildcard pattern like *@domain.com
func matchesPattern(pattern, email string) bool {
	pattern = strings.ToLower(pattern)
	email = strings.ToLower(email)

	if !strings.Contains(pattern, "*") {
		return pattern == email
	}

	// Handle *@domain.com pattern
	if strings.HasPrefix(pattern, "*@") {
		domain := pattern[2:]
		return strings.HasSuffix(email, "@"+domain)
	}

	// Handle user@* pattern
	if strings.HasSuffix(pattern, "@*") {
		prefix := pattern[:len(pattern)-2]
		return strings.HasPrefix(email, prefix+"@")
	}

	// Handle *substring* pattern
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		substr := pattern[1 : len(pattern)-1]
		return strings.Contains(email, substr)
	}

	return false
}

func (r *Registry) handleMessageGetHeaders(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	headers, err := client.GetMessageHeaders(folder, uid)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get message headers: %v", err)), nil
	}

	result, err := json.Marshal(headers)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (r *Registry) handleMessageDelete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)
	safeID := request.GetString("safe_id", "")

	// Get account secret for HMAC validation
	secret, err := r.getAccountSecret(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get account secret: %v", err)), nil
	}

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	// Fetch message headers to validate safe_id
	headers, err := client.GetMessageHeaders(folder, uid)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get message: %v", err)), nil
	}

	// Validate that safe_id matches the actual message using HMAC
	if err := validateSafeID(secret, safeID, uid, headers); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("safety check failed: %v - the message may have changed or safe_id is incorrect", err)), nil
	}

	// Get trash folder - either from config or auto-detect
	trashFolder := r.protectMgr.GetTrashFolder()
	if trashFolder == "" {
		trashFolder = client.GetTrashFolder()
	}

	if trashFolder == "" {
		return mcp.NewToolResultError("cannot delete: trash folder not found"), nil
	}

	if folder == trashFolder {
		return mcp.NewToolResultError("cannot delete messages from trash folder"), nil
	}

	// Move to trash
	if err := client.MoveMessage(folder, uid, trashFolder); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to move message to trash: %v", err)), nil
	}

	// Return with message info for confirmation
	fromAddr := ""
	if len(headers.From) > 0 {
		fromAddr = headers.From[0].Address
	}
	return mcp.NewToolResultText(fmt.Sprintf("Message deleted: UID %d from '%s' with subject '%s' moved to trash (%s)",
		uid, fromAddr, truncate(headers.Subject, 50), trashFolder)), nil
}

func (r *Registry) handleMessageMove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)
	safeID := request.GetString("safe_id", "")
	targetFolder := request.GetString("target_folder", "")

	// Get account secret for HMAC validation
	secret, err := r.getAccountSecret(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get account secret: %v", err)), nil
	}

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	// Fetch message headers to validate safe_id
	headers, err := client.GetMessageHeaders(folder, uid)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get message: %v", err)), nil
	}

	// Validate that safe_id matches the actual message using HMAC
	if err := validateSafeID(secret, safeID, uid, headers); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("safety check failed: %v - the message may have changed or safe_id is incorrect", err)), nil
	}

	if err := client.MoveMessage(folder, uid, targetFolder); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to move message: %v", err)), nil
	}

	// Return with message info for confirmation
	fromAddr := ""
	if len(headers.From) > 0 {
		fromAddr = headers.From[0].Address
	}
	return mcp.NewToolResultText(fmt.Sprintf("Message moved: UID %d from '%s' with subject '%s' moved to '%s'",
		uid, fromAddr, truncate(headers.Subject, 50), targetFolder)), nil
}

func (r *Registry) handleMessageCopy(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	folder := request.GetString("folder", "")
	uid := getUint32(request, "uid", 0)
	safeID := request.GetString("safe_id", "")
	targetFolder := request.GetString("target_folder", "")

	// Get account secret for HMAC validation
	secret, err := r.getAccountSecret(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get account secret: %v", err)), nil
	}

	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	// Fetch message headers to validate safe_id
	headers, err := client.GetMessageHeaders(folder, uid)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get message: %v", err)), nil
	}

	// Validate that safe_id matches the actual message using HMAC
	if err := validateSafeID(secret, safeID, uid, headers); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("safety check failed: %v - the message may have changed or safe_id is incorrect", err)), nil
	}

	if err := client.CopyMessage(folder, uid, targetFolder); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to copy message: %v", err)), nil
	}

	// Return with message info for confirmation
	fromAddr := ""
	if len(headers.From) > 0 {
		fromAddr = headers.From[0].Address
	}
	return mcp.NewToolResultText(fmt.Sprintf("Message copied: UID %d from '%s' with subject '%s' copied to '%s'",
		uid, fromAddr, truncate(headers.Subject, 50), targetFolder)), nil
}
