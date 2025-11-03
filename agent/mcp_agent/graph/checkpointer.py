"""Checkpointer configuration for LangGraph state persistence.

Supports both in-memory (testing) and PostgreSQL (production) checkpointers.
"""

import os
from typing import Optional
from langgraph.checkpoint.memory import MemorySaver

# Optional PostgreSQL support - only import if available
try:
    from langgraph.checkpoint.postgres import PostgresSaver
    POSTGRES_AVAILABLE = True
except ImportError:
    POSTGRES_AVAILABLE = False


def create_checkpointer(
    postgres_url: Optional[str] = None,
    use_postgres: bool = False
) -> MemorySaver:
    """
    Create a checkpointer for LangGraph state persistence.

    Args:
        postgres_url: PostgreSQL connection string (e.g., "postgresql://user:pass@localhost/db")
        use_postgres: Force PostgreSQL usage (default: auto-detect from environment)

    Returns:
        Checkpointer instance (MemorySaver or PostgresSaver)

    Environment Variables:
        MCPPROXY_POSTGRES_URL: PostgreSQL connection string
        MCPPROXY_USE_POSTGRES: "true" to enable PostgreSQL (default: "false")

    Examples:
        # In-memory (testing)
        checkpointer = create_checkpointer()

        # PostgreSQL (production)
        checkpointer = create_checkpointer(
            postgres_url="postgresql://user:pass@localhost:5432/mcpproxy"
        )

        # PostgreSQL from environment
        os.environ["MCPPROXY_POSTGRES_URL"] = "postgresql://..."
        os.environ["MCPPROXY_USE_POSTGRES"] = "true"
        checkpointer = create_checkpointer()
    """
    # Check environment variables
    env_postgres_url = os.getenv("MCPPROXY_POSTGRES_URL")
    env_use_postgres = os.getenv("MCPPROXY_USE_POSTGRES", "false").lower() == "true"

    # Determine final settings
    final_postgres_url = postgres_url or env_postgres_url
    final_use_postgres = use_postgres or env_use_postgres

    # Create PostgreSQL checkpointer if requested and available
    if final_use_postgres and final_postgres_url:
        if not POSTGRES_AVAILABLE:
            raise ImportError(
                "PostgreSQL checkpointer requested but langgraph.checkpoint.postgres not available. "
                "Install with: pip install langgraph[postgres]"
            )

        return PostgresSaver(connection_string=final_postgres_url)

    # Default: in-memory checkpointer for testing
    return MemorySaver()


def get_checkpointer_info(checkpointer) -> dict:
    """
    Get information about the checkpointer type and configuration.

    Args:
        checkpointer: Checkpointer instance

    Returns:
        Dictionary with checkpointer information
    """
    checkpointer_type = type(checkpointer).__name__

    info = {
        "type": checkpointer_type,
        "persistent": checkpointer_type == "PostgresSaver",
        "production_ready": checkpointer_type == "PostgresSaver",
    }

    if checkpointer_type == "PostgresSaver":
        # Add PostgreSQL-specific info if available
        info["connection"] = getattr(checkpointer, "connection_string", "configured")

    return info
