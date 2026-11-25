# ETHER API 服務（services/api）

狀態：Phase 01 骨架，草案／待批准。本機 VM 無 go／docker／psql，建置、測試、資料庫連線皆為「待 CI 驗證」，不得宣稱已通過。

## 服務範圍

本階段只提供 `GET /healthz`（行程存活檢查）。18 個 API 前綴（`API_STANDARDS.md` §2）與全部業務端點皆尚未實作，不得以佔位路由冒充功能。業務 handler 待 Phase 03 起逐階段落地。

## 建置／執行（待 CI 驗證）

```sh
cd services/api
go build ./...          # 靜態建置檢查（待 CI）
go vet ./...            # 靜態檢查（待 CI）
go test ./...           # 單元測試骨架（待 CI）
go run ./cmd/api        # 本機執行；需先設定環境變數（見下）
```

`go.mod` 的 module path 為暫定 `ether.local/api`，待 repo 建立後修訂；`go` directive 固定為 `1.25`（2026-11-25，與本目錄 Dockerfile 的 `golang:1.25.0-alpine3.21` 一致）。

### 本機執行所需的環境變數

依賴位址由 `docker-compose.yml` 的 `api` 服務組裝；本機直接執行時須自行設定（以下為佔位值，不得填入真憑證）：

```sh
export DATABASE_URL="postgres://ether:placeholder@localhost:5432/ether?sslmode=disable"
export VALKEY_URL="redis://localhost:6379/0"
export NATS_URL="nats://localhost:4222"
export MINIO_ENDPOINT="localhost:9000"
export MEILISEARCH_URL="http://localhost:7700"
export KEYCLOAK_URL="http://localhost:8080"
export API_PORT="8080"   # 可選，未設定時預設 8080
```

任一依賴位址缺值時，程式明確報錯退出（見 `internal/config`）；`_OVERRIDE` 後綴可覆寫任一位址（例如 `DATABASE_URL_OVERRIDE`），覆寫清單的最終定稿待 Phase 01 確認。本階段不建立任何依賴連線，`/healthz` 的 `ok` 只代表行程存活，不得解讀為依賴就緒（依賴檢查是 TODO，見 `internal/health`）。

## 模組地圖

| 目錄 | 用途 |
|---|---|
| `cmd/api/` | 主程式（設定載入、路由組裝、優雅關停）與 healthz 測試骨架 |
| `internal/transport/` | 框架隔離層：`Router` 介面＋標準庫 `StdRouter` 實作、錯誤信封寫出器 |
| `internal/config/` | 環境變數載入（缺值明確報錯，不得給預設真憑證） |
| `internal/secrets/` | Secrets Provider 介面＋環境變數實作＋OpenBao stub（未設定時回傳 `not configured`，不得假裝讀到 secret） |
| `internal/telemetry/` | 註解型 stub：Logger 介面＋NoopLogger，環境變數名 only，不得 import SDK |
| `internal/health/` | `GET /healthz` handler（只含 status／service／version） |
| `internal/<模組>/`（22 個） | 模組邊界文件（各一個 `doc.go`）：責任、擁有資料表、允許依賴、禁止事項、所屬階段；本階段不得寫功能程式碼 |
| `migrations/` | 版本化 migration 慣例與 `000001_baseline`（僅 `schema_meta` 佔位表，不建 64 張業務表） |

22 個模組目錄名：`auth`、`user`、`profile`、`friend`、`contact`、`chat`、`message`、`group`、`community`、`media`、`sticker`、`call`、`notification`、`search`、`officialaccount`、`bot`、`timeline`、`payment`、`moderation`、`admin`、`analytics`、`device`。命名慣例沿用 `SPEC-SOURCE.md` §26 的短名；§26 未列的 10 個模組依同一慣例補齊（見 `MODULE_BOUNDARIES.md` §6 要求），目錄名與模組全名的對照見各 `doc.go` 檔頭與 `docs/phase01/go-notes.md`，待 ADR／Phase 01 文件確認。

## 框架決策阻塞說明

Gin／Echo 二選一尚未決定，最終決定權在使用者（比較見 `docs/phase01/gin-vs-echo-memo.md`）。本骨架只用標準庫 `net/http`，handler 只依賴 `transport.Router` 介面與標準 context，不得 import 任何框架套件；決定後以 Gin／Echo adapter 實作同一 `Router` 介面替換 `StdRouter`。

## 已知缺口

- 業務 handler、認證／授權、資料庫連線、NATS／Valkey／MinIO／Meilisearch／Keycloak 接線：全部 TODO，待後階段。
- 依賴就緒檢查（readiness）：TODO；`SYSTEM_ARCHITECTURE.md` §8.3 的 liveness／readiness 分離與降級策略待 Phase 01 定案。
- 錯誤回應格式與成功回應包裝：草案（`ERROR_CODES.md` §5.1、`API_STANDARDS.md` §4.2），待定稿確認。
- Secrets Provider 的選擇方式（env／OpenBao 切換）：待部署定稿；本階段預設環境變數實作。
- Telemetry SDK 接線（OTLP／Loki／GlitchTip）：TODO，不得 import SDK。
- Migration 工具選用與執行主體：待定稿（見 `migrations/MIGRATIONS.md` §5）；現行 Dockerfile 未複製 `migrations/`，接線時須一併定稿。
- 與 `docker-compose.yml`／Dockerfile 的對齊缺口：見 `docs/phase01/go-notes.md`（三服務共用 `services/api` context、worker 健康檢查等），本軌不得擅自改 compose 或 Dockerfile。
