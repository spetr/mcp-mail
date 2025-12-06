#!/bin/bash
# Quick launcher for MCP Test Client

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Build if needed
if [ ! -f "$PROJECT_DIR/mcp-imap" ]; then
    echo "Building mcp-imap..."
    (cd "$PROJECT_DIR" && go build -o mcp-imap .)
fi

# Find config file
CONFIG_FILE="$PROJECT_DIR/config.json"
if [ ! -f "$CONFIG_FILE" ]; then
    CONFIG_FILE="$PROJECT_DIR/config.example.json"
fi

# Run client
python3 "$SCRIPT_DIR/main.py" "$PROJECT_DIR/mcp-imap" --config "$CONFIG_FILE"
