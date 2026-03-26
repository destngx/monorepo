# Vnstock_Data - Tài Liệu Chi Tiết cho Context7 MCP

Đây là bộ tài liệu toàn diện về thư viện `vnstock_data`, được thiết kế đặc biệt để sử dụng với Context7 MCP (Model Context Protocol) để giúp AI Agents viết code chính xác và hiệu quả.

## 📚 Cấu Trúc Tài Liệu

### Phần Cơ Bản (Getting Started)

1. **[01-overview.md](01-overview.md)** - Tổng quan thư viện, các loại dữ liệu, cách sử dụng cơ bản

### Phần Chính (Core Modules)

2. **[02-listing.md](02-listing.md)** - `Listing`: Danh sách cổ phiếu, phân ngành, chỉ số
3. **[03-quote.md](03-quote.md)** - `Quote`: Lịch sử giá, OHLCV, intraday, price depth
4. **[04-company.md](04-company.md)** - `Company`: Thông tin công ty, cổ đông, ban lãnh đạo
5. **[05-finance.md](05-finance.md)** - `Finance`: Báo cáo tài chính, chỉ số, kế hoạch
6. **[06-trading.md](06-trading.md)** - `Trading`: Giao dịch, bảng giá, thống kê, nước ngoài
7. **[07-market.md](07-market.md)** - `Market`: Định giá thị trường (P/E, P/B)
8. **[08-insights.md](08-insights.md)** - `Insights/TopStock`: Top cổ phiếu (gainer, loser, etc)
9. **[09-macro.md](09-macro.md)** - `Macro`: Kinh tế vĩ mô (GDP, CPI, FDI, tỷ giá)
10. **[10-commodity.md](10-commodity.md)** - `Commodity`: Giá hàng hóa (vàng, dầu, lợn, etc)
11. **[11-fund.md](11-fund.md)** - `Fund`: Dữ liệu quỹ ETF, chứng chỉ quỹ

### Phần Nâng Cao (Advanced)

12. **[12-data-sources.md](12-data-sources.md)** - Ma trận hỗ trợ nguồn dữ liệu, so sánh VCI vs VND vs MAS
13. **[13-best-practices.md](13-best-practices.md)** - Best practices, patterns, optimization tips

## 🎯 Cách Sử Dụng Tài Liệu Này

### Cho AI Agents / Context7 MCP

Tài liệu này được thiết kế tối ưu để AI Agents nạp vào context:

1. **Toàn bộ tài liệu**: Nạp tất cả file `.md` vào context để AI có hiểu biết đầy đủ
2. **Theo chủ đề**: Nạp từng phần dành cho từng module cụ thể
3. **Ma trận nguồn**: Luôn nạp `13-data-sources.md` để tránh lỗi "method not supported"

### Cho Người Dùng

- **Beginner**: Đọc 01 (Overview) → 02 (Listing) → 03 (Quote)
- **Intermediate**: Tiếp tục 04-11 tùy theo nhu cầu
- **Advanced**: Đọc 12-13 để hiểu chi tiết

## ⚙️ Cài Đặt Vnstock

Các gói thư viện vnstock_data được cài đặt **chung** thông qua chương trình cài đặt của Vnstock. Để cài đặt và kích hoạt vnstock_data, vui lòng tham khảo hướng dẫn chi tiết tại:

**🔗 [Hướng Dẫn Cài Đặt Vnstock](https://vnstocks.com/onboard-member)**

## 🔍 Danh Sách API Nhanh

### Listing

```python
listing.all_symbols()
listing.symbols_by_industries(industry="Ngân hàng")
listing.symbols_by_exchange(exchange="HOSE")
listing.symbols_by_group(group="VN30")
```

### Quote

```python
quote.history(start="2024-01-01", end="2024-12-31", interval="1D")
quote.intraday()
quote.price_depth()
```

### Company

```python
company.overview()
company.shareholders()
company.officers()
company.subsidiaries()
company.news()
company.events()
company.trading_stats()
```

### Finance

```python
fin.balance_sheet(lang="vi")
fin.income_statement(lang="vi")
fin.cash_flow(lang="vi")
fin.ratio(lang="vi")
fin.annual_plan(lang="vi")  # MAS only
```

### Trading

```python
trading.price_board(symbol_list=[...])
trading.price_history(start="2024-01-01", end="2024-12-31")
trading.foreign_trade(start="2024-01-01", end="2024-12-31")
trading.prop_trade(start="2024-01-01", end="2024-12-31")
trading.insider_deal()
trading.order_stats(start="2024-01-01", end="2024-12-31")
```

### Market

```python
market.pe()
market.pb()
market.evaluation()
```

### Macro

```python
macro.gdp()
macro.cpi()
macro.exchange_rate()
macro.fdi()
macro.money_supply()
```

### Commodity

```python
commodity.gold_vn()
commodity.oil_crude()
commodity.steel_hrc()
commodity.pork_north_vn()
```

## ⚠️ Những Lỗi Phổ Biến Cần Tránh

### Lỗi 1: Method Not Supported

```python
# ❌ SAI: VND không support method này
listing = Listing(source="vnd")
listing.industry()  # Error!

# ✅ ĐÚNG: VCI support đầy đủ
listing = Listing(source="vci")
listing.industry()
```

**Giải Pháp**: Kiểm tra file **[12-data-sources.md](12-data-sources.md)** để xác nhận nguồn hỗ trợ.

### Lỗi 2: Format Ngày Không Đúng

```python
# ❌ SAI
df = quote.history(start="01/01/2024", end="31/12/2024")

# ✅ ĐÚNG
df = quote.history(start="2024-01-01", end="2024-12-31")
```

### Lỗi 3: Dữ Liệu Quá Cũ/Lâu

```python
# ❌ SAI: Yêu cầu 5 năm dữ liệu phút (max 3 năm)
df = quote.history(start="2019-01-01", end="2024-12-31", interval="1m")

# ✅ ĐÚNG
df = quote.history(start="2022-01-01", end="2024-12-31", interval="1m")
```

### Lỗi 4: Không Kiểm Tra Dữ Liệu Rỗng

```python
# ❌ SAI
df = quote.history(start="2100-01-01", end="2100-12-31")
df['MA20'] = df['close'].rolling(20).mean()  # Error!

# ✅ ĐÚNG
df = quote.history(start="2024-01-01", end="2024-12-31")
if df.empty:
    print("Không có dữ liệu!")
else:
    df['MA20'] = df['close'].rolling(20).mean()
```

> **Tìm hiểu thêm**: Xem **[13-best-practices.md](13-best-practices.md)** để biết chi tiết hơn về các mẫu, gợi ý, và xử lý lỗi.

## 📊 Use Cases Phổ Biến

### Phân Tích Kỹ Thuật

→ Sử dụng: Quote (history, intraday) + Price Depth

### Fundamental Analysis

→ Sử dụng: Listing + Company + Finance + Trading

### Stock Screening

→ Sử dụng: Listing + Quote + Company

### Market Analysis

→ Sử dụng: Market (P/E, P/B) + TopStock + Macro

### Sector Analysis

→ Sử dụng: Listing (by industries) + Quote + Finance

## 🚀 Tips Tối Ưu

**Hiệu Suất**: Dùng **VCI** cho danh sách đầy đủ, dùng **VND** cho Quote nếu cần tốc độ cao, cache dữ liệu để tránh gọi API nhiều lần.

**Chất Lượng Dữ Liệu**: Luôn kiểm tra dữ liệu trống (`.empty`), validate dữ liệu (high >= low, OHLC hợp lệ), so sánh từ nhiều nguồn khi cần đảm bảo độ chính xác.

**Xử Lý Lỗi**: Implement retry logic cho API calls, sử dụng try-except và log chi tiết, fallback sang nguồn khác nếu một nguồn thất bại.

> **Chi tiết nâng cao**: Xem **[13-best-practices.md](13-best-practices.md)** để tìm các patterns tối ưu, caching layer, error handling chi tiết, và data validation.

## 📝 Template Nhanh

### Template 1: Lấy Giá Lịch Sử

```python
from vnstock_data import Quote

quote = Quote(source="vnd", symbol="VCB")
df = quote.history(start="2024-01-01", end="2024-12-31", interval="1D")
print(df[['time', 'close']].head())
```

### Template 2: Lấy BCTC

```python
from vnstock_data import Finance

fin = Finance(source="vci", symbol="VCB", period="year")
df_bs = fin.balance_sheet(lang="vi")
df_ic = fin.income_statement(lang="vi")
df_cf = fin.cash_flow(lang="vi")
```

### Template 3: Screening

```python
from vnstock_data import Listing, Quote

listing = Listing(source="vci")
all_stocks = listing.all_symbols()
hose = all_stocks[all_stocks['exchange'] == 'HOSE']

# Lấy giá cho mỗi cổ phiếu
for symbol in hose['symbol'].head(10):
    quote = Quote(source="vnd", symbol=symbol)
    df = quote.history(start="2024-11-01", end="2024-11-30")
    price = df['close'].iloc[-1]
    print(f"{symbol}: {price}")
```

### Template 4: Kinh Tế Vĩ Mô

```python
from vnstock_data import Macro

macro = Macro(source="mbk")
df_gdp = macro.gdp()
df_cpi = macro.cpi()
df_fdi = macro.fdi()
```

## 📄 Lưu Ý Bản Quyền

- Tất cả dữ liệu được cung cấp cho **mục đích nghiên cứu, giáo dục, sử dụng cá nhân**
- **Không** sử dụng dữ liệu này cho mục đích thương mại mà không có sự cho phép
- Vnstock không chịu trách nhiệm về bất kỳ tổn thất phát sinh từ việc sử dụng dữ liệu

---

**Version**: 1.0  
**Cập nhật lần cuối**: December 2025  
**Tương thích với**: vnstock_data >= 2.3.0
