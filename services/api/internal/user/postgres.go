// PostgreSQL 儲存實作（M2b-1）：UserStore 與 SettingsStore。
//
// 對映規則（schema 見 migrations/000002＋000003）：
//   - users.keycloak_subject（000003 補正）為穩定綁定鍵，NOT NULL
//     UNIQUE；email 可空（NULL＝未提供），email_verified 承載驗證狀態。
//   - users.phone_number 欄位存在但本階段不寫入（安全性架構未定稿），
//     寫入一律留 NULL，讀出不回傳。
//   - user_settings 的 media_auto_download 欄位不屬於領域模型，寫入
//     與更新皆不觸碰（留資料庫預設／既有值）。
//   - 唯一性違反：僅 email 衝突對映為 ValidationError（與 Provision
//     的事前檢查語義一致，含競態下的行為一致）；subject／主鍵衝突
//     屬伺服器異常，原樣回傳（500），不得包裝成欄位錯誤。
package user

import (
	"context"
	"database/sql"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/transport"
)

// PGUserStore 是 UserStore 的 PostgreSQL 實作。
type PGUserStore struct{ q postgres.Querier }

// NewPGUserStore 建立 PGUserStore。q 須為已連接資料庫的 Querier。
func NewPGUserStore(q postgres.Querier) *PGUserStore { return &PGUserStore{q: q} }

const userColumns = `id::text, keycloak_subject, email, email_verified, status, registered_at, last_login_at, created_at, updated_at`

func scanUser(row postgres.Row) (*User, error) {
	var u User
	var email sql.NullString
	var lastLogin sql.NullTime
	var status string
	if err := row.Scan(&u.ID, &u.KeycloakSubject, &email, &u.EmailVerified, &status, &u.RegisteredAt, &lastLogin, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	u.Email = postgres.StringFromNull(email)
	u.LastLoginAt = postgres.TimePtrFromNull(lastLogin)
	u.Status = Status(status)
	return &u, nil
}

func (s *PGUserStore) GetByID(ctx context.Context, id string) (*User, error) {
	return scanUser(s.q.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1::uuid`, id))
}

func (s *PGUserStore) GetBySubject(ctx context.Context, subject string) (*User, error) {
	return scanUser(s.q.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE keycloak_subject = $1`, subject))
}

func (s *PGUserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	if email == "" {
		return nil, ErrUserNotFound
	}
	return scanUser(s.q.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1`, email))
}

func mapUserUniqueError(err error) error {
	if postgres.IsUniqueViolation(err) && postgres.UniqueConstraintName(err) == "users_email_key" {
		return &ValidationError{Fields: []transport.FieldError{{
			Field:   "email",
			Issue:   "DUPLICATE",
			Message: "Email 已被其他帳號使用。",
		}}}
	}
	return err
}

func (s *PGUserStore) Create(ctx context.Context, u *User) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO users (id, keycloak_subject, email, email_verified, status, registered_at, last_login_at, created_at, updated_at)
		 VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9)`,
		u.ID, u.KeycloakSubject, postgres.NullString(u.Email), u.EmailVerified,
		string(u.Status), u.RegisteredAt, postgres.NullTime(u.LastLoginAt),
		u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return mapUserUniqueError(err)
	}
	return nil
}

func (s *PGUserStore) Save(ctx context.Context, u *User) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE users SET keycloak_subject = $2, email = $3, email_verified = $4,
		 status = $5, last_login_at = $6, updated_at = $7
		 WHERE id = $1::uuid`,
		u.ID, u.KeycloakSubject, postgres.NullString(u.Email), u.EmailVerified,
		string(u.Status), postgres.NullTime(u.LastLoginAt), u.UpdatedAt)
	if err != nil {
		return mapUserUniqueError(err)
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

// PGSettingsStore 是 SettingsStore 的 PostgreSQL 實作。
type PGSettingsStore struct{ q postgres.Querier }

// NewPGSettingsStore 建立 PGSettingsStore。
func NewPGSettingsStore(q postgres.Querier) *PGSettingsStore { return &PGSettingsStore{q: q} }

const settingsColumns = `user_id::text, language, theme, font_size, chat_background, created_at, updated_at`

func scanSettings(row postgres.Row) (*Settings, error) {
	var st Settings
	var fontSize, chatBackground sql.NullString
	if err := row.Scan(&st.UserID, &st.Language, &st.Theme, &fontSize, &chatBackground, &st.CreatedAt, &st.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	st.FontSize = postgres.StringFromNull(fontSize)
	st.ChatBackground = postgres.StringFromNull(chatBackground)
	return &st, nil
}

func (s *PGSettingsStore) GetByUserID(ctx context.Context, userID string) (*Settings, error) {
	return scanSettings(s.q.QueryRowContext(ctx,
		`SELECT `+settingsColumns+` FROM user_settings WHERE user_id = $1::uuid`, userID))
}

func (s *PGSettingsStore) Create(ctx context.Context, st *Settings) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO user_settings (user_id, language, theme, font_size, chat_background, created_at, updated_at)
		 VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)`,
		st.UserID, st.Language, st.Theme, postgres.NullString(st.FontSize),
		postgres.NullString(st.ChatBackground), st.CreatedAt, st.UpdatedAt)
	if err != nil {
		// user_id 唯一衝突或 users FK 違反皆屬伺服器異常（Provision
		// 先建使用者），原樣回傳，不得包裝成欄位錯誤。
		return err
	}
	return nil
}

func (s *PGSettingsStore) Save(ctx context.Context, st *Settings) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE user_settings SET language = $2, theme = $3, font_size = $4,
		 chat_background = $5, updated_at = $6
		 WHERE user_id = $1::uuid`,
		st.UserID, st.Language, st.Theme, postgres.NullString(st.FontSize),
		postgres.NullString(st.ChatBackground), st.UpdatedAt)
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

var (
	_ UserStore     = (*PGUserStore)(nil)
	_ SettingsStore = (*PGSettingsStore)(nil)
)
