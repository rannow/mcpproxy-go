"""Configuration management tools for MCP servers."""

from typing import Optional, Dict, Any
from pydantic import BaseModel, Field
from datetime import datetime
import httpx


class ServerConfig(BaseModel):
    """MCP server configuration."""
    name: str
    url: Optional[str] = None
    command: Optional[str] = None
    args: list[str] = Field(default_factory=list)
    env: Dict[str, str] = Field(default_factory=dict)
    protocol: str = "auto"
    enabled: bool = True
    quarantined: bool = False
    working_dir: Optional[str] = None


class ValidationResult(BaseModel):
    """Configuration validation result."""
    is_valid: bool
    errors: list[str] = Field(default_factory=list)
    warnings: list[str] = Field(default_factory=list)
    suggestions: list[str] = Field(default_factory=list)


class ConfigUpdateResult(BaseModel):
    """Result of configuration update."""
    success: bool
    message: str
    previous_config: Optional[Dict[str, Any]] = None
    new_config: Optional[Dict[str, Any]] = None
    requires_restart: bool = False


class BackupResult(BaseModel):
    """Configuration backup result."""
    backup_id: str
    timestamp: datetime
    servers: list[str]
    path: str


class ConfigTools:
    """Tools for managing MCP server configurations."""

    def __init__(self, base_url: str = "http://localhost:8080", api_token: Optional[str] = None):
        self.base_url = base_url
        self.headers = {}
        if api_token:
            self.headers["Authorization"] = f"Bearer {api_token}"
        self.client = httpx.AsyncClient(base_url=base_url, headers=self.headers)

    async def read_server_config(self, server_name: str) -> ServerConfig:
        """
        Read server configuration.

        Args:
            server_name: Name of the MCP server

        Returns:
            ServerConfig with current configuration
        """
        response = await self.client.get(f"/api/v1/agent/servers/{server_name}/config")
        response.raise_for_status()
        config_data = response.json()

        return ServerConfig(**config_data)

    async def update_server_config(
        self,
        server_name: str,
        updates: Dict[str, Any],
        validate: bool = True,
    ) -> ConfigUpdateResult:
        """
        Update server configuration.

        Args:
            server_name: Name of the MCP server
            updates: Dictionary of configuration updates
            validate: Whether to validate before applying

        Returns:
            ConfigUpdateResult with status and changes
        """
        # Get current config
        current_config = await self.read_server_config(server_name)

        # Validate if requested
        if validate:
            validation = await self.validate_config({**current_config.model_dump(), **updates})
            if not validation.is_valid:
                return ConfigUpdateResult(
                    success=False,
                    message=f"Validation failed: {', '.join(validation.errors)}",
                    previous_config=current_config.model_dump(),
                    requires_restart=False,
                )

        # Apply updates
        response = await self.client.patch(
            f"/api/v1/agent/servers/{server_name}/config",
            json=updates,
        )
        response.raise_for_status()
        new_config = response.json()

        requires_restart = self._check_if_restart_needed(updates)

        return ConfigUpdateResult(
            success=True,
            message="Configuration updated successfully",
            previous_config=current_config.model_dump(),
            new_config=new_config,
            requires_restart=requires_restart,
        )

    async def validate_config(self, config: Dict[str, Any]) -> ValidationResult:
        """
        Validate configuration structure and values.

        Args:
            config: Configuration dictionary to validate

        Returns:
            ValidationResult with validation status and issues
        """
        errors = []
        warnings = []
        suggestions = []

        # Basic validation
        if not config.get("name"):
            errors.append("Server name is required")

        protocol = config.get("protocol", "auto")
        if protocol == "stdio":
            if not config.get("command"):
                errors.append("Command is required for stdio protocol")
        elif protocol in ["http", "sse"]:
            if not config.get("url"):
                errors.append("URL is required for HTTP/SSE protocol")

        # Check for common issues
        if config.get("enabled") and config.get("quarantined"):
            warnings.append("Server is enabled but quarantined - it won't be accessible")

        # Suggestions
        if config.get("protocol") == "auto":
            suggestions.append("Consider setting explicit protocol for better performance")

        return ValidationResult(
            is_valid=len(errors) == 0,
            errors=errors,
            warnings=warnings,
            suggestions=suggestions,
        )

    async def backup_config(self, server_name: Optional[str] = None) -> BackupResult:
        """
        Backup configurations.

        Args:
            server_name: Specific server to backup, or None for all servers

        Returns:
            BackupResult with backup information
        """
        endpoint = "/api/v1/agent/config/backup"
        params = {}
        if server_name:
            params["server"] = server_name

        response = await self.client.post(endpoint, params=params)
        response.raise_for_status()
        result = response.json()

        return BackupResult(**result)

    async def restore_config(self, backup_id: str) -> ConfigUpdateResult:
        """
        Restore configuration from backup.

        Args:
            backup_id: ID of the backup to restore

        Returns:
            ConfigUpdateResult with restoration status
        """
        response = await self.client.post(
            "/api/v1/agent/config/restore",
            json={"backup_id": backup_id},
        )
        response.raise_for_status()
        result = response.json()

        return ConfigUpdateResult(
            success=result["success"],
            message=result["message"],
            requires_restart=True,
        )

    def _check_if_restart_needed(self, updates: Dict[str, Any]) -> bool:
        """Check if configuration changes require server restart."""
        restart_required_fields = {
            "command",
            "args",
            "env",
            "url",
            "protocol",
            "working_dir",
        }

        return any(field in updates for field in restart_required_fields)
