-- 000013_calls.down.sql — 撤銷 000013（僅刪除本版本新增的三表）。
-- 注意：通話紀錄與事件為稽核紀錄；down 僅供隔離環境重建使用，
-- 不得對含真實資料的資料庫執行。

DROP TABLE IF EXISTS call_events;
DROP TABLE IF EXISTS call_participants;
DROP TABLE IF EXISTS calls;
