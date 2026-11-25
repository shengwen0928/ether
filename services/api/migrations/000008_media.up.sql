-- 000008_media.up.sql — 媒體四表（Phase 06，離線可驗證部分）。
--
-- 狀態：草案，待有 PostgreSQL 的隔離環境驗證。本機 VM 無 psql／docker，
-- 本檔案未經資料庫套用驗證，不得宣稱已套用、不得宣稱欄位已定稿。
-- 依據：docs/database/ERD.md §6.6 草案欄位（欄位名、型別、約束皆為草案，
-- 待 Migration 定稿後 ERD 再同步修訂）；模組邊界見
-- docs/architecture/MODULE_BOUNDARIES.md §5.10（Media）。
-- 是否破壞性：否（僅新增表，不動既有結構）。
--
-- 與 ERD 草案的差異與 Phase 06 決定（皆有理由，待定稿覆核）：
--   1. media_uploads 加 storage_key（TEXT NOT NULL）與 reserved_file_id
--      （UUID NOT NULL）：ERD 草案無此二欄；上傳會話須記住建立時核發
--      的物件鍵與預留檔案識別碼，完成時據此建立檔案列，否則鍵與檔案
--      無法對應。storage_key 唯一（會話與檔案共用同一鍵，檔案列建
--      立後兩表鍵值相同，跨表唯一不強制，僅各表內唯一）。
--   2. width／height／duration_seconds 為 NOT NULL DEFAULT 0（0 表示
--      未知）：ERD 草案稱可為空、型別未定；本里程碑以整數承載
--      （duration_seconds 取整數秒），0 承載「未知」語義，避免可空
--      整數的掃描與對映複雜度，待定稿覆核。size_bytes／uploaded_bytes
--      ／total_bytes 為 BIGINT NOT NULL；total_bytes 為 0 表示總量未知
--      （ERD 草案稱可為空，語義相同，承載方式不同）。
--   3. scan_status／status／access_type／access_result／variant_type
--      一律 text，不加 CHECK：值域為本里程碑實作詞彙（pending／
--      clean／rejected；uploading／completed／failed／cancelled／
--      expired；download／preview／thumbnail；allowed／denied；
--      thumbnail／preview／compressed），待定稿覆核。
--   4. media_thumbnails 僅 created_at（ERD 草案相同，append-only 方向）；
--      (media_file_id, variant_type) 唯一。FK 動作取 ON DELETE
--      RESTRICT（ERD 草案稱隨原檔 CASCADE，本檔案依本目錄慣例取
--      RESTRICT：原檔刪除時縮圖不得殘留為可存取檔案，由服務層在軟
--      刪除後阻擋分發，物理清理待保留期流程定稿）；ON UPDATE CASCADE。
--   5. media_access_logs 僅 created_at，append-only，不得修改；FK 動作
--      一律 ON DELETE RESTRICT（稽核紀錄不得隨帳號物理刪除，ERD §8
--      原則草案）；ON UPDATE CASCADE。ip_address_hash／
--      user_agent_hash 欄位保留但本里程碑不寫入（雜湊方式待隱私
--      文件定稿，不得存原始值）。
--   6. 其餘 FK 動作一律 ON DELETE RESTRICT（關係／稽核紀錄不得隨帳號
--      物理刪除，ERD §8 原則草案）；ON UPDATE CASCADE。
--   7. gen_random_uuid() 為 PostgreSQL 13+ 內建函式，compose 的
--      pgvector:0.8.7-pg16 映像可用，無須額外擴充。
--   8. 掃描供應商／方式待定稿（ROADMAP 手冊 H 阻塞項）：scan_status
--      初始 pending，掃描結果由後續掃描器經服務層寫入，本檔案只建
--      狀態欄位，不預設掃描流程。

CREATE TABLE media_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    storage_key TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    original_filename TEXT,
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    duration_seconds BIGINT NOT NULL DEFAULT 0,
    checksum TEXT,
    scan_status TEXT NOT NULL,
    scan_completed_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (storage_key)
);

CREATE INDEX media_files_owner_idx ON media_files (owner_id, created_at DESC);
CREATE INDEX media_files_checksum_idx ON media_files (checksum);

CREATE TABLE media_uploads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    media_file_id UUID REFERENCES media_files (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    uploader_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    storage_key TEXT NOT NULL,
    reserved_file_id UUID NOT NULL,
    upload_id_external TEXT,
    status TEXT NOT NULL,
    uploaded_bytes BIGINT NOT NULL DEFAULT 0,
    total_bytes BIGINT NOT NULL DEFAULT 0,
    part_count INTEGER NOT NULL DEFAULT 0,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (storage_key)
);

CREATE INDEX media_uploads_uploader_idx ON media_uploads (uploader_id, status);
CREATE INDEX media_uploads_file_idx ON media_uploads (media_file_id, status);
CREATE INDEX media_uploads_stale_idx ON media_uploads (status, updated_at);

CREATE TABLE media_thumbnails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    media_file_id UUID NOT NULL REFERENCES media_files (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    variant_type TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT media_thumbnails_file_variant_unique UNIQUE (media_file_id, variant_type),
    UNIQUE (storage_key)
);

CREATE INDEX media_thumbnails_file_idx ON media_thumbnails (media_file_id);

CREATE TABLE media_access_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    media_file_id UUID NOT NULL REFERENCES media_files (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    accessor_id UUID REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    access_type TEXT NOT NULL,
    access_result TEXT NOT NULL,
    ip_address_hash TEXT,
    user_agent_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX media_access_logs_file_idx ON media_access_logs (media_file_id, created_at DESC);
CREATE INDEX media_access_logs_accessor_idx ON media_access_logs (accessor_id, created_at DESC);
