"""
Intent Validator

Stage 1 of Compiled AI 4-stage validation:
1. Schema validation (Pydantic)
2. Semantic validation (intent coherence)
3. Operator availability (tool registry)
4. Constraint satisfaction (resource limits)
"""

from typing import Dict, List, Set, Optional, Any
from ..schemas.intent import IntentExtraction, IntentAction


class IntentValidator:
    """Validates IntentExtraction for semantic coherence and operator availability."""
    
    def __init__(self, available_operators: Optional[Set[str]] = None):
        """
        Initialize validator.
        
        Args:
            available_operators: Set of available operator names.
                                If None, all operators are assumed available.
        """
        self.available_operators = available_operators or set()
    
    def validate(self, intent: IntentExtraction) -> Dict[str, Any]:
        """
        Validate intent extraction.
        
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
        
        # Stage 3: Operator availability
        warnings.extend(self._validate_operator_availability(intent))
        
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
        
        # Check: No orphaned actions (actions with no path to end)
        # This is a basic check; full reachability analysis happens in WorkflowValidator
        
        return errors
    
    def _validate_operator_availability(self, intent: IntentExtraction) -> List[str]:
        """Validate that all operators are available."""
        warnings = []
        
        if not self.available_operators:
            # No operator registry provided; skip this check
            return warnings
        
        for action in intent.actions:
            if action.operator not in self.available_operators:
                warnings.append(
                    f"Action '{action.id}' uses operator '{action.operator}' "
                    f"which is not in the available operators registry"
                )
        
        return warnings
    
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
