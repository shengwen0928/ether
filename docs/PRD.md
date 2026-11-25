狀態：草案／待批准

# ETHER 產品需求文件（PRD）

> 依據：`~/workspace/ether/SPEC-SOURCE.md`（最高指導摘要，表名／API 前綴／事件名／模組名以它為準）、`~/workspace/messenger-app/MASTER_SPEC.md`（14 面向功能總規格草案，功能深度補充）、`~/workspace/messenger-app/PLAN.md` 與 `CONTRACTS.md`（舊 Kotlin＋Supabase 路線，僅參考，技術已由 ETHER 規格取代）。
>
> 本文件為 Phase 00 產品文件，不構成完成宣告。任何功能在以真實裝置、真實帳號、真實後端、真實資料驗收之前，一律視為未驗證，不得宣稱已完成或已上線。成本一律寫「待查證」，不得編造額度、數字或效能指標。
>
> 架構調整（使用者 2026-11-25 00:33 指令，優先於舊技術棧）：全面替換付費服務，改採免費及開源自架方案，詳見 §1.4 與 `docs/COST_OPTIMIZATION.md`。舊服務名只准出現在比較文件，不得殘留在架構正文。功能不得刪、不得降級，不得宣稱全系統零成本。Docker Compose 啟動／連線／上傳／訊息／AI 切換／LiveKit／監控皆為「待 CI 或有 Docker 的環境驗證」，此 VM 無 docker，不得寫已驗證。

## 1. 產品定位

### 1.1 專案名稱與一句話定位

ETHER——全功能跨平台即時通訊生態系統。LINE 等級功能完整度，真正可註冊、登入、加好友、聊天、建群組、傳檔案、語音／視訊通話、收通知、管理社群的完整產品，不是 Demo、不是只有前端畫面。

### 1.2 產品標準

以成熟商業級 App 體驗、上架 Google Play 與 Apple App Store 的產品標準開發。不得用假資料冒充真實功能、不得有無作用按鈕、不得以靜態畫面代替後端。所有功能須有明確資料模型、API、狀態管理、錯誤處理。

### 1.3 品牌原則

品牌：ETHER（Every Thread Holds Endless Remembrance），定位 A Human Connection Platform，標語 One Connection. Endless Possibilities.；核心為連結、情感、記憶、陪伴、無界；視覺方向 Ethereal Modernism（空靈現代主義），詳見 `docs/brand/BRAND_GUIDELINES.md` 與 `docs/brand/DESIGN_TOKENS.md`，Phase 02 設計系統以其為準。品牌原創，不得使用 LINE 的 Logo、角色、貼圖或受保護素材。名稱、標誌、色彩、圖示、貼圖角色、預設主題、預設鈴聲皆須原創；自訂貼圖上傳須由上傳者確認其擁有權利或授權。品牌正式對外使用前須完成商標近似檢索、商標類別評估、網域與社群帳號檢查、App Store／Google Play 名稱檢查（皆為阻塞項，未完成）。

### 1.4 技術棧（已確定，不得擅自更換）

- 行動／桌面：Flutter＋Dart；視覺以 ETHER Design Tokens（Ethereal Modernism，見 `docs/brand/DESIGN_TOKENS.md`）為準，Material 3 僅為 Flutter 實作基底之一，不得以 Material 3 預設外觀代替品牌設計；Riverpod；GoRouter；Dio；WebSocket；Drift／SQLite；flutter_secure_storage；WebRTC；FCM（Android）；APNs（iOS）。
- Web／Admin：React＋TypeScript。
- 後端：Go；Echo v5（≥v5.4.7，已定案，見 ADR-016）；PostgreSQL（自架）；Valkey；WebSocket；NATS JetStream；gRPC 內部通訊；MinIO 物件儲存；Meilisearch；Docker Compose＋Coolify；OpenTofu 基礎設施即程式碼；Traefik 反向代理；Keycloak 身分驗證；OpenBao 憑證管理。
- 通話：WebRTC＋LiveKit Community Edition 自架；STUN／TURN；SFU；TURN 頻寬與公網部署需求須特別評估，成本：待查證。
- AI：Ollama＋vLLM＋LocalAI 優先＋外部 AI API Adapter 可選擴充（AI Provider Adapter，不得讓商業邏輯直接依賴特定供應商 SDK）；向量檢索用 pgvector（已定案，見 ADR-017）。
- 部署與可觀測性：Docker Compose＋Coolify 自架部署；Cloudflare Free（DNS 與內容分發）；Coraza WAF；Let's Encrypt 憑證；Prometheus＋Grafana＋Loki＋OpenTelemetry＋GlitchTip 自架監控與錯誤追蹤；CI／CD 為 GitHub Actions。
- 身分驗證：Keycloak（Realm／Client／角色概念，詳見 `docs/architecture/IDENTITY_AND_PERMISSIONS.md`）；保留 Email 註冊驗證、登入登出、OAuth、MFA、Session、Access／Refresh Token、裝置管理、遠端登出、密碼重設。
- 平台：Android、iOS、Windows、macOS、Web 五平台。

### 1.5 後端模組（22 個，責任邊界明確）

Authentication、User、Profile、Friend、Contact、Chat、Message、Group、Community、Media、Sticker、Call、Notification、Search、Official Account、Bot、Timeline、Payment Integration、Moderation、Administration、Analytics、Device Management。初期可為模組化單體，不得互相依賴混亂。所有重要資料操作必須經過後端驗證；重要商業邏輯不得只寫在前端；管理員權限不得由前端自行判斷。

### 1.6 舊路線對照

舊 Kotlin＋Supabase 路線（`PLAN.md`、`CONTRACTS.md`、Android 先行）已被 ETHER 規格取代（Flutter＋Go＋PostgreSQL 自架／Valkey／NATS／MinIO／Meilisearch、五平台、22 模組）。`MASTER_SPEC.md` 的功能分析可重用；其 Supabase 表名與 ETHER §20 表名不同，以 ETHER §20 為準，差異寫入 ADR。`CONTRACTS.md` 的 Supabase Realtime 頻道概念可參考，傳輸層改為 ETHER WebSocket 事件協定（§21），以 ETHER 為準，差異寫入 ADR。本文件不覆蓋、不刪除舊路線文件。

### 1.7 仍可能付費項目摘要（不得宣稱零成本）

本產品改採免費及開源自架方案，不等於全系統零成本。仍可能付費的項目，摘要如下，逐項分析見 `docs/COST_OPTIMIZATION.md`：網域、伺服器（含自架主機的運算、儲存與頻寬成本）、簡訊 OTP、Apple Developer、Google Play 上架、外部 AI API、TURN 頻寬等。任何一項的金額一律寫「待查證」，不得編造。部署與維運的人力、自架服務的備份與監控責任，仍由營運方承擔，不得以「免費方案」為由忽略。

## 2. 目標使用者

### 2.1 主要使用者

- 一般使用者：以即時訊息、語音／視訊通話、群組與社群進行日常溝通、聯絡與社群參與的人。
- 群組與社群參與者：需要多人聊天、公告、投票、活動、多聊天室與分級管理的成員與管理者。
- 內容與社群經營者：經營動態、貼圖、官方帳號與機器人客服的個人或組織。
- 平台管理員：使用 Web 管理後台進行使用者、群組、社群、內容、檢舉與系統管理的管理員（獨立管理員認證與權限，與一般使用者帳號分離）。

### 2.2 使用情境與裝置

使用者在手機、平板、桌面與網頁之間切換，同一帳號多裝置使用，包含低網速與離線情境。須支援不同螢幕尺寸、平板、深淺色模式、動態字體、無障礙、多語系、高 DPI。

### 2.3 非目標與排除

本文件不將下列事項列為本產品的承諾：以假資料冒充的功能、以靜態畫面代替的後端、未經批准的金流與付費承諾、未經審計的 E2EE 安全宣稱。遠期項目見 `FEATURE_LIST.md`，實作前不得宣稱可用。

## 3. 核心目標 10 項

對照 SPEC-SOURCE 專案定位，核心目標共 10 項：

1. Android＋iOS App：可安裝、可註冊登入、可日常使用的行動應用。
2. 可擴充即時通訊後端：模組化、可水平擴充的 Go 後端與資料底座。
3. 可持續運作的即時訊息傳輸架構：WebSocket 即時傳輸、斷線重連、離線投遞、多裝置同步與去重。
4. 使用者／好友／群組／社群系統：完整的身分、關係、群組與社群治理。
5. 多媒體與檔案傳輸：圖片、影片、語音、檔案的安全上傳、預覽、快取與權限驗證。
6. 語音／視訊通話：WebRTC＋LiveKit Community Edition 自架的一對一與多人通話，保留房間與參與者、螢幕分享、STUN／TURN、SFU、品質監控。
7. 官方帳號與機器人：官方帳號、客服、自動回覆與 Bot API。
8. 通知推播：跨平台推播與 App 內通知，尊重隱私與靜音設定。
9. Web 管理後台：React＋TypeScript 的獨立管理後台，含儀表板與審核流程。
10. 安全性／隱私／權限管理：身分、權限、檢舉、封鎖、資料保護與 E2EE 路線。

## 4. 功能總覽（對照 §03–§19）

下表為功能面向總覽，逐項可驗收清單見 `FEATURE_LIST.md`，平台差異見 `PLATFORM_MATRIX.md`。

| 規格節 | 面向 | 摘要 | 主要階段 |
|---|---|---|---|
| §03 | 視覺設計規範 | 精緻、現代、高級、友善、清晰、有品牌識別度；Design Token（Light／Dark 兩套）；完整元件庫與狀態 | Phase 02（手冊 D） |
| §04 | 導覽架構 | 底部五入口：聊天、聯絡人、探索、動態、我的；聊天首頁即時更新，不得要求手動刷新 | Phase 01–02、Phase 10 |
| §05 | 帳號系統 | 手機／Email／驗證碼／密碼／OAuth（Google、Apple）、生物辨識、裝置信任、MFA、停用／刪除／匯出 | Phase 03（手冊 E） |
| §06 | 好友系統 | ID／電話／QR／連結／聯絡人同步／推薦／群組內新增；邀請生命週期；刪除／封鎖／隱藏為不同狀態 | Phase 04（手冊 F） |
| §07 | 一對一聊天 | 文字、多媒體、語音訊息、回覆／引用／反應／轉傳／編輯／收回／刪除、已讀、搜尋、跨裝置同步 | Phase 05（手冊 G） |
| §08 | 群組 | 建立、Owner／Admin／Member 角色、公告、@、投票、已讀資訊、邀請管理、轉移所有權、解散 | Phase 06（手冊 H） |
| §09 | 社群 | 公開／私密、多聊天室、子群組、討論區、活動、投票、檢舉、黑名單、分級權限與操作紀錄、資料分析 | Phase 06（手冊 H） |
| §10 | 通話 | 一對一語音／視訊、多人通話、通話紀錄、平台來電整合（Android 背景來電、iOS CallKit、桌面原生通知、Web 權限） | Phase 07（手冊 I） |
| §11 | 貼圖與主題 | 貼圖包／商店、主題、自訂貼圖；一律原創 | Phase 08（手冊 J） |
| §12 | 動態 | 發布、按讚、留言、分享、收藏、隱藏、檢舉、隱私範圍、編輯／刪除 | Phase 08（手冊 J） |
| §13 | 官方帳號與 Bot | 官方帳號、客服、自動回覆、Bot API、Webhook、開發者文件與測試環境 | Phase 08（手冊 J） |
| §14 | 通知 | 新訊息、邀請、公告、互動、來電、系統、安全警告；分類、靜音時段、個別聊天室控制、預覽隱私 | Phase 05 起、Phase 11 |
| §15 | 搜尋 | 好友、使用者、群組、社群、官方帳號、聊天室、訊息、檔案、照片、影片、貼圖、動態；遵守存取權限與封鎖規則 | Phase 05 起、Phase 08 |
| §16 | 生活服務（預留入口） | 付款／錢包、優惠券、票券、活動、商家、會員卡、迷你應用、外部網站、服務通知、地圖、位置分享、行事曆整合 | 遠期／預留入口，見未決問題 |
| §17 | 隱私與安全 | 資料保護、E2EE、隱私控制、檢舉封鎖；事後補充不算，須自始納入設計 | 貫穿各階段、Phase 11 門檻 |
| §18 | 設定中心 | 帳號、隱私、通知、聊天、外觀、其他（語言、地區、幫助、條款、隱私政策、版本） | Phase 03 起、Phase 10 |
| §19 | Web 管理後台 | 儀表板、使用者管理、群組社群管理、貼圖內容管理、系統管理；管理員不得任意讀取 E2EE 訊息 | Phase 10（§25 Phase 10） |

補充說明：

- 訊息狀態 7 種（§07）：Pending、Sending、Sent、Delivered、Read、Failed、Recalled。訊息須有唯一 ID、時間戳記、發送者、接收對象、內容、狀態。
- 可靠性要求（§22）：Message ID、Idempotency Key、Sequence Number、ACK、Retry、Exponential Backoff、Message Persistence、Offline Queue、Delivery Receipt、Reconnection Recovery。不得將 WebSocket 連線本身當成可靠送達保證。
- WebSocket 事件名以 §21 為準：`message.send`、`message.received`、`message.delivered`、`message.read`、`message.recalled`、`typing.start`、`typing.stop`、`presence.online`、`presence.offline`、`conversation.updated`、`group.member.joined`、`group.member.left`、`call.incoming`、`call.accepted`、`call.ended`。每個事件須有事件 ID、時間戳記、版本、關聯識別碼。

## 5. 非功能需求

### 5.1 效能

- 須處理大量同時在線、高頻訊息、大型群組、大量媒體上傳、高併發搜尋與多區域部署情境，並以 Connection Pool、Valkey Cache、Message Queue（NATS JetStream）、索引、Cursor Pagination、Cloudflare Free、MinIO 物件儲存、Background Jobs、Horizontal Scaling、Observability（Prometheus＋Grafana＋Loki＋OpenTelemetry）支援水平擴充。
- 不得在無壓力測試下宣稱可承載特定同時在線數。任何併發、延遲或吞吐量數字，須以壓力測試證據為準，本文件不預填數字；測試策略見 `docs/testing/TEST_STRATEGY.md`（Phase 00 另一文件工人負責，本文件不代寫）。
- 列表與搜尋須以分頁載入，不得一次載入全部資料；分頁方式以 Cursor Pagination 為方向。

### 5.2 安全

- 傳輸與儲存保護：HTTPS／TLS、敏感資料加密、安全憑證儲存、API 權限驗證、Token 管理、輸入驗證、Rate Limiting、防暴力破解、防帳號接管、SQL Injection／XSS／CSRF 防護、檔案上傳安全檢查、稽核紀錄。
- 身分與權限：OAuth 2.0／OIDC＋PKCE、短效 Access Token＋Refresh Token Rotation、Session 管理、裝置信任、MFA、遠端登出；權限檢查點一律在後端強制，前端不得自判。詳見 `docs/architecture/IDENTITY_AND_PERMISSIONS.md`。
- E2EE：優先採用公開審查過的成熟協定（如 Signal Protocol），不得自發明加密演算法；E2EE 訊息伺服器不得明文儲存或讀取；須明確區分 E2EE 聊天室與一般聊天室的功能差異。詳見 `docs/security/E2EE_RESEARCH.md`。未經審計不得上線，E2EE Protocol Audit 列為 Phase 11 門檻。
- 檔案安全：檔案須經權限驗證、格式檢查、惡意檔案掃描；可執行檔不得自動開啟或自動執行。
- 金鑰與憑證：不得將敏感金鑰寫入原始碼；Production Secrets 不入 Git；外部服務須有 Adapter，不得讓商業邏輯直接依賴特定供應商 SDK。
- AI 安全：E2EE 明文絕對不得自動送往任何 AI（含本地自架 AI 與外部 AI API）；詳見 `docs/security/AI_DATA_POLICY.md`。

### 5.3 可靠性

- 須處理斷網、WebSocket 自動重連、重複、遺失、亂序、離線訊息、多裝置同步、送達確認、已讀同步、伺服器重啟、快速切網、App 背景、App 被系統終止、推播與即時訊息重複、大量同時上線。
- 訊息唯一識別碼、版本控制、冪等，避免重複；離線佇列須持久化，恢復連線後依序重放。
- 備份、還原測試、災難復原、Retry、Circuit Breaker、DLQ、Health Check、Alerting、Incident Logging 列為 Phase 11 可靠性驗收範圍。
- 監控與告警、備份策略、版本更新與相容期的具體門檻與期限均待決定，本文件不預填數字。

### 5.4 可維護性

- Monorepo 目錄規範（§26）為準：`apps/mobile`、`apps/admin`、`apps/web`、`services/api`、`infrastructure`、`docs`。模組責任邊界清晰，不得全塞單一巨大檔案；不得忽略錯誤處理；所有操作須有 Loading／Success／Empty／Error 狀態。
- 每個功能須有 UI、狀態管理、API、資料庫、權限驗證、錯誤處理、測試、文件。每階段須提供：完成功能、新增／修改檔案、DB 變更、API 變更、測試結果、未完成項、已知問題、下一階段。
- 技術決策不確定時先提 ADR；外部服務／金鑰／憑證／第三方帳號需求須明列，不得假裝完成。
- 響應式佈局：支援不同螢幕尺寸、平板、深淺色模式、動態字體、無障礙、多語系、高 DPI、低網速與離線。

### 5.5 無障礙與法規

- 無障礙：語音輔助相容、字體大小與系統設定連動、高對比、僅眼睛／語音操作相容的方向，承接 `MASTER_SPEC.md` 第 11 節；是否全數納入 ETHER 範圍與驗收方式為規格缺口，見第 8 節。
- 法規：個資法遵循、未成年人保護、資料保存政策、資料主體權利、跨境傳輸揭露，承接 `MASTER_SPEC.md` 第 14 節；年齡門檻、保存期限等均待法律意見與使用者決定，本文件不提供法律意見、不代決定。

## 6. 上線標準

最終驗收以真實裝置、真實帳號、真實後端、真實資料為準；不允許單機假聊天、Mock API、靜態 UI、假按鈕宣稱完成。具體而言：

1. 真實裝置驗收：Android、iOS、Windows、macOS、Web 各平台須以真實裝置或真實建置環境驗收；iOS 建置在 Linux 上不可能，須走 macOS CI runner；本機 VM 現況為有 node、無 flutter／go／docker，文件須如實記載，不得假裝工具鏈齊全。
2. 真實帳號驗收：註冊、登入、加好友、聊天、群組、通話、推播、登出等端到端流程，須以兩個以上真實帳號、在不同裝置上完成（E2E 12 步，見 §24）。
3. 真實後端驗收：功能須接真實 Go 後端、PostgreSQL 自架、Valkey、NATS JetStream、MinIO 物件儲存與 Meilisearch，不得以 Mock API 或固定假資料冒充即時訊息。Docker Compose 啟動／連線／上傳／訊息／AI 切換／LiveKit／監控皆為「待 CI 或有 Docker 的環境驗證」，此 VM 無 docker，不得寫已驗證。
4. 真實資料驗收：訊息、媒體、通話、搜尋、通知須以真實資料流驗證，含離線、重連、去重、順序、已讀、多裝置與重啟恢復。
5. 安全與 E2EE 門檻：安全測試（權限繞過、Token 竊取、注入、XSS、Rate Limit、非法檔案上傳、未授權讀取、帳號接管）須通過；E2EE 未經 Protocol Audit 不得上線（Phase 11 門檻）。
6. 效能門檻：效能測試（延遲、WebSocket 連線數、吞吐量、API 回應、DB 負載、媒體上傳）須以實測數據為準，不得以編譯通過或規格文字代替驗收。
7. 發布審查：Phase 11 須產出發布審查報告，列出未解決風險；Phase 12 最終稽核逐項比對 PRD、功能矩陣、API、資料庫、前端與實測，僅通過者可標完成；完成度以實際驗收為準，不得因階段執行完就判定完成。
8. 禁 Mock 宣稱完成：任何以 Mock、假資料、靜態畫面、無事件按鈕完成的宣稱，一律視為未完成；未完成功能須明列，不得標為完成。

## 7. 未決問題清單（只列問題，不代決定）

下列問題須由使用者決定，決定後寫入 ADR 或新版合約，不得由文件或實作人員自行決定。編號沿用 `MASTER_SPEC.md` 的 OQ 編號以便對照，ETHER 新增問題另列為 NQ。

| 編號 | 問題 |
|---|---|
| OQ-01 | 貼圖市集／付費內容的金流供應商與方式為何？（含稅務與手續費處理。） |
| OQ-02 | 以電話號碼加好友的隱私預設為何？（預設可搜尋或預設關閉？可否逐一設定例外？） |
| OQ-03 | 各類資料的保存期限各為多少？（訊息、媒體、日誌、備份、檢舉紀錄，是否分別訂定？） |
| OQ-04 | 雙重驗證（MFA）的第二因素方式為何？（含備用碼的產生與保管方式。） |
| OQ-05 | 聊天備份的加密口令 UX 為何？（口令設定、確認、遺失處理、換機移轉時的口令傳遞方式。） |
| OQ-06 | E2EE 的啟用範圍與時程為何？（是否全部對話預設啟用？群組是否同樣啟用？啟用前伺服器端搜尋與備份的替代方案為何？） |
| OQ-07 | 公開社群的內容審查人力與流程為何？（版主是否為志願者？平台是否設專責審查人員？審查時限為何？） |
| OQ-08／OQ-10 | 官方帳號的認證審查流程與申請文件為何？（須提交何種組織證明文件？審查期限與撤銷條件為何？） |
| OQ-09 | 訊息翻譯的服務供應商與支援語言為何？（翻譯內容是否會傳送至第三方服務？準確度驗收方式為何？） |
| OQ-11 | TURN 頻寬與 LiveKit Community Edition 自架部署：TURN 頻寬成本與公網部署需求如何評估？是否設流量上限保護？成本：待查證。 |
| OQ-12 | 監控告警的通知管道與負責人為何？（告警以何種管道發送？夜間告警處理流程為何？） |
| OQ-13 | 未成年人帳號的年齡門檻與法定代理人同意方式為何？（年齡門檻為幾歲？同意以何種方式驗證？未確認前是否暫停未成年人註冊？） |
| OQ-14 | 推播供應商與設定方式為何？（FCM／APNs 之外的推播服務是否需要？裝置權杖管理方式為何？推播內容是否含訊息摘要？成本：待查證。） |
| OQ-15 | 本機多帳號切換是否納入範圍？（切換時的本機快取清除範圍為何？） |
| NQ-01 | 金流供應商：生活服務（§16）付款／錢包整合採用哪一家合法合規第三方？不得自行儲存信用卡完整資訊；成本：待查證。 |
| NQ-02 | OTP 供應商：手機驗證碼（SMS／語音）採用哪一家供應商？發送頻率、有效期與防濫用規則為何？成本：待查證。 |
| NQ-03 | TURN／LiveKit Community Edition 自架：自架主機的部署區域、頻寬與維運成本為何？成本：待查證。 |
| NQ-04 | 翻譯供應商：訊息翻譯與 AI 翻譯是否同一供應商？支援語言與資料邊界為何？成本：待查證。 |
| NQ-05 | 推播供應商：除 FCM／APNs 外，Web Push 與桌面通知的供應商與實作方式為何？成本：待查證。 |
| NQ-06 | AI 供應商：AI Provider Adapter 首批接入的本地模型（Ollama／vLLM／LocalAI）與可選外部 AI API 為何？無 Key 時的介面範圍為何？成本：待查證。 |
| NQ-07 | 物件儲存與內容分發：MinIO 自架物件儲存與 Cloudflare Free 的部署區域、儲存與頻寬成本為何？成本：待查證。 |
| NQ-08 | 惡意檔案掃描供應商與檔案類型允許清單為何？ |
| NQ-09 | 搜尋服務：Meilisearch 自架的部署方式、索引範圍與 E2EE 訊息的搜尋邊界為何？ |
| NQ-10 | 無障礙範圍：`MASTER_SPEC.md` 第 11 節的僅眼睛／語音操作相容，是否全數納入 ETHER 範圍？驗收方式為何？ |

## 8. 規格缺口（本文件發現，只列、不補決定）

1. `MASTER_SPEC.md` 的大量數字門檻（文字長度、檔案大小、群組人數、通話人數、重送次數、逾時秒數等）皆標「暫定」，未經實測覆核；ETHER 是否沿用、修改或重新定案，尚未決定。
2. ETHER §20 表名與舊路線表名差異的 ADR 尚未撰寫（例如 `profiles` 與 `users`／`user_profiles`、`friendships` 與 `friendships`／`friend_requests`、`attachments` 與 `message_attachments` 等），須由架構文件工人補 ADR，本文件不代定。
3. 舊 `CONTRACTS.md` 的 Supabase Realtime 頻道與事件，與 ETHER §21 WebSocket 事件的對照與遷移規則尚未定案。
4. 社群分級權限的具體等級與操作對照表尚未定案（僅有表名 `community_roles`）。
5. 管理員權限的具體權限項（`admin_permissions` 表的權限列舉）尚未定案。
6. 生活服務（§16）的迷你應用安全模型、商家入駐流程、付款合規細節尚未定案。
7. E2EE 群組訊息的金鑰分發與成員變更處理、伺服器端搜尋與備份的替代方案尚未定案。
8. 本機工具鏈缺口：本機 VM 無 flutter、無 go、無 docker，iOS 須走 macOS CI；Phase 01 驗收環境如何搭建尚未定案。
9. 資料保存、跨境傳輸目的地、隱私權政策版本管理等法規細節，須以法律意見為準，尚未取得。

---

*本文件為 Phase 00 草案，未批准；批准權在統籌與使用者。不得宣稱產品完成。*
