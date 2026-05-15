"""
Example Usage: Pattern 1 - Intent to Workflow Generation

This example demonstrates the complete workflow generation pipeline:
1. User provides natural language intent
2. LLM extracts structured IntentExtraction (single call)
3. Deterministic validators check coherence
4. Generator creates WorkflowSpec DAG
5. Workflow is ready for execution

Example: Research a topic and summarize findings
"""

from .schemas import IntentExtraction, IntentAction, IntentParameter
from .generators import IntentToWorkflowGenerator


def example_research_workflow():
    """
    Example: Generate a workflow to research a topic and summarize findings.
    
    This demonstrates:
    - Multi-step workflow with dependencies
    - Parameter passing between steps
    - Success criteria definition
    """
    
    # Step 1: Create IntentExtraction (this would come from LLM)
    intent = IntentExtraction(
        goal="Research and summarize a topic",
        description="Search for information about a topic, fetch detailed articles, and create a summary",
        
        actions=[
            IntentAction(
                id="search_topic",
                name="Search for Topic",
                description="Search the web for information about the topic",
                operator="web_search",
                parameters=[
                    IntentParameter(
                        name="query",
                        type="string",
                        description="Search query",
                        required=True
                    ),
                    IntentParameter(
                        name="max_results",
                        type="number",
                        description="Maximum number of results",
                        required=False,
                        default=5
                    )
                ],
                timeout_seconds=30
            ),
            
            IntentAction(
                id="fetch_articles",
                name="Fetch Articles",
                description="Fetch full content from top search results",
                operator="fetch_url",
                parameters=[
                    IntentParameter(
                        name="urls",
                        type="array",
                        description="URLs to fetch",
                        required=True
                    )
                ],
                dependencies=["search_topic"],
                timeout_seconds=60,
                retry_policy={"max_attempts": 3, "backoff_strategy": "exponential"}
            ),
            
            IntentAction(
                id="summarize",
                name="Summarize Findings",
                description="Create a comprehensive summary of the research",
                operator="llm_call",
                parameters=[
                    IntentParameter(
                        name="content",
                        type="string",
                        description="Content to summarize",
                        required=True
                    ),
                    IntentParameter(
                        name="max_length",
                        type="number",
                        description="Maximum summary length in words",
                        required=False,
                        default=500
                    )
                ],
                dependencies=["fetch_articles"],
                timeout_seconds=45
            )
        ],
        
        success_criteria=[
            "All articles fetched successfully",
            "Summary generated without errors",
            "Summary is between 100-500 words"
        ],
        
        constraints={
            "max_cost": 0.50,
            "max_duration": 300,  # 5 minutes
            "required_tools": ["web_search", "fetch_url", "llm_call"]
        },
        
        metadata={
            "user_id": "user_123",
            "session_id": "session_456",
            "source": "research_assistant"
        }
    )
    
    # Step 2: Generate workflow
    generator = IntentToWorkflowGenerator(
        available_operators={"web_search", "fetch_url", "llm_call"},
        workflow_id_prefix="research"
    )
    
    result = generator.generate(intent)
    
    # Step 3: Check results
    print("=" * 80)
    print("WORKFLOW GENERATION RESULT")
    print("=" * 80)
    
    print(f"\nIntent Validation: {'✓ PASS' if result['intent_validation']['valid'] else '✗ FAIL'}")
    if result['intent_validation']['errors']:
        print("  Errors:")
        for error in result['intent_validation']['errors']:
            print(f"    - {error}")
    if result['intent_validation']['warnings']:
        print("  Warnings:")
        for warning in result['intent_validation']['warnings']:
            print(f"    - {warning}")
    
    print(f"\nWorkflow Validation: {'✓ PASS' if result['workflow_validation']['valid'] else '✗ FAIL'}")
    if result['workflow_validation']['errors']:
        print("  Errors:")
        for error in result['workflow_validation']['errors']:
            print(f"    - {error}")
    if result['workflow_validation']['warnings']:
        print("  Warnings:")
        for warning in result['workflow_validation']['warnings']:
            print(f"    - {warning}")
    
    if result['success']:
        workflow = result['workflow']
        print(f"\n✓ Workflow Generated Successfully!")
        print(f"  ID: {workflow.id}")
        print(f"  Name: {workflow.name}")
        print(f"  Nodes: {len(workflow.nodes)}")
        print(f"  Edges: {len(workflow.edges)}")
        
        print(f"\n  Nodes:")
        for node in workflow.nodes:
            print(f"    - {node.id}: {node.name} ({node.type.value})")
        
        print(f"\n  Edges:")
        for edge in workflow.edges:
            print(f"    - {edge.source} → {edge.target} [{edge.condition}]")
        
        print(f"\n  Topological Order:")
        try:
            topo_order = workflow.topological_sort()
            for i, node_id in enumerate(topo_order, 1):
                print(f"    {i}. {node_id}")
        except ValueError as e:
            print(f"    Error: {e}")
        
        return workflow
    else:
        print(f"\n✗ Workflow Generation Failed!")
        print(f"  Errors: {result['errors']}")
        return None


def example_simple_workflow():
    """
    Example: Simple linear workflow with no dependencies.
    """
    
    intent = IntentExtraction(
        goal="Fetch and process data",
        description="Fetch data from an API and process it",
        
        actions=[
            IntentAction(
                id="fetch_data",
                name="Fetch Data",
                description="Fetch data from API",
                operator="fetch_url",
                parameters=[
                    IntentParameter(
                        name="url",
                        type="string",
                        description="API endpoint",
                        required=True
                    )
                ]
            ),
            
            IntentAction(
                id="process_data",
                name="Process Data",
                description="Process the fetched data",
                operator="python_script",
                parameters=[
                    IntentParameter(
                        name="script",
                        type="string",
                        description="Python script to execute",
                        required=True
                    )
                ],
                dependencies=["fetch_data"]
            )
        ]
    )
    
    generator = IntentToWorkflowGenerator()
    result = generator.generate(intent)
    
    if result['success']:
        print("\n✓ Simple workflow generated successfully!")
        return result['workflow']
    else:
        print(f"\n✗ Failed: {result['errors']}")
        return None


if __name__ == "__main__":
    print("\n" + "=" * 80)
    print("PATTERN 1: INTENT → STRUCTURED WORKFLOW")
    print("=" * 80)
    
    print("\n\n### EXAMPLE 1: Research Workflow ###")
    workflow1 = example_research_workflow()
    
    print("\n\n### EXAMPLE 2: Simple Workflow ###")
    workflow2 = example_simple_workflow()
