class PreCommitValidator:
    async def validate(self, workflow_json: dict) -> ValidationResult:
        try:
            validated = WorkflowDefinition(**workflow_json)
        except ValidationError as e:
            return ValidationResult(valid=False, errors=e.errors())

        for skill_id in validated.skills.available_tier2:
            if not await self.mcp_registry.exists(skill_id):
                return ValidationResult(
                    valid=False, errors=[f"MCP skill {skill_id} not found"]
                )

        graph = nx.DiGraph()
        for rule in validated.routing_rules:
            graph.add_edge(rule.source, rule.target)

        if not nx.is_directed_acyclic_graph(graph):
            return ValidationResult(
                valid=False,
                errors=[f"Cycles detected: {list(nx.simple_cycles(graph))}"],
            )

        reachable = nx.descendants(graph, "START")
        unreachable = set(graph.nodes) - reachable - {"END"}
        if unreachable:
            return ValidationResult(
                valid=False, errors=[f"Unreachable nodes: {unreachable}"]
            )

        if not re.match(r"^\d+\.\d+\.\d+$", validated.version):
            return ValidationResult(
                valid=False, errors=["Version must be SemVer format"]
            )

        if validated.limits.max_steps > 50:
            return ValidationResult(valid=False, errors=["max_steps cannot exceed 50"])

        return ValidationResult(valid=True, data=validated)


import networkx as nx
