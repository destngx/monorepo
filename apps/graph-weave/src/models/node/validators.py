import re
from typing import Tuple, Optional

NODE_ID_PATTERN = re.compile(r"^[a-z][a-z0-9_]*:v[0-9]+\.[0-9]+\.[0-9]+$")


class NodeIdFormatError(Exception):
    pass


class NodeImmutableFieldError(Exception):
    pass


def validate_node_id(node_id: str) -> None:
    """Validate node_id format: {name}:v{major}.{minor}.{patch}"""
    if not NODE_ID_PATTERN.match(node_id):
        raise NodeIdFormatError(
            f"Invalid node_id format: {node_id}. Expected: name:v1.0.0"
        )


def parse_node_id(node_id: str) -> Tuple[str, str]:
    """Parse node_id into (node_name, version)"""
    validate_node_id(node_id)
    name, version = node_id.split(":v", 1)
    return name, f"v{version}"


def build_node_id(node_name: str, version: str) -> str:
    """Build node_id from name and version"""
    if not version.startswith("v"):
        version = f"v{version}"
    return f"{node_name}:{version}"


def generate_namespaced_name(
    base_name: str, domain: Optional[str] = None, node_type: Optional[str] = None
) -> str:
    """Generate a namespaced node name to avoid collisions.
    
    Examples:
        - generate_namespaced_name("extractor", "pkm", "agent") -> "pkm_extractor_agent"
        - generate_namespaced_name("extractor", "pkm") -> "pkm_extractor"
        - generate_namespaced_name("extractor") -> "extractor"
    """
    parts = []
    if domain:
        parts.append(domain.lower())
    parts.append(base_name.lower())
    if node_type:
        parts.append(node_type.lower())
    return "_".join(parts)


def validate_node_name(node_name: str) -> None:
    """Validate node name format (lowercase, underscores, no spaces)."""
    if not re.match(r"^[a-z][a-z0-9_]*$", node_name):
        raise ValueError(
            f"Invalid node name: {node_name}. "
            "Must be lowercase, start with letter, use underscores only."
        )
