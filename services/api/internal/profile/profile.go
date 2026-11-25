// Package profile 是 Profile（公開檔案）模組的 Phase 03 實作。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：公開檔案（顯示名稱、公開 ID、
// 頭像參照、簡介、狀態訊息等）。擁有資料表：user_profiles。
// 不得存放電話、Email 等私密聯絡資料；不得替 Friend 判斷關係；
// 公開檔案回應不得洩漏私密欄位。
package profile

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"ether.local/api/internal/transport"
)

// ErrProfileNotFound 表示查無公開檔案。
var ErrProfileNotFound = errors.New("profile: not found")

// ValidationError 承載欄位驗證失敗清單。
type ValidationError struct {
	Fields []transport.FieldError
}

func (e *ValidationError) Error() string { return "profile: validation failed" }

// FieldErrors 供 transport.WriteServiceError 寫出 details。
func (e *ValidationError) FieldErrors() []transport.FieldError { return e.Fields }

// UserResolver 是 Profile 對 User 模組的最小依賴（依賴 DAG：
// Profile --> User）：僅確認使用者存在，不讀取使用者私密欄位。
type UserResolver interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// Profile 是 user_profiles 的一列（ERD §6.1.2 草案欄位）。
// avatar_media_id 的參照方式待媒體模組定稿，本階段以字串參照
// （媒體識別碼）承載，不得假裝已接通媒體儲存。
// birthday／birthday_visibility／profile_visibility／qr_code_token
// 的值域與流程待定稿，本階段不暴露於 API。
type Profile struct {
	UserID        string
	DisplayName   string
	PublicID      string // 公開識別碼；生成方式待定稿，本階段僅接受使用者自定。
	AvatarMediaID string
	Bio           string
	StatusMessage string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Store 是 user_profiles 的儲存介面。
type Store interface {
	GetByUserID(ctx context.Context, userID string) (*Profile, error)
	GetByPublicID(ctx context.Context, publicID string) (*Profile, error)
	Create(ctx context.Context, p *Profile) error
	Save(ctx context.Context, p *Profile) error
}

// Service 是 Profile 領域服務。
type Service struct {
	store Store
	users UserResolver
	now   func() time.Time
}

// NewService 建立 Service。now 為空時用 time.Now。
func NewService(store Store, users UserResolver, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, users: users, now: now}
}

// GetByUserID 唯讀取用者檔案（供 Friend 模組豐富化清單用，
// 依賴 DAG：Friend --> Profile）。檔案不存在時回傳
// ErrProfileNotFound，不得建立檔案（與 GetOrCreate 不同）。
func (s *Service) GetByUserID(ctx context.Context, userID string) (*Profile, error) {
	return s.store.GetByUserID(ctx, userID)
}

// GetByPublicID 依公開識別碼唯讀取用者檔案（供 Friend 搜尋用）。
// 空值或查無時回傳 ErrProfileNotFound。
func (s *Service) GetByPublicID(ctx context.Context, publicID string) (*Profile, error) {
	return s.store.GetByPublicID(ctx, publicID)
}

// GetOrCreate 取用者檔案；不存在時以預設顯示名稱建立
// （顯示名稱取 Email 前綴或 subject 縮寫，不得為空）。
// 使用者不存在時回傳 ErrProfileNotFound，不得為幽靈使用者建檔。
func (s *Service) GetOrCreate(ctx context.Context, userID, defaultName string) (*Profile, error) {
	if p, err := s.store.GetByUserID(ctx, userID); err == nil {
		return p, nil
	} else if !errors.Is(err, ErrProfileNotFound) {
		return nil, err
	}
	ok, err := s.users.Exists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrProfileNotFound
	}
	name := strings.TrimSpace(defaultName)
	if name == "" {
		name = "ETHER 使用者"
	}
	now := s.now().UTC()
	p := &Profile{
		UserID:      userID,
		DisplayName: name,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Patch 是 PATCH 語義的檔案更新：nil 表示未提供（不更動）；
// 空字串指標表示清空該欄位（bio／status_message／avatar 適用；
// display_name 不接受空值）。
type Patch struct {
	DisplayName   *string
	PublicID      *string
	AvatarMediaID *string
	Bio           *string
	StatusMessage *string
}

var publicIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{2,31}$`)

// ValidatePatch 驗證檔案更新值。長度上限為 M1 實作預設
// （display_name 1–64、bio ≤500、status_message ≤140），
// 待端點 Schema 定稿覆核，不得以本值冒充規格定稿。
func ValidatePatch(p Patch) []transport.FieldError {
	var fields []transport.FieldError
	if p.DisplayName != nil {
		v := strings.TrimSpace(*p.DisplayName)
		if v == "" {
			fields = append(fields, transport.FieldError{Field: "display_name", Issue: "REQUIRED", Message: "顯示名稱不得為空。"})
		} else if len([]rune(v)) > 64 {
			fields = append(fields, transport.FieldError{Field: "display_name", Issue: "TOO_LONG", Message: "顯示名稱過長。"})
		}
	}
	if p.PublicID != nil && *p.PublicID != "" && !publicIDRe.MatchString(*p.PublicID) {
		fields = append(fields, transport.FieldError{Field: "public_id", Issue: "INVALID_FORMAT", Message: "公開識別碼須為 3–32 字元的小寫英數字、底線、點或連字號。"})
	}
	if p.AvatarMediaID != nil && len(*p.AvatarMediaID) > 128 {
		fields = append(fields, transport.FieldError{Field: "avatar_media_id", Issue: "TOO_LONG", Message: "頭像參照過長。"})
	}
	if p.Bio != nil && len([]rune(*p.Bio)) > 500 {
		fields = append(fields, transport.FieldError{Field: "bio", Issue: "TOO_LONG", Message: "簡介過長。"})
	}
	if p.StatusMessage != nil && len([]rune(*p.StatusMessage)) > 140 {
		fields = append(fields, transport.FieldError{Field: "status_message", Issue: "TOO_LONG", Message: "狀態訊息過長。"})
	}
	return fields
}

// Update 套用檔案 PATCH（僅更新提供的欄位）。public_id 變更時
// 檢查唯一性（非空時）。
func (s *Service) Update(ctx context.Context, userID string, p Patch) (*Profile, error) {
	if fields := ValidatePatch(p); len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	cur, err := s.store.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p.PublicID != nil && *p.PublicID != "" && *p.PublicID != cur.PublicID {
		if _, err := s.store.GetByPublicID(ctx, *p.PublicID); err == nil {
			return nil, &ValidationError{Fields: []transport.FieldError{{
				Field:   "public_id",
				Issue:   "DUPLICATE",
				Message: "公開識別碼已被使用。",
			}}}
		} else if !errors.Is(err, ErrProfileNotFound) {
			return nil, err
		}
	}
	if p.DisplayName != nil {
		cur.DisplayName = strings.TrimSpace(*p.DisplayName)
	}
	if p.PublicID != nil {
		cur.PublicID = *p.PublicID
	}
	if p.AvatarMediaID != nil {
		cur.AvatarMediaID = *p.AvatarMediaID
	}
	if p.Bio != nil {
		cur.Bio = *p.Bio
	}
	if p.StatusMessage != nil {
		cur.StatusMessage = *p.StatusMessage
	}
	cur.UpdatedAt = s.now().UTC()
	if err := s.store.Save(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

// MemoryStore 是 Store 的記憶體實作（開發與測試用）。
type MemoryStore struct {
	mu       sync.RWMutex
	byUser   map[string]*Profile
	byPublic map[string]*Profile
}

// NewMemoryStore 建立空的記憶體檔案儲存。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byUser: map[string]*Profile{}, byPublic: map[string]*Profile{}}
}

func (m *MemoryStore) GetByUserID(_ context.Context, userID string) (*Profile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.byUser[userID]
	if !ok {
		return nil, ErrProfileNotFound
	}
	cp := *p
	return &cp, nil
}

// GetByPublicID 依公開識別碼查找；空值永遠查無。
func (m *MemoryStore) GetByPublicID(_ context.Context, publicID string) (*Profile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if publicID == "" {
		return nil, ErrProfileNotFound
	}
	p, ok := m.byPublic[publicID]
	if !ok {
		return nil, ErrProfileNotFound
	}
	cp := *p
	return &cp, nil
}

func (m *MemoryStore) Create(_ context.Context, p *Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byUser[p.UserID]; exists {
		return errors.New("profile: 已存在")
	}
	if p.PublicID != "" {
		if _, exists := m.byPublic[p.PublicID]; exists {
			return errors.New("profile: public_id 已存在")
		}
	}
	cp := *p
	m.byUser[p.UserID] = &cp
	if p.PublicID != "" {
		m.byPublic[p.PublicID] = &cp
	}
	return nil
}

func (m *MemoryStore) Save(_ context.Context, p *Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.byUser[p.UserID]
	if !ok {
		return ErrProfileNotFound
	}
	if old.PublicID != "" && old.PublicID != p.PublicID {
		delete(m.byPublic, old.PublicID)
	}
	if p.PublicID != "" {
		if other, exists := m.byPublic[p.PublicID]; exists && other.UserID != p.UserID {
			return errors.New("profile: public_id 已存在")
		}
	}
	cp := *p
	m.byUser[p.UserID] = &cp
	if p.PublicID != "" {
		m.byPublic[p.PublicID] = &cp
	}
	return nil
}
