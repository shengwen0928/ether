-- 000005_friends.up.sql — 好友關係四表（Phase 04）。
--
-- 狀態：草案，待有 PostgreSQL 的隔離環境驗證。本機 VM 無 psql／docker，
-- 本檔案未經資料庫套用驗證，不得宣稱已套用、不得宣稱欄位已定稿。
-- 依據：docs/database/ERD.md §6.2 草案欄位（欄位名、型別、約束皆為草案，
-- 待 Migration 定稿後 ERD 再同步修訂）；好友語義見
-- docs/architecture/MODULE_BOUNDARIES.md §5.4（封鎖／刪除／隱藏分開建模）。
-- 是否破壞性：否（僅新增表，不動既有結構）。
--
-- 與 ERD 草案的差異與 M3 決定（皆有理由，待定稿覆核）：
--   1. friendships 僅一列存放一對使用者：user_id／friend_id 以字典序
--      存放（user_id < friend_id），以 CHECK 約束保證，不得雙向存兩列；
--      ERD 稱儲存方式待定稿，本檔案以此方式定為 M3 實作，不得以
--      雙列假設查詢本表。status 值域為 M3 實作詞彙 active／ended
--     （text，不加 CHECK，值域待定稿）。
--   2. friend_requests 待處理唯一性以部分唯一索引保證（同一方向同時
--      僅一筆 status='pending'）；雙向同時待處理的合併規則待定稿，
--      M3 實作以後送出者收到重複錯誤為準（服務層預檢）。status 值域
--      為 M3 實作詞彙 pending／accepted／declined／cancelled／expired
--      （text，不加 CHECK）。expires_at 可為空；到期自動轉態須由
--      排程／Worker 落實，本階段不自動過期。
--   3. blocked_users／hidden_users 的 blocked_at／hidden_at 與
--      created_at 在建立時取同一時間（ERD 註記二者語義重複待定稿；
--      M3 保留兩欄，值相等，定稿時再決定是否合併）。再次封鎖／隱藏
--      建新列（歷史保留），生效唯一性由服務層預檢保證，不加部分
--      唯一索引（歷史列與生效列共存，約束形態待定稿）。
--   4. FK 動作一律 ON DELETE RESTRICT（關係／稽核紀錄不得隨帳號物理
--      刪除，ERD §8 原則草案）；ON UPDATE CASCADE。
--   5. gen_random_uuid() 為 PostgreSQL 13+ 內建函式，compose 的
--      pgvector:0.8.7-pg16 映像可用，無須額外擴充。
--
-- 不得將密碼明文、Token 明文、私鑰寫入任何資料表（ERD §4）。

CREATE TABLE IF NOT EXISTS friendships (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    friend_id      uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    status         text NOT NULL DEFAULT 'active',
    established_at timestamptz,
    ended_at       timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT friendships_ordered_pair CHECK (user_id < friend_id),
    CONSTRAINT friendships_pair_unique UNIQUE (user_id, friend_id)
);

CREATE INDEX IF NOT EXISTS friendships_user_status_idx ON friendships (user_id, status);
CREATE INDEX IF NOT EXISTS friendships_friend_status_idx ON friendships (friend_id, status);

CREATE TABLE IF NOT EXISTS friend_requests (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    addressee_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    status       text NOT NULL DEFAULT 'pending',
    message      text,
    expires_at   timestamptz,
    responded_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT friend_requests_no_self CHECK (requester_id <> addressee_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS friend_requests_pending_unique_idx
    ON friend_requests (requester_id, addressee_id) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS friend_requests_addressee_status_idx ON friend_requests (addressee_id, status);
CREATE INDEX IF NOT EXISTS friend_requests_requester_status_idx ON friend_requests (requester_id, status);
CREATE INDEX IF NOT EXISTS friend_requests_expires_idx ON friend_requests (expires_at);

CREATE TABLE IF NOT EXISTS blocked_users (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    blocked_user_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    reason          text,
    blocked_at      timestamptz NOT NULL DEFAULT now(),
    unblocked_at    timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT blocked_users_no_self CHECK (user_id <> blocked_user_id)
);

CREATE INDEX IF NOT EXISTS blocked_users_pair_idx ON blocked_users (user_id, blocked_user_id);
CREATE INDEX IF NOT EXISTS blocked_users_blocked_idx ON blocked_users (blocked_user_id);

CREATE TABLE IF NOT EXISTS hidden_users (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    hidden_user_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    hidden_at      timestamptz NOT NULL DEFAULT now(),
    unhidden_at    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT hidden_users_no_self CHECK (user_id <> hidden_user_id)
);

CREATE INDEX IF NOT EXISTS hidden_users_pair_idx ON hidden_users (user_id, hidden_user_id);
