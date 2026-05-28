import json
from pathlib import Path

from src.models.crawl import CrawlCheckpoint

from .crawl_paths import CrawlPaths, domain_from_url


class CrawlStorage:
    def __init__(self, output_dir: str, checkpoint_suffix: str = ".checkpoint.json"):
        self.output_dir = Path(output_dir).expanduser()
        self.checkpoint_suffix = checkpoint_suffix

    def thread_dir(self, url: str, thread_id: str) -> Path:
        domain = domain_from_url(url)
        return self.output_dir / domain / thread_id

    def resolve_thread_paths(self, url: str, thread_id: str) -> CrawlPaths:
        thread_dir = self.thread_dir(url, thread_id)
        return CrawlPaths(
            thread_dir=thread_dir,
            output_path=thread_dir / f"{thread_id}.json",
            checkpoint_path=thread_dir / f"{thread_id}{self.checkpoint_suffix}",
        )

    def output_path(self, url: str, thread_id: str) -> Path:
        return self.resolve_thread_paths(url, thread_id).output_path

    def checkpoint_path(self, url: str, thread_id: str) -> Path:
        return self.resolve_thread_paths(url, thread_id).checkpoint_path

    def load_crawl(self, output_path: Path) -> list[dict] | None:
        if not output_path.exists():
            return None
        return json.loads(output_path.read_text(encoding="utf-8"))

    def load_checkpoint(self, checkpoint_path: Path) -> CrawlCheckpoint | None:
        if not checkpoint_path.exists():
            return None
        return CrawlCheckpoint.model_validate_json(
            checkpoint_path.read_text(encoding="utf-8")
        )

    def write_crawl(self, output_path: Path, crawled_data: list[dict]) -> None:
        output_path.write_text(
            json.dumps(crawled_data, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )

    def write_checkpoint(
        self,
        checkpoint_path: Path,
        checkpoint: CrawlCheckpoint,
    ) -> None:
        checkpoint_path.write_text(
            checkpoint.model_dump_json(indent=2),
            encoding="utf-8",
        )

    @staticmethod
    def clear_thread_state(*paths: Path) -> None:
        for path in paths:
            path.unlink(missing_ok=True)

    @staticmethod
    def ensure_thread_dir(path: Path) -> None:
        path.mkdir(parents=True, exist_ok=True)
