# Compose 對照筆記（Phase 01，Infra 軌）

狀態：靜態逐項核對，待 CI 或有 Docker 的環境驗證。本文件只核對、不修改
`infrastructure/docker/docker-compose.yml`；列出的不一致與缺口待協調員裁決，
不得由 Infra 軌擅自改 compose。

核對基準：`infrastructure/docker/docker-compose.yml`（451 行，`name: ether`）、
`SPEC-SOURCE.md` §26（目錄規範）、`docs/deployment/CLOUD_ARCHITECTURE.md` §5
（服務清單）、`docs/deployment/SELF_HOSTING.md` §3／§9（啟動順序與缺口）。

## 1. 服務總數

Compose 共 **18 個服務**（`make verify-static` 以 PyYAML 解析覆核為 18）：

`traefik`、`api`、`realtime-gateway`、`worker`、`postgresql`、`valkey`、`nats`、
`minio`、`meilisearch`、`qdrant`、`openbao`、`keycloak`、`livekit`、`ollama`、
`prometheus`、`grafana`、`loki`、`glitchtip`。

與 CLOUD_ARCHITECTURE.md §5 的 18 行服務清單逐名一致（§5 另以文字說明
Worker／vLLM 等，見 §3 缺口）。

## 2. 逐項對照

`build` 欄的 context 以 compose 檔所在目錄（`infrastructure/docker/`）為相對基準；
`../../services/api` 即 `~/workspace/ether/services/api`。

| # | 服務 | build／image | healthcheck | depends_on（service_healthy） | 環境變數（compose 組裝） | Phase 01 骨架對應 |
|---|---|---|---|---|---|---|
| 1 | `traefik` | `traefik:v3`，無 build | `traefik healthcheck --ping` | — | —（ACME email 經 command 參照 `${ACME_EMAIL}`） | 無 Dockerfile（基礎設施服務，正確）；見缺口 G-05（ping 設定待驗證） |
| 2 | `api` | build context `../../services/api`，dockerfile `Dockerfile`；image `ether-api:dev` | `wget http://localhost:8080/healthz` | postgresql、valkey、nats、minio、meilisearch、keycloak | DATABASE_URL、VALKEY_URL、NATS_URL、MINIO_ENDPOINT、MEILISEARCH_URL、KEYCLOAK_URL | `services/api/Dockerfile` 路徑一致；HEALTHCHECK 路徑一致（8080／healthz）；Go 骨架已由程式碼軌建立（`cmd/api`、`internal/health` 的 `/healthz` liveness handler，依賴檢查為 TODO），編譯未驗證 |
| 3 | `realtime-gateway` | build context `../../services/api`，dockerfile `Dockerfile.gateway`；image `ether-realtime-gateway:dev` | `wget http://localhost:8081/healthz` | postgresql、valkey、nats、api | DATABASE_URL、VALKEY_URL、NATS_URL | **不一致 G-01**：compose 未參照 `services/realtime-gateway/Dockerfile`，而是 `services/api/Dockerfile.gateway`（檔案尚不存在）；HEALTHCHECK 路徑一致（8081／healthz）；Go 骨架已由程式碼軌建立（`cmd/realtime-gateway`、`GATEWAY_PORT`，預設 8081），編譯未驗證 |
| 4 | `worker` | build context `../../services/api`，dockerfile `Dockerfile.worker`；image `ether-worker:dev` | **未定義** | postgresql、valkey、nats、minio、meilisearch | DATABASE_URL、VALKEY_URL、NATS_URL、MINIO_ENDPOINT、MEILISEARCH_URL | **不一致 G-01**：compose 未參照 `services/worker/Dockerfile`，而是 `services/api/Dockerfile.worker`（檔案尚不存在）；worker Dockerfile 依規不自創 HEALTHCHECK；Go 骨架雖有 `/healthz` 佔位 handler（`WORKER_PORT` 佔位名，compose 未設定），現階段不被探測，見 G-08 |
| 5 | `postgresql` | `postgres:16` | `pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}` | — | POSTGRES_DB／USER／PASSWORD | 無 Dockerfile（正確）；volume `postgres-data` |
| 6 | `valkey` | `valkey/valkey:8` | `valkey-cli ping \| grep PONG` | — | — | 無 Dockerfile（正確）；volume `valkey-data` |
| 7 | `nats` | `nats:2` | `wget http://localhost:8222/healthz` | — | — | 無 Dockerfile（正確）；volume `nats-data` |
| 8 | `minio` | `minio/minio`（無標籤） | `curl http://localhost:9000/minio/health/ready` | — | MINIO_ROOT_USER／PASSWORD | 無 Dockerfile（正確）；volume `minio-data`；見 G-04 |
| 9 | `meilisearch` | `getmeili/meilisearch:v1` | `curl http://localhost:7700/health` | — | MEILI_MASTER_KEY | 無 Dockerfile（正確）；volume `meilisearch-data` |
| 10 | `qdrant` | `qdrant/qdrant`（無標籤） | `curl http://localhost:6333/healthz` | — | — | 無 Dockerfile（正確）；向量承載選擇待定（Qdrant 或 pgvector），未選定前不得假設為主力；volume `qdrant-data` |
| 11 | `openbao` | `openbao/openbao`（無標籤） | `curl http://localhost:8200/v1/sys/health` | — | — | 無 Dockerfile（正確）；開發模式佔位，見 G-03；volume `openbao-data` |
| 12 | `keycloak` | `quay.io/keycloak/keycloak:26` | `curl http://localhost:8080/health/ready` | postgresql | KC_DB、KC_DB_URL、KC_DB_USERNAME、KC_DB_PASSWORD、KC_HOSTNAME、KEYCLOAK_ADMIN、KEYCLOAK_ADMIN_PASSWORD | 無 Dockerfile（正確）；`start-dev` 開發模式佔位，見 G-03 |
| 13 | `livekit` | `livekit/livekit-server`（無標籤） | `wget http://localhost:7880/`（草案，待 Phase 07 定稿） | valkey | LIVEKIT_KEYS | 無 Dockerfile（正確）；配置檔 `./livekit.yaml` 尚未建立，見 G-02 |
| 14 | `ollama` | `ollama/ollama`（無標籤） | `curl http://localhost:11434/api/tags` | — | — | 無 Dockerfile（正確）；模型清單與 GPU 需求待 Phase 09；vLLM 容器未預寫，見 G-06 |
| 15 | `prometheus` | `prom/prometheus`（無標籤） | `wget http://localhost:9090/-/healthy` | — | — | 無 Dockerfile（正確）；配置檔 `./prometheus.yml` 尚未建立，見 G-02 |
| 16 | `grafana` | `grafana/grafana`（無標籤） | `curl http://localhost:3000/api/health` | prometheus、loki | GF_SECURITY_ADMIN_USER／PASSWORD | 無 Dockerfile（正確） |
| 17 | `loki` | `grafana/loki`（無標籤） | `curl http://localhost:3100/ready` | — | — | 無 Dockerfile（正確）；以容器內建 `local-config.yaml` 啟動 |
| 18 | `glitchtip` | `glitchtip/glitchtip`（無標籤） | `wget http://localhost:8000/`（草案，待驗證後定稿） | postgresql、valkey | DATABASE_URL、REDIS_URL（指向 Valkey，見 compose 註解）、SECRET_KEY | 無 Dockerfile（正確） |

補充核對：

- **網路**：`ether-front`（對外，僅 traefik 掛載）與 `ether-back`
 （`internal: true`，全部 18 服務中除 traefik 外皆只掛 `ether-back`；
  traefik 同時掛兩網）。與 CLOUD_ARCHITECTURE.md §4 拓撲（客戶端只與邊緣層
  對話）一致。
- **Volumes**：12 個命名 volume（`traefik-letsencrypt`、`postgres-data`、
  `valkey-data`、`nats-data`、`minio-data`、`meilisearch-data`、`qdrant-data`、
  `openbao-data`、`ollama-data`、`prometheus-data`、`grafana-data`、`loki-data`），
  與 SELF_HOSTING.md §6 清單一致。
- **Traefik 路由標籤**：`api`（`api.${DOMAIN}`:8080）、`realtime-gateway`
  （`ws.${DOMAIN}`:8081）、`keycloak`（`auth.${DOMAIN}`:8080）、`grafana`
  （`grafana.${DOMAIN}`:3000），與 SELF_HOSTING.md §5 子網域草案一致；
  `worker` 無路由標籤、無對外埠（正確，背景任務）。
- **應用服務環境變數**：`api`／`realtime-gateway`／`worker` 的 compose
  `environment` 只含資料庫／快取／佇列／儲存／搜尋／身分組裝值，**未注入**
  Phase 01 附加的 `_OVERRIDE`／`OTEL_*`／`LOKI_ENDPOINT`／`SENTRY_DSN`／OpenBao 客戶端變數。
  變數名保留不等於已接線；是否注入 compose 待裁決（不得擅自改 compose）。
- **埠號**：對外埠只有 traefik（80／443）與 livekit（7880／7881／7882-udp，
  RTC 埠範圍待 Phase 07 評估）；應用與資料服務皆無主機埠對映，與內部網路
  設計一致。

## 3. 不一致與缺口清單

### G-01（須裁決）：gateway／worker 的建置參照與獨立 Dockerfile 路徑不一致

- 現況：compose 的 `realtime-gateway`／`worker` 以 `services/api` 為 context，
  分別參照 `Dockerfile.gateway`／`Dockerfile.worker`；該兩檔目前**不存在**。
- Phase 01 任務要求的獨立路徑 `services/realtime-gateway/Dockerfile` 與
  `services/worker/Dockerfile` 已建立（含 `.dockerignore` 各一份），但 compose
  目前不會使用它們。
- 影響：`docker compose build realtime-gateway worker` 在兩檔補齊前失敗；
  即使補齊，獨立 Dockerfile 與 `Dockerfile.gateway`／`Dockerfile.worker`
  兩套檔案並存即 drift 來源。
- 裁決選項（Infra 軌不選邊）：(A) compose 改為參照三個獨立 context／
  Dockerfile；(B) 維持單一 `services/api` context，gateway／worker 改以
  `Dockerfile.gateway`／`Dockerfile.worker` 為準，獨立目錄的 Dockerfile 撤回
  或改為文件參照。裁決前，獨立 Dockerfile 檔頭已註明此缺口。

### G-02（Phase 00 已知缺口）：參照的配置檔尚未建立

- `./livekit.yaml`（livekit 服務掛載，待 Phase 07 定稿後建立）與
  `./prometheus.yml`（prometheus 服務掛載，待監控文件定稿後建立）皆不存在；
  建立前兩服務無法啟動。見 SELF_HOSTING.md §9 第 1–2 項。

### G-03（Phase 00 已知缺口）：開發模式佔位不得用於正式環境

- `openbao` 以 `server -dev`、`keycloak` 以 `start-dev` 佔位；正式模式、
  解封流程、正式啟動參數皆待定稿。見 SELF_HOSTING.md §9 第 4–5 項與
  `docs/phase01/secrets-management.md` §5。

### G-04：映像標籤未固定，與版本固定要求不一致

- 9 個服務無任何標籤（implicit latest）：`minio`、`qdrant`、`openbao`、
  `livekit`、`ollama`、`prometheus`、`grafana`、`loki`、`glitchtip`。
- 其餘服務只固定到 major（`traefik:v3`、`nats:2`、`meilisearch:v1`、
  `keycloak:26`、`valkey:8`、`postgres:16`）；無一服務固定至 patch 版本。
- 與 CLOUD_ARCHITECTURE.md §3.1「容器映像版本須固定，不得用浮動標籤」不一致；
  標籤定稿待有 Docker 的環境驗證後進行（compose 檔頭已有相同註記）。

### G-05（待驗證）：traefik healthcheck 的 ping 設定

- compose 的 traefik `command` 未見 ping 入口設定，但 healthcheck 執行
  `traefik healthcheck --ping`。是否能在現行 command 下通過，待有 Docker 的
  環境驗證，不得預斷。

### G-06：compose 中不存在的服務（架構文件提及，compose 未定義）

- **Coraza WAF**：ADR-013 將 Coraza 列為 Compose 組合的一部分，
  CLOUD_ARCHITECTURE.md §4 拓撲亦置於 Traefik 前置，但 compose 無 coraza
  服務、無對應配置；部署形態待定稿。
- **vLLM**：compose 未預寫 vLLM 容器，服務形態待 Phase 09 定稿
  （SELF_HOSTING.md §9 第 3 項）。
- **OTel Collector**：ADR-012 的 OpenTelemetry 接入無對應 compose 服務；
  Collector 形態待定稿（見 observability-wiring.md §4）。
- **Web／Admin**：拓撲中的 Web／Admin（React 靜態）無 compose 服務；
  屬靜態部署抑或容器，待後階段定稿，本筆記只記錄現況。

### G-07：compose 檔案位置與 §26 字面不一致

- SPEC-SOURCE.md §26 將 `docker-compose.yml` 列於 `ether/` 根目錄層級；
  實際檔案位於 `infrastructure/docker/docker-compose.yml`，根目錄並無
  compose 檔。Makefile 與 SELF_HOSTING.md 皆以 `infrastructure/docker/` 路徑
  為準。是否補根目錄參照或修訂 §26 敘述，待協調員裁決（Infra 軌不移動檔案）。

### G-08：worker 無 healthcheck（現況記錄，非錯誤）

- compose 未為 `worker` 定義 healthcheck；無其他服務以 `service_healthy`
  依賴 worker，故不阻塞啟動順序。程式碼骨架雖有 `/healthz` 佔位 handler
  （`internal/health`，`WORKER_PORT` 佔位名，compose 未設定），現階段不被
  compose 或 Dockerfile 探測；worker 是否保留 HTTP 端點、是否補 healthcheck，
  待後階段定稿；Phase 01 的 worker Dockerfile 不自創 HEALTHCHECK，與 compose 一致。

## 4. Phase 00 已知缺口重申（本筆記未新增，僅對照）

`livekit.yaml`、`prometheus.yml` 未建、vLLM 未預寫、OpenBao／Keycloak 正式模式
待定稿、全部 Health Check 指令與映像標籤為草案，皆為 SELF_HOSTING.md §9 已列
缺口，本筆記的 G-02／G-03／G-04／G-06 與之對應，未擅自補決定。

## 5. 2026-11-25 決議修訂（dated amendment，原文不動）

- **G-01 已裁決方案 A**：`api`、`realtime-gateway`、`worker` 各自以自己的服務目錄為建置 context、各用目錄內的 `Dockerfile`，刪除共用 context 參照（見 ADR-015）。本文件 §3 G-01 的裁決選項敘述保留為核對紀錄，選項 B 未採納。
- **框架與向量庫已決議**：後端框架採用 Echo v5（版本不得低於 v5.4.7，見 ADR-016）；向量承載採用 pgvector（見 ADR-017）。
- **qdrant 服務已移除**：Compose 服務總數 18 → **17**，`qdrant-data` volume 一併移除。本文件 §1／§2 含 `qdrant` 的 18 服務清單保留為核對時點的原始紀錄。
- **G-02 按執行計畫 §4 處理**：`livekit` 加入 `profiles: ["phase07"]`（預設不啟動）；`prometheus` 補最小有效設定（僅自我抓取）。
- **G-04 清單減 qdrant 一項**：無標籤服務 9 → 8；其餘 G-03～G-08 維持追蹤。
