狀態：草案／待批准

# ETHER 自架部署架構（CLOUD_ARCHITECTURE.md）

## 1. 依據與效力

- 本文件以 `~/workspace/ether/SPEC-SOURCE.md` 頂部「架構調整」（2026-11-25 00:33 使用者指令，優先於舊技術棧）為最高準據，並對照 §22、§23、§25、手冊 C／M。功能不得刪、不得降級、不得破壞 E2EE。
- 本文件定自架部署架構（Docker Compose＋Coolify 為部署形態），不定應用程式碼、不定具體主機規格、不定成本數字。成本一律寫「待查證」，不得宣稱全系統零成本（見 §9）。
- 本文件為 Phase 00 草案，全部部署驗證皆為「待 CI 或有 Docker 的環境驗證」：本機 VM 無 docker，不得寫已驗證（見 §3）。
- 舊服務名只准出現在 §2 比較段落（且須標明已取代），正文不得殘留。比較段落以外的舊付費服務名，皆已按強制替換對照替換。

## 2. 舊方案比較（已取代，僅供對照）

以下對照為 2026-11-25 00:33 使用者指定的強制替換，舊方案已取代，不得再以舊方案部署：

| 舊方案（已取代） | 新方案（本文件） |
|---|---|
| AWS ECS Fargate | Docker Compose＋Coolify |
| AWS RDS／Aurora | PostgreSQL 自架 |
| ElastiCache | Valkey |
| S3 | MinIO |
| OpenSearch | Meilisearch |
| Secrets Manager | OpenBao |
| CloudWatch | Prometheus＋Grafana＋Loki |
| CloudFront | Cloudflare Free |
| WAF | Coraza WAF |
| Route 53 | Cloudflare DNS |
| Certificate Manager | Let's Encrypt |
| Cognito | Keycloak |
| API Gateway | Traefik |
| SQS／SNS | NATS JetStream |
| CodePipeline | GitHub Actions |
| Terraform | OpenTofu |
| LiveKit Cloud | LiveKit Community Edition 自架 |
| 雲端 AI API | Ollama＋vLLM＋AI Provider Adapter（LocalAI 優先＋外部 AI API Adapter 可選擴充，見 §7.5） |
| 向量資料庫付費服務 | Qdrant／pgvector（選擇待定；已決議採用 pgvector，見 ADR-017） |
| Sentry 託管 | GlitchTip 自架 |

本節之後，正文不再使用舊方案名稱；部署、監控、備份、CI／CD 皆以新方案敘述。

## 3. 部署形態與阻塞項（必讀）

### 3.1 部署形態

- 部署形態：單一主機或多主機上的 Docker Compose 服務組合，由 Coolify 管理部署、網域、憑證與服務生命週期（Coolify 的使用範圍待定稿：是否全部環境皆用 Coolify，或僅正式環境用，見 §10 缺口）。
- 基礎設施即程式碼（IaC）：OpenTofu（選用）。OpenTofu 的使用範圍（主機佈建、網路、防火牆、DNS 紀錄等）待定稿；如主機為單一手動佈建的伺服器，OpenTofu 僅作紀錄用途，不得以「有 OpenTofu 檔案」假裝主機佈建已自動化。
- 容器化：全部服務皆以容器運行（Go 後端、Realtime Gateway、Worker、PostgreSQL（含 pgvector 擴充）、Valkey、NATS、MinIO、Meilisearch、OpenBao、Keycloak、Traefik、Prometheus、Grafana、Loki、GlitchTip、LiveKit CE、Ollama），容器映像版本須固定（不得用浮動標籤），映像來源與簽章驗證待定稿。
- 後續大型部署選項（Kubernetes）在本文件不展開：如未來流量需要，須另行評估並以 ADR 記錄，不得以本文件的 Compose 架構假裝已覆蓋叢集部署。

### 3.2 開發期準則

- **開發期以本機 Docker Compose 為準**：開發、整合測試、staging 驗收，皆以 `infrastructure/docker/docker-compose.yml` 啟動的同一組服務為準，不得以各開發者本機的零散安裝代替。
- 本機 VM 無 docker：本地驗證待環境就緒（有 Docker 的環境或 CI），不得假裝已在本機驗證啟動、連線、上傳、訊息、AI 切換、LiveKit、監控。

### 3.3 正式部署阻塞項

- **正式部署為阻塞項**：正式部署的伺服器、網域、憑證、備份、監控，成本待查證，須使用者決定，不得新增付款方式，不得擅自開通任何付費服務。
- 仍可能付費項目（須明列，不得宣稱零成本）：網域、伺服器（主機租賃／頻寬）、簡訊 OTP、Apple Developer、Google Play 上架、外部 AI API、TURN 頻寬等。每一項的金額、計價方式，皆待查證，不得預寫數字。
- 正式部署前須完成：成本查證與使用者決定、備份與還原演練、監控告警驗證、安全測試（見 TEST_STRATEGY.md）、發布審查報告（手冊 M）。缺任一項不得發布。

## 4. 部署拓撲（文字圖）

```
                        網際網路
                            │
                    ┌───────┴────────┐
                    │ Cloudflare Free │
                    │  DNS ＋ CDN     │
                    └───────┬────────┘
                            │  HTTPS（Let's Encrypt 憑證）
                    ┌───────┴────────┐
                    │  Coraza WAF     │
                    │ （Traefik 前置） │
                    └───────┬────────┘
                            │
                    ┌───────┴────────┐
                    │    Traefik      │
                    │  邊緣閘道／路由  │
                    └───────┬────────┘
         ┌──────────────────┼──────────────────┐
         │                  │                  │
  ┌──────┴──────┐    ┌──────┴──────┐    ┌──────┴──────┐
  │  Go API     │    │ Realtime    │    │ Web／Admin  │
  │  （模組化單體）│    │ Gateway     │    │ （React 靜態）│
  └──────┬──────┘    └──────┬──────┘    └─────────────┘
         │                  │
         └────────┬─────────┘
                  │
     ┌────────────┼────────────┬─────────────┬──────────────┐
     │            │            │             │              │
┌────┴─────┐ ┌────┴─────┐ ┌────┴─────┐ ┌─────┴──────┐ ┌─────┴──────┐
│PostgreSQL│ │ Valkey   │ │  NATS    │ │  MinIO     │ │ Meilisearch│
│（系統紀錄）│ │（快取／   │ │ JetStream│ │（物件儲存）│ │（搜尋投影）│
│＋pgvector │ │ Session） │ │（佇列／  │ │            │ │            │
│  待定）   │ │          │ │  事件）  │ │            │ │            │
└──────────┘ └──────────┘ └──────────┘ └────────────┘ └────────────┘
     │            │            │
     └────────────┼────────────┘
                  │
     ┌────────────┼────────────┬──────────────┬─────────────┐
     │            │            │              │             │
┌────┴─────┐ ┌────┴─────┐ ┌────┴──────┐ ┌────┴──────┐ ┌────┴─────┐
│ OpenBao  │ │ Keycloak │ │ LiveKit   │ │  pgvector │ │  Ollama  │
│（Secrets）│ │（身分）  │ │  CE       │ │PG 擴充    │ │ ＋ vLLM  │
│          │ │          │ │ （SFU）   │ │  已定案   │ │ （AI）   │
└──────────┘ └──────────┘ └───────────┘ └───────────┘ └──────────┘
     │            │            │
     └────────────┼────────────┘
                  │
     ┌────────────┼────────────┬─────────────┐
     │            │            │             │
┌────┴─────┐ ┌────┴─────┐ ┌────┴─────┐ ┌────┴─────┐
│Prometheus│ │ Grafana  │ │  Loki    │ │GlitchTip │
│（指標）   │ │（儀表板）│ │（日誌）  │ │（錯誤追蹤）│
└──────────┘ └──────────┘ └──────────┘ └──────────┘
```

拓撲說明：

- 客戶端（Flutter App、Web、桌面）永遠只與邊緣層（Cloudflare＋Coraza＋Traefik）對話，不直連資料庫、NATS、Meilisearch、MinIO、OpenBao、Keycloak 內部端點（媒體上傳走簽名 URL，簽名由 API 核發、權限由 API 驗證，細節見 DATA_FLOW.md，另冊）。
- Go API 與 Realtime Gateway 為兩個行程角色（模組化單體＋即時閘道分離，見架構文件，另冊）；Worker 為背景任務行程（索引投影、推播投遞、媒體處理、AI 任務等，任務清單待定稿）。
- 資料層：PostgreSQL 為系統紀錄；Valkey 為快取與 Session 儲存；NATS JetStream 為事件與佇列；MinIO 為物件儲存；Meilisearch 為搜尋投影；pgvector 為向量承載（已定案，見 ADR-017）。
- 身分與 Secrets：Keycloak 為身分提供者；OpenBao 為 Secrets 儲存（見 §6）。
- 通話與 AI：LiveKit CE 為 SFU；Ollama＋vLLM 為本地 AI 推理（見 §7.5）。
- 監控：Prometheus＋Grafana＋Loki＋GlitchTip＋OpenTelemetry（見 §8）。

## 5. 服務清單與職責

| 服務 | 職責 | 持久化（Volume） | 備註 |
|---|---|---|---|
| `traefik` | 邊緣閘道、路由、TLS 終止、限流（閘道層） | 憑證與配置（見 SELF_HOSTING.md） | 憑證由 Let's Encrypt 核發，經 Cloudflare DNS 或 HTTP 驗證，方式待定稿。 |
| `api` | Go 後端（REST，模組化單體） | 無（無狀態） | 健康檢查、Migration 執行者（定稿時確認）。 |
| `realtime-gateway` | WebSocket 閘道（Session Registry、Heartbeat、重連、路由、恢復） | 無（Session 在 Valkey） | 水平擴充方式待定稿（Session 共享經 Valkey／NATS）。 |
| `worker` | 背景任務（索引投影、推播、媒體、AI、清理） | 無（任務狀態在 PostgreSQL／NATS） | 任務重試、死信佇列（DLQ）規則待定稿。 |
| `postgresql` | 系統紀錄（§20 全部 64 張表）、pgvector（待定） | 資料 Volume | 備份見 §7.2。 |
| `valkey` | 快取、Session、限流計數、Presence | 快取 Volume（可重建，須標示） | 快取遺失不得導致資料遺失；Session 遺失的恢復規則待定稿。 |
| `nats` | JetStream 事件匯流排與佇列 | 串流 Volume | 串流保留、重放規則待定稿。 |
| `minio` | 物件儲存（媒體原檔、縮圖、備份檔） | 物件 Volume | 存取須經簽名 URL＋權限驗證，不得公開 Bucket。 |
| `meilisearch` | 搜尋索引投影 | 索引 Volume（可重建，須標示） | 索引可由系統紀錄重建，不得作為系統紀錄。 |
| `openbao` | Secrets 儲存（資料庫憑證、簽章金鑰、第三方憑證參照） | Secrets Volume（加密，見 §6） | 解封（Unseal）流程待定稿，不得將解封金鑰寫入 Git。 |
| `keycloak` | 身分提供者（OIDC／OAuth 2.0、MFA、Session） | 資料庫（PostgreSQL 內或獨立庫，待定稿） | Realm／Client 配置以程式碼管理（待定稿），不得僅以管理介面手動配置。 |
| `livekit` | LiveKit CE（SFU、房間、STUN／TURN 協調） | 無（房間狀態為短暫） | TURN 頻寬與公網部署需求須特別評估（見 §7.5）。 |
| `postgresql`（pgvector 擴充） | 向量資料庫（已定案，見 ADR-017） | 隨 `postgres-data` Volume | 無獨立向量服務；向量資料隨 PostgreSQL 備份。 |
| `ollama` | 本地 AI 模型推理（模型載入、Streaming） | 模型 Volume | 模型清單、硬體需求待評估（見 §7.5），不得假設一般伺服器跑得動所有模型。 |
| `prometheus` | 指標蒐集 | 指標 Volume | 保留期限待定稿，不得預寫天數。 |
| `grafana` | 儀表板與告警檢視 | 設定 Volume | 儀表板定義以程式碼管理（待定稿）。 |
| `loki` | 日誌彙總 | 日誌 Volume | 日誌不得含敏感資料明文（見 API_STANDARDS.md §9）。 |
| `glitchtip` | 錯誤追蹤（自架） | 資料 Volume | 錯誤報告不得含敏感資料（見 ERROR_CODES.md §6.5）。 |

服務啟動順序、Health Check、Volume、網路，見 `infrastructure/docker/docker-compose.yml` 與 SELF_HOSTING.md；本文件不重寫 Compose 內容，Compose 檔案為準（尚待有 Docker 的環境驗證）。

## 6. 環境劃分

| 環境 | 用途 | 資料 | Secrets | 部署方式 |
|---|---|---|---|---|
| local | 開發者本機開發 | 虛構測試資料 | 本機 `.env`（佔位符，見 `.env.example`），不得含真實金鑰 | 本機 Docker Compose |
| development | 共享開發驗證 | 虛構測試資料 | OpenBao 開發 Vault，不得入 Git | Docker Compose＋Coolify（待定稿） |
| staging | 發布前驗收（與 production 同形） | 類生產結構的測試資料；外部服務用測試模式／沙箱 | OpenBao，不入 Git | 與 production 相同的部署流程，成本待查證 |
| production | 正式服務 | 真實使用者資料；備份與還原演練為必要條件 | OpenBao，不入 Git；Production Secrets 絕對不入 Git | Docker Compose＋Coolify＋OpenTofu（選用），成本待查證（阻塞項，見 §3.3） |

- 環境變數：以 `.env.example` 為範本（佔位符，不得含真實值）；各環境的實際值以 OpenBao／環境注入提供，不得寫入 Git、不得寫入映像。
- 配置差異：各環境的差異（網域、憑證、推播模式、AI 模型、TURN 配置）須以配置檔案區分，不得以程式碼分支區分環境。

## 7. Secrets 管理（OpenBao）

- Secrets 一律存於 OpenBao，不得寫入原始碼、不得寫入 Git、不得寫入映像、不得寫入日誌、不得寫入 Compose 檔案（Compose 只寫參照，不寫值）。
- Secrets 範圍（草案）：資料庫憑證、Valkey／NATS 憑證、MinIO 憑證、Meilisearch 金鑰、Keycloak 管理憑證、JWT／Token 簽章金鑰、Bot 金鑰參照、推播憑證、外部 AI API 金鑰（可選擴充）、TURN 憑證、備份加密金鑰。清單待定稿，定稿時逐項確認存放位置與輪換週期。
- 存取：服務以最小權限讀取所需 Secrets（OpenBao 政策＋身分認證，方式待定稿）；服務不得讀取無關 Secrets；Secrets 讀取須有稽核紀錄。
- 輪換：資料庫憑證、簽章金鑰、Bot 金鑰的輪換週期與流程待定稿；輪換不得造成服務中斷（雙金鑰過渡期規則待定稿）。
- 解封與備份：OpenBao 的解封流程、主金鑰保管、Secrets 備份與還原，待定稿；解封金鑰不得與 Secrets 備份同處存放，不得寫入 Git。
- 本機開發：本機 Secrets 以 `.env` 佔位符＋本機 OpenBao（或開發模式）提供，不得以正式 Secrets 在本機開發。

## 8. 備份與災復、監控告警

### 8.1 備份與災復

- 備份範圍：PostgreSQL 資料庫、MinIO 物件、OpenBao Secrets 備份、Keycloak 配置、Meilisearch 索引（可重建，備份優先級待定稿）、NATS 串流（保留策略待定稿）、Volume 快照。備份清單待定稿，不得遺漏任一持久化 Volume。
- 備份方式：資料庫以邏輯／物理備份擇一（待定稿）、MinIO 以物件複製或快照（待定稿）、Volume 以主機快照或檔案備份（待定稿）；備份檔案須加密、須存於異地或獨立儲存（異地方式待定稿，成本待查證）。
- 備份頻率、保留期限、還原點目標（RPO）、還原時間目標（RTO），一律待定稿／待查證，不得預寫數字。
- 還原演練：每季一次（頻率為草案方向，待定稿確認），在隔離環境演練，不得在正式資料庫上演練；演練結果須留存紀錄；演練失敗時，正式備份策略須重新檢討，不得宣稱「已具備復原能力」。
- 災復流程：主機故障、資料庫損毀、Secrets 遺失、憑證過期，各情境的復原步驟待定稿；復原流程須有負責人與聯絡方式（待定稿，不得預寫）。

### 8.2 監控告警

- 監控採 Prometheus＋Grafana＋Loki＋OpenTelemetry＋GlitchTip，覆蓋 API、WebSocket、資料庫、Worker、Media、AI（見 SPEC-SOURCE.md 架構調整一節）。
- 指標：API 錯誤率、WebSocket 連線數、資料庫連線數、MinIO 用量、推播發送成功率、NATS 佇列深度、Valkey 命中率、Meilisearch 索引延遲、LiveKit 房間數、AI 推理延遲與用量、主機資源（CPU／記憶體／磁碟）。指標清單待定稿，告警門檻一律待定稿／待實測，不得預寫數字。
- 日誌：結構化日誌經 Loki 彙總，含 `request_id`，不得含敏感資料明文；日誌保留期限待定稿。
- 告警：告警管道與負責人待定稿（見舊規格未決問題同類事項，ETHER 待決定）；告警須通知負責人，夜間告警流程待定稿；監控故障不得影響訊息傳送主路徑，監控數據缺失時報告須標示「數據缺失」，不得以推測數字填補。
- 儀表板：Grafana 儀表板定義以程式碼管理，須含系統健康、即時在線、訊息量、通話用量、媒體用量、AI 用量（管理後台儀表板與 Grafana 的分工待定稿，不得重複定義而不說明）。

## 9. CI／CD（GitHub Actions）

- CI：每次提交／PR 觸發 GitHub Actions，執行格式化／Lint、靜態分析、單元測試、整合測試（待 CI 或有 Docker 的環境驗證）、OpenAPI 一致性檢查、Secrets Scan、依賴掃描、Migration 檢查；CI 失敗不得合併。
- CD：staging 部署經 Coolify 自動部署（觸發條件待定稿）；production 部署須經批准（批准流程待定稿，不得自動部署至 production 而不經審查）。
- 部署流程：建置映像（版本固定）→ 推送至映像倉庫（倉庫選定待定稿）→ Coolify 部署 → 健康檢查 → 冒煙測試 → 監控確認。任一環節失敗須停止並回滾（回滾流程見 ROADMAP.md）。
- IaC：OpenTofu 配置（如採用）須納入版本管理、須經 CI 驗證（`validate`／`plan`，待定稿），不得以手動操作代替 IaC。
- 憑證：Let's Encrypt 憑證由 Traefik／Coolify 自動續期（續期失敗告警待定稿）；憑證不得手動上傳私鑰至 Git。

## 10. 通話與 AI 的自架評估（必讀缺口）

### 10.1 LiveKit CE 自架

- LiveKit CE 保留一對一／多人、螢幕分享、房間與參與者、STUN／TURN、SFU、品質監控（見架構調整一節）。
- 特別評估（待 Phase 07 定稿，不得假設已評估）：TURN 頻寬與成本（仍可能付費，見 §3.3）、公網部署需求（UDP 埠、NAT、防火牆）、房間容量、錄製需求（如有，儲存位置待定稿）、與 Keycloak／API 的認證整合（房間 Token 簽發方式待定稿）。

### 10.2 AI 自架（Ollama＋vLLM＋Adapter）

- AI 採 Ollama＋vLLM＋LocalAI 優先＋外部 AI API Adapter 可選擴充（見架構調整一節）：可切換模型、Streaming、Tool Calling、上下文、Token 統計、資源管理；供應商不得寫死。
- 特別評估（待 Phase 09 定稿，不得假設）：CPU／RAM／GPU／VRAM 需求（不得假設一般伺服器跑得動所有模型）、模型清單與授權、推理延遲、併發、Token 預算、E2EE 明文不得自動送往雲端 AI（本地模型與外部 API 的邊界待定稿）、向量承載（pgvector，已定案，見 ADR-017；細節見 ERD.md §10.4）。

## 11. 缺口與待定稿事項

1. Coolify 的使用範圍、OpenTofu 的使用範圍、映像倉庫選定，待定稿。
2. Keycloak 的 Realm／Client 配置管理方式、PostgreSQL 佈局（獨立庫或共用庫），待身分文件定稿。
3. 向量承載已定案為 pgvector（見 ADR-017）；Meilisearch 與 PostgreSQL 全文搜尋的分工，待定稿。
4. 備份頻率／保留期限／RPO／RTO、告警門檻與管道、負責人，待定稿／待查證。
5. TURN 頻寬評估、AI 硬體評估，待 Phase 07／09 定稿。
6. 正式部署的伺服器、網域、成本，待查證與使用者決定（阻塞項，見 §3.3）；不得新增付款方式。
7. 本機 VM 無 docker，全部部署驗證待 CI 或有 Docker 的環境驗證。
8. 舊方案比較以外的舊服務名不得殘留；本文件以外的其他 Phase 00 文件如仍含舊服務名，須由各文件負責人修訂（本文件不代修他人文件）。
