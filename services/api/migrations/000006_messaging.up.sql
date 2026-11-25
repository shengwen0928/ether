-- 000006_messaging.up.sql — 對話與訊息五表（Phase 05，離線可驗證部分）。
--
-- 狀態：草案，待有 PostgreSQL 的隔離環境驗證。本機 VM 無 psql／docker，
-- 本檔案未經資料庫套用驗證，不得宣稱已套用、不得宣稱欄位已定稿。
-- 依據：docs/database/ERD.md §6.3 草案欄位（欄位名、型別、約束皆為草案，
-- 待 Migration 定稿後 ERD 再同步修訂）；模組邊界見
-- docs/architecture/MODULE_BOUNDARIES.md §5.6（Chat）與 §5.7（Message）。
-- 是否破壞性：否（僅新增表，不動既有結構）。
--
-- 與 ERD 草案的差異與 Phase 05 決定（皆有理由，待定稿覆核）：
--   1. 只建五表：conversations、conversation_members、messages、
--      message_receipts、message_replies。conversation_settings、
--      message_reactions、message_attachments、message_forwards、
--      message_pins 屬後續里程碑（設定／反應／附件／轉傳／置頂），
--      本里程碑不建表、不寫欄位，不得以缺表假裝功能已實作。
--   2. conversations 不建 encryption_mode：E2EE 啟用範圍與值域待
--      E2EE 文件定稿（E2EE_RESEARCH.md）；本里程碑不分 E2EE／一般
--      聊天室，content 一律視為不透明字串，不得以此宣稱 E2EE。
--      亦不建 avatar_media_id（頭像屬 Media／Profile 後續整合）。
--      last_message_id／last_message_at 為快取欄位，由 Message 寫入
--      路徑經 Chat 介面更新（不得由 Message 直寫 conversations 表，
--      見 MODULE_BOUNDARIES.md §5.7 禁止事項；本檔案只定義欄位）。
--   3. conversation_members 增 last_read_sequence_number（BIGINT，
--      預設 0）：ERD 只有 last_read_message_id；已讀進度不得倒退，
--      須以序號比較，僅憑訊息識別碼無法比大小，故補此欄，與
--      last_read_message_id／last_read_at 三者一致寫入。role 值域
--      為 Phase 05 實作詞彙 member／admin（text，不加 CHECK）。
--      (conversation_id, user_id) 唯一：離開為狀態標記（left_at），
--      重新加入復用同一列（joined_at 更新、left_at 清空），離開／
--      重入歷史不另存列，待定稿覆核。
--   4. messages.client_message_id 唯一性以部分唯一索引保證
--      （同一對話＋同一傳送者＋同一去重鍵僅一列，NULL 不參予）；
--      (conversation_id, sequence_number) 唯一索引保證序號不重複
--      （併發寫入衝突時儲存層以重試收斂，見 internal/message）。
--      status／kind 值域為 Phase 05 實作詞彙（text，不加 CHECK）：
--      status 僅 sent／recalled（伺服器端狀態；Pending／Sending 等
--      為客戶端狀態，不入庫）；kind 本里程碑僅 text。
--      reply_to 關係以 message_replies 表承載（ERD §6.3.8 備註稱
--      欄位或表待定稿；本檔案以表承載，messages 不另設 reply 欄位，
--      不得兩處各存一份而無主從關係）。
--   5. message_receipts 以 (message_id, user_id, receipt_type) 唯一：
--      同一使用者對同一訊息的同一種回條僅一列（首次寫入為準）；
--      device_id 僅記錄首次寫入的裝置，多裝置送達不另存列（已知
--      限制，見 STATUS.md）。receipt_type 值域 delivered／read。
--   6. FK 動作一律 ON DELETE RESTRICT（關係／稽核紀錄不得隨帳號
--      物理刪除，ERD §8 原則草案）；ON UPDATE CASCADE。
--      對 messages 的 FK（conversations.last_message_id、
--      conversation_members.last_read_message_id）因建表順序以
--      ALTER TABLE 後加，不得在建表時前向參照尚未建立的表。
--   7. gen_random_uuid() 為 PostgreSQL 13+ 內建函式，compose 的
--      pgvector:0.8.7-pg16 映像可用，無須額外擴充。

CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL,
    title TEXT,
    created_by UUID REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    last_message_id UUID,
    last_message_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE conversation_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    role TEXT NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at TIMESTAMPTZ,
    last_read_message_id UUID,
    last_read_sequence_number BIGINT NOT NULL DEFAULT 0,
    last_read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (conversation_id, user_id)
);

CREATE INDEX conversation_members_user_idx ON conversation_members (user_id, updated_at DESC);

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    sender_id UUID REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    kind TEXT NOT NULL,
    content TEXT NOT NULL,
    status TEXT NOT NULL,
    client_message_id TEXT,
    sequence_number BIGINT NOT NULL,
    edited_at TIMESTAMPTZ,
    original_content TEXT,
    recalled_at TIMESTAMPTZ,
    deleted_for_sender_at TIMESTAMPTZ,
    edit_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (conversation_id, sequence_number)
);

CREATE UNIQUE INDEX messages_client_message_unique_idx
    ON messages (conversation_id, sender_id, client_message_id)
    WHERE client_message_id IS NOT NULL;

CREATE INDEX messages_conversation_seq_idx ON messages (conversation_id, sequence_number DESC);

CREATE TABLE message_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    device_id UUID REFERENCES user_devices (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    receipt_type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (message_id, user_id, receipt_type)
);

CREATE TABLE message_replies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    replied_to_message_id UUID NOT NULL REFERENCES messages (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (message_id)
);

-- 後加對 messages 的 FK（建表時 messages 尚未建立，不得前向參照）。
ALTER TABLE conversations
    ADD CONSTRAINT conversations_last_message_fk
    FOREIGN KEY (last_message_id) REFERENCES messages (id)
    ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE conversation_members
    ADD CONSTRAINT conversation_members_last_read_fk
    FOREIGN KEY (last_read_message_id) REFERENCES messages (id)
    ON DELETE RESTRICT ON UPDATE CASCADE;
