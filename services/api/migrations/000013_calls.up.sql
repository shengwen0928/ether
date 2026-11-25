-- 000013_calls.up.sql — 通話三表（Phase 07，離線可驗證部分）。
--
-- 狀態：草案，待有 PostgreSQL 的隔離環境驗證。本機 VM 無 psql／docker，
-- 本檔案未經資料庫套用驗證，不得宣稱已套用、不得宣稱欄位已定稿。
-- 依據：docs/database/ERD.md §6.9 草案欄位（欄位名、型別、約束皆為草案，
-- 待 Migration 定稿後 ERD 再同步修訂）；模組邊界見
-- docs/architecture/MODULE_BOUNDARIES.md §5.12（Call）。
-- 是否破壞性：否（僅新增表，不動既有結構）。
--
-- 與 ERD 草案的差異與本里程碑決定（皆有理由，待定稿覆核）：
--   1. 只建三表：calls、call_participants、call_events。
--   2. calls 不建 livekit_room_ref：LiveKit 房間參照形態待 Phase 07
--      定稿，且本里程碑不建立房間、不簽發 Token；該欄屬後續
--      migration，不得預寫為已定。
--   3. call_participants 不建 device_id：裝置／推播資訊經 Notification
--      事件傳遞，Call 不得直連 push_tokens；裝置粒度待定稿覆核。
--   4. call_type／status／result／event_type／end_reason 為 text，不加
--      CHECK：值域為本里程碑實作詞彙（voice／video；
--      ringing／active／declined／missed／busy／ended；
--      pending／joined／declined／busy／missed；
--      initiated／answered／declined／busy／ended；
--      hangup／declined／missed／busy），待端點 Schema 定稿覆核。
--      可見性篩選與狀態轉移須與服務層判定併用，不得僅靠索引假設。
--   5. call_participants 以 (call_id, user_id) 全唯一約束承載「每通話
--      每使用者恰好一列」（ERD 草案 (call_id, user_id, joined_at)
--      為多次加入／離開各記一筆的方向參考；本里程碑通話結束為終端
--      狀態、不建重新加入，單列＋joined_at／left_at 已足，待定稿覆核）。
--   6. call_events.event_payload 為 TEXT（沿本目錄慣例，JSON 文字只含
--      狀態轉移資料 from_status／to_status／reason）；不得存 SDP／ICE、
--      Token、金鑰材料或通話內容。occurred_at 為事件實際發生時間。
--   7. calls.started_at 可為空（首次接聽前為空，不得以建立時間代替）；
--      duration_seconds 可為空（僅有接聽期間的結束通話才有值）。
--   8. FK 動作一律 ON DELETE RESTRICT（關係／稽核紀錄不得隨帳號
--      物理刪除，ERD §8 原則草案）；ON UPDATE CASCADE。
--   9. gen_random_uuid() 為 PostgreSQL 13+ 內建函式，compose 的
--      pgvector:0.8.7-pg16 映像可用，無須額外擴充。

CREATE TABLE calls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID REFERENCES conversations (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    group_id UUID REFERENCES groups (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    initiator_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    call_type TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    duration_seconds BIGINT,
    ended_by UUID REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    end_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT calls_scope_check CHECK (
        (conversation_id IS NOT NULL AND group_id IS NULL) OR
        (conversation_id IS NULL AND group_id IS NOT NULL)
    )
);

CREATE INDEX calls_initiator_idx ON calls (initiator_id, started_at);
CREATE INDEX calls_conversation_idx ON calls (conversation_id, created_at DESC);
CREATE INDEX calls_group_idx ON calls (group_id, created_at DESC);
CREATE INDEX calls_active_scope_idx ON calls (status, conversation_id, group_id);

CREATE TABLE call_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_id UUID NOT NULL REFERENCES calls (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    joined_at TIMESTAMPTZ,
    left_at TIMESTAMPTZ,
    result TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT call_participants_call_user_unique UNIQUE (call_id, user_id)
);

CREATE INDEX call_participants_call_idx ON call_participants (call_id, created_at);
CREATE INDEX call_participants_user_idx ON call_participants (user_id, created_at DESC);

CREATE TABLE call_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_id UUID NOT NULL REFERENCES calls (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    user_id UUID REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    event_type TEXT NOT NULL,
    event_payload TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX call_events_call_idx ON call_events (call_id, occurred_at);
