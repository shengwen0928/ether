// PostgreSQL 儲存實作：sticker.Store。
//
// 對映規則（schema 見 migrations/000014）：uuid 一律 ::text 讀出／
// ::uuid 傳入；可空欄位以 NULL 語義（見 internal/postgres 慣例）；
// 查詢一律具名欄位列。查無列時 Get* 回傳 (nil, nil)，由服務層決定
// 404 語義（不得洩漏存在性）。游標為服務層編碼的排序鍵不透明字串，
// 儲存層解碼後以元組比較分頁，格式錯誤時回傳錯誤，不得靜默接受。
package sticker

import (
	"context"
	"database/sql"

	"ether.local/api/internal/postgres"
)

// PGStore 是 Store 的 PostgreSQL 實作。
type PGStore struct{ q postgres.Querier }

// NewPGStore 建立 PGStore。
func NewPGStore(q postgres.Querier) *PGStore { return &PGStore{q: q} }

const packColumns = `id::text, creator_id::text, name, description, category, cover_media_id::text, is_free, status, created_at, updated_at`

func scanPack(row postgres.Row) (*Pack, error) {
	var p Pack
	var category, coverID sql.NullString
	var status string
	if err := row.Scan(&p.ID, &p.CreatorID, &p.Name, &p.Description, &category, &coverID, &p.IsFree, &status, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	p.Category = PackCategory(postgres.StringFromNull(category))
	p.CoverMediaID = postgres.StringFromNull(coverID)
	p.Status = PackStatus(status)
	return &p, nil
}

func scanPackRows(rows postgres.Rows) ([]*Pack, error) {
	defer func() { _ = rows.Close() }()
	var out []*Pack
	for rows.Next() {
		var p Pack
		var category, coverID sql.NullString
		var status string
		if err := rows.Scan(&p.ID, &p.CreatorID, &p.Name, &p.Description, &category, &coverID, &p.IsFree, &status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Category = PackCategory(postgres.StringFromNull(category))
		p.CoverMediaID = postgres.StringFromNull(coverID)
		p.Status = PackStatus(status)
		out = append(out, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetPack(ctx context.Context, id string) (*Pack, error) {
	row := s.q.QueryRowContext(ctx, `SELECT `+packColumns+` FROM sticker_packs WHERE id = $1::uuid`, id)
	return scanPack(row)
}

func (s *PGStore) CreatePack(ctx context.Context, p *Pack) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO sticker_packs (id, creator_id, name, description, category, cover_media_id, is_free, status, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::uuid, $7, $8, $9, $10)`,
		p.ID, p.CreatorID, p.Name, p.Description, postgres.NullString(string(p.Category)), postgres.NullString(p.CoverMediaID), p.IsFree, string(p.Status), p.CreatedAt, p.UpdatedAt)
	return err
}

func (s *PGStore) SavePack(ctx context.Context, p *Pack) error {
	res, err := s.q.ExecContext(ctx, `UPDATE sticker_packs SET name = $2, description = $3, category = $4, cover_media_id = $5::uuid, is_free = $6, status = $7, updated_at = $8
		WHERE id = $1::uuid`,
		p.ID, p.Name, p.Description, postgres.NullString(string(p.Category)), postgres.NullString(p.CoverMediaID), p.IsFree, string(p.Status), p.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrPackNotFound
	}
	return nil
}

func (s *PGStore) ListPacks(ctx context.Context, cursor string, limit int) ([]*Pack, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	var rows postgres.Rows
	if cursor == "" {
		rows, err = s.q.QueryContext(ctx, `SELECT `+packColumns+` FROM sticker_packs
			WHERE status <> 'archived' ORDER BY created_at DESC, id DESC LIMIT $1`, limit)
	} else {
		rows, err = s.q.QueryContext(ctx, `SELECT `+packColumns+` FROM sticker_packs
			WHERE status <> 'archived' AND (created_at, id) < ($1, $2::uuid)
			ORDER BY created_at DESC, id DESC LIMIT $3`, bound, boundID, limit)
	}
	if err != nil {
		return nil, err
	}
	return scanPackRows(rows)
}

func (s *PGStore) ListPacksByCreator(ctx context.Context, creatorID, cursor string, limit int) ([]*Pack, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	var rows postgres.Rows
	if cursor == "" {
		rows, err = s.q.QueryContext(ctx, `SELECT `+packColumns+` FROM sticker_packs
			WHERE status <> 'archived' AND creator_id = $1::uuid
			ORDER BY created_at DESC, id DESC LIMIT $2`, creatorID, limit)
	} else {
		rows, err = s.q.QueryContext(ctx, `SELECT `+packColumns+` FROM sticker_packs
			WHERE status <> 'archived' AND creator_id = $1::uuid AND (created_at, id) < ($2, $3::uuid)
			ORDER BY created_at DESC, id DESC LIMIT $4`, creatorID, bound, boundID, limit)
	}
	if err != nil {
		return nil, err
	}
	return scanPackRows(rows)
}

const stickerColumns = `id::text, sticker_pack_id::text, media_file_id::text, name, is_animated, position, status, created_at, updated_at`

func scanSticker(row postgres.Row) (*Sticker, error) {
	var st Sticker
	var status string
	if err := row.Scan(&st.ID, &st.PackID, &st.MediaFileID, &st.Name, &st.IsAnimated, &st.Position, &status, &st.CreatedAt, &st.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	st.Status = StickerStatus(status)
	return &st, nil
}

func scanStickerRows(rows postgres.Rows) ([]*Sticker, error) {
	defer func() { _ = rows.Close() }()
	var out []*Sticker
	for rows.Next() {
		var st Sticker
		var status string
		if err := rows.Scan(&st.ID, &st.PackID, &st.MediaFileID, &st.Name, &st.IsAnimated, &st.Position, &status, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, err
		}
		st.Status = StickerStatus(status)
		out = append(out, &st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetSticker(ctx context.Context, id string) (*Sticker, error) {
	row := s.q.QueryRowContext(ctx, `SELECT `+stickerColumns+` FROM stickers WHERE id = $1::uuid`, id)
	return scanSticker(row)
}

func (s *PGStore) CreateSticker(ctx context.Context, st *Sticker) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO stickers (id, sticker_pack_id, media_file_id, name, is_animated, position, status, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9)`,
		st.ID, st.PackID, st.MediaFileID, st.Name, st.IsAnimated, st.Position, string(st.Status), st.CreatedAt, st.UpdatedAt)
	return err
}

func (s *PGStore) SaveSticker(ctx context.Context, st *Sticker) error {
	res, err := s.q.ExecContext(ctx, `UPDATE stickers SET name = $2, is_animated = $3, position = $4, status = $5, updated_at = $6
		WHERE id = $1::uuid`,
		st.ID, st.Name, st.IsAnimated, st.Position, string(st.Status), st.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrStickerNotFound
	}
	return nil
}

func (s *PGStore) ListStickersByPack(ctx context.Context, packID string) ([]*Sticker, error) {
	rows, err := s.q.QueryContext(ctx, `SELECT `+stickerColumns+` FROM stickers
		WHERE sticker_pack_id = $1::uuid ORDER BY position ASC, id ASC`, packID)
	if err != nil {
		return nil, err
	}
	return scanStickerRows(rows)
}

func (s *PGStore) NextStickerPosition(ctx context.Context, packID string) (int, error) {
	var max sql.NullInt64
	row := s.q.QueryRowContext(ctx, `SELECT MAX(position) FROM stickers WHERE sticker_pack_id = $1::uuid`, packID)
	if err := row.Scan(&max); err != nil {
		return 0, err
	}
	if !max.Valid {
		return 0, nil
	}
	return int(max.Int64) + 1, nil
}

func (s *PGStore) CountActiveStickers(ctx context.Context, packID string) (int, error) {
	var n int
	row := s.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM stickers WHERE sticker_pack_id = $1::uuid AND status = 'active'`, packID)
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *PGStore) ListAllStickers(ctx context.Context, cursor string, limit int) ([]*Sticker, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	var rows postgres.Rows
	if cursor == "" {
		rows, err = s.q.QueryContext(ctx, `SELECT `+stickerColumns+` FROM stickers
			WHERE status = 'active' ORDER BY created_at DESC, id DESC LIMIT $1`, limit)
	} else {
		rows, err = s.q.QueryContext(ctx, `SELECT `+stickerColumns+` FROM stickers
			WHERE status = 'active' AND (created_at, id) < ($1, $2::uuid)
			ORDER BY created_at DESC, id DESC LIMIT $3`, bound, boundID, limit)
	}
	if err != nil {
		return nil, err
	}
	return scanStickerRows(rows)
}

const userStickerColumns = `id::text, user_id::text, sticker_pack_id::text, sticker_id::text, source, last_used_at, is_favorite, created_at, updated_at`

func scanUserSticker(row postgres.Row) (*UserSticker, error) {
	var u UserSticker
	var packID, stickerID sql.NullString
	var source string
	var lastUsedAt sql.NullTime
	if err := row.Scan(&u.ID, &u.UserID, &packID, &stickerID, &source, &lastUsedAt, &u.IsFavorite, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	u.PackID = postgres.StringFromNull(packID)
	u.StickerID = postgres.StringFromNull(stickerID)
	u.Source = UserStickerSource(source)
	u.LastUsedAt = postgres.TimePtrFromNull(lastUsedAt)
	return &u, nil
}

func scanUserStickerRows(rows postgres.Rows) ([]*UserSticker, error) {
	defer func() { _ = rows.Close() }()
	var out []*UserSticker
	for rows.Next() {
		var u UserSticker
		var packID, stickerID sql.NullString
		var source string
		var lastUsedAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.UserID, &packID, &stickerID, &source, &lastUsedAt, &u.IsFavorite, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.PackID = postgres.StringFromNull(packID)
		u.StickerID = postgres.StringFromNull(stickerID)
		u.Source = UserStickerSource(source)
		u.LastUsedAt = postgres.TimePtrFromNull(lastUsedAt)
		out = append(out, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetUserStickerByPack(ctx context.Context, userID, packID string) (*UserSticker, error) {
	row := s.q.QueryRowContext(ctx, `SELECT `+userStickerColumns+` FROM user_stickers
		WHERE user_id = $1::uuid AND sticker_pack_id = $2::uuid`, userID, packID)
	return scanUserSticker(row)
}

func (s *PGStore) GetUserStickerBySticker(ctx context.Context, userID, stickerID string) (*UserSticker, error) {
	row := s.q.QueryRowContext(ctx, `SELECT `+userStickerColumns+` FROM user_stickers
		WHERE user_id = $1::uuid AND sticker_id = $2::uuid`, userID, stickerID)
	return scanUserSticker(row)
}

func (s *PGStore) SaveUserSticker(ctx context.Context, u *UserSticker) error {
	// 包級列與單張級列各以其部分唯一索引為 upsert 仲裁子：衝突時
	// 保留舊列識別碼、更新來源／時間／旗標（服務層載入舊列後寫入，
	// 識別碼本就穩定）。CHECK 約束保證一列只關聯包或單張其中之一。
	if u.PackID != "" {
		_, err := s.q.ExecContext(ctx, `INSERT INTO user_stickers (id, user_id, sticker_pack_id, sticker_id, source, last_used_at, is_favorite, created_at, updated_at)
			VALUES ($1::uuid, $2::uuid, $3::uuid, NULL, $4, $5, $6, $7, $8)
			ON CONFLICT (user_id, sticker_pack_id) WHERE sticker_pack_id IS NOT NULL
				DO UPDATE SET source = EXCLUDED.source, last_used_at = EXCLUDED.last_used_at, is_favorite = EXCLUDED.is_favorite, updated_at = EXCLUDED.updated_at`,
			u.ID, u.UserID, u.PackID, string(u.Source), u.LastUsedAt, u.IsFavorite, u.CreatedAt, u.UpdatedAt)
		return err
	}
	_, err := s.q.ExecContext(ctx, `INSERT INTO user_stickers (id, user_id, sticker_pack_id, sticker_id, source, last_used_at, is_favorite, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, NULL, $3::uuid, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, sticker_id) WHERE sticker_id IS NOT NULL
			DO UPDATE SET source = EXCLUDED.source, last_used_at = EXCLUDED.last_used_at, is_favorite = EXCLUDED.is_favorite, updated_at = EXCLUDED.updated_at`,
		u.ID, u.UserID, u.StickerID, string(u.Source), u.LastUsedAt, u.IsFavorite, u.CreatedAt, u.UpdatedAt)
	return err
}

func (s *PGStore) DeleteUserSticker(ctx context.Context, id string) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM user_stickers WHERE id = $1::uuid`, id)
	return err
}

func (s *PGStore) ListFavorites(ctx context.Context, userID, cursor string, limit int) ([]*UserSticker, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	var rows postgres.Rows
	if cursor == "" {
		rows, err = s.q.QueryContext(ctx, `SELECT `+userStickerColumns+` FROM user_stickers
			WHERE user_id = $1::uuid AND is_favorite AND sticker_pack_id IS NOT NULL
			ORDER BY updated_at DESC, id DESC LIMIT $2`, userID, limit)
	} else {
		rows, err = s.q.QueryContext(ctx, `SELECT `+userStickerColumns+` FROM user_stickers
			WHERE user_id = $1::uuid AND is_favorite AND sticker_pack_id IS NOT NULL AND (updated_at, id) < ($2, $3::uuid)
			ORDER BY updated_at DESC, id DESC LIMIT $4`, userID, bound, boundID, limit)
	}
	if err != nil {
		return nil, err
	}
	return scanUserStickerRows(rows)
}

func (s *PGStore) ListRecent(ctx context.Context, userID, cursor string, limit int) ([]*UserSticker, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	var rows postgres.Rows
	if cursor == "" {
		rows, err = s.q.QueryContext(ctx, `SELECT `+userStickerColumns+` FROM user_stickers
			WHERE user_id = $1::uuid AND source = 'recent' AND last_used_at IS NOT NULL
			ORDER BY last_used_at DESC, id DESC LIMIT $2`, userID, limit)
	} else {
		rows, err = s.q.QueryContext(ctx, `SELECT `+userStickerColumns+` FROM user_stickers
			WHERE user_id = $1::uuid AND source = 'recent' AND last_used_at IS NOT NULL AND (last_used_at, id) < ($2, $3::uuid)
			ORDER BY last_used_at DESC, id DESC LIMIT $4`, userID, bound, boundID, limit)
	}
	if err != nil {
		return nil, err
	}
	return scanUserStickerRows(rows)
}

func (s *PGStore) PruneRecent(ctx context.Context, userID string, keep int) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM user_stickers WHERE id IN (
			SELECT id FROM user_stickers
			WHERE user_id = $1::uuid AND source = 'recent'
			ORDER BY last_used_at DESC, id DESC OFFSET $2)`, userID, keep)
	return err
}
