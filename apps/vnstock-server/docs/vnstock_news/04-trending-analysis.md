# 04. Phân Tích Xu Hướng & Keyword

> ⚠️ **LƯU Ý QUAN TRỌNG:** TrendingAnalyzer hiện **KHÔNG được export** trong package chính và API thực tế **KHÁC** với tài liệu này.
>
> Tài liệu này chỉ mang tính **THAM KHẢO** cho tính năng sẽ có trong tương lai hoặc bạn cần tự implement dựa trên các công cụ phân tích text khác.

---

## 1. TrendingAnalyzer - Khái Niệm & API Thực Tế

### TrendingAnalyzer Là Gì?

`TrendingAnalyzer` là công cụ nội bộ để phân tích xu hướng tin tức dựa trên n-grams.

### ⚠️ API Thực Tế (Khác với tài liệu cũ)

**Import (không phải từ package chính):**

```python
# KHÔNG thể: from vnstock_news import TrendingAnalyzer
# Phải dùng:
from vnstock_news.trending.analyzer import TrendingAnalyzer
```

**Khởi tạo:**

```python
analyzer = TrendingAnalyzer(
    stop_words_file="path/to/stopwords.txt",  # Optional
    min_token_length=3                         # Minimum token length
)
```

**Parameters:**

- `stop_words_file` (str, optional): Đường dẫn đến file stopwords
- `min_token_length` (int): Độ dài tối thiểu của token (default: 3)

### ⚠️ Các Method Thực Tế

#### 1. `update_trends(text, ngram_range=None)`

Cập nhật trending counter với text mới.

```python
analyzer = TrendingAnalyzer()

# Cập nhật từ nhiều text
texts = ["Chứng khoán tăng mạnh", "Thị trường chứng khoán hôm nay"]

for text in texts:
    analyzer.update_trends(text, ngram_range=[2, 3, 4, 5])
```

#### 2. `get_top_trends(top_n=20)`

Lấy top trending phrases.

```python
# Sau khi update_trends
top_trends = analyzer.get_top_trends(top_n=10)

print(top_trends)
# {'chứng khoán tăng': 5, 'thị trường chứng': 3, ...}
```

#### 3. `reset_trends()`

Reset counter về 0.

```python
analyzer.reset_trends()
```

---

## 2. Các Method KHÔNG Tồn Tại ❌

Những method này **KHÔNG có** trong mã nguồn thực tế:

- ❌ `extract_keywords(texts, top_n)` - KHÔNG TỒN TẠI
- ❌ `extract_topics(articles_df, content_column, top_n)` - KHÔNG TỒN TẠI
- ❌ `get_trending(articles_df, time_window, top_n)` - KHÔNG TỒN TẠI
- ❌ `analyze_sentiment(texts)` - KHÔNG TỒN TẠI

---

## 3. Workflow Thực Tế (Dùng API Có Sẵn)

---

## 3. Workflow Thực Tế (Dùng API Có Sẵn)

### Ví Dụ: Phân Tích Trending Với API Thật

```python
from vnstock_news.trending.analyzer import TrendingAnalyzer
from vnstock_news import Crawler
import pandas as pd

# Bước 1: Lấy tin tức
crawler = Crawler(site_name="vnexpress")
articles = crawler.get_articles_from_feed(limit_per_feed=30)

# Bước 2: Khởi tạo analyzer
analyzer = TrendingAnalyzer(min_token_length=3)

# Bước 3: Cập nhật trends từng text
for article in articles:
    title = article.get('title', '')
    description = article.get('description', '')
    full_text = f"{title} {description}"

    # Update với n-grams 2,3,4,5
    analyzer.update_trends(full_text, ngram_range=[2, 3, 4, 5])

# Bước 4: Lấy top trending
top_trends = analyzer.get_top_trends(top_n=20)

print("🔥 Top Trending Phrases:")
for i, (phrase, count) in enumerate(top_trends.items(), 1):
    print(f"{i:2d}. {phrase:30s} - {count:3d} lần")
```

**Output:**

```
🔥 Top Trending Phrases:
 1. chứng khoán tăng             -  15 lần
 2. thị trường hôm                -  12 lần
 3. nhà đầu tư                    -  10 lần
...
```

---

## 4. Giải Pháp Thay Thế Cho Keyword Extraction

Vì API thực tế khác với tài liệu cũ, đây là cách tự implement keyword extraction:

### Option 1: Dùng Collections.Counter

```python
from collections import Counter
from vnstock_news import Crawler
import re
import pandas as pd

# Lấy tin tức
crawler = Crawler(site_name="vnexpress")
articles = crawler.get_articles_from_feed(limit_per_feed=50)

# Extract tất cả từ
all_words = []
for article in articles:
    title = article.get('title', '')
    # Tách từ đơn giản
    words = re.findall(r'\w+', title.lower())
    # Lọc từ có ít nhất 3 ký tự
    words = [w for w in words if len(w) >= 3]
    all_words.extend(words)

# Đếm tần suất
word_freq = Counter(all_words)

# Top 20 keywords
top_keywords = dict(word_freq.most_common(20))

print("🔥 Top Keywords:")
for word, count in top_keywords.items():
    print(f"{word:20s}: {count}")
```

### Option 2: Dùng Pandas

```python
from vnstock_news import Crawler
import pandas as pd
import re

crawler = Crawler(site_name="vnexpress")
articles = crawler.get_articles_from_feed(limit_per_feed=50)

# Convert to DataFrame
df = pd.DataFrame(articles)

# Extract keywords từ title
def extract_words(text):
    if not text:
        return []
    words = re.findall(r'\w+', text.lower())
    return [w for w in words if len(w) >= 3]

df['keywords'] = df['title'].apply(extract_words)

# Flatten và đếm
all_keywords = [word for keywords in df['keywords'] for word in keywords]
keyword_counts = pd.Series(all_keywords).value_counts().head(20)

print(keyword_counts)
```

---

## 5. Best Practices & Tips

---

## 5. Best Practices & Tips

1. **Sử dụng TrendingAnalyzer thực tế:**

   ```python
   from vnstock_news.trending.analyzer import TrendingAnalyzer  # Đúng
   # KHÔNG: from vnstock_news import TrendingAnalyzer
   ```

2. **Update trends từng text một:**

   ```python
   for text in texts:
       analyzer.update_trends(text, ngram_range=[2, 3, 4])
   ```

3. **Reset sau mỗi phiên phân tích:**

   ```python
   analyzer.reset_trends()  # Reset counter
   ```

4. **Load Vietnamese stopwords:**
   ```python
   import os
   stopwords_path = os.path.join(
       os.path.dirname(vnstock_news.__file__),
       'config', 'vietnamese-stopwords.txt'
   )
   analyzer = TrendingAnalyzer(stop_words_file=stopwords_path)
   ```

---

## 6. Roadmap & Tương Lai

Module TrendingAnalyzer đang được phát triển. Các tính năng có thể có trong tương lai:

- ✅ N-gram phrase extraction (đã có)
- ⏳ Simple keyword extraction method
- ⏳ Time-based trending analysis
- ⏳ Sentiment analysis
- ⏳ Multi-source comparison
- ⏳ Visualization helpers

---

## 7. Kết Luận

**Tóm tắt:**

- ⚠️ TrendingAnalyzer KHÔNG được export trong package chính
- ⚠️ API thực tế: `update_trends()`, `get_top_trends()`, `reset_trends()`
- ✅ Bạn có thể tự implement keyword extraction với `Counter` hoặc `pandas`
- ✅ Hoặc dùng thư viện khác như `sklearn.feature_extraction.text.CountVectorizer`

**Khuyến nghị:**

- Sử dụng `collections.Counter` cho keyword extraction đơn giản
- Dùng TrendingAnalyzer thực tế cho n-gram phrase extraction
- Tham khảo ví dụ trong section 4 để implement tính năng tương tự
