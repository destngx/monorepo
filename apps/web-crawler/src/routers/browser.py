from fastapi import APIRouter, Depends

from src.models.browser import BrowserProbeRequest, BrowserProbeResponse
from src.modules.shared.deps import get_browser_probe_service
from src.services.browser_probe import BrowserProbeService

router = APIRouter(tags=["Browser"])


@router.post("/v1/pages/probe/browser", response_model=BrowserProbeResponse)
async def probe_browser(
    request: BrowserProbeRequest,
    service: BrowserProbeService = Depends(get_browser_probe_service),
) -> BrowserProbeResponse:
    step = await service.probe(
        url=request.url,
        page_url=request.page_url,
        cookie=request.cookie,
        timeout_seconds=request.timeout_seconds or 45,
    )
    return BrowserProbeResponse(step=step)
