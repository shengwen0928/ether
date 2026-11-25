# ETHER

ETHER（Every Thread Holds Endless Remembrance）— A Human Connection Platform。
全功能跨平台即時通訊生態系統：Flutter＋Dart（Android／iOS／Windows／macOS／Web）、
React＋TypeScript（Web／Admin）、Go 後端（PostgreSQL、Valkey、NATS JetStream、
WebSocket、gRPC、S3 相容儲存、搜尋），WebRTC＋LiveKit 通話、AI Provider Adapter。

> **狀態：Phase 01 基礎建設骨架，待 CI 或有 Docker 的環境驗證。**
> 本機 VM 無 docker、無 go、無 flutter，Dockerfile 建置、Compose 啟動、CI 執行
> 皆未驗證，不得宣稱已完成或已驗證。功能描述以 `docs/PRD.md` 與
> `SPEC-SOURCE.md` 為準；本 README 不描述任何已完成的功能。

## Monorepo 結構（SPEC-SOURCE.md §26）

```text
ether/
├── SPEC-SOURCE.md            # 規格來源（§26 目錄規範、§27 Agent 執行規則、手冊 C 摘要）
├── README.md                 # 本檔案
├── Makefile                  # build／lint／test／migrate／compose-up／verify-static（Phase 01 骨架）
├── .env.example              # 環境變數範本，全部佔位符；填入真實值的 .env 絕不進 Git
├── .gitignore
├── apps/
│   ├── mobile/               # Flutter Mobile＋Desktop（骨架由 Flutter 軌建立，見 apps/mobile/README.md）
│   ├── admin/                # React＋TypeScript 管理後台（骨架由 React 軌建立）
│   └── web/                  # React＋TypeScript Web（骨架由 React 軌建立）
├── services/
│   ├── api/                  # Go 後端骨架（獨立 go.mod、cmd/api、22 模組 doc.go、migrations 基線）＋ Dockerfile
│   ├── realtime-gateway/     # Realtime Gateway 骨架（/healthz、WS 501 佔位）＋ Dockerfile
│   └── worker/               # Worker 骨架（/healthz 佔位）＋ Dockerfile
├── infrastructure/
│   ├── docker/               # docker-compose.yml（18 服務，草案／待批准）
│   ├── ci-drafts/            # GitHub Actions workflow 草稿（絕對不得以 push 建立 .github/workflows/）
│   ├── monitoring/           # 尚未建立（§26 預留）
│   ├── deployment/           # 尚未建立（§26 預留）
│   └── scripts/              # 尚未建立（§26 預留）
└── docs/                     # Phase 00 架構與規格文件（見 docs/）
    ├── phase01/              # Phase 01 文件：四軌筆記、兩份比較備忘錄、secrets／可觀測性／compose 對照、本審查報告
    └── ...                   # architecture／database／api／security／deployment／testing／decisions／brand
```

注意：`services/api`、`services/realtime-gateway`、`services/worker` 的 Go
骨架（各服務獨立 `go.mod`，`go 1.25`，僅標準庫依賴、go.sum 為空檔；`cmd/`、
`internal/` 佔位、api 的 migrations baseline）皆已建立，編譯皆未驗證。Dockerfile 的建置
路徑已對照實際 `cmd/` 佈局（api：`cmd/api`；gateway：`cmd/realtime-gateway`；
worker：`cmd/worker`），仍待 CI 建置驗證。Compose 的 gateway／worker 建置
參照與獨立 Dockerfile 路徑存在對齊缺口，見
`docs/phase01/compose-alignment-notes.md`（G-01），待裁決，不得擅自修改
`docker-compose.yml`。

## 快速開始

### 前置工具鏈

- Docker 與 Docker Compose v2（部署與整合測試用；本機 VM 無 docker）
- Go 工具鏈（後端建置用；版本與 Dockerfile 固定版本對齊，見各 Dockerfile 檔頭）
- Flutter（行動／桌面應用用；版本待定稿）
- Python 3＋PyYAML（僅 `make verify-static` 靜態檢查用）

工具鏈版本與測試框架選定待 Phase 01 定稿（見 `docs/testing/TEST_STRATEGY.md` §9），
不得以本機已安裝的版本假裝已定稿。

### 指令

```sh
# 1. 靜態檢查（本機可用，只用 python3＋grep，不需 docker／go）
make verify-static

# 2. 複製環境變數範本後填入各環境實際值（實際值以 OpenBao／環境注入提供）
cp .env.example .env
# .env 絕對不要提交至 Git。

# 3. 啟動 Compose（需 Docker；分組啟動與健康檢查見 docs/deployment/SELF_HOSTING.md §3）
make compose-up

# 4. 建置／檢查（需對應工具鏈；缺工具時會以明確訊息退出，見 Makefile 檔頭）
make build
make lint
make test
make migrate   # 佔位檢查：Migration 執行者與工具待定稿，不執行任何 Migration
```

`make build`／`make lint`／`make test`／`make compose-up` 在工具鏈或程式碼
尚未就緒時會以明確訊息、非零碼退出，那是骨架階段的預期行為，不是靜默失敗。

## 環境變數

全部環境變數以 `.env.example` 為範本，全部值皆為佔位符（`change-me`／
`example.invalid`），不得含真實金鑰、真實密碼、真實 Token，不得用於正式環境。

- 網域、PostgreSQL、Valkey、NATS、MinIO、Meilisearch、pgvector（已定案）、OpenBao、Keycloak、
  LiveKit、Ollama、Grafana、GlitchTip、推播、簡訊 OTP、付款：見 `.env.example` 上半部。
- Phase 01 附加段落（檔尾，「Phase 01 附加」）：`<NAME>_OVERRIDE` 覆寫變數
  （與 `services/api/internal/config/config.go` 的 resolve 一致，不加服務前綴）、
  `OTEL_EXPORTER_OTLP_ENDPOINT`、`LOKI_ENDPOINT`／`SENTRY_DSN`、OpenBao
  客戶端變數，全部停用預設（行首加 `#`），僅保留慣例與佔位符。
- Secrets 管理慣例：見 `docs/phase01/secrets-management.md`（OpenBao KV 路徑、
  policy 命名、環境注入方式）。Production Secrets 絕對不入 Git、不寫入映像。

## 文件索引

| 文件 | 內容 |
|---|---|
| `SPEC-SOURCE.md` | 規格來源：§26 目錄規範、§27 Agent 執行規則、手冊 C–N 摘要 |
| `docs/PRD.md` | 產品需求文件 |
| `docs/ROADMAP.md` | Phase 00–12 Roadmap；Phase 01 為基礎建設（手冊 C） |
| `docs/deployment/CLOUD_ARCHITECTURE.md` | 自架部署架構（Docker Compose＋Coolify） |
| `docs/deployment/SELF_HOSTING.md` | 自架部署操作說明、啟動順序、缺口清單 |
| `docs/COST_OPTIMIZATION.md` | 成本最佳化比較；§4 仍可能付費項目，§5 部署與維運段落 |
| `docs/testing/TEST_STRATEGY.md` | 測試策略：五層、CI 範圍、驗收門檻寫法 |
| `docs/decisions/ADR-012-可觀測性自架.md` | Prometheus＋Grafana＋Loki＋OpenTelemetry＋GlitchTip |
| `docs/decisions/ADR-013-Docker-Compose與Coolify部署.md` | Docker Compose＋Coolify＋OpenTofu＋GitHub Actions |
| `docs/phase01/infra-notes.md` | Infra 軌筆記：檔案清單、靜態驗證結果、缺口、阻塞項 |
| `docs/phase01/compose-alignment-notes.md` | Compose 18 服務逐項對照與不一致清單 |
| `docs/phase01/secrets-management.md` | OpenBao 接入骨架（慣例與文件，無真實 token） |
| `docs/phase01/observability-wiring.md` | Loki＋OpenTelemetry 接入點（已接線 vs 待接線） |
| `infrastructure/ci-drafts/README.md` | CI 草稿說明與手動建立 workflow 的強制流程 |

## 目前狀態

- Phase 01 骨架：Monorepo 目錄骨架、四軌骨架（Flutter、React Web／Admin、Go 三服務、
  Dockerfile、Makefile、`.gitignore`、`.env.example` 附加段落、OpenBao／可觀測性接入文件、
  CI 草稿、Compose 對照筆記、兩份比較備忘錄）。全部編譯、建置、啟動皆未驗證。
- 驗證：僅靜態檢查（`make verify-static`：YAML 解析、Dockerfile 指令檢查、
  引用一致性、secrets 掃描）。Dockerfile 建置、Compose 啟動、CI 執行皆為
  「待 CI 或有 Docker 的環境驗證」。
- 每階段完成後停止審查，審查通過前不得進入下一階段（見 `docs/ROADMAP.md` §5）。

## 缺口與阻塞項

- Compose 對齊缺口（gateway／worker 建置參照、`livekit.yaml`／`prometheus.yml`
  未建、映像標籤未固定等）：見 `docs/phase01/compose-alignment-notes.md`。
- Infra 軌已知缺口與阻塞項總表：見 `docs/phase01/infra-notes.md`。
- 正式部署阻塞項（伺服器、網域、成本待查證）：見
  `docs/deployment/CLOUD_ARCHITECTURE.md` §3.3。
- 審查決議指引：G-01 方案 A、Echo v5、pgvector 三項決議見
  `docs/decisions/ADR-015-G-01獨立Dockerfile與建置context.md`、
  `docs/decisions/ADR-016-採用Echo-v5.md`、`docs/decisions/ADR-017-採用pgvector.md`。

## 成本說明（不得宣稱零成本）

本專案採免費／開源自架方案為主，但仍可能付費項目（網域、伺服器、簡訊 OTP、
Apple／Google 上架費、外部 AI、TURN 頻寬等）金額與計價方式皆待查證，不得預寫
數字、不得宣稱全系統零成本。詳見 `docs/COST_OPTIMIZATION.md` §4。
