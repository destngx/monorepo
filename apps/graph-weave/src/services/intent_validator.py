"""
Intent Validator

Stage 1 of Compiled AI 4-stage validation:
1. Schema validation (Pydantic)
2. Semantic validation (intent coherence)
3. Operator availability (Redis Node Store capabilities)
4. Constraint satisfaction (resource limits)
"""

from typing import Dict, List, Set, Optional, Any
from ..models import IntentExtraction, IntentAction


class IntentValidator:
    """Validates IntentExtraction for semantic coherence and operator availability."""
    
    def __init__(
        self, 
        available_operators: Optional[Set[str]] = None,
        node_store: Optional[Any] = None
    ):
        """
        Initialize validator.
        
        Args:
            available_operators: Set of available operator names.
                                If None, dynamic catalog nodes are queried.
            node_store: RedisNodeStore instance to dynamically query active catalog nodes.
        """
        self.available_operators = available_operators or set()
        self.node_store = node_store
    
    def validate(self, intent: IntentExtraction) -> Dict[str, Any]:
        """
        Validate intent extraction synchronously (static checks only).
        
        Returns:
            {
                "valid": bool,
                "errors": List[str],
                "warnings": List[str],
                "metadata": Dict
            }
        """
        errors = []
        warnings = []
        
        # Stage 1: Schema validation (already done by Pydantic)
        
        # Stage 2: Semantic validation
        errors.extend(self._validate_semantic_coherence(intent))
        
        # Stage 3: Operator availability (static check only)
        if self.available_operators:
            for action in intent.actions:
                if action.operator not in self.available_operators:
                    warnings.append(
                        f"Action '{action.id}' uses operator '{action.operator}' "
                        f"which is not in the available operators registry"
                    )
        
        # Stage 4: Constraint satisfaction
        errors.extend(self._validate_constraints(intent))
        
        return {
            "valid": len(errors) == 0,
            "errors": errors,
            "warnings": warnings,
            "metadata": {
                "action_count": len(intent.actions),
                "has_dependencies": any(a.dependencies for a in intent.actions),
                "operators_used": {a.operator for a in intent.actions},
            }
        }
    
    async def validate_async(self, intent: IntentExtraction) -> Dict[str, Any]:
        """
        Validate intent extraction asynchronously, querying the Redis Node Store.
        
        Returns:
            {
                "valid": bool,
                "errors": List[str],
                "warnings": List[str],
                "metadata": Dict
            }
        """
        errors = []
        warnings = []
        
        # Stage 1: Schema validation (already done by Pydantic)
        
        # Stage 2: Semantic validation
        errors.extend(self._validate_semantic_coherence(intent))
        
        # Stage 3: Dynamic Operator availability
        for action in intent.actions:
            # 1. Check static overrides
            if self.available_operators and action.operator in self.available_operators:
                continue
            
            # 2. Check Redis Node Store dynamically
            if self.node_store:
                try:
                    # Check by capability
                    cap_nodes = await self.node_store.find_by_capability(action.operator)
                    if cap_nodes:
                        continue
                    # Check by name
                    name_nodes = await self.node_store.find_by_name(action.operator)
                    if name_nodes:
                        continue
                except Exception as e:
                    warnings.append(
                        f"Error querying node store for action '{action.id}' "
                        f"operator '{action.operator}': {str(e)}"
                    )
                    continue
            
            # If not found statically or dynamically (and we actually checked)
            if self.available_operators or self.node_store:
                warnings.append(
                    f"Action '{action.id}' uses operator/capability '{action.operator}' "
                    f"which is not dynamically available in the Node Store"
                )
        
        # Stage 4: Constraint satisfaction
        errors.extend(self._validate_constraints(intent))
        
        return {
            "valid": len(errors) == 0,
            "errors": errors,
            "warnings": warnings,
            "metadata": {
                "action_count": len(intent.actions),
                "has_dependencies": any(a.dependencies for a in intent.actions),
                "operators_used": {a.operator for a in intent.actions},
            }
        }
    
    def _validate_semantic_coherence(self, intent: IntentExtraction) -> List[str]:
        """Validate that actions form a coherent workflow."""
        errors = []
        
        # Check: At least one action
        if not intent.actions:
            errors.append("Intent must have at least one action")
        
        # Check: Goal and description are not empty
        if not intent.goal or not intent.goal.strip():
            errors.append("Goal cannot be empty")
        
        if not intent.description or not intent.description.strip():
            errors.append("Description cannot be empty")
        
        # Check: Action names are unique
        action_ids = [a.id for a in intent.actions]
        if len(action_ids) != len(set(action_ids)):
            errors.append("Action IDs must be unique")
        
        return errors
    
    def _validate_constraints(self, intent: IntentExtraction) -> List[str]:
        """Validate that constraints are satisfiable."""
        errors = []
        
        if not intent.constraints:
            return errors
        
        # Check: max_cost is positive
        if "max_cost" in intent.constraints:
            if intent.constraints["max_cost"] <= 0:
                errors.append("max_cost must be positive")
        
        # Check: max_duration is positive
        if "max_duration" in intent.constraints:
            if intent.constraints["max_duration"] <= 0:
                errors.append("max_duration must be positive")
        
        # Check: required_tools are non-empty
        if "required_tools" in intent.constraints:
            if not intent.constraints["required_tools"]:
                errors.append("required_tools cannot be empty")
        
        return errors
