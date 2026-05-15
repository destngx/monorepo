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

    try:
        node.validate_type()
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))

    errors = validator.validate_standalone(node)
    if errors:
        raise HTTPException(status_code=400, detail=errors)

    try:
        return await store.create(node)
    except ConflictError as e:
        raise HTTPException(status_code=409, detail=str(e))


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
