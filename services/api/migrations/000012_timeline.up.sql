-- 000012_timeline.up.sql — 動態四表（Phase 08，離線可驗證部分）。
--
-- 狀態：草案，待有 PostgreSQL 的隔離環境驗證。本機 VM 無 psql／docker，
-- 本檔案未經資料庫套用驗證，不得宣稱已套用、不得宣稱欄位已定稿。
-- 依據：docs/database/ERD.md §6.8 草案欄位（欄位名、型別、約束皆為草案，
-- 待 Migration 定稿後 ERD 再同步修訂）；模組邊界見
-- docs/architecture/MODULE_BOUNDARIES.md §5.17（Timeline）。
-- 是否破壞性：否（僅新增表，不動既有結構）。
--
-- 與 ERD 草案的差異與本里程碑決定（皆有理由，待定稿覆核）：
--   1. 只建四表：posts、post_media、post_likes、post_comments。
--      post_shares（分享）與 post_bookmarks（收藏）屬後續里程碑，
--      本里程碑不建表、不提供端點。
--   2. posts.visibility 為 text，不加 CHECK：值域為本里程碑實作詞彙
--      public／friends／only_me（指定好友可見屬後續里程碑，不另建表）。
--      可見性篩選須與權限判定併用，不得僅靠索引假設可見性正確。
--   3. posts.content 為 TEXT NOT NULL DEFAULT ''（ERD 草案為可為空）：
--      純媒體貼文以空字串承載，服務層驗證「內容與媒體不得皆為空」，
--      不以 NULL 承載空內容（沿本目錄可空 text 慣例的反向選擇，
--      讀寫一律以空字串為準，待定稿覆核）。
--   4. post_likes 以 (post_id, user_id) 全唯一約束承載「每人每則限一次」，
--      取消按讚為 unliked_at 標記（ERD 草案稱生效期間唯一性待定稿；
--      本檔案取全唯一＋標記，不另存計數快取，計數一律即時彙總）。
--      liked_at 與 created_at 並存（ERD 草案稱二者關係待定稿）：
--      liked_at 為最近一次按讚時間，created_at 為列建立時間。
--   5. post_comments 含 edited_at（ERD 草案欄位），但留言編輯端點屬
--      後續里程碑，本里程碑不寫入該欄；刪除為 is_deleted＋deleted_at
--      標記，不得物理刪除；父留言刪除後子留言的顯示規則待定稿。
--   6. post_media 只存 media_file_id 直接 FK（沿 M10 附件關聯先例，
--      不存 storage_path）；(post_id, position) 唯一、(post_id,
--      media_file_id) 唯一；caption 欄位保留，寫入時為空（逐媒體說明
--      文字屬後續里程碑）。貼文刪除後媒體關聯保留，不得單獨物理刪除。
--   7. FK 動作一律 ON DELETE RESTRICT（關係／稽核紀錄不得隨帳號
--      物理刪除，ERD §8 原則草案）；ON UPDATE CASCADE。
--   8. gen_random_uuid() 為 PostgreSQL 13+ 內建函式，compose 的
--      pgvector:0.8.7-pg16 映像可用，無須額外擴充。

CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    content TEXT NOT NULL DEFAULT '',
    visibility TEXT NOT NULL,
    edited_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX posts_author_idx ON posts (author_id, created_at DESC);
CREATE INDEX posts_visibility_idx ON posts (visibility, created_at DESC);

CREATE TABLE post_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    media_file_id UUID NOT NULL REFERENCES media_files (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    position INTEGER NOT NULL,
    caption TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT post_media_post_position_unique UNIQUE (post_id, position),
    CONSTRAINT post_media_post_file_unique UNIQUE (post_id, media_file_id)
);

CREATE INDEX post_media_post_idx ON post_media (post_id, position);

CREATE TABLE post_likes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    liked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    unliked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT post_likes_post_user_unique UNIQUE (post_id, user_id)
);

CREATE INDEX post_likes_post_idx ON post_likes (post_id, liked_at);

CREATE TABLE post_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    author_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    parent_comment_id UUID REFERENCES post_comments (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    content TEXT NOT NULL,
    edited_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX post_comments_post_idx ON post_comments (post_id, created_at);
CREATE INDEX post_comments_parent_idx ON post_comments (parent_comment_id);
