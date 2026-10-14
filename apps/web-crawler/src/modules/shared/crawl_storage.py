import json
import os
from datetime import UTC, datetime
from pathlib import Path

from src.models.crawl import CrawlCheckpoint

from .crawl_paths import CrawlPaths, domain_from_url


class CrawlStorage:
    """Thread crawl state on disk.

    `<thread_id>.pages.jsonl` is the durable source of truth: one page record
    per line, appended and fsynced before the checkpoint moves past it. The
    checkpoint and the assembled `<thread_id>.json` are replaced atomically.
    """

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
            pages_path=thread_dir / f"{thread_id}.pages.jsonl",
            checkpoint_path=thread_dir / f"{thread_id}{self.checkpoint_suffix}",
        )

    def load_pages(self, pages_path: Path) -> dict[int, dict]:
        """Return saved pages keyed by page number; later records win.

        A torn trailing line left by a crash mid-append is truncated so the
        next append starts on a clean line. Corruption anywhere else raises.
        """
        if not pages_path.exists():
            return {}

        raw = pages_path.read_bytes()
        pages: dict[int, dict] = {}
        offset = 0
        for line in raw.splitlines(keepends=True):
            if not line.endswith(b"\n"):
                with pages_path.open("r+b") as handle:
                    handle.truncate(offset)
                break
            if line.strip():
                record = json.loads(line)
                pages[int(record["page"])] = record
            offset += len(line)

        return pages

    def append_page(self, pages_path: Path, page_record: dict) -> None:
        line = json.dumps(page_record, ensure_ascii=False) + "\n"
        with pages_path.open("a", encoding="utf-8") as handle:
            handle.write(line)
            handle.flush()
            os.fsync(handle.fileno())

    def load_checkpoint(self, checkpoint_path: Path) -> CrawlCheckpoint | None:
        if not checkpoint_path.exists():
            return None
        return CrawlCheckpoint.model_validate_json(
            checkpoint_path.read_text(encoding="utf-8")
        )

    def write_crawl(self, output_path: Path, url: str, pages: dict[int, dict]) -> None:
        crawled_data = [
            {"url": url, "pages": [pages[number] for number in sorted(pages)]}
        ]
        self._write_atomic(
            output_path, json.dumps(crawled_data, ensure_ascii=False, indent=2)
        )

    def write_checkpoint(
        self,
        checkpoint_path: Path,
        checkpoint: CrawlCheckpoint,
    ) -> None:
        self._write_atomic(checkpoint_path, checkpoint.model_dump_json(indent=2))

    def write_snapshot(self, thread_dir: Path, page: int, kind: str, html: str) -> Path:
        """Keep the HTML of a page that failed so the cause can be inspected."""
        snapshot_dir = thread_dir / "snapshots"
        snapshot_dir.mkdir(parents=True, exist_ok=True)
        stamp = datetime.now(UTC).strftime("%Y%m%dT%H%M%S%fZ")
        path = snapshot_dir / f"page-{page}-{kind}-{stamp}.html"
        path.write_text(html, encoding="utf-8")
        return path

    @staticmethod
    def clear_thread_state(*paths: Path) -> None:
        for path in paths:
            path.unlink(missing_ok=True)

    @staticmethod
    def ensure_thread_dir(path: Path) -> None:
        path.mkdir(parents=True, exist_ok=True)

    @staticmethod
    def _write_atomic(path: Path, text: str) -> None:
        tmp_path = path.with_name(f"{path.name}.tmp")
        with tmp_path.open("w", encoding="utf-8") as handle:
            handle.write(text)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(tmp_path, path)
