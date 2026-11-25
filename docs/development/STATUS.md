# ETHER 持續開發狀態（STATUS.md）

> 精簡進度檔：只記實際完成狀態、驗證證據、真正阻斷項與下一步。
> 更新於每個里程碑完成時，不得反覆重寫。授權依據：2026-11-25
> 持續自主開發總授權（完成一階段自動推進下一階段）。

## 1. 總況

| 工作線 | 狀態 |
|---|---|
| Track A：Phase 01 步驟 7（Docker／CI 驗證） | 等待隔離環境。本機無 Docker／Buildx，不得擅自使用未核准環境。 |
| Track B：Phase 02（設計系統與 Web UI） | 已驗收（2026-11-25 13:31）。`apps/web` 可建置、可啟動；展示資料，未接後端。 |
| Track B：Phase 03 M1（身分基礎） | 已完成，離線驗證通過（見 §2）。未接真實 Keycloak／PostgreSQL。 |
| Track B：Phase 03 M2a（前端 API 接線） | 已完成，測試全過（見 §2 之 M2a）。預設仍為展示模式；正式 OIDC 登入未接。 |
| Track B：Phase 03 M2b（PostgreSQL＋Migration＋Keycloak 準備） | 程式碼已完成，離線驗證通過（見 §2c）。migration 真實套用與 Keycloak 實機整合仍待隔離環境（測試已備妥，見 §3）。 |
| Track B：Phase 03 M2c（Web 裝置與工作階段管理） | 已完成，測試全過（見 §2d）。展示模式與 API 模式皆可操作；實機後端驗證仍待隔離環境。 |
| Track B：Phase 04 M3（好友聯絡人，離線可驗證部分） | 已完成，離線驗證通過（見 §2e）。聯絡人同步（Contact）未實作，待隱私文件定稿；migration 000005 與實機驗證仍待隔離環境。 |
| Track B：Phase 03 M2d（Web OIDC 授權碼＋PKCE 登入） | 已完成，測試全過（見 §2f）。開發者 Token 登入已退為後備路徑；Token 只存記憶體；實機 Keycloak 驗證仍待隔離環境。 |
| Track B：Phase 05 M5（即時訊息核心，離線可驗證部分） | 已完成，離線驗證通過（見 §2g）。對話／訊息端點＋Gateway 行程內扇出＋Web REST 接線；E2EE 未實作；NATS 轉接器僅佔位；migration 000006 與實機驗證仍待隔離環境。 |
| Track B：Web 對話清單與歷史遷移（M5 後續） | 已完成，測試全過（見 §2h）。API 模式下清單與歷史改讀後端；展示模式不變；實機後端驗證仍待隔離環境。 |
| Track B：Phase 06 M6（群組，後端離線可驗證部分） | 已完成，離線驗證通過（見 §2i）。群組／成員／角色／邀請／公告／設定端點＋migration 000007；投票與社群未實作；Web 接線已完成（見 §2j），實機驗證仍待隔離環境。 |
| Track B：Phase 06 M6 後續（群組 Web 接線） | 已完成，測試全過（見 §2j）。展示模式與 API 模式雙軌；實機後端驗證仍待隔離環境。 |
| Track B：Phase 06 媒體模組（後端離線可驗證部分） | 已完成，離線驗證通過（見 §2k）。上傳會話／檔案詮釋資料／縮圖紀錄／存取授權檔案側＋migration 000008；掃描與縮圖產生未實作；MinIO 轉接器僅介面；Web 接線已完成（見 §2l）；實機驗證仍待隔離環境。 |
| Track B：Phase 06 媒體 Web 接線 | 已完成，測試全過（見 §2l）。展示模式與 API 模式雙軌；實機後端驗證仍待隔離環境。 |
| Track B：訊息推播中繼（Outbox＋Relay）與 Gateway 訂閱成員驗證 | 已完成，離線驗證通過（見 §2m）。Outbox 同交易寫入＋Relay 迴圈＋Gateway 成員驗證介面；NATS 發佈者已落地（見 §2y），api 側成員查詢待整合；migration 000009 與實機驗證仍待隔離環境。 |
| Track B：已讀回條 Web 接線 | 已完成，測試全過（見 §2n）。API 模式開啟對話送 read 回條、重載清單取權威未讀數；展示模式行為不變；實機後端驗證仍待隔離環境。 |
| Track B：建立對話 UI | 已完成，測試全過（見 §2o）。展示／API 雙模式經好友清單建立一對一對話；既有 DM 直接開啟；實機後端驗證仍待隔離環境。 |
| Track B：訊息反應與置頂（後端離線可驗證部分） | 已完成，離線驗證通過（見 §2p）。反應（新增／更換／取消／彙總）與置頂（置頂／取消／清單、上限 5）端點＋migration 000010；反應／置頂不寫 Outbox 事件（§21 無事件名）；Web 接線已完成（見 §2q）；實機驗證仍待隔離環境。 |
| Track B：訊息反應與置頂 Web 接線 | 已完成，測試全過（見 §2q）。展示模式與 API 模式雙軌（反應列、置頂區塊）；實機後端驗證仍待隔離環境。 |
| Track B：delivered 回條與回條列表 Web 接線 | 已完成，測試全過（見 §2r）。API 模式歷史載入時送 delivered 回條、回條列表查詢、自己訊息已送達／已讀標示；展示模式行為不變；實機後端驗證仍待隔離環境。 |
| Track B：訊息附件關聯與轉傳（後端離線可驗證部分） | 已完成，離線驗證通過（見 §2s）。發送時／事後附件關聯、附件列出、轉傳與轉傳紀錄查詢端點＋migration 000011；轉傳以 message.received 事件承載；Web 接線已完成（見 §2t）；實機驗證仍待隔離環境。 |
| Track B：訊息附件與轉傳 Web 接線 | 已完成，測試全過（見 §2t）。展示模式與 API 模式雙軌（附件顯示區塊、轉傳對話框、附件挑選對話框、發送時關聯）；下載按鈕已於 §2aa 接線；實機後端驗證仍待隔離環境。 |
| Track B：訊息編輯／收回／僅自己刪除 Web 接線 | 已完成，測試全過（見 §2u）。展示模式與 API 模式雙軌（自己訊息的操作列、編輯輸入框、收回／刪除確認對話框）；實機後端驗證仍待隔離環境。 |
| Track B：訊息搜尋（後端離線可驗證部分） | 已完成，離線驗證通過（見 §2v）。記憶體索引＋事件投影器（冪等）＋權限過濾查詢端點；Meilisearch 轉接器已落地、填充迴圈已接線（見 §2y）；重建 PG 掃描待後續；實機驗證仍待隔離環境。 |
| Track B：訊息搜尋 Web 接線 | 已完成，測試全過（見 §2w）。展示模式與 API 模式雙軌（搜尋面板、篩選、結果清單、跳轉醒目）；索引填充迴圈未接線，API 模式結果可能不完整（面板明載）；實機驗證仍待隔離環境。 |
| Track B：對話成員管理 Web 接線 | 已完成，測試全過（見 §2x）。展示模式與 API 模式雙軌（成員區塊、新增成員對話框、離開對話確認）；實機後端驗證仍待隔離環境。 |
| Track A：NATS 推播發佈者＋Meilisearch 轉接器＋索引填充迴圈 | 已完成，離線驗證通過（見 §2y）。新依賴 nats.go v1.52.0、meilisearch-go v0.37.0；Relay 與消費迴圈依開關承載於 api 行程（Worker 接線列後續重構）；實機驗證仍待隔離環境。 |
| Track B：媒體位元組上傳通道（後端離線可驗證部分） | 已完成，離線驗證通過（見 §2z）。MinIO presigned 直傳、完成 Stat 校驗、授權核發下載 URL；零新依賴；Web 接線已完成（見 §2aa）；實機驗證仍待隔離環境。 |
| Track B：媒體位元組上傳通道 Web 接線 | 已完成，測試全過（見 §2aa）。展示模式與 API 模式雙軌（上傳流程、重試、附件下載）；實機後端驗證仍待隔離環境。 |
| Track B：Phase 08 社群動態（Timeline）後端離線可驗證部分 | 已完成，離線驗證通過（見 §2ab）。貼文／留言／按讚端點＋migration 000012；分享／收藏／檢舉／置頂／限時動態未實作；Web 接線已完成（見 §2ac）；實機驗證仍待隔離環境。 |
| Track B：社群動態（Timeline）Web 接線 | 已完成，測試全過（見 §2ac）。展示模式與 API 模式雙軌（時間線／貼文／留言／按讚、媒體關聯挑選）；實機後端驗證仍待隔離環境。 |
| Track B：Phase 07 通話會話（Call）後端離線可驗證部分 | 已完成，離線驗證通過（見 §2ad）。發起／接聽／拒接／忙線／未接／結束、參與者、事件、紀錄查詢端點＋migration 000013；LiveKit 房間建立、Token 簽發、即時信令未實作；Web 接線已完成（見 §2ae）；實機驗證仍待隔離環境。 |
| Track B：通話會話（Call）Web 接線 | 已完成，測試全過（見 §2ae）。展示模式與 API 模式雙軌（通話清單／發起／來電／通話中操作、參與者與事件）；媒體與即時信令未接通；實機驗證仍待隔離環境。 |
| Track B：Phase 08 貼圖（Sticker）後端離線可驗證部分 | 已完成，離線驗證通過（見 §2af）。貼圖包／貼圖／收藏／最近使用／搜尋端點＋migration 000014；購買交易與商店展示未實作；Web 接線已完成（見 §2ah）；實機驗證仍待隔離環境。 |
| Track B：貼圖（Sticker）Web 接線 | 已完成，測試全過（見 §2ah）。展示模式與 API 模式雙軌（貼圖包清單／詳情／收藏／最近使用／搜尋、建立／編輯／刪除／新增／移除）；無購買流程、貼圖只顯示詮釋資料；實機驗證仍待隔離環境。 |
| Timeline `feedPage` 游標修正 | 已完成（見 §2ag）。回傳游標改取最後一筆回傳列，掃描游標與回傳游標分離；2 項分頁回歸測試全過，全樹 311 項全過；實機多頁續拉併入 Timeline 實機流程。 |
| ETHER AI Provider Adapter 介面層 | 僅完成規格盤點（見 §2ai）。核心介面語義未定稿（介面名、請求／回應型別、串流格式、模型清單與預設皆待定；AI_DATA_POLICY 為草案未批准），未寫功能程式碼；`internal/ai` 不存在；實作須待規格定稿。 |
| 社群（Community）後端 | 僅完成規格盤點（見 §2aj）。核心管理語義未定稿（分級權限等級與操作對照表、visibility／category／狀態／邀請狀態值域、角色與 role_id 主從、頻道與 conversations 對應皆待定），未寫功能程式碼；`internal/community` 僅 doc.go 佔位；實作須待規格定稿。 |
| GitHub Actions 一次性隔離整合測試環境（草稿） | 草稿已撰寫（見 §2ak）。Workflow、compose override、驗收報告範本皆未執行、未驗證；第一次實機執行須待 Repository 建立＋workflow 檔案經網頁建立（API 推送 `.github/workflows/` 會回 403），且 checkout／upload-artifact 的完整 commit SHA 須先查證填入。 |

## 2. M1：Phase 03 身分基礎（2026-11-25）

實作（`services/api`，純標準庫，零新依賴，`go.mod`／`go.sum` 未動）：

- `internal/auth/oidc.go`：Keycloak OIDC 驗證。僅 RS256、JWKS（kid 選取＋未知 kid 強制重抓）、iss／exp（60 秒寬容）／sub 驗證、aud 選配；discovery 探測輔助函式。
- `internal/auth/middleware.go`：`RequireAuth`（Bearer 標頭、無驗證器時 503 fail-closed、身分寫入 context）。
- `internal/auth/session.go`：Session 領域。只存 Refresh 雜湊；輪換＋重放偵測（重放即撤銷家族）；撤銷為狀態標記；記憶體 Store。
- `internal/user/user.go`：User＋Settings 領域。subject 綁定建置（JIT、冪等）、狀態轉換（僅 active↔disabled）、設定 PATCH 驗證；記憶體 Store。
- `internal/profile/profile.go`：公開檔案領域。GetOrCreate、PATCH 驗證、public_id 唯一性；記憶體 Store。
- `internal/device/device.go`：裝置領域。登記只收 SHA-256 雜湊識別碼、撤銷標記、信任設定；記憶體 Store。
- `internal/transport/{errors,json}.go`：含 details 的驗證錯誤寫出、嚴格 JSON 解碼、領域錯誤→HTTP 對應（404 不洩漏存在性、500 不洩漏細節）。
- `internal/ids/`：UUID v4 產生器。
- Handler 與路由（`cmd/api/main.go` 接線，經 `transport.Router`，`router.go` 未動）：`GET /api/v1/auth/me`、`GET/POST /api/v1/auth/sessions…`、`POST /api/v1/auth/logout`、`GET/POST /api/v1/auth/devices…`、`GET/PATCH /api/v1/profiles/me`、`GET/PATCH /api/v1/users/me/settings`。`config` 新增 `KEYCLOAK_REALM`（見 §4 缺口 1）。
- `migrations/000002_identity.up/down.sql`：users、user_profiles、user_settings、user_devices、user_sessions、user_security_keys（ERD §6.1 草案欄位，差異見檔頭註解）。

驗證證據（Go 1.25.0，專案樹外工具鏈，雜湊已驗；`GOFLAGS=-mod=readonly GOPROXY=off`）：

- `go build ./...`：通過。`go vet ./...`：退出碼 0。
- `go test ./...`（api 模組）：30 個測試全過，0 失敗；gateway／worker 模組測試亦通過。
- `gofmt -l`：M1 檔案全數乾淨（`health.go` 既有漂移未動，待授權）。

## 2b. M2a：前端 API 接線（2026-11-25）

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile）：

- `src/data/api.ts`：`EtherApiClient`（M1 全部 11 個身分端點）、`ApiError`／`AuthError`
  （ETHER 錯誤封包解析、401 專類、網路失敗 status 0）、`LocalStorageTokenStore`
  （localStorage 暫行儲存，待 SECURITY_ARCHITECTURE 定稿——見 §3）。
- `src/data/api-provider.ts`：混合式 `ApiChatDataProvider`。身分（me）與公開檔案
  來自 API；聯絡人／對話／訊息仍委派展示資料（Phase 05 前無對話端點，UI 徽章
  明示）。Token 登入（開發階段貼上 Keycloak Access Token）、401 回登入畫面、登出
  盡力撤銷＋清除 Token、檔案更新樂觀寫入＋失敗還原。
- 接線：`VITE_DATA_MODE=api` 啟用（預設 `demo`，行為與 Phase 02 驗收版一致）；
  `VITE_API_BASE_URL` 指定 API 來源。`AuthView` 新增 API 專用 Token 表單；
  `ProfileView` 以 `updateProfileAsync` 同步後端並回報真實成敗；連線失敗顯示
  ErrorState（重試／改用展示模式）。

驗證證據（`node tests/run.mjs`）：**43 項全過**（原 26 項＋新增 17 項 API 測試，
stub fetch 覆蓋封包／401／網路失敗／還原／重試）。`tsc --noEmit`、`eslint`、
`prettier` 全過；`vite build` 通過（74 modules）。對比度與響應式靜態稽核仍通過。

## 2c. M2b：PostgreSQL 儲存庫、Migration 管理、Keycloak 整合準備（2026-11-25）

依使用者 M2b 持續開發指令執行：不等待隔離環境，完成全部不需
Docker／真實資料庫／真實 Keycloak 的程式工作；實機驗證項目標示
「環境待驗證」，不冒充通過。

實作（`services/api`，新增唯一外部依賴 `github.com/lib/pq v1.12.3`
純 Go 驅動；`go.mod`／`go.sum` 因此變動，餘同 M1 慣例）：

- `internal/postgres/`：`Querier` 查詢介面（`WrapDB`／`WrapTx` 適配、
  uuid `::text` 讀出／`::uuid` 傳入、可空欄位 NULL 語義、23505／23503
  錯誤對映）；`Migrator`（`schema_migrations` 記帳、每 migration 單一
  交易、advisory lock 序列化、`MigrateUp`／`MigrateDownOne`／
  `MigrateDownTo`／`Status`）；`cmd/migrate` 執行工具（`up`／`down`／
  `down-to`／`status`，DSN 取 `DATABASE_URL`，不印 DSN）。
- 儲存庫（各領域 `postgres.go`，介面與模組邊界不變）：User／Settings、
  Profile、Device、Session 四組；唯一性違反依約束名對映（僅 email／
  public_id 衝突為欄位錯誤，其餘為伺服器錯誤）；Session 撤銷雙欄
  （`is_revoked`＋`revoked_at`）一致寫入、讀出合成規則。
- `SessionStore` 介面擴充 `IsHashConsumed`／`MarkHashConsumed`；
  `SessionService.Rotate` 改以儲存層追蹤已消耗雜湊（先標記、後寫入，
  fail-closed；併發嚴格序列化限制見 §3 待決定 8）。`MemorySessionStore`
  同步實作（行程內語義不變）。
- Migration（`services/api/migrations/`，內嵌 `migrations.FS`＋靜態驗證
  `Validate`：編號連續、up／down 配對、up 不得含 DROP）：
  `000003_identity_corrections`（補 `users.keycloak_subject` UNIQUE 與
  `email_verified`——M2b 盤點發現 000002 缺此二欄，領域模型無法完整
  落地；前提見檔頭：截至 2026-11-25 無任何環境套用過本目錄 migration）、
  `000004_session_refresh_hashes`（已消耗雜湊持久化表，append-only）。
  `MIGRATIONS.md` §5 定稿為專案內 migrator 方案；`schema_meta` 保留。
- 接線（`cmd/api/main.go`＋`internal/config`）：`IDENTITY_STORE`
 （預設 `postgres`，Ping 失敗即報錯退出；`memory` 僅開發用，日誌明示）。
  `KEYCLOAK_AUDIENCE`（空值不強制，設定值強制 aud 含該值）與
  `KEYCLOAK_CLIENT_ID` 經環境變數讀入；`.env.example` 補佔位符。
  `Dockerfile` 依其檔頭既有規則補 `COPY go.sum`／`migrations/` 並建置
  `migrate` 執行檔（compose／Makefile／EXECUTION_PLAN／`router.go` 未動）。
- Keycloak 準備：`docs/development/KEYCLOAK_INTEGRATION.md`（設定一覽、
  流程對照、實機驗證清單、Mock／實機測試區分）；
  `infrastructure/docker/keycloak/realm-ether.template.json`（Realm 範本，
  僅開發佔位符）；`oidc_integration_test.go`（`integration` tag，
  `KEYCLOAK_TEST_ISSUER`／`KEYCLOAK_TEST_TOKEN`）。

驗證證據（Go 1.25.0，`GOFLAGS=-mod=readonly GOPROXY=off`）：

- `go build ./...`、`go vet ./...`（含 `-tags integration`）皆通過；
  gateway／worker 模組建置亦通過。
- `go test ./...`（api 模組）：**64 項全過**（M1 的 30 項＋M2b 新增
  34 項：儲存庫假查詢單元測試、錯誤對映、migration 靜態驗證、aud
  陣列／未設定模式、config 載入）；`go test -tags integration`
  時 6 項整合測試（migration 套用／回滾、4 領域持久化、Keycloak
  discovery／Token）全部跳過並標示「環境待驗證」（無
  `ETHER_TEST_DATABASE_URL`／`KEYCLOAK_TEST_ISSUER`），非冒充通過。
- `gofmt -l`：M2b 檔案全數乾淨（`health.go` 既有漂移仍未動）。

## 2d. M2c：Web 裝置與工作階段管理（2026-11-25）

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile）：

- `src/data/types.ts`：`SessionItem`／`DeviceItem`（展示層形態，不含
  任何 Token 資料）；`src/data/demo.ts`：`DEMO_SESSIONS`／`DEMO_DEVICES`
  展示資料（虛構紀錄，撤銷只改本機狀態）。
- `src/data/provider.ts`：`ChatDataProvider` 新增可選非同步方法
  `listSessionsAsync`／`listDevicesAsync`／`revokeSessionAsync`／
  `revokeDeviceAsync`（與 `updateProfileAsync` 相同的可選模式）；
  展示 provider 以 fixture 實作、撤銷只翻本地旗標。
- `src/data/api-provider.ts`：四方法經 `EtherApiClient` 讀後端
  （wire→展示形態對映）；401 時清除 Token、回未授權並 rethrow，
  與既有身分流程一致。
- `src/views/ProfileView.tsx`：「裝置與工作階段」區塊取代「隱私與
  裝置」佔位文字：載入／錯誤／重試／不支援四態、裝置與工作階段
  清單、撤銷確認對話框、成功／失敗通知；資料來源依模式誠實標示
  （API／展示）；封鎖名單與隱私設定仍為後續階段。
- 樣式：`et-device-list`／`et-device-row` 等（既有 token 變數，無新色）。

驗證證據（`node tests/run.mjs`）：**48 項全過**（M2a 的 43 項＋新增
5 項：清單對映、撤銷端點呼叫、401 清除 Token、展示資料撤銷、
SSR 區塊標題與載入文字）。`tsc --noEmit`、`eslint`、`prettier` 全過；
`vite build` 通過。真實瀏覽器互動驗證仍待 CI（本機無 Chromium）。

## 2e. M3：Phase 04 好友聯絡人（2026-11-25，離線可驗證部分）

範圍（ROADMAP 手冊 F 的最小可驗收子集）：好友邀請 lifecycle
（發送／接受／拒絕／取消）、好友清單、搜尋（公開識別碼）、
封鎖／解除封鎖、隱藏／取消隱藏、解除好友。聯絡人同步
（Contact 模組，`contact_sync_records`）未實作：雜湊演算法、
授權撤回流程待隱私文件定稿，不得以未定稿的雜湊方案上傳
通訊錄資料；`internal/contact/doc.go` 未動。備註／分類、
推薦、邀請連結、QR 邀請皆未實作，待後續里程碑。

實作（`services/api`，零新依賴，`go.mod`／`go.sum` 未動；
`router.go`、compose、Makefile、EXECUTION_PLAN.md、
000001–000004 皆未動）：

- `internal/friend/friend.go`：Friend 領域（依賴 DAG：User 存在
  確認、Profile 公開檔案摘要、Auth 由中介層落實；不得直連
  他模組資料表）。`friendships` 以字典序單列存放無序配對
  （`user_id < friend_id`）；邀請狀態機僅待處理可回應／取消；
  雙向同時待處理時後送出者收重複錯誤；封鎖同時解除好友關係
  並取消雙向待處理邀請，解除封鎖不恢復關係；封鎖任一方向
  存在時，搜尋與邀請皆回 404（不可區分，不得洩漏封鎖狀態）；
  隱藏僅限有效好友，不影響關係與訊息。記憶體 Store 供開發／測試。
- `internal/friend/postgres.go`：PG 儲存庫（SQL 慣例同 M2b：具名
  欄位、`::text`／`::uuid`、NULL 語義；待處理唯一索引衝突對映
  為欄位錯誤）。`internal/profile` 新增唯讀 `GetByUserID`／
  `GetByPublicID`（供 Friend 豐富化與搜尋，不建立檔案）。
- `migrations/000005_friends.up/down.sql`：四表（`friendships`、
  `friend_requests`、`blocked_users`、`hidden_users`）；與 ERD 草案
  的差異（單列配對＋CHECK、部分唯一索引、blocked_at／hidden_at
  與 created_at 同值保留、狀態值域為 M3 詞彙）見檔頭，待定稿覆核。
- 接線（`cmd/api/main.go`）：`/api/v1/friends/*` 13 個端點，皆經
  `RequireAuth`；`IDENTITY_STORE` 兩模式皆可用（memory／postgres）。

實作（`apps/web`，零新依賴，`package.json` 不動、無 lockfile）：

- `ChatDataProvider` 加可選方法（好友清單／邀請清單／搜尋／
  發送／回應／取消／解除／隱藏／封鎖清單／封鎖／解封）；
  `EtherApiClient` 對映後端端點，401 沿用 `failUnauthorized`；
  展示 provider 以展示資料實作（操作只改本機狀態）。
- `ContactsView` 加 `dataMode`：API 模式讀真實資料（載入／錯誤／
  重試／不支援四態、搜尋＋邀請、邀請回應、隱藏／解除／封鎖
  確認對話框、封鎖清單），資料來源誠實標示；展示模式行為不變。

驗證證據（Go 1.25.0，`GOFLAGS=-mod=readonly GOPROXY=off`）：

- `go build ./...`、`go vet ./...`（含 `-tags integration`）皆通過；
  `gofmt -l` 乾淨（`health.go` 既有漂移仍未動）。
- `go test ./...`：**79 項全過**（M2b 的 64 項＋M3 新增 15 項：
  領域狀態機／驗證／冪等 7 項、PG 假查詢 6 項、Profile 唯讀
  方法 1 項、HTTP 整合流程 1 項）；`go test -tags integration`
  時新增的 `TestPG_FriendFlowPersists` 跳過並標示「環境待驗證」。
- Web（`TZ=Asia/Taipei node tests/run.mjs`）：**56 項全過**（M2c 的
  48 項＋新增 8 項：好友對映／邀請／搜尋 404／端點呼叫／401
  清除 Token／展示資料好友操作／SSR API 模式標題與載入文字）。
  `tsc --noEmit`、`eslint`、`prettier`、`vite build`、對比度與
  響應式靜態稽核全過。註：裸 UTC 環境下 1 項 Phase 02 既有時間
  測試（`formatMessageTime`）因時區假設失敗，與 M3 無關；在
  Asia/Taipei 時區下全過。
- 已知限制：邀請到期不自動轉態（須排程／Worker）；清單無游標
  分頁（沿 M1／M2 清單端點慣例，待 API 定稿）；併發接受同一
  邀請的競態以唯一索引收斂為欄位錯誤，未以交易序列化。

## 2f. M2d：Web OIDC 授權碼＋PKCE 登入（2026-11-25）

範圍：取代 M2a 的開發者貼上 Token 暫行作法，改為標準 OIDC
授權碼＋PKCE 流程；Token 儲存、換發、登出語義依 IDENTITY §3–§4
與 SECURITY_ARCHITECTURE §5（未定稿處以暫行選擇明載，不得自稱
定稿）。後端驗證邏輯未動（oidc.go 的 iss／aud／kid 規則不變，
前端設定不得與之矛盾）。

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile）：

- `src/data/oidc.ts`（新）：設定解析（`VITE_KEYCLOAK_ISSUER`／
  `VITE_KEYCLOAK_CLIENT_ID`／`VITE_KEYCLOAK_REDIRECT_URI` 三者缺一
  即停用此路徑）；PKCE（64 位元組 verifier、S256 challenge，
  WebCrypto 缺失時明確報錯、不得降級 plain）；OIDC discovery
  （端點取自 .well-known 文件、issuer 須一致，不得猜測預設路徑）；
  授權 URL 組裝（`openid profile email`、state／nonce）；回呼解析
  （error 參數轉可讀訊息、state 驗證）；Token 交換與換發
  （form-urlencoded、public client 無 secret、Rotation）；登入暫存
  （sessionStorage，只存 state／nonce／verifier，完成即清）；
  `OidcSession`（端點快取、記憶體 Token 組、到期前換發判斷、
  換發失敗清除）。
- `src/data/api.ts`：`MemoryTokenStore`（記憶體唯一接線儲存）；
  `LocalStorageTokenStore` 標示已取代、僅保留供測試編譯；
  `EtherApiClient` 加 refresh hook（到期前主動換發、401 時重試
  一次，皆為 best-effort）。
- `src/data/api-provider.ts`：`createApiProvider` 加可選 OIDC 支援；
  `signInWithKeycloak`（導向授權端點）、`handleOidcCallback`
  （驗 state→交換→initialize，三態回報）；登出與 401 路徑同步
  清除 OIDC 工作階段（雙 Token）；無支援時行為與 M2a 相同。
- `src/App.tsx`：API 模式下有設定即建 OIDC 工作階段並接線
  refresh hook；啟動時偵測 `?code=`／`?error=` 回呼（處理中
  顯示載入、失敗顯示錯誤、完成後清查詢字串；StrictMode 重跑
  effect 時以 ref 保證授權碼只處理一次）；`AuthView` 依設定顯示
  Keycloak 主按鈕，開發者 Token 表單僅在未設定 OIDC 或
  `VITE_ALLOW_DEV_TOKEN_LOGIN=true` 時顯示。
- `src/views/AuthView.tsx`：Keycloak 登入按鈕＋載入／錯誤狀態、
  回呼錯誤顯示、Token 儲存改為記憶體說明文字。

驗證證據（`TZ=Asia/Taipei node tests/run.mjs`）：**80 項全過**
（M3 的 56 項＋新增 24 項：設定解析、PKCE 產生與 challenge、
授權 URL 參數、回呼解析與錯誤對映、discovery 驗證、交換／換發
請求 shape、暫存往返、session 輪換與失敗清除、provider 回呼
端到端與 state 不符拒絕、無支援路徑、client 401 重試與到期前
換發）。`tsc --noEmit`、`eslint`、`prettier`、`vite build`、
對比度與響應式靜態稽核全過。真實瀏覽器互動與真實 Keycloak
驗證仍待隔離環境（本機無 Chromium、無 Keycloak）。
- 已知限制：換發只在到期前 60 秒內（預設 skew）與 401 時觸發，
  無背景計時換發；重新整理分頁即失去工作階段（memory-only 的
  直接後果）；登出未呼叫 Keycloak end_session_endpoint，Keycloak
  端 Session 仍在（後端 logout 已呼叫，見 IDENTITY §3.2 同步缺口
  12）；nonce 僅隨請求送出，未驗證 id_token（本端不處理 id_token，
  身分以後端驗證的 Access Token 為準）。

## 2g. M5：Phase 05 即時訊息核心（2026-11-25，離線可驗證部分）

範圍（ROADMAP 手冊 G 的最小可驗收子集）：對話（DM／群組建立、
成員管理、離開）、訊息（發送／游標分頁／編輯／收回／僅自己
刪除／已讀回條）、每對話序號與冪等鍵、Gateway 行程內訂閱與
扇出、Web REST 接線。未實作：E2EE 任何密碼學功能（內容一律視
為不透明字串；金鑰分發與棘輪狀態不在範圍）、Transactional
Outbox（待定稿）、NATS JetStream 轉接器（僅佔位）、WebSocket
送出訊息、訂閱成員身份驗證、Heartbeat／Rate Limit／
Backpressure、訊息反應／附件／轉發／置頂、群組設定。

實作（`services/api`，零新依賴，`go.mod`／`go.sum` 未動；
`router.go`、compose、Makefile、EXECUTION_PLAN.md、
000001–000005 皆未動；模組依賴 DAG 未違反，Chat 不 import
Message，Message→Chat 只經 `ConversationAccess` 介面）：

- `internal/chat/chat.go`：Conversation 領域（類型 direct／group、
  角色 member／admin 為 Phase 05 實作詞彙，ERD 值域待定稿）。
  DM 建立須對方存在、無任一方向封鎖（封鎖時回 404，不得洩漏）、
  雙方為好友（非好友為欄位錯誤）；同一對使用者僅一個 DM，存在
  時回傳既有（`isNew=false`），曾離開者重新加入；群組標題必填
  1–64 runes、至少一名其他成員、去重、成員須存在、建立者為
  admin；僅 admin 可新增成員，DM 不可新增；離開為 `left_at`
  狀態標記，對話列永不物理刪除。`NoteMessage` 更新最後訊息快取
  與 `updated_at`；`MarkRead` 序號不得倒退。記憶體 Store。
- `internal/message/message.go`：Message 領域（`kind` 本里程碑僅
  `text`；內容 1–4000 runes 實作預設、去重鍵 ≤128 字元，皆待
  定稿）。每對話序號單調遞增；`client_message_id` 冪等命中回傳
  既有訊息且不重發 `NoteMessage`；引用以 `message_replies` 表
  承載；編輯僅作者、首次編輯保留原值、`edit_count` 遞增、收回／
  已刪除不可編輯；收回僅作者、列保留、回應內容為空、重複收回
  冪等；僅自己刪除僅影響作者檢視；回條 `delivered`／`read`，
  同一回條重複寫入冪等，不得為自己的訊息寫回條，已讀回條推進
  水位；未讀數排除自己所發與已收回訊息。伺服器端狀態僅
  `sent`／`recalled`。游標分頁：`before_seq`（倒序讀舊歷史）與
  `since_seq`（順序續拉）二選一，`limit` 預設 30、上限 100，
  `next_cursor` 為頁內最小／最大序號。記憶體 Store。
- `internal/friend/friend.go`：新增唯讀 `AreFriends`／`BlockedBetween`
  （讀既有 Store 方法，無行為變更），供 Chat／Message 驗證 DM 前提。
- PG 儲存庫（`postgres.go`，SQL 慣例同 M2b）：序號以
  `MAX(sequence_number)+1` 子查詢分配，`(conversation_id,
  sequence_number)` 唯一衝突時至多重試 3 次；去重鍵競態以哨兵
  回服務層重讀。`internal/postgrestest` 假實作擴充 `*int`／
  `*int64` 掃描支援。
- `migrations/000006_messaging.up/down.sql`：只建五表
  （conversations、conversation_members、messages、
  message_receipts、message_replies）；`conversation_members`
  增 `last_read_sequence_number`（ERD 只有 `last_read_message_id`，
  水位比較須用序號）；`conversations.last_message_id` 與
  `conversation_members.last_read_message_id` 的 FK 以 `ALTER TABLE`
  後加（建表順序不允許前向參照）；與 ERD 草案的差異見檔頭。
- 接線（`cmd/api/main.go`）：13 個新路由，全部經 `RequireAuth`，
  兩種儲存模式皆接線：`GET/POST /api/v1/conversations`、
  `GET /api/v1/conversations/{id}`、`GET/POST .../members`、
  `POST .../leave`、`GET/POST .../messages`、
  `PATCH /api/v1/messages/{id}`、`POST .../recall`、
  `POST .../delete`、`GET/POST .../receipts`。

實作（`services/realtime-gateway`，零新依賴，純標準庫）：

- `internal/broker/broker.go`：`Broker` 介面（Publish／Subscribe／
  Close）＋行程內 `MemoryBroker`（訂閱通道緩衝 64、滿則丟棄最舊
  並計數，不得阻塞發佈者；`Drops` 計數供監控）＋`JetStreamBroker`
  佔位（任何呼叫回 `ErrNotConfigured`，不新增 NATS 依賴）。
- `internal/auth/auth.go`：OIDC 驗證器（與 api 規則一致：僅 RS256、
  JWKS kid 選取＋未知 kid 強制重抓、iss／exp 60 秒寬容／sub、aud
  僅設定時強制；internal 套件不得跨 module，為有意的重複實作，
  規則變更須兩處同步）＋discovery 探測。
- `internal/ws/ws.go`：RFC 6455 標準庫實作（交握、訊框、分段重組、
  Ping 自動回 Pong、客戶端訊框須遮罩、預設訊息上限 1 MiB、
  伺服器不發 Ping——Heartbeat 間隔待定稿）。
- `internal/hub/hub.go`：連線中樞。認證（查詢參數 `access_token`，
  Token 不得記錄；驗證器為 nil 時 503 fail-closed）、`connection.ready`
  （session_id、server_time、supported_versions）、訂閱／取消訂閱
  （主題慣例 `conversation.<id>`）、typing 轉發（sender_id 與 user_id
  由伺服器以認證身份填寫，不得信任客戶端）、Broker 扇出（跳過
  發起連線）、`message.send` 一律回 `ETHER_COMMON_UNAVAILABLE`
  （請改用 REST）、關閉與清理、寫入序列化。訂閱時不驗證成員身份
  （待 api 整合，不得宣稱已驗證）。
- 接線（`cmd/realtime-gateway/main.go`）：`/ws/v1` 走 Hub（驗證器由
  `KEYCLOAK_URL`／`KEYCLOAK_REALM` discovery 建立，建不出來時 WS
  fail-closed、`/healthz` 照常提供）；`main_test.go` 的 501 佔位測試
  已改為認證／fail-closed 測試。

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile）：

- `src/data/types.ts`：`ConversationItem`／`ConversationMemberItem`／
  `MessageItem`／`MessagePage`／`ReceiptItem`。
- `src/data/api.ts`：對話／訊息／回條 wire 型別與 13 個客戶端方法
  （游標參數以 `before_seq`／`since_seq`／`limit` 傳遞）。
- `src/data/provider.ts`：`ChatDataProvider` 新增可選非同步方法
  （列對話／建對話／列成員／離開／列訊息／發送／編輯／收回／
  刪除／回條）；展示 provider 以 fixture 實作（含游標語義與僅作者
  覆蓋）；`conversationSequence` 依時間戳排序（字串比較會被不同
  時區字尾誤導）。
- `src/data/api-provider.ts`：wire→展示形態對映，401 沿用
  `failUnauthorized`；發送時產生 CSPRNG 去重鍵。
- `src/data/realtime.ts`（新）：WebSocket 客戶端介面與未連線 stub
  （一切操作明確拒絕，`isConnected` 恆 false，不得假裝即時連線已通）。
- `src/views/ChatsView.tsx`：composer 改走 `sendMessageAsync`（存在
  時 await、失敗以 toast 回報，不存在時沿用同步 `sendMessage`）。

驗證證據（Go 1.25.0，`GOFLAGS=-mod=readonly GOPROXY=off`；Web
`TZ=Asia/Taipei`）：

- api 模組：`go build ./...`、`go vet ./...`（含 `-tags integration`）
  皆通過；`go test -count=1 ./...`：**104 項全過、0 失敗**（先前
  79 項＋新增 25 項：chat 領域 7、message 領域 7、friend 唯讀
  方法 1、chat PG 假查詢 4、message PG 假查詢 5、HTTP 流程 1）；
  `go test -tags integration` 時新增的 `TestPG_ConversationPersists`
  與 `TestPG_MessageFlowPersists` 皆 SKIP 並標示環境待驗證。
- gateway 模組：`go build ./...`、`go vet ./...`（含 `-tags integration`）
  皆通過；`go test -count=1 ./...`：**19 項全過、0 失敗**（main
  3、auth 3、broker 4、hub 5、ws 4，含原始 TCP 客戶端交握／訊框／
  分段／Ping-Pong 與訂閱扇出／typing 隔離測試）。
- `gofmt -l`：兩模組新檔全數乾淨（`health.go` 既有漂移仍未動）。
- Web（`node tests/run.mjs`）：**89 項全過**（M2d 的 80 項＋新增
  9 項：游標參數／對映／去重鍵／編輯收回刪除回條／401 清除
  Token／展示資料游標與僅作者覆蓋／realtime stub）；
  `tsc --noEmit`、`eslint`、`prettier`（本次異動檔）、`vite build`
  （75 modules）、對比度與響應式靜態稽核全過。真實瀏覽器互動
  驗證仍待 CI（本機無 Chromium）。
- 已知限制：序號併發重試上限 3 次（高併發寫入同一對話時可能
  回伺服器錯誤，見 §3 待決定 13）；訊息列與 `message_replies` 列
  未包在同一交易（儲存介面未提供交易邊界）；回條為訊息＋使用者
  粒度（device 粒度待定稿）；群組內封鎖語義未定義（封鎖檢查僅
  限 DM 建立）；API 模式下對話清單與歷史仍為展示資料（清單遷移
  為下一里程碑，已於 §2h 完成），composer 經 REST 發送須以真實對話識別碼為前提；
  訊息推播事件（`message.received` 等）尚無生產者——須後續中繼
  接線（api 寫入後經 Broker 發佈），Gateway 目前只負責訂閱與轉發。

## 2h. Web 對話清單與歷史遷移（2026-11-25，M5 後續）

範圍：API 模式下 ChatsView 改讀後端對話清單與歷史；展示模式
行為不變。建立對話 UI、已讀回條同步、即時推播皆不在範圍。

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile）：

- `src/data/provider.ts`／`api-provider.ts`：`ChatDataProvider` 新增
  可選方法 `getCurrentUserId`（展示 provider 回 `'me'`；API
  provider 取 `GET /api/v1/auth/me` 的 `user_id`，登出／401 時
  清除），供訊息 own／other 判斷；其餘 M5 方法簽名不變。
  `api.ts`／`api-provider.ts`／`ContactsView` 的「清單仍為展示
  資料」舊註解與文案已同步更新。
- `src/views/ChatsView.tsx`：依 `dataMode` 分流（ContactsView 相同
  模式）。展示版原樣保留為 `DemoChatsView`；`ApiChatsView`：掛載
  載入對話清單（未讀數取後端 `unread_count`，依後端 `updated_at`
  降序）；切換對話載入最新一頁歷史（`before_seq` 游標「載入更
  早的訊息」，wire 降序頁以序號排序升序顯示、去重合併）；發送
  走 `sendMessageAsync`，以伺服器回應更新歷史與清單最後活動
  時間；載入／錯誤＋重試／不支援／就緒四態；重新載入失敗保留
  既有畫面；資料來源標示「對話與訊息來自後端 API」；DM 無標題
  時以成員＋好友名解析，否則「一對一對話」，群組無標題為
  「未命名群組」；寄件人名稱以好友清單解析，否則以使用者識別碼
  前 8 字元顯示；收回／僅自己刪除以文字標示。
- `src/App.tsx`：ChatsView 接 `dataMode`；側邊列未讀徽章在 API
  模式取後端回報總數；頁尾徽章改為「後端身分 · 對話來自後端
  API」。樣式新增 `et-chats__source`／`et-thread__load-more`
  （既有 token，無新色）。

驗證證據（`TZ=Asia/Taipei node tests/run.mjs`）：**95 項全過**
（M5 的 89 項＋新增 6 項：可空最後訊息欄位對映、wire 順序與空
游標、歷史 401 清除 Token、目前使用者識別碼、展示資料對話清單、
API 模式 SSR 載入文字與來源標示）。`tsc --noEmit`、`eslint`、
`prettier`、`vite build`、對比度與響應式靜態稽核全過。真實
瀏覽器互動與真實後端驗證仍待隔離環境（本機無 Chromium、無後端）。

- 已知限制：已讀回條未接線（開啟對話不清除後端未讀數，須重新
  載入清單）；無即時推播（新訊息須手動「重新載入」，realtime
  stub 不動）；歷史以工作階段內快取保存（切換對話不重抓）；
  清單搜尋僅比對標題（DM 對方名稱須開啟對話載入成員後才解析）；
  建立對話 UI 未實作；清單無分頁（沿後端清單端點現況）。

## 2i. M6：Phase 06 群組（2026-11-25，後端離線可驗證部分）

範圍（ROADMAP 手冊 H 的群組最小可驗收子集，Web 接線為後續
里程碑）：群組建立／更新、成員與角色管理（Owner／Admin／Member）、
邀請（建立／接受／拒絕／取消）、公告管理、群組設定、群組與
成員查詢、離開／解散、所有權轉移。群組聊天室容器經 Chat 介面
協作（建立時建對話容器、成員異動時同步對話成員），不得直接
寫入 conversations／messages 表。未實作：投票（表歸屬未定，
MODULE_BOUNDARIES.md 缺口 2）、社群（Community，平級模組）、
連結／QR 邀請、自訂角色（group_roles 表）、禁言／群內暱稱、
管理紀錄、@成員、群組搜尋。

實作（`services/api`，零新依賴，`go.mod`／`go.sum` 未動；
`router.go`、compose、Makefile、EXECUTION_PLAN.md、
000001–000006 皆未動；模組依賴 DAG 未違反，Group 不 import
Chat／Friend／User 套件，只經介面協作）：

- `internal/group/group.go`：Group 領域。角色權限矩陣全在後端
  裁決（僅 Owner 轉移所有權與解散、Admin 可管理成員／邀請／
  公告、一般成員僅讀取與離開；權限不足一律 404 不洩漏存在性）。
  邀請須經 Friend 唯讀介面做封鎖檢查（任一方向封鎖皆 404）；
  被邀請者無須為好友。邀請政策 `invite_policy`（預設 admin、
  all 時全體成員可邀請）。公告同時僅一則現行（新公告自動撤下
  舊公告）。解散為狀態標記，成員列與對話容器保留。記憶體 Store。
- `internal/chat/chat.go`：新增 Group 協作方法
  `CreateGroupConversation`／`AddConversationMembers`／
  `RemoveConversationMember`（權限已由 Group 層裁決，僅驗證
  對話存在與群組類型；群組角色不帶入對話成員權限）。
- `internal/group/postgres.go`：PG 儲存庫（SQL 慣例同 M2b）；
  待處理邀請部分唯一索引衝突對映為欄位錯誤（競態兜底）。
- `migrations/000007_groups.up/down.sql`：只建五表（groups、
  group_members、group_invitations、group_announcements、
  group_settings）；與 ERD 草案的差異（無 group_roles 表、無
  role_id／muted_until／nickname／invite_token／expires_at、
  無 member_count_cache、設定 FK 取 RESTRICT）見檔頭。
- 接線（`cmd/api/main.go`）：21 個新路由，全部經 `RequireAuth`，
  兩種儲存模式皆接線：`GET/POST /api/v1/groups`、
  `GET /api/v1/groups/invitations`（我的待處理邀請）、
  `POST .../invitations/{invitation_id}/{accept,decline,cancel}`、
  `GET/PATCH /api/v1/groups/{group_id}`、`POST .../disband`、
  `POST .../transfer`、`GET/POST .../members`、
  `PATCH .../members/{user_id}`、`POST .../members/{user_id}/remove`、
  `POST .../leave`、`GET/POST .../invitations`、
  `GET/POST .../announcements`、`PATCH .../announcements/{announcement_id}`、
  `POST .../announcements/{announcement_id}/retract`、
  `GET/PATCH .../settings`。

驗證證據（Go 1.25.0，`GOFLAGS=-mod=readonly GOPROXY=off`）：

- `go build ./...`、`go vet ./...`（含 `-tags integration`）皆通過；
  gateway／worker 模組建置亦通過。
- `go test -count=1 ./...`：**133 項全過、0 失敗**（先前 104 項＋
  新增 29 項：group 領域 20、group PG 假查詢 8、HTTP 流程 1）；
  `go test -tags integration` 時新增的 `TestPG_GroupFlowPersists`
  跳過並標示環境待驗證。
- `gofmt -l`：新檔全數乾淨（`health.go` 既有漂移仍未動）。
- 已知限制：跨模組無交易邊界（對話容器先建、群組列後建，
  群組列失敗時殘留孤兒對話；所有權轉移兩步寫入）；公告
  撤下＋新建兩步操作，併發發佈時後者收唯一衝突（fail-closed）；
  清單無游標分頁（沿既有清單端點慣例）；邀請不設到期；
  群組重新命名不同步對話標題（標題屬 Chat）。

## 2j. M6 後續：群組 Web 接線（2026-11-25）

範圍（§2i 的 21 個群組端點全數接線；投票、社群、@成員、群組
搜尋不在範圍）：群組清單、建立群組、群組內容（成員與角色、
邀請、公告、設定）、編輯群組資料、所有權轉移、離開／解散。
後端 Go 程式碼與 migration 皆未動（純前端里程碑）。

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile）：

- `src/data/types.ts`：`GroupItem`／`GroupMemberItem`／
  `GroupInvitationItem`／`GroupAnnouncementItem`／
  `GroupSettingsItem`（展示層形態；成員顯示名稱未知時為空字串，
  檢視以識別碼前 8 字元顯示，與 ChatsView 的 fallback 一致）。
- `src/data/api.ts`：群組 wire 型別與 21 個客戶端方法（路徑與
  請求／回應 shape 逐一對照 `internal/group/handler.go`，含
  `GET /api/v1/groups/invitations` 我的待處理邀請與群組內邀請
  清單的路徑區分）。
- `src/data/provider.ts`：`ChatDataProvider` 新增可選非同步方法
  21 個（與 M2c／M3／M5 相同的可選模式）；展示 provider 以
  fixture 實作，含與後端一致的 headline 權限規則（僅管理者可
  管理、僅擁有者可轉移／解散／變更角色、擁有者不得離開、
  同時僅一則現行公告），操作只改本機狀態；已解散群組不列入
  清單（與後端 `ListGroupsByUser` 的排除語義一致）。
- `src/data/api-provider.ts`：wire→展示形態對映，401 沿用
  `failUnauthorized`（清除 Token、回未授權並 rethrow）。
- `src/data/demo.ts`：`DEMO_GROUPS`／`DEMO_GROUP_MEMBERS`／
  `DEMO_GROUP_INVITATIONS`／`DEMO_GROUP_ANNOUNCEMENTS`／
  `DEMO_GROUP_SETTINGS`（虛構紀錄；`'me'` 為第一群組擁有者、
  第二群組一般成員，另有一則收到的群組邀請）。
- `src/views/GroupsView.tsx`（新）：單一元件雙模式（僅資料來源
  標示不同）。清單：收到的群組邀請（接受／拒絕）、我的群組、
  建立群組對話框；內容頁：成員清單與角色徽章、角色變更／
  移除（權限不足不顯示：角色變更僅擁有者、移除僅管理者且
  管理員不得移除其他管理員，皆以確認對話框執行）、直接加入
  成員、邀請（管理者或政策為 all 時顯示，否則說明政策限制）、
  待處理邀請取消（僅管理者可見）、公告（現行／已撤下徽章、
  發佈／編輯／撤下，發佈前提示會撤下現行公告）、設定
  （invite_policy 唯讀或 Select＋儲存，僅管理者可變更）、
  編輯群組資料／轉移所有權（僅擁有者）、離開（非擁有者，確認
  對話框）／解散（僅擁有者，確認對話框）；載入／錯誤＋重試／
  不支援／就緒四態；成功／失敗 toast；樣式沿用既有
  `et-friend-list`／`et-form-grid` 等，無新增色板與 CSS。
- `src/App.tsx`：導覽新增「群組」（`MainView`＋`VIEW_TITLES`＋
  檢視切換，`dataMode` 傳入）。

驗證證據（`TZ=Asia/Taipei node tests/run.mjs`）：**104 項全過**
（§2h 的 95 項＋新增 9 項：群組清單／建立請求 shape／成員與
角色對映／邀請公告設定對映／群組清單 401 清除 Token／展示資料
群組操作與單一現行公告規則／SSR 雙模式標題與載入文字）。
`tsc --noEmit`、`eslint`、`prettier`、`vite build`、對比度與
響應式靜態稽核全過。真實瀏覽器互動與真實後端驗證仍待隔離
環境（本機無 Chromium、無後端）。

- 已知限制：成員顯示名稱在 API 模式為識別碼縮寫（後端
  `memberResponse` 無顯示名稱欄位，豐富化待後續里程碑）；
  邀請與直接加入皆以使用者識別碼輸入（無使用者搜尋端點）；
  群組內容頁無即時更新（成員／公告異動須重新載入，realtime
  stub 不動）；建立群組後不會自動開啟其對話容器（對話標題
  與群組重新命名不同步，見 §2i）；解散後清單以重新載入為準。

## 2k. 媒體模組：Phase 06 媒體（2026-11-25，後端離線可驗證部分）

範圍（ROADMAP 手冊 H 的媒體最小可驗收子集，Web 接線為後續
里程碑）：上傳會話（建立／進度回報／完成／取消／逾期標記）、
檔案詮釋資料（查詢／擁有者軟刪除）、縮圖紀錄（登記／查詢）、
存取授權的檔案側檢查＋稽核紀錄寫入。未實作：掃描實作
（僅狀態欄位＋`ReportScanResult` 服務層介面，無 HTTP 端點）、
縮圖產生（僅詮釋資料）、MinIO 轉接器（僅 `ObjectStore` 介面＋
記憶體 stub，不新增依賴）、位元組傳輸／presigned URL、壓縮／
轉碼、CDN、訊息附件參照（`message_attachments` 對接待定稿）。

實作（`services/api`，零新依賴，`go.mod`／`go.sum` 未動；
`router.go`、compose、Makefile、EXECUTION_PLAN.md、
000001–000007 皆未動；模組依賴 DAG 未違反，Media 不 import
他領域套件，只經 `UserResolver` 唯讀介面確認使用者存在）：

- `internal/media/media.go`：MediaFile／Upload／Thumbnail／
  AccessLog 領域。物件鍵一律服務端核發
  （`media/<ownerID>/<fileID>`，縮圖加 `/thumb/<variant>`），
  客戶端不得指定；上傳狀態機僅 uploading 可進度／完成／取消，
  完成時建立檔案列（掃描 pending）並關聯會話，重複完成冪等；
  進度不得倒退、不得超過已知總量；逾期標記供排程／Worker 呼叫
  （無 HTTP 端點，中止物件採 best-effort）；軟刪除為狀態標記；
  非擁有者一切查詢皆 404；授權檢查只做檔案側（存在／未刪除／
  掃描 clean）＋呼叫方傳入的 businessAllowed，擁有者不受業務
  權限限制但受掃描閘門限制，通過與否皆寫稽核紀錄、寫入失敗
  則 fail-closed；掃描終結狀態不得覆寫。記憶體 Store 供開發／測試。
- `internal/media/objectstore.go`：`ObjectStore` 介面
  （CreateUpload／AbortUpload）＋`MemoryObjectStore` stub（物件
  不持久）；MinIO 轉接器只留 TODO，不得擅自新增依賴。
- `internal/media/postgres.go`：PG 儲存庫（SQL 慣例同 M2b）；
  縮圖 (media_file_id, variant_type) 唯一衝突對映為欄位錯誤。
- `migrations/000008_media.up/down.sql`：只建四表
  （media_files、media_uploads、media_thumbnails、
  media_access_logs）；與 ERD 草案的差異（會話加 storage_key／
  reserved_file_id、整數欄 0＝未知、值域 text 無 CHECK、縮圖
  FK 取 RESTRICT、稽核雜湊欄位保留不寫入）見檔頭。
- 接線（`cmd/api/main.go`）：12 個新路由，全部經 `RequireAuth`，
  兩種儲存模式皆接線（物件儲存皆為記憶體 stub，啟動日誌明示）：
  `POST/GET /api/v1/media/uploads`、`GET .../uploads/{upload_id}`、
  `POST .../progress`、`POST .../complete`、`POST .../cancel`、
  `GET /api/v1/media/files`、`GET/DELETE .../files/{file_id}`、
  `GET/POST .../files/{file_id}/thumbnails`、
  `POST .../files/{file_id}/authorize`（僅擁有者自查：業務權限
  不得由請求傳入，非擁有者一律拒絕；跨模組授權經 Go 介面）。

驗證證據（Go 1.25.0，`GOFLAGS=-mod=readonly GOPROXY=off`）：

- `go build ./...`、`go vet ./...`（含 `-tags integration`）皆通過；
  gateway／worker 模組建置亦通過。
- `go test -count=1 ./...`：**150 項全過、0 失敗**（先前 133 項＋
  新增 17 項：media 領域 10、media PG 假查詢 6、HTTP 流程 1）；
  `go test -tags integration` 時新增的 `TestPG_MediaFlowPersists`
  跳過並標示環境待驗證。
- `gofmt -l`：新檔全數乾淨（`health.go` 既有漂移仍未動）。
- 已知限制：上傳完成時詮釋資料由呼叫方再次提供（會話列不存
  詮釋資料，見待決定 18）；檔案列先建、會話列後寫，無交易
  邊界（會話寫入失敗時殘留孤兒檔案列）；逾期中止失敗不重試；
  清單無游標分頁（沿既有清單端點慣例）；掃描未通過檔案的
  處理（通知／刪除／重掃）待定稿；縮圖分發同樣受父檔案掃描
  閘門限制（經同一授權檢查）。

## 2l. 媒體 Web 接線（2026-11-25）

範圍（§2k 的 12 個媒體端點中，檢視實際使用的 9 個；後端 Go
程式碼與 migration 皆未動，純前端里程碑）：檔案清單、上傳
會話清單與建立、會話取消、檔案刪除、縮圖紀錄查詢、檔案側
授權檢查。刻意未接線：進度回報、完成上傳、縮圖登記——三者
皆會宣稱位元組已傳輸或已產生，而後端尚無位元組上傳通道
（無 presigned URL、無位元組端點），見待決定 19；掃描器接線
與 MinIO 直傳不在範圍。

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile）：

- `src/data/types.ts`：`MediaFileItem`／`MediaUploadItem`／
  `MediaThumbnailItem`／`MediaAuthorization`／`InitiateUploadInput`
  （展示層形態；掃描／上傳／縮圖變體值域為後端實作詞彙，
  待定稿覆核）。
- `src/data/api.ts`：媒體 wire 型別與 9 個客戶端方法（路徑與
  請求／回應 shape 逐一對照 `internal/media/handler.go`，含
  `DELETE .../files/{file_id}` 與授權檢查的 `access_type` 請求體）。
- `src/data/provider.ts`：`ChatDataProvider` 新增可選非同步方法
  7 個（與 M2c／M3／M5／M6 相同的可選模式）；展示 provider 以
  fixture 實作（會話建立只記詮釋資料、取消只翻狀態、刪除只
  改本機清單、授權檢查只認 `clean` 為可存取，`pending`／
  `rejected` 一律不可存取）。
- `src/data/api-provider.ts`：wire→展示形態對映，401 沿用
  `failUnauthorized`（清除 Token、回未授權並 rethrow）。
- `src/data/demo.ts`：`DEMO_MEDIA_FILES`（三種掃描狀態各一）／
  `DEMO_MEDIA_UPLOADS`／`DEMO_MEDIA_THUMBNAILS`（虛構紀錄）。
- `src/views/MediaView.tsx`（新）：單一元件雙模式（僅資料來源
  標示不同）。建立上傳：檔案選擇器（MIME 白名單與後端一致，
  無 MIME 類型、超 100 MiB、檔名超 255 字皆先擋下；寬高／秒數
  以 0 表未知、校驗值不提供）＋常駐限制說明（僅建立會話記
  詮釋資料、位元組尚未上傳）；上傳會話清單（狀態徽章、已傳／
  總量、分段數、物件鍵，進行中可經確認對話框取消）；檔案清單
  （掃描徽章：pending／rejected 一律標示不可分發、clean 標示
  掃描通過；下載授權檢查按鈕與結果顯示；縮圖紀錄以佔位徽章
  顯示變體與尺寸並註明尚無分發位址；刪除經確認對話框）；載入／
  錯誤＋重試／不支援／就緒四態；成功／失敗 toast；樣式沿用既有
  `et-device-list`／`et-card` 等，無新增色板與 CSS。
- `src/App.tsx`：導覽新增「媒體」（`MainView`＋`VIEW_TITLES`＋
  檢視切換，`dataMode` 傳入）。

驗證證據（`TZ=Asia/Taipei node tests/run.mjs`）：**110 項全過**
（§2j 的 104 項＋新增 6 項：上傳會話建立請求 shape／媒體清單
取消刪除縮圖授權對映／檔案清單 401 清除 Token／展示資料媒體
操作與掃描閘門／SSR 雙模式標題與載入文字）。`tsc --noEmit`、
`eslint`、`prettier`、`vite build`、對比度與響應式靜態稽核全過。
真實瀏覽器互動與真實後端驗證仍待隔離環境（本機無 Chromium、
無後端）。

- 已知限制：上傳會話建立後無法繼續傳輸位元組（通道未定，見
  待決定 19）；縮圖無影像可顯示（僅詮釋資料佔位）；授權檢查
  僅支援本人自查語義（後端 `authorize` 端點的擁有者自查；跨
  模組授權經 Go 介面，不經 HTTP）；清單無分頁（沿後端清單端點
  現況）；縮圖載入單一檔案失敗時以空清單計，不拖垮整個清單。

## 2m. 訊息推播中繼（Outbox＋Relay）與 Gateway 訂閱成員驗證（2026-11-25，離線可驗證部分）

範圍（DATA_FLOW.md §3、WEBSOCKET_PROTOCOL.md §6–§7；STATUS 待決定
15 的後續）：訊息寫入時同一交易寫入 Outbox 事件（發送→
`message.received`、收回→`message.recalled`）；Relay 認領／發佈／
標記迴圈（Publisher 介面，NATS 發佈者未落地）；Gateway 訂閱與
typing 的成員身份驗證（MembershipChecker 介面，api 側查詢待
整合）。未實作：NATS JetStream 發佈者／轉接器（僅既有佔位）、
E2EE、編輯／僅自己刪除的推播事件（§21 無事件名，見待決定 20）、
WebSocket 送出訊息。

實作（`services/api`，零新依賴，`go.mod`／`go.sum` 未動；
`router.go`、compose、Makefile、EXECUTION_PLAN.md、
000001–000008 皆未動）：

- `internal/outbox`（新）：Event／Store 介面（Append／Claim／
  MarkPublished／MarkFailed）＋MemoryStore＋PGStore（Claim 以單一
  UPDATE 陳述式、`FOR UPDATE SKIP LOCKED` 搶佔＋租約逾期重認領；
  MarkFailed 達重試上限改 dead 死信）。事件只含已公開的 wire
  欄位與識別碼，不得含 Token、雜湊或 E2EE 明文。
- `internal/message`：`events.go`（新，§21 事件名常量與 payload
  建構，欄位名依 §6.1.2／§6.1.5 草案）；`AtomicWriter` 介面
  （CreateMessageWithEvent／SaveMessageWithEvent）；Service 加
  `NewServiceWithOutbox`（未接線時行為不變；已接線但儲存層
  不支援原子寫入時 fail-closed）；MemoryStore（同一互斥鎖內
  寫入訊息列與事件列、失敗補償刪除）與 PGStore
  （`NewPGStoreWithDB`，交易內寫入訊息列＋引用列＋事件列，
  序號衝突時整個交易重試至多 3 次）實作 AtomicWriter；Send
  冪等命中與重複收回不重寫事件；Edit／DeleteForSelf 不寫事件
  （註解載明 §21 缺口）。
- `internal/relay`（新）：Publication（EventID 為消費端冪等鍵）／
  Publisher 介面／Relay（RunOnce 認領→發佈→標記，單一事件
  失敗不卡整批；Run 輪詢至 ctx 取消）。
- `migrations/000009_message_outbox.up/down.sql`：只建
  `message_outbox` 一表（＋認領索引）；與 ERD 草案的差異（表名
  待定案、狀態值域、JSONB payload、FK RESTRICT）見檔頭。
- 接線（`cmd/api/main.go`）：兩種儲存模式皆接線 Outbox 儲存
  （postgres：PGStoreWithDB；memory：MemoryStoreWithOutbox）；
  Relay 迴圈不在 api 行程內啟動（行程內無 Broker 可發佈，見
  relay 套件註解）。

實作（`services/realtime-gateway`，零新依賴）：

- `internal/hub`：`MembershipChecker` 介面（未知對話與非成員一律
  回 false，不得洩漏存在性）；`NewHub` 加 checker 參數（nil 時
  訂閱與 typing 一律拒絕，fail-closed）；`handleSubscribe`（含
  重複訂閱）與 `handleTyping` 皆先驗證（checker 查詢失敗回
  `ETHER_COMMON_UNAVAILABLE`，非成員回 `ETHER_AUTH_FORBIDDEN`；
  取消訂閱不驗證）；驗證在持有訂閱鎖之前執行。
- 接線（`cmd/realtime-gateway/main.go`）：nil checker＋啟動日誌
  明示 fail-closed（真實 checker 待 api 整合，不得假裝已驗證）。

驗證證據（Go 1.25.0，`GOFLAGS=-mod=readonly GOPROXY=off`）：

- api 模組：`go build ./...`、`go vet ./...`（含 `-tags integration`）
  皆通過；`go test -count=1 ./...`：**168 項全過、0 失敗**（先前
  150 項＋新增 18 項：outbox 儲存 8、message 事件接線 5、relay 迴圈
  5）；`go test -tags integration` 時新增的 `TestPG_OutboxAtomicWrite`
  跳過並標示環境待驗證。
- gateway 模組：`go build ./...`、`go vet ./...` 皆通過；
  `go test -count=1 ./...`：**23 項全過、0 失敗**（先前 19 項＋
  新增 4 項：非成員訂閱拒絕、nil checker fail-closed、非成員
  typing 拒絕、checker 失敗 fail-closed）；worker 模組建置通過。
- `gofmt -l`：新檔全數乾淨（`health.go` 既有漂移仍未動）。
- 已知限制：Relay 無生產 Publisher（NATS 發佈者待落地與依賴授權）；
  Gateway 以 nil checker 啟動時全部訂閱被拒（開發環境亦同，待
  api 整合）；NoteMessage（對話快取更新）仍在 Outbox 交易之外
  （既有限制，見 §2g）；編輯／僅自己刪除無推播事件（§21 缺口，
  見待決定 20）；Outbox 事件 payload 含 content（E2EE 未實作，
  E2EE 落地後須改為密文）。

## 2n. 已讀回條 Web 接線（2026-11-25）

範圍（§2h 遺留的「已讀回條未接線」缺口；回條端點本身屬 M5，
後端 Go 程式碼與 migration 皆未動，純前端里程碑）：API 模式下
開啟對話時送已讀回條並更新未讀數；展示模式行為不變。`delivered`
回條、回條列表查詢、訊息氣泡的已讀標示皆不在範圍。

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile）：

- 回條接線的底層（`EtherApiClient.postReceipt`／
  `ChatDataProvider.postReceiptAsync`／api-provider 對映／展示
  provider 實作）屬 M5 既有，本里程碑未改：請求體只含
  `receipt_type`，不送 `device_id`（Web 無裝置識別；回條 device
  粒度待定稿，見待決定 14）。
- `src/views/ChatsView.tsx`：`ApiChatsView` 新增回條 effect——歷史
  載入完成（`ready`）且對話未讀數大於 0 時，以純函式
  `selectReadReceiptTarget` 挑出序號最大的他人未收回訊息，送一則
  `read` 回條（後端水位推進至該序號，一則即涵蓋其前全部訊息；
  自己的訊息後端拒寫、收回／僅自己刪除的訊息排除；身分未知時
  不送）。成功後重載對話清單取權威未讀數，不在本機扣減（App
  側邊列徽章經既有 `onUnreadChange` 同步）；失敗時靜默，不以通知
  干擾閱讀，未讀數維持後端舊值，下次開啟對話或手動「重新載入」
  時重試；401 由 provider 的 `failUnauthorized` 集中處理。去重以
  工作階段內 `對話：序號` 鍵計（`receiptedRef`）：切換對話、載入
  更早訊息、重新載入皆不重送，僅新訊息到達（新最大序號）才再送。
  展示模式 `DemoChatsView` 不動（既有 `markConversationRead` 本機
  標示已讀，行為不變）。

驗證證據（`TZ=Asia/Taipei node tests/run.mjs`）：**114 項全過**
（§2l 的 110 項＋新增 4 項：回條目標挑選——最大序號他人訊息／
排除自己、收回與僅自己刪除／空歷史與身分未知；回條請求 shape
不含 device_id；回條 401 清除 Token 並回未授權）。
`tsc --noEmit`、`eslint`、`prettier`、`vite build`、對比度與
響應式靜態稽核全過。回條 effect 本身須真實瀏覽器互動驗證
（SSR 不執行 effect），見 §3 第 17 項。

- 已知限制：無即時推播（新訊息仍須手動「重新載入」，realtime
  stub 不動）；回條失敗時無任何可見提示（刻意靜默，見上）；清單
  重載失敗時未讀數維持舊值（沿既有清單過時標示）；device 粒度
  未送（見上）；`delivered` 回條未接線（開啟對話只送 `read`）。

## 2o. 建立對話 UI（2026-11-25）

範圍（§2h 遺留的「建立對話 UI 未實作」缺口；建立端點本身屬 M5，
後端 Go 程式碼與 migration 皆未動，純前端里程碑）：展示／API 雙模式
下經好友清單建立一對一對話；群組對話建立走群組模組（GroupsView），
不在範圍。

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile）：

- 建立鏈的底層（`EtherApiClient.createConversation`／
  `ChatDataProvider.createConversationAsync`／api-provider 對映／展示
  provider 實作）屬 M5 既有，本里程碑未改其簽名：請求體只含
  `type`＋`other_user_id`（`type=direct`）；後端既有 DM 回 200＋既有
  對話、新建回 201，provider 皆對映為同一 `ConversationItem`。
  展示 provider 補去重：同一聯絡人已有對話時回傳既有項目，不另
  建重複列（對齊後端 200 語義）。
- `src/views/ChatsView.tsx`：共用 `CreateConversationDialog`（展示／
  API 雙模式同一元件）：人選一律來自好友清單（展示模式開啟對話框
  時取 `listFriendsAsync` 預設不含隱藏好友；API 模式沿用清單載入時
  一併取回的好友並排除隱藏），不得以使用者識別碼自由輸入（非好
  友建 DM 後端拒絕）；搜尋依顯示名稱／公開識別碼過濾（純函式
  `filterCreateCandidates`）；單選（listbox／option 語義、已選標示）；
  建立失敗時對話框內顯示錯誤文字並 toast，可重試；成功後父層關閉
  對話框、選取回傳對話並重載清單（API 模式取權威狀態；展示模式
  經快照訂閱重繪）。對話框開啟時重置查詢與選取；人選載入／錯誤＋
  重試／不支援／就緒四態；無好友時提示先到「聯絡人」新增好友。
  樣式新增 `et-friend-row--pick`（既有 token，無新色）。
- `src/App.tsx` 不動（ChatsView 已接 `dataMode`）。

驗證證據（`TZ=Asia/Taipei node tests/run.mjs`）：**120 項全過**
（§2n 的 114 項＋新增 6 項：建立請求 shape（僅 `type`＋
`other_user_id`）、既有對話 200 回應對映、建立 401 清除 Token 並
回未授權、展示資料既有對話去重與新增、對話框 SSR（標題／搜尋／
人選／未選取時建立按鈕停用）、人選過濾純函式）。`tsc --noEmit`、
`eslint`、`prettier`、`vite build`、對比度與響應式靜態稽核全過。
對話框的開啟、挑選與建立流程須真實瀏覽器互動驗證（SSR 不執行
effect 與點擊），見 §3 第 18 項。

- 已知限制：無即時推播（新訊息仍須手動「重新載入」，realtime
  stub 不動）；建立時不檢查對方封鎖狀態（後端以 404 回應，錯誤文
  字與 toast 呈現，見後端 CreateDirect 語義）；群組對話建立不在此
  對話框範圍（走 GroupsView 群組建立）；清單無分頁（沿後端清單端
  點現況）。

## 2p. 訊息反應與置頂（2026-11-25，後端離線可驗證部分）

範圍（Phase 05 剩餘端點；Web 接線列為後續里程碑，本里程碑不做
前端）：訊息反應（新增／更換／取消／彙總）與置頂（置頂／取消／
清單）。附件關聯與轉傳仍未實作（對應表屬 000006 檔頭所列後續
里程碑）。反應／置頂寫入不產生 Outbox 事件：WEBSOCKET_PROTOCOL.md
§21（§6）的 15 個事件名中沒有反應與置頂對應的事件名，不得自創
（缺口見待決定 21）；客戶端以 REST 查詢同步彙總與清單。

實作（`services/api`，零新依賴，`go.mod`／`go.sum` 未動；
`router.go`、compose、Makefile、EXECUTION_PLAN.md、
000001–000009 皆未動）：

- `internal/message/reactions.go`（新）：Reaction／Pin 領域型別、
  彙總聚合、Service 方法、MemoryStore 實作。實作預設（待定稿
  覆核，見待決定 21）：反應允許清單 8 種表情（👍／❤️／😂／😮／
  😢／😡／🙏／🎉）；每人每訊息同時僅一則生效反應（更換時舊列
  以 removed_at 標記，非物理刪除）；每對話同時生效置頂上限 5 則。
  守衛：非成員一切查詢 404；收回訊息不得新增反應／置頂（取消
  與列出不在此限）；作者僅自己刪除的訊息，作者本人不得新增
  反應／置頂（他人檢視不受影響）；置頂的訊息須屬於該對話；
  達上限時重送已置頂訊息為冪等成功。
- `internal/message/postgres_reactions.go`（新）：PG 儲存庫（SQL
  慣例同 M5／M6／M8；反應寫入以部分唯一索引 `ON CONFLICT DO
  NOTHING` 收斂併發重複；置頂寫入命中部分唯一索引時重讀生效列
  收斂）。`handler_reactions.go`（新）：6 個端點（全部經
  RequireAuth，DTO snake_case，未知欄位拒絕）——
  GET／POST／DELETE `/api/v1/messages/{message_id}/reactions`
  （反應彙總；POST 為 upsert，一律 200＋彙總）、
  GET／POST `/api/v1/conversations/{conversation_id}/pins`（置頂
  清單／置頂，新建 201、已置頂 200）、DELETE
  `/api/v1/conversations/{conversation_id}/pins/{message_id}`
  （取消置頂，冪等 200＋pinned=false）。
- `migrations/000010_message_reactions_pins.up/down.sql`：只建
  `message_reactions`、`message_pins` 兩表（＋部分唯一索引）；
  000006 檔頭已載明此二表屬後續里程碑，本版本補建；與 ERD 草案
  的差異（值域服務層實作、取消標記語義、上限服務層 enforced）
  見檔頭。
- 接線（`cmd/api/main.go`）：6 條路由；兩種儲存模式皆經同一
  Service（無額外接線分支）。

驗證證據（Go 1.25.0，`GOFLAGS=-mod=readonly GOPROXY=off`）：

- api 模組：`go build ./...`、`go vet ./...`（含 `-tags integration`）
  皆通過；`go test -count=1 ./...`：**180 項全過、0 失敗**（先前
  168 項＋新增 12 項：領域 5、PG 假查詢 6、HTTP 流程 1）；
  `go test -tags integration` 時新增的 `TestPG_ReactionsPinsPersist`
  跳過並標示環境待驗證（15 項整合測試全數跳過、0 失敗）；
  gateway 模組測試 23 項全過（未動）、worker 建置通過。
- `gofmt -l`：新檔全數乾淨（`health.go` 既有漂移仍未動）。
- 已知限制：反應／置頂無即時推播（§21 缺口，見待決定 21）；
  置頂上限檢查與寫入之間無序列化（併發置頂可能短暫超過上限，
  與 M5 序號重試為同類已知限制）；反應彙總排序為人數降序、
  表情升序（實作預設）。

## 2q. 訊息反應與置頂 Web 接線（2026-11-25）

範圍（§2p 後端端點的 Web 接線；後端零改動）：訊息氣泡反應列
（8 種表情挑選、彙總 chip、自己反應標示、點按切換／取消）與
置頂區塊（生效置頂清單、置頂／取消置頂操作）。反應彙總與置頂
清單皆以 REST 另查（歷史回應不含彙總，置頂回應不含內容）；
反應／置頂無即時推播（§21 缺口，見待決定 21），他人的變更須
以「重新載入」才會出現。

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile；
後端 Go 程式碼、migration、Phase 01 凍結檔、compose、Makefile、
EXECUTION_PLAN.md、router.go 皆未動）：

- `src/data/types.ts`：`ReactionCountItem`／`ReactionSummaryItem`／
  `PinItem`、`REACTION_EMOJIS`（與後端允許清單一致）。
- `src/data/api.ts`：反應／置頂 wire 型別＋6 個客戶端方法，路徑
  與請求／回應逐一對照 `handler_reactions.go`。
- `src/data/api-provider.ts`：wire→展示形態對映，401 沿用
  `failUnauthorized`。
- `src/data/provider.ts`：`ChatDataProvider` 新增 6 個可選方法；
  展示 provider 以覆蓋 map 實作，與後端一致的 headline 守衛
  （允許清單、每人每訊息一則生效反應、置頂上限 5、收回／僅
  自己刪除的訊息不得新增反應／置頂、置頂冪等）。
- `src/components/MessageBubble.tsx`：新增可選 `actions` 插槽（未
  傳入時渲染不變）。
- `src/views/ChatsView.tsx`：共用 `useReactionsAndPins` hook（彙總
  逐訊息去重載入、失敗靜默；置頂清單隨對話載入、失敗留重試；
  寫入後以伺服器回應更新本機狀態、失敗 toast；收回／僅自己
  刪除的訊息不顯示操作列）＋`MessageReactionBar`＋
  `PinnedMessagesBlock`（置頂內容取自已載入歷史，未載入者顯示
  佔位文字）；展示／API 雙模式同一元件。
- `src/styles/components.css`：反應列／chip／挑選器／置頂區塊
  樣式，沿既有 token，無新色板。

驗證證據（`TZ=Asia/Taipei`）：

- `node tests/run.mjs`：**127 項全過、0 失敗**（先前 120 項＋新增
  7 項：端點呼叫與請求 shape、彙總／置頂對映、反應 401 清除
  Token、展示資料單一生效反應與允許清單、展示資料置頂上限與
  冪等取消、反應列與置頂區塊 SSR）。
- `tsc --noEmit`、`eslint`、`prettier`、`vite build`、對比度與
  響應式靜態稽核全過。
- 已知限制：彙總為逐訊息查詢（無批次端點，失敗靜默）；置頂
  內容僅取自已載入歷史；挑選器無外部點按關閉；`delivered`
  回條與已讀標示見 §2r（本里程碑之後完成）。

## 2r. delivered 回條與回條列表 Web 接線（2026-11-25）

範圍（§2n 遺留的「`delivered` 回條、回條列表查詢、訊息氣泡
已讀標示」缺口；回條端點本身屬 M5，後端 Go 程式碼與
migration 皆未動，純前端里程碑）：API 模式下歷史載入時送
delivered 回條、查詢回條列表、自己訊息顯示已送達／已讀標示；
展示模式行為不變（展示 provider 只補回條記帳方法）。

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile；
後端 Go 程式碼、migration、Phase 01 凍結檔、compose、Makefile、
EXECUTION_PLAN.md、router.go 皆未動）：

- 回條語義以後端為準（message.go PostReceipt／ListReceipts）：
  `receipt_type` 僅 delivered／read，不送 `device_id`；自己訊息
  後端拒寫、非成員 404、重複寫入冪等；read 推進已讀水位，
  delivered 不推進；列表僅成員可讀。
- `src/data/provider.ts`：`ChatDataProvider` 新增可選方法
  `listReceiptsAsync`；展示 provider 以本機記帳實作（同一訊息
  ＋使用者＋型別只記一則，與後端冪等一致；未知訊息拋錯）。
- `src/data/api-provider.ts`：`listReceiptsAsync` 對映
  `client.listReceipts`（既有方法），401 沿用 `failUnauthorized`。
- `src/views/ChatsView.tsx`：純函式 `selectDeliveredReceiptTargets`
 （當頁全部他人未收回／未刪除訊息，逐則送 delivered；後端無
  批次端點）與 `deriveReceiptStatus`（已讀須全部其他在線成員
  皆有 read 回條，已送達須全部皆有 delivered 或 read 回條；
  成員未知或未達標時回 null、維持「已傳送」）；共用
  `useMessageReceipts` hook（送達回條工作階段內以「對話：
  訊息」去重、失敗靜默；回條列表只查自己訊息、逐訊息去重、
  失敗靜默，手動「重新載入」歷史時重查；無即時推播）；自己
  訊息的氣泡狀態依回條列表與成員清單推導覆蓋（`toChatMessage`
  預設 `sent` 不變）；展示模式 `DemoChatsView` 不動。

驗證證據（`TZ=Asia/Taipei`）：

- `node tests/run.mjs`：**133 項全過、0 失敗**（先前 127 項＋新增
  6 項：送達目標挑選純函式、回條狀態推導純函式、回條列表
  對映、列表 401 清除 Token、delivered 請求 shape 不含
  device_id、展示資料回條記帳與去重）。
- `tsc --noEmit`、`eslint`、`prettier`、`vite build`、對比度與
  響應式靜態稽核全過。回條 effect 本身須真實瀏覽器互動驗證
  （SSR 不執行 effect），見 §3 第 21 項。
- 已知限制：delivered 回條逐則送出（無批次端點，單頁至多
  HISTORY_PAGE_LIMIT 則）；回條列表逐訊息查詢（失敗靜默）；
  他人回條須手動「重新載入」才會出現（realtime stub 不動）；
  成員清單載入失敗時標示維持「已傳送」；device 粒度未送
  （見待決定 14）。

## 2s. 訊息附件關聯與轉傳（2026-11-25，後端離線可驗證部分）

範圍（M5 後續；回覆參照 message_replies 已於 M5 實作，本里程碑
不重做；Web 接線列為後續里程碑，本里程碑不動前端）：發送訊息
時關聯已完成上傳的 media_file、事後為既有訊息新增附件、列出某
訊息的附件、將訊息轉傳至另一對話（新訊息＋轉傳紀錄列）、查詢
訊息的轉傳紀錄。轉傳內容與 kind 原樣承載、發送者為轉傳者，
不複製附件位元組、不帶引用。

盤點結論：000006 只建五表（conversations、conversation_members、
messages、message_receipts、message_replies），message_attachments
與 message_forwards 屬後續里程碑未建，故以只往前的新 migration
000011 補建（檔頭載明與 ERD 差異）；000001–000010 既有 SQL 未動。
WEBSOCKET_PROTOCOL.md §21 無轉傳獨立事件名，轉傳新訊息以既有
message.received 事件承載（payload 欄位不擴充），不自創事件名。

實作（`services/api`，零新依賴；Phase 01 凍結檔、compose、
Makefile、EXECUTION_PLAN.md、router.go、.github/ 皆未動）：

- 附件（`internal/message/attachments.go`）：`SendWithAttachments`
  （發送時關聯，附件列與訊息列同一寫入路徑提交）、
  `AddAttachments`（僅作者可事後新增，upsert 語義）、
  `ListAttachments`（成員可讀）；`MediaAccess` 介面經 Media 模組
  驗證可關聯性（`media.CheckAttachable`：檔案存在、未刪除、為
  呼叫者所有、掃描 clean 四項缺一不可；查無／已刪除／非擁有者
  回 ErrMediaNotFound，pending／rejected 回欄位錯誤；唯讀不寫
  稽核紀錄）。附件詮釋資料（mime／大小／檔名）於關聯時快照；
  參照方式採直接 FK `media_file_id`，不存 storage_path（ERD §10
  缺口：不得並存兩套參照）；`thumbnail_media_id` 不存（經 Media
  模組解析）；唯一索引 `(message_id, media_file_id)` 收斂重複
  關聯；上限每訊息 10 筆；收回／僅自己刪除的訊息不得新增附件；
  非成員一切查詢 404；Media 未接線時 fail-closed（503
  ETHER_COMMON_UNAVAILABLE）。讀取訊息時不以 Message 結構攜帶
  附件（須另查）。
- 轉傳（同檔）：`Forward`（新訊息 201、冪等重送 200；轉傳者須為
  來源與目標對話的生效成員，收回／僅自己刪除的訊息不得轉傳，
  目標 DM 封鎖時 404；新訊息不帶附件、不帶引用）、`GetForward`
  （非轉傳產物時與查無／非成員同為 404）；轉傳鏈深度上限 5
  （原始訊息深度 1，逐跳計算、visited 斷環）；轉傳紀錄列寫入在
  訊息列提交之後（與 NoteMessage 同類已知限制）；Outbox 事件為
  message.received（見上）；轉傳本身不另寫事件。
- 儲存層：MemoryStore 附件／轉傳 map（createLocked 寫入附件列、
  removeLocked 刪除附件列）；PG 儲存層（`postgres_attachments.go`，
  `ON CONFLICT DO NOTHING` 收斂）與 `insertMessageRow` 同一寫入
  路徑寫入附件列；`000011_message_attachments_forwards.up/down.sql`
  只建／刪 message_attachments、message_forwards 兩表＋索引
  （FK 一律 `ON DELETE RESTRICT`，append-only，僅 created_at）。
- 端點（全部經 RequireAuth，DTO snake_case，未知欄位拒絕）：
  `POST /api/v1/messages/{message_id}/attachments`（200＋完整附件
  清單）、`GET /api/v1/messages/{message_id}/attachments`、
  `POST /api/v1/messages/{message_id}/forward`、
  `GET /api/v1/messages/{message_id}/forward`，另擴充
  `POST /api/v1/conversations/{conversation_id}/messages` 的 body
  （`attachment_file_ids`）；`cmd/api/main.go` 兩種儲存模式皆先建
  mediaSvc 再建 messageSvc，改用 `NewServiceWithDeps` 接線。

驗證證據（`TZ=Asia/Taipei`，全部實際執行）：

- `go build ./...`、`go vet ./...`（含 `-tags integration`）全過；
  `gofmt -l` 僅既有 `internal/health/health.go` 漂移（未動）；
  realtime-gateway 與 worker 建置通過；零新依賴。
- `go test -count=1 ./...`：**api 模組 200 項全過、0 失敗**
  （先前 180 項＋新增 20 項：領域守衛 13 項、media 掃描閘門 1 項、
  PG 假查詢 5 項、HTTP 流程 1 項）；`go test -tags integration`
  編譯通過，新整合測試無資料庫時 SKIP（標示環境待驗證）。
- 已知限制：轉傳紀錄列寫入在訊息列提交之後（訊息列提交後、
  紀錄列寫入前失敗時，訊息存在而無紀錄列；與 NoteMessage 同類
  限制）；併發轉傳深度檢查與寫入無序列化，併發下深度可能短暫
  超限；附件上限檢查與寫入無序列化，併發下可能短暫超限；附件
  列出無分頁（單訊息上限 10 筆）；轉傳紀錄查詢不分頁（單訊息
  至多一筆）；發送時附件解析置於封鎖檢查之後（封鎖對話的附件
  檢查不執行，不得以附件錯誤洩漏封鎖狀態以外的資訊）。

## 2t. 訊息附件與轉傳 Web 接線（2026-11-25）

範圍（§2s 後續里程碑，純前端、雙模式；後端 Go 程式碼、migration
SQL、Phase 01 凍結檔、compose、Makefile、EXECUTION_PLAN.md、
router.go、.github/ 皆未動；零新依賴）：訊息附件顯示區塊、轉傳
對話框（目標對話人選、確認、失敗重試）、附件挑選對話框（以
媒體檔案清單為來源，僅關聯既有檔案，不新增位元組上傳流程）、
發送訊息時關聯附件。附件列出的取得時機：開啟對話的訊息渲染時，
以 `conversationId:messageId` 去重鍵逐訊息查詢附件清單與轉傳紀
錄（失敗靜默、每訊息獨立，不阻塞歷史渲染）；轉傳目標人選來源：
已載入的對話清單（標題以 `resolveConversationTitle` 解析，來源
對話亦可選，後端以成員資格守衛為準）；轉傳成功後：展示模式以
訂閱快照重渲染、API 模式目標為開啟對話時重載全部，否則 toast
提示。

實作（`apps/web`，沿 §2q／§2r 接線模式）：

- 型別與客戶端（`src/data/types.ts`、`api.ts`）：`MessageAttachmentItem`
  （id／messageId／mediaFileId／mimeType／sizeBytes／fileName／
  createdAt）、`ForwardRecordItem`（id／sourceMessageId／
  forwardedMessageId／sourceConversationId／targetConversationId／
  forwardedBy／createdAt）；`EtherApiClient` 新增
  `listAttachments`、`addAttachments`（請求體
  `{media_file_ids}`）、`forwardMessage`（請求體
  `{target_conversation_id, client_message_id}`，冪等鍵由 provider
  產生）、`getForward`；`SendMessageInput` 擴充可選
  `attachment_file_ids`。
- Provider（`src/data/provider.ts`、`api-provider.ts`）：可選方法
  `listAttachmentsAsync`／`addAttachmentsAsync`／`forwardMessageAsync`／
  `getForwardAsync`，`sendMessageAsync` 選項擴充 `attachmentFileIds`；
  api-provider 對映 snake_case 欄位、401 沿用 `failUnauthorized`
  清除 Token；展示 provider 以展示資料實作對應行為：附件關聯前
  以 `demoResolveAttachable` 驗證檔案存在且掃描 clean（去重保序、
  10 筆上限、關聯時快照詮釋資料、upsert 收斂）、僅作者可事後
  新增、收回／僅自己刪除訊息拒絕、發送時先驗證可關聯性再寫入
  （與後端同一寫入路徑語義一致）；轉傳複製內容與 kind、發送者
  為轉傳者、不帶附件與引用、深度上限 5（原始訊息深度 1）、紀錄
  與深度同步寫入、`getForward` 查無紀錄時拋錯。
- 檢視（`src/views/ChatsView.tsx`，Demo／API 雙模式皆接線）：
  `MessageAttachmentBlock`（檔名／MIME／大小、轉傳標籤、新增附件
  按鈕；位元組分發未定故不設下載按鈕、明載「尚未開放下載」；
  空清單且非轉傳且不可新增時不渲染）、`ForwardDialog`（搜尋過濾
  `filterForwardTargets`、單選、錯誤 toast＋重試、成功關閉；明載
  轉傳僅複製內容、不帶附件與引用）、`AttachFilesDialog`（媒體檔
  案清單、非 clean 檔案禁用並註明掃描狀態、上限 10、確認回傳
  Promise、成功關閉）、`useMessageAttachments` hook（逐訊息附件
  與轉傳紀錄載入、去重、靜默失敗、`refreshAll`、`addAttachments`
  只更新本機狀態不發 toast，toast 由對話框負責）；反應列新增
  「轉傳」按鈕（`onForward`，未提供時不渲染）；發送時帶入待選
  附件識別碼、成功後清空待選；待選附件以 chip 列顯示於輸入區
  上方（關聯既有檔案、不重新上傳）。
- 樣式（`src/styles/components.css`）：沿既有 token（反應／置頂區塊
  慣例）新增附件區塊、轉傳標籤、待選附件 chip 樣式，未新增色板。
- 測試（`tests/`）：`api.test.mjs` 新增 6 項（附件／轉傳端點呼叫
  與請求 shape、轉傳紀錄與附件對映、發送附帶 attachment_file_ids、
  附件 401 清除 Token、展示資料附件掃描守衛與去重、展示資料
  轉傳內容複製不帶附件與深度上限）；`components.test.mjs` 新增
  4 項（附件區塊、轉傳對話框、附件挑選對話框 SSR 斷言、轉傳人
  選過濾與大小格式純函式）；`entry-components.tsx` 新增三個渲染
  入口；`tests/README.md` 補說明。

驗證證據（`TZ=Asia/Taipei`，全部實際執行，於 `apps/web`）：

- `node tests/run.mjs`：**143 項全過、0 失敗**（先前 133 項＋新增
  10 項：api 6 項、components 4 項）。
- `npx tsc --noEmit`、`npx eslint src tests eslint.config.js`、
  `npx prettier --check`（本里程碑異動檔）全過；`npx vite build`
  成功（dist 產物 404.27 kB JS／46.96 kB CSS）。
- `node tests/contrast-check.mjs` 與 `node tests/responsive-audit.mjs`
  全數 PASS（本里程碑未新增色板與斷點，沿既有 token）。
- 已知限制：附件清單與轉傳紀錄以逐訊息 REST 查詢取得，無即時
  推播（§21 無附件／轉傳事件名，他人的附件／轉傳變更須重載才
  顯示）；附件挑選僅列媒體檔案清單前 100 筆（沿媒體清單既有限
  制）；轉傳對話框僅列已載入的對話清單，未載入的對話須先開啟
  清單載入；附件新增失敗時對話框顯示錯誤並可重試，失敗原因來
  自後端錯誤封包（展示模式為展示守衛訊息）；位元組下載／分發
  未定（待決定 19），附件區塊刻意不設下載按鈕。

## 2u. 訊息編輯／收回／僅自己刪除 Web 接線（2026-11-25）

範圍（M5 後續里程碑，純前端、雙模式；後端 Go 程式碼、migration
SQL、Phase 01 凍結檔、compose、Makefile、EXECUTION_PLAN.md、
router.go、.github/ 皆未動；零新依賴）：自己訊息的操作列
（編輯／收回／僅自己刪除）、編輯輸入框、收回與刪除的確認
對話框、操作後以伺服器回應更新本機狀態、失敗 toast。收回／
僅自己刪除的訊息不提供操作。後端語義（見 handler.go 與
Service.Edit／Recall／DeleteForSelf）：僅作者可操作，非作者
一律 404；收回或僅自己刪除的訊息不可編輯（400）；收回與
僅自己刪除皆不可逆、重複操作冪等成功；編輯不寫 Outbox 事件
（§21 無事件名）、內容 1–4000 runes、無編輯次數上限、首次
編輯保留原文。

實作（`apps/web`，沿 §2q／§2r／§2t 接線模式）：

- 資料層（`src/data/api.ts`、`api-provider.ts`）：編輯（PATCH、
  請求體僅 `{content}`）、收回、僅自己刪除三端點 M5 既有，
  本里程碑確認對映未改；401 沿用 `failUnauthorized` 清除
  Token。展示 provider（`src/data/provider.ts`）補後端守衛：
  收回／僅自己刪除的訊息拒絕編輯（與後端同一錯誤文字）、
  快照 overlay（收回／刪除佔位文字與旗標、編輯時間），否則
  展示模式收回後仍顯示原文；`ChatMessage` 增可選
  `editedAt`／`recalled`／`deletedForSelf` 欄位（`toChatMessage`
  同步填寫）。
- 檢視（`src/views/ChatsView.tsx`，Demo／API 雙模式皆接線）：
  共用 `MessageOwnActions`（操作列、行內編輯框、確認對話
  框、失敗行內文字＋toast、可重試；切換訊息時重置草稿與
  確認狀態）與純函式 `validateEditContent`（不可為空、至多
  4000 runes，與後端同規則）；「已編輯」標示（API 模式取
  `editCount`、展示模式取快照 `editedAt`）。API 模式操作成功
  後以伺服器回應替換本機歷史訊息並重載對話清單（權威預覽
  與未讀數）；展示模式經快照訂閱重繪。確認對話框明載語義
  差異：收回所有成員皆不可見且無法復原；僅自己刪除只影響
  自己的檢視。
- 樣式（`src/styles/components.css`）：沿既有 token 新增操作列
  ／編輯框樣式，未新增色板。`MessageBubble` 的 `actions`
  註解同步提及 §2u。
- 測試（`tests/`）：`api.test.mjs` 新增 4 項（編輯請求 shape
  僅含 content、編輯 401 清除 Token、展示資料編輯拒絕收回／
  已刪除訊息、展示快照收回佔位顯示）；`components.test.mjs`
  新增 3 項（操作列 SSR 斷言、無支援時不渲染可操作內容、
  編輯內容檢查純函式）；`entry-components.tsx` 新增兩個渲染
  入口與 `validateEditContent` 轉出；`tests/README.md` 補說明。

驗證證據（`TZ=Asia/Taipei`，全部實際執行，於 `apps/web`）：

- `node tests/run.mjs`：**150 項全過、0 失敗**（先前 143 項＋新增
  7 項：api 4 項、components 3 項）。
- `npx tsc --noEmit`、`npx eslint src tests eslint.config.js`、
  `npx prettier --check`（本里程碑異動檔）全過；`npx vite build`
  成功（dist 產物 409.60 kB JS／48.09 kB CSS）。
- `node tests/contrast-check.mjs` 與 `node tests/responsive-audit.mjs`
  全數 PASS（本里程碑未新增色板與斷點，沿既有 token）。
- 已知限制：編輯／收回／刪除無即時推播（§21 無對應事件名，
  他人變更須重載才顯示）；編輯框不支援 Enter 送出（須按儲存
  按鈕）；收回訊息的反應／置頂清單不隨收回刷新（清單內容取
  已載入歷史，見 §2q）；僅自己刪除後附件區塊仍列於佔位文字
  下方（後端僅遮蔽內容，附件列出不在遮蔽範圍）。

## 2v. 訊息搜尋（2026-11-25，後端離線可驗證部分）

範圍（Phase 06 後續里程碑，僅訊息範圍；Web 接線列為後續；
後端 Go 程式碼僅新增 internal/search 與 cmd/api 接線，000001–
000011 既有 SQL、Phase 01 凍結檔、compose、Makefile、
EXECUTION_PLAN.md、router.go、.github/、Web 程式碼皆未動；
零新依賴、無新 migration——Search 無自有 PostgreSQL 表）：
記憶體索引、事件驅動投影器、權限過濾後的查詢服務與 HTTP
端點 `GET /api/v1/search/messages`。其他搜尋範圍（好友、使
用者、群組、社群、官方帳號、檔案、貼圖、動態）不在範圍。

架構判斷（DATA_FLOW.md §8，高於任務簡述）：索引同步是事件
驅動的投影，不得由業務寫入路徑同步寫索引——本里程碑不掛鉤
訊息寫入路徑（訊息寫入程式碼零改動），投影由 Projector 消費
Outbox 事件（`message.received`／`message.recalled`，M8 既有）
完成；填充迴圈（Worker 經 NATS 消費）與 Meilisearch 轉接器落
地屬後續里程碑，本行程不啟動任何填充迴圈，索引為空時查詢
回傳空結果，不得謊稱結果完整。

實作（`services/api`，沿 §2p／§2s／M8 模式）：

- 領域（`internal/search/search.go`）：`Document` 投影文件（僅
  訊息識別碼、對話識別碼、發送者、kind、內容、序號、建立
  時間；E2EE 未實作前內容為不透明字串）；`Index` 介面
  （Upsert／Remove／Search／ReplaceAll，Upsert／Remove 依
  eventID 冪等、Upsert 依 SequenceNumber 擋舊版本）；
  `MemoryIndex`（行程內索引，全量掃描後過濾，僅開發與測試）；
  `MeilisearchIndex` 佔位（任何呼叫皆回 `ErrNotConfigured`，
  不得假裝已連接）；`Projector`（`ApplyReceived`／
  `ApplyRecalled`／`Rebuild`，重建文件來源須為 PostgreSQL
  全量掃描，由呼叫方提供）；`Service`（查詢＋權限過濾＋游標
  分頁，checker 為 nil 或檢查失敗時 fail-closed 回
  `ErrIndexUnavailable`）。
- 比對語義（實作預設，待定稿覆核）：查詢字串去空白、小寫、
  依空白切分，AND 語義、子字串比對（含 CJK）、停用詞不預設；
  排序為建立時間降序、訊息識別碼降序 tiebreak；游標為上一頁
  最末文件的排序鍵（不透明 base64）；篩選含對話、發送者、
  建立時間區間（含邊界）；媒體／檔案類型篩選未實作（kind
  值域目前僅 text，附件類型篩選須待事件 payload 擴充）。
- 權限過濾（DATA_FLOW.md §8）：結果回傳前逐筆經
  `MembershipChecker`（main.go 以 chat.Service 接線）驗證生效
  成員身份，非成員對話的文件直接略過、不報錯；分頁在過濾
  後執行，不得以分頁跳過過濾。
- 端點（`internal/search/handler.go`，全部經 RequireAuth）：
  `GET /api/v1/search/messages?q=...&conversation_id=...&
  sender_id=...&since=...&until=...&limit=...&cursor=...`（參數
  一律 snake_case；q 必填、至多 200 runes；limit 預設 20、
  上限 50）；回應為 `{results, next_cursor?}`（wire 欄位與
  訊息回應一致）；索引不可用時回 503
  `ETHER_COMMON_UNAVAILABLE`，不得回空結果假裝完整。
- 接線（`cmd/api/main.go`，兩模式）：記憶體索引＋chatSvc 檢查
  器，啟動日誌明示「重啟後索引為空」；`main_test.go` 測試
  伺服器同樣接線，新增 `TestSearchMessages_Wiring`（空索引回
  200＋空結果、缺 q 回 400）。
- 已知缺口（與 §21 事件缺口同源）：編輯無對應事件名，編輯後
  內容不經事件更新索引（須待事件協定定稿或全量重建）；僅自己
  刪除無事件，不影響他人索引（符合預期）；收回事件投影為索
  引刪除，內容不得殘留。

驗證證據（全部實際執行，於 `services/api`）：

- `go build ./...`、`go vet ./...`（含 `-tags integration`）全過；
  gateway／worker 建置通過；`gofmt` 新檔全數乾淨（`health.go`
  既有漂移未動）。
- `go test -count=1 ./...`：**214 項全過、0 失敗**（先前 200 項
  ＋新增 14 項：索引 5、投影器 1、服務過濾／分頁／驗證 4、
  handler 3、接線流程 1）。
- `go test -tags integration ./...`：0 失敗，16 項整合測試全數
  SKIP 並標示環境待驗證（Search 無自有表、無外部依賴，故無
  新增 integration 測試；Meilisearch 實機驗證待轉接器落地）。
- 已知限制：MemoryIndex 全量掃描後過濾（Meilisearch 落地時須
  改為候選頁＋補取）；事件冪等集合無界增長（重建後集合重置，
  重建期間重放舊事件可能重複套用，呼叫方須保證消費端暫停或
  事件順序）；索引填充迴圈未接線（api 行程內索引平時為空，
  僅測試與未來重建／消費迴圈填充）；`since`／`until` 含邊界、
  limit 預設／上限、游標編碼皆為實作預設，待定稿覆核。

## 2w. 訊息搜尋 Web 接線（2026-11-25）

範圍（§2v 後續里程碑，純前端；後端 Go 程式碼、migration
SQL、Phase 01 凍結檔、compose、Makefile、EXECUTION_PLAN.md、
router.go、.github/ 皆未動；零新依賴、package.json 不動、無
lockfile）：對話檢視內新增搜尋面板（關鍵字、對話／發送者／
日期區間篩選、結果清單、游標載入更多），展示／API 雙模式；
點選結果跳至對話並醒目提示該訊息。

實作（`apps/web`，沿 §2t／§2u 接線模式）：

- 查詢語義以後端 §2v 為準：`GET /api/v1/search/messages`
  snake_case 參數、q 必填、limit 預設 20／上限 50、游標為
  排序鍵 base64（前後端同一格式，`logic.ts` 手寫 base64url
  編解碼，與後端 `cursorOf` 一致）；索引不可用 503 沿既有
  `ApiError` 拋出、面板顯示失敗重試，不得回空結果假裝完整；
  空索引回 200＋空結果，API 模式面板誠實標示「索引填充迴圈
  尚未接線，結果可能不完整」。
- 接線（`api.ts`／`api-provider.ts`／`provider.ts`）：client
  `searchMessages` 組參數（空值不送）；provider 可選方法
  `searchMessagesAsync`（註解載明展示／API 語義與空索引誠實
  原則）；api-provider 對映 wire→展示形態、401 沿用
  `failUnauthorized`（清 Token＋unauthorized 狀態）；展示
  provider 本機比對（AND、子字串、大小寫不敏感、建立時間
  降序＋識別碼降序、游標分頁、收回與僅自己刪除排除、篩選
  含邊界、空關鍵字拒絕，與後端 q 必填一致）。
- UI（`ChatsView.tsx`）：共用 `MessageSearchPanel`（idle／
  loading／error／ready 四態；表單驗證：空關鍵字與結束早於
  起始不送出；失敗留重試；結果列含對話標題、發送者標籤、
  時間、內容片段）；`ApiChatsView` 與 `DemoChatsView` 皆以面板
  取代對話列表區（「搜尋訊息」切換按鈕、「返回對話列表」）；
  跳轉＝開啟對話＋醒目提示（`et-message--jump-target` 外框）
  ＋捲入視野；API 模式以 `sinceSeq` 錨定載入該訊息前後一頁
  再合併（錨定失敗時 toast、對話仍以最新歷史開啟）；展示模式
  僅開啟＋醒目（快照已含全部訊息）；手動切換對話清除醒目。
- 樣式（`components.css`）：沿既有 token，不新增色板；結果
  內容兩行截斷；面板欄內捲動、無固定寬度。
- 測試說明：`tests/README.md` 補搜尋段；`contrast-check.mjs`／
  `responsive-audit.mjs` 納入搜尋面板配對與契約。

驗證證據（全部實際執行，於 `apps/web`）：

- `TZ=Asia/Taipei node tests/run.mjs`：**164 項全過、0 失敗**
  （先前 150 項＋新增 14 項：api 對映／503／401／展示語義
  7、logic 純函式 5、元件 SSR 2）。
- `npx tsc --noEmit`、`npx eslint src tests`、`npx prettier
  --check src tests` 全過；`npm run build`（tsc＋vite）成功，
  78 modules。
- `node tests/contrast-check.mjs`：22 配對全過（含搜尋面板
  tertiary／accent 於 bg-surface：深色 5.81:1／6.91:1、淺色
  5.49:1／5.62:1）；`node tests/responsive-audit.mjs`：16 檢查
  全過（含搜尋面板 4 項契約）。
- 已知限制：面板互動（送出、載入更多、跳轉捲動）與 API 模式
  錨定載入未在真實瀏覽器執行；索引填充迴圈未接線，API 模式
  結果可能不完整（面板明載）；編輯後內容不更新索引（§2v 事件
  缺口同源）；游標為排序鍵，跨頁期間新訊息寫入可能致重複或
  遺漏（與後端分頁語義一致，待定稿覆核）。

## 2x. 對話成員管理 Web 接線（2026-11-25）

範圍（§2h／§2o 後續里程碑，純前端；後端 Go 程式碼、migration
SQL、Phase 01 凍結檔、compose、Makefile、EXECUTION_PLAN.md、
router.go、.github/ 皆未動；零新依賴、package.json 不動、無
lockfile）：對話檢視新增成員區塊（成員清單、新增成員、離開對
話），展示／API 雙模式。

實作（`apps/web`，沿 §2o／§2t／§2u 接線模式）：

- 語義以後端 §2h 為準：ListMembers／AddMembers／Leave 三端點；
  新增僅群組對話且僅管理員（一對一對話回 400 欄位錯誤、非管理
  員回 404、非成員一切查詢 404）；離開標記 left_at、對話與訊息
  保留；一對一對話離開後可經建立對話重新開啟（CreateDirect 重新
  加入），群組對話須由管理員重新新增。
- 接線（`api.ts` 既有 client 方法／`api-provider.ts`／
  `provider.ts`）：provider 新增可選方法 `addMembersAsync`（註解
  載明守衛）；api-provider 對映 wire→展示形態、401 沿用
  `failUnauthorized`（清 Token＋unauthorized 狀態）；展示
  provider 守衛與後端一致（展示對話皆為一對一，新增一律拒絕
  並以同一錯誤文字回應；離開自快照移除對話）。
- UI（`ChatsView.tsx`）：共用 `useConversationMembers` hook（切
  換對話重置、失敗退回空清單不阻擋歷史、reload 供新增後重查）；
  共用 `ConversationMembersBlock`（生效／已離開成員、角色標示、
  新增與離開入口）；共用 `AddMembersDialog`（好友人選多選、搜尋
  過濾、失敗重試）；共用 `LeaveConversationDialog`（依對話類型明
  載離開後果）；`ApiChatsView` 與 `DemoChatsView` 皆接線。
- 樣式（`components.css`）：沿既有 token，不新增色板；成員列可
  換行、無固定寬度。
- 測試說明：`tests/README.md` 補成員管理段；
  `responsive-audit.mjs` 納入成員區塊契約（對比度沿既有文字配
  對，未新增配對）。

驗證證據（全部實際執行，於 `apps/web`）：

- `TZ=Asia/Taipei node tests/run.mjs`：**173 項全過、0 失敗**
  （先前 164 項＋新增 9 項：api 對映／401／展示守衛 5、元件
  SSR 4）。
- `npx tsc --noEmit`、`npx eslint src tests`、`npx prettier
  --check src tests` 全過；`npm run build`（tsc＋vite）成功。
- `node tests/contrast-check.mjs`：22 配對全過（未新增配對）；
  `node tests/responsive-audit.mjs`：17 檢查全過（含成員區塊契
  約）。
- 已知限制：對話框互動（新增挑選／確認、離開確認、失敗重試）
  與新增後成員重查未在真實瀏覽器執行；展示模式無群組對話，
  新增成員一律以一對一守衛錯誤回應（誠實顯示）；移除他人成員
  無對應後端端點，UI 不提供；成員清單無即時推播（§21 無對應
  事件名）。

## 2y. NATS 推播發佈者＋Meilisearch 轉接器＋索引填充迴圈（Track A，2026-11-25，離線可驗證部分）

- 範圍：JetStream 發佈者（`internal/relay/nats.go`，`Publisher`
  的生產實作）、Relay 發佈迴圈的 api 行程接線（環境開關
  `RELAY_ENABLED`，預設關閉）、Meilisearch 轉接器
  （`internal/search/meilisearch.go`，取代先前佔位）、NATS 訂閱
  消費迴圈（`internal/search/consumer.go`＋`nats_consumer.go`，
  訂閱→Projector→Index）、重建入口
  （`Projector.RebuildFromScanner`，掃描來源介面已定、PG 掃描實
  作待後續）、搜尋索引後端選擇（環境變數 `SEARCH_INDEX`：
  memory 預設／meilisearch，後者缺位址或金鑰時啟動明確報錯、
  不得靜默退回記憶體）、索引消費迴圈開關
  （`SEARCH_CONSUMER_ENABLED`，預設關閉）。Gateway 的
  JetStreamBroker 佔位未動（屬另一模組的扇出里程碑）。
- 新依賴（皆固定版本，`go mod tidy` 後 `go.mod`／`go.sum`
  一致；go directive 經 tidy 正規化為 `go 1.25.0`，與
  Dockerfile 的 golang:1.25.0 一致）：
  `github.com/nats-io/nats.go v1.52.0`（go 1.25 可建置的最新
  版；v1.53＋要求 go 1.26，與釘選的 go 1.25 衝突，不得升級）、
  `github.com/meilisearch/meilisearch-go v0.37.0`（及其間接依賴
  nkeys／nuid／x/crypto v0.49.0／x/sys v0.42.0）。
- 線路格式（NATS 承載，JSON，消費端解碼結構欄位名須一致）：
  `{"event_id","event","topic","data"}`；subject 為
  `ether.events.`＋Topic（例
  `ether.events.conversation.<id>`），Stream 名
  `ETHER_EVENTS`（由部署方預先建立、subject 含
  `ether.events.>`，發佈者不自動建 Stream）；EventID 同時寫入
  Msg-Id 標頭供 JetStream 去重。消費訂閱主題
  `ether.events.>`、durable／queue group 皆
  `ether-search-indexer`、ManualAck、AckWait 30 秒、MaxDeliver
  5 次；未知事件 Ack 略過，解碼／投影失敗 Nak（重投上限後不再
  投遞，經 ErrHandler 回報）。
- Meilisearch 轉接器實作預設（待定稿覆核）：索引 UID
  `messages`、主鍵 `message_id`；文件欄位 message_id／
  conversation_id／sender_id／kind／content／sequence_number／
  created_at（RFC3339Nano UTC 字串）；篩選 conversation_id／
  sender_id／created_at 區間、排序 created_at 降序＋message_id
  降序；寫入經 WaitForTask 確認 succeeded（100ms 輪詢）；冪等
  為行程內 seen 集合（寫入成功後標記，重建後重置）＋
  SequenceNumber 版本守衛。索引設定（filterable／sortable
  屬性）由部署方預先建立，轉接器不寫入設定。
- 架構判斷（已記錄， parent 彙整時注意）：任務簡述要求 Relay
  迴圈與消費迴圈接線於 worker 行程，但 worker 為獨立 Go
  module、無法 import api 的 internal 套件（Outbox／Projector
  邏輯若重複實作將違反一表一主與模組邊界）；本軌道改在 api
  模組內落地、api 行程依環境開關承載，Worker 行程接線列為
  後續重構事項（須抽共用套件或合併模組，屬架構決策）。未繞
  過 NATS 以 Outbox 輪詢直寫索引（DATA_FLOW.md §8 紅線）。
- 驗證（實際執行，go1.25.0 工具鏈、GOTOOLCHAIN=local）：
  `go build ./...`、`go vet ./...`（含 `-tags integration`）通
  過；gateway／worker 建置通過；`go test ./... -count=1`：**255
  項全過、0 失敗**（先前 214 項＋Track B 增量＋本軌道新增 24
  項：發佈者 7、轉接器 8、消費迴圈 9；含假 JetStream、假
  Meilisearch HTTP 伺服器、假投遞通道測試）；`go test -tags
  integration`：3 項新增整合測試全數 SKIP（無
  ETHER_TEST_NATS_URL／ETHER_TEST_MEILISEARCH_URL 時標示環境
  待驗證）；gofmt 僅剩既有 health.go 漂移（未動）。
- 五種狀態：已實作（上述全部）；已通過本機測試（255 項）；
  已通過整合測試：無；已通過實機驗證：無；因環境限制尚未驗
  證：真實 NATS／Meilisearch 整合測試（Stream 建立、發佈→訂
  閱端到端、轉接器寫入／查詢／刪除、消費迴圈填充）、Relay 與
  消費迴圈的 api 行程實機啟動、索引設定預建、PG 全量掃描重
  建，皆待隔離環境。
- 已知限制：發佈者不自動建 Stream（Stream 缺失時發佈失敗、
  Relay 重試）；消費端毒訊息超過 MaxDeliver 後不再投遞（須以
  日誌／監控發現）；Meilisearch seen 集合為行程內（重啟後重
  建前重放舊事件可能重複套用，以版本守衛收斂）；Search 分頁
  抓取以 1000 筆為頁（全量掃描後過濾的既有限制不變）；重建
  期間消費端須暫停或保證事件順序；`go.mod` 的 go directive
  經 tidy 為 `go 1.25.0`（原 `go 1.25`，語義相同）。
- 待決定（記入 §3 者由 parent 彙整，本段僅列）：Stream 名稱
  與 subject 前綴、消費端 durable／AckWait／MaxDeliver、索引
  UID 與文件欄位、索引設定預建方式、PG 掃描實作與重建 CLI
  落點、Worker 行程接線重構方案、Relay／消費迴圈的部署預設
  開關值。

## 2z. 媒體位元組上傳通道：MinIO presigned 直傳（2026-11-25，後端離線可驗證部分）

範圍（Track B，雙軌進行；Web 接線為後續里程碑）：MinIO
presigned 直傳——會話建立核發 presigned PUT URL（客戶端直傳位元
組至物件儲存，不經 api 中繼）、完成端點以 Stat 校驗物件存在與
大小、授權端點核發 presigned GET URL（掃描閘門 fail-closed）。
此決定與已核准架構一致（DATA_FLOW.md §5 簽名 URL → MinIO →
縮圖 → CDN），記為實作決定、待定稿覆核。掃描實作、縮圖產生、
CDN 分發仍未實作。

實作（`services/api`，零新依賴，`go.mod`／`go.sum` 未動；
`router.go`、compose、Makefile、EXECUTION_PLAN.md、
000001–000011 皆未動，無新 migration——上傳 URL 為暫存欄位，
不持久化）：

- `internal/media/objectstore.go`：`ObjectStore` 介面擴充 `Stat`；
  新增 `Presigner`（PresignPut／PresignGet）、`Statter`、
  `ByteStore`（行程內位元組儲存，僅記憶體 stub 實作）介面；
  `MemoryObjectStore` 承載行程內位元組（Stat／PutObject／
  GetObject，中止時清除位元組），刻意不實作 `Presigner`。
- `internal/media/minio.go`（新）：`MinIOStore`，標準庫 SigV4
  簽名（path-style、區域固定 us-east-1、presigned canonical
  request 以 UNSIGNED-PAYLOAD 計入、header 簽名供 HEAD／DELETE）；
  CreateUpload 回傳固定標記（單次 PUT 無伺服器端狀態）；
  AbortUpload 以 DELETE 清理（404 視為成功）；Stat 以 HEAD 校驗；
  PresignPut／PresignGet 核發短效 URL（有效期夾取至 7 天上限）。
  憑證一律由建構參數傳入，不得寫入程式碼。
- `internal/media/media.go`：`Upload` 新增暫存欄位
  `UploadURL`／`UploadURLExpiresAt`（不持久化，重新查詢須經
  IssueUploadURL 重新核發）；`UploadURLTTL`／`DownloadURLTTL`
  皆 15 分鐘（實作預設）；`InitiateUpload` 先核發上傳 URL 再建
  會話（核發失敗不建會話，fail-closed）；`CompleteUpload` 以
  Stat 校驗物件存在且大小與回報進度一致（無位元組、大小不符
  皆拒絕；物件儲存未接線回 503）；新增 `IssueUploadURL`（僅
  uploading 可重新核發）、`IssueDownloadURL`（經 AuthorizeAccess
  寫入稽核紀錄，掃描閘門適用；未通過不核發 URL；非擁有者掃描
  狀態以 pending 佔位）、`StoreRelayBytes`／`LoadRelayBytes`
  （中繼位元組寫入／讀出，僅 ByteStore 模式）。
- `internal/media/handler.go`：回應新增 `upload_url`／
  `upload_url_expires_at`（會話）與 `download_url`／
  `download_url_expires_at`（授權）；新增三端點，全部經
  `RequireAuth`：`POST /api/v1/media/uploads/{upload_id}/upload-url`
  （重新核發）、`PUT /api/v1/media/relay/uploads/{upload_id}`
  （中繼位元組寫入，超過上限回 413
  `ETHER_COMMON_PAYLOAD_TOO_LARGE`）、`GET
  /api/v1/media/relay/files/{file_id}`（中繼位元組分發，未通過
  回 403、查無回 404）；中繼端點僅記憶體 stub 模式可用，
  MinIO 模式回 503（位元組走 presigned URL，不得經 api 中繼）。
- 接線（`cmd/api/main.go`）：新增 `buildObjectStore`——
  `MINIO_BUCKET`／`MINIO_ACCESS_KEY`／`MINIO_SECRET_KEY` 三者
  齊備時以 MinIO 轉接器承載，否則記憶體 stub 並以啟動日誌明示；
  兩種儲存模式皆經同一選擇；三個新路由已註冊。

驗證證據（Go 1.25.0，`GOFLAGS=-mod=readonly GOPROXY=off`）：

- `go build ./...`、`go vet ./...`（含 `-tags integration`）皆通過；
  gateway／worker 模組建置亦通過。
- `go test -count=1 ./...`：**231 項全過、0 失敗**（先前 214 項＋
  新增 17 項：MinIO 簽名／Stat／Abort 6、服務層 URL 核發與
  Stat 校驗 8、handler 中繼流程與守衛 3）；`go test -tags
  integration` 時 17 項整合測試全數跳過並標示環境待驗證（含
  新增的 `TestLiveMinIO_PresignedRoundTrip`）。
- `gofmt -l`：新檔全數乾淨（`health.go` 既有漂移仍未動）。
- 已更新的既有測試：`TestMedia_Flow` 與附件流程改經中繼端點
  上傳位元組後完成（舊的「無位元組完成」語義已由 Stat 校驗
  取代）；`media_test.go` 的 `complete` 輔助、`TestPG_MediaFlowPersists`
  與附件整合測試皆改為先寫入位元組再完成。

已知限制：中繼端點位元組不持久（行程重啟遺失；正式環境須用
MinIO 模式）；重新查詢會話不含上傳 URL（須重新核發）；完成時
檔案列先建、會話列後寫，無交易邊界（沿 §2k 既有限制）；分段
上傳／續傳未實作（單次 PUT 語義，大檔續傳待定稿）；縮圖分發
URL 未核發（縮圖登記僅詮釋資料）；MinIO 模式的 Stat／Abort
網路錯誤一律對映為物件儲存不可用（503），未區分逾時與拒絕；
presigned URL 的 CDN 前綴未接線（DATA_FLOW §5 的 Cloudflare
Free CDN 分發待部署定案，現核發 MinIO 直連 URL）。

待決定（本里程碑決定，待 API／ERD／部署文件定稿覆核）：
presigned 直傳通道（本段落決定）；URL 有效期 15 分鐘；簽名區域
us-east-1；單次 PUT（非分段）；中繼端點僅開發／測試用；記憶體
模式下載 URL 為相對路徑；`upload_url_expires_at`／
`download_url_expires_at` 欄位名；413 錯誤碼
`ETHER_COMMON_PAYLOAD_TOO_LARGE`；中繼下載未通過回 403。
MinIO 實機驗證（presigned 往返、Stat、Abort、雙帳號下載流程、
migration 不涉及）與 Web 接線實機驗證待隔離環境（Web 接線見
§2aa）。

## 2aa. 媒體位元組上傳通道 Web 接線（2026-11-25）

範圍（§2z 的四個取用點，後端 Go 程式碼與 migration 皆未動，
純前端里程碑）：MediaView 上傳流程（選檔→建會話→PUT 位元組→
回報進度→完成→狀態更新，含進度顯示與失敗重試）、附件區塊
下載按鈕（經授權解析後分發）、展示模式本機位元組流程。

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile）：

- `src/data/api.ts`：wire 型別補 `upload_url`／
  `upload_url_expires_at`（會話）與 `download_url`／
  `download_url_expires_at`（授權）；新增 `issueUploadUrl`／
  `reportUploadProgress`／`completeUpload`（完成請求體不含
  `total_bytes`，後端 `DisallowUnknownFields` 會拒絕）／
  `putUploadBytes`／`fetchDownloadBytes`。位元組 PUT 分流：
  絕對 URL（MinIO presigned）原樣 fetch、不帶 Authorization
  標頭（簽名僅含 host 標頭，bearer 不得送往物件儲存）；相對
  URL（記憶體中繼）以 API baseUrl 組裝並帶 Authorization。
- `src/data/provider.ts`：`ChatDataProvider` 新增可選方法 5 個
  （核發／PUT 位元組／進度回報／完成／下載解析）；展示
  provider 以記憶體位元組儲存實作本機流程（PUT 大小須等於
  宣告總量、完成須有位元組且進度等於總量，fail-closed，與
  後端 Stat 校驗語義一致；完成檔案掃描 pending；fixture 無
  位元組者下載解析為 unavailable，不得假裝可下載）。
- `src/data/api-provider.ts`：wire→展示形態對映，401 沿用
  `failUnauthorized`；下載解析四態：未通過→forbidden、已授
  權無 URL→unavailable、絕對 URL→ready-url（檢視新分頁開
  啟）、相對 URL→經認證 fetch 取位元組→ready-blob。
- `src/views/MediaView.tsx`：上傳流程三階段文字（位元組／進
  度／完成）、PUT 403 時重新核發一次並重試、失敗保留會話與
  檔案並提供「重試上傳」、完成後 toast「等待掃描」並重載；
  「下載」按鈕經解析後分發（新分頁或 Blob 儲存），已授權無
  位址與掃描未通過皆 toast 說明；舊「僅記錄詮釋資料／位元
  組尚未上傳」文字已同步修正。
- `src/views/ChatsView.tsx`：`MessageAttachmentBlock` 新增下載
  按鈕（`downloadSupported` 時顯示，否則註記不提供下載）與
  共用 `useMediaDownload` hook（雙檢視共用，授權解析→新分
  頁／Blob 儲存→失敗 toast）；舊「尚未開放下載」文字已移除。
- `src/views/downloadFile.ts`（新）：新分頁開啟與 Blob 儲存
  兩個瀏覽器輔助函式，僅由點按處理呼叫。
- `eslint.config.js`：tests globals 補 `Blob`（Node 內建全域，
  與既有 `Response`／`URL` 同列）。

驗證證據（`TZ=Asia/Taipei node tests/run.mjs`）：**184 項全過**
（§2x 的 173 項＋新增 11 項：api 對映／PUT 分流／403／完成
請求 shape／下載解析四態／401／展示位元組流程與守衛 9、
元件 SSR 下載按鈕／不支援註記 2）。`tsc --noEmit`、`eslint`、
`prettier`、`vite build`、對比度與響應式靜態稽核全過。真實
瀏覽器互動與真實後端／MinIO 驗證仍待隔離環境。

- 已知限制：上傳進度為階段文字（fetch 無上傳進度事件，
  未用 XHR）；單次 PUT（無分段／續傳，沿 §2z）；PUT 失敗重
  試僅 403 重新核發一次；中繼下載整檔載入記憶體後儲存
  （大檔待分段／串流定稿）；presigned 下載以新分頁開啟，
  須物件儲存 CORS 允許（部署設定待定）；展示模式完成檔案
  掃描恆 pending，下載解析恆 forbidden（掃描未實作）；舊
  `kind: 'attachment'` 展示訊息氣泡的下載提示維持原樣（無媒
  體檔案背書，非本里程碑範圍）。
- 待決定（本里程碑決定，待定稿覆核）：單次 PUT 分段數回報
  為 1；403 僅重試一次；絕對 URL 新分頁開啟、相對 URL 認證
  fetch 取位元組；展示 pseudo-URL（`demo://`）語義。

## 2ab. 社群動態（Timeline）後端（2026-11-25，Phase 08 離線可驗證部分）

範圍（僅貼文／留言／按讚；Web 接線列後續里程碑）：貼文建立／
查詢（個人時間線與好友時間線）／編輯／刪除、留言建立／列出／
刪除（含回覆）、按讚／取消按讚與計數、貼文媒體關聯（建立時
關聯，經 Media 掃描閘門）。分享／轉發動態、收藏、檢舉／審核、
置頂動態、限時動態、隱藏、商業功能一律未實作，見待決定 29。

實作（`services/api`，零新依賴，`go.mod`／`go.sum` 未動）：

- `internal/timeline/timeline.go`：領域＋記憶體儲存＋服務。能見度
  public／friends／only_me（指定好友可見屬後續）；任一方向封鎖
  一律 404；刪除為軟刪除標記（貼文與留言皆然）；計數一律即時
  彙總（按讚計生效列、留言計未刪除列），不得以快取代替；游標
  分頁以 (created_at, id) 倒序鍵（貼文）／升序鍵（留言）編碼；
  時間線採「掃描候選後以可見性過濾」作法（Meilisearch 落地前
  的已知限制）；留言不是訊息，不得寫入 messages。只經唯讀介面
  讀取 User／Friend／Media，不直讀他模組資料表。
- `internal/timeline/postgres.go`：PG 儲存庫（元組比較游標分頁、
  按讚 upsert、零列更新回 404）。
- `internal/timeline/handler.go`：12 個端點，全部經 RequireAuth
  （`/api/v1/posts` 建立／作者清單、`/api/v1/timeline` 好友
  時間線、單則讀取／編輯／刪除、媒體關聯列出、按讚／取消、
  留言建立／列出、留言刪除）。動態無對應 WebSocket 事件名
  （§21 的 15 個事件皆為訊息事件），不寫 Outbox 事件，客戶端
  以 REST 查詢同步。
- `migrations/000012_timeline.up/down.sql`：只建 posts、post_media、
  post_likes、post_comments 四表（000001–000011 未動；檔頭載明
  與 ERD §6.8 草案的八點差異）。
- `cmd/api/main.go`：兩模式接線（PostgreSQL／記憶體儲存皆以
  friendSvc＋mediaSvc 注入唯讀介面）。

驗證證據（Go 1.25.0，專案樹外工具鏈；`GOFLAGS=-mod=readonly GOPROXY=off`）：

- `go build ./...`、`go vet ./...`（含 `-tags integration`）通過；
  gateway／worker 模組建置與測試通過；`gofmt` 新檔乾淨
  （`health.go` 既有漂移未動）。
- `go test ./... -count=1`：**269 項全過、0 失敗**（先前 255 項＋
  新增 14 項：領域守衛 8、PG 假查詢 5、HTTP 流程 1，含三帳號
  好友／陌生人時間線流程）。`go test -tags integration`：0 失敗，
  21 項整合測試全數 SKIP 標示環境待驗證（含新增的
  TestPG_TimelineFlowPersists）。

- 已知限制：時間線掃描後過濾（候選頁內被濾除者多時須多輪
  掃描）；貼文列與媒體關聯列分步寫入（跨模組無交易邊界，
  媒體列寫入失敗時殘留無媒體貼文列）；按讚／留言計數為逐項
  查詢（清單 N+1）；併發按讚／留言寫入無序列化（唯一約束與
  標記收斂）；貼文媒體詮釋資料不快照（讀取時以檔案列為準，
  與 M10 附件快照語義不同）；留言編輯未實作（edited_at 欄位
  已建、端點未提供）；父留言刪除後子留言保留（顯示規則待定稿）。
- 待決定（本里程碑決定，待定稿覆核，見 §3 待決定 29）：能見度
  值域與預設 friends、內容上限（貼文 2000／留言 1000 runes）、
  媒體上限 9 筆、分頁預設與上限、游標格式、編輯歷史不保留、
  分享／收藏／檢舉／置頂／限時動態範圍、動態事件名缺口。

## 2ac. 社群動態（Timeline）Web 接線（2026-11-25）

範圍：時間線清單（好友動態／我的貼文雙頁籤、游標載入更多）、
貼文建立（內容＋能見度選擇＋媒體關聯挑選）、貼文編輯／刪除
（僅作者）、留言列出／建立／回覆／刪除、按讚／取消按讚與計數
顯示、四態與失敗 toast、資料來源誠實標示。分享／轉發動態、
收藏、檢舉／審核、置頂動態、限時動態、隱藏、商業功能一律未
實作（見待決定 29）；留言編輯未實作（後端無端點）。

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile）：

- `src/data/types.ts`：`PostItem`／`PostPage`／`PostCommentItem`／
  `PostCommentPage`／`PostMediaItem`／`PostVisibility`／
  `CreatePostInput`／`UpdatePostInput`／`PostLikeResult`。
- `src/data/demo.ts`：`DEMO_POSTS`（public／friends／only_me 各一、
  封鎖作者的公開貼文一則）／`DEMO_POST_LIKED_BY`／
  `DEMO_POST_COMMENTS`（含回覆與一則已刪除留言）。
- `src/data/api.ts`：12 個端點的 wire 型別與客戶端方法
  （`GET /api/v1/posts` 作者清單、`GET /api/v1/timeline` 好友
  時間線、建立／讀取／編輯／刪除、媒體關聯列出、按讚／取消、
  留言建立／列出、留言刪除；游標與 limit 以查詢參數傳遞）。
- `src/data/provider.ts`：11 個可選方法＋展示實作（能見度過濾
  與後端 canView 同規則：作者恆可見、封鎖一律不可見、friends
  僅好友、only_me 僅作者；軟刪除以集合標記、讀取皆以不存在
  回應；計數即時彙總；游標為 (created_at, id) base64url、貼文
  倒序／留言升序；建立／編輯／留言上限與媒體掃描閘門與後端
  同規則）。
- `src/data/api-provider.ts`：snake_case 對映、401 沿用
  `failUnauthorized`。
- `src/views/TimelineView.tsx`：`TimelineView`（雙頁籤、建立表單、
  載入更多、重新載入、四態、資料來源標示與無即時推播說明）、
  `PostCard`（內容、能見度徽章、媒體詮釋資料、按讚、留言區、
  作者專屬編輯／刪除）、`PostMediaPickerDialog`（僅 clean 檔案
  可選、上限 9 筆）；貼文媒體只作詮釋資料顯示，不設下載按鈕。
- `src/App.tsx`：導覽新增「社群動態」（`FeedIcon` 為新增的
  stroke 圖示，沿既有圖示集作法）；`src/styles/components.css`
  沿既有 token 新增時間線樣式，未新增色板。

驗證證據（`TZ=Asia/Taipei`，於 `apps/web`）：

- `node tests/run.mjs`：**197 項全過、0 失敗**（先前 184 項＋
  新增 13 項：api 對映／401／展示守衛 8、元件 SSR／純函式 5）。
  中途 2 項測試本身問題已修正（SSR 註解節點比對、留言計數
  期望），未改元件行為。
- `npx tsc --noEmit`、`npx eslint`、`npx prettier --check` 全過；
  `npx vite build` 成功；`contrast-check.mjs` 全數 PASS（時間線
  文字／強調色沿既有 bg-surface 配對，不新增配對）；
  `responsive-audit.mjs` 全數 PASS（含時間線 3 項新契約）。

- 已知限制：貼文與留言無即時推播（§21 無對應事件名，他人
  變更須重新載入才顯示）；留言區須展開才載入；回覆只顯示已載入
  留言中的父留言作者（未載入的父留言不顯示回覆標籤）；已刪除
  父留言的回覆標籤為「回覆已刪除的留言」（顯示規則待定稿）；
  作者名稱以好友清單解析，未知作者顯示識別碼前 8 字元；展示
  模式游標格式僅展示資料內部使用。
- 待決定（本里程碑決定，待定稿覆核，見 §3 待決定 30）：回覆
  顯示規則、作者名稱解析來源、媒體挑選上限 9 筆的 UI 呈現、
  FeedIcon 圖示、貼文卡片不設下載按鈕。

## 2ad. 通話會話（Call）後端（2026-11-25，Phase 07 離線可驗證部分）

範圍（僅會話與紀錄；Web 接線列後續里程碑）：發起通話（一對一
與群組）、接聽／拒接／忙線／未接、結束通話、參與者列出、通話
事件紀錄（僅狀態事件）、通話紀錄查詢（游標分頁）。LiveKit 房間
建立、Access Token 簽發、媒體品質資料寫入、通話中即時信令、來電
推播一律未實作，見待決定 31。

實作（`services/api`，零新依賴，`go.mod`／`go.sum` 未動）：

- `internal/call/call.go`：領域＋記憶體儲存＋服務。狀態機
  ringing → active／declined／busy／missed／ended（結束為終端）；
  一對一通話須為直接對話、任一方向封鎖一律 404（經 Chat 的
  DirectBlocked，不直連 Friend 表）；群組通話受邀者為全部其他
  生效成員（經 Group 的 ListMembers）；同一範圍內已有進行中通話
  時拒絕重複發起；響鈴中結束且無人接聽記 missed，已接聽後結束
  記 ended（掛斷），全員回應且無人接聽時按拒接優先彙總為
  declined／busy；started_at 為首次接聽時間、duration 僅有接聽
  期間的結束通話才有值；事件 payload 只含狀態轉移資料，不得含
  SDP／ICE、Token、金鑰材料或通話內容；信令協調只留
  SignalCoordinator 佔位介面，Service 不依賴。只經唯讀介面讀取
  User／Chat／Group，不直讀他模組資料表。
- `internal/call/postgres.go`：PG 儲存庫（游標分頁元組比較、零列
  更新回 404、參與者唯一衝突對映欄位錯誤）。
- `internal/call/handler.go`：9 個端點，全部經 RequireAuth
  （`/api/v1/calls` 發起／清單、單則讀取、接聽／拒接／忙線／
  結束、參與者列出、事件列出）。事件回應不含 payload。通話不寫
  Outbox 事件（信令經 Gateway，屬後續里程碑），客戶端以 REST
  查詢同步。
- `migrations/000013_calls.up/down.sql`：只建 calls、
  call_participants、call_events 三表（000001–000012 未動；檔頭
  載明與 ERD §6.9 草案的九點差異：不建 livekit_room_ref 與
  device_id、值域 text 不加 CHECK、參與者 (call_id, user_id)
  唯一、payload 為 TEXT 僅狀態資料、started_at／duration 可為空）。
- `cmd/api/main.go`：兩模式接線（PostgreSQL／記憶體儲存皆以
  userSvc＋chatSvc＋groupSvc 注入唯讀介面）。
- `internal/postgrestest/fake.go`：FakeRow 補 `*sql.NullInt64`
  掃描支援（加法，不影響既有測試）。

驗證證據（Go 1.25.0，專案樹外工具鏈；`GOFLAGS=-mod=readonly GOPROXY=off`）：

- `go build ./...`、`go vet ./...`（含 `-tags integration`）通過；
  gateway／worker 模組建置與測試通過；`gofmt` 新檔乾淨
  （`health.go` 既有漂移未動）。
- `go test ./... -count=1`：**291 項全過、0 失敗**（先前 269 項＋
  新增 22 項：領域狀態機 15、PG 假查詢 6、HTTP 流程 1，含三帳號
  好友／對話／群組通話流程）。`go test -tags integration`：0 失敗，
  22 項整合測試全數 SKIP 標示環境待驗證（含新增的
  TestPG_CallFlowPersists）。

- 已知限制：通話列、參與者列、事件列分步寫入（跨表無交易邊界，
  發起寫入中途失敗時可能殘留通話列）；狀態轉移讀取—檢查—寫入無
  序列化（併發接聽／拒接以最後寫入為準）；重複發起守衛以
  FindActiveCall 查詢為準（併發發起可能同時通過）；發起人忙線不
  檢查（無 presence 系統，忙線只由受邀者回報）；參與者／事件列出
  無分頁（以通話生命週期短為前提）；清單 N+1（逐通話查參與者計數）。
- 待決定（本里程碑決定，待定稿覆核，見 §3 待決定 31）：狀態／
  類型／結果／事件／結束原因值域、游標分頁預設與上限、重複發起
  守衛、未接判定（響鈴逾時自動轉未接須排程／Worker）、LiveKit
  房間參照形態、裝置粒度、事件 payload 形態、信令事件與 Outbox
  關係、通話紀錄保留期限。
32. 通話 Web 接線實作預設（本里程碑決定，待定稿覆核）：來電橫幅
    只顯示本人參與者結果為待回應的來電（參與者查詢失敗時顯示
    通話、操作仍受後端守衛）；對象名稱以對話標題／群組名稱解析、
    未知使用者顯示識別碼前 8 字元；無標題直接對話顯示「一對一
    通話」；通話計時僅在詳情開啟且通話中時每秒更新；結束確認
    對話框只在詳情內，來電橫幅操作無確認步驟；導覽圖示沿用既有
    PhoneIcon；通話卡片不顯示任何媒體狀態。

## 2ae. 通話會話（Call）Web 接線（2026-11-25）

範圍（僅會話狀態 UI；媒體面一律未接通）：通話紀錄清單（游標分頁、
載入更多）、發起通話（一對一對話／群組人選、語音／視訊選擇）、
來電橫幅（僅本人尚未回應的響鈴來電）、接聽／拒接／忙線／結束
（含結束確認對話框）、參與者狀態列、通話事件列、通話計時（僅以
startedAt 計算）。WebRTC／getUserMedia／媒體串流、Token／金鑰材料
處理、假裝音訊或視訊已接通的 UI 一律未實作；通話中與響鈴中介面
皆明載「媒體尚未接通，僅會話狀態」。

實作（`apps/web`，零新依賴，`package.json` 不動、無 lockfile）：

- `src/data/types.ts`：`CallItem`／`CallPage`／`CallParticipantItem`／
  `CallEventItem`／`InitiateCallInput` 與狀態／類型／結果／事件／
  結束原因值域型別（與後端 §2ad 詞彙一致）；事件型別不含 payload
  欄位（後端永不回傳）。
- `src/data/demo.ts`：`DEMO_CALLS`（已結束／未接／全員拒接／響鈴
  中來電各一）／`DEMO_CALL_PARTICIPANTS`／`DEMO_CALL_EVENTS`
  （事件不含 payload）。
- `src/data/api.ts`：9 個端點的 wire 型別與客戶端方法（發起／清單／
  單則／接聽／拒接／忙線／結束／參與者／事件；游標與 limit 以查詢
  參數傳遞）。
- `src/data/provider.ts`：9 個可選方法＋展示實作（狀態機、範圍／
  封鎖／重複發起／發起人守衛、startedAt 僅首次接聽、duration 僅
  有接聽期間的結束通話才有值、事件不含 payload，皆與後端同規則；
  展示受邀者永不自行接聽，響鈴通話只由呼叫者操作解決）。
- `src/data/api-provider.ts`：snake_case 對映、401 沿用
  `failUnauthorized`；發起請求恰好含一個範圍識別碼。
- `src/views/CallsView.tsx`：`CallsView`（清單／來電橫幅／重新載入／
  四態、資料來源標示與無即時推播說明）、`CallCard`（狀態徽章、類型、
  對象、時長摘要）、`CallDetailPanel`（參與者／事件、依身分與狀態的
  操作按鈕、結束確認對話框、通話中每秒計時）、`InitiateCallDialog`
  （雙頁籤人選、名稱搜尋、類型選擇、重複發起錯誤顯示於框內）；
  對象名稱以對話標題／群組名稱解析，未知使用者顯示識別碼前 8 字元；
  來電橫幅只顯示本人參與者結果為待回應的來電（查不到時顯示通話，
  操作仍受後端守衛）。
- `src/App.tsx`：導覽新增「通話」（PhoneIcon）；`src/styles/components.css`：
  沿既有 token 新增通話樣式，未新增色板。

驗證證據（`TZ=Asia/Taipei`）：

- `node tests/run.mjs`：**210 項全過、0 失敗**（197＋13：api 對映／
  401／展示守衛 6、元件 SSR／純函式 7）。中途 3 項測試本身期望
  寫錯已修正（JSX 註解分隔、已加入者結束規則、後端守衛檢查順序），
  未改元件行為。
- `npx tsc --noEmit`、`npx eslint`、`npx prettier --check` 全過；
  `npx vite build` 成功；`contrast-check.mjs` 全數 PASS（通話卡片
  沿既有 bg-surface 文字配對，不新增配對）；`responsive-audit.mjs`
  全數 PASS（卡片／列無固定寬度、操作列換行）。

- 已知限制：無即時信令與來電推播（他人狀態變更須重新載入）；清單
  不含逐通話參與者結果，來電橫幅須為響鈴來電另查參與者；展示模式
  受邀者永不自行接聽；通話計時僅在詳情開啟且通話中時每秒更新；
  無標題的直接對話以「一對一通話」顯示，不猜測對方身分；結束確認
  對話框只在詳情內，來電橫幅的拒接／忙線無確認步驟（與後端立即
  生效語義一致）。
- 待決定（本里程碑決定，待定稿覆核，見 §3 待決定 32）：來電橫幅
  的參與者結果查不到時的顯示規則、對象名稱解析來源（對話標題／
  群組名稱／好友清單）、無標題直接對話的顯示文字、通話計時更新
  頻率、結束確認對話框的適用範圍（僅詳情）、導覽圖示為既有的
  PhoneIcon。

## 2af. 貼圖（Sticker）後端（2026-11-25，Phase 08 離線可驗證部分）

範圍（僅貼圖包／貼圖／收藏／分類／搜尋；Web 接線列後續里程碑）：
貼圖包建立／查詢／編輯／刪除（作者限定，刪除為狀態標記）、貼圖
新增／移除（本體只存 media_file_id，經 Media 掃描閘門）、使用者
收藏／取消收藏、最近使用清單、分類篩選與關鍵字搜尋、授權發放
介面（GrantPack／RevokeGrant，無 HTTP 端點）。購買交易、商店展示
排序與推薦、排行榜、創作者頁、分享連結、下載計數商業統計一律未
實作，見待決定 33。

實作（`services/api`，零新依賴，`go.mod`／`go.sum` 未動）：

- `internal/sticker/sticker.go`：領域＋記憶體儲存＋服務。包狀態
  draft／published／unlisted／archived（刪除為狀態標記）；貼圖狀態
  active／removed；分類 general／animals／food／emotions／festivals／
  characters／other（空值未分類）。可見性：作者本人恆可見；已上架
  包免費者全體可見、付費者須授權發放紀錄（fail-closed）；草稿／未
  上架僅作者；已封存一律 404。`user_stickers` 一列只關聯包或單張
  其中之一（CHECK 約束）；source 為 favorite／granted／recent；
  收藏與授權互不吃列（授權列可立收藏旗標、撤回授權時已收藏者來源
  改回 favorite）；最近使用去重、上限 30 筆、舊列裁剪。貼圖序號
  接在最大序號之後（含已移除列，不得重用）；單包生效貼圖上限
  100 筆。搜尋 token AND 子字串比對（貼圖名與包名聯集）。游標
  語義取最後一筆回傳列（見已知限制）。只經唯讀介面讀取
  User／Media，不直讀他模組資料表；不寫 Outbox 事件（§21 無貼圖
  事件名）。
- `internal/sticker/postgres.go`：PG 儲存庫（游標分頁元組比較、零列
  更新回 404、包級／單張級部分唯一索引 upsert、最近使用 OFFSET
  裁剪）。
- `internal/sticker/handler.go`：14 個端點，全部經 RequireAuth
  （`/api/v1/sticker-packs` 建立／清單、單則讀取／編輯／刪除、貼圖
  新增／列出、收藏／取消收藏；`/api/v1/stickers` 收藏清單／最近
  使用／記錄使用／移除／搜尋）。不可見者一律 404，不得洩漏存在性。
- `migrations/000014_stickers.up/down.sql`：只建 sticker_packs、
  stickers、user_stickers 三表（000001–000013 未動；檔頭載明與
  ERD §6.7 草案的九點差異：不建 sticker_purchases、不建
  author_name、值域 text 不加 CHECK、恰居其一 CHECK、部分唯一索引、
  FK 一律 RESTRICT）。
- `cmd/api/main.go`：兩模式接線（PostgreSQL／記憶體儲存皆以
  userSvc＋mediaSvc 注入唯讀介面）；`main_test.go` 新增
  `TestStickers_Flow`（1 項）。

驗證證據（Go 1.25.0，專案樹外工具鏈；`GOFLAGS=-mod=readonly GOPROXY=off`）：

- `go build ./...`、`go vet ./...`（含 `-tags integration`）通過；
  gateway／worker 模組建置與測試通過；`gofmt` 新檔乾淨
  （`health.go` 既有漂移未動）。
- `go test ./... -count=1`：**309 項全過、0 失敗**（先前 291 項＋
  新增 18 項：領域守衛 10、PG 假查詢 7、HTTP 流程 1）。`go test
  -tags integration`：0 失敗，23 項整合測試全數 SKIP 標示環境待驗證
  （含新增的 TestPG_StickerFlowPersists）。`migrations` 靜態驗證通過
  （000014 連續編號、up／down 配對）。

- 已知限制：貼圖包列與貼圖列分步寫入（跨模組無交易邊界）；狀態
  轉移讀取—檢查—寫入無序列化（併發以最後寫入為準）；清單 N+1
  （逐包查生效貼圖數與收藏狀態）；搜尋與清單為掃描後過濾；最近
  使用裁剪與寫入分步執行。游標語義與修正後的 Timeline `feedPage`
  一致（回傳游標取最後一筆回傳列；§2af 探測發現的舊 `feedPage`
  漏列缺陷已於 §2ag 修正）。
- 待決定（本里程碑決定，待定稿覆核，見 §3 待決定 33）：包／貼圖／
  分類／來源值域、單包貼圖上限 100 筆、最近使用保留 30 筆、分頁
  預設與上限、搜尋聯集比對語義、授權發放與撤回介面形態、封存包
  的收藏清理語義、貼圖包管理員角色（現僅作者；管理員模組為佔位）。

## 2ag. Timeline `feedPage` 游標修正（2026-11-25，parent 直接修正）

§2af 探測實證的漏列缺陷：`feedPage` 回傳游標取最後一個候選列，
而 Store 介面契約為「呼叫方以回傳列的最後一筆組裝下一頁游標」、
掃描為游標之後（更舊）的一頁；探測列（多取一列）在下一頁被跳過。
五筆全可見、limit 2 時合計只取回四筆（`[p5 p4 p2 p1]`，p3 遺失），
探測已於修正前重現、修正後通過。

修正（`internal/timeline/timeline.go`，僅 `feedPage` 一函式，migration
與端點 shape 未動）：掃描游標與回傳游標分離——掃描游標每輪推進
至最後一個候選列（避免全數被濾除時原地重掃），回傳游標取最後一
筆回傳列的排序鍵；候選不足一輪需求量時回空游標。`ListComments`
原已採 limit+1 探測與最後回傳列游標，未動。新增回歸測試
`feed_pagination_regression_test.go` 兩項：全可見五筆分頁無漏列、
被濾除列夾在中間時分頁無漏列。

驗證證據（Go 1.25.0，`GOFLAGS=-mod=readonly GOPROXY=off`）：
`go test ./... -count=1` **311 項全過、0 失敗**（309＋2 項回歸測試，
含 §2ab 既有 `TestFeedPaginationAndFiltering`）；`go vet` 通過；
`gofmt` 新檔乾淨（`health.go` 既有漂移未動）；`go test -tags
integration` 全數 SKIP 標示環境待驗證。實機驗證仍待隔離環境
（多頁續拉併入 Timeline 雙帳號實機流程）。

## 2ah. 貼圖（Sticker）Web 接線（2026-11-25）

範圍（純前端；後端 Go 程式碼、migration SQL、凍結檔皆未動）：
貼圖包清單（分類篩選、關鍵字搜尋、游標載入更多）、貼圖包詳情
（貼圖格、作者限定管理操作）、收藏／取消收藏、我的收藏清單、
最近使用列、建立／編輯／刪除貼圖包、新增／移除貼圖；展示／API
雙模式；四態與失敗 toast；資料來源誠實標示。購買／付款流程、
第三方素材、假裝已下載貼圖檔案的行為一律未實作（亦無此 UI）；
貼圖只以名稱／代號方塊與詮釋資料呈現。

實作（`apps/web`，零新依賴，`package.json` 未動、無 lockfile）：

- `src/data/types.ts`：`StickerPackCategory`／`StickerPackStatus`／
  `StickerStatus`／`StickerPackItem`／`StickerPackPage`／
  `StickerItem`／`StickerPage`／`CreateStickerPackInput`／
  `UpdateStickerPackInput`／`AddStickerInput`（TS 型別完整）。
- `src/data/demo.ts`：展示資料與後端可見性同規則——付費包無授權
  發放紀錄者不可見、草稿僅作者、未上架僅作者、封存一律不存在；
  7 包（免費已上架、免費已上架、付費已上架且已授權、付費已上架
  未授權、本人草稿、他人未上架、已封存）、12 張貼圖（含 1 張
  removed 列，驗序號接續規則）、授權／收藏／最近使用列各 1 組。
- `src/data/provider.ts`：`ChatDataProvider` 新增 14 個可選方法與
  展示實作（可見性裁決、作者管理守衛、收藏冪等、授權保留、掃描
  閘門 `demoResolveAttachable`、100 筆上限、最近使用 30 筆去重、
  搜尋 AND token 聯集比對、游標取最後一筆回傳列）。
- `src/data/api.ts`：wire shape（`pack_id`／`sticker_id`，清單鍵名
  `packs`）與 14 個 client 方法，對映後端實際路徑與方法（含刪除
  為 POST `.../delete`、記錄使用為 POST `.../use`、收藏清單為
  GET `/api/v1/stickers/favorites`）；`searchStickers` 空關鍵字於
  發請求前拒絕（q 必填，與後端一致）。
- `src/data/api-provider.ts`：snake_case 對映與 14 個方法，401 沿用
  `failUnauthorized`（清除 Token、連線狀態轉 unauthorized）。
  後端 wire 無 `granted_by_me` 欄位（`packResponse` 未發出），故
  item 不帶授權旗標；可見的付費包皆為已授權（不可見者後端回
  404），UI 於付費包標示「付費 · 已授權」。
- `src/views/StickersView.tsx`：`StickersView`（瀏覽／我的收藏／
  最近使用三頁籤；搜尋為瀏覽頁籤內的模式，送出後以貼圖結果列
  取代包清單、可清除；分類篩選與搜尋可併用）與子元件
  `StickerPackCard`／`StickerTile`／`StickerPackDetailPanel`／
  `StickerMediaPickerDialog`（僅 clean 檔案可選，關聯既有檔案、
  不重新上傳位元組）／`StickerPackFormDialog`（新建一律草稿、狀態
  欄只在編輯時出現、封面以媒體挑選器選取）／`AddStickerDialog`
  （名稱、動態旗標、單一媒體檔案挑選；序號與上限由後端裁決）。
  驗證 helper 與後端上限同規則（包名／貼圖名 64 runes、描述
  500 runes）；作者管理權以 `creatorId` 判定；刪除以確認對話框
  呈現封存語義；無即時推播，介面明載重新載入後才同步。
- `src/App.tsx`：導覽新增「貼圖」視圖（`StickerIcon`，沿既有
  AppShell 模式）；`src/styles/components.css` 新增貼圖樣式（沿
  既有 token，不新增色板）；`src/components/icons.tsx` 新增
  `StickerIcon`（圓角方塊＋笑臉，待設計審查）。

驗證證據（`apps/web`，`node tests/run.mjs`，Asia/Taipei 時區）：

- Web 測試 **229 項全過、0 失敗**（先前 210 項＋新增 19 項：
  `api.test.mjs` 10 項——client 端點對映與參數組裝、空關鍵字守衛、
  provider wire 對映與 snake_case body、貼圖包清單 401 清除 Token、
  展示可見性／收藏冪等／管理守衛／掃描閘門／100 張上限／搜尋
  AND token／最近使用去重；`components.test.mjs` 9 項——雙模式
  標題／頁籤／載入狀態／資料來源標示、卡片／詳情／方塊／建立
  表單／新增對話框／媒體挑選器 SSR 斷言、標籤與名稱檢查純函式）。
  裸 UTC 環境下 228／229，唯一失敗為 Phase 02 既有
  `formatMessageTime` 時區假設測試（M3 已知問題，非本里程碑引入）。
- `npx tsc --noEmit`、`npx eslint src tests`、`npx prettier --check
  src tests` 全過；`npm run build`（vite）成功；對比度稽核 22 項
  PASS、響應式稽核 24 項 PASS（含新增的 2 項貼圖契約：包卡片無
  固定寬度＋網格自動填充、工具列與搜尋列換行）。
- `tests/README.md` 補貼圖測試說明；`tests/responsive-audit.mjs`
  補 2 項貼圖契約。`tests/entry-components.tsx` 新增 8 個 render
  函式與 helper 重匯出。

- 已知限制：無即時推播與來電式更新，狀態只在重新載入後同步；
  貼圖格只顯示名稱與詮釋資料，不提供檔案下載；分類篩選無
  「僅未分類」選項（後端以空 category 表示不過濾）；搜尋結果
  只顯示貼圖名與所屬包名；最近使用列只在記錄使用或切換頁籤
  時重載；展示模式的 `updateStickerPack`
  不檢查狀態 allowlist（後端拒絕 `archived` 等非法值，展示實作
  直接寫入，僅展示資料內部差異）；瀏覽器互動（篩選、搜尋送出、
  收藏切換、對話框流程、載入更多）與雙帳號實機流程未執行，待
  隔離環境。
- 待決定（本里程碑決定，待定稿覆核，見 §3 待決定 33）：Web 實作
  預設——搜尋為瀏覽頁籤內模式、付費可見包標示文字、封面挑選
  沿用媒體挑選器、client 空關鍵字守衛；`StickerIcon` 設計審查；
  貼圖包管理員角色現僅作者。

## 2ai. ETHER AI Provider Adapter 介面層：規格盤點（2026-11-25，僅盤點、未實作）

本里程碑依任務指示先盤點 Phase 09 規格是否已定稿到可實作程度。
結論：**核心介面語義未定稿，停在盤點，未寫任何功能程式碼**
（`services/api/internal/ai` 不存在；`go.mod`／`go.sum`、migration、
後端程式碼、Web 程式碼皆未動）。以下逐項以文件為準，文件無載明
者記缺口，不得自行發明。

已定稿（方向層，見 ROADMAP Phase 09、PRD、ADR-009／017、
SYSTEM_ARCHITECTURE §7、MODULE_BOUNDARIES §7）：

- AI 採 Ollama＋vLLM＋LocalAI 優先、外部 AI API Adapter 可選擴充；
  供應商不得寫死於商業邏輯；模型切換不得改商業邏輯。
- Adapter 為跨模組共用能力、非第 23 個模組；商業模組只呼叫
  Adapter 介面；授權、用量、遮罩、保留策略由 Adapter 層統一落實，
  不得由各模組各自實作。
- 向量承載為 pgvector（ADR-017 已定案）；但維度、距離度量、索引
  結構未定，ERD §10.4 明定須由模型決定、不得預寫。
- E2EE 明文絕對不得自動送往任何 AI（含本地自架 AI）；E2EE 對話
  的 AI 功能預設關閉（SECURITY_ARCHITECTURE §7、AI_DATA_POLICY 鐵律；
  該政策文件本身未批准，見下）。
- 無 Key／無模型時只做介面並標記未完成，不得以假回應冒充 AI
  結果（ROADMAP Phase 09 驗收條件）。
- 錯誤碼保留 `AI` 領域段（ERROR_CODES.md），但註明「§21 未定前綴…
  不得以本行假裝 AI 端點已定」。

未定稿（缺口，任一項皆足以擋住介面實作；編號見 §3 待決定 34）：

1. `docs/security/AI_DATA_POLICY.md` 全文狀態為「草案／待批准」，
   末段明載「不得宣稱產品完成或 AI 功能已可用」；其 §2.2 所列
   Adapter 職責（格式轉換、認證、重試與退避、逾時、錯誤轉譯、
   Token 計量回傳、串流事件轉換）僅為原則，無簽名、無數值。
2. 介面方法名與簽名：政策 §2.2 明載「介面名待 API 文件定案」；
   API_STANDARDS／ERROR_CODES／WEBSOCKET_PROTOCOL 皆無 AI 端點、
   請求／回應型別、串流事件格式。對話補全、串流補全、嵌入、
   語音轉文字、圖片理解五種能力的 Go 簽名（訊息角色、上下文
   結構、chunk 型別、向量型別、輸入格式）全無文件依據。
3. 串流語義：僅方向（逐步顯示、中斷標示「未完成」、不得拼湊舊
   片段）；片段級遮罩與內容安全檢查方式「待定案」。
4. 錯誤／逾時／重試：僅原則（Adapter 負責）；逾時值、重試次數與
   退避、供應商錯誤對映——全無數值與對映表。
5. 模型選擇與預設：模型清單、預設模型、備援順序、切換權限——
   全部「待定案」；PRD NQ-06 仍為開放問題；ROADMAP 阻塞項明載
   「模型清單與授權待查證」。Ollama／vLLM／LocalAI 的分工與分流
   規則亦待部署文件定案；三者 API shape（端點路徑、請求格式）
   在規格文件中全無載明，不得以公開文件知識預寫。
6. 憑證／金鑰管理：僅原則（OpenBao 管理、不得入 Git）；外部供應商
   選擇、Key 申請、資料處理協議、成本——未定案（ADR-004 阻塞清單）。
7. 成本／用量上限：Token Budget、Rate Limit 具體數字一律「待定，
   不得編造」；外部成本待查證；資源需求（CPU／RAM／GPU／VRAM）
   全部待實測，ROADMAP 列為阻塞項。
8. E2EE 與 AI 的例外：鐵律方向已定，但例外授權三要件的落實方式、
   E2EE 對話上下文納入方式、例外範圍——「尚未定案，預設關閉」。
9. Tool Calling：工具白名單、參數驗證規則、疑似注入的通報流程——
   未定案；工具呼叫紀錄的表／欄位未定（ERD 無 AI 使用紀錄表）。
10. 使用紀錄與保留：僅詮釋資料方向；完整明文是否記錄、保留期限——
    未定案；無對應 migration 設計。
11. 向量細節：維度、距離度量、索引更新方式、與 Meilisearch／PostgreSQL
    的分工——ERD §10.4 明定待模型決定；RAG 權限隔離與刪除同步的
    實作細節待定案。

後續實作的前置條件（皆須先定稿，見 §4）：AI_DATA_POLICY 批准；
API 文件定稿 Adapter 介面名、請求／回應型別與串流事件格式；
模型清單與預設模型定案；逾時／重試／預算／頻率數值定案；
使用紀錄表與向量維度定案。定稿前，任何 Provider 實作（含
Ollama 轉接器）皆屬自行發明規格，不得開工。

## 2aj. 社群（Community）後端：規格盤點（2026-11-25，僅盤點、未實作）

本里程碑依任務指示先盤點社群管理實體語義是否已定稿到可實作
程度。結論：**核心語義未定稿，停在盤點，未寫任何功能程式碼**
（`services/api/internal/community` 僅 Phase 01 `doc.go` 佔位；
`go.mod`／`go.sum`、migration、後端程式碼、Web 程式碼皆未動）。
以下逐項以文件為準，文件無載明者記缺口，不得自行發明。

已定稿（方向層，僅框架與表名）：

- 模組邊界與五表表名已定（MODULE_BOUNDARIES §5.9、ERD §6.5）：
  `communities`、`community_members`、`community_channels`、
  `community_roles`、`community_invitations`；不得依賴 Group 模組、
  不得直接寫入訊息表、不得以社群角色代替全平台管理員權限。
- 功能方向已定（FEATURE_LIST §09 F-09-01～F-09-10、ROADMAP
  Phase 06）：公開／私密、搜尋與加入、邀請與審核、成員管理與
  分級權限、管理員與公告／規則、多聊天室與子群組、討論區、
  活動與投票、檢舉與黑名單、資料分析。
- 框架方向已定（IDENTITY_AND_PERMISSIONS §7.3）：社群建立者為
  最高級、其下設管理層級與一般成員層級、管理操作須留存操作
  紀錄、同一社群內每人僅一個角色、社群角色不得與群組角色混用。
- 刪除方向已定（ERD §6.5）：社群關閉／解散、成員離開／移除、
  頻道刪除／封存、邀請紀錄皆為狀態標記，不得物理刪除。

未定稿（缺口，任一項皆足以擋住忠實實作；編號見 §3 待決定 35）：

1. 分級權限的具體等級與操作對照表：PRD §195 明載「尚未定案
   （僅有表名 `community_roles`）」；IDENTITY §7.3 明載「具體等級
   劃分尚未定案，本文件只定框架，不代定等級」；FEATURE_LIST
   F-09-04 標註「規格缺口」。等級名詞、層級數、管理操作與等級
   的對照、任命／降級流程——全無文件依據。
2. `role` 與 `role_id` 主從關係待定稿（ERD §6.5.2）：成員以固定
   角色詞彙為準、或以 `community_roles` 自訂角色列為準，未定；
   `community_roles.permissions` 形態為草案，`is_system_role`／
   `position` 為草案，系統角色清單未定。
3. `visibility` 值域待定稿（ERD §6.5.1，僅舉例公開／私密）；
   `category` 值域待定稿；社群狀態欄位本身待定稿（關閉／解散
   的狀態名詞未定）；`member_count_cache` 為草案。
4. 邀請／審核流程僅方向（F-09-02／F-09-03）：`community_invitations`
   的 `status` 值域待定稿；`invite_token` 為草案（雜湊儲存）；
   誰可邀請、誰可審核、申請與邀請是否同一狀態機、過期與重發
   語義——全無文件依據。
5. 頻道語義未定：`channel_type` 值域待定稿（僅舉例文字／公告／
   討論）；`position`／`is_private` 為草案；頻道與 `conversations`
   的對應方式為 ERD §10 缺口（ROADMAP Phase 06 阻塞項），
   `conversation_id` 僅為草案參照，不得預寫為已定。
6. 成員唯一性：`(community_id, user_id)` 在生效期間的唯一性待定稿；
   重新加入、封鎖／黑名單與成員資格的關係未定（黑名單僅
   F-09-09 方向，F-09-09 的檢舉流程屬 Moderation／Phase 11）。
7. 公告／規則、討論區、活動／投票、操作紀錄、資料分析：皆僅
   功能方向，無承載表定義（ERD 五表之外無公告／規則／討論／
   活動／投票／操作紀錄表）；公告與規則的欄位、討論主題狀態機、
   活動與投票的截止語義、操作紀錄的表與欄位、分析指標定義——
   全部待定（F-09-10 指標定義待定）。
8. 搜尋語義僅方向（公開社群可被搜尋）；比對方式、排序、游標、
   上限——無文件依據。端點路徑、請求／回應型別——API 文件
   （API_STANDARDS／ERROR_CODES／WEBSOCKET_PROTOCOL）皆無社群端點。
9. 規格源頭文件 `SPEC-SOURCE.md` §09 不在本樹文件內（`docs/`
   下無此檔案）；ERD／PRD／FEATURE_LIST 對 §09 的引用皆為轉述，
   無法以原文核對。`member_count_cache` 與群組備註一致的快取
   更新語義亦未定。

後續實作的前置條件（皆須先定稿，見 §4）：分級權限等級與操作
對照表；`visibility`／`category`／狀態／邀請狀態值域；`role` 與
`role_id` 主從；頻道與 `conversations` 對應方式；邀請／審核狀態機；
公告／規則／討論／活動／投票／操作紀錄的承載設計；端點
Schema。定稿前，任何 Community 領域實作（含 migration）皆屬自行
發明規格，不得開工。群組里程碑（§2i）的「角色固定三種」為該
里程碑的實作預設，不得類推為社群等級已定。

## 2ak. GitHub Actions 一次性隔離整合測試環境：草稿撰寫（2026-11-25，本機撰寫、未執行）

使用者 2026-11-25 19:13 下令以 GitHub Actions Hosted Runner 為一次性
隔離環境並授權 Phase 01 Step 7；查證結果為 shengwen0928 名下尚無
ETHER Repository（認證搜尋 ether 得 0 筆；本機樹無 `.git`、無
`.github/`），帳務唯讀查核為 GitHub Pro $0、Actions 預算 $0 上限
（超額即阻擋）、11 月 Linux 已用 218 分鐘全數折扣、內含運算額度 $18
尚餘。本里程碑只產出「可直接放入新 repo 就能跑」的草稿，未推送、
未建 repo、未執行 Docker；第一次真正執行須待 repo 建立＋workflow
檔案經網頁建立（GitHub App 無 workflows 寫入權限，API 推送
`.github/workflows/` 會回 403，不得以 API 推送冒充已建立）。

草稿清單（皆為新增檔案，未改任何既有檔案）：

- `.github/workflows/ether-isolated-integration.yml`：僅
  `workflow_dispatch` 觸發，`permissions: contents: read`，concurrency
  單一群組，timeout integration 60 分／platform-checks 30 分。兩個
  job：(a) `integration`（Stage 1 環境與 §9.1 安全檢查含新 Runner 無
  既有 ether volume 證明、digest 以 `buildx imagetools inspect` 覆核、
  Stage 2 三映像建置＋S2-6 負向 context 檢查＋S2-7 pgvector 探測、
  Stage 3 compose config 雙流程＋S1-2、Stage 4 僅啟動 9 項服務並逐項
  健康檢查、Stage 5 建 keycloak 資料庫＋`cmd/migrate up`＋S2-7 對照＋
  integration 測試、Stage 6 Keycloak 實機 OIDC、Stage 6b 腳本化授權碼
  ＋PKCE 嘗試、Stage 7 雙帳號互動、Stage 8 NATS 經 Relay／消費迴圈＋
  監控 API、Stage 9 Meilisearch 經消費迴圈＋HTTP API、Stage 10 MinIO
  經 S3 API 與 api presigned 往返、Stage 11 Go／Web 全量測試、Stage 12
  `always()` 清理＋殘留檢查）；(b) `platform-checks`（S1-4 promtool、
  S3-9 預設 up 與 livekit 未啟動、S3-10 監控健康檢查、S3-11 livekit
  profile 負向流程含 ErrKeysNotSet、S4-14 keycloak＋glitchtip 嘗試）。
  外部 Action 只用 `actions/checkout` 與 `actions/upload-artifact`，
  皆為佔位 SHA 加 TODO（須查證後填入完整 commit SHA，不得編造）；
  隨機密碼以 `::add-mask::` 處理，不 echo、不寫入檔案、不上傳。
- `infrastructure/docker/docker-compose.ci.yml`（override，不改 base
  compose、不改任何映像與 digest）：project 名由 workflow 傳入
  `ether-isolated-integration`；僅測試所需服務加 `127.0.0.1` 埠對映；
  Keycloak 覆寫 `KC_HOSTNAME` 為空＋`KC_HOSTNAME_STRICT=false`，並由
  workflow 寫入 runner 的 `/etc/hosts`（`127.0.0.1 keycloak`／`minio`），
  全程以 `http://keycloak:8080` 為唯一 Keycloak 位址；隨機測試密碼由
  workflow 環境變數注入。Traefik、Ollama 不在 integration job 啟動
  範圍（S4-12 Traefik 路由段與 Ollama 段記 NOT VERIFIED 附原因）。
- `docs/testing/PHASE01_STEP07_INTEGRATION_REPORT.md`（範本）：狀態
  BLOCKED（尚無 repo、尚無 run），含五態圖例與全部待填證據欄位；
  未執行項一律未標 PASS。

設計決定（皆有樹內依據，不得視為已驗證）：專案根目錄自動辨識
（Repository 根目錄或其下 `ether` 子目錄，須含 compose 與
`services/api/go.mod`）；Keycloak 專屬 disposable realm `ether-ci`
（範本檔不改，以 jq 在記憶體內改寫 realm 名匯入）＋direct grant 專用
client 與 audience mapper（aud 須含 `ether-api`）；PKCE 嘗試失敗記
NOT VERIFIED，不得以 direct grant 冒充；Session 輪換／重放以 Go 整合
測試覆蓋（HTTP 層無建立／輪換端點，清單為空時撤銷檢查記
NOT VERIFIED）；NATS Stream `ETHER_EVENTS` 與 MinIO bucket 由部署前置
建立（程式碼不自動建立）；MinIO presigned 簽名綁定 `minio:9000` 主機，
故 runner 以 `/etc/hosts` 用同一主機名存取；glitchtip 首啟失敗記
NOT VERIFIED 不得阻塞整 run；清理只對本次專屬 project 執行
`down -v`（使用者授權範圍，專案名限定）。

阻塞原因：尚無 Repository（撰寫時查證）；workflow 檔案須經網頁建立
（403 限制）；checkout／upload-artifact 的完整 commit SHA 待查證填入；
glitchtip／traefik／ollama 的取捨見草稿註記與報告範本。本機驗證僅
YAML 解析、路徑／變數名勾稽與未改動核對（見本節驗證段）；Docker、
compose、Keycloak 實機行為一律未執行。

本機驗證（2026-11-25，實際執行）：`python3 -c yaml.safe_load` 解析
workflow 與 override 皆通過；`uses:` 僅 checkout 與 upload-artifact
（皆佔位 SHA）；workflow 參照的路徑（compose、Realm 範本、migrations、
`cmd/migrate`、integration 測試環境變數名、api／gateway 端點）皆經
grep 核對存在於樹內；base compose、Dockerfile、migration、凍結檔皆
未改動；新檔案無真實憑證字面、無 `.env` 寫入、無 `Bearer`＋token 完整
字面（token 一律取自變數）。

## 3. 未驗證與待決定事項

未驗證（不得宣稱為通過）：

1. Migration 000001–000005 與 000007 未經 PostgreSQL 套用（本機無 psql／Docker；
   套用／回滾／狀態測試已備妥，`integration` tag，待隔離資料庫）。
2. Keycloak 實機整合未驗證（無 Keycloak 執行個體；Verifier 僅以 httptest JWKS 伺服器測試；Web 端 OIDC 流程僅以 stub fetch 測試，未對真實 Keycloak 發出請求；實機測試案例已備妥，見 KEYCLOAK_INTEGRATION.md §4 清單；Web 端實機驗收須另補：授權導向、回呼交換、換發、登出）。
3. PostgreSQL 儲存庫未接真實資料庫驗證（程式碼已完成、假查詢單元測試
   通過；真實持久化測試待隔離資料庫；api 預設 `IDENTITY_STORE=postgres`，
   無資料庫時啟動即明確報錯，不得靜默降級）。
4. ETHER 端 Session 撤銷與 Keycloak 的同步／補償流程未定稿（IDENTITY §12 缺口 12）；請求期強制檢查 ETHER 撤銷狀態仍待 M2 後續（中介層只驗 Token）。
5. 真實瀏覽器響應式／輔助技術驗證（Phase 02 遺留）。
6. Web OIDC 授權碼＋PKCE 登入流程已實作（M2d，見 §2f，stub fetch
   測試通過）；實機 Keycloak 驗證仍待隔離環境。Web Token 儲存機制
   採 memory-only 暫行選擇（見待決定 12），localStorage 作法已退場。
7. 好友端點未經真實資料庫與多帳號實機驗證（M3 流程僅以記憶體儲存＋
   stub fetch 測試；三帳號互搜互邀、封鎖語義的實機驗收待隔離環境）。
8. Migration 000006 未經 PostgreSQL 套用；對話／訊息 PG 持久化
   未經真實資料庫驗證（2 項整合測試已備妥，`integration` tag，
   無 `ETHER_TEST_DATABASE_URL` 時跳過）。
9. Gateway 未經真實 Keycloak／NATS 驗證（OIDC 驗證僅以 httptest
   JWKS 測試；JetStream 轉接器僅佔位；WebSocket 真實瀏覽器連線、
   多連線扇出、斷線重連續拉皆未實測；訊息推播事件尚無生產者）。
10. Web 對話清單與歷史未經真實後端／瀏覽器驗證（本里程碑：
    stub fetch 對映測試與 SSR 測試通過；雙帳號實機流程——建 DM／
    群組、發送、游標續拉、未讀數——待隔離環境，見 §4）。
11. 群組端點未經真實資料庫與多帳號實機驗證（M6 流程僅以記憶體
    儲存＋假驗證器測試；三帳號建群組、邀請、角色、公告、轉移、
    解散的實機驗收待隔離環境；migration 000007 套用／回滾測試
    已備妥，`integration` tag）。
12. 群組 Web 接線未經真實後端／瀏覽器驗證（本里程碑：stub fetch
    對映測試與 SSR 測試通過；雙模式實機流程——建群組、邀請／
    接受、角色變更、公告、設定、轉移、解散——待隔離環境，見 §4）。
13. 媒體端點未經真實資料庫／物件儲存／多帳號實機驗證（本里程碑：
    記憶體儲存＋假驗證器測試；上傳→完成→掃描→授權→縮圖的實機
    流程、migration 000008 套用／回滾、MinIO 實機上傳皆待隔離環境）。
14. 媒體 Web 接線未經真實後端／瀏覽器驗證（本里程碑：stub fetch
    對映測試與 SSR 測試通過；雙模式實機流程——建會話、取消、
    授權檢查、刪除——待隔離環境，見 §4；位元組上傳通道已決定 presigned 直傳（見 §2z），定稿覆核待辦）。
15. Outbox／Relay 未經真實資料庫／NATS 驗證（本里程碑：記憶體
    儲存＋假查詢單元測試通過；同交易寫入與認領流程的整合測試
    已備妥，`integration` tag，無 `ETHER_TEST_DATABASE_URL` 時
    跳過；Relay 發佈迴圈僅以假 Publisher 測試，NATS 發佈者已落地（見 §2y），實機未驗證。
16. Gateway 訂閱成員驗證未經真實 api 整合驗證（本里程碑：stub
    checker 測試通過；Gateway 以 nil checker 啟動、訂閱與 typing
    一律拒絕；真實成員查詢的內部端點／事件投影設計待定稿，
    雙帳號實機流程待隔離環境，見 §2m）。
17. 已讀回條 Web 接線未經真實後端／瀏覽器驗證（本里程碑：
    stub fetch 對映測試、回條目標純函式單元測試與 SSR 測試通過，
    effect 本身未在瀏覽器執行；雙帳號實機流程——開啟對話送 read
    回條、未讀數歸零、側邊列徽章同步、失敗靜默與重試——待隔離
    環境，見 §2n 與 §4）。
18. 建立對話 UI 未經真實後端／瀏覽器驗證（本里程碑：stub fetch
    對映測試、人選過濾純函式單元測試與 SSR 測試通過，對話框的
    開啟、挑選、建立與失敗重試未在瀏覽器執行；雙帳號實機流程——
    選好友建 DM、新建 201 與既有 200 直接開啟、非好友不得選、
    建立失敗重試——待隔離環境，見 §2o 與 §4）。
19. 訊息反應與置頂未經真實資料庫／多帳號實機驗證（本里程碑：
    記憶體儲存＋假驗證器測試；反應新增／更換／取消／彙總、
    置頂／取消／清單／上限的雙帳號實機流程、migration 000010
    套用／回滾、PG 持久化整合測試皆待隔離環境，見 §2p 與 §4）。
20. 反應與置頂 Web 接線未經真實後端／瀏覽器驗證（本里程碑：
    stub fetch 對映測試、展示資料操作測試與 SSR 測試通過；挑選
    器互動、反應寫入／取消、置頂／取消置頂、清單載入與失敗重
    試未在瀏覽器執行；雙帳號實機流程——反應新增／更換／取消／
    彙總顯示、置頂清單與上限錯誤——待隔離環境，見 §2q 與 §4）。
21. delivered 回條與回條列表 Web 接線未經真實後端／瀏覽器驗證
    （本里程碑：stub fetch 對映測試、回條目標與狀態推導純函式
    單元測試與 SSR 測試通過，effect 本身未在瀏覽器執行；雙帳號
    實機流程——歷史載入送 delivered 回條、回條列表查詢、自己訊息
    已送達／已讀標示、群組全部成員語義——待隔離環境，見 §2r
    與 §4）。
22. 附件與轉傳 Web 接線未經真實後端／瀏覽器驗證（本里程碑：
    stub fetch 對映測試、展示資料操作測試與 SSR 測試通過；附件
    區塊載入、轉傳對話框的開啟／挑選／確認／失敗重試、附件挑
    選對話框的開啟／挑選／確認、發送時關聯附件未在瀏覽器執行；
    雙帳號實機流程——附件顯示與掃描狀態、轉傳至另一對話、轉傳
    後目標對話收到新訊息、附件新增守衛——待隔離環境，見 §2t
    與 §4；migration 000011 套用／回滾見 §2s）。
23. 編輯／收回／僅自己刪除 Web 接線未經真實後端／瀏覽器驗證
    （本里程碑：stub fetch 對映測試、展示資料操作測試、編輯內容
    檢查純函式單元測試與 SSR 測試通過；編輯框輸入／儲存、確認
    對話框的開啟／確認／失敗重試、操作後狀態更新未在瀏覽器執行；
    雙帳號實機流程——編輯內容與已編輯標示、收回後對方檢視、
    僅自己刪除後對方檢視不受影響、非作者不得操作——待隔離環境，
    見 §2u 與 §4）。
24. 訊息搜尋未經真實 Meilisearch／Worker 消費迴圈／多帳號實機驗證
    （本里程碑：記憶體索引＋假檢查器單元測試、handler stub 測試、
    接線流程測試通過；Worker 消費 Outbox 事件填充索引、Meilisearch
    轉接器查詢、重建流程、雙帳號實機流程——關鍵字／日期／發送者
    篩選、非成員對話過濾、收回後內容不殘留——皆待隔離環境與
    轉接器落地，見 §2v 與 §4）。
25. 訊息搜尋 Web 接線未經真實後端／瀏覽器驗證（本里程碑：
    stub fetch 對映測試、純函式單元測試、SSR 測試、對比度與
    響應式靜態稽核通過；面板送出／載入更多／跳轉捲動／錨定
    載入、API 模式空索引誠實顯示、503 失敗重試未在瀏覽器執行；
    雙帳號實機流程——關鍵字搜尋、篩選、跳轉至舊訊息、索引填
    充後的結果完整性——待隔離環境與填充迴圈接線，見 §2w
    與 §4）。
26. 對話成員管理 Web 接線未經真實後端／瀏覽器驗證（本里程碑：
    stub fetch 對映測試、展示資料守衛測試與 SSR 測試通過；新增
    成員對話框的開啟／挑選／確認／失敗重試、離開確認對話框、
    新增後成員重查未在瀏覽器執行；雙帳號實機流程——群組對話
    新增成員（管理員／非管理員）、一對一對話新增拒絕、離開對
    話與重新加入——待隔離環境，見 §2x 與 §4）。
27. Track A／Track B 未經真實服務／多帳號實機驗證（本里程碑：
    假 JetStream／假 Meilisearch HTTP／假物件儲存單元測試通過；
    真實 NATS 整合測試（3 項）、真實 MinIO presigned 往返測試、
    Relay 與消費迴圈的 api 行程實機啟動、索引設定預建、PG 全量
    掃描重建、雙帳號下載流程皆待隔離環境，見 §2y／§2z 與 §4）。
28. 位元組上傳通道 Web 接線未經真實後端／瀏覽器驗證（本里程碑：
    stub fetch 對映測試、展示資料位元組流程測試、SSR 測試與
    對比度／響應式靜態稽核通過；上傳流程（選檔、PUT、進度、
    完成、重試）、附件下載（新分頁／Blob 儲存、掃描拒絕、無
    位址）未在瀏覽器執行；雙帳號實機流程——MinIO presigned
    上傳往返、Stat 校驗失敗拒絕完成、中繼模式上傳／下載、簽
    名逾期重新核發——待隔離環境，見 §2aa 與 §4）。
29. 社群動態（Timeline）未經真實資料庫／多帳號實機驗證（本里程碑：
    記憶體儲存＋假驗證器測試通過；時間線過濾、按讚／留言／編輯／
    刪除的雙帳號實機流程、migration 000012 套用／回滾、PG 持久化
    整合測試皆待隔離環境，見 §2ab 與 §4）。
30. 社群動態 Web 接線未經真實後端／瀏覽器驗證（本里程碑：
    stub fetch 對映測試、展示資料操作測試、內容檢查純函式單元
    測試與 SSR 測試通過；發佈表單、媒體挑選、按讚切換、留言／
    回覆／刪除、編輯框、確認對話框、載入更多的互動未在瀏覽器
    執行；雙帳號實機流程——能見度過濾、按讚／留言／編輯／刪除、
    媒體關聯顯示——待隔離環境，見 §2ac 與 §4）。
31. 通話會話（Call）未經真實資料庫／LiveKit／多帳號實機驗證
    （本里程碑：記憶體儲存＋假驗證器測試通過；一對一／群組
    發起、接聽／拒接／忙線／未接／結束的雙帳號實機流程、
    migration 000013 套用／回滾、PG 持久化整合測試、LiveKit
    房間建立與 Token 簽發、Gateway 信令扇出（call.incoming／
    call.accepted／call.ended）皆待隔離環境與 Phase 07 後續，
    見 §2ad 與 §4）。
32. 通話 Web 接線未經真實後端／瀏覽器驗證（本里程碑：
    stub fetch 對映測試、展示資料狀態機測試、SSR 測試與純函式
    測試通過；發起挑選、接聽／拒接／忙線／結束確認、載入更多
    的互動未在瀏覽器執行；雙帳號實機流程——發起、接聽、拒接、
    忙線、未接、結束、參與者與事件顯示、來電橫幅——待隔離環境，
    見 §2ae 與 §4）。
33. 貼圖（Sticker）後端與 Web 接線未經真實資料庫／瀏覽器驗證
    （本里程碑：記憶體儲存＋假驗證器測試、stub fetch 對映測試、
    展示資料守衛測試、SSR 測試與純函式測試通過；瀏覽器互動——
    分類篩選、搜尋送出、收藏切換、建立／編輯／刪除／新增／
    移除對話框流程、載入更多——未執行；雙帳號實機流程——建包／
    上架／加貼圖、授權發放、收藏／取消、最近使用、搜尋、刪除——、
    migration 000014 套用／回滾、PG 持久化整合測試皆待隔離環境，
    見 §2af、§2ah 與 §4）。
34. ETHER AI Provider Adapter 未實作、未驗證（本里程碑僅規格盤點，
    見 §2ai：核心介面語義未定稿，未寫功能程式碼；無任何 AI 端點、
    模型呼叫、用量紀錄；Ollama／vLLM／外部 AI 皆未連線驗證）。
35. 社群（Community）後端未實作、未驗證（本里程碑僅規格盤點，
    見 §2aj：核心管理語義未定稿，未寫功能程式碼；無任何社群端點、
    migration、Web 接線；`internal/community` 僅 doc.go 佔位）。

待決定（M1 以註解＋本檔明載暫行選擇，不得視為定稿）：

1. `KEYCLOAK_REALM` 預設 `"ether"`（Realm 隔離方式未定稿，IDENTITY §12 缺口 11）。
2. 驗證錯誤用 HTTP 400（400／422 二選一未定稿，ERROR_CODES §8 缺口 2）。
3. 裝置端點掛 `/api/v1/auth`（/auth 或 /users 二選一未定稿，API_STANDARDS §5.3）。
4. `users` 不建 `password_hash`（密碼由 Keycloak 承接；ERD 草案欄位與此衝突，待 ERD 定稿覆核）。
5. 驗證長度上限（display_name 64、bio 500 等）為 M1 實作預設，待端點 Schema 定稿。
6. 欄位錯誤 `issue` 詞彙（REQUIRED／TOO_LONG 等）為 M1 實作詞彙，待錯誤碼定稿。
7. 已消耗 Refresh 雜湊持久化：M2b 已決，以 `session_refresh_hashes` 表
   （000004）持久化；ERD 同步（補該表定義）待定稿覆核。
8. Refresh 輪換的併發嚴格序列化未實作（Rotate 為「先標記、後寫入」，
   無交易＋`SELECT FOR UPDATE`；併發輪換下最後寫入者勝出，舊雜湊仍
   標記為已消耗。隔離環境驗證時須以交易強化或接受此限制，待決定）。
9. api 啟動不自動套用 migration（M2b 決定，明確操作步驟 `cmd/migrate`）；
   `IDENTITY_STORE` 預設 `postgres`（M2b 決定，fail-fast）。
10. 好友儲存語義（M3 決定，見 000005 檔頭，待 ERD 定稿覆核）：
    `friendships` 單列字典序配對＋CHECK；待處理邀請部分唯一索引；
    雙向同時待處理以後送出者收重複錯誤為準；狀態值域
    （active／ended、pending／accepted／declined／cancelled／expired）
    為 M3 實作詞彙；`blocked_at`／`hidden_at` 與 `created_at` 同值保留。
11. 聯絡人同步未實作（M3 決定）：雜湊演算法、授權撤回流程、推薦
    規則待隱私文件定稿；同步結果只作推薦輸入、不得自動加好友
    （MODULE_BOUNDARIES.md §5.5）。
12. Web Token 儲存（M2d 暫行選擇，待 SECURITY_ARCHITECTURE／身分
    文件定稿覆核）：Access Token 與 Refresh Token 一律只存記憶體
    （MemoryTokenStore＋OidcSession 行程內狀態），不寫入
    localStorage／sessionStorage／Cookie；分頁關閉或重新整理後須
    重新登入。localStorage 暫行作法（M2a）已退場，類別僅保留供
    既有測試編譯，不得接線。HttpOnly Cookie／BFF 方案未定，不得
    視 memory-only 為最終定稿；Keycloak end_session_endpoint 登出
    （RP-Initiated Logout）未實作，列為後續事項。
13. 訊息序號併發策略（M5 實作預設，待定稿）：`MAX+1` 子查詢＋
    唯一衝突重試上限 3 次；未以 `SELECT FOR UPDATE` 或序列
    序列化，高併發寫入同一對話時可能回伺服器錯誤。訊息列與
    `message_replies` 列未包在同一交易（儲存介面未提供交易邊界）。
14. 訊息端點詞彙與上限（M5 實作預設，待 API／ERD 定稿覆核）：
    內容 1–4000 runes、去重鍵 ≤128 字元、`kind` 僅 `text`、狀態
    僅 `sent`／`recalled`、游標為序號（`before_seq`／`since_seq`）、
    對話類型／角色詞彙（direct／group、member／admin）、回條
    device 粒度、群組內封鎖語義、`message_replies` 以表承載
    （ERD 欄位或表待定稿）。
15. Transactional Outbox 採用與否待定稿（DATA_FLOW.md §3）：
    M5 不實作 outbox；訊息寫入後經 Broker 發佈推播事件的中繼
    接線為後續里程碑，含 Gateway 訂閱成員身份驗證（待 api 整合）
    與 NATS JetStream 轉接器落地（須新增依賴，待授權）。
16. E2EE 明確未實作（M5 邊界）：不撰寫、不拼裝、不宣稱任何
    密碼學功能；`content` 一律視為不透明字串；`encryption_mode`
    欄位未建（待 E2EE 文件定稿）；金鑰分發與棘輪狀態不在範圍；
    Signal Protocol 為唯一候選，不得自創協定。
17. 群組端點詞彙與上限（M6 實作預設，待 API／ERD 定稿覆核）：
    名稱 1–64 runes、介紹 ≤500 runes、頭像媒體識別碼 ≤128 runes、
    公告內容 1–1000 runes、角色值域（owner／admin／member）、
    邀請狀態值域（pending／accepted／declined／cancelled）、
    設定鍵目前僅 invite_policy（admin／all）；group_roles 表與
    自訂角色、連結／QR 邀請、到期、禁言／群內暱稱、管理紀錄
    皆屬後續里程碑；群組與 conversations 一對一對應為本里程碑
    實作決定（ERD §10 缺口仍待文件定稿）；跨模組寫入無交易
    邊界（見 §2i 已知限制）。
18. 媒體端點詞彙與上限（媒體模組實作預設，待 API／ERD 定稿覆核）：
    單檔上限 100 MiB、MIME 白名單（圖片 4 種／影片 3 種／音訊
    4 種／PDF）、原始檔名 ≤255 runes、校驗值 ≤128 字元（演算法
    待定稿）、掃描／上傳／存取／縮圖變體值域（見 §2k）、
    duration 取整數秒（0＝未知）；物件鍵格式
    `media/<ownerID>/<fileID>`（縮圖加 `/thumb/<variant>`）為本
    里程碑實作決定，Bucket 劃分與生命週期策略待部署文件定案；
    掃描供應商／方式、縮圖產生方式、IP／User-Agent 雜湊方式、
    存取紀錄全量／取樣策略、保留期與孤兒清理皆待定稿；上傳完成
    時詮釋資料由呼叫方再次提供（會話列不存詮釋資料）；跨模組寫
    入無交易邊界（檔案列先建、會話列後寫）。
19. 位元組上傳通道已決定（2026-11-25 雙軌指令，見 §2z，待架構／API 定稿覆核）：presigned URL 直傳 MinIO；原先未定的中繼／分段選項不再考慮，除非定稿覆核推翻。
    Web 接線已完成（見 §2aa：進度回報與完成上傳已接線，縮圖
    登記仍刻意未接線——會宣稱縮圖已產生，屬假資料，不得實
    作）；掃描器接線與 MinIO 轉接器落地（須新增依賴，待授權）
    同屬後續事項。
20. Outbox／Relay 實作預設與事件缺口（本里程碑決定，待 API／ERD／
    事件協定定稿覆核）：`message_outbox` 表名與欄位（狀態值域
    pending／processing／published／dead、claimed_at 租約 5 分鐘、
    (status, created_at) 認領索引）為實作決定；Relay 批次上限 100、
    重試上限 5 次、輪詢間隔 1 秒為實作預設；Claim 以
    `FOR UPDATE SKIP LOCKED` 搶佔；Edit／DeleteForSelf 不寫 Outbox
    事件（WEBSOCKET_PROTOCOL.md §21 無對應事件名，不得自創，待
    事件協定定稿補事件名或確認以 REST 歷史同步為準）；Gateway
    MembershipChecker 的 api 側實作方式（內部端點＋服務認證，或
    事件投影）待定稿；NATS JetStream 發佈者落地須新增依賴，待
    授權；Relay 迴圈的落點（api 行程內或 Worker）待部署定案；
    Outbox 事件 payload 含 content（E2EE 未實作，E2EE 落地後須
    改為密文，不得再含明文）。
21. 反應與置頂實作預設與事件缺口（本里程碑決定，待 API／ERD／
    事件協定定稿覆核）：反應允許清單 8 種表情、每人每訊息至多
    一則生效反應（更換標記舊列）、置頂上限 5 則、彙總排序
    （人數降序／表情升序）、`message_reactions`／`message_pins`
    表名與欄位（取消皆為標記、部分唯一索引保證生效唯一性）皆為
    實作決定；WEBSOCKET_PROTOCOL.md §21 無反應／置頂事件名，
    反應／置頂寫入不寫 Outbox 事件，客戶端以 REST 同步為準，待
    事件協定定稿補事件名或確認 REST 同步為準；置頂上限檢查與
    寫入無序列化，併發下可能短暫超限，待定稿覆核。
22. 附件／轉傳實作預設與事件缺口（本里程碑決定，待 API／ERD／
    事件協定定稿覆核）：附件關聯上限每訊息 10 筆、轉傳鏈深度上限
    5、附件關聯與轉傳紀錄皆 append-only、參照方式只以
    `media_file_id` 直接 FK（不存 storage_path／thumbnail_media_id）、
    附件詮釋資料關聯時快照、轉傳不帶附件與引用（原附件不得逕行
    連結至目標對話，重新授權流程待定稿）、轉傳新訊息以既有
    message.received 事件承載（§21 無轉傳獨立事件名，不得自創）、
    `message_attachments`／`message_forwards` 表名與欄位（含
    (message_id, media_file_id) 唯一索引與 forwarded_message_id
    唯一索引）皆為實作決定；轉傳紀錄列寫入在訊息列提交之後、
    併發深度／上限檢查無序列化，皆為已知限制，待定稿覆核；
    重新授權流程、轉傳鏈深度語義、附件上限值皆待定稿。
23. 訊息搜尋實作預設與事件缺口（本里程碑決定，待 API／ERD／
    事件協定定稿覆核）：比對語義（AND、子字串、大小寫不敏感、
    停用詞不預設）、排序（建立時間降序、訊息識別碼降序
    tiebreak）、游標編碼（排序鍵 base64）、limit 預設 20／上限
    50、q 上限 200 runes、時間區間含邊界皆為實作決定；索引
    文件欄位（僅 wire 欄位）與 `Document` 結構為實作決定；
    編輯內容不經事件更新索引（§21 無編輯事件名，待事件協定
    定稿補事件名或確認重建為準）；媒體／檔案類型篩選未實作
    （須待事件 payload 擴充）；Meilisearch 索引結構、排名／分詞
    語義、重建流程待轉接器落地與維運文件定案；Worker 消費
    迴圈落點與投遞語義（至少一次＋冪等）待 NATS 落地定案。
24. 訊息搜尋 Web 接線實作預設（本里程碑決定，待定稿覆核）：
    面板每頁 20 筆（與後端 limit 預設一致）；跳轉錨定載入以
    `sinceSeq` 取目標訊息前後一頁（`HISTORY_PAGE_LIMIT` 30）
    再合併；醒目提示僅外框、切換對話清除；空關鍵字面板端攔
    截、展示端拒絕（與後端 q 必填一致）；日期篩選以本地日界
    轉 ISO（起始 00:00:00、結束 23:59:59.999）；游標編解碼與
    後端同一格式（排序鍵 base64url），跨頁新寫入的重複／遺漏
    語義待定稿。
25. 對話成員管理實作預設（本里程碑決定，待 API／ERD 定稿覆核）：
    成員區塊同時列出已離開成員（後端回應順序，已離開者列於
    末尾）；新增人選排除既有生效成員與自己；離開後果文案（一
    對一對話可經建立對話重新開啟、群組對話須管理員重新新增）
    依後端 CreateDirect／AddMembers 語義；展示模式新增成員以一
    對一守衛拒絕（展示資料無群組對話）；移除他人成員未實作
    （無後端端點）。
26. Track A 實作預設（本里程碑決定，待 API／事件協定／部署文件定稿覆核）：
    Stream 名 `ETHER_EVENTS`、subject 前綴 `ether.events.`、durable／
    group `ether-search-indexer`、AckWait 30 秒、MaxDeliver 5、索引
    UID 與文件欄位、索引設定預建方式、發佈者不自動建 Stream、
    毒訊息超 MaxDeliver 後靠日誌發現；Relay 與消費迴圈承載於 api
    行程（Worker 行程接線須抽共用套件或合併模組，屬架構決策，
    待重構方案定案）；部署預設開關值（`RELAY_ENABLED`／
    `SEARCH_CONSUMER_ENABLED` 預設關閉）待部署文件定案；PG 掃描
    實作與重建 CLI 落點待定案。
27. Track B 實作預設（本里程碑決定，待 API／ERD／部署文件定稿覆核）：
    presigned 直傳通道；URL 有效期 15 分鐘；簽名區域 us-east-1；
    單次 PUT（非分段）；中繼端點僅開發／測試用；記憶體模式下載
    URL 為相對路徑；`upload_url_expires_at`／`download_url_expires_at`
    欄位名；413 錯誤碼 `ETHER_COMMON_PAYLOAD_TOO_LARGE`；中繼下載
    未通過回 403；CDN 前綴未接線（現核發 MinIO 直連 URL）。
28. 位元組上傳通道 Web 接線實作預設（本里程碑決定，待定稿覆核）：
    單次 PUT 分段數回報為 1；PUT 403 時重新核發一次並重試一次，
    其餘失敗不重試；絕對下載 URL 以新分頁開啟、相對 URL 經認證
    fetch 取位元組後儲存；展示 pseudo-URL（`demo://`）僅本機
    流程語義；上傳進度以階段文字顯示（fetch 無上傳進度事件）。
29. 社群動態（Timeline）實作預設（本里程碑決定，待 API／ERD 定稿覆核）：
    能見度 public／friends／only_me、空值預設 friends（指定好友
    可見屬後續）；內容上限貼文 2000／留言 1000 runes；單則貼文
    媒體上限 9 筆（去重保序、經 Media 掃描閘門）；分頁預設
    20（上限 50）、留言預設 30（上限 100）；游標為 (created_at,
    id) 排序鍵 base64url；貼文編輯記 edited_at、編輯歷史不保留；
    留言編輯／分享／收藏／檢舉／置頂／限時動態／隱藏皆未實作；
    動態無 WebSocket 事件名（§21 缺口），不寫 Outbox 事件；貼文
    媒體詮釋資料不快照（與 M10 附件語義不同）；父留言刪除後子
    留言保留（顯示規則待定稿）。
30. 社群動態 Web 接線實作預設（本里程碑決定，待定稿覆核）：
    回覆標籤只認已載入留言（已刪除父留言顯示「回覆已刪除的
    留言」）；作者名稱以好友清單解析、未知作者顯示識別碼前 8
    字元；媒體挑選上限 9 筆、僅 clean 檔案可選；貼文媒體只作
    詮釋資料顯示、不設下載按鈕；導覽圖示為新增的 FeedIcon。
31. 通話會話（Call）實作預設（本里程碑決定，待 API／ERD／信令
    協定定稿覆核）：狀態 ringing／active／declined／missed／
    busy／ended、類型 voice／video、參與者結果 pending／joined／
    declined／busy／missed、事件 initiated／answered／declined／
    busy／ended、結束原因 hangup／declined／missed／busy；清單
    分頁預設 20（上限 50）、游標為 (created_at, id) 排序鍵
    base64url；同一範圍重複發起拒絕；未接只由「響鈴中結束且無
    人接聽」判定（逾時自動轉未接須排程／Worker，屬後續）；事件
    payload 只含 from_status／to_status／reason；不建
    livekit_room_ref 與 device_id 欄；通話不寫 Outbox 事件；
    SignalCoordinator 只留佔位介面；通話紀錄保留期限待決定。
33. 貼圖（Sticker）實作預設（本里程碑決定，待 API／ERD 定稿覆核）：
    包狀態 draft／published／unlisted／archived、貼圖狀態
    active／removed、分類七種詞彙、來源 favorite／granted／
    recent；單包生效貼圖上限 100 筆、最近使用保留 30 筆；分頁
    預設 20（上限 50）、游標為排序鍵 base64url（取最後一筆回傳
    列）；搜尋 token 同時比對貼圖名與包名的聯集；授權發放只經
    GrantPack／RevokeGrant 介面（無 HTTP 端點）；封存包的取消
    收藏為清理操作（曾收藏者回包檢視）；貼圖包管理僅作者（管理
    員角色待定）；貼圖不寫 Outbox 事件（§21 無事件名）。Web 接線
    實作預設（§2ah 決定，待定稿覆核）：搜尋為瀏覽頁籤內模式
    （送出後以貼圖結果取代包清單）、分類篩選無「僅未分類」選項、
    付費可見包標示「付費 · 已授權」（wire 無授權旗標，可見性由
    後端強制）、封面以既有媒體挑選器選取（僅 clean 檔案可選）、
    client 搜尋空關鍵字於發請求前拒絕、`StickerIcon` 為圓角方塊
    ＋笑臉佔位圖形（待設計審查）。
34. ETHER AI Provider Adapter 缺口（§2ai 盤點結論，全部待定稿，
    不得自行發明）：AI_DATA_POLICY 批准；Adapter 介面名、請求／
    回應型別、串流事件格式（API 文件定稿）；Ollama／vLLM／LocalAI
    分工與 API shape；模型清單、預設模型、備援順序、切換權限；
    逾時／重試／退避數值與錯誤對映；Token Budget／Rate Limit 數值；
    外部供應商選擇與 Key 管理；E2EE 例外授權落實方式；Tool 白名單
    與參數驗證規則；使用紀錄表與保留期限；向量維度／距離度量／
    索引更新方式；CPU／RAM／GPU／VRAM 實測數據。
35. 社群（Community）缺口（§2aj 盤點結論，全部待定稿，不得自行
    發明）：分級權限等級與操作對照表（等級名詞、層級數、操作對照、
    任命／降級流程）；`visibility`／`category`／社群狀態／邀請狀態
    值域；`role` 與 `role_id` 主從與 `permissions` 形態；邀請／審核
    狀態機（誰可邀請／審核、過期與重發）；頻道 `channel_type` 值域
    與 `conversations` 對應方式；成員生效期唯一性；公告／規則／
    討論／活動／投票／操作紀錄的承載設計；資料分析指標定義；
    搜尋語義與端點 Schema；SPEC-SOURCE §09 原文。

## 4. 下一步

- 一次性隔離整合測試環境（§2ak 草稿）：下一步為 Repository 建立後，
  將 workflow 檔案經網頁建立為 `.github/workflows/` 正式檔案（API
  推送會回 403），並查證填入 checkout／upload-artifact 的完整 commit
  SHA；首次執行前重新核對帳務額度與 EXECUTION_PLAN §9.1 前置檢查，
  執行後以實際證據填寫驗收報告範本（不得將未執行項標 PASS）。
- ETHER AI Provider Adapter 實作前置（§2ai 盤點結論）：須先定稿
  AI_DATA_POLICY 批准、Adapter 介面名與請求／回應／串流格式（API
  文件）、模型清單與預設、逾時／重試／預算數值、使用紀錄表與
  向量維度；定稿前不得開工 Provider 實作（含 Ollama 轉接器），
  不得以公開文件知識預寫三者 API shape。
- 社群（Community）後端實作前置（§2aj 盤點結論）：須先定稿分級
  權限等級與操作對照表、`visibility`／`category`／狀態／邀請狀態
  值域、`role` 與 `role_id` 主從、頻道與 `conversations` 對應方式、
  邀請／審核狀態機、公告／規則／討論／活動／投票／操作紀錄的
  承載設計、端點 Schema；定稿前不得開工 Community 領域實作（含
  migration），不得以群組里程碑的實作預設類推社群等級已定。
- M2b／M3／M2d／M5／M6 實機驗證（需隔離環境）：`cmd/migrate up`（000001–000014）＋
  整合測試＋Keycloak 實機登入驗證（清單見 KEYCLOAK_INTEGRATION.md §4，
  含 Web 端授權導向、回呼交換、換發、登出實測）；好友三帳號互搜
  互邀、封鎖、解封、刪除的實機驗收（ROADMAP 手冊 F）；對話／訊息
  雙帳號實機流程（建 DM／群組、發送、編輯、收回、回條、游標續拉、反應、置頂）與
  Gateway 真實連線扇出驗證（ROADMAP 手冊 G）；群組三帳號實機流程
  （建群組、邀請／接受、角色變更、公告、所有權轉移、解散）與
  migration 000007 套用／回滾驗證（ROADMAP 手冊 H），含群組 Web
  接線雙模式實機流程（§2j）；媒體實機流程（上傳→完成→掃描→授權→
  縮圖）與 migration 000008 套用／回滾驗證，含 MinIO 實機上傳
  （須 MinIO 轉接器落地與依賴授權，見 §2k）與媒體 Web 接線雙模式
  實機流程（§2l／§2aa，含真實位元組上傳、簽名逾期重試與下載）；反應／置頂
  實機流程與 migration 000010 套用／回滾驗證（§2p），含反應／置頂
  Web 接線雙模式實機流程（§2q）；附件關聯與轉傳實機流程與
  migration 000011 套用／回滾驗證（§2s），含附件／轉傳 Web 接線
  雙模式實機流程（§2t）。
 - 訊息推播中繼離線部分已完成（§2m），NATS 發佈者與索引填充迴圈
  已落地（§2y）；後續：Relay 與消費迴圈的 api 行程實機啟動、Worker
  行程接線重構方案定案、Gateway MembershipChecker 的 api 側成員查詢
  整合、雙帳號實機扇出驗證（建 DM、發送、訂閱收到 message.received、
  非成員訂閱拒絕），皆待隔離環境與定稿。
- 已讀回條 Web 接線已完成（§2n）；實機驗收（開啟對話送 read 回條、
  未讀數與側邊列徽章同步）併入對話／訊息雙帳號實機流程，待隔離環境。
- delivered 回條與回條列表 Web 接線已完成（§2r）；實機驗收（歷史載入
  送 delivered 回條、回條列表查詢、自己訊息已送達／已讀標示）併入
  對話／訊息雙帳號實機流程，待隔離環境。
- 附件／轉傳 Web 接線已完成（§2t）；實機驗收（附件顯示與掃描狀態、
  轉傳至另一對話、轉傳後目標對話收到新訊息、附件新增守衛）併入
  對話／訊息雙帳號實機流程，待隔離環境；附件下載實機驗收併入
  媒體雙帳號流程（見 §2aa）。
- 編輯／收回／僅自己刪除 Web 接線已完成（§2u）；實機驗收（編輯
  內容與已編輯標示、收回後對方檢視、僅自己刪除後對方檢視不受
  影響、非作者不得操作）併入對話／訊息雙帳號實機流程，待隔離環境。
 - 訊息搜尋後端離線部分（§2v）與 Web 接線（§2w）已完成，轉接器與
  填充迴圈已落地（§2y）；後續：PG 掃描重建實作與 CLI 落點定案、
  索引設定預建，實機驗收（關鍵字／日期／發送者篩選、非成員過濾、
  收回後內容不殘留、游標分頁、跳轉至舊訊息）併入對話／訊息雙帳號
  實機流程，待隔離環境。
- 建立對話 UI 已完成（§2o）；實機驗收（選好友建 DM、新建與既有
  對話流程、建立失敗重試）併入對話／訊息雙帳號實機流程，待隔離環境。
- 對話成員管理 Web 接線已完成（§2x）；實機驗收（群組對話新增成員、
  一對一對話新增拒絕、離開對話與重新加入、成員清單顯示）併入對話／
  訊息雙帳號實機流程，待隔離環境。
- 媒體位元組上傳通道 Web 接線已完成（§2aa）；實機驗收（MinIO
  presigned 上傳往返、Stat 校驗、簽名逾期重新核發、中繼模式上傳／
  下載、附件下載）併入媒體雙帳號實機流程，待隔離環境。
- 社群動態（Timeline）後端離線部分（§2ab）與 Web 接線（§2ac）已完成；
  雙帳號實機流程（能見度過濾、按讚／留言／回覆／編輯／刪除、媒體
  關聯顯示、migration 000012 套用／回滾）待隔離環境。
- 通話會話（Call）後端離線部分（§2ad）與 Web 接線（§2ae）已完成；後續：
  LiveKit 房間建立與 Token 簽發、Gateway 信令扇出（call.incoming／
  call.accepted／call.ended）、來電推播、雙帳號實機流程與 migration 000013
  套用／回滾，皆待隔離環境與 Phase 07 定稿。
- 貼圖（Sticker）後端離線部分（§2af）與 Web 接線（§2ah）已完成；
  後續：購買交易與商店展示（Payment Integration 定稿後）、雙帳號
  實機流程與 migration 000014 套用／回滾，皆待隔離環境。
- Timeline `feedPage` 游標缺陷已於 §2ag 修正並以回歸測試覆蓋
  （多頁續拉實機驗證併入 Timeline 雙帳號實機流程）。
- M4：Phase 04 剩餘（聯絡人同步、備註／分類、邀請連結／QR）——
  依賴隱私文件定稿（雜湊與授權撤回），不得先於定稿實作同步上傳。
- Token 儲存定稿覆核（memory-only 暫行選擇，見 §3 待決定 12）與
  Keycloak end_session_endpoint 登出，待身分／安全文件定稿後實作。
- 媒體位元組上傳通道後端已完成（§2z）；後續 Web 接線（上傳 URL 取用、完成呼叫時機、下載 URL 取用）與真實 MinIO 往返／雙帳號下載實機驗收，待隔離環境。
