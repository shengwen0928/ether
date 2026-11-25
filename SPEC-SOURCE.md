> **品牌更名追溯註記（2026-11-25 00:42 使用者品牌企劃書）：** NEXUS 為本規格下達時的暫定品牌名，現已正式改為 ETHER（Every Thread Holds Endless Remembrance）；專案目錄同步改為 `~/workspace/ether/`。本文件內文保留原始摘要寫法，內文中的「NEXUS」指原始規格文件及其暫定名，僅供追溯；除本註記外，一律以 ETHER 為準。§26 專案根目錄名已更新為 `ether/`。
>
> ---

# NEXUS 規格來源摘要（SPEC-SOURCE.md）

> 本文件為使用者 2026-11-25 00:29 下達的 NEXUS 完整規格（28 節＋Phase 00–12 執行手冊）
> 的忠實摘要，供 Phase 00 文件工人使用。凡本摘要與使用者原文有出入，以使用者原文為準；
> 使用者原文未存檔，摘要中「必須」「不得」等規範性文字皆轉述原文，不得弱化。
> 狀態：草案／待批准。不得編造額度、數字、效能指標；成本一律寫「待查證」。

## 架構調整（2026-11-25 00:33 使用者指令，優先於下文舊技術棧）

全面替換付費服務，改採免費及開源方案。舊服務名只准出現在比較文件
（COST_OPTIMIZATION.md），不得殘留在架構正文。功能不得刪、不得降級。

強制替換對照（使用者指定）：AWS ECS Fargate→Docker Compose＋Coolify；AWS RDS／Aurora→
PostgreSQL（自架）；ElastiCache→Valkey；S3→MinIO；OpenSearch→Meilisearch；Secrets
Manager→OpenBao；CloudWatch→Prometheus＋Grafana＋Loki；CloudFront→Cloudflare Free；
WAF→Coraza WAF；Route 53→Cloudflare DNS；Certificate Manager→Let's Encrypt；Cognito→
Keycloak；API Gateway→Traefik；SQS／SNS→NATS JetStream；CodePipeline→GitHub Actions；
Terraform→OpenTofu；LiveKit Cloud→LiveKit Community Edition 自架；雲端 AI API→
Ollama＋vLLM＋AI Provider Adapter；向量資料庫付費服務→pgvector（已定案，見 ADR-017）；Sentry 託管→
GlitchTip 自架。

新增規則：身分驗證改 Keycloak（保留 Email 驗證、登入登出、OAuth、MFA、Session、
Access／Refresh Token、裝置管理、遠端登出、密碼重設；不得破壞使用者／好友／裝置關聯）；
AI 採 Ollama＋vLLM＋LocalAI 優先＋外部 AI API Adapter 可選擴充（可切換模型、Streaming、
Tool Calling、上下文、Token 統計、資源管理；供應商不得寫死；評估 CPU／RAM／GPU／VRAM，
不得假設一般伺服器跑得動所有模型）；通話採 LiveKit CE 自架（保留一對一／多人、螢幕分享、
房間與參與者、STUN／TURN、SFU、品質監控；特別評估 TURN 頻寬與公網部署需求）；
監控採 Prometheus＋Grafana＋Loki＋OpenTelemetry＋GlitchTip（覆蓋 API、WebSocket、資料庫、
Worker、Media、AI）。Docker Compose 啟動／連線／上傳／訊息／AI 切換／LiveKit／監控皆為
「待 CI 或有 Docker 的環境驗證」，此 VM 無 docker，不得寫已驗證。仍可能付費項目
（網域、伺服器、簡訊 OTP、Apple Developer、Google Play 上架、外部 AI API、TURN 頻寬等）
須明列，不得宣稱全系統零成本。
追溯說明：本摘要下文保留使用者原文的服務名寫法以供追溯，一律以本節替換對照為準；
舊服務名除 COST_OPTIMIZATION.md 外，亦可出現在 CLOUD_ARCHITECTURE.md 的比較段落
（須標明已取代），其餘架構正文不得殘留。

## 專案定位

專案名稱：NEXUS — 全功能跨平台即時通訊生態系統。LINE 等級功能完整度，真正可註冊、
登入、加好友、聊天、建群組、傳檔案、語音／視訊通話、收通知、管理社群的完整產品，
不是 Demo、不是只有前端畫面。成熟商業級 App 體驗，以上架 Google Play 與 Apple App Store
的產品標準開發。不得用假資料冒充真實功能、不得有無作用按鈕、不得以靜態畫面代替後端。
所有功能須有明確資料模型、API、狀態管理、錯誤處理。品牌原創，不得用 LINE 的 Logo、
角色、貼圖或受保護素材。品牌概念：連結、溝通、交流、社群、數位生活。

核心目標 10 項：Android＋iOS App；可擴充即時通訊後端；可持續運作的即時訊息傳輸架構；
使用者／好友／群組／社群系統；多媒體與檔案傳輸；語音／視訊通話；官方帳號與機器人；
通知推播；Web 管理後台；安全性／隱私／權限管理。

## 技術棧（已確定，不得擅自更換）

- 行動／桌面：Flutter＋Dart；Material 3 深度客製；Riverpod；GoRouter；Dio；WebSocket；
  Drift／SQLite；flutter_secure_storage；WebRTC；FCM（Android）；APNs（iOS）。
  支援 Android 手機、iPhone、不同螢幕尺寸、平板、深淺色模式、動態字體、無障礙、
  多語系、高 DPI、低網速與離線。
- Web／Admin：React＋TypeScript。
- 後端：Go；Echo v5（≥v5.4.7，已定案，見 ADR-016）；PostgreSQL；Redis；WebSocket；NATS JetStream；gRPC 內部通訊；
  S3 相容儲存；OpenSearch；Docker；Kubernetes（後續大型部署選項）。
- 通話：WebRTC＋LiveKit；STUN／TURN；SFU。
- AI：Provider Adapter（不得讓商業邏輯直接依賴特定供應商 SDK）。
- 雲：AWS 為參考部署平台。基礎設施：Docker＋Terraform；CI／CD：GitHub Actions。
- 平台：Android、iOS、Windows、macOS、Web 五平台（手冊 A 節）。

## 22 個後端模組（責任邊界明確，初期可模組化單體，不得互相依賴混亂）

1 Authentication 2 User 3 Profile 4 Friend 5 Contact 6 Chat 7 Message 8 Group
9 Community 10 Media 11 Sticker 12 Call 13 Notification 14 Search 15 Official Account
16 Bot 17 Timeline 18 Payment Integration 19 Moderation 20 Administration 21 Analytics
22 Device Management。所有重要資料操作必須經過後端驗證；重要商業邏輯不得只寫在前端；
管理員權限不得由前端自行判斷。

## 視覺設計規範（§03）

定位：精緻、現代、高級、友善、清晰、有品牌識別度。禁：廉價模板、大量無意義漸層、
過度圓角卡片、制式 AI Dashboard 風、過度空洞留白。須建立 App Icon、Logo、啟動畫面、
品牌色彩、字體、圖示、插畫、貼圖風格、動效規範。
Design Token：Primary、Secondary、Background、Surface、Surface Elevated、Text Primary、
Text Secondary、Border、Success、Warning、Error、Info；Light／Dark 兩套。
元件庫（每個元件須有 Default／Hover／Pressed／Disabled／Loading／Error／Selected 狀態，
一致動畫與觸覺回饋）：Primary／Secondary／Icon Button、FAB、Input、Search、Avatar、
Avatar Group、Badge、Status Indicator、Message Bubble、Chat Composer、Bottom Navigation、
App Bar、Context Menu、Bottom Sheet、Dialog、Toast、Snackbar、Tab Bar、Segmented Control、
Skeleton、Empty／Error State、Pull To Refresh、Media Preview、Attachment Picker、
User／Contact／Group／Official Account Card。

## 導覽架構（§04）：底部五入口

聊天、聯絡人、探索、動態、我的。
- 聊天首頁：最近聊天、群組／一對一／官方帳號、未讀數、預覽、時間、傳送失敗、置頂、
  封存、靜音、分類、搜尋、編輯、建立新聊天；左滑、長按、下拉刷新、未讀篩選、排序、
  批次管理；即時更新，不得要求手動刷新。
- 聯絡人：好友列表、新好友通知、邀請、推薦、群組、官方帳號、分類、搜尋、新增入口、
  生日提醒（須授權）；支援姓名／帳號／電話／公開識別搜尋。
- 探索：官方帳號、品牌服務、熱門、生活資訊、活動、精選、分類、機器人、迷你應用、
  網頁入口；推薦內容須有後端管理機制。
- 動態：好友貼文（圖／影／文／多圖）、按讚、留言、分享、收藏、隱藏、檢舉、隱私、
  編輯、刪除。
- 我的：頭像、名稱、ID、狀態訊息、QR Code、檔案、好友設定、貼圖／主題管理、付款入口、
  設定、隱私、安全性、裝置管理、幫助中心。

## 帳號系統（§05）

註冊：手機號碼、Email、驗證碼、密碼、OAuth、Google、Apple、條款＋隱私同意。
登入：手機／Email／密碼／驗證碼、Google、Apple、生物辨識、裝置信任、登入紀錄。
安全：MFA、異常通知、改密、重設、裝置管理、遠端登出、Session 管理、Refresh Token
Rotation、停用、刪除、資料匯出。
個資：名稱、唯一 ID、頭像、背景、簡介、狀態訊息、生日（可選公開範圍）、電話（私密）、
Email（私密）、QR Code、分享連結、狀態；可見範圍控制。

## 好友系統（§06）

新增：ID／電話搜尋、QR 掃描、連結邀請、聯絡人同步（須明確授權、限制範圍）、推薦、
群組內新增、邀請連結、邀請通知。邀請：發送／接受／拒絕／取消／過期／紀錄／通知。
管理：刪除、封鎖、解除封鎖、隱藏、備註、分類、搜尋、排序、個人檔案。
封鎖／刪除／隱藏是不同狀態，不得混為一談。

## 一對一聊天（§07，核心）

文字：即時、多行、表情、Unicode、多語言、連結辨識、網頁預覽、時間、傳送狀態、
已送達、已讀、收回、刪除、編輯、引用、轉傳、複製、收藏、搜尋、置頂、回覆。
訊息狀態 7 種：Pending、Sending、Sent、Delivered、Read、Failed、Recalled。
訊息須有唯一 ID、時間戳記、發送者、接收對象、內容、狀態。
多媒體：單／多張照片、原畫質、影片、GIF、動態影像、語音、音訊、PDF／Word／Excel／
PowerPoint／ZIP／其他允許格式；上傳／下載進度、取消、重送、重試、縮圖、預覽、
大小／類型、下載、本機快取；檔案須經權限驗證、格式檢查、惡意檔案掃描。
語音訊息：按住錄音、計時、取消、預覽、播放、速度、暫停／繼續、進度、耳機／喇叭、
背景播放、波形。
互動：回覆、引用、表情反應、轉傳、多選、批次刪除／收藏、置頂、複製、搜尋、分享。
聊天室設定：背景、主題、通知、靜音、封存、置頂、搜尋、資料、媒體總覽、連結總覽、
對方檔案、封鎖、檢舉、刪除紀錄。
跨裝置同步：多裝置登入、列表／訊息／已讀／撤回／設定／草稿同步（可設定）、離線同步；
訊息唯一識別碼、版本控制、冪等，避免重複。

## 群組（§08）

建立：名稱、頭像、背景、介紹、邀請連結、QR、邀好友、邀請審核、公告。
角色：Owner、Admin、Member；權限：新增／移除成員、改名、換頭像、公告、邀請管理、
成員管理、權限管理、轉移所有權、解散。
聊天：文字／圖／影／檔案／語音、反應、回覆、轉傳、收回、搜尋、已讀資訊、公告、
@成員、@所有人、置頂、投票、活動通知。
管理：成員列表、管理紀錄、加入／離開紀錄、權限、邀請設定、隱私、封存、離開、檢舉。

## 社群（§09）

建立、名稱、介紹、封面、分類、公開／私密、搜尋、邀請、審核、成員管理、管理員、
公告、規則、多聊天室、子群組、討論區、活動、投票、檢舉、黑名單、資料分析；
分級權限與操作紀錄。

## 通話（§10，WebRTC＋LiveKit，SFU）

一對一語音：撥打／接聽／拒接／忙線／未接／通話中／靜音／揚聲器／藍牙／時間／結束／
斷線重連。視訊：前後鏡頭切換、麥克風／攝影機開關、畫面切換、全螢幕、網路品質、
凍結偵測、連線恢復。多人：語音／視訊、參與者列表／邀請、管理員、靜音管理、移除、
主持人、邀請、結束、紀錄。通話紀錄：撥出／接聽／未接／拒接、時間、對象、日期。
平台：Android 背景來電、iOS CallKit、Windows／macOS 原生通知、Web 權限處理。

## 貼圖與主題（§11）

貼圖：靜態／動態／表情／大型、貼圖包、預覽、收藏、最近使用、分類、搜尋、下載、
管理、分享。商店：免費／付費／限時、推薦、排行榜、創作者頁、詳情、購買紀錄、
已購買。主題：官方／免費／付費／深／淺／自訂、預覽、套用、還原預設。

## 動態（§12）

發布：文字、單／多圖、影片、GIF、連結、位置、多媒體混合。互動：按讚、留言、回覆、
分享、收藏、隱藏、檢舉、刪除、編輯、置頂。隱私：公開／好友／指定好友／僅自己、
封鎖排除、留言／互動權限。管理：貼文／留言管理、按讚列表、分享紀錄、通知、排序、推薦。

## 官方帳號與 Bot（§13）

官方帳號：註冊、名稱、頭像、認證標章、介紹、網站、聯絡方式、搜尋、加入、封鎖、檢舉。
客服：一對一客服、自動回覆、關鍵字回覆、歡迎／離線訊息、FAQ、圖文訊息、選單、
快速回覆、客服分配、標籤、備註。
Bot API：建立、Token、Webhook、事件訂閱、收／發訊息、群組／使用者事件、自動化流程、
權限、用量、停用；須提供開發者文件與測試環境。

## 通知（§14）

類型：新訊息、好友邀請／接受、群組邀請／公告、社群、按讚、留言、回覆、官方訊息、
語音／視訊來電、系統、安全警告。支援 Android／iOS 推播、App 內通知、分類、已讀、
清除、設定、靜音時段、個別聊天室控制。推播內容須尊重隱私設定，關閉預覽時不得洩漏內容。

## 搜尋（§15）

範圍：好友、使用者 ID、群組、社群、官方帳號、聊天室、訊息、檔案、照片、影片、
貼圖、動態貼文。訊息搜尋：關鍵字、日期區間、發送者、媒體／檔案類型、群組、
一對一聊天室。結果須遵守存取權限與封鎖規則。

## 生活服務（§16，預留入口）

付款／錢包、優惠券、電子票券、活動報名、商家服務、會員卡、迷你應用、外部網站、
服務通知、地圖、位置分享、即時位置分享、行事曆整合。付款須經合法合規第三方，
不得自行儲存信用卡完整資訊。

## 隱私與安全（§17，核心，事後補充不算）

資料保護：HTTPS／TLS、敏感資料加密、安全憑證儲存、API 權限驗證、Token 管理、
輸入驗證、Rate Limiting、防暴力破解、防帳號接管、SQL Injection／XSS／CSRF 防護、
檔案上傳安全檢查、稽核紀錄。
E2EE：每裝置獨立金鑰、身分金鑰、一次性金鑰、訊息金鑰輪替、前向保密、裝置撤銷、
金鑰備份、新裝置驗證、安全金鑰驗證。優先採用公開審查過的成熟協定（如 Signal
Protocol），不得自發明加密演算法。E2EE 訊息伺服器不得明文儲存或讀取。
須明確區分 E2EE 聊天室與一般聊天室的功能差異。
隱私控制：誰可加好友／搜尋／看檔案／看動態／傳訊息／加群組、是否公開上線／已讀／
生日、是否允許聯絡人同步。
檢舉封鎖：使用者／訊息／群組／社群檢舉、垃圾訊息偵測、惡意連結警告、帳號限制、
暫時／永久停權、申訴流程。

## 設定中心（§18）

帳號（檔案、資訊、電話、Email、密碼、登入裝置、刪除）、隱私（搜尋、訊息接收、
可見性、動態、已讀、上線、封鎖名單）、通知（訊息／群組／好友／社群／動態／來電、
音效、震動、預覽）、聊天（背景、字體、備份、匯出、媒體自動下載、儲存、快取）、
外觀（淺／深／系統、字體、主題、動畫、泡泡樣式）、其他（語言、地區、幫助、條款、
隱私政策、版本、開發者資訊）。

## Web 管理後台（§19，React＋TypeScript，獨立管理員認證與權限）

儀表板：總使用者、今日新增、活躍、DAU／MAU、即時在線、訊息量、群組／社群／官方帳號數、
媒體儲存用量、通話用量、系統健康。
使用者管理：搜尋、檢視、狀態、登入／裝置紀錄、停權／解封、檢舉、風險標記；
管理員不得任意讀取 E2EE 訊息。
群組社群管理：搜尋、成員、狀態、檢舉審核、違規處理、操作紀錄。
貼圖內容管理：上／下架、分類、審核、官方／推薦內容、動態檢舉。
系統管理：API／WebSocket／Redis／PostgreSQL／佇列／儲存監控、錯誤日誌、警報、
權限管理、管理員操作稽核。

## 資料庫（§20，PostgreSQL，表名為準，不得改名）

使用者：users、user_profiles、user_settings、user_devices、user_sessions、user_security_keys。
好友：friendships、friend_requests、blocked_users、hidden_users、contact_sync_records。
聊天室：conversations、conversation_members、conversation_settings、messages、
message_receipts、message_reactions、message_attachments、message_replies、
message_forwards、message_pins。
群組：groups、group_members、group_roles、group_invitations、group_announcements、
group_settings。
社群：communities、community_members、community_channels、community_roles、
community_invitations。
媒體：media_files、media_uploads、media_thumbnails、media_access_logs。
貼圖：sticker_packs、stickers、user_stickers、sticker_purchases。
動態：posts、post_media、post_likes、post_comments、post_shares、post_bookmarks。
通話：calls、call_participants、call_events。
通知：notifications、notification_preferences、push_tokens、notification_deliveries。
官方帳號：official_accounts、official_account_members、bot_applications、bot_webhooks、
bot_events。
管理：admin_users、admin_roles、admin_permissions、admin_audit_logs、reports、
moderation_actions。
所有表須有 PK、FK、Unique Constraint、必要索引、建立／更新時間、合理刪除策略。
訊息／事件／狀態資料須考慮高併發、分頁、索引、保留政策。須提供完整 Migration、Seed、
資料庫關聯圖。

## API（§21，版本化 REST＋OpenAPI／Swagger）

前綴（為準）：/api/v1/auth、/users、/profiles、/friends、/contacts、/conversations、
/messages、/groups、/communities、/media、/stickers、/posts、/calls、/notifications、
/search、/official-accounts、/bots、/admin。
每個 API 須有 Request／Response Schema、Authentication、Authorization、Validation、
Error Handling、Pagination、Rate Limit、Logging。
WebSocket 事件（事件名為準，須有事件 ID、時間戳記、版本、關聯識別碼）：
message.send、message.received、message.delivered、message.read、message.recalled、
typing.start、typing.stop、presence.online、presence.offline、conversation.updated、
group.member.joined、group.member.left、call.incoming、call.accepted、call.ended。

## 即時通訊品質（§22）

須處理：斷網、WS 自動重連、重複、遺失、亂序、離線訊息、多裝置同步、送達確認、
已讀同步、伺服器重啟、快速切網、App 背景、App 被系統終止、推播與即時訊息重複、
大量同時上線。採用：Message ID、Idempotency Key、Sequence Number、ACK、Retry、
Exponential Backoff、Message Persistence、Offline Queue、Delivery Receipt、
Reconnection Recovery。不得將 WS 連線本身當成可靠送達保證。

## 效能與擴充（§23）

目標：大量同時在線、高頻訊息、大型群組、大量媒體上傳、高併發搜尋、多區域部署、
水平擴充。措施：Connection Pool、Redis Cache、Message Queue、Index、Cursor Pagination、
CDN、Object Storage、Background Jobs、Horizontal Scaling、Observability。
不得在無壓力測試下宣稱可承載特定同時在線數。

## 測試（§24）

Unit（核心商業邏輯）；Integration（API／DB／Redis／WS／訊息／媒體／推播）；
E2E（註冊→登入→加好友→傳訊→收訊→傳照片→建群組→邀成員→群組訊息→通話→推播→登出，
共 12 步）；Security（權限繞過、Token 竊取、SQL Injection、XSS、Rate Limit、
非法檔案上傳、未授權讀取、帳號接管）；Performance（延遲、WS 連線數、吞吐量、
API 回應、DB 負載、媒體上傳）。

## 階段（§25，逐步實作，每階段測試驗收，不得一次生成全部）

Phase 1 基礎：Flutter、Go 後端、PostgreSQL、Redis、Docker、Migration、API 架構、
Design System、路由／狀態管理。驗收：App 啟動、前後端連線、DB 操作。
Phase 2 帳號：註冊／登入／驗證碼／個資／Token／裝置／重設。驗收：完整註冊登入登出改個資。
Phase 3 好友：新增／邀請／列表／QR／搜尋／封鎖／刪除。驗收：兩帳號互加好友。
Phase 4 即時聊天：一對一、WS、文字、狀態、已讀、收回、回覆、轉傳、搜尋、離線。
驗收：兩台不同裝置真正即時交換訊息。
Phase 5 多媒體：圖／影／檔案／語音、預覽、進度、快取。驗收：兩裝置互傳真實檔案。
Phase 6 群組社群：建群組、成員管理、群組訊息、公告、社群、多聊天室、投票。
驗收：多名使用者同時群組交流。
Phase 7 語音視訊：WebRTC、STUN／TURN、SFU、語音／視訊／多人。驗收：不同網路實際通話。
Phase 8 貼圖動態：貼圖系統／商店、動態牆、留言、按讚、分享、收藏。
Phase 9 官方帳號與 Bot：官方帳號、客服、自動回覆、Bot API、Webhook、開發者文件。
Phase 10 管理後台：儀表板、使用者／社群／貼圖管理、檢舉審核、系統監控。
Phase 11 安全與正式部署：E2EE、安全測試、效能測試、推播、CI／CD、備份、監控、正式部署。

## 目錄規範（§26，Monorepo）

ether/：apps/mobile（lib/core、common、features/auth、chat、contacts、friends、groups、
communities、calls、timeline、stickers、notifications、settings、navigation、design_system、
main.dart；test/）、apps/admin（src、package.json）、apps/web、services/api（cmd、internal/
auth、user、friend、chat、message、group、community、media、call、notification、search、
moderation、migrations、api、tests）、infrastructure（docker、monitoring、deployment、
scripts）、docs（architecture、database、api、security、deployment）、docker-compose.yml、
README.md、.env.example。

## Agent 執行規則（§27）

1 先檢查工作目錄與既有專案，不得未確認覆蓋既有檔案。2 先架構規劃再實作。
3 不得只輸出程式碼範例而不建實際檔案。4 不得用假 API 冒充後端。5 不得用固定假資料
冒充即時訊息。6 不得有無事件處理的裝飾按鈕。7 不得全塞單一巨大檔案。8 不得忽略錯誤
處理。9 不得忽略 Loading／Empty／Error。10 不得忽略手機尺寸適配。11 不得將敏感金鑰寫入
原始碼。12 不得將管理員權限交前端判斷。13 不得跳過 Migration。14 不得跳過測試。
15 不得將未完成功能標為完成。每功能須有 UI、狀態管理、API、資料庫、權限驗證、
錯誤處理、測試、文件。每階段須提供：完成功能、新增／修改檔案、DB 變更、API 變更、
測試結果、未完成項、已知問題、下一階段。外部服務／金鑰／憑證／第三方帳號需求須明列，
不得假裝完成。

## 最終交付（§28）

Flutter App、Android 安裝包、iOS 可建置專案、Go 後端、PostgreSQL Schema、Migration、
Redis 設定、WS 服務、WebRTC 整合、媒體儲存、推播、管理後台、OpenAPI 文件、Docker 部署、
環境範例、自動化測試、架構文件、資安文件、部署手冊、維運手冊。最終驗收以真實裝置、
真實帳號、真實後端、真實資料；不允許單機假聊天、Mock API、靜態 UI、假按鈕宣稱完成。

## 執行手冊 A（全域指令，每階段遵守）

讀根目錄 README、ARCHITECTURE、DECISIONS、ROADMAP 及當前任務文件；不得未檢查現有
專案就重建或覆蓋；不得擅自換技術棧；商業邏輯不得只放前端；不得 Mock 冒充；不得無事件
按鈕；功能須實際測試才標完成；不得略過 Migration／API Contract／測試；不得硬編碼密碼／
Token／Key；不得忽略錯誤處理與權限驗證；不得刪未完成需求；新增功能須符合既有架構；
模組責任邊界清晰；優先安全／一致性／可維護性；所有操作須有 Loading／Success／Empty／
Error 狀態；響應式佈局；外部服務須有 Adapter；不得自發明密碼學協定；未取得的第三方憑證
不得當已完成整合；不得以編譯通過代替驗收。
工作流程每階段：讀前置文件→分析程式碼→列計畫→任務清單→按模組實作→靜態分析→單元測試
→整合測試→更新文件→階段驗收。不確定技術決策先提 ADR。缺少憑證／服務／資訊時先做可獨立
部分並標記阻塞，不得假裝成功。

## 手冊 B（Phase 00）— 本階段任務

先不寫大量應用程式碼。建立完整產品架構與工程規格：PRD、功能清單、五平台功能矩陣、
系統架構圖、模組依賴圖、資料流圖、身分與權限模型、ERD、API 命名規範、WS Event
Contract、錯誤碼規範、安全性架構、E2EE 研究與風險文件、AI 資料處理規範、雲端部署架構、
測試策略、版本與發布策略、完整 Roadmap。產出：docs/PRD.md、docs/architecture/
SYSTEM_ARCHITECTURE.md、MODULE_BOUNDARIES.md、DATA_FLOW.md、SECURITY_ARCHITECTURE.md、
SCALABILITY.md、docs/database/ERD.md、docs/api/API_STANDARDS.md、WEBSOCKET_PROTOCOL.md、
docs/testing/TEST_STRATEGY.md、docs/deployment/CLOUD_ARCHITECTURE.md、docs/ROADMAP.md、
docs/decisions/。驗收：功能定義明確、模組責任清楚、平台適配策略、資料流與權限無明顯
缺口、決策有文件、順序與依賴明確。完成後停止，交架構審查報告，不得宣稱產品完成。

## 手冊 C–N 摘要（後續階段，Phase 00 文件須為其留介面，不得超前實作）

- C Phase 01 基礎建設：Monorepo、Flutter Mobile＋Desktop、React Web＋Admin、Go Backend、
  Realtime Gateway、Worker、PostgreSQL、Redis、NATS、Docker Compose、Terraform AWS、
  GitHub Actions、環境變數／Secrets 管理、Health Check、Logging、OpenTelemetry、CI。
  區分 local／development／staging／production；Production Secrets 不入 Git；容器 Health
  Check；Migration；Makefile／Task Runner；版本固定；格式化／Lint；Commit Convention；
  PR 檢查；部署文件。驗收：全部專案可建置、Docker 啟動、API 健康檢查、App／Web／Admin
  啟動、PG／Redis／NATS 連線、CI 自動檢查、無硬編碼憑證。
- D Phase 02 設計系統與品牌：Logo、Icon、色彩、字體、圖形、插畫、啟動畫面、Loading／
  Empty／Error 插畫；Design Tokens（Color／Typography／Spacing／Radius／Border／Shadow／
  Elevation／Motion／Icon／Z-Index，Light／Dark）；Flutter 元件庫；React 元件庫（視覺一致）；
  平台差異化（Mobile 底導航／手勢／附件面板；Desktop 多欄／側欄／快捷鍵／拖放／托盤；
  Web 響應式／PWA／鍵盤；Admin 表格／篩選／圖表／審核）。驗收：共用元件實際可用、
  風格一致、四尺寸版型驗證。
- E Phase 03 身分帳號：註冊、Email 驗證、手機 OTP、密碼、Google、Apple、MFA、重設、
  Token Refresh、Session、登入紀錄、裝置管理、遠端登出、停用、刪除、檔案、ID、頭像、
  狀態、QR。安全：OAuth 2.0／OIDC、PKCE、短效 Access Token、Refresh Rotation、安全
  Cookie、Secure Storage、Session Revocation、Rate Limit、防濫用、裝置信任。驗收：兩帳號
  多裝置完成註冊登入登出重登遠端登出，資料真正寫入 DB。
- F Phase 04 好友聯絡人：ID／電話／Email 搜尋、QR、邀請流程、列表、備註、分類、隱藏、
  封鎖、刪除、推薦、聯絡人同步、邀請連結。聯絡人同步須授權；拒絕通訊錄權限不得阻擋
  其他功能；四種狀態分開建模。驗收：三帳號互搜互邀接受封鎖解封刪除。
- G Phase 05 即時訊息核心：WS Gateway（認證、Registry、Heartbeat、重連、多裝置 Session、
  路由、恢復、Rate Limit、Backpressure、Graceful Shutdown）；訊息（文字、回覆、引用、編輯、
  收回、刪除、轉傳、置頂、反應、已讀／送達回條、輸入中、上線狀態、離線投遞）；
  可靠性（唯一 ID、Client ID、冪等、每對話 Sequence、Transactional Outbox、ACK、Retry、
  去重、離線佇列、Cursor 分頁、歷史同步）；聊天 UI（列表、聊天室、泡泡、Composer、
  輸入中、已讀、未讀計數、回覆預覽、選單、搜尋、日期分隔、捲動位置恢復）。驗收：兩帳號
  至少兩台實際裝置，測即時／離線／重連／去重／順序／已讀／多裝置／重啟恢復；
  不許單機假資料測試。
- H Phase 06 媒體群組社群：媒體（上傳、壓縮、縮圖、轉碼、音訊、錄音、分段／續傳、
  下載、預覽、快取、CDN、安全掃描）；群組（建立、名稱、頭像、邀請、連結、成員、角色、
  權限、公告、置頂、@、投票、已讀、離開、解散、管理紀錄）；社群（公開／私密、搜尋、
  申請、審核、管理員、子頻道、公告、活動、投票、檢舉）。驗收：三帳號以上建群組傳媒體
  管成員社群互動；媒體用真實物件儲存並驗權限。
- I Phase 07 語音視訊會議：WebRTC＋LiveKit、STUN／TURN、SFU、信令、權限、Audio Session、
  Push Call Notification；一對一語音／視訊、多人、來電、接／拒／忙線、靜音、鏡頭切換、
  螢幕分享、參與者管理、主持人、紀錄、品質監測、重連。平台：Android 背景來電、iOS
  CallKit、Win／mac 原生通知、Web 權限。驗收：不同網路不同平台實際通話，不得模擬畫面。
- J Phase 08 社群貼圖商業：Timeline（發文編輯刪除、圖影、留言回覆、讚、分享、收藏、
  隱私、檢舉）；貼圖（包、靜／動態、收藏、最近、搜尋、分類、免費／付費、購買紀錄、
  創作者管理）；官方帳號（認證、客服、多人客服、自動回覆、關鍵字、圖文、快速回覆、
  圖文選單、標籤、備註）；Bot（申請、Token、Webhook、訂閱、Messaging API、Rich Message、
  Interactive Card、權限、用量、開發者 Console）；商業（優惠券、票券、商家頁、會員、
  付款供應商整合、交易紀錄、訂單通知；付款須合規）。
- K Phase 09 NEXUS AI：AI 對話（Streaming、模型切換、上下文、歷史、用量、設定、
  Tool Calling）；訊息 AI（摘要、群組摘要、翻譯、智慧回覆、潤飾、語氣、語音轉文字、
  文件摘要、圖片理解）；知識 AI（個人知識庫、文件索引、向量搜尋、RAG、語意搜尋、
  權限隔離、刪除同步）；商業 AI（官方 AI 客服、FAQ、Bot、分類、摘要、回覆建議）。
  安全：授權、Provider Adapter、Rate Limit、Token Budget、敏感遮罩、Prompt Injection 防護、
  Tool 權限、使用紀錄、保留策略。E2EE 明文不得自動送往雲端 AI。驗收：每種 AI 功能用真實
  模型供應商或可驗證服務；無 Key 時只做介面與 Adapter 並標記未完成。
- L Phase 10 Web／桌面／多裝置：Web（完整聊天、好友、群組、多媒體、通知、PWA、Service
  Worker、Push、Storage、響應式）；Desktop（多欄、系統通知、啟動設定、拖放、快捷鍵、
  托盤、視窗管理、多視窗、下載、系統分享）；跨裝置（註冊、驗證、多登入、同步、已讀同步、
  推播 Token、撤銷、金鑰同步）。驗收：同一帳號五平台登入驗同步；各平台用合理介面，
  不可複製手機畫面。
- M Phase 11 安全效能發布：安全（OWASP ASVS／MASVS、OAuth、API 權限、Session、SQLi、
  XSS、CSRF、Rate Limit、檔案上傳、E2EE Protocol Audit、Secrets Audit、Dependency Scan、
  SAST、DAST）；效能（API／WS／DB 負載、吞吐量、上傳、通話品質、Memory／CPU Profiling）；
  可靠性（備份、還原測試、災復、Retry、Circuit Breaker、DLQ、Health Check、Alerting、
  Incident Logging）；發布（Android Release、iOS Archive、Win Installer、macOS Package、
  Web／Admin Production Build、Production Env、Release Notes、部署指南、回滾流程）。
  驗收：僅通過者可標完成；須發布審查報告列未解決風險。
- N Phase 12 最終稽核：逐項比對 PRD、功能矩陣、API、DB、前端、實測；檢查 15 項
  （未實作、UI 無 API、API 無 UI、Mock、無作用按鈕、權限漏洞、同步錯誤、未處理錯誤、
  平台落差、不可復原操作、不合理查詢、未授權 AI 傳輸、不安全金鑰、未測外部整合、
  無法部署）。產出 docs/audit/FINAL_AUDIT.md、FEATURE_COMPLETENESS.md、
  SECURITY_AUDIT.md、PLATFORM_MATRIX.md、KNOWN_ISSUES.md、RELEASE_READINESS.md。
  不得因階段執行完就判定完成，完成度以實際驗收為準。

## 既有成果對照（Phase 00 須標註差異，不得覆蓋刪除）

- ~/workspace/messenger-app/PLAN.md：Kotlin＋Supabase 路線、Android 先行、核心版三階段。
  已被 NEXUS 規格取代（Flutter＋Go＋PG／Redis／NATS／S3／OpenSearch、五平台、22 模組）。
- MASTER_SPEC.md（1,408 行）：14 面向功能總規格草案（訊息／通話／好友／群組社群／
  貼圖主題／動態／官方 Bot／通知設定／隱私資安／多裝置／無障礙／效能／部署／法規），
  功能分析可重用；其 Supabase 表名（profiles、friendships、conversations、messages、
  attachments 等）與 NEXUS §20 表名（users、user_profiles、conversations、messages、
  message_attachments 等）不同，以 NEXUS §20 為準，差異寫入 ADR。
- CONTRACTS.md（v1 凍結＋v2 增補草案）：Supabase Realtime 頻道（chat:／presence:／
  call:／sync:／account:）、30 秒逾時、append-only 等約束，概念可參考；傳輸層改為
  NEXUS WebSocket 事件協定（§21），以 NEXUS 為準，差異寫入 ADR。

## 環境實情（2026-11-25 實測）

本機 Linux VM：有 node v24.22.1、npm；無 flutter、無 go、無 docker、無 psql、
無 redis-server。iOS 建置在 Linux 上不可能，須走 macOS CI runner。文件須如實記載，
不得假裝工具鏈齊全。
