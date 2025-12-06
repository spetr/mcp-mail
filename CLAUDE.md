# MCP-Mail Server Guide

## Overview

MCP server for IMAP email access via Model Context Protocol. Enables AI models to read, search, organize emails, and remember user preferences.

## Quick Start for AI Assistants

**First step with any email request:** Check which accounts are available and their connection status.

```
1. account_list → Get available accounts
2. account_connect(account_id) → Connect if not connected
3. folder_list(account_id) → See available folders
```

**Key concept:** All operations require `account_id` parameter. Get it from `account_list` first.

## Common Workflows

### Reading New/Unread Emails
```
1. search_unread(account_id, folder="INBOX") → Get UIDs of unread messages
2. For each UID: message_get(account_id, folder, uid) → Get full message
```

### Finding Specific Emails
Use the most specific search tool:
- `search_from` - From specific sender
- `search_subject` - By subject text
- `search_today` - Today's emails
- `search_date_range` - Within date range
- `search` - Complex queries (multiple criteria)

### Working with Conversations/Threads
```
1. thread_list(account_id, folder) → Get conversation threads overview
2. thread_get(account_id, folder, uid) → Get full conversation from any message in it
```
Threads are grouped by Message-ID, In-Reply-To, and References headers.

### Organizing Emails
```
- message_move(account_id, folder, uid, target_folder) → Move to folder
- message_delete(account_id, folder, uid) → Move to trash (not permanent)
- message_mark_read/message_mark_unread → Update read status
- message_flag/message_unflag → Star/unstar messages
```

### Bulk Operations
For multiple messages at once:
```
- bulk_move(account_id, folder, uids[], target_folder)
- bulk_delete(account_id, folder, uids[])
- bulk_mark_read/bulk_mark_unread(account_id, folder, uids[])
- bulk_set_flags(account_id, folder, uids[], flags[], remove_flags[])
```

### Remembering User Preferences (Memory System)
The server has a persistent memory system. Use it to:
- Remember important contacts: `memory_add_contact`
- Track unwanted senders: `memory_add_unwanted`
- Store notes and preferences: `memory_add_note`, `memory_update_preferences`
- Record facts about contacts: `memory_add_observation`

**Best practice:** At session start, call `memory_get(account_id)` to load remembered context.

## Tool Reference

### Account Management

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `account_list` | List all configured accounts | - |
| `account_connect` | Connect to IMAP server | account_id |
| `account_disconnect` | Close connection | account_id |
| `account_status` | Get connection status and capabilities | account_id |

### Folder Operations

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `folder_list` | List all folders | account_id |
| `folder_info` | Get folder stats (message count, unread) | account_id, folder |
| `folder_create` | Create new folder | account_id, name |
| `folder_rename` | Rename folder | account_id, old_name, new_name |
| `folder_delete` | Delete folder | account_id, name |

### Message Operations

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `message_list` | List messages with pagination | account_id, folder, limit?, offset? |
| `message_get` | Get complete message (body + attachments info) | account_id, folder, uid |
| `message_get_headers` | Get headers only (faster) | account_id, folder, uid |
| `message_move` | Move to another folder | account_id, folder, uid, target_folder |
| `message_copy` | Copy to another folder | account_id, folder, uid, target_folder |
| `message_delete` | Move to trash | account_id, folder, uid |

### Thread/Conversation Operations

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `thread_list` | List conversation threads with summaries | account_id, folder, limit?, offset? |
| `thread_get` | Get full thread from any message UID | account_id, folder, uid |
| `thread_get_by_id` | Get thread by Message-ID header | account_id, folder, message_id |

Thread results include: all messages in chronological order, participants, unread count, normalized subject (without Re:/Fwd:).

### Flag Operations

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `message_mark_read` | Mark as read (add \Seen) | account_id, folder, uid |
| `message_mark_unread` | Mark as unread (remove \Seen) | account_id, folder, uid |
| `message_flag` | Star message (add \Flagged) | account_id, folder, uid |
| `message_unflag` | Unstar message | account_id, folder, uid |
| `message_set_flags` | Set arbitrary flags | account_id, folder, uid, flags[], remove[] |

### Search Operations

| Tool | Purpose | When to Use |
|------|---------|-------------|
| `search` | Full IMAP search | Complex queries with multiple criteria |
| `search_unread` | Find unread messages | "Show my unread emails" |
| `search_flagged` | Find starred messages | "Show my starred emails" |
| `search_from` | Search by sender | "Emails from john@example.com" |
| `search_subject` | Search by subject | "Find email about project X" |
| `search_today` | Today's messages | "Show today's emails" |
| `search_date_range` | Messages in date range | "Emails from last week" |

**Search returns UIDs** - use `message_get` to fetch actual content.

### Bulk Operations

| Tool | Purpose | Limits |
|------|---------|--------|
| `bulk_move` | Move multiple messages | Max 1000 UIDs |
| `bulk_delete` | Delete multiple messages | Max 1000 UIDs |
| `bulk_mark_read` | Mark multiple as read | Max 1000 UIDs |
| `bulk_mark_unread` | Mark multiple as unread | Max 1000 UIDs |
| `bulk_set_flags` | Set flags on multiple | Max 1000 UIDs |

### Memory Operations

| Tool | Purpose |
|------|---------|
| `memory_get` | Get summary of stored memory |
| `memory_get_full` | Get complete memory data |
| `memory_add_contact` | Add/update important contact |
| `memory_remove_contact` | Remove contact |
| `memory_add_unwanted` | Mark sender/subject as spam |
| `memory_remove_unwanted` | Remove from spam list |
| `memory_add_note` | Add a note/rule/preference |
| `memory_add_topic` | Add priority topic to watch |
| `memory_set_folder_purpose` | Document what folder is for |
| `memory_update_preferences` | Set language, timezone, etc. |
| `memory_add_observation` | Add fact about a contact |
| `memory_get_contact` | Get all info about a contact |
| `memory_search` | Search across all memory |

## Important Notes

### UID vs Sequence Numbers
- **Always use UID** - unique and persistent identifier
- Sequence numbers change when messages are deleted
- All tools use UIDs

### Folder Names
Common folder names (may vary by provider):
- `INBOX` - Main inbox
- `Sent`, `[Gmail]/Sent Mail` - Sent messages
- `Drafts`, `[Gmail]/Drafts` - Draft messages
- `Trash`, `[Gmail]/Trash` - Deleted messages
- `Spam`, `[Gmail]/Spam`, `Junk` - Spam folder

Use `folder_list` to see exact folder names for each account.

### Date Formats
For search dates, use: `2024-01-15` or `2024-01-15T10:30:00Z` (RFC3339)

### Message Flags
Standard IMAP flags:
- `\Seen` - Read
- `\Answered` - Replied
- `\Flagged` - Starred/Flagged
- `\Deleted` - Marked for deletion
- `\Draft` - Draft message

### Best Practices

1. **Check connection first** - Call `account_status` before operations
2. **Use specific search tools** - `search_from` is faster than generic `search`
3. **Load memory at session start** - `memory_get` provides valuable context
4. **Use threads for conversations** - `thread_list` groups related messages
5. **Use bulk operations** - More efficient than individual operations
6. **Use `message_get_headers`** - When you only need sender/subject/date

## Configuration

### Config File Format
```json
{
  "accounts": [
    {
      "id": "personal",
      "name": "Personal Email",
      "host": "imap.gmail.com",
      "port": 993,
      "tls": true,
      "username": "user@gmail.com",
      "password": "${GMAIL_APP_PASSWORD}"
    }
  ],
  "settings": {
    "idle_timeout": 300,
    "max_message_size": 10485760,
    "auto_connect": false
  }
}
```

Passwords support `${ENV_VAR}` substitution.

## Running the Server

```bash
# Build
go build -o mcp-imap .

# Stdio transport (default)
./mcp-imap --config config.json

# SSE transport
./mcp-imap --config config.json --transport sse --port 8080

# With authentication
./mcp-imap --config config.json --transport sse --port 8080 --auth-token "secret"
```

## Tech Stack

- Go 1.21+
- `github.com/emersion/go-imap/v2` - IMAP client
- `github.com/emersion/go-message` - MIME parsing
- `github.com/mark3labs/mcp-go` - MCP SDK
