// PostgreSQL 儲存實作（M3）：friend.Store。
//
// 對映規則（schema 見 migrations/000005）：friendships 以字典序
// 配對存放（user_id < friend_id），查詢一律先排序再比對；
// friend_requests 待處理唯一衝突（部分唯一索引
// friend_requests_pending_unique_idx）對映為 ValidationError
// （與 SendRequest 的事前檢查語義一致）；blocked_users／
// hidden_users 的解除為狀態標記，不物理刪除。查無列時
// GetRequest 回傳 ErrRequestNotFound；GetFriendship／
// GetActiveBlock／GetActiveHidden 查無時回傳 (nil, nil)，由
// 服務層決定 404 語義（不得洩漏存在性）。
package friend

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/transport"
)

// PGStore 是 Store 的 PostgreSQL 實作。
type PGStore struct{ q postgres.Querier }

// NewPGStore 建立 PGStore。
func NewPGStore(q postgres.Querier) *PGStore { return &PGStore{q: q} }

const requestColumns = `id::text, requester_id::text, addressee_id::text, status, message, expires_at, responded_at, created_at, updated_at`

func scanRequest(row postgres.Row) (*FriendRequest, error) {
	var r FriendRequest
	var status string
	var message sql.NullString
	var expires, responded sql.NullTime
	if err := row.Scan(&r.ID, &r.RequesterID, &r.AddresseeID, &status, &message, &expires, &responded, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}
	r.Status = RequestStatus(status)
	r.Message = postgres.StringFromNull(message)
	r.ExpiresAt = postgres.TimePtrFromNull(expires)
	r.RespondedAt = postgres.TimePtrFromNull(responded)
	return &r, nil
}

func (s *PGStore) GetRequest(ctx context.Context, id string) (*FriendRequest, error) {
	return scanRequest(s.q.QueryRowContext(ctx,
		`SELECT `+requestColumns+` FROM friend_requests WHERE id = $1::uuid`, id))
}

func (s *PGStore) GetPendingBetween(ctx context.Context, a, b string) (*FriendRequest, error) {
	row := s.q.QueryRowContext(ctx,
		`SELECT `+requestColumns+` FROM friend_requests
		 WHERE status = 'pending'
		 AND ((requester_id = $1::uuid AND addressee_id = $2::uuid)
		   OR (requester_id = $2::uuid AND addressee_id = $1::uuid))
		 ORDER BY created_at, id LIMIT 1`, a, b)
	r, err := scanRequest(row)
	if err != nil {
		if errors.Is(err, ErrRequestNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r, nil
}

func scanRequestRows(rows postgres.Rows) ([]*FriendRequest, error) {
	defer func() { _ = rows.Close() }()
	var out []*FriendRequest
	for rows.Next() {
		var r FriendRequest
		var status string
		var message sql.NullString
		var expires, responded sql.NullTime
		if err := rows.Scan(&r.ID, &r.RequesterID, &r.AddresseeID, &status, &message, &expires, &responded, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.Status = RequestStatus(status)
		r.Message = postgres.StringFromNull(message)
		r.ExpiresAt = postgres.TimePtrFromNull(expires)
		r.RespondedAt = postgres.TimePtrFromNull(responded)
		out = append(out, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) ListIncomingPending(ctx context.Context, userID string) ([]*FriendRequest, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+requestColumns+` FROM friend_requests
		 WHERE addressee_id = $1::uuid AND status = 'pending'
		 ORDER BY created_at, id`, userID)
	if err != nil {
		return nil, err
	}
	return scanRequestRows(rows)
}

func (s *PGStore) ListOutgoingPending(ctx context.Context, userID string) ([]*FriendRequest, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+requestColumns+` FROM friend_requests
		 WHERE requester_id = $1::uuid AND status = 'pending'
		 ORDER BY created_at, id`, userID)
	if err != nil {
		return nil, err
	}
	return scanRequestRows(rows)
}

func mapFriendUniqueError(err error) error {
	if postgres.IsUniqueViolation(err) {
		switch postgres.UniqueConstraintName(err) {
		case "friend_requests_pending_unique_idx":
			return &ValidationError{Fields: []transport.FieldError{{
				Field:   "addressee_id",
				Issue:   "DUPLICATE",
				Message: "已送出待處理的好友邀請。",
			}}}
		case "friendships_pair_unique":
			return &ValidationError{Fields: []transport.FieldError{{
				Field:   "addressee_id",
				Issue:   "DUPLICATE",
				Message: "對方已是好友。",
			}}}
		}
	}
	return err
}

func (s *PGStore) CreateRequest(ctx context.Context, req *FriendRequest) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO friend_requests (id, requester_id, addressee_id, status, message, expires_at, responded_at, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9)`,
		req.ID, req.RequesterID, req.AddresseeID, string(req.Status),
		postgres.NullString(req.Message), postgres.NullTime(req.ExpiresAt),
		postgres.NullTime(req.RespondedAt), req.CreatedAt, req.UpdatedAt)
	if err != nil {
		return mapFriendUniqueError(err)
	}
	return nil
}

func (s *PGStore) SaveRequest(ctx context.Context, req *FriendRequest) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE friend_requests SET status = $2, message = $3, expires_at = $4,
		 responded_at = $5, updated_at = $6 WHERE id = $1::uuid`,
		req.ID, string(req.Status), postgres.NullString(req.Message),
		postgres.NullTime(req.ExpiresAt), postgres.NullTime(req.RespondedAt), req.UpdatedAt)
	if err != nil {
		return mapFriendUniqueError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrRequestNotFound
	}
	return nil
}

func (s *PGStore) CancelPendingBetween(ctx context.Context, a, b string, at time.Time) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE friend_requests SET status = 'cancelled', responded_at = $3, updated_at = $3
		 WHERE status = 'pending'
		 AND ((requester_id = $1::uuid AND addressee_id = $2::uuid)
		   OR (requester_id = $2::uuid AND addressee_id = $1::uuid))`, a, b, at)
	return err
}

const friendshipColumns = `id::text, user_id::text, friend_id::text, status, established_at, ended_at, created_at, updated_at`

func scanFriendship(row postgres.Row) (*Friendship, error) {
	var f Friendship
	var status string
	var established, ended sql.NullTime
	if err := row.Scan(&f.ID, &f.FirstID, &f.SecondID, &status, &established, &ended, &f.CreatedAt, &f.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	f.Status = FriendshipStatus(status)
	f.EstablishedAt = postgres.TimePtrFromNull(established)
	f.EndedAt = postgres.TimePtrFromNull(ended)
	return &f, nil
}

func (s *PGStore) GetFriendship(ctx context.Context, a, b string) (*Friendship, error) {
	first, second := orderedPair(a, b)
	return scanFriendship(s.q.QueryRowContext(ctx,
		`SELECT `+friendshipColumns+` FROM friendships WHERE user_id = $1::uuid AND friend_id = $2::uuid`,
		first, second))
}

func (s *PGStore) ListFriendships(ctx context.Context, userID string) ([]*Friendship, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+friendshipColumns+` FROM friendships
		 WHERE status = 'active' AND (user_id = $1::uuid OR friend_id = $1::uuid)
		 ORDER BY created_at, id`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Friendship
	for rows.Next() {
		var f Friendship
		var status string
		var established, ended sql.NullTime
		if err := rows.Scan(&f.ID, &f.FirstID, &f.SecondID, &status, &established, &ended, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		f.Status = FriendshipStatus(status)
		f.EstablishedAt = postgres.TimePtrFromNull(established)
		f.EndedAt = postgres.TimePtrFromNull(ended)
		out = append(out, &f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) CreateFriendship(ctx context.Context, f *Friendship) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO friendships (id, user_id, friend_id, status, established_at, ended_at, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8)`,
		f.ID, f.FirstID, f.SecondID, string(f.Status),
		postgres.NullTime(f.EstablishedAt), postgres.NullTime(f.EndedAt),
		f.CreatedAt, f.UpdatedAt)
	if err != nil {
		return mapFriendUniqueError(err)
	}
	return nil
}

func (s *PGStore) SaveFriendship(ctx context.Context, f *Friendship) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE friendships SET status = $2, established_at = $3, ended_at = $4, updated_at = $5
		 WHERE id = $1::uuid`,
		f.ID, string(f.Status), postgres.NullTime(f.EstablishedAt),
		postgres.NullTime(f.EndedAt), f.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrFriendshipNotFound
	}
	return nil
}

const blockColumns = `id::text, user_id::text, blocked_user_id::text, reason, blocked_at, unblocked_at, created_at, updated_at`

func scanBlock(row postgres.Row) (*Block, error) {
	var b Block
	var reason sql.NullString
	var unblocked sql.NullTime
	if err := row.Scan(&b.ID, &b.UserID, &b.BlockedUserID, &reason, &b.BlockedAt, &unblocked, &b.CreatedAt, &b.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	b.Reason = postgres.StringFromNull(reason)
	b.UnblockedAt = postgres.TimePtrFromNull(unblocked)
	return &b, nil
}

func (s *PGStore) GetActiveBlock(ctx context.Context, userID, blockedID string) (*Block, error) {
	return scanBlock(s.q.QueryRowContext(ctx,
		`SELECT `+blockColumns+` FROM blocked_users
		 WHERE user_id = $1::uuid AND blocked_user_id = $2::uuid AND unblocked_at IS NULL
		 ORDER BY created_at DESC, id LIMIT 1`, userID, blockedID))
}

func (s *PGStore) BlockedBetween(ctx context.Context, a, b string) (bool, error) {
	var blocked bool
	row := s.q.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM blocked_users
		 WHERE unblocked_at IS NULL
		 AND ((user_id = $1::uuid AND blocked_user_id = $2::uuid)
		   OR (user_id = $2::uuid AND blocked_user_id = $1::uuid)))`, a, b)
	if err := row.Scan(&blocked); err != nil {
		return false, err
	}
	return blocked, nil
}

func (s *PGStore) ListBlocks(ctx context.Context, userID string) ([]*Block, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+blockColumns+` FROM blocked_users
		 WHERE user_id = $1::uuid AND unblocked_at IS NULL
		 ORDER BY created_at DESC, id`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Block
	for rows.Next() {
		var b Block
		var reason sql.NullString
		var unblocked sql.NullTime
		if err := rows.Scan(&b.ID, &b.UserID, &b.BlockedUserID, &reason, &b.BlockedAt, &unblocked, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		b.Reason = postgres.StringFromNull(reason)
		b.UnblockedAt = postgres.TimePtrFromNull(unblocked)
		out = append(out, &b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) CreateBlock(ctx context.Context, b *Block) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO blocked_users (id, user_id, blocked_user_id, reason, blocked_at, unblocked_at, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8)`,
		b.ID, b.UserID, b.BlockedUserID, postgres.NullString(b.Reason),
		b.BlockedAt, postgres.NullTime(b.UnblockedAt), b.CreatedAt, b.UpdatedAt)
	return err
}

func (s *PGStore) SaveBlock(ctx context.Context, b *Block) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE blocked_users SET reason = $2, blocked_at = $3, unblocked_at = $4, updated_at = $5
		 WHERE id = $1::uuid`,
		b.ID, postgres.NullString(b.Reason), b.BlockedAt,
		postgres.NullTime(b.UnblockedAt), b.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

const hiddenColumns = `id::text, user_id::text, hidden_user_id::text, hidden_at, unhidden_at, created_at, updated_at`

func scanHidden(row postgres.Row) (*Hidden, error) {
	var h Hidden
	var unhidden sql.NullTime
	if err := row.Scan(&h.ID, &h.UserID, &h.HiddenUserID, &h.HiddenAt, &unhidden, &h.CreatedAt, &h.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	h.UnhiddenAt = postgres.TimePtrFromNull(unhidden)
	return &h, nil
}

func (s *PGStore) GetActiveHidden(ctx context.Context, userID, hiddenID string) (*Hidden, error) {
	return scanHidden(s.q.QueryRowContext(ctx,
		`SELECT `+hiddenColumns+` FROM hidden_users
		 WHERE user_id = $1::uuid AND hidden_user_id = $2::uuid AND unhidden_at IS NULL
		 ORDER BY created_at DESC, id LIMIT 1`, userID, hiddenID))
}

func (s *PGStore) ListHiddenIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT hidden_user_id::text FROM hidden_users
		 WHERE user_id = $1::uuid AND unhidden_at IS NULL
		 ORDER BY created_at, id`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) CreateHidden(ctx context.Context, h *Hidden) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO hidden_users (id, user_id, hidden_user_id, hidden_at, unhidden_at, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)`,
		h.ID, h.UserID, h.HiddenUserID, h.HiddenAt,
		postgres.NullTime(h.UnhiddenAt), h.CreatedAt, h.UpdatedAt)
	return err
}

func (s *PGStore) SaveHidden(ctx context.Context, h *Hidden) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE hidden_users SET hidden_at = $2, unhidden_at = $3, updated_at = $4
		 WHERE id = $1::uuid`,
		h.ID, h.HiddenAt, postgres.NullTime(h.UnhiddenAt), h.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrFriendshipNotFound
	}
	return nil
}

var _ Store = (*PGStore)(nil)
