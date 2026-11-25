# Infra 軌筆記（Phase 01，2026-11-25）

狀態：靜態檢查完成；Dockerfile 建置、Compose 啟動、CI 執行皆為「待 CI 或有
Docker 的環境驗證」。本機 VM 無 docker、無 go、無 flutter。

## 1. 檔案清單（本軌新增／修改）

| 路徑 | 用途 |
|---|---|
| `services/api/Dockerfile` | api 多階段 Dockerfile（golang:1.25.0-alpine3.21＋alpine:3.21.3，非 root，HEALTHCHECK 8080／healthz） |
| `services/realtime-gateway/Dockerfile` | gateway 多階段 Dockerfile（同上固定版本，非 root，HEALTHCHECK 8081／healthz；建置路徑 `cmd/realtime-gateway`） |
| `services/worker/Dockerfile` | worker 多階段 Dockerfile（同上固定版本，非 root，無 HEALTHCHECK，compose 未定義） |
| `services/api/.dockerignore` | api 建置 context 排除清單（.env／憑證／金鑰不得進映像） |
| `services/realtime-gateway/.dockerignore` | gateway 建置 context 排除清單（context 改為本目錄後生效，見 compose 對照 G-01） |
| `services/worker/.dockerignore` | worker 建置 context 排除清單（同上） |
| `Makefile` | build／lint／test／migrate／compose-up／verify-static；缺工具鏈時明確報錯、非零碼退出 |
| `.gitignore` | Go／Flutter／Node／.env／IDE／Docker 產物；`.env.example` 以否定規則保留 |
| `README.md`（新建） | 專案簡介、Monorepo 結構、快速開始、環境變數說明、文件索引、狀態與缺口連結；不描述已完成功能 |
| `.env.example`（僅附加） | 檔尾附加「Phase 01 附加」段落；上半部 4851 位元組逐位元組未動（已驗證，見 §2） |
| `docs/phase01/secrets-management.md` | OpenBao KV 路徑／policy 命名／環境注入／Provider 介面對照／開發模式限制（無真實 token） |
| `docs/phase01/observability-wiring.md` | 日誌格式慣例、OTLP／Loki 變數、compose 監控現況、已接線 vs 待接線清單 |
| `docs/phase01/compose-alignment-notes.md` | Compose 18 服務逐項對照、不一致與缺口 G-01～G-08（只核對，未改 compose） |
| `infrastructure/ci-drafts/github-actions-build.yml` | CI 建置草稿（Go／Docker／compose config／secrets 掃描），未驗證 |
| `infrastructure/ci-drafts/github-actions-pr-checks.yml` | CI PR 檢查草稿（閘門／secrets／依賴掃描示例），未驗證 |
| `infrastructure/ci-drafts/README.md` | 粗體強制流程：絕對不得以 push 建立 `.github/workflows/`，須經雲端瀏覽器＋使用者工作階段手動建立 |
| `docs/phase01/infra-notes.md` | 本檔案 |

未建立：`.github/`（全程不存在，`make verify-static` 防呆檢查通過）；
`services/api/Dockerfile.gateway`／`Dockerfile.worker`（compose 參照的兩檔，
見 G-01，待裁決後由裁決方決定補齊或改參照，本軌不擅自二選一）；
`infrastructure/docker/livekit.yaml`／`prometheus.yml`（Phase 00 已知缺口）。

## 2. 靜態驗證結果（皆於本機執行，均為靜態檢查，非建置／啟動驗證）

1. **Compose YAML 解析**（python3＋PyYAML 6.0.1）：18 個服務，名稱與
   CLOUD_ARCHITECTURE.md §5 一致；build／healthcheck／depends_on／環境變數
   逐項萃取核對，結果寫入 compose-alignment-notes.md §2。
2. **`make verify-static`**：初跑因 Dockerfile HEALTHCHECK 跨行書寫、
   grep 單行 pattern 失效而 FAIL，已修正檢查為「HEALTHCHECK 行＋URL 行分別
   存在」；修正後 **PASS（exit 0）**，警告 3 項（G-01 對齊缺口、
   livekit.yaml／prometheus.yml 未建），皆為已知缺口，不計入失敗。
3. **工具鏈缺失目標**：`make build`／`lint`／`test`／`migrate`／
   `compose-up` 皆以明確缺失訊息退出（make exit 2，非零碼），無靜默失敗。
   本機確認 `go`／`docker`／`flutter` 皆不存在（僅 `node`／`npm` 存在，
   本軌目標未使用）。
4. **`.env.example` 附加驗證**：附加前備份比對，附加後前 4851 位元組與
   原檔逐位元組一致；全文 secrets 掃描（`sk-`／`ghp_`／`gho_`／`AKIA`／
   私鑰標頭）無真實金鑰樣式的值，僅 `change-me`／`example.invalid` 佔位符。
   Makefile 自身的掃描 pattern 字串為唯一的 pattern 自我參照，非 secret 值。
5. **受保護檔案未動**：`SPEC-SOURCE.md`、`docker-compose.yml`、
   CLOUD_ARCHITECTURE.md、SELF_HOSTING.md、COST_OPTIMIZATION.md、
   TEST_STRATEGY.md、ROADMAP.md 的 sha256 與工作前快照一致；唯一允許的
   修改只有 `.env.example` 附加。
6. **Dockerfile 指令檢查**：三份皆為兩階段、固定基礎映像、非 root
  （`USER ether`，無 root）、api／gateway HEALTHCHECK 路徑與 compose 一致、
   worker 無 HEALTHCHECK；建置路徑已對照實際 `cmd/`（見 §4）。
7. **CI 草稿 YAML 解析**：兩份皆可解析；build 草稿 4 jobs、pr-checks 草稿
   3 jobs；`on` 觸發如 README 所述。未實跑。
8. **Go 骨架對照**（唯讀，未編譯）：三服務 go.mod 皆為 `go 1.25`，與
   Dockerfile 的 golang 1.25.0 一致；全樹 grep 無外部 import 路徑
   （僅標準庫），故 `go.sum` 不存在，Dockerfile 不 COPY `go.sum`（見 §4）。

## 3. 已知缺口與阻塞項

- **G-01（須協調員裁決，最高優先）**：compose 的 gateway／worker 參照
  `services/api` context＋`Dockerfile.gateway`／`Dockerfile.worker`（兩檔不
  存在），與本軌的獨立 Dockerfile 路徑不一致。裁決選項見
  compose-alignment-notes.md §3。裁決前 `docker compose build` 必然失敗。
- **G-02**：`livekit.yaml`、`prometheus.yml` 未建（Phase 00 已知缺口）。
- **G-03**：OpenBao `-dev`、Keycloak `start-dev` 開發模式佔位，不得用於正式環境。
- **G-04**：映像標籤未固定（9 個服務無標籤、其餘只到 major），與版本固定
  要求不一致。
- **G-05**：traefik healthcheck `--ping` 的 ping 入口設定待有 Docker 的環境驗證。
- **G-06**：Coraza、vLLM、OTel Collector、Web／Admin 在 compose 中無服務定義。
- **G-07**：compose 檔位置（`infrastructure/docker/`）與 §26 字面的根目錄
  位置不一致。
- **G-08**：worker 無 healthcheck；骨架雖有 `/healthz` 佔位，不被探測。
- **其他**：Migration 執行者與工具待定稿（Makefile `migrate` 為佔位檢查）；
  測試框架、Go／Flutter 版本定稿待 Phase 01 確認；Secrets 掃描／依賴掃描
  工具待 Phase 11 定稿；映像倉庫、分支保護規則待定稿；正式部署成本待查證
  （阻塞項，見 CLOUD_ARCHITECTURE.md §3.3）。仍可能付費項目（網域、伺服器、
  簡訊 OTP、Apple／Google 上架費、外部 AI、TURN 頻寬）維持「待查證」寫法，
  本軌未宣稱零成本。

## 4. 並行軌道對照（本軌執行期間他軌落地，本軌唯讀對照、未撰寫未修改）

本軌開始時 `services/` 不存在；執行期間程式碼軌並行建立了 Go 骨架
（各服務獨立 go.mod、`cmd/`、`internal/` 佔位、api migrations baseline、
health／config／secrets／telemetry stub），React 軌建立了
`apps/admin`／`apps/web` 的 `App.tsx` 佔位。對照後本軌修正了三處初稿錯誤
（皆為本軌自有檔案的修正，他軌檔案一字未動）：

1. Gateway 建置路徑 `cmd/gateway` → 實際的 `cmd/realtime-gateway`，
   二進位名同步改為 `realtime-gateway`。
2. Dockerfile 取消 `COPY go.sum`：go.mod 皆無外部依賴、`go.sum` 不存在，
   COPY 不存在的檔案會使建置失敗；新增依賴時須補 `go.sum` 與 COPY。
3. `.env.example` 附加段落：覆寫變數改為與 `config.go` 一致的無前綴
   `<NAME>_OVERRIDE` 形式（初稿的服務前綴形式無程式碼讀取，已刪除）；
   Loki 變數改為與 `telemetry.go` 一致的 `LOKI_ENDPOINT`，並補 `SENTRY_DSN`；
   secrets 文件 §4 改為對照實際 `Provider.Get(ctx, name) (string, error)`
   介面與 OpenBao stub，並記入 KV 路徑與鍵名的對映缺口。

他軌檔案的正確性與編譯狀態不在本軌驗證範圍；`docs/phase01/` 內他軌的
`flutter-notes.md`／`react-notes.md`（及後續出現的決策備忘）非本軌產出，
本軌未覆核、未修改。另注意：程式碼軌的
`services/realtime-gateway/cmd/realtime-gateway/main.go` 註解與同目錄
`README.md` 仍稱 Dockerfile 假設建置路徑為 `./cmd/gateway`，該敘述在本軌
修正後已過時（Dockerfile 現為 `./cmd/realtime-gateway`），本軌未動他軌檔案，
請協調員通知程式碼軌更新其註解。`~/workspace/messenger-app/` 本軌全程未碰。

## 5. 硬規則遵守自查

- 未建立 `.github/`、未 push、未建 repo／PR／Release；成果只留本機。
- 除 `.env.example` 附加外，未修改任何 Phase 00 文件與 compose；新增文件
  只在對應新目錄或 `docs/phase01/`。
- 無假資料、無憑證／金鑰／E2EE 明文；secrets 只走佔位符與 OpenBao 慣例。
- 未宣稱未驗證的事；未宣稱零成本；目錄結構以 §26 為準（compose 位置
  差異記為 G-07）。

## 6. 2026-11-25 決議修訂（dated amendment，原文不動）

- **三項選型已決議**：G-01 採用方案 A（獨立 Dockerfile 與獨立建置 context，見 ADR-015）；後端框架採用 Echo v5（版本不得低於 v5.4.7，見 ADR-016）；向量承載採用 pgvector（見 ADR-017）。
- **G-02 按執行計畫 §4 處理**：`livekit` 加入 `profiles: ["phase07"]`（預設不啟動）；`prometheus` 補最小有效設定（僅自我抓取）；其餘 G-03～G-08 維持追蹤。
- **G-01 裁決結果**：compose 改為參照三個獨立 context／Dockerfile；`Dockerfile.gateway`／`Dockerfile.worker` 不補齊。本文件 §3 G-01 的裁決選項敘述保留為核對紀錄。
