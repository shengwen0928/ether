狀態：草案／待批准

# ETHER WebSocket 事件協定（WEBSOCKET_PROTOCOL.md）

## 1. 依據與效力

- 本文件以 `~/workspace/ether/SPEC-SOURCE.md` §21、§22 為最高準據。WebSocket 事件名以 §21 為準，一字不得改；可靠性機制以 §22 為準（Message ID、Idempotency Key、Sequence Number、ACK、Retry、Exponential Backoff、Message Persistence、Offline Queue、Delivery Receipt、Reconnection Recovery）。不得將 WebSocket 連線本身當成可靠送達保證。
- 功能深度補充參考 `~/workspace/messenger-app/MASTER_SPEC.md`；舊合約 `~/workspace/messenger-app/CONTRACTS.md` 僅供參考。舊路線的 Realtime 頻道（`chat:`／`presence:`／`call:`／`sync:`／`account:`）與事件名（`message_created` 等）與 ETHER §21 不同，一律以 §21 為準，差異寫入 ADR，不得以舊頻道／舊事件名實作。
- 本文件定 WebSocket 連線、事件信封、15 個事件的 payload、ACK／重試／去重／Sequence、限流與 Backpressure。個別事件的完整 Schema（欄位必填性、值域）須在 OpenAPI／AsyncAPI 文件中定稿（定稿形式待定，見 §10）；本文件所列 payload 欄位為草案定義，待定稿確認。
- 不得編造額度、數字、效能指標。Heartbeat 間隔、重連退避、限流數值、序號窗口等，一律寫「待定稿／待查證」，不得預寫具體數字。舊規格中的數字（如 30 秒逾時、心跳間隔）為舊路線已定／暫定值，不得直接搬入而不經定稿。
- 架構調整（2026-11-25 00:33 使用者指令，優先於舊技術棧）：邊緣閘道為 Traefik、身分提供者為 Keycloak、訊息佇列為 NATS JetStream、快取為 Valkey、通話 SFU 為 LiveKit Community Edition 自架。WebSocket 事件協定本身不變；舊付費服務名不得殘留。

## 2. 總覽

- 傳輸：WebSocket over TLS（`wss`）。明文 `ws` 不得用於正式環境；開發環境的明文使用須在部署文件中明定並標示風險。
- 端點：WebSocket 連線端點路徑待定稿（草案方向：版本化路徑，例如 `/ws/v1`，待定稿確認，不得以草案為已定）。端點不得與 REST 前綴（§21 的 18 個前綴）混用；REST 與 WebSocket 的職責分工見 §2.1。
- 訊息格式：JSON。事件信封見 §5；payload 欄位一律 `snake_case`；時間一律 ISO 8601（UTC），不得以本地時間、Unix 秒數混用而不說明。
- 方向：事件分為客戶端→伺服器（C→S）、伺服器→客戶端（S→C）、雙向（見 §6 逐事件標示）。客戶端不得假設所有事件皆可由客戶端發起；伺服器不得接受客戶端偽造的他人事件（例如偽造已讀、偽造他人輸入中狀態），每事件的發起權限見 §6。
- 可靠性：WebSocket 只負責即時投遞，不保證送達。送達保證由 ACK、Sequence、去重、離線佇列、REST 歷史同步共同實現（見 §7、§8）。客戶端不得以「WebSocket 已連線」斷言訊息已送達，不得以「WebSocket 收到事件」斷言對方已讀。

### 2.1 REST 與 WebSocket 的分工（草案）

| 操作 | 通道 | 說明 |
|---|---|---|
| 送出訊息 | WebSocket（`message.send`）為主，REST 為備援／定稿時確認 | 送出結果以 ACK 與 REST 歷史查詢為準，不得僅以 WebSocket 事件為準。分工待定稿，定稿前不得假設僅有其一。 |
| 讀取歷史、搜尋 | REST（`/api/v1/messages` 等） | 歷史同步一律走 REST Cursor 分頁，不得以 WebSocket 重放全量歷史。 |
| 即時投遞、已讀／送達回條、輸入中、上線狀態、群組成員變更、來電通知 | WebSocket | 見 §6。 |
| 媒體上傳／下載 | REST／簽名 URL（MinIO） | 不走 WebSocket；進度與狀態以 REST／上傳協定為準，見 API_STANDARDS.md。 |
| 通話媒體 | WebRTC＋LiveKit CE | 信令事件走 WebSocket（`call.*`，見 §6.5），媒體不走 WebSocket。SDP／ICE 交換的承載方式待 Phase 07 定稿，見 §10 缺口。 |

## 3. 連線認證

- 連線建立時須認證：認證方式待定稿（草案方向：連線時以查詢參數或首個認證訊息傳遞 Access Token，Token 由 Keycloak 簽發；二選一，定稿時明定，不得並存兩種方式而不說明）。Token 不得以明文 `ws` 傳輸，不得記錄於日誌、不得以 URL 參數形式留存於伺服器日誌（查詢參數方式如採用，須有日誌遮罩規則）。
- 認證失敗時，伺服器須拒絕連線（關閉碼與原因待定稿），不得建立未認證連線、不得回傳部分資料。
- 認證成功後，伺服器須建立 Session（見 §6 多裝置 Session）：Session 與使用者、裝置（`user_devices`）綁定，Session 識別碼（`session_id`，草案）由伺服器產生並於連線歡迎事件中回傳（見 §5.3）。
- Token 過期：連線期間 Token 過期時，伺服器須通知客戶端重新認證（通知事件待定稿），不得靜默斷線而不說明；客戶端須以刷新後的 Token 重新連線，不得以舊 Token 無限重連。
- 裝置撤銷／遠端登出：裝置被撤銷時，伺服器須主動關閉該裝置的 WebSocket 連線（關閉原因待定稿），該裝置不得再以舊 Token 重連；撤銷須即時生效，不得等待下次心跳。
- 管理員、Bot 的 WebSocket 連線（如有）須以獨立認證方式定稿，不得與一般使用者連線混用；本文件不預寫管理員／Bot 的連線端點。

## 4. Heartbeat、重連與恢復

### 4.1 Heartbeat

- Heartbeat 機制：客戶端與伺服器定期交換心跳（Ping／Pong 或應用層心跳事件，形式待定稿）。Heartbeat 間隔、逾時判定、失敗次數上限，一律待定稿，不得預寫數字。
- Heartbeat 僅用於連線存活判定，不得作為已讀保證、上線狀態的唯一依據（上線狀態見 `presence.*` 事件，見 §6.3）。
- Heartbeat 失敗時，客戶端須按 §4.2 重連，不得以高頻重試耗電（背景連線的心跳與重試策略須考慮電池，見 TEST_STRATEGY.md）。

### 4.2 重連與恢復

- 自動重連：斷網、伺服器重啟、快速切網、App 背景／被系統終止後，客戶端須自動重連。重連須採 Exponential Backoff（退避間隔上限、重試次數上限待定稿，不得預寫數字），不得立即連續重連、不得無限重連而不設上限（上限到達後的行為待定稿，例如轉為手動重連提示）。
- 重連恢復（Reconnection Recovery）：重連成功後，客戶端須以「恢復游標」向伺服器請求離線期間的遺失事件／訊息（游標形式見 §7.3 Sequence 規則）。恢復方式為「WebSocket 恢復＋REST 歷史同步」二者併用（草案方向）：WebSocket 恢復投遞即時事件，REST 歷史同步補齊訊息，定稿時明定二者的分工與去重規則，不得以其一代替另一而不說明。
- 恢復期間，客戶端不得顯示重複訊息、不得遺失訊息、不得亂序顯示；恢復完成的判定（伺服器通知或客戶端游標追平）待定稿。
- 伺服器重啟：伺服器重啟後，Session Registry（見 §6 多裝置 Session）須重建，客戶端重連後須重新註冊裝置與訂閱（訂閱範圍待定稿），不得假設重啟前訂閱自動恢復。
- 推播與即時訊息重複：推播通知與 WebSocket 事件可能重複到達，客戶端須以訊息識別碼去重，不得重複顯示、不得重複計未讀數。

### 4.3 連線狀態與 UI

- 客戶端須向使用者顯示連線狀態（已連線／連線中／離線／重連中），狀態變更須即時反映，不得以「已連線」假裝離線佇列已送達。
- 離線期間的送出操作進入離線佇列（見 §7.4），佇列狀態須可查，不得靜默丟棄。

## 5. 事件信封

### 5.1 信封欄位（草案，待定稿確認）

每個 WebSocket 訊息皆為一個信封物件，含以下欄位：

| 欄位 | 型別（草案） | 說明 |
|---|---|---|
| `event_id` | 字串 | 事件唯一識別碼（UUID 草案）。同一事件重投時 `event_id` 不變，供去重用（見 §7.2）。 |
| `event` | 字串 | 事件名（§21 的 15 個事件之一，或系統事件，見 §5.3）。一字不得改。 |
| `version` | 字串／整數 | 事件協定版本（草案形式待定稿，例如 `"1"`）。版本升級規則見 §10；客戶端須校驗版本，不支援的版本須拒絕並記錄，不得以舊版解析新版事件。 |
| `timestamp` | 字串（ISO 8601，UTC） | 伺服器產生事件的時間（S→C 事件）或客戶端送出時間（C→S 事件，僅供參考，不得作為排序鍵）。排序以 Sequence 為準（見 §7.3）。 |
| `correlation_id` | 字串（可為空） | 關聯識別碼：請求／回應、ACK、重試的關聯鍵。C→S 請求的 `correlation_id` 由客戶端產生，伺服器的 ACK／回應須原樣回傳（見 §7.1）。S→C 廣播事件的 `correlation_id` 為觸發該事件的原始請求識別碼（若有），否則為空。 |
| `sequence` | 整數（可為空） | Sequence Number（見 §7.3）。並非所有事件皆有 Sequence，適用範圍待定稿。 |
| `sender_id` | 字串（可為空） | 事件發起者識別碼（S→C 事件）。客戶端不得信任 `sender_id` 而不經伺服器驗證；伺服器須以認證身份填寫，不得接受客戶端自填的 `sender_id`。 |
| `data` | 物件 | 事件 payload（見 §6 逐事件定義）。 |
| `error` | 物件（可為空，僅錯誤回應時） | 錯誤資訊（格式見 §8，錯誤碼以 ERROR_CODES.md 為準，WebSocket 錯誤碼對應待定稿）。 |

- 未知欄位：信封中含未定義欄位時，處理方式（忽略或拒絕）待定稿，全事件一致。
- 信封欄位名一律 `snake_case`，不得混用其他命名。
- `event_id`、`timestamp`、`version`、`correlation_id` 為 §21 所要求的「事件 ID、時間戳記、版本、關聯識別碼」，四項缺一不可（`correlation_id` 在無關聯時可為空字串或 `null`，定稿時明定）。

### 5.2 客戶端送出格式（示例）

```json
{
  "event_id": "01J...",
  "event": "message.send",
  "version": "1",
  "timestamp": "2026-11-25T00:00:00Z",
  "correlation_id": "client-req-001",
  "data": {
    "conversation_id": "conv-001",
    "client_message_id": "client-msg-001",
    "kind": "text",
    "content": "你好"
  }
}
```

示例僅說明信封形式，payload 欄位為草案，待定稿確認。

### 5.3 系統事件（草案，非 §21 事件）

除 §21 的 15 個事件外，連線層可能需要以下系統事件（草案名，待定稿確認；如定稿不採用，須刪除並以其他機制代替，不得擅自新增事件名而不經定稿）：

| 系統事件（草案） | 方向 | 說明 |
|---|---|---|
| `connection.ready` | S→C | 連線認證成功，含 `session_id`、伺服器時間、支援的事件版本。 |
| `connection.error` | S→C | 連線層錯誤（認證失敗、協定版本不支援、限流），含錯誤碼。 |
| `heartbeat.ping`／`heartbeat.pong` | 雙向 | 應用層心跳（形式待定稿；如用 WebSocket Ping／Pong 幀，則不需此事件）。 |
| `sync.required` | S→C | 伺服器通知客戶端須以 REST 歷史同步補齊（恢復游標已過期等），含同步起點。 |
| `session.revoked` | S→C | Session 被撤銷，客戶端須停止重連並導向重新登入。 |

系統事件不得與 §21 事件同名、不得冒充 §21 事件；系統事件的增刪須經定稿。

### 5.4 多裝置 Session

- 每個裝置一條 WebSocket 連線、一個 Session；同一帳號的多裝置 Session 由伺服器端的 Session Registry 管理（Registry 的儲存位置：Valkey 草案方向，待定稿）。
- Session 內容（草案）：`session_id`、`user_id`、`device_id`、連線建立時間、最後心跳時間、訂閱範圍（對話清單／群組清單，待定稿）、恢復游標。
- 訊息投遞：伺服器須向同一使用者的全部線上裝置投遞（多裝置同步），不得僅投遞至送出裝置以外的裝置、不得遺漏送出裝置本身的確認（送出裝置的確認以 ACK 為準，見 §7.1）。
- 已讀／送達回條、撤回、設定、草稿的同步，須經 WebSocket 事件投遞至全部裝置（事件見 §6），並以 REST 歷史同步為最終一致依據。
- 裝置上限：同一帳號同時線上裝置數上限待定稿（不得預寫數字）；超限時新裝置連線的處理（拒絕或提示登出舊裝置）待定稿，不得靜默踢掉舊裝置。
- Session 過期／閒置：閒置 Session 的回收規則待定稿；回收不得導致訊息遺失（離線佇列須接手，見 §7.4）。

## 6. 事件定義（§21 全部 15 個，一字不得改）

以下逐事件列出：方向、發起權限、payload 欄位（草案）、語義、邊界。payload 欄位名為草案，待定稿確認；欄位必填性、值域在定稿文件中逐項確認。

### 6.1 訊息事件

#### 6.1.1 `message.send`（C→S）

- 方向：客戶端→伺服器。客戶端送出訊息的請求事件。
- 發起權限：對話成員（須經伺服器驗證成員身份、封鎖關係、禁言狀態等，不得僅信客戶端聲明）。
- Payload（草案）：
  - `conversation_id`（必填）：目標對話識別碼。
  - `client_message_id`（必填）：客戶端產生的唯一識別碼，去重鍵（見 §7.2）。
  - `kind`（必填）：訊息類型（值域待定稿）。
  - `content`（必填，依 `kind` 而定）：訊息內容；E2EE 聊天室的內容形態待 E2EE 文件定稿，不得以明文欄位假設 E2EE。
  - `reply_to_message_id`（可為空）：引用訊息識別碼。
  - `attachments`（可為空）：附件參照清單（媒體識別碼，形態待定稿；檔案本體不走 WebSocket）。
  - `idempotency_key`（可為空）：冪等鍵，與 `client_message_id` 的關係待定稿（見 API_STANDARDS.md §7）。
- 語義：伺服器收到後，須驗證、寫入（PostgreSQL）、分配 Sequence、持久化後回傳 ACK（見 §7.1），再向對話成員廣播 `message.received`。寫入失敗時，須回傳錯誤 ACK，不得廣播。
- 邊界：送出逾時、重送、離線佇列，見 §7。

#### 6.1.2 `message.received`（S→C）

- 方向：伺服器→客戶端。廣播給對話全部線上成員（含送出者裝置，作為確認）。
- Payload（草案）：
  - `message_id`（必填）：伺服器分配的訊息識別碼。
  - `conversation_id`（必填）。
  - `sender_id`（必填）。
  - `kind`、`content`（必填，內容形態同 `message.send`；E2EE 訊息的內容為密文，客戶端解密，不得以伺服器明文假設）。
  - `sequence`（必填）：每對話 Sequence Number（見 §7.3）。
  - `client_message_id`（可為空）：送出者的去重鍵，原樣回傳，供送出裝置對帳。
  - `created_at`（必填）：伺服器寫入時間。
  - `reply_to_message_id`、`attachments`（可為空，同 `message.send`）。
  - `status`（必填，草案）：訊息狀態（例如已送達，值域待定稿）。
- 語義：客戶端收到後，須按 Sequence 排序顯示、去重（見 §7.2、§7.3），並更新本地狀態。不得以收到事件即斷言對方已讀（已讀另見 `message.read`）。

#### 6.1.3 `message.delivered`（S→C）

- 方向：伺服器→客戶端。送達回條：訊息已投遞至接收者裝置。
- Payload（草案）：`message_id`、`conversation_id`、`recipient_id`、`delivered_at`、`device_id`（可為空）。
- 語義：送達僅表示投遞至裝置，不表示接收者已閱讀。客戶端須更新訊息狀態為已送達，不得顯示為已讀。
- 發起：由伺服器在投遞成功後產生，不得由客戶端偽造他人送達回條。

#### 6.1.4 `message.read`（雙向）

- 方向：客戶端→伺服器（回報已讀）與伺服器→客戶端（廣播已讀狀態）皆用同一事件名（草案方向，待定稿確認是否分拆為回報與廣播兩種信封，不得混淆）。
- Payload（草案）：`message_id`、`conversation_id`、`reader_id`、`read_at`。
- 語義：接收者開啟對話並使訊息進入可見範圍時回報已讀；伺服器廣播後，傳送者更新已讀狀態。已讀回報不得保證接收者本人已閱讀內容。使用者關閉「傳送已讀回條」時，不得回報、亦不得查看他人已讀狀態（對等原則，待定稿確認）。
- 邊界：已讀進度不得倒退（較舊的 `read_at` 不得覆寫較新進度）；重複回報須以第一次為準（規則待定稿）。

#### 6.1.5 `message.recalled`（S→C）

- 方向：伺服器→客戶端。訊息收回廣播。
- Payload（草案）：`message_id`、`conversation_id`、`recalled_by`、`recalled_at`。
- 語義：收回後，全部成員檢視中原內容改為佔位，附件不得再被下載；訊息列保留（append-only），不得物理刪除。收回時限、收回權限（僅傳送者）由伺服器驗證，超限請求須回傳錯誤，不得廣播。
- 客戶端收到後，須清除本地原內容顯示（本地快取的原內容處理待定稿，不得以本地殘留假裝收回未生效）。

### 6.2 輸入中事件

#### 6.2.1 `typing.start`（C→S，伺服器轉發為 S→C）

- 方向：客戶端→伺服器（開始輸入），伺服器轉發給對話其他成員（S→C，同名事件）。
- Payload（草案）：`conversation_id`、`user_id`、`started_at`。
- 語義：使用者開始輸入時發送；伺服器須驗證成員身份後轉發，不得轉發給非成員。輸入中狀態為短暫狀態，不得持久化為訊息，不得計入未讀。
- 節流：客戶端不得高頻重送 `typing.start`（節流間隔待定稿）；伺服器須限流（見 §9）。

#### 6.2.2 `typing.stop`（C→S，伺服器轉發為 S→C）

- 方向：同 `typing.start`。
- Payload（草案）：`conversation_id`、`user_id`、`stopped_at`。
- 語義：使用者停止輸入、送出訊息、離開對話時發送；伺服器轉發後，其他成員清除輸入中顯示。`typing.start` 後未收到 `typing.stop` 時，輸入中顯示的自動清除逾時待定稿，不得永久顯示。

### 6.3 上線狀態事件

#### 6.3.1 `presence.online`（S→C）

- 方向：伺服器→客戶端。使用者上線通知。
- Payload（草案）：`user_id`、`device_id`（可為空）、`online_at`。
- 語義：使用者任一裝置連線成功後，伺服器向其好友／相關成員廣播上線（廣播範圍待定稿，須遵守隱私設定：關閉「公開上線狀態」時不得廣播）。上線狀態僅供顯示，不得作為已讀、送達保證。
- 客戶端不得以 `presence.online` 推斷對方正在使用 App 的特定功能。

#### 6.3.2 `presence.offline`（S→C）

- 方向：伺服器→客戶端。使用者離線通知。
- Payload（草案）：`user_id`、`last_seen_at`（可為空，草案，最後上線時間，是否揭露待隱私文件定稿）。
- 語義：使用者全部裝置斷線（或閒置逾時，規則待定稿）後廣播離線。短暫斷線／重連期間不得頻繁翻轉上線／離線狀態（防抖規則待定稿）。

### 6.4 對話與群組事件

#### 6.4.1 `conversation.updated`（S→C）

- 方向：伺服器→客戶端。對話屬性變更通知（標題、頭像、置頂、封存、靜音、最後訊息等，變更範圍待定稿）。
- Payload（草案）：`conversation_id`、`updated_fields`（草案，變更欄位清單）、`updated_at`、`last_message_id`（可為空）、`unread_count`（可為空，草案，未讀數的計算與同步規則待定稿）。
- 語義：客戶端收到後，須更新對話列表顯示，不得要求手動刷新。未讀數的增減須與 `message.received`／`message.read` 一致，不得重複計數。

#### 6.4.2 `group.member.joined`（S→C）

- 方向：伺服器→客戶端。群組成員加入通知。
- Payload（草案）：`group_id`、`conversation_id`（可為空，群組與對話對應待定稿）、`user_id`（加入者）、`invited_by`（可為空）、`joined_at`、`role`（可為空，值域待定稿）。
- 語義：成員加入後，向群組成員廣播；同時是否產生系統訊息（`kind` 為系統的訊息列）待定稿，不得以事件廣播代替系統訊息列而不說明。

#### 6.4.3 `group.member.left`（S→C）

- 方向：伺服器→客戶端。群組成員離開／被移除通知。
- Payload（草案）：`group_id`、`conversation_id`（可為空）、`user_id`（離開者）、`removed_by`（可為空，被移除時）、`left_at`、`reason`（值域待定稿，例如自行離開／被移除）。
- 語義：離開者不再收到新訊息；歷史訊息查看權限以離開時設定為準（規則待定稿）。廣播範圍須排除已離開者（不得向已離開者投遞新事件）。

### 6.5 通話事件

通話事件僅為通話生命週期的通知；通話媒體走 WebRTC＋LiveKit CE，信令細節（SDP／ICE）待 Phase 07 定稿，見 §10 缺口。

#### 6.5.1 `call.incoming`（S→C）

- 方向：伺服器→客戶端。來電通知。
- Payload（草案）：`call_id`、`conversation_id`（可為空）、`caller_id`、`call_type`（值域待定稿，例如語音／視訊）、`invited_at`。
- 語義：客戶端收到後，須以前景通知與鈴聲提醒（含 App 在背景時，平台來電整合見 Phase 07 文件）；使用者可接聽、拒接。來電逾時未接聽即轉為未接（逾時值待定稿）。

#### 6.5.2 `call.accepted`（雙向）

- 方向：客戶端→伺服器（接聽回報）與伺服器→客戶端（廣播已接聽）皆用同一事件名（草案方向，待定稿確認）。
- Payload（草案）：`call_id`、`conversation_id`（可為空）、`accepter_id`、`accepted_at`。
- 語義：接聽後，通話進入建立流程（LiveKit 房間加入等，見 Phase 07）；其他裝置的來電通知須失效（多裝置接聽一致性規則待定稿）。

#### 6.5.3 `call.ended`（S→C）

- 方向：伺服器→客戶端。通話結束通知。
- Payload（草案）：`call_id`、`conversation_id`（可為空）、`ended_by`（可為空）、`end_reason`（值域待定稿，例如掛斷／拒接／未接／忙線／中斷）、`ended_at`、`duration_seconds`（可為空）。
- 語義：任何一方結束即結束，不得殘留連線；通話紀錄寫入 `calls` 表（見 ERD.md），紀錄不得含通話內容。客戶端收到後，須釋放通話資源、更新通話紀錄顯示。

## 7. ACK／重試／去重／Sequence 規則

### 7.1 ACK

- 客戶端送出的每個請求事件（以 `message.send` 為主），伺服器須回傳 ACK。ACK 形式待定稿（草案方向：以同一 `event` 名＋ `correlation_id` 原樣回傳的確認信封，含 `status`（成功／失敗）、`message_id`（成功時）、`sequence`（成功時）、`error`（失敗時））。
- 客戶端收到 ACK 前，訊息狀態為待確認（Pending／Sending）；收到成功 ACK 後標為已送出；收到失敗 ACK 後標為失敗、可重送。未收到 ACK（逾時）時，狀態判定與重送見 §7.4；逾時值待定稿，不得預寫。
- S→C 廣播事件是否需要客戶端 ACK，待定稿（草案方向：廣播事件不要求逐一 ACK，送達回條以 `message.delivered`／`message.read` 為準，不得以廣播 ACK 代替回條）。
- ACK 不得偽造：伺服器僅在訊息持久化成功後回傳成功 ACK，不得在寫入前回傳成功。

### 7.2 去重

- 去重鍵：`client_message_id`（同一對話內唯一，見 ERD.md `messages` 唯一約束）。伺服器端與客戶端皆須以 `client_message_id` 去重；重送、弱網重試、離線佇列重放、恢復重投，皆不得產生重複訊息列。
- 事件去重鍵：`event_id`。同一 `event_id` 的事件重投時，接收方須去重，不得重複顯示、不得重複計數。`event_id` 的保留期限（去重窗口）待定稿。
- 去重命中時，伺服器須回傳已存在的完整訊息（不得新建、不得報錯為衝突，具體行為待與 ERROR_CODES.md 一致定稿）。

### 7.3 Sequence 規則

- Sequence Number：每對話一個遞增序號（草案方向，起始值、步長、溢位處理待定稿）。伺服器在寫入訊息時分配，不得由客戶端分配、不得跳號、不得重用。
- 排序：客戶端以 Sequence 為主排序鍵顯示訊息；Sequence 與 `(created_at, id)` 的主從關係待與 ERD.md 一致定稿（見 ERD.md §10.2 缺口），不得並存兩套排序而不說明。
- 亂序處理：客戶端收到亂序事件時，須按 Sequence 緩衝、排序後顯示，不得直接按到達順序顯示；缺口等待逾時待定稿，逾時後須以 REST 歷史同步補齊，不得永久等待。
- 恢復游標：重連恢復時，客戶端以最後收到的 Sequence（每對話）為游標請求遺失事件；伺服器須回傳游標之後的事件，不得重投游標之前者（重投時去重鍵仍生效）。

### 7.4 重試與離線佇列

- 重試：送出未收到 ACK 時，客戶端按 Exponential Backoff 重試（重試次數上限、退避間隔待定稿，不得預寫數字）；重試皆帶同一 `client_message_id` 與同一 `correlation_id`（或新 `correlation_id` 但同一去重鍵，定稿時明定）。
- 離線佇列：無網路時，送出、編輯、收回、已讀回報等操作進入本地離線佇列（持久化，App 重啟後不遺失）；恢復連線後依序重放，重放仍帶原去重鍵。佇列長度上限、滿佇列時的行為，待定稿，不得預寫數字。
- 超過重試上限後，訊息標為失敗（Failed），由使用者手動重送；手動重送仍帶原 `client_message_id`，不得產生重複。
- 伺服器端重試：伺服器向客戶端投遞失敗時，須將事件寫入離線佇列（NATS JetStream 持久化，草案方向），待客戶端重連後投遞；投遞重試的次數、保留期限待定稿。不得將 WebSocket 連線本身當成可靠送達保證。

## 8. WebSocket 錯誤格式（草案）

- 錯誤回應以信封 `error` 欄位承載（見 §5.1），含 `code`（錯誤碼，以 ERROR_CODES.md 為準，WebSocket 專屬錯誤碼的對應待定稿）、`message`（人類可讀訊息，不得含敏感資訊）、`request_id`（關聯識別碼，見 ERROR_CODES.md §5）。
- 常見 WebSocket 錯誤（草案）：認證失敗、事件格式錯誤、未知事件、權限不足、限流、序號錯誤、Session 已撤銷。逐碼的錯誤碼、關閉碼、客戶端處理，待定稿。
- 連線關閉碼：WebSocket 關閉碼（Close Code）的使用方式待定稿，不得以自訂關閉碼代替錯誤事件而不說明；關閉原因須以錯誤事件或關閉原因字串明示，不得靜默斷線。

## 9. Rate Limit 與 Backpressure

- 限流：伺服器須對 WebSocket 事件限流（按使用者、裝置、對話、事件類型分級，維度與數值待定稿，不得預寫數字）。限流重點事件：`message.send`、`typing.start`／`typing.stop`、`message.read`。限流超限時，伺服器須回傳限流錯誤（見 §8），不得靜默丟棄而不告知（丟棄策略待定稿）。
- Backpressure：伺服器投遞速率超過客戶端處理能力時，須有 Backpressure 機制（草案方向：伺服器端緩衝＋離線佇列、客戶端流量控制，具體機制待定稿）；不得以無限緩衝假裝無 Backpressure，不得以斷線作為唯一 Backpressure 手段。
- 閘道層限流：Traefik 的連線數／速率限流與應用層限流的分工待部署文件定稿；二者不得重複計算而不說明，不得互相矛盾。
- 大型群組：大型群組的廣播扇出（Fan-out）須考慮 NATS JetStream 與 Valkey 的承載方式（草案方向），扇出策略、批次投遞、降級方式待定稿，不得以單機廣播假設支撐大型群組。

## 10. 缺口與待定稿事項

1. 連線端點路徑、認證傳遞方式（查詢參數或首訊息）、Token 過期通知事件，待定稿。
2. Heartbeat 形式、間隔、逾時、重連退避、重試上限，一律待定稿，不得預寫數字。
3. ACK 信封形式、廣播事件是否 ACK、系統事件清單，待定稿。
4. Sequence 起始值、步長、適用事件範圍、與 `(created_at, id)` 的主從關係，待與 ERD.md 一致定稿。
5. 通話信令細節：SDP／ICE 的承載事件（是否新增 WebSocket 事件或走 LiveKit 信令）、`call.*` 事件與 LiveKit 房間事件的對應，待 Phase 07 定稿；本文件不得以 `call.*` 三事件假裝已覆蓋全部信令。
6. WebSocket 錯誤碼與 HTTP 錯誤碼的對應、關閉碼使用方式，待定稿。
7. 限流數值、Backpressure 機制、大型群組扇出策略，待定稿／待壓測。
8. 訂閱範圍（客戶端訂閱哪些對話／群組的事件）、Session Registry 儲存、裝置上限，待定稿。
9. 舊路線頻道／事件與 §21 事件的差異，須寫入 ADR；舊事件名不得殘留於實作。
10. OpenAPI／AsyncAPI 的定稿形式（是否以 AsyncAPI 承載 WebSocket Schema）待定稿。
