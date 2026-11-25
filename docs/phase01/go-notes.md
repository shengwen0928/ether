# Go 後端軌筆記（Phase 01，go-notes.md）

狀態：草案／待協調員審查。本軌（Go 後端）交付範圍：三個 Go 服務骨架、Migration 骨架、兩份比較備忘錄、本筆記。本機 VM 無 go／docker／psql，一切建置驗證皆為「待 CI 驗證」，只做靜態檢查。

## 1. 新增檔案清單（本軌）

### services/api（module `ether.local/api`，暫定，待 repo 建立後修訂）

| 路徑 | 用途 |
|---|---|
| `go.mod`、`go.sum`（空） | module 定義，`go 1.25`（見 §4.1）；零外部依賴，故 `go.sum` 為空檔（Dockerfile `COPY go.mod go.sum` 要求檔案存在，見 §4.5） |
| `README.md` | 建置／執行指令、模組地圖、缺口、框架決策阻塞說明 |
| `cmd/api/main.go` | 主程式：設定載入、僅註冊 `GET /healthz`、優雅關停 |
| `cmd/api/main_test.go` | healthz 測試骨架＋「無業務路由」否定測試（待 CI 驗證） |
| `internal/transport/router.go` | `Router` 介面（`Handle(method, path, handler)`）＋標準庫 `StdRouter`；檔頭明載 Gin／Echo adapter 替換規則 |
| `internal/transport/envelope.go` | `ERROR_CODES.md` §5.1 草案錯誤信封寫出器＋`request_id` 產生器 |
| `internal/config/config.go` | 環境變數載入：compose 組裝位址＋`_OVERRIDE` 覆寫，缺值明確報錯 |
| `internal/secrets/secrets.go` | Secrets Provider 介面＋環境變數實作＋OpenBao stub（未設定回傳 `not configured`，永不假裝讀到 secret） |
| `internal/telemetry/telemetry.go` | 註解型 stub：Logger 介面＋NoopLogger，僅環境變數名，不得 import SDK |
| `internal/health/health.go` | `GET /healthz` handler（只含 status／service／version） |
| `internal/<模組>/doc.go`（22 個） | 模組邊界文件（見 §2），本階段只放 doc.go，不得寫功能程式碼 |
| `migrations/MIGRATIONS.md` | 版本化慣例（命名、只往前、破壞性須備份註記）、工具選用說明（待定稿） |
| `migrations/000001_baseline.up.sql`／`.down.sql` | 基線：僅建 `schema_meta` 佔位表（含 `version` 欄），不建 64 張業務表 |

### services/realtime-gateway（module `ether.local/realtime-gateway`，暫定）

| 路徑 | 用途 |
|---|---|
| `go.mod`、`go.sum`（空） | 同上，`go 1.25` |
| `README.md` | 建置／執行指令、模組地圖、對齊問題（`cmd/gateway` 衝突、compose context） |
| `cmd/realtime-gateway/main.go` | HTTP 骨架＋`GET /healthz`＋WS 佔位（`/ws/v1` 回 501，不得假握手） |
| `cmd/realtime-gateway/main_test.go` | healthz＋WS 501 佔位測試骨架（待 CI 驗證） |
| `internal/health/health.go` | `GET /healthz` handler（與 api 的同名套件為獨立 module 下有意的重複，見 §4.6） |

### services/worker（module `ether.local/worker`，暫定）

| 路徑 | 用途 |
|---|---|
| `go.mod`、`go.sum`（空） | 同上，`go 1.25` |
| `README.md` | 建置／執行指令、模組地圖、worker 健康檢查對齊問題 |
| `cmd/worker/main.go` | `/healthz` 佔位＋優雅關停，不消費佇列、不處理任務 |
| `cmd/worker/main_test.go` | healthz 測試骨架（待 CI 驗證） |
| `internal/health/health.go` | `GET /healthz` handler（有意的重複，見 §4.6） |
| `internal/jobs/doc.go` | 任務定義佔位文件（無任務程式碼，NATS 消費者待 Phase 05） |

### docs/phase01（本軌）

| 路徑 | 用途 |
|---|---|
| `gin-vs-echo-memo.md` | Gin vs Echo 比較備忘錄（查證日 2026-11-25，推薦 Echo v5，決定權在使用者） |
| `vector-store-memo.md` | Qdrant vs pgvector 比較備忘錄（查證日 2026-11-25，推薦 pgvector，決定權在使用者） |
| `go-notes.md`（本檔案） | 軌道筆記：檔案清單、模組邊界對照、compose 一致性核對、缺口、阻塞項 |

### 本軌未建立／未修改

- 未建立 `.github/` 目錄，未 push，未建 repo／PR／Release；成果只留本機。
- 未修改任何已批准的 Phase 00 文件（`SPEC-SOURCE.md`、docs 下既有文件、`docker-compose.yml`、`.env.example` 皆未動；`.env.example` 的 Phase 01 附加段落為其他軌道所加，本軌僅引用其變數名，未編輯該檔）。
- 未修改其他軌道的檔案（`services/*/Dockerfile`、`.dockerignore`、`docs/phase01/` 下的 `flutter-notes.md`／`react-notes.md`／`compose-alignment-notes.md`／`secrets-management.md`／`observability-wiring.md` 皆為他軌所有，本軌只讀取以核對一致性）。
- 未碰 `~/workspace/messenger-app/` 一字。

## 2. 模組邊界對照（MODULE_BOUNDARIES.md）

22 個模組各有一個 `internal/<目錄>/doc.go`，內容為責任邊界、擁有資料表、允許依賴方向、禁止事項、所屬階段。目錄名沿用 `SPEC-SOURCE.md` §26 的短名慣例；§26 只列 12 個目錄，其餘 10 個依同一慣例補齊（`MODULE_BOUNDARIES.md` §6 要求 Phase 01 沿同一慣例補齊並以 ADR 或 Phase 01 文件確認），對照如下，待 ADR／Phase 01 文件確認，不得出現同一模組兩個目錄或兩個模組共用一個目錄。

| 模組（MODULE_BOUNDARIES.md） | 目錄 | 擁有資料表（§20） | 所屬階段（ROADMAP.md） |
|---|---|---|---|
| Authentication | `auth` | `user_sessions`、`user_security_keys` | Phase 03 |
| User | `user` | `users`、`user_settings` | Phase 03 |
| Profile | `profile` | `user_profiles` | Phase 03 |
| Friend | `friend` | `friendships`、`friend_requests`、`blocked_users`、`hidden_users` | Phase 04 |
| Contact | `contact` | `contact_sync_records` | Phase 04 |
| Chat | `chat` | `conversations`、`conversation_members`、`conversation_settings` | Phase 05 |
| Message | `message` | `messages`、`message_receipts`、`message_reactions`、`message_attachments`、`message_replies`、`message_forwards`、`message_pins` | Phase 05 |
| Group | `group` | `groups`、`group_members`、`group_roles`、`group_invitations`、`group_announcements`、`group_settings` | Phase 06 |
| Community | `community` | `communities`、`community_members`、`community_channels`、`community_roles`、`community_invitations` | Phase 06 |
| Media | `media` | `media_files`、`media_uploads`、`media_thumbnails`、`media_access_logs` | Phase 06 |
| Sticker | `sticker` | `sticker_packs`、`stickers`、`user_stickers` | Phase 08 |
| Call | `call` | `calls`、`call_participants`、`call_events` | Phase 07 |
| Notification | `notification` | `notifications`、`notification_preferences`、`push_tokens`、`notification_deliveries` | Phase 05–06 |
| Search | `search` | （無自有 PostgreSQL 表；Meilisearch 索引投影） | Phase 06 起 |
| Official Account | `officialaccount` | `official_accounts`、`official_account_members` | Phase 08 |
| Bot | `bot` | `bot_applications`、`bot_webhooks`、`bot_events` | Phase 08 |
| Timeline | `timeline` | `posts`、`post_media`、`post_likes`、`post_comments`、`post_shares`、`post_bookmarks` | Phase 08 |
| Payment Integration | `payment` | `sticker_purchases` | Phase 08 |
| Moderation | `moderation` | `reports`、`moderation_actions` | Phase 08–10 |
| Administration | `admin` | `admin_users`、`admin_roles`、`admin_permissions`、`admin_audit_logs` | Phase 10–11 |
| Analytics | `analytics` | （無自有 PostgreSQL 表） | Phase 11 |
| Device Management | `device` | `user_devices` | Phase 03 |

對帳：上表 22 行，表數 2＋2＋1＋4＋1＋3＋7＋6＋5＋4＋3＋3＋4＋0＋2＋3＋6＋1＋2＋4＋0＋1＝64，與 `MODULE_BOUNDARIES.md` §4 的 64 張對帳一致（`sticker_purchases` 歸 Payment Integration，`user_settings` 歸 User，見該文件 §4 註記）。依賴 DAG 逐模組見各 `doc.go` 檔頭；本階段無功能程式碼，DAG 尚未以 import 關係落為 CI 門檻（`MODULE_BOUNDARIES.md` §2 要求 Phase 01 以靜態檢查落為 CI 門檻，待 CI 軌接手）。

## 3. 與 docker-compose.yml 的一致性核對結果

核對基準：`infrastructure/docker/docker-compose.yml`（`name: ether`）、`.env.example`。完整 18 服務對照見他軌的 `docs/phase01/compose-alignment-notes.md`（G-01～G-08）；本節只列 Go 軌直接相關的核對結果，編號沿用該文件的 G 編號，不得重複裁決。

| 核對項 | 結果 |
|---|---|
| api 健康檢查 | 一致：compose `wget http://localhost:8080/healthz`；骨架監聽 `API_PORT`（預設 8080），`GET /healthz` 回 `status/service/version`。`wget` 由 Dockerfile 執行階段的 busybox 提供（見 api Dockerfile 註解）。 |
| realtime-gateway 健康檢查 | 一致：compose `wget http://localhost:8081/healthz`；骨架監聽 `GATEWAY_PORT`（預設 8081），`GET /healthz` 同上。 |
| worker 健康檢查 | **不一致／缺口（G-08）**：compose 未定義 worker healthcheck，worker Dockerfile 亦不設 HEALTHCHECK；本軌依任務要求仍實作 `/healthz` 佔位（`WORKER_PORT` 佔位名，預設 8082，compose 未設定），現階段不被探測。worker 是否保留 HTTP 端點待後階段定稿。 |
| api 依賴位址 | 一致：`config.Load` 要求的六個變數（DATABASE_URL、VALKEY_URL、NATS_URL、MINIO_ENDPOINT、MEILISEARCH_URL、KEYCLOAK_URL）與 compose 的 api `environment` 逐名一致；缺值明確報錯。`_OVERRIDE` 覆寫慣例與 `.env.example` Phase 01 附加段落一致（覆寫優先於組裝值）。 |
| gateway 依賴位址 | 一致（範圍）：compose 為 gateway 組裝三項（DATABASE_URL、VALKEY_URL、NATS_URL）；gateway 骨架本階段不讀取依賴位址（無連線），Phase 05 接線時須按此範圍載入，不得自行假設 KEYCLOAK_URL 等未組裝的值。 |
| worker 依賴位址 | 一致（範圍）：compose 為 worker 組裝五項（無 KEYCLOAK_URL）；worker 骨架本階段不讀取，Phase 05 接線時同上。 |
| 埠號變數名 | `API_PORT`／`GATEWAY_PORT` 取自 `.env.example` 末節既有註解行；`WORKER_PORT` 為本軌佔位名（`.env.example` 附加段落亦註明僅為骨架佔位名，compose 未設定）。 |
| Secrets／Telemetry 變數名 | 與他軌文件一致，未自創新名：`OPENBAO_ADDR`／`OPENBAO_TOKEN`（`.env.example` 上半部註解行，見 `secrets-management.md`）；`OTEL_SERVICE_NAME`／`OTEL_EXPORTER_OTLP_ENDPOINT`／`LOKI_ENDPOINT`／`SENTRY_DSN`（`.env.example` Phase 01 附加段落，見 `observability-wiring.md`）。compose 未向應用服務注入上述變數（見 compose-alignment-notes.md 補充核對），是否注入待裁決；本骨架的讀取皆為「未設定即停用／報錯」，不得假裝已接線。 |
| 建置參照 | **不一致（G-01，須裁決）**：compose 的三個應用服務皆以 `services/api` 為 build context，分別參照 `Dockerfile`／`Dockerfile.gateway`／`Dockerfile.worker`（後兩檔不存在）；本軌依任務要求建三個獨立 module（`services/api`、`services/realtime-gateway`、`services/worker`），各目錄的獨立 Dockerfile 已由他軌建立。裁決選項見 compose-alignment-notes.md §3（G-01），本軌不選邊。 |
| Gateway 建置路徑 | **已更正（協調員 2026-11-25 實查）**：`services/realtime-gateway/Dockerfile` 建置路徑為 `./cmd/realtime-gateway`，與本軌一致；先前「Dockerfile 假設 `./cmd/gateway`」的說法已過時、作廢。剩餘待裁決者僅 compose context 對齊缺口 G-01。 |
| api Dockerfile 與 migrations | **缺口**：api Dockerfile 只複製 `cmd/` 與 `internal/`，未複製 `migrations/`；migration 檔如何到達資料庫（獨立 Job、init container、複製進映像）待 Phase 03 定稿（見 `migrations/MIGRATIONS.md` §5）。 |
| go.sum | **缺口（已補空檔）**：三個 Dockerfile 皆 `COPY go.mod go.sum`；零依賴的 module 本無 `go.sum`，本軌已各補一個空 `go.sum` 供 COPY 使用。CI 須驗證空 `go.sum` 可建置；有依賴加入後須以 `go mod tidy` 重新產生，不得手寫。 |
| Healthcheck 語義 | **缺口**：骨架 `/healthz` 只證明 liveness；`SYSTEM_ARCHITECTURE.md` §8.3 要求 liveness／readiness 分離、就緒檢查含依賴連通性、降級策略待 Phase 01 定案。compose 以同一端點作 `service_healthy` 判定，現階段等同 liveness，不得解讀為 readiness。 |

## 4. 設計決策記錄（本軌所作，待審查）

1. **Go directive 1.25**：於 2026-11-25 固定，與三個 Dockerfile 的 `golang:1.25.0-alpine3.21` 一致。Gin v1.12.0 的 Go 要求在來源間說法不一（1.25+ vs 1.26+，見 gin-vs-echo-memo.md §2），若決定採用 Gin，CI 須驗證；Echo v5 的要求亦同。
2. **三個獨立 module，`ether.local/<服務名>`**：暫定佔位 path，待 repo 建立後修訂；`internal` 套件不得跨 module import。
3. **傳輸層隔離**：`Router` 介面＋`StdRouter`（標準庫 ServeMux，Go 1.22+ 的 `METHOD /path` 模式）；handler 只用標準庫型別，不得 import 框架套件；Gin／Echo adapter 未來實作同一介面。
4. **健康檢查回應**：只含 `status`／`service`／`version`；版本值 `0.1.0-placeholder`（版號規則待 `ROADMAP.md` §4.1 定稿）；成功回應包裝規則待 `API_STANDARDS.md` 定稿，本形式不得視為已定合約。
5. **WS 佔位**：路徑 `/ws/v1`（`WEBSOCKET_PROTOCOL.md` §2 草案方向，待定稿）；錯誤碼 `ETHER_NOT_IMPLEMENTED` 為佔位值——`ERROR_CODES.md` 無 501／NOT_IMPLEMENTED 定稿碼，本碼僅遵循 `ETHER_` 前綴格式，待 Phase 05 定稿。
6. **`internal/health` 三處重複**：有意為之（獨立 module 間不得 import 他方 internal）；共用抽取方式待後續決定，不得擅自新增第四個 module。
7. **Migration 基線**：採 `schema_meta` 佔位表而非空基線（理由見 up.sql 檔頭）；`schema_meta` 不是 §20 業務表。
8. **Worker `/healthz`**：依任務要求實作，與 worker Dockerfile「不假設 HTTP 端點」的立場衝突，已列為 G-08 相關缺口，待協調員裁決（見 §3）。

## 5. 靜態驗證結果（本機執行，無 go／docker／psql）

- 檔案結構：22 個模組 `doc.go`＋1 個 `jobs/doc.go`，共 23 個，逐一存在；三個 `go.mod` 的 module path 與 `go 1.25` directive 已核對；37 個 Go 檔案的大括號／小括號數量平衡（粗查，非編譯驗證）。
- 引用一致性：三個服務註冊的路由只限 `GET /healthz`（gateway 另有 WS 佔位路徑）；程式碼中無業務前綴路由，`"/api/v1/users"` 只出現在否定測試中（斷言非 200）；程式碼中無 §21 的 15 個 WS 事件名（事件實作待 Phase 05）。
- 框架隔離：`gin-gonic`、`labstack/echo`、`gorilla/websocket` 在 Go 程式碼中零引用（grep 無命中，測試檔除外之說明：測試檔亦無）；`go.mod` 零外部依賴。
- 憑證掃描：新檔案中無 `change-me` 字面值、無 `ghp_`／`sk-`／`AKIA` 式金鑰字面值、無私鑰區塊；secret 值只經環境變數／OpenBao 介面，錯誤訊息不含 secret 值。
- §21 逐字核對：骨架不註冊任何 §21 前綴（18 個）與事件（15 個），故無逐字引用可錯；前綴與事件的逐字核對責任在各階段 API／WS 文件落地時，本軌只在註解與備忘錄中引用文件名，不重寫前綴／事件清單。
- **未驗證（誠實聲明）**：`go build`、`go vet`、`go test` 皆未執行（無工具鏈）；Docker 建置、compose 啟動、PostgreSQL 套用 migration 皆未執行（無 docker／psql）。全部標示「待 CI 驗證」，不得宣稱通過。

## 6. 已知缺口

1. 業務 handler、認證／授權、資料庫／Valkey／NATS／MinIO／Meilisearch／Keycloak 連線：全部 TODO。
2. Readiness 檢查與降級策略：TODO（見 §3 健康檢查語義）。
3. Secrets Provider 選擇方式（env／OpenBao 切換）與 OpenBao 正式接線（KV 路徑、Policy、解封流程）：待部署定稿（見 `secrets-management.md`）。
4. Telemetry SDK 接線（OTLP Collector 服務尚未在 compose 定義，見 compose-alignment-notes.md G-06）：TODO。
5. Migration 工具選用、執行主體、CI 整合：待定稿（見 `migrations/MIGRATIONS.md` §5）。
6. 模組 DAG 的 CI 靜態檢查門檻（import 關係掃描）：待 CI 軌接手（`MODULE_BOUNDARIES.md` §2）。
7. Outbox 表名與欄位未定（`DATA_FLOW.md` §11 缺口 1），不得預寫 migration。
8. 兩份備忘錄的版本數字以 2026-11-25 查證為準，決定時須覆核。

## 7. 阻塞項（須使用者／協調員決定，本軌不得擅自決定）

1. **Gin／Echo 二選一**：比較與推薦見 `gin-vs-echo-memo.md`（推薦 Echo v5），最終決定權在使用者；決定後以 ADR 記錄。
2. **Qdrant／pgvector 二選一**：比較與推薦見 `vector-store-memo.md`（推薦 pgvector），最終決定權在使用者；決定後以 ADR 記錄，並觸發 compose 修訂（qdrant 服務去留／資料庫映像更換）。
3. **建置佈局裁決（G-01）**：三個獨立 context  vs 單一 `services/api` context（`Dockerfile.gateway`／`Dockerfile.worker` 尚不存在），待協調員裁決。
4. **Gateway 建置路徑**：已更正——`cmd/realtime-gateway`（本軌）與 gateway Dockerfile 實查一致，`cmd/gateway` 說法作廢；待裁決者僅 G-01 compose 佈局。
5. **Worker HTTP 端點存廢（G-08 相關）**：保留 `/healthz` 佔位 vs 依 worker Dockerfile 立場移除 HTTP 端點，待後階段定稿。
6. **CI 環境**：Go 1.25 工具鏈、Docker、PostgreSQL 皆須在 CI 具備，否則本骨架的建置／測試／migration 驗證無法完成；本機 VM 無上述工具，不得以本機狀態冒充 CI 結果。
7. **Repo 建立與 module path 修訂**：`ether.local/*` 為暫定佔位，待 repo 建立後修訂三個 `go.mod`（與 Dockerfile 的建置 context 一併覆核）。

## 8. 2026-11-25 決議修訂（dated amendment，原文不動）

- **三項選型已決議**：G-01 採用方案 A（獨立 Dockerfile 與獨立建置 context，見 ADR-015）；後端框架採用 Echo v5（版本不得低於 v5.4.7，見 ADR-016）；向量承載採用 pgvector（見 ADR-017）。
- **§7 阻塞項第 1–3 項已決議**：Gin／Echo 二選一、Qdrant／pgvector 二選一、建置佈局裁決（G-01）皆已定案，後續修改引用 ADR；其餘阻塞項維持追蹤。
- **G-02／G-03 處理方案**：見執行計畫 §4（LiveKit profiles 隔離、Prometheus 最小有效設定、開發模式佔位維持追蹤）。
- 兩份比較備忘錄保留為決策證據；與 ADR-016／ADR-017 牴觸的技術描述以 ADR 為準。
