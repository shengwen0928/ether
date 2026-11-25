// PostgreSQL 儲存實作（Phase 05）：chat.Store。
//
// 對映規則（schema 見 migrations/000006）：uuid 一律 ::text 讀出／
// ::uuid 傳入；可空欄位以 NULL 語義（見 internal/postgres 慣例）；
// 查詢一律具名欄位列。查無列時 GetConversation／GetMember 回傳
// (nil, nil)，由服務層決定 404 語義（不得洩漏存在性）。
package chat

import (
	"context"
	"database/sql"

	"ether.local/api/internal/postgres"
)

// PGStore 是 Store 的 PostgreSQL 實作。
type PGStore struct{ q postgres.Querier }

// NewPGStore 建立 PGStore。
func NewPGStore(q postgres.Querier) *PGStore { return &PGStore{q: q} }

const conversationColumns = `id::text, type, title, created_by::text, last_message_id::text, last_message_at, created_at, updated_at`

func scanConversation(row postgres.Row) (*Conversation, error) {
	var c Conversation
	var typ string
	var title, createdBy, lastMessageID sql.NullString
	var lastMessageAt sql.NullTime
	if err := row.Scan(&c.ID, &typ, &title, &createdBy, &lastMessageID, &lastMessageAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	c.Type = ConversationType(typ)
	c.Title = postgres.StringFromNull(title)
	c.CreatedBy = postgres.StringFromNull(createdBy)
	c.LastMessageID = postgres.StringFromNull(lastMessageID)
	c.LastMessageAt = postgres.TimePtrFromNull(lastMessageAt)
	return &c, nil
}

func scanConversationRows(rows postgres.Rows) ([]*Conversation, error) {
	defer func() { _ = rows.Close() }()
	var out []*Conversation
	for rows.Next() {
		var c Conversation
		var typ string
		var title, createdBy, lastMessageID sql.NullString
		var lastMessageAt sql.NullTime
		if err := rows.Scan(&c.ID, &typ, &title, &createdBy, &lastMessageID, &lastMessageAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.Type = ConversationType(typ)
		c.Title = postgres.StringFromNull(title)
		c.CreatedBy = postgres.StringFromNull(createdBy)
		c.LastMessageID = postgres.StringFromNull(lastMessageID)
		c.LastMessageAt = postgres.TimePtrFromNull(lastMessageAt)
		out = append(out, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	return scanConversation(s.q.QueryRowContext(ctx,
		`SELECT `+conversationColumns+` FROM conversations WHERE id = $1::uuid`, id))
}

// GetDirectBetween 以兩名成員反查一對一對話：對話須恰好兩名
// 成員列（歷史上皆未物理刪除成員列，離開者以 left_at 標記，故
// 計數含離開者；重新加入復用同一列）。
func (s *PGStore) GetDirectBetween(ctx context.Context, a, b string) (*Conversation, error) {
	return scanConversation(s.q.QueryRowContext(ctx,
		`SELECT `+conversationColumns+` FROM conversations c
		 WHERE c.type = 'direct'
		 AND EXISTS (SELECT 1 FROM conversation_members m1
		             WHERE m1.conversation_id = c.id AND m1.user_id = $1::uuid)
		 AND EXISTS (SELECT 1 FROM conversation_members m2
		             WHERE m2.conversation_id = c.id AND m2.user_id = $2::uuid)
		 AND (SELECT COUNT(*) FROM conversation_members m3
		      WHERE m3.conversation_id = c.id) = 2
		 ORDER BY c.created_at, c.id LIMIT 1`, a, b))
}

func (s *PGStore) ListConversationsByUser(ctx context.Context, userID string) ([]*Conversation, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+conversationColumns+` FROM conversations c
		 JOIN conversation_members m ON m.conversation_id = c.id
		 WHERE m.user_id = $1::uuid AND m.left_at IS NULL
		 ORDER BY c.updated_at DESC, c.id`, userID)
	if err != nil {
		return nil, err
	}
	return scanConversationRows(rows)
}

func (s *PGStore) CreateConversation(ctx context.Context, c *Conversation) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO conversations (id, type, title, created_by, created_at, updated_at)
		 VALUES ($1::uuid, $2, $3, $4::uuid, $5, $6)`,
		c.ID, string(c.Type), postgres.NullString(c.Title), postgres.NullString(c.CreatedBy), c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *PGStore) SaveConversation(ctx context.Context, c *Conversation) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE conversations
		 SET type = $2, title = $3, created_by = $4::uuid,
		     last_message_id = $5::uuid, last_message_at = $6, updated_at = $7
		 WHERE id = $1::uuid`,
		c.ID, string(c.Type), postgres.NullString(c.Title), postgres.NullString(c.CreatedBy),
		postgres.NullString(c.LastMessageID), postgres.NullTime(c.LastMessageAt), c.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrConversationNotFound
	}
	return nil
}

const memberColumns = `conversation_id::text, user_id::text, role, joined_at, left_at, last_read_message_id::text, last_read_sequence_number, last_read_at, created_at, updated_at`

func scanMember(row postgres.Row) (*Member, error) {
	var m Member
	var role string
	var leftAt, lastReadAt sql.NullTime
	var lastReadMessageID sql.NullString
	if err := row.Scan(&m.ConversationID, &m.UserID, &role, &m.JoinedAt, &leftAt, &lastReadMessageID, &m.LastReadSequence, &lastReadAt, &m.CreatedAt, &m.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	m.Role = MemberRole(role)
	m.LeftAt = postgres.TimePtrFromNull(leftAt)
	m.LastReadMessageID = postgres.StringFromNull(lastReadMessageID)
	m.LastReadAt = postgres.TimePtrFromNull(lastReadAt)
	return &m, nil
}

func scanMemberRows(rows postgres.Rows) ([]*Member, error) {
	defer func() { _ = rows.Close() }()
	var out []*Member
	for rows.Next() {
		var m Member
		var role string
		var leftAt, lastReadAt sql.NullTime
		var lastReadMessageID sql.NullString
		if err := rows.Scan(&m.ConversationID, &m.UserID, &role, &m.JoinedAt, &leftAt, &lastReadMessageID, &m.LastReadSequence, &lastReadAt, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		m.Role = MemberRole(role)
		m.LeftAt = postgres.TimePtrFromNull(leftAt)
		m.LastReadMessageID = postgres.StringFromNull(lastReadMessageID)
		m.LastReadAt = postgres.TimePtrFromNull(lastReadAt)
		out = append(out, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetMember(ctx context.Context, conversationID, userID string) (*Member, error) {
	return scanMember(s.q.QueryRowContext(ctx,
		`SELECT `+memberColumns+` FROM conversation_members
		 WHERE conversation_id = $1::uuid AND user_id = $2::uuid`, conversationID, userID))
}

func (s *PGStore) ListMembers(ctx context.Context, conversationID string) ([]*Member, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+memberColumns+` FROM conversation_members
		 WHERE conversation_id = $1::uuid
		 ORDER BY (left_at IS NOT NULL), joined_at, user_id::text`, conversationID)
	if err != nil {
		return nil, err
	}
	return scanMemberRows(rows)
}

func (s *PGStore) CreateMember(ctx context.Context, m *Member) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO conversation_members
		 (conversation_id, user_id, role, joined_at, left_at,
		  last_read_message_id, last_read_sequence_number, last_read_at, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::uuid, $7, $8, $9, $10)`,
		m.ConversationID, m.UserID, string(m.Role), m.JoinedAt, postgres.NullTime(m.LeftAt),
		postgres.NullString(m.LastReadMessageID), m.LastReadSequence, postgres.NullTime(m.LastReadAt),
		m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *PGStore) SaveMember(ctx context.Context, m *Member) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE conversation_members
		 SET role = $3, joined_at = $4, left_at = $5,
		     last_read_message_id = $6::uuid, last_read_sequence_number = $7,
		     last_read_at = $8, updated_at = $9
		 WHERE conversation_id = $1::uuid AND user_id = $2::uuid`,
		m.ConversationID, m.UserID, string(m.Role), m.JoinedAt, postgres.NullTime(m.LeftAt),
		postgres.NullString(m.LastReadMessageID), m.LastReadSequence, postgres.NullTime(m.LastReadAt),
		m.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrConversationNotFound
	}
	return nil
}
