# Security Mechanisms in MCP-IMAP Server

This document describes the security mechanisms implemented in the MCP-IMAP server to protect against accidental or malicious operations, especially when used with AI assistants.

## 1. HMAC-Based Safe Message Identifiers

### Problem
When AI assistants interact with email, they might "hallucinate" message UIDs or confuse messages, potentially leading to operations on wrong emails (deletion, moving, etc.).

### Solution
Every destructive operation requires both a message UID **and** a cryptographic `safe_id` token. The `safe_id` is an HMAC-SHA256 hash that cryptographically binds:
- Message UID
- Sender email address
- Message subject
- Account-specific secret

### How It Works

1. **Secret Generation**: When an account is added via `account_add`, a random 32-byte secret is generated and stored in the account configuration.

2. **Safe ID Generation**: When listing or getting messages, the server generates a `safe_id` for each message:
   ```
   safe_id = HMAC-SHA256(secret, "uid|from|subject")[:12]
   ```
   The first 12 hex characters (48 bits) provide sufficient entropy for validation.

3. **Validation**: Before any destructive operation, the server:
   - Fetches the message headers from IMAP
   - Regenerates the expected `safe_id` using the same HMAC
   - Compares with the provided `safe_id`
   - Rejects the operation if they don't match

### Security Properties

- **AI cannot forge safe_ids**: Without knowing the secret, an AI cannot generate valid `safe_id` tokens
- **safe_id is bound to specific message**: Even if AI remembers a `safe_id` from a previous message, it won't work for a different message
- **Per-account isolation**: Each account has its own secret, so compromise of one doesn't affect others
- **Persistence across restarts**: Secrets are stored in config, so valid `safe_ids` work across server restarts

### Affected Tools

Tools requiring `safe_id`:
- `message_delete` - requires `uid` + `safe_id`
- `message_move` - requires `uid` + `safe_id`
- `message_copy` - requires `uid` + `safe_id`
- `messages_delete_bulk` - requires array of `{uid, safe_id}` objects
- `messages_move_bulk` - requires array of `{uid, safe_id}` objects
- `messages_copy_bulk` - requires array of `{uid, safe_id}` objects

Tools that provide `safe_id` in response:
- `message_list` - includes `safe_id` for each message
- `message_get` - includes `safe_id` for the message
- `thread_get` - includes `safe_id` for each message in thread
- `thread_get_by_id` - includes `safe_id` for each message in thread

### Legacy Account Migration

If an account was created before this feature (no secret in config), the server automatically generates and saves a secret on first use.

## 2. Protected Folders

### Problem
Some folders like INBOX, Sent, Drafts, Trash should never be accidentally deleted.

### Solution
The `folder_delete` tool maintains a list of protected folder names:
- INBOX
- Sent
- Drafts
- Trash
- Spam
- Junk
- Gmail special folders ([Gmail]/All Mail, etc.)

Attempting to delete these folders returns an error.

### Non-Empty Folder Protection

Deleting a folder containing messages requires explicit confirmation via the `force_if_empty` parameter. Without it, the operation fails with a message indicating how many messages would be lost.

## 3. Read-Only Mode

### Configuration
The server can be configured in read-only mode via `security.read_only: true` in config. In this mode, all write operations (delete, move, send, etc.) are blocked.

## 4. Trash-Based Deletion

### Problem
Permanent deletion of emails is irreversible and dangerous.

### Solution
The `message_delete` and `messages_delete_bulk` tools don't permanently delete messages. Instead, they:
1. Detect the trash folder (from config or auto-detect from server)
2. Move messages to trash
3. Report the action in the response

Direct deletion from the trash folder is blocked - messages can only be permanently deleted through email client interfaces.

## 5. Bulk Operation Limits

### Problem
Accidentally operating on thousands of messages could be catastrophic.

### Solution
All bulk operations have a limit of 1000 messages per call (`maxBulkOperations`). Operations exceeding this limit are rejected.

## 6. Chunked Operations

All bulk operations are internally chunked (50 messages per IMAP command) to:
- Prevent server timeouts
- Allow partial success reporting
- Avoid overwhelming the IMAP server

## Best Practices for AI Integration

1. **Always use fresh safe_ids**: Don't cache or remember `safe_id` values between sessions
2. **Verify before destructive actions**: Have AI confirm message details before deletion/moving
3. **Use read-only mode for testing**: Enable `security.read_only` when testing AI integrations
4. **Review bulk operations**: Be cautious with bulk operations, verify the message list first
5. **Monitor trash folder**: Deleted messages go to trash, can be recovered if needed

## Configuration Example

```json
{
  "accounts": [
    {
      "id": "personal",
      "name": "Personal Email",
      "host": "imap.example.com",
      "port": 993,
      "tls": true,
      "username": "user@example.com",
      "password": "...",
      "secret": "auto-generated-base64-secret"
    }
  ],
  "security": {
    "read_only": false,
    "trash_folder": "Trash"
  }
}
```

The `secret` field is automatically generated and should not be manually edited.
