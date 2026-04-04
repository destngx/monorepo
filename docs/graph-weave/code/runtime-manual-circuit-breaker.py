class CircuitBreaker:
    """Redis-backed kill switch for emergency stops"""

    async def activate_tenant_kill(self, tenant_id: str, ttl_seconds: int = 300):
        """Immediately stop ALL workflows for a tenant"""
        key = f"graphweave:circuit_breaker:tenant:{tenant_id}:kill"
        await redis.setex(key, ttl_seconds, "1")
        await self._broadcast_kill(tenant_id)

    async def activate_workflow_kill(self, tenant_id: str, workflow_id: str, ttl_seconds: int = 300):
        """Stop all instances of a specific workflow"""
        key = f"graphweave:circuit_breaker:workflow:{tenant_id}:{workflow_id}:kill"
        await redis.setex(key, ttl_seconds, "1")

        # Force interrupt all active threads for this workflow
        active_threads = await redis.smembers(f"graphweave:active_threads:{tenant_id}:{workflow_id}")
        for thread_id in active_threads:
            await self._interrupt_thread(thread_id)

    async def _interrupt_thread(self, thread_id: str):
        """Use LangGraph's interrupt() to stop execution"""
        # Get the graph instance for this thread
        graph = await self._get_graph_for_thread(thread_id)

        # Interrupt at next circuit_breaker node
        await graph.interrupt(
            thread_id=thread_id,
            reason="Kill switch activated",
            resume=False
        )
