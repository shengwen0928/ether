狀態：已定案（2026-11-25 審查決議）
決議來源：使用者（審查者）2026-11-25 07:19 於對話發出的《ETHER Phase 01 審查決議》全文（八節）；執行計畫 docs/phase01/EXECUTION_PLAN.md（修訂版 v5）§3。

# ADR-017：採用 pgvector


- **狀態**：已定案（2026-11-25 審查決議）。備忘錄 `docs/phase01/vector-store-memo.md` 保留為決策紀錄；其中關於安全性結果的描述以本 ADR 為準，不得將採用 pgvector 描述成安全性結果自動成立。
- **背景**：Phase 00 要求向量承載二選一；compose 內含未釘選標籤的 `qdrant` 服務（僅佔位，選擇待定）；`postgres:16` 官方映像未含 pgvector 擴充。
- **決議**：採用 pgvector，以 PostgreSQL＋pgvector 為初期向量儲存方案；移除 compose 的 `qdrant` 服務與 `qdrant-data` volume；清理 Qdrant 環境變數與設定引用；保留 AI Provider Adapter 後方的向量存取抽象介面；業務模組不得直接綁定特定向量資料庫。**採用 pgvector 不得降低 `AI_DATA_POLICY.md` 的安全要求**；權限隔離、刪除同步、撤銷後不可見的證明責任見「安全性要求保留」。保留 pgvector 選型，不得自行改選 Qdrant。
- **理由（按備忘錄權重，已按 v4 審查意見調整用語；v3 的過度保證描述作廢）**：① 交易一致性條件較佳——PostgreSQL 與 pgvector 位於同一資料庫，關聯資料與向量可在同一交易內操作，**有利於**交易一致性及資料管理，減少跨系統協調；此為有利條件，不是自動保證，安全性結果仍須實現與驗證（見下）；② 自架棧最簡——不新增狀態服務、備份 runbook 與監控面，沿用 PostgreSQL 備份與監控；③ Go 接入邊際成本最低——沿用 `pgx`＋`pgvector-go`，無新協定；④ 規模證據缺席——無 Phase 00 證據顯示需要專用引擎，不得以未證實規模引入。
- **安全性要求保留（修訂版 v4 新增）**：權限隔離、刪除同步、撤銷後不可見仍是**必須實現與驗證的設計要求**，不是採用 pgvector 後即成立的結果。「向量與權限事實同一交易」「刪除即消失」「撤銷即不可見」「無同步窗口」等表述不得寫成採用後的既成結果。上述安全性結果必須由 Phase 09 的資料模型、交易邊界、查詢授權條件、刪除流程及測試證明；證明前不得宣稱已達成。`AI_DATA_POLICY.md` 的安全要求維持不變，不得因採用 pgvector 而降低。
- **替代方案（Qdrant）與代價**：專用引擎的檢索表達力（payload 過濾、稀疏＋稠密混合、量化）與獨立擴充；代價為獨立服務的部署／備份／監控負擔、Outbox／NATS 投影同步與刪除對帳、查詢層的 PostgreSQL 權限再過濾、同步延遲窗口的安全論證。未採納，備忘錄保留為決策紀錄。
- **版本與映像固定策略（修訂版 v2，審查意見 1）**：備忘錄查證 pgvector 0.8.7（2026-10-01，修正 CVE-2026-103484 IVFFlat）。PostgreSQL 16 固定採用官方映像 **`pgvector/pgvector:0.8.7-pg16-bookworm@sha256:484e838076e04939d27d3e176e91c86eea96236126023be8d3059761122e579f`**——標籤＋digest 雙固定，浮動標籤禁用。查證：Docker Hub 標籤 `0.8.7-pg16-bookworm` 存在（last pushed 2026-11-12；amd64 digest `sha256:018b4a92…0e7957`、arm64 `sha256:d153dea9…e5a3853`）；manifest list digest `sha256:484e8380…122e579f` 已於 2026-11-25 向 Docker Hub registry 查證。執行時須以 `docker buildx imagetools inspect` 覆核 digest，不符即停下並回報（見 §7 R9、§9）。映像與 PostgreSQL 16 相容（官方 `pg16` 標籤命名）；bookworm 基底與現行 `postgres:16` 同系。更換前檢查與回退方案見 §9。
- **影響範圍**：compose（服務移除、volume 移除、PostgreSQL 映像更換）、`.env.example`、部署／自架／成本／ERD／AI 規範等 16 處文件註記、對照文件補註。安全性結果的證明屬 Phase 09（見「安全性要求保留」），不屬本次驗證範圍，本次不得宣稱已證明。
- **重估條件（升級觸發，皆須 Phase 09 實測證據）**：HNSW 記憶體佔用影響 OLTP、查詢延遲實測不達標、或出現 pgvector 無法表達的檢索需求；切換時只須替換 Adapter 實作＋匯出／重建向量（維度與距離度量須一致），不得改動商業邏輯。E2EE 明文禁入向量索引的規則不變，預設不得索引。
- **執行檢查點**：`docker compose config` 渲染無 `qdrant` 服務與 volume；PostgreSQL 映像按 digest 拉取成功且擴充可用性驗證通過（§6 S2-7）；grep 證明程式碼與設定檔零 Qdrant 引用（歷史／決策紀錄除外）；Phase 09 驗收須含權限隔離／刪除同步測試證據（非本次範圍，僅列為後續驗收條件）。
