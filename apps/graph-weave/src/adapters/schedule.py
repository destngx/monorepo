from typing import Dict, Any, Optional, List
from datetime import datetime
import uuid
import json
import logging
from ..adapters.redis_circuit_breaker import NamespacedRedisClient

logger = logging.getLogger(__name__)

class RedisScheduleStore:
    def __init__(self, redis_client: NamespacedRedisClient):
        self.redis_client = redis_client

    def create(self, tenant_id: str, schedule: Dict[str, Any]) -> Dict[str, Any]:
        schedule_id = str(uuid.uuid4())
        timestamp = datetime.utcnow().isoformat() + "Z"
        
        schedule_data = {
            **schedule,
            "schedule_id": schedule_id,
            "tenant_id": tenant_id,
            "created_at": timestamp,
            "updated_at": timestamp,
        }
        
        key = self.redis_client.schedule_key(tenant_id)
        self.redis_client.hset(key, schedule_id, schedule_data)
        return schedule_data

    def get(self, tenant_id: str, schedule_id: str) -> Optional[Dict[str, Any]]:
        key = self.redis_client.schedule_key(tenant_id)
        return self.redis_client.hget(key, schedule_id)

    def list_for_tenant(self, tenant_id: str) -> List[Dict[str, Any]]:
        key = self.redis_client.schedule_key(tenant_id)
        schedules_dict = self.redis_client.hgetall(key)
        return list(schedules_dict.values())

    def update(self, tenant_id: str, schedule_id: str, updates: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        schedule = self.get(tenant_id, schedule_id)
        if not schedule:
            return None
        
        timestamp = datetime.utcnow().isoformat() + "Z"
        schedule.update(updates)
        schedule["updated_at"] = timestamp
        
        key = self.redis_client.schedule_key(tenant_id)
        self.redis_client.hset(key, schedule_id, schedule)
        return schedule

    def delete(self, tenant_id: str, schedule_id: str) -> bool:
        key = self.redis_client.schedule_key(tenant_id)
        return bool(self.redis_client.hdel(key, schedule_id))
