-- 000005_friends.down.sql — 回滾 000005。
--
-- 狀態：草案，待隔離環境驗證。
-- 警告：回滾將刪除全部好友關係、邀請、封鎖與隱藏紀錄；僅限開發
-- 資料庫，共用／正式環境回滾前須備份並經審查（MIGRATIONS.md §3）。

DROP TABLE IF EXISTS hidden_users;
DROP TABLE IF EXISTS blocked_users;
DROP TABLE IF EXISTS friend_requests;
DROP TABLE IF EXISTS friendships;
