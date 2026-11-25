狀態：已定案（2026-11-25 審查決議）
決議來源：使用者（審查者）2026-11-25 07:19 於對話發出的《ETHER Phase 01 審查決議》全文（八節）；執行計畫 docs/phase01/EXECUTION_PLAN.md（修訂版 v5）§3。

# ADR-016：採用 Echo v5


- **狀態**：已定案（2026-11-25 審查決議）。備忘錄 `docs/phase01/gin-vs-echo-memo.md` 保留為決策紀錄；其中與本 ADR 牴觸的技術描述（`Context` 型別、WebSocket 責任）以本 ADR 為準。
- **背景**：Phase 00 要求框架二選一在 Phase 01 前以 ADR 定案（SYSTEM_ARCHITECTURE.md §10 缺口 1）；骨架以 `internal/transport` 的 Router 介面＋標準庫實作維持框架無關，等待決定。
- **決議**：採用 Echo v5，版本不得低於 **v5.4.7**；實際引入依賴前重新核對官方最新穩定 patch；保留 `internal/transport` 的 Router 介面，Handler 不得直接依賴 Echo 型別，經 Transport Adapter 整合；不得藉技術描述更正自行改選 Gin 或其他框架。
- **技術事實查證（2026-11-25，官方文件／API 證據）**：
  1. **`Context` 為具體型別**：Echo v5 的 `Context` 是具體 struct 型別（`type Context struct`，`echo/v5/context.go`；`NewContext(...) *Context`），handler 接收 `*echo.Context`。**不是 interface**，不得將其描述為介面。證據：Echo v5 原始碼（Go Documentation Server 鏡像）與 pkg.go.dev `github.com/labstack/echo/v5` 套件頁。
  2. **集中式錯誤處理**：`Echo` 具 `HTTPErrorHandler` 欄位，型別 `HTTPErrorHandler func(c *Context, err error)`，未設定時使用 `DefaultHTTPErrorHandler(false)`；另有 `ProblemDetailsHTTPErrorHandler`（v5.4.0 起）。證據：`echo/v5/echo.go` 原始碼與 pkg.go.dev v5 套件頁。
  3. **中介軟體**：`github.com/labstack/echo/v5/middleware` 含 CORS、CSRF、RateLimiter、Recover、RequestID、Secure（各含 `WithConfig` 變體）。證據：pkg.go.dev `github.com/labstack/echo/v5/middleware`（2026-11-25 查閱）。
  4. **WebSocket 責任分界**：Echo 官方 cookbook 的 WebSocket 範例以外部套件（`golang.org/x/net/websocket` 或 `gorilla/websocket`）的 Upgrader 在 handler 內升級，取用 `c.Response()`／`c.Request()` 完成交握；Echo 本身不提供 WebSocket 實作。證據：echo.labstack.com cookbook〈WebSocket〉。WebSocket 套件選型屬 Phase 05，本次不定。
  5. **與 `net/http` 互通**：Echo handler 可取用底層 `*http.Request`／`http.ResponseWriter`（同上 cookbook 範例），外部 WebSocket 套件可在同一路由內完成升級；此為配合條件，不是 Echo 的 WebSocket 能力。
- **理由（按備忘錄權重，已按證據修正；v3 的「`Context` 為介面」「WS 升級輔助」兩項描述作廢）**：① 集中式錯誤處理——自訂 `HTTPErrorHandler`（`func(c *Context, err error)`）可統一輸出 `ERROR_CODES.md` 的 `ETHER_` 信封與 `request_id` 規則；404／405 是否經同一 handler 覆蓋，須於 adapter 實作時以測試驗證，不得預設；② 單一 module 內含常用中介軟體——CORS、Recover、RateLimiter、CSRF、Secure headers、RequestID 皆屬 `echo/v5/middleware`（見技術事實 3），減少依賴管理面；③ 與 `net/http` 互通——Echo 提供 HTTP 路由與 middleware 配合，Phase 05 閘道可在同一 adapter 內完成 HTTP 部分；**WebSocket 升級與通訊由獨立 WebSocket 套件負責，Echo 不提供完整 WebSocket 實作**（見技術事實 4）；④ 框架隔離機制——隔離由既有 `internal/transport` Router 介面及 Transport Adapter 負責，**不依賴 Echo `Context` 是介面**：Adapter 在內部接收 `*echo.Context` 並轉換為 transport 介面，Handler 只見 transport 介面；`Context` 為具體型別的事實不削弱隔離，隔離來自 Adapter 邊界本身。分散式限流仍須自寫 Valkey 後端，框架內建限流器不得為唯一邊界（兩框架相同）。
- **替代方案（Gin）與代價**：最大社群與 v1 API 凍結的穩定性；代價為多個 `gin-contrib` 模組逐一釘選、錯誤信封須以全路由包裝紀律落實（禁用 `Bind` 自動 400，統一 `ShouldBind`＋自訂轉譯）。未採納，備忘錄保留為決策紀錄。
- **版本查證紀錄（修訂版 v2 修正，審查意見 2）**：**v5.4.7 已有發布紀錄**。查證：Go module proxy `proxy.golang.org/github.com/labstack/echo/v5/@v/v5.4.7.info`，2026-11-25 查得 `{"Version":"v5.4.7","Time":"2026-11-20T18:11:23Z","Origin":{"VCS":"git","URL":"https://github.com/labstack/echo","Hash":"59aacddc4a13904b8a80a76f1ec838c495b5feca","Ref":"refs/tags/v5.4.7"}}`，與審查者查證的發布日 2026-11-20 一致。原 R1「是否存在尚未驗證」的描述作廢。正式引入依賴前仍須查證官方最新穩定版本；**版本下限不得低於 v5.4.7，不得自行降低**；如官方資訊或相容性出現問題，停止並提交評估（見 §7 R1）。
- **影響範圍**：僅決策紀錄與註解同步（本次不改 `go.mod`／`go.sum`、不引入依賴）；後續 adapter 實作須遵守 `gin-vs-echo-memo.md` §3.8 的關鍵約束：handler 不得讀取框架 context 的擴充能力（Echo 即 `*echo.Context` 的 `Get`／`Set` 傳值）；跨 handler 的請求狀態一律走標準 `context.Context`，以自定 key 型別傳值，不得用字串 key 冒充；handler 簽名與程式碼不得出現 echo 型別，echo 型別只出現在 Adapter 內部。
- **重估條件**：Echo v5 出現無法繞過的重大安全或相容問題、或 profiling 證明 HTTP 層為瓶頸且 Gin 實測顯著更優，須以實測證據與新 ADR 重估；不得以社群規模預測觸發。
- **執行檢查點**：依賴引入批次覆核官方 API（`Context` 型別、`HTTPErrorHandler` 簽名、middleware 清單）與本 ADR 技術事實一致；adapter 測試證明錯誤信封覆蓋含 404／405；grep 證明 handler 層零 echo 型別（Adapter 內部除外）。
