from typing import Dict, Any, Optional, List


class MockCheckpointStore:
    def __init__(self):
        self._checkpoints: Dict[str, Dict[str, Any]] = {}

    def save(self, run_id: str, thread_id: str, checkpoint_data: Any) -> None:
        key = f"{run_id}:{thread_id}"
        self._checkpoints[key] = checkpoint_data

    def load(self, run_id: str, thread_id: str) -> Optional[Any]:
        key = f"{run_id}:{thread_id}"
        return self._checkpoints.get(key)

    def delete(self, run_id: str, thread_id: str) -> None:
        key = f"{run_id}:{thread_id}"
        if key in self._checkpoints:
            del self._checkpoints[key]

    def list_for_run(self, run_id: str) -> List[Any]:
        prefix = f"{run_id}:"
        return [v for k, v in self._checkpoints.items() if k.startswith(prefix)]

    def clear(self) -> None:
        self._checkpoints.clear()
