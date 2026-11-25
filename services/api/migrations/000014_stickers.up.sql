-- 000014_stickers.up.sql — 貼圖三表（Phase 08，離線可驗證部分）。
--
-- 狀態：草案，待有 PostgreSQL 的隔離環境驗證。本機 VM 無 psql／docker，
-- 本檔案未經資料庫套用驗證，不得宣稱已套用、不得宣稱欄位已定稿。
-- 依據：docs/database/ERD.md §6.7 草案欄位（欄位名、型別、約束皆為草案，
-- 待 Migration 定稿後 ERD 再同步修訂）；模組邊界見
-- docs/architecture/MODULE_BOUNDARIES.md §5.11（Sticker）。
-- 是否破壞性：否（僅新增表，不動既有結構）。
--
-- 與 ERD 草案的差異與本里程碑決定（皆有理由，待定稿覆核）：
--   1. 只建三表：sticker_packs、stickers、user_stickers。
--      sticker_purchases（購買紀錄）歸 Payment Integration 模組，
--      本里程碑不建表、不提供端點（購買交易屬後續里程碑）。
--   2. sticker_packs.status／stickers.status 為 text，不加 CHECK：
--      值域為本里程碑實作詞彙（draft／published／unlisted／
--      archived；active／removed），待端點 Schema 定稿覆核。
--   3. sticker_packs.is_free 為 BOOLEAN NOT NULL DEFAULT true
--      （ERD 草案稱免費／付費區分，價格欄位待商業決定，不得預寫
--      價格）：付費包的可見性以 user_stickers 的授權列為準，
--      fail-closed。
--   4. 不建 author_name 欄（ERD 草案欄位）：作者顯示名稱由 User
--      模組公開檔案解析，不得在本表重複存快照。
--   5. sticker_packs.category 為可空 text，不加 CHECK：值域為本里程碑
--      實作詞彙（general／animals／food／emotions／festivals／
--      characters／other），空值表示未分類。
--   6. stickers 只存 media_file_id 直接 FK（沿 M10 附件關聯先例，
--      不存 storage_path）；(sticker_pack_id, position) 唯一；
--      移除列保留原序號，新貼圖序號接在最大序號之後，不得重用。
--   7. user_stickers 以 CHECK 約束強制「包級與單張級恰居其一」
--      （ERD 草案稱二者主從待定稿，不得並存而不說明；本檔案取
--      恰居其一，包級列承載收藏／授權、單張級列承載最近使用）；
--      包級與單張級各以部分唯一索引承載唯一性；source 為 text
--      不加 CHECK（favorite／granted／recent，待定稿覆核）。
--   8. FK 動作一律 ON DELETE RESTRICT（關係／稽核紀錄不得隨帳號
--      物理刪除，ERD §8 原則草案）；ON UPDATE CASCADE。
--   9. gen_random_uuid() 為 PostgreSQL 13+ 內建函式，compose 的
--      pgvector:0.8.7-pg16 映像可用，無須額外擴充。

CREATE TABLE sticker_packs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT,
    cover_media_id UUID REFERENCES media_files (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    is_free BOOLEAN NOT NULL DEFAULT true,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sticker_packs_status_idx ON sticker_packs (status, created_at DESC);
CREATE INDEX sticker_packs_category_idx ON sticker_packs (category, created_at DESC);
CREATE INDEX sticker_packs_creator_idx ON sticker_packs (creator_id, created_at DESC);

CREATE TABLE stickers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sticker_pack_id UUID NOT NULL REFERENCES sticker_packs (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    media_file_id UUID NOT NULL REFERENCES media_files (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    name TEXT NOT NULL DEFAULT '',
    is_animated BOOLEAN NOT NULL DEFAULT false,
    position INTEGER NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT stickers_pack_position_unique UNIQUE (sticker_pack_id, position)
);

CREATE INDEX stickers_pack_idx ON stickers (sticker_pack_id, position);

CREATE TABLE user_stickers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    sticker_pack_id UUID REFERENCES sticker_packs (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    sticker_id UUID REFERENCES stickers (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    source TEXT NOT NULL,
    last_used_at TIMESTAMPTZ,
    is_favorite BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_stickers_exactly_one_target CHECK (
        (sticker_pack_id IS NULL) <> (sticker_id IS NULL)
    )
);

CREATE UNIQUE INDEX user_stickers_user_pack_unique ON user_stickers (user_id, sticker_pack_id) WHERE sticker_pack_id IS NOT NULL;
CREATE UNIQUE INDEX user_stickers_user_sticker_unique ON user_stickers (user_id, sticker_id) WHERE sticker_id IS NOT NULL;
CREATE INDEX user_stickers_user_recent_idx ON user_stickers (user_id, last_used_at DESC) WHERE source = 'recent';
