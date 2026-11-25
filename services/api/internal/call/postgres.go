// PostgreSQL 儲存實作（Phase 07）：call.Store。
//
// 對映規則（schema 見 migrations/000013）：uuid 一律 ::text 讀出／
// ::uuid 傳入；可空欄位以 NULL 語義（見 internal/postgres 慣例）；
// 查詢一律具名欄位列。查無列時 Get* 回傳 (nil, nil)，由服務層決定
// 404 語義（不得洩漏存在性）。參與者 (call_id, user_id) 唯一衝突
// 對映為欄位錯誤（競態兜底）；其餘唯一性違反原樣回傳。
//
// 已知限制：通話列、參與者列、事件列分步寫入（無跨表交易邊界；
// 發起寫入中途失敗時可能殘留通話列，見 STATUS.md §2ad）。狀態
// 轉移的讀取—檢查—寫入無序列化，併發接聽／拒接以最後寫入為準。
package call

import (
	"context"
	"database/sql"
	"time"
	"unicode/utf8"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/transport"
)

// PGStore 是 Store 的 PostgreSQL 實作。
type PGStore struct{ q postgres.Querier }

// nullInt64 將空指標轉為 NULL 參數（可空 bigint 欄位寫入用）。
func nullInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

// NewPGStore 建立 PGStore。
func NewPGStore(q postgres.Querier) *PGStore { return &PGStore{q: q} }

const callColumns = `id::text, conversation_id::text, group_id::text, initiator_id::text, call_type, status, started_at, ended_at, duration_seconds, ended_by::text, end_reason, created_at, updated_at`

func scanCall(row postgres.Row) (*Call, error) {
	var c Call
	var conversationID, groupID, initiatorID, endedBy sql.NullString
	var callType, status string
	var startedAt, endedAt sql.NullTime
	var duration sql.NullInt64
	var endReason sql.NullString
	if err := row.Scan(&c.ID, &conversationID, &groupID, &initiatorID, &callType, &status, &startedAt, &endedAt, &duration, &endedBy, &endReason, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	c.Type = CallType(callType)
	c.Status = Status(status)
	c.ConversationID = postgres.StringFromNull(conversationID)
	c.GroupID = postgres.StringFromNull(groupID)
	c.InitiatorID = postgres.StringFromNull(initiatorID)
	c.StartedAt = postgres.TimePtrFromNull(startedAt)
	c.EndedAt = postgres.TimePtrFromNull(endedAt)
	if duration.Valid {
		c.Duration = &duration.Int64
	}
	c.EndedBy = postgres.StringFromNull(endedBy)
	c.EndReason = EndReason(postgres.StringFromNull(endReason))
	return &c, nil
}

func (s *PGStore) GetCall(ctx context.Context, id string) (*Call, error) {
	row := s.q.QueryRowContext(ctx, `SELECT `+callColumns+` FROM calls WHERE id = $1::uuid`, id)
	return scanCall(row)
}

func (s *PGStore) CreateCall(ctx context.Context, c *Call) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO calls
		(id, conversation_id, group_id, initiator_id, call_type, status, started_at, ended_at, duration_seconds, ended_by, end_reason, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9, $10::uuid, $11, $12, $13)`,
		c.ID, postgres.NullString(c.ConversationID), postgres.NullString(c.GroupID), c.InitiatorID,
		string(c.Type), string(c.Status), postgres.NullTime(c.StartedAt), postgres.NullTime(c.EndedAt),
		nullInt64(c.Duration), postgres.NullString(c.EndedBy), postgres.NullString(string(c.EndReason)),
		c.CreatedAt, c.UpdatedAt)
	if postgres.IsUniqueViolation(err) {
		return validationError("call_id", "DUPLICATE", "通話已存在。")
	}
	return err
}

func (s *PGStore) SaveCall(ctx context.Context, c *Call) error {
	res, err := s.q.ExecContext(ctx, `UPDATE calls SET
		conversation_id = $2::uuid, group_id = $3::uuid, initiator_id = $4::uuid,
		call_type = $5, status = $6, started_at = $7, ended_at = $8,
		duration_seconds = $9, ended_by = $10::uuid, end_reason = $11, updated_at = $12
		WHERE id = $1::uuid`,
		c.ID, postgres.NullString(c.ConversationID), postgres.NullString(c.GroupID), c.InitiatorID,
		string(c.Type), string(c.Status), postgres.NullTime(c.StartedAt), postgres.NullTime(c.EndedAt),
		nullInt64(c.Duration), postgres.NullString(c.EndedBy), postgres.NullString(string(c.EndReason)),
		c.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrCallNotFound
	}
	return nil
}

func (s *PGStore) FindActiveCall(ctx context.Context, conversationID, groupID string) (*Call, error) {
	row := s.q.QueryRowContext(ctx, `SELECT `+callColumns+` FROM calls
		WHERE status IN ('ringing', 'active')
		AND (($1::uuid IS NOT NULL AND conversation_id = $1::uuid)
			OR ($2::uuid IS NOT NULL AND group_id = $2::uuid))
		ORDER BY created_at DESC LIMIT 1`,
		postgres.NullString(conversationID), postgres.NullString(groupID))
	return scanCall(row)
}

func (s *PGStore) ListCallsByUser(ctx context.Context, userID, cursor string, limit int) ([]*Call, error) {
	var cursorTime time.Time
	var cursorID string
	var cursorArg any
	if cursor != "" {
		t, id, err := decodeCursor(cursor)
		if err != nil {
			return nil, err
		}
		cursorTime, cursorID = t, id
		cursorArg = cursorTime
	}
	rows, err := s.q.QueryContext(ctx, `SELECT `+callColumns+` FROM calls c
		JOIN call_participants p ON p.call_id = c.id
		WHERE p.user_id = $1::uuid
		AND ($2::timestamptz IS NULL OR (c.created_at, c.id) < ($2::timestamptz, $3::uuid))
		ORDER BY c.created_at DESC, c.id DESC LIMIT $4`,
		userID, cursorArg, postgres.NullString(cursorID), limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Call
	for rows.Next() {
		var c Call
		var conversationID, groupID, initiatorID, endedBy sql.NullString
		var callType, status string
		var startedAt, endedAt sql.NullTime
		var duration sql.NullInt64
		var endReason sql.NullString
		if err := rows.Scan(&c.ID, &conversationID, &groupID, &initiatorID, &callType, &status, &startedAt, &endedAt, &duration, &endedBy, &endReason, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.Type = CallType(callType)
		c.Status = Status(status)
		c.ConversationID = postgres.StringFromNull(conversationID)
		c.GroupID = postgres.StringFromNull(groupID)
		c.InitiatorID = postgres.StringFromNull(initiatorID)
		c.StartedAt = postgres.TimePtrFromNull(startedAt)
		c.EndedAt = postgres.TimePtrFromNull(endedAt)
		if duration.Valid {
			c.Duration = &duration.Int64
		}
		c.EndedBy = postgres.StringFromNull(endedBy)
		c.EndReason = EndReason(postgres.StringFromNull(endReason))
		out = append(out, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

const participantColumns = `id::text, call_id::text, user_id::text, joined_at, left_at, result, created_at`

func scanParticipant(row postgres.Row) (*Participant, error) {
	var p Participant
	var joinedAt, leftAt sql.NullTime
	var result string
	if err := row.Scan(&p.ID, &p.CallID, &p.UserID, &joinedAt, &leftAt, &result, &p.CreatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	p.Result = ParticipantResult(result)
	p.JoinedAt = postgres.TimePtrFromNull(joinedAt)
	p.LeftAt = postgres.TimePtrFromNull(leftAt)
	return &p, nil
}

func (s *PGStore) GetParticipant(ctx context.Context, callID, userID string) (*Participant, error) {
	row := s.q.QueryRowContext(ctx, `SELECT `+participantColumns+` FROM call_participants
		WHERE call_id = $1::uuid AND user_id = $2::uuid`, callID, userID)
	return scanParticipant(row)
}

func (s *PGStore) ListParticipants(ctx context.Context, callID string) ([]*Participant, error) {
	rows, err := s.q.QueryContext(ctx, `SELECT `+participantColumns+` FROM call_participants
		WHERE call_id = $1::uuid ORDER BY created_at ASC, id ASC`, callID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Participant
	for rows.Next() {
		var p Participant
		var joinedAt, leftAt sql.NullTime
		var result string
		if err := rows.Scan(&p.ID, &p.CallID, &p.UserID, &joinedAt, &leftAt, &result, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Result = ParticipantResult(result)
		p.JoinedAt = postgres.TimePtrFromNull(joinedAt)
		p.LeftAt = postgres.TimePtrFromNull(leftAt)
		out = append(out, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) CreateParticipant(ctx context.Context, p *Participant) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO call_participants
		(id, call_id, user_id, joined_at, left_at, result, created_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)`,
		p.ID, p.CallID, p.UserID, postgres.NullTime(p.JoinedAt), postgres.NullTime(p.LeftAt),
		string(p.Result), p.CreatedAt)
	if postgres.IsUniqueViolation(err) {
		return validationError("user_id", "DUPLICATE", "參與者已存在。")
	}
	return err
}

func (s *PGStore) SaveParticipant(ctx context.Context, p *Participant) error {
	res, err := s.q.ExecContext(ctx, `UPDATE call_participants SET
		joined_at = $3, left_at = $4, result = $5
		WHERE call_id = $1::uuid AND user_id = $2::uuid`,
		p.CallID, p.UserID, postgres.NullTime(p.JoinedAt), postgres.NullTime(p.LeftAt), string(p.Result))
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrCallNotFound
	}
	return nil
}

func (s *PGStore) CreateEvent(ctx context.Context, e *CallEvent) error {
	if utf8.RuneCountInString(e.Payload) > 2000 {
		return &ValidationError{Fields: []transport.FieldError{{
			Field: "event_payload", Issue: "TOO_LONG", Message: "事件內容過長。",
		}}}
	}
	_, err := s.q.ExecContext(ctx, `INSERT INTO call_events
		(id, call_id, user_id, event_type, event_payload, occurred_at, created_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)`,
		e.ID, e.CallID, postgres.NullString(e.UserID), string(e.Type), e.Payload, e.OccurredAt, e.CreatedAt)
	return err
}

func (s *PGStore) ListEvents(ctx context.Context, callID string) ([]*CallEvent, error) {
	rows, err := s.q.QueryContext(ctx, `SELECT id::text, call_id::text, user_id::text, event_type, event_payload, occurred_at, created_at
		FROM call_events WHERE call_id = $1::uuid ORDER BY occurred_at ASC, id ASC`, callID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*CallEvent
	for rows.Next() {
		var e CallEvent
		var userID sql.NullString
		var eventType string
		if err := rows.Scan(&e.ID, &e.CallID, &userID, &eventType, &e.Payload, &e.OccurredAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Type = EventType(eventType)
		e.UserID = postgres.StringFromNull(userID)
		out = append(out, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
