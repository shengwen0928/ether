-- 000007_groups.down.sql — 回滾 000007。
--
-- 狀態：草案，待隔離環境驗證。
-- 警告：回滾將刪除全部群組、成員、邀請、公告與設定紀錄；僅限開發
-- 資料庫，共用／正式環境回滾前須備份並經審查（MIGRATIONS.md §3）。

DROP TABLE IF EXISTS group_settings;
DROP TABLE IF EXISTS group_announcements;
DROP TABLE IF EXISTS group_invitations;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS groups;
