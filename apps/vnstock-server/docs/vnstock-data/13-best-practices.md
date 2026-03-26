# Best Practices & Tips cho Vnstock_Data

## 1. Cấu Trúc Code Tối Ưu

### Pattern 1: Factory Function cho Khởi Tạo ✅ Verified

```python
from vnstock_data import Quote, Company, Finance, Listing

def get_stock_data(symbol, data_types=['quote', 'company', 'finance']):
    """
    Factory function để lấy nhiều loại dữ liệu cùng lúc

    Args:
        symbol: Mã cổ phiếu
        data_types: Danh sách loại dữ liệu cần lấy

    Returns:
        dict: Từ điển chứa các object đã khởi tạo
    """
    objects = {}

    if 'quote' in data_types:
        objects['quote'] = Quote(source="vnd", symbol=symbol)
    if 'company' in data_types:
        objects['company'] = Company(source="vci", symbol=symbol)
    if 'finance' in data_types:
        objects['finance'] = Finance(source="vci", symbol=symbol)
    if 'listing' in data_types:
        objects['listing'] = Listing(source="vci")

    return objects

# Sử dụng
data = get_stock_data('VCB', ['quote', 'finance'])
df_price = data['quote'].history(start="2024-01-01", end="2024-12-31")
df_finance = data['finance'].balance_sheet()

# Output (verified 2025-12-02):
# Quote data shape: (250, 6) - 250 trading days
# Finance data shape: (43, 100) - 43 reports, 100 fields
```

**Ưu điểm**:

- ✅ Tái sử dụng code
- ✅ Dễ đọc, dễ bảo trì
- ✅ Flexible - thêm/bớt data types dễ dàng

### Pattern 2: Caching Layer

```python
import pandas as pd
import os
from functools import wraps
from vnstock_data import Quote

CACHE_DIR = './data_cache'

def ensure_cache_dir():
    os.makedirs(CACHE_DIR, exist_ok=True)

def cached_quote(ttl_days=30):
    """Decorator để cache dữ liệu Quote"""
    def decorator(func):
        def wrapper(symbol, start, end, *args, **kwargs):
            cache_file = f"{CACHE_DIR}/{symbol}_{start}_{end}.csv"

            # Kiểm tra cache
            if os.path.exists(cache_file):
                age_days = (pd.Timestamp.now() - pd.Timestamp(os.path.getmtime(cache_file))) / pd.Timedelta(days=1)
                if age_days < ttl_days:
                    print(f"📦 Load từ cache: {cache_file}")
                    return pd.read_csv(cache_file)

            # Lấy từ API
            print(f"🌐 Lấy từ API: {symbol}")
            df = func(symbol, start, end, *args, **kwargs)

            # Lưu cache
            ensure_cache_dir()
            df.to_csv(cache_file, index=False)
            return df

        return wrapper
    return decorator

@cached_quote(ttl_days=1)
def get_quote_history(symbol, start, end):
    quote = Quote(source="vnd", symbol=symbol)
    return quote.history(start=start, end=end, interval="1D")

# Sử dụng
df = get_quote_history('VCB', '2024-01-01', '2024-12-31')
```

## 2. Error Handling Patterns

### Pattern 1: Multi-source Fallback ✅ Verified

```python
from vnstock_data import Quote
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def get_quote_with_fallback(symbol, start, end, sources=['vnd', 'vci']):
    """
    Thử lấy dữ liệu từ nhiều nguồn
    """
    for source in sources:
        try:
            logger.info(f"Trying {source} for {symbol}...")
            quote = Quote(source=source, symbol=symbol)
            df = quote.history(start=start, end=end, interval="1D")

            if df.empty:
                logger.warning(f"{source} returned empty DataFrame")
                continue

            logger.info(f"✅ Success from {source}")
            return df, source

        except Exception as e:
            logger.error(f"❌ {source} failed: {e}")
            continue

    raise Exception(f"All sources {sources} failed")

# Sử dụng
df, source_used = get_quote_with_fallback('VCB', '2024-11-01', '2024-12-02')

# Output (verified 2025-12-02):
# Trying vnd for VCB...
#   ✅ Success from vnd
# Data shape: (22, 6)
# Last 2 rows:
#          time   close    open    high     low   volume
# 20 2024-11-29  61.959  61.494  61.959  61.428  1010500
# 21 2024-12-02  62.557  63.022  63.154  62.357  1573100
```

**Lợi ích**:

- ✅ Tự động fallback nếu API chính down
- ✅ Log chi tiết mỗi attempt
- ✅ Tăng độ tin cậy của application

### Pattern 2: Retry Logic

```python
from vnstock_data import Quote
import time
from functools import wraps

def retry(max_attempts=3, delay_seconds=1):
    """Decorator để retry khi lỗi"""
    def decorator(func):
        def wrapper(*args, **kwargs):
            for attempt in range(max_attempts):
                try:
                    return func(*args, **kwargs)
                except Exception as e:
                    if attempt == max_attempts - 1:
                        raise
                    print(f"Lần thử {attempt + 1} thất bại: {e}")
                    time.sleep(delay_seconds * (2 ** attempt))  # Exponential backoff

        return wrapper
    return decorator

@retry(max_attempts=3, delay_seconds=2)
def get_quote_data(symbol):
    quote = Quote(source="vnd", symbol=symbol)
    return quote.history(start="2024-01-01", end="2024-12-31")

# Sử dụng
try:
    df = get_quote_data('VCB')
except Exception as e:
    print(f"Tất cả lần thử đều thất bại: {e}")
```

## 3. Data Validation & Cleaning ✅ Verified

### Kiểm Tra Dữ Liệu

```python
from vnstock_data import Quote
import pandas as pd

def validate_quote_data(df):
    """Kiểm tra chất lượng dữ liệu Quote"""
    issues = []

    # Kiểm tra cột cần có
    required_cols = ['time', 'open', 'high', 'low', 'close', 'volume']
    missing_cols = [col for col in required_cols if col not in df.columns]
    if missing_cols:
        issues.append(f"Missing columns: {missing_cols}")

    # Kiểm tra dòng rỗng
    if df.isnull().sum().sum() > 0:
        null_info = df.isnull().sum()
        issues.append(f"NULL values: {null_info[null_info > 0].to_dict()}")

    # Kiểm tra giá logic
    invalid_high_low = df[df['high'] < df['low']]
    if not invalid_high_low.empty:
        issues.append(f"High < Low in {len(invalid_high_low)} rows")

    invalid_ohlc = df[~((df['open'] <= df['high']) & (df['low'] <= df['close']))]
    if not invalid_ohlc.empty:
        issues.append(f"Invalid OHLC in {len(invalid_ohlc)} rows")

    # Kiểm tra volume
    if (df['volume'] < 0).any():
        issues.append(f"Negative volume in {(df['volume'] < 0).sum()} rows")

    return issues

# Sử dụng
quote = Quote(source="vnd", symbol="VCB")
df = quote.history(start="2024-11-01", end="2024-12-02")

issues = validate_quote_data(df)
if issues:
    print("⚠️ Issues detected:")
    for issue in issues:
        print(f"  - {issue}")
else:
    print("✅ Data is valid!")
    print(f"Rows: {len(df)}, Columns: {list(df.columns)}")

# Output (verified 2025-12-02):
# ✅ Data is valid!
# Rows: 22, Columns: ['time', 'close', 'open', 'high', 'low', 'volume']
```

**Lợi ích**:

- ✅ Phát hiện dữ liệu hỏng trước khi xử lý
- ✅ Tránh lỗi trong phân tích
- ✅ Đảm bảo chất lượng backtest
