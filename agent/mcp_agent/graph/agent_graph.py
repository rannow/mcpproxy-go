"""LangGraph-based MCP agent orchestration."""

from typing import TypedDict, Annotated, Literal
from langgraph.graph import StateGraph, END
from langgraph.checkpoint.memory import MemorySaver
from pydantic import BaseModel, Field
import operator


class AgentState(TypedDict):
    """State for the MCP management agent."""
    # User input
    user_request: str
    conversation_history: Annotated[list[dict], operator.add]

    # Current task
    current_task: str
    task_type: Literal[
        "diagnose", "test", "configure", "install", "monitor", "learn"
    ]

    # Server context
    target_server: str | None
    server_status: dict | None

    # Analysis results
    diagnostic_results: dict | None
    test_results: dict | None
    config_changes: dict | None

    # Agent decisions
    suggested_fixes: list[dict]
    requires_approval: bool
    approval_granted: bool

    # Workflow control
    next_action: str | None
    error: str | None
    completed: bool


class AgentInput(BaseModel):
    """Input to the agent."""
    request: str = Field(description="User's request or question")
    server_name: str | None = Field(default=None, description="Target MCP server")
    auto_approve: bool = Field(default=False, description="Auto-approve safe fixes")


class AgentOutput(BaseModel):
    """Output from the agent."""
    response: str
    actions_taken: list[str]
    recommendations: list[str]
    requires_user_action: bool
    server_status: dict | None = None


class MCPAgentGraph:
    """LangGraph-based agent for MCP server management."""

    def __init__(self, tools_registry: dict):
        """
        Initialize the agent graph.

        Args:
            tools_registry: Dictionary of tool instances
        """
        self.tools = tools_registry
        self.graph = self._build_graph()
        self.memory = MemorySaver()

    def _build_graph(self) -> StateGraph:
        """Build the LangGraph state machine."""
        workflow = StateGraph(AgentState)

        # Add nodes for each state
        workflow.add_node("analyze_request", self._analyze_request)
        workflow.add_node("check_server_status", self._check_server_status)
        workflow.add_node("diagnose", self._diagnose)
        workflow.add_node("test", self._test)
        workflow.add_node("configure", self._configure)
        workflow.add_node("install", self._install)
        workflow.add_node("suggest_fixes", self._suggest_fixes)
        workflow.add_node("await_approval", self._await_approval)
        workflow.add_node("execute_fixes", self._execute_fixes)
        workflow.add_node("monitor", self._monitor)
        workflow.add_node("report", self._report)

        # Define the workflow edges
        workflow.set_entry_point("analyze_request")

        workflow.add_conditional_edges(
            "analyze_request",
            self._route_after_analysis,
            {
                "check_status": "check_server_status",
                "diagnose": "diagnose",
                "test": "test",
                "configure": "configure",
                "install": "install",
                "end": END,
            }
        )

        workflow.add_edge("check_server_status", "diagnose")
        workflow.add_edge("diagnose", "suggest_fixes")
        workflow.add_edge("test", "report")
        workflow.add_edge("configure", "report")
        workflow.add_edge("install", "monitor")

        workflow.add_conditional_edges(
            "suggest_fixes",
            self._check_approval_needed,
            {
                "needs_approval": "await_approval",
                "execute": "execute_fixes",
                "report": "report",
            }
        )

        workflow.add_conditional_edges(
            "await_approval",
            self._check_approval_status,
            {
                "approved": "execute_fixes",
                "denied": "report",
            }
        )

        workflow.add_edge("execute_fixes", "monitor")
        workflow.add_edge("monitor", "report")
        workflow.add_edge("report", END)

        return workflow.compile(checkpointer=self.memory)

    async def _analyze_request(self, state: AgentState) -> AgentState:
        """Analyze the user's request and determine task type."""
        request = state["user_request"].lower()

        # Simple keyword-based routing (would use LLM in production)
        if any(word in request for word in ["debug", "diagnose", "error", "failing"]):
            state["task_type"] = "diagnose"
            state["current_task"] = "Diagnosing server issues"
        elif any(word in request for word in ["test", "check", "validate"]):
            state["task_type"] = "test"
            state["current_task"] = "Testing server functionality"
        elif any(word in request for word in ["configure", "config", "settings"]):
            state["task_type"] = "configure"
            state["current_task"] = "Managing configuration"
        elif any(word in request for word in ["install", "add", "new server"]):
            state["task_type"] = "install"
            state["current_task"] = "Installing new server"
        else:
            state["task_type"] = "monitor"
            state["current_task"] = "Monitoring server status"

        # Extract server name if mentioned
        # In production, would use NER or LLM extraction
        state["target_server"] = self._extract_server_name(request)

        return state

    async def _check_server_status(self, state: AgentState) -> AgentState:
        """Check the status of the target server."""
        if not state["target_server"]:
            state["error"] = "No target server specified"
            return state

        # Use diagnostic tools to get server status
        diagnostic_tools = self.tools["diagnostic"]
        status = await diagnostic_tools.identify_connection_issues(state["target_server"])

        state["server_status"] = status.model_dump()
        return state

    async def _diagnose(self, state: AgentState) -> AgentState:
        """Diagnose server issues."""
        diagnostic_tools = self.tools["diagnostic"]

        # Analyze logs
        log_analysis = await diagnostic_tools.analyze_server_logs(
            state["target_server"],
            time_range="1h"
        )

        # Check connection
        connection_status = await diagnostic_tools.identify_connection_issues(
            state["target_server"]
        )

        # Analyze tool failures
        tool_analysis = await diagnostic_tools.analyze_tool_failures(
            state["target_server"]
        )

        state["diagnostic_results"] = {
            "log_analysis": log_analysis.model_dump(),
            "connection_status": connection_status.model_dump(),
            "tool_analysis": tool_analysis.model_dump(),
        }

        return state

    async def _test(self, state: AgentState) -> AgentState:
        """Test server functionality."""
        # Would use testing tools
        state["test_results"] = {"status": "passed", "tests_run": 0}
        return state

    async def _configure(self, state: AgentState) -> AgentState:
        """Manage server configuration."""
        # Would use config tools
        state["config_changes"] = {}
        return state

    async def _install(self, state: AgentState) -> AgentState:
        """Install new server."""
        # Would use discovery tools
        return state

    async def _suggest_fixes(self, state: AgentState) -> AgentState:
        """Generate fix suggestions based on diagnostic results."""
        diagnostic_tools = self.tools["diagnostic"]

        fixes = await diagnostic_tools.suggest_fixes(
            state["diagnostic_results"]
        )

        state["suggested_fixes"] = [fix.model_dump() for fix in fixes]
        state["requires_approval"] = any(fix.requires_approval for fix in fixes)

        return state

    async def _await_approval(self, state: AgentState) -> AgentState:
        """Wait for user approval of suggested fixes."""
        # In production, this would pause execution and wait for user input
        # For now, we'll just mark it as requiring approval
        state["approval_granted"] = False  # User must explicitly approve
        return state

    async def _execute_fixes(self, state: AgentState) -> AgentState:
        """Execute approved fixes."""
        # Would execute the approved fixes
        return state

    async def _monitor(self, state: AgentState) -> AgentState:
        """Monitor server after changes."""
        # Would monitor server health
        return state

    async def _report(self, state: AgentState) -> AgentState:
        """Generate final report."""
        state["completed"] = True
        return state

    def _route_after_analysis(self, state: AgentState) -> str:
        """Route to appropriate node after analysis."""
        task_type = state.get("task_type")

        if not state.get("target_server"):
            return "end"

        if task_type == "diagnose":
            return "check_status"
        elif task_type == "test":
            return "test"
        elif task_type == "configure":
            return "configure"
        elif task_type == "install":
            return "install"
        else:
            return "check_status"

    def _check_approval_needed(self, state: AgentState) -> str:
        """Check if approval is needed for suggested fixes."""
        if not state.get("suggested_fixes"):
            return "report"

        if state.get("requires_approval", False):
            return "needs_approval"
        else:
            return "execute"

    def _check_approval_status(self, state: AgentState) -> str:
        """Check if approval was granted."""
        return "approved" if state.get("approval_granted", False) else "denied"

    def _extract_server_name(self, request: str) -> str | None:
        """Extract server name from request."""
        # Simple extraction - would use NER or LLM in production
        words = request.split()
        for i, word in enumerate(words):
            if word in ["server", "for"] and i + 1 < len(words):
                return words[i + 1].strip(",.:;")
        return None

    async def run(self, user_input: AgentInput) -> AgentOutput:
        """
        Run the agent on a user request.

        Args:
            user_input: User's request and parameters

        Returns:
            AgentOutput with response and actions
        """
        initial_state: AgentState = {
            "user_request": user_input.request,
            "conversation_history": [{"role": "user", "content": user_input.request}],
            "current_task": "",
            "task_type": "diagnose",
            "target_server": user_input.server_name,
            "server_status": None,
            "diagnostic_results": None,
            "test_results": None,
            "config_changes": None,
            "suggested_fixes": [],
            "requires_approval": False,
            "approval_granted": user_input.auto_approve,
            "next_action": None,
            "error": None,
            "completed": False,
        }

        # Run the graph
        final_state = await self.graph.ainvoke(initial_state)

        # Build response
        response = self._build_response(final_state)

        return response

    def _build_response(self, state: AgentState) -> AgentOutput:
        """Build the output response from final state."""
        actions_taken = []
        recommendations = []

        if state.get("diagnostic_results"):
            actions_taken.append("Analyzed server logs and diagnostics")
            if state["diagnostic_results"].get("log_analysis"):
                recs = state["diagnostic_results"]["log_analysis"].get("recommendations", [])
                recommendations.extend(recs)

        if state.get("suggested_fixes"):
            actions_taken.append(f"Generated {len(state['suggested_fixes'])} fix suggestions")

        response_text = f"Completed: {state.get('current_task', 'Task')}\n"
        if state.get("error"):
            response_text += f"Error: {state['error']}\n"

        return AgentOutput(
            response=response_text,
            actions_taken=actions_taken,
            recommendations=recommendations,
            requires_user_action=state.get("requires_approval", False),
            server_status=state.get("server_status"),
        )
