"""MCP Inspector control tools for the agent.

Provides tools to start, stop, and monitor the MCP Inspector
for live protocol debugging and interaction visualization.
"""

import httpx
import webbrowser
from typing import Optional
from pydantic import BaseModel, Field


class InspectorStatus(BaseModel):
    """MCP Inspector status response."""
    running: bool = Field(description="Whether the inspector is currently running")
    url: Optional[str] = Field(default=None, description="Inspector web UI URL with auth token")


class InspectorStartResponse(BaseModel):
    """Response from starting the inspector."""
    success: bool = Field(description="Whether the inspector started successfully")
    message: str = Field(description="Status message")
    url: Optional[str] = Field(default=None, description="Inspector URL to access")


class InspectorStopResponse(BaseModel):
    """Response from stopping the inspector."""
    success: bool = Field(description="Whether the inspector stopped successfully")
    message: str = Field(description="Status message")


class InspectorTools:
    """Tools for controlling the MCP Inspector process.

    The MCP Inspector is a live debugging tool that provides:
    - Real-time MCP protocol visualization
    - Interactive tool testing
    - Request/response inspection
    - Server connection monitoring
    """

    def __init__(self, base_url: str = "http://localhost:8080"):
        """Initialize inspector tools.

        Args:
            base_url: Base URL of the mcpproxy server
        """
        self.base_url = base_url.rstrip('/')
        self.client = httpx.Client(timeout=30.0)

    async def start_inspector(self, open_browser: bool = True) -> InspectorStartResponse:
        """Start the MCP Inspector process.

        Starts the inspector using npx @modelcontextprotocol/inspector.
        The inspector provides a web UI for live MCP protocol debugging.

        Args:
            open_browser: Whether to automatically open the inspector in the default browser

        Returns:
            InspectorStartResponse with status and URL

        Example:
            result = await tools.start_inspector()
            print(f"Inspector running at: {result.url}")
        """
        try:
            response = self.client.post(f"{self.base_url}/api/inspector/start")
            response.raise_for_status()

            data = response.json()
            result = InspectorStartResponse(
                success=data.get("success", False),
                message=data.get("message", ""),
                url=data.get("url")
            )

            # Open browser if requested and URL available
            if open_browser and result.url:
                try:
                    webbrowser.open(result.url)
                except Exception as e:
                    # Non-critical error, just log it
                    print(f"Could not open browser: {e}")

            return result

        except httpx.HTTPStatusError as e:
            return InspectorStartResponse(
                success=False,
                message=f"HTTP error: {e.response.status_code} - {e.response.text}"
            )
        except Exception as e:
            return InspectorStartResponse(
                success=False,
                message=f"Failed to start inspector: {str(e)}"
            )

    async def stop_inspector(self) -> InspectorStopResponse:
        """Stop the MCP Inspector process.

        Gracefully stops the running inspector instance.

        Returns:
            InspectorStopResponse with status

        Example:
            result = await tools.stop_inspector()
            print(result.message)
        """
        try:
            response = self.client.post(f"{self.base_url}/api/inspector/stop")
            response.raise_for_status()

            data = response.json()
            return InspectorStopResponse(
                success=data.get("success", False),
                message=data.get("message", "")
            )

        except httpx.HTTPStatusError as e:
            return InspectorStopResponse(
                success=False,
                message=f"HTTP error: {e.response.status_code} - {e.response.text}"
            )
        except Exception as e:
            return InspectorStopResponse(
                success=False,
                message=f"Failed to stop inspector: {str(e)}"
            )

    async def get_inspector_status(self) -> InspectorStatus:
        """Get the current status of the MCP Inspector.

        Returns:
            InspectorStatus with running state and URL

        Example:
            status = await tools.get_inspector_status()
            if status.running:
                print(f"Inspector is running at {status.url}")
            else:
                print("Inspector is not running")
        """
        try:
            response = self.client.get(f"{self.base_url}/api/inspector/status")
            response.raise_for_status()

            data = response.json()
            return InspectorStatus(
                running=data.get("running", False),
                url=data.get("url")
            )

        except Exception as e:
            # If we can't get status, assume not running
            return InspectorStatus(running=False, url=None)

    async def open_inspector_browser(self) -> bool:
        """Open the inspector in the default web browser.

        Only works if the inspector is already running.

        Returns:
            True if browser opened successfully, False otherwise

        Example:
            if await tools.open_inspector_browser():
                print("Inspector opened in browser")
        """
        status = await self.get_inspector_status()

        if not status.running or not status.url:
            return False

        try:
            webbrowser.open(status.url)
            return True
        except Exception:
            return False

    def __del__(self):
        """Cleanup HTTP client."""
        try:
            self.client.close()
        except Exception:
            pass
