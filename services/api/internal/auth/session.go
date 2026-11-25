// Package auth 的 Session 領域：ETHER 端的 user_sessions 映象
// （IDENTITY_AND_PERMISSIONS.md §4–§5）。
//
// 規則：
//   - Session 記錄只存 Token 雜湊（SHA-256 hex），絕不存 Token 明文。
//   - Refresh Token 每次使用後輪換；舊雜湊即失效；重放已輪換的
//     Refresh Token 視為盜用，須撤銷該使用者全部 Session
//     （Session 家族撤銷，IDENTITY §4）。
//   - 撤銷為狀態標記（is_revoked／revoked_at），到期或撤銷的紀錄
//     不得立即物理刪除（稽核需要，ERD §6.1.5）。
//   - Token 本身由 Keycloak 簽發；本領域只維護 ETHER 端的撤銷
//     映象，不得以本紀錄代替 Keycloak 的簽名驗證。
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"sync"
	"time"

	"ether.local/api/internal/ids"
)

// ErrSessionNotFound 表示查無 Session。呼叫方對外一律回 404
// ETHER_COMMON_NOT_FOUND，不得以此區分「不存在」與「無權存取」。
var ErrSessionNotFound = errors.New("auth: session not found")

// ErrSessionInvalid 表示 Session 已撤銷、已過期或 Refresh 雜湊不符。
var ErrSessionInvalid = errors.New("auth: session invalid")

// ErrRefreshTokenReplayed 表示重放已輪換的 Refresh Token（盜用訊號）。
// 呼叫方收到此錯誤時，該使用者的 Session 家族已被撤銷，須導向
// 重新登入並發出安全通知（通知機制待 Phase 05 通知模組）。
var ErrRefreshTokenReplayed = errors.New("auth: refresh token replayed")

var sha256HexRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// HashToken 以 SHA-256 雜湊 Token，僅存雜湊值。呼叫方不得將
// 明文 Token 傳入儲存層。
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Session 是 user_sessions 的一列（ERD §6.1.5 草案欄位）。
type Session struct {
	ID                string
	UserID            string
	DeviceID          string // 可空（無裝置關聯時為空字串）。
	KeycloakSessionID string // Keycloak sid；無此聲明時為空字串。
	RefreshTokenHash  string // 僅雜湊，不得存明文。
	ExpiresAt         time.Time
	LastActiveAt      time.Time
	RevokedAt         *time.Time
	RevokedReason     string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Revoked 報告 Session 是否已被撤銷。
func (s *Session) Revoked() bool { return s.RevokedAt != nil }

// SessionStore 是 Session 的儲存介面。IsHashConsumed／
// MarkHashConsumed 記錄已輪換的 Refresh 雜湊（重放偵測）：
// PostgreSQL 實作存於 session_refresh_hashes 表（000004），重啟
// 後仍有效；記憶體實作僅行程內有效，僅供開發與測試。
type SessionStore interface {
	Create(ctx context.Context, s *Session) error
	Get(ctx context.Context, id string) (*Session, error)
	GetByKeycloakSessionID(ctx context.Context, sid string) (*Session, error)
	ListByUser(ctx context.Context, userID string) ([]*Session, error)
	Save(ctx context.Context, s *Session) error
	IsHashConsumed(ctx context.Context, sessionID, hash string) (bool, error)
	MarkHashConsumed(ctx context.Context, sessionID, hash string) (bool, error)
}

// SessionService 是 Session 領域服務：建立、輪換、撤銷。
type SessionService struct {
	store SessionStore
	now   func() time.Time
}

// NewSessionService 建立 SessionService。now 為空時用 time.Now。
func NewSessionService(store SessionStore, now func() time.Time) *SessionService {
	if now == nil {
		now = time.Now
	}
	return &SessionService{store: store, now: now}
}

// Create 建立 Session。refreshTokenHash 須為 64 字元 hex（HashToken
// 的輸出），否則拒絕，避免明文 Token 被誤存。
func (s *SessionService) Create(ctx context.Context, userID, deviceID, keycloakSessionID, refreshTokenHash string, expiresAt time.Time) (*Session, error) {
	if userID == "" || !sha256HexRe.MatchString(refreshTokenHash) {
		return nil, ErrSessionInvalid
	}
	now := s.now().UTC()
	sess := &Session{
		ID:                ids.New(),
		UserID:            userID,
		DeviceID:          deviceID,
		KeycloakSessionID: keycloakSessionID,
		RefreshTokenHash:  refreshTokenHash,
		ExpiresAt:         expiresAt.UTC(),
		LastActiveAt:      now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if sess.ID == "" {
		return nil, errors.New("auth: 識別碼產生失敗")
	}
	if err := s.store.Create(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

// Rotate 執行 Refresh Token 輪換：舊雜湊須與現行雜湊一致，否則：
//   - 舊雜湊曾被輪換過 → 盜用重放：撤銷該使用者全部 Session，
//     回傳 ErrRefreshTokenReplayed。
//   - 其他不符 → 回傳 ErrSessionInvalid，不更動狀態。
func (s *SessionService) Rotate(ctx context.Context, sessionID, oldHash, newHash string, newExpiry time.Time) (*Session, error) {
	if !sha256HexRe.MatchString(oldHash) || !sha256HexRe.MatchString(newHash) {
		return nil, ErrSessionInvalid
	}
	sess, err := s.store.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Revoked() || s.now().After(sess.ExpiresAt) {
		return nil, ErrSessionInvalid
	}
	if sess.RefreshTokenHash != oldHash {
		consumed, err := s.store.IsHashConsumed(ctx, sessionID, oldHash)
		if err != nil {
			return nil, err
		}
		if consumed {
			// 盜用：撤銷家族，不得只撤銷單一 Session。
			_ = s.RevokeAllForUser(ctx, sess.UserID, "refresh_token_replayed")
			return nil, ErrRefreshTokenReplayed
		}
		return nil, ErrSessionInvalid
	}

	// 先標記舊雜湊已消耗、再寫入新雜湊：標記優先於寫入，確保
	// Save 失敗時舊雜湊亦不可再用（fail-closed）；代價是 Save
	// 因暫時性錯誤失敗後，以舊雜湊重試會觸發家族撤銷——此為
	// 安全優先的刻意取捨，呼叫方不得以舊雜湊重建 Session。
	if _, err := s.store.MarkHashConsumed(ctx, sessionID, oldHash); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	sess.RefreshTokenHash = newHash
	sess.ExpiresAt = newExpiry.UTC()
	sess.LastActiveAt = now
	sess.UpdatedAt = now
	if err := s.store.Save(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

// Revoke 撤銷單一 Session（冪等：已撤銷者直接回傳現狀）。
func (s *SessionService) Revoke(ctx context.Context, sessionID, reason string) (*Session, error) {
	sess, err := s.store.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Revoked() {
		return sess, nil
	}
	now := s.now().UTC()
	sess.RevokedAt = &now
	sess.RevokedReason = reason
	sess.UpdatedAt = now
	if err := s.store.Save(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

// RevokeAllForUser 撤銷使用者的全部 Session（遠端登出全部裝置、
// 盜用家族撤銷）。已撤銷者保持原狀。
func (s *SessionService) RevokeAllForUser(ctx context.Context, userID, reason string) error {
	sessions, err := s.store.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	for _, sess := range sessions {
		if sess.Revoked() {
			continue
		}
		revokedAt := now
		sess.RevokedAt = &revokedAt
		sess.RevokedReason = reason
		sess.UpdatedAt = now
		if err := s.store.Save(ctx, sess); err != nil {
			return err
		}
	}
	return nil
}

// Touch 更新 Session 的最後活動時間（已撤銷／已過期者不更新）。
func (s *SessionService) Touch(ctx context.Context, sessionID string) error {
	sess, err := s.store.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if sess.Revoked() || s.now().After(sess.ExpiresAt) {
		return ErrSessionInvalid
	}
	sess.LastActiveAt = s.now().UTC()
	sess.UpdatedAt = sess.LastActiveAt
	return s.store.Save(ctx, sess)
}

// MemorySessionStore 是 SessionStore 的記憶體實作：僅供開發與測試，
// 資料不持久、單一行程，不得用於正式環境。已消耗雜湊同樣只存
// 行程內，重啟後重放偵測失效（與 PostgreSQL 實作的語義差異，
// 僅測試與開發可接受）。
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	consumed map[string]map[string]bool // sessionID -> 已輪換的舊雜湊。
}

// NewMemorySessionStore 建立空的記憶體 Session 儲存。
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: map[string]*Session{},
		consumed: map[string]map[string]bool{},
	}
}

func cloneSession(s *Session) *Session {
	if s == nil {
		return nil
	}
	cp := *s
	if s.RevokedAt != nil {
		t := *s.RevokedAt
		cp.RevokedAt = &t
	}
	return &cp
}

func (m *MemorySessionStore) Create(_ context.Context, s *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.sessions[s.ID]; exists {
		return errors.New("auth: session id 已存在")
	}
	m.sessions[s.ID] = cloneSession(s)
	return nil
}

func (m *MemorySessionStore) Get(_ context.Context, id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return cloneSession(s), nil
}

func (m *MemorySessionStore) GetByKeycloakSessionID(_ context.Context, sid string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.sessions {
		if s.KeycloakSessionID != "" && s.KeycloakSessionID == sid {
			return cloneSession(s), nil
		}
	}
	return nil, ErrSessionNotFound
}

func (m *MemorySessionStore) ListByUser(_ context.Context, userID string) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Session
	for _, s := range m.sessions {
		if s.UserID == userID {
			out = append(out, cloneSession(s))
		}
	}
	return out, nil
}

func (m *MemorySessionStore) Save(_ context.Context, s *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[s.ID]; !ok {
		return ErrSessionNotFound
	}
	m.sessions[s.ID] = cloneSession(s)
	return nil
}

// IsHashConsumed 報告該雜湊是否曾被輪換消耗。
func (m *MemorySessionStore) IsHashConsumed(_ context.Context, sessionID, hash string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.consumed[sessionID][hash], nil
}

// MarkHashConsumed 記錄雜湊已被輪換消耗；已記錄時回傳 false。
func (m *MemorySessionStore) MarkHashConsumed(_ context.Context, sessionID, hash string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.consumed[sessionID] == nil {
		m.consumed[sessionID] = map[string]bool{}
	}
	if m.consumed[sessionID][hash] {
		return false, nil
	}
	m.consumed[sessionID][hash] = true
	return true, nil
}
