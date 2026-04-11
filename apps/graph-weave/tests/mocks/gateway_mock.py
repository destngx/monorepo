"""
MOCK GATEWAY for unit tests.
Implements the modern AIGatewayClient interface.
"""
from typing import Any, Dict, List, Optional, Protocol
import json

class MockGatewayClient:
    """Mock implementation of AIGatewayClient for testing."""
    def __init__(self, base_url: str = "http://localhost:8080/v1"):
        self.base_url = base_url
        self.last_payload = {}
        self.responses = {}

    def chat_completion(
        self,
        messages: List[Dict[str, str]],
        provider: str,
        model: str,
        tools: Optional[List[Dict[str, Any]]] = None,
        temperature: float = 0.7,
        max_tokens: int = 2000,
        stream: bool = False
    ) -> Dict[str, Any]:
        self.last_payload = {
            "messages": messages,
            "provider": provider,
            "model": model,
            "tools": tools,
        }
        
        # Determine content from responses or default
        user_msg = messages[-1]["content"] if messages else ""
        content = self.responses.get(user_msg)
        
        if not content:
            # Check if this looks like a tool call test
            if tools and "search" in user_msg.lower():
                return {
                    "choices": [{
                        "message": {
                            "role": "assistant",
                            "tool_calls": [{
                                "id": "call_123",
                                "type": "function",
                                "function": {
                                    "name": "search",
                                    "arguments": json.dumps({"query": user_msg})
                                }
                            }]
                        }
                    }],
                    "model": model
                }
            
            content = json.dumps({"status": "completed", "message": "Mock Gateway Response"})
            
        return {
            "id": "mock_id",
            "model": model,
            "choices": [{
                "message": {
                    "role": "assistant",
                    "content": content
                }
            }],
            "usage": {"total_tokens": 10}
        }
