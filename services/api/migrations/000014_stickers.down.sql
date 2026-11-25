-- 000014_stickers.down.sql — 回滾貼圖三表（Phase 08）。
--
-- 僅刪除本版本新增的三張表；依賴順序由外而內：user_stickers、
-- stickers、sticker_packs。既有表不受影響。

DROP TABLE IF EXISTS user_stickers;
DROP TABLE IF EXISTS stickers;
DROP TABLE IF EXISTS sticker_packs;
