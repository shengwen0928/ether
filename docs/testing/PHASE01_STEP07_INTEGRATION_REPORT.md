# Phase 01 Step 7 隔離整合驗收報告（範本）

> 狀態：**BLOCKED**——尚無 workflow 執行紀錄。Repository 已建立
> （shengwen0928/ether，公開、空、預設分支 main，2026-11-25 建成並複驗）。
> 本檔案為待填範本，不得將任何未執行項標示為 PASS。第一次實機執行後，
> 由執行者依實際證據逐項填寫，並將狀態改為實際結果。
>
> 前置查核摘要（撰寫時背景，執行前須重新核對）：
>
> - Repository 已建立：shengwen0928/ether（公開、空、預設分支 main；
>   2026-11-25 19:19 以使用者登入工作階段經網頁建立，API 複驗 size 0、
>   無分支；本機 `~/workspace/ether/` 無 `.git`，以 snapshot push 首次上傳）。
> - 帳務（2026-11-25，parent 以瀏覽器唯讀查核，本草稿未再查證）：GitHub
>   Pro $0（折扣剩 2 年）；Actions 預算 $0.00 上限、狀態為 Usage blocked
>   （超出內含用量即阻擋，不會自動超額計費）；11 月已用 Linux 218 分鐘
>   （$1.31 全數折扣）、儲存 391.72 GB-hr（$0.13 全數折扣）、內含運算
>   額度 $18 尚餘、儲存 0 GB／2 GB。
> - Workflow 檔案須經網頁建立：GitHub App 沒有 workflows 寫入權限，以
>   API 推送 `.github/workflows/` 會回 403，不得以 API 推送冒充已建立。
> - Checkout／upload-artifact 的完整 commit SHA 已查證填入
>   （2026-11-25 經 get_tag 查得 v4：checkout
>   11d5960a326750d5838078e36cf38b85af677262、upload-artifact
>   ea165f8d65b6e75b540449e92b4886f43607fa02）。

## 五態圖例

| 狀態 | 定義 |
|---|---|
| PASS | 有真實執行證據且通過。 |
| FAIL | 已執行且失敗。 |
| BLOCKED | 因外部條件無法執行（須寫明真正阻斷原因）。 |
| NOT IMPLEMENTED | 產品功能尚未實作（不得以測試通過冒充）。 |
| NOT VERIFIED | 尚無足夠證據（執行未完成、證據不足或觀察結果無法斷言）。 |

## 執行資訊（待填）

| 欄位 | 值 |
|---|---|
| GitHub Actions Run ID | 待填 |
| Repository（可見性） | 待填（不得擅自變更可見性） |
| Workflow 檔案建立方式 | 待填（須為網頁建立；API 推送會回 403） |
| Checkout commit SHA | 待填 |
| Runner OS | 待填 |
| CPU／RAM／Disk | 待填 |
| Docker Engine 版本 | 待填 |
| Docker Compose 版本 | 待填 |
| Buildx 版本 | 待填 |
| Go 工具鏈版本 | 待填 |
| Node 版本 | 待填 |
| Compose project 名 | `ether-isolated-integration` |
| 測試 Realm | `ether-ci`（專屬 disposable realm） |
| 測試 MinIO bucket | 待填 |
| 測試 NATS Stream | `ETHER_EVENTS` |
| 測試 Meilisearch 索引 | `messages` |

## 映像 Tag 與 Digest（待填，須附 `buildx imagetools inspect` 輸出）

| 映像 | Tag | Digest（實際） | 覆核結果 |
|---|---|---|---|
| `pgvector/pgvector` | `0.8.7-pg16-bookworm` | 待填（固定值 `sha256:484e838076e04939d27d3e176e91c86eea96236126023be8d3059761122e579f`，不符即停） | 待填 |
| `traefik` | `v3`（浮動，G-04） | 待填 | 待填 |
| `valkey/valkey` | `8`（浮動，G-04） | 待填 | 待填 |
| `nats` | `2`（浮動，G-04） | 待填 | 待填 |
| `minio/minio` | 未標 tag（`latest`，G-04） | 待填 | 待填 |
| `getmeili/meilisearch` | `v1`（浮動，G-04） | 待填 | 待填 |
| `openbao/openbao` | 未標 tag（`latest`，G-04） | 待填 | 待填 |
| `quay.io/keycloak/keycloak` | `26`（浮動，G-04） | 待填 | 待填 |
| `livekit/livekit-server` | 未標 tag（`latest`，G-04） | 待填 | 待填 |
| `ollama/ollama` | 未標 tag（`latest`，G-04） | 待填 | 待填 |
| `prom/prometheus` | 未標 tag（`latest`，G-04） | 待填 | 待填 |
| `grafana/grafana` | 未標 tag（`latest`，G-04） | 待填 | 待填 |
| `grafana/loki` | 未標 tag（`latest`，G-04） | 待填 | 待填 |
| `glitchtip/glitchtip` | 未標 tag（`latest`，G-04） | 待填 | 待填 |
| `golang`（建置階段） | `1.25.0-alpine3.21` | 待填 | 待填 |
| `alpine`（執行階段） | `3.21.3` | 待填 | 待填 |

## 逐 Stage 結果（待填）

| Stage | 檢查點 | 狀態 | 證據（artifact 檔名／命令輸出） |
|---|---|---|---|
| Stage 1 | 環境檢查、§9.1 前置檢查、新 Runner 無既有 ether volume 證明、無 `.env`、無正式憑證 | 待填 | 待填 |
| Stage 2 | pgvector digest 覆核、三映像建置、S2-6 負向 context 檢查、S2-7 pgvector 探測（擴充可用未安裝） | 待填 | 待填 |
| Stage 3 | `compose config` 雙流程（S1-2 檢查點） | 待填 | 待填 |
| Stage 4 | 9 項服務健康檢查、Keycloak 資料庫建立、MinIO bucket 預建、NATS Stream 預建 | 待填 | 待填 |
| Stage 5 | `cmd/migrate up`（000001–000014）、資料表／約束抽查、資料庫整合測試、Session 輪換／重放持久化測試 | 待填 | 待填 |
| Stage 6 | Realm 匯入、雙使用者、direct grant、discovery／aud、`TestLiveKeycloak`、換發／重放觀察、登出、無效 token 401 | 待填 | 待填 |
| Stage 6b | 授權碼＋PKCE 腳本化嘗試 | 待填 | 待填 |
| Stage 7 | 雙帳號 Profile／裝置／好友／DM／訊息持久化／權限隔離；工作階段撤銷（清單為空時為 NOT VERIFIED，見下） | 待填 | 待填 |
| Stage 8 | NATS Stream 訊息數增加、Relay／消費迴圈日誌、durable 消費端存在性 | 待填 | 待填 |
| Stage 9 | Meilisearch 索引建立、消費迴圈寫入、搜尋命中、刪除後搜尋不到 | 待填 | 待填 |
| Stage 10 | S3 直接往返、api presigned 上傳往返、完成 Stat 校驗、授權 fail-closed 觀察 | 待填 | 待填 |
| Stage 11 | Go 三模組 build／vet／test、Web typecheck／lint／format／build／tests、gofmt 漂移記錄 | 待填 | 待填 |
| Stage 12 | `down -v` 清理（僅本次 project）、容器／volume／網路殘留檢查、臨時憑證清除 | 待填 | 待填 |

## EXECUTION_PLAN §6 對照（待填）

| 檢查項 | 狀態 | 備註 |
|---|---|---|
| S1-2（設定渲染） | 待填 | 見 Stage 3 |
| S1-4（promtool） | 待填 | platform-checks job |
| S2-6（建置＋負向檢查） | 待填 | 見 Stage 2 |
| S2-7（pgvector 探測） | 待填 | 見 Stage 2 |
| S2-8（Echo／pgvector 版本覆核） | NOT VERIFIED | 須於依賴引入批次重查官方最新版本；CVE-2026-103484 須附官方公告與適用性確認，不得只列編號 |
| S3-9（預設啟動、livekit 未啟動） | 待填 | platform-checks job，以實際容器狀態判定 |
| S3-10（監控健康檢查） | 待填 | platform-checks job |
| S3-11（LiveKit profile 負向流程） | 待填 | platform-checks job；預期 ErrKeysNotSet，不得加金鑰 |
| S4-12（api／gateway 健康檢查，經 Traefik） | NOT VERIFIED | Traefik 路由段未啟動（須網域＋ACME／TLS 測試設計定案）；直連健康檢查不得冒充 |
| S4-13（各服務連線） | 待填 | 見 Stage 4 |
| S4-14（Keycloak／GlitchTip 共用資料庫） | 待填 | Keycloak 見 Stage 4；GlitchTip 見 platform-checks（首啟失敗時記 NOT VERIFIED 附原因） |
| S5-15（驗收文件） | 待填 | 本報告即 S5-15 交付物 |

## 未通過／未驗證項目與真正阻斷原因（待填）

- 待填：逐項列出 FAIL／BLOCKED／NOT VERIFIED 項目、實際觀察與真正阻斷原因。
- 已知缺口（撰寫時即確定，執行後須覆核是否仍成立）：
  - HTTP 層沒有建立 Session 的端點：工作階段撤銷的 HTTP 檢查須視
    `/auth/sessions` 清單是否非空而定；清單為空時記 NOT VERIFIED，
    輪換／重放以 `TestPG_SessionLifecycleAndReplayPersists` 為證。
  - ETHER Session 撤銷的請求期強制檢查未實作（M1 已知缺口）：
    撤銷 ETHER Session 後，Keycloak token 在到期前仍可通過
    `RequireAuth`，不得將此描述為撤銷即時生效。
  - 授權碼＋PKCE 腳本化流程脆弱：登入表單結構隨 Keycloak 版本而變，
    失敗時只記 NOT VERIFIED 附原因，不得以 direct grant 冒充。
  - Keycloak refresh token 重放行為取決於 realm 設定（範本未開啟
    撤銷舊 token 的強制設定）：重放觀察結果只記錄，不得斷言為
    ETHER 家族撤銷證明。
  - 掃描器未接線：媒體檔案完成後掃描狀態恆為 pending，授權端點
    fail-closed；clean 檔案的授權下載往返尚無法驗證。
  - Meilisearch 索引設定預建（filterable／sortable）未在本 workflow
    完成，屬待決定 26；Stage 9 只建最小索引。
  - Traefik 路由段與 Ollama 段未驗證（見上表），Ollama 模型清單與
    GPU／VRAM 評估待 Phase 09 定稿。
  - gofmt 既有漂移（api `health.go`）未修復（須授權），Stage 11 只記錄。

## 清理結果（待填）

| 檢查 | 結果 |
|---|---|
| 本次 project 容器已清除 | 待填 |
| 本次 project volume 已清除 | 待填 |
| 本次 project 網路已清除 | 待填 |
| 臨時密碼／token 檔案已刪除 | 待填 |
| Runner 已退役（一次性 VM） | 待填 |

## 結論（待填）

待填：依五態圖例逐項定性，不得將 BLOCKED／NOT IMPLEMENTED／
NOT VERIFIED 標示為 PASS；Phase 01 最終驗收是否通過，須待全部必須項
皆為 PASS（或經審查者裁決的不適用項）後才可判定。
