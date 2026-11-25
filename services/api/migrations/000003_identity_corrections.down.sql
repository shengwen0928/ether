-- 000003_identity_corrections.down.sql — 回滾 000003_identity_corrections。
--
-- 狀態：草案，待隔離環境驗證。
-- 警告：回滾將刪除 users 表的 keycloak_subject 與 email_verified 欄位
-- 及其全部資料（身分綁定鍵與驗證狀態），僅限開發資料庫；共用／正式
-- 環境回滾前須備份並經審查（MIGRATIONS.md §3）。

ALTER TABLE users DROP COLUMN IF EXISTS keycloak_subject;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
