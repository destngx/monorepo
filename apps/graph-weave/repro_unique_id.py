import sys
import os

# Add src to path
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "src")))

from src.adapters.workflow import MockWorkflowStore

def test_unique_workflow_id():
    store = MockWorkflowStore()
    
    workflow_data = {
        "workflow_id": "test-workflow:v1.0.0",
        "name": "Test Workflow",
        "version": "1.0.0",
        "description": "Test",
        "owner": "user1",
        "definition": {"nodes": [], "edges": [], "entry_point": "a", "exit_point": "b"}
    }
    
    # Create for tenant1
    res1 = store.create("tenant1", workflow_data)
    print(f"Tenant1 creation: {'Success' if res1 else 'Failed'}")
    
    # Create for tenant1 again (should fail)
    res2 = store.create("tenant1", workflow_data)
    print(f"Tenant1 creation again: {'Success' if res2 else 'Failed'}")
    
    # Create for tenant2 (should now fail)
    res3 = store.create("tenant2", workflow_data)
    print(f"Tenant2 creation: {'Success' if res3 else 'Failed'}")

if __name__ == "__main__":
    test_unique_workflow_id()
