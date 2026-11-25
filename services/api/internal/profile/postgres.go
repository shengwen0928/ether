// PostgreSQL 儲存實作（M2b-1）：profile.Store。
//
// 對映規則（schema 見 migrations/000002）：user_profiles 的 id 主鍵
// 由資料庫產生，領域模型不持有（以 user_id 為鍵）；birthday／
// birthday_visibility／qr_code_token／profile_visibility 欄位不屬於
// 領域模型，寫入與更新皆不觸碰。public_id 唯一衝突對映為
// ValidationError（與 Update 的事前檢查語義一致）。
package profile

import (
	"context"
	"database/sql"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/transport"
)

// PGStore 是 Store 的 PostgreSQL 實作。
type PGStore struct{ q postgres.Querier }

// NewPGStore 建立 PGStore。
func NewPGStore(q postgres.Querier) *PGStore { return &PGStore{q: q} }

const profileColumns = `user_id::text, display_name, public_id, avatar_media_id, bio, status_message, created_at, updated_at`

func scanProfile(row postgres.Row) (*Profile, error) {
	var p Profile
	var publicID, avatar, bio, statusMsg sql.NullString
	if err := row.Scan(&p.UserID, &p.DisplayName, &publicID, &avatar, &bio, &statusMsg, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	p.PublicID = postgres.StringFromNull(publicID)
	p.AvatarMediaID = postgres.StringFromNull(avatar)
	p.Bio = postgres.StringFromNull(bio)
	p.StatusMessage = postgres.StringFromNull(statusMsg)
	return &p, nil
}

func (s *PGStore) GetByUserID(ctx context.Context, userID string) (*Profile, error) {
	return scanProfile(s.q.QueryRowContext(ctx,
		`SELECT `+profileColumns+` FROM user_profiles WHERE user_id = $1::uuid`, userID))
}

func (s *PGStore) GetByPublicID(ctx context.Context, publicID string) (*Profile, error) {
	if publicID == "" {
		return nil, ErrProfileNotFound
	}
	return scanProfile(s.q.QueryRowContext(ctx,
		`SELECT `+profileColumns+` FROM user_profiles WHERE public_id = $1`, publicID))
}

func mapProfileUniqueError(err error) error {
	if postgres.IsUniqueViolation(err) && postgres.UniqueConstraintName(err) == "user_profiles_public_id_key" {
		return &ValidationError{Fields: []transport.FieldError{{
			Field:   "public_id",
			Issue:   "DUPLICATE",
			Message: "公開識別碼已被使用。",
		}}}
	}
	return err
}

func (s *PGStore) Create(ctx context.Context, p *Profile) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO user_profiles (user_id, display_name, public_id, avatar_media_id, bio, status_message, created_at, updated_at)
		 VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)`,
		p.UserID, p.DisplayName, postgres.NullString(p.PublicID),
		postgres.NullString(p.AvatarMediaID), postgres.NullString(p.Bio),
		postgres.NullString(p.StatusMessage), p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return mapProfileUniqueError(err)
	}
	return nil
}

func (s *PGStore) Save(ctx context.Context, p *Profile) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE user_profiles SET display_name = $2, public_id = $3, avatar_media_id = $4,
		 bio = $5, status_message = $6, updated_at = $7
		 WHERE user_id = $1::uuid`,
		p.UserID, p.DisplayName, postgres.NullString(p.PublicID),
		postgres.NullString(p.AvatarMediaID), postgres.NullString(p.Bio),
		postgres.NullString(p.StatusMessage), p.UpdatedAt)
	if err != nil {
		return mapProfileUniqueError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrProfileNotFound
	}
	return nil
}

var _ Store = (*PGStore)(nil)
