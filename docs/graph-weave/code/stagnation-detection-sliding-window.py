class StagnationDetector:
    """Detects loops and repetitive routing decisions"""

    def __init__(self, window_size: int = 3, similarity_threshold: float = 0.95):
        self.window_size = window_size
        self.similarity_threshold = similarity_threshold

    async def detect(self, state: GraphWeaveState) -> StagnationResult:
        history = state.get("stagnation_history", [])
        if len(history) < self.window_size:
            return StagnationResult(is_stagnated=False)

        # Get last N routing directives
        recent = [json.loads(h) for h in history[-self.window_size:]]

        # Check 1: Exact match (all routing directives identical)
        if all(r["routing_directive"] == recent[0]["routing_directive"] for r in recent):
            return StagnationResult(
                is_stagnated=True,
                reason=f"Same routing directive {recent[0]['routing_directive']} repeated {self.window_size} times"
            )

        # Check 2: Intent repetition (same subagent_payload objective)
        objectives = [r.get("subagent_payload", {}).get("objective", "") for r in recent]
        if all(obj == objectives[0] for obj in objectives):
            return StagnationResult(
                is_stagnated=True,
                reason=f"Same objective '{objectives[0]}' repeated"
            )

        # Check 3: Semantic similarity (using embeddings)
        if self.window_size >= 3:
            embeddings = await self._get_embeddings([json.dumps(r) for r in recent])
            similarity_matrix = cosine_similarity(embeddings)
            if (similarity_matrix > self.similarity_threshold).all():
                return StagnationResult(
                    is_stagnated=True,
                    reason="Semantically similar orchestrator outputs detected"
                )

        return StagnationResult(is_stagnated=False)
