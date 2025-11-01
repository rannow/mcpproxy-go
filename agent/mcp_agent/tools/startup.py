"""Startup and service management tools."""

import httpx
from typing import Literal, Optional, Dict, Any, List
from pydantic import BaseModel


class StartupConfig(BaseModel):
    """Startup configuration for server."""
    server_name: str
    command: Optional[str] = None
    args: List[str] = []
    env: Dict[str, str] = {}
    working_dir: Optional[str] = None
    auto_start: bool = False
    restart_on_failure: bool = False


class StartupScriptResult(BaseModel):
    """Result of startup script operation."""
    success: bool
    message: str
    config: Optional[StartupConfig] = None


class ServiceStatus(BaseModel):
    """Status of managed service."""
    service_name: str
    running: bool
    status: str
    uptime: Optional[str] = None
    details: Dict[str, Any] = {}


class StartupTools:
    """Tools for managing startup scripts and services."""

    def __init__(self, base_url: str = "http://localhost:8080"):
        """Initialize startup tools.

        Args:
            base_url: Base URL for mcpproxy agent API
        """
        self.base_url = base_url
        self.client = httpx.AsyncClient(timeout=30.0)

    async def read_startup_script(self, server_name: str) -> StartupScriptResult:
        """Read server startup configuration.

        Args:
            server_name: Server to read startup config from

        Returns:
            StartupScriptResult with configuration
        """
        try:
            response = await self.client.get(
                f"{self.base_url}/api/v1/agent/servers/{server_name}/config"
            )
            response.raise_for_status()
            data = response.json()

            config = StartupConfig(
                server_name=server_name,
                command=data.get("command"),
                args=data.get("args", []),
                env=data.get("env", {}),
                working_dir=data.get("working_dir"),
                auto_start=data.get("enabled", False),
                restart_on_failure=False  # Not exposed in current API
            )

            return StartupScriptResult(
                success=True,
                message="Startup configuration retrieved",
                config=config
            )

        except httpx.HTTPError as e:
            return StartupScriptResult(
                success=False,
                message=f"Failed to read startup config: {str(e)}"
            )

    async def update_startup_script(
        self,
        server_name: str,
        script_updates: Dict[str, Any],
        validate: bool = True
    ) -> StartupScriptResult:
        """Modify startup script configuration.

        Args:
            server_name: Server to update
            script_updates: Updates to apply (command, args, env, working_dir, etc.)
            validate: Whether to validate before applying

        Returns:
            StartupScriptResult with update status
        """
        try:
            # Validate updates if requested
            if validate:
                validation_errors = self._validate_startup_updates(script_updates)
                if validation_errors:
                    return StartupScriptResult(
                        success=False,
                        message=f"Validation failed: {', '.join(validation_errors)}"
                    )

            # Apply updates via API
            response = await self.client.patch(
                f"{self.base_url}/api/v1/agent/servers/{server_name}/config",
                json=script_updates
            )
            response.raise_for_status()
            data = response.json()

            return StartupScriptResult(
                success=data.get("success", True),
                message=data.get("message", "Startup configuration updated")
            )

        except httpx.HTTPError as e:
            return StartupScriptResult(
                success=False,
                message=f"Failed to update startup config: {str(e)}"
            )

    def _validate_startup_updates(self, updates: Dict[str, Any]) -> List[str]:
        """Validate startup configuration updates.

        Args:
            updates: Updates to validate

        Returns:
            List of validation errors (empty if valid)
        """
        errors = []

        # Validate command
        if "command" in updates:
            if not isinstance(updates["command"], str):
                errors.append("command must be a string")
            elif not updates["command"].strip():
                errors.append("command cannot be empty")

        # Validate args
        if "args" in updates:
            if not isinstance(updates["args"], list):
                errors.append("args must be a list")
            elif not all(isinstance(arg, str) for arg in updates["args"]):
                errors.append("all args must be strings")

        # Validate env
        if "env" in updates:
            if not isinstance(updates["env"], dict):
                errors.append("env must be a dictionary")
            elif not all(isinstance(k, str) and isinstance(v, str) for k, v in updates["env"].items()):
                errors.append("env keys and values must be strings")

        # Validate working_dir
        if "working_dir" in updates:
            if not isinstance(updates["working_dir"], str):
                errors.append("working_dir must be a string")

        return errors

    async def manage_docker_services(
        self,
        server_name: str,
        action: Literal["start", "stop", "restart", "status"]
    ) -> Dict[str, Any]:
        """Manage Docker services for MCP servers.

        Args:
            server_name: Server to manage
            action: Action to perform

        Returns:
            Dict with operation result
        """
        # Note: This would integrate with mcpproxy's Docker isolation feature
        # Current implementation returns status information

        try:
            # Get server configuration
            response = await self.client.get(
                f"{self.base_url}/api/v1/agent/servers/{server_name}/config"
            )
            response.raise_for_status()
            config = response.json()

            # Check if Docker isolation is enabled
            # (This would need to be exposed in the API)
            docker_enabled = False  # Placeholder

            if action == "status":
                return {
                    "success": True,
                    "service_name": server_name,
                    "docker_enabled": docker_enabled,
                    "status": "running" if config.get("enabled") else "stopped",
                    "message": f"Server {server_name} status retrieved"
                }
            elif action == "start":
                # Enable the server
                update_result = await self.update_startup_script(
                    server_name,
                    {"enabled": True}
                )
                return {
                    "success": update_result.success,
                    "message": f"Server {server_name} started" if update_result.success else update_result.message
                }
            elif action == "stop":
                # Disable the server
                update_result = await self.update_startup_script(
                    server_name,
                    {"enabled": False}
                )
                return {
                    "success": update_result.success,
                    "message": f"Server {server_name} stopped" if update_result.success else update_result.message
                }
            elif action == "restart":
                # Restart by disabling then enabling
                await self.update_startup_script(server_name, {"enabled": False})
                update_result = await self.update_startup_script(
                    server_name,
                    {"enabled": True}
                )
                return {
                    "success": update_result.success,
                    "message": f"Server {server_name} restarted" if update_result.success else update_result.message
                }

        except httpx.HTTPError as e:
            return {
                "success": False,
                "message": f"Docker service management failed: {str(e)}"
            }

    async def get_service_status(self, server_name: str) -> ServiceStatus:
        """Get detailed status of managed service.

        Args:
            server_name: Server to check

        Returns:
            ServiceStatus with detailed information
        """
        try:
            # Get server details
            response = await self.client.get(
                f"{self.base_url}/api/v1/agent/servers/{server_name}"
            )
            response.raise_for_status()
            data = response.json()

            status_info = data.get("status", {})
            is_running = status_info.get("connected", False)
            state = status_info.get("state", "Unknown")

            return ServiceStatus(
                service_name=server_name,
                running=is_running,
                status=state,
                details={
                    "enabled": data.get("enabled", False),
                    "quarantined": data.get("quarantined", False),
                    "tool_count": data.get("tools", {}).get("count", 0)
                }
            )

        except httpx.HTTPError as e:
            return ServiceStatus(
                service_name=server_name,
                running=False,
                status="Error",
                details={"error": str(e)}
            )

    async def install_dependencies(
        self,
        server_name: str,
        dependencies: List[str]
    ) -> Dict[str, Any]:
        """Install additional dependencies for server.

        Args:
            server_name: Server to install dependencies for
            dependencies: List of dependencies to install

        Returns:
            Dict with installation result
        """
        # Note: This would require integration with package managers
        # For now, return guidance

        return {
            "success": False,
            "message": "Dependency installation requires manual setup",
            "recommendations": [
                f"Install {dep} using appropriate package manager" for dep in dependencies
            ],
            "dependencies": dependencies
        }

    async def close(self):
        """Close HTTP client."""
        await self.client.aclose()
