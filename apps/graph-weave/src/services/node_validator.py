from ..models.node import NodeCreate, NodeResponse


class NodeValidator:
    def validate_standalone(self, node: NodeCreate) -> list[str]:
        errors = []

        if not node.config.system_prompt and not node.config.command:
            errors.append(
                "Node must have system_prompt (agent_node) or command (cli_node)"
            )

        if not node.output_contract.produced:
            errors.append("Node must declare produced outputs")

        if not node.input_contract.required and not node.input_contract.optional:
            errors.append("Node must declare at least one input (required or optional)")

        for field in node.output_contract.produced:
            if not field.state_path:
                errors.append(f"Output field '{field.name}' must have state_path")

        for field in node.input_contract.required + node.input_contract.optional:
            if not field.type:
                errors.append(f"Input field '{field.name}' must have type")

        return errors

    def validate_contract_compatibility(
        self, upstream: NodeResponse, downstream: NodeResponse
    ) -> list[str]:
        errors = []

        upstream_produced = {
            f.name: f.type for f in upstream.output_contract.produced
        }

        for required_field in downstream.input_contract.required:
            if required_field.name not in upstream_produced:
                errors.append(
                    f"Missing required input '{required_field.name}' "
                    f"(upstream {upstream.node_id} does not produce it)"
                )
            elif upstream_produced[required_field.name] != required_field.type:
                errors.append(
                    f"Type mismatch for '{required_field.name}': "
                    f"upstream produces {upstream_produced[required_field.name]}, "
                    f"downstream expects {required_field.type}"
                )

        return errors

    def validate_compositional_workflow(self, workflow: dict) -> list[str]:
        errors = []

        nodes = workflow.get("nodes", [])
        edges = workflow.get("edges", [])

        aliases = set()
        for node in nodes:
            if "alias" in node:
                aliases.add(node["alias"])
            elif node.get("id") in ("entry", "exit"):
                pass
            else:
                errors.append(f"Node missing 'alias' field: {node}")

        for edge in edges:
            from_id = edge.get("from")
            to_id = edge.get("to")

            if from_id not in ("entry",) and from_id not in aliases:
                errors.append(f"Edge 'from' references unknown alias: {from_id}")

            if to_id not in ("exit",) and to_id not in aliases:
                errors.append(f"Edge 'to' references unknown alias: {to_id}")

        return errors
