-- 000004_session_refresh_hashes.up.sql — 已輪換 Refresh 雜湊紀錄表（Phase 03 M2b）。
--
-- 狀態：草案，待有 PostgreSQL 的隔離環境驗證。本機 VM 無 psql／docker，
-- 本檔案未經資料庫套用驗證，不得宣稱已套用。
-- 依據：IDENTITY §4 的重放偵測要求「重放已輪換的 Refresh Token 視為
-- 盜用」。M1 的已消耗雜湊只存在 SessionService 的行程內記憶體，重啟
-- 即失；PostgreSQL 持久化後，同一雜湊的重放必須在重啟後仍被偵測。
-- 本表即該持久化紀錄：append-only（只有寫入與查詢，不得 UPDATE／
-- DELETE），保留期限與清理方式待保留政策決定（ERD.md §3 缺口），
-- 不得在本檔案預填保留天數。
-- 是否破壞性：否（僅新增表，不動既有結構）。
-- FK 動作：ON DELETE RESTRICT（Session 紀錄為稽核資料，不得隨刪）。

CREATE TABLE IF NOT EXISTS session_refresh_hashes (
    session_id         uuid NOT NULL REFERENCES user_sessions (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    refresh_token_hash text NOT NULL,
    consumed_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT session_refresh_hashes_pkey PRIMARY KEY (session_id, refresh_token_hash)
);

CREATE INDEX IF NOT EXISTS session_refresh_hashes_session_idx ON session_refresh_hashes (session_id);
