from dataclasses import dataclass
from pathlib import Path
from urllib.parse import urlparse


@dataclass(frozen=True, slots=True)
class CrawlPaths:
    thread_dir: Path
    output_path: Path
    checkpoint_path: Path


def domain_from_url(url: str) -> str:
    parsed = urlparse(url)
    return parsed.netloc or "unknown"
