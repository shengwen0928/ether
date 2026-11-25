-- 000004_session_refresh_hashes.down.sql — 回滾 000004。
--
-- 狀態：草案，待隔離環境驗證。
-- 警告：回滾將刪除全部已消耗雜湊紀錄，重放偵測將退回「行程內記憶體」
-- 狀態（重啟後失效）；僅限開發資料庫，共用／正式環境回滾前須備份
-- 並經審查（MIGRATIONS.md §3）。

DROP TABLE IF EXISTS session_refresh_hashes;
