// Package user 是 User（使用者）模組的 Phase 03 實作。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：使用者主檔（帳號本體）的建立、
// 帳號狀態、帳號層設定。擁有資料表：users、user_settings。
// 身分簽發屬 Keycloak（見 internal/auth）；本套件不驗證 Token、
// 不儲存密碼（ERD 草案的 password_hash 欄位刻意不在領域模型與
// migration 中落地，見 migration 000002 說明）。
package user

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"ether.local/api/internal/ids"
	"ether.local/api/internal/transport"
)

// ErrUserNotFound 表示查無使用者。對外一律 404，不得區分
// 「不存在」與「無權存取」。
var ErrUserNotFound = errors.New("user: not found")

// ErrInvalidStatusTransition 表示不允許的帳號狀態轉換。
var ErrInvalidStatusTransition = errors.New("user: invalid status transition")

// Status 是帳號狀態。值域取 ERD §6.1.1 草案示例（啟用／停用／
// 待刪除），尚待定稿；pending_deletion 的進入流程（冷靜期）未
// 實作，不得以 SetStatus 直接設定。
type Status string

const (
	StatusActive          Status = "active"
	StatusDisabled        Status = "disabled"
	StatusPendingDeletion Status = "pending_deletion"
)

// ValidationError 承載欄位驗證失敗清單；handler 據此寫出
// ETHER_VALIDATION_INVALID_FIELD。
type ValidationError struct {
	Fields []transport.FieldError
}

func (e *ValidationError) Error() string { return "user: validation failed" }

// FieldErrors 供 transport.WriteServiceError 寫出 details。
func (e *ValidationError) FieldErrors() []transport.FieldError { return e.Fields }

// User 是 users 的一列（ERD §6.1.1 草案欄位，扣除 password_hash）。
type User struct {
	ID              string
	KeycloakSubject string // Keycloak subject：穩定綁定識別碼，不得僅以 Email 比對。
	Email           string
	EmailVerified   bool
	PhoneNumber     string // ERD 草案欄位；本階段不寫入、不回傳。
	Status          Status
	RegisteredAt    time.Time
	LastLoginAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Settings 是 user_settings 的一列（ERD §6.1.3 草案欄位，扣除
// media_auto_download／other_settings：值域與結構待定稿，本階段
// 不暴露於 API）。
type Settings struct {
	UserID         string
	Language       string
	Theme          string
	FontSize       string
	ChatBackground string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UserStore 是 users 的儲存介面。
type UserStore interface {
	GetByID(ctx context.Context, id string) (*User, error)
	GetBySubject(ctx context.Context, subject string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, u *User) error
	Save(ctx context.Context, u *User) error
}

// SettingsStore 是 user_settings 的儲存介面。
type SettingsStore interface {
	GetByUserID(ctx context.Context, userID string) (*Settings, error)
	Create(ctx context.Context, s *Settings) error
	Save(ctx context.Context, s *Settings) error
}

// Service 是 User 領域服務。
type Service struct {
	users    UserStore
	settings SettingsStore
	now      func() time.Time
}

// NewService 建立 Service。now 為空時用 time.Now。
func NewService(users UserStore, settings SettingsStore, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{users: users, settings: settings, now: now}
}

// Provision 首次登入建置（just-in-time provisioning）：依 Keycloak
// subject 查找，不存在則建立 users 列與預設設定；已存在則更新
// Email／驗證狀態與最後登入時間。冪等，可重複呼叫。
func (s *Service) Provision(ctx context.Context, subject, email string, emailVerified bool) (*User, error) {
	if strings.TrimSpace(subject) == "" {
		return nil, errors.New("user: subject 不得為空")
	}
	existing, err := s.users.GetBySubject(ctx, subject)
	if err == nil {
		now := s.now().UTC()
		existing.Email = email
		existing.EmailVerified = emailVerified
		existing.LastLoginAt = &now
		existing.UpdatedAt = now
		if err := s.users.Save(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}
	// Email 唯一（非空時）：避免兩個 subject 佔用同一 Email。
	if email != "" {
		if _, err := s.users.GetByEmail(ctx, email); err == nil {
			return nil, &ValidationError{Fields: []transport.FieldError{{
				Field:   "email",
				Issue:   "DUPLICATE",
				Message: "Email 已被其他帳號使用。",
			}}}
		} else if !errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
	}
	now := s.now().UTC()
	u := &User{
		ID:              ids.New(),
		KeycloakSubject: subject,
		Email:           email,
		EmailVerified:   emailVerified,
		Status:          StatusActive,
		RegisteredAt:    now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	u.LastLoginAt = &now
	if u.ID == "" {
		return nil, errors.New("user: 識別碼產生失敗")
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	// 預設設定：跟隨 Phase 02 Web 預設（系統主題、繁體中文）。
	// 此為 M1 實作預設，非規格定稿值。
	def := &Settings{
		UserID:    u.ID,
		Language:  "zh-Hant",
		Theme:     "system",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.settings.Create(ctx, def); err != nil {
		return nil, err
	}
	return u, nil
}

// GetByID 依 ID 取使用者。
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	return s.users.GetByID(ctx, id)
}

// SetStatus 切換帳號狀態：僅允許 active ↔ disabled。pending_deletion
// 須走刪除申請流程（未實作），不得直接設定。
func (s *Service) SetStatus(ctx context.Context, id string, status Status) (*User, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	switch status {
	case StatusActive, StatusDisabled:
	case StatusPendingDeletion:
		return nil, ErrInvalidStatusTransition
	default:
		return nil, ErrInvalidStatusTransition
	}
	u.Status = status
	u.UpdatedAt = s.now().UTC()
	if err := s.users.Save(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// SettingsPatch 是 PATCH 語義的設定更新：nil 表示未提供（不更動）。
type SettingsPatch struct {
	Language       *string
	Theme          *string
	FontSize       *string
	ChatBackground *string
}

// ValidateSettingsPatch 驗證設定值。theme 值域取 Phase 02 Web 實作
// （system／light／dark），待 ERD 定稿覆核。
func ValidateSettingsPatch(p SettingsPatch) []transport.FieldError {
	var fields []transport.FieldError
	if p.Language != nil {
		v := *p.Language
		if len(v) < 2 || len(v) > 16 || !isLangTag(v) {
			fields = append(fields, transport.FieldError{Field: "language", Issue: "INVALID_FORMAT", Message: "語言標籤格式不正確。"})
		}
	}
	if p.Theme != nil {
		switch *p.Theme {
		case "system", "light", "dark":
		default:
			fields = append(fields, transport.FieldError{Field: "theme", Issue: "INVALID_VALUE", Message: "主題須為 system、light 或 dark。"})
		}
	}
	if p.FontSize != nil && len(*p.FontSize) > 32 {
		fields = append(fields, transport.FieldError{Field: "font_size", Issue: "TOO_LONG", Message: "字體大小設定過長。"})
	}
	if p.ChatBackground != nil && len(*p.ChatBackground) > 256 {
		fields = append(fields, transport.FieldError{Field: "chat_background", Issue: "TOO_LONG", Message: "聊天背景設定過長。"})
	}
	return fields
}

func isLangTag(s string) bool {
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

// UpdateSettings 套用設定 PATCH（僅更新提供的欄位）。
func (s *Service) UpdateSettings(ctx context.Context, userID string, p SettingsPatch) (*Settings, error) {
	if fields := ValidateSettingsPatch(p); len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	cur, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p.Language != nil {
		cur.Language = *p.Language
	}
	if p.Theme != nil {
		cur.Theme = *p.Theme
	}
	if p.FontSize != nil {
		cur.FontSize = *p.FontSize
	}
	if p.ChatBackground != nil {
		cur.ChatBackground = *p.ChatBackground
	}
	cur.UpdatedAt = s.now().UTC()
	if err := s.settings.Save(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

// GetSettings 取使用者設定。
func (s *Service) GetSettings(ctx context.Context, userID string) (*Settings, error) {
	return s.settings.GetByUserID(ctx, userID)
}

// Exists 供 Profile／Device 模組確認使用者存在（依賴 DAG 介面，
// 不暴露完整 UserStore）。
func (s *Service) Exists(ctx context.Context, userID string) (bool, error) {
	_, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, ErrUserNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// MemoryUserStore 是 UserStore 的記憶體實作（開發與測試用）。
type MemoryUserStore struct {
	mu      sync.RWMutex
	byID    map[string]*User
	bySubj  map[string]*User
	byEmail map[string]*User
}

// NewMemoryUserStore 建立空的記憶體使用者儲存。
func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		byID:    map[string]*User{},
		bySubj:  map[string]*User{},
		byEmail: map[string]*User{},
	}
}

func cloneUser(u *User) *User {
	if u == nil {
		return nil
	}
	cp := *u
	if u.LastLoginAt != nil {
		t := *u.LastLoginAt
		cp.LastLoginAt = &t
	}
	return &cp
}

func (m *MemoryUserStore) GetByID(_ context.Context, id string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.byID[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return cloneUser(u), nil
}

func (m *MemoryUserStore) GetBySubject(_ context.Context, subject string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.bySubj[subject]
	if !ok {
		return nil, ErrUserNotFound
	}
	return cloneUser(u), nil
}

func (m *MemoryUserStore) GetByEmail(_ context.Context, email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.byEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return cloneUser(u), nil
}

func (m *MemoryUserStore) Create(_ context.Context, u *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byID[u.ID]; exists {
		return errors.New("user: id 已存在")
	}
	if _, exists := m.bySubj[u.KeycloakSubject]; exists {
		return errors.New("user: subject 已存在")
	}
	if u.Email != "" {
		if _, exists := m.byEmail[u.Email]; exists {
			return errors.New("user: email 已存在")
		}
	}
	m.byID[u.ID] = cloneUser(u)
	m.bySubj[u.KeycloakSubject] = cloneUser(u)
	if u.Email != "" {
		m.byEmail[u.Email] = cloneUser(u)
	}
	return nil
}

func (m *MemoryUserStore) Save(_ context.Context, u *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.byID[u.ID]
	if !ok {
		return ErrUserNotFound
	}
	if old.Email != "" && old.Email != u.Email {
		delete(m.byEmail, old.Email)
	}
	if u.Email != "" {
		if other, exists := m.byEmail[u.Email]; exists && other.ID != u.ID {
			return errors.New("user: email 已存在")
		}
		m.byEmail[u.Email] = cloneUser(u)
	}
	m.byID[u.ID] = cloneUser(u)
	m.bySubj[u.KeycloakSubject] = cloneUser(u)
	return nil
}

// MemorySettingsStore 是 SettingsStore 的記憶體實作。
type MemorySettingsStore struct {
	mu   sync.RWMutex
	rows map[string]*Settings
}

// NewMemorySettingsStore 建立空的記憶體設定儲存。
func NewMemorySettingsStore() *MemorySettingsStore {
	return &MemorySettingsStore{rows: map[string]*Settings{}}
}

func (m *MemorySettingsStore) GetByUserID(_ context.Context, userID string) (*Settings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.rows[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *s
	return &cp, nil
}

func (m *MemorySettingsStore) Create(_ context.Context, s *Settings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.rows[s.UserID]; exists {
		return errors.New("user: settings 已存在")
	}
	cp := *s
	m.rows[s.UserID] = &cp
	return nil
}

func (m *MemorySettingsStore) Save(_ context.Context, s *Settings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rows[s.UserID]; !ok {
		return ErrUserNotFound
	}
	cp := *s
	m.rows[s.UserID] = &cp
	return nil
}
