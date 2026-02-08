# Syncrules 開發指南

## 先決條件

| 工具 | 最低版本 | 用途 |
|------|---------|------|
| Go | 1.24+ | 編譯與執行 |
| Git | 2.0+ | 版本控制 |
| Make | 任意版本 | 建置自動化 |
| goimports | latest | 程式碼格式化（`make fmt` 需要） |
| golangci-lint | latest | 靜態分析（選用） |

> **安裝 goimports**：`go install golang.org/x/tools/cmd/goimports@latest`

---

## 快速開始

```bash
# 1. 克隆專案
git clone https://github.com/Ning0612/Syncrules.git
cd Syncrules

# 2. 下載相依套件
make deps

# 3. 建置
make build

# 4. 執行測試
make test

# 5. 確認建置成功
./dist/syncrules version
```

---

## 專案結構

```
syncrules/
├── cmd/
│   ├── syncrules/           # CLI 進入點
│   │   ├── main.go          # 根命令與版本命令
│   │   ├── sync.go          # sync 子命令
│   │   └── auth.go          # auth 子命令（Google Drive 認證）
│   └── syncrules-gui/       # GUI 進入點（未實作，規劃中）
├── internal/
│   ├── domain/              # 核心模型與錯誤定義
│   │   ├── file.go          # FileInfo 檔案元資料
│   │   ├── sync.go          # SyncMode, SyncAction, SyncPlan
│   │   ├── rule.go          # SyncRule, Transport, Endpoint
│   │   └── errors.go        # 分層錯誤
│   ├── adapter/             # 儲存適配器
│   │   ├── adapter.go       # Adapter 介面定義
│   │   ├── local/           # 本地檔案系統適配器
│   │   └── gdrive/          # Google Drive 適配器
│   ├── core/                # 同步引擎
│   │   ├── diff/            # 檔案比較邏輯
│   │   ├── planner/         # 同步計畫生成
│   │   ├── conflict/        # 衝突解決策略
│   │   └── rule/            # 規則執行器
│   ├── config/              # 設定檔解析
│   ├── service/             # 應用程式編排層
│   ├── lock/                # 檔案互斥鎖
│   ├── progress/            # 進度回報
│   └── testutil/            # 測試工具函式
├── configs/                 # 設定檔範例
├── docs/                    # 文件
├── tests/                   # E2E/整合測試資料
├── Makefile                 # 建置腳本
├── go.mod                   # Go 模組定義
└── .editorconfig            # 編輯器設定
```

---

## 建置命令

| 命令 | 說明 |
|------|------|
| `make build` | 建置 CLI 二進位檔到 `dist/syncrules` |
| `make build-gui` | 建置 GUI 二進位檔到 `dist/syncrules-gui` |
| `make build-all` | 同時建置 CLI 和 GUI |
| `make clean` | 清除建置產出 |
| `make test` | 執行所有測試 |
| `make test-cover` | 執行測試並產生覆蓋率報告 |
| `make lint` | 執行 golangci-lint 靜態分析 |
| `make fmt` | 格式化程式碼 |
| `make vet` | 執行 go vet |
| `make deps` | 下載並整理相依套件 |
| `make install` | 安裝到 `$GOPATH/bin` |

### 直接使用 Go 命令

```bash
# 建置
go build -o dist/syncrules ./cmd/syncrules

# 測試
go test ./...
go test -v -run TestFunctionName ./internal/core/diff
go test -cover ./...

# 靜態檢查
go vet ./...
```

---

## 程式碼規範

### 風格

- 使用 **tab** 縮排（依 `.editorconfig` 設定）
- YAML 檔案使用 2 空格縮排
- 匯入順序：標準庫 → 第三方 → 專案內部
- 遵循 Go 官方 [Effective Go](https://go.dev/doc/effective_go) 慣例

### 命名慣例

| 對象 | 慣例 | 範例 |
|------|------|------|
| 套件 | 小寫單字 | `planner`, `diff` |
| 匯出函式/型別 | PascalCase | `SyncService`, `NewDefaultPlanner` |
| 私有函式/變數 | camelCase | `getAdapter`, `listAllFiles` |
| 常數 | PascalCase | `SyncModeOneWayPush` |
| 檔案名 | snake_case | `process_windows.go` |

### 錯誤處理

1. 使用 `internal/domain/errors.go` 中定義的錯誤
2. Adapter 層必須將原生錯誤映射到 domain 錯誤
3. 使用 `fmt.Errorf("%w", err)` 包裝錯誤以保留鏈
4. 不要丟棄錯誤 — 回傳或記錄

```go
// 正確
if err != nil {
    return fmt.Errorf("failed to list files: %w", err)
}

// 錯誤
if err != nil {
    log.Println(err) // 不要只記錄而不回傳
}
```

### 介面設計

- 介面定義在消費者（呼叫方）的套件中
- 使用小介面（1-3 個方法為佳）
- 總是提供 `Default` 前綴的具體實作

```go
// 在 core/diff/ 中定義
type Comparer interface {
    Compare(src, tgt *domain.FileInfo) DiffResult
}
type DefaultComparer struct{}
```

---

## 測試策略

### 測試分類

| 類型 | 位置 | 說明 |
|------|------|------|
| 單元測試 | `*_test.go`（同套件） | 測試單一函式或型別 |
| 整合測試 | `*_test.go`（同套件） | 測試模組間互動 |
| E2E 測試 | `tests/`（規劃中） | 端到端測試 |

### 測試命名

```go
func TestDefaultComparer_Compare_FilesIdentical(t *testing.T) { ... }
func TestDefaultComparer_Compare_FileOnlyInSource(t *testing.T) { ... }
```

格式：`Test<Type>_<Method>_<Scenario>`

### 測試工具

`internal/testutil/` 提供測試輔助函式：

- `TempDir()` — 建立臨時測試目錄
- `CreateTestFile()` — 建立測試檔案
- `CreateTestFileWithSize()` — 建立指定大小的測試檔案
- `WaitForCondition()` — 等待條件滿足
- `AssertEventually()` — 斷言條件最終為真

### 執行測試

```bash
# 全部測試
make test

# 指定套件
go test -v ./internal/core/planner/

# 指定測試
go test -v -run TestPlanOneWay_NewFiles ./internal/core/planner/

# 產生覆蓋率報告
make test-cover
# 覆蓋率 HTML 報告輸出至 coverage.html
```

---

## 新增功能指南

### 新增 Adapter

1. 在 `internal/adapter/` 下建立新目錄（如 `s3/`）
2. 實作 `Adapter` 介面的所有方法
3. 在 `domain/rule.go` 新增 `TransportType` 常數
4. 在 `service/sync.go` 的 `getAdapter()` 中新增 case
5. 撰寫測試

```go
// internal/adapter/s3/s3.go
package s3

type Adapter struct { ... }

func New(bucket, region string) (*Adapter, error) { ... }
func (a *Adapter) List(ctx context.Context, path string) ([]domain.FileInfo, error) { ... }
// ... 實作所有 Adapter 介面方法
```

### 新增同步模式

1. 在 `domain/sync.go` 新增 `SyncMode` 常數
2. 更新 `SyncMode.IsValid()` 方法
3. 在 `core/planner/` 新增對應的 Plan 方法
4. 在 `core/rule/executor.go` 的 `Plan()` 中新增 case

### 新增衝突策略

1. 在 `domain/sync.go` 新增 `ConflictStrategy` 常數
2. 更新 `ConflictStrategy.IsValid()` 方法
3. 在 `core/conflict/resolver.go` 的 `Resolve()` 中新增 case

### 新增 CLI 命令

1. 在 `cmd/syncrules/` 新增 `<command>.go`
2. 定義 `*cobra.Command`
3. 在 `init()` 中使用 `rootCmd.AddCommand()` 註冊

---

## 相依套件

### 直接相依

| 套件 | 用途 |
|------|------|
| `github.com/spf13/cobra` | CLI 框架 |
| `github.com/spf13/viper` | 設定檔解析 |

### 間接相依

| 套件 | 用途 |
|------|------|
| `golang.org/x/oauth2` | OAuth2 認證 |
| `google.golang.org/api/drive/v3` | Google Drive API |
| `golang.org/x/sys/windows` | Windows 系統呼叫（PID 檢查） |

---

## 常見開發情境

### 偵錯同步問題

```bash
# 使用 dry-run 檢視計畫
./dist/syncrules sync --config configs/test.yaml --dry-run

# JSON 格式輸出便於分析
./dist/syncrules sync --config configs/test.yaml --dry-run -o json

# 顯示進度
./dist/syncrules sync --config configs/test.yaml --progress
```

### 測試 Google Drive 整合

```bash
# 先完成認證
./dist/syncrules auth gdrive --client-id YOUR_ID --client-secret YOUR_SECRET

# 執行含 gdrive 的同步規則
./dist/syncrules sync --config configs/sync.yaml --rule backup-to-cloud --dry-run
```
