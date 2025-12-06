"""
Session management - save/load state, history, scenarios
"""

import json
import os
import readline
from pathlib import Path
from typing import Any, Optional
from dataclasses import dataclass, field, asdict
from datetime import datetime


@dataclass
class ToolCall:
    """Record of a single tool call"""
    tool: str
    arguments: dict
    timestamp: str = ""
    success: bool = True
    duration_ms: float = 0

    def __post_init__(self):
        if not self.timestamp:
            self.timestamp = datetime.now().isoformat()


@dataclass
class Scenario:
    """A saved test scenario - sequence of tool calls"""
    name: str
    description: str
    calls: list[ToolCall] = field(default_factory=list)
    created: str = ""

    def __post_init__(self):
        if not self.created:
            self.created = datetime.now().isoformat()


@dataclass
class Session:
    """Session state that persists between runs"""
    # Default values for common parameters
    defaults: dict[str, str] = field(default_factory=dict)

    # Last used values per tool
    last_values: dict[str, dict[str, str]] = field(default_factory=dict)

    # Saved scenarios
    scenarios: dict[str, Scenario] = field(default_factory=dict)

    # Command history (tool names)
    tool_history: list[str] = field(default_factory=list)

    # Recording state
    recording: bool = False
    recording_name: str = ""
    recorded_calls: list[ToolCall] = field(default_factory=list)


class SessionManager:
    """Manages session persistence and history"""

    def __init__(self, session_dir: str = None):
        if session_dir:
            self.session_dir = Path(session_dir)
        else:
            # Default to ~/.mcp-test-client/
            self.session_dir = Path.home() / ".mcp-test-client"

        self.session_dir.mkdir(parents=True, exist_ok=True)

        self.session_file = self.session_dir / "session.json"
        self.history_file = self.session_dir / "history"
        self.scenarios_dir = self.session_dir / "scenarios"
        self.scenarios_dir.mkdir(exist_ok=True)

        self.session = Session()
        self._setup_readline()

    def _setup_readline(self):
        """Setup readline with history"""
        try:
            readline.set_history_length(1000)
            if self.history_file.exists():
                readline.read_history_file(str(self.history_file))
        except Exception:
            pass

        # Enable tab completion - handle libedit vs GNU readline
        try:
            if 'libedit' in (readline.__doc__ or ''):
                # macOS libedit
                readline.parse_and_bind("bind ^I rl_complete")
            else:
                # GNU readline
                readline.parse_and_bind("tab: complete")
        except Exception:
            pass

    def save_history(self):
        """Save readline history"""
        try:
            readline.write_history_file(str(self.history_file))
        except Exception:
            pass

    def load(self) -> Session:
        """Load session from file"""
        if self.session_file.exists():
            try:
                with open(self.session_file, 'r') as f:
                    data = json.load(f)

                self.session = Session(
                    defaults=data.get('defaults', {}),
                    last_values=data.get('last_values', {}),
                    tool_history=data.get('tool_history', []),
                    scenarios={
                        name: Scenario(
                            name=s['name'],
                            description=s['description'],
                            calls=[ToolCall(**c) for c in s['calls']],
                            created=s.get('created', '')
                        )
                        for name, s in data.get('scenarios', {}).items()
                    }
                )
            except Exception as e:
                print(f"Warning: Failed to load session: {e}")
                self.session = Session()

        return self.session

    def save(self):
        """Save session to file"""
        try:
            data = {
                'defaults': self.session.defaults,
                'last_values': self.session.last_values,
                'tool_history': self.session.tool_history[-100:],  # Keep last 100
                'scenarios': {
                    name: {
                        'name': s.name,
                        'description': s.description,
                        'calls': [asdict(c) for c in s.calls],
                        'created': s.created
                    }
                    for name, s in self.session.scenarios.items()
                }
            }

            with open(self.session_file, 'w') as f:
                json.dump(data, f, indent=2)

            self.save_history()

        except Exception as e:
            print(f"Warning: Failed to save session: {e}")

    def set_default(self, key: str, value: str):
        """Set a default value"""
        self.session.defaults[key] = value

    def get_default(self, key: str) -> Optional[str]:
        """Get a default value"""
        return self.session.defaults.get(key)

    def set_last_value(self, tool: str, param: str, value: str):
        """Remember last used value for a tool parameter"""
        if tool not in self.session.last_values:
            self.session.last_values[tool] = {}
        self.session.last_values[tool][param] = value

    def get_last_value(self, tool: str, param: str) -> Optional[str]:
        """Get last used value for a tool parameter"""
        return self.session.last_values.get(tool, {}).get(param)

    def add_to_history(self, tool: str):
        """Add tool to history"""
        self.session.tool_history.append(tool)

    def get_recent_tools(self, limit: int = 10) -> list[str]:
        """Get recently used tools (unique, most recent first)"""
        seen = set()
        recent = []
        for tool in reversed(self.session.tool_history):
            if tool not in seen:
                seen.add(tool)
                recent.append(tool)
                if len(recent) >= limit:
                    break
        return recent

    # Scenario recording

    def start_recording(self, name: str):
        """Start recording a scenario"""
        self.session.recording = True
        self.session.recording_name = name
        self.session.recorded_calls = []

    def stop_recording(self, description: str = "") -> Optional[Scenario]:
        """Stop recording and save scenario"""
        if not self.session.recording:
            return None

        scenario = Scenario(
            name=self.session.recording_name,
            description=description,
            calls=self.session.recorded_calls.copy()
        )

        self.session.scenarios[scenario.name] = scenario
        self.session.recording = False
        self.session.recording_name = ""
        self.session.recorded_calls = []

        return scenario

    def cancel_recording(self):
        """Cancel current recording"""
        self.session.recording = False
        self.session.recording_name = ""
        self.session.recorded_calls = []

    def is_recording(self) -> bool:
        """Check if currently recording"""
        return self.session.recording

    def record_call(self, call: ToolCall):
        """Record a tool call if recording"""
        if self.session.recording:
            self.session.recorded_calls.append(call)

    def get_scenario(self, name: str) -> Optional[Scenario]:
        """Get a saved scenario"""
        return self.session.scenarios.get(name)

    def list_scenarios(self) -> list[Scenario]:
        """List all saved scenarios"""
        return list(self.session.scenarios.values())

    def delete_scenario(self, name: str) -> bool:
        """Delete a scenario"""
        if name in self.session.scenarios:
            del self.session.scenarios[name]
            return True
        return False

    def export_scenario(self, name: str, filepath: str) -> bool:
        """Export scenario to file"""
        scenario = self.session.scenarios.get(name)
        if not scenario:
            return False

        try:
            with open(filepath, 'w') as f:
                json.dump({
                    'name': scenario.name,
                    'description': scenario.description,
                    'calls': [asdict(c) for c in scenario.calls],
                    'created': scenario.created
                }, f, indent=2)
            return True
        except Exception:
            return False

    def import_scenario(self, filepath: str) -> Optional[Scenario]:
        """Import scenario from file"""
        try:
            with open(filepath, 'r') as f:
                data = json.load(f)

            scenario = Scenario(
                name=data['name'],
                description=data.get('description', ''),
                calls=[ToolCall(**c) for c in data['calls']],
                created=data.get('created', '')
            )

            self.session.scenarios[scenario.name] = scenario
            return scenario
        except Exception:
            return None
