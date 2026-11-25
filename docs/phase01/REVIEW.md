# ETHER Phase 01 基礎建設骨架 — 審查報告

狀態：草案／待審查。協調員：Phase 01 協調員（四軌並行：Flutter、React、Go、Infra）。
執行期間：2026-11-25 01:01（使用者批准 Phase 00）至本報告。**本階段結束後停止，
審查通過前不得進入 Phase 02。** 全部成果只留在 `~/workspace/ether/` 本機，
未 push、未建 repo、未發 PR／Release、未建立 `.github/` 目錄。

> **審查結論先行（協調員親自交叉核對後）：骨架已按手冊 C 交付齊全，靜態檢查全數通過；
> 但 Phase 01 驗收條件中所有需要「執行」的項目（建置、Docker 啟動、健康檢查、資料層連線、
> CI 通過）皆未驗證——本機 VM 無 flutter／go／docker，無法驗證。審查者須決定：接受骨架現況、
> 把執行驗證併入 CI／部署環境建立後補（建議），或要求先有 CI 驗證再算 Phase 01 完成。
> 另有兩項選型（Gin／Echo、Qdrant／pgvector）與一項佈局裁決（G-01）須使用者決定，
> 詳見 §5。**

## 1. 完成功能清單（骨架層級，無 Phase 02 以後的功能）

| # | 工作項（手冊 C／ROADMAP Phase 01） | 交付狀態 |
|---|---|---|
| 1 | Monorepo 目錄（§26） | 完成：`apps/mobile`、`apps/admin`、`apps/web`、`services/api`、`services/realtime-gateway`、`services/worker`、`infrastructure/`、`docs/`；根目錄 `README.md`、`.gitignore`、`Makefile` 新建 |
| 2 | Flutter 骨架 | 完成：pubspec 固定版本、Material 3 基底、Riverpod／GoRouter／Dio／Drift／flutter_secure_storage 宣告、8 色 tokens、單一佔位頁（零功能、零按鈕）、12 個 feature 空目錄、runner 說明文件 |
| 3 | React 骨架（Web＋Admin） | 完成：各自 package.json 固定版本、TS 嚴格模式、Vite、ESLint／Prettier、8 色 CSS 變數、單一佔位頁（零功能、零 API 呼叫）；Admin 另註明管理員權限絕不由前端判斷 |
| 4 | Go 後端骨架（框架無關） | 完成：三個獨立 module（`ether.local/*` 暫定）、stdlib `net/http`＋`Router` 介面隔離（框架綁定限於未來 adapter）、`GET /healthz` 契約、config／secrets／telemetry 骨架、22 模組 `doc.go`（零功能程式碼）、realtime-gateway WS 501 佔位、worker 任務佔位 |
| 5 | Migration 骨架 | 完成：`MIGRATIONS.md` 版本化慣例＋`000001_baseline`（僅 `schema_meta` 佔位表）；64 張業務表一張未寫，待 Phase 03 起逐階段落地 |
| 6 | Infra：Dockerfile | 完成：三份多階段建置、固定映像版本、非 root、HEALTHCHECK（worker 依 compose 未設，不自創） |
| 7 | Infra：Makefile／Task Runner | 完成：build／lint／test／migrate／compose-up／verify-static；缺工具鏈時明確報錯、非零碼退出 |
| 8 | Infra：`.env.example` 擴充 | 完成：僅檔尾附加 Phase 01 段落（`_OVERRIDE`、OTEL、Loki、OpenBao 變數），全佔位符；上半部原有內容一字未動（見 §7） |
| 9 | Infra：OpenBao／Loki／OTel 接入 | 完成：`secrets-management.md`（KV／policy 慣例）、`observability-wiring.md`（已接線 vs 待接線清單）；SDK 接線待後階段，骨架只留介面與環境變數 |
| 10 | Infra：GitHub Actions workflow 草稿 | 完成：`infrastructure/ci-drafts/` 兩份草稿＋README（粗體警告：絕對不得以 push 建立 `.github/workflows/`，須經雲端瀏覽器＋使用者工作階段手動建立） |
| 11 | 比較備忘錄 | 完成：`gin-vs-echo-memo.md`（推薦 Echo v5，見 §6 更正）、`vector-store-memo.md`（推薦 pgvector）；皆附查證日期與來源、皆明載推薦≠決定 |

**未完成（誠實列出，非缺漏隱瞞）：** 全部專案建置、Docker 啟動、API 健康檢查實跑、
App／Web／Admin 啟動、PostgreSQL／Valkey／NATS 連線、CI 自動檢查——皆為「待 CI 或有
Docker 的環境驗證」，本階段一項未執行。`livekit.yaml`、`prometheus.yml` 未建
（Phase 00 已知缺口，compose 參照它們，見 G-02／G-03）。

## 2. 新增／修改檔案清單

- **新增 108 個檔案**（以 Phase 00 的 42 個檔案為基線比對，`find` 實查）：
  Flutter 軌 15 個、React 軌 25 個、Go 軌 49 個、Infra 軌 19 個（含根目錄
  `README.md`、`.gitignore`、`Makefile`、3 份 Dockerfile、3 份 `.dockerignore`、
  CI 草稿 3 個、Phase 01 文件 9 個）。完整清單見各軌筆記
  （`flutter-notes.md`、`react-notes.md`、`go-notes.md`、`infra-notes.md`）。
- **空目錄 12 個**：`apps/mobile/lib/features/` 下 auth、chat、contacts、friends、
  groups、communities、calls、timeline、stickers、notifications、settings、navigation
  （皆為空骨架，無程式碼）。
- **修改 1 個檔案**：`.env.example`，僅檔尾附加（第 112 行起「Phase 01 附加」段落，
  全檔案 156 行）；上半部原有內容經逐位元組比對一字未動。
- **協調員整合修正（皆為新產生 Phase 01 檔案內的文字更正，未動 Phase 00 文件）：**
  ① gateway `main.go` 註解與 `README.md` 稱「Dockerfile 假設 `./cmd/gateway`」——
  實查 Dockerfile 建置路徑為 `./cmd/realtime-gateway`，說法已過時，已更正；
  `go-notes.md` 兩處同款舊說法一併更正。② 三份 Dockerfile「尚無 go.sum」註解——
  實查三個 `go.sum` 為 0 位元組空檔，已更正為「空檔（零外部依賴），不 COPY」。
  ③ 根目錄 `README.md` 的並行軌道舊敘述（稱 mobile／Go 骨架「尚未建立／未見」），
  已更新為四軌完成後的現況。
- **未修改**：全部 42 個 Phase 00 檔案（SPEC-SOURCE.md、docs/ 既有文件、
  docker-compose.yml）一字未動；`~/workspace/messenger-app/` 一字未動。

## 3. 測試與驗證報告（只做靜態檢查，皆標待 CI 驗證）

本機 VM 無 flutter／go／docker／psql，**沒有任何編譯、建置、啟動、測試被執行過**。
以下為協調員親自重跑／覆核的靜態檢查結果：

1. `make verify-static`：PASS（exit 0）。檢查：compose YAML 解析（18 服務）、
   Dockerfile 指令、引用一致性、secrets 掃描；警告 3 項（G-01、livekit.yaml、
   prometheus.yml 未建），見 §4。
2. 檔案樹核對：Phase 00 基線 42 檔全數存在；新增 108 檔；無 `.github/`、無 `.git`、
   無 `node_modules`、無 `messenger-app` 字串殘留。
3. 色碼逐字核對：Flutter `app_colors.dart`（`0xFF` 格式）與兩份 `tokens.css` 皆恰為
   DESIGN_TOKENS.md §1.1 的 8 個色碼；三端程式碼內無其他十六進位色碼。
4. Secrets 掃描：全部 Phase 01 新檔案以憑證賦值模式掃描，零命中；`.env.example`
   全文掃描無真實金鑰樣式的值（僅 `change-me`／`example.invalid` 佔位符）。
5. Go 骨架核對：三服務僅註冊 `GET /healthz`（gateway 另有 `/ws/v1` 501 佔位）；
   零業務前綴路由、零 §21 WS 事件名、零框架 import（`gin-gonic`／`labstack/echo`／
   `gorilla/websocket` 零引用）；22 個模組 `doc.go` 齊全；migration 僅 `schema_meta`。
6. Compose 對照：api（8080）／gateway（8081）healthcheck 路徑與程式一致；
   config 要求的 6 個依賴位址與 compose 的 api `environment` 逐名一致；
   `_OVERRIDE`、OTEL／LOKI／SENTRY、OPENBAO 變數名與 `.env.example` 附加段落一致。
7. 版本固定核對：pubspec（flutter_riverpod 2.5.1、go_router 13.2.0、dio 5.7.0、
   drift 2.18.0、flutter_secure_storage 9.2.2）、package.json（React 19.1.1、
   TypeScript 5.8.3、Vite 6.3.5，全部 exact）、go.mod（`go 1.25`，與 Dockerfile
   的 `golang:1.25.0-alpine3.21` 一致）、Dockerfile 映像（`golang:1.25.0-alpine3.21`、
   `alpine:3.21.3`）。**全部固定版本皆未經套件解析驗證，CI 首次實跑須覆核。**
8. 比較備忘錄抽查（協調員 2026-11-25 另行查證）：
   - Gin v1.12.0（2026-02-28、Go 1.25+）：多來源一致，屬實。
   - Echo v5（2026-01-18 主線）、v5.4.4（2026-10-30）：屬實；但**備忘錄的版本下限已過時**：
     2026-11-12 出 v5.4.5、2026-11-15 出 **v5.4.6（含 proxy 安全修正 GHSA-h6x5-phvw-c55c）**。
     決定時下限須提高為不低於 v5.4.6，並在 CI 覆核當日最新 patch（見 §6 更正）。
   - Echo 2026-10 安全公告批次（v4.16.1／v5.4.1 修正）：屬實。
   - Qdrant 1.20.0（2026-10-22、BM25／TurboQuant）：屬實；備忘錄的 v1.20.2 patch
     號未逐項覆核，決定時以官方 release 頁為準。
   - pgvector 0.8.7（2026-10-01、PostgreSQL License、維度上限）：屬實；另 0.8.7
     修正 CVE-2026-103484（IVFFlat），採用時須不低於此版。
   - 備忘錄版本來源部分為第三方彙整站（非官方 release 頁），決定前須以官方來源覆核；
     兩份備忘錄皆未引用基準測試數字（明確拒絕編造），此點符合要求。

## 4. 已知問題

- **G-01（最優先，見 §5 阻塞項 1）**：compose 的 gateway／worker 以 `services/api`
  為 build context、參照不存在的 `Dockerfile.gateway`／`Dockerfile.worker`；
  Phase 01 另建了三份獨立 Dockerfile。裁決前 `docker compose build` 必然失敗。
- **G-02／G-03**：compose 參照的 `infrastructure/docker/livekit.yaml` 與
  `prometheus.yml` 未建（Phase 00 已知缺口）。
- **G-04～G-08**（見 `compose-alignment-notes.md`）：OpenBao／Keycloak 開發模式佔位、
  映像標籤未固定（含 qdrant 未釘選標籤）、traefik ping 待驗證、Coraza／vLLM／
  OTel Collector 無 compose 服務、compose 位置與 §26 字面不符、worker 無 healthcheck
  （但 worker 程式有 `/healthz` 佔位，存廢待定稿）。
- **佔位碼**：gateway WS 佔位用 `ETHER_NOT_IMPLEMENTED`，`ERROR_CODES.md` 並無
  501／NOT_IMPLEMENTED 定稿碼，待 Phase 05 定稿，不得視為已定錯誤碼。
- **重複程式碼**：`internal/health` 在三個獨立 module 下各一份，為有意為之（獨立部署、
  避免跨 module 依賴），非疏漏。
- **空 `go.sum`**：三個 0 位元組空檔，僅因零外部依賴；新增第一個外部依賴時須產生
  真實 `go.sum` 並補回 Dockerfile 的 COPY。
- **共用 token 缺口**：Web／Admin 的 `tokens.css` 為兩份獨立副本（§26 未定義共用
  套件目錄，未擅自發明 `packages/`），漂移風險記為 Phase 02 待決。
- **React 修正未驗證**：`App.tsx` 以 `import type { ReactElement }` 避 React 19 UMD
  global 型別問題，待 `tsc` 驗證。
- **Migration 工具未定**：執行者與工具（golang-migrate 或等價）待定稿；api Dockerfile
  未複製 `migrations/`，接線待 Phase 03。
- **`/healthz` 僅 liveness**：依賴連線檢查為 TODO；readiness 分離待定案。
- **Module path 暫定**：`ether.local/*` 為佔位符，待 repo 建立後修訂。

## 5. 阻塞項（須使用者決定，一件一件問）

1. **G-01 佈局裁決（二選一，影響 compose 與 Dockerfile 的對應關係）：**
   (A) 改 `docker-compose.yml`，使 gateway／worker 參照各自獨立的 Dockerfile
   （`services/realtime-gateway/Dockerfile`、`services/worker/Dockerfile`）；
   (B) 維持單一 context（`services/api`），補 `Dockerfile.gateway`／`Dockerfile.worker`
   兩檔。協調員建議 (A)（與三個獨立 module 一致），但修改已批准的 compose 須使用者批准。
2. **Gin／Echo 二選一**：備忘錄推薦 Echo v5（下限更正為不低於 v5.4.6，見 §3.8）。
   決定後須以 ADR 記錄，並由 transport adapter 落地；決定前骨架維持標準庫實作。
3. **Qdrant／pgvector 二選一**：備忘錄推薦 pgvector（向量與權限事實同一交易、維運面
   最小）；選 pgvector 時 compose 內的 qdrant 服務處置（移除或保留評估）須一併決定。
4. **部署主機與成本方案**（Phase 00 既有阻塞項，Phase 01 驗證的前提）：CI／Docker
   環境、正式主機皆未定；不得新增付款方式。
5. **外部憑證申請順序**（Phase 00 既有阻塞項）：FCM／APNs／Apple／Google／簡訊 OTP／付款。
6. **Keycloak Realm／Session 銜接細節、Migration 工具、測試框架、映像倉庫與分支規則**：
   皆待定稿（見各軌筆記）。
7. **CI workflow 建立方式**：草稿已備妥，但 `.github/workflows/` 須經雲端瀏覽器＋
   使用者工作階段手動建立（GitHub App 無 workflows 權限，push 會回 403），
   須使用者配合操作或授權。

## 6. 決策紀錄

- **本階段未新增 ADR。** 兩項選型僅有「推薦」，決定權在使用者，決定後須補 ADR。
- **框架無關設計定案（骨架層級）**：`internal/transport` 的 `Router` 介面＋stdlib
  實作；handler 只依賴標準庫型別；未來 Gin／Echo adapter 實作同一介面替換，不得讓
  handler 直接 import 框架。此設計已落地，不因選型結果改變 handler 程式碼。
- **向量存取介面化**：`MODULE_BOUNDARIES.md` §7 方向——向量檢索歸屬在定案前不得由
  Search／Analytics 模組預設擁有，選擇藏在 AI Provider Adapter 介面之後。
- **備忘錄更正（協調員抽查後）**：Echo 推薦下限由「不低於 v5.4.4」更正為
  「不低於 v5.4.6（2026-11-15 安全修正），CI 覆核當日最新 patch」。
  備忘錄原文未改（它是查證日的紀錄），更正以本報告為準，決定時須再覆核。
- **其餘約定**：module path 暫定 `ether.local/*`；`go` directive 固定 1.25；
  映像版本固定於 2026-11-25；版本升級須經審查，不得改浮動標籤。

## 7. 文件同步說明

- Phase 00 文件：42 個檔案一字未動（僅 `.env.example` 檔尾附加，見 §2）。
  `docs/phase01/` 新增 9 個文件（四軌筆記、兩份備忘錄、secrets、可觀測性、
  compose 對照）＋本審查報告。
- 根目錄 `README.md` 為 Phase 01 新建，已由協調員更新為四軌完成後的現況；
  各服務／各 app 的 README 為各軌撰寫，狀態皆標「骨架、待 CI 驗證」。
- 舊路線 `~/workspace/messenger-app/` 未碰；品牌文件未動；App Icon 原稿未動，
  launcher PNG 匯出仍為 Phase 02 工作，不得宣稱已完成。
- 本報告的數字（檔案數、版本號、抽查結果）皆以 2026-11-25 實查為準；
  之後版本變動須重新查證，不得沿用。

## 8. 審查請求

請審查者（統籌／使用者）逐項確認：

1. Phase 01 骨架是否接受現況（靜態檢查通過、執行驗證全數待 CI），可否視為
   「骨架定稿」、把執行驗證併入 CI／部署環境建立後補？若要求先有 CI 驗證，
   Phase 01 不算完成，須先建 CI 環境（阻塞項 4、7）。
2. G-01 佈局裁決選 (A) 還是 (B)（見 §5.1）？
3. Gin／Echo 選型：是否採納 Echo v5 推薦（下限 v5.4.6）？或選 Gin？
4. Qdrant／pgvector 選型：是否採納 pgvector 推薦？qdrant 服務如何處置？

**以上四項未全部確認前，不得進入 Phase 02。** 確認後，決定事項須補 ADR
（選型兩項）與 compose 修改（G-01 選 A 時），再由下一階段承接。
