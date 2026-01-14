"""Shell execution tools for the agent.

Provides tools to run system commands and programs.
"""

import asyncio
import shlex
from typing import Optional
from pydantic import BaseModel, Field

class CommandResult(BaseModel):
    """Result from a shell command execution."""
    command: str = Field(description="The executed command")
    exit_code: int = Field(description="Exit code of the process")
    stdout: str = Field(description="Standard output")
    stderr: str = Field(description="Standard error")
    success: bool = Field(description="Whether the command succeeded (exit code 0)")

class ShellTools:
    """Tools for executing system programs and commands."""

    def __init__(self):
        """Initialize shell tools."""
        pass

    async def execute_command(
        self,
        command: str,
        timeout: int = 300,
        working_dir: Optional[str] = None
    ) -> CommandResult:
        """Execute a shell command.
        
        Args:
            command: The command line string to execute
            timeout: Execution timeout in seconds (default: 300)
            working_dir: Working directory for execution (optional)
            
        Returns:
            CommandResult with output and status
        """
        try:
            # Create subprocess
            process = await asyncio.create_subprocess_shell(
                command,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=working_dir
            )
            
            # Wait for execution with timeout
            try:
                stdout_bytes, stderr_bytes = await asyncio.wait_for(
                    process.communicate(), 
                    timeout=timeout
                )
            except asyncio.TimeoutError:
                if process.returncode is None:
                    try:
                        process.kill()
                    except ProcessLookupError:
                        pass
                return CommandResult(
                    command=command,
                    exit_code=-1,
                    stdout="",
                    stderr=f"Command timed out after {timeout} seconds",
                    success=False
                )

            stdout = stdout_bytes.decode('utf-8', errors='replace')
            stderr = stderr_bytes.decode('utf-8', errors='replace')
            exit_code = process.returncode

            return CommandResult(
                command=command,
                exit_code=exit_code,
                stdout=stdout,
                stderr=stderr,
                success=(exit_code == 0)
            )

        except Exception as e:
            return CommandResult(
                command=command,
                exit_code=-1,
                stdout="",
                stderr=str(e),
                success=False
            )
