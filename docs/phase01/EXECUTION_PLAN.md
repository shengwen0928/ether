# ETHER Phase 01 審查決議執行計畫（修訂版 v5）

狀態：**修訂版 v5，待再次審查批准**。本文件為本次交付物；未經審查者批准，不得修改本計畫所列的任何檔案，不得開始執行。

決議來源：使用者（審查者）2026-11-25 07:19 於對話發出的《ETHER Phase 01 審查決議》全文（八節）。本計畫逐項對應決議 §八要求的七項內容。

修訂說明（v5，針對 v4 有條件批准的兩項修正，架構決議不變；v4 已認可的四項修正——ADR-016／ADR-017、LiveKit profile 驗證、§6 四類分類——保留）：(1) §10 執行順序與 §6 分類一致：C 類靜態工作計畫批准後可先行，§9.1 前置檢查僅 gating Docker 操作；(2) §9.2 回退條件補充：前置檢查失敗且未更換／未啟動時停止並回報，不得無必要停止現有服務；僅已實際更換／啟動且符合觸發條件時執行回退；任何情況不得 `down -v`、不得刪除 Volume。必要交叉引用同步：§6 A 類標籤、§9.1 標題。數量不變：38 個修改檔＋4 個新增檔。

## 決議摘要（審查者定案，本計畫不得更改）

1. **Phase 01 骨架：有條件接受**。接受四軌骨架交付為骨架定稿基礎；靜態檢查結果可接受；全部執行驗證（建置、編譯、Docker 啟動、健康檢查、資料層連線、CI）仍列未完成，Phase 01 不得標記最終驗收完成；須在完整工具鏈的 CI／Docker 環境完成執行驗證，通過後才可提交最終驗收報告。不得將靜態檢查 PASS 描述成完整系統測試通過。
2. **G-01：採用 A**。Gateway 與 Worker 各自使用獨立 Dockerfile 與建置 context。
3. **框架：採用 Echo v5**，版本不得低於 **v5.4.7**，實際引入依賴前重新核對官方最新穩定 patch；保留 `internal/transport` 的 Router 介面，Handler 不得直接依賴 Echo 型別，經 Transport Adapter 整合，不得破壞框架隔離設計。Gin 備忘錄保留為決策紀錄。
4. **向量庫：採用 pgvector**。以 PostgreSQL＋pgvector 為初期向量方案；移除 Compose 中的 Qdrant 服務；清理 Qdrant 環境變數與設定引用；保留 AI Provider Adapter 後方的向量存取抽象介面；不得讓 Search、Analytics 或其他業務模組直接綁定特定向量資料庫；pgvector 版本採用時須核對官方穩定版本與安全修正。未來若實測證明不足，再重估 Qdrant。
5. **G-02／G-03**：須提處理方案，不得讓 Compose 長期保留指向不存在檔案的掛載或設定；可補最小有效設定或暫時移除引用，但須明確記錄未啟用功能與後續接線階段。G-04～G-08 維持追蹤，不擴大本次範圍。
6. **ADR**：為 G-01、Echo、pgvector 三項決策建立正式 ADR，同步相關文件；ADR 須記錄理由、替代方案、影響範圍與重估條件。

### 0.2 ADR 編號確認（修訂版 v2 新增，審查意見 5）

- **ADR-014 已存在**：`docs/decisions/ADR-014-品牌更名ETHER暨設計系統方向.md`（3,097 bytes，2026-11-25 00:44 建立），主題為品牌更名 ETHER 暨設計系統方向，與本次三項決策無主題重疊。
- **編號規則**：`docs/decisions/` 內現有 ADR-001～014，檔名格式為 `ADR-NNN-<主題>.md`，按建立順序編號；**無獨立 ADR 索引檔**（目錄內僅 14 份 ADR 正文），故無索引同步事項。
- **結論**：ADR-015／016／017 順延編號，無衝突，可沿用。本次不新增索引檔（屬範圍外新增）。

## 0. 範圍邊界

**在本次範圍內**：§1 所列 38 個修改檔＋4 個新檔；靜態驗證（`make verify-static` 與 grep 勾稽，本機可執行部分）；計畫核准後、停止點前的全部修改。

**不在本次範圍內（決議 §七明令禁止）**：進入 Phase 02 或實作 Phase 02 以後業務功能；建立 GitHub Repository、`.github/workflows`、push／PR／Release；建立外部帳號或申請外部服務；新增付款方式或產生費用；修改與本次決策無關的 Phase 00 內容；改變 ETHER 品牌定義與既有產品規格。

**特別界定**：
- **不引入 Echo 依賴**：本次不改任何 `go.mod`／`go.sum`，不執行 `go get`。決議的 Echo 採用以 ADR 記錄為準；依賴引入與 Transport Adapter 實作屬後續 CI 驗證批次，引入前須重查官方最新 patch（見 §7 R1）。三個 `go.mod` 目前皆零外部依賴，`go.sum` 為空檔，此狀態本次不變。
- **不建立向量索引**：本次不寫任何向量 Migration，不執行 `CREATE EXTENSION`，不定 embedding 模型、維度與距離度量（皆為 Phase 09 範圍）。本次只讓 PostgreSQL 具備 pgvector 擴充可用性（`pg_available_extensions` 查得到 `vector` 且 `installed_version` 為 NULL，驗證 SQL 與標準見 §6 S2-7）。
- **G-04～G-08 不修**：浮動映像標籤、traefik ping 待驗證、Coraza／vLLM／OTel Collector 缺席、compose 位置與 SPEC-SOURCE.md §26 字面差異、worker 無 healthcheck，全部維持現況追蹤，僅在 §1 E1 補註中記錄「本次未動」。

## 1. 預計修改的檔案與目的

共 38 個修改檔、4 個新檔。Phase 00 文件同步範圍以 §1-D 附表 D-1 為準：每份文件列出原始位置、原始內容、修改後內容、修改理由與是否涉及結構或語意變更；比較內文與歷史敘述不改寫（僅加決議註記處除外，見附表）。Phase 01 審查報告 REVIEW.md 不改（歷史紀錄），決議以 ADR 為準。

### A. 建置與設定（9 檔）

| 檔案 | 修改內容 | 目的 |
|---|---|---|
| `infrastructure/docker/docker-compose.yml` | ① `realtime-gateway` 的 build 改為 context `../../services/realtime-gateway`、dockerfile `Dockerfile`；② `worker` 改為 context `../../services/worker`、dockerfile `Dockerfile`；③ 刪除 `qdrant` 服務定義與 `qdrant-data` volume；④ 檔頭註解「Qdrant／pgvector（選擇待定）」改為 pgvector 已定案＋ADR-017 指引；⑤ `postgresql` 的 `image: postgres:16` 改為固定映像 `pgvector/pgvector:0.8.7-pg16-bookworm@sha256:484e838076e04939d27d3e176e91c86eea96236126023be8d3059761122e579f`（標籤＋digest 雙固定；PostgreSQL 16、pgvector 0.8.7、bookworm 基底；digest 已於 2026-11-25 向 Docker Hub registry 查證，執行時覆核，不符即停下，見 §9；浮動標籤禁用）；⑥ `livekit` 刪除 `./livekit.yaml` 掛載與 `--config` 命令，加入 `profiles: ["phase07"]`（預設不啟動），補「未啟用」註解（見 §4）；⑦ `prometheus` 掛載改指新建的 `./prometheus.yml`（見 §4）；⑧ 檔頭「草案／待批准」狀態註記改為「已批准＋本次修訂」 | 落實 G-01 方案 A、移除 Qdrant、接通 pgvector 擴充、消除指向不存在檔案的掛載 |
| `.env.example` | ① 刪除「Qdrant（向量，選擇待定）」段落（`QDRANT_API_KEY` 佔位行）；② 第 11 行註解「Qdrant／pgvector（選擇待定）」改為 pgvector 已定案 | 清理 Qdrant 環境變數與設定引用；pgvector 不新增環境變數（擴充隨資料庫映像提供） |
| `services/api/Dockerfile` | 僅更新檔頭註解：建置參照現況說明（已與 compose 一致） | 註記與 compose 一致，無建置邏輯變更 |
| `services/realtime-gateway/Dockerfile` | 更新檔頭「對齊缺口」段落為「G-01 已裁決 A，compose 已改指本檔案，見 ADR-015」 | 消除過時缺口敘述；建置指令不變（`cmd/realtime-gateway`、8081／healthz） |
| `services/worker/Dockerfile` | 同上，更新檔頭缺口段落為已裁決 | 消除過時缺口敘述；不補 HEALTHCHECK（維持 G-08 現況） |
| `services/api/.dockerignore` | 更新檔頭：刪除「三服務皆以本目錄為 context」敘述，改為僅 api 服務以本目錄為 context | 反映獨立 context 生效後的實際行為 |
| `services/realtime-gateway/.dockerignore` | 更新檔頭：「只在 context 改為本目錄後生效」改為「已生效」 | 同上；排除規則本身不變 |
| `services/worker/.dockerignore` | 同上 | 同上 |
| `Makefile` | 第 79–80 行的 `livekit.yaml`／`prometheus.yml` 檢查提示，改為與 §4 方案一致的預期狀態敘述（prometheus.yml 應存在；livekit.yaml 未建是決議預期，見 SELF_HOSTING.md §9） | 避免靜態檢查對決議預期狀態誤報 WARNING |

### B. 程式註解與服務說明（5 檔）

| 檔案 | 修改內容 | 目的 |
|---|---|---|
| `services/api/internal/transport/router.go` | 「Gin／Echo 二選一尚未決定」註解改為「已決議 Echo v5（≥v5.4.7），見 ADR-016；依賴尚未引入，StdRouter 維持」 | 消除過時未定敘述；介面設計不變 |
| `services/api/cmd/api/main.go` | 「框架決策阻塞」註解改為決議狀態＋ADR-016 指引 | 同上 |
| `services/realtime-gateway/cmd/realtime-gateway/main.go` | 「build context（Dockerfile.gateway）」註解改為獨立 context 已生效＋ADR-015 指引 | 消除過時參照 |
| `services/realtime-gateway/README.md` | 更新 compose 參照段落與框架決策段落為已決議狀態 | 服務說明與現況一致 |
| `services/worker/README.md` | 更新 compose 參照段落為已決議狀態 | 同上 |

### C. 專案狀態與 CI 草稿（2 檔）

| 檔案 | 修改內容 | 目的 |
|---|---|---|
| `README.md` | ① 服務清單行的 Qdrant 改為 pgvector（已定案）；② 缺口段落補 ADR-015／016／017 指引 | 專案狀態紀錄與決議一致 |
| `infrastructure/ci-drafts/github-actions-build.yml` | 第 71 行附近「build 參照（Dockerfile.gateway／Dockerfile.worker）」註解改為獨立 context 參照 | CI 草稿與 G-01 方案 A 一致；仍為草稿，不得以 push 建立 `.github/workflows/` |

### D. Phase 00 文件同步（16 檔；逐檔範圍見附表 D-1，含原始內容與修改後內容對照）

| 檔案 | 修改位置與內容 | 目的 |
|---|---|---|
| `docs/architecture/SYSTEM_ARCHITECTURE.md` | :159「Gin 或 Echo（二選一…）」→「Echo v5（已定案，見 ADR-016）」；:272「向量檢索走 Qdrant／pgvector（二選一待定…）」→「向量檢索走 pgvector（已定案，見 ADR-017）」；:328 缺口 1 註記為已定案；:334「向量庫二選一（Qdrant／pgvector）」→「向量庫（pgvector，已定案，見 ADR-017）」 | 架構文件與兩項選型決議一致 |
| `docs/architecture/MODULE_BOUNDARIES.md` | :361「二選一待定」→ pgvector 已定案；向量歸屬「待 AI 規範文件定歸屬」維持不變 | 同上；歸屬問題不屬本次決議 |
| `docs/architecture/SCALABILITY.md` | :80／:118「二選一待定／未定」→ pgvector 已定案 | 同上 |
| `docs/database/ERD.md` | :11／:886／:925／:939 選項收斂為 pgvector 已定案；:926 移除 Qdrant 列名；向量維度／距離度量／索引方式維持「待 Phase 09 定稿」 | 同上；Phase 09 待定項不預寫 |
| `docs/deployment/CLOUD_ARCHITECTURE.md` | :36 比較表加決議註記（不改比較內文）；:47 容器清單刪 Qdrant、補 pgvector（隨 PostgreSQL）；:103 拓撲圖 Qdrant 方塊改為 pgvector 註記；:122「Qdrant／pgvector（選擇待定）」→ 已定案；:143 `qdrant` 服務行改為 pgvector 行（隨 PostgreSQL、無獨立服務）；:209／:215 狀態更新 | 部署文件與決議一致 |
| `docs/deployment/SELF_HOSTING.md` | :24 啟動順序項刪 qdrant 條件句、補 pgvector 註記；:70 volume 清單刪 `qdrant-data`；§9 第 1–2 項按 §4 方案更新（livekit profiles 保留、prometheus 最小設定已建） | 自架文件與 G-02／G-03 方案一致 |
| `docs/decisions/ADR-009-免費開源替代總決策.md` | :47 表格行加決議註記：「向量庫二選一已於 2026-11-25 決議採用 pgvector，見 ADR-017」（不改表格歷史內文） | 保留決策沿革可追溯 |
| `docs/security/AI_DATA_POLICY.md` | :37／:173／:231「選型待定案」→ pgvector 已定案；權限隔離與刪除同步實作細節維持待定案 | 同上 |
| `docs/PRD.md` | :29「Gin 或 Echo」→「Echo v5（已定案，見 ADR-016）」；:31「Qdrant／pgvector」→「pgvector（已定案，見 ADR-017）」 | 產品文件與決議一致 |
| `docs/FEATURE_LIST.md` | :9／:268「Qdrant／pgvector」→ pgvector 已定案註記 | 同上 |
| `docs/PLATFORM_MATRIX.md` | :11／:156 同上 | 同上 |
| `docs/COST_OPTIMIZATION.md` | :45 第 19 行、:224、:232–236 §3.19、:292 加決議註記（保留 Qdrant／pgvector 比較內文為評估紀錄，註明已選 pgvector 與重估條件）；授權段落不改 | 成本文件保留評估沿革＋決議狀態 |
| `docs/ROADMAP.md` | :10／:114「Qdrant／pgvector（選擇待定）」→ pgvector 已定案 | Roadmap 與決議一致 |
| `SPEC-SOURCE.md` | :23 註記向量替換已定案為 pgvector；:61「Gin 或 Echo」註記已定案為 Echo v5 | 規格來源摘要與決議一致 |
| `docs/ARCHITECTURE_REVIEW.md` | 批准註記追加：四項決議已定（G-01 A、Echo v5、pgvector、G-02／G-03 方案），見 ADR-015／016／017；「Phase 01 最終驗收仍待執行驗證」維持 | 審查狀態紀錄完整 |
| `docs/testing/TEST_STRATEGY.md` | :100 服務清單刪 `qdrant`（18→17 個服務） | 測試環境清單與 compose 一致 |

#### 附表 D-1：Phase 00 文件逐檔修改範圍（修訂版 v2 新增，審查意見 3）

16 檔中，**12 檔含語意變更**（皆為決議直接結果：選項收斂、容器／服務清單、拓撲圖、檢查項、狀態語意），**4 檔僅狀態註記**（ADR-009、COST_OPTIMIZATION、SPEC-SOURCE、ARCHITECTURE_REVIEW）。比較表內文、歷史敘述、Phase 09 待定細節（向量維度、距離度量、索引方式、歸屬、實作細節）一律不動。以下「原始內容」為逐字引文，「修改後內容」為執行時寫入的完整文字（僅改動片段處以「同文，僅……改為……」標示其餘不動）；v3 已逐行實查全部條目原文，D-1.1／D-1.4／D-1.5／D-1.6 的 v2 引文有誤，已按實查原文更正；本附表無「無法取得原文」的條目。

**D-1.1 `docs/architecture/SYSTEM_ARCHITECTURE.md`**（語意變更；v3 按實查原文更正 :272／:334，v2 引文有誤）
- :159 Go API 元件列（表格行，僅改第 2 欄）。原始第 2 欄：「Go、Gin 或 Echo（二選一，Phase 01 定案前不得並存兩套）」。修改後第 2 欄：「Go、Echo v5（≥v5.4.7，已定案，見 ADR-016）」。其餘欄不動。理由：框架決議。
- :272。原始全文：「- 本機模型（Ollama／vLLM／LocalAI）與外部 AI API 共用同一 Adapter 介面，切換模型不得改商業邏輯；向量檢索走 Qdrant／pgvector（二選一待定，不得預設已選），RAG 索引的權限隔離與刪除同步為強制要求。」修改後全文：「- 本機模型（Ollama／vLLM／LocalAI）與外部 AI API 共用同一 Adapter 介面，切換模型不得改商業邏輯；向量檢索走 pgvector（已定案，見 ADR-017），RAG 索引的權限隔離與刪除同步為強制要求。」理由：向量庫決議。
- :328 §10 缺口 1。原始全文：「1. Gin 或 Echo 二選一尚未定案；兩套並存會分裂中介軟體與錯誤處理，Phase 01 前須以 ADR 定案。」修改後全文：「1. Gin 或 Echo 二選一已定案為 Echo v5（見 ADR-016）；本缺口項關閉。」理由：缺口狀態隨決議關閉。
- :334。原始全文：「7. AI Adapter 的介面清單、向量庫二選一（Qdrant／pgvector）、模型資源評估皆未定，屬 AI 規範文件範圍。」修改後全文：「7. AI Adapter 的介面清單、向量庫（pgvector，已定案，見 ADR-017）、模型資源評估皆未定，屬 AI 規範文件範圍。」理由：向量庫決議。

**D-1.2 `docs/architecture/MODULE_BOUNDARIES.md`**（語意變更）
- :361。原始：「向量索引（Qdrant／pgvector，二選一待定）」。修改後：「向量索引（pgvector，已定案，見 ADR-017）」。理由：向量庫決議。「待 AI 規範文件定歸屬」維持不變。

**D-1.3 `docs/architecture/SCALABILITY.md`**（語意變更）
- :80。原始：「Qdrant（PostgreSQL＋pgvector）兩種向量儲存方案的二選一（優先 pgvector）」。修改後：「pgvector（已定案，見 ADR-017）」。理由：向量庫決議。
- :118。原始：「Qdrant 或 pgvector 擇一（未定）」。修改後：「pgvector（已定案，見 ADR-017）」。理由：同上。

**D-1.4 `docs/database/ERD.md`**（語意變更；v3 按實查原文更正，v2 引文有誤）
- :11。原始全文：「- 架構調整（2026-11-25 00:33 使用者指令，優先於舊技術棧）：搜尋投影改 Meilisearch、物件儲存改 MinIO、快取改 Valkey、向量承載為 pgvector 或 Qdrant（選擇待定）。本文件表名與結構不變（§20 為準），僅修搜尋／向量／儲存／快取相關敘述；舊付費服務名不得殘留。」修改後全文：同文，僅「向量承載為 pgvector 或 Qdrant（選擇待定）」改為「向量承載為 pgvector（已定案，見 ADR-017）」。理由：向量庫決議。
- :886。原始全文：「- `updated_at` 自動更新、全文搜尋索引投影（Meilisearch；PostgreSQL 全文搜尋是否併用待定稿）、向量需求（以 pgvector（PostgreSQL 擴充）或 Qdrant 承載，選擇待定，見 §10.4）、分區（Partitioning）等，皆須在 Migration 或配套文件中明定，不得以「後續再說」留空而不標記為待定稿。」修改後全文：同文，僅「向量需求（以 pgvector（PostgreSQL 擴充）或 Qdrant 承載，選擇待定，見 §10.4）」改為「向量需求（以 pgvector（PostgreSQL 擴充）承載，已定案，見 ADR-017；細節見 §10.4）」。理由：同上。
- :925。原始全文：「- 向量需求以 pgvector（PostgreSQL 擴充）或 Qdrant 承載，選擇待定：二選一，定稿時須明定選用者、向量維度與距離度量的來源（模型決定，不得預寫）、索引更新方式、與 Meilisearch／PostgreSQL 的分工。定稿前不得同時假設兩者皆建、不得以向量索引代替全文搜尋索引。向量資料的保留、刪除同步（原資料刪除時向量須同步刪除）須在定稿文件中明定。」修改後全文：「- 向量需求以 pgvector（PostgreSQL 擴充）承載，已定案（見 ADR-017）：定稿時須明定向量維度與距離度量的來源（模型決定，不得預寫）、索引更新方式、與 Meilisearch／PostgreSQL 的分工。定稿前不得以向量索引代替全文搜尋索引。向量資料的保留、刪除同步（原資料刪除時向量須同步刪除）須在定稿文件中明定。」理由：同上（「二選一／選用者／不得同時假設兩者皆建」僅決議前有意義，隨決議刪除）。
- :926。原始全文：「- 本文件不為 Meilisearch、pgvector、Qdrant 預寫額度、容量、延遲等數字，一律待查證／待壓測。」修改後全文：「- 本文件不為 Meilisearch、pgvector 預寫額度、容量、延遲等數字，一律待查證／待壓測。」理由：Qdrant 已非系統組成，列名隨之移除。
- :939。原始全文：「10. 保留期限、計數快取規則、分區／讀寫分離、搜尋索引選用（Meilisearch 投影）與向量承載選擇（pgvector 或 Qdrant），皆待定稿，不得以本文件草案為已定。」修改後全文：「10. 保留期限、計數快取規則、分區／讀寫分離、搜尋索引選用（Meilisearch 投影）與向量承載（pgvector，已定案，見 ADR-017），其餘皆待定稿，不得以本文件草案為已定。」理由：同上。向量維度／距離度量／索引方式維持「待 Phase 09 定稿」，不預寫。

**D-1.5 `docs/deployment/CLOUD_ARCHITECTURE.md`**（語意變更；:36 僅狀態註記；v3 按實查原文更正，v2 引文有誤）
- :36 比較表行。原始全文：「| 向量資料庫付費服務 | Qdrant／pgvector（選擇待定） |」。修改後全文：「| 向量資料庫付費服務 | Qdrant／pgvector（選擇待定；已決議採用 pgvector，見 ADR-017） |」。理由：保留評估沿革＋決議狀態。變更性質：僅狀態註記（比較內文不動）。
- :47 容器清單列。原始全文：「- 容器化：全部服務皆以容器運行（Go 後端、Realtime Gateway、Worker、PostgreSQL、Valkey、NATS、MinIO、Meilisearch、OpenBao、Keycloak、Traefik、Prometheus、Grafana、Loki、GlitchTip、LiveKit CE、Qdrant、Ollama），容器映像版本須固定（不得用浮動標籤），映像來源與簽章驗證待定稿。」修改後全文：同文，僅列舉改為「（Go 後端、Realtime Gateway、Worker、PostgreSQL（含 pgvector 擴充）、Valkey、NATS、MinIO、Meilisearch、OpenBao、Keycloak、Traefik、Prometheus、Grafana、Loki、GlitchTip、LiveKit CE、Ollama）」（刪 Qdrant，PostgreSQL 補 pgvector 註記）。理由：向量庫決議，容器清單隨之變更。
- :102–106 拓撲圖 Qdrant 方塊。原始方塊三行文字：「│  Qdrant   │」「│（向量，   │」「│  待定）   │」。修改後：三行文字分別改為「pgvector」「（隨 PostgreSQL）」「（空白）」，方塊邊框與對齊維持原樣。理由：同上。
- :122。原始全文：「- 資料層：PostgreSQL 為系統紀錄；Valkey 為快取與 Session 儲存；NATS JetStream 為事件與佇列；MinIO 為物件儲存；Meilisearch 為搜尋投影；Qdrant／pgvector 為向量承載（選擇待定，不得同時假設兩者皆為主力）。」修改後全文：同文，僅末句改為「pgvector 為向量承載（已定案，見 ADR-017）。」理由：同上。
- :143 服務表行。原始全文：「| `qdrant` | 向量資料庫（待定，與 pgvector 二選一） | 向量 Volume | 未選定前不得假設已部署；選定後另一選項不得並存為主力。 |」修改後全文：「| `postgresql`（pgvector 擴充） | 向量資料庫（已定案，見 ADR-017） | 隨 `postgres-data` Volume | 無獨立向量服務；向量資料隨 PostgreSQL 備份。 |」理由：服務表隨決議變更（Qdrant 行由 pgvector 行取代）。
- :209。原始全文：「- 特別評估（待 Phase 09 定稿，不得假設）：CPU／RAM／GPU／VRAM 需求（不得假設一般伺服器跑得動所有模型）、模型清單與授權、推理延遲、併發、Token 預算、E2EE 明文不得自動送往雲端 AI（本地模型與外部 API 的邊界待定稿）、向量承載選擇（Qdrant／pgvector，見 ERD.md §10.4）。」修改後全文：同文，僅末段改為「向量承載（pgvector，已定案，見 ADR-017；細節見 ERD.md §10.4）。」理由：狀態語意更新。
- :215。原始全文：「3. 向量承載選擇（Qdrant 或 pgvector）、Meilisearch 與 PostgreSQL 全文搜尋的分工，待定稿。」修改後全文：「3. 向量承載已定案為 pgvector（見 ADR-017）；Meilisearch 與 PostgreSQL 全文搜尋的分工，待定稿。」理由：同上。

**D-1.6 `docs/deployment/SELF_HOSTING.md`**（語意變更；v3 按實查原文更正並補齊 §9 全文，v2 引文有誤）
- :24。原始全文：「1. 資料層先行：`postgresql`、`valkey`、`nats`、`minio`、`meilisearch`（`qdrant` 僅在向量承載選定為 Qdrant 時啟動，否則停用，見 §9）。」修改後全文：「1. 資料層先行：`postgresql`（含 pgvector 擴充）、`valkey`、`nats`、`minio`、`meilisearch`（`qdrant` 服務已移除，見 ADR-017）。」理由：向量庫決議，啟動順序項隨之變更。
- :70。原始全文：「- 命名 Volume 清單見 docker-compose.yml：`postgres-data`、`valkey-data`、`nats-data`、`minio-data`、`meilisearch-data`、`qdrant-data`、`openbao-data`、`ollama-data`、`prometheus-data`、`grafana-data`、`loki-data`、`traefik-letsencrypt`。」修改後全文：同文，僅刪「`qdrant-data`、」。理由：volume 隨 Qdrant 移除而刪除。
- §9 第 1 項。原始全文：「1. `livekit.yaml`（LiveKit CE 配置）：docker-compose.yml 參照 `./livekit.yaml`，檔案尚未建立，待 Phase 07 定稿後建立；建立前 LiveKit 服務無法啟動，不得假裝已配置。」修改後全文：「1. `livekit.yaml`（LiveKit CE 配置）：`livekit` 服務以 `profiles: ["phase07"]` 保留定義，預設不啟動；livekit.yaml 待 Phase 07 定稿後建立接線；建立前 LiveKit 服務不啟動，不得假裝已配置。」理由：G-02 方案（profiles，見 §4）。
- §9 第 2 項。原始全文：「2. `prometheus.yml`（Prometheus 配置）：docker-compose.yml 參照 `./prometheus.yml`，檔案尚未建立，待監控文件定稿後建立；建立前 Prometheus 無法蒐集指標，不得假裝監控已生效。」修改後全文：「2. `prometheus.yml`（Prometheus 配置）：最小有效設定已建立（僅自我抓取）；抓取目標定稿待監控文件定稿後加入；加入前不得假裝監控已覆蓋全部服務。」理由：G-03 方案（見 §4）。
- §9 第 3–8 項：不動。
- 本文件本次修改範圍內無 `pg_extension` 引用；擴充可用性驗證僅見 §6 S2-7（`pg_available_extensions` 查詢）。

**D-1.7 `docs/decisions/ADR-009-免費開源替代總決策.md`**（僅狀態註記）
- :47 表格行。原始全文：「| 向量資料庫 | 向量資料庫付費服務 | Qdrant／pgvector（二選一待定） | AI 規範文件（另冊） |」。修改後全文：「| 向量資料庫 | 向量資料庫付費服務 | Qdrant／pgvector（二選一待定；已於 2026-11-25 決議採用 pgvector，見 ADR-017） | AI 規範文件（另冊） |」。理由：保留決策沿革可追溯，表格歷史內文不動。

**D-1.8 `docs/security/AI_DATA_POLICY.md`**（語意變更）
- :37 架構圖行。原始：「        ├─ 向量檢索：Qdrant／pgvector（二選一或搭配，選型待定案）」。修改後：「        ├─ 向量檢索：pgvector（已定案，見 ADR-017）」。理由：向量庫決議。
- :173 表格行。原始：「| 知識 AI：個人知識庫、文件索引、向量搜尋、RAG、語意搜尋 | 知識庫文件須經使用者授權上傳與索引；索引與向量資料須有權限隔離：使用者僅能檢索自己有權存取的文件；刪除文件時，向量索引須同步刪除；RAG 檢索結果納入 AI 請求前，須經權限與遮罩檢查；向量資料庫用 Qdrant／pgvector，選型待定案 |」。修改後：同行，僅「向量資料庫用 Qdrant／pgvector，選型待定案」改為「向量資料庫用 pgvector（已定案，見 ADR-017）」。理由：同上。
- :231。原始全文：「2. 向量資料庫選型（Qdrant／pgvector）尚未定案；權限隔離與刪除同步的實作細節待定案。」修改後全文：「2. 向量資料庫選型已定案為 pgvector（見 ADR-017）；權限隔離與刪除同步的實作細節待定案。」理由：同上。權限隔離與刪除同步實作細節維持待定案，不預寫。

**D-1.9 `docs/PRD.md`**（語意變更）
- :29。原始全文：「- 後端：Go；Gin 或 Echo；PostgreSQL（自架）；Valkey；WebSocket；NATS JetStream；gRPC 內部通訊；MinIO 物件儲存；Meilisearch；Docker Compose＋Coolify；OpenTofu 基礎設施即程式碼；Traefik 反向代理；Keycloak 身分驗證；OpenBao 憑證管理。」修改後全文：同文，僅「Gin 或 Echo」改為「Echo v5（≥v5.4.7，已定案，見 ADR-016）」。理由：框架決議。
- :31。原始全文：「- AI：Ollama＋vLLM＋LocalAI 優先＋外部 AI API Adapter 可選擴充（AI Provider Adapter，不得讓商業邏輯直接依賴特定供應商 SDK）；向量檢索用 Qdrant／pgvector。」修改後全文：同文，僅「向量檢索用 Qdrant／pgvector」改為「向量檢索用 pgvector（已定案，見 ADR-017）」。理由：向量庫決議。

**D-1.10 `docs/FEATURE_LIST.md`**（語意變更）
- :9。原始全文：「> 架構調整（使用者 2026-11-25 00:33 指令）：部署與 AI 改採免費及開源自架方案——Docker Compose＋Coolify、PostgreSQL 自架、Valkey、MinIO、Meilisearch、OpenBao、Prometheus＋Grafana＋Loki、Cloudflare Free、Coraza WAF、Cloudflare DNS、Let's Encrypt、Keycloak、Traefik、NATS JetStream、GitHub Actions、OpenTofu、LiveKit Community Edition 自架、Ollama＋vLLM＋LocalAI 優先＋外部 AI API Adapter 可選擴充、Qdrant／pgvector、GlitchTip 自架。功能一項不得刪、不得降級。Docker Compose 啟動／連線／上傳／訊息／AI 切換／LiveKit／監控皆為「待 CI 或有 Docker 的環境驗證」，此 VM 無 docker，不得寫已驗證。仍可能付費項目見 `docs/COST_OPTIMIZATION.md`，不得宣稱零成本。」修改後全文：同文，僅「Qdrant／pgvector」改為「pgvector」。理由：向量庫決議（摘要行，決議狀態見 ADR-017）。
- :268。原始全文：「| F-X-07 | AI 功能 | 對話、摘要、翻譯、RAG、客服（手冊 K）；AI 採 Ollama＋vLLM＋LocalAI 優先＋外部 AI API Adapter 可選擴充，可切換模型、Streaming、Tool Calling、上下文管理、Token 統計、資源管理，向量檢索用 Qdrant／pgvector；資料邊界見 AI_DATA_POLICY；E2EE 明文絕對不得送往任何 AI；無 Key 時只做介面並標記未完成 | Phase 09 |」。修改後全文：同文，僅「向量檢索用 Qdrant／pgvector」改為「向量檢索用 pgvector（已定案，見 ADR-017）」。理由：同上。

**D-1.11 `docs/PLATFORM_MATRIX.md`**（語意變更）
- :11。原始全文：「> 架構調整（使用者 2026-11-25 00:33 指令）：部署與 AI 改採免費及開源自架方案——Docker Compose＋Coolify、PostgreSQL 自架、Valkey、MinIO、Meilisearch、OpenBao、Prometheus＋Grafana＋Loki、Cloudflare Free、Coraza WAF、Cloudflare DNS、Let's Encrypt、Keycloak、Traefik、NATS JetStream、GitHub Actions、OpenTofu、LiveKit Community Edition 自架、Ollama＋vLLM＋LocalAI 優先＋外部 AI API Adapter 可選擴充、Qdrant／pgvector、GlitchTip 自架。功能一項不得刪、不得降級。Docker Compose 啟動／連線／上傳／訊息／AI 切換／LiveKit／監控皆為「待 CI 或有 Docker 的環境驗證」，此 VM 無 docker，不得寫已驗證。仍可能付費項目見 `docs/COST_OPTIMIZATION.md`，不得宣稱零成本。」修改後全文：同文，僅「Qdrant／pgvector」改為「pgvector」。理由：向量庫決議（摘要行，決議狀態見 ADR-017）。
- :156。原始全文：「| AI 後端 | 全平台一致：Ollama＋vLLM＋LocalAI 優先＋外部 AI API Adapter 可選擴充，可切換模型、Streaming、Tool Calling；向量檢索用 Qdrant／pgvector；資源需求見 AI_DATA_POLICY，待查證／待實測 | 全平台一致 | 全平台一致 | 全平台一致 | 全平台一致 |」。修改後全文：同文，僅「向量檢索用 Qdrant／pgvector」改為「向量檢索用 pgvector（已定案，見 ADR-017）」。理由：同上。

**D-1.12 `docs/COST_OPTIMIZATION.md`**（僅狀態註記；比較內文與授權段落不改）
- :45 表格行。原始全文：「| 19 | 向量資料庫付費服務 | Qdrant／pgvector | A＋D | 是 | 否（向量規模擴大時須評估記憶體與儲存） | 不適用（自架） | 待查證（伺服器成本） |」。修改後全文：同行，僅第 3 欄改為「Qdrant／pgvector（已決議採用 pgvector，見 ADR-017）」。理由：保留評估沿革＋決議狀態。
- :224。原始全文：「- **對應替代**：Ollama（模型管理與推理）＋vLLM（高吞吐推理）＋LocalAI（相容介面閘道，方向）優先，外部 AI API 經 AI Provider Adapter 可選擴充；向量檢索用 Qdrant／pgvector（見第 3.19 節）。」修改後全文：同文，僅「向量檢索用 Qdrant／pgvector（見第 3.19 節）」改為「向量檢索用 pgvector（已定案，見 ADR-017；評估比較見第 3.19 節）」。理由：同上。
- §3.19（:232–236）。原始標題：「### 3.19 向量資料庫：付費向量服務 → Qdrant／pgvector」。修改後標題：「### 3.19 向量資料庫：付費向量服務 → pgvector（已決議採用 pgvector，見 ADR-017；本節比較內文保留為評估紀錄）」。原始首段：「- **對應替代**：Qdrant（專用向量資料庫）或 pgvector（PostgreSQL 向量擴充），二選一或搭配，選型待定案。」修改後：「- **對應替代**：Qdrant（專用向量資料庫）或 pgvector（PostgreSQL 向量擴充），二選一或搭配；已決議採用 pgvector，見 ADR-017（本節比較內文保留為評估紀錄）。」「功能差異」與「授權條款」兩段不動。理由：同上。
- :292。原始全文：「2. 向量資料庫選型（Qdrant／pgvector）、本地 AI 三者分工、OpenTofu 適用範圍尚未定案。」修改後全文：「2. 向量資料庫選型已定案為 pgvector（見 ADR-017）；本地 AI 三者分工、OpenTofu 適用範圍尚未定案。」理由：同上。

**D-1.13 `docs/ROADMAP.md`**（語意變更）
- :10。原始全文：「- 架構調整（2026-11-25 00:33 使用者指令，優先於舊技術棧）：技術敘述一律改新技術棧（Docker Compose＋Coolify、Traefik、Keycloak、MinIO、Valkey、Meilisearch、OpenBao、NATS JetStream、OpenTofu、GitHub Actions、LiveKit CE、Ollama＋vLLM、Qdrant／pgvector、Prometheus＋Grafana＋Loki＋GlitchTip）。階段順序與驗收不變；舊付費服務名不得殘留。」修改後全文：同文，僅「Qdrant／pgvector」改為「pgvector」。理由：向量庫決議（摘要行，決議狀態見 ADR-017）。
- :114。原始全文：「- 工作項：AI 對話（Streaming、模型切換、上下文、歷史、用量、設定、Tool Calling）；訊息 AI（摘要、群組摘要、翻譯、智慧回覆、潤飾、語氣、語音轉文字、文件摘要、圖片理解）；知識 AI（個人知識庫、文件索引、向量搜尋、RAG、語意搜尋、權限隔離、刪除同步）；商業 AI（官方 AI 客服、FAQ、Bot、分類、摘要、回覆建議）。技術：Ollama＋vLLM＋LocalAI 優先＋外部 AI API Adapter 可選擴充（可切換模型、Streaming、Tool Calling、上下文、Token 統計、資源管理；供應商不得寫死）；向量承載為 Qdrant／pgvector（選擇待定）。安全：授權、Provider Adapter、Rate Limit、Token Budget、敏感遮罩、Prompt Injection 防護、Tool 權限、使用紀錄、保留策略。E2EE 明文不得自動送往雲端 AI。」修改後全文：同文，僅「向量承載為 Qdrant／pgvector（選擇待定）」改為「向量承載為 pgvector（已定案，見 ADR-017）」。理由：同上。

**D-1.14 `SPEC-SOURCE.md`**（僅狀態註記）
- :23。原始：「Ollama＋vLLM＋AI Provider Adapter；向量資料庫付費服務→Qdrant／pgvector；Sentry 託管→」。修改後：「Ollama＋vLLM＋AI Provider Adapter；向量資料庫付費服務→pgvector（已定案，見 ADR-017）；Sentry 託管→」。理由：規格來源摘要與決議一致。
- :61。原始：「- 後端：Go；Gin 或 Echo；PostgreSQL；Redis；WebSocket；NATS JetStream；gRPC 內部通訊；」。修改後：「- 後端：Go；Echo v5（≥v5.4.7，已定案，見 ADR-016）；PostgreSQL；Redis；WebSocket；NATS JetStream；gRPC 內部通訊；」。理由：同上。

**D-1.15 `docs/ARCHITECTURE_REVIEW.md`**（僅狀態註記）
- 檔頭批准註記。原始全文：「> **批准註記：** Phase 00 文件＋品牌文件全數定稿。Gin／Echo、Qdrant／pgvector 選型、部署主機與成本方案、外部憑證申請順序四項仍待決定，不得視為已批准。」修改後：該段不動，另起一段追加：「> **Phase 01 決議註記（2026-11-25）：** 四項決議已定（G-01 方案 A、Echo v5、pgvector、G-02／G-03 方案），見 ADR-015／016／017；「Phase 01 最終驗收仍待執行驗證」維持。」理由：審查狀態紀錄完整，原始批准紀錄不改。

**D-1.16 `docs/testing/TEST_STRATEGY.md`**（語意變更）
- :100。原始全文：「- 測試環境：以 Docker Compose＋Coolify 啟動（見部署文件與 SELF_HOSTING.md），服務含 api、realtime-gateway、worker、postgresql、valkey、nats、minio、meilisearch、openbao、keycloak、traefik、prometheus、grafana、loki、glitchtip、livekit、qdrant、ollama（服務清單見 docker-compose.yml，尚待有 Docker 的環境驗證，不得寫已驗證）。」修改後全文：同文，僅刪「、qdrant」（17 服務）。理由：測試環境清單與 compose 一致。

明確排除：`docs/decisions/ADR-005-舊規格差異對照.md`（歷史對照，不改內文；ADR-016 引用即可）；`docs/brand/BRAND_GUIDELINES.md` :118 的品牌元素「Echo（記憶與回聲）」與框架無關，不得誤改；`docs/phase01/REVIEW.md`（Phase 01 審查歷史紀錄，不改，決議以 ADR 為準）。

### E. Phase 01 紀錄補註（6 檔，皆為文末 dated amendment，不改原文）

| 檔案 | 補註內容 | 目的 |
|---|---|---|
| `docs/phase01/compose-alignment-notes.md` | 增補「2026-11-25 決議修訂」節：G-01 已裁決 A、qdrant 服務已移除（17 服務）、G-02 按 §4 處理、G-04 清單減 qdrant 一項、其餘 G-03～G-08 維持追蹤 | 對照文件與現況一致，保留原始核對紀錄 |
| `docs/phase01/go-notes.md` | 增補決議修訂節：G-01、框架、向量三項已決議＋ADR 指引 | 同上 |
| `docs/phase01/infra-notes.md` | 增補決議修訂節：同上＋G-02 處理 | 同上 |
| `docs/phase01/gin-vs-echo-memo.md` | 檔頭增補 3 行決議註記：已決議 Echo v5（≥v5.4.7），見 ADR-016；比較內文保留為證據 | 備忘錄狀態不滯留「待決定」 |
| `docs/phase01/vector-store-memo.md` | 檔頭增補 3 行決議註記：已決議 pgvector，見 ADR-017；比較內文保留為證據 | 同上 |
| `docs/phase01/observability-wiring.md` | :54 prometheus 行「配置檔 `./prometheus.yml` 尚未建立（compose 參照，見 SELF_HOSTING.md §9），建立前無法蒐集指標」改為「配置檔 `./prometheus.yml` 已建立最小有效設定（僅自我抓取，見 SELF_HOSTING.md §9），其餘抓取目標待定稿」；:90–91 檢查項「`prometheus.yml` 建立與抓取目標定稿（含 API／WS／NATS／Worker／LiveKit／AI 指標，見 ADR-012 指標清單）」改為兩項：「`prometheus.yml` 最小有效設定已建立（僅自我抓取）」已完成、「抓取目標定稿（含 API／WS／NATS／Worker／LiveKit／AI 指標，見 ADR-012 指標清單）」仍待定稿 | 可觀測性接線紀錄與 §4 方案一致（v3 方案 A 正式納入，見審查意見 1） |

### F. 新增檔案（4 檔）

| 檔案 | 內容 | 目的 |
|---|---|---|
| `docs/decisions/ADR-015-G-01獨立Dockerfile與建置context.md` | 見 §3 大綱 | G-01 正式決策紀錄 |
| `docs/decisions/ADR-016-採用Echo-v5.md` | 見 §3 大綱 | 框架正式決策紀錄 |
| `docs/decisions/ADR-017-採用pgvector.md` | 見 §3 大綱 | 向量庫正式決策紀錄 |
| `infrastructure/docker/prometheus.yml` | 見 §4：最小有效設定（僅自我抓取） | 消除指向不存在檔案的掛載 |

## 2. 逐檔修改目的補述

- **Compose 是本次核心**：G-01 A、Qdrant 移除、pgvector 映像更換、G-02／G-03 掛載處理四件事全部落在同一個檔案，須一次改完、一次驗證，避免分次修改造成中間態不一致。`api` 服務的 build 參照本已正確（context `../../services/api`、dockerfile `Dockerfile`），本次不動。
- **Dockerfile 與 .dockerignore 只改註解**：三份 Dockerfile 的建置指令（`golang:1.25.0-alpine3.21`、`cmd/*`、healthcheck）皆不變；三份 `.dockerignore` 的排除規則不變，只改「context 生效狀態」敘述。獨立 context 生效後，每份 `.dockerignore` 只管自己的 context，COPY 集合以各服務目錄為準（見 §7 R4）。
- **文件同步範圍以 §1-D 附表 D-1 為準**：12 檔含決議直接造成的語意變更（選項收斂、容器／服務清單、拓撲圖、檢查項、狀態語意），4 檔僅狀態註記；比較表、歷史敘述、待定細節（維度、歸屬、實作細節）一律不動，避免以決議之名重寫 Phase 00 規格。

## 3. 三項 ADR 內容（ADR-015 為大綱；ADR-016／ADR-017 為完整內容，修訂版 v4）

### ADR-015：G-01 採用獨立 Dockerfile 與獨立建置 context

- **背景**：Phase 00 的 compose 以 `services/api` 為共用 context，參照不存在的 `Dockerfile.gateway`／`Dockerfile.worker`（G-01）；Phase 01 已建三份獨立 Dockerfile，兩套佈局並存即 drift 來源。
- **決議**：採用方案 A——`api`、`realtime-gateway`、`worker` 各自以自己的服務目錄為 context、各用目錄內的 `Dockerfile`；刪除共用 context 參照。
- **理由**：與三個獨立 Go module（`ether.local/*`）的佈局一致；各服務的 `.dockerignore` 各自生效，COPY 集合清晰；建置快取互不污染；與 MODULE_BOUNDARIES.md 的模組邊界一致。
- **替代方案（B）與代價**：維持單一 `services/api` context，補 `Dockerfile.gateway`／`Dockerfile.worker`——須撤回或降級已建的獨立目錄 Dockerfile，獨立 module 與共用 context 佈局長期不一致，未來拆分成本更高。未採納。
- **影響範圍**：compose 三個服務的 build 段、三份 Dockerfile 檔頭、三份 `.dockerignore` 檔頭、Makefile 提示、CI 草稿註解、對照文件補註。
- **重估條件**：未來若三服務合併為單一部署單元（架構變更），須以新 ADR 重估；僅有效能或快取理由不得自行改回共用 context。
- **執行檢查點**：`docker compose config` 渲染三服務 build 段正確；三個映像各自建置成功；grep 零 `Dockerfile.gateway`／`Dockerfile.worker` 引用（歷史紀錄文件除外，見 §6）。

### ADR-016：採用 Echo v5（完整內容，修訂版 v4）

- **狀態**：已定案（2026-11-25 審查決議）。本節為 ADR-016 的完整內容，執行時逐字寫入 `docs/decisions/ADR-016-採用Echo-v5.md`。備忘錄 `docs/phase01/gin-vs-echo-memo.md` 保留為決策紀錄；其中與本 ADR 牴觸的技術描述（`Context` 型別、WebSocket 責任）以本 ADR 為準。
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

### ADR-017：採用 pgvector（完整內容，修訂版 v4）

- **狀態**：已定案（2026-11-25 審查決議）。本節為 ADR-017 的完整內容，執行時逐字寫入 `docs/decisions/ADR-017-採用pgvector.md`。備忘錄 `docs/phase01/vector-store-memo.md` 保留為決策紀錄；其中關於安全性結果的描述以本 ADR 為準，不得將採用 pgvector 描述成安全性結果自動成立。
- **背景**：Phase 00 要求向量承載二選一；compose 內含未釘選標籤的 `qdrant` 服務（僅佔位，選擇待定）；`postgres:16` 官方映像未含 pgvector 擴充。
- **決議**：採用 pgvector，以 PostgreSQL＋pgvector 為初期向量儲存方案；移除 compose 的 `qdrant` 服務與 `qdrant-data` volume；清理 Qdrant 環境變數與設定引用；保留 AI Provider Adapter 後方的向量存取抽象介面；業務模組不得直接綁定特定向量資料庫。**採用 pgvector 不得降低 `AI_DATA_POLICY.md` 的安全要求**；權限隔離、刪除同步、撤銷後不可見的證明責任見「安全性要求保留」。保留 pgvector 選型，不得自行改選 Qdrant。
- **理由（按備忘錄權重，已按 v4 審查意見調整用語；v3 的過度保證描述作廢）**：① 交易一致性條件較佳——PostgreSQL 與 pgvector 位於同一資料庫，關聯資料與向量可在同一交易內操作，**有利於**交易一致性及資料管理，減少跨系統協調；此為有利條件，不是自動保證，安全性結果仍須實現與驗證（見下）；② 自架棧最簡——不新增狀態服務、備份 runbook 與監控面，沿用 PostgreSQL 備份與監控；③ Go 接入邊際成本最低——沿用 `pgx`＋`pgvector-go`，無新協定；④ 規模證據缺席——無 Phase 00 證據顯示需要專用引擎，不得以未證實規模引入。
- **安全性要求保留（修訂版 v4 新增）**：權限隔離、刪除同步、撤銷後不可見仍是**必須實現與驗證的設計要求**，不是採用 pgvector 後即成立的結果。「向量與權限事實同一交易」「刪除即消失」「撤銷即不可見」「無同步窗口」等表述不得寫成採用後的既成結果。上述安全性結果必須由 Phase 09 的資料模型、交易邊界、查詢授權條件、刪除流程及測試證明；證明前不得宣稱已達成。`AI_DATA_POLICY.md` 的安全要求維持不變，不得因採用 pgvector 而降低。
- **替代方案（Qdrant）與代價**：專用引擎的檢索表達力（payload 過濾、稀疏＋稠密混合、量化）與獨立擴充；代價為獨立服務的部署／備份／監控負擔、Outbox／NATS 投影同步與刪除對帳、查詢層的 PostgreSQL 權限再過濾、同步延遲窗口的安全論證。未採納，備忘錄保留為決策紀錄。
- **版本與映像固定策略（修訂版 v2，審查意見 1）**：備忘錄查證 pgvector 0.8.7（2026-10-01，修正 CVE-2026-103484 IVFFlat）。PostgreSQL 16 固定採用官方映像 **`pgvector/pgvector:0.8.7-pg16-bookworm@sha256:484e838076e04939d27d3e176e91c86eea96236126023be8d3059761122e579f`**——標籤＋digest 雙固定，浮動標籤禁用。查證：Docker Hub 標籤 `0.8.7-pg16-bookworm` 存在（last pushed 2026-11-12；amd64 digest `sha256:018b4a92…0e7957`、arm64 `sha256:d153dea9…e5a3853`）；manifest list digest `sha256:484e8380…122e579f` 已於 2026-11-25 向 Docker Hub registry 查證。執行時須以 `docker buildx imagetools inspect` 覆核 digest，不符即停下並回報（見 §7 R9、§9）。映像與 PostgreSQL 16 相容（官方 `pg16` 標籤命名）；bookworm 基底與現行 `postgres:16` 同系。更換前檢查與回退方案見 §9。
- **影響範圍**：compose（服務移除、volume 移除、PostgreSQL 映像更換）、`.env.example`、部署／自架／成本／ERD／AI 規範等 16 處文件註記、對照文件補註。安全性結果的證明屬 Phase 09（見「安全性要求保留」），不屬本次驗證範圍，本次不得宣稱已證明。
- **重估條件（升級觸發，皆須 Phase 09 實測證據）**：HNSW 記憶體佔用影響 OLTP、查詢延遲實測不達標、或出現 pgvector 無法表達的檢索需求；切換時只須替換 Adapter 實作＋匯出／重建向量（維度與距離度量須一致），不得改動商業邏輯。E2EE 明文禁入向量索引的規則不變，預設不得索引。
- **執行檢查點**：`docker compose config` 渲染無 `qdrant` 服務與 volume；PostgreSQL 映像按 digest 拉取成功且擴充可用性驗證通過（§6 S2-7）；grep 證明程式碼與設定檔零 Qdrant 引用（歷史／決策紀錄除外）；Phase 09 驗收須含權限隔離／刪除同步測試證據（非本次範圍，僅列為後續驗收條件）。

## 4. G-02／G-03 處理方案

現況（已核對）：`livekit` 服務以唯讀掛載參照不存在的 `./livekit.yaml`（compose 第 311 行）並以 `--config /etc/livekit.yaml` 啟動；`prometheus` 以唯讀掛載參照不存在的 `./prometheus.yml`（第 354 行）並以 `--config.file=/etc/prometheus/prometheus.yml` 啟動。掛載來源不存在時，兩服務皆無法正常啟動。

### 推薦方案（提請批准）

| 項目 | 處理 | 理由與界線 |
|---|---|---|
| `prometheus.yml` | **補最小有效設定**（新檔，見 §1 F）：`global.scrape_interval: 15s`＋單一 `prometheus` 自我抓取 job；其餘目標（api／gateway／worker／NATS／LiveKit 等）以註解列出，標明待 Phase 05／11 定稿後加入，不得假裝監控已生效 | Prometheus 以自我抓取即可乾淨啟動，設定檔語法可在 CI 以 `promtool check config` 靜態驗證（環境具備時）；不產生「監控已覆蓋全部服務」的假象 |
| `livekit.yaml` | **加入 `profiles: ["phase07"]`**：刪除 `livekit` 服務的 volume 掛載與 `--config` 命令，服務定義保留供 Phase 07 後續接線；一般 `docker compose up -d` 不啟動 LiveKit；**不得為了讓容器啟動而擅自加入開發金鑰或啟用不安全模式** | LiveKit 官方程式碼顯示預設 Development 模式未啟用，若沒有 `LIVEKIT_KEYS` 或 key file 會回傳 `ErrKeysNotSet`，不能假設移除設定檔後就能正常啟動；profile 隔離避免未配置服務混入預設啟動流程，Phase 07 接線時只需啟用 profile＋補檔，不須改服務定義。驗證方式見 §6（版本記錄、雙 config 檢查、實際容器狀態判定）。 |

### 替代方案（未推薦、未採納；審查已定 profiles 方案，僅供紀錄）

- **兩者皆補最小設定**：為 LiveKit 補 `port: 7880` 等最小 `livekit.yaml`。風險：未經 TURN 評估的組態可能讓服務看似啟動、實則通話不可用，且日後 Phase 07 須先推翻本次設定；未採納（審查已定 profiles 方案，見推薦方案）。
- **兩者皆移除引用**：Prometheus 亦移除掛載與 `--config.file`，改用映像內建預設組態。風險：組態意圖不留在 repo 內，日後監控定稿時須從零重建；不推薦。

### 預設行為確認（修訂版 v3 改寫，審查意見 §一-2）

LiveKit 以 `profiles: ["phase07"]` 保留定義後，執行時分兩種流程確認，不得混淆：

1. **預設流程**：一般 `docker compose up -d` 啟動後，以 `docker compose ps` 與啟動日誌確認 `livekit` **未啟動**（S3-9）；預設啟動服務數為 16，定義服務數為 17。隔離成敗以實際容器狀態判定，不以 `docker compose config` 渲染輸出判定（渲染行為因 Compose 版本而異，見 §6 S1-2）；執行時須記錄實際 Docker Compose 版本。
2. **Profile 流程（僅驗證觀察，不啟用功能）**：以 `docker compose --profile phase07 up -d livekit` 單獨啟動，記錄映像預設 entrypoint／command、無 `LIVEKIT_KEYS` 或 key file 時的 `ErrKeysNotSet` 行為、healthcheck 結果。**不得加入開發金鑰**，不得為讓容器維持啟動而啟用不安全模式；記錄到的啟動失敗（如 `ErrKeysNotSet`）即為預期觀察結果，不得掩飾為成功。

以上觀察結果寫入驗證報告（§6 S3-11）。**任何流程下皆不得宣稱 LiveKit 語音或視訊功能已可用**；通話可用性屬 Phase 07 接線範圍。

### 記錄更新（隨方案一併執行）

- `SELF_HOSTING.md` §9 第 1–2 項改為：livekit.yaml 未建是決議預期（`livekit` 以 `profiles: ["phase07"]` 保留定義、預設不啟動，Phase 07 接線）、prometheus.yml 已建最小設定（自我抓取 only）。
- `Makefile` 第 79–80 行檢查提示改為與上一致。
- `docs/phase01/observability-wiring.md` 的 prometheus 行（配置檔未建）改為「最小設定已建，僅自我抓取」。
- `docs/phase01/compose-alignment-notes.md` G-02 節增補決議修訂（不改原文）。
- `README.md` 缺口段落維持「livekit.yaml 未建」敘述為「livekit.yaml 未建是預期，見 §4」。

## 5. Compose 修正後的預期服務拓撲

服務總數：Compose **定義 17 個服務**（18 減 `qdrant`）；**預設啟動 16 個服務**（`livekit` 以 `profiles: ["phase07"]` 保留定義，預設不啟動；預設啟動數以實際容器狀態為準，config 渲染結果因 Compose 版本而異，不得僅以 config 輸出判定，見 §6）。其餘服務的定義、網路、依賴關係皆不變（除 §1 所列的 build 段、PostgreSQL 映像、livekit 掛載＋profiles、prometheus 掛載）。

| 服務 | 形態 | 備註 |
|---|---|---|
| `traefik` | `traefik:v3` | 不變；G-05 ping 待驗證維持 |
| `api` | build：`services/api`＋`Dockerfile` | 不變（本已正確）；8080／healthz |
| `realtime-gateway` | build：`services/realtime-gateway`＋`Dockerfile` | **G-01 A 生效**；8081／healthz |
| `worker` | build：`services/worker`＋`Dockerfile` | **G-01 A 生效**；無 healthcheck（G-08 維持） |
| `postgresql` | 固定映像 `pgvector/pgvector:0.8.7-pg16-bookworm@sha256:484e838076e04939d27d3e176e91c86eea96236126023be8d3059761122e579f` | **映像更換（標籤＋digest 雙固定）**；`CREATE EXTENSION` 屬 Phase 09，本次不執行；更換前檢查與回退見 §9 |
| `valkey`、`nats`、`minio`、`meilisearch` | 不變 | 不變 |
| `openbao`、`keycloak` | 不變 | 開發模式佔位維持（G-03 追蹤） |
| `livekit` | `livekit/livekit-server`，無 config 掛載，含 `profiles: ["phase07"]` | **定義保留、預設不啟動**，供 Phase 07 接線，見 §4 |
| `ollama` | 不變 | vLLM 未預寫維持（G-06 追蹤） |
| `prometheus` | 掛載新建的最小 `prometheus.yml` | **僅自我抓取**，見 §4 |
| `grafana`、`loki`、`glitchtip` | 不變 | 依賴 prometheus／loki 健康狀態不變 |

網路：`ether-front`（僅 traefik）與 `ether-back`（`internal: true`，定義上含其餘 16 個服務；預設啟動時實際連接 15 個容器，`livekit` 未啟動）不變。Volumes：12 → **11**（刪 `qdrant-data`）。Traefik 路由標籤（api／ws／auth／grafana）不變。對外埠：僅 traefik（80／443）；livekit 埠定義保留（7880／7881／7882-udp，預設不啟動，RTC 範圍待 Phase 07 評估）。

## 6. 後續 CI／Docker 執行驗證清單

全部項目皆為「待 CI 或有 Docker 的環境驗證」，本機 VM 無工具鏈，本次不執行、不預寫結果。通過標準須逐項勾稽，未通過項不得以靜態檢查代替。

### 檢查分類與工具鏈限制界線（修訂版 v4 新增，審查意見 §一-4）

以下四類界線適用全部檢查項與 §9.1／§10；計畫敘述不得代替執行證據，未執行項一律標示「未驗證」並註明受阻的操作範圍。

| 類別 | 定義 | 包含項目 | 執行條件 |
|---|---|---|---|
| A. Docker 操作前必須完成的安全檢查 | 在任何 Docker 操作（拉取、啟動）之前必須完成並記錄，否則不得執行可能影響資料的操作；不阻塞 C 類靜態工作（見 §10） | §9.1 第 1–5 項（volume 狀態、`POSTGRES_*` 比對、連線設定、digest 覆核、下游清單）；S2-7 的 digest 覆核部分 | 需 Docker／Buildx 環境；環境缺失時標示未驗證，Docker 操作全部受阻 |
| B. 需要 Docker／Buildx 才能完成的檢查 | 只能在具備 Docker／Buildx 的環境執行 | S1-2、S1-4（經映像執行 promtool）、S2、S3、S4 | 需 Docker／Buildx 環境；環境缺失時標示未驗證，不得以 YAML 解析或計畫敘述代替 |
| C. 本次可執行的靜態修改與驗證 | 不需 Docker 即可執行的檔案修改與靜態檢查 | §1 全部檔案修改（計畫批准後）、S1-1、S1-3、S1-5、ADR 撰寫 | 計畫批准後即可執行；本機 VM 可執行 S1-1／S1-3／S1-5 |
| D. 必須等待 CI／Docker 環境才能完成的項目 | 執行驗證與最終驗收 | B 類全部、S5 驗收報告的執行證據部分 | 須在完整工具鏈的 CI／Docker 環境完成；完成前 Phase 01 不得標記最終驗收完成 |

界線規則：
- **Volume 未確認時不得假設**：PostgreSQL volume 狀態未確認前，不得假設沒有資料，也不得執行可能影響資料的操作（見 §9.1）。
- **Digest 規則**：映像 digest 無法覆核或與固定值不符時，必須停止映像更換，不得自行換標籤、換基底或採用其他映像（見 §7 R9、§9.1）。
- **未驗證標示**：若目前環境無法完成必要檢查，該項標示「未驗證」，並在驗收報告中說明受阻的操作範圍（見 S5-15）；不得以計畫敘述、靜態檢查或渲染輸出代替執行證據。
- **Config 渲染≠啟動行為**：`docker compose config` 的渲染結果與實際容器啟動行為分開記錄（見 S1-2、S3-9）；隔離成敗以實際容器狀態判定。

### S1 靜態驗證（本機可先跑，計畫核准後、修改完成時執行）

1. `make verify-static` PASS（YAML 解析 17 服務、Dockerfile 指令、引用一致性、secrets 掃描），警告清單須不再含 G-01 與 qdrant 項，僅餘已知追蹤項（G-03～G-08）。
2. 設定渲染檢查（**B 類，必須項；設定渲染與實際啟動行為分開記錄**；需 Docker 環境時執行，否則記未驗證）：① 記錄實際 Docker Compose 版本（`docker compose version` 輸出全文）；② 分別執行一般 `docker compose config` 與 `docker compose --profile phase07 config`，兩份輸出分別記錄；③ 一般 config 應以三應用服務的 build 段如 §5、`qdrant` 服務與 `qdrant-data` volume 不存在、`prometheus` 掛載存在且指向新建檔案、`postgresql` 的 image 欄位為 §9 固定映像字串（含 digest）為檢查點；`--profile phase07 config` 應含 `livekit` 且其無 `./livekit.yaml` 掛載、含 `profiles: ["phase07"]`；④ **若一般 config 輸出仍包含 LiveKit，不得直接認定 profile 隔離失敗**（渲染行為因 Compose 版本而異）；profile 隔離成敗**只以 S3-9 的實際容器狀態判定**。
3. grep 勾稽：程式碼與設定檔（compose、Dockerfile、`.dockerignore`、`.env.example`、Makefile、CI 草稿、Go 檔）零 `qdrant` 字串、零 `QDRANT_` 變數、零 `Dockerfile.gateway`／`Dockerfile.worker` 引用；歷史／決策紀錄文件（備忘錄、ADR-009 比較表、COST 比較節、REVIEW.md）內的 `qdrant` 僅出現於「已移除／已決議」註記語境，逐處核對。
4. `promtool check config infrastructure/docker/prometheus.yml` 通過（**必須項**；本機無 promtool 時以 `docker run --rm -v $PWD/infrastructure/docker:/etc/prometheus prom/prometheus promtool check config /etc/prometheus/prometheus.yml` 執行；兩者皆不可行時記未驗證，不得宣稱已驗）。
5. Secrets 掃描：全部修改檔零真實憑證樣式命中；`.env.example` 全佔位符。

### S2 建置驗證（需 Docker＋Go 工具鏈）

6. 三個 Docker Build Context **分別驗證**：`services/api`、`services/realtime-gateway`、`services/worker` 各自 `docker build` 成功，建置日誌的 context 路徑與輸出二進位名稱（`api`／`realtime-gateway`／`worker`）逐一核對；COPY 集合不含 `.env`、憑證與測試檔；並以負向檢查確認 gateway／worker 映像不含 api 服務檔案（context 隔離證據）。
7. PostgreSQL 映像按 digest 拉取成功（`docker buildx imagetools inspect` 覆核 digest 與 §9 固定值一致，不符即停下）；以獨立一次性容器驗證 pgvector 擴充**可用但未安裝**。完整程序如下（任一命令退出碼非 0 即停止並回報）：

① 拉取固定映像（須先通過 §9.1 digest 覆核；完整 digest，不得省略）：

```sh
docker pull pgvector/pgvector:0.8.7-pg16-bookworm@sha256:484e838076e04939d27d3e176e91c86eea96236126023be8d3059761122e579f
```

② 建立獨立一次性容器：tmpfs 資料目錄（若環境不支援 tmpfs，改用等效暫存資料目錄並記錄）、不掛載 ETHER 專案的 PostgreSQL 持久化 volume、不發布主機埠、不連接正式資料庫（`--network none`，僅容器內部通訊；`POSTGRES_PASSWORD` 僅為本次探測用的暫時值，非真實憑證）：

```sh
docker run --rm -d --name ether-pgvector-probe --network none \
  --tmpfs /var/lib/postgresql/data:rw,noexec,nosuid,size=256m \
  -e POSTGRES_PASSWORD=probe-only-ephemeral -e POSTGRES_DB=probe \
  pgvector/pgvector:0.8.7-pg16-bookworm@sha256:484e838076e04939d27d3e176e91c86eea96236126023be8d3059761122e579f
```

③ 等待 PostgreSQL 就緒（容器內 `pg_isready`，不經主機埠；限時等待，逾時即停止並回報）：

```sh
ready=0; for i in $(seq 1 30); do if docker exec ether-pgvector-probe pg_isready -U postgres; then ready=1; break; fi; sleep 2; done; test "$ready" = 1
```

④ 連線方式：容器內 `docker exec` 執行 `psql`（Unix socket，不經主機埠、不連正式資料庫），執行原定查詢：

```sh
docker exec ether-pgvector-probe psql -U postgres -d probe -c "SELECT name, default_version, installed_version FROM pg_available_extensions WHERE name = 'vector';"
```

查詢原文：

```sql
SELECT name, default_version, installed_version
FROM pg_available_extensions
WHERE name = 'vector';
```

⑤ 驗證標準：查得到 `vector` 一行；`default_version` 有值；`installed_version` 為 NULL。不執行 `CREATE EXTENSION`，不執行 migration，不寫入產品資料。

⑥ 完成後清除一次性容器，確認沒有留下持久化資料：

```sh
docker stop ether-pgvector-probe
docker ps -a --filter name=ether-pgvector-probe
docker volume ls
```

清除確認標準：`docker ps -a` 無該容器（`--rm` 已自動刪除）；`docker volume ls` 無因本次驗證新增的 volume；tmpfs 隨容器刪除而消失。

證據須附：上述每條命令、退出碼、實際輸出（含查詢結果全文）。
8. Echo 版本覆核（依賴引入批次的前置檢查，不屬本次修改）：官方 release 頁確認最新穩定 patch ≥ v5.4.7；pgvector 版本覆核：官方穩定版本與安全修正（含 CVE-2026-103484 修正狀態）已核對並記錄。證據須附 Echo 與 pgvector 官方版本資訊、官方來源（URL）及查核日期；CVE-2026-103484 須確認官方公告與實際適用性（影響版本、修正版本、本專案是否受影響），不得只列 CVE 編號便視為完成。

### S3 啟動與健康驗證（需 Docker 環境）

9. `docker compose up -d`（預設，不帶 profile）基礎設施組啟動成功；以 `docker compose ps` 的實際容器狀態確認 `livekit` 未啟動（`profiles: ["phase07"]`，見 §4；**隔離成敗以此實際狀態判定，不以 config 渲染輸出判定**），並記錄本次 Docker Compose 版本（與 S1-2 同一版本輸出）；除 worker（無 healthcheck，G-08）外，全部預設啟動服務 healthcheck 為 healthy。
10. `prometheus` 自我抓取 `up{job="prometheus"} == 1`；Grafana 與 Loki 健康端點回 200。
11. `livekit` 負向驗證標準（本次修訂，2026-11-25 審查決定）：分兩種流程，皆附本次 Docker Compose 版本記錄。預設流程：`docker compose up -d`（不帶 profile）後，以 `docker compose ps` 的實際容器狀態確認 `livekit` **不得啟動**（隔離成敗以此實際狀態判定，不以 config 渲染輸出判定）；若預設流程中 livekit 已啟動，即為未通過。Profile 流程：`docker compose --profile phase07 up -d livekit`，記錄容器狀態、日誌與 healthcheck 結果、預設 entrypoint／command；無 keys 時預期觀察 `ErrKeysNotSet`，**預期的無 keys 錯誤行為不應直接判定為驗證失敗**（須與非預期的啟動失敗區分，並記錄日誌全文）；不得加入開發金鑰；任何流程下皆不得宣稱通話功能可用。測試後恢復操作前的服務狀態（見 §9.2 回退前狀態證據與恢復：原本停止的服務不得擅自啟動，不得以無參數 `docker compose up -d` 啟動全部服務）。

### S4 連線驗證（需 Docker 環境）

12. `api` 的 `GET /healthz` 與 `realtime-gateway` 的 `GET /healthz` 回 200（經 compose 網路與 Traefik 路由各驗一次，Traefik 憑證為 Let's Encrypt 測試路徑時以內網驗證為準）。
13. PostgreSQL／Valkey／NATS／MinIO／Meilisearch／Keycloak 連線成功（各服務 healthcheck＋應用服務環境變數組裝值逐名核對）。
14. Keycloak 與 GlitchTip 以更換後的 PostgreSQL 映像正常啟動並能使用資料庫：Keycloak（`start-dev` 佔位模式）管理端點健康檢查通過；GlitchTip 服務健康檢查通過且能連線資料庫（`DATABASE_URL` 與 `REDIS_URL` 組裝值逐名核對）——確認映像更換未破壞共用資料庫的下游服務。

### S5 驗收文件

15. 提交完整差異報告（`git diff` 形式或等價檔案清單，含 §1 全部檔案）與本清單逐項結果；每項未驗證者須註明受阻的操作範圍（例如「Docker 操作全部受阻」「映像更換受阻」「啟動驗證受阻」）與所屬檢查類別（見 §6 分類表）；未通過項列為未完成，不得標 Phase 01 最終驗收完成；等待下一次審查。

### 結果分類規則（修訂版 v2 新增，審查意見 §二-9）

驗證報告中，每個檢查項必須標示且僅標示以下三類之一，不得混用：**已驗證**（在具備工具鏈的環境實際執行且通過，附命令與輸出證據）、**未驗證**（因環境缺失未能執行，或執行未通過）、**不適用**（該項不適用於本次範圍，須註明理由）。「未驗證」項即阻塞 Phase 01 最終驗收；「不適用」不得用於掩蓋未執行的必須項。

## 7. 可能產生的新風險

| 編號 | 風險 | 說明與應對 |
|---|---|---|
| R1 | Echo 官方最新版本或相容性問題 | v5.4.7 發布紀錄已查證（2026-11-20，Go proxy，見 §3 ADR-016）；殘餘風險為引入時官方最新穩定版高於 v5.4.7 且出現相容問題。應對：引入前以官方 release 頁重查；**版本下限不得低於 v5.4.7，不得自行降低**；如官方資訊或相容性出現問題，**停止並提交評估**，不得自行改 floor（ADR-016 已載明） |
| R2 | pgvector 映像更換與資料目錄相容性 | 映像已固定為標籤＋digest（見 §9），殘餘風險為更換後 Keycloak／GlitchTip 共用資料庫行為差異、或既有資料卷狀態與預期不符。應對：執行前按 §9 逐項檢查（volume 狀態、`POSTGRES_*` 比對、連線設定不變、digest 覆核）；compose 從未啟動、預期無既有資料，但仍須驗證，不得假設相容；備妥 §9 回退方案，觸發條件成立即回退 |
| R3 | LiveKit 無金鑰時無法啟動（`ErrKeysNotSet`） | 移除掛載與 `--config` 後，服務行為取決於映像預設與金鑰狀態；無 `LIVEKIT_KEYS` 或 key file 時回傳 `ErrKeysNotSet`，不能假設可正常啟動。應對：§4 profiles 方案——預設流程不啟動 LiveKit，profile 流程僅記錄預設行為與金鑰要求，不得加入開發金鑰；S3-11 分兩種流程記錄，不得寫成可用；config 渲染行為因 Compose 版本而異，隔離成敗只以實際容器狀態判定（見 §6 S1-2／S3-9）；Phase 07 接線時重估 |
| R4 | .dockerignore 行為變化 | 各 context 獨立生效後，COPY 集合以各目錄為準；若某服務誤排必要檔案，建置期才會暴露。應對：S2-6 逐映像驗證；三份排除規則本次不變，風險可驗證、可回退 |
| R5 | 文件同步漂移 | 16 處 Phase 00 註記＋5 處 Phase 01 補註，漏一處即不一致。應對：§6 S1-3 grep 清單逐項勾稽；ADR 為準，註記只引 ADR、不複述決議理由 |
| R6 | G-04～G-08 仍在，compose 仍非最終可用態 | 浮動標籤、traefik ping、Coraza／vLLM／OTel 缺席、§26 位置差異皆未動。應對：本次不擴大範圍；驗收報告須明列「compose 仍含已知追蹤缺口」，不得宣稱部署就緒 |
| R7 | 「有條件接受」被誤讀為最終驗收 | 靜態 PASS 與執行驗證的區分若在轉述中丟失，後續階段會建立在未驗證基礎上。應對：全部對外敘述（報告、README、ADR）一律寫「骨架定稿、執行驗證待補」，禁用「測試通過」「驗證完成」等措辭 |
| R8 | 決議執行中新發現架構問題 | 例如 pgvector 映像無合適官方標籤、或 Echo 官方資訊與備忘錄衝突。應對：決議 §七——先提具體方案、等批准，不得自行擴大範圍或改規格 |
| R9 | 映像 digest 覆核不符或 registry 無法存取 | 固定 digest 是本次的版本錨點；覆核不符表示標籤被重推或查證有誤，拉取未固定映像將破壞可重現性。應對：**停止，不得拉取、不得改用其他標籤（含 trixie 變體）**；重新查證官方映像資訊並回報審查者，等候指示 |

## 8. Qdrant 全專案引用盤點（修訂版 v2 新增，審查意見 4）

盤點方法：2026-11-25 全樹 `grep -rni "qdrant"`（含程式碼、設定、文件、範例）＋逐檔核對 Go 設定結構與環境變數讀取程式。結論：**現行執行設定中的 Qdrant 引用僅 3 處**（compose 服務＋volume、`.env.example` 唯一 `QDRANT_*` 變數）；Go 程式碼、測試設定、範例中零引用。清理範圍覆蓋全部現行執行設定；歷史比較文件與 ADR 評估內容保留。

| 類別 | 位置 | 內容 | 處理 |
|---|---|---|---|
| 執行設定：Compose 服務 | `infrastructure/docker/docker-compose.yml` :238–251 | `qdrant` 服務（`image: qdrant/qdrant`，未釘選標籤）＋`qdrant-data:/qdrant/storage` 掛載＋healthcheck | **移除**（服務定義整段刪除） |
| 執行設定：Compose volume | 同檔 :439 | `qdrant-data:` | **移除** |
| 執行設定：Compose 註解 | 同檔 :15、:251 | 「Qdrant／pgvector（選擇待定）」「向量承載選擇待定（Qdrant 或 pgvector）」 | **更新**為 pgvector 已定案＋ADR-017 指引 |
| 執行設定：環境變數 | `.env.example` :49–51、:11 | 唯一 `QDRANT_*` 變數：`QDRANT_API_KEY`（註解佔位符）；:11 註解「Qdrant／pgvector（選擇待定）」 | **移除段落**＋註解更新；全樹**無** `QDRANT_URL`／`QDRANT_PORT`／其他 `QDRANT_*` 變數，pgvector 不新增環境變數 |
| Go 設定結構與讀取程式 | `services/api/internal/config/config.go`、`services/api/internal/secrets/secrets.go`、`services/api/internal/telemetry/telemetry.go`、各服務 `main.go` | 設定結構僅讀 `API_PORT`、`OPENBAO_*`、`OTEL_*`／`LOKI_ENDPOINT`／`SENTRY_DSN`、`GATEWAY_PORT`／`WORKER_PORT` 等；**零 Qdrant 欄位、零 Qdrant 讀取程式** | **無需修改**；執行後 grep 驗證零新增引用（§6 S1-3） |
| Go 程式碼（其他） | `services/` 全樹 | 零 `qdrant`／`Qdrant` 字串（僅 transport 註解提及框架未定狀態，與 Qdrant 無關） | **無需修改** |
| 測試設定與範例 | 全樹（含 `*_test.go`、範例設定） | 零 `qdrant` 引用 | **無需修改** |
| 部署文件（執行清單） | `docs/deployment/SELF_HOSTING.md` :24／:70／§9、`docs/deployment/CLOUD_ARCHITECTURE.md` :47／:103／:122／:143／:209／:215、`docs/testing/TEST_STRATEGY.md` :100、根 `README.md` 服務清單 | 服務清單、volume 清單、拓撲圖、檢查項中的 Qdrant 引用 | **按 §1-D 附表更新**（執行清單，非歷史評估） |
| 保留：歷史比較與評估 | `docs/phase01/vector-store-memo.md` 全文、`docs/decisions/ADR-009-…md` :47 決策表、`docs/COST_OPTIMIZATION.md` §3.19、`docs/deployment/CLOUD_ARCHITECTURE.md` :36 比較表、`docs/phase01/compose-alignment-notes.md` 對照紀錄 | Qdrant 評估內容、比較內文、歷史敘述 | **保留**，僅加「已決議採用 pgvector，見 ADR-017」註記；不得改寫比較內文 |
| 保留：審查歷史 | `docs/phase01/REVIEW.md` | Phase 01 審查紀錄 | **不改** |

清理完成判定：§6 S1-3 grep——程式碼與設定檔（compose、Dockerfile、`.dockerignore`、`.env.example`、Makefile、CI 草稿、Go 檔）零 `qdrant` 字串、零 `QDRANT_` 變數；歷史／決策紀錄文件內的 `qdrant` 僅出現於「已移除／已決議」註記語境，逐處核對。

## 9. 映像更換前檢查與回退方案（修訂版 v2 新增，審查意見 §二-3／4／8）

### 9.1 PostgreSQL 映像更換前檢查（Docker 操作前必須完成，任一項不通過即停下；不阻塞 C 類靜態工作，見 §10）

1. **資料 Volume 狀態**：`docker volume ls`／`docker volume inspect` 確認 `postgres-data` 不存在或為空。**在 volume 狀態確認前，不得假設沒有資料**：volume 是否存在、是否含資料，一律以檢查命令輸出為準；「compose 從未啟動」僅為背景資訊，不是無資料的證據。若 volume 已存在且含資料，**停止**：先 `pg_dump` 完整備份並回報審查者，不得執行可能造成資料遺失的操作，不得直接刪除 volume。
2. **`POSTGRES_*` 初始化環境變數**：逐項比對 compose `postgresql` 服務的 `environment` 段與 `.env.example` 佔位值，更換映像前後**完全不變**（使用者、密碼佔位、資料庫名）。
3. **資料庫連線設定不變**：`DATABASE_URL`／JDBC 參數、埠 5432、healthcheck 指令（`pg_isready`）更換前後不變；各服務的資料庫環境變數組裝值逐名核對（§6 S4-13）。
4. **映像相容確認**：`pgvector/pgvector:0.8.7-pg16-bookworm` 為官方 `pg16` 標籤，與 PostgreSQL 16 相容；bookworm 基底與現行 `postgres:16` 同系；以 `docker buildx imagetools inspect` 覆核 digest 與本計畫固定值 `sha256:484e838076e04939d27d3e176e91c86eea96236126023be8d3059761122e579f` 一致（不符即停下，見 §7 R9；**不得自行換標籤、換基底或採用其他映像**）；擴充可用性驗證方式以 §6 S2-7 的 `pg_available_extensions` 查詢為準（`installed_version` 為 NULL，不執行 `CREATE EXTENSION`）。
5. **下游服務清單**：確認共用 PostgreSQL 的服務為 Keycloak 與 GlitchTip（見 §6 S4-14），更換後兩者皆須重新驗證。

### 9.2 回退方案（修訂版 v5：補充前置檢查失敗的處理界線；本次修訂：增加回退停機影響檢查，2026-11-25 審查決定）

- **回退停機影響檢查（本次修訂，2026-11-25 審查決定）**：執行 `docker compose down` 前，必須先確認：① 該 Compose 專案包含的全部服務（以 `docker compose config --services` 輸出為準）；② 是否有其他正在使用的服務（同專案內非本次範圍的服務，以及同主機上共用網路／volume 的外部服務）；③ 停止整個 Compose 專案的影響範圍（列出將停止的容器與受影響的服務）；④ 是否具備必要的維護窗口（時間窗口與負責人確認）。若任一項無法確認，立即停止回退並回報審查者，**不得自行執行 `down`**。檢查證據須附：`docker compose config --services` 完整服務清單（並以 `docker compose config` 完整渲染核對服務定義）、`docker compose ps -a` 實際服務與容器狀態、`docker ps` 該 Compose project 的實際容器、專案網路與 volume 清單、外部服務共用網路／volume 的確認結果、操作前各服務原本的啟動狀態。

- **回退前狀態證據與恢復（本次修訂，2026-11-25 審查決定）**：回退時必須：① 確認整個 Compose 專案停止的影響；② 確認維護窗口與負責人；③ 無法確認影響時立即停止並回報，不得執行 `down`；④ 若執行回退，必須依照操作前的服務狀態恢復；⑤ 原本停止的服務不得擅自啟動；⑥ 不得以無參數的 `docker compose up -d` 直接啟動原本未運作的所有服務，除非已證明操作前全部服務都在運作（以回退前狀態證據為準）。

- **前置檢查失敗（映像尚未更換、stack 尚未因本次操作啟動）**：若僅是 §9.1 前置檢查失敗（例如 digest 覆核不符、volume 狀態未確認、`POSTGRES_*` 比對不符），且本次映像更換尚未執行、stack 尚未因本次操作啟動，應停止後續操作並回報審查者，**不得為了執行回退而無必要地停止現有服務**；不執行下述回退步驟（無可回退的狀態變更）。受阻項目依 §6 標示「未驗證」並記錄受阻範圍。
- **回退觸發條件（僅適用本次映像更換或相關啟動操作已經實際發生，且任一即回退）**：digest 覆核不符（限已實際使用該映像啟動 stack 後覆核發現；拉取後、啟動前發現者屬前置檢查失敗，見前款）、PostgreSQL healthcheck 失敗、Keycloak 或 GlitchTip 更換後無法使用資料庫。
- **回退步驟**：① 停機影響檢查通過後，停止 stack（`docker compose down`，**不帶 `-v`**，不得刪除 volume）；② compose `postgresql` 的 `image` 改回 `postgres:16`；③ 重新啟動並執行 `pg_isready`＋Keycloak／GlitchTip 健康檢查；④ 回退結果寫入驗證報告，**不得沿用更換前的驗證結果**。
- **資料安全（本次修訂，2026-11-25 審查決定）**：§9.1 確認 volume 不存在或為空，只代表操作開始前的狀態。若操作前 volume 已存在且有資料，必須完整備份（`pg_dump`）並回報審查者，**停止等待指示，不得因完成備份而自行繼續**。若操作開始時 volume 不存在或為空，PostgreSQL 啟動後仍可能產生新資料。觸發回退時必須重新檢查 volume 與資料狀態；只要已有資料，必須先確認回退前備份完整性，確認完成前不得執行回退。不得將操作前的空 volume 狀態當作回退時無資料風險的證據。回退本身不得刪除任何 volume，不得使用 `down -v`。
- **回退後狀態**：回退即表示本次 pgvector 映像更換未完成，須回報審查者並等候指示，不得自行改用其他映像或標籤。
- **通用禁令**：無論前置檢查失敗或回退執行，均不得使用 `docker compose down -v`，不得刪除 Volume。

### 9.3 LiveKit 預設行為確認（併入 §4）

見 §4「預設行為確認」與 §6 S1-2／S3-9／S3-11：預設流程以實際容器狀態確認 `livekit` 未啟動（附 Docker Compose 版本記錄）；profile 流程記錄預設 entrypoint／command、無 keys 時的 `ErrKeysNotSet` 行為與 healthcheck 結果，不得加入開發金鑰；任何流程下皆不得宣稱語音或視訊功能已可用。

### 9.4 Registry 映像清單（本次修訂，2026-11-25 審查決定）

盤點方法：2026-11-25 實查 `infrastructure/docker/docker-compose.yml` 全部 `image:` 欄位、三份 Dockerfile 全部 `FROM`（含 `COPY --from=builder` 皆為階段內引用，無額外基底映像）與 CI 草稿（僅 buildx／build-push action，無容器映像）。未標 tag 者以 Registry 預設 `latest` 為準，浮動 tag 屬 G-04 追蹤範圍，本次不更動。Digest 完整性：本計畫的命令與驗證條件一律使用完整 digest（下表 pgvector 列）；§3 所引 ADR-017 內容為已驗收檔案逐字引用，其內省略形式的架構 digest 僅為背景紀錄，不得用作比對值；比對一律以完整 digest 為準，不得僅比對頭尾。存取結果欄的「未查核」指尚未在驗證環境以 manifest 查詢確認，本次修訂未拉取任何映像；不得因部分映像可以存取，就宣告全部 Registry 前置檢查通過，須逐項確認後才可認定通過。

| 映像 | Tag 或 Digest | Registry | 存取結果 | 本次實際建置或啟動所需 |
|---|---|---|---|---|
| `traefik` | `v3`（浮動 tag） | Docker Hub | 未查核 | S3-9 預設啟動 |
| `ether-api` | `dev`（本地建置） | —（本地建置） | 不適用（無 Registry 拉取） | S2-6 建置 |
| `ether-realtime-gateway` | `dev`（本地建置） | —（本地建置） | 不適用（無 Registry 拉取） | S2-6 建置 |
| `ether-worker` | `dev`（本地建置） | —（本地建置） | 不適用（無 Registry 拉取） | S2-6 建置 |
| `pgvector/pgvector` | `0.8.7-pg16-bookworm@sha256:484e838076e04939d27d3e176e91c86eea96236126023be8d3059761122e579f`（標籤＋digest 雙固定；架構 digest：amd64 `sha256:018b4a92e634c3a5715f613dd43497e3d8028ae051be7ba6fd127bcffb0e7957`、arm64 `sha256:d153dea9f9bb6ee63ec539d2febf8264848ba962808484054ea77bb32e5a3853`，2026-11-25 向 Docker Hub registry 查得） | Docker Hub | 固定 digest 已於 2026-11-25 查證（見 §9.1）；執行時須以 `docker buildx imagetools inspect` 覆核 | S2-7、S3-9 |
| `valkey/valkey` | `8`（浮動 tag） | Docker Hub | 未查核 | S3-9 預設啟動 |
| `nats` | `2`（浮動 tag） | Docker Hub | 未查核 | S3-9 預設啟動 |
| `minio/minio` | 未標 tag（預設 `latest`） | Docker Hub | 未查核 | S3-9 預設啟動 |
| `getmeili/meilisearch` | `v1`（浮動 tag） | Docker Hub | 未查核 | S3-9 預設啟動 |
| `openbao/openbao` | 未標 tag（預設 `latest`） | Docker Hub | 未查核 | S3-9 預設啟動 |
| `quay.io/keycloak/keycloak` | `26`（浮動 tag） | Quay.io | 未查核 | S3-9 預設啟動、S4-14 |
| `livekit/livekit-server` | 未標 tag（預設 `latest`） | Docker Hub | 未查核 | S3-11 profile 流程 |
| `ollama/ollama` | 未標 tag（預設 `latest`） | Docker Hub | 未查核 | S3-9 預設啟動 |
| `prom/prometheus` | 未標 tag（預設 `latest`） | Docker Hub | 未查核 | S3-9 預設啟動、S1-4（經映像執行 promtool 時） |
| `grafana/grafana` | 未標 tag（預設 `latest`） | Docker Hub | 未查核 | S3-9 預設啟動、S3-10 |
| `grafana/loki` | 未標 tag（預設 `latest`） | Docker Hub | 未查核 | S3-9 預設啟動、S3-10 |
| `glitchtip/glitchtip` | 未標 tag（預設 `latest`） | Docker Hub | 未查核 | S3-9 預設啟動、S4-14 |
| `golang` | `1.25.0-alpine3.21`（建置階段） | Docker Hub | 未查核 | S2-6 建置階段 |
| `alpine` | `3.21.3`（執行階段） | Docker Hub | 未查核 | S2-6 執行階段 |

## 10. 執行順序與停止點（修訂版 v5：與 §6 四類分類一致）

界線原則：計畫取得批准後，C 類靜態工作（步驟 1–6）可先行，不受 §9.1 前置檢查阻塞；§9.1 前置檢查為 A 類門檻，僅 gating 任何 Docker／Buildx 操作（步驟 7）。靜態修改完成不得解讀為 Docker 驗證完成或 Phase 01 最終驗收完成。

執行順序（計畫批准後）：

1. [C 類] ADR-015／016／017 先行建立（決策紀錄為準，後續修改皆引用 ADR）。
2. [C 類] Compose 修改（G-01、Qdrant 移除、PostgreSQL 映像、livekit 掛載＋profiles、prometheus 掛載）＋三份 Dockerfile／`.dockerignore` 註解＋Makefile。（檔案修改本身不需 Docker；修改後的 stack 啟動與驗證屬 B／D 類，見步驟 7。）
3. [C 類] 新建 `prometheus.yml` 最小設定。
4. [C 類] 程式註解、服務 README、根 README、CI 草稿註解。
5. [C 類] Phase 00 文件 16 處註記＋Phase 01 紀錄 6 處補註＋兩份備忘錄檔頭註記。
6. [C 類] 靜態驗證（§6 S1 中不需 Docker 的部分：S1-1／S1-3／S1-5）全數執行並記錄；S1-2／S1-4 需 Docker 的部分列為未驗證並註明受阻範圍（見 §6 分類表）。
7. [A 門檻 → B／D 類] Docker 操作前必須完成 §9.1 全部前置檢查（volume 狀態、digest 覆核、`POSTGRES_*` 與連線設定比對、下游服務清單）；任一項不通過，或 Docker／Buildx 環境不存在，即停止所有受影響的 Docker 操作及映像更換，不得假設檢查通過；受阻項目依 §6 標示「未驗證」並記錄受阻範圍。檢查通過後，執行 §6 S2–S5 驗證（待 CI 或有 Docker 的環境）。
8. **停止**：提交完整差異與驗證報告，等待下一次審查；未取得 Phase 01 最終驗收批准前，不得進入 Phase 02。

### 步驟 7 驗證環境規格與隔離要求（本次修訂，2026-11-25 審查決定）

環境三類之一（由負責人明確指定）：已安裝 Docker Engine、Docker Compose v2、Buildx 的專用 VM；能存取 ETHER 專案工作目錄及 Docker daemon 的 CI Runner；其他經負責人明確指定的驗證環境。本機 VM（Ubuntu 24.04.5，已確認無 Docker／Buildx／Compose／promtool／Go 及 Docker daemon）不符合本規格，不得用於步驟 7。

- **隔離性**：專用且隔離的 Docker daemon；不使用正式環境；不使用其他專案的資料 Volume；不得與其他專案共用可能被 `docker compose down` 影響的 Compose project；操作前以 `docker ps`、`docker network ls`、`docker volume ls` 確認主機上的容器、網路、Volume 歸屬並記錄；無法確認隔離性時停止，不得執行。
- **環境資訊記錄**（操作前提交並確認）：主機或 Runner 識別資訊、OS 與版本、Docker Engine／Compose／Buildx 版本、Docker daemon 可用性、ETHER 工作目錄及 Git commit、Registry 存取能力（見 §9.4）、執行驗證所需的權限；以及 `git rev-parse HEAD`、`git status --short`、已驗收 37 個修改檔案及 4 個新增檔案的實際狀態、必要的差異與檔案雜湊證據。若工作目錄非 git 倉庫，上述 git 命令失敗本身即為須記錄的狀態，不得偽造 commit；以檔案雜湊清單為基準證據。不得自行修改、還原或建立新的提交來改變已驗收基準。

靜態修改完成（步驟 1–6）不等於 Docker 驗證完成（步驟 7），亦不等於 Phase 01 最終驗收完成。
