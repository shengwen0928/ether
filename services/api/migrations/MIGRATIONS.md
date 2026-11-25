# Migration 慣例（MIGRATIONS.md）

狀態：草案／待批准。本機 VM 無 psql／docker，全部 migration 皆為「待 CI 或有 PostgreSQL 的環境驗證」，不得宣稱已套用。

## 1. 依據

- 表名以 `SPEC-SOURCE.md` §20 為準，不得增刪改名。§20 實際為 64 張表（見 `docs/database/ERD.md` §2 的表數核對警示：任務交辦稱 60 張，逐字清點為 64 張），不得以 60 張為準刪表。
- 欄位級定義見 `docs/database/ERD.md`（草案欄位、待 Migration 定稿）；本目錄的 migration 定稿後，ERD 文件再同步修訂。
- 命名與通用慣例（ERD.md §4，草案）：表名 `snake_case`；主鍵一律名為 `id`（複合主鍵的關聯表除外）；外鍵欄位名為 `<被參照表單數>_id`（例如 `user_id`、`conversation_id`）；時間欄位以 `timestamptz` 為草案預設；每張表皆須有 PK、FK、Unique Constraint、必要索引、建立／更新時間（`created_at`／`updated_at`，append-only 表僅 `created_at`，見 ERD.md §3）、合理刪除策略。
- 不得將密碼明文、Token 明文、Bot 金鑰明文、私鑰、信用卡完整資訊寫入任何資料表。

## 2. 版本化命名與只往前原則

- 檔案命名：`NNNNNN_<名稱>.up.sql`／`NNNNNN_<名稱>.down.sql`，編號六位數字、遞增、不得重用、不得跳號後回填。名稱為 `snake_case` 簡短描述（例如 `000001_baseline`）。
- 只往前（forward-only）：已套用（已合併／已部署）的 migration 不得編輯、不得改名、不得刪除；發現錯誤時以新的 migration 修正，並在新檔案註解中說明修正的版本與原因。
- 每個 migration 只做一件事：一個 migration 不得同時建多個無關領域的表；業務表按模組歸屬（`MODULE_BOUNDARIES.md` §4）分批落地，表結構待 Phase 03 起逐階段定稿。
- 每個 migration 檔頭須註明：狀態（草案／待驗證）、依據（ERD 段落或階段文件）、是否破壞性。

## 3. 破壞性 migration

- 破壞性變更（刪表、刪欄位、改欄位型別、改主鍵、加 NOT NULL 於既有含資料欄位、改唯一約束等）須在檔頭以「破壞性」標示，並註明：備份方式、還原驗證步驟、回滾條件。未經備份與還原驗證的破壞性 migration 不得套用至 staging／production（見 `ROADMAP.md` §4.3 回滾流程，待定稿）。
- 不可回滾的 migration 須先有備份與還原驗證，並在檔頭明載「不可回滾」與替代方案（例如以新 migration 補償），不得以 down.sql 假裝可回滾。

## 4. 基線（000001_baseline）

- `000001_baseline` 只建 `schema_meta` 佔位表（含 `version` 欄），記錄基線版本；不建任何 §20 業務表。理由見該 up.sql 的檔頭註解。
- `schema_meta` 不是業務表，不得被當作系統紀錄查詢；migration 工具自帶版本表的去留，待工具接線時決定。

## 5. 工具選用（M2b 定稿，2026-11-25）

- 採用專案內 migrator，不引入外部 migration 工具：`services/api/internal/postgres`
  的 `MigrateUp`／`MigrateDownOne`／`MigrateDownTo`／`Status`，migration
  檔案取自 `services/api/migrations` 目錄的內嵌檔案（`migrations.FS`），
  不接受外部傳入的檔案系統，避免套用內容與程式碼版本脫節。
- 版本表：`schema_migrations(version text PRIMARY KEY, applied_at timestamptz)`。
  000001 建立的 `schema_meta` 佔位表予以保留（本節 §4 的理由不變），
  版本追蹤一律以 `schema_migrations` 為準，兩者不得混淆。
- 交易語義：每個 migration 在各自的交易內套用，版本記帳寫入同一交易；
  migration SQL 須為交易安全（不得含 `CREATE INDEX CONCURRENTLY`、
  `VACUUM` 等不得在交易內執行的陳述）。
- 併發：以 session 級 advisory lock（固定鍵）序列化同一資料庫的套用。
- 執行主體：`services/api/cmd/migrate`（`up`／`down`／`down-to <版本>`／
  `status`），DSN 一律取 `DATABASE_URL` 環境變數，不得印出 DSN。
  api 服務啟動時不自動套用 migration（明確的操作步驟，避免共用
  資料庫被意外變更）；套用狀態以 `status` 核對。
- Dockerfile：api 映像須複製 `migrations/`（內嵌用）與真實 `go.sum`，
  並建置 `migrate` 執行檔（見 `services/api/Dockerfile`，其檔頭註解
  已預先授權「新增外部依賴時補回 COPY」的調整）。
- 驗證狀態：migrator 與 migration 靜態驗證已於本機 `go test` 通過；
  真實 PostgreSQL 套用／回滾測試為 `integration` build tag，需隔離
  資料庫（`ETHER_TEST_DATABASE_URL`），本機無資料庫，標示「環境待驗證」。
- Seed 資料：本階段無 seed；seed 的有無、內容（須為虛構資料，不得含真實使用者資料）、套用時機，待後階段定稿，不得以 seed 冒充真實資料。

## 6. 缺口

1. 業務表 64 張的欄位級定稿待 Phase 03 起逐階段進行（ERD 草案欄位須逐欄覆核）。
2. Outbox 表名與欄位未定（`DATA_FLOW.md` §11 缺口 1）：Outbox 表須由 ERD 定案後以 migration 落地，不得在本目錄預寫。
3. Migration 工具、執行主體、CI 整合方式未定（見 §5）。
4. 保留期限、匿名化方式待保留政策決定（ERD.md §3），不得在 migration 中預填天數。
