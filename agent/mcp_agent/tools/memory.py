"""Tools for interacting with the Diagnostic Memory."""

from typing import Dict, Any, Optional
from pydantic import BaseModel
from mcp_agent.tools.diagnostic import MCPProxyClient

class MemoryContent(BaseModel):
    """Content of the diagnostic memory."""
    content: str
    success: bool
    message: Optional[str] = None

class MemoryTools:
    """Tools for reading and writing diagnostic memory."""

    def __init__(self, client: Optional[MCPProxyClient] = None):
        self.client = client or MCPProxyClient()

    async def read_memory(self) -> MemoryContent:
        """
        Read the content of the diagnostic memory.
        
        Returns:
            MemoryContent object containing the markdown content.
        """
        response = await self.client.client.get("/api/memory")
        response.raise_for_status()
        data = response.json()
        return MemoryContent(**data)

    async def update_memory(self, content: str) -> MemoryContent:
        """
        Update the content of the diagnostic memory.
        
        Args:
            content: The new markdown content for the memory file.
            
        Returns:
            MemoryContent object with success status.
        """
        response = await self.client.client.post(
            "/api/memory", 
            json={"content": content}
        )
        response.raise_for_status()
        data = response.json()
        return MemoryContent(
            content=content if data.get("success") else "",
            success=data.get("success", False),
            message=data.get("message")
        )

    async def append_memory(self, entry: str) -> MemoryContent:
        """
        Append a new entry to the diagnostic memory.
        
        Args:
            entry: The new memory entry to append.
            
        Returns:
            Updated MemoryContent.
        """
        current_memory = await self.read_memory()
        new_content = current_memory.content + "\n\n" + entry
        return await self.update_memory(new_content)
