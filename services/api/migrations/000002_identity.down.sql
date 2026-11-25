-- 000002_identity.down.sql — 回滾 000002_identity。
--
-- 狀態：草案，待 CI 或有 PostgreSQL 的環境驗證。
-- 僅刪除 000002 新增的六張表；依 FK 依賴反序刪除。
-- 警告：回滾將遺失身分資料，僅限開發資料庫；共用／正式環境
-- 回滾前須備份並經審查（MIGRATIONS.md §3）。

DROP TABLE IF EXISTS user_security_keys;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS user_devices;
DROP TABLE IF EXISTS user_settings;
DROP TABLE IF EXISTS user_profiles;
DROP TABLE IF EXISTS users;
