-- 000003_identity_corrections.up.sql — users 表的身分欄位補正（Phase 03 M2b）。
--
-- 狀態：草案，待有 PostgreSQL 的隔離環境驗證。本機 VM 無 psql／docker，
-- 本檔案未經資料庫套用驗證，不得宣稱已套用。
-- 依據：M2b 盤點發現 000002 的 users 表缺少領域模型必需的兩個欄位——
-- keycloak_subject（User 領域的穩定綁定鍵，Provision 依 subject 查找）
-- 與 email_verified（Me 端點回傳、帳號驗證狀態）。缺少此二欄位時，
-- PostgreSQL Repository 無法完整落地 M1 領域模型，故以新 migration
-- 補正（只往前原則：000002 已定稿，不得編輯，見 MIGRATIONS.md §2）。
-- 是否破壞性：對全新資料庫否（僅新增欄位）。前提：截至 2026-11-25，
-- 本目錄的 migration 未在任何環境套用於含資料的資料庫（Track A 阻塞，
-- 從未執行）；若某資料庫已含 users 資料列，本 migration 會因 NOT NULL
-- 欄位無值而失敗——此為刻意的安全失敗，不得繞過；須先以人工協調
-- 既有資料列的 keycloak_subject 後再套用。
-- 套用方式：一律經 migrator（internal/postgres）套用，版本記帳保證
-- 單次套用；IF NOT EXISTS 僅防手動重跑誤傷，不得以手動套用代替
-- migrator 的版本追蹤。

ALTER TABLE users ADD COLUMN IF NOT EXISTS keycloak_subject text NOT NULL UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified boolean NOT NULL DEFAULT false;
