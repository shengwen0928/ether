// PostgreSQL 儲存實作：timeline.Store。
//
// 對映規則（schema 見 migrations/000012）：uuid 一律 ::text 讀出／
// ::uuid 傳入；可空欄位以 NULL 語義（見 internal/postgres 慣例）；
// 查詢一律具名欄位列。查無列時 Get* 回傳 (nil, nil)，由服務層決定
// 404 語義（不得洩漏存在性）。游標為服務層編碼的 (created_at, id)
// 不透明字串，儲存層解碼後以元組比較分頁，格式錯誤時回傳錯誤，
// 不得靜默接受。
package timeline

import (
	"context"
	"database/sql"

	"ether.local/api/internal/postgres"
)

// PGStore 是 Store 的 PostgreSQL 實作。
type PGStore struct{ q postgres.Querier }

// NewPGStore 建立 PGStore。
func NewPGStore(q postgres.Querier) *PGStore { return &PGStore{q: q} }

const postColumns = `id::text, author_id::text, content, visibility, edited_at, is_deleted, deleted_at, created_at, updated_at`

func scanPost(row postgres.Row) (*Post, error) {
	var p Post
	var visibility string
	var editedAt, deletedAt sql.NullTime
	if err := row.Scan(&p.ID, &p.AuthorID, &p.Content, &visibility, &editedAt, &p.IsDeleted, &deletedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	p.Visibility = Visibility(visibility)
	p.EditedAt = postgres.TimePtrFromNull(editedAt)
	p.DeletedAt = postgres.TimePtrFromNull(deletedAt)
	return &p, nil
}

func scanPostRows(rows postgres.Rows) ([]*Post, error) {
	defer func() { _ = rows.Close() }()
	var out []*Post
	for rows.Next() {
		var p Post
		var visibility string
		var editedAt, deletedAt sql.NullTime
		if err := rows.Scan(&p.ID, &p.AuthorID, &p.Content, &visibility, &editedAt, &p.IsDeleted, &deletedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Visibility = Visibility(visibility)
		p.EditedAt = postgres.TimePtrFromNull(editedAt)
		p.DeletedAt = postgres.TimePtrFromNull(deletedAt)
		out = append(out, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetPost(ctx context.Context, id string) (*Post, error) {
	row := s.q.QueryRowContext(ctx, `SELECT `+postColumns+` FROM posts WHERE id = $1::uuid`, id)
	return scanPost(row)
}

func (s *PGStore) CreatePost(ctx context.Context, p *Post) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO posts (id, author_id, content, visibility, edited_at, is_deleted, deleted_at, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9)`,
		p.ID, p.AuthorID, p.Content, string(p.Visibility), p.EditedAt, p.IsDeleted, p.DeletedAt, p.CreatedAt, p.UpdatedAt)
	return err
}

func (s *PGStore) SavePost(ctx context.Context, p *Post) error {
	res, err := s.q.ExecContext(ctx, `UPDATE posts SET content = $2, visibility = $3, edited_at = $4, is_deleted = $5, deleted_at = $6, updated_at = $7
		WHERE id = $1::uuid`,
		p.ID, p.Content, string(p.Visibility), p.EditedAt, p.IsDeleted, p.DeletedAt, p.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrPostNotFound
	}
	return nil
}

func (s *PGStore) ListRecentPosts(ctx context.Context, cursor string, limit int) ([]*Post, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	var rows postgres.Rows
	if cursor == "" {
		rows, err = s.q.QueryContext(ctx, `SELECT `+postColumns+` FROM posts
			WHERE NOT is_deleted ORDER BY created_at DESC, id DESC LIMIT $1`, limit)
	} else {
		rows, err = s.q.QueryContext(ctx, `SELECT `+postColumns+` FROM posts
			WHERE NOT is_deleted AND (created_at, id) < ($1, $2::uuid)
			ORDER BY created_at DESC, id DESC LIMIT $3`, bound, boundID, limit)
	}
	if err != nil {
		return nil, err
	}
	return scanPostRows(rows)
}

func (s *PGStore) ListPostsByAuthor(ctx context.Context, authorID, cursor string, limit int) ([]*Post, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	var rows postgres.Rows
	if cursor == "" {
		rows, err = s.q.QueryContext(ctx, `SELECT `+postColumns+` FROM posts
			WHERE NOT is_deleted AND author_id = $1::uuid
			ORDER BY created_at DESC, id DESC LIMIT $2`, authorID, limit)
	} else {
		rows, err = s.q.QueryContext(ctx, `SELECT `+postColumns+` FROM posts
			WHERE NOT is_deleted AND author_id = $1::uuid AND (created_at, id) < ($2, $3::uuid)
			ORDER BY created_at DESC, id DESC LIMIT $4`, authorID, bound, boundID, limit)
	}
	if err != nil {
		return nil, err
	}
	return scanPostRows(rows)
}

const postMediaColumns = `id::text, post_id::text, media_file_id::text, position, caption, created_at`

func (s *PGStore) GetPostMedia(ctx context.Context, postID string) ([]*PostMedia, error) {
	rows, err := s.q.QueryContext(ctx, `SELECT `+postMediaColumns+` FROM post_media
		WHERE post_id = $1::uuid ORDER BY position ASC, id ASC`, postID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*PostMedia
	for rows.Next() {
		var m PostMedia
		var caption sql.NullString
		if err := rows.Scan(&m.ID, &m.PostID, &m.MediaFileID, &m.Position, &caption, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Caption = postgres.StringFromNull(caption)
		out = append(out, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) CreatePostMedia(ctx context.Context, m *PostMedia) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO post_media (id, post_id, media_file_id, position, caption, created_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6)`,
		m.ID, m.PostID, m.MediaFileID, m.Position, postgres.NullString(m.Caption), m.CreatedAt)
	return err
}

func (s *PGStore) GetLike(ctx context.Context, postID, userID string) (*Like, error) {
	var l Like
	var unlikedAt sql.NullTime
	row := s.q.QueryRowContext(ctx, `SELECT post_id::text, user_id::text, liked_at, unliked_at, created_at, updated_at
		FROM post_likes WHERE post_id = $1::uuid AND user_id = $2::uuid`, postID, userID)
	if err := row.Scan(&l.PostID, &l.UserID, &l.LikedAt, &unlikedAt, &l.CreatedAt, &l.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	l.UnlikedAt = postgres.TimePtrFromNull(unlikedAt)
	return &l, nil
}

func (s *PGStore) SaveLike(ctx context.Context, l *Like) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO post_likes (post_id, user_id, liked_at, unliked_at, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
		ON CONFLICT (post_id, user_id) DO UPDATE SET liked_at = EXCLUDED.liked_at, unliked_at = EXCLUDED.unliked_at, updated_at = EXCLUDED.updated_at`,
		l.PostID, l.UserID, l.LikedAt, l.UnlikedAt, l.CreatedAt, l.UpdatedAt)
	return err
}

func (s *PGStore) CountActiveLikes(ctx context.Context, postID string) (int, error) {
	var n int
	row := s.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM post_likes WHERE post_id = $1::uuid AND unliked_at IS NULL`, postID)
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

const commentColumns = `id::text, post_id::text, author_id::text, parent_comment_id::text, content, edited_at, is_deleted, deleted_at, created_at, updated_at`

func scanComment(row postgres.Row) (*Comment, error) {
	var c Comment
	var parentID sql.NullString
	var editedAt, deletedAt sql.NullTime
	if err := row.Scan(&c.ID, &c.PostID, &c.AuthorID, &parentID, &c.Content, &editedAt, &c.IsDeleted, &deletedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	c.ParentCommentID = postgres.StringFromNull(parentID)
	c.EditedAt = postgres.TimePtrFromNull(editedAt)
	c.DeletedAt = postgres.TimePtrFromNull(deletedAt)
	return &c, nil
}

func (s *PGStore) GetComment(ctx context.Context, id string) (*Comment, error) {
	row := s.q.QueryRowContext(ctx, `SELECT `+commentColumns+` FROM post_comments WHERE id = $1::uuid`, id)
	return scanComment(row)
}

func (s *PGStore) CreateComment(ctx context.Context, c *Comment) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO post_comments (id, post_id, author_id, parent_comment_id, content, edited_at, is_deleted, deleted_at, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9, $10)`,
		c.ID, c.PostID, c.AuthorID, postgres.NullString(c.ParentCommentID), c.Content, c.EditedAt, c.IsDeleted, c.DeletedAt, c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *PGStore) SaveComment(ctx context.Context, c *Comment) error {
	res, err := s.q.ExecContext(ctx, `UPDATE post_comments SET content = $2, edited_at = $3, is_deleted = $4, deleted_at = $5, updated_at = $6
		WHERE id = $1::uuid`,
		c.ID, c.Content, c.EditedAt, c.IsDeleted, c.DeletedAt, c.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrCommentNotFound
	}
	return nil
}

func (s *PGStore) ListComments(ctx context.Context, postID, cursor string, limit int) ([]*Comment, error) {
	bound, boundID, err := cursorBound(cursor)
	if err != nil {
		return nil, err
	}
	var rows postgres.Rows
	if cursor == "" {
		rows, err = s.q.QueryContext(ctx, `SELECT `+commentColumns+` FROM post_comments
			WHERE post_id = $1::uuid ORDER BY created_at ASC, id ASC LIMIT $2`, postID, limit)
	} else {
		rows, err = s.q.QueryContext(ctx, `SELECT `+commentColumns+` FROM post_comments
			WHERE post_id = $1::uuid AND (created_at, id) > ($2, $3::uuid)
			ORDER BY created_at ASC, id ASC LIMIT $4`, postID, bound, boundID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Comment
	for rows.Next() {
		var c Comment
		var parentID sql.NullString
		var editedAt, deletedAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.PostID, &c.AuthorID, &parentID, &c.Content, &editedAt, &c.IsDeleted, &deletedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.ParentCommentID = postgres.StringFromNull(parentID)
		c.EditedAt = postgres.TimePtrFromNull(editedAt)
		c.DeletedAt = postgres.TimePtrFromNull(deletedAt)
		out = append(out, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) CountVisibleComments(ctx context.Context, postID string) (int, error) {
	var n int
	row := s.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM post_comments WHERE post_id = $1::uuid AND NOT is_deleted`, postID)
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
