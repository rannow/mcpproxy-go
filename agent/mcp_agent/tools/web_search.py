"""Web search tools for the agent.

Provides tools to search the web using DuckDuckGo.
"""

from typing import List, Optional
from pydantic import BaseModel, Field
from duckduckgo_search import DDGS

class SearchResult(BaseModel):
    """Result from a web search."""
    title: str = Field(description="Title of the search result")
    href: str = Field(description="URL of the result")
    body: str = Field(description="Snippet or summary of the content")

class WebSearchTools:
    """Tools for searching the web."""

    def __init__(self):
        """Initialize web search tools."""
        pass

    async def search_web(
        self,
        query: str,
        max_results: int = 5
    ) -> List[SearchResult]:
        """Search the web for information.
        
        Args:
            query: The search query string
            max_results: Maximum number of results to return (default: 5)
            
        Returns:
            List of search results
        """
        results = []
        try:
            with DDGS() as ddgs:
                # Use text search
                search_results = list(ddgs.text(query, max_results=max_results))
                
                for r in search_results:
                    results.append(SearchResult(
                        title=r.get("title", ""),
                        href=r.get("href", ""),
                        body=r.get("body", "")
                    ))
                    
            return results
        except Exception as e:
            print(f"Web search failed: {e}")
            return []
