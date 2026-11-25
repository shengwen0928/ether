# ETHER Realtime Gateway（services/realtime-gateway）

狀態：Phase 01 骨架，草案／待批准。本機 VM 無 go／docker，建置與測試皆為「待 CI 驗證」，不得宣稱已通過。

## 服務範圍

本階段只啟動 HTTP 骨架與 `GET /healthz`，WebSocket 端點（`/ws/v1` 佔位路徑）只回 501 錯誤信封。不得寫假 WS 握手、假廣播，不得假裝具備即時投遞能力。WS 協定（認證、Registry、Heartbeat、重連恢復、Rate Limit、Backpressure、Graceful Shutdown）與 15 個事件（`WEBSOCKET_PROTOCOL.md` §6）待 Phase 05 落地。

## 建置／執行（待 CI 驗證）

```sh
cd services/realtime-gateway
go build ./...          # 待 CI
go vet ./...            # 待 CI
go test ./...           # 待 CI
go run ./cmd/realtime-gateway   # 本機執行；埠號見下
```

`go.mod` 的 module path 為暫定 `ether.local/realtime-gateway`，待 repo 建立後修訂；`go` directive 固定為 `1.25`（2026-11-25，與本目錄 Dockerfile 的 `golang:1.25.0-alpine3.21` 一致）。

埠號：`GATEWAY_PORT`（`.env.example` 末節列出的名稱），未設定時預設 `8081`，與 `docker-compose.yml` 的 healthcheck（`wget http://localhost:8081/healthz`）一致。

## 模組地圖

| 目錄 | 用途 |
|---|---|
| `cmd/realtime-gateway/` | 主程式（HTTP 骨架、`/healthz`、WS 501 佔位、優雅關停）與測試骨架 |
| `internal/health/` | `GET /healthz` handler（只含 status／service／version；與其他服務的同名套件為獨立 module 下有意的重複） |

WS 佔位回應使用 `ETHER_NOT_IMPLEMENTED` 佔位碼：`ERROR_CODES.md` 並無 501／NOT_IMPLEMENTED 定稿碼，本碼僅遵循 `ETHER_` 前綴格式，待 Phase 05 定稿，不得視為已定錯誤碼。`/ws/v1` 路徑為 `WEBSOCKET_PROTOCOL.md` §2 的草案方向，待定稿確認。

## 已知缺口與對齊問題（待協調員裁決，不得擅自改）

- 本目錄的 Dockerfile 建置路徑為 `./cmd/realtime-gateway`，與本程式一致（先前草稿稱 `./cmd/gateway` 的說法已過時，以 Dockerfile 實查為準，協調員 2026-11-25 更正）。
- `docker-compose.yml` 的 `realtime-gateway` 服務已以本目錄為獨立 build context，使用本目錄的 Dockerfile（G-01 採方案 A，見 ADR-015）。
- 框架已決議為 Echo v5（≥v5.4.7），見 ADR-016；依賴尚未引入。Gateway 的傳輸層 adapter 與 WS 函式庫選擇待 Phase 05 定稿，本階段只用標準庫。
- 依賴連線（PostgreSQL／Valkey／NATS）與 readiness 檢查：TODO。
- 其餘對齊核對結果見 `docs/phase01/go-notes.md`。
