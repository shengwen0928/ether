-- 000001_baseline.up.sql — 基線 migration（佔位）。
--
-- 狀態：草案，待 CI 或有 PostgreSQL 的環境驗證。本機 VM 無 psql／docker。
-- 本 migration 只建立 migration 記帳用的 schema_meta 佔位表，不得
-- 在此憑空寫 SPEC-SOURCE.md §20 的 64 張業務表；業務表結構待
-- Phase 03 起逐階段落地，每張表以其專屬 migration 定稿（欄位、
-- PK／FK／Unique／索引、建立／更新時間、刪除策略，見 ERD.md）。
--
-- 選擇 schema_meta 而非空基線的理由：空基線在 CI 無法驗證任何
-- 可觀測的結果；schema_meta 提供一個可驗證的落地物，並記錄基線
-- 版本，且不冒充任何業務結構。schema_meta 不是 §20 的業務表，
-- 不得與業務表混淆；若日後選用的 migration 工具自帶版本表
-- （例如 golang-migrate 的 schema_migrations），schema_meta 的去留
-- 由該工具的接線決定另提 migration 處理，不得擅自刪除本檔案。
--
-- 命名與通用慣例（ERD.md §4，草案）：表名 snake_case；主鍵一律
-- 名為 id；外鍵欄位名為 <被參照表單數>_id；時間欄位 timestamptz；
-- 不得存密碼／Token／金鑰／私鑰明文，不得存信用卡完整資訊。

CREATE TABLE IF NOT EXISTS schema_meta (
    id         integer PRIMARY KEY DEFAULT 1,
    version    text NOT NULL,
    applied_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT schema_meta_single_row CHECK (id = 1)
);

INSERT INTO schema_meta (id, version)
VALUES (1, '000001')
ON CONFLICT (id) DO NOTHING;
