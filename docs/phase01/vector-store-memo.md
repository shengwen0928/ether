# Qdrant vs pgvector 比較備忘錄（Phase 09 決策支援）

> **決議註記（2026-11-25）：** 已決議採用 pgvector，見 ADR-017。
> 本備忘錄比較內文保留為決策證據；關於安全性結果的描述以 ADR-017 為準，不得將採用 pgvector 描述成安全性結果自動成立。
> 相關決議：G-01 方案 A（ADR-015）、Echo v5（ADR-016）、G-02／G-03 處理方案（執行計畫 §4）。

狀態：草案／待使用者決定。本備忘錄只提供比較與推薦，**最終決定權在使用者**；推薦不等於決定。在決定前，不得預設任一選項已選定：`MODULE_BOUNDARIES.md` §7 明載向量檢索的歸屬在定案前不得由 Search 或 Analytics 模組預設擁有；向量存取一律經 AI Provider Adapter 介面（見 §7），選擇藏在介面之後。

查證日期：2026-11-25。版本與授權狀態皆為當日公開來源查證結果，決定時須在 CI／部署環境覆核是否仍為最新；本備忘錄不引用、亦不編造任何基準測試數字。

## 1. 比較範圍：Phase 09 的實際需求

ETHER AI（手冊 K，`ROADMAP.md` Phase 09）的向量相關需求，以 Phase 00 文件為準：

- 知識 AI：個人知識庫、文件索引、向量搜尋、RAG、語意搜尋；**權限隔離與刪除同步為強制要求**（文件被刪除或權限變更時，向量索引須同步，不得殘留無權內容）。
- 訊息／商業 AI：摘要、翻譯、智慧回覆、官方 AI 客服等——向量需求集中於知識庫檢索，關鍵字搜尋另有 Meilisearch（`DATA_FLOW.md` §8 的搜尋索引同步）；向量庫**不是** Meilisearch 的替代品，兩者分工須在 AI 規範文件中定稿，不得混為一談。
- 安全紅線：E2EE 聊天室的明文不得自動送往雲端 AI；AI 呼叫須經授權、用量、遮罩、保留策略（手冊 K）。向量庫不得接收 E2EE 明文；E2EE 內容的 embedding 是否可建索引，須經 AI 資料處理規範定稿，預設為**不得索引**。
- 部署前提：自架免費棧（Docker Compose＋Coolify、PostgreSQL 自架、Valkey、MinIO、Meilisearch、NATS；`SPEC-SOURCE.md` 架構調整一節）。向量庫的選擇直接影響自架主機的服務數量、備份策略與維運負擔。
- 規模證據：**無**。Phase 00 文件未給出向量數量、查詢量、文件規模的任何數字；embedding 模型與向量維度亦未定（待 Phase 09 評估）。本次比較不得以假設的規模數字為依據；「專用引擎效能較佳」在未證實的規模下不是決定理由。

## 2. 查證到的版本與授權狀態（2026-11-25）

| 項目 | Qdrant | pgvector |
|---|---|---|
| 性質 | 獨立的向量搜尋引擎（Rust 撰寫），REST＋gRPC 雙介面 | PostgreSQL 擴充套件，在資料庫內提供向量型別與近似最近鄰索引 |
| 最新版（查證日） | v1.20.2（2026-11-16 發布）；v1.20（2026-10-22）新增 BM25 全文索引、8／16-bit TurboQuant 等 | 0.8.7（2026-10-01 發布）；0.9.0 開發中 |
| 授權 | Apache-2.0（寬鬆授權，可自架、可商用，須保留授權與聲明） | PostgreSQL License（寬鬆授權，BSD 風格；自架免費，無授權費用） |
| 檢索能力（查證日） | 稠密＋稀疏＋多向量檢索、payload 過濾、量化（Scalar／Binary／Product／TurboQuant）、1.20 起 BM25 關鍵字評分 | `vector`／`halfvec`／`bit`／`sparsevec` 型別、HNSW 與 IVFFlat 索引、L2／內積／餘弦等距離算子；`vector` 上限 2,000 維、`halfvec` 4,000 維（查證來源） |
| 部署形態 | 單一程序＋單一資料目錄（容器化）；預設埠 6333（REST）／6334（gRPC） | 隨 PostgreSQL 一起部署：須在映像中安裝擴充並 `CREATE EXTENSION vector` |

來源：Qdrant 版本與授權見 <https://selfhostmap.com/apps/qdrant/>、<https://www.opentechhub.io/qdrant/>、<https://www.motiontic.com/compare/pgvector-vs-qdrant>；pgvector 版本、授權與維度上限見 <https://andrew.ooo/answers/best-postgres-vector-search-extensions-2026/>、<https://codewithsql.com/compare/pgvector-vs-pinecone>。

## 3. 逐項比較

### 3.1 RAG／語意搜尋場景

- **Qdrant**：為檢索而生的專用引擎。payload 過濾（metadata 條件與向量查詢同一請求）、稀疏向量與稠密向量混合、量化儲存、collection／point 模型，對「複雜過濾＋混合檢索」的表達力最強。1.20 起的 BM25 可在同一引擎內做關鍵字＋向量混合，但 ETHER 已有 Meilisearch 承擔關鍵字搜尋，此能力是重疊而非必需。
- **pgvector**：向量即資料行，檢索即 SQL：`ORDER BY embedding <=> $1 LIMIT n`，權限條件、文件關聯、時間篩選皆以同一 SQL 查詢的 `WHERE` 表達，與業務查詢的結合最自然。混合檢索（關鍵字＋向量）須跨 Meilisearch 與 pgvector 兩邊合併結果，合併邏輯自寫；稀疏向量支援較新，能力邊界待 Phase 09 以實際模型驗證。
- **ETHER 場景的關鍵差異**：知識庫檢索的核心難題不是「檢索表達力」，而是「權限隔離＋刪除同步的正確性」（見 §1）。pgvector 把向量與權限事實放在同一交易內，正確性由資料庫保證；Qdrant 把權限事實放在 PostgreSQL、把向量放在另一系統，正確性由同步機制保證（見 §3.7）。在 Phase 09 的個人知識庫規模下，正確性優先於檢索表達力。

### 3.2 部署型態（與現有 compose 的關係）

現行 `docker-compose.yml`（草案／待批准，**本軌不得修改**）的現況：

- `qdrant` 服務**已存在**：映像 `qdrant/qdrant`（**未釘選標籤**，浮動風險待 compose 軌處理）、`qdrant-data` volume、healthcheck `http://localhost:6333/healthz`、僅掛 `ether-back` 內網。服務未設 API key 環境變數；`.env.example` 的 `QDRANT_API_KEY` 為註解狀態（選擇待定，不得假設已啟用）。
- `postgresql` 服務為 `postgres:16` 官方映像，**未含 pgvector 擴充**。選 pgvector 須更換資料庫映像（例如官方 `pgvector/pgvector` 映像的對應 PG 16 標籤，標籤須在 CI 以官方映像清單確認，不得預寫）或以其他方式安裝擴充；此為 compose 變更，須由協調員／compose 軌裁決，本備忘錄只指出必要性。
- **結論**：以「compose 現況可直接啟動」論，Qdrant 佔優（服務已寫好）；以「部署後須改的東西」論，兩者皆須改（Qdrant 須釘選標籤＋API key；pgvector 須更換資料庫映像）。不得以「compose 已有 qdrant 服務」推斷 Qdrant 已選定——compose 檔頭註解明載「向量承載選擇待定，未選定前不得假設本服務為主力」。

### 3.3 備份／還原

- **Qdrant**：快照（snapshot）機制匯出 collection 資料，另須備份儲存 volume；還原是「Qdrant 快照＋PostgreSQL 備份」兩套流程，兩者之間無交易一致性保證，還原後須以對帳／重建流程驗證向量與業務資料的一致（例如文件已刪、向量仍在的殘留檢查）。備份排程、保留、還原演練皆須另寫 runbook（`ROADMAP.md` Phase 11 要求備份還原演練通過）。
- **pgvector**：向量與業務資料在同一 PostgreSQL 內，現有資料庫備份（`pg_dump`／base backup，方式待維運文件定稿）一次覆蓋兩者，還原即一致，無跨系統對帳問題。還原演練沿用資料庫既有流程，不新增 runbook。
- **結論**：pgvector 的備份／還原故事顯著簡單，且與 ETHER「系統紀錄在 PostgreSQL」的架構（`SYSTEM_ARCHITECTURE.md` §3.2）一致。

### 3.4 授權

- 兩者皆為寬鬆授權（Qdrant Apache-2.0；pgvector PostgreSQL License），自架、商用皆無授權費用，皆符合 ETHER 的免費／開源自架原則。授權不是本次決定的區分因素；決定時仍須以兩專案儲存庫內的 LICENSE 檔案為準覆核（授權狀態可能變更，本備忘錄的查證日為 2026-11-25）。

### 3.5 Go 客戶端生態

- **Qdrant**：官方 Go 客戶端 `github.com/qdrant/go-client`（gRPC 為主，含連線池、TLS、API key 支援；REST 亦可用，官方建議原型用 REST、正式用 gRPC 視除錯需求而定）。另有 `qdrant.tech/documentation/interfaces/` 的介面文件。Go 後端以 gRPC 呼叫 Qdrant，與 ETHER 內部 gRPC 預留方向一致（`SYSTEM_ARCHITECTURE.md` §3.2）。
- **pgvector**：Go 路徑為 `pgx`（`github.com/jackc/pgx/v5`）＋ `pgvector-go`（`github.com/pgvector/pgvector-go`，含 pgx 整合），向量操作即 SQL，無新增的網路協定、無新增的客戶端依賴面；資料庫連線池沿用既有 `pgx` 設定（`SYSTEM_ARCHITECTURE.md` §4 要求 Connection Pool）。若後端資料層已選 `pgx`，pgvector 的 Go 接入幾乎零新增依賴；若資料層另選驅動，須一併確認 `pgvector-go` 的相容性（待 Phase 03 資料層定稿）。
- **結論**：兩者的 Go 生態皆成熟可用；pgvector 的邊際依賴較小（沿用資料庫驅動），Qdrant 的官方客戶端較完整（連線池、TLS、認證內建）。

### 3.6 效能取捨

- 本備忘錄**不引用任何基準測試數字**：未在 ETHER 的硬體、模型、維度、資料量下實測，公開的向量庫基準數字因測試條件不同而無法直接比較；編造或轉抄數字會誤導決定。
- 定性取捨：專用引擎（Qdrant）在「大量向量、高查詢量、複雜過濾」下的調校空間較大（量化、記憶體層級、sharding 皆為引擎原生能力）；pgvector 的檢索效能與 PostgreSQL 共享資源（記憶體、連線、I/O），HNSW 索引的記憶體需求與 OLTP 負載互相影響，須以索引調校與資源隔離（例如唯讀複本，`SYSTEM_ARCHITECTURE.md` §4.2）應對。反之，pgvector 免除了跨系統的網路往返與序列化開銷，小規模查詢的端到端延遲未必較差。
- **決定原則**：效能取捨須以 Phase 09 的實測（實際模型維度、實際文件量、實際查詢型態）為準；Phase 00–01 不得預設任一選項「效能足夠」或「效能不足」。`SYSTEM_ARCHITECTURE.md` §4.2 的擴充原則同樣適用：只在觀測到瓶頸證據後擴充，不得跳級拆分。

### 3.7 與現有自架棧的整合

- **pgvector**：整合點只有一個——PostgreSQL。索引同步不需要 Outbox／NATS 投影（向量寫入與業務寫入同一交易）；搜尋結果的權限過濾即 SQL `WHERE`，與 `DATA_FLOW.md` §8「搜尋結果回傳前經權限介面過濾」的規則天然一致；監控沿用資料庫監控（連線池、慢查詢，`SYSTEM_ARCHITECTURE.md` §8.2）。代價：向量負載與業務負載共用同一資料庫，容量規劃須合併考量。
- **Qdrant**：整合點為新增的獨立服務。寫入路徑須走事件投影（Outbox → NATS → Worker → Qdrant，類比 `DATA_FLOW.md` §8 的 Meilisearch 同步）：業務寫入不得因索引失敗而失敗，投影寫入須冪等，刪除／權限變更須同步至 Qdrant，索引遺失須可由 PostgreSQL 全量重建。監控須新增 Qdrant 指標（佇列積壓、同步延遲、collection 大小）與告警；部署須新增 volume 備份與升級流程。
- **結論**：pgvector 的整合面最小；Qdrant 的整合面等同於「再引入一個 Meilisearch 式投影」，維運複雜度相應增加。

### 3.8 選 pgvector 時 qdrant 服務的處置

1. 從 `docker-compose.yml` 移除 `qdrant` 服務定義與 `qdrant-data` volume（**compose 變更，須由協調員裁決，本軌不得修改 compose**）；移除前確認無任何程式碼或文件依賴 Qdrant 端點。
2. `.env.example` 的 `QDRANT_API_KEY` 註解行隨之成為死佔位，須由 compose／環境軌清理或標示已廢棄，不得留作誤導。
3. 決策記錄：以 ADR 記錄「pgvector 選定、Qdrant 服務移除」與回退條件（見 §5）；`SELF_HOSTING.md`／部署文件中提及 qdrant 的段落須同步修訂，不得殘留「qdrant 服務待定」的矛盾敘述。
4. 不得刪除 Qdrant 的評估資料：本備忘錄與查證來源保留，作為日後重估的依據。

### 3.9 選 Qdrant 時資料一致性議題

1. **同步延遲窗口**：業務寫入提交後、Qdrant 投影到達前，搜尋結果可能含已刪文件或缺新文件；視窗長短取決於 Outbox Relay 與 Worker 延遲，須監控並訂出上限（數值待 Phase 09 定稿，不得預寫）。權限撤銷的同步延遲是安全議題，不得只以「最終一致」帶過：撤銷後、同步前的查詢不得回傳已無權內容——查詢層須以 PostgreSQL 當下權限狀態再過濾（類比 `DATA_FLOW.md` §8 的規則：不得信任索引內的可見性標記為唯一依據）。
2. **刪除同步**：文件刪除、帳號刪除、權限變更皆須產生刪除／更新事件並投遞至 Qdrant；投遞失敗走 NATS 重試與 DLQ，不得無聲丟棄；須有定期對帳任務（PostgreSQL vs Qdrant 的存在性比對），對帳頻率與修復方式待定稿。
3. **雙寫失敗模式**：Outbox 保證「業務寫入＋事件寫入同一交易」，但 Qdrant 寫入本身不在該交易內；Worker 崩潰、Qdrant 停機、網路分割時，投影落後但業務不受影響（符合 `DATA_FLOW.md` §8「業務寫入不得因索引失敗而失敗」），恢復後須追趕，追趕期間的搜尋降級方式須定稿（例如標示結果可能不完整，不得謊稱完整）。
4. **備份一致性**：見 §3.3，還原後須對帳，不得假設兩套備份時間點一致。

## 4. 推薦（有依據，決定權在使用者）

**推薦：pgvector。**

理由（按權重排序）：

1. **正確性優先**：Phase 09 的強制要求是權限隔離與刪除同步。pgvector 以同一交易保證向量與權限事實的一致，刪除即消失、撤銷即不可見，無同步窗口、無對帳任務；Qdrant 須以投影機制事後追趕一致，正確性成本較高。
2. **自架棧最簡**：不新增狀態服務、不新增備份 runbook、不新增監控面；向量與 64 張業務表共用 PostgreSQL 的備份、權限、連線池與監控，符合 ETHER「免費自架、維運負擔最小」的約束。
3. **Go 接入邊際成本最低**：沿用 `pgx`＋`pgvector-go`，無新協定、無新客戶端依賴面。
4. **規模證據缺席**：沒有任何 Phase 00 證據顯示個人知識庫的向量規模需要專用引擎；以未證實的規模假設引入專用引擎，違反「擴充只在瓶頸證據後進行」的架構原則。

## 5. 若選另一個（Qdrant）的代價與回退路徑

- 代價：新增獨立服務的部署、備份、監控、升級負擔；Outbox／NATS 投影同步與刪除對帳的開發與維運成本；查詢層的 PostgreSQL 權限再過濾（不得信任 payload）；同步延遲窗口的安全論證。換得的好處：專用引擎的檢索表達力（payload 過濾、稀疏＋稠密混合、量化）與獨立擴充能力；若 Phase 09 實測證明向量規模或查詢型態超出 pgvector 的合理範圍，Qdrant 是正確的升級路徑。
- **回退／升級路徑（雙向）**：向量存取必須藏在 AI Adapter 層的介面之後（`MODULE_BOUNDARIES.md` §7：商業模組只呼叫 Adapter 介面，不得直接依賴供應商 SDK；向量庫的歸屬待 AI 規範文件定歸屬）。介面內容（upsert／query／delete／權限過濾參數）須在 Phase 09 實作任一選項前定稿；屆時 pgvector ↔ Qdrant 的切換只須替換 Adapter 實作＋匯出／重建向量（維度與距離度量須一致），不得改動商業邏輯。
- **選 pgvector 後的升級觸發條件**（建議寫入 ADR）：HNSW 記憶體佔用影響 OLTP、查詢延遲在實測中不達標、或出現 pgvector 無法表達的檢索需求——三者皆須以 Phase 09 實測證據為準，不得以預測觸發。

## 6. 決定後的第一步（建議，不屬本備忘錄的決定）

1. 以 ADR 記錄選擇與回退條件（見 §5），並同步修訂 compose（qdrant 服務的去留、資料庫映像、標籤釘選、API key）與 `.env.example`（`QDRANT_API_KEY` 的去留）。
2. 定稿 embedding 模型與向量維度（`vector` vs `halfvec`、距離度量、HNSW 參數或 Qdrant collection 參數），未定模型前不得建索引。
3. 定稿向量存取的 Adapter 介面與權限／刪除語義（含 E2EE 內容禁入規則），再實作任一選項。
4. 定稿備份／還原 runbook（pgvector 沿用資料庫流程；Qdrant 須新增快照＋對帳流程）與監控指標，納入 Phase 11 的備份還原演練。

## 7. 未查證／未決定事項

- Embedding 模型、向量維度、文件規模、查詢量：全部未定，不得以假設數字選型。
- 資料層驅動選型（`pgx` 與否）：待 Phase 03 定稿，影響 `pgvector-go` 的接入方式。
- 向量資料的模組歸屬（Search／Analytics／AI Adapter 層）：待 AI 規範文件定歸屬（`MODULE_BOUNDARIES.md` §7）。
- Qdrant 的 API key 啟用方式與內網存取控制：待部署定稿（compose 現況未設 key）。
- Meilisearch 與向量庫的混合檢索分工：待 AI 規範文件定稿，不得以本備忘錄預設。
- 兩選項在 ETHER 實際負載下的延遲與吞吐：未實測，不得以公開基準數字代替。
