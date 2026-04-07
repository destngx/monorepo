class StagnationDetector:
    def __init__(self, window_size: int = 3, similarity_threshold: float = 0.95):
        self.window_size = window_size
        self.similarity_threshold = similarity_threshold

    async def detect(self, state: GraphWeaveState) -> StagnationResult:
        history = state.get("stagnation_history", [])
        if len(history) < self.window_size:
            return StagnationResult(is_stagnated=False)

        recent = [json.loads(item) for item in history[-self.window_size :]]

        if all(
            r["routing_directive"] == recent[0]["routing_directive"] for r in recent
        ):
            return StagnationResult(
                is_stagnated=True, reason="Repeated routing directive"
            )

        objectives = [r.get("agent_payload", {}).get("objective", "") for r in recent]
        if all(obj == objectives[0] for obj in objectives):
            return StagnationResult(is_stagnated=True, reason="Repeated objective")

        if self.window_size >= 3:
            embeddings = await self._get_embeddings([json.dumps(r) for r in recent])
            if (cosine_similarity(embeddings) > self.similarity_threshold).all():
                return StagnationResult(is_stagnated=True, reason="Semantic repetition")

        return StagnationResult(is_stagnated=False)
