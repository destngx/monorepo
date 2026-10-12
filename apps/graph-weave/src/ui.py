"""
Serves the built GraphWeave console (apps/graph-weave/ui) from the API process.

The UI is a SvelteKit SPA built with adapter-static under the `/ui` base path.
API routes stay at the root, so the UI never shadows them.
"""

from pathlib import Path

from fastapi import FastAPI
from fastapi.responses import RedirectResponse
from fastapi.staticfiles import StaticFiles
from starlette.exceptions import HTTPException
from starlette.types import Scope

from .app_logging import get_logger

logger = get_logger(__name__)

UI_MOUNT_PATH = "/ui"
DEFAULT_UI_BUILD_DIR = Path(__file__).resolve().parent.parent / "ui" / "build"
SPA_INDEX = "index.html"


class SpaStaticFiles(StaticFiles):
    """Static files that fall back to the SPA shell for client-side routes."""

    async def get_response(self, path: str, scope: Scope):
        try:
            return await super().get_response(path, scope)
        except HTTPException as exc:
            # Paths with a file extension are real assets; a miss there is a genuine 404.
            if exc.status_code != 404 or Path(path).suffix:
                raise
            return await super().get_response(SPA_INDEX, scope)


def mount_ui(app: FastAPI, build_dir: Path = DEFAULT_UI_BUILD_DIR) -> bool:
    """Mount the UI at /ui and redirect / to it. Returns False when no build exists."""
    if not (build_dir / SPA_INDEX).is_file():
        logger.info(f"UI build not found at {build_dir}; run `pnpx nx run graph-weave:build-ui` to serve the console")
        return False

    app.mount(UI_MOUNT_PATH, SpaStaticFiles(directory=build_dir, html=True), name="ui")

    @app.get("/", include_in_schema=False)
    async def redirect_to_ui():
        return RedirectResponse(f"{UI_MOUNT_PATH}/")

    logger.info(f"Serving UI from {build_dir} at {UI_MOUNT_PATH}/")
    return True
