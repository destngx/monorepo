"""
Stagnation Detection for GraphWeave Executor.

Prevents infinite loops by tracking node visits and enforcing max hop limits.

Implements:
- Per-node visit tracking with counter
- Stagnation detection (max_hops exceeded)
- Reset for new workflow execution
"""

from typing import Optional


class StagnationDetector:
    """Detects execution stagnation (infinite loops) during workflow execution."""

    def __init__(self, max_hops: int = 20):
        """
        Initialize stagnation detector.

        Args:
            max_hops: Maximum allowed node visits before declaring stagnation (default 20)
        """
        self.max_hops: int = max_hops
        self.visit_counts: dict[str, int] = {}
        self.total_visits: int = 0

    def track_node_visit(self, node_id: str) -> int:
        """
        Track a visit to a node.

        Args:
            node_id: ID of the node being visited

        Returns:
            Updated visit count for this node
        """
        if node_id not in self.visit_counts:
            self.visit_counts[node_id] = 0

        self.visit_counts[node_id] += 1
        self.total_visits += 1

        return self.visit_counts[node_id]

    def is_stagnated(self, max_hops: Optional[int] = None) -> bool:
        """
        Check if execution has stagnated (exceeded max hops).

        Args:
            max_hops: Optional override for max hop limit (uses instance default if not provided)

        Returns:
            True if stagnation detected (total visits >= max_hops), False otherwise
        """
        limit = max_hops if max_hops is not None else self.max_hops
        return self.total_visits >= limit

    def get_visit_count(self, node_id: str) -> int:
        """
        Get visit count for a specific node.

        Args:
            node_id: ID of the node

        Returns:
            Number of times this node has been visited
        """
        return self.visit_counts.get(node_id, 0)

    def reset(self) -> None:
        """
        Reset stagnation detector for new workflow execution.

        Clears all visit counts and resets total visits counter.
        """
        self.visit_counts.clear()
        self.total_visits = 0

    def get_summary(self) -> dict[str, int]:
        """
        Get summary of node visit counts.

        Returns:
            Dictionary mapping node_id to visit count
        """
        return dict(self.visit_counts)
