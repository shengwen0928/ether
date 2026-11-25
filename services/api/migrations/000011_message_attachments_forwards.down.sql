-- 000011_message_attachments_forwards.down.sql — 回滾 000011（僅刪除本版本新增的表）。
--
-- 注意：回滾將刪除附件關聯與轉傳紀錄；套用前須確認無須保留，
-- 不得在須保留稽核歷史時靜默回滾。

DROP TABLE IF EXISTS message_forwards;
DROP TABLE IF EXISTS message_attachments;
