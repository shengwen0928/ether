# ETHER Worker（services/worker）

狀態：Phase 01 骨架，草案／待批准。本機 VM 無 go／docker，建置與測試皆為「待 CI 驗證」，不得宣稱已通過。

## 服務範圍

本階段只啟動 `/healthz` 佔位，不消費任何佇列、不處理任何任務。NATS JetStream 消費者（推播扇出、搜尋索引同步、媒體縮圖／轉碼、Bot Webhook 投遞、重試與死信處理）與任務定義待 Phase 05 起落地（見 `internal/jobs/doc.go`）。不得寫假任務處理，不得以空迴圈或定時器冒充背景工作。

## 建置／執行（待 CI 驗證）

```sh
cd services/worker
go build ./...          # 待 CI
go vet ./...            # 待 CI
go test ./...           # 待 CI
go run ./cmd/worker     # 本機執行；埠號見下
```

`go.mod` 的 module path 為暫定 `ether.local/worker`，待 repo 建立後修訂；`go` directive 固定為 `1.25`（2026-11-25，與本目錄 Dockerfile 的 `golang:1.25.0-alpine3.21` 一致）。

埠號：`WORKER_PORT`（Phase 01 佔位名），未設定時預設 `8082`。注意：`docker-compose.yml` 未為 worker 設定埠號與 healthcheck，worker 的 Dockerfile 亦不設 HEALTHCHECK、不假設 worker 有 HTTP 端點；本 `/healthz` 監聽埠現階段不被 compose 或 Dockerfile 探測，worker 是否保留 HTTP 端點待後階段定稿（見 `docs/phase01/go-notes.md`）。

## 模組地圖

| 目錄 | 用途 |
|---|---|
| `cmd/worker/` | 主程式（`/healthz` 佔位、優雅關停）與測試骨架 |
| `internal/health/` | `GET /healthz` handler（只含 status／service／version；與其他服務的同名套件為獨立 module 下有意的重複） |
| `internal/jobs/` | 任務定義佔位文件（`doc.go` only，無任務程式碼） |

## 已知缺口與對齊問題（待協調員裁決，不得擅自改）

- `docker-compose.yml` 的 `worker` 服務已以本目錄為獨立 build context，使用本目錄的 Dockerfile（G-01 採方案 A，見 ADR-015）。
- Worker 的 HTTP 端點存廢、NATS 消費者接線、任務定義、重試／DLQ 策略：全部待 Phase 05 起定稿。
- 其餘對齊核對結果見 `docs/phase01/go-notes.md`。
