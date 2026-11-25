-- 000002_identity.up.sql — 身分帳號六表（Phase 03）。
--
-- 狀態：草案，待 CI 或有 PostgreSQL 的環境驗證。本機 VM 無 psql／docker，
-- 本檔案未經資料庫套用驗證，不得宣稱已套用、不得宣稱欄位已定稿。
-- 依據：docs/database/ERD.md §6.1 草案欄位（欄位名、型別、約束皆為草案，
-- 待 Migration 定稿後 ERD 再同步修訂）；身分語義見
-- docs/architecture/IDENTITY_AND_PERMISSIONS.md。
-- 是否破壞性：否（僅新增表，不動既有結構）。
--
-- 與 ERD 草案的差異（皆有理由，待定稿覆核）：
--   1. users 不建 password_hash：密碼由 Keycloak 承接（IDENTITY §3.3／
--      ADR-010），ETHER 不得自寫一套身分儲存與 Keycloak 並存。
--   2. user_settings 不建 other_settings：ERD 明言不得以無結構欄位掩蓋
--      未定稿的設定項；media_auto_download 保留為 text（值域待定稿）。
--   3. user_sessions 增加 keycloak_session_id：Keycloak sid 映象，M1 單一
--      登出端點需要，待 ERD 定稿覆核。
--   4. 狀態／值域欄位（status、theme、platform、key_type 等）一律 text
--      不加 CHECK 約束：值域皆待定稿，不得以本檔案預寫列舉值。
--   5. user_security_keys 僅 created_at、無 updated_at：append-only
--      方向（ERD §6.1.6）。
--   6. FK 動作：user_settings ON DELETE CASCADE（ERD 草案明示）；其餘
--      一律 ON DELETE RESTRICT（撤銷／稽核紀錄不得隨帳號物理刪除，
--      ERD §8 原則草案）。
--   7. gen_random_uuid() 為 PostgreSQL 13+ 內建函式，compose 的
--      pgvector:0.8.7-pg16 映像可用，無須額外擴充。
--
-- 不得將密碼明文、Token 明文、Bot 金鑰明文、私鑰、信用卡完整資訊
-- 寫入任何資料表（ERD §4）。Refresh／Session 雜湊僅存雜湊值。

CREATE TABLE IF NOT EXISTS users (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- phone_number 明文／雜湊待安全性架構定稿（ERD §6.1.1）；本階段不寫入。
    phone_number   text UNIQUE,
    email          text UNIQUE,
    status         text NOT NULL DEFAULT 'active',
    registered_at  timestamptz NOT NULL DEFAULT now(),
    last_login_at  timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS users_status_idx ON users (status);

CREATE TABLE IF NOT EXISTS user_profiles (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            uuid NOT NULL UNIQUE REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    display_name       text NOT NULL,
    public_id          text UNIQUE,
    -- avatar_media_id 參照媒體的參照方式待媒體模組定稿（ERD §6.1.2）。
    avatar_media_id    text,
    bio                text,
    status_message     text,
    birthday           date,
    birthday_visibility text,
    qr_code_token      text,
    profile_visibility text,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

-- display_name 搜尋索引為草案（搜尋實作待定稿）；trigram 索引須
-- pg_trgm 擴充，本階段不引入，先以一般 btree 索引佔位。
CREATE INDEX IF NOT EXISTS user_profiles_display_name_idx ON user_profiles (display_name);
CREATE INDEX IF NOT EXISTS user_profiles_public_id_idx ON user_profiles (public_id);

CREATE TABLE IF NOT EXISTS user_settings (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE ON UPDATE CASCADE,
    language        text NOT NULL DEFAULT 'zh-Hant',
    theme           text NOT NULL DEFAULT 'system',
    font_size       text,
    chat_background text,
    -- media_auto_download 值域待定稿（ERD §6.1.3），暫以 text 承載。
    media_auto_download text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_devices (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    device_name            text NOT NULL,
    platform               text NOT NULL,
    -- 僅存雜湊或去識別化識別碼，不得存原始識別碼明文（ERD §6.1.4）。
    device_identifier_hash text NOT NULL,
    last_active_at         timestamptz,
    is_trusted             boolean NOT NULL DEFAULT false,
    is_revoked             boolean NOT NULL DEFAULT false,
    revoked_at             timestamptz,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS user_devices_user_last_active_idx ON user_devices (user_id, last_active_at);
CREATE INDEX IF NOT EXISTS user_devices_user_revoked_idx ON user_devices (user_id, is_revoked);

CREATE TABLE IF NOT EXISTS user_sessions (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    device_id             uuid REFERENCES user_devices (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    -- session_token_hash 為 ERD 草案欄位；本階段未寫入（Session 參照
    -- 由 Keycloak 承接，見 keycloak_session_id）。
    session_token_hash    text UNIQUE,
    -- Keycloak sid 映象（M1 單一登出需要，待 ERD 定稿覆核）。
    keycloak_session_id   text,
    -- 僅存雜湊，不得存 Token 明文（ERD §6.1.5）。
    refresh_token_hash    text NOT NULL UNIQUE,
    expires_at            timestamptz NOT NULL,
    last_active_at        timestamptz,
    is_revoked            boolean NOT NULL DEFAULT false,
    revoked_at            timestamptz,
    revoked_reason        text,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS user_sessions_user_expires_idx ON user_sessions (user_id, expires_at);
CREATE INDEX IF NOT EXISTS user_sessions_user_revoked_idx ON user_sessions (user_id, is_revoked);

CREATE TABLE IF NOT EXISTS user_security_keys (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    device_id  uuid REFERENCES user_devices (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    key_type   text NOT NULL,
    key_id     text NOT NULL,
    -- 僅公開金鑰 material；私鑰一律只存裝置本機（ERD §6.1.6）。
    public_key text NOT NULL,
    signature  text,
    is_used    boolean NOT NULL DEFAULT false,
    used_at    timestamptz,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_security_keys_unique UNIQUE (user_id, device_id, key_type, key_id)
);

CREATE INDEX IF NOT EXISTS user_security_keys_user_type_idx ON user_security_keys (user_id, key_type);
CREATE INDEX IF NOT EXISTS user_security_keys_user_used_idx ON user_security_keys (user_id, is_used);
