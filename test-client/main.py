#!/usr/bin/env python3
"""
MCP Test Client - Cisco IOS-style CLI for testing MCP servers
"""

import sys
import os
import time
import json
import argparse
from typing import Optional

from mcp_client import MCPClient
from session import SessionManager, ToolCall
from console import CiscoConsole, ToolCompleter
from pretty import (
    Colors, colorize, colorize_prompt, print_header, print_success,
    print_error, print_warning, print_info, print_json,
    print_tool_list, print_response
)


class MCPTestClient:
    """Cisco IOS-style MCP test client"""

    def __init__(self, session_dir: str = None):
        self.client = MCPClient()
        self.session_mgr = SessionManager(session_dir)
        self.session_mgr.load()
        self.running = False
        self.connected = False

        # Cisco-style console
        self.console = CiscoConsole()
        self.completer = ToolCompleter(self.console)

    def connect(self, command: list[str], cwd: str = None) -> bool:
        """Connect to MCP server"""
        print_info(f"Connecting to: {' '.join(command)}")

        if self.client.connect(command, cwd):
            self.connected = True
            tools = self.client.get_tools()
            print_success(f"Connected! {len(tools)} tools available")
            self.completer.set_tools(tools)
            return True
        else:
            print_error("Failed to connect")
            return False

    def disconnect(self):
        """Disconnect from server"""
        if self.connected:
            self.client.disconnect()
            self.connected = False

    def call_tool(self, tool_name: str, args_override: dict = None) -> bool:
        """Call a tool directly without confirmation"""
        tool = self.client.get_tool_schema(tool_name)
        if not tool:
            print_error(f"Unknown tool: {tool_name}")
            return False

        schema = tool.get('inputSchema', {})
        properties = schema.get('properties', {})
        required = set(schema.get('required', []))

        arguments = args_override.copy() if args_override else {}

        # Only prompt for missing required parameters
        if not args_override:
            missing_required = [p for p in required if p not in arguments]
            missing_optional = [p for p in properties if p not in required and p not in arguments]

            try:
                # Prompt for required params
                for param_name in missing_required:
                    param_info = properties[param_name]
                    value = self._get_param_value(tool_name, param_name, param_info, required=True)
                    if value is not None:
                        arguments[param_name] = self._convert_value(value, param_info)

                # Prompt for optional params only if there are any
                if missing_optional and not missing_required:
                    # If no required params, ask for optional ones too
                    for param_name in missing_optional:
                        param_info = properties[param_name]
                        value = self._get_param_value(tool_name, param_name, param_info, required=False)
                        if value is not None and value != '':
                            arguments[param_name] = self._convert_value(value, param_info)

            except KeyboardInterrupt:
                print()
                print_warning("Cancelled")
                return False

        # Execute immediately
        start_time = time.time()
        response = self.client.call_tool(tool_name, arguments)
        duration_ms = (time.time() - start_time) * 1000

        # Show result
        print_response(response, duration_ms)

        # Record for history/scenarios
        call = ToolCall(
            tool=tool_name,
            arguments=arguments,
            success=not response.is_error,
            duration_ms=duration_ms
        )
        self.session_mgr.record_call(call)
        self.session_mgr.add_to_history(tool_name)

        return not response.is_error

    def _get_param_value(self, tool_name: str, param_name: str, param_info: dict, required: bool) -> Optional[str]:
        """Get parameter value with smart defaults"""
        param_type = param_info.get('type', 'string')

        # Try defaults
        default = self.session_mgr.get_default(param_name)
        if not default:
            default = self.session_mgr.get_last_value(tool_name, param_name)

        # Setup completions for this param
        self.completer.setup_param_input(tool_name, param_name, param_info)

        # Build prompt - use colorize_prompt for readline compatibility
        req_mark = colorize_prompt("*", Colors.RED) if required else ""
        type_hint = colorize_prompt(f"({param_type})", Colors.DIM)

        prompt_text = f"  {req_mark}{param_name} {type_hint}"
        value = self.console.prompt(prompt_text, default)

        # Validate required
        if required and not value:
            print_error(f"    {param_name} is required")
            return self._get_param_value(tool_name, param_name, param_info, required)

        # Remember value
        if value:
            self.session_mgr.set_last_value(tool_name, param_name, value)

        return value

    def _convert_value(self, value: str, param_info: dict):
        """Convert string value to appropriate type"""
        param_type = param_info.get('type', 'string')

        if param_type in ('number', 'integer'):
            try:
                return int(value) if param_type == 'integer' else float(value)
            except ValueError:
                return value
        elif param_type == 'boolean':
            return value.lower() in ('true', '1', 'yes', 'y')
        elif param_type == 'array':
            try:
                return json.loads(value)
            except json.JSONDecodeError:
                return [v.strip() for v in value.split(',')]
        return value

    def cmd_show_tools(self, filter_prefix: str = ""):
        """Show available tools"""
        tools = self.client.get_tools()
        if filter_prefix:
            tools = {k: v for k, v in tools.items() if k.startswith(filter_prefix)}

        if not tools:
            print_warning(f"No tools matching '{filter_prefix}'")
            return

        print_tool_list(tools)

    def cmd_show_tool(self, tool_name: str):
        """Show detailed info about a tool"""
        tool = self.client.get_tool_schema(tool_name)
        if not tool:
            print_error(f"Unknown tool: {tool_name}")
            return

        print()
        print(f"  {colorize(tool_name, Colors.CYAN + Colors.BOLD)}")
        print(f"  {tool.get('description', 'No description')}")
        print()

        schema = tool.get('inputSchema', {})
        properties = schema.get('properties', {})
        required = set(schema.get('required', []))

        if properties:
            print(colorize("  Parameters:", Colors.BOLD))
            for name, info in properties.items():
                req = colorize("*", Colors.RED) if name in required else " "
                ptype = info.get('type', 'any')
                desc = info.get('description', '')
                print(f"    {req}{colorize(name, Colors.YELLOW):20} {Colors.DIM}({ptype:8}){Colors.RESET} {desc}")
            print()

    def cmd_defaults(self, action: str = None, key: str = None, value: str = None):
        """Manage default values"""
        defaults = self.session_mgr.session.defaults

        if action == 'set' and key:
            if value:
                self.session_mgr.set_default(key, value)
                print_success(f"Default set: {key} = {value}")
            else:
                print_error("Usage: defaults set <key> <value>")
        elif action == 'unset' and key:
            if key in defaults:
                del defaults[key]
                print_success(f"Default removed: {key}")
            else:
                print_warning(f"No default for: {key}")
        elif action == 'clear':
            defaults.clear()
            print_success("All defaults cleared")
        else:
            # Show defaults
            if defaults:
                print()
                print(colorize("  Current defaults:", Colors.BOLD))
                for k, v in defaults.items():
                    print(f"    {colorize(k, Colors.CYAN):20} = {v}")
                print()
            else:
                print_info("No defaults set. Use: defaults set <key> <value>")

    def cmd_recent(self):
        """Show recently used tools"""
        recent = self.session_mgr.get_recent_tools(10)
        if not recent:
            print_info("No recent tools")
            return

        print()
        print(colorize("  Recent tools:", Colors.BOLD))
        for i, tool in enumerate(recent, 1):
            desc = self.client.get_tool_schema(tool).get('description', '')[:40]
            print(f"    {colorize(str(i), Colors.CYAN):3} {tool:30} {Colors.DIM}{desc}{Colors.RESET}")
        print()
        print(colorize("  Type tool name to call it", Colors.DIM))
        print()

    def cmd_scenario(self, action: str = None, name: str = None):
        """Manage test scenarios"""
        if action == 'record' and name:
            self.session_mgr.start_recording(name)
            print_success(f"Recording started: {name}")
            print_info("All tool calls will be recorded. Use 'scenario stop' to finish.")
        elif action == 'stop':
            scenario = self.session_mgr.stop_recording("")
            if scenario:
                print_success(f"Saved scenario '{scenario.name}' with {len(scenario.calls)} calls")
            else:
                print_warning("Not recording")
        elif action == 'play' and name:
            scenario = self.session_mgr.get_scenario(name)
            if scenario:
                print_info(f"Playing scenario: {name} ({len(scenario.calls)} calls)")
                for i, call in enumerate(scenario.calls, 1):
                    print(f"\n{colorize(f'[{i}/{len(scenario.calls)}]', Colors.CYAN)} {call.tool}")
                    if not self.call_tool(call.tool, call.arguments):
                        print_warning("Scenario stopped due to error")
                        break
                print_success("Scenario completed")
            else:
                print_error(f"Scenario not found: {name}")
        elif action == 'delete' and name:
            if self.session_mgr.delete_scenario(name):
                print_success(f"Deleted: {name}")
            else:
                print_error(f"Not found: {name}")
        elif action == 'list' or not action:
            scenarios = self.session_mgr.list_scenarios()
            if scenarios:
                print()
                print(colorize("  Saved scenarios:", Colors.BOLD))
                for s in scenarios:
                    print(f"    {colorize(s.name, Colors.CYAN):20} {len(s.calls)} calls")
                print()
            else:
                print_info("No scenarios. Use: scenario record <name>")
        else:
            print(colorize("  Usage:", Colors.BOLD))
            print("    scenario                    List scenarios")
            print("    scenario record <name>      Start recording")
            print("    scenario stop               Stop recording")
            print("    scenario play <name>        Play scenario")
            print("    scenario delete <name>      Delete scenario")

    def show_help(self, topic: str = ""):
        """Show help"""
        if topic:
            # Help for specific tool
            if topic in self.client.get_tools():
                self.cmd_show_tool(topic)
                return
            # Help for command prefix
            matches = [t for t in self.client.get_tools() if t.startswith(topic)]
            if matches:
                self.cmd_show_tools(topic)
                return

        print()
        print(colorize("  ╔══════════════════════════════════════════════════════════════╗", Colors.CYAN))
        print(colorize("  ║", Colors.CYAN) + colorize("  MCP Test Client - Cisco-style CLI                          ", Colors.BOLD) + colorize("║", Colors.CYAN))
        print(colorize("  ╚══════════════════════════════════════════════════════════════╝", Colors.CYAN))
        print()
        print(colorize("  Commands:", Colors.BOLD))
        print(f"    {colorize('<tool_name>', Colors.YELLOW):30} Call a tool directly")
        print(f"    {colorize('show tools [prefix]', Colors.YELLOW):30} List available tools")
        print(f"    {colorize('show tool <name>', Colors.YELLOW):30} Show tool details")
        print(f"    {colorize('show recent', Colors.YELLOW):30} Recently used tools")
        print(f"    {colorize('defaults [set|unset|clear]', Colors.YELLOW):30} Manage default values")
        print(f"    {colorize('scenario [record|stop|play]', Colors.YELLOW):30} Test scenarios")
        print(f"    {colorize('exit | quit', Colors.YELLOW):30} Exit")
        print()
        print(colorize("  Tips:", Colors.BOLD))
        print(f"    {colorize('?', Colors.CYAN)}           Show this help")
        print(f"    {colorize('<prefix>?', Colors.CYAN)}   Show tools starting with prefix")
        print(f"    {colorize('Tab', Colors.CYAN)}         Auto-complete")
        print(f"    {colorize('Ctrl+C', Colors.CYAN)}      Cancel current input")
        print()

    def parse_command(self, line: str) -> bool:
        """Parse and execute a command. Returns False to exit."""
        line = line.strip()
        if not line:
            return True

        parts = line.split()
        cmd = parts[0].lower()
        args = parts[1:]

        # Help
        if cmd == '?' or cmd == 'help':
            self.show_help(args[0] if args else "")
            return True

        # Help suffix (e.g., "account_?")
        if cmd.endswith('?'):
            self.show_help(cmd[:-1])
            return True

        # Exit
        if cmd in ('exit', 'quit', 'q'):
            return False

        # Show commands
        if cmd == 'show':
            if not args:
                print_error("Usage: show tools|tool|recent")
            elif args[0] == 'tools':
                self.cmd_show_tools(args[1] if len(args) > 1 else "")
            elif args[0] == 'tool' and len(args) > 1:
                self.cmd_show_tool(args[1])
            elif args[0] == 'recent':
                self.cmd_recent()
            else:
                print_error(f"Unknown: show {args[0]}")
            return True

        # Defaults
        if cmd == 'defaults':
            action = args[0] if args else None
            key = args[1] if len(args) > 1 else None
            value = ' '.join(args[2:]) if len(args) > 2 else None
            self.cmd_defaults(action, key, value)
            return True

        # Scenarios
        if cmd == 'scenario':
            action = args[0] if args else None
            name = args[1] if len(args) > 1 else None
            self.cmd_scenario(action, name)
            return True

        # Try as tool name (exact match)
        if cmd in self.client.get_tools():
            self.call_tool(cmd)
            return True

        # Try partial match
        matches = [t for t in self.client.get_tools() if t.startswith(cmd)]
        if len(matches) == 1:
            self.call_tool(matches[0])
            return True
        elif len(matches) > 1:
            print(colorize("  Ambiguous command:", Colors.YELLOW))
            for m in matches:
                print(f"    {m}")
            return True

        print_error(f"Unknown command: {cmd}")
        print_info("Type '?' for help")
        return True

    def run(self):
        """Main loop"""
        self.running = True

        print()
        print(colorize("  ┌─────────────────────────────────────────────┐", Colors.CYAN))
        print(colorize("  │", Colors.CYAN) + f"  MCP Test Client    {colorize(f'{len(self.client.get_tools())} tools', Colors.GREEN):>22}" + colorize("  │", Colors.CYAN))
        print(colorize("  │", Colors.CYAN) + colorize("  Type '?' for help, Tab to complete         ", Colors.DIM) + colorize("│", Colors.CYAN))
        print(colorize("  └─────────────────────────────────────────────┘", Colors.CYAN))
        print()

        # Setup completions
        all_commands = ['show', 'defaults', 'scenario', 'exit', 'quit', 'help']
        all_commands.extend(self.client.get_tools().keys())
        self.console.set_completions(all_commands)

        recording_shown = False

        while self.running:
            try:
                # Check connection
                if not self.client.is_connected():
                    print_error("Connection lost!")
                    break

                # Recording indicator - use colorize_prompt for readline compatibility
                if self.session_mgr.is_recording():
                    if not recording_shown:
                        recording_shown = True
                    prompt_text = colorize_prompt("●REC", Colors.RED) + colorize_prompt(" mcp# ", Colors.GREEN)
                else:
                    recording_shown = False
                    prompt_text = colorize_prompt("mcp# ", Colors.GREEN)

                line = self.console.prompt(prompt_text)

                if not self.parse_command(line):
                    self.running = False

            except KeyboardInterrupt:
                print()
                continue
            except EOFError:
                print()
                self.running = False

        # Cleanup
        print()
        print_info("Saving session...")
        self.session_mgr.save()
        self.disconnect()
        print_info("Goodbye!")


def main():
    parser = argparse.ArgumentParser(
        description="MCP Test Client - Cisco IOS-style CLI",
        usage="%(prog)s [options] [--] <mcp-server-command> [server-args...]"
    )
    parser.add_argument(
        "--cwd",
        help="Working directory for the MCP server"
    )
    parser.add_argument(
        "--session-dir",
        help="Directory for session data (default: ~/.mcp-test-client)"
    )
    parser.add_argument(
        "command",
        nargs=argparse.REMAINDER,
        help="MCP server command"
    )

    args = parser.parse_args()

    # Remove leading '--' if present
    if args.command and args.command[0] == '--':
        args.command = args.command[1:]

    # Default command
    if not args.command:
        if os.path.exists("./mcp-imap"):
            args.command = ["./mcp-imap"]
            if os.path.exists("./config.json"):
                args.command.extend(["--config", "config.json"])
        elif os.path.exists("../mcp-imap"):
            args.command = ["../mcp-imap"]
            args.cwd = ".."
        else:
            print_error("No command specified and couldn't find mcp-imap")
            print("Usage: python main.py <mcp-server-command>")
            sys.exit(1)

    client = MCPTestClient(args.session_dir)

    if not client.connect(args.command, args.cwd):
        sys.exit(1)

    client.run()


if __name__ == "__main__":
    main()
