"""
Pattern 1: Intent → Structured Workflow + LangGraph Integration

This example shows how to convert a WorkflowSpec to a LangGraph StateGraph
and execute it with full state management.

Note: This is a demonstration of the pattern. To run with actual LangGraph,
install: pip install langgraph
"""

import sys
from pathlib import Path

# Add src to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from typing import TypedDict
from workflow_generation.schemas.intent import (
    IntentExtraction,
    IntentAction,
)
from workflow_generation.generators.intent_to_workflow import IntentToWorkflowGenerator


# Define state schema
class WorkflowState(TypedDict):
    """State for workflow execution."""
    workflow_id: str
    current_node: str
    results: dict
    errors: list


def create_research_intent() -> IntentExtraction:
    """Create a research workflow intent."""
    return IntentExtraction(
        goal="Research and summarize a topic",
        description="Search for information, fetch content, and create a summary",
        actions=[
            IntentAction(
                id="search",
                name="Search",
                description="Search the web for information",
                operator="web_search",
            ),
            IntentAction(
                id="fetch",
                name="Fetch",
                description="Fetch content from URLs",
                operator="fetch_url",
                dependencies=["search"],
            ),
            IntentAction(
                id="summarize",
                name="Summarize",
                description="Create a summary of the content",
                operator="llm_call",
                dependencies=["fetch"],
            ),
        ],
    )


def main():
    """Main example: Generate workflow from intent and show execution order."""
    print("=" * 70)
    print("Pattern 1: Intent → Structured Workflow + LangGraph Integration")
    print("=" * 70)
    print()

    # Create intent
    intent = create_research_intent()
    print("1. Created Intent:")
    print(f"   Goal: {intent.goal}")
    print(f"   Actions: {len(intent.actions)}")
    for action in intent.actions:
        deps = f" (depends on: {', '.join(action.dependencies)})" if action.dependencies else ""
        print(f"     - {action.name}: {action.operator}{deps}")
    print()

    # Generate workflow
    print("2. Generating Workflow from Intent...")
    generator = IntentToWorkflowGenerator()
    result = generator.generate(intent)

    if not result["success"]:
        print(f"   ✗ Error: {result['errors']}")
        return

    workflow = result["workflow"]
    print(f"   ✓ Workflow generated: {workflow.id}")
    print(f"   Nodes: {len(workflow.nodes)}")
    print(f"   Edges: {len(workflow.edges)}")
    print()

    # Show topological sort (execution order)
    print("3. Execution Order (Topological Sort):")
    execution_order = workflow.topological_sort()
    for i, node_id in enumerate(execution_order, 1):
        node = next((n for n in workflow.nodes if n.id == node_id), None)
        if node:
            operator_info = f" ({node.operator})" if node.operator else ""
            print(f"   {i}. {node.id}{operator_info}")
    print()

    # Show workflow structure
    print("4. Workflow Structure:")
    print(f"   Start Node: {workflow.start_node_id}")
    print(f"   End Nodes: {', '.join(workflow.end_node_ids)}")
    print()

    # Show edges
    print("5. Dependencies (Edges):")
    for edge in workflow.edges:
        print(f"   {edge.source} → {edge.target}")
    print()

    # Simulate LangGraph StateGraph construction
    print("6. LangGraph StateGraph Construction (Simulated):")
    print("   from langgraph.graph import StateGraph, END")
    print()
    print("   graph = StateGraph(WorkflowState)")
    for node in workflow.nodes:
        print(f"   graph.add_node('{node.id}', execute_{node.id})")
    print()
    for edge in workflow.edges:
        print(f"   graph.add_edge('{edge.source}', '{edge.target}')")
    print()
    print("   compiled = graph.compile()")
    print("   result = compiled.invoke(initial_state)")
    print()

    print("=" * 70)
    print("✓ Pattern 1 example completed successfully!")
    print("=" * 70)


if __name__ == "__main__":
    main()
