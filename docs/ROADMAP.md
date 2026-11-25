狀態：草案／待批准

# ETHER 完整開發 Roadmap（ROADMAP.md）

## 1. 依據與效力

- 本文件以 `~/workspace/ether/SPEC-SOURCE.md` §25（階段）、手冊 B–N（Phase 00–12 執行手冊）、§27（Agent 執行規則）、§28（最終交付）為最高準據。階段順序、驗收、停止審查門檻以手冊為準，不得跳階段、不得把後階段工作寫成已完成。
- 功能深度補充參考 `~/workspace/messenger-app/MASTER_SPEC.md`；舊合約 `~/workspace/messenger-app/CONTRACTS.md` 僅供參考。舊路線（Kotlin＋Supabase、Android 先行、三階段）已被 ETHER 規格取代，不得以舊階段劃分代替本文件。
- 本文件定 Roadmap，不定個別功能的詳細設計；各階段的詳細設計在各階段文件中定稿。本文件不預寫任何階段的完成日期、工時、成本數字；時程與成本一律待查證／待決定，不得編造。
- 架構調整（2026-11-25 00:33 使用者指令，優先於舊技術棧）：技術敘述一律改新技術棧（Docker Compose＋Coolify、Traefik、Keycloak、MinIO、Valkey、Meilisearch、OpenBao、NATS JetStream、OpenTofu、GitHub Actions、LiveKit CE、Ollama＋vLLM、pgvector、Prometheus＋Grafana＋Loki＋GlitchTip）。階段順序與驗收不變；舊付費服務名不得殘留。

## 2. 階段編號對照（重要缺口）

SPEC-SOURCE.md §25 與手冊 B–N 的階段編號存在差異，本文件以手冊 B–N 為主幹，並對照如下，**待 Phase 00 審查確認**：

| 手冊 | 本文件階段 | §25 對應 | 差異說明 |
|---|---|---|---|
| B | Phase 00 架構與規格 | （Phase 00，先行） | 一致。 |
| C | Phase 01 基礎建設 | §25 Phase 1 基礎 | 手冊 C 含 Realtime Gateway、Worker、NATS、Docker Compose、OpenTofu、GitHub Actions、Secrets、Health Check、Logging、OpenTelemetry、CI，較 §25 詳盡，以手冊為準。 |
| D | Phase 02 設計系統與品牌 | （§25 未單列） | 手冊 D 為獨立階段（Logo、Design Tokens、Flutter／React 元件庫、平台差異化）；§25 將設計系統併入 Phase 1，本文件以手冊為準單列。 |
| E | Phase 03 身分帳號 | §25 Phase 2 帳號 | 編號順延一位，內容對應。 |
| F | Phase 04 好友聯絡人 | §25 Phase 3 好友 | 編號順延一位，內容對應。 |
| G | Phase 05 即時訊息核心 | §25 Phase 4 即時聊天 | 編號順延一位，內容對應。 |
| H | Phase 06 媒體群組社群 | §25 Phase 5 多媒體＋Phase 6 群組社群 | 手冊 H 合併 §25 的兩階段為一階段，本文件以手冊為準。 |
| I | Phase 07 語音視訊會議 | §25 Phase 7 語音視訊 | 內容對應，技術改 LiveKit CE。 |
| J | Phase 08 社群貼圖商業 | §25 Phase 8 貼圖動態＋Phase 9 官方帳號與 Bot（部分） | 手冊 J 合併並擴充商業能力，本文件以手冊為準。 |
| K | Phase 09 ETHER AI | （§25 未單列） | 手冊 K 為獨立 AI 階段（AI 對話、訊息 AI、知識 AI、商業 AI）；§25 無獨立 AI 階段，本文件以手冊為準單列。 |
| L | Phase 10 Web／桌面／多裝置 | （§25 未單列，後台見 Phase 10 管理後台） | 手冊 L 為 Web／桌面／多裝置階段；管理後台見 Phase 10 備註（見 §3.11），二者分工待 Phase 00 審查確認。 |
| M | Phase 11 安全效能發布 | §25 Phase 11 安全與正式部署（部分） | 手冊 M 含安全、效能、可靠性、發布，較 §25 詳盡，以手冊為準。 |
| N | Phase 12 最終稽核 | （§25 未單列，驗收見 §28） | 手冊 N 為獨立稽核階段，產出稽核文件，不得因階段執行完就判定完成。 |

- 如審查決定以 §25 編號為準，本文件須整體重編並同步全部引用，不得僅改標題而不改內文引用。
- 管理後台（§25 Phase 10）：手冊 J／L／M 皆含管理後台相關能力（官方帳號管理、開發者 Console、發布），獨立的「管理後台階段」在本文件併入 Phase 08／10／11 的相關工作項（見 §3.11），待 Phase 00 審查確認是否單列。

## 3. 各階段

每階段皆含：目標、工作項、依賴、驗收條件、阻塞項。每階段完成後停止審查（見 §5），審查通過前不得進入下一階段。以下「未完成」為撰寫時的狀態（Phase 00 進行中，其餘階段皆未開始），不得寫成已完成。

### Phase 00 — 架構與規格（手冊 B）

- 目標：建立完整產品架構與工程規格，不寫大量應用程式碼；完成後停止，交架構審查報告，不得宣稱產品完成。
- 工作項：PRD、功能清單、五平台功能矩陣、系統架構圖、模組依賴圖、資料流圖、身分與權限模型、ERD、API 命名規範、WS Event Contract、錯誤碼規範、安全性架構、E2EE 研究與風險文件、AI 資料處理規範、自架部署架構、測試策略、版本與發布策略、完整 Roadmap、決策文件（ADR）。本文件為其中一部分（ERD、API、WS、錯誤碼、測試、自架部署、Roadmap），其餘文件由其他工人負責，本文件不代寫。
- 依賴：SPEC-SOURCE.md（§20 表名、§21 前綴／事件名為準）、MASTER_SPEC.md（功能補充）、CONTRACTS.md（舊合約參考）。
- 驗收條件：功能定義明確、模組責任清楚、平台適配策略明確、資料流與權限無明顯缺口、決策有文件、順序與依賴明確；表名／前綴／事件名逐字核對無誤；無編造的額度／數字／效能指標；架構調整後的新技術棧敘述一致、舊付費服務名不殘留（比較文件除外）。
- 阻塞項：階段編號對照待確認（見 §2）；前綴解讀待確認（見 API_STANDARDS.md §2.1）；保留期限、搜尋／向量選擇、Keycloak 配置等待定稿事項見各文件缺口清單；本機 VM 無 flutter／go／docker，本地驗證待環境就緒。

### Phase 01 — 基礎建設（手冊 C）

- 目標：Monorepo、全部專案可建置、Docker 啟動、API 健康檢查、App／Web／Admin 啟動、資料層連線、CI 自動檢查、無硬編碼憑證。
- 工作項：Monorepo（`apps/mobile`、 `apps/admin`、 `apps/web`、 `services/api`、 `infrastructure`、 `docs`，目錄見 §26）；Flutter Mobile＋Desktop、React Web＋Admin、Go Backend、Realtime Gateway、Worker、PostgreSQL 自架、Valkey、NATS JetStream、Docker Compose＋Coolify、Traefik、Keycloak、MinIO、Meilisearch、OpenBao、GitHub Actions、OpenTofu（選用）、環境變數／Secrets 管理（OpenBao）、Health Check、Logging（Loki）、OpenTelemetry、CI；區分 local／development／staging／production；Production Secrets 不入 Git；容器 Health Check；Migration；Makefile／Task Runner；版本固定；格式化／Lint；Commit Convention；PR 檢查；部署文件。
- 依賴：Phase 00 審查通過；ERD（Migration 依據）、API 命名規範、部署架構。
- 驗收條件（手冊 C）：全部專案可建置、Docker 啟動、API 健康檢查通過、App／Web／Admin 啟動、PostgreSQL／Valkey／NATS 連線成功、CI 自動檢查通過、無硬編碼憑證。驗證環境為「待 CI 或有 Docker 的環境驗證」。
- 阻塞項：本機 VM 無 flutter／go／docker、iOS 須走 macOS CI；Keycloak／MinIO／Meilisearch／OpenBao 的配置定稿；正式部署成本待查證（阻塞項，見部署文件）。

### Phase 02 — 設計系統與品牌（手冊 D）

- 目標：品牌與設計系統落地，共用元件實際可用、風格一致、四尺寸版型驗證。
- 工作項：Logo、Icon、色彩、字體、圖形、插畫、啟動畫面、Loading／Empty／Error 插畫；Design Tokens（Color／Typography／Spacing／Radius／Border／Shadow／Elevation／Motion／Icon／Z-Index，Light／Dark）；Flutter 元件庫；React 元件庫（視覺一致）；平台差異化（Mobile 底導航／手勢／附件面板；Desktop 多欄／側欄／快捷鍵／拖放／托盤；Web 響應式／PWA／鍵盤；Admin 表格／篩選／圖表／審核）。
- 依賴：Phase 01 基礎建設；SPEC-SOURCE.md §03 視覺設計規範、§04 導覽架構。
- 驗收條件（手冊 D）：共用元件實際可用、風格一致、四尺寸版型驗證通過。品牌原創，不得用第三方受保護素材。
- 阻塞項：品牌素材的原創確認；字體授權查證（待查證，不得預寫）。

### Phase 03 — 身分帳號（手冊 E）

- 目標：完整註冊／登入／登出／重登／遠端登出流程，資料真正寫入資料庫。
- 工作項：註冊、Email 驗證、手機 OTP、密碼、Google、Apple、MFA、重設、Token Refresh、Session、登入紀錄、裝置管理、遠端登出、停用、刪除、檔案、ID、頭像、狀態、QR。安全：Keycloak（OIDC／OAuth 2.0、PKCE、短效 Access Token、Refresh Rotation、安全 Cookie、Secure Storage、Session Revocation、Rate Limit、防濫用、裝置信任）。保留 Email 驗證、登入登出、OAuth、MFA、Session、Access／Refresh Token、裝置管理、遠端登出、密碼重設，不得破壞使用者／好友／裝置關聯。
- 依賴：Phase 01／02；身分與權限模型、ERD（`users`、`user_profiles`、`user_settings`、`user_devices`、`user_sessions`、`user_security_keys`）。
- 驗收條件（手冊 E）：兩帳號多裝置完成註冊、登入、登出、重登、遠端登出，資料真正寫入資料庫。
- 阻塞項：Keycloak 配置定稿；手機 OTP 供應商與成本待查證（仍可能付費，見部署文件）；Google／Apple OAuth 憑證取得（未取得的憑證不得當已完成整合）；簡訊 OTP 成本待查證。

### Phase 04 — 好友聯絡人（手冊 F）

- 目標：好友關係完整流程，三帳號互搜互邀、接受、封鎖、解封、刪除。
- 工作項：ID／電話／Email 搜尋、QR、邀請流程、列表、備註、分類、隱藏、封鎖、刪除、推薦、聯絡人同步、邀請連結。聯絡人同步須授權；拒絕通訊錄權限不得阻擋其他功能；封鎖／刪除／隱藏／好友四種狀態分開建模。
- 依賴：Phase 03；ERD（好友 5 張表）；API 前綴 `/friends`、`/contacts`。
- 驗收條件（手冊 F）：三帳號互搜互邀、接受、封鎖、解封、刪除，狀態正確寫入資料庫，封鎖語義（單向、訊息不補送）驗證通過。
- 阻塞項：電話搜尋隱私預設待決定；聯絡人同步的雜湊與授權撤回流程待隱私文件定稿。

### Phase 05 — 即時訊息核心（手冊 G）

- 目標：兩帳號至少兩台實際裝置，真正即時交換訊息；不許單機假資料測試。
- 工作項：WS Gateway（認證、Registry、Heartbeat、重連、多裝置 Session、路由、恢復、Rate Limit、Backpressure、Graceful Shutdown）；訊息（文字、回覆、引用、編輯、收回、刪除、轉傳、置頂、反應、已讀／送達回條、輸入中、上線狀態、離線投遞）；可靠性（唯一 ID、Client ID、冪等、每對話 Sequence、Transactional Outbox、ACK、Retry、去重、離線佇列、Cursor 分頁、歷史同步）；聊天 UI（列表、聊天室、泡泡、Composer、輸入中、已讀、未讀計數、回覆預覽、選單、搜尋、日期分隔、捲動位置恢復）。
- 依賴：Phase 03／04；WS Event Contract、ERD（聊天室 10 張表）、NATS JetStream、Valkey。
- 驗收條件（手冊 G）：兩帳號至少兩台實際裝置，測即時／離線／重連／去重／順序／已讀／多裝置／重啟恢復；不許單機假資料測試。
- 阻塞項：WS 協定定稿（Sequence、ACK、恢復游標）；Transactional Outbox 的採用與否待定稿；推播與即時訊息重複的去重規則待定稿。

### Phase 06 — 媒體群組社群（手冊 H）

- 目標：三帳號以上建群組傳媒體、管成員、社群互動；媒體用真實物件儲存並驗權限。
- 工作項：媒體（上傳、壓縮、縮圖、轉碼、音訊、錄音、分段／續傳、下載、預覽、快取、CDN（Cloudflare Free）、安全掃描，儲存為 MinIO）；群組（建立、名稱、頭像、邀請、連結、成員、角色、權限、公告、置頂、@、投票、已讀、離開、解散、管理紀錄）；社群（公開／私密、搜尋、申請、審核、管理員、子頻道、公告、活動、投票、檢舉）。
- 依賴：Phase 05；ERD（群組 6 張、社群 5 張、媒體 4 張）；MinIO、Meilisearch（搜尋）。
- 驗收條件（手冊 H）：三帳號以上建群組傳媒體、管成員、社群互動；媒體用真實物件儲存並驗權限。
- 阻塞項：群組／社群與 `conversations` 的對應方式待定稿（見 ERD.md 缺口）；媒體掃描供應商／方式待定稿；大型群組扇出策略待定稿。

### Phase 07 — 語音視訊會議（手冊 I）

- 目標：不同網路、不同平台實際通話，不得模擬畫面。
- 工作項：WebRTC＋LiveKit Community Edition 自架、STUN／TURN、SFU、信令、權限、Audio Session、Push Call Notification；一對一語音／視訊、多人、來電、接／拒／忙線、靜音、鏡頭切換、螢幕分享、參與者管理、主持人、紀錄、品質監測、重連。平台：Android 背景來電、iOS CallKit、Windows／macOS 原生通知、Web 權限處理。
- 依賴：Phase 05／06；ERD（通話 3 張表）；LiveKit CE 部署。
- 驗收條件（手冊 I）：不同網路、不同平台實際通話，不得模擬畫面。
- 阻塞項：TURN 頻寬與公網部署需求須特別評估（仍可能付費）；LiveKit CE 的房間 Token 與 Keycloak 整合待定稿；iOS CallKit 須走 macOS CI。

### Phase 08 — 社群貼圖商業（手冊 J）

- 目標：動態、貼圖、官方帳號、Bot、商業能力落地，開發者文件齊全。
- 工作項：Timeline（發文編輯刪除、圖影、留言回覆、讚、分享、收藏、隱私、檢舉）；貼圖（包、靜／動態、收藏、最近、搜尋、分類、免費／付費、購買紀錄、創作者管理）；官方帳號（認證、客服、多人客服、自動回覆、關鍵字、圖文、快速回覆、圖文選單、標籤、備註）；Bot（申請、Token、Webhook、訂閱、Messaging API、Rich Message、Interactive Card、權限、用量、開發者 Console）；商業（優惠券、票券、商家頁、會員、付款供應商整合、交易紀錄、訂單通知；付款須合規，不得自行儲存信用卡完整資訊）。
- 依賴：Phase 06；ERD（動態 6 張、貼圖 4 張、官方帳號／Bot 5 張）；管理後台相關能力（見 §3.11）。
- 驗收條件：動態／貼圖／官方帳號／Bot 的功能以真實帳號驗證；Bot API 以真實 Webhook 驗證；開發者文件與測試環境齊全；付款以合規第三方沙箱驗證（未取得的憑證不得當已完成）。
- 阻塞項：付款供應商與金流方式待決定；官方帳號認證流程待決定；貼圖商店的價格與稅務待決定；管理後台分工待確認（見 §2）。

### Phase 09 — ETHER AI（手冊 K）

- 目標：每種 AI 功能用真實模型供應商或可驗證服務驗證；無 Key 時只做介面與 Adapter 並標記未完成。
- 工作項：AI 對話（Streaming、模型切換、上下文、歷史、用量、設定、Tool Calling）；訊息 AI（摘要、群組摘要、翻譯、智慧回覆、潤飾、語氣、語音轉文字、文件摘要、圖片理解）；知識 AI（個人知識庫、文件索引、向量搜尋、RAG、語意搜尋、權限隔離、刪除同步）；商業 AI（官方 AI 客服、FAQ、Bot、分類、摘要、回覆建議）。技術：Ollama＋vLLM＋LocalAI 優先＋外部 AI API Adapter 可選擴充（可切換模型、Streaming、Tool Calling、上下文、Token 統計、資源管理；供應商不得寫死）；向量承載為 pgvector（已定案，見 ADR-017）。安全：授權、Provider Adapter、Rate Limit、Token Budget、敏感遮罩、Prompt Injection 防護、Tool 權限、使用紀錄、保留策略。E2EE 明文不得自動送往雲端 AI。
- 依賴：Phase 05／08；AI 資料處理規範、ERD（向量承載選擇）；Ollama／vLLM 部署。
- 驗收條件（手冊 K）：每種 AI 功能用真實模型供應商或可驗證服務驗證；無 Key 時只做介面與 Adapter 並標記未完成。
- 阻塞項：CPU／RAM／GPU／VRAM 評估（不得假設一般伺服器跑得動所有模型）；模型清單與授權待查證；外部 AI API 成本待查證（仍可能付費）；向量承載選擇待定；AI 資料處理規範待定稿。

### Phase 10 — Web／桌面／多裝置（手冊 L）

- 目標：同一帳號五平台登入驗同步；各平台用合理介面，不可複製手機畫面。
- 工作項：Web（完整聊天、好友、群組、多媒體、通知、PWA、Service Worker、Push、Storage、響應式）；Desktop（多欄、系統通知、啟動設定、拖放、快捷鍵、托盤、視窗管理、多視窗、下載、系統分享）；跨裝置（註冊、驗證、多登入、同步、已讀同步、推播 Token、撤銷、金鑰同步）。
- 依賴：Phase 05／06／07；多裝置 Session 與同步設計。
- 驗收條件（手冊 L）：同一帳號五平台登入驗同步；各平台用合理介面，不可複製手機畫面。
- 阻塞項：iOS／macOS 建置須走 macOS CI；Web Push 憑證與配置待定稿；金鑰同步與 E2EE 的關係待 E2EE 文件定稿。

### Phase 11 — 安全效能發布（手冊 M）

- 目標：僅通過者可標完成；須發布審查報告列未解決風險。
- 工作項：安全（OWASP ASVS／MASVS、OAuth、API 權限、Session、SQLi、XSS、CSRF、Rate Limit、檔案上傳、E2EE Protocol Audit、Secrets Audit、Dependency Scan、SAST、DAST）；效能（API／WS／DB 負載、吞吐量、上傳、通話品質、Memory／CPU Profiling）；可靠性（備份、還原測試、災復、Retry、Circuit Breaker、DLQ、Health Check、Alerting、Incident Logging）；發布（Android Release、iOS Archive、Windows Installer、macOS Package、Web／Admin Production Build、Production Env、Release Notes、部署指南、回滾流程）。監控為 Prometheus＋Grafana＋Loki＋GlitchTip＋OpenTelemetry（覆蓋 API、WebSocket、資料庫、Worker、Media、AI）。
- 依賴：全部先前階段；TEST_STRATEGY.md、安全性架構、E2EE 文件、部署架構。
- 驗收條件（手冊 M）：僅通過者可標完成；須發布審查報告列未解決風險；備份還原演練通過；監控告警驗證通過。
- 阻塞項：正式部署成本待查證與使用者決定（阻塞項）；上架憑證（Apple Developer、Google Play）待取得，成本待查證；TURN 頻寬、外部 AI 成本待查證；不得新增付款方式。

### Phase 12 — 最終稽核（手冊 N）

- 目標：逐項比對 PRD、功能矩陣、API、DB、前端、實測；完成度以實際驗收為準，不得因階段執行完就判定完成。
- 工作項：檢查 15 項（未實作、UI 無 API、API 無 UI、Mock、無作用按鈕、權限漏洞、同步錯誤、未處理錯誤、平台落差、不可復原操作、不合理查詢、未授權 AI 傳輸、不安全金鑰、未測外部整合、無法部署）。產出 `docs/audit/FINAL_AUDIT.md`、`FEATURE_COMPLETENESS.md`、`SECURITY_AUDIT.md`、`PLATFORM_MATRIX.md`、`KNOWN_ISSUES.md`、`RELEASE_READINESS.md`。
- 依賴：全部先前階段的驗收證據；PRD、功能清單、平台矩陣、API 文件、ERD、測試報告。
- 驗收條件：稽核文件齊全、15 項檢查逐項有結論與證據、已知問題已列出、發布就緒判定有依據。稽核未通過者不得發布。
- 阻塞項：先前階段的未解決風險；正式部署阻塞項未解除者，發布就緒判定須為「未就緒」。

### 3.11 管理後台（§25 Phase 10 的對照）

- 管理後台（React＋TypeScript，獨立管理員認證與權限）的工作項分散於 Phase 08（官方帳號／Bot 管理、開發者 Console）、Phase 10（Web／Admin 介面）、Phase 11（系統監控、發布）中；獨立的管理後台驗收（儀表板、使用者／社群／貼圖管理、檢舉審核、系統監控）在 Phase 11 前須有專項驗收，驗收場景待管理後台文件定稿。是否單列管理後台階段，待 Phase 00 審查確認（見 §2）。

## 4. 版本與發布策略

### 4.1 版號規則

- 版號形式：語義化版本（`MAJOR.MINOR.PATCH`，草案方向）＋建置號（行動 App 的 `versionCode` 等平台建置號，規則待定稿）。版號規則待定稿確認，不得以草案為已定。
- 升級規則：MAJOR 升級對應破壞性變更（API 主版本升級、資料庫破壞性 Migration、E2EE 協定變更等，範圍待定稿）；MINOR 升級對應新增功能（向後相容）；PATCH 升級對應修復。版號升級須與 API 版本策略（見 API_STANDARDS.md §11）、Migration 版本、發布列車一致。
- 預發布版號：測試版、內測版的版號後綴規則待定稿，不得以正式版號發布測試版。

### 4.2 發布列車

- 發布列車（Release Train）：固定節奏的發布窗口（節奏待決定，不得預寫天數）；每個窗口含功能凍結、測試、審查、發布、監控五站，缺一站不得發車。
- 發布內容：Release Notes（功能、修復、已知問題、API／DB 變更）、部署指南、回滾流程，皆須隨版本發布，不得事後補寫。
- 平台發布：Android（Google Play）、iOS（App Store）、Windows Installer、macOS Package、Web／Admin Production Build，各平台的發布流程、審查、憑證，待發布文件定稿；上架成本待查證（仍可能付費）。

### 4.3 回滾

- 回滾流程：版本回滾、資料庫 Migration 回滾、配置回滾、映像回滾，各自的步驟、前置條件、驗證方式，待定稿；破壞性 Migration 的回滾須特別定稿（不可回滾的 Migration 須先有備份與還原驗證）。
- 回滾演練：在 staging 演練回滾，不得在 production 首次演練；回滾演練結果須留存。
- 回滾判定：發布後監控告警、錯誤率、健康檢查失敗時的回滾判定標準與負責人，待定稿；回滾不得以「重啟服務」代替版本回滾而不說明。

## 5. 每階段完成後停止審查的門檻

每階段完成後，皆須停止並審查，審查通過前不得進入下一階段。審查門檻（通用，適用全部階段）：

1. 本階段的完成功能清單與驗收條件逐項核對，驗收證據齊全（測試報告、日誌、截圖／錄影、訊息識別碼等），未通過者列為未完成項，不得標完成。
2. 新增／修改檔案清單、DB 變更（Migration）、API 變更（OpenAPI 更新）齊全，文件已同步更新。
3. 測試結果齊全（Unit／Integration／E2E／Security／Performance 按階段適用範圍），未完成項、已知問題已列出，下一階段已明確。
4. 無假資料冒充、無 Mock 冒充後端、無無作用按鈕、無未處理錯誤、無硬編碼憑證、無管理員權限交前端判斷。
5. 阻塞項已列出並有負責人／決定者；需使用者決定的事項已一件一件提出，不得以文件代替提問。
6. 審查報告已產出，審查者（統籌／使用者）已簽核；未簽核不得進入下一階段。

Phase 00 的審查門檻見各 Phase 00 文件的審查門檻節；Phase 12 的審查門檻見 §3 Phase 12。

## 6. 缺口與待決定事項

1. 階段編號對照（§25 與手冊 B–N）待 Phase 00 審查確認（見 §2）。
2. 管理後台是否單列階段，待確認（見 §3.11）。
3. 時程、工時、成本，一律待查證／待決定，本文件不預寫。
4. 版號規則、發布列車節奏、回滾流程，待定稿。
5. 正式部署阻塞項（成本、伺服器、網域）待查證與使用者決定，不得新增付款方式。
6. 各階段的詳細設計、測試案例、API 端點清單，待各階段文件定稿，本文件不代寫。

## 7. 品牌發展藍圖對照（品牌企劃書 §14）

品牌藍圖 Phase 01–05 與產品開發階段的對應關係（品牌工作不得超前產品能力宣傳，不得誇大未實現功能）：

| 品牌藍圖 | 內容 | 對應產品階段 |
|---|---|---|
| Phase 01 品牌根基 | 名稱、全名、故事、使命願景價值、定位、語言規範 | Phase 00（本文件＋`docs/brand/BRAND_GUIDELINES.md`） |
| Phase 02 視覺識別 | Logo、App Icon、色彩、字體、圖形語言、動態識別、使用手冊 | Phase 02 設計系統（手冊 D） |
| Phase 03 產品識別 | 設計系統、跨平台介面、元件圖示、動效、通知互動、AI 品牌體驗 | Phase 02–03 設計系統與帳號體驗 |
| Phase 04 品牌發布 | 官網、影片、社群帳號、展示頁、商店素材、初期測試、正式發布 | Phase 11 發布前（須先完成商標／網域／商店名稱檢查，皆為阻塞項） |
| Phase 05 品牌擴張 | AI、社群、創作者、企業服務、開發者平台、第三方整合、國際市場 | Phase 08–10 之後，依產品實際能力分批 |

商標近似檢索、商標類別評估、網域與社群帳號檢查、App Store／Google Play 名稱檢查，皆為品牌對外使用前阻塞項，未完成前不得對外使用 ETHER 品牌。
