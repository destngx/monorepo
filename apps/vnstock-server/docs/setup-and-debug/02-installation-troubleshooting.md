# 02 - Cài Đặt & Gỡ Lỗi (Installation & Troubleshooting)

Tài liệu này tập trung vào việc giải quyết các vấn đề "nhức đầu" nhất khi cài đặt, đặc biệt là trên Windows hoặc các môi trường server/container.

## 1. Phương Pháp Cài Đặt (GUI vs CLI)

Chúng tôi cung cấp 2 công cụ cài đặt phù hợp với từng nhu cầu:

### A. Vnstock Installer CLI (Server/Cloud/DevOps hoặc cho máy macOS)

Dành cho môi trường không có giao diện (Headless) hoặc người thích dùng terminal.

- **Ưu điểm**: Nhanh, nhẹ, dễ script automation.
- **Cách dùng**:

  Tải file với wget (Linux/Mac)

  ```bash
  wget https://vnstocks.com/files/vnstock-cli-installer.run
  ```

  Hoặc tải với curl (Linux/Mac)

  ```bash
  curl -O https://vnstocks.com/files/vnstock-cli-installer.run
  ```

  Nếu môi trường không có wget/curl, bạn có thể tải file thủ công từ [vnstocks.com/files/vnstock-cli-installer.run](https://vnstocks.com/files/vnstock-cli-installer.run).

  Sau khi tải file, chạy lệnh sau:

  ```bash
  chmod +x vnstock-cli-installer.run
  ./vnstock-cli-installer.run
  ```

  #### 🤖 Cài Đặt Tự Động (Non-Interactive / CI/CD)

  Dành cho các hệ thống tự động (CI/CD, Github Actions) hoặc máy chủ không có giao diện tương tác.

  **Cách 1: One-Liner (Khuyên Dùng)**
  Tải, cài đặt, và xác thực chỉ với 1 dòng lệnh:

  ```bash
  wget -q https://vnstocks.com/files/vnstock-cli-installer.run -O installer.run && chmod +x installer.run && echo "2" | ./installer.run --quiet --accept -- --api-key "API_KEY_CỦA_BẠN"
  ```

  **Cách 2: Command Line Arguments**

  ```bash
  ./vnstock-cli-installer.run -- --non-interactive --api-key "API_KEY_CỦA_BẠN"
  ```

  _Các tham số hỗ trợ:_
  - `--api-key`: Nhập API key trực tiếp.
  - `--non-interactive`: Tắt chế độ hỏi đáp (prompt).
  - `--quiet`: Chế độ im lặng (ít output).
  - `--accept`: Tự động đồng ý các điều khoản.
  - `--language`: `vi` (Tiếng Việt) hoặc `en` (Tiếng Anh).

  **Cách 3: Biến Môi Trường (Environment Variables)**

  ```bash
  export VNSTOCK_API_KEY="api_key_cua_ban"
  export VNSTOCK_INTERACTIVE=0
  export VNSTOCK_LANGUAGE=2
  ./vnstock-cli-installer.run
  ```

  #### 🐳 Cài Đặt Trong Docker

  Bạn có thể dùng Dockerfile mẫu đã được tối ưu cho Vnstock (chạy tốt trên Huggingface Spaces, cần tuỳ biến lại cho phù hợp yêu cầu của bạn):
  - **Dockerfile Mẫu**: [Tải tại đây](https://vnstocks.com/files/Dockerfile)

### B. Vnstock Installer GUI (Desktop Users)

Dành cho người dùng cá nhân trên Windows/Mac.

- **Ưu điểm**: Giao diện trực quan, click-and-run, tự động check môi trường.
- **Lưu ý**: Cần môi trường desktop để hiển thị cửa sổ cài đặt.

```bash
pip install --extra-index-url https://vnstocks.com/api/simple vnstock_installer
```

Kích hoạt giao diện cài đặt:

```bash
vnstock-installer
```

Hoặc gọi chương trình được cài vào 1 phiên bản python cụ thể, ví dụ 3.14 nếu máy bạn cùng lúc cài nhiều phiên bản và lệnh `vnstock-installer` bị xung đột không thể chạy.

```bash
python3.14 -m vnstock_installer
```

## 2. Quản Lý Thư Viện & Dependencies

_(Bỏ qua bước này nếu bạn đã dùng Installer ở trên và chỉ phải sử dụng nếu quá trình cài đặt gặp lỗi thiếu gói phụ thuộc)_

### 🚀 Quản Lý Với `uv` (Khuyên Dùng)

`uv` là công cụ thay thế cho `pip`, nhanh hơn từ 10-100 lần.

1.  **Cài đặt uv** (Chỉ 1 lần):
    ```bash
    pip install uv
    ```
2.  **Tạo Virtual Environment**:
    ```bash
    uv venv
    ```
3.  **Cài gói thư viện**:
    ```bash
    uv pip install vnstock
    ```
4.  **Cài từ URL Requirements** (Siêu nhanh):
    ```bash
    uv pip install -r https://vnstocks.com/files/requirements.txt
    ```

### 🐢 Quản Lý Với `pip` (Cơ Bản)

Nếu bạn chưa quen với `uv`, hãy dùng `pip` truyền thống.

1.  **Kiểm tra gói đã cài**:
    ```bash
    pip list
    ```
2.  **Cài đặt từ requirements.txt**:
    ```bash
    pip install -r https://vnstocks.com/files/requirements.txt
    ```

## 3. Các Lỗi Cài Đặt Phổ Biến (Common Issues)

### 🔴 Lỗi 1: "No module named pip"

Lỗi này thường xảy ra khi bạn dùng `uv` hoặc một số bản python rút gọn.

- **Nguyên nhân**: Môi trường python thiếu trình quản lý gói `pip` mặc định.
- **Cách khắc phục**:

  ```bash
  # Cài đặt pip thủ công (Linux/Mac)
  python -m ensurepip --upgrade

  # Nếu dùng uv
  uv pip install vnstock
  ```

### 🔴 Lỗi 2: Thiếu thư viện `vnii` (System Error)

- **Dấu hiệu**: Báo lỗi import liên quan đến `vnii` hoặc xung đột phiên bản.
- **Cách khắc phục**:
  ```bash
  pip install --extra-index-url https://vnstocks.com/api/simple vnii
  ```

### 🔴 Lỗi 3: Windows thiếu Visual C++ (Build Failed)

Khi cài `vnstock_pipeline` hoặc `vnstock_ta`, tiến trình bị treo hoặc báo lỗi build wheel.

- **Nguyên nhân**: Thiếu bộ biên dịch C++.
- **Cách khắc phục**:
  1.  Tải **Microsoft Visual C++ Redistributable (v14)** mới nhất cho kiến trúc máy (x64 hoặc x86). Link: [Microsoft VC++ Downloads](https://learn.microsoft.com/en-us/cpp/windows/latest-supported-vc-redist?view=msvc-170)
  2.  Cài đặt file `.exe`.
  3.  Chạy lại lệnh cài đặt Vnstock.

### 🔴 Lỗi 4: Lỗi Xác Thực / License (Authentication Error)

Bạn đã đăng ký nhưng hệ thống vẫn báo lỗi license hoặc không nhận diện được gói cước.

**Giải pháp 1: Cài đặt lại sạch sẽ (Clean Install)**
Đây là giải pháp triệt để nhất.

1.  **Gỡ cài đặt cũ**:
    ```bash
    pip uninstall vnstock vnai vnii vnstock_data vnstock_ta vnstock_news vnstock_pipeline vnstock_installer -y
    ```
2.  **Xóa file cấu hình cũ**:
    - **Mac/Linux**: `rm -rf ~/.vnstock`
    - **Windows (CMD)**: `rmdir /S /Q %USERPROFILE%\.vnstock`
    - **Windows (PowerShell)**: `rm -r -force "$env:USERPROFILE\.vnstock"`
3.  **Cài lại trình cài đặt mới nhất**:
    ```bash
    pip install --upgrade --extra-index-url https://vnstocks.com/api/simple vnstock-installer
    vnstock-installer
    ```

**Giải pháp 2: Tạo file cấu hình thủ công (Manual Config)**
Nếu cài lại không được, bạn có thể tạo file `api_key.json` thủ công.

1.  Tạo thư mục `~/.vnstock/` (Mac/Linux) hoặc `%USERPROFILE%\.vnstock\` (Windows).
2.  Tạo file `api_key.json` với nội dung:
    ```json
    {
      "api_key": "YOUR_ACTUAL_API_KEY"
    }
    ```
    _(Lấy API Key tại vnstocks.com/account)_.

### 🔴 Lỗi 5: Google Colab không nhận thư viện

- **Nguyên nhân**: Colab cần reset runtime để load lại các thư viện C++ extension vừa cài.
- **Cách khắc phục**: Vào menu **Runtime** > **Restart Session** (hoặc Ctrl + M .).

### 🔴 Lỗi 6: "vnai is required for device identification"

- **Dấu hiệu**: Báo lỗi `No module named 'vnai'` dù đã chạy installer.
- **Nguyên nhân**: Gói `vnai` chưa được kích hoạt đúng trong môi trường ảo.
- **Cách khắc phục**:
  1.  Kích hoạt lại môi trường ảo: `source ~/.vnstock/bin/activate`
  2.  Chạy lại installer: `./vnstock-cli-installer.run`
  3.  Hoặc cài thủ công: `pip install vnai -U`

### 🔴 Lỗi 7: Nhập sai API Key

Nếu lỡ nhập sai API Key lúc cài đặt, bạn có thể reset như sau:

- **Cách 1 (Xóa Clean)**:
  ```bash
  rm -rf ~/.vnstock
  ./vnstock-cli-installer.run
  ```
- **Cách 2 (Ghi đè bằng biến môi trường)**:
  ```bash
  export VNSTOCK_API_KEY="KEY_DUNG_CUA_BAN"
  ./vnstock-cli-installer.run --non-interactive
  ```

### 🔴 Lỗi 8: Sai phiên bản Python trong Môi trường ảo (Version Mismatch)

**Tình huống**: Bạn cài môi trường ảo với Python 3.14 (qua vnstock installer) nhưng khi chạy lệnh `python` lại ra version 3.11 của hệ thống.

**Giải pháp**:

1.  **Kiểm tra phiên bản thực tế**:
    - Mac/Linux: `which python3.14`
    - Windows: `Get-Command python3.14 | Select-Object Source`
2.  **Gọi chính xác phiên bản**: Luôn dùng lệnh cụ thể thay vì lệnh chung chung.
    ```bash
    python3.14 your_script.py
    # Thay vì: python your_script.py
    ```
3.  **Tạo Alias (Biệt danh)**:
    - **Mac/Linux** (`~/.zshrc`):
      ```bash
      alias python='python3.14'
      ```
    - **Windows Powershell**:
      ```powershell
      Set-Alias -Name python -Value C:\Python314\python.exe -Option AllScope -Scope CurrentUser -Force
      ```
4.  **Kiểm tra lại**: `python3.14 --version`

> [!TIP]
> Cách tốt nhất là luôn kích hoạt môi trường ảo (`source ~/.venv/bin/activate`) trước khi chạy bất kỳ lệnh nào. Khi đã activate, lệnh `python` sẽ tự động trỏ đúng vào phiên bản của môi trường ảo.

### 🔴 Lỗi 9: "error: externally-managed-environment" trên Linux/macOS

Lỗi này xảy ra khi bạn cố gắng cài đặt thư viện (như `vnstock_installer`) bằng `pip` vào môi trường Python global của hệ thống trên một số bản phân phối Linux hoặc macOS mới.

- **Dấu hiệu**: Báo lỗi `error: externally-managed-environment` khi chạy lệnh cài đặt:

  ```text
  error: externally-managed-environment

  × This environment is externally managed
  ╰─> To install Python packages system-wide, try apt install
      python3-xyz, where xyz is the package you are trying to
      install.
  ```

- **Nguyên nhân**: Hệ điều hành khóa môi trường Python mặc định để tránh việc `pip` cài gói trực tiếp làm xung đột các package của hệ thống.
- **Cách khắc phục**: Cần tạo một môi trường ảo (virtual environment) riêng cho dự án, rồi mới chạy lệnh cài đặt:

  ```bash
  # 1. Tạo môi trường ảo có tên .venv trong thư mục hiện tại
  python3 -m venv .venv

  # 2. Kích hoạt môi trường ảo
  source .venv/bin/activate

  # 3. Chạy lệnh cài đặt lại vnstock_installer
  pip install --extra-index-url https://vnstocks.com/api/simple vnstock_installer
  ```

### 🔴 Lỗi 10: "Cài đặt thất bại" trên GUI Installer

Khi sử dụng trình cài đặt giao diện đồ họa (`vnstock-installer`), bạn có thể gặp thông báo "Cài đặt thất bại".

- **Dấu hiệu**: Cửa sổ trình duyệt hoặc ứng dụng báo lỗi đỏ, trong khi Terminal/Command Prompt báo thiếu thư viện (ví dụ: `No module named 'pandas'`).
  ```text
  2026-03-15 23:04:43,298 - vnstock_installer - INFO - Vnstock Installer v3.1.1 starting...
  Opening in Chrome...
  2026-03-15 23:05:25,863 - vnstock_installer.ui - ERROR - Failed to import vnai after bootstrap: No module named 'pandas'
  ```
- **Nguyên nhân**: Môi trường hiện tại thiếu các gói thư viện phụ thuộc (dependencies) cần thiết để khởi chạy bộ công cụ.
- **Cách khắc phục**:
  1.  **Đóng trình cài đặt**: Tắt cửa sổ trình duyệt/giao diện đang hiển thị hoặc nhấn `Ctrl + C` tại Terminal để dừng chương trình.
  2.  **Cài đặt môi trường**: Chạy lệnh sau để cài đặt đầy đủ các gói phụ thuộc:
      ```bash
      pip install -r https://vnstocks.com/files/requirements.txt
      ```
  3.  **Chạy lại installer**: Sau khi cài xong, nhấn phím mũi tên lên `↑` trong Terminal để nạp lại lệnh và nhấn `Enter` để khởi động lại giao diện cài đặt:
      ```bash
      vnstock-installer
      ```

## 4. Xác Minh Cài Đặt (Verification)

Sau khi cài xong, làm sao biết đã đủ chưa?

### Kiểm tra gói cài đặt

Dựa trên gói Sponsor của bạn, số lượng package sẽ khác nhau:

- **Free**: Chỉ có `vnstock`.
- **Bronze**: Thêm `vnstock-data`.
- **Silver**: Thêm `vnstock-news`, `vnstock-ta`.
- **Golden**: Thêm `vnstock-pipeline`.

### Verify Script

Chạy lệnh python sau để kiểm tra nhanh:

```python
try:
    import vnstock
    print(f"✅ Vnstock version: {vnstock.__version__}")
except ImportError:
    print("❌ Vnstock chưa được cài đặt.")

# Kiểm tra các gói khác tương tự
packages = ['vnstock_data', 'vnstock_news', 'vnstock_ta', 'vnstock_pipeline']
for pkg in packages:
    try:
        module = __import__(pkg)
        print(f"✅ {pkg} INSTALLED")
    except ImportError:
        print(f"⚪ {pkg} not installed (Check your sponsorship tier)")
```

### 🔴 Lỗi 6: Windows Store Conflict (Alias Error)

Khi gõ `python` trên Windows Terminal, nó mở ra Microsoft Store thay vì chạy python hoặc khi cài bộ thư viện sponsor với chương trình Vnstock Installer - luôn báo thiếu các gói phụ thuộc dù đã cài đặt với requirements.txt.

- **Dấu hiệu**: Cài đặt báo thành công nhưng khi import lại báo `ModuleNotFoundError`.
- **Nguyên nhân**: Windows ưu tiên alias của App Installer hơn path của Python thật.
- **Cách khắc phục triệt để**:
  1.  Mở menu **Start**, tìm kiếm và chọn **Apps & features** (Ứng dụng & tính năng).
  2.  Nhấn vào dòng chữ nhỏ **App execution aliases** (Biệt danh thực thi ứng dụng).
  3.  Tìm các mục có tên **Python** (App Installer, python.exe, python3.exe) và gạt công tắc sang **OFF** (Tắt) để vô hiệu hóa alias của Windows Store.

## 5. Cập Nhật (Update)

Hệ sinh thái Vnstock cập nhật rất thường xuyên. Để đảm bảo tính ổn định cao nhất, quy trình cập nhật chuẩn là **Gỡ bỏ sạch sẽ và cài lại**.

### Quy trình cập nhật chuẩn:

**1. Gỡ cài đặt toàn bộ gói cũ:**

```bash
pip uninstall vnstock vnai vnii vnstock_installer vnstock_chart vnstock_data vnstock_ta vnstock_news vnstock_pipeline -y
```

**2. Cài lại thư viện lõi (Core) công khai từ PyPI:**

```bash
pip install vnstock vnai
```

**3. Cài lại thư viện tiện ích (System) công khai từ Vnstocks Index:**

```bash
pip install --extra-index-url https://vnstocks.com/api/simple vnii vnstock_installer vnstock_chart
```

**4. Cài lại thư viện nâng cao phân phối riêng trong gói Sponsor (`data`, `ta`, `news`, `pipeline`) dùng Installer:**

- **Cách 1 (GUI)**: Chạy lệnh `vnstock-installer`
- **Cách 2 (CLI)**: Tải và chạy `vnstock-cli-installer.run` như sau:

```bash
wget https://vnstocks.com/files/vnstock-cli-installer.run
chmod +x vnstock-cli-installer.run
./vnstock-cli-installer.run
```

## 6. Câu hỏi thường gặp (FAQ)

### ❓ "Tôi đã tài trợ rồi nhưng sao nhập API Key vẫn báo dùng phiên bản miễn phí?"

Đây là câu hỏi khá thường gặp từ các bạn mới tham gia gói tài trợ của Vnstock.

**Dấu hiệu nhận biết:** Khi chạy code, bạn thấy thông báo xác nhận đã lưu API Key nhưng hệ thống vẫn báo đang dùng bản Community:

```text
✓ API key đã được lưu thành công! (API key saved successfully!)
Bạn đang sử dụng Phiên bản cộng đồng (60 requests/phút)
(You are using Community version - 60 requests/minute)

Để tham gia gói thành viên tài trợ (To join sponsor membership):
  Truy cập: https://vnstocks.com/insiders-program
✓ API key đã được lưu thành công! vnst***1234b
✓ Bạn đang sử dụng Phiên bản cộng đồng (60 requests/phút)
```

**Nguyên nhân gốc rễ:** Nhiều người dùng lầm tưởng rằng chỉ cần nhập API Key thì thư viện `vnstock` đang dùng sẽ tự động "nâng cấp" lên bản tài trợ. Tuy nhiên, thực tế là:

1.  Vnstock có 2 thư viện riêng biệt: `vnstock` (Miễn phí) và `vnstock_data` (Sponsor).
2.  Bản miễn phí `vnstock` sẽ luôn báo "Community version" bất kể bạn có API Key hay chưa (API Key ở bản này chủ yếu để chuẩn bị cho các đợt cập nhật tương lai).

**Cách khắc phục:** Bạn cần cài đặt thư viện dành riêng cho nhà tài trợ thông qua `vnstock-installer` (GUI hoặc CLI), sau đó đổi tên thư viện trong câu lệnh import từ `vnstock` thành `vnstock_data`.

```python
# ❌ Sai: Vẫn dùng bản miễn phí dù đã lưu API Key
from vnstock import Listing, Quote

# ✅ Đúng: Chuyển sang dùng bản tài trợ (sau khi đã cài vnstock_data)
from vnstock_data import Listing, Quote
```

Về cơ bản, bạn chỉ cần một thao tác **Find & Replace**: tìm `vnstock` và thay thế bằng `vnstock_data` trong dự án của mình.

> [!NOTE]
> Trong tương lai, `vnstock_data` có thể được tích hợp trực tiếp như một add-on của `vnstock` để tiện sử dụng hơn khi cấu trúc schema dữ liệu chuẩn hoá thế hệ tiếp theo được giới thiệu.
