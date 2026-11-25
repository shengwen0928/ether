狀態：草案／待批准

# ETHER 自架部署說明（SELF_HOSTING.md）

## 1. 依據與效力

- 本文件以 `~/workspace/ether/SPEC-SOURCE.md` 頂部「架構調整」（2026-11-25 00:33 使用者指令，優先於舊技術棧）與 `docs/deployment/CLOUD_ARCHITECTURE.md` 為準。部署形態為 Docker Compose＋Coolify，邊緣閘道為 Traefik，身分提供者為 Keycloak，Secrets 為 OpenBao，監控為 Prometheus＋Grafana＋Loki＋GlitchTip。
- 本文件為部署操作說明，不寫應用程式碼。全部步驟皆為「待 CI 或有 Docker 的環境驗證」：本機 VM 無 docker，專案尚無應用程式碼，不得宣稱已實測啟動。
- 成本一律寫「待查證」，不得宣稱全系統零成本。仍可能付費項目見 §8。
- 舊付費服務名不得殘留於正文；比較內容見 CLOUD_ARCHITECTURE.md §2。

## 2. 前置條件

- 主機：已安裝 Docker 與 Docker Compose 的 Linux 主機（主機規格、作業系統版本、CPU／RAM／磁碟需求，待查證／待 Phase 07／09 的 TURN 與 AI 評估，不得預寫數字）。本機 VM 無 docker，前置條件在本機不滿足，不得在本機執行本文件步驟。
- 網域：已擁有並可設定 DNS 的網域（網域成本待查證）；DNS 由 Cloudflare Free 管理（DNS 紀錄指向主機，代理／CDN 設定待定稿）。
- 憑證：Let's Encrypt 憑證由 Traefik 自動核發與續期（見 §5）；ACME 聯絡 Email 以 `.env` 的 `ACME_EMAIL` 提供（佔位符，不得填入未確認的真實 Email）。
- 檔案：本專案的 `infrastructure/docker/docker-compose.yml`、`.env.example`（複製為 `.env` 後填入）、`livekit.yaml`（待建立，見 §9 缺口）、`prometheus.yml`（待建立，見 §9 缺口）。
- 權限：操作者須有主機管理權限、網域 DNS 管理權限、Coolify 管理權限（如採用）；Production Secrets 的存取權限須最小化，不得共用管理帳號。

## 3. 啟動順序

啟動順序由 `docker-compose.yml` 的 `depends_on`＋`healthcheck` 保證，不得手動跳過順序啟動應用服務。順序如下（草案，待有 Docker 的環境驗證）：

1. 資料層先行：`postgresql`（含 pgvector 擴充）、`valkey`、`nats`、`minio`、`meilisearch`（`qdrant` 服務已移除，見 ADR-017）。
2. Secrets 與身分：`openbao`、`keycloak`（`keycloak` 依賴 `postgresql` 健康）。
3. 監控先行（或與應用並行，待定稿）：`prometheus`、`loki`、`grafana`（依賴 `prometheus`／`loki` 健康）、`glitchtip`（依賴 `postgresql`／`valkey` 健康）。
4. 通話與 AI：`livekit`（依賴 `valkey` 健康）、`ollama`（模型載入待 Phase 09 評估）。
5. 邊緣閘道：`traefik`（憑證核發依賴 DNS 解析與公網可達，見 §5）。
6. 應用服務：`api`、`realtime-gateway`、`worker`（依賴資料層與身分服務健康；程式碼尚無，待 Phase 01 完成 Dockerfile 後方可啟動，見 §9）。
7. 資料庫 Migration：由 `api` 啟動流程或獨立 Migration 任務執行（執行者待定稿），Migration 完成前不得接受流量；Migration 失敗時停止部署，不得以舊 Schema 接受新版流量。

啟動指令（草案，目錄為 `infrastructure/docker/`，待驗證）：

```sh
cp ../../.env.example ../../.env
# 填入 .env（佔位符改為各環境實際值；實際值以 OpenBao／環境注入提供，不得提交）
docker compose up -d postgresql valkey nats minio meilisearch openbao keycloak
docker compose up -d prometheus loki grafana glitchtip livekit ollama traefik
docker compose up -d api realtime-gateway worker
docker compose ps
docker compose logs -f traefik api realtime-gateway
```

- 不得以 `docker compose up -d` 一次啟動全部服務而不檢查健康狀態作為正式部署流程；正式部署須按順序檢查每組服務的健康狀態（見 §7 監控驗證清單）。
- 停止指令（草案）：`docker compose down`（不刪 Volume）；刪除 Volume 的指令（`docker compose down -v`）不得在正式環境執行，Volume 刪除即刪除資料，須經批准並先備份。

## 4. 備份策略

- 備份範圍：PostgreSQL 資料庫、MinIO 物件、OpenBao Secrets 備份、Keycloak 配置、Meilisearch 索引（可重建，優先級待定稿）、NATS 串流、全部命名 Volume。備份清單以 CLOUD_ARCHITECTURE.md §8.1 為準。
- 備份方法（草案，待定稿）：
  - PostgreSQL：資料庫備份工具備份（邏輯或物理，待定稿），備份檔加密後存於獨立儲存。
  - MinIO：物件複製或 Volume 快照（待定稿），Bucket 不得公開。
  - Volume：主機快照或檔案級備份（待定稿），備份前須確認服務一致性（資料庫備份與 Volume 快照的時間一致性規則待定稿）。
  - OpenBao：Secrets 備份與解封金鑰分開存放，不得同處存放，不得寫入 Git。
- 備份頻率、保留期限、RPO、RTO，一律待定稿／待查證，不得預寫數字。
- Volume 備份驗證：備份後須在隔離環境還原驗證（還原演練見 CLOUD_ARCHITECTURE.md §8.1），不得以「備份檔已產生」假裝「可還原」。
- 還原步驟（草案，待演練定稿）：停止應用服務 → 還原 PostgreSQL → 還原 MinIO 物件 → 還原 OpenBao／Keycloak 配置 → 執行 Migration 至對應版本 → 啟動服務 → 健康檢查與冒煙測試 → 監控確認。還原時不得以新版 Migration 直接套用舊備份而不經版本對照。

## 5. 網路與 SSL 配置（Cloudflare＋Let's Encrypt）

- DNS：網域的 DNS 由 Cloudflare Free 管理；紀錄指向主機公網位址（A／AAAA 紀錄，待定稿）；子網域劃分（草案）：`api.`（Go API）、`ws.`（Realtime Gateway）、`auth.`（Keycloak）、`grafana.`（Grafana），正式子網域待定稿，不得以草案子網域對外承諾。
- CDN／代理：Cloudflare Free 的 CDN／代理設定（快取規則、快取對象）待定稿；API 與 WebSocket 不得被 CDN 快取動態內容，快取規則須明定不得快取認證回應、訊息內容、E2EE 內容。
- WAF：Coraza WAF 置於 Traefik 前置（規則集、模式（阻擋／僅記錄）、誤攔處理，待定稿）；WAF 不得作為唯一安全邊界，API 層的驗證與授權仍須完整執行。
- SSL：Traefik 以 Let's Encrypt 自動核發憑證（TLS Challenge，見 docker-compose.yml）；憑證續期由 Traefik 自動處理，續期失敗須告警（告警管道待定稿）；私鑰存於 Traefik Volume（`traefik-letsencrypt`），不得寫入 Git、不得手動複製私鑰。
- 內部網路：Compose 的 `ether-back` 網路為內部網路（`internal: true`），資料層、Secrets、身分、監控服務不得直接對外暴露；僅 Traefik 的 80／443 對外，LiveKit 的 RTC 埠按 Phase 07 評估開放（埠範圍待定稿，不得預寫）。
- 強制 HTTPS：HTTP（80）一律重定向至 HTTPS（443），由 Traefik 配置保證；HSTS、TLS 版本、加密套件，待定稿，不得預寫。

## 6. Volume 備份（補充）

- 命名 Volume 清單見 docker-compose.yml：`postgres-data`、`valkey-data`、`nats-data`、`minio-data`、`meilisearch-data`、`openbao-data`、`ollama-data`、`prometheus-data`、`grafana-data`、`loki-data`、`traefik-letsencrypt`。
- 可重建 Volume（`valkey-data` 快取、`meilisearch-data` 索引、`prometheus-data` 指標）遺失時不得導致資料遺失，但須有重建流程（快取重建、索引重建、指標缺失標示），重建流程待定稿。
- 不可重建 Volume（`postgres-data`、`minio-data`、`openbao-data`）遺失即資料遺失，備份優先級最高，備份驗證最嚴格。
- Volume 的主機路徑、檔案系統、快照工具，待主機定稿；Volume 不得以主機綁定路徑（bind mount）代替命名 Volume 而不經定稿（備份與遷移規則不同）。

## 7. 升級與回滾、監控驗證清單

### 7.1 升級

- 升級流程：新版映像建置（版本固定）→ staging 部署與冒煙測試 → 備份（升級前必備份）→ production 部署（Coolify）→ 健康檢查 → 冒煙測試 → 監控確認。任一環節失敗停止升級並回滾。
- Migration 升級：Migration 須先於應用升級執行（或隨應用啟動執行，待定稿），破壞性 Migration 須經批准並先有備份與還原驗證。
- 配置升級：Keycloak 配置、Traefik 配置、Prometheus／Grafana 配置、LiveKit 配置的變更，須納入版本管理，不得僅以管理介面手動變更。

### 7.2 回滾

- 回滾流程：停止新版服務 → 還原上一版映像 → 還原配置 → 執行 Migration 回滾（如可回滾；不可回滾時以備份還原）→ 啟動服務 → 健康檢查與冒煙測試 → 監控確認。回滾步驟、驗證方式待定稿；回滾演練在 staging 進行，不得在 production 首次演練。
- 回滾判定：發布後監控告警、錯誤率、健康檢查失敗時的回滾標準與負責人待定稿。

### 7.3 監控驗證清單（部署後必查）

部署完成後，須逐項勾選並留存證據，不得僅寫「監控正常」：

- [ ] Traefik 健康：80／443 可達、HTTP 重定向至 HTTPS、憑證有效且為 Let's Encrypt 核發、憑證到期日可查。
- [ ] 各服務 Health Check：`docker compose ps` 中全部應啟動服務為健康狀態（`healthy`），不健康者已記錄原因與處置。
- [ ] API 健康檢查：`GET /health`（路徑待定稿）回傳正常，不得回傳敏感資訊。
- [ ] 資料庫連線：API 可連 PostgreSQL、Migration 版本與程式碼版本一致。
- [ ] Valkey／NATS 連線：快取讀寫、事件發布／訂閱正常。
- [ ] MinIO：Bucket 存在且不公開、上傳／下載經簽名 URL＋權限驗證正常。
- [ ] Meilisearch：索引健康、搜尋查詢正常、索引內容經權限過濾。
- [ ] Keycloak：登入／登出、Token 核發、MFA（如啟用）正常；Realm／Client 配置與程式碼一致。
- [ ] OpenBao：Secrets 讀取正常、解封狀態正常、稽核紀錄可查。
- [ ] LiveKit：房間建立／加入正常（待 Phase 07 通話驗證）；TURN 連通性已評估。
- [ ] Prometheus／Grafana／Loki：指標蒐集正常、儀表板可查、日誌彙總正常、告警規則已載入。
- [ ] GlitchTip：錯誤上報正常、錯誤報告不含敏感資料。
- [ ] 備份：最近一次備份已產生、備份檔加密、備份位置可查、還原演練紀錄可查。
- [ ] 冒煙測試：註冊、登入、加好友、傳訊、收訊、傳照片、建群組、群組訊息、推播、登出（E2E 12 步中的部署冒煙子集，通話冒煙待 Phase 07）通過，證據留存。

## 8. 仍可能付費項目（不得宣稱零成本）

網域、伺服器（主機租賃／頻寬）、簡訊 OTP、Apple Developer、Google Play 上架、外部 AI API、TURN 頻寬等，金額與計價方式皆待查證，不得預寫數字。正式部署前，須完成成本查證並取得使用者決定，不得新增付款方式。

## 9. 缺口與待建立檔案

1. `livekit.yaml`（LiveKit CE 配置）：`livekit` 服務以 `profiles: ["phase07"]` 保留定義，預設不啟動；livekit.yaml 待 Phase 07 定稿後建立接線；建立前 LiveKit 服務不啟動，不得假裝已配置。
2. `prometheus.yml`（Prometheus 配置）：最小有效設定已建立（僅自我抓取）；抓取目標定稿待監控文件定稿後加入；加入前不得假裝監控已覆蓋全部服務。
3. `vLLM` 容器：docker-compose.yml 未預寫 vLLM 容器，服務形態待 Phase 09 定稿後補寫。
4. OpenBao 正式模式：docker-compose.yml 的 OpenBao 以開發模式佔位，正式模式、解封流程待定稿，不得以開發模式部署正式環境。
5. Keycloak 正式啟動參數：docker-compose.yml 的 Keycloak 以開發模式佔位，正式參數待定稿。
6. 全部 Health Check 指令、映像標籤、連接埠，皆為草案，待有 Docker 的環境驗證後定稿。
7. Coolify 的具體操作（專案建立、網域綁定、部署觸發）待 Coolify 採用定稿後補寫，本文件不預寫 Coolify 介面操作。
8. 本機 VM 無 docker，本文件全部步驟待 CI 或有 Docker 的環境驗證。
