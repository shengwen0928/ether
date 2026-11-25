-- 000011_message_attachments_forwards.up.sql — 訊息附件關聯與轉傳兩表（Phase 05 後續，離線可驗證部分）。
--
-- 狀態：草案，待有 PostgreSQL 的隔離環境驗證。本機 VM 無 psql／docker，
-- 本檔案未經資料庫套用驗證，不得宣稱已套用、不得宣稱欄位已定稿。
-- 依據：docs/database/ERD.md §6.3.7（message_attachments）與 §6.3.9
-- （message_forwards）草案欄位；000006 檔頭已載明此二表屬後續里程碑，
-- 本版本補建。模組邊界見 docs/architecture/MODULE_BOUNDARIES.md §5.7
-- （Message 擁有此二表）與 §5.10（Media 擁有 media_files）。
-- 是否破壞性：否（僅新增表，不動既有結構）。
--
-- 與 ERD 草案的差異與本里程碑決定（皆有理由，待定稿覆核）：
--   1. 媒體參照方式：ERD §10 缺口稱直接參照 media_files 或經由儲存
--      路徑二選一待定稿；本里程碑只以 media_file_id 直接 FK 參照，
--      不建 storage_path 備援欄位（不得並存兩套參照而不說明，ERD
--      §6.3.7）。位元組上傳通道未定（待決定 19），本表不存任何可
--      分發的位址或物件鍵。
--   2. thumbnail_media_id 不建：縮圖以 media_thumbnails 經 media
--      模組另查，不得於附件列重複參照衍生檔案（單一事實來源）。
--   3. 附件關聯唯一性：ERD 稱無草案唯一約束；本里程碑以
--      (message_id, media_file_id) 唯一索引保證同一檔案不重複關聯
--      至同一訊息（關聯寫入冪等收斂，見 internal/message）。
--   4. file_name 以 NOT NULL DEFAULT '' 承載（ERD 稱可為空；快照
--      來源 MediaFile 的原始檔名可為空字串，空值與空字串不得並存
--      兩種「未知」語義）。size_bytes 不設預設值（須由 Media 檔案
--      列快照，Media 驗證保證大於 0）。
--   5. 僅建 created_at（ERD §6.3.7／§6.3.9 刪除策略：附件關聯與
--      轉傳紀錄皆 append-only，不得修改、不得單獨物理刪除）。
--   6. FK 動作一律 ON DELETE RESTRICT（關係／稽核紀錄不得隨帳號
--      物理刪除，ERD §8 原則草案）；ON UPDATE CASCADE。
--   7. gen_random_uuid() 為 PostgreSQL 13+ 內建函式，compose 的
--      pgvector:0.8.7-pg16 映像可用，無須額外擴充。

CREATE TABLE message_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    media_file_id UUID NOT NULL REFERENCES media_files (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    file_name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX message_attachments_message_file_unique_idx
    ON message_attachments (message_id, media_file_id);

CREATE INDEX message_attachments_message_idx ON message_attachments (message_id, created_at);

CREATE TABLE message_forwards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_message_id UUID NOT NULL REFERENCES messages (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    forwarded_message_id UUID NOT NULL REFERENCES messages (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    source_conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    target_conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    forwarded_by UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX message_forwards_forwarded_unique_idx
    ON message_forwards (forwarded_message_id);

CREATE INDEX message_forwards_source_idx ON message_forwards (source_message_id);

CREATE INDEX message_forwards_target_idx ON message_forwards (target_conversation_id, created_at);
