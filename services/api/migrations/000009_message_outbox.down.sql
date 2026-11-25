-- 000009_message_outbox.down.sql — 回滾 000009（僅刪除本版本新增的表）。
--
-- 注意：回滾將刪除尚未發佈的 Outbox 事件列（待發佈的推播事件
-- 遺失）；套用前須確認 Relay 已排空或接受事件遺失，不得在
-- 有未發佈事件時靜默回滾。

DROP TABLE IF EXISTS message_outbox;
