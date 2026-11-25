狀態：草案／待批准

# ETHER API 命名與設計規範（API_STANDARDS.md）

## 1. 依據與效力

- 本文件以 `~/workspace/ether/SPEC-SOURCE.md` §21 為最高準據。API 前綴、WebSocket 事件名以 §21 為準，一字不得改。
- 功能深度補充參考 `~/workspace/messenger-app/MASTER_SPEC.md`；舊合約 `~/workspace/messenger-app/CONTRACTS.md` 僅供參考。舊路線的端點、頻道、事件與 ETHER §21 不同者，一律以 §21 為準，差異寫入 ADR。
- 本文件定規範，不定個別端點的完整 Schema。個別端點的 Request／Response Schema 須在各階段 API 文件中逐端點定稿，並由 OpenAPI 文件承載（見 §10）；本文件不定稿任何個別端點的欄位清單，不得以本文件的示例為已定合約。
- 不得編造額度、數字、效能指標。Rate Limit 數值、逾時秒數、頁大小上限等，一律寫「待定稿／待查證」，不得預寫具體數字作為承諾。舊規格中的數字（如 30 秒逾時）為舊路線（Supabase Realtime）已定值，不得直接搬入 ETHER WebSocket／REST 規範而不經定稿，見 §12 缺口。
- 本文件與 ERROR_CODES.md、WEBSOCKET_PROTOCOL.md 不得衝突。錯誤回應格式以 ERROR_CODES.md 為準，本文件只引用，不重寫；WebSocket 事件以 WEBSOCKET_PROTOCOL.md 為準，本文件只列前綴對應，不重寫事件定義。
- 架構調整（2026-11-25 00:33 使用者指令，優先於舊技術棧）：邊緣閘道改 Traefik、身分提供者改 Keycloak（見 §5.4）。其餘規範不變；舊付費服務名不得殘留。

## 2. 版本化前綴（§21 全部 18 個，一字不得改）

SPEC-SOURCE.md §21 所列前綴原文如下（依原文順序）：

1. `/api/v1/auth`
2. `/users`
3. `/profiles`
4. `/friends`
5. `/contacts`
6. `/conversations`
7. `/messages`
8. `/groups`
9. `/communities`
10. `/media`
11. `/stickers`
12. `/posts`
13. `/calls`
14. `/notifications`
15. `/search`
16. `/official-accounts`
17. `/bots`
18. `/admin`

### 2.1 前綴解讀（待定稿確認）

§21 原文第一項寫為 `/api/v1/auth`，其餘 17 項僅寫資源名（`/users`、`/profiles` 等），未逐項寫出 `/api/v1`。本文件作如下解讀，**待 Phase 00 審查確認**，確認前不得以本解讀為已定：

- 完整路徑形式為 `/api/v1` ＋ §21 所列資源前綴（第一項除外，其本身已含 `/api/v1`）。例如：第 2 項完整形式為 `/api/v1/users`，第 18 項為 `/api/v1/admin`。
- 如審查決定其餘 17 項不掛 `/api/v1`（即字面路徑即為 `/users` 等），須修訂本節並同步修訂 OpenAPI 文件，不得以本文件解讀擅自定稿。
- 版本號 `v1` 為 API 主版本。主版本升級規則見 §11。

### 2.2 前綴與模組對應（草案）

前綴與 SPEC-SOURCE.md 22 個後端模組的對應為草案方向，定稿時覆核；前綴本身不得改名，模組邊界見架構文件（另冊，本文件不重寫）。

| 前綴 | 主要對應模組（草案） | 備註 |
|---|---|---|
| `/api/v1/auth` | Authentication、Device Management | 註冊、登入、Token、MFA、Session、裝置信任相關。 |
| `/api/v1/users` | User | 帳號、停用、刪除、資料匯出相關。 |
| `/api/v1/profiles` | Profile | 公開檔案、QR Code、分享連結相關。 |
| `/api/v1/friends` | Friend | 好友邀請、好友關係、封鎖、隱藏、備註相關。 |
| `/api/v1/contacts` | Contact | 聯絡人同步、推薦相關；同步須明確授權。 |
| `/api/v1/conversations` | Chat | 聊天室、成員、設定、置頂、封存相關。 |
| `/api/v1/messages` | Message | 訊息送出、編輯、收回、反應、回條、搜尋（對話內）相關。 |
| `/api/v1/groups` | Group | 群組、成員、角色、邀請、公告、設定相關。 |
| `/api/v1/communities` | Community | 社群、成員、頻道、角色、邀請相關。 |
| `/api/v1/media` | Media | 媒體上傳、下載、縮圖、存取相關。 |
| `/api/v1/stickers` | Sticker | 貼圖包、貼圖、使用者貼圖、購買紀錄相關。 |
| `/api/v1/posts` | Timeline | 動態貼文、媒體、按讚、留言、分享、收藏相關。 |
| `/api/v1/calls` | Call | 通話紀錄、參與者、事件相關；即時信令走 WebSocket／LiveKit，見 WEBSOCKET_PROTOCOL.md。 |
| `/api/v1/notifications` | Notification | 通知、偏好、推播權杖、投遞相關。 |
| `/api/v1/search` | Search | 全域搜尋相關；須遵守存取權限與封鎖規則。 |
| `/api/v1/official-accounts` | Official Account | 官方帳號、成員、客服相關。 |
| `/api/v1/bots` | Bot | Bot 申請、Webhook、事件、開發者相關。 |
| `/api/v1/admin` | Administration、Moderation、Analytics | 管理後台專用；獨立管理員認證與權限，不得與一般使用者認證混用。 |

未在上表出現的模組（Payment Integration、Analytics 的部分能力）是否有獨立前綴，§21 未定，不得擅自新增前綴；如需新增，須修訂 §21 並以 ADR 記錄，見 §12 缺口。

## 3. REST 命名規則

### 3.1 一般規則

- 一律使用 HTTPS。HTTP 明文不得用於正式環境；開發環境的明文使用須在部署文件中明定並標示風險，不得默許。
- 資源名一律小寫、`kebab-case`（多詞資源以連字號連接，例如 `official-accounts`，以 §21 為準）。路徑中不得使用大寫、底線、動詞（動作以 HTTP 方法＋子資源表達，例外見 §3.3）。
- 集合以複數名詞命名（例如 `/conversations`）；單一資源以集合＋識別碼命名（例如 `/conversations/{conversation_id}`）。
- 識別碼一律以路徑參數 `{<resource>_id}` 命名，名稱須與資源對應（例如 `{conversation_id}`、`{message_id}`、`{group_id}`），不得以泛稱 `{id}` 混用多種資源。識別碼格式（UUID 或其他）待定稿，定稿前不得假設。
- 查詢參數一律 `snake_case`（例如 `cursor`、`limit`、`created_before`，具體參數名待各端點定稿）。
- 路徑末端不得有斜線（trailing slash）；有斜線與無斜線不得視為同一路徑而不說明，定稿時明定正規化規則。

### 3.2 HTTP 方法語義

| 方法 | 語義 | 冪等性 |
|---|---|---|
| `GET` | 讀取單一資源或集合，不得有副作用。 | 冪等 |
| `POST` | 建立資源，或執行非冪等的動作（動作型子資源，見 §3.3）。 | 建立時以 Idempotency Key 保證去重（見 §7）；動作型須在端點文件中明定是否冪等。 |
| `PATCH` | 部分更新資源。未提供的欄位不得被重設。 | 冪等（同一請求重複執行結果一致） |
| `PUT` | 僅在端點文件明定全量替換語義時使用；預設不使用 `PUT`。不得以 `PUT` 代替 `PATCH` 而不說明。 | 冪等 |
| `DELETE` | 刪除或取消資源。刪除語義（軟刪除／狀態標記／硬刪除）須與 ERD.md 的刪除策略一致，並在端點文件中明定；不得以 `DELETE` 假裝物理刪除而實際僅標記，反之亦然。 | 冪等（重複刪除同一已刪除資源不得報錯為「不存在」以外的誤導錯誤，具體行為待定稿） |

- 不得使用 `GET` 執行寫入、不得使用 `POST` 執行純讀取查詢而規避快取與冪等語義（搜尋的 `POST` 例外須經定稿並說明理由）。
- 不得新增 `HEAD`、`OPTIONS` 以外的自訂方法；CORS 預檢由框架處理，不得在業務端點中手寫。

### 3.3 動作型子資源

- 動作以 `POST /<集合>/{id}/<動作>` 形式表達，動作為 `kebab-case` 動詞名詞，例如（示例，非已定端點）：`POST /conversations/{conversation_id}/members`（新增成員為集合建立，非動作）、`POST /messages/{message_id}/recall`（收回為動作，示例）。
- 本文件所舉示例僅說明命名形式，**不是已定端點**；個別動作端點的路徑、方法、Schema 須在各階段 API 文件中定稿。
- 動作型端點須在端點文件中明定：是否冪等、重複呼叫的結果、狀態機前置條件、錯誤碼（見 ERROR_CODES.md）。

### 3.4 巢狀資源

- 巢狀深度不得超過兩層（例如 `/conversations/{conversation_id}/messages` 為一層巢狀，尚可；更深巢狀須以頂層前綴＋查詢參數代替，並經定稿）。
- 巢狀資源的父識別碼須經授權檢查（呼叫者須有權存取父資源），不得僅檢查子資源識別碼。
- 屬於多個父資源的資源（例如訊息附件同時屬於訊息與媒體），以一個主前綴為準，另一前綴以查詢參數或參照欄位表達，不得建兩套路徑而不明主從。主從關係待各端點定稿。

### 3.5 集合與單一資源的路徑形式（示例）

以下僅為形式示例，非已定端點：

- `GET /api/v1/conversations`：列出呼叫者的聊天室（Cursor 分頁，見 §6）。
- `POST /api/v1/conversations`：建立聊天室。
- `GET /api/v1/conversations/{conversation_id}`：讀取單一聊天室。
- `PATCH /api/v1/conversations/{conversation_id}`：部分更新聊天室。
- `DELETE /api/v1/conversations/{conversation_id}`：刪除／退出聊天室（語義見端點文件）。
- `GET /api/v1/conversations/{conversation_id}/messages`：列出聊天室訊息（Cursor 分頁）。

## 4. Request／Response Schema 慣例

### 4.1 通用慣例

- 內容類型一律 `application/json`（媒體上傳的 `multipart/form-data` 例外，見 §4.4）。請求與回應皆須以 UTF-8 編碼。
- JSON 欄位名一律 `snake_case`。不得混用 `camelCase`。時間欄位一律 ISO 8601（UTC，含時區標示），不得以本地時間、Unix 秒數、毫秒數混用而不說明；定稿時明定時間格式與精度。
- 識別碼欄位名須與資源對應（例如 `conversation_id`、`message_id`），不得以泛稱 `id` 作為關聯欄位名；資源本身的主鍵欄位名為 `id`（草案，與 ERD.md 一致）。
- 不得回傳密碼明文、Token 明文、Bot 金鑰明文、私鑰、信用卡完整資訊。雜湊值、遮罩值亦不得無故回傳，回傳必要性待各端點定稿。
- 不得回傳未經授權的欄位：回應 Schema 須按呼叫者權限裁剪（例如封鎖關係、他人私密資料），裁剪規則在端點文件中明定，不得以「前端不顯示」代替後端裁剪。
- 未知欄位：請求中含未定義欄位時，處理方式（忽略或拒絕）待定稿，全 API 一致，不得各端點自行決定。
- 空值語義：`null` 表示「無值」，缺失欄位表示「未提供」（`PATCH` 時不得重設）；二者不得混用，定稿時明定。

### 4.2 成功回應

- 單一資源：回應體為資源物件本身，不得包多一層與資源名相同的鍵（例如不得回傳 `{"conversation": {...}}`，除非端點文件明定包裝規則並全 API 一致）。包裝規則待定稿，本文件不預寫。
- 集合：回應體須含資料陣列與分頁資訊（見 §6），結構待定稿，全 API 一致。
- 建立成功：`201 Created`，回應體為完整資源物件，並含 `Location` 標頭（草案，指向新資源路徑）。
- 更新／刪除成功：`200 OK`（回傳更新後資源或操作結果）或 `204 No Content`（無回應體）；二者的選用按端點文件定稿，不得各端點自行決定。
- 不得回傳與請求無關的除錯資訊、堆疊追蹤、內部識別碼。

### 4.3 錯誤回應

- 錯誤回應格式一律以 ERROR_CODES.md 為準，本文件不重寫。所有錯誤回應皆須含 `request_id`（見 ERROR_CODES.md）。
- HTTP 狀態碼與錯誤碼的對應見 ERROR_CODES.md；不得以 `200 OK` 包裝錯誤，不得以錯誤碼代替 HTTP 狀態碼而不回傳正確狀態碼。

### 4.4 媒體上傳

- 媒體上傳端點的請求格式（`multipart/form-data`、分段上傳、簽名 URL 直傳）待媒體模組定稿，本文件不定稿。無論何種方式，上傳前須經權限驗證、格式檢查、惡意檔案掃描（見 SPEC-SOURCE.md §07），掃描狀態須在回應或後續查詢中可查，不得宣稱未掃描的檔案已安全。
- 上傳回應須回傳媒體識別碼與詮釋資料，不得僅回傳儲存路徑；儲存路徑的暴露方式待定稿（簽名 URL 的有效期等不得預寫數字）。

### 4.5 欄位驗證與 Schema 定稿

- 每個端點的 Request／Response Schema 須在 OpenAPI 文件中以 JSON Schema 定稿（見 §10），含欄位名、型別、必填性、值域、長度／大小限制、示例（示例須為虛構資料，不得含真實使用者資料）。
- 長度／大小限制、值域清單待各端點定稿，不得以本文件預寫數字；舊規格中的暫定數字（如文字長度上限）不得直接搬入而不經定稿。

## 5. 認證與授權

### 5.1 認證

- 認證方式：以 Bearer Token（Access Token）為草案方向，Token 傳輸一律經 `Authorization: Bearer <token>` 標頭，不得以查詢參數、Cookie（管理後台例外，見下）、請求體傳輸 Access Token。Token 格式、簽發、驗證方式待身分文件定稿（手冊 E 方向：OAuth 2.0／OIDC、PKCE、短效 Access Token、Refresh Token Rotation，僅供參考，定稿時確認；身分提供者為 Keycloak，見 §5.4）。
- Refresh Token：輪換（Rotation）為草案方向；重放已輪換的 Refresh Token 的處理（撤銷 Session 家族等）待定稿，不得預寫。
- 未認證的請求一律回傳認證錯誤（見 ERROR_CODES.md），不得回傳部分資料或以預設身份放行。
- 管理後台（`/api/v1/admin`）：獨立管理員認證與權限，不得與一般使用者認證混用；管理員 Token 與使用者 Token 不得互通，驗證失敗的處理待定稿。
- Bot API（`/api/v1/bots` 的伺服器對伺服器呼叫）：以 Bot 金鑰認證為草案方向，金鑰傳輸、撤銷、輪換方式待 Bot 文件定稿；金鑰明文只在建立／輪換時顯示一次，不得再次查詢。
- 匿名端點（註冊、登入、公開搜尋等）清單待定稿；清單以外的端點一律須認證，不得以「公開」為由默許未認證存取。

### 5.2 授權

- 授權一律在後端 enforced，不得由前端自行判斷。管理員權限、群組角色權限、聊天室成員權限，皆以後端判定為準。
- 授權檢查點：每個端點須在端點文件中明定所需的權限（資源擁有者、成員、角色、管理員等），並在實作中於業務邏輯前檢查，不得在業務邏輯後補查。
- 封鎖規則：被封鎖者的請求（傳訊、通話、查看動態等）須被拒絕，且不得向被封鎖者揭露封鎖狀態（錯誤回應須與「不存在」不可區分，具體方式見 ERROR_CODES.md）。
- E2EE 聊天室：管理員不得任意讀取 E2EE 訊息（見 SPEC-SOURCE.md §19）；API 層不得提供繞過 E2EE 的讀取端點，E2EE 與一般聊天室的功能差異須在端點文件中明定。
- 權限變更即時生效，不得以快取的舊權限放行；快取如用，須明定失效規則。

### 5.3 裝置與 Session

- 多裝置登入、裝置信任、遠端登出、Session 撤銷的 API 語義待身分文件定稿；本文件只定：裝置相關端點屬 `/api/v1/auth` 或 `/api/v1/users` 前綴（定稿時二選一，不得兩處並存），Session 撤銷須即時生效，不得等待裝置下次連線。

### 5.4 閘道與身分提供者（架構調整後）

- 邊緣閘道為 Traefik：對外 TLS 終止、路由、限流（閘道層）、標頭處理等邊緣職責由 Traefik 承載，部署與憑證細節見 `docs/deployment/CLOUD_ARCHITECTURE.md` 與 `SELF_HOSTING.md`。API 服務本身不得假設閘道已完成認證或授權：認證與授權一律在 API 後端再驗證（見 §5.1、§5.2），閘道不得成為唯一授權邊界；閘道傳遞的識別標頭不得被 API 無條件信任，信任方式待定稿。
- 身分提供者為 Keycloak（OIDC／OAuth 2.0）：Email 驗證、登入／登出、OAuth（Google、Apple）、MFA、Session、Access／Refresh Token、裝置管理、遠端登出、密碼重設等身分能力由 Keycloak 承載，不得破壞使用者／好友／裝置關聯（關聯以 §20 表名為準，見 ERD.md）。Keycloak 的 Realm、Client、使用者聯邦、Token 對映等配置待身分文件定稿；本文件只定 API 層的認證／授權規範，不重寫 Keycloak 配置。
- 閘道與身分提供者的選型變更不改變本文件的 REST 命名、Schema、錯誤、分頁、限流規範；如 Keycloak／Traefik 的行為與本規範衝突，以本規範與身分／部署文件的定稿為準，並以 ADR 記錄。

## 6. Pagination（Cursor 分頁）

- 集合端點一律使用 Cursor 分頁，不得以 OFFSET 分頁作為預設方案。OFFSET 分頁僅在端點文件明定理由（如管理後台小量列表）時例外使用。
- 請求參數（草案名，待定稿）：`cursor`（不透明游標，客戶端不得解析、不得構造）、`limit`（每頁筆數，須為正整數，上限待定稿，不得預寫數字）。
- 回應須含：資料陣列、`next_cursor`（無更多資料時為 `null`，不得省略而不說明）、`has_more`（草案，是否與 `next_cursor` 並存待定稿，不得兩者語義衝突）。
- 游標的組成與排序鍵待定稿，須與 ERD.md 的排序主鍵一致（`(created_at, id)` 或 `sequence_number`，二者主從待定稿）。排序方向（正序／倒序）須在端點文件中明定，預設方向全 API 一致。
- `limit` 超出上限、非正整數、游標無效或過期時，錯誤處理見 ERROR_CODES.md（驗證錯誤），不得靜默修正為預設值而不告知。
- 分頁結果不得含重複、不得遺漏；游標指向的資料被刪除／收回時，分頁不得中斷，須跳過並繼續（具體行為待定稿）。
- 不得回傳總筆數（`total_count`）作為分頁依據，除非端點文件明定其計算方式與一致性保證；總數如回傳，須註明為估算或快照，不得假裝精確。

## 7. Idempotency（冪等）

- 建立訊息、建立交易相關資源等非冪等 `POST` 端點，須支援 Idempotency Key（草案方向）。Key 的傳輸方式（標頭或請求體）、Key 的範圍（同一使用者／同一對話）、Key 的保留期限，待定稿。
- 訊息送出的去重鍵為 `client_message_id`（草案，與 ERD.md `messages` 唯一約束一致）；`client_message_id` 與通用 Idempotency Key 的關係（同一機制或兩套機制）待定稿，不得並存而不說明。
- 重送命中去重鍵時，須回傳已存在的完整資源，不得新建，不得報錯為衝突（具體狀態碼見 ERROR_CODES.md 定稿）。

## 8. Validation

- 驗證一律在後端執行，不得只在前端驗證。前端驗證僅為體驗輔助，不得作為安全邊界。
- 驗證範圍：必填性、型別、值域、長度／大小、格式（Email、電話、URL 等，格式規則待定稿）、參照完整性（參照的資源存在且呼叫者有權存取）、業務規則（狀態機前置條件、權限、頻率限制）。
- 驗證失敗一律回傳驗證錯誤（見 ERROR_CODES.md），錯誤回應須指明失敗欄位（欄位路徑）與原因，不得僅回傳籠統的「參數錯誤」。
- 輸入驗證須防範注入（SQL Injection、XSS、CSRF 等，見 SPEC-SOURCE.md §17）；驗證方式（參數化查詢、輸出編碼、CSRF Token 等）待安全性架構文件定稿，本文件不預寫實作細節。
- 檔案上傳驗證：格式、大小、惡意檔案掃描，見 §4.4；驗證失敗的檔案不得寫入儲存，不得留下孤兒檔案。

## 9. Rate Limit 與 Logging

### 9.1 Rate Limit

- Rate Limit 一律在後端 enforced，按使用者、裝置、IP、端點等維度分級（維度組合待定稿）。Rate Limit 數值、時間窗口、超限後的行為（拒絕、排隊、降級），一律待定稿／待查證，不得預寫數字。
- 超限時回傳限流錯誤（見 ERROR_CODES.md），並含 `Retry-After` 標頭（草案，秒數或 HTTP 日期，格式待定稿）。不得以靜默丟棄、無限排隊代替明確的限流回應。
- 限流不得以「前端節流」代替後端限流；認證端點、OTP 端點、Bot API 的限流須特別定稿（防暴力破解、防濫用，見 SPEC-SOURCE.md §17）。
- Rate Limit 的豁免清單（如有）須經批准並記錄，不得擅自豁免。

### 9.2 Logging

- 每個 API 請求須記錄結構化日誌，含 `request_id`、時間、方法、路徑、狀態碼、耗時、使用者識別（去識別化形式，待定稿）、裝置識別（去識別化形式），不得記錄請求體明文中的敏感資料（密碼、Token、金鑰、訊息明文、E2EE 內容）。
- 日誌的保留期限、存取權限、去識別化方式，待定稿；日誌不得含個人資料明文，不得用於非營運目的。
- 錯誤日誌須含錯誤碼與 `request_id`，不得含堆疊追蹤以外的敏感資訊；堆疊追蹤不得回傳給客戶端。
- 稽核日誌（管理員操作、權限變更、檢舉處置等）與一般請求日誌分離，見 ERD.md `admin_audit_logs`；稽核日誌不得修改、不得提前刪除。

## 10. OpenAPI／Swagger 產出方式

- OpenAPI 文件為 API 的唯一真實來源（Single Source of Truth，草案方向）：以 OpenAPI 規格檔案（YAML／JSON）承載全部端點、Schema、認證、錯誤回應、分頁、限流標頭，定稿後以該檔案為準，不得以手寫文件代替。
- 產出方式待定稿，二選一（定稿時明定，不得並存）：(a) 以註解／結構化定義從 Go 程式碼產生（Code-first）；(b) 以手寫 OpenAPI 規格為準，程式碼以規格驗證（Spec-first）。產出方式的選定須考慮：規格審查流程、CI 驗證（規格與實作一致性檢查）、客戶端 SDK 產生需求。
- OpenAPI 文件須納入版本管理，與 API 版本一致；文件須可公開查閱的範圍（公開端點）與須認證的範圍待定稿，不得將管理後台端點公開暴露而不說明。
- Swagger UI（或等效文件瀏覽器）的部署方式、環境（開發／測試／正式）待定稿；正式環境的文件瀏覽器不得暴露未經認證的管理端點細節。
- 每次 API 變更須同步更新 OpenAPI 文件，並在階段報告中列出 API 變更（新增／修改／廢棄端點），不得以程式碼變更代替文件更新。

## 11. 版本策略

- 主版本（`v1`）升級：破壞性變更（刪除端點、刪除／改名欄位、改變語義、改變認證方式）須升級主版本（`v2`），並經批准；不得在 `v1` 內擅自破壞相容性。
- 次要變更（新增端點、新增可選欄位、新增列舉值）在 `v1` 內以向後相容方式進行，須在 OpenAPI 文件中標示新增版本與相容性說明。
- 廢棄（Deprecation）：端點廢棄須先標示廢棄（回應標頭或文件標示，方式待定稿），並給予相容期（相容期長短待決定，不得預寫天數）；相容期屆滿後方可移除，移除前須通知客戶端開發者並記錄。
- 客戶端版本與 API 版本的對應、強制更新規則，見 ROADMAP.md 與部署文件，本文件不預寫。
- 多版本並存期間，舊版本的行為不得以新版本的語義回應；版本路由須明確，不得以請求內容推斷版本。

## 12. 缺口與待定稿事項

1. 前綴解讀：§21 其餘 17 項是否掛 `/api/v1`，待 Phase 00 審查確認（見 §2.1）。
2. 未定前綴：Payment Integration、Analytics 等模組是否有獨立前綴，§21 未定，不得擅自新增。
3. 個別端點清單：本文件不定稿任何個別端點；端點清單、方法、路徑、Schema 須在各階段 API 文件中逐項定稿。
4. 認證細節：Token 格式、Refresh Rotation 細節、Bot 金鑰傳輸方式，待身分／Bot 文件定稿。
5. 分頁游標組成：`(created_at, id)` 與 `sequence_number` 的主從關係，待與 ERD.md 一致定稿。
6. 數字缺口：Rate Limit 數值、頁大小上限、逾時秒數、相容期長短，一律待定稿／待查證，不得預寫。
7. 舊規格數字不得直接搬入：MASTER_SPEC.md／CONTRACTS.md 中的 30 秒逾時、暫定上限等，為舊路線已定／暫定值，ETHER 定稿時須重新確認，不得默認沿用。
8. 舊路線端點與 ETHER 前綴的差異（例如舊 `POST /messages`、Supabase RPC `send_message` 與新 `/api/v1/messages`），須寫入 ADR，不得以舊端點實作。
9. OpenAPI 產出方式（Code-first／Spec-first）待定稿。
10. 包裝規則（成功回應是否包裝、集合回應結構）待定稿，全 API 一致。
