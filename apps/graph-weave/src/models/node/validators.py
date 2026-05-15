import re

NODE_ID_PATTERN = re.compile(r"^[a-z][a-z0-9_]*:v[0-9]+\.[0-9]+\.[0-9]+$")


class NodeIdFormatError(ValueError):
    pass


class NodeImmutableFieldError(ValueError):
    pass


def validate_node_id(node_id: str) -> None:
    if not NODE_ID_PATTERN.match(node_id):
        raise NodeIdFormatError(
            f"Invalid node_id format: {node_id}. Expected: name:v1.0.0"
        )


def parse_node_id(node_id: str) -> tuple[str, str]:
    validate_node_id(node_id)
    name, version = node_id.split(":v", 1)
    return name, f"v{version}"


def build_node_id(node_name: str, version: str) -> str:
    if not version.startswith("v"):
        version = f"v{version}"
    return f"{node_name}:{version}"
