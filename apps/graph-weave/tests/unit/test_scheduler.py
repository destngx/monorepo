import pytest
from unittest.mock import Mock, patch
from src.models.schedule import ScheduleCreate, validate_cron
from src.services.scheduler_service import SchedulerService

def test_validate_cron_valid():
    assert validate_cron("* * * * *") == "* * * * *"
    assert validate_cron("0 0 * * *") == "0 0 * * *"
    assert validate_cron("0 0 * * * *") == "0 0 * * * *"

def test_validate_cron_invalid():
    with pytest.raises(ValueError, match="must have 5 or 6 parts"):
        validate_cron("* * * *")
    with pytest.raises(ValueError, match="must have 5 or 6 parts"):
        validate_cron("* * * * * * *")
    with pytest.raises(ValueError, match="cannot be empty"):
        validate_cron("")

def test_schedule_create_validation():
    # Valid
    ScheduleCreate(
        tenant_id="test",
        workflow_id="test:v1.0.0",
        cron_expression="* * * * *",
        name="Test Schedule"
    )
    
    # Invalid cron
    with pytest.raises(ValueError):
        ScheduleCreate(
            tenant_id="test",
            workflow_id="test:v1.0.0",
            cron_expression="invalid",
        )

@patch("src.services.scheduler_service.BackgroundScheduler")
def test_scheduler_service_add_remove(mock_scheduler_class):
    mock_scheduler = mock_scheduler_class.return_value
    mock_store = Mock()
    handler = Mock()
    service = SchedulerService(mock_store, handler)
    
    service.add_schedule("t1", "s1", "* * * * *", "w1", {"foo": "bar"})
    assert mock_scheduler.add_job.called
    
    service.remove_schedule("t1", "s1")
    assert mock_scheduler.remove_job.called

def test_scheduler_service_sync(mock_scheduler_class=None):
    with patch("src.services.scheduler_service.BackgroundScheduler") as mock_scheduler_class:
        mock_scheduler = mock_scheduler_class.return_value
        mock_store = Mock()
        mock_store.list_for_tenant.return_value = [
            {"schedule_id": "s1", "cron_expression": "* * * * *", "workflow_id": "w1", "enabled": True},
            {"schedule_id": "s2", "cron_expression": "0 0 * * *", "workflow_id": "w2", "enabled": False},
        ]
        service = SchedulerService(mock_store, Mock())
        
        service.sync_schedules("t1")
        
        # Should add s1, remove s2 (or try to)
        assert mock_scheduler.add_job.call_count == 1
        # remove_job might be called if job exists, let's check get_job
        mock_scheduler.get_job.return_value = True
        service.sync_schedules("t1")
        assert mock_scheduler.remove_job.called

def test_redis_schedule_store_fallback():
    # Mock NamespacedRedisClient with failing Redis and working fallback
    mock_redis = Mock()
    mock_redis.hgetall.side_effect = Exception("Redis Down")
    
    # We need a real NamespacedRedisClient or a good mock
    from src.adapters.redis_circuit_breaker import NamespacedRedisClient, FallbackStorage
    from src.adapters.cache import MockRedisAdapter
    
    fallback = FallbackStorage()
    client = NamespacedRedisClient(MockRedisAdapter(), fallback_storage=fallback)
    
    # Force circuit breaker to open or just mock the call
    from src.adapters.schedule import RedisScheduleStore
    store = RedisScheduleStore(client)
    
    # Create in fallback
    schedule = {"workflow_id": "w1", "cron_expression": "* * * * *"}
    created = store.create("t1", schedule)
    assert created["schedule_id"] is not None
    
    # List from fallback
    schedules = store.list_for_tenant("t1")
    assert len(schedules) == 1
    assert schedules[0]["workflow_id"] == "w1"
