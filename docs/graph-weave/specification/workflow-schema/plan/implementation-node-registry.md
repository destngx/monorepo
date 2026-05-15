# Implementation Plan: Node Registry & Composable Workflows

**Date**: 2026-05-14
**Status**: Ready for Execution
**Reference**: `decomposed-generation-and-node-registry.md`
**Scope**: Breaking refactor — no legacy support, no feature flags

---

## Overview

This plan implements the node registry architecture. It is a **breaking refactor**: legacy embedded workflows are removed, all workflows become compositional.

### Breaking Changes

| Component      | Before                              | After                                 |
| -------------- | ----------------------------------- | ------------------------------------- |
| Workflow JSON  | Embedded nodes `{id, type, config}` | Compositional `{alias, node_id}`      |
| Node storage   | In workflow JSON                    | Independent in Redis                  |
| Parser         | Parses embedded nodes               | Parses compositional references only  |
| Builder        | Builds from embedded config         | Compiles node_ids → embedded → builds |
| API            | Workflow CRUD                       | Node CRUD + Workflow CRUD             |
| Persistence    | Embedded workflow in Redis          | Compositional source in Redis         |
| Legacy support | Yes                                 | **Removed**                           |

### File Impact

**New Files (12)**:

- `src/models/node/__init__.py`
- `src/models/node/create.py`
- `src/models/node/update.py`
- `src/models/node/response.py`
- `src/models/node/validators.py`
- `src/adapters/node.py`
- `src/routers/nodes.py`
- `src/services/node_compiler.py`
- `src/services/node_validator.py`
- `src/resources/workflows/workflow-generator:v2.0.0.json`
- `src/resources/nodes/*.json` (8 seed nodes)

**Modified Files (6)**:

- `src/adapters/redis/namespaced.py` — add node key helpers
- `src/adapters/langgraph/graph/parser.py` — compositional format only
- `src/adapters/langgraph/graph/builder.py` — use compiler
- `src/adapters/workflow.py` — store compositional source
- `src/modules/shared/deps.py` — wire node store
- `src/main.py` — register node router

**Removed Files (1)**:

- `src/resources/workflows/workflow-generator:v1.0.1.json` — replaced by v2.0.0

**New Test Files (6)**:

- `tests/unit/test_node_store.py`
- `tests/unit/test_node_compiler.py`
- `tests/unit/test_node_api.py`
- `tests/unit/test_node_validator.py`
- `tests/e2e/test_e2e_node_registry.py`
- `tests/e2e/test_e2e_generator_v2.py`

---

## Phase 1: Node Registry Infrastructure

**Goal**: Node CRUD API with Redis storage, typed contracts, immutability enforcement.

### Task GW-REG-001: Node Models

**Files**: `src/models/node/` (5 files)

**Create**: `src/models/node/__init__.py`

**Create**: `src/models/node/validators.py`

```python
import re

NODE_ID_PATTERN = re.compile(r"^[a-z][a-z0-9_]*:v[0-9]+\.[0-9]+\.[0-9]+$")

def validate_node_id(node_id: str) -> None:
    """Validate node_id format: {name}:v{major}.{minor}.{patch}"""
    if not NODE_ID_PATTERN.match(node_id):
        raise ValueError(f"Invalid node_id format: {node_id}. Expected: name:v1.0.0")

def parse_node_id(node_id: str) -> tuple[str, str]:
    """Parse node_id into (node_name, version)"""
    validate_node_id(node_id)
    name, version = node_id.split(":v", 1)
    return name, f"v{version}"

def build_node_id(node_name: str, version: str) -> str:
    """Build node_id from name and version"""
    if not version.startswith("v"):
        version = f"v{version}"
    return f"{node_name}:{version}"
```

**Create**: `src/models/node/create.py`

```python
from pydantic import BaseModel, Field, validator
from .validators import validate_node_id

class ContractField(BaseModel):
    name: str
    type: str  # "string", "number", "boolean", "object", "array"
    description: str = ""
    state_path: str = ""  # For output fields: JSONPath to state location

class InputContract(BaseModel):
    required: list[ContractField] = []
    optional: list[ContractField] = []

class OutputContract(BaseModel):
    produced: list[ContractField] = []

class NodeConfig(BaseModel):
    system_prompt: str = ""
    user_prompt_template: str = ""
    input_mapping: dict = {}
    output_schema: dict = {}
    output_key: str = ""
    # CLI node fields
    command: str = ""
    cwd: str = ""
    # Provider fields
    provider: str = ""
    model: str = ""
    temperature: float = 0.0

class Provenance(BaseModel):
    generated_by_workflow: str | None = None
    generated_by_run_id: str | None = None
    generated_at: str | None = None
    source_intent: str | None = None

class NodeCreate(BaseModel):
    tenant_id: str
    node_id: str
    node_name: str
    version: str
    name: str
    type: str  # "agent_node", "cli_node"
    description: str = ""
    config: NodeConfig
    input_contract: InputContract
    output_contract: OutputContract
    tags: list[str] = []
    owner: str = "system"
    status: str = "active"
    reuse_eligible: bool = True
    provenance: Provenance = Provenance()

    @validator("node_id")
    def validate_node_id_format(cls, v):
        validate_node_id(v)
        return v

    @validator("type")
    def validate_type(cls, v):
        if v not in ("agent_node", "cli_node"):
            raise ValueError(f"Invalid type: {v}. Must be agent_node or cli_node")
        return v
```

**Create**: `src/models/node/update.py`

```python
from pydantic import BaseModel

class NodeUpdate(BaseModel):
    """Only metadata fields can be updated. Config/contracts are immutable."""
    name: str | None = None
    description: str | None = None
    tags: list[str] | None = None
```

**Create**: `src/models/node/response.py`

```python
from pydantic import BaseModel
from .create import NodeConfig, InputContract, OutputContract, Provenance

class NodeResponse(BaseModel):
    tenant_id: str
    node_id: str
    node_name: str
    version: str
    name: str
    type: str
    description: str
    config: NodeConfig
    input_contract: InputContract
    output_contract: OutputContract
    tags: list[str]
    owner: str
    status: str
    reuse_eligible: bool
    provenance: Provenance
    created_at: str
    immutable_fields: list[str]

class NodeListResponse(BaseModel):
    nodes: list[NodeResponse]
    total: int
```

**Acceptance Criteria**:

- [ ] `validate_node_id` rejects invalid formats
- [ ] `parse_node_id` extracts name and version correctly
- [ ] `NodeCreate` validates all required fields
- [ ] `NodeCreate` rejects missing config, missing output schema, invalid type
- [ ] `ContractField` includes `type` field
- [ ] `OutputContract` uses `produced` with `state_path`
- [ ] `NodeUpdate` only allows metadata fields (name, description, tags)

---

### Task GW-REG-002: Node Redis Store

**Files**: `src/adapters/node.py`, `src/adapters/redis/namespaced.py`

**Modify**: `src/adapters/redis/namespaced.py`

```python
class NamespacedRedisClient:
    def node_key(self, node_id: str) -> str:
        """HSET key for node data. Never parse node_id from this key."""
        return f"nodes:{self.tenant_id}:{node_id}"

    def node_index_key(self) -> str:
        """SET of all node_ids in tenant"""
        return f"nodes:{self.tenant_id}:index"

    def node_name_key(self, node_name: str) -> str:
        """SET of node_ids for this logical name"""
        return f"nodes:{self.tenant_id}:name:{node_name}"

    def node_tag_key(self, tag: str) -> str:
        """SET of node_ids with this tag"""
        return f"nodes:{self.tenant_id}:tags:{tag}"
```

**Create**: `src/adapters/node.py`

```python
class RedisNodeStore:
    def __init__(self, redis: NamespacedRedisClient):
        self.redis = redis

    async def create(self, node: NodeCreate) -> NodeResponse:
        """Store node in Redis, update indexes"""
        key = self.redis.node_key(node.node_id)

        # Check if exists
        if await self.redis.exists(key):
            raise ConflictError(f"Node {node.node_id} already exists")

        # Store node JSON (HSET for structured access)
        await self.redis.hset(key, mapping=node.dict())

        # Update index SET
        await self.redis.sadd(self.redis.node_index_key(), node.node_id)

        # Update name SET
        await self.redis.sadd(self.redis.node_name_key(node.node_name), node.node_id)

        # Update tag SETs
        for tag in node.tags:
            await self.redis.sadd(self.redis.node_tag_key(tag), node.node_id)

        return NodeResponse(**node.dict())

    async def get(self, node_id: str) -> NodeResponse | None:
        """Get node by exact node_id"""
        key = self.redis.node_key(node_id)
        data = await self.redis.hgetall(key)
        if not data:
            return None
        return NodeResponse(**data)

    async def list(self, tags: list[str] = None, node_name: str = None) -> NodeListResponse:
        """List nodes with optional filters"""
        if node_name:
            node_ids = await self.redis.smembers(self.redis.node_name_key(node_name))
        elif tags:
            # Intersect tag SETs
            tag_keys = [self.redis.node_tag_key(t) for t in tags]
            node_ids = await self.redis.sinter(*tag_keys)
        else:
            node_ids = await self.redis.smembers(self.redis.node_index_key())

        nodes = []
        for node_id in node_ids:
            node = await self.get(node_id)
            if node:
                nodes.append(node)

        return NodeListResponse(nodes=nodes, total=len(nodes))

    async def update(self, node_id: str, update: NodeUpdate) -> NodeResponse:
        """Update node metadata ONLY (name, description, tags)"""
        node = await self.get(node_id)
        if not node:
            raise NotFoundError(f"Node {node_id} not found")

        # Update allowed fields only
        update_data = update.dict(exclude_unset=True)
        for field in update_data:
            setattr(node, field, update_data[field])

        # Save
        key = self.redis.node_key(node_id)
        await self.redis.hset(key, mapping=node.dict())

        # Update tag indexes if changed
        if "tags" in update_data:
            for tag in node.tags:
                await self.redis.srem(self.redis.node_tag_key(tag), node_id)
            for tag in update_data["tags"]:
                await self.redis.sadd(self.redis.node_tag_key(tag), node_id)

        return node

    async def delete(self, node_id: str) -> bool:
        """Delete node and update indexes"""
        node = await self.get(node_id)
        if not node:
            return False

        key = self.redis.node_key(node_id)

        # Remove from indexes
        await self.redis.srem(self.redis.node_index_key(), node_id)
        await self.redis.srem(self.redis.node_name_key(node.node_name), node_id)
        for tag in node.tags:
            await self.redis.srem(self.redis.node_tag_key(tag), node_id)

        # Delete node
        await self.redis.delete(key)
        return True

    async def exists(self, node_id: str) -> bool:
        """Check if node exists"""
        return await self.redis.exists(self.redis.node_key(node_id))

    async def find_by_name(self, node_name: str) -> list[NodeResponse]:
        """Find all versions of a node by name"""
        node_ids = await self.redis.smembers(self.redis.node_name_key(node_name))
        nodes = []
        for node_id in node_ids:
            node = await self.get(node_id)
            if node:
                nodes.append(node)
        return nodes
```

**Acceptance Criteria**:

- [ ] Stores at `nodes:{tenant_id}:{node_id}` using HSET
- [ ] Maintains index SET at `nodes:{tenant_id}:index`
- [ ] Maintains name SET at `nodes:{tenant_id}:name:{node_name}`
- [ ] Maintains tag SETs at `nodes:{tenant_id}:tags:{tag}`
- [ ] Never parses node_id from Redis key
- [ ] Tenant isolation enforced

---

### Task GW-REG-003: Node Validation Service

**Files**: `src/services/node_validator.py`

**Create**: `src/services/node_validator.py`

```python
class NodeValidator:
    def validate_standalone(self, node: NodeCreate) -> list[str]:
        """Validate node is executable standalone"""
        errors = []

        # Must have config
        if not node.config.system_prompt and not node.config.command:
            errors.append("Node must have system_prompt (agent_node) or command (cli_node)")

        # Must have output_contract with produced fields
        if not node.output_contract.produced:
            errors.append("Node must declare produced outputs")

        # Must have input_contract
        if not node.input_contract.required and not node.input_contract.optional:
            errors.append("Node must declare at least one input (required or optional)")

        # Output fields must have state_path
        for field in node.output_contract.produced:
            if not field.state_path:
                errors.append(f"Output field '{field.name}' must have state_path")

        # Input fields must have type
        for field in node.input_contract.required + node.input_contract.optional:
            if not field.type:
                errors.append(f"Input field '{field.name}' must have type")

        return errors

    def validate_contract_compatibility(
        self, upstream: NodeResponse, downstream: NodeResponse
    ) -> list[str]:
        """Validate upstream outputs satisfy downstream inputs (type-aware)"""
        errors = []

        # Build lookup: name → type for upstream produced
        upstream_produced = {f.name: f.type for f in upstream.output_contract.produced}

        # Check downstream required inputs
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
        """Validate compositional workflow structure"""
        errors = []

        nodes = workflow.get("nodes", [])
        edges = workflow.get("edges", [])

        # Build alias set
        aliases = set()
        for node in nodes:
            if "alias" in node:
                aliases.add(node["alias"])
            elif node.get("id") in ("entry", "exit"):
                pass
            else:
                errors.append(f"Node missing 'alias' field: {node}")

        # Validate edges reference valid aliases (both from AND to)
        for edge in edges:
            from_id = edge.get("from")
            to_id = edge.get("to")

            if from_id not in ("entry",) and from_id not in aliases:
                errors.append(f"Edge 'from' references unknown alias: {from_id}")

            if to_id not in ("exit",) and to_id not in aliases:
                errors.append(f"Edge 'to' references unknown alias: {to_id}")

        return errors
```

**Acceptance Criteria**:

- [ ] Rejects node missing config
- [ ] Rejects node missing output_contract.produced
- [ ] Rejects output field missing state_path
- [ ] Rejects input field missing type
- [ ] Validates contract compatibility (type-aware)
- [ ] Validates compositional workflow structure (both from AND to)

---

### Task GW-REG-004: Node CRUD API

**Files**: `src/routers/nodes.py`, `src/modules/shared/deps.py`, `src/main.py`

**Create**: `src/routers/nodes.py`

```python
router = APIRouter(prefix="/nodes", tags=["nodes"])

@router.post("/", response_model=NodeResponse)
async def create_node(
    node: NodeCreate,
    store: RedisNodeStore = Depends(get_node_store),
    validator: NodeValidator = Depends(get_node_validator),
):
    """Create node; tenant_id required in body"""
    errors = validator.validate_standalone(node)
    if errors:
        raise HTTPException(400, detail=errors)
    return await store.create(node)

@router.get("/", response_model=NodeListResponse)
async def list_nodes(
    tenant_id: str = Query(...),
    tags: str = Query(None),
    node_name: str = Query(None),
    store: RedisNodeStore = Depends(get_node_store),
):
    """List/search nodes"""
    tag_list = tags.split(",") if tags else None
    return await store.list(tags=tag_list, node_name=node_name)

@router.get("/{node_id:path}", response_model=NodeResponse)
async def get_node(
    node_id: str,
    tenant_id: str = Query(...),
    store: RedisNodeStore = Depends(get_node_store),
):
    """Get specific node ID"""
    node = await store.get(node_id)
    if not node:
        raise HTTPException(404, detail=f"Node {node_id} not found")
    return node

@router.put("/{node_id:path}", response_model=NodeResponse)
async def update_node(
    node_id: str,
    update: NodeUpdate,
    tenant_id: str = Query(...),
    store: RedisNodeStore = Depends(get_node_store),
):
    """Update metadata for a node ID (name, description, tags only)"""
    try:
        return await store.update(node_id, update)
    except NotFoundError:
        raise HTTPException(404, detail=f"Node {node_id} not found")

@router.delete("/{node_id:path}")
async def delete_node(
    node_id: str,
    tenant_id: str = Query(...),
    store: RedisNodeStore = Depends(get_node_store),
):
    """Delete node ID"""
    deleted = await store.delete(node_id)
    if not deleted:
        raise HTTPException(404, detail=f"Node {node_id} not found")
    return {"deleted": True}
```

**Modify**: `src/modules/shared/deps.py`

```python
def get_node_store() -> RedisNodeStore:
    redis = get_redis_client()
    return RedisNodeStore(redis)

def get_node_validator() -> NodeValidator:
    return NodeValidator()
```

**Modify**: `src/main.py`

```python
from .routers import nodes
app.include_router(nodes.router)
```

**Acceptance Criteria**:

- [ ] POST /nodes creates node (validates before save)
- [ ] GET /nodes lists nodes with tag/name filters
- [ ] GET /nodes/{node_id} returns node
- [ ] PUT /nodes/{node_id} updates metadata only
- [ ] DELETE /nodes/{node_id} deletes node
- [ ] Returns 400 for invalid node_id format
- [ ] Returns 409 for duplicate node_id
- [ ] Returns 404 for missing node

---

### Task GW-REG-005: Node Discovery

**Files**: `src/routers/nodes.py` (already included above)

Discovery is handled by the list endpoint with `tags` and `node_name` filters.

**Acceptance Criteria**:

- [ ] Filter by tags (comma-separated)
- [ ] Filter by node_name
- [ ] Returns all versions when filtering by name

---

## Phase 2: Seed Node Library + Runtime Compiler

**Goal**: Seed the registry with common nodes, implement compiler, switch to compositional-only.

### Task GW-LIB-001: Extract normalize_input Node

**Files**: `src/resources/nodes/normalize_input:v1.0.0.json`

**Create**: Node JSON with typed contracts:

```json
{
  "tenant_id": "system",
  "node_id": "normalize_input:v1.0.0",
  "node_name": "normalize_input",
  "version": "1.0.0",
  "name": "Normalize Input",
  "type": "agent_node",
  "description": "Trim content and normalize tag input",
  "config": {
    "system_prompt": "You are a data normalizer...",
    "user_prompt_template": "Normalize this input:\n{file_content}",
    "input_mapping": {},
    "output_schema": { "type": "object", "required": ["file_content", "existing_tags"] }
  },
  "input_contract": {
    "required": [
      { "name": "file_path", "type": "string" },
      { "name": "file_content", "type": "string" }
    ],
    "optional": [{ "name": "existing_tags", "type": "array" }]
  },
  "output_contract": {
    "produced": [
      { "name": "file_path", "type": "string", "state_path": "$.normalize_input.file_path" },
      { "name": "file_content", "type": "string", "state_path": "$.normalize_input.file_content" },
      { "name": "existing_tags", "type": "array", "state_path": "$.normalize_input.existing_tags" }
    ]
  },
  "tags": ["normalization", "preprocessing"],
  "owner": "system",
  "status": "active",
  "reuse_eligible": true,
  "provenance": {}
}
```

**Modify**: `src/main.py`

- Load predefined nodes on startup from `src/resources/nodes/`

**Acceptance Criteria**:

- [ ] Node JSON is valid
- [ ] Contracts have types and state_paths
- [ ] Node loaded on startup
- [ ] Node accessible via API

---

### Task GW-LIB-002: Extract classify_metadata Node

**Files**: `src/resources/nodes/classify_metadata:v1.0.0.json`

Similar structure to normalize_input.

---

### Task GW-LIB-003: Extract Common Nodes

**Files**: `src/resources/nodes/*.json`

**Extract** (8 nodes):

- `normalize_input:v1.0.0`
- `classify_metadata:v1.0.0`
- `research_analyst:v1.0.0`
- `fact_checker:v1.0.0`
- `summarizer:v1.0.0`
- `classifier:v1.0.0`
- `data_transformer:v1.0.0`
- `cli_executor:v1.0.0`

All nodes must have typed contracts with `state_path`.

---

### Task GW-RUN-001: Workflow Compiler

**Files**: `src/services/node_compiler.py`

**Create**: `src/services/node_compiler.py`

```python
class WorkflowCompiler:
    def __init__(self, node_store: RedisNodeStore):
        self.node_store = node_store

    async def compile(self, workflow: dict) -> dict:
        """Resolve node_ids into embedded nodes for execution"""
        nodes = workflow.get("nodes", [])

        if not nodes:
            return workflow

        # Detect format
        has_node_refs = any("node_id" in n for n in nodes)

        if has_node_refs:
            return await self._compile_compositional(workflow)

        # Reject legacy embedded format
        raise WorkflowFormatError(
            "Legacy embedded workflow format not supported. "
            "Use compositional format with alias/node_id references."
        )

    async def _compile_compositional(self, workflow: dict) -> dict:
        """Compile compositional workflow to embedded format"""
        compiled_nodes = []

        for node_ref in workflow.get("nodes", []):
            # Pass through entry/exit
            if node_ref.get("type") in ("entry", "exit"):
                compiled_nodes.append(node_ref)
                continue

            # Resolve node reference
            node_id = node_ref.get("node_id")
            alias = node_ref.get("alias")
            overrides = node_ref.get("overrides", {})

            if not node_id:
                raise WorkflowCompileError(f"Node reference missing node_id: {node_ref}")

            # Load from registry
            stored_node = await self.node_store.get(node_id)
            if not stored_node:
                raise WorkflowCompileError(f"Node {node_id} not found in registry")

            # Build embedded node
            compiled_node = {
                "id": alias,
                "type": stored_node.type,
                "display_name": stored_node.name,
                "description": stored_node.description,
                "config": {
                    **stored_node.config.dict(),
                    **overrides.get("input_mapping", {}),
                },
            }

            # Apply output_key override
            if "output_key" in overrides:
                compiled_node["config"]["output_key"] = overrides["output_key"]

            compiled_nodes.append(compiled_node)

        return {**workflow, "nodes": compiled_nodes}

    async def validate_references(self, workflow: dict) -> list[str]:
        """Validate all node_ids exist in registry"""
        errors = []
        for node_ref in workflow.get("nodes", []):
            if "node_id" in node_ref:
                if not await self.node_store.exists(node_ref["node_id"]):
                    errors.append(f"Node {node_ref['node_id']} not found")
        return errors

    async def validate_contracts(self, workflow: dict) -> list[str]:
        """Validate input/output contracts between connected nodes (type-aware)"""
        errors = []
        validator = NodeValidator()

        # Build node lookup by alias
        nodes_by_alias = {}
        for node_ref in workflow.get("nodes", []):
            alias = node_ref.get("alias") or node_ref.get("id")
            if "node_id" in node_ref:
                stored_node = await self.node_store.get(node_ref["node_id"])
                if stored_node:
                    nodes_by_alias[alias] = stored_node

        # Validate edges
        for edge in workflow.get("edges", []):
            from_node = nodes_by_alias.get(edge["from"])
            to_node = nodes_by_alias.get(edge["to"])

            if from_node and to_node:
                edge_errors = validator.validate_contract_compatibility(from_node, to_node)
                for err in edge_errors:
                    errors.append(f"Edge {edge['from']}→{edge['to']}: {err}")

        return errors
```

**Acceptance Criteria**:

- [ ] Compositional workflow: resolves node_ids to embedded nodes
- [ ] Legacy embedded workflow: rejects with clear error
- [ ] Missing node: raises WorkflowCompileError
- [ ] Validates all references before compilation
- [ ] Validates contracts between connected nodes (type-aware)
- [ ] Compiled workflow matches existing embedded format

---

### Task GW-RUN-002: Contract Validation

**Included in GW-RUN-001 above** (part of WorkflowCompiler.validate_contracts).

**Acceptance Criteria**:

- [ ] Type-aware validation
- [ ] Reports specific type mismatches
- [ ] Validates all edges

---

### Task GW-RUN-003: Parser Update (Compositional Only)

**Files**: `src/adapters/langgraph/graph/parser.py`

**Modify**: `src/adapters/langgraph/graph/parser.py`

```python
class WorkflowParser:
    def parse_workflow_json(self, workflow: dict) -> ParsedWorkflow:
        """Parse compositional workflow JSON only"""
        nodes = workflow.get("nodes", [])

        if not nodes:
            raise WorkflowParseError("Workflow has no nodes")

        # Detect format
        has_node_refs = any("node_id" in n for n in nodes)
        has_entry_exit = any(n.get("type") in ("entry", "exit") for n in nodes)

        if not has_node_refs and not has_entry_exit:
            raise WorkflowParseError(
                "Invalid workflow format. Expected compositional format with "
                "alias/node_id references. Legacy embedded format not supported."
            )

        # Parse compositional format
        return self._parse_compositional(workflow)

    def _parse_compositional(self, workflow: dict) -> ParsedWorkflow:
        """Parse compositional workflow with node_id references"""
        nodes = workflow.get("nodes", [])
        edges = workflow.get("edges", [])

        # Validate structure
        aliases = set()
        for node in nodes:
            if node.get("type") in ("entry", "exit"):
                continue
            if "alias" not in node:
                raise WorkflowParseError(f"Node missing 'alias': {node}")
            if "node_id" not in node:
                raise WorkflowParseError(f"Node missing 'node_id': {node}")
            aliases.add(node["alias"])

        # Validate edges reference valid aliases (both from AND to)
        for edge in edges:
            from_id = edge.get("from")
            to_id = edge.get("to")

            if from_id not in ("entry",) and from_id not in aliases:
                raise WorkflowParseError(f"Edge references unknown alias: {from_id}")
            if to_id not in ("exit",) and to_id not in aliases:
                raise WorkflowParseError(f"Edge references unknown alias: {to_id}")

        return ParsedWorkflow(nodes=nodes, edges=edges, is_compositional=True)
```

**Acceptance Criteria**:

- [ ] Parses compositional format
- [ ] Rejects legacy embedded format with clear error
- [ ] Validates alias/node_id presence
- [ ] Validates edge references (both from AND to)

---

### Task GW-RUN-004: Update Builder

**Files**: `src/adapters/langgraph/graph/builder.py`

**Modify**: `src/adapters/langgraph/graph/builder.py`

```python
class GraphBuilder:
    def __init__(self, compiler: WorkflowCompiler):
        self.compiler = compiler

    async def build(self, workflow: dict) -> CompiledGraph:
        """Build executable graph from compositional workflow"""
        # Compile node_ids to embedded nodes
        compiled = await self.compiler.compile(workflow)

        # Validate contracts
        contract_errors = await self.compiler.validate_contracts(workflow)
        if contract_errors:
            raise WorkflowValidationError(contract_errors)

        # Build with existing logic using compiled (embedded) workflow
        return self._build_from_embedded(compiled)
```

**Acceptance Criteria**:

- [ ] Uses compiler to resolve node_ids
- [ ] Validates contracts before build
- [ ] Rejects legacy format

---

### Task GW-RUN-005: Update Workflow Storage

**Files**: `src/adapters/workflow.py`

**Modify**: `src/adapters/workflow.py`

```python
class RedisWorkflowStore:
    def __init__(self, redis: NamespacedRedisClient, node_store: RedisNodeStore):
        self.redis = redis
        self.node_store = node_store
        self.compiler = WorkflowCompiler(node_store)

    async def create(self, workflow: dict, tenant_id: str, workflow_id: str) -> dict:
        """Store compositional workflow source (NOT compiled)"""
        # Validate format
        parser = WorkflowParser()
        parsed = parser.parse_workflow_json(workflow)

        # Validate node references exist
        ref_errors = await self.compiler.validate_references(workflow)
        if ref_errors:
            raise WorkflowValidationError(ref_errors)

        # Store compositional source
        key = self.redis.workflow_key(workflow_id)
        await self.redis.set(key, workflow)

        # Update index
        await self.redis.sadd(self.redis.workflow_index_key(), workflow_id)

        return workflow

    async def get(self, workflow_id: str, tenant_id: str) -> dict | None:
        """Get compositional workflow source"""
        key = self.redis.workflow_key(workflow_id)
        return await self.redis.get(key)

    async def get_compiled(self, workflow_id: str, tenant_id: str) -> dict | None:
        """Get workflow and compile for execution"""
        workflow = await self.get(workflow_id, tenant_id)
        if not workflow:
            return None
        return await self.compiler.compile(workflow)
```

**Acceptance Criteria**:

- [ ] Stores compositional source only (not compiled)
- [ ] Validates references on create
- [ ] get() returns compositional source
- [ ] get_compiled() returns embedded form for execution

---

## Phase 3: Generator v2.0.0

**Goal**: New generator that selects/saves nodes from registry using seed library.

### Task GW-GEN-001: Generator Workflow JSON

**Files**: `src/resources/workflows/workflow-generator:v2.0.0.json`

**Delete**: `src/resources/workflows/workflow-generator:v1.0.1.json`

**Create**: v2.0.0 with:

- `node_selector` orchestrator that uses registry tools
- `edge_generator` with contract validation
- Output is compositional format (alias/node_id references)

**Acceptance Criteria**:

- [ ] Valid compositional workflow JSON
- [ ] Uses orchestrator for node selection
- [ ] Output references node_ids
- [ ] v1.0.1 deleted

---

### Task GW-GEN-002: Node Selector Tools

**Files**: `src/adapters/mcp/tools/node_registry.py`

**Create**: MCP tools for node registry:

- `search_nodes(tenant_id, query, tags)` → list of matching nodes
- `get_node(node_id, tenant_id)` → full node definition
- `create_node(node_data)` → create and return node
- `find_similar_node(tenant_id, description)` → best match or None

**Modify**: `src/adapters/mcp/registry.py` and `router.py`

- Register node registry tools

**Acceptance Criteria**:

- [ ] Tools registered in MCP
- [ ] Accessible by orchestrator
- [ ] search_nodes returns matches
- [ ] create_node stores in registry

---

### Task GW-GEN-003: Edge Generator

**Part of generator workflow prompt** — validates contracts when generating edges.

---

### Task GW-GEN-004: Custom Node Generation

**Part of node_selector** — when no match found:

1. Generate node config via LLM
2. Validate standalone executability
3. Auto-save with provenance metadata
4. Return node_id

**Generated node includes**:

```json
{
  "reuse_eligible": true,
  "provenance": {
    "generated_by_workflow": "workflow-generator:v2.0.0",
    "generated_by_run_id": "run_abc123",
    "generated_at": "2026-05-14T10:30:00Z",
    "source_intent": "Research AI ethics..."
  }
}
```

---

### Task GW-GEN-005: Comparison Tests

**Files**: `tests/e2e/test_e2e_generator_v2.py`

**Create**: Tests comparing v1 and v2 outputs.

---

## Phase 4: Cleanup & Documentation

**Goal**: Remove legacy, update docs.

### Task GW-CLEAN-001: Remove Legacy

- Delete v1.0.1 generator JSON
- Remove embedded workflow format from all docs
- Update all tests to use compositional format
- Remove any backward compatibility code

---

### Task GW-CLEAN-002: Update WORKFLOW_JSON_SPEC.md

- Add compositional workflow schema
- Document compilation process
- Remove embedded format documentation

---

### Task GW-CLEAN-003: Node Authoring Guide

- Document node JSON format
- Document typed contracts with state_path
- Document immutability rules
- Document provenance metadata

---

## Execution Order

```
Phase 1 (Infrastructure):
├── GW-REG-001: Node Models (with typed contracts)
├── GW-REG-002: Node Redis Store (HSET + SET indexes)
├── GW-REG-003: Node Validation Service (type-aware)
├── GW-REG-004: Node CRUD API
└── GW-REG-005: Node Discovery

Phase 2 (Seed Library + Runtime):
├── GW-LIB-001: normalize_input node
├── GW-LIB-002: classify_metadata node
├── GW-LIB-003: Common nodes (6 more)
├── GW-RUN-001: Workflow Compiler (with contract validation)
├── GW-RUN-003: Parser (compositional only)
├── GW-RUN-004: Builder update
└── GW-RUN-005: Workflow storage update

Phase 3 (Generator):
├── GW-GEN-001: Generator v2.0.0 JSON
├── GW-GEN-002: Node selector tools
├── GW-GEN-003: Edge generator (in workflow)
├── GW-GEN-004: Custom node generation (with provenance)
└── GW-GEN-005: Comparison tests

Phase 4 (Cleanup):
├── GW-CLEAN-001: Remove legacy
├── GW-CLEAN-002: Update spec doc
└── GW-CLEAN-003: Authoring guide
```

**Dependencies**:

- Phase 2 depends on Phase 1 (needs node store)
- Phase 3 depends on Phase 2 (needs compiler + seed library)
- Phase 4 depends on Phase 3 (needs generator v2 working)

---

## Test Matrix

| Test File                         | Phase | Coverage                                                |
| --------------------------------- | ----- | ------------------------------------------------------- |
| `test_node_store.py`              | 1     | CRUD, tenant isolation, indexes, immutability           |
| `test_node_validator.py`          | 1     | Standalone validation, contract validation (type-aware) |
| `test_node_api.py`                | 1     | API endpoints, error responses                          |
| `test_node_compiler.py`           | 2     | Compilation, contract validation, legacy rejection      |
| `test_langgraph_graph_builder.py` | 2     | Parser (compositional), builder with compiled nodes     |
| `test_e2e_node_registry.py`       | 2     | Full node lifecycle via API                             |
| `test_e2e_generator_v2.py`        | 3     | Generator v2 end-to-end                                 |
| `test_e2e_workflow_execution.py`  | 2     | Execution with compositional workflows                  |

### Key Test Cases

```python
# Node immutability
async def test_node_immutability():
    """PUT rejects changes to config, input_contract, output_contract"""
    await create_node({"node_id": "test:v1.0.0", "config": {...}, ...})
    with pytest.raises(ImmutableFieldError):
        await update_node("test:v1.0.0", {"config": {"system_prompt": "new"}})

# Type-aware contract validation
async def test_contract_type_mismatch():
    """Rejects type mismatch between upstream output and downstream input"""
    node_a = {"output_contract": {"produced": [{"name": "data", "type": "object"}]}}
    node_b = {"input_contract": {"required": [{"name": "data", "type": "string"}]}}
    with pytest.raises(IncompatibleContract):
        validate_edge(node_a, node_b)

# Legacy rejection
async def test_legacy_format_rejected():
    """Parser rejects embedded workflow format"""
    workflow = {"nodes": [{"id": "test", "type": "agent_node", "config": {...}}]}
    with pytest.raises(WorkflowFormatError):
        await compile_workflow(workflow)

# Edge validation (both from AND to)
async def test_edge_to_validation():
    """E2E validates edge 'to' values, not just 'from'"""
    generated = result["generated_workflow"]
    aliases = {n["alias"] for n in generated["nodes"]}
    valid_from = {"entry"} | aliases
    valid_to = {"exit"} | aliases
    assert all(e["from"] in valid_from for e in generated["edges"])
    assert all(e["to"] in valid_to for e in generated["edges"])
```

---

## Rollback Plan

If issues arise:

1. **Phase 1**: Remove node router, keep workflow CRUD unchanged
2. **Phase 2**: Revert parser/builder changes, keep legacy format only
3. **Phase 3**: Keep generator v1.0.1, don't deploy v2.0.0
4. **Phase 4**: Remove predefined nodes, keep manual workflow creation

---

## Test Matrix

---

## Success Criteria

| Metric                           | Target                  |
| -------------------------------- | ----------------------- |
| All existing tests updated       | 100% pass               |
| Node CRUD API functional         | All endpoints working   |
| Compositional workflows execute  | End-to-end success      |
| Generator v2 produces valid DAGs | >95% success rate       |
| No legacy format support         | Clean removal           |
| Typed contracts enforced         | Type mismatches caught  |
| Node immutability enforced       | Config changes rejected |
| Seed library available           | 8+ nodes in registry    |

---

## References

- Plan: `decomposed-generation-and-node-registry.md`
- Workflow JSON spec: `WORKFLOW_JSON_SPEC.md` (to be updated)
- Current generator: `src/resources/workflows/workflow-generator:v1.0.1.json` (to be deleted)

---

**End of Document**
