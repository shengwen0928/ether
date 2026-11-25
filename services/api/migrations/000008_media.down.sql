-- 000008_media.down.sql — 回滾 000008。
--
-- 狀態：草案，待隔離環境驗證。
-- 警告：回滾將刪除全部媒體檔案詮釋資料、上傳會話、縮圖與存取紀錄；
-- 僅限開發資料庫，共用／正式環境回滾前須備份並經審查（MIGRATIONS.md §3）。
-- 物件儲存內的實體檔案不在本回滾範圍，須另行清理（孤兒清理流程待定稿）。

DROP TABLE IF EXISTS media_access_logs;
DROP TABLE IF EXISTS media_thumbnails;
DROP TABLE IF EXISTS media_uploads;
DROP TABLE IF EXISTS media_files;
