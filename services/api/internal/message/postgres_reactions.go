// PostgreSQL 儲存實作（Phase 05 後續）：反應與置頂。
//
// 對映規則（schema 見 migrations/000010）：uuid 一律 ::text 讀出／
// ::uuid 傳入；可空欄位以 NULL 語義（見 internal/postgres 慣例）；
// 查詢一律具名欄位列。生效唯一性以部分唯一索引保證，寫入以
// ON CONFLICT DO NOTHING 收斂併發重複（不得以錯誤冒充成功，
// 亦不得以重試掩蓋真正的唯一衝突）。
package message

import (
	"context"
	"database/sql"
	"time"

	"ether.local/api/internal/postgres"
)

const reactionColumns = `id::text, message_id::text, user_id::text, reaction, created_at, updated_at, removed_at`

func scanReaction(row postgres.Row) (*Reaction, error) {
	var r Reaction
	var removedAt sql.NullTime
	if err := row.Scan(&r.ID, &r.MessageID, &r.UserID, &r.Emoji, &r.CreatedAt, &r.UpdatedAt, &removedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	r.RemovedAt = postgres.TimePtrFromNull(removedAt)
	return &r, nil
}

func scanReactionRows(rows postgres.Rows) ([]*Reaction, error) {
	defer func() { _ = rows.Close() }()
	var out []*Reaction
	for rows.Next() {
		var r Reaction
		var removedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.MessageID, &r.UserID, &r.Emoji, &r.CreatedAt, &r.UpdatedAt, &removedAt); err != nil {
			return nil, err
		}
		r.RemovedAt = postgres.TimePtrFromNull(removedAt)
		out = append(out, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// SetReaction 寫入或更換反應：先標記同一使用者對同一訊息的其他
// 生效反應為取消，再以 ON CONFLICT DO NOTHING 寫入新列（相同
// 表情已生效時不重複寫入），最後讀回生效列。併發更換時部分
// 唯一索引收斂為單一生效列。
func (s *PGStore) SetReaction(ctx context.Context, r *Reaction) (*Reaction, error) {
	if _, err := s.q.ExecContext(ctx,
		`UPDATE message_reactions SET removed_at = now(), updated_at = now()
		 WHERE message_id = $1::uuid AND user_id = $2::uuid AND removed_at IS NULL AND reaction <> $3`,
		r.MessageID, r.UserID, r.Emoji); err != nil {
		return nil, err
	}
	if _, err := s.q.ExecContext(ctx,
		`INSERT INTO message_reactions (id, message_id, user_id, reaction, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6)
		 ON CONFLICT (message_id, user_id, reaction) WHERE removed_at IS NULL DO NOTHING`,
		r.ID, r.MessageID, r.UserID, r.Emoji, r.CreatedAt, r.UpdatedAt); err != nil {
		return nil, err
	}
	return scanReaction(s.q.QueryRowContext(ctx,
		`SELECT `+reactionColumns+` FROM message_reactions
		 WHERE message_id = $1::uuid AND user_id = $2::uuid AND reaction = $3 AND removed_at IS NULL`,
		r.MessageID, r.UserID, r.Emoji))
}

// RemoveReaction 取消使用者對訊息的全部生效反應（冪等）。
func (s *PGStore) RemoveReaction(ctx context.Context, messageID, userID string) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE message_reactions SET removed_at = now(), updated_at = now()
		 WHERE message_id = $1::uuid AND user_id = $2::uuid AND removed_at IS NULL`,
		messageID, userID)
	return err
}

// ListActiveReactions 列出生效中的反應列（依建立時間順序）。
func (s *PGStore) ListActiveReactions(ctx context.Context, messageID string) ([]*Reaction, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+reactionColumns+` FROM message_reactions
		 WHERE message_id = $1::uuid AND removed_at IS NULL ORDER BY created_at, id::text`,
		messageID)
	if err != nil {
		return nil, err
	}
	return scanReactionRows(rows)
}

const pinColumns = `id::text, conversation_id::text, message_id::text, pinned_by::text, pinned_at, unpinned_at, created_at, updated_at`

func scanPin(row postgres.Row) (*Pin, error) {
	var p Pin
	var unpinnedAt sql.NullTime
	if err := row.Scan(&p.ID, &p.ConversationID, &p.MessageID, &p.PinnedBy, &p.PinnedAt, &unpinnedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	p.UnpinnedAt = postgres.TimePtrFromNull(unpinnedAt)
	return &p, nil
}

func scanPinRows(rows postgres.Rows) ([]*Pin, error) {
	defer func() { _ = rows.Close() }()
	var out []*Pin
	for rows.Next() {
		var p Pin
		var unpinnedAt sql.NullTime
		if err := rows.Scan(&p.ID, &p.ConversationID, &p.MessageID, &p.PinnedBy, &p.PinnedAt, &unpinnedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.UnpinnedAt = postgres.TimePtrFromNull(unpinnedAt)
		out = append(out, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) getActivePin(ctx context.Context, conversationID, messageID string) (*Pin, error) {
	return scanPin(s.q.QueryRowContext(ctx,
		`SELECT `+pinColumns+` FROM message_pins
		 WHERE conversation_id = $1::uuid AND message_id = $2::uuid AND unpinned_at IS NULL`,
		conversationID, messageID))
}

// GetActivePin 讀取對話中某訊息的生效置頂列；未置頂時回傳 (nil, nil)。
func (s *PGStore) GetActivePin(ctx context.Context, conversationID, messageID string) (*Pin, error) {
	return s.getActivePin(ctx, conversationID, messageID)
}

// CountActivePins 回傳對話的生效置頂數。
func (s *PGStore) CountActivePins(ctx context.Context, conversationID string) (int, error) {
	row := s.q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM message_pins
		 WHERE conversation_id = $1::uuid AND unpinned_at IS NULL`, conversationID)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// PinMessage 寫入置頂列；已置頂時回傳既有列與 isNew=false。
// 併發寫入命中部分唯一索引時重讀生效列收斂，不得新建重複列。
func (s *PGStore) PinMessage(ctx context.Context, p *Pin) (*Pin, bool, error) {
	existing, err := s.getActivePin(ctx, p.ConversationID, p.MessageID)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		return existing, false, nil
	}
	_, err = s.q.ExecContext(ctx,
		`INSERT INTO message_pins (id, conversation_id, message_id, pinned_by, pinned_at, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7)`,
		p.ID, p.ConversationID, p.MessageID, p.PinnedBy, p.PinnedAt, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		if postgres.IsUniqueViolation(err) &&
			postgres.UniqueConstraintName(err) == "message_pins_active_unique_idx" {
			existing, readErr := s.getActivePin(ctx, p.ConversationID, p.MessageID)
			if readErr != nil {
				return nil, false, readErr
			}
			if existing != nil {
				return existing, false, nil
			}
		}
		return nil, false, err
	}
	return p, true, nil
}

// UnpinMessage 取消置頂（冪等）：回傳被取消的列；已取消時回傳
// 最近一筆歷史列；從未置頂時回傳 (nil, nil)。
func (s *PGStore) UnpinMessage(ctx context.Context, conversationID, messageID string, at time.Time) (*Pin, error) {
	row := s.q.QueryRowContext(ctx,
		`UPDATE message_pins SET unpinned_at = $3, updated_at = $3
		 WHERE conversation_id = $1::uuid AND message_id = $2::uuid AND unpinned_at IS NULL
		 RETURNING `+pinColumns,
		conversationID, messageID, at)
	p, err := scanPin(row)
	if err != nil {
		return nil, err
	}
	if p != nil {
		return p, nil
	}
	return scanPin(s.q.QueryRowContext(ctx,
		`SELECT `+pinColumns+` FROM message_pins
		 WHERE conversation_id = $1::uuid AND message_id = $2::uuid
		 ORDER BY pinned_at DESC, id::text DESC LIMIT 1`,
		conversationID, messageID))
}

// ListActivePins 列出生效中的置頂列（依置頂時間順序）。
func (s *PGStore) ListActivePins(ctx context.Context, conversationID string) ([]*Pin, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+pinColumns+` FROM message_pins
		 WHERE conversation_id = $1::uuid AND unpinned_at IS NULL ORDER BY pinned_at, id::text`,
		conversationID)
	if err != nil {
		return nil, err
	}
	return scanPinRows(rows)
}
