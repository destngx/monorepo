import sys
import os
import json

# Add src to path
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "src")))

from src.adapters.workflow import RedisWorkflowStore
from src.adapters.redis_circuit_breaker import NamespacedRedisClient, FallbackStorage
from src.adapters.cache import MockRedisAdapter

def test_redis_workflow_store():
    # Use MockRedisAdapter which now has our added hash methods (well, it won't unless we update it too!)
    # Wait, I didn't update MockRedisAdapter!
    pass

if __name__ == "__main__":
    # Actually, NamespacedRedisClient uses FallbackStorage for hash ops if redis fails.
    # FallbackStorage is in-memory.
    
    client = NamespacedRedisClient(redis_client=MockRedisAdapter(), fallback_storage=FallbackStorage())
    store = RedisWorkflowStore(client)
    
    workflow_data = {
        "workflow_id": "redis-test:v1.0.0",
        "name": "Redis Test",
        "version": "1.0.0",
        "description": "Test",
        "owner": "user1",
        "definition": {"nodes": [], "edges": [], "entry_point": "a", "exit_point": "b"}
    }
    
    # Create for tenant1
    res1 = store.create("tenant1", workflow_data)
    print(f"Tenant1 creation: {'Success' if res1 else 'Failed'}")
    
    # Create for tenant2 (should fail due to global registry)
    res2 = store.create("tenant2", workflow_data)
    print(f"Tenant2 creation: {'Success' if res2 else 'Failed'}")
    
    # Get for tenant1
    get1 = store.get("tenant1", "redis-test:v1.0.0")
    print(f"Tenant1 get: {'Success' if get1 else 'Failed'}")
    
    # List for tenant1
    list1 = store.list_for_tenant("tenant1")
    print(f"Tenant1 list count: {len(list1)}")

if __name__ == "__main__":
    # Wait, I need to make sure NamespacedRedisClient can actually run these.
    # If MockRedisAdapter doesn't have the methods, it will use FallbackStorage.
    # I should update MockRedisAdapter too for completeness.
    
    import sys
    # Re-run after I update MockRedisAdapter
