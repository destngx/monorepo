import re

def validate_workflow_id_format(v: str) -> str:
    """Validate workflow_id format: name:vX.Y.Z"""
    if not re.match(r"^[a-z][a-z0-9_-]*:v\d+\.\d+\.\d+$", v):
        raise ValueError(
            f"workflow_id must be in format 'name:vX.Y.Z' (e.g., workflow-generator:v1.0.0), got '{v}'"
        )
    return v

def validate_semantic_version(v: str) -> str:
    """Validate semantic versioning X.Y.Z"""
    if not re.match(r"^\d+\.\d+\.\d+$", v):
        raise ValueError(
            f"version must be semantic versioning X.Y.Z (e.g., 1.0.0), got '{v}'"
        )
    return v
