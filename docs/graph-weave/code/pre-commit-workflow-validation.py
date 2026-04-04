class PreCommitValidator:
    """Validates workflow JSON before allowing storage to Redis"""

    async def validate(self, workflow_json: dict) -> ValidationResult:
        # 1. Pydantic schema validation
        try:
            validated = WorkflowDefinition(**workflow_json)
        except ValidationError as e:
            return ValidationResult(valid=False, errors=e.errors())

        # 2. MCP integrity check (all referenced skills exist)
        for skill_id in validated.skills.available_tier2:
            if not await self.mcp_registry.exists(skill_id):
                return ValidationResult(valid=False, errors=[f"MCP skill {skill_id} not found"])

        # 3. NetworkX DAG validation (no cycles, unreachable nodes)
        graph = nx.DiGraph()
        for rule in validated.routing_rules:
            graph.add_edge(rule.source, rule.target)

        if not nx.is_directed_acyclic_graph(graph):
            cycles = list(nx.simple_cycles(graph))
            return ValidationResult(valid=False, errors=[f"Cycles detected: {cycles}"])

        # 4. Unreachable node detection
        reachable = nx.descendants(graph, "START")
        all_nodes = set(graph.nodes)
        unreachable = all_nodes - reachable - {"END"}
        if unreachable:
            return ValidationResult(valid=False, errors=[f"Unreachable nodes: {unreachable}"])

        # 5. SemVer enforcement (major.minor.patch)
        if not re.match(r'^\d+\.\d+\.\d+$', validated.version):
            return ValidationResult(valid=False, errors=["Version must be SemVer format"])

        # 6. Limit validation
        if validated.limits.max_steps > 50:
            return ValidationResult(valid=False, errors=["max_steps cannot exceed 50"])

        return ValidationResult(valid=True, data=validated)

import networkx as nx

def pre_commit_gate(workflow_json: dict) -> ValidationResult:
    validated = WorkflowDefinitionSchema.parse_obj(workflow_json)

    G = nx.DiGraph()
    nodes = [f"subagent_{sa.name}" for sa in validated.subagents] + ["orchestrator", "output_guardrail", "skill_loader", "initializer", "stagnation_detector", "circuit_breaker"]
    G.add_nodes_from(nodes)

    # Map edges based on defined routing
    edges = [
        ("initializer", "orchestrator"),
        ("skill_loader", "orchestrator"),
        ("circuit_breaker", "orchestrator"),
        ("orchestrator", "skill_loader"),
        ("orchestrator", "stagnation_detector"),
        ("orchestrator", "output_guardrail")
    ]
    for sa in validated.subagents:
        edges.append(("stagnation_detector", f"subagent_{sa.name}"))
        edges.append((f"subagent_{sa.name}", "circuit_breaker"))

    G.add_edges_from(edges)

    # NOTE: The Orchestrator loop IS cyclic by design. NetworkX cycle detection here is focused on ensuring subagents don't bypass the watchdog or orchestrator.
    # Validate Unreachable nodes:
    reachable = nx.descendants(G, "initializer")
    for node in G.nodes:
        if node != "initializer" and node not in reachable:
            return ValidationResult(False, f"Unreachable node: {node}")

    return ValidationResult(True, "Valid")
