# Syncrules

跨電腦、跨路徑的設定檔與文件同步工具。

## Features

- **多電腦同步**：筆電、桌機、工作站之間設定檔一致
- **多路徑映射**：同一台電腦不同路徑共用單一真實來源
- **雙向同步**：支援 one-way（push/pull）和 two-way 模式
- **規則驅動**：透過 YAML 設定檔定義同步行為
- **多後端支援**：Local filesystem、Google Drive

## Installation

```bash
# From source
go install github.com/Ning0612/Syncrules/cmd/syncrules@latest

# Or build locally
make build
```

## Quick Start

1. 複製設定檔範例：
```bash
cp configs/example.yaml ~/.config/syncrules/config.yaml
```

2. 編輯設定檔，定義你的 endpoints 和 rules

3. 預覽同步計畫：
```bash
syncrules sync --dry-run
```

4. 執行同步：
```bash
syncrules sync
```

## Configuration

參考 `configs/example.yaml` 了解完整設定選項。

### 基本概念

- **Transport**：儲存後端類型（local / gdrive）
- **Endpoint**：具體的根目錄位置
- **Rule**：定義兩個 endpoint 之間的同步關係

### Sync Modes

| Mode | Description |
|------|-------------|
| `one-way-push` | Source → Target |
| `one-way-pull` | Target → Source |
| `two-way` | Bidirectional |

### Conflict Strategies

| Strategy | Description |
|----------|-------------|
| `keep_local` | 本地端優先 |
| `keep_remote` | 遠端優先 |
| `keep_newest` | 較新的版本優先 |
| `manual` | 需要使用者介入（預設） |

## Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Build
make build

# Lint
make lint
```

## License

MIT
