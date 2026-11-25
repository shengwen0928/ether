package chat

import (
	"context"
	"errors"
	"sync"
	"time"
	"unicode/utf8"

	"ether.local/api/internal/ids"
	"ether.local/api/internal/transport"
)

// ErrConversationNotFound 表示查無對話，或呼叫者不是生效成員。
// 兩種情形對外一律 404，不得區分「不存在」與「無權存取」。
var ErrConversationNotFound = errors.New("chat: conversation not found")

// ConversationType 是對話類型。值域為 Phase 05 實作詞彙
// direct／group（ERD §6.3.1 稱值域待定稿），待端點 Schema 定稿覆核。
type ConversationType string

const (
	// TypeDirect 是一對一對話：恰好兩名成員，建立須以好友關係為前提。
	TypeDirect ConversationType = "direct"
	// TypeGroup 是群組對話：三名以上成員皆可，由建立者指定初始成員。
	TypeGroup ConversationType = "group"
)

// MemberRole 是成員角色。值域為 Phase 05 實作詞彙 member／admin
// （ERD §6.3.2 稱值域待定稿）。群組建立者為 admin，其餘為 member；
// 一對一對話雙方皆為 member。
type MemberRole string

const (
	RoleMember MemberRole = "member"
	RoleAdmin  MemberRole = "admin"
)

// Conversation 是 conversations 的一列。Title 為空表示未命名
// （一對一對話預設無標題）。LastMessageID／LastMessageAt 為快取
// 欄位，由 Message 寫入路徑經 NoteMessage 更新。
type Conversation struct {
	ID            string
	Type          ConversationType
	Title         string
	CreatedBy     string
	LastMessageID string
	LastMessageAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Member 是 conversation_members 的一列。LeftAt 為空表示生效成員。
type Member struct {
	ConversationID    string
	UserID            string
	Role              MemberRole
	JoinedAt          time.Time
	LeftAt            *time.Time
	LastReadMessageID string
	LastReadSequence  int64
	LastReadAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Active 回傳該成員是否為生效成員（未離開）。
func (m *Member) Active() bool { return m.LeftAt == nil }

// ValidationError 承載欄位驗證失敗清單。
type ValidationError struct {
	Fields []transport.FieldError
}

func (e *ValidationError) Error() string { return "chat: validation failed" }

// FieldErrors 供 transport.WriteServiceError 寫出 details。
func (e *ValidationError) FieldErrors() []transport.FieldError { return e.Fields }

func validationError(field, issue, message string) *ValidationError {
	return &ValidationError{Fields: []transport.FieldError{{Field: field, Issue: issue, Message: message}}}
}

// UserResolver 是 Chat 對 User 模組的最小依賴（依賴 DAG：
// Chat --> User）：僅確認使用者存在。
type UserResolver interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// FriendChecker 是 Chat 對 Friend 模組的最小依賴（依賴 DAG：
// Chat --> Friend）：僅讀取好友關係與封鎖狀態，不得寫入好友表。
type FriendChecker interface {
	AreFriends(ctx context.Context, a, b string) (bool, error)
	BlockedBetween(ctx context.Context, a, b string) (bool, error)
}

// Store 是 Chat 兩張表的儲存介面。
type Store interface {
	GetConversation(ctx context.Context, id string) (*Conversation, error)
	GetDirectBetween(ctx context.Context, a, b string) (*Conversation, error)
	ListConversationsByUser(ctx context.Context, userID string) ([]*Conversation, error)
	CreateConversation(ctx context.Context, c *Conversation) error
	SaveConversation(ctx context.Context, c *Conversation) error

	GetMember(ctx context.Context, conversationID, userID string) (*Member, error)
	ListMembers(ctx context.Context, conversationID string) ([]*Member, error)
	CreateMember(ctx context.Context, m *Member) error
	SaveMember(ctx context.Context, m *Member) error
}

// Service 是 Chat 領域服務。
type Service struct {
	store   Store
	users   UserResolver
	friends FriendChecker
	now     func() time.Time
}

// NewService 建立 Service。now 為空時用 time.Now。
func NewService(store Store, users UserResolver, friends FriendChecker, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, users: users, friends: friends, now: now}
}

// CreateDirect 建立（或取回）與 otherID 的一對一對話。建立前提：
// 對方存在、任一方向無生效封鎖（封鎖時回 404，不得洩漏）、雙方為
// 好友（非好友為欄位錯誤）。同一對使用者僅一個一對一對話：已存在
// 時回傳既有對話（isNew=false）；呼叫者曾離開時重新加入。
func (s *Service) CreateDirect(ctx context.Context, creatorID, otherID string) (*Conversation, bool, error) {
	if otherID == "" {
		return nil, false, validationError("other_user_id", "REQUIRED", "請提供對方使用者識別碼。")
	}
	if otherID == creatorID {
		return nil, false, validationError("other_user_id", "INVALID_VALUE", "不能與自己建立一對一對話。")
	}
	exists, err := s.users.Exists(ctx, otherID)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, ErrConversationNotFound
	}
	blocked, err := s.friends.BlockedBetween(ctx, creatorID, otherID)
	if err != nil {
		return nil, false, err
	}
	if blocked {
		return nil, false, ErrConversationNotFound
	}
	areFriends, err := s.friends.AreFriends(ctx, creatorID, otherID)
	if err != nil {
		return nil, false, err
	}
	if !areFriends {
		return nil, false, validationError("other_user_id", "INVALID_VALUE", "一對一對話須先建立好友關係。")
	}
	existing, err := s.store.GetDirectBetween(ctx, creatorID, otherID)
	if err != nil {
		return nil, false, err
	}
	now := s.now().UTC()
	if existing != nil {
		if err := s.ensureMember(ctx, existing.ID, creatorID, RoleMember, now); err != nil {
			return nil, false, err
		}
		if err := s.ensureMember(ctx, existing.ID, otherID, RoleMember, now); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}
	conv := &Conversation{
		ID:        ids.New(),
		Type:      TypeDirect,
		CreatedBy: creatorID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if conv.ID == "" {
		return nil, false, errors.New("chat: 識別碼產生失敗")
	}
	if err := s.store.CreateConversation(ctx, conv); err != nil {
		return nil, false, err
	}
	for _, memberID := range []string{creatorID, otherID} {
		if err := s.store.CreateMember(ctx, &Member{
			ConversationID: conv.ID,
			UserID:         memberID,
			Role:           RoleMember,
			JoinedAt:       now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}); err != nil {
			return nil, false, err
		}
	}
	return conv, true, nil
}

// CreateGroup 建立群組對話。標題為必填（Phase 05 實作預設，待定稿）；
// 成員須存在、去重、不得只有建立者一人；建立者為 admin。
func (s *Service) CreateGroup(ctx context.Context, creatorID, title string, memberIDs []string) (*Conversation, error) {
	if utf8.RuneCountInString(title) == 0 || utf8.RuneCountInString(title) > 64 {
		return nil, validationError("title", "INVALID_VALUE", "群組標題須為 1 至 64 字。")
	}
	seen := map[string]bool{creatorID: true}
	members := make([]string, 0, len(memberIDs))
	for _, id := range memberIDs {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		members = append(members, id)
	}
	if len(members) == 0 {
		return nil, validationError("member_ids", "REQUIRED", "群組須至少包含一名其他成員。")
	}
	for _, id := range members {
		exists, err := s.users.Exists(ctx, id)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrConversationNotFound
		}
	}
	now := s.now().UTC()
	conv := &Conversation{
		ID:        ids.New(),
		Type:      TypeGroup,
		Title:     title,
		CreatedBy: creatorID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if conv.ID == "" {
		return nil, errors.New("chat: 識別碼產生失敗")
	}
	if err := s.store.CreateConversation(ctx, conv); err != nil {
		return nil, err
	}
	if err := s.store.CreateMember(ctx, &Member{
		ConversationID: conv.ID,
		UserID:         creatorID,
		Role:           RoleAdmin,
		JoinedAt:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		return nil, err
	}
	for _, id := range members {
		if err := s.store.CreateMember(ctx, &Member{
			ConversationID: conv.ID,
			UserID:         id,
			Role:           RoleMember,
			JoinedAt:       now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}); err != nil {
			return nil, err
		}
	}
	return conv, nil
}

// CreateGroupConversation 供 Group 模組建立群組聊天室容器
// （依賴 DAG：Group --> Chat）：CreateGroup 的薄包裝，回傳對話
// 識別碼供 groups.conversation_id 寫入。群組角色不帶入對話成員
// 權限（對話角色為本模組的 member／admin，與群組角色分離）。
func (s *Service) CreateGroupConversation(ctx context.Context, creatorID, title string, memberIDs []string) (string, error) {
	conv, err := s.CreateGroup(ctx, creatorID, title, memberIDs)
	if err != nil {
		return "", err
	}
	return conv.ID, nil
}

// AddConversationMembers 供 Group 模組同步群組成員加入對話容器：
// 權限已由 Group 服務層裁決，本方法不檢查呼叫者角色，僅驗證對話
// 存在且為群組對話；已生效者跳過，曾離開者重新加入。
func (s *Service) AddConversationMembers(ctx context.Context, conversationID string, memberIDs []string) error {
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return err
	}
	if conv == nil || conv.Type != TypeGroup {
		return ErrConversationNotFound
	}
	now := s.now().UTC()
	for _, id := range memberIDs {
		if id == "" {
			continue
		}
		if err := s.ensureMember(ctx, conversationID, id, RoleMember, now); err != nil {
			return err
		}
	}
	return nil
}

// RemoveConversationMember 供 Group 模組同步群組成員移出對話容器：
// 標記該成員離開對話，對話與訊息歷史保留（與 Leave 語義相同，
// 呼叫者身份為群組服務層，不重複檢查）。
func (s *Service) RemoveConversationMember(ctx context.Context, conversationID, userID string) error {
	return s.Leave(ctx, userID, conversationID)
}

// ensureMember 確保 userID 為對話的生效成員：已生效時不動；
// 曾離開時重新加入（left_at 清空、joined_at 更新）；無紀錄時新建。
func (s *Service) ensureMember(ctx context.Context, conversationID, userID string, role MemberRole, now time.Time) error {
	m, err := s.store.GetMember(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if m == nil {
		return s.store.CreateMember(ctx, &Member{
			ConversationID: conversationID,
			UserID:         userID,
			Role:           role,
			JoinedAt:       now,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	if m.Active() {
		return nil
	}
	m.LeftAt = nil
	m.JoinedAt = now
	m.UpdatedAt = now
	return s.store.SaveMember(ctx, m)
}

// GetConversation 讀取單一對話：非生效成員一律 404。
func (s *Service) GetConversation(ctx context.Context, userID, conversationID string) (*Conversation, error) {
	m, err := s.store.GetMember(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	if m == nil || !m.Active() {
		return nil, ErrConversationNotFound
	}
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, ErrConversationNotFound
	}
	return conv, nil
}

// ListConversations 列出使用者的生效對話，依最近活動倒序
// （updated_at 倒序，寫入路徑經 NoteMessage 更新）。
func (s *Service) ListConversations(ctx context.Context, userID string) ([]*Conversation, error) {
	return s.store.ListConversationsByUser(ctx, userID)
}

// ListMembers 列出對話成員：呼叫者須為生效成員。離開者列於
// 末尾（LeftAt 非空者），呼叫方依需求過濾。
func (s *Service) ListMembers(ctx context.Context, userID, conversationID string) ([]*Member, error) {
	if _, err := s.GetConversation(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	return s.store.ListMembers(ctx, conversationID)
}

// AddMembers 為群組對話新增成員：僅 admin 可新增；一對一對話
// 不可新增（須另建群組）。已生效者跳過；曾離開者重新加入。
func (s *Service) AddMembers(ctx context.Context, userID, conversationID string, memberIDs []string) ([]*Member, error) {
	conv, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.Type != TypeGroup {
		return nil, validationError("conversation_id", "INVALID_VALUE", "一對一對話不可新增成員。")
	}
	self, err := s.store.GetMember(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	if self == nil || !self.Active() || self.Role != RoleAdmin {
		return nil, ErrConversationNotFound
	}
	now := s.now().UTC()
	var added []*Member
	seen := map[string]bool{}
	for _, id := range memberIDs {
		if id == "" || id == userID || seen[id] {
			continue
		}
		seen[id] = true
		exists, err := s.users.Exists(ctx, id)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrConversationNotFound
		}
		if err := s.ensureMember(ctx, conversationID, id, RoleMember, now); err != nil {
			return nil, err
		}
		m, err := s.store.GetMember(ctx, conversationID, id)
		if err != nil {
			return nil, err
		}
		added = append(added, m)
	}
	return added, nil
}

// Leave 離開對話：成員列標記 left_at；對話本身保留（訊息
// append-only，不得連帶刪除）。最後一名成員離開亦同，不得
// 物理刪除對話列。
func (s *Service) Leave(ctx context.Context, userID, conversationID string) error {
	m, err := s.store.GetMember(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if m == nil || !m.Active() {
		return ErrConversationNotFound
	}
	now := s.now().UTC()
	m.LeftAt = &now
	m.UpdatedAt = now
	return s.store.SaveMember(ctx, m)
}

// IsActiveMember 回傳使用者是否為對話的生效成員（供 Message
// 模組驗證寫入／讀取權限，依賴 DAG：Message --> Chat）。
func (s *Service) IsActiveMember(ctx context.Context, conversationID, userID string) (bool, error) {
	m, err := s.store.GetMember(ctx, conversationID, userID)
	if err != nil {
		return false, err
	}
	return m != nil && m.Active(), nil
}

// ConversationKind 供 Message 模組判斷對話類型與存在性。
type ConversationKind struct {
	Type   ConversationType
	Exists bool
}

// GetKind 讀取對話類型：不存在時 Exists=false，不得洩漏成員資訊。
func (s *Service) GetKind(ctx context.Context, conversationID string) (ConversationKind, error) {
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return ConversationKind{}, err
	}
	if conv == nil {
		return ConversationKind{}, nil
	}
	return ConversationKind{Type: conv.Type, Exists: true}, nil
}

// IsDirect 回傳對話是否為一對一對話（供 Message 的
// ConversationAccess 介面；不存在時回 false, nil）。
func (s *Service) IsDirect(ctx context.Context, conversationID string) (bool, error) {
	kind, err := s.GetKind(ctx, conversationID)
	if err != nil {
		return false, err
	}
	return kind.Exists && kind.Type == TypeDirect, nil
}

// DirectBlocked 判斷一對一對話中，另一成員與呼叫者之間是否存在
// 任一方向的生效封鎖（供 Message 在 DM 發送時驗證）。群組對話
// 一律回 false（群組內的封鎖語義待後續里程碑）。
func (s *Service) DirectBlocked(ctx context.Context, conversationID, userID string) (bool, error) {
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return false, err
	}
	if conv == nil || conv.Type != TypeDirect {
		return false, nil
	}
	members, err := s.store.ListMembers(ctx, conversationID)
	if err != nil {
		return false, err
	}
	for _, m := range members {
		if m.UserID == userID || !m.Active() {
			continue
		}
		return s.friends.BlockedBetween(ctx, userID, m.UserID)
	}
	return false, nil
}

// NoteMessage 更新對話的最後訊息快取與活動時間（供 Message 寫入
// 路徑在訊息持久化成功後呼叫；Message 不得直寫 conversations 表）。
func (s *Service) NoteMessage(ctx context.Context, conversationID, messageID string, at time.Time) error {
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return err
	}
	if conv == nil {
		return ErrConversationNotFound
	}
	conv.LastMessageID = messageID
	conv.LastMessageAt = &at
	conv.UpdatedAt = at
	return s.store.SaveConversation(ctx, conv)
}

// MarkRead 推進成員的已讀水位：序號不得倒退（seq 小於等於現有
// 水位時不動）。呼叫方須先確認成員身份（Message 服務層負責）。
func (s *Service) MarkRead(ctx context.Context, conversationID, userID, messageID string, seq int64, at time.Time) error {
	m, err := s.store.GetMember(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if m == nil || !m.Active() {
		return ErrConversationNotFound
	}
	if seq <= m.LastReadSequence {
		return nil
	}
	m.LastReadMessageID = messageID
	m.LastReadSequence = seq
	m.LastReadAt = &at
	m.UpdatedAt = at
	return s.store.SaveMember(ctx, m)
}

// ReadSequence 回傳成員的已讀序號水位與成員身份（供 Message
// 計算未讀數與驗證回條）。
func (s *Service) ReadSequence(ctx context.Context, conversationID, userID string) (int64, bool, error) {
	m, err := s.store.GetMember(ctx, conversationID, userID)
	if err != nil {
		return 0, false, err
	}
	if m == nil || !m.Active() {
		return 0, false, nil
	}
	return m.LastReadSequence, true, nil
}

// MemoryStore 是 Store 的記憶體實作（開發與測試用）。
type MemoryStore struct {
	mu            sync.RWMutex
	conversations map[string]*Conversation
	members       map[string]map[string]*Member // conversationID -> userID -> Member
}

// NewMemoryStore 建立空的記憶體對話儲存。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		conversations: map[string]*Conversation{},
		members:       map[string]map[string]*Member{},
	}
}

func cloneConversation(c *Conversation) *Conversation {
	if c == nil {
		return nil
	}
	cp := *c
	return &cp
}

func cloneMember(m *Member) *Member {
	if m == nil {
		return nil
	}
	cp := *m
	return &cp
}

func (s *MemoryStore) GetConversation(_ context.Context, id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConversation(s.conversations[id]), nil
}

func (s *MemoryStore) GetDirectBetween(_ context.Context, a, b string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.conversations {
		if c.Type != TypeDirect {
			continue
		}
		members := s.members[c.ID]
		if len(members) != 2 {
			continue
		}
		if _, ok := members[a]; !ok {
			continue
		}
		if _, ok := members[b]; !ok {
			continue
		}
		return cloneConversation(c), nil
	}
	return nil, nil
}

func (s *MemoryStore) ListConversationsByUser(_ context.Context, userID string) ([]*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Conversation
	for id, members := range s.members {
		if m, ok := members[userID]; ok && m.Active() {
			out = append(out, cloneConversation(s.conversations[id]))
		}
	}
	// 依 updated_at 倒序、id 為次鍵（決定性排序）。
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].UpdatedAt.After(out[i].UpdatedAt) ||
				(out[j].UpdatedAt.Equal(out[i].UpdatedAt) && out[j].ID < out[i].ID) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (s *MemoryStore) CreateConversation(_ context.Context, c *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.conversations[c.ID]; exists {
		return errors.New("chat: conversation already exists")
	}
	s.conversations[c.ID] = cloneConversation(c)
	return nil
}

func (s *MemoryStore) SaveConversation(_ context.Context, c *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.conversations[c.ID]; !exists {
		return ErrConversationNotFound
	}
	s.conversations[c.ID] = cloneConversation(c)
	return nil
}

func (s *MemoryStore) GetMember(_ context.Context, conversationID, userID string) (*Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneMember(s.members[conversationID][userID]), nil
}

func (s *MemoryStore) ListMembers(_ context.Context, conversationID string) ([]*Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Member
	for _, m := range s.members[conversationID] {
		out = append(out, cloneMember(m))
	}
	// 生效成員在前，依加入時間排序；離開者在後。
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			ai, aj := out[i].Active(), out[j].Active()
			if (!ai && aj) || (ai == aj && out[j].JoinedAt.Before(out[i].JoinedAt)) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (s *MemoryStore) CreateMember(_ context.Context, m *Member) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.members[m.ConversationID] == nil {
		s.members[m.ConversationID] = map[string]*Member{}
	}
	if _, exists := s.members[m.ConversationID][m.UserID]; exists {
		return errors.New("chat: member already exists")
	}
	s.members[m.ConversationID][m.UserID] = cloneMember(m)
	return nil
}

func (s *MemoryStore) SaveMember(_ context.Context, m *Member) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.members[m.ConversationID] == nil || s.members[m.ConversationID][m.UserID] == nil {
		return ErrConversationNotFound
	}
	s.members[m.ConversationID][m.UserID] = cloneMember(m)
	return nil
}
