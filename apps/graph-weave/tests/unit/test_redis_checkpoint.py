import pytest
from unittest.mock import MagicMock
from src.adapters.checkpoint import RedisCheckpointStore
from src.adapters.langgraph_executor import RealLangGraphExecutor

def test_redis_checkpoint_store_save_load():
    mock_redis = MagicMock()
    store = RedisCheckpointStore(mock_redis)
    
    checkpoint_data = {"workflow_state": {"foo": "bar"}}
    store.save("run-1", "thread-1", checkpoint_data)
    
    mock_redis.set.assert_called_once_with("checkpoint:run-1:thread-1", checkpoint_data)
    
    mock_redis.get.return_value = checkpoint_data
    loaded = store.load("run-1", "thread-1")
    
    assert loaded == checkpoint_data
    mock_redis.get.assert_called_once_with("checkpoint:run-1:thread-1")

def test_real_executor_calls_checkpoint_service():
    mock_checkpoint_service = MagicMock()
    mock_mcp_router = MagicMock()
    
    executor = RealLangGraphExecutor(
        mcp_router=mock_mcp_router,
        checkpoint_service=mock_checkpoint_service
    )
    
    workflow = {
        "workflow_id": "test-wf",
        "nodes": [
            {"id": "start", "type": "entry"},
            {"id": "end", "type": "exit"}
        ],
        "edges": [
            {"from": "start", "to": "end"}
        ]
    }
    
    input_data = {"initial": "data"}
    
    executor.execute(
        run_id="run-1",
        thread_id="thread-1",
        tenant_id="tenant-1",
        workflow=workflow,
        input_data=input_data
    )
    
    # Checkpoint should be saved after 'start' node (entry node)
    mock_checkpoint_service.save_checkpoint.assert_called_once_with(
        tenant_id="tenant-1",
        thread_id="thread-1",
        workflow_state={"initial": "data"}
    )
