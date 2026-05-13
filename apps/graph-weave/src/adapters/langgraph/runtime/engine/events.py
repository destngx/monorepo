import json
import logging
from datetime import datetime, timezone
from typing import Any, Dict

logger = logging.getLogger(__name__)

def emit_event(executor: Any, run_id: str, event_type: str, data: Dict[str, Any]) -> None:
    if run_id not in executor.execution_events:
        executor.execution_events[run_id] = []
        
    event = {
        "type": event_type,
        "run_id": run_id,
        "timestamp": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "data": data
    }
    
    executor.execution_events[run_id].append(event)
    
    if executor.redis_client:
        try:
            executor.redis_client.rpush(f"event:{run_id}", json.dumps(event))
        except Exception as e:
            logger.warning(f"Failed to persist event to Redis: {e}")
