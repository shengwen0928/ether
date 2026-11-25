// PostgreSQL 儲存實作（Phase 06）：group.Store。
//
// 對映規則（schema 見 migrations/000007）：uuid 一律 ::text 讀出／
// ::uuid 傳入；可空欄位以 NULL 語義（見 internal/postgres 慣例）；
// 查詢一律具名欄位列。查無列時 Get* 回傳 (nil, nil)，由服務層決定
// 404 語義（不得洩漏存在性）。待處理邀請的部分唯一索引衝突對映
// 為欄位錯誤（競態兜底）；其餘唯一性違反原樣回傳。
package group

import (
	"context"
	"database/sql"
	"time"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/transport"
)

// PGStore 是 Store 的 PostgreSQL 實作。
type PGStore struct{ q postgres.Querier }

// NewPGStore 建立 PGStore。
func NewPGStore(q postgres.Querier) *PGStore { return &PGStore{q: q} }

const groupColumns = `id::text, name, description, avatar_media_id, conversation_id::text, created_by::text, is_disbanded, disbanded_at, created_at, updated_at`

func scanGroup(row postgres.Row) (*Group, error) {
	var g Group
	var description, avatarMediaID, conversationID, createdBy sql.NullString
	var disbandedAt sql.NullTime
	if err := row.Scan(&g.ID, &g.Name, &description, &avatarMediaID, &conversationID, &createdBy, &g.IsDisbanded, &disbandedAt, &g.CreatedAt, &g.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	g.Description = postgres.StringFromNull(description)
	g.AvatarMediaID = postgres.StringFromNull(avatarMediaID)
	g.ConversationID = postgres.StringFromNull(conversationID)
	g.CreatedBy = postgres.StringFromNull(createdBy)
	g.DisbandedAt = postgres.TimePtrFromNull(disbandedAt)
	return &g, nil
}

func scanGroupRows(rows postgres.Rows) ([]*Group, error) {
	defer func() { _ = rows.Close() }()
	var out []*Group
	for rows.Next() {
		var g Group
		var description, avatarMediaID, conversationID, createdBy sql.NullString
		var disbandedAt sql.NullTime
		if err := rows.Scan(&g.ID, &g.Name, &description, &avatarMediaID, &conversationID, &createdBy, &g.IsDisbanded, &disbandedAt, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		g.Description = postgres.StringFromNull(description)
		g.AvatarMediaID = postgres.StringFromNull(avatarMediaID)
		g.ConversationID = postgres.StringFromNull(conversationID)
		g.CreatedBy = postgres.StringFromNull(createdBy)
		g.DisbandedAt = postgres.TimePtrFromNull(disbandedAt)
		out = append(out, &g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetGroup(ctx context.Context, id string) (*Group, error) {
	return scanGroup(s.q.QueryRowContext(ctx,
		`SELECT `+groupColumns+` FROM groups WHERE id = $1::uuid`, id))
}

func (s *PGStore) ListGroupsByUser(ctx context.Context, userID string) ([]*Group, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+groupColumns+` FROM groups g
		 JOIN group_members m ON m.group_id = g.id
		 WHERE m.user_id = $1::uuid AND m.left_at IS NULL AND NOT g.is_disbanded
		 ORDER BY g.updated_at DESC, g.id`, userID)
	if err != nil {
		return nil, err
	}
	return scanGroupRows(rows)
}

func (s *PGStore) CreateGroup(ctx context.Context, g *Group) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO groups (id, name, description, avatar_media_id, conversation_id, created_by, is_disbanded, disbanded_at, created_at, updated_at)
		 VALUES ($1::uuid, $2, $3, $4, $5::uuid, $6::uuid, $7, $8, $9, $10)`,
		g.ID, g.Name, postgres.NullString(g.Description), postgres.NullString(g.AvatarMediaID),
		postgres.NullString(g.ConversationID), postgres.NullString(g.CreatedBy),
		g.IsDisbanded, postgres.NullTime(g.DisbandedAt), g.CreatedAt, g.UpdatedAt)
	return err
}

func (s *PGStore) SaveGroup(ctx context.Context, g *Group) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE groups
		 SET name = $2, description = $3, avatar_media_id = $4, conversation_id = $5::uuid,
		     created_by = $6::uuid, is_disbanded = $7, disbanded_at = $8, updated_at = $9
		 WHERE id = $1::uuid`,
		g.ID, g.Name, postgres.NullString(g.Description), postgres.NullString(g.AvatarMediaID),
		postgres.NullString(g.ConversationID), postgres.NullString(g.CreatedBy),
		g.IsDisbanded, postgres.NullTime(g.DisbandedAt), g.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrGroupNotFound
	}
	return nil
}

func (s *PGStore) CountActiveMembers(ctx context.Context, groupID string) (int, error) {
	var n int
	row := s.q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM group_members WHERE group_id = $1::uuid AND left_at IS NULL`, groupID)
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

const memberColumns = `group_id::text, user_id::text, role, joined_at, left_at, invited_by::text, created_at, updated_at`

func scanMember(row postgres.Row) (*Member, error) {
	var m Member
	var role string
	var leftAt sql.NullTime
	var invitedBy sql.NullString
	if err := row.Scan(&m.GroupID, &m.UserID, &role, &m.JoinedAt, &leftAt, &invitedBy, &m.CreatedAt, &m.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	m.Role = Role(role)
	m.LeftAt = postgres.TimePtrFromNull(leftAt)
	m.InvitedBy = postgres.StringFromNull(invitedBy)
	return &m, nil
}

func scanMemberRows(rows postgres.Rows) ([]*Member, error) {
	defer func() { _ = rows.Close() }()
	var out []*Member
	for rows.Next() {
		var m Member
		var role string
		var leftAt sql.NullTime
		var invitedBy sql.NullString
		if err := rows.Scan(&m.GroupID, &m.UserID, &role, &m.JoinedAt, &leftAt, &invitedBy, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		m.Role = Role(role)
		m.LeftAt = postgres.TimePtrFromNull(leftAt)
		m.InvitedBy = postgres.StringFromNull(invitedBy)
		out = append(out, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetMember(ctx context.Context, groupID, userID string) (*Member, error) {
	return scanMember(s.q.QueryRowContext(ctx,
		`SELECT `+memberColumns+` FROM group_members
		 WHERE group_id = $1::uuid AND user_id = $2::uuid`, groupID, userID))
}

func (s *PGStore) ListMembers(ctx context.Context, groupID string) ([]*Member, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+memberColumns+` FROM group_members
		 WHERE group_id = $1::uuid AND left_at IS NULL
		 ORDER BY joined_at, user_id::text`, groupID)
	if err != nil {
		return nil, err
	}
	return scanMemberRows(rows)
}

func (s *PGStore) CreateMember(ctx context.Context, m *Member) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO group_members
		 (group_id, user_id, role, joined_at, left_at, invited_by, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::uuid, $7, $8)`,
		m.GroupID, m.UserID, string(m.Role), m.JoinedAt, postgres.NullTime(m.LeftAt),
		postgres.NullString(m.InvitedBy), m.CreatedAt, m.UpdatedAt)
	return mapMemberUniqueError(err)
}

func (s *PGStore) SaveMember(ctx context.Context, m *Member) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE group_members
		 SET role = $3, joined_at = $4, left_at = $5, invited_by = $6::uuid, updated_at = $7
		 WHERE group_id = $1::uuid AND user_id = $2::uuid`,
		m.GroupID, m.UserID, string(m.Role), m.JoinedAt, postgres.NullTime(m.LeftAt),
		postgres.NullString(m.InvitedBy), m.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrGroupNotFound
	}
	return nil
}

func mapMemberUniqueError(err error) error {
	if postgres.IsUniqueViolation(err) && postgres.UniqueConstraintName(err) == "group_members_group_user_unique" {
		return &ValidationError{Fields: []transport.FieldError{{
			Field:   "member_ids",
			Issue:   "DUPLICATE",
			Message: "對方已是群組成員。",
		}}}
	}
	return err
}

const invitationColumns = `id::text, group_id::text, inviter_id::text, invitee_id::text, status, responded_at, created_at, updated_at`

func scanInvitation(row postgres.Row) (*Invitation, error) {
	var inv Invitation
	var status string
	var respondedAt sql.NullTime
	if err := row.Scan(&inv.ID, &inv.GroupID, &inv.InviterID, &inv.InviteeID, &status, &respondedAt, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	inv.Status = InvitationStatus(status)
	inv.RespondedAt = postgres.TimePtrFromNull(respondedAt)
	return &inv, nil
}

func scanInvitationRows(rows postgres.Rows) ([]*Invitation, error) {
	defer func() { _ = rows.Close() }()
	var out []*Invitation
	for rows.Next() {
		var inv Invitation
		var status string
		var respondedAt sql.NullTime
		if err := rows.Scan(&inv.ID, &inv.GroupID, &inv.InviterID, &inv.InviteeID, &status, &respondedAt, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		inv.Status = InvitationStatus(status)
		inv.RespondedAt = postgres.TimePtrFromNull(respondedAt)
		out = append(out, &inv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetInvitation(ctx context.Context, id string) (*Invitation, error) {
	return scanInvitation(s.q.QueryRowContext(ctx,
		`SELECT `+invitationColumns+` FROM group_invitations WHERE id = $1::uuid`, id))
}

func (s *PGStore) GetPendingInvitation(ctx context.Context, groupID, inviteeID string) (*Invitation, error) {
	return scanInvitation(s.q.QueryRowContext(ctx,
		`SELECT `+invitationColumns+` FROM group_invitations
		 WHERE group_id = $1::uuid AND invitee_id = $2::uuid AND status = 'pending'
		 ORDER BY created_at, id LIMIT 1`, groupID, inviteeID))
}

func (s *PGStore) ListGroupInvitations(ctx context.Context, groupID string) ([]*Invitation, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+invitationColumns+` FROM group_invitations
		 WHERE group_id = $1::uuid AND status = 'pending'
		 ORDER BY created_at, id`, groupID)
	if err != nil {
		return nil, err
	}
	return scanInvitationRows(rows)
}

func (s *PGStore) ListIncomingInvitations(ctx context.Context, userID string) ([]*Invitation, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+invitationColumns+` FROM group_invitations
		 WHERE invitee_id = $1::uuid AND status = 'pending'
		 ORDER BY created_at, id`, userID)
	if err != nil {
		return nil, err
	}
	return scanInvitationRows(rows)
}

func (s *PGStore) CreateInvitation(ctx context.Context, inv *Invitation) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO group_invitations
		 (id, group_id, inviter_id, invitee_id, status, responded_at, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8)`,
		inv.ID, inv.GroupID, inv.InviterID, inv.InviteeID, string(inv.Status),
		postgres.NullTime(inv.RespondedAt), inv.CreatedAt, inv.UpdatedAt)
	return mapInvitationUniqueError(err)
}

func (s *PGStore) SaveInvitation(ctx context.Context, inv *Invitation) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE group_invitations
		 SET status = $2, responded_at = $3, updated_at = $4
		 WHERE id = $1::uuid`,
		inv.ID, string(inv.Status), postgres.NullTime(inv.RespondedAt), inv.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrGroupNotFound
	}
	return nil
}

func mapInvitationUniqueError(err error) error {
	if postgres.IsUniqueViolation(err) && postgres.UniqueConstraintName(err) == "group_invitations_pending_unique_idx" {
		return &ValidationError{Fields: []transport.FieldError{{
			Field:   "invitee_id",
			Issue:   "DUPLICATE",
			Message: "已送出待處理的群組邀請。",
		}}}
	}
	return err
}

const announcementColumns = `id::text, group_id::text, author_id::text, content, is_active, published_at, retracted_at, created_at, updated_at`

func scanAnnouncement(row postgres.Row) (*Announcement, error) {
	var a Announcement
	var retractedAt sql.NullTime
	if err := row.Scan(&a.ID, &a.GroupID, &a.AuthorID, &a.Content, &a.IsActive, &a.PublishedAt, &retractedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	a.RetractedAt = postgres.TimePtrFromNull(retractedAt)
	return &a, nil
}

func scanAnnouncementRows(rows postgres.Rows) ([]*Announcement, error) {
	defer func() { _ = rows.Close() }()
	var out []*Announcement
	for rows.Next() {
		var a Announcement
		var retractedAt sql.NullTime
		if err := rows.Scan(&a.ID, &a.GroupID, &a.AuthorID, &a.Content, &a.IsActive, &a.PublishedAt, &retractedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.RetractedAt = postgres.TimePtrFromNull(retractedAt)
		out = append(out, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetAnnouncement(ctx context.Context, id string) (*Announcement, error) {
	return scanAnnouncement(s.q.QueryRowContext(ctx,
		`SELECT `+announcementColumns+` FROM group_announcements WHERE id = $1::uuid`, id))
}

func (s *PGStore) ListAnnouncements(ctx context.Context, groupID string) ([]*Announcement, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+announcementColumns+` FROM group_announcements
		 WHERE group_id = $1::uuid
		 ORDER BY published_at DESC, id`, groupID)
	if err != nil {
		return nil, err
	}
	return scanAnnouncementRows(rows)
}

func (s *PGStore) RetractActiveAnnouncements(ctx context.Context, groupID string, at time.Time) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE group_announcements
		 SET is_active = false, retracted_at = $2, updated_at = $2
		 WHERE group_id = $1::uuid AND is_active`, groupID, at)
	return err
}

func (s *PGStore) CreateAnnouncement(ctx context.Context, a *Announcement) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO group_announcements
		 (id, group_id, author_id, content, is_active, published_at, retracted_at, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9)`,
		a.ID, a.GroupID, a.AuthorID, a.Content, a.IsActive, a.PublishedAt,
		postgres.NullTime(a.RetractedAt), a.CreatedAt, a.UpdatedAt)
	return err
}

func (s *PGStore) SaveAnnouncement(ctx context.Context, a *Announcement) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE group_announcements
		 SET content = $2, is_active = $3, published_at = $4, retracted_at = $5, updated_at = $6
		 WHERE id = $1::uuid`,
		a.ID, a.Content, a.IsActive, a.PublishedAt, postgres.NullTime(a.RetractedAt), a.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrGroupNotFound
	}
	return nil
}

// GetSettings 讀取群組的設定列（目前僅 invite_policy 一項鍵；
// 未知鍵忽略，不得影響已知設定）。無列時回 (nil, nil)，由服務層
// 補預設值。
func (s *PGStore) GetSettings(ctx context.Context, groupID string) (*Settings, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT setting_key, setting_value, updated_by::text, updated_at
		 FROM group_settings WHERE group_id = $1::uuid`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out *Settings
	for rows.Next() {
		var key, value string
		var updatedBy sql.NullString
		var updatedAt time.Time
		if err := rows.Scan(&key, &value, &updatedBy, &updatedAt); err != nil {
			return nil, err
		}
		if key != "invite_policy" {
			continue
		}
		updated := updatedAt
		out = &Settings{
			GroupID:      groupID,
			InvitePolicy: InvitePolicy(value),
			UpdatedBy:    postgres.StringFromNull(updatedBy),
			UpdatedAt:    &updated,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) SaveSettings(ctx context.Context, st *Settings) error {
	updatedAt := time.Now().UTC()
	if st.UpdatedAt != nil {
		updatedAt = *st.UpdatedAt
	}
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO group_settings (group_id, setting_key, setting_value, updated_by, created_at, updated_at)
		 VALUES ($1::uuid, 'invite_policy', $2, $3::uuid, $4, $4)
		 ON CONFLICT (group_id, setting_key)
		 DO UPDATE SET setting_value = EXCLUDED.setting_value,
		               updated_by = EXCLUDED.updated_by,
		               updated_at = EXCLUDED.updated_at`,
		st.GroupID, string(st.InvitePolicy), postgres.NullString(st.UpdatedBy), updatedAt)
	return err
}
