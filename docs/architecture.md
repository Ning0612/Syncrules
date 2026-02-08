# Syncrules 架構文件

## 概述

Syncrules 採用**分層架構（Layered Architecture）**設計，將系統分為三個清晰的層級：表現層（CLI/GUI）、核心業務邏輯層（Sync Core）、以及儲存適配器層（Adapters）。各層之間透過定義良好的介面通訊，確保低耦合與高內聚。

---

## 系統架構圖

```
┌─────────────────────────────────────────────────────────┐
│                     表現層 (Presentation)                │
│  ┌───────────────┐  ┌───────────────┐                   │
│  │ CLI (Cobra)   │  │ GUI (Fyne)    │                   │
│  │ cmd/syncrules │  │ cmd/syncrules │                   │
│  │  - main.go    │  │   -gui/       │                   │
│  │  - sync.go    │  │  (未實作)     │                   │
│  │  - auth.go    │  │               │                   │
│  └──────┬────────┘  └──────┬────────┘                   │
│         │                  │                             │
│         └────────┬─────────┘                             │
│                  ▼                                       │
│  ┌───────────────────────────────────────────────────┐  │
│  │           Service 層 (Orchestration)               │  │
│  │  internal/service/sync.go                          │  │
│  │  - SyncService: 統一協調同步操作                    │  │
│  │  - 管理 Adapter 生命週期                           │  │
│  │  - 管理 Lock / Progress                            │  │
│  └──────────────────────┬────────────────────────────┘  │
│                         ▼                                │
│  ┌───────────────────────────────────────────────────┐  │
│  │              Core 層 (Business Logic)              │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────────┐      │  │
│  │  │ Rule     │ │ Planner  │ │ Conflict     │      │  │
│  │  │ Executor │→│          │→│ Resolver     │      │  │
│  │  └──────────┘ └──────────┘ └──────────────┘      │  │
│  │                    ▲                               │  │
│  │                    │                               │  │
│  │               ┌────┴────┐                          │  │
│  │               │  Diff   │                          │  │
│  │               │Comparer │                          │  │
│  │               └─────────┘                          │  │
│  └──────────────────────┬────────────────────────────┘  │
│                         ▼                                │
│  ┌───────────────────────────────────────────────────┐  │
│  │             Adapter 層 (Storage Backends)          │  │
│  │  ┌──────────────┐  ┌──────────────┐               │  │
│  │  │ Local FS     │  │ Google Drive │               │  │
│  │  │ adapter/     │  │ adapter/     │               │  │
│  │  │ local/       │  │ gdrive/      │               │  │
│  │  └──────────────┘  └──────────────┘               │  │
│  │             ▲ 統一 Adapter 介面 ▲                  │  │
│  └───────────────────────────────────────────────────┘  │
│                                                         │
│  ┌───────────────────────────────────────────────────┐  │
│  │            橫切關注點 (Cross-Cutting)               │  │
│  │  ┌────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐ │  │
│  │  │ Domain │ │ Config   │ │ Lock     │ │Progress│ │  │
│  │  │ Models │ │ (Viper)  │ │ (File)   │ │Reporter│ │  │
│  │  └────────┘ └──────────┘ └──────────┘ └────────┘ │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

---

## 模組詳解

### 1. Domain 層 (`internal/domain/`)

Domain 層定義所有核心資料模型與業務錯誤，不依賴任何外部套件。

| 檔案 | 職責 |
|------|------|
| `file.go` | `FileInfo` 結構 — 檔案元資料（路徑、類型、大小、mtime、checksum） |
| `sync.go` | `SyncMode`、`ConflictStrategy`、`SyncAction`、`SyncPlan` 等同步核心型別 |
| `rule.go` | `SyncRule`、`Transport`、`Endpoint` 等設定模型 |
| `errors.go` | 分層錯誤定義（Adapter/Sync/Config 三組） |

#### 核心型別

```
FileInfo        ── 檔案/目錄元資料（Path, Type, Size, ModTime, Checksum, ETag, IsDeleted）
SyncRule        ── 同步規則定義
Transport       ── 儲存後端設定（local / gdrive）
Endpoint        ── 具體位置定義（transport + root path）
SyncAction      ── 單一同步操作（copy/delete/mkdir/conflict/skip）
SyncPlan        ── 完整同步計畫（Actions + Conflicts + Stats）
SyncMode        ── 同步模式（one-way-push / one-way-pull / two-way）
ConflictStrategy── 衝突策略（keep_local / keep_remote / keep_newest / manual）
```

> **注意**：`ETag` 主要供雲端 Adapter 使用（Google Drive），`IsDeleted` 用於追蹤刪除的 tombstone 記錄。

### 2. Adapter 層 (`internal/adapter/`)

Adapter 層為儲存後端提供統一介面，所有實作必須遵守 `Adapter` 介面契約。

#### Adapter 介面

```go
type Adapter interface {
    List(ctx context.Context, path string) ([]domain.FileInfo, error)
    Read(ctx context.Context, path string) (io.ReadCloser, error)
    Write(ctx context.Context, path string, r io.Reader) error
    Delete(ctx context.Context, path string) error
    Stat(ctx context.Context, path string) (domain.FileInfo, error)
    Mkdir(ctx context.Context, path string) error
    Exists(ctx context.Context, path string) (bool, error)
    Close() error
}
```

#### 設計原則

- Adapter **不知道**同步策略，只負責儲存操作
- 所有路徑均為相對路徑（相對於 Endpoint root）
- 錯誤必須轉換為 `domain` 層錯誤（如 `ErrNotFound`、`ErrPermissionDenied`）
- 路徑安全：防止路徑穿越攻擊（`..` 檢查）

#### Local Adapter (`adapter/local/`)

- 基於 `os` 套件操作本地檔案系統
- 寫入使用臨時檔 + 原子重命名，確保寫入安全
- 支援 SHA256 checksum 計算（`StatWithChecksum`）
- 路徑正規化：跨平台路徑分隔符處理

#### Google Drive Adapter (`adapter/gdrive/`)

- 基於 Google Drive API v3
- OAuth2 授權（Authorization Code Flow）
- 路徑 → 資料夾 ID 快取（`idCache`，執行緒安全）
- 自動建立中間資料夾
- 分頁查詢（每頁 100 筆）
- API 錯誤碼映射到 domain 錯誤

### 3. Core 層 (`internal/core/`)

Core 層包含所有同步業務邏輯，分為四個子模組：

#### Diff (`core/diff/`)

- `Comparer` 介面：比較來源與目標檔案
- `DefaultComparer`：使用 mtime + size 策略判斷檔案差異
- 回傳 `DiffResult`：`FilesIdentical`、`FileModified`、`FileOnlyInSource`、`FileOnlyInTarget`

#### Planner (`core/planner/`)

- `Planner` 介面：生成同步計畫
- `PlanOneWay()`：單向同步計畫（push 或 pull）
- `PlanTwoWay()`：雙向同步計畫
- 動作排序：Mkdir → Copy → Delete → Conflict
- 支援 ignore pattern（glob 語法）

#### Conflict Resolver (`core/conflict/`)

- `Resolver` 介面：解決衝突
- 四種策略實作：
  - `keep_local`：保留 **Target** 端版本（skip，不從 Source 複製）
  - `keep_remote`：使用 **Source** 端版本覆蓋 Target
  - `keep_newest`：依 mtime 決定較新者勝出；mtime 相同且 size 相同則跳過；mtime 相同但 size 不同則標記衝突
  - `manual`：標記為衝突待手動處理（**預設策略**）

#### Rule Executor (`core/rule/`)

- `Executor` 介面：協調同步規劃流程
- 遞迴列出來源與目標所有檔案
- 建立檔案路徑 → FileInfo 映射表
- 根據 SyncMode 呼叫對應 Planner 方法

### 4. Service 層 (`internal/service/`)

Service 層是應用程式編排層，連接 CLI/GUI 與 Core 層。

**`SyncService`** 負責：
- 配置解析 → Adapter 建立 → Rule 執行
- 檔案鎖管理（防止並行同步）
- 進度回報
- Adapter 生命週期管理

### 5. Config 層 (`internal/config/`)

- 使用 Viper 解析 YAML 設定檔
- 支援多路徑搜尋：`./`、`./configs/`、`~/.config/syncrules/`、`~/.syncrules/`
- 完整驗證：名稱唯一性、引用完整性、規則合法性
- 路徑展開：`~` 與環境變數

### 6. Lock 機制 (`internal/lock/`)

- 基於檔案的互斥鎖（`.syncrules.lock`）
- 原子建立（`O_CREATE|O_EXCL`）
- 陳舊鎖偵測（雙層判斷）：
  - **同主機**：檢查持有者 PID 是否仍存活，PID 存活則鎖**不**視為陳舊（無論時間）
  - **跨主機**：無法檢查 PID，改用 30 分鐘逾時作為 fallback
- 跨平台 PID 檢查（Windows `OpenProcess` / Unix `signal 0`）
- 支援 `ForceRelease()` 強制解鎖（需確認持有者已停止）

### 7. Progress 模組 (`internal/progress/`)

- `Reporter` 介面：追蹤檔案傳輸進度
- `CallbackReporter`：回呼式實作（執行緒安全）
- `ProgressReader`/`ProgressWriter`：包裝 io.Reader/Writer
- `NullReporter`：空實作（無進度回報）
- 輔助函式：格式化檔案大小、速度、進度條

---

## 資料流

### 同步執行流程

```
使用者執行 CLI 命令
        │
        ▼
┌─ config.Load() ──────────────────────────┐
│  解析 YAML → 驗證 → Config 物件          │
└──────────────────────────────────────────┘
        │
        ▼
┌─ service.NewSyncService() ───────────────┐
│  建立 SyncService（含 FileLock）          │
└──────────────────────────────────────────┘
        │
        ▼
┌─ service.PlanSync() ─────────────────────┐
│  1. 取得 Rule                             │
│  2. 建立/取得 Source & Target Adapter     │
│  3. Rule Executor: 遞迴列出雙方檔案      │
│  4. Planner: 依 SyncMode 生成 SyncPlan   │
│     - Diff 比較檔案差異                   │
│     - Conflict Resolver 處理衝突         │
└──────────────────────────────────────────┘
        │
        ▼
┌─ CLI 顯示計畫 ───────────────────────────┐
│  Table 或 JSON 格式輸出 SyncPlan         │
└──────────────────────────────────────────┘
        │ (非 dry-run 時)
        ▼
┌─ service.ExecuteSync() ──────────────────┐
│  1. 取得 FileLock                         │
│  2. 依序執行 SyncAction：                 │
│     - Copy: Read → ProgressReader → Write │
│     - Mkdir: 建立目錄                    │
│     - Delete: 刪除檔案                   │
│     - Skip/Conflict: 跳過               │
│  3. 釋放 FileLock                         │
└──────────────────────────────────────────┘
```

### 方向感知同步

Syncrules 的同步操作具備方向感知能力。`SyncAction.Direction` 決定 `fromAdapter` 與 `toAdapter` 的映射：

- `DirSourceToTarget`：從 Source Adapter 讀取，寫入 Target Adapter
- `DirTargetToSource`：從 Target Adapter 讀取，寫入 Source Adapter

這使得雙向同步（two-way）可以在同一個 SyncPlan 中混合包含兩個方向的操作。

---

## 設計決策

### 為何選擇 Go？
- 跨平台單一二進位檔，無需安裝執行環境
- 優秀的並行處理能力
- 成熟的 Google API 生態系
- 靜態型別確保正確性

### 為何選擇 YAML？
- 人類可讀性高
- AI 友善（便於 LLM 理解與生成）
- Git 可追蹤差異
- 支援註解

### 為何用 mtime + size 而非純 checksum？
- 速度優先：不需讀取檔案內容
- 足夠準確：大部分情境下 mtime + size 能可靠偵測變更
- 未來擴展：已預留 checksum 欄位供進階比較

### 為何使用檔案鎖而非記憶體鎖？
- 防止跨程序並行同步（例如 cron 排程重疊）
- 在程序意外終止後可偵測陳舊鎖
- 跨平台支援

---

## 安全考量

1. **路徑穿越防護**：Local 與 GDrive adapter 均驗證路徑不會逃脫 root
2. **Token 安全**：OAuth token 使用 `0600` 權限儲存，目錄使用 `0700`
3. **原子寫入**：檔案寫入使用臨時檔 + rename，避免部分寫入
4. **Query 注入防護**：Google Drive API 查詢字串會跳脫特殊字元
5. **CSRF 防護**：OAuth 流程使用密碼學安全的隨機 state 參數
