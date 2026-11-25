-- 000009_message_outbox.up.sql — 訊息 Outbox 事件表（訊息推播中繼，離線可驗證部分）。
--
-- 狀態：草案，待有 PostgreSQL 的隔離環境驗證。本機 VM 無 psql／docker，
-- 本檔案未經資料庫套用驗證，不得宣稱已套用、不得宣稱欄位已定稿。
-- 依據：docs/architecture/DATA_FLOW.md §3（業務寫入與 Outbox 事件寫入
-- 必須在同一 PostgreSQL 交易，提交後才可發布；發布為「至少一次」）與
-- §10 缺口 1（Outbox 表名與欄位未定，ERD 文件須定案）；模組邊界見
-- docs/architecture/MODULE_BOUNDARIES.md §4（一表一主）與 §5.7（Message）。
-- 是否破壞性：否（僅新增表，不動既有結構）。
--
-- 與 ERD 草案的差異與本里程碑決定（皆有理由，待定稿覆核）：
--   1. 只建一表 message_outbox：SPEC-SOURCE.md §20 無 Outbox 表，
--      DATA_FLOW.md §10 缺口 1 明載表名待 ERD 定案；本表由 Message
--      模組擁有（訊息事件專用），他模組日後如需 Outbox 須另建所屬
--      表或另提共用表 ADR，不得擅自寫入本表。
--   2. status 值域為本里程碑實作詞彙 pending／processing／published／
--      dead（text，不加 CHECK）：processing 加 claimed_at 租約，認領
--      逾期的列可被再次認領；dead 為死信終態，須人工介入。
--   3. payload 為 JSONB：事件 data 物件（WebSocket 信封的 data 欄位
--      內容），只含已公開的 wire 欄位與識別碼，不得含 Token、雜湊
--      或 E2EE 明文（見 internal/outbox 檔頭）。
--   4. conversation_id／message_id FK 一律 ON DELETE RESTRICT
--      （關係／稽核紀錄不得隨帳號物理刪除，ERD §8 原則草案）；
--      ON UPDATE CASCADE。
--   5. gen_random_uuid() 為 PostgreSQL 13+ 內建函式，compose 的
--      pgvector:0.8.7-pg16 映像可用，無須額外擴充。

CREATE TABLE message_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_name TEXT NOT NULL,
    topic TEXT NOT NULL,
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    message_id UUID NOT NULL REFERENCES messages (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    claimed_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX message_outbox_claim_idx
    ON message_outbox (status, created_at);
