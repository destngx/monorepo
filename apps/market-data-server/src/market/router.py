"""
Provider routing (ported from WM `service/market_provider`): try a chain in
order. A source without the capability, or one raising Unsupported, is
skipped; one raising NotFound has no such item; any other error is a
failure. In every case the chain moves on.
"""

import logging
from dataclasses import dataclass, field
from typing import Any, Awaitable, Callable, Dict, List

from .errors import NotFound, Unsupported, UpstreamError, ValidationError

logger = logging.getLogger(__name__)


@dataclass
class Routed:
    value: Any
    provider: str
    # Failures and misses before the provider that answered (the data-source
    # indicator).
    fallback: List[str] = field(default_factory=list)


class Router:
    def __init__(self, sources: Dict[str, Any]):
        self.sources = {name.lower(): source for name, source in sources.items()}

    async def route(self, operation: str, chain: List[str], method: str, call: Callable[[Any], Awaitable[Any]]) -> Routed:
        """
        `call(source)` runs the capability on one source. When nothing
        answers: an UpstreamError listing every failure, then every skip, if
        any source failed; else a NotFound if any source had no such item;
        else (all skipped) a ValidationError ("no provider covers ...").
        """
        if not chain:
            raise ValidationError(f"no provider chain configured for {operation}")
        # Failures and misses in chain order; `failed` tells them apart.
        failures: List[str] = []
        failed = False
        skipped: List[str] = []
        for name in chain:
            source = self.sources.get(name.lower())
            if source is None:
                failures.append(f"unknown market provider: {name}")
                failed = True
                continue
            if not callable(getattr(source, method, None)):
                skipped.append(f"{source.name}: capability not implemented")
                continue
            try:
                value = await call(source)
            except Unsupported as e:
                skipped.append(f"{source.name}: {e}")
                continue
            except NotFound as e:
                failures.append(f"{source.name}: {e}")
                continue
            except Exception as e:
                logger.warning("%s: %s failed: %s", operation, source.name, e)
                failures.append(f"{source.name}: {str(e) or type(e).__name__}")
                failed = True
                continue
            if failures:
                logger.info("%s: fallback %s used after: %s", operation, source.name, "; ".join(failures))
            return Routed(value=value, provider=source.name, fallback=failures)

        if failed:
            raise UpstreamError(f"{operation} failed: {'; '.join(failures + skipped)}")
        if failures:
            raise NotFound("; ".join(failures))
        raise ValidationError(f"no provider covers {operation} ({'; '.join(skipped)})")
