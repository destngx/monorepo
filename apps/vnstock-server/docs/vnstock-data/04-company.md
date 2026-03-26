# Company - Thông Tin Công Ty, Cổ Đông, Ban Lãnh Đạo

Lớp `Company` cung cấp thông tin chi tiết về các công ty niêm yết, bao gồm hồ sơ, cổ đông, ban lãnh đạo và các sự kiện liên quan.

## Tổng Quan & So Sánh Nguồn Dữ Liệu

Package hỗ trợ 2 nguồn dữ liệu: **VCI** (Vietcap) và **KBS** (KB Securities). Dưới đây là bảng so sánh khả năng hỗ trợ của từng nguồn:

| Tính năng (Method)  | VCI | KBS | Mô tả                                                                                                                      |
| :------------------ | :-: | :-: | :------------------------------------------------------------------------------------------------------------------------- |
| **overview**        | ✅  | ✅  | Thông tin hồ sơ doanh nghiệp. VCI chi tiết hơn về cơ cấu cổ đông.                                                          |
| **shareholders**    | ✅  | ✅  | Danh sách cổ đông lớn.                                                                                                     |
| **officers**        | ✅  | ✅  | Ban lãnh đạo. VCI hỗ trợ lọc trạng thái (đương nhiệm/từ nhiệm).                                                            |
| **subsidiaries**    | ✅  | ✅  | Công ty con & liên kết.                                                                                                    |
| **news**            | ✅  | ✅  | Tin tức liên quan.                                                                                                         |
| **events**          | ✅  | ⚠️  | Sự kiện quyền (trả cổ tức, họp ĐHCĐ). KBS hiện tại trả về dữ liệu trống.                                                   |
| **capital_history** | ❌  | ✅  | Lịch sử tăng vốn. **Chỉ có trên KBS**.                                                                                     |
| **insider_trading** | ❌  | ⚠️  | Giao dịch nội bộ. **Chỉ có trên KBS** (tuy nhiên dữ liệu trả về tuỳ cổ phiếu, có thể trống nếu không có giao dịch nội bộ). |
| **ratio_summary**   | ✅  | ❌  | Tóm tắt tỷ số tài chính (P/E, ROE...). **Chỉ có trên VCI**.                                                                |
| **trading_stats**   | ✅  | ❌  | Thống kê giao dịch (NN mua/bán, Room). **Chỉ có trên VCI**.                                                                |

**Khuyến nghị**:

- Sử dụng **VCI** làm mặc định cho hầu hết nhu cầu phân tích cơ bản.
- Sử dụng **KBS** khi cần lấy **Lịch sử vốn (Capital History)**.

---

## Khởi Tạo

```python
from vnstock_data import Company

# Nguồn VCI
company_vci = Company(source="VCI", symbol="TCB")

# Nguồn KBS
company_kbs = Company(source="KBS", symbol="TCB")
```

---

## 1. overview() - Thông Tin Tổng Quan

### Nguồn VCI

```python
df = company_vci.overview()
```

| Cột (Column)                  | Kiểu Dữ Liệu | Mô Tả                            |
| :---------------------------- | :----------- | :------------------------------- |
| `symbol`                      | string       | Mã chứng khoán                   |
| `id`                          | string       | ID nội bộ                        |
| `issue_share`                 | int          | Số lượng cổ phiếu lưu hành       |
| `history`                     | string       | Lịch sử hình thành và phát triển |
| `company_profile`             | string       | Giới thiệu chung về công ty      |
| `icb_name3`                   | string       | Ngành cấp 3 (ICB)                |
| `icb_name2`                   | string       | Ngành cấp 2 (ICB)                |
| `icb_name4`                   | string       | Ngành cấp 4 (ICB)                |
| `financial_ratio_issue_share` | int          | SLCP dùng tính chỉ số tài chính  |
| `charter_capital`             | int          | Vốn điều lệ (VND)                |

### Nguồn KBS

```python
df = company_kbs.overview()
```

| Cột (Column)          | Kiểu Dữ Liệu | Mô Tả                           |
| :-------------------- | :----------- | :------------------------------ |
| `business_model`      | string       | Mô hình kinh doanh              |
| `symbol`              | string       | Mã chứng khoán                  |
| `founded_date`        | string       | Ngày thành lập                  |
| `charter_capital`     | int          | Vốn điều lệ                     |
| `number_of_employees` | int          | Số lượng nhân viên              |
| `listing_date`        | string       | Ngày niêm yết                   |
| `exchange`            | string       | Sàn giao dịch                   |
| `listing_price`       | int          | Giá chào sàn                    |
| `listed_volume`       | int          | Khối lượng niêm yết lần đầu     |
| `ceo_name`            | string       | Tên CEO                         |
| `ceo_position`        | string       | Chức vụ CEO                     |
| `inspector_name`      | string       | Trưởng Ban Kiểm Soát            |
| `address`             | string       | Địa chỉ trụ sở                  |
| `phone`               | string       | Số điện thoại                   |
| `email`               | string       | Email liên hệ                   |
| `website`             | string       | Website công ty                 |
| `history`             | string       | Lịch sử công ty                 |
| `outstanding_shares`  | int          | Số lượng cổ phiếu đang lưu hành |

---

## 2. shareholders() - Cổ Đông Lớn

### Nguồn VCI

```python
df = company_vci.shareholders()
```

| Cột (Column)        | Kiểu Dữ Liệu | Mô Tả                            |
| :------------------ | :----------- | :------------------------------- |
| `id`                | string       | ID cổ đông                       |
| `share_holder`      | string       | Tên cổ đông                      |
| `quantity`          | int          | Số lượng cổ phiếu nắm giữ        |
| `share_own_percent` | float        | Tỷ lệ sở hữu (ví dụ 0.15 là 15%) |
| `update_date`       | string       | Ngày cập nhật                    |

### Nguồn KBS

```python
df = company_kbs.shareholders()
```

| Cột (Column)           | Kiểu Dữ Liệu | Mô Tả             |
| :--------------------- | :----------- | :---------------- |
| `name`                 | string       | Tên cổ đông       |
| `update_date`          | string       | Ngày cập nhật     |
| `shares_owned`         | int          | Số lượng cổ phiếu |
| `ownership_percentage` | float        | Tỷ lệ sở hữu (%)  |

---

## 3. officers() - Ban Lãnh Đạo

### Nguồn VCI

Dữ liệu chi tiết, bao gồm cả số lượng cổ phiếu sở hữu.

```python
# filter_by='working' (đương nhiệm), 'resigned' (đã nghỉ), 'all'
df = company_vci.officers(filter_by='working')
```

| Cột (Column)          | Kiểu Dữ Liệu | Mô Tả                   |
| :-------------------- | :----------- | :---------------------- |
| `id`                  | string       | ID lãnh đạo             |
| `officer_name`        | string       | Tên lãnh đạo            |
| `officer_position`    | string       | Chức vụ                 |
| `position_short_name` | string       | Chức vụ viết tắt        |
| `update_date`         | string       | Ngày cập nhật           |
| `officer_own_percent` | float        | Tỷ lệ sở hữu cổ phần    |
| `quantity`            | int          | Số lượng cổ phần sở hữu |

### Nguồn KBS

Danh sách cơ bản.

```python
df = company_kbs.officers()
```

| Cột (Column)  | Kiểu Dữ Liệu | Mô Tả                |
| :------------ | :----------- | :------------------- |
| `from_date`   | string       | Năm bắt đầu/bổ nhiệm |
| `position`    | string       | Chức vụ              |
| `name`        | string       | Tên lãnh đạo         |
| `position_en` | string       | Chức vụ (Tiếng Anh)  |
| `owner_code`  | string       | Mã chức vụ           |

---

## 4. subsidiaries() - Công Ty Con & Liên Kết

### Nguồn VCI

```python
df = company_vci.subsidiaries()
```

| Cột (Column)        | Kiểu Dữ Liệu | Mô Tả                                  |
| :------------------ | :----------- | :------------------------------------- |
| `sub_organ_code`    | string       | Mã công ty con                         |
| `organ_name`        | string       | Tên công ty con                        |
| `ownership_percent` | float        | Tỷ lệ sở hữu                           |
| `type`              | string       | Loại hình ('công ty con' / 'liên kết') |

### Nguồn KBS

```python
df = company_kbs.subsidiaries()
```

| Cột (Column)        | Kiểu Dữ Liệu | Mô Tả            |
| :------------------ | :----------- | :--------------- |
| `name`              | string       | Tên công ty con  |
| `charter_capital`   | int          | Vốn điều lệ      |
| `ownership_percent` | float        | Tỷ lệ sở hữu (%) |
| `type`              | string       | Loại hình        |

---

## 5. news() - Tin Tức

### Nguồn VCI

```python
df = company_vci.news()
```

| Cột (Column)         | Kiểu Dữ Liệu | Mô Tả                    |
| :------------------- | :----------- | :----------------------- |
| `news_title`         | string       | Tiêu đề tin              |
| `news_image_url`     | string       | Link ảnh minh họa        |
| `news_source_link`   | string       | Link nguồn bài viết      |
| `public_date`        | int          | Ngày công bố (timestamp) |
| `news_short_content` | string       | Tóm tắt                  |
| `news_full_content`  | string       | Nội dung đầy đủ (HTML)   |

### Nguồn KBS

```python
df = company_kbs.news()
```

| Cột (Column)   | Kiểu Dữ Liệu | Mô Tả                              |
| :------------- | :----------- | :--------------------------------- |
| `title`        | string       | Tiêu đề tin                        |
| `publish_time` | string       | Thời gian công bố                  |
| `url`          | string       | Đường dẫn chi tiết (relative path) |
| `article_id`   | int          | ID bài viết                        |

---

## 6. events() - Sự Kiện Quyền

### Nguồn VCI (Khuyên dùng)

VCI cung cấp dữ liệu sự kiện chi tiết.

```python
df = company_vci.events()
```

| Cột (Column)      | Kiểu Dữ Liệu | Mô Tả                            |
| :---------------- | :----------- | :------------------------------- |
| `event_title`     | string       | Tên sự kiện                      |
| `public_date`     | string       | Ngày công bố                     |
| `exright_date`    | string       | Ngày giao dịch không hưởng quyền |
| `record_date`     | string       | Ngày đăng ký cuối cùng           |
| `value`           | int          | Giá trị (cổ tức tiền mặt, v.v.)  |
| `ratio`           | float        | Tỷ lệ thực hiện                  |
| `event_list_name` | string       | Loại sự kiện                     |

### Nguồn KBS

> ⚠️ Hiện tại chưa ghi nhận dữ liệu sự kiện từ API KBS cho các mã VN30 phổ biến. API có thể trả về DataFrame rỗng.

---

## 7. capital_history() - Lịch Sử Tăng Vốn (Chỉ KBS)

Chỉ hỗ trợ trên nguồn **KBS**.

```python
df = company_kbs.capital_history()
```

| Cột (Column)      | Kiểu Dữ Liệu | Mô Tả                    |
| :---------------- | :----------- | :----------------------- |
| `date`            | datetime     | Ngày thay đổi            |
| `charter_capital` | int          | Vốn điều lệ sau thay đổi |
| `currency`        | string       | Đơn vị tiền tệ           |

---

## 8. insider_trading() - Giao Dịch Nội Bộ (Chỉ KBS)

Chỉ hỗ trợ trên nguồn **KBS**.

> ⚠️ Hiện tại chưa ghi nhận dữ liệu từ API KBS cho các mã VN30 phổ biến.

---

## 9. ratio_summary() - Chỉ Số Tài Chính (Chỉ VCI)

Chỉ hỗ trợ trên nguồn **VCI**. Cung cấp ảnh chụp nhanh các chỉ số quan trọng.

```python
df = company_vci.ratio_summary()
```

| Cột (Column)        | Kiểu Dữ Liệu | Mô Tả                                                                |
| :------------------ | :----------- | :------------------------------------------------------------------- |
| `pe`                | float        | P/E                                                                  |
| `pb`                | float        | P/B                                                                  |
| `roe`               | float        | ROE                                                                  |
| `roa`               | float        | ROA                                                                  |
| `eps`               | float        | EPS cơ bản                                                           |
| `revenue_growth`    | float        | Tăng trưởng doanh thu                                                |
| `net_profit_growth` | float        | Tăng trưởng lợi nhuận                                                |
| `dividend`          | int          | Cổ tức tiền mặt gần nhất                                             |
| `net_profit_margin` | float        | Biên lợi nhuận ròng                                                  |
| `debt_equity`       | float        | Tỷ số Nợ/Vốn chủ sở hữu (vắng mặt trong một số trường hợp, xem `de`) |

---

## 10. trading_stats() - Thống Kê Giao Dịch (Chỉ VCI)

Chỉ hỗ trợ trên nguồn **VCI**.

```python
df = company_vci.trading_stats()
```

| Cột (Column)            | Kiểu Dữ Liệu | Mô Tả                   |
| :---------------------- | :----------- | :---------------------- |
| `open`                  | int          | Giá mở cửa              |
| `high`                  | int          | Giá cao nhất            |
| `low`                   | int          | Giá thấp nhất           |
| `close_price`           | int          | Giá đóng cửa            |
| `total_volume`          | int          | Tổng khối lượng         |
| `foreign_volume`        | int          | Khối lượng nước ngoài   |
| `foreign_room`          | int          | Room nước ngoài         |
| `current_holding_ratio` | float        | Tỷ lệ sở hữu nước ngoài |
