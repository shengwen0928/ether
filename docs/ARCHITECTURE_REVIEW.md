狀態：已批准（2026-11-25 01:01 使用者於對話批准全部 Phase 00 文件，開始 Phase 01）

> **批准註記：** Phase 00 文件＋品牌文件全數定稿。Gin／Echo、Qdrant／pgvector 選型、部署主機與成本方案、外部憑證申請順序四項仍待決定，不得視為已批准。

> **Phase 01 決議註記（2026-11-25）：** 四項決議已定（G-01 方案 A、Echo v5、pgvector、G-02／G-03 方案），見 ADR-015／016／017；「Phase 01 最終驗收仍待執行驗證」維持。

> **品牌更名修訂註記（2026-11-25 00:42 品牌企劃書，品牌替換協調員）：** 本報告完成後，專案品牌由暫定名 NEXUS 正式改為 ETHER，目錄由 `~/workspace/nexus/` 改為 `~/workspace/ether/`；本報告內文已隨更名修訂（NEXUS→ETHER），架構結論不變。新增文件 3 份：`docs/brand/BRAND_GUIDELINES.md`、`docs/brand/DESIGN_TOKENS.md`、`docs/decisions/ADR-014-品牌更名ETHER暨設計系統方向.md`（見 §1 數量變化）；`SPEC-SOURCE.md` 與 `ADR-005` 內文保留原始寫法僅供追溯。

# ETHER Phase 00 架構審查報告

審查人：Phase 00 協調員（親自交叉核對）。審查範圍：`~/workspace/ether/` 全部 Phase 00
文件。結論：**Phase 00 文件齊全，可送使用者審查；不得進入 Phase 01，不得宣稱產品完成。**
全部文件皆為草案，未經使用者批准；全部實作驗證皆為「待 CI 或有 Docker 的環境驗證」。

## 1. 文件清單（共 33 份文件＋2 份設定檔）

| 類別 | 文件 |
|---|---|
| 規格來源 | `SPEC-SOURCE.md`（使用者規格忠實摘要＋00:33 架構調整置頂節） |
| 產品 | `docs/PRD.md`（204 行）、`docs/FEATURE_LIST.md`（273 行）、`docs/PLATFORM_MATRIX.md`（187 行） |
| 架構 | `docs/architecture/SYSTEM_ARCHITECTURE.md`（345 行）、`MODULE_BOUNDARIES.md`（368 行）、`DATA_FLOW.md`（365 行）、`SECURITY_ARCHITECTURE.md`（183 行）、`SCALABILITY.md`（119 行）、`IDENTITY_AND_PERMISSIONS.md`（217 行） |
| 安全 | `docs/security/E2EE_RESEARCH.md`（190 行）、`docs/security/AI_DATA_POLICY.md`（243 行） |
| 成本 | `docs/COST_OPTIMIZATION.md`（301 行，20 項替代對照） |
| 資料 | `docs/database/ERD.md`（948 行，64 張表） |
| API | `docs/api/API_STANDARDS.md`（250 行）、`docs/api/ERROR_CODES.md`（191 行）、`docs/api/WEBSOCKET_PROTOCOL.md`（308 行，15 事件） |
| 測試／部署 | `docs/testing/TEST_STRATEGY.md`（125 行）、`docs/deployment/CLOUD_ARCHITECTURE.md`（220 行）、`docs/deployment/SELF_HOSTING.md`（120 行）、`docs/ROADMAP.md`（187 行） |
| 決策 | `docs/decisions/` ADR-001～ADR-013（13 份） |
| 設定方案 | `infrastructure/docker/docker-compose.yml`（451 行，18 服務，YAML 解析通過）、`.env.example`（109 行，全佔位符） |

舊文件（`~/workspace/messenger-app/` PLAN.md、MASTER_SPEC.md、CONTRACTS.md）未覆蓋、未刪除，
差異寫入 ADR-005；README／AGENTS.md／PROJECT_STATE.md／ARCHITECTURE.md 經盤點確認不存在，
未假設存在。

## 2. 關鍵決策

- 模組化單體起步（ADR-001），22 模組責任邊界明確、依賴無循環（拓撲排序驗證為 DAG）。
- 免費開源自架總決策（ADR-009，取代 ADR-002）：見第 3 節替換清單。
- 身分改 Keycloak（ADR-010），保留全部帳號流程，不得破壞使用者／好友／裝置關聯。
- E2EE 唯一候選 Signal Protocol（ADR-007），不得自研演算法，須經 Phase 11 協定審計。
- iOS 須走 macOS CI（ADR-003）；本機 VM 實測僅有 node／npm，無 flutter／go／docker。
- WebSocket＋NATS JetStream 即時架構（ADR-006），冪等／Sequence／ACK／去重為鐵律。

## 3. 付費→免費／開源替換清單（使用者 00:33 指定，20 項）

ECS Fargate→Docker Compose＋Coolify；RDS／Aurora→PostgreSQL 自架；ElastiCache→Valkey；
S3→MinIO；OpenSearch→Meilisearch；Secrets Manager→OpenBao；CloudWatch→Prometheus＋
Grafana＋Loki；CloudFront→Cloudflare Free；WAF→Coraza；Route 53→Cloudflare DNS；
Certificate Manager→Let's Encrypt；Cognito→Keycloak；API Gateway→Traefik；
SQS／SNS→NATS JetStream；CodePipeline→GitHub Actions；Terraform→OpenTofu；
LiveKit Cloud→LiveKit CE 自架；雲端 AI API→Ollama＋vLLM＋Adapter；向量庫→Qdrant／
pgvector；Sentry 託管→GlitchTip 自架。功能一項未刪、未降級。

## 4. 協調員親自核對與修正

- 全文 grep：架構／產品／API／測試／部署正文舊付費服務名零殘留；殘留僅見於三處允許
  的比較／追溯段落（CLOUD_ARCHITECTURE.md §2 比較表、ADR-009 對照表、ADR-002 已取代
  原文），皆標明已取代。`SPEC-SOURCE.md` 頂部已加追溯說明，調和「舊名只准出現在比較
  文件」與部署文件比較段落的 tension。
- 表名：§20 共 64 張（非任務簡報誤稱的 60 張，ERD.md §2 已警示），ERD 逐字齊全；
  全部出現於模組邊界文件，無遺漏。
- API 前綴 18 個、WebSocket 事件 15 個，逐字核對無缺漏、無改名。
- Compose YAML 以 Python 解析通過：18 服務、17 個 healthcheck（worker 為背景任務故無，
  屬刻意設定）；`.env.example` 全佔位符，無真實金鑰；文件級 secrets 掃描零命中。
- 「零成本」全部出現皆為「不得宣稱零成本」否定句；成本數字一律「待查證」，無編造。
- 修正紀錄：僅文字層（SPEC-SOURCE.md 追溯說明）；未改任何工人文件的實質內容。

## 5. 阻塞項（未取得前相關整合只寫介面與 Adapter，不得宣稱完成）

1. 正式部署主機與成本：待使用者決定，不得新增付款方式（ADR-002 阻塞持續有效）。
2. 外部憑證：FCM、APNs、Apple Sign-In、Google OAuth、簡訊 OTP、付款供應商（ADR-004）。
3. iOS 驗證路徑：macOS CI runner（ADR-003）。
4. Docker 驗證環境：本機 VM 無 docker，Compose 啟動／連線／上傳／訊息／AI 切換／
   LiveKit／監控全數待驗證。
5. Gin 或 Echo 二選一、Qdrant／pgvector 二選一、Keycloak Realm／Session 銜接細節未定。
6. 使用者批准：全部 Phase 00 文件皆待批准；OQ／NQ 未決問題清單見 PRD。

## 6. 未完成項與缺口

- `livekit.yaml`、`prometheus.yml` 尚未建立；vLLM 容器未預寫（待 Phase 09 定稿）；
  OpenBao／Keycloak 在 Compose 中為開發模式佔位，不得用於正式環境；api／gateway／
  worker 尚無程式碼與 Dockerfile。
- 群組／社群投票表、聊天備份資料模型、官方帳號／Bot 在 `messages` 的發送者表示方式未定
  （§20 無對應表）；Outbox 表名、Sequence 分配、`client_message_id` 與 Idempotency Key
  關係未定；檔案掃描方案未定；向量庫選型未定。
- TURN 公網拓撲／頻寬實測、AI 硬體實測、備份頻率／RPO／RTO、告警門檻、成本查證皆未做。
- Cloudflare Free／GitHub Actions／Let's Encrypt 當期免費額度數字未查證；COST 文件授權
  標示須以官方文件覆核。

## 7. 仍可能付費清單（不得宣稱零成本）

網域註冊、伺服器（主機租賃／頻寬）、簡訊 OTP、Apple Developer 年費、Google Play
開發者帳號費、外部 AI API 用量、TURN 頻寬。金額一律待查證。

## 8. 硬體規格建議（定性分級，數字待實測，不得假設）

- 一般服務（API／Gateway／Worker／PG／Valkey／NATS／MinIO／Meilisearch／監控）：以
  Compose 單主機起步，規格待 Phase 01 實測後定案。
- AI：分小型（CPU 可跑，GPU 可選）／中型（建議 GPU，十數 GB 級 VRAM 方向）／大型
  （須 GPU，數十 GB 級以上 VRAM 方向，可能多卡）三級，區間皆標待查證；實測前不得宣稱
  某模型可在某硬體運作；大型模型須評估改用外部 AI API（須授權與資料邊界確認）。
- 通話：LiveKit CE＋TURN 須評估併發伺服器與頻寬，TURN 頻寬為主要持續性成本之一。

## 9. 尚存風險

- 自架把「按量付費」改為「主機＋頻寬＋人力維運」成本，成本未消失；維運、備份、升級、
  安全更新責任全在營運方。
- 免費額度（Cloudflare／GitHub Actions／Let's Encrypt）有上限與功能限制，超額即收費或限流。
- Keycloak／LiveKit CE／Meilisearch／Valkey 等自架元件的維運複雜度與故障模式，須在
  Phase 01 以演練驗證，不得以文件完備代替。
- E2EE 協定未經審計前不得上線；E2EE 明文絕對不得送往任何 AI（含本地 AI，預設關閉）。

## 10. 下一步（須使用者批准）

一件一件決定：(1) Phase 00 文件定稿；(2) 部署主機與成本方案；(3) 外部憑證申請順序；
(4) Gin／Echo 與向量庫選型。批准後方可進入 Phase 01。
