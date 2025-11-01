"""CLI interface for MCP agent."""

import asyncio
from typing import Optional
import typer
from rich.console import Console
from rich.panel import Panel
from rich.markdown import Markdown
from rich.table import Table

from mcp_agent.tools.diagnostic import DiagnosticTools, MCPProxyClient
from mcp_agent.tools.config import ConfigTools
from mcp_agent.graph.agent_graph import MCPAgentGraph, AgentInput

app = typer.Typer(help="MCP Server Management Agent")
console = Console()


@app.command()
def diagnose(
    server: str = typer.Argument(..., help="Server name to diagnose"),
    auto_fix: bool = typer.Option(False, "--auto-fix", help="Automatically apply safe fixes"),
):
    """Diagnose issues with an MCP server."""
    console.print(Panel(f"[bold blue]Diagnosing server:[/bold blue] {server}"))

    asyncio.run(_run_diagnosis(server, auto_fix))


@app.command()
def test(
    server: str = typer.Argument(..., help="Server name to test"),
    tool: Optional[str] = typer.Option(None, "--tool", help="Specific tool to test"),
):
    """Test MCP server functionality."""
    console.print(Panel(f"[bold blue]Testing server:[/bold blue] {server}"))

    if tool:
        console.print(f"Testing specific tool: {tool}")


@app.command()
def search(
    query: str = typer.Argument(..., help="Search query for MCP servers"),
    registry: Optional[str] = typer.Option(None, "--registry", help="Specific registry to search"),
):
    """Search for MCP servers in registries."""
    console.print(Panel(f"[bold blue]Searching for:[/bold blue] {query}"))


@app.command()
def install(
    server_id: str = typer.Argument(..., help="Server ID to install"),
    auto_configure: bool = typer.Option(True, "--auto-configure", help="Auto-configure after install"),
):
    """Install a new MCP server."""
    console.print(Panel(f"[bold blue]Installing server:[/bold blue] {server_id}"))


@app.command()
def chat(
    message: Optional[str] = typer.Argument(None, help="Message to send to agent"),
    interactive: bool = typer.Option(False, "--interactive", "-i", help="Start interactive mode"),
):
    """Chat with the MCP agent."""
    if interactive:
        asyncio.run(_interactive_chat())
    elif message:
        asyncio.run(_single_chat(message))
    else:
        console.print("[red]Error:[/red] Provide a message or use --interactive")


@app.command()
def status(
    server: Optional[str] = typer.Option(None, "--server", help="Specific server status"),
):
    """Show status of MCP servers."""
    asyncio.run(_show_status(server))


async def _run_diagnosis(server: str, auto_fix: bool):
    """Run diagnostic workflow."""
    try:
        # Initialize tools
        client = MCPProxyClient()
        diagnostic_tools = DiagnosticTools(client)

        # Create tools registry
        tools_registry = {
            "diagnostic": diagnostic_tools,
            "config": ConfigTools(),
        }

        # Create agent
        agent = MCPAgentGraph(tools_registry)

        # Run diagnosis
        with console.status("[bold green]Analyzing server..."):
            result = await agent.run(
                AgentInput(
                    request=f"Diagnose issues with {server}",
                    server_name=server,
                    auto_approve=auto_fix,
                )
            )

        # Display results
        console.print("\n[bold green]Diagnosis Complete[/bold green]\n")
        console.print(Markdown(result.response))

        if result.actions_taken:
            console.print("\n[bold yellow]Actions Taken:[/bold yellow]")
            for action in result.actions_taken:
                console.print(f"  • {action}")

        if result.recommendations:
            console.print("\n[bold cyan]Recommendations:[/bold cyan]")
            for rec in result.recommendations:
                console.print(f"  • {rec}")

        if result.requires_user_action:
            console.print("\n[bold red]⚠️  User approval required for suggested fixes[/bold red]")

    except Exception as e:
        console.print(f"[bold red]Error:[/bold red] {str(e)}")


async def _interactive_chat():
    """Start interactive chat mode."""
    console.print(Panel("[bold]MCP Agent Interactive Mode[/bold]\nType 'exit' to quit"))

    # Initialize agent
    client = MCPProxyClient()
    diagnostic_tools = DiagnosticTools(client)
    config_tools = ConfigTools()

    tools_registry = {
        "diagnostic": diagnostic_tools,
        "config": config_tools,
    }

    agent = MCPAgentGraph(tools_registry)

    while True:
        try:
            user_input = console.input("\n[bold blue]You:[/bold blue] ")

            if user_input.lower() in ["exit", "quit", "q"]:
                console.print("[yellow]Goodbye![/yellow]")
                break

            with console.status("[bold green]Thinking..."):
                result = await agent.run(AgentInput(request=user_input))

            console.print(f"\n[bold green]Agent:[/bold green]\n{result.response}")

        except KeyboardInterrupt:
            console.print("\n[yellow]Goodbye![/yellow]")
            break
        except Exception as e:
            console.print(f"[bold red]Error:[/bold red] {str(e)}")


async def _single_chat(message: str):
    """Process a single chat message."""
    client = MCPProxyClient()
    diagnostic_tools = DiagnosticTools(client)
    config_tools = ConfigTools()

    tools_registry = {
        "diagnostic": diagnostic_tools,
        "config": config_tools,
    }

    agent = MCPAgentGraph(tools_registry)

    with console.status("[bold green]Processing..."):
        result = await agent.run(AgentInput(request=message))

    console.print(Markdown(result.response))


async def _show_status(server_name: Optional[str]):
    """Show server status."""
    client = MCPProxyClient()

    try:
        if server_name:
            # Show specific server status
            status = await client.get_server_status(server_name)

            table = Table(title=f"Server Status: {server_name}")
            table.add_column("Property", style="cyan")
            table.add_column("Value", style="green")

            for key, value in status.items():
                table.add_row(key, str(value))

            console.print(table)
        else:
            # Show all servers (would need new API endpoint)
            console.print("[yellow]Showing all servers status...[/yellow]")
            console.print("[dim]API endpoint not yet implemented[/dim]")

    except Exception as e:
        console.print(f"[bold red]Error:[/bold red] {str(e)}")


if __name__ == "__main__":
    app()
