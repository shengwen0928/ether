# Gin vs Echo 比較備忘錄（Phase 01 決策支援）

> **決議註記（2026-11-25）：** 已決議採用 Echo v5（版本不得低於 v5.4.7），見 ADR-016。
> 本備忘錄比較內文保留為決策證據；與 ADR-016 牴觸的技術描述（`Context` 型別、WebSocket 責任）以 ADR-016 為準。
> 相關決議：G-01 方案 A（ADR-015）、pgvector（ADR-017）、G-02／G-03 處理方案（執行計畫 §4）。

狀態：草案／待使用者決定。本備忘錄只提供比較與推薦，**最終決定權在使用者**；推薦不等於決定。在決定前，後端骨架一律以標準庫 `net/http` 實作，不得擅自以任一框架建置（見 `services/api/internal/transport` 的隔離設計）。

查證日期：2026-11-25。版本與維護狀態皆為當日公開來源查證結果，決定時須在 CI 覆核是否仍為最新；本備忘錄不引用、亦不編造任何基準測試數字。

## 1. 比較範圍：Phase 01 的實際需求

ETHER 後端在 Phase 01–05 的實際需求（以 Phase 00 文件為準）：

- 版本化 REST（18 個前綴，`API_STANDARDS.md` §2），成功／錯誤回應須符合 `ERROR_CODES.md` 的 `ETHER_` 錯誤信封與 `request_id` 規則。
- 後端強制驗證與授權（`IDENTITY_AND_PERMISSIONS.md` §8）：身分由 Keycloak 簽發 Token，後端再驗證；閘道（Traefik）不得成為唯一授權邊界（`API_STANDARDS.md` §5.4）。框架內建的 JWT 中介軟體因此**不能**代替 Keycloak 驗證，只能作輔助。
- WebSocket 閘道（Phase 05）：認證、Registry、Heartbeat、重連恢復、多裝置 Session、Rate Limit、Backpressure、Graceful Shutdown（`WEBSOCKET_PROTOCOL.md`）。
- Rate Limit 與日誌：後端強制、分級限流、結構化日誌含 `request_id`，不得記錄敏感明文（`API_STANDARDS.md` §9）。
- 傳輸層隔離：handler 不得直接依賴框架；框架綁定只存在於單一 adapter（本階段為 `internal/transport` 的 `Router` 介面＋標準庫實作）。

## 2. 查證到的版本與維護狀態（2026-11-25）

| 項目 | Gin | Echo |
|---|---|---|
| 最新穩定版（查證日） | v1.12.0，2026-02-28 發布 | v5 為現行主線，2026-01-18 發布；最新 patch 線為 v5.4.4（2026-10-30） |
| 授權 | MIT | MIT（核心）；Echo Pro 為 LabStack 另售的商業函式庫，與核心無關，本專案不需要 |
| Go 版本要求 | 發行公告稱 Go 1.25+；另有來源稱現行 README 要求 Go 1.26+（說法不一，見備註） | v5 支援最近的三個 Go 版本（README）；v4 要求 Go 1.26+ |
| 舊主線支援 | v1 API 宣稱凍結，新版不破壞既有程式碼 | v4 只收 bug 修正至 2026-12-31、安全修正至 2027-12-31，不再新增功能；新專案須用 v5，v4 不是可退回的選項 |
| 社群規模（GitHub stars，第三方彙整，2026-11-03 判定） | 約 89k–90k | 約 33k |
| 近期活動 | 2026-02 發布 v1.12.0 後，儲存庫仍有提交至 2026-10；發版節奏慢 | 2026-10 有一批安全修正（見 §7），v5 持續發版中 |

備註：Gin 的 Go 最低版本在不同來源說法不一（1.25+ vs 1.26+）。本骨架的 `go` directive 固定為 1.25（與 Dockerfile 的 `golang:1.25.0` 一致）；若決定採用 Gin，CI 須以 `go.mod` 實際解析結果驗證，不得以本備忘錄的任一說法為準。Echo v5 的 Go 要求同樣須以 CI 驗證。

來源：Gin 發布與 Go 要求見 <https://www.ithub.directory/web-frameworks/gin>；Gin 活動與 stars 見 <https://assemblyline.org/software/gin-gonic/gin>、<https://techloghub.com/open-source/gin-web-framework>；Echo 版本政策見 <https://github.com/labstack/echo/blob/master/README.md>；Echo v5.4.4 與 stars 見 <https://assemblyline.org/software/labstack/echo>、<https://hub.xerj.org/corpus/labstack--echo>；Go 發布線見 <https://endoflife.date/go>。

## 3. 逐項比較

### 3.1 WebSocket 閘道需求

- **Gin**：無內建 WebSocket 實作，須以 `gorilla/websocket`（或同類函式庫）在 handler 內以 `http.ResponseWriter`／`*http.Request` 升級連線；升級前仍可享有 Gin 的中介軟體（如認證檢查）。WS 函式庫的選擇是獨立決定，框架本身不替你選。
- **Echo**：提供 WebSocket 升級輔助（以 `gorilla/websocket` 為底層），在 `echo.Context` 內完成升級；同樣須另選 WS 函式庫的細節（連線管理、ping／pong、寫入序列化皆不在框架內）。
- **結論**：兩者在 WS 上皆為「升級輔助＋自帶函式庫」，沒有誰內建了 Registry、Heartbeat、重連恢復或 Backpressure——Phase 05 的閘道邏輯一律自寫。Echo 的輔助略省樣板碼，但差距不足以單獨決定選型。WS 函式庫（`gorilla/websocket` vs `coder/websocket` 等）的選型待 Phase 05 另行決定，不得在本備忘錄中預設。

### 3.2 Middleware 生態

- **Gin**：核心內建 Logger、Recovery、BasicAuth；CORS、Session、JWT、Rate Limiting、壓縮、指標、追蹤等皆在獨立的 `gin-contrib` 組織，各為獨立 Go module，須逐一 `go get` 與釘選版本。生態大，但依賴面較散。
- **Echo**：核心附帶較完整的中介軟體套件（Logger、Recover、CORS、RateLimiter、CSRF、Secure headers、BodyLimit、RequestID、Gzip、Timeout 等，見 `middleware` 套件），同一 module 內解決大部分 Phase 01–05 需求。JWT 另有 `echo-jwt` 模組。
- **ETHER 特別提醒**：兩者的內建 RateLimiter 皆為行程內／記憶體儲存。ETHER 須水平擴充（`SYSTEM_ARCHITECTURE.md` §4.2：Gateway 須經 Valkey／NATS 跨副本路由），跨副本限流必須以 Valkey 為後端自寫中介軟體，不得以框架內建限流器冒充分散式限流。內建限流器只可作本機輔助，不得為唯一限流邊界。
- **結論**：Echo 在「單一 module 內含常用中介軟體」上佔優，減少 Phase 01 的依賴管理面；Gin 須多個 `gin-contrib` 模組，版本釘選工作較多。

### 3.3 驗證／綁定

- **Gin**：內建綁定係統，以 struct tag（`json`／`form`／`uri`／`header`＋`binding`）從 JSON、XML、表單、查詢字串、路徑參數綁定，驗證語法沿用 `go-playground/validator`；`Bind` 家族在驗證失敗時直接以 400 中斷，`ShouldBind` 家族把錯誤交還 handler。開箱即用程度高，但 `Bind` 的自動 400 行為須小心：ETHER 的錯誤信封（`ETHER_VALIDATION_INVALID_FIELD`＋`details`）要求自訂錯誤格式，必須統一使用 `ShouldBind` 家族再轉譯，不得讓框架的預設 400 繞過信封。
- **Echo**：綁定（JSON／XML／表單／查詢／路徑）內建，驗證為可插拔的 `Validator` 介面（`e.Validator`），實務上同樣接 `go-playground/validator`。驗證器是顯式接線的，開發者必須自己註冊，換來的是驗證失敗的錯誤一定經過自己的程式碼，較不易被框架預設回應繞過。
- **結論**：兩者最終都落在同一個驗證函式庫上，能力無實質差距；Gin 開箱較順，Echo 的顯式接線與 ETHER「錯誤一律經信封」的要求較合拍。差距小。

### 3.4 錯誤處理

- **Gin**：錯誤以 `c.Error()` 累積於 context，須由 handler 或中介軟體自行彙整為回應；框架沒有集中的錯誤轉譯點。ETHER 信封的落實方式是「包裝 handler／中介軟體統一轉譯」，須自訂且須全路由覆蓋，遺漏一個路由就漏一個信封。
- **Echo**：集中式錯誤處理（`HTTPError`＋自訂 `HTTPErrorHandler`），全部 handler 錯誤（含 404／405）皆經同一轉譯點；`ERROR_CODES.md` 的信封、狀態碼對應、`request_id` 注入在此一處落實即可，覆蓋性較易保證。
- **結論**：Echo 的集中式錯誤處理與 ETHER 的錯誤碼規範配合度較高；Gin 亦可做到，但須紀律（全部路由經同一包裝）才能達到同等覆蓋。

### 3.5 效能特性

- 本備忘錄**不引用任何基準測試數字**：未在相同硬體／相同 Go 版本／相同測試 harness 下實測，公開的框架基準數字因測試條件不同而無法直接比較；編造或轉抄數字會誤導決定。
- 定性事實：兩者皆為 radix-tree 路由、皆建於 `net/http` 之上，皆屬 Go 生態中效能前段的框架；ETHER 的延遲瓶頸在 Phase 01–05 幾乎不可能落在 HTTP 路由層（資料庫交易、Outbox、NATS 投遞、WS 扇出才是主導因素，見 `DATA_FLOW.md`）。效能不得作為本次決定的主因；若日後 profiling 證明 HTTP 層是瓶頸，再以實測數據重估（`SYSTEM_ARCHITECTURE.md` §4.2：擴充只在觀測到瓶頸證據後進行）。

### 3.6 長期維護活躍度

- **Gin**：歷史最久（2014 年起）、v1 API 凍結帶來的相容穩定性是真優勢；但發版節奏慢（v1.12.0 距查證日已約 9 個月），新 Go 版本／新標準庫能力的跟進較被動。
- **Echo**：v5 是 2026-01 才發布的新主線，仍在積極發版；但「新」本身是風險：v5 的 API 仍在收斂期（v5.0 初期曾保留破壞性修正窗口至 2026-03-31），且 2026-10 曝出一批安全公告（含 WebSocket 代理繞過、標頭偽造、解壓縮／BodyDump 無界配置等），修正版為 v4.16.1／v5.4.1 起（含後續強化）。這批公告一方面證明維護者在積極修補，另一方面要求採用方必須嚴格釘選版本並跟進安全更新，不得以舊 patch 上線。
- 來源：Echo 安全公告批次見 <https://corgea.com/research/ghsa-cxf3-9h37-7rr8-echo-v4-16-1-v5-4-1-websocket-proxy-auth-bypass-dos>（2026-10-23 公告，修正於 v4.16.1／v5.4.1，v5.4.2 追加強化）。
- **結論**：維護活躍度兩者皆可接受，風險型態不同：Gin 是「慢而穩」，Echo 是「新而勤」。採用 Echo 須配套「依賴更新紀律」（CI 定期檢查、CVE 訂閱、安全修正即時升級）；採用 Gin 須接受「新能力跟進慢」。

### 3.7 社群與文件

- **Gin**：社群最大（約 89k stars）、教學與問答最多，招募與交接成本最低；文件以官網與 `gin-gonic.com` 為準，`gin-contrib` 各模組文件分散。
- **Echo**：社群較小（約 33k stars）但文件集中（同一 repo 的 README、 cookbook、API 變更文件 `API_CHANGES_V5.md`）；v5 的升級文件齊全，但 v5 問答量尚不及 Gin／v4。
- **結論**：Gin 在「找得到人、找得到答案」上佔優；Echo 的文件集中度較好。對 ETHER 這種長期專案，兩者皆不構成否決理由。

### 3.8 與 transport 介面隔離設計的配合度

- 本骨架的 `Router` 介面（`Handle(method, path, handler)`＋標準庫 handler 簽名）對兩者皆可對映：Gin 以 engine／group 方法註冊，Echo 以 `e.GET/POST` 與 group 註冊；中介軟體皆可掛在 adapter 層，handler 保持框架無關。
- 細微差異：Echo 的 `Context` 是介面（interface），adapter 要把框架 context 轉為標準 `http.ResponseWriter`／`*http.Request` 時，包裝較乾淨；Gin 的 `*gin.Context` 是具體結構，adapter 須經 `c.Writer`／`c.Request` 取出底層標準型別，同樣可行但多一層轉接。兩者皆不影響 handler 程式碼。
- 關鍵約束（不分框架）：adapter 不得讓 handler 讀取框架 context 的擴充能力（如 Gin 的 `c.Get`／Echo 的 `c.Get` 傳值）；跨 handler 的請求狀態一律走標準 `context.Context`（`r.Context().WithValue` 須以自定 key 型別，不得用字串 key 冒充）。此約束寫入 adapter 實作審查清單，框架決定後由實作軌落實。

## 4. 推薦（有依據，決定權在使用者）

**推薦：Echo v5（版本釘選不低於 v5.4.4，CI 覆核當日最新 patch；Go 要求以 CI 驗證為準）。**

理由（按權重排序）：

1. **集中式錯誤處理最貼合 ETHER 的錯誤碼規範**：`ERROR_CODES.md` 的信封、`request_id`、狀態碼對應在一處落實，覆蓋全部路由（含 404／405），遺漏風險最低。
2. **單一 module 內的常用中介軟體**：CORS、Recover、RateLimiter、CSRF、Secure headers、RequestID 等一次到位，減少 Phase 01 的依賴管理面；分散式限流仍須自寫 Valkey 後端（見 §3.2 提醒），此點兩框架相同。
3. **WS 升級輔助與 `net/http` 互通**（`WrapHandler`／`WrapMiddleware`）：Phase 05 閘道可在同一 adapter 內完成升級與中介軟體鏈，標準庫 handler 簽名不變。
4. **context 為介面**，與 transport 隔離設計的配合度略優；切換成本被限制在單一 adapter 內（見 §5）。

## 5. 若選另一個（Gin）的代價

- 須引入多個 `gin-contrib` 模組（CORS、JWT 輔助、Rate Limit、壓縮、指標等）並逐一釘選版本；依賴面較散，升級時須逐模組核對。
- 錯誤信封須以「全部 handler 經同一包裝／中介軟體轉譯」的紀律落實，`Bind` 家族的自動 400 行為必須禁用（統一用 `ShouldBind`＋自訂轉譯），否則會有路由繞過信封；審查時須逐路由核對。
- WS 升級須直接以 `gorilla/websocket`（或同類函式庫）在 handler 內完成，與 Echo 方案的底層函式庫相同，工作量差距不大。
- 換得的好處：最大社群與最多現成問答、v1 API 凍結的相容穩定性、發版慢但變動小的可預測性。若使用者更看重「社群規模與 API 穩定」而非「集中式錯誤處理」，Gin 是合理的第二選擇，代價如上、可接受。

## 6. 決定後的第一步（建議，不屬本備忘錄的決定）

1. 在 CI 以 `go 1.25` 工具鏈驗證所選框架版本的建置（Gin 的 Go 要求說法不一，見 §2 備註）。
2. 以一個 Phase 03 尖峰（spike）驗證 adapter：一個需認證的端點＋錯誤信封＋Rate Limit 中介軟體，經 `Router` 介面註冊，handler 不 import 框架套件；通過後才大規模採用。
3. 建立依賴更新紀律：釘選版本、定期檢查安全公告（Echo 的 2026-10 批次為前車之鑑；Gin 亦同，不得假設任一框架無 CVE）。
4. 框架決定以 ADR 記錄（`SYSTEM_ARCHITECTURE.md` §10 缺口 1 要求 Phase 01 前以 ADR 定案），並同步更新 `services/api/internal/transport` 的 adapter 實作與本骨架的 README。

## 7. 未查證／未決定事項

- 兩框架在 ETHER 實際負載下的延遲與吞吐：未實測，不得以公開基準數字代替；待 profiling 證據。
- WS 函式庫選型（`gorilla/websocket` vs `coder/websocket` 等）：待 Phase 05 決定，不屬本次框架決定。
- 驗證函式庫的最終選型（`go-playground/validator` 的版本與自訂規則）：待 Phase 03 依端點 Schema 定稿。
- OpenAPI 產出方式（Code-first／Spec-first，`API_STANDARDS.md` §10）：待定稿，框架選擇不預設其一。
