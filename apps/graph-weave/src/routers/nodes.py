from fastapi import APIRouter, Depends, HTTPException, Query

from ..models.node import NodeCreate, NodeUpdate, NodeResponse, NodeListResponse
from ..models.node.validators import NodeIdFormatError
from ..modules.shared.deps import get_node_store, get_node_validator
from ..adapters.node import ConflictError, NotFoundError

router = APIRouter(prefix="/nodes", tags=["Nodes"])


@router.post("/", response_model=NodeResponse, status_code=201)
async def create_node(
    node: NodeCreate,
    store=Depends(get_node_store),
    validator=Depends(get_node_validator),
):
    try:
        node.validate_node_id_format()
    except NodeIdFormatError as e:
        raise HTTPException(status_code=400, detail=str(e))

    errors = validator.validate_standalone(node)
    if errors:
        raise HTTPException(status_code=400, detail=errors)

    try:
        return await store.create(node)
    except ConflictError:
        existing = await store.get(node.node_id)
        if existing:
            # Review existing vs requested node to detect contract or configuration drift
            existing_cmd = (existing.config or {}).get("command")
            new_cmd = (node.config or {}).get("command")
            
            existing_produced = {p.get("name") for p in (existing.output_contract or {}).get("produced", []) if isinstance(p, dict)}
            new_produced = {p.name for p in (node.output_contract or {}).get("produced", [])}
            
            existing_required = {r.get("name") for r in (existing.input_contract or {}).get("required", []) if isinstance(r, dict)}
            new_required = {r.name for r in (node.input_contract or {}).get("required", [])}
            
            needs_update = (
                existing.type != node.type or
                existing_cmd != new_cmd or
                existing_produced != new_produced or
                existing_required != new_required
            )
            
            if needs_update:
                # Catalog self-healing: Atomically recreate the node to refresh contracts and commands
                await store.delete(node.node_id)
                return await store.create(node)
            else:
                return existing
        else:
            raise HTTPException(status_code=409, detail="Node conflict occurred")


@router.get("/", response_model=NodeListResponse)
async def list_nodes(
    tenant_id: str = Query(...),
    tags: str = Query(None),
    node_name: str = Query(None),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    store=Depends(get_node_store),
):
    tag_list = tags.split(",") if tags else None
    return await store.list(
        tags=tag_list,
        node_name=node_name,
        page=page,
        page_size=page_size,
    )


@router.get("/{node_id:path}", response_model=NodeResponse)
async def get_node(
    node_id: str,
    tenant_id: str = Query(...),
    store=Depends(get_node_store),
):
    node = await store.get(node_id)
    if not node:
        raise HTTPException(status_code=404, detail=f"Node {node_id} not found")
    return node


@router.put("/{node_id:path}", response_model=NodeResponse)
async def update_node(
    node_id: str,
    update: NodeUpdate,
    tenant_id: str = Query(...),
    store=Depends(get_node_store),
):
    try:
        return await store.update(node_id, update)
    except NotFoundError:
        raise HTTPException(status_code=404, detail=f"Node {node_id} not found")


@router.delete("/{node_id:path}")
async def delete_node(
    node_id: str,
    tenant_id: str = Query(...),
    store=Depends(get_node_store),
):
    deleted = await store.delete(node_id)
    if not deleted:
        raise HTTPException(status_code=404, detail=f"Node {node_id} not found")
    return {"deleted": True}


@router.get("/verify", response_model=dict)
async def verify_node(
    node_id: str = Query(...),
    tenant_id: str = Query(...),
    type: str = Query(...),
    command: str = Query(None),
    produced_fields: str = Query(None),
    required_fields: str = Query(None),
    store=Depends(get_node_store),
):
    node = await store.get(node_id)
    if not node:
        return {"is_reusable": False, "reason": "Node not found"}

    # 1. Verify Type
    if node.type != type:
        return {"is_reusable": False, "reason": f"Type mismatch: catalog has '{node.type}', step wants '{type}'"}

    # 2. Verify Command
    if type == "cli_node":
        existing_cmd = (node.config or {}).get("command")
        if existing_cmd != command:
            return {"is_reusable": False, "reason": "Command mismatch"}

    # 3. Verify Produced Fields
    if produced_fields:
        fields = {f.strip() for f in produced_fields.split(",") if f.strip()}
        existing_produced = {p.get("name") for p in (node.output_contract or {}).get("produced", []) if isinstance(p, dict)}
        for f in fields:
            if f not in existing_produced:
                return {"is_reusable": False, "reason": f"Output field '{f}' is missing in catalog"}

    # 4. Verify Required Fields
    if required_fields:
        fields = {f.strip() for f in required_fields.split(",") if f.strip()}
        existing_required = {r.get("name") for r in (node.input_contract or {}).get("required", []) if isinstance(r, dict)}
        for f in fields:
            if f not in existing_required:
                return {"is_reusable": False, "reason": f"Input field '{f}' is missing in catalog"}

    return {"is_reusable": True}
