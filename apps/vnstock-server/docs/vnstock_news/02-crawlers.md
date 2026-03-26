# 02. Các Loại Crawler & API

Tài liệu này mô tả chi tiết tất cả các crawler và cách sử dụng từng loại.

---

## 1. Crawler - Unified Interface

`Crawler` là lớp chính đơn giản nhất, hỗ trợ cả RSS và Sitemap.

### Khởi Tạo

```python
from vnstock_news import Crawler

# Cách 1: Dùng site_name (có sẵn config) -> Lưu ý: Đối với CafeF, config mặc định KHÔNG có RSS
crawler = Crawler(site_name="cafef")

# Cách 2: Custom config cho trang đã có hoặc site mới
custom_config = {
    "name": "My Custom News",
    "domain": "mynews.com",
    "rss_urls": ["https://mynews.com/feed.xml"],
    "sitemap_url": "https://mynews.com/sitemap.xml",
    "config": {
        "title_selector": {"class": "title"},
    }
}
# LƯU Ý QUAN TRỌNG: Để dùng custom config, bạn KHÔNG NÊN truyền site_name
# (hoặc phải truyền site_name="custom" và tự định nghĩa name, rss_urls, sitemap_url trong dict)
# Nếu truyền cùng lúc Crawler(site_name="cafef", custom_config=...), thư viện sẽ ưu tiên site_name
crawler = Crawler(custom_config=custom_config)
```

### Các Phương Thức

#### `get_articles_from_feed(limit_per_feed=20)`

Lấy tin từ RSS feed.

```python
import pandas as pd
from vnstock_news import Crawler

crawler = Crawler(site_name="vnexpress")  # Chọn báo có RSS
articles = crawler.get_articles_from_feed(limit_per_feed=10)  # Returns List[Dict]

# Convert to DataFrame nếu cần
df = pd.DataFrame(articles)
print(df.head())
```

**Output:**

```
                              url                    title  publish_time
0  https://cafef.vn/investment/...  Thị trường chứng khoán...  2025-01-15
1  https://cafef.vn/investment/...  Nhà đầu tư tiếp tục...  2025-01-15
```

**Parameters:**

- `limit_per_feed` (int): Lấy tối đa bao nhiêu bài từ mỗi RSS feed

**Returns:** `List[Dict[str, Any]]` với các key `[url, title, short_description, publish_time, author]`

---

#### `get_articles(sitemap_url=None, limit=10, limit_per_feed=None)`

Lấy metadata của bài viết từ RSS hoặc URL thô từ Sitemap (smart fallback).

> ⚠️ **Lưu Ý Quan Trọng về Sitemap:**
> Sitemap theo chuẩn XML chỉ chứa các thẻ `<loc>` (URL) và `<lastmod>` (thời gian cập nhật). Nếu `Crawler` fallback sang lấy tin từ Sitemap, kết quả trả về sẽ **KHÔNG có title, description, hay nội dung bài viết**, mà chỉ có `url` và `lastmod`. Để lấy nội dung chi tiết từ sitemap URL, hãy cân nhắc sử dụng `get_article_details(url)` hoặc chuyển sang dùng `BatchCrawler` / `AsyncBatchCrawler`.

```python
import pandas as pd

# Lấy 100 bài (ưu tiên RSS trước, nếu web không có RSS sẽ fallback dùng sitemap)
articles = crawler.get_articles(limit=100)  # Returns List[Dict]

# Hoặc chỉ định rõ sitemap URL
articles = crawler.get_articles(
    sitemap_url="https://cafef.vn/latest-news-sitemap.xml",
    limit=200
)

print(f"Lấy được {len(articles)} bài")
df = pd.DataFrame(articles)
print(df.head())
```

**Parameters:**

- `sitemap_url` (str or list, optional): Custom sitemap URL(s). Nếu không cung cấp, dùng RSS trước.
- `limit` (int): Tối đa bao nhiêu bài
- `limit_per_feed` (int, optional): Giới hạn mỗi feed khi có nhiều feeds

**Returns:** `List[Dict[str, Any]]`

---

## 2. Lưu Ý về RSS và Sitemap Classes

> ⚠️ **Lưu Ý:** Các class `RSS`, `Sitemap`, và `News` là internal modules và **không được export** trong `vnstock_news.__init__`.
>
> **Khuyến nghị:** Sử dụng `Crawler` class thay vì truy cập trực tiếp các internal modules.
>
> Nếu bạn thực sự cần, bạn có thể import trực tiếp:
>
> ```python
> from vnstock_news.core.rss import RSS
> from vnstock_news.core.sitemap import Sitemap
> from vnstock_news.core.news import News
> ```

---

## 3. BatchCrawler - Batch Processing (Đồng Bộ)

Lấy hàng loạt bài từ một trang báo, với hỗ trợ resume nếu bị lỗi.

### Khởi Tạo

```python
from vnstock_news import BatchCrawler

crawler = BatchCrawler(
    site_name="cafef",
    request_delay=1.5,      # Delay 1.5s giữa mỗi request
    output_path="./data",   # Lưu kết quả vào thư mục này
    debug=False             # Tắt debug output
)
```

### Phương Thức

#### `fetch_articles(sitemap_url=None, limit=10, top_n_per_feed=None, within=None)`

Tải chi tiết hàng loạt bài viết dựa trên danh sách meta links gom được từ sitemap. Tự động tránh tải lại URL đã tồn tại trong file `temp`.

```python
# Tải chi tiết 100 bài mới nhất dựa trên config sitemap mặc định của trang
articles = crawler.fetch_articles(limit=100)

# Tải 500 bài dựa trên một sitemap URL cụ thể
articles = crawler.fetch_articles(
    limit=500,
    sitemap_url="https://cafef.vn/latest-news-sitemap.xml"
)

print(f"✅ Lấy được chi tiết {len(articles)} bài")
print(articles.info())
```

**Parameters:**

- `sitemap_url` (str or list, optional): Truyền tham số này nếu không muốn dùng config mặc định
- `limit` (int): Tối đa lấy chi tiết bao nhiêu bài.
- `top_n_per_feed` (int, optional): Số tin giới hạn nếu trang có nhiều RSS feed riêng lẻ.
- `within` (str, optional): Bộ lọc biên độ thời gian (được dự định cho phiên bản tương lai).

**Returns:** `pd.DataFrame` chứa chi tiết nội dung phân tích (parsed) của các website.

---

#### `fetch_articles_batch(batch_size=10, max_workers=2)`

Lấy từng batch nhỏ.

```python
# Lấy theo batch 50 bài, 2 workers
batches = crawler.fetch_articles_batch(
    batch_size=50,
    max_workers=2
)

for i, batch in enumerate(batches):
    print(f"Batch {i}: {len(batch)} bài")
    batch.to_csv(f"batch_{i}.csv", index=False)
```

---

## 4. AsyncBatchCrawler - Batch Processing (Bất Đồng Bộ)

Lấy hàng loạt bài **nhanh hơn gấp 5-10 lần** bằng concurrent requests.

### Khởi Tạo

```python
from vnstock_news import AsyncBatchCrawler

crawler = AsyncBatchCrawler(
    site_name="cafef",
    max_concurrency=5       # Tối đa 5 requests cùng lúc
)
```

### Phương Thức

#### `fetch_articles_async(sources, top_n=10, top_n_per_feed=None, within=None)`

```python
import asyncio
from vnstock_news import AsyncBatchCrawler

async def main():
    crawler = AsyncBatchCrawler(site_name="cafef")

    articles = await crawler.fetch_articles_async(
        sources=["https://cafef.vn/latest-news-sitemap.xml"],  # Bắt buộc: sitemap URL
        top_n=100  # Lấy 100 bài
    )

    print(f"✅ Lấy được {len(articles)} bài")
    return articles

# Chạy
articles = asyncio.run(main())
articles.to_csv("cafef_async.csv", index=False)
```

**Output:**

```
✅ Lấy được 187 bài trong 45 days, 12:34:56
```

---

#### `fetch_articles_concurrently(urls, max_concurrency=5)`

Lấy content chi tiết từ danh sách URLs nhanh chóng.

```python
import asyncio

async def main():
    urls = [
        "https://cafef.vn/article1",
        "https://cafef.vn/article2",
        # ... nhiều URLs khác
    ]

    crawler = AsyncBatchCrawler(site_name="cafef")
    articles = await crawler.fetch_articles_concurrently(
        urls=urls,
        max_concurrency=10  # 10 requests cùng lúc
    )

    return articles

articles = asyncio.run(main())
```

---

## 5. EnhancedNewsCrawler - Advanced API

API nâng cao với đầy đủ tính năng: caching, validation, content cleaning.

### Khởi Tạo

```python
from vnstock_news import EnhancedNewsCrawler

crawler = EnhancedNewsCrawler(
    cache_enabled=True,           # Bật caching
    cache_ttl=7200,               # Cache 2 giờ
    max_concurrency=5,            # Tối đa 5 requests cùng lúc
    debug=False                   # Tắt debug output
)
```

### Phương Thức

#### `fetch_articles_async(sources, top_n=10, top_n_per_feed=None, within=None, sort_order='desc', save_to_file=False)`

Lấy bài với đầy đủ tính năng (caching, validation, cleaning).

```python
import asyncio
from vnstock_news import EnhancedNewsCrawler

async def main():
    crawler = EnhancedNewsCrawler(cache_enabled=True, cache_ttl=7200)

    articles = await crawler.fetch_articles_async(
        sources=["https://cafef.vn/latest-news-sitemap.xml"],
        top_n=100                  # Lấy 100 bài
    )

    print(f"✅ Lấy {len(articles)} bài (with caching & validation)")
    return articles

articles = asyncio.run(main())
```

**Features:**

- ✅ **Caching**: Lưu cache tránh lấy lại
- ✅ **Validation**: Kiểm tra dữ liệu hợp lệ
- ✅ **Content Cleaning**: Tự động xóa HTML, ký tự lạ
- ✅ **Retries**: Retry tự động nếu lỗi
- ✅ **Time Filtering**: Lọc theo khoảng thời gian

**Parameters:**

- `sources` (list): Danh sách URLs của sitemap/RSS
- `site_name` (str): Tên trang báo
- `max_articles` (int): Tối đa bao nhiêu bài
- `time_frame` (str): Khoảng thời gian ("24h", "7d", "30d", "all")
- `clean_content` (bool): Làm sạch content hay không

**Returns:** `pd.DataFrame`

---

## So Sánh Các Crawler

| Tiêu Chí             | Crawler | RSS        | Sitemap | BatchCrawler | AsyncBatchCrawler | Enhanced |
| -------------------- | ------- | ---------- | ------- | ------------ | ----------------- | -------- |
| **Tốc độ**           | ⭐⭐⭐  | ⭐⭐⭐⭐⭐ | ⭐⭐    | ⭐⭐         | ⭐⭐⭐⭐⭐        | ⭐⭐⭐⭐ |
| **Dữ liệu lịch sử**  | ❌      | ❌         | ✅      | ✅           | ✅                | ✅       |
| **Đơn giản**         | ✅      | ✅         | ✅      | ✅           | ❌                | ❌       |
| **Caching**          | ❌      | ❌         | ❌      | ❌           | ❌                | ✅       |
| **Content Cleaning** | ❌      | ❌         | ❌      | ❌           | ❌                | ✅       |
| **Async**            | ❌      | ❌         | ❌      | ❌           | ✅                | ✅       |
| **Resume Support**   | ❌      | ❌         | ❌      | ✅           | ⚠️                | ✅       |

---

## Ví Dụ Thực Tế

### Ví Dụ 1: Lấy Nhanh 20 Bài Mới Từ VnExpress

```python
from vnstock_news import Crawler
import pandas as pd

crawler = Crawler(site_name="vnexpress")
articles = crawler.get_articles_from_feed(limit_per_feed=20)  # List[Dict]

print(f"✅ Lấy {len(articles)} bài")
df = pd.DataFrame(articles)
print(df[['title', 'publish_time']].head())
```

### Ví Dụ 2: Lấy Chi Tiết 500 Bài Kể Từ Hiện Tại Từ CafeF

```python
from vnstock_news import BatchCrawler

crawler = BatchCrawler(
    site_name="cafef",
    request_delay=1.0,
    output_path="cafef_500_articles.csv"
)

articles = crawler.fetch_articles(limit=500)

print(f"✅ Đã tải và trích xuất chi tiết {len(articles)} bài")
# print(f"Từ {articles['publish_time'].min()} đến {articles['publish_time'].max()}")
```

### Ví Dụ 3: Lấy Từ 3 Báo Cùng Lúc (Nhanh)

```python
import asyncio
from vnstock_news import AsyncBatchCrawler, SITES_CONFIG
import pandas as pd

async def fetch_all_sites():
    sites = ["cafef", "tuoitre", "vietstock"]
    all_articles = []

    for site_name in sites:
        crawler = AsyncBatchCrawler(
            site_name=site_name,
            max_concurrency=5
        )

        config = SITES_CONFIG[site_name]
        sitemap_url = config.get("sitemap_url") or config.get("sitemap", {}).get("current_url")

        print(f"⏳ Đang lấy từ {site_name}...")
        articles = await crawler.fetch_articles_async(
            sources=[sitemap_url],
            top_n=100  # Lấy 100 bài
        )
        articles['source'] = site_name
        all_articles.append(articles)

    result = pd.concat(all_articles, ignore_index=True)
    return result

articles = asyncio.run(fetch_all_sites())
articles.to_csv("all_sites_articles.csv", index=False)

print(f"✅ Tổng {len(articles)} bài từ {articles['source'].nunique()} báo")
print(articles['source'].value_counts())
```

---

## Xử Lý Lỗi

### Retry Tự Động

```python
from vnstock_news import EnhancedNewsCrawler
import asyncio

crawler = EnhancedNewsCrawler(
    cache_enabled=True  # Tự động cache để tránh retry không cần thiết
)

async def safe_fetch():
    try:
        articles = await crawler.fetch_articles_async(
            sources=["https://cafef.vn/latest-news-sitemap.xml"],
            top_n=100
        )
        return articles
    except Exception as e:
        print(f"❌ Lỗi: {e}")
        return None

articles = asyncio.run(safe_fetch())
```

### Xử Lý Rate Limiting

```python
from vnstock_news import BatchCrawler
import time

crawler = BatchCrawler(
    site_name="cafef",
    request_delay=2.0  # Delay 2 giây giữa mỗi request
)

try:
    articles = crawler.fetch_articles(limit=1000)
except Exception as e:
    if "429" in str(e) or "Too Many Requests" in str(e):
        print("⚠️ Bị rate limit, đợi 1 giờ...")
        time.sleep(3600)
        # Retry
        articles = crawler.fetch_articles(limit=1000)
```

---

## Tips Tối Ưu

1. **Chọn crawler đúng**:
   - Chỉ cần tin mới → `Crawler.get_articles_from_feed()`
   - Cần lịch sử → `BatchCrawler.fetch_articles()`
   - Cần nhanh + lịch sử → `AsyncBatchCrawler`
   - Cần cache + validation → `EnhancedNewsCrawler`

2. **Điều chỉnh request_delay**:
   - Tăng delay nếu bị 429 (rate limit)
   - Giảm delay nếu mạng nhanh

3. **Sử dụng cache**:
   - Tiết kiệm bandwidth
   - Tăng tốc độ
   - Tránh load server

4. **Batch processing**:
   - Xử lý dữ liệu theo batch nhỏ
   - Lưu thường xuyên
   - Tránh mất dữ liệu
