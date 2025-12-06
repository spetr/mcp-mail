"""
Cisco-style console with tab completion and ? help
"""

import readline
import sys
import os
from typing import Callable, Optional

from pretty import Colors, colorize, colorize_prompt, print_info


class CiscoConsole:
    """Cisco IOS-style console with tab completion and ? help"""

    def __init__(self):
        self.completions: list[str] = []
        self.completion_context: str = ""  # What we're completing (tools, params, etc.)
        self.help_callback: Optional[Callable[[str], None]] = None
        self._matches: list[str] = []  # Initialize matches list
        self._is_libedit = False
        self._setup_readline()

    def _setup_readline(self):
        """Setup readline with proper bindings"""
        try:
            # Detect libedit (macOS) vs GNU readline (Linux)
            self._is_libedit = 'libedit' in (readline.__doc__ or '')

            # Set completer
            readline.set_completer(self._completer)

            # Set delimiters - space separated completion
            # Note: libedit handles this differently
            try:
                readline.set_completer_delims(' \t\n')
            except Exception:
                pass  # Some readline implementations don't support this

            # Setup tab completion based on implementation
            if self._is_libedit:
                # macOS libedit - use different binding syntax
                readline.parse_and_bind("bind ^I rl_complete")
                # libedit doesn't support show-all-if-ambiguous the same way
            else:
                # GNU readline
                readline.parse_and_bind("tab: complete")
                readline.parse_and_bind("set show-all-if-ambiguous on")
                readline.parse_and_bind("set completion-ignore-case on")

            # History settings
            readline.set_history_length(100)

        except Exception as e:
            print(f"Warning: readline setup failed: {e}")

    def _completer(self, text: str, state: int) -> Optional[str]:
        """Readline completer function"""
        try:
            if state == 0:
                # First call - build list of matches
                line = readline.get_line_buffer()
                # For libedit, we might need to look at the whole line
                if text:
                    self._matches = [c for c in self.completions if c.lower().startswith(text.lower())]
                else:
                    self._matches = self.completions[:]

                # Sort matches for consistent ordering
                self._matches.sort()

            if state < len(self._matches):
                match = self._matches[state]
                # Add space after completion for convenience
                if len(self._matches) == 1:
                    return match + ' '
                return match
            return None
        except Exception:
            return None

    def set_completions(self, completions: list[str], context: str = ""):
        """Set available completions"""
        self.completions = sorted(completions)
        self.completion_context = context

    def set_help_callback(self, callback: Callable[[str], None]):
        """Set callback for ? help"""
        self.help_callback = callback

    def prompt(self, prompt_text: str, default: str = None) -> str:
        """
        Cisco-style prompt with:
        - Tab completion
        - ? for help
        - Partial command matching
        """
        if default:
            # Use colorize_prompt for the default value display in prompt
            display_prompt = f"{prompt_text} [{colorize_prompt(default, Colors.DIM)}]: "
        else:
            display_prompt = f"{prompt_text}: "

        while True:
            try:
                # Clear any stale state before prompting
                self._matches = []

                # Use input() which properly integrates with readline
                line = input(display_prompt)

                # Strip whitespace but preserve original for processing
                line = line.strip()

                # Handle ? for help
                if line == '?' or line.endswith('?'):
                    query = line.rstrip('?').strip()
                    self._show_help(query)
                    continue

                # Empty input - use default
                if not line:
                    return default or ""

                # Check for partial match if we have completions
                if self.completions and line:
                    # Case-insensitive matching
                    matches = [c for c in self.completions if c.lower().startswith(line.lower())]
                    if len(matches) == 1:
                        # Unique partial match - use it
                        return matches[0]
                    elif len(matches) > 1 and line.lower() not in [c.lower() for c in self.completions]:
                        # Ambiguous - show options
                        print(colorize("  Ambiguous command. Options:", Colors.YELLOW))
                        for m in sorted(matches):
                            print(f"    {m}")
                        continue

                return line

            except EOFError:
                print()
                return default or ""
            except KeyboardInterrupt:
                # Re-raise to let caller handle it
                raise

    def _show_help(self, query: str = ""):
        """Show help for available commands/options"""
        if self.help_callback:
            self.help_callback(query)
            return

        # Default help - show completions
        matches = self.completions
        if query:
            matches = [c for c in self.completions if c.startswith(query)]

        if matches:
            print()
            for item in matches:
                print(f"  {colorize(item, Colors.CYAN)}")
            print()
        else:
            print(colorize("  No matches", Colors.DIM))

    def prompt_choice(self, prompt_text: str, choices: list[str], default: str = None) -> str:
        """Prompt with specific choices"""
        old_completions = self.completions
        self.set_completions(choices)
        try:
            return self.prompt(prompt_text, default)
        finally:
            self.completions = old_completions


class ToolCompleter:
    """Context-aware completer for MCP tools"""

    def __init__(self, console: CiscoConsole):
        self.console = console
        self.tools: dict[str, dict] = {}
        self.menu_commands = ['t', 'l', 'r', 's', 'd', 'q', 'quit', 'exit', 'help', '?']

    def set_tools(self, tools: dict[str, dict]):
        """Set available tools"""
        self.tools = tools

    def setup_main_menu(self):
        """Setup completions for main menu"""
        # Allow both menu shortcuts and direct tool names
        completions = self.menu_commands + list(self.tools.keys())
        self.console.set_completions(completions, "main_menu")
        self.console.set_help_callback(self._main_menu_help)

    def setup_tool_input(self):
        """Setup completions for tool name input"""
        self.console.set_completions(list(self.tools.keys()), "tool_name")
        self.console.set_help_callback(self._tool_name_help)

    def setup_param_input(self, tool_name: str, param_name: str, param_info: dict):
        """Setup completions for parameter input"""
        completions = []

        # Add type-specific completions
        param_type = param_info.get('type', 'string')
        if param_type == 'boolean':
            completions = ['true', 'false']
        elif param_name == 'folder':
            # Common folder names
            completions = ['INBOX', 'Sent', 'Drafts', 'Trash', 'Spam', 'Archive']
        elif param_name == 'account_id':
            # Get account IDs from tools response if we have them cached
            pass  # Could add account IDs here

        self.console.set_completions(completions, f"param:{param_name}")
        self.console.set_help_callback(lambda q: self._param_help(tool_name, param_name, param_info))

    def _main_menu_help(self, query: str):
        """Help for main menu"""
        print()
        print(colorize("  Main Menu Commands:", Colors.BOLD))
        print(f"    {colorize('t', Colors.CYAN)}         Call a tool")
        print(f"    {colorize('l', Colors.CYAN)}         List all tools")
        print(f"    {colorize('r', Colors.CYAN)}         Recent tools")
        print(f"    {colorize('s', Colors.CYAN)}         Scenarios")
        print(f"    {colorize('d', Colors.CYAN)}         Set defaults")
        print(f"    {colorize('q', Colors.CYAN)}         Quit")
        print()
        print(colorize("  Or type tool name directly (tab to complete)", Colors.DIM))
        print()

    def _tool_name_help(self, query: str):
        """Help for tool name input"""
        matches = [name for name in self.tools.keys() if name.startswith(query)] if query else list(self.tools.keys())

        if not matches:
            print(colorize("  No matching tools", Colors.DIM))
            return

        print()
        # Group by prefix
        groups = {}
        for name in matches:
            prefix = name.split('_')[0]
            if prefix not in groups:
                groups[prefix] = []
            groups[prefix].append(name)

        for prefix in sorted(groups.keys()):
            print(colorize(f"  {prefix}:", Colors.YELLOW))
            for name in sorted(groups[prefix]):
                desc = self.tools[name].get('description', '')[:50]
                print(f"    {colorize(name, Colors.CYAN):35} {Colors.DIM}{desc}{Colors.RESET}")
        print()

    def _param_help(self, tool_name: str, param_name: str, param_info: dict):
        """Help for parameter input"""
        print()
        print(f"  {colorize(param_name, Colors.CYAN)}: {param_info.get('description', 'No description')}")
        print(f"  Type: {param_info.get('type', 'string')}")
        if self.console.completions:
            print(f"  Options: {', '.join(self.console.completions)}")
        print()
