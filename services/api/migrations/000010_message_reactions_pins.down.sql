-- 000010_message_reactions_pins.down.sql — 回滾 000010（僅刪除本版本新增的表）。
--
-- 注意：回滾將刪除反應與置頂歷史列；套用前須確認無須保留，
-- 不得在須保留稽核歷史時靜默回滾。

DROP TABLE IF EXISTS message_pins;
DROP TABLE IF EXISTS message_reactions;
