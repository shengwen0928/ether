-- 000001_baseline.down.sql — 基線 migration 的回滾。
--
-- 警告：本回滾僅在 Phase 03 業務表落地前安全。業務表落地後，
-- 破壞性 migration 的回滾須先有備份與還原驗證（見 MIGRATIONS.md），
-- 不得以本檔案的 DROP 語義回滾已含業務資料的資料庫。
-- 回滾一律只往前修訂：已套用的 migration 不得編輯，修正一律以
-- 新版本 migration 進行（見 MIGRATIONS.md）。

DROP TABLE IF EXISTS schema_meta;
