"""
Pretty printing utilities for terminal output
"""

import json
from typing import Any

# ANSI color codes
class Colors:
    RESET = '\033[0m'
    BOLD = '\033[1m'
    DIM = '\033[2m'

    # Regular colors
    RED = '\033[31m'
    GREEN = '\033[32m'
    YELLOW = '\033[33m'
    BLUE = '\033[34m'
    MAGENTA = '\033[35m'
    CYAN = '\033[36m'
    WHITE = '\033[37m'

    # Bright colors
    BRIGHT_RED = '\033[91m'
    BRIGHT_GREEN = '\033[92m'
    BRIGHT_YELLOW = '\033[93m'
    BRIGHT_BLUE = '\033[94m'
    BRIGHT_MAGENTA = '\033[95m'
    BRIGHT_CYAN = '\033[96m'


def colorize(text: str, color: str) -> str:
    """Wrap text with color codes"""
    return f"{color}{text}{Colors.RESET}"


def colorize_prompt(text: str, color: str) -> str:
    """
    Wrap text with color codes for use in input() prompts.

    Uses \x01 and \x02 markers to tell readline that these characters
    are non-printing. This fixes cursor positioning and backspace issues.

    See: https://stackoverflow.com/questions/9468435/
    """
    # Wrap each escape sequence with \x01...\x02
    colored = f"\x01{color}\x02{text}\x01{Colors.RESET}\x02"
    return colored


def print_header(text: str):
    """Print a header line"""
    print()
    print(colorize(f"═══ {text} ═══", Colors.BOLD + Colors.CYAN))
    print()


def print_subheader(text: str):
    """Print a subheader"""
    print(colorize(f"─── {text} ───", Colors.CYAN))


def print_success(text: str):
    """Print success message"""
    print(colorize(f"✓ {text}", Colors.GREEN))


def print_error(text: str):
    """Print error message"""
    print(colorize(f"✗ {text}", Colors.RED))


def print_warning(text: str):
    """Print warning message"""
    print(colorize(f"⚠ {text}", Colors.YELLOW))


def print_info(text: str):
    """Print info message"""
    print(colorize(f"ℹ {text}", Colors.BLUE))


def format_json_value(value: Any, indent: int = 0) -> str:
    """Format a JSON value with colors"""
    prefix = "  " * indent

    if value is None:
        return colorize("null", Colors.DIM)
    elif isinstance(value, bool):
        return colorize(str(value).lower(), Colors.MAGENTA)
    elif isinstance(value, (int, float)):
        return colorize(str(value), Colors.CYAN)
    elif isinstance(value, str):
        # Truncate long strings
        if len(value) > 200:
            display = value[:200] + "..."
        else:
            display = value
        # Escape special characters for display
        display = display.replace('\n', '\\n').replace('\r', '\\r').replace('\t', '\\t')
        return colorize(f'"{display}"', Colors.GREEN)
    elif isinstance(value, list):
        if not value:
            return colorize("[]", Colors.WHITE)
        lines = [colorize("[", Colors.WHITE)]
        for i, item in enumerate(value):
            comma = "," if i < len(value) - 1 else ""
            formatted = format_json_value(item, indent + 1)
            lines.append(f"{prefix}  {formatted}{comma}")
        lines.append(f"{prefix}{colorize(']', Colors.WHITE)}")
        return "\n".join(lines)
    elif isinstance(value, dict):
        if not value:
            return colorize("{}", Colors.WHITE)
        lines = [colorize("{", Colors.WHITE)]
        items = list(value.items())
        for i, (k, v) in enumerate(items):
            comma = "," if i < len(items) - 1 else ""
            key_str = colorize(f'"{k}"', Colors.YELLOW)
            val_str = format_json_value(v, indent + 1)
            if isinstance(v, (dict, list)) and v:
                lines.append(f"{prefix}  {key_str}: {val_str}{comma}")
            else:
                lines.append(f"{prefix}  {key_str}: {val_str}{comma}")
        lines.append(f"{prefix}{colorize('}', Colors.WHITE)}")
        return "\n".join(lines)
    else:
        return str(value)


def print_json(data: Any, title: str = None):
    """Print formatted JSON with colors"""
    if title:
        print_subheader(title)
    print(format_json_value(data))
    print()


def print_tool_list(tools: dict[str, dict]):
    """Print formatted list of available tools"""
    print_header("Available Tools")

    # Group by category (prefix before _)
    categories = {}
    for name, tool in tools.items():
        parts = name.split('_', 1)
        category = parts[0] if len(parts) > 1 else "other"
        if category not in categories:
            categories[category] = []
        categories[category].append((name, tool))

    for category in sorted(categories.keys()):
        print(colorize(f"\n  {category.upper()}", Colors.BOLD + Colors.YELLOW))
        for name, tool in sorted(categories[category]):
            desc = tool.get('description', '')[:60]
            if len(tool.get('description', '')) > 60:
                desc += "..."
            print(f"    {colorize(name, Colors.CYAN):40} {Colors.DIM}{desc}{Colors.RESET}")
    print()


def print_tool_detail(name: str, tool: dict):
    """Print detailed info about a tool"""
    print_header(f"Tool: {name}")

    print(colorize("Description:", Colors.BOLD))
    print(f"  {tool.get('description', 'No description')}")
    print()

    schema = tool.get('inputSchema', {})
    properties = schema.get('properties', {})
    required = schema.get('required', [])

    if properties:
        print(colorize("Parameters:", Colors.BOLD))
        for param_name, param_info in properties.items():
            req = colorize("*", Colors.RED) if param_name in required else " "
            param_type = param_info.get('type', 'any')
            param_desc = param_info.get('description', '')
            print(f"  {req} {colorize(param_name, Colors.CYAN):20} ({param_type:8}) {Colors.DIM}{param_desc}{Colors.RESET}")
        print()
        print(f"  {colorize('*', Colors.RED)} = required")
    else:
        print(colorize("  No parameters", Colors.DIM))
    print()


def print_response(response, duration_ms: float = None):
    """Print MCP response with formatting"""
    if response.is_error:
        print_error("Error Response")
        print_json(response.error)
    else:
        print_success("Success")
        if duration_ms:
            print(colorize(f"  (took {duration_ms:.0f}ms)", Colors.DIM))

        # Try to parse content if it's a tool result
        result = response.result
        if isinstance(result, dict) and 'content' in result:
            content = result['content']
            if isinstance(content, list) and content:
                first = content[0]
                if isinstance(first, dict) and first.get('type') == 'text':
                    text = first.get('text', '')
                    # Try to parse as JSON
                    try:
                        parsed = json.loads(text)
                        print_json(parsed, "Result")
                        return
                    except json.JSONDecodeError:
                        pass

        print_json(result, "Result")


def print_menu(options: list[tuple[str, str]], title: str = "Menu"):
    """Print a menu with options"""
    print_subheader(title)
    for key, desc in options:
        print(f"  {colorize(key, Colors.CYAN):6} {desc}")
    print()


def prompt(text: str, default: str = None, allow_cancel: bool = True) -> str:
    """Prompt for input with optional default. Raises KeyboardInterrupt on Ctrl+C if allow_cancel=True."""
    if default:
        prompt_text = f"{colorize(text, Colors.YELLOW)} [{colorize(default, Colors.DIM)}]: "
    else:
        prompt_text = f"{colorize(text, Colors.YELLOW)}: "

    try:
        value = input(prompt_text).strip()
        return value if value else (default or "")
    except EOFError:
        print()
        return default or ""
    except KeyboardInterrupt:
        if allow_cancel:
            raise  # Let caller handle it
        print()
        return default or ""


def confirm(text: str, default: bool = False) -> bool:
    """Ask for yes/no confirmation"""
    suffix = "[Y/n]" if default else "[y/N]"
    response = prompt(f"{text} {suffix}")

    if not response:
        return default
    return response.lower() in ('y', 'yes', 'ano', 'a')
