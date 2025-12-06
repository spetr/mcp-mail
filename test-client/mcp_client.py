"""
MCP Protocol Client - handles communication with MCP server via stdio or SSE
"""

import json
import subprocess
import sys
import threading
import queue
from typing import Optional, Any
from dataclasses import dataclass


@dataclass
class MCPResponse:
    """Response from MCP server"""
    id: int
    result: Optional[Any] = None
    error: Optional[dict] = None

    @property
    def is_error(self) -> bool:
        return self.error is not None


class MCPClient:
    """MCP Client using stdio transport"""

    def __init__(self):
        self.process: Optional[subprocess.Popen] = None
        self.request_id = 0
        self.responses: dict[int, MCPResponse] = {}
        self.response_queue = queue.Queue()
        self.reader_thread: Optional[threading.Thread] = None
        self.running = False
        self.tools: dict[str, dict] = {}

    def connect(self, command: list[str], cwd: Optional[str] = None) -> bool:
        """Start MCP server process and connect via stdio"""
        try:
            self.process = subprocess.Popen(
                command,
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL,  # Ignore server logs
                cwd=cwd,
                text=True,
                bufsize=1
            )
            self.running = True

            # Start reader thread
            self.reader_thread = threading.Thread(target=self._read_responses, daemon=True)
            self.reader_thread.start()

            # Initialize MCP connection
            init_result = self._initialize()
            if init_result.is_error:
                print(f"Initialization error: {init_result.error}")
                return False

            # Get available tools
            tools_result = self._list_tools()
            if not tools_result.is_error and tools_result.result:
                self.tools = {
                    tool['name']: tool
                    for tool in tools_result.result.get('tools', [])
                }

            return True

        except Exception as e:
            print(f"Failed to connect: {e}")
            return False

    def disconnect(self):
        """Close connection to MCP server"""
        self.running = False
        if self.process:
            self.process.terminate()
            try:
                self.process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self.process.kill()
            self.process = None

    def _read_responses(self):
        """Background thread to read responses from server"""
        while self.running and self.process and self.process.poll() is None:
            try:
                line = self.process.stdout.readline()
                if not line:
                    # EOF - process probably died
                    break

                line = line.strip()
                if not line:
                    continue

                try:
                    msg = json.loads(line)
                    if 'id' in msg:
                        response = MCPResponse(
                            id=msg['id'],
                            result=msg.get('result'),
                            error=msg.get('error')
                        )
                        self.response_queue.put(response)
                except json.JSONDecodeError:
                    # Not JSON - might be log output, ignore
                    pass

            except Exception as e:
                # Log error but don't break - try to keep reading
                print(f"Reader error: {e}")
                break

        self.running = False

    def _send_request(self, method: str, params: Optional[dict] = None) -> MCPResponse:
        """Send JSON-RPC request and wait for response"""
        # Check if still connected
        if not self.is_connected():
            return MCPResponse(id=self.request_id + 1, error={"message": "Server is not connected"})

        self.request_id += 1
        request = {
            "jsonrpc": "2.0",
            "id": self.request_id,
            "method": method,
        }
        if params:
            request["params"] = params

        request_line = json.dumps(request) + "\n"

        try:
            self.process.stdin.write(request_line)
            self.process.stdin.flush()

            # Wait for response with timeout
            try:
                response = self.response_queue.get(timeout=30)
                return response
            except queue.Empty:
                # Check if process died while waiting
                if not self.is_connected():
                    return MCPResponse(id=self.request_id, error={"message": "Server process died while waiting for response"})
                return MCPResponse(id=self.request_id, error={"message": "Timeout waiting for response (30s)"})

        except BrokenPipeError:
            self.running = False
            return MCPResponse(id=self.request_id, error={"message": "Broken pipe - server connection lost"})
        except Exception as e:
            return MCPResponse(id=self.request_id, error={"message": str(e)})

    def _initialize(self) -> MCPResponse:
        """Send initialize request"""
        return self._send_request("initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {
                "name": "mcp-test-client",
                "version": "1.0.0"
            }
        })

    def _list_tools(self) -> MCPResponse:
        """Get list of available tools"""
        return self._send_request("tools/list")

    def call_tool(self, name: str, arguments: dict) -> MCPResponse:
        """Call a tool with given arguments"""
        return self._send_request("tools/call", {
            "name": name,
            "arguments": arguments
        })

    def get_tools(self) -> dict[str, dict]:
        """Return available tools"""
        return self.tools

    def get_tool_schema(self, name: str) -> Optional[dict]:
        """Get schema for a specific tool"""
        return self.tools.get(name)

    def is_connected(self) -> bool:
        """Check if the server process is still running"""
        return self.running and self.process is not None and self.process.poll() is None

    def get_stderr(self) -> str:
        """Get any stderr output from the server (not available - stderr is suppressed)"""
        return ""
