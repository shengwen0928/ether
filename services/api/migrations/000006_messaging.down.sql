-- 000006_messaging.down.sql — 回滾 000006（僅刪除本版本新增的表）。
--
-- 依賴順序的反向刪除：先刪參照 messages 的表，再刪 messages，
-- 再刪 conversation_members，最後刪 conversations。

DROP TABLE IF EXISTS message_replies;
DROP TABLE IF EXISTS message_receipts;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversation_members;
DROP TABLE IF EXISTS conversations;
