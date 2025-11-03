"""Unit tests for ConfigTools."""

import pytest
from datetime import datetime
from unittest.mock import AsyncMock, MagicMock, patch
from httpx import Response

from mcp_agent.tools.config import (
    ConfigTools,
    ServerConfig,
    ValidationResult,
    ConfigUpdateResult,
    BackupResult,
)


# Fixtures

@pytest.fixture
def config_tools():
    """Create ConfigTools instance."""
    return ConfigTools(base_url="http://localhost:8080")


@pytest.fixture
def config_tools_with_token():
    """Create ConfigTools instance with API token."""
    return ConfigTools(base_url="http://localhost:8080", api_token="test-token")


@pytest.fixture
def sample_server_config():
    """Sample server configuration."""
    return {
        "name": "github-server",
        "url": "https://api.github.com/mcp",
        "protocol": "http",
        "enabled": True,
        "quarantined": False,
        "env": {"API_KEY": "test-key"},
        "args": []
    }


@pytest.fixture
def sample_stdio_config():
    """Sample stdio server configuration."""
    return {
        "name": "python-server",
        "command": "uvx",
        "args": ["some-package"],
        "protocol": "stdio",
        "enabled": True,
        "quarantined": False,
        "working_dir": "/home/user/project",
        "env": {"PYTHON_ENV": "production"}
    }


@pytest.fixture
def sample_backup_result():
    """Sample backup result."""
    return {
        "backup_id": "backup_123",
        "timestamp": "2025-01-01T12:00:00",
        "servers": ["github-server", "python-server"],
        "path": "/backups/backup_123.json"
    }


# Pydantic Model Tests

class TestServerConfig:
    """Test ServerConfig model."""

    def test_server_config_http(self, sample_server_config):
        """Test HTTP server config creation."""
        config = ServerConfig(**sample_server_config)

        assert config.name == "github-server"
        assert config.url == "https://api.github.com/mcp"
        assert config.protocol == "http"
        assert config.enabled is True
        assert config.quarantined is False
        assert config.env == {"API_KEY": "test-key"}
        assert config.args == []
        assert config.command is None
        assert config.working_dir is None

    def test_server_config_stdio(self, sample_stdio_config):
        """Test stdio server config creation."""
        config = ServerConfig(**sample_stdio_config)

        assert config.name == "python-server"
        assert config.command == "uvx"
        assert config.args == ["some-package"]
        assert config.protocol == "stdio"
        assert config.working_dir == "/home/user/project"
        assert config.env == {"PYTHON_ENV": "production"}
        assert config.url is None

    def test_server_config_defaults(self):
        """Test server config with default values."""
        config = ServerConfig(name="test-server")

        assert config.name == "test-server"
        assert config.url is None
        assert config.command is None
        assert config.args == []
        assert config.env == {}
        assert config.protocol == "auto"
        assert config.enabled is True
        assert config.quarantined is False
        assert config.working_dir is None


class TestValidationResult:
    """Test ValidationResult model."""

    def test_validation_result_valid(self):
        """Test valid validation result."""
        result = ValidationResult(
            is_valid=True,
            errors=[],
            warnings=["Warning message"],
            suggestions=["Suggestion message"]
        )

        assert result.is_valid is True
        assert result.errors == []
        assert result.warnings == ["Warning message"]
        assert result.suggestions == ["Suggestion message"]

    def test_validation_result_invalid(self):
        """Test invalid validation result."""
        result = ValidationResult(
            is_valid=False,
            errors=["Error 1", "Error 2"],
            warnings=["Warning"],
            suggestions=[]
        )

        assert result.is_valid is False
        assert len(result.errors) == 2
        assert result.warnings == ["Warning"]
        assert result.suggestions == []

    def test_validation_result_defaults(self):
        """Test validation result with defaults."""
        result = ValidationResult(is_valid=True)

        assert result.is_valid is True
        assert result.errors == []
        assert result.warnings == []
        assert result.suggestions == []


class TestConfigUpdateResult:
    """Test ConfigUpdateResult model."""

    def test_config_update_success(self, sample_server_config):
        """Test successful config update result."""
        result = ConfigUpdateResult(
            success=True,
            message="Configuration updated",
            previous_config=sample_server_config,
            new_config={**sample_server_config, "enabled": False},
            requires_restart=True
        )

        assert result.success is True
        assert result.message == "Configuration updated"
        assert result.previous_config is not None
        assert result.new_config is not None
        assert result.requires_restart is True

    def test_config_update_failure(self):
        """Test failed config update result."""
        result = ConfigUpdateResult(
            success=False,
            message="Validation failed",
            requires_restart=False
        )

        assert result.success is False
        assert result.message == "Validation failed"
        assert result.previous_config is None
        assert result.new_config is None
        assert result.requires_restart is False


class TestBackupResult:
    """Test BackupResult model."""

    def test_backup_result(self, sample_backup_result):
        """Test backup result creation."""
        result = BackupResult(**sample_backup_result)

        assert result.backup_id == "backup_123"
        assert isinstance(result.timestamp, datetime)
        assert result.servers == ["github-server", "python-server"]
        assert result.path == "/backups/backup_123.json"


# ConfigTools Tests

class TestConfigToolsInit:
    """Test ConfigTools initialization."""

    def test_init_without_token(self, config_tools):
        """Test initialization without API token."""
        assert config_tools.base_url == "http://localhost:8080"
        assert config_tools.headers == {}
        assert config_tools.client is not None

    def test_init_with_token(self, config_tools_with_token):
        """Test initialization with API token."""
        assert config_tools_with_token.base_url == "http://localhost:8080"
        assert config_tools_with_token.headers == {"Authorization": "Bearer test-token"}
        assert config_tools_with_token.client is not None


class TestReadServerConfig:
    """Test read_server_config method."""

    @pytest.mark.asyncio
    async def test_read_server_config_success(self, config_tools, sample_server_config):
        """Test successful server config retrieval."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_server_config
        mock_response.raise_for_status = MagicMock()

        config_tools.client.get = AsyncMock(return_value=mock_response)

        config = await config_tools.read_server_config("github-server")

        assert isinstance(config, ServerConfig)
        assert config.name == "github-server"
        assert config.url == "https://api.github.com/mcp"
        assert config.protocol == "http"

        config_tools.client.get.assert_called_once_with(
            "/api/v1/agent/servers/github-server/config"
        )

    @pytest.mark.asyncio
    async def test_read_server_config_not_found(self, config_tools):
        """Test server config not found error."""
        mock_response = AsyncMock(spec=Response)
        mock_response.raise_for_status.side_effect = Exception("404 Not Found")

        config_tools.client.get = AsyncMock(return_value=mock_response)

        with pytest.raises(Exception, match="404 Not Found"):
            await config_tools.read_server_config("nonexistent-server")


class TestUpdateServerConfig:
    """Test update_server_config method."""

    @pytest.mark.asyncio
    async def test_update_server_config_success(self, config_tools, sample_server_config):
        """Test successful server config update."""
        updates = {"enabled": False}
        new_config = {**sample_server_config, "enabled": False}

        # Mock read_server_config
        with patch.object(config_tools, 'read_server_config', new_callable=AsyncMock) as mock_read:
            mock_read.return_value = ServerConfig(**sample_server_config)

            # Mock validate_config
            with patch.object(config_tools, 'validate_config', new_callable=AsyncMock) as mock_validate:
                mock_validate.return_value = ValidationResult(is_valid=True)

                # Mock HTTP client
                mock_response = AsyncMock(spec=Response)
                mock_response.json.return_value = new_config
                mock_response.raise_for_status = MagicMock()
                config_tools.client.patch = AsyncMock(return_value=mock_response)

                result = await config_tools.update_server_config("github-server", updates)

                assert result.success is True
                assert result.message == "Configuration updated successfully"
                assert result.previous_config["enabled"] is True
                assert result.new_config["enabled"] is False
                assert result.requires_restart is False

    @pytest.mark.asyncio
    async def test_update_server_config_validation_failure(self, config_tools, sample_server_config):
        """Test update with validation failure."""
        updates = {"command": ""}

        with patch.object(config_tools, 'read_server_config', new_callable=AsyncMock) as mock_read:
            mock_read.return_value = ServerConfig(**sample_server_config)

            with patch.object(config_tools, 'validate_config', new_callable=AsyncMock) as mock_validate:
                mock_validate.return_value = ValidationResult(
                    is_valid=False,
                    errors=["Command is required for stdio protocol"]
                )

                result = await config_tools.update_server_config("github-server", updates)

                assert result.success is False
                assert "Validation failed" in result.message
                assert result.requires_restart is False

    @pytest.mark.asyncio
    async def test_update_server_config_requires_restart(self, config_tools, sample_server_config):
        """Test update that requires restart."""
        updates = {"command": "new-command", "args": ["--new-arg"]}
        new_config = {**sample_server_config, **updates}

        with patch.object(config_tools, 'read_server_config', new_callable=AsyncMock) as mock_read:
            mock_read.return_value = ServerConfig(**sample_server_config)

            with patch.object(config_tools, 'validate_config', new_callable=AsyncMock) as mock_validate:
                mock_validate.return_value = ValidationResult(is_valid=True)

                mock_response = AsyncMock(spec=Response)
                mock_response.json.return_value = new_config
                mock_response.raise_for_status = MagicMock()
                config_tools.client.patch = AsyncMock(return_value=mock_response)

                result = await config_tools.update_server_config("github-server", updates)

                assert result.success is True
                assert result.requires_restart is True

    @pytest.mark.asyncio
    async def test_update_server_config_without_validation(self, config_tools, sample_server_config):
        """Test update without validation."""
        updates = {"enabled": False}
        new_config = {**sample_server_config, "enabled": False}

        with patch.object(config_tools, 'read_server_config', new_callable=AsyncMock) as mock_read:
            mock_read.return_value = ServerConfig(**sample_server_config)

            mock_response = AsyncMock(spec=Response)
            mock_response.json.return_value = new_config
            mock_response.raise_for_status = MagicMock()
            config_tools.client.patch = AsyncMock(return_value=mock_response)

            result = await config_tools.update_server_config(
                "github-server",
                updates,
                validate=False
            )

            assert result.success is True


class TestValidateConfig:
    """Test validate_config method."""

    @pytest.mark.asyncio
    async def test_validate_config_valid_http(self, config_tools, sample_server_config):
        """Test validation of valid HTTP config."""
        result = await config_tools.validate_config(sample_server_config)

        assert result.is_valid is True
        assert len(result.errors) == 0
        assert len(result.suggestions) == 0

    @pytest.mark.asyncio
    async def test_validate_config_valid_stdio(self, config_tools, sample_stdio_config):
        """Test validation of valid stdio config."""
        result = await config_tools.validate_config(sample_stdio_config)

        assert result.is_valid is True
        assert len(result.errors) == 0

    @pytest.mark.asyncio
    async def test_validate_config_missing_name(self, config_tools):
        """Test validation with missing name."""
        config = {"protocol": "http", "url": "https://example.com"}
        result = await config_tools.validate_config(config)

        assert result.is_valid is False
        assert any("name" in error.lower() for error in result.errors)

    @pytest.mark.asyncio
    async def test_validate_config_stdio_missing_command(self, config_tools):
        """Test validation of stdio config without command."""
        config = {
            "name": "test-server",
            "protocol": "stdio"
        }
        result = await config_tools.validate_config(config)

        assert result.is_valid is False
        assert any("command" in error.lower() for error in result.errors)

    @pytest.mark.asyncio
    async def test_validate_config_http_missing_url(self, config_tools):
        """Test validation of HTTP config without URL."""
        config = {
            "name": "test-server",
            "protocol": "http"
        }
        result = await config_tools.validate_config(config)

        assert result.is_valid is False
        assert any("url" in error.lower() for error in result.errors)

    @pytest.mark.asyncio
    async def test_validate_config_enabled_and_quarantined_warning(self, config_tools):
        """Test warning for enabled+quarantined server."""
        config = {
            "name": "test-server",
            "protocol": "http",
            "url": "https://example.com",
            "enabled": True,
            "quarantined": True
        }
        result = await config_tools.validate_config(config)

        assert result.is_valid is True
        assert len(result.warnings) > 0
        assert any("quarantined" in warning.lower() for warning in result.warnings)

    @pytest.mark.asyncio
    async def test_validate_config_auto_protocol_suggestion(self, config_tools):
        """Test suggestion for auto protocol."""
        config = {
            "name": "test-server",
            "protocol": "auto",
            "command": "test-command"
        }
        result = await config_tools.validate_config(config)

        assert result.is_valid is True
        assert len(result.suggestions) > 0
        assert any("protocol" in suggestion.lower() for suggestion in result.suggestions)


class TestBackupConfig:
    """Test backup_config method."""

    @pytest.mark.asyncio
    async def test_backup_config_all_servers(self, config_tools, sample_backup_result):
        """Test backing up all servers."""
        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = sample_backup_result
        mock_response.raise_for_status = MagicMock()

        config_tools.client.post = AsyncMock(return_value=mock_response)

        result = await config_tools.backup_config()

        assert isinstance(result, BackupResult)
        assert result.backup_id == "backup_123"
        assert len(result.servers) == 2

        config_tools.client.post.assert_called_once_with(
            "/api/v1/agent/config/backup",
            params={}
        )

    @pytest.mark.asyncio
    async def test_backup_config_single_server(self, config_tools, sample_backup_result):
        """Test backing up single server."""
        single_server_backup = {
            **sample_backup_result,
            "servers": ["github-server"]
        }

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = single_server_backup
        mock_response.raise_for_status = MagicMock()

        config_tools.client.post = AsyncMock(return_value=mock_response)

        result = await config_tools.backup_config(server_name="github-server")

        assert isinstance(result, BackupResult)
        assert len(result.servers) == 1
        assert result.servers[0] == "github-server"

        config_tools.client.post.assert_called_once_with(
            "/api/v1/agent/config/backup",
            params={"server": "github-server"}
        )


class TestRestoreConfig:
    """Test restore_config method."""

    @pytest.mark.asyncio
    async def test_restore_config_success(self, config_tools):
        """Test successful config restoration."""
        restore_result = {
            "success": True,
            "message": "Configuration restored from backup_123"
        }

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = restore_result
        mock_response.raise_for_status = MagicMock()

        config_tools.client.post = AsyncMock(return_value=mock_response)

        result = await config_tools.restore_config("backup_123")

        assert isinstance(result, ConfigUpdateResult)
        assert result.success is True
        assert "backup_123" in result.message
        assert result.requires_restart is True

        config_tools.client.post.assert_called_once_with(
            "/api/v1/agent/config/restore",
            json={"backup_id": "backup_123"}
        )

    @pytest.mark.asyncio
    async def test_restore_config_failure(self, config_tools):
        """Test failed config restoration."""
        restore_result = {
            "success": False,
            "message": "Backup not found: backup_invalid"
        }

        mock_response = AsyncMock(spec=Response)
        mock_response.json.return_value = restore_result
        mock_response.raise_for_status = MagicMock()

        config_tools.client.post = AsyncMock(return_value=mock_response)

        result = await config_tools.restore_config("backup_invalid")

        assert result.success is False
        assert "not found" in result.message.lower()


class TestCheckIfRestartNeeded:
    """Test _check_if_restart_needed method."""

    def test_restart_needed_command_change(self, config_tools):
        """Test restart needed for command change."""
        updates = {"command": "new-command"}
        assert config_tools._check_if_restart_needed(updates) is True

    def test_restart_needed_args_change(self, config_tools):
        """Test restart needed for args change."""
        updates = {"args": ["--new-arg"]}
        assert config_tools._check_if_restart_needed(updates) is True

    def test_restart_needed_env_change(self, config_tools):
        """Test restart needed for env change."""
        updates = {"env": {"NEW_VAR": "value"}}
        assert config_tools._check_if_restart_needed(updates) is True

    def test_restart_needed_url_change(self, config_tools):
        """Test restart needed for URL change."""
        updates = {"url": "https://new-url.com"}
        assert config_tools._check_if_restart_needed(updates) is True

    def test_restart_needed_protocol_change(self, config_tools):
        """Test restart needed for protocol change."""
        updates = {"protocol": "http"}
        assert config_tools._check_if_restart_needed(updates) is True

    def test_restart_needed_working_dir_change(self, config_tools):
        """Test restart needed for working_dir change."""
        updates = {"working_dir": "/new/path"}
        assert config_tools._check_if_restart_needed(updates) is True

    def test_restart_not_needed_enabled_change(self, config_tools):
        """Test restart not needed for enabled flag change."""
        updates = {"enabled": False}
        assert config_tools._check_if_restart_needed(updates) is False

    def test_restart_not_needed_quarantined_change(self, config_tools):
        """Test restart not needed for quarantined flag change."""
        updates = {"quarantined": True}
        assert config_tools._check_if_restart_needed(updates) is False

    def test_restart_not_needed_empty_updates(self, config_tools):
        """Test restart not needed for empty updates."""
        updates = {}
        assert config_tools._check_if_restart_needed(updates) is False

    def test_restart_needed_multiple_fields(self, config_tools):
        """Test restart needed when any field requires restart."""
        updates = {
            "enabled": False,
            "command": "new-command",
            "quarantined": True
        }
        assert config_tools._check_if_restart_needed(updates) is True
