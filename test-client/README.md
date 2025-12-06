# MCP Test Client

Interactive CLI tool for testing MCP servers. Cisco IOS-style console with tab completion and `?` help.

## Features

- **Tab completion** - Complete tool names with Tab
- **? help** - Type `?` anytime for context-sensitive help
- **Partial matching** - Type `account_l` → auto-expands to `account_list`
- **Ambiguous detection** - Shows options when multiple matches exist
- **Auto-discovery** - Loads available tools from MCP server
- **Readline history** - Command history with arrow keys
- **Smart defaults** - Remember last used values
- **Pretty printing** - Colored JSON output
- **Scenarios** - Record and replay test sequences
- **Session persistence** - Settings saved between runs

## Usage

```bash
# Run with mcp-imap server
cd test-client
./run.sh

# Or manually
python3 main.py ../mcp-imap --config ../config.json

# Specify any MCP server
python3 main.py /path/to/mcp-server --arg1 --arg2
```

## Cisco-Style Commands

```
mcp: ?                    # Show help
mcp: account_?            # Show account_* tools
mcp: acc<Tab>             # Tab-complete to account_
mcp: account_l            # Partial match → account_list
mcp: account_list         # Call tool directly
```

## Main Commands

| Command | Description |
|---------|-------------|
| `t` | Enter tool name (with completion) |
| `l` or `list` | List all available tools |
| `r` or `recent` | Show recently used tools |
| `s` or `scenarios` | Manage test scenarios |
| `d` or `defaults` | Set default parameter values |
| `q` or `quit` | Exit |
| `?` or `help` | Show help |
| `<tool_name>` | Call tool directly |

## Setting Defaults

Common parameters like `account_id` and `folder` can be set as defaults:

```
mcp: d
  a                        # Add default
  account_id: personal     # Now all tools use this
```

## Recording Scenarios

```
mcp: s
  r                        # Start recording
  Scenario name: my-test   # Enter name
mcp: account_list          # Commands get recorded
mcp: s
  s                        # Stop recording
```

## Replaying Scenarios

```
mcp: s
  p                        # Play scenario
  Scenario name: my-test   # Enter name
  # Each step runs with confirmation
```

## File Structure

```
~/.mcp-test-client/
├── session.json    # Defaults, scenarios, tool history
└── history         # Readline command history
```

## Tips

- **Partial commands work** - `acc` matches `account_*` tools
- **Type `?` after partial** - `folder_?` shows folder tools
- **Tab twice** - Shows all completions
- **Arrow up/down** - Command history
- **Ctrl+C** - Cancel current operation (returns to menu)
