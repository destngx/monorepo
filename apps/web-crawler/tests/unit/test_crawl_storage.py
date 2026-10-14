import json

from src.modules.shared.crawl_storage import CrawlStorage


def _record(page: int, text: str = "post") -> dict:
    return {
        "page": page,
        "url": f"https://voz.vn/t/x.1/page-{page}",
        "replies": [{"author": "a", "created_at": "", "text": text}],
    }


def test_load_pages_truncates_torn_trailing_line(tmp_path):
    storage = CrawlStorage(str(tmp_path))
    pages_path = tmp_path / "1.pages.jsonl"
    storage.append_page(pages_path, _record(1))
    with pages_path.open("a", encoding="utf-8") as handle:
        handle.write('{"page": 2, "url": "https://voz')

    pages = storage.load_pages(pages_path)

    assert list(pages) == [1]
    storage.append_page(pages_path, _record(2))
    assert list(storage.load_pages(pages_path)) == [1, 2]


def test_load_pages_keeps_latest_record_per_page(tmp_path):
    storage = CrawlStorage(str(tmp_path))
    pages_path = tmp_path / "1.pages.jsonl"
    storage.append_page(pages_path, _record(1, "old"))
    storage.append_page(pages_path, _record(2))
    storage.append_page(pages_path, _record(1, "new"))

    pages = storage.load_pages(pages_path)

    assert sorted(pages) == [1, 2]
    assert pages[1]["replies"][0]["text"] == "new"


def test_write_crawl_is_sorted_and_leaves_no_temp_file(tmp_path):
    storage = CrawlStorage(str(tmp_path))
    output_path = tmp_path / "1.json"

    storage.write_crawl(
        output_path, "https://voz.vn/t/x.1/", {2: _record(2), 1: _record(1)}
    )

    payload = json.loads(output_path.read_text(encoding="utf-8"))
    assert [page["page"] for page in payload[0]["pages"]] == [1, 2]
    assert [path.name for path in tmp_path.iterdir()] == ["1.json"]
