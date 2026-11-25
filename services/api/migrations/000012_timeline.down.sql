-- 000012_timeline.down.sql — 撤銷動態四表（僅供隔離環境回滾驗證）。
-- 依賴順序反向刪除：先刪子表，再刪 posts。

DROP TABLE IF EXISTS post_comments;
DROP TABLE IF EXISTS post_likes;
DROP TABLE IF EXISTS post_media;
DROP TABLE IF EXISTS posts;
