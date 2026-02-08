# Syncrules 部署指南

## 建置

### 從原始碼建置

```bash
# 安裝 Go 1.24+
# 參考 https://go.dev/doc/install

# 克隆專案
git clone https://github.com/Ning0612/Syncrules.git
cd Syncrules

# 下載相依
make deps

# 建置 CLI
make build

# 建置產出位於 dist/syncrules（Linux/macOS）或 dist/syncrules.exe（Windows）
```

### 跨平台建置

Go 原生支援跨平台編譯。設定 `GOOS` 與 `GOARCH` 環境變數即可：

```bash
# Windows (64-bit)
GOOS=windows GOARCH=amd64 go build -o dist/syncrules.exe ./cmd/syncrules

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o dist/syncrules-darwin ./cmd/syncrules

# Linux (64-bit)
GOOS=linux GOARCH=amd64 go build -o dist/syncrules-linux ./cmd/syncrules
```

### 嵌入版本資訊

Makefile 自動嵌入 Git 版本資訊：

```bash
make build
# 等同於：
go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" \
    -o dist/syncrules ./cmd/syncrules
```

驗證：
```bash
./dist/syncrules version
# 輸出：
# syncrules v0.1.0-dev
#   commit: abc1234
#   built:  2026-02-07T12:00:00Z
```

---

## 安裝

### 方式一：直接複製二進位檔

將建置好的二進位檔複製到 `PATH` 中的目錄：

```bash
# Linux / macOS
sudo cp dist/syncrules /usr/local/bin/

# Windows
# 將 dist\syncrules.exe 複製到 PATH 中的目錄
# 例如 C:\Users\<user>\bin\ 或加入環境變數
```

### 方式二：go install

```bash
go install github.com/Ning0612/Syncrules/cmd/syncrules@latest
# 安裝到 $GOPATH/bin/syncrules
```

### 方式三：Makefile install

```bash
make install
# 安裝到 $GOPATH/bin/syncrules
```

---

## 設定

### 設定檔位置

Syncrules 依以下順序搜尋設定檔：

1. `./config.yaml`（目前目錄）
2. `./configs/config.yaml`
3. `~/.config/syncrules/config.yaml`（Linux/macOS）或 `%APPDATA%/syncrules/config.yaml`（Windows）
4. `~/.syncrules/config.yaml`

也可透過 `--config` 旗標指定路徑：

```bash
syncrules sync --config /path/to/my-config.yaml
```

### 建立設定檔

```bash
# 複製範例設定
mkdir -p ~/.config/syncrules
cp configs/example.yaml ~/.config/syncrules/config.yaml

# 編輯設定
vi ~/.config/syncrules/config.yaml
```

### 最小設定範例（本地同步）

```yaml
transports:
  - name: local
    type: local

endpoints:
  - name: source
    transport: local
    root: /home/user/documents

  - name: backup
    transport: local
    root: /mnt/backup/documents

rules:
  - name: backup-documents
    mode: one-way-push
    source: source
    target: backup
    conflict: keep_newest
```

---

## Google Drive 設定

### 第一步：建立 Google Cloud 專案

1. 前往 [Google Cloud Console](https://console.cloud.google.com/)
2. 建立新專案或選擇現有專案
3. 啟用 **Google Drive API**：
   - 進入「API 和服務」→「程式庫」
   - 搜尋「Google Drive API」→ 點擊「啟用」

### 第二步：建立 OAuth 2.0 憑證

1. 進入「API 和服務」→「憑證」
2. 點擊「建立憑證」→「OAuth 用戶端 ID」
3. 應用程式類型：選擇「**桌面應用程式**」（Desktop app / Installed application）
4. 不需設定 Redirect URI（桌面應用程式使用手動授權碼流程）
5. 記下 `Client ID` 和 `Client Secret`

### 第三步：執行認證

```bash
syncrules auth gdrive \
    --client-id "YOUR_CLIENT_ID.apps.googleusercontent.com" \
    --client-secret "YOUR_CLIENT_SECRET"
```

程式會：
1. 印出授權 URL
2. 等待你在瀏覽器中完成授權
3. 貼回授權碼
4. 將 Token 儲存到 `~/.config/syncrules/gdrive-token.json`

### 第四步：設定含 Google Drive 的同步

```yaml
transports:
  - name: local
    type: local
  - name: gdrive
    type: gdrive
    config:
      client_id: "YOUR_CLIENT_ID.apps.googleusercontent.com"
      client_secret: "YOUR_CLIENT_SECRET"
      token_path: "~/.config/syncrules/gdrive-token.json"

endpoints:
  - name: local-config
    transport: local
    root: ~/.config/myapp
  - name: cloud-backup
    transport: gdrive
    root: /SyncRules/config-backup

rules:
  - name: backup-to-cloud
    mode: one-way-push
    source: local-config
    target: cloud-backup
    conflict: keep_local
```

---

## 自動化排程

### 方式一：使用內建 Daemon（推薦）

Syncrules 提供內建的排程守護程序，支援規則層級的排程配置：

```yaml
# 設定檔範例
scheduler:
  enabled: true
  default_interval: "1h"  # 全域預設間隔

rules:
  - name: critical-backup
    source: important-docs
    target: backup
    schedule:
      enabled: true
      interval: "5m"  # 每 5 分鐘同步

  - name: archive-sync
    source: archives
    target: cloud-backup
    schedule:
      enabled: true
      interval: "6h"  # 每 6 小時同步
```

#### 啟動守護程序

```bash
# 背景執行（推薦）
syncrules daemon start --detach

# 檢查狀態
syncrules daemon status

# 停止
syncrules daemon stop

# 前景模式（用於測試）
syncrules daemon start --interval 1m
```

#### 系統服務整合

**systemd (Linux)**：

```ini
# /etc/systemd/system/syncrules.service
[Unit]
Description=Syncrules Daemon
After=network.target

[Service]
Type=forking
User=your-user
ExecStart=/usr/local/bin/syncrules daemon start --detach
ExecStop=/usr/local/bin/syncrules daemon stop
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

啟用自動啟動：
```bash
sudo systemctl enable syncrules
sudo systemctl start syncrules
sudo systemctl status syncrules
```

**launchd (macOS)**：

```xml
<!-- ~/Library/LaunchAgents/com.syncrules.daemon.plist -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" 
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.syncrules.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/syncrules</string>
        <string>daemon</string>
        <string>start</string>
        <string>--detach</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

載入服務：
```bash
launchctl load ~/Library/LaunchAgents/com.syncrules.daemon.plist
launchctl start com.syncrules.daemon
```

更多詳情請參閱 [Daemon 使用指南](daemon-usage.md)。

---

### 方式二：使用系統排程器（傳統方式）

如果您不想使用內建 daemon，也可以使用系統排程器：

#### Linux / macOS (cron)

```bash
# 每小時執行一次同步
crontab -e

# 加入：
0 * * * * /usr/local/bin/syncrules sync --config ~/.config/syncrules/config.yaml 2>&1 >> ~/.config/syncrules/sync.log
```

#### Windows (Task Scheduler)

1. 開啟「工作排程器」
2. 建立基本工作
3. 觸發程序：依需要設定（每小時/每天）
4. 動作：啟動程式
   - 程式/指令碼：`C:\path\to\syncrules.exe`
   - 引數：`sync --config C:\Users\user\.config\syncrules\config.yaml`

---

## 故障排除

### 常見問題

#### 1. Lock 檔案殘留

如果同步意外中斷，可能留下鎖檔：

```bash
# 檢視鎖狀態
ls ~/.config/syncrules/.syncrules.lock

# 如果確定沒有其他同步在執行，刪除鎖檔
rm ~/.config/syncrules/.syncrules.lock
```

Syncrules 會自動偵測並清除以下陳舊鎖：
- **同主機**：鎖持有者 PID 已不存在（PID 存活則鎖不會被清除，無論時間多久）
- **跨主機**：無法檢查 PID，超過 30 分鐘自動視為陳舊

> **警告**：手動刪除鎖檔前，請務必確認沒有其他同步程序正在執行。如果持有者仍在運行，刪除鎖檔可能導致多個同步操作同時執行。

#### 2. Daemon 無法啟動

```bash
# 檢查配置是否啟用 scheduler
syncrules daemon status

# 確認配置檔中有：
# scheduler:
#   enabled: true
#   default_interval: "5m"
```

#### 3. Google Drive Token 過期

```bash
# 重新認證
syncrules auth gdrive --client-id YOUR_ID --client-secret YOUR_SECRET
```

Syncrules 會自動使用 Refresh Token 更新過期的 Access Token。如果 Refresh Token 也過期，需要重新認證。

#### 4. 權限問題

確認：
- 本地目錄有讀寫權限
- Google Drive API 已啟用
- OAuth scope 包含檔案存取權限

#### 5. 路徑不存在

本地 Adapter 要求 root 路徑在建立時已存在。如果目錄不存在：

```bash
mkdir -p /path/to/sync/root
```

---

## 安全性建議

1. **Token 保護**：確保 `gdrive-token.json` 的權限為 `0600`
2. **不提交機密**：`.gitignore` 已排除 `*-token.json` 和 `.env`
3. **最小權限**：Google Drive API 使用 `drive.file` scope（僅存取應用建立的檔案）
4. **設定檔敏感資料**：`configs/*.yaml` 已被 `.gitignore` 排除，僅保留 `example.yaml`
5. **PID 檔案安全**：Daemon PID 檔案儲存在使用者目錄，防止權限問題
