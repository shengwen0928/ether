-- 000007_groups.up.sql — 群組五表（Phase 06，離線可驗證部分）。
--
-- 狀態：草案，待有 PostgreSQL 的隔離環境驗證。本機 VM 無 psql／docker，
-- 本檔案未經資料庫套用驗證，不得宣稱已套用、不得宣稱欄位已定稿。
-- 依據：docs/database/ERD.md §6.4 草案欄位（欄位名、型別、約束皆為草案，
-- 待 Migration 定稿後 ERD 再同步修訂）；模組邊界見
-- docs/architecture/MODULE_BOUNDARIES.md §5.8（Group）。
-- 是否破壞性：否（僅新增表，不動既有結構）。
--
-- 與 ERD 草案的差異與 Phase 06 決定（皆有理由，待定稿覆核）：
--   1. 只建五表：groups、group_members、group_invitations、
--      group_announcements、group_settings。group_roles 屬自訂角色
--      後續里程碑，本里程碑不建表：角色為固定列舉 owner／admin／
--      member，以 group_members.role 文字欄為唯一角色來源
--      （ERD §6.4.2 稱 role 與 role_id 主從待定稿；本檔案不並存
--      二者，不建 role_id，避免雙來源不一致）。
--   2. groups 含 conversation_id（可為空，→ conversations.id，UNIQUE）：
--      群組聊天室容器經 Chat 介面建立後寫入；對應方式（一對一）
--      為本里程碑實作決定，ERD §10 缺口仍待文件定稿覆核。
--      不建 member_count_cache（快取欄位，更新規則待定稿）：成員數
--      一律即時計數 group_members，不得以快取代替而不說明。
--      解散為 is_disbanded＋disbanded_at 狀態標記，不得物理刪除。
--   3. group_members 不建 muted_until／nickname_in_group（禁言／群內
--      暱稱屬後續里程碑）；(group_id, user_id) 唯一：離開為狀態標記
--      （left_at），重新加入復用同一列（joined_at 更新、left_at 清空、
--      角色重設為 member），離開／重入歷史不另存列，待定稿覆核。
--   4. group_invitations 只承載直接邀請（inviter_id／invitee_id 皆
--      必填）：不建 invite_token／expires_at（連結／QR 邀請與到期
--      屬後續里程碑）；待處理邀請以部分唯一索引保證同一群組對同一
--      被邀請者僅一筆（競態兜底，服務層先查後建）。status 值域為
--      本里程碑實作詞彙 pending／accepted／declined／cancelled
--      （text，不加 CHECK）。
--   5. group_announcements 同時僅一則現行公告，以部分唯一索引
--      (group_id) WHERE is_active 保證；新公告發佈時服務層先撤下
--      舊公告（兩步操作無交易邊界，併發發佈時後者收唯一衝突，
--      fail-closed，見 STATUS.md 已知限制）。撤下為 is_active＋
--      retracted_at 標記，不得物理刪除；編輯歷史不保留。
--   6. group_settings 為鍵值表（ERD 草案）：(group_id, setting_key)
--      唯一；本里程碑僅 invite_policy 一項鍵（admin／all），未知鍵
--      由儲存層讀取時忽略、服務層拒絕寫入。FK 動作一律
--      ON DELETE RESTRICT（ERD 草案稱設定隨 groups CASCADE 處理，
--      本檔案依本目錄慣例取 RESTRICT，群組解散後設定保留，待定稿
--      覆核）；ON UPDATE CASCADE。
--   7. FK 動作一律 ON DELETE RESTRICT（關係／稽核紀錄不得隨帳號
--      物理刪除，ERD §8 原則草案）；ON UPDATE CASCADE。
--   8. gen_random_uuid() 為 PostgreSQL 13+ 內建函式，compose 的
--      pgvector:0.8.7-pg16 映像可用，無須額外擴充。

CREATE TABLE groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    avatar_media_id TEXT,
    conversation_id UUID REFERENCES conversations (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    created_by UUID REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    is_disbanded BOOLEAN NOT NULL DEFAULT false,
    disbanded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (conversation_id)
);

CREATE INDEX groups_created_by_idx ON groups (created_by, updated_at DESC);

CREATE TABLE group_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    role TEXT NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at TIMESTAMPTZ,
    invited_by UUID REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT group_members_group_user_unique UNIQUE (group_id, user_id)
);

CREATE INDEX group_members_group_idx ON group_members (group_id, joined_at);
CREATE INDEX group_members_user_idx ON group_members (user_id, left_at);

CREATE TABLE group_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    inviter_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    invitee_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    status TEXT NOT NULL,
    responded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX group_invitations_pending_unique_idx
    ON group_invitations (group_id, invitee_id)
    WHERE status = 'pending';

CREATE INDEX group_invitations_group_idx ON group_invitations (group_id, status);
CREATE INDEX group_invitations_invitee_idx ON group_invitations (invitee_id, status);

CREATE TABLE group_announcements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    author_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    content TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    retracted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX group_announcements_active_unique_idx
    ON group_announcements (group_id)
    WHERE is_active;

CREATE INDEX group_announcements_group_idx ON group_announcements (group_id, published_at DESC);

CREATE TABLE group_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    setting_key TEXT NOT NULL,
    setting_value TEXT NOT NULL,
    updated_by UUID REFERENCES users (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (group_id, setting_key)
);

CREATE INDEX group_settings_group_idx ON group_settings (group_id);
