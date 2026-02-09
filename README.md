# Syncrules

**跨電腦、跨路徑的設定檔與文件同步工具**

Syncrules 提供單一事實來源（Single Source of Truth），讓設定檔、知識庫和 Agent 上下文能在多台電腦之間可靠同步。

## 功能特色

- **排程同步** — 定時自動同步，支援後台守護程序
- **多種同步模式** — 單向推送、單向拉取、雙向同步
- **多儲存後端** — 本地檔案系統、Google Drive（可擴充）
- **規則式管理** — 透過 YAML 定義同步規則，Git 可追蹤
- **彈性排程** — 全域預設或個別規則自訂同步間隔
- **衝突解決** — keep_local / keep_remote / keep_newest / manual 四種策略
- **安全同步** — 檔案鎖防止並行操作、原子寫入防止部分覆寫
- **Dry-run 預覽** — 執行前預覽所有變更
- **進度顯示** — 即時傳輸進度與速度
- **安全日誌** — 結構化日誌輸出、PII 自動脫敏
- **跨平台** — Windows / macOS / Linux

## 快速開始

### 安裝

```bash
# 從原始碼建置
git clone https://github.com/Ning0612/Syncrules.git
cd Syncrules
make build

# 或使用 go install
go install github.com/Ning0612/Syncrules/cmd/syncrules@latest
```

### 建立設定檔

```bash
mkdir -p ~/.config/syncrules
cp configs/example.yaml ~/.config/syncrules/config.yaml
```

編輯 `config.yaml`：

```yaml
# 啟用排程器（可選）
scheduler:
  enabled: true
  default_interval: "5m"  # 每 5 分鐘同步一次

transports:
  - name: local
    type: local

endpoints:
  - name: source
    transport: local
    root: ~/Documents/config
  - name: backup
    transport: local
    root: /mnt/backup/config

rules:
  - name: backup-config
    mode: one-way-push
    source: source
    target: backup
    schedule:
      enabled: true       # 包含在排程同步中
      interval: "10m"     # 覆寫預設間隔（可選）
    ignore:
      - "*.log"
      - ".cache/"
    conflict: keep_newest
```

### 執行同步

```bash
# 預覽變更
syncrules sync --dry-run

# 執行同步
syncrules sync

# 指定規則
syncrules sync --rule backup-config

# 顯示進度
syncrules sync --progress
```

## 設定概念

```
Transport（儲存後端）
  └── Endpoint（具體位置）
        └── Rule（同步關係）
```

| 概念 | 說明 |
|------|------|
| **Transport** | 儲存類型（`local` 或 `gdrive`） |
| **Endpoint** | Transport 中的特定根路徑 |
| **Rule** | 兩個 Endpoint 之間的同步關係 |

## 同步模式

| 模式 | 方向 | 說明 |
|------|------|------|
| `one-way-push` | Source → Target | 單向推送，Target 鏡像 Source |
| `one-way-pull` | Target → Source | 單向拉取，Source 鏡像 Target |
| `two-way` | Source ↔ Target | 雙向同步，雙方皆可變更 |

> 預設值：`conflict` 預設 `manual`（需手動處理衝突），`enabled` 預設 `true`。

## 命令列

### 手動同步

```bash
syncrules sync [flags]       # 執行同步
syncrules auth gdrive [flags] # Google Drive 認證
syncrules version            # 顯示版本
```

| 旗標 | 說明 |
|------|------|
| `--config, -c` | 設定檔路徑 |
| `--rule` | 僅執行指定規則 |
| `--dry-run` | 預覽模式 |
| `--output, -o` | 輸出格式（table/json） |
| `--progress, -p` | 顯示進度 |
| `--log-level` | 日誌層級（debug/info/warn/error） |
| `--log-format` | 日誌格式（text/json） |
| `--log-file-only` | 僅輸出到日誌檔案 |

### 排程守護程序

```bash
# 啟動守護程序（背景執行）
syncrules daemon start --detach

# 檢查狀態
syncrules daemon status

# 停止守護程序
syncrules daemon stop

# 前景模式（測試用）
syncrules daemon start --interval 1m
```

更多詳情請參閱 [Daemon 使用指南](docs/daemon-usage.md)

## 技術架構

```
CLI (Cobra)
  └── Service (編排層)
        ├── Daemon Service   — 排程守護程序
        │   ├── Scheduler     — 定時排程器
        │   ├── State Manager — 同步狀態追蹤（SQLite）
        │   └── PID Manager   — 程序管理（跨平台）
        ├── Core (業務邏輯)
        │   ├── Rule Executor  — 規則執行
        │   ├── Planner        — 計畫生成
        │   ├── Diff Comparer  — 檔案比較
        │   └── Conflict Resolver — 衝突解決
        └── Adapters (儲存後端)
            ├── Local FS
            └── Google Drive
```

**技術選型**：Go + Cobra + Viper + SQLite + Google Drive API v3

> GUI（Fyne 框架）目前規劃中，尚未實作。

## 文件

| 文件 | 說明 |
|------|------|
| [Daemon 使用指南](docs/daemon-usage.md) | 守護程序使用、排程配置、系統服務整合 |
| [配置範例](docs/config-examples.md) | 各種排程模式與配置範例 |
| [架構文件](docs/architecture.md) | 系統架構、模組設計、資料流 |
| [開發指南](docs/development.md) | 開發環境、程式碼規範、測試策略 |
| [部署指南](docs/deployment.md) | 建置、安裝、設定、自動化排程 |
| [使用手冊](docs/usage.md) | CLI 命令、設定格式、同步模式 |

## 開發

```bash
make deps        # 下載相依
make build       # 建置
make test        # 測試
make test-cover  # 覆蓋率報告
make lint        # 靜態分析
make fmt         # 格式化
```

## 設計原則

> 本專案追求**清晰、正確、可信賴** — 而非極致的同步速度。

- 同步引擎與 UI 完全解耦
- 所有同步行為皆可透過設定檔描述
- 設定檔可版本控制
- 明確設定優於隱含行為

## 授權

MIT License
