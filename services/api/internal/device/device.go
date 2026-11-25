// Package device 是 Device Management（裝置管理）模組的 Phase 03 實作。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：登入裝置清單與裝置信任管理。
// 擁有資料表：user_devices。不得管理 Session 本身（屬 Authentication）；
// 不得替 Notification 管理推播 Token；不得在裝置紀錄中存放 Token 明文；
// 不得存放可直接識別裝置的原始識別碼明文，僅存雜湊。
package device

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"ether.local/api/internal/ids"
	"ether.local/api/internal/transport"
)

// ErrDeviceNotFound 表示查無裝置。
var ErrDeviceNotFound = errors.New("device: not found")

// ValidationError 承載欄位驗證失敗清單。
type ValidationError struct {
	Fields []transport.FieldError
}

func (e *ValidationError) Error() string { return "device: validation failed" }

// FieldErrors 供 transport.WriteServiceError 寫出 details。
func (e *ValidationError) FieldErrors() []transport.FieldError { return e.Fields }

// UserResolver 是 Device 對 User 模組的最小依賴（依賴 DAG：
// DeviceMgmt --> User）：僅確認使用者存在。
type UserResolver interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// Device 是 user_devices 的一列（ERD §6.1.4 草案欄位）。
// DeviceIdentifierHash 僅存 SHA-256 hex 雜湊（客戶端雜湊後上傳），
// 伺服器不得接受、不得儲存原始裝置識別碼。
type Device struct {
	ID                   string
	UserID               string
	DeviceName           string
	Platform             string // 值域待定稿（例如 android／ios／web／windows／macos）。
	DeviceIdentifierHash string
	LastActiveAt         *time.Time
	Trusted              bool
	Revoked              bool
	RevokedAt            *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// Store 是 user_devices 的儲存介面。
type Store interface {
	Get(ctx context.Context, id string) (*Device, error)
	ListByUser(ctx context.Context, userID string) ([]*Device, error)
	Create(ctx context.Context, d *Device) error
	Save(ctx context.Context, d *Device) error
}

// Service 是 Device 領域服務。
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

var sha256HexRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Register 登記裝置：裝置識別碼須已由客戶端雜湊為 64 字元 hex；
// 伺服器拒收任何非雜湊形態的識別碼，避免原始識別碼流入資料庫。
func (s *Service) Register(ctx context.Context, userID, deviceName, platform, identifierHash string) (*Device, error) {
	var fields []transport.FieldError
	if strings.TrimSpace(deviceName) == "" || len([]rune(deviceName)) > 128 {
		fields = append(fields, transport.FieldError{Field: "device_name", Issue: "INVALID_VALUE", Message: "裝置名稱須為 1–128 字元。"})
	}
	if strings.TrimSpace(platform) == "" || len(platform) > 64 {
		fields = append(fields, transport.FieldError{Field: "platform", Issue: "INVALID_VALUE", Message: "平台不得為空且不得超過 64 字元。"})
	}
	if !sha256HexRe.MatchString(identifierHash) {
		fields = append(fields, transport.FieldError{Field: "device_identifier_hash", Issue: "INVALID_FORMAT", Message: "裝置識別碼須為 SHA-256 雜湊（64 字元 hex）。"})
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	ok, err := s.users.Exists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrDeviceNotFound
	}
	now := s.now().UTC()
	d := &Device{
		ID:                   ids.New(),
		UserID:               userID,
		DeviceName:           strings.TrimSpace(deviceName),
		Platform:             platform,
		DeviceIdentifierHash: identifierHash,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if d.ID == "" {
		return nil, errors.New("device: 識別碼產生失敗")
	}
	if err := s.store.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// Revoke 撤銷裝置（狀態標記，不得物理刪除撤銷紀錄，ERD §6.1.4）。
// 撤銷後該裝置的 Session 在 Auth 側以裝置狀態判定為失效（見
// auth handler 的裝置狀態檢查）；本方法不直接操作 Session 表
// （依賴 DAG 禁止反向呼叫）。
func (s *Service) Revoke(ctx context.Context, userID, deviceID string) (*Device, error) {
	d, err := s.store.Get(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if d.UserID != userID {
		// 非擁有者：以「不存在」回應，不得洩漏他人裝置存在性。
		return nil, ErrDeviceNotFound
	}
	if d.Revoked {
		return d, nil
	}
	now := s.now().UTC()
	d.Revoked = true
	d.RevokedAt = &now
	d.UpdatedAt = now
	if err := s.store.Save(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// SetTrusted 設定裝置信任狀態（已撤銷裝置不得設為信任）。
func (s *Service) SetTrusted(ctx context.Context, userID, deviceID string, trusted bool) (*Device, error) {
	d, err := s.store.Get(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if d.UserID != userID {
		return nil, ErrDeviceNotFound
	}
	if d.Revoked && trusted {
		return nil, errors.New("device: 已撤銷裝置不得設為信任")
	}
	d.Trusted = trusted
	d.UpdatedAt = s.now().UTC()
	if err := s.store.Save(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// List 列出使用者的裝置。
func (s *Service) List(ctx context.Context, userID string) ([]*Device, error) {
	return s.store.ListByUser(ctx, userID)
}

// MemoryStore 是 Store 的記憶體實作（開發與測試用）。
type MemoryStore struct {
	mu      sync.RWMutex
	devices map[string]*Device
}

// NewMemoryStore 建立空的記憶體裝置儲存。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{devices: map[string]*Device{}}
}

func cloneDevice(d *Device) *Device {
	if d == nil {
		return nil
	}
	cp := *d
	if d.LastActiveAt != nil {
		t := *d.LastActiveAt
		cp.LastActiveAt = &t
	}
	if d.RevokedAt != nil {
		t := *d.RevokedAt
		cp.RevokedAt = &t
	}
	return &cp
}

func (m *MemoryStore) Get(_ context.Context, id string) (*Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.devices[id]
	if !ok {
		return nil, ErrDeviceNotFound
	}
	return cloneDevice(d), nil
}

func (m *MemoryStore) ListByUser(_ context.Context, userID string) ([]*Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Device
	for _, d := range m.devices {
		if d.UserID == userID {
			out = append(out, cloneDevice(d))
		}
	}
	return out, nil
}

func (m *MemoryStore) Create(_ context.Context, d *Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.devices[d.ID]; exists {
		return errors.New("device: id 已存在")
	}
	m.devices[d.ID] = cloneDevice(d)
	return nil
}

func (m *MemoryStore) Save(_ context.Context, d *Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.devices[d.ID]; !ok {
		return ErrDeviceNotFound
	}
	m.devices[d.ID] = cloneDevice(d)
	return nil
}
