// PostgreSQL 儲存實作（M2b-1）：auth.SessionStore。
//
// 對映規則（schema 見 migrations/000002）：
//   - user_sessions.session_token_hash 欄位不屬於領域模型，寫入留
//     NULL、讀出不取用（Session 參照由 Keycloak sid 承接）。
//   - 撤銷狀態以 revoked_at 為準寫入 revoked_at＋is_revoked 兩欄；
//     讀出時 revoked_at 為空但 is_revoked 為真（舊資料／不一致列）
//     時，以 updated_at 合成 RevokedAt，不得將該列誤判為有效。
//   - DeviceID／KeycloakSessionID 空字串 ↔ NULL；LastActiveAt
//     零值 ↔ NULL；RevokedReason 空字串 ↔ NULL。
//   - 已消耗 Refresh 雜湊存於 session_refresh_hashes 表（000004），
//     IsHashConsumed／MarkHashConsumed 供 Rotate 的重放偵測使用，
//     持久化後重啟仍有效（M1 僅行程內記憶體）。
package auth

import (
	"context"
	"database/sql"
	"time"

	"ether.local/api/internal/postgres"
)

// PGSessionStore 是 SessionStore 的 PostgreSQL 實作。
type PGSessionStore struct{ q postgres.Querier }

// NewPGSessionStore 建立 PGSessionStore。
func NewPGSessionStore(q postgres.Querier) *PGSessionStore { return &PGSessionStore{q: q} }

const sessionColumns = `id::text, user_id::text, device_id::text, keycloak_session_id, refresh_token_hash, expires_at, last_active_at, is_revoked, revoked_at, revoked_reason, created_at, updated_at`

func scanSession(row postgres.Row) (*Session, error) {
	var s Session
	var deviceID, kcSID, revokedReason sql.NullString
	var lastActive, revokedAt sql.NullTime
	var isRevoked bool
	if err := row.Scan(&s.ID, &s.UserID, &deviceID, &kcSID, &s.RefreshTokenHash,
		&s.ExpiresAt, &lastActive, &isRevoked, &revokedAt, &revokedReason,
		&s.CreatedAt, &s.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	s.DeviceID = postgres.StringFromNull(deviceID)
	s.KeycloakSessionID = postgres.StringFromNull(kcSID)
	s.RevokedReason = postgres.StringFromNull(revokedReason)
	if lastActive.Valid {
		s.LastActiveAt = lastActive.Time
	}
	s.RevokedAt = postgres.TimePtrFromNull(revokedAt)
	if s.RevokedAt == nil && isRevoked {
		// revoked_at 為空但 is_revoked 為真：以 updated_at 合成，
		// 確保 Revoked() 判讀與資料庫狀態一致。
		t := s.UpdatedAt
		s.RevokedAt = &t
	}
	return &s, nil
}

func (s *PGSessionStore) Create(ctx context.Context, sess *Session) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO user_sessions (id, user_id, device_id, keycloak_session_id, refresh_token_hash, expires_at, last_active_at, is_revoked, revoked_at, revoked_reason, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		sess.ID, sess.UserID, nullableUUID(sess.DeviceID), postgres.NullString(sess.KeycloakSessionID),
		sess.RefreshTokenHash, sess.ExpiresAt, nullableTime(sess.LastActiveAt),
		sess.Revoked(), postgres.NullTime(sess.RevokedAt), postgres.NullString(sess.RevokedReason),
		sess.CreatedAt, sess.UpdatedAt)
	return err
}

func (s *PGSessionStore) Get(ctx context.Context, id string) (*Session, error) {
	return scanSession(s.q.QueryRowContext(ctx,
		`SELECT `+sessionColumns+` FROM user_sessions WHERE id = $1::uuid`, id))
}

func (s *PGSessionStore) GetByKeycloakSessionID(ctx context.Context, sid string) (*Session, error) {
	if sid == "" {
		return nil, ErrSessionNotFound
	}
	return scanSession(s.q.QueryRowContext(ctx,
		`SELECT `+sessionColumns+` FROM user_sessions WHERE keycloak_session_id = $1 ORDER BY created_at DESC LIMIT 1`, sid))
}

func (s *PGSessionStore) ListByUser(ctx context.Context, userID string) ([]*Session, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+sessionColumns+` FROM user_sessions WHERE user_id = $1::uuid ORDER BY created_at ASC, id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGSessionStore) Save(ctx context.Context, sess *Session) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE user_sessions SET device_id = $2::uuid, keycloak_session_id = $3, refresh_token_hash = $4,
		 expires_at = $5, last_active_at = $6, is_revoked = $7, revoked_at = $8, revoked_reason = $9, updated_at = $10
		 WHERE id = $1::uuid`,
		sess.ID, nullableUUID(sess.DeviceID), postgres.NullString(sess.KeycloakSessionID),
		sess.RefreshTokenHash, sess.ExpiresAt, nullableTime(sess.LastActiveAt),
		sess.Revoked(), postgres.NullTime(sess.RevokedAt), postgres.NullString(sess.RevokedReason),
		sess.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrSessionNotFound
	}
	return nil
}

// IsHashConsumed 報告該雜湊是否曾被輪換消耗（重放偵測）。
func (s *PGSessionStore) IsHashConsumed(ctx context.Context, sessionID, hash string) (bool, error) {
	var exists bool
	err := s.q.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM session_refresh_hashes WHERE session_id = $1::uuid AND refresh_token_hash = $2)`,
		sessionID, hash).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// MarkHashConsumed 記錄雜湊已被輪換消耗；已記錄時回傳 false
// （冪等，不報錯）。
func (s *PGSessionStore) MarkHashConsumed(ctx context.Context, sessionID, hash string) (bool, error) {
	res, err := s.q.ExecContext(ctx,
		`INSERT INTO session_refresh_hashes (session_id, refresh_token_hash) VALUES ($1::uuid, $2) ON CONFLICT DO NOTHING`,
		sessionID, hash)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// nullableUUID 將空字串轉為 NULL（可空 uuid 欄位寫入用；非空時
// 由 ::uuid 轉型驗證格式）。
func nullableUUID(id string) any {
	if id == "" {
		return nil
	}
	return id
}

// nullableTime 將零值時間轉為 NULL（LastActiveAt 寫入用）。
func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

var _ SessionStore = (*PGSessionStore)(nil)
