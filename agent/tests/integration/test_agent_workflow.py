"""Integration tests for LangGraph agent workflows.

This module tests the complete agent workflow execution, including:
- State transitions and graph execution
- Tool coordination and data flow
- Decision-making logic
- Error handling across components

Usage:
    # Run all integration tests
    pytest tests/integration/test_agent_workflow.py -v

    # Run specific test
    pytest tests/integration/test_agent_workflow.py::TestAgentDiagnosticWorkflow -v
"""

from typing import Any, Dict
from unittest.mock import AsyncMock, Mock, patch

import pytest

from mcp_agent.graph.agent_graph import (
    AgentInput,
    AgentOutput,
    AgentState,
    MCPAgentGraph,
)
from mcp_agent.tools.diagnostic import DiagnosticTools


# ============================================================================
# Agent Workflow Integration Tests
# ============================================================================


@pytest.mark.integration
@pytest.mark.graph
class TestAgentDiagnosticWorkflow:
    """Test complete diagnostic workflow through LangGraph."""

    @pytest.fixture
    def tools_registry(self, configured_mock_client):
        """Create tools registry for agent."""
        return {
            "diagnostic": DiagnosticTools(configured_mock_client),
        }

    @pytest.mark.asyncio
    async def test_diagnostic_workflow_success(
        self,
        tools_registry,
        initial_agent_state,
    ):
        """Test successful diagnostic workflow execution."""
        agent = MCPAgentGraph(tools_registry)

        user_input = AgentInput(
            request="Debug test-server that is failing",
            server_name="test-server",
            auto_approve=False,
        )

        # Execute workflow
        result = await agent.run(user_input)

        # Verify output structure
        assert isinstance(result, AgentOutput)
        assert len(result.actions_taken) > 0
        assert isinstance(result.recommendations, list)

    @pytest.mark.asyncio
    async def test_diagnostic_workflow_analyzes_request(
        self,
        tools_registry,
    ):
        """Test that workflow correctly analyzes user request."""
        agent = MCPAgentGraph(tools_registry)

        # Create initial state
        state = {
            "user_request": "Debug test-server that is failing",
            "conversation_history": [],
            "current_task": "",
            "task_type": "diagnose",
            "target_server": None,
            "server_status": None,
            "diagnostic_results": None,
            "test_results": None,
            "config_changes": None,
            "suggested_fixes": [],
            "requires_approval": False,
            "approval_granted": False,
            "next_action": None,
            "error": None,
            "completed": False,
        }

        # Run analyze_request node
        updated_state = await agent._analyze_request(state)

        # Verify task type detection
        assert updated_state["task_type"] == "diagnose"
        assert updated_state["current_task"] == "Diagnosing server issues"
        assert updated_state["target_server"] == "test-server"

    @pytest.mark.asyncio
    async def test_diagnostic_workflow_checks_server_status(
        self,
        tools_registry,
        initial_agent_state,
    ):
        """Test that workflow checks server status."""
        agent = MCPAgentGraph(tools_registry)

        state = initial_agent_state.copy()
        state["target_server"] = "test-server"

        # Run check_server_status node
        updated_state = await agent._check_server_status(state)

        # Verify server status was retrieved
        assert updated_state["server_status"] is not None
        assert "server_name" in updated_state["server_status"]

    @pytest.mark.asyncio
    async def test_diagnostic_workflow_diagnoses_issues(
        self,
        tools_registry,
        initial_agent_state,
    ):
        """Test that workflow performs diagnostic analysis."""
        agent = MCPAgentGraph(tools_registry)

        state = initial_agent_state.copy()
        state["target_server"] = "test-server"

        # Run diagnose node
        updated_state = await agent._diagnose(state)

        # Verify diagnostic results
        assert updated_state["diagnostic_results"] is not None
        assert "log_analysis" in updated_state["diagnostic_results"]
        assert "connection_status" in updated_state["diagnostic_results"]
        assert "tool_analysis" in updated_state["diagnostic_results"]

    @pytest.mark.asyncio
    async def test_diagnostic_workflow_suggests_fixes(
        self,
        tools_registry,
        diagnostic_state_with_results,
    ):
        """Test that workflow generates fix suggestions."""
        agent = MCPAgentGraph(tools_registry)

        # Run suggest_fixes node
        updated_state = await agent._suggest_fixes(diagnostic_state_with_results)

        # Verify fix suggestions
        assert "suggested_fixes" in updated_state
        assert isinstance(updated_state["suggested_fixes"], list)

    @pytest.mark.asyncio
    async def test_diagnostic_workflow_routing(
        self,
        tools_registry,
        initial_agent_state,
    ):
        """Test that workflow routes correctly based on task type."""
        agent = MCPAgentGraph(tools_registry)

        # Test diagnose route
        state = initial_agent_state.copy()
        state["task_type"] = "diagnose"
        state["target_server"] = "test-server"

        route = agent._route_after_analysis(state)
        assert route == "check_status"

        # Test no server route
        state["target_server"] = None
        route = agent._route_after_analysis(state)
        assert route == "end"

    @pytest.mark.asyncio
    async def test_diagnostic_workflow_approval_check(
        self,
        tools_registry,
        diagnostic_state_with_results,
    ):
        """Test approval requirement logic."""
        agent = MCPAgentGraph(tools_registry)

        # Test with fixes requiring approval
        state = diagnostic_state_with_results.copy()
        state["suggested_fixes"] = [
            {"requires_approval": True, "fix_type": "authentication"}
        ]
        state["requires_approval"] = True

        route = agent._check_approval_needed(state)
        assert route == "needs_approval"

        # Test without fixes
        state["suggested_fixes"] = []
        route = agent._check_approval_needed(state)
        assert route == "report"

    @pytest.mark.asyncio
    async def test_diagnostic_workflow_error_handling(
        self,
        tools_registry,
    ):
        """Test workflow handles errors gracefully."""
        agent = MCPAgentGraph(tools_registry)

        user_input = AgentInput(
            request="Debug unknown-server",
            server_name=None,  # No server specified
            auto_approve=False,
        )

        # Should not crash on missing server
        result = await agent.run(user_input)
        assert isinstance(result, AgentOutput)


@pytest.mark.integration
@pytest.mark.graph
class TestAgentStateTransitions:
    """Test LangGraph state transitions."""

    @pytest.fixture
    def tools_registry(self, configured_mock_client):
        """Create tools registry for agent."""
        return {
            "diagnostic": DiagnosticTools(configured_mock_client),
        }

    @pytest.mark.asyncio
    async def test_state_persists_across_nodes(
        self,
        tools_registry,
        initial_agent_state,
    ):
        """Test that state is preserved across node transitions."""
        agent = MCPAgentGraph(tools_registry)

        # Start with initial state
        state = initial_agent_state.copy()

        # Run through multiple nodes
        state = await agent._analyze_request(state)
        initial_task = state["current_task"]

        state = await agent._check_server_status(state)

        # Verify state from previous node persists
        assert state["current_task"] == initial_task
        assert state["target_server"] == "test-server"

    @pytest.mark.asyncio
    async def test_conversation_history_accumulates(
        self,
        tools_registry,
    ):
        """Test that conversation history accumulates correctly."""
        agent = MCPAgentGraph(tools_registry)

        state = {
            "user_request": "Test request",
            "conversation_history": [
                {"role": "user", "content": "First message"},
            ],
            "current_task": "",
            "task_type": "diagnose",
            "target_server": "test-server",
            "server_status": None,
            "diagnostic_results": None,
            "test_results": None,
            "config_changes": None,
            "suggested_fixes": [],
            "requires_approval": False,
            "approval_granted": False,
            "next_action": None,
            "error": None,
            "completed": False,
        }

        # Process through workflow
        state = await agent._analyze_request(state)

        # Verify history preserved
        assert len(state["conversation_history"]) >= 1


@pytest.mark.integration
@pytest.mark.graph
class TestAgentResponseBuilding:
    """Test agent response generation."""

    @pytest.fixture
    def tools_registry(self, configured_mock_client):
        """Create tools registry for agent."""
        return {
            "diagnostic": DiagnosticTools(configured_mock_client),
        }

    def test_build_response_with_diagnostic_results(
        self,
        tools_registry,
        diagnostic_state_with_results,
    ):
        """Test response building with diagnostic results."""
        agent = MCPAgentGraph(tools_registry)

        response = agent._build_response(diagnostic_state_with_results)

        # Verify response structure
        assert isinstance(response, AgentOutput)
        assert len(response.actions_taken) > 0
        assert "Analyzed server logs" in response.actions_taken[0]
        assert len(response.recommendations) > 0

    def test_build_response_with_fixes(
        self,
        tools_registry,
        diagnostic_state_with_results,
    ):
        """Test response building with suggested fixes."""
        agent = MCPAgentGraph(tools_registry)

        state = diagnostic_state_with_results.copy()
        state["suggested_fixes"] = [
            {"fix_type": "authentication", "requires_approval": True}
        ]

        response = agent._build_response(state)

        assert any("fix suggestions" in action for action in response.actions_taken)
        assert response.requires_user_action is False  # requires_approval not set in state

    def test_build_response_with_error(
        self,
        tools_registry,
        initial_agent_state,
    ):
        """Test response building with error state."""
        agent = MCPAgentGraph(tools_registry)

        state = initial_agent_state.copy()
        state["error"] = "Server not found"

        response = agent._build_response(state)

        assert "Error: Server not found" in response.response


@pytest.mark.integration
@pytest.mark.slow
class TestAgentMemoryPersistence:
    """Test agent memory and state persistence."""

    @pytest.fixture
    def tools_registry(self, configured_mock_client):
        """Create tools registry for agent."""
        return {
            "diagnostic": DiagnosticTools(configured_mock_client),
        }

    @pytest.mark.asyncio
    async def test_memory_checkpointer_initialization(
        self,
        tools_registry,
    ):
        """Test that memory checkpointer is initialized."""
        agent = MCPAgentGraph(tools_registry)

        assert agent.memory is not None

    # Note: More comprehensive memory tests would require
    # actual database operations and are better suited for E2E tests
