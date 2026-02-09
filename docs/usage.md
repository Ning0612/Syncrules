# Syncrules 使用手冊

## 概述

Syncrules 是一個跨電腦、跨路徑的設定檔與文件同步工具。透過 YAML 設定檔定義同步規則，支援本地檔案系統與 Google Drive 之間的同步。

---

## 命令列介面

### 全域結構

```
syncrules [command] [flags]
```

### 可用命令

| 命令 | 說明 |
|------|------|
| `sync` | 執行同步規則 |
| `auth` | 雲端儲存認證 |
| `version` | 顯示版本資訊 |

---

## sync 命令

執行設定檔中定義的同步規則。

### 語法

```bash
syncrules sync [flags]
```

### 旗標

| 旗標 | 縮寫 | 預設值 | 說明 |
|------|------|--------|------|
| `--config` | `-c` | （自動搜尋） | 設定檔路徑 |
| `--rule` | | （全部） | 僅執行指定名稱的規則 |
| `--dry-run` | | `false` | 預覽模式，不實際執行 |
| `--output` | `-o` | `table` | 輸出格式：`table` 或 `json` |
| `--progress` | `-p` | `false` | 顯示傳輸進度（僅在非 dry-run 且有檔案複製時生效） |
| `--log-level` | | `info` | 日誌層級：`debug`, `info`, `warn`, `error`（不區分大小寫） |
| `--log-format` | | `text` | 日誌格式：`text` 或 `json` |
| `--log-file-only`| | `false` | 僅輸出到日誌檔案，不輸出到終端 |

### 使用範例

```bash
# 執行所有啟用（enabled: true）的規則（未指定 --rule 時的預設行為）
syncrules sync

# 指定設定檔
syncrules sync --config ~/my-sync.yaml

# 僅執行指定規則
syncrules sync --rule backup-to-cloud

# 預覽（不實際變更）
syncrules sync --dry-run

# JSON 輸出（適合腳本處理）
syncrules sync --dry-run -o json

# 顯示進度
syncrules sync --progress

# 組合使用
syncrules sync --config configs/sync.yaml --rule sync-docs --dry-run -o json
```

### 輸出說明

#### Table 格式

```
=== Rule: sync-home-to-work ===
Mode: two-way | home-config -> work-config

Action    Path              Size     Reason
------    ----              ----     ------
[COPY]    settings.json     1.2 KB   file modified
[COPY] [<-] notes.md       3.4 KB   file only exists on target
[MKDIR]   backups/          -        directory does not exist
[DEL]     old-config.yaml   512 B    file does not exist on source
[CONFLICT] data.db          2.1 MB   manual resolution required

Summary:
  Files to copy:   2 (4.6 KB)
  Files to delete: 1
  Dirs to create:  1
  Conflicts:       1 (require manual resolution)
```

- `[COPY]`：複製檔案
- `[COPY] [<-]`：反向複製（target → source，出現在 two-way 模式）
- `[MKDIR]`：建立目錄
- `[DEL]`：刪除檔案
- `[CONFLICT]`：衝突，需手動處理
- `[SKIP]`：跳過（依衝突策略決定）

#### JSON 格式

```json
{
  "RuleName": "sync-home-to-work",
  "Actions": [
    {
      "Type": "copy",
      "Direction": 0,
      "Path": "settings.json",
      "SourceInfo": {
        "Path": "settings.json",
        "Type": 0,
        "Size": 1234,
        "ModTime": "2026-02-07T10:00:00Z"
      },
      "Reason": "file modified"
    }
  ],
  "Stats": {
    "TotalFiles": 5,
    "FilesToCopy": 2,
    "FilesToDelete": 1,
    "DirsToCreate": 1,
    "Conflicts": 1,
    "BytesToSync": 4710
  }
}
```

---

## auth 命令

管理雲端儲存的認證。

### Google Drive 認證

```bash
syncrules auth gdrive --client-id YOUR_ID --client-secret YOUR_SECRET [--token-path PATH]
```

| 旗標 | 必要 | 說明 |
|------|------|------|
| `--client-id` | 是 | Google OAuth2 Client ID |
| `--client-secret` | 是 | Google OAuth2 Client Secret |
| `--token-path` | 否 | Token 儲存路徑（預設 `~/.config/syncrules/gdrive-token.json`） |

### 認證流程

1. 執行命令後，終端會顯示授權 URL
2. 在瀏覽器中開啟該 URL
3. 使用 Google 帳號登入並授權
4. 複製授權碼貼回終端
5. Token 自動儲存

```
$ syncrules auth gdrive --client-id xxx --client-secret yyy
Starting Google Drive authentication...
Token will be saved to: C:\Users\user\AppData\Roaming\syncrules\gdrive-token.json

To authorize Syncrules to access Google Drive:

1. Visit this URL:
   https://accounts.google.com/o/oauth2/auth?...

2. Sign in with your Google account and authorize the application

3. Copy the authorization code and paste it below

Enter authorization code: 4/0Axxxxxxxxx

Authentication successful! Token saved.
```

---

## version 命令

```bash
syncrules version
```

輸出：
```
syncrules v0.1.0-dev
  commit: abc1234
  built:  2026-02-07T12:00:00Z
```

---

## 設定檔格式

### 完整結構

```yaml
# Transport — 定義儲存後端
transports:
  - name: <唯一名稱>
    type: local | gdrive
    config:              # gdrive 專用設定
      client_id: "..."
      client_secret: "..."
      token_path: "..."

# Endpoint — 定義具體位置
endpoints:
  - name: <唯一名稱>
    transport: <transport 名稱>
    root: <根路徑>

# Rule — 定義同步關係
rules:
  - name: <唯一名稱>
    mode: one-way-push | one-way-pull | two-way
    source: <endpoint 名稱>
    target: <endpoint 名稱>
    ignore:              # 選用，glob 模式
      - "*.log"
      - ".cache/"
    conflict: keep_local | keep_remote | keep_newest | manual  # 預設 manual
    enabled: true | false   # 預設 true

# Logging — 定義日誌配置（選用）
logging:
  level: "info"          # debug | info | warn | "error"
  format: "text"         # text | json
  file:
    enabled: true        # 啟用檔案日誌
    path: "path/to.log"  # 日誌檔案路徑
    max_size: 100        # 單個檔案最大 MB
    max_age: 30          # 保留天數
    max_backups: 5       # 保留備份數量
    compress: true       # 壓縮舊日誌
```

### 概念階層

```
Transport（儲存後端類型）
  └── Endpoint（具體位置）
        └── Rule（同步關係）
              ├── Source Endpoint
              └── Target Endpoint
```

---

## 日誌配置

Syncrules 提供結構化日誌（Structured Logging）功能，支援多個輸出目標與 PII 自動脫敏。

### 配置參數

| 參數 | 預設值 | 說明 |
|------|--------|------|
| `level` | `info` | 日誌層級（`debug`, `info`, `warn`, `error`）。不區分大小寫，`warning` 為 `warn` 的別名。 |
| `format` | `text` | 輸出格式（`text` 或 `json`）。JSON 格式適合 ELK 或 CloudWatch 等監控系統。 |
| `file.enabled` | `false` | 是否將日誌寫入檔案。 |
| `file.path` | （自動） | 日誌路徑。支援 `~` 與環境變數。預設：`~/.config/syncrules/logs/syncrules.log`。 |
| `file.max_size` | `100` | 日誌檔案輪轉前的最大大小（MB）。 |
| `file.max_age` | `30` | 舊日誌保留天數。`0` 表示無限期保留。 |
| `file.max_backups` | `5` | 保留的舊日誌備份數量。`0` 表示無限期保留。 |
| `file.compress` | `false` | 是否使用 Gzip 壓縮輪轉後的舊日誌。 |

### PII 自動脫敏

Syncrules 會自動識別並遮蔽日誌中的敏感資訊，包括：
- 密碼（`pwd=...`）
- 認證 Token（`token=...`, `bearer ...`）
- API Key（`api_key=...`）
- Windows 路徑中的使用者名稱（例如 `C:\Users\***\Documents`）
- UNC 路徑中的伺服器與共享名稱
- Email 地址

> **限制說明**：僅對結構化參數中的「敏感 Key」（如 `password`, `token`）對應的 Value，或符合正則表達式的內容進行遮蔽。請避免將敏感資訊直接放入非敏感 Key 的 Value 中。

### 守護程序日誌 (Daemon Logging)

在 `logging.daemon` 下可以獨立設定守護程序的日誌，這對於追蹤背景同步任務非常有用：

```yaml
logging:
  daemon:
    enabled: true
    level: "info"
    file_path: "~/.config/syncrules/logs/daemon.log"
```

---

## 同步模式

### one-way-push（單向推送）

Source → Target。Source 為權威來源，Target 完全鏡像 Source。

- Source 有而 Target 沒有的檔案 → 複製到 Target
- Source 有且 Target 也有但不同 → 以 Source 覆蓋 Target
- Source 沒有而 Target 有 → 從 Target 刪除

**適用場景**：備份、發佈

### one-way-pull（單向拉取）

Target → Source。Target 為權威來源，Source 被 Target 內容覆蓋。

**適用場景**：從雲端拉取最新設定

### two-way（雙向同步）

Source ↔ Target。雙方都可以新增或修改檔案。

- 僅存在一方 → 複製到另一方
- 雙方都有且不同 → 依衝突策略處理

**適用場景**：多機協作、設定同步

---

## 衝突策略

當兩端的同一檔案有不同的版本時（在 `two-way` 模式下或 `one-way` 中的 type mismatch），Syncrules 依據衝突策略決定如何處理。

| 策略 | 行為 | 說明 |
|------|------|------|
| `keep_local` | 跳過複製（保留 **Target** 端版本） | Local = Target，即使用者所在端 |
| `keep_remote` | 從 **Source** 複製到 Target | Remote = Source，覆蓋 Target |
| `keep_newest` | 比較 mtime，較新者勝出 | 詳見下方說明 |
| `manual` | 標記為衝突，不自動處理 | **預設策略**（未指定 conflict 時） |

### keep_newest 細節

- Source mtime > Target mtime → 複製 Source 到 Target
- Target mtime > Source mtime → 複製 Target 到 Source
- mtime 相同 且 size 相同 → 視為相同，跳過
- mtime 相同 但 size 不同 → 標記為衝突

---

## Ignore 模式

使用 glob 語法排除不需要同步的檔案：

```yaml
ignore:
  - "*.log"        # 所有 .log 檔案
  - ".cache/"      # .cache 目錄
  - "*.tmp"        # 所有暫存檔
  - "secrets.yaml" # 特定檔案
  - ".git"         # Git 目錄
```

匹配規則：
- 先匹配檔名（`filepath.Base(path)`）
- 再匹配完整相對路徑

---

## 檔案比較策略

Syncrules 使用 **mtime + size** 作為預設的檔案比較策略：

1. **Size 不同** → 檔案已修改
2. **Size 相同，mtime 不同** → 檔案已修改
3. **Size 相同，mtime 相同** → 檔案相同

注意事項：
- 不依賴系統時鐘完全準確
- 跨平台 mtime 精度可能不同
- 未來支援 checksum 比較以偵測 size 相同但內容不同的情況

---

## 鎖機制

Syncrules 使用檔案鎖防止多個同步操作同時執行：

- 鎖檔位置：`~/.config/syncrules/.syncrules.lock`
- 包含 PID、主機名、啟動時間、規則名稱
- 同主機：透過 PID 偵測陳舊鎖
- 跨主機：30 分鐘後自動視為陳舊

如果遇到鎖衝突：
```
cannot acquire lock: lock is held by another process
(held by PID 12345 on DESKTOP-ABC since 2026-02-07T10:00:00Z, rule: my-sync)
```

等待另一個同步完成，或確認無其他同步後手動刪除鎖檔。

---

## 實際使用範例

### 範例 1：兩台電腦間同步設定

```yaml
transports:
  - name: local
    type: local

endpoints:
  - name: laptop-config
    transport: local
    root: C:\Users\user\.config\myapp
  - name: desktop-config
    transport: local
    root: D:\Config\myapp

rules:
  - name: sync-config
    mode: two-way
    source: laptop-config
    target: desktop-config
    ignore:
      - "*.log"
      - ".cache/"
    conflict: keep_newest
```

### 範例 2：備份到 Google Drive

```yaml
transports:
  - name: local
    type: local
  - name: gdrive
    type: gdrive
    config:
      client_id: "xxx.apps.googleusercontent.com"
      client_secret: "xxx"
      token_path: "~/.config/syncrules/gdrive-token.json"

endpoints:
  - name: local-docs
    transport: local
    root: ~/Documents/important
  - name: drive-backup
    transport: gdrive
    root: /SyncRules/documents

rules:
  - name: backup-docs
    mode: one-way-push
    source: local-docs
    target: drive-backup
    conflict: keep_local
```

### 範例 3：多組 AI Agent 設定同步

```yaml
transports:
  - name: local
    type: local

endpoints:
  - name: claude-skills
    transport: local
    root: C:\Users\user\.claude\skills
  - name: gemini-skills
    transport: local
    root: C:\Users\user\.gemini\skills
  - name: codex-skills
    transport: local
    root: C:\Users\user\.codex\skills

rules:
  - name: sync-claude-gemini
    mode: two-way
    source: claude-skills
    target: gemini-skills
    conflict: keep_newest

  - name: sync-claude-codex
    mode: two-way
    source: claude-skills
    target: codex-skills
    conflict: keep_newest
```
