狀態：已定案（2026-11-25 審查決議）
決議來源：使用者（審查者）2026-11-25 07:19 於對話發出的《ETHER Phase 01 審查決議》全文（八節）；執行計畫 docs/phase01/EXECUTION_PLAN.md（修訂版 v5）§3。

# ADR-015：G-01 採用獨立 Dockerfile 與獨立建置 context


- **背景**：Phase 00 的 compose 以 `services/api` 為共用 context，參照不存在的 `Dockerfile.gateway`／`Dockerfile.worker`（G-01）；Phase 01 已建三份獨立 Dockerfile，兩套佈局並存即 drift 來源。
- **決議**：採用方案 A——`api`、`realtime-gateway`、`worker` 各自以自己的服務目錄為 context、各用目錄內的 `Dockerfile`；刪除共用 context 參照。
- **理由**：與三個獨立 Go module（`ether.local/*`）的佈局一致；各服務的 `.dockerignore` 各自生效，COPY 集合清晰；建置快取互不污染；與 MODULE_BOUNDARIES.md 的模組邊界一致。
- **替代方案（B）與代價**：維持單一 `services/api` context，補 `Dockerfile.gateway`／`Dockerfile.worker`——須撤回或降級已建的獨立目錄 Dockerfile，獨立 module 與共用 context 佈局長期不一致，未來拆分成本更高。未採納。
- **影響範圍**：compose 三個服務的 build 段、三份 Dockerfile 檔頭、三份 `.dockerignore` 檔頭、Makefile 提示、CI 草稿註解、對照文件補註。
- **重估條件**：未來若三服務合併為單一部署單元（架構變更），須以新 ADR 重估；僅有效能或快取理由不得自行改回共用 context。
- **執行檢查點**：`docker compose config` 渲染三服務 build 段正確；三個映像各自建置成功；grep 零 `Dockerfile.gateway`／`Dockerfile.worker` 引用（歷史紀錄文件除外，見 §6）。
