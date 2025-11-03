"""Unit tests for StartupTools."""

import pytest
from unittest.mock import AsyncMock, MagicMock
from httpx import Response, HTTPError

from mcp_agent.tools.startup import (
    StartupTools,
    StartupConfig,
    StartupScriptResult,
    ServiceStatus,
)


# Fixtures

@pytest.fixture
def startup_tools():
    """Create StartupTools instance."""
    return StartupTools(base_url="http://localhost:8080")


@pytest.fixture
def sample_server_config():
    """Sample server configuration."""
    return {
        "name": "github-server",
        "command": "npx",
        "args": ["@github/mcp-server"],
        "env": {"GITHUB_TOKEN": "secret"},
        "working_dir": "/home/user/projects",
        "enabled": True,
        "quarantined": False
    }


@pytest.fixture
def sample_service_status():
    """Sample service status response."""
    return {
        "status": {
            "connected": True,
            "state": "Ready"
        },
        "enabled": True,
        "quarantined": False,
        "tools": {
            "count": 5
        }
    }


# Pydantic Model Tests

class TestStartupConfig:
    """Test StartupConfig model."""

    def test_startup_config_full(self):
        """Test startup config with all fields."""
        config = StartupConfig(
            server_name="github-server",
            command="npx",
            args=["@github/mcp-server"],
            env={"TOKEN": "secret"},
            working_dir="/home/user",
            auto_start=True,
            restart_on_failure=True
        )

        assert config.server_name == "github-server"
        assert config.command == "npx"
        assert config.args == ["@github/mcp-server"]
        assert config.env == {"TOKEN": "secret"}
        assert config.working_dir == "/home/user"
        assert config.auto_start is True
        assert config.restart_on_failure is True

    def test_startup_config_minimal(self):
        """Test startup config with minimal fields."""
        config = StartupConfig(server_name="test-server")

        assert config.server_name == "test-server"
        assert config.command is None
        assert config.args == []
        assert config.env == {}
        assert config.working_dir is None
        assert config.auto_start is False
        assert config.restart_on_failure is False


class TestStartupScriptResult:
    """Test StartupScriptResult model."""

    def test_result_success(self):
        """Test successful result."""
        config = StartupConfig(server_name="test")
        result = StartupScriptResult(
            success=True,
            message="Configuration updated",
            config=config
        )

        assert result.success is True
        assert result.message == "Configuration updated"
        assert result.config is not None

    def test_result_failure(self):
        """Test failed result."""
        result = StartupScriptResult(
            success=False,
            message="Validation failed"
        )

        assert result.success is False
        assert result.config is None


class TestServiceStatus:
    """Test ServiceStatus model."""

    def test_service_status_running(self):
        """Test running service status."""
        status = ServiceStatus(
            service_name="github-server",
            running=True,
            status="Ready",
            uptime="2h 30m",
            details={"enabled": True}
        )

        assert status.running is True
        assert status.status == "Ready"
        assert status.uptime == "2h 30m"

    def test_service_status_stopped(self):
        """Test stopped service status."""
        status = ServiceStatus(
            service_name="github-server",
            running=False,
            status="Stopped"
        )

        assert status.running is False
        assert status.uptime is None
        assert status.details == {}


# StartupTools Tests

class TestStartupToolsInit:
    """Test StartupTools initialization."""

    def test_init_default(self):
        """Test initialization with default values."""
        tools = StartupTools()

        assert tools.base_url == "http://localhost:8080"
        assert tools.client is not None

    def test_init_custom(self):
        """Test initialization with custom base URL."""
        tools = StartupTools(base_url="http://custom:9000")

        assert tools.base_url == "http://custom:9000"


class TestReadStartupScript:
    """Test read_startup_script method."""

    @pytest.mark.asyncio
    async def test_read_success(self, startup_tools, sample_server_config):
        """Test successful startup script read."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_server_config
        mock_response.raise_for_status = MagicMock()

        startup_tools.client.get = AsyncMock(return_value=mock_response)

        result = await startup_tools.read_startup_script("github-server")

        assert isinstance(result, StartupScriptResult)
        assert result.success is True
        assert result.config is not None
        assert result.config.server_name == "github-server"
        assert result.config.command == "npx"
        assert result.config.args == ["@github/mcp-server"]

    @pytest.mark.asyncio
    async def test_read_http_error(self, startup_tools):
        """Test read with HTTP error."""
        startup_tools.client.get = AsyncMock(
            side_effect=HTTPError("Server not found")
        )

        result = await startup_tools.read_startup_script("nonexistent")

        assert result.success is False
        assert "Failed to read" in result.message


class TestUpdateStartupScript:
    """Test update_startup_script method."""

    @pytest.mark.asyncio
    async def test_update_success(self, startup_tools):
        """Test successful update."""
        updates = {"command": "new-command"}
        response_data = {
            "success": True,
            "message": "Configuration updated"
        }

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = response_data
        mock_response.raise_for_status = MagicMock()

        startup_tools.client.patch = AsyncMock(return_value=mock_response)

        result = await startup_tools.update_startup_script(
            "github-server",
            updates
        )

        assert result.success is True
        assert "updated" in result.message.lower()

    @pytest.mark.asyncio
    async def test_update_validation_failure(self, startup_tools):
        """Test update with validation failure."""
        updates = {"command": ""}  # Empty command

        result = await startup_tools.update_startup_script(
            "github-server",
            updates,
            validate=True
        )

        assert result.success is False
        assert "Validation failed" in result.message

    @pytest.mark.asyncio
    async def test_update_without_validation(self, startup_tools):
        """Test update without validation."""
        updates = {"enabled": True}
        response_data = {"success": True, "message": "Updated"}

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = response_data
        mock_response.raise_for_status = MagicMock()

        startup_tools.client.patch = AsyncMock(return_value=mock_response)

        result = await startup_tools.update_startup_script(
            "github-server",
            updates,
            validate=False
        )

        assert result.success is True

    @pytest.mark.asyncio
    async def test_update_http_error(self, startup_tools):
        """Test update with HTTP error."""
        startup_tools.client.patch = AsyncMock(
            side_effect=HTTPError("Update failed")
        )

        result = await startup_tools.update_startup_script(
            "github-server",
            {"command": "test"}
        )

        assert result.success is False
        assert "Failed to update" in result.message


class TestValidateStartupUpdates:
    """Test _validate_startup_updates method."""

    def test_validate_valid_command(self, startup_tools):
        """Test validation of valid command."""
        updates = {"command": "npx"}
        errors = startup_tools._validate_startup_updates(updates)

        assert len(errors) == 0

    def test_validate_invalid_command_type(self, startup_tools):
        """Test validation of invalid command type."""
        updates = {"command": 123}
        errors = startup_tools._validate_startup_updates(updates)

        assert len(errors) > 0
        assert any("string" in err.lower() for err in errors)

    def test_validate_empty_command(self, startup_tools):
        """Test validation of empty command."""
        updates = {"command": "   "}
        errors = startup_tools._validate_startup_updates(updates)

        assert len(errors) > 0
        assert any("cannot be empty" in err.lower() for err in errors)

    def test_validate_valid_args(self, startup_tools):
        """Test validation of valid args."""
        updates = {"args": ["--flag", "value"]}
        errors = startup_tools._validate_startup_updates(updates)

        assert len(errors) == 0

    def test_validate_invalid_args_type(self, startup_tools):
        """Test validation of invalid args type."""
        updates = {"args": "not-a-list"}
        errors = startup_tools._validate_startup_updates(updates)

        assert len(errors) > 0
        assert any("list" in err.lower() for err in errors)

    def test_validate_invalid_args_content(self, startup_tools):
        """Test validation of invalid args content."""
        updates = {"args": ["valid", 123]}
        errors = startup_tools._validate_startup_updates(updates)

        assert len(errors) > 0
        assert any("strings" in err.lower() for err in errors)

    def test_validate_valid_env(self, startup_tools):
        """Test validation of valid env."""
        updates = {"env": {"KEY": "value"}}
        errors = startup_tools._validate_startup_updates(updates)

        assert len(errors) == 0

    def test_validate_invalid_env_type(self, startup_tools):
        """Test validation of invalid env type."""
        updates = {"env": ["not", "dict"]}
        errors = startup_tools._validate_startup_updates(updates)

        assert len(errors) > 0
        assert any("dictionary" in err.lower() for err in errors)

    def test_validate_invalid_env_content(self, startup_tools):
        """Test validation of invalid env content."""
        updates = {"env": {"KEY": 123}}
        errors = startup_tools._validate_startup_updates(updates)

        assert len(errors) > 0
        assert any("strings" in err.lower() for err in errors)

    def test_validate_valid_working_dir(self, startup_tools):
        """Test validation of valid working_dir."""
        updates = {"working_dir": "/home/user"}
        errors = startup_tools._validate_startup_updates(updates)

        assert len(errors) == 0

    def test_validate_invalid_working_dir_type(self, startup_tools):
        """Test validation of invalid working_dir type."""
        updates = {"working_dir": 123}
        errors = startup_tools._validate_startup_updates(updates)

        assert len(errors) > 0
        assert any("string" in err.lower() for err in errors)

    def test_validate_multiple_errors(self, startup_tools):
        """Test validation with multiple errors."""
        updates = {
            "command": "",
            "args": "not-list",
            "env": ["not", "dict"]
        }
        errors = startup_tools._validate_startup_updates(updates)

        assert len(errors) >= 3


class TestManageDockerServices:
    """Test manage_docker_services method."""

    @pytest.mark.asyncio
    async def test_status_action(self, startup_tools, sample_server_config):
        """Test status action."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_server_config
        mock_response.raise_for_status = MagicMock()

        startup_tools.client.get = AsyncMock(return_value=mock_response)

        result = await startup_tools.manage_docker_services(
            "github-server",
            "status"
        )

        assert result["success"] is True
        assert result["service_name"] == "github-server"

    @pytest.mark.asyncio
    async def test_start_action(self, startup_tools, sample_server_config):
        """Test start action."""
        # Mock config read
        mock_get_response = AsyncMock(spec=Response)
        mock_get_response.json.return_value = sample_server_config
        mock_get_response.raise_for_status = MagicMock()

        # Mock config update
        mock_patch_response = AsyncMock(spec=Response)
        mock_patch_response.json.return_value = {"success": True}
        mock_patch_response.raise_for_status = MagicMock()

        startup_tools.client.get = AsyncMock(return_value=mock_get_response)
        startup_tools.client.patch = AsyncMock(return_value=mock_patch_response)

        result = await startup_tools.manage_docker_services(
            "github-server",
            "start"
        )

        assert result["success"] is True
        assert "started" in result["message"].lower()

    @pytest.mark.asyncio
    async def test_stop_action(self, startup_tools, sample_server_config):
        """Test stop action."""
        mock_get_response = AsyncMock(spec=Response)
        mock_get_response.json.return_value = sample_server_config
        mock_get_response.raise_for_status = MagicMock()

        mock_patch_response = AsyncMock(spec=Response)
        mock_patch_response.json.return_value = {"success": True}
        mock_patch_response.raise_for_status = MagicMock()

        startup_tools.client.get = AsyncMock(return_value=mock_get_response)
        startup_tools.client.patch = AsyncMock(return_value=mock_patch_response)

        result = await startup_tools.manage_docker_services(
            "github-server",
            "stop"
        )

        assert result["success"] is True
        assert "stopped" in result["message"].lower()

    @pytest.mark.asyncio
    async def test_restart_action(self, startup_tools, sample_server_config):
        """Test restart action."""
        mock_get_response = AsyncMock(spec=Response)
        mock_get_response.json.return_value = sample_server_config
        mock_get_response.raise_for_status = MagicMock()

        mock_patch_response = AsyncMock(spec=Response)
        mock_patch_response.json.return_value = {"success": True}
        mock_patch_response.raise_for_status = MagicMock()

        startup_tools.client.get = AsyncMock(return_value=mock_get_response)
        startup_tools.client.patch = AsyncMock(return_value=mock_patch_response)

        result = await startup_tools.manage_docker_services(
            "github-server",
            "restart"
        )

        assert result["success"] is True
        assert "restarted" in result["message"].lower()

    @pytest.mark.asyncio
    async def test_action_http_error(self, startup_tools):
        """Test action with HTTP error."""
        startup_tools.client.get = AsyncMock(
            side_effect=HTTPError("Connection failed")
        )

        result = await startup_tools.manage_docker_services(
            "github-server",
            "status"
        )

        assert result["success"] is False
        assert "failed" in result["message"].lower()


class TestGetServiceStatus:
    """Test get_service_status method."""

    @pytest.mark.asyncio
    async def test_status_running(self, startup_tools, sample_service_status):
        """Test status for running service."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_service_status
        mock_response.raise_for_status = MagicMock()

        startup_tools.client.get = AsyncMock(return_value=mock_response)

        status = await startup_tools.get_service_status("github-server")

        assert isinstance(status, ServiceStatus)
        assert status.service_name == "github-server"
        assert status.running is True
        assert status.status == "Ready"
        assert status.details["enabled"] is True

    @pytest.mark.asyncio
    async def test_status_stopped(self, startup_tools):
        """Test status for stopped service."""
        stopped_status = {
            "status": {
                "connected": False,
                "state": "Disconnected"
            },
            "enabled": False,
            "quarantined": False,
            "tools": {"count": 0}
        }

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = stopped_status
        mock_response.raise_for_status = MagicMock()

        startup_tools.client.get = AsyncMock(return_value=mock_response)

        status = await startup_tools.get_service_status("github-server")

        assert status.running is False
        assert status.status == "Disconnected"

    @pytest.mark.asyncio
    async def test_status_http_error(self, startup_tools):
        """Test status with HTTP error."""
        startup_tools.client.get = AsyncMock(
            side_effect=HTTPError("Server not found")
        )

        status = await startup_tools.get_service_status("github-server")

        assert status.running is False
        assert status.status == "Error"
        assert "error" in status.details


class TestInstallDependencies:
    """Test install_dependencies method."""

    @pytest.mark.asyncio
    async def test_install_dependencies(self, startup_tools):
        """Test dependency installation (returns guidance)."""
        deps = ["package1", "package2"]

        result = await startup_tools.install_dependencies("github-server", deps)

        assert result["success"] is False  # Manual installation required
        assert "manual" in result["message"].lower()
        assert len(result["recommendations"]) == 2
        assert result["dependencies"] == deps


class TestClose:
    """Test close method."""

    @pytest.mark.asyncio
    async def test_close(self, startup_tools):
        """Test closing HTTP client."""
        startup_tools.client.aclose = AsyncMock()

        await startup_tools.close()

        startup_tools.client.aclose.assert_called_once()
