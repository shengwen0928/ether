-- 000010_message_reactions_pins.up.sql — 訊息反應與置頂兩表（Phase 05 後續，離線可驗證部分）。
--
-- 狀態：草案，待有 PostgreSQL 的隔離環境驗證。本機 VM 無 psql／docker，
-- 本檔案未經資料庫套用驗證，不得宣稱已套用、不得宣稱欄位已定稿。
-- 依據：docs/database/ERD.md §6.3.6（message_reactions）與 §6.3.10
-- （message_pins）草案欄位；000006 檔頭已載明此二表屬後續里程碑，
-- 本版本補建。模組邊界見 docs/architecture/MODULE_BOUNDARIES.md §5.7
-- （Message 擁有此二表）。
-- 是否破壞性：否（僅新增表，不動既有結構）。
--
-- 與 ERD 草案的差異與本里程碑決定（皆有理由，待定稿覆核）：
--   1. 反應值域：ERD 稱值域待定稿；本里程碑以服務層允許清單
--      實作（8 種表情，見 internal/message/reactions.go），資料庫
--      不加 CHECK（沿全專案 text 值域慣例）。
--   2. 每人每訊息至多一則生效反應：更換反應時舊列以 removed_at
--      標記（非物理刪除，ERD 取消標記方向），新列另行寫入；
--      生效唯一性以部分唯一索引保證
--      （(message_id, user_id, reaction) WHERE removed_at IS NULL）。
--      計數一律由本表彙總，不得另存計數快取（ERD §6.3.6）。
--   3. created_at／updated_at 皆建（ERD 稱二者關係待定稿）；
--      removed_at／unpinned_at 皆保留（取消為標記，不得物理刪除
--      歷史，ERD §6.3.6／§6.3.10 刪除策略）。
--   4. 置頂生效唯一性以部分唯一索引保證
--      （(conversation_id, message_id) WHERE unpinned_at IS NULL）；
--      同時置頂數上限由服務層 enforced（實作預設 5 則，待定稿），
--      資料庫不設約束。
--   5. FK 動作一律 ON DELETE RESTRICT（關係／稽核紀錄不得隨帳號
--      物理刪除，ERD §8 原則草案）；ON UPDATE CASCADE。
--   6. gen_random_uuid() 為 PostgreSQL 13+ 內建函式，compose 的
--      pgvector:0.8.7-pg16 映像可用，無須額外擴充。

CREATE TABLE message_reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    reaction TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    removed_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX message_reactions_active_unique_idx
    ON message_reactions (message_id, user_id, reaction)
    WHERE removed_at IS NULL;

CREATE INDEX message_reactions_message_idx ON message_reactions (message_id, reaction);

CREATE TABLE message_pins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    message_id UUID NOT NULL REFERENCES messages (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    pinned_by UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    pinned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    unpinned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX message_pins_active_unique_idx
    ON message_pins (conversation_id, message_id)
    WHERE unpinned_at IS NULL;

CREATE INDEX message_pins_conversation_idx ON message_pins (conversation_id, pinned_at);
