狀態：草案／待批准

# ETHER 五平台功能矩陣（PLATFORM_MATRIX）

> 依據：`SPEC-SOURCE.md` 手冊 A（五平台）、手冊 D（平台差異化設計）、手冊 L（Phase 10 Web／桌面／多裝置）、§04–§19 功能面向；功能深度補充自 `MASTER_SPEC.md`。
>
> 平台：Android、iOS、Windows、macOS、Web。行動／桌面共用 Flutter＋Dart，Web／Admin 為 React＋TypeScript；共用邏輯不等於共用介面，各平台須用合理介面，不可複製手機畫面。
>
> 標記定義：Full＝該平台完整提供該功能，含全部狀態與錯誤處理；Partial＝該平台提供該功能的核心部分，但受平台能力、權限或階段限制，備註說明限制；N-A＝該功能不適用於該平台，備註說明原因。成本一律「待查證」。本文件為 Phase 00 草案，不得宣稱產品完成。
>
> 架構調整（使用者 2026-11-25 00:33 指令）：部署與 AI 改採免費及開源自架方案——Docker Compose＋Coolify、PostgreSQL 自架、Valkey、MinIO、Meilisearch、OpenBao、Prometheus＋Grafana＋Loki、Cloudflare Free、Coraza WAF、Cloudflare DNS、Let's Encrypt、Keycloak、Traefik、NATS JetStream、GitHub Actions、OpenTofu、LiveKit Community Edition 自架、Ollama＋vLLM＋LocalAI 優先＋外部 AI API Adapter 可選擴充、pgvector、GlitchTip 自架。功能一項不得刪、不得降級。Docker Compose 啟動／連線／上傳／訊息／AI 切換／LiveKit／監控皆為「待 CI 或有 Docker 的環境驗證」，此 VM 無 docker，不得寫已驗證。仍可能付費項目見 `docs/COST_OPTIMIZATION.md`，不得宣稱零成本。

## 1. 平台差異化設計（手冊 D 節）

平台差異化不是同一介面的縮放，而是依平台慣例與能力設計的合理介面：

### 1.1 Mobile（Android／iOS）：底導航／手勢／附件面板

- 底部五入口導覽（聊天、聯絡人、探索、動態、我的），符合行動慣例。
- 手勢操作：左滑快捷操作、長按選單、下拉刷新、滑動返回（依平台慣例）。
- 附件面板：拍照、相簿、檔案、位置、語音等附件入口以底部面板呈現，適合單手操作。
- 來電：Android 背景來電、iOS CallKit 全螢幕來電介面，須符合各平台來電規範。
- 推播：Android 用 FCM，iOS 用 APNs；推播內容尊重預覽隱私設定。
- 生物辨識：依平台生物辨識能力（指紋／臉部）提供快速登入，生物特徵不得上傳伺服器。
- 權限：相機、麥克風、相簿、聯絡人、通知、位置等權限，皆須在首次使用時以系統對話框請求，拒絕後有引導說明，不得重複彈出無效請求。

### 1.2 Desktop（Windows／macOS）：多欄／側欄／快捷鍵／拖放／托盤

- 多欄佈局：左側導覽／對話列表、中間對話內容、右側詳情或成員資訊（依視窗寬度調整），不得複製手機單欄畫面。
- 側欄：聯絡人、群組、社群、設定以側欄或分頁呈現，支援快速切換。
- 快捷鍵：常用操作（搜尋、新聊天、靜音、切換對話）須有鍵盤快捷鍵，快捷鍵清單可在設定中查看。
- 拖放：可將檔案拖放至聊天室傳送；可將圖片拖放至貼圖或頭像上傳入口。
- 托盤：支援系統托盤常駐、未讀徽章、來電與訊息的系統通知；啟動設定（開機啟動、最小化至托盤）可在設定中管理。
- 視窗管理：支援多視窗、視窗大小記憶、系統分享（從其他應用分享至 ETHER）。
- 下載：檔案下載至系統下載資料夾或使用者指定位置，進度在介面可見。
- 原生通知：Windows／macOS 原生通知，來電以原生通知呈現，不得以網頁式彈窗代替。

### 1.3 Web：響應式／PWA／鍵盤

- 響應式佈局：依視窗寬度在單欄與多欄之間調整；窄視窗時以手機式單欄呈現，寬視窗時以多欄呈現。
- PWA：可安裝為 PWA，含 Service Worker 離線快取與 Web Push；離線時可查看已快取的對話與訊息，恢復連線後同步。
- 鍵盤操作：完整鍵盤導覽，焦點順序符合視覺順序；常用操作有快捷鍵。
- 權限處理：相機、麥克風、通知權限以瀏覽器權限對話框請求，拒絕後有引導說明；通話前須確認權限，不得在無權限時發起通話。
- 儲存：Web Storage 與 IndexedDB 的使用範圍受限，敏感憑證不得以明文存於 Web Storage；Token 儲存方式以安全架構文件為準。

### 1.4 Admin（Web 管理後台，獨立於一般 Web 客戶端）

- 表格／篩選／圖表／審核：以資料表格、篩選器、圖表與審核工作流程為核心介面，不得複製一般聊天介面。
- 獨立認證與權限：管理員帳號與一般使用者帳號分離，權限以後端強制。
- 審核：檢舉、內容、貼圖、官方帳號的審核流程，皆有操作紀錄與狀態追蹤。

### 1.5 共通原則

- Design Token（Color／Typography／Spacing／Radius／Border／Shadow／Elevation／Motion／Icon／Z-Index，Light／Dark 兩套）跨平台一致，Flutter 元件庫與 React 元件庫視覺一致。
- 四尺寸版型驗證：手機、平板、桌面窄視窗、桌面寬視窗皆須驗證（手冊 D 驗收）。
- 深淺色模式、動態字體、無障礙、多語系、高 DPI、低網速與離線，皆為跨平台共通要求。

## 2. 核心功能 × 五平台矩陣

### 2.1 帳號與身分（§05）

| 功能 | Android | iOS | Windows | macOS | Web |
|---|---|---|---|---|---|
| 手機號碼註冊／登入（OTP） | Full | Full | Partial：桌面版可註冊登入，但 OTP 收取須經手機或 Email，桌面端不直接收簡訊 | Partial：同 Windows | Full：可註冊登入，OTP 經手機或 Email 收取 |
| Email 註冊／登入 | Full | Full | Full | Full | Full |
| Google OAuth 登入 | Full | Full | Full | Full | Full |
| Apple OAuth 登入 | Partial：Android 上 Apple 登入依 Apple 政策與實作方式而定，備註：須以官方文件為準 | Full | Partial：依 Apple 政策與實作方式而定 | Full | Partial：依 Apple 政策與實作方式而定 |
| 密碼登入／驗證碼登入 | Full | Full | Full | Full | Full |
| 生物辨識登入 | Full：依裝置生物辨識能力 | Full：依裝置生物辨識能力 | Partial：依系統生物辨識能力（如 Windows Hello），不支援的裝置退回密碼／驗證碼 | Partial：依系統生物辨識能力（如 Touch ID），不支援的裝置退回密碼／驗證碼 | N-A：瀏覽器生物辨識以 WebAuthn 方向評估，尚未定案；本階段不承諾 |
| 身分驗證後端 | 全平台一致：Keycloak（Realm／Client／角色），保留 Email 註冊驗證、登入登出、OAuth、MFA、Session、Access／Refresh Token、裝置管理、遠端登出、密碼重設；憑證以 OpenBao 管理 | 全平台一致 | 全平台一致 | 全平台一致 | 全平台一致 |
| 裝置信任與登入紀錄 | Full | Full | Full | Full | Full |
| MFA | Full | Full | Full | Full | Full |
| 改密／重設密碼 | Full | Full | Full | Full | Full |
| 裝置管理與遠端登出 | Full | Full | Full | Full | Full |
| 停用／刪除帳號／資料匯出 | Full | Full | Full | Full | Full |
| 個人檔案編輯（名稱、ID、頭像、背景、簡介、狀態） | Full | Full | Full | Full | Full |
| QR Code 出示與掃描 | Full：出示與掃描皆可用 | Full：出示與掃描皆可用 | Partial：可出示 QR Code，掃描須經相機，桌面相機支援範圍待定；可改用連結邀請 | Partial：同 Windows | Partial：可出示 QR Code，掃描依瀏覽器相機權限而定；可改用連結邀請 |

### 2.2 好友與聯絡人（§06）

| 功能 | Android | iOS | Windows | macOS | Web |
|---|---|---|---|---|---|
| ID／電話搜尋加好友 | Full | Full | Full | Full | Full |
| QR 掃描加好友 | Full | Full | Partial：見上，桌面相機支援範圍待定 | Partial：見上 | Partial：依瀏覽器相機權限而定 |
| 連結邀請加好友 | Full | Full | Full | Full | Full |
| 聯絡人同步 | Full：須明確授權 | Full：須明確授權 | N-A：桌面版無系統通訊錄同步，以搜尋與連結邀請為主 | N-A：同 Windows | N-A：網頁版無系統通訊錄同步 |
| 好友推薦 | Full | Full | Full | Full | Full |
| 邀請生命週期（發送／接受／拒絕／取消／過期） | Full | Full | Full | Full | Full |
| 刪除／封鎖／隱藏／備註／分類 | Full | Full | Full | Full | Full |

### 2.3 一對一聊天與群組社群（§07–§09）

| 功能 | Android | iOS | Windows | macOS | Web |
|---|---|---|---|---|---|
| 文字訊息即時收發 | Full | Full | Full | Full | Full |
| 訊息狀態、已讀、收回、編輯、刪除、引用、轉傳、反應、置頂、搜尋 | Full | Full | Full | Full | Full |
| 圖片／影片／檔案訊息 | Full | Full | Full：拖放上傳為桌面加分能力 | Full：拖放上傳為桌面加分能力 | Full：拖放上傳可用 |
| 語音訊息（錄音／播放／倍速／波形） | Full | Full | Partial：可播放與倍速；錄音依桌面麥克風權限而定 | Partial：同 Windows | Partial：可播放與倍速；錄音依瀏覽器麥克風權限而定 |
| 語音／視訊通話 | Full：LiveKit Community Edition 自架 | Full：LiveKit Community Edition 自架 | Full：LiveKit Community Edition 自架 | Full：LiveKit Community Edition 自架 | Partial：依瀏覽器 WebRTC 與權限而定，背景來電能力受限 |
| 多人通話 | Full：LiveKit Community Edition 自架 | Full：LiveKit Community Edition 自架 | Full：LiveKit Community Edition 自架 | Full：LiveKit Community Edition 自架 | Partial：同上 |
| 螢幕分享 | Partial：依平台分享能力而定 | Partial：依平台分享能力而定 | Full | Full | Partial：依瀏覽器分享能力而定 |
| 群組建立／管理／公告／投票／@ | Full | Full | Full | Full | Full |
| 社群建立／多聊天室／子群組／討論區／活動 | Full | Full | Full：多欄呈現多聊天室 | Full：多欄呈現多聊天室 | Full：多欄或響應式呈現 |
| 社群檢舉／黑名單／分級權限 | Full | Full | Full | Full | Full |
| 群組相簿／記事本 | Full | Full | Full | Full | Full |
| 離線訊息與重連恢復 | Full | Full | Full | Full | Partial：PWA 離線快取範圍受 Service Worker 限制，恢復後同步 |
| 跨裝置同步（列表／訊息／已讀／撤回／設定／草稿） | Full | Full | Full | Full | Full |
| 輸入中／上線狀態 | Full | Full | Full | Full | Full |

### 2.4 貼圖、主題、動態（§11–§12）

| 功能 | Android | iOS | Windows | macOS | Web |
|---|---|---|---|---|---|
| 貼圖包瀏覽／下載／傳送 | Full | Full | Full | Full | Full |
| 動態貼圖播放 | Full | Full | Full | Full | Full |
| 自訂貼圖上傳 | Full | Full | Full：拖放上傳可用 | Full：拖放上傳可用 | Full：拖放上傳可用 |
| 貼圖商店 | Full | Full | Full | Full | Full |
| 主題／桌布／字體大小 | Full | Full | Full | Full | Full |
| 動態發布（文字／圖／影／連結／位置） | Full | Full | Full | Full | Full |
| 動態互動（按讚／留言／分享／收藏／隱藏／檢舉） | Full | Full | Full | Full | Full |
| 動態隱私範圍 | Full | Full | Full | Full | Full |

### 2.5 官方帳號與 Bot（§13）

| 功能 | Android | iOS | Windows | macOS | Web |
|---|---|---|---|---|---|
| 官方帳號搜尋／加入／封鎖／檢舉 | Full | Full | Full | Full | Full |
| 官方帳號客服對話 | Full | Full | Full | Full | Full |
| Bot 訊息收發（使用者端） | Full | Full | Full | Full | Full |
| Bot 管理（建立、Token、Webhook、自動回覆） | Partial：行動端可查看與簡易管理，完整管理以 Web／桌面為主 | Partial：同 Android | Full | Full | Full：開發者 Console 以 Web 為主 |
| 開發者文件與測試環境 | Partial：行動端可查看文件連結 | Partial：同 Android | Full | Full | Full |

### 2.6 通知、搜尋、設定（§14–§15、§18）

| 功能 | Android | iOS | Windows | macOS | Web |
|---|---|---|---|---|---|
| 推播通知（FCM／APNs／Web Push／原生通知） | Full：FCM | Full：APNs | Full：系統原生通知 | Full：系統原生通知 | Partial：Web Push 依瀏覽器支援與權限而定 |
| App 內通知中心 | Full | Full | Full | Full | Full |
| 靜音時段／個別聊天室控制／預覽隱私 | Full | Full | Full | Full | Full |
| 全域搜尋 | Full：Meilisearch 自架 | Full：Meilisearch 自架 | Full：Meilisearch 自架，快捷鍵搜尋為桌面加分能力 | Full：Meilisearch 自架，快捷鍵搜尋為桌面加分能力 | Full：Meilisearch 自架，快捷鍵搜尋可用 |
| 訊息搜尋（關鍵字／日期／發送者／類型） | Full | Full | Full | Full | Full |
| 設定中心（帳號／隱私／通知／聊天／外觀／其他） | Full | Full | Full | Full | Full |
| 語言／地區切換 | Full | Full | Full | Full | Full |

### 2.7 生活服務與 AI（§16、手冊 K）

| 功能 | Android | iOS | Windows | macOS | Web |
|---|---|---|---|---|---|
| 生活服務入口（付款／票券／商家／迷你應用） | Partial：預留入口，遠期；金流供應商待定 | Partial：同 Android | Partial：預留入口，遠期 | Partial：預留入口，遠期 | Partial：預留入口，遠期 |
| 位置分享／地圖 | Full | Full | Partial：可查看位置與地圖，傳送位置以手動選點為主 | Partial：同 Windows | Partial：可查看位置與地圖，依瀏覽器定位權限而定 |
| AI 對話（Streaming、模型切換、上下文） | Full | Full | Full | Full | Full |
| 訊息 AI（摘要／翻譯／智慧回覆／潤飾／語音轉文字／文件摘要／圖片理解） | Full | Full | Full | Full | Full |
| 知識 AI（個人知識庫、RAG、語意搜尋） | Full | Full | Full | Full | Full |
| 商業 AI（官方 AI 客服、FAQ、分類、摘要） | Full | Full | Full | Full | Full |
| AI 資料邊界 | 全平台一致：E2EE 明文絕對不得送往任何 AI（含本地自架 AI 與外部 AI API），詳見 AI_DATA_POLICY；本地 AI 亦須經授權與遮罩規範 | 全平台一致 | 全平台一致 | 全平台一致 | 全平台一致 |
| AI 後端 | 全平台一致：Ollama＋vLLM＋LocalAI 優先＋外部 AI API Adapter 可選擴充，可切換模型、Streaming、Tool Calling；向量檢索用 pgvector（已定案，見 ADR-017）；資源需求見 AI_DATA_POLICY，待查證／待實測 | 全平台一致 | 全平台一致 | 全平台一致 | 全平台一致 |

### 2.8 管理後台（§19，僅 Web／Admin）

| 功能 | Android | iOS | Windows | macOS | Web |
|---|---|---|---|---|---|
| 管理後台儀表板 | N-A：管理後台不以行動 App 提供，以 Web Admin 為主 | N-A | N-A：桌面版不提供管理後台，以瀏覽器開啟 Web Admin 為主 | N-A：同 Windows | Full：React＋TypeScript Admin，獨立認證與權限 |
| 使用者／群組／社群／貼圖／系統管理 | N-A | N-A | N-A | N-A | Full |
| 檢舉審核與操作稽核 | N-A | N-A | N-A | N-A | Full |

## 3. 平台建置與發布備註

- **iOS 須 macOS CI**：iOS 建置在 Linux 上不可能，須走 macOS CI runner；本機 VM 無 flutter、無 go、無 docker，文件須如實記載，不得假裝工具鏈齊全。iOS 發布須經 App Store 審查，審查規則與時程以官方當期公告為準，待查證。
- **Android**：須經 Google Play 審查，審查規則與時程以官方當期公告為準，待查證；簽名金鑰不得寫入原始碼，不得以固定假資料冒充。
- **Windows／macOS**：安裝包格式（Windows Installer、macOS Package）與簽名／公證流程，待 Phase 11 發布階段定案，成本：待查證。
- **Web**：Production Build 與部署環境，待 Phase 11 定案；部署走 Docker Compose＋Coolify＋Traefik＋Cloudflare Free＋Let's Encrypt；PWA 的 Service Worker 快取策略與 Web Push 供應商待定案。
- **監控與部署驗證**：監控走 Prometheus＋Grafana＋Loki＋OpenTelemetry＋GlitchTip 自架，覆蓋 API、WebSocket、資料庫、Worker、Media、AI；Docker Compose 啟動／連線／上傳／訊息／AI 切換／LiveKit／監控皆為「待 CI 或有 Docker 的環境驗證」，此 VM 無 docker，不得寫已驗證。
- **五平台同步驗收**：Phase 10 驗收要求同一帳號五平台登入驗同步；各平台用合理介面，不可複製手機畫面（手冊 L）。

## 4. 規格缺口（只列、不補決定）

1. 桌面相機掃描 QR Code 的支援範圍尚未定案。
2. Web 生物辨識（WebAuthn）是否納入範圍尚未定案。
3. 桌面與 Web 的錄音、螢幕分享能力邊界，須以各平台與瀏覽器實測為準，尚未實測。
4. Web Push 供應商與 PWA 快取策略尚未定案。
5. Windows／macOS 安裝包格式與簽名／公證流程尚未定案。
6. 行動端 Bot 完整管理的範圍劃分（行動端簡易管理與 Web 完整管理的邊界）尚未定案。
7. 四尺寸版型驗證的具體尺寸與裝置清單尚未定案。

---

*本文件為 Phase 00 草案，未批准；批准權在統籌與使用者。不得宣稱產品完成。*
