package group

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"ether.local/api/internal/ids"
	"ether.local/api/internal/transport"
)

// ErrGroupNotFound 表示查無群組、呼叫者不是生效成員、群組已解散，
// 或權限不足而以「不存在」回應的情形。對外一律 404，不得區分原因，
// 不得洩漏群組存在性（ERROR_CODES.md §6.3）。
var ErrGroupNotFound = errors.New("group: group not found")

// Role 是群組成員角色。值域為本里程碑實作詞彙 owner／admin／
// member（ERD §6.4.2 稱 role 與 role_id 主從待定稿；本里程碑以
// group_members.role 文字欄為唯一角色來源，不建 group_roles 表，
// 自訂角色屬後續里程碑），待端點 Schema 定稿覆核。
type Role string

const (
	// RoleOwner 是群組擁有者：每群組恰好一名；僅擁有者可轉移
	// 所有權與解散群組。
	RoleOwner Role = "owner"
	// RoleAdmin 是群組管理員：可管理成員、邀請與公告。
	RoleAdmin Role = "admin"
	// RoleMember 是一般成員。
	RoleMember Role = "member"
)

// IsManager 回傳該角色是否具管理權限（擁有者或管理員）。
func (r Role) IsManager() bool { return r == RoleOwner || r == RoleAdmin }

// Group 是 groups 的一列。ConversationID 為群組聊天室容器的
// 識別碼（經 Chat 介面建立；為空表示未建容器）。解散為狀態標記，
// 不得物理刪除群組列與成員歷史。
type Group struct {
	ID             string
	Name           string
	Description    string
	AvatarMediaID  string
	ConversationID string
	CreatedBy      string
	IsDisbanded    bool
	DisbandedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Member 是 group_members 的一列。LeftAt 為空表示生效成員；
// 重新加入復用同一列（joined_at 更新、left_at 清空、角色重設
// 為 member）。InvitedBy 為邀請／直接加入的操作者（建立時為建立者）。
type Member struct {
	GroupID   string
	UserID    string
	Role      Role
	JoinedAt  time.Time
	LeftAt    *time.Time
	InvitedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Active 回傳該成員是否為生效成員（未離開／未被移除）。
func (m *Member) Active() bool { return m.LeftAt == nil }

// InvitationStatus 是群組邀請狀態。值域為本里程碑實作詞彙
// pending／accepted／declined／cancelled（ERD §6.4.4 稱值域待定稿，
// 待定稿覆核）。邀請不設到期（到期自動轉態須由排程／Worker 落實，
// 本里程碑不自動過期）；連結／QR 邀請不在範圍，invite_token 欄位
// 不建（見 migrations/000007 檔頭）。
type InvitationStatus string

const (
	InvitationPending   InvitationStatus = "pending"
	InvitationAccepted  InvitationStatus = "accepted"
	InvitationDeclined  InvitationStatus = "declined"
	InvitationCancelled InvitationStatus = "cancelled"
)

// Invitation 是 group_invitations 的一列（直接邀請：邀請者與
// 被邀請者皆明確）。紀錄不得物理刪除；回應／取消皆為狀態標記。
type Invitation struct {
	ID          string
	GroupID     string
	InviterID   string
	InviteeID   string
	Status      InvitationStatus
	RespondedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Announcement 是 group_announcements 的一列。同一群組同時僅一則
// 現行公告（is_active）：新公告發佈時舊公告自動撤下（部分唯一索引
// 兜底，見 migrations/000007）。撤下為狀態標記，不得物理刪除。
type Announcement struct {
	ID          string
	GroupID     string
	AuthorID    string
	Content     string
	IsActive    bool
	PublishedAt time.Time
	RetractedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// InvitePolicy 是群組邀請政策：僅管理者可邀請，或全體成員可邀請。
// 值域為本里程碑實作詞彙，待設定 Schema 定稿覆核。
type InvitePolicy string

const (
	// InvitePolicyAdmin 僅擁有者／管理員可建立邀請（預設）。
	InvitePolicyAdmin InvitePolicy = "admin"
	// InvitePolicyAll 全體生效成員皆可建立邀請。
	InvitePolicyAll InvitePolicy = "all"
)

// Settings 是群組設定（group_settings 鍵值表的本里程碑對映：
// 目前僅 invite_policy 一項鍵，未來鍵須經定稿後擴充，不得以
// 未定稿的鍵寫入）。UpdatedBy／UpdatedAt 為空表示未曾儲存（預設值）。
type Settings struct {
	GroupID      string
	InvitePolicy InvitePolicy
	UpdatedBy    string
	UpdatedAt    *time.Time
}

// ValidationError 承載欄位驗證失敗清單。
type ValidationError struct {
	Fields []transport.FieldError
}

func (e *ValidationError) Error() string { return "group: validation failed" }

// FieldErrors 供 transport.WriteServiceError 寫出 details。
func (e *ValidationError) FieldErrors() []transport.FieldError { return e.Fields }

func validationError(field, issue, message string) *ValidationError {
	return &ValidationError{Fields: []transport.FieldError{{Field: field, Issue: issue, Message: message}}}
}

// UserResolver 是 Group 對 User 模組的最小依賴（依賴 DAG：
// Group --> User）：僅確認使用者存在。
type UserResolver interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// FriendChecker 是 Group 對 Friend 模組的最小依賴（依賴 DAG：
// Group --> Friend）：僅讀取封鎖狀態，不得寫入好友表。邀請與
// 直接加入時，任一方向封鎖皆以「不存在」回應，不得洩漏封鎖狀態。
type FriendChecker interface {
	BlockedBetween(ctx context.Context, a, b string) (bool, error)
}

// ChatCollaborator 是 Group 對 Chat 模組的最小依賴（依賴 DAG：
// Group --> Chat）：群組聊天室容器的建立與成員同步。Group 不得
// 直接寫入 conversations／conversation_members 表；群組角色
// 不得代替對話成員權限（同步呼叫本身不攜帶群組角色）。
type ChatCollaborator interface {
	// CreateGroupConversation 建立群組對話容器，回傳對話識別碼。
	CreateGroupConversation(ctx context.Context, creatorID, title string, memberIDs []string) (string, error)
	// AddConversationMembers 將成員加入對話容器（權限已由 Group
	// 服務層裁決，本呼叫不重複檢查呼叫者角色）。
	AddConversationMembers(ctx context.Context, conversationID string, memberIDs []string) error
	// RemoveConversationMember 將成員移出對話容器（標記離開，
	// 對話與訊息歷史保留）。
	RemoveConversationMember(ctx context.Context, conversationID, userID string) error
}

// Store 是 Group 五張表的儲存介面。
type Store interface {
	GetGroup(ctx context.Context, id string) (*Group, error)
	ListGroupsByUser(ctx context.Context, userID string) ([]*Group, error)
	CreateGroup(ctx context.Context, g *Group) error
	SaveGroup(ctx context.Context, g *Group) error
	CountActiveMembers(ctx context.Context, groupID string) (int, error)

	GetMember(ctx context.Context, groupID, userID string) (*Member, error)
	ListMembers(ctx context.Context, groupID string) ([]*Member, error)
	CreateMember(ctx context.Context, m *Member) error
	SaveMember(ctx context.Context, m *Member) error

	GetInvitation(ctx context.Context, id string) (*Invitation, error)
	GetPendingInvitation(ctx context.Context, groupID, inviteeID string) (*Invitation, error)
	ListGroupInvitations(ctx context.Context, groupID string) ([]*Invitation, error)
	ListIncomingInvitations(ctx context.Context, userID string) ([]*Invitation, error)
	CreateInvitation(ctx context.Context, inv *Invitation) error
	SaveInvitation(ctx context.Context, inv *Invitation) error

	GetAnnouncement(ctx context.Context, id string) (*Announcement, error)
	ListAnnouncements(ctx context.Context, groupID string) ([]*Announcement, error)
	RetractActiveAnnouncements(ctx context.Context, groupID string, at time.Time) error
	CreateAnnouncement(ctx context.Context, a *Announcement) error
	SaveAnnouncement(ctx context.Context, a *Announcement) error

	GetSettings(ctx context.Context, groupID string) (*Settings, error)
	SaveSettings(ctx context.Context, s *Settings) error
}

// Service 是 Group 領域服務。
type Service struct {
	store   Store
	users   UserResolver
	friends FriendChecker
	chats   ChatCollaborator // 可為空：未接線時不建對話容器。
	now     func() time.Time
}

// NewService 建立 Service。now 為空時用 time.Now；chats 為空時
// 群組建立不建對話容器（ConversationID 留空），僅供測試。
func NewService(store Store, users UserResolver, friends FriendChecker, chats ChatCollaborator, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, users: users, friends: friends, chats: chats, now: now}
}

func validateName(name string) *ValidationError {
	n := utf8.RuneCountInString(name)
	if n == 0 || n > 64 {
		return validationError("name", "INVALID_VALUE", "群組名稱須為 1 至 64 字。")
	}
	return nil
}

func validateDescription(description string) *ValidationError {
	if utf8.RuneCountInString(description) > 500 {
		return validationError("description", "TOO_LONG", "群組介紹過長。")
	}
	return nil
}

func validateAvatarMediaID(avatarMediaID string) *ValidationError {
	if utf8.RuneCountInString(avatarMediaID) > 128 {
		return validationError("avatar_media_id", "TOO_LONG", "群組頭像媒體識別碼過長。")
	}
	return nil
}

// loadGroup 讀取群組列：不存在或已解散皆回 ErrGroupNotFound。
func (s *Service) loadGroup(ctx context.Context, groupID string) (*Group, error) {
	g, err := s.store.GetGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if g == nil || g.IsDisbanded {
		return nil, ErrGroupNotFound
	}
	return g, nil
}

// activeMember 讀取生效成員：群組不存在／已解散、非成員、已離開
// 皆回 ErrGroupNotFound（不得區分原因）。
func (s *Service) activeMember(ctx context.Context, groupID, userID string) (*Member, error) {
	if _, err := s.loadGroup(ctx, groupID); err != nil {
		return nil, err
	}
	m, err := s.store.GetMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if m == nil || !m.Active() {
		return nil, ErrGroupNotFound
	}
	return m, nil
}

// requireManager 要求呼叫者為生效的管理者（擁有者或管理員）；
// 非管理者一律回 ErrGroupNotFound（不得以權限差異洩漏群組存在性）。
func (s *Service) requireManager(ctx context.Context, groupID, userID string) (*Member, error) {
	m, err := s.activeMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if !m.Role.IsManager() {
		return nil, ErrGroupNotFound
	}
	return m, nil
}

func dedupeMemberIDs(creatorID string, memberIDs []string) []string {
	seen := map[string]bool{creatorID: true}
	out := make([]string, 0, len(memberIDs))
	for _, id := range memberIDs {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// checkInvitees 確認被邀請／加入對象皆存在且與操作者無任一方向
// 封鎖：不存在或封鎖皆回 ErrGroupNotFound，不得區分原因。
func (s *Service) checkInvitees(ctx context.Context, operatorID string, targetIDs []string) error {
	for _, id := range targetIDs {
		exists, err := s.users.Exists(ctx, id)
		if err != nil {
			return err
		}
		if !exists {
			return ErrGroupNotFound
		}
		blocked, err := s.friends.BlockedBetween(ctx, operatorID, id)
		if err != nil {
			return err
		}
		if blocked {
			return ErrGroupNotFound
		}
	}
	return nil
}

// CreateGroup 建立群組：建立者為擁有者，其餘成員為 member。
// 須至少一名其他成員（群組聊天室容器須有對話成員）；成員須存在
// 且與建立者無封鎖。對話容器先於群組列建立（失敗時群組不建立；
// 群組列建立失敗時殘留的對話容器為孤兒列，由對話側保留，不得
// 以此回溯刪除——跨模組無交易邊界，為已知限制）。
func (s *Service) CreateGroup(ctx context.Context, creatorID, name, description, avatarMediaID string, memberIDs []string) (*GroupEntry, error) {
	if ve := validateName(name); ve != nil {
		return nil, ve
	}
	if ve := validateDescription(description); ve != nil {
		return nil, ve
	}
	if ve := validateAvatarMediaID(avatarMediaID); ve != nil {
		return nil, ve
	}
	members := dedupeMemberIDs(creatorID, memberIDs)
	if len(members) == 0 {
		return nil, validationError("member_ids", "REQUIRED", "群組須至少包含一名其他成員。")
	}
	if err := s.checkInvitees(ctx, creatorID, members); err != nil {
		return nil, err
	}
	var conversationID string
	if s.chats != nil {
		convID, err := s.chats.CreateGroupConversation(ctx, creatorID, name, members)
		if err != nil {
			return nil, err
		}
		conversationID = convID
	}
	now := s.now().UTC()
	g := &Group{
		ID:             ids.New(),
		Name:           name,
		Description:    description,
		AvatarMediaID:  avatarMediaID,
		ConversationID: conversationID,
		CreatedBy:      creatorID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if g.ID == "" {
		return nil, errors.New("group: 識別碼產生失敗")
	}
	if err := s.store.CreateGroup(ctx, g); err != nil {
		return nil, err
	}
	if err := s.store.CreateMember(ctx, &Member{
		GroupID:   g.ID,
		UserID:    creatorID,
		Role:      RoleOwner,
		JoinedAt:  now,
		InvitedBy: creatorID,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return nil, err
	}
	for _, id := range members {
		if err := s.store.CreateMember(ctx, &Member{
			GroupID:   g.ID,
			UserID:    id,
			Role:      RoleMember,
			JoinedAt:  now,
			InvitedBy: creatorID,
			CreatedAt: now,
			UpdatedAt: now,
		}); err != nil {
			return nil, err
		}
	}
	return s.entryOf(ctx, g)
}

// GroupEntry 是群組查詢的一項：群組列＋生效成員數。
type GroupEntry struct {
	Group       Group
	MemberCount int
}

// GetGroup 讀取單一群組：僅生效成員可讀。
func (s *Service) GetGroup(ctx context.Context, userID, groupID string) (*GroupEntry, error) {
	if _, err := s.activeMember(ctx, groupID, userID); err != nil {
		return nil, err
	}
	g, err := s.loadGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	count, err := s.store.CountActiveMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return &GroupEntry{Group: *g, MemberCount: count}, nil
}

// ListGroups 列出使用者為生效成員的群組（已解散者排除），
// 依最近更新倒序。
func (s *Service) ListGroups(ctx context.Context, userID string) ([]GroupEntry, error) {
	groups, err := s.store.ListGroupsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]GroupEntry, 0, len(groups))
	for _, g := range groups {
		count, err := s.store.CountActiveMembers(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, GroupEntry{Group: *g, MemberCount: count})
	}
	return out, nil
}

// GroupPatch 是群組更新的指標語義：nil 表示不改；空字串表示清空
// （僅 description／avatar_media_id 可清空；name 不可為空）。
type GroupPatch struct {
	Name          *string
	Description   *string
	AvatarMediaID *string
}

// UpdateGroup 更新群組資料：僅管理者可更新。
func (s *Service) UpdateGroup(ctx context.Context, userID, groupID string, patch GroupPatch) (*GroupEntry, error) {
	if _, err := s.requireManager(ctx, groupID, userID); err != nil {
		return nil, err
	}
	g, err := s.loadGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if patch.Name != nil {
		if ve := validateName(*patch.Name); ve != nil {
			return nil, ve
		}
		g.Name = *patch.Name
	}
	if patch.Description != nil {
		if ve := validateDescription(*patch.Description); ve != nil {
			return nil, ve
		}
		g.Description = *patch.Description
	}
	if patch.AvatarMediaID != nil {
		if ve := validateAvatarMediaID(*patch.AvatarMediaID); ve != nil {
			return nil, ve
		}
		g.AvatarMediaID = *patch.AvatarMediaID
	}
	g.UpdatedAt = s.now().UTC()
	if err := s.store.SaveGroup(ctx, g); err != nil {
		return nil, err
	}
	return s.entryOf(ctx, g)
}

// entryOf 以群組列組裝 GroupEntry（成員數即時計數，不得以快取代替）。
func (s *Service) entryOf(ctx context.Context, g *Group) (*GroupEntry, error) {
	count, err := s.store.CountActiveMembers(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	return &GroupEntry{Group: *g, MemberCount: count}, nil
}

// Disband 解散群組：僅擁有者可解散。解散為狀態標記，成員列與
// 對話容器皆保留（訊息 append-only，不得連帶刪除）；解散後一切
// 群組端點皆以 404 回應。
func (s *Service) Disband(ctx context.Context, userID, groupID string) error {
	m, err := s.activeMember(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if m.Role != RoleOwner {
		return ErrGroupNotFound
	}
	g, err := s.loadGroup(ctx, groupID)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	g.IsDisbanded = true
	g.DisbandedAt = &now
	g.UpdatedAt = now
	return s.store.SaveGroup(ctx, g)
}

// TransferOwnership 轉移所有權：僅擁有者可轉移；對象須為生效
// 成員；原擁有者降為管理員，新擁有者接任（每群組恰好一名擁有者）。
func (s *Service) TransferOwnership(ctx context.Context, userID, groupID, newOwnerID string) (*GroupEntry, error) {
	m, err := s.activeMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if m.Role != RoleOwner {
		return nil, ErrGroupNotFound
	}
	if newOwnerID == userID {
		return nil, validationError("new_owner_id", "INVALID_VALUE", "所有權轉移對象不得為自己。")
	}
	target, err := s.store.GetMember(ctx, groupID, newOwnerID)
	if err != nil {
		return nil, err
	}
	if target == nil || !target.Active() {
		return nil, ErrGroupNotFound
	}
	now := s.now().UTC()
	target.Role = RoleOwner
	target.UpdatedAt = now
	if err := s.store.SaveMember(ctx, target); err != nil {
		return nil, err
	}
	m.Role = RoleAdmin
	m.UpdatedAt = now
	if err := s.store.SaveMember(ctx, m); err != nil {
		return nil, err
	}
	g, err := s.loadGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	g.UpdatedAt = now
	if err := s.store.SaveGroup(ctx, g); err != nil {
		return nil, err
	}
	return s.entryOf(ctx, g)
}

// ListMembers 列出群組的生效成員：僅生效成員可讀，依加入時間排序。
func (s *Service) ListMembers(ctx context.Context, userID, groupID string) ([]*Member, error) {
	if _, err := s.activeMember(ctx, groupID, userID); err != nil {
		return nil, err
	}
	return s.store.ListMembers(ctx, groupID)
}

// AddMembers 直接加入成員：僅管理者可操作。已生效者跳過；曾離開
// 者重新加入（角色重設為 member）。對象不存在或封鎖時回 404。
// 加入後同步對話容器成員。
func (s *Service) AddMembers(ctx context.Context, userID, groupID string, memberIDs []string) ([]*Member, error) {
	if _, err := s.requireManager(ctx, groupID, userID); err != nil {
		return nil, err
	}
	g, err := s.loadGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	targets := dedupeMemberIDs(userID, memberIDs)
	if err := s.checkInvitees(ctx, userID, targets); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	var added []*Member
	var addedIDs []string
	for _, id := range targets {
		m, err := s.store.GetMember(ctx, groupID, id)
		if err != nil {
			return nil, err
		}
		if m != nil && m.Active() {
			continue
		}
		if m == nil {
			m = &Member{
				GroupID:   groupID,
				UserID:    id,
				Role:      RoleMember,
				JoinedAt:  now,
				InvitedBy: userID,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := s.store.CreateMember(ctx, m); err != nil {
				return nil, err
			}
		} else {
			m.LeftAt = nil
			m.JoinedAt = now
			m.Role = RoleMember
			m.InvitedBy = userID
			m.UpdatedAt = now
			if err := s.store.SaveMember(ctx, m); err != nil {
				return nil, err
			}
		}
		added = append(added, m)
		addedIDs = append(addedIDs, id)
	}
	if len(addedIDs) > 0 && s.chats != nil && g.ConversationID != "" {
		if err := s.chats.AddConversationMembers(ctx, g.ConversationID, addedIDs); err != nil {
			return nil, err
		}
	}
	return added, nil
}

// SetRole 變更成員角色：僅擁有者可變更；僅可在 admin／member
// 間切換；不得變更自己、不得變更擁有者。變更後對話成員權限不受
// 影響（群組角色不得代替對話成員權限）。
func (s *Service) SetRole(ctx context.Context, userID, groupID, targetID string, role Role) (*Member, error) {
	m, err := s.activeMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if m.Role != RoleOwner {
		return nil, ErrGroupNotFound
	}
	if targetID == userID {
		return nil, validationError("user_id", "INVALID_VALUE", "不得變更自己的角色。")
	}
	if role != RoleAdmin && role != RoleMember {
		return nil, validationError("role", "INVALID_VALUE", "角色須為 admin 或 member。")
	}
	target, err := s.store.GetMember(ctx, groupID, targetID)
	if err != nil {
		return nil, err
	}
	if target == nil || !target.Active() || target.Role == RoleOwner {
		return nil, ErrGroupNotFound
	}
	if target.Role == role {
		return target, nil
	}
	target.Role = role
	target.UpdatedAt = s.now().UTC()
	if err := s.store.SaveMember(ctx, target); err != nil {
		return nil, err
	}
	return target, nil
}

// RemoveMember 移除成員：管理者可操作；不得移除擁有者；管理員
// 不得移除其他管理員；不得移除自己（請走離開流程）。移除為
// left_at 狀態標記，並同步移出對話容器。
func (s *Service) RemoveMember(ctx context.Context, userID, groupID, targetID string) error {
	self, err := s.requireManager(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if targetID == userID {
		return validationError("user_id", "INVALID_VALUE", "不得移除自己，請使用離開流程。")
	}
	g, err := s.loadGroup(ctx, groupID)
	if err != nil {
		return err
	}
	target, err := s.store.GetMember(ctx, groupID, targetID)
	if err != nil {
		return err
	}
	if target == nil || !target.Active() || target.Role == RoleOwner {
		return ErrGroupNotFound
	}
	if self.Role == RoleAdmin && target.Role == RoleAdmin {
		return ErrGroupNotFound
	}
	now := s.now().UTC()
	target.LeftAt = &now
	target.UpdatedAt = now
	if err := s.store.SaveMember(ctx, target); err != nil {
		return err
	}
	if s.chats != nil && g.ConversationID != "" {
		if err := s.chats.RemoveConversationMember(ctx, g.ConversationID, targetID); err != nil {
			return err
		}
	}
	return nil
}

// Leave 離開群組：生效成員皆可離開；擁有者須先轉移所有權或
// 解散群組，不得直接離開。離開為狀態標記，並同步移出對話容器。
func (s *Service) Leave(ctx context.Context, userID, groupID string) error {
	m, err := s.activeMember(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if m.Role == RoleOwner {
		return validationError("user_id", "INVALID_VALUE", "擁有者離開前須先轉移所有權或解散群組。")
	}
	g, err := s.loadGroup(ctx, groupID)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	m.LeftAt = &now
	m.UpdatedAt = now
	if err := s.store.SaveMember(ctx, m); err != nil {
		return err
	}
	if s.chats != nil && g.ConversationID != "" {
		if err := s.chats.RemoveConversationMember(ctx, g.ConversationID, userID); err != nil {
			return err
		}
	}
	return nil
}

// canInvite 判斷邀請者是否可建立邀請：管理者恆可；一般成員僅在
// invite_policy 為 all 時可邀請。
func (s *Service) canInvite(ctx context.Context, groupID string, m *Member) (bool, error) {
	if m.Role.IsManager() {
		return true, nil
	}
	settings, err := s.store.GetSettings(ctx, groupID)
	if err != nil {
		return false, err
	}
	if settings == nil {
		return false, nil
	}
	return settings.InvitePolicy == InvitePolicyAll, nil
}

// CreateInvitation 建立直接邀請：邀請者須為生效成員且具邀請權限；
// 被邀請者須存在、非邀請者本人、與邀請者無封鎖、非生效成員、
// 無待處理邀請（重複邀請為欄位錯誤；部分唯一索引兜底競態）。
func (s *Service) CreateInvitation(ctx context.Context, userID, groupID, inviteeID string) (*InvitationEntry, error) {
	m, err := s.activeMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	allowed, err := s.canInvite(ctx, groupID, m)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrGroupNotFound
	}
	if inviteeID == "" {
		return nil, validationError("invitee_id", "REQUIRED", "請提供被邀請者識別碼。")
	}
	if inviteeID == userID {
		return nil, validationError("invitee_id", "INVALID_VALUE", "不得邀請自己。")
	}
	if err := s.checkInvitees(ctx, userID, []string{inviteeID}); err != nil {
		return nil, err
	}
	existing, err := s.store.GetMember(ctx, groupID, inviteeID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Active() {
		return nil, validationError("invitee_id", "DUPLICATE", "對方已是群組成員。")
	}
	pending, err := s.store.GetPendingInvitation(ctx, groupID, inviteeID)
	if err != nil {
		return nil, err
	}
	if pending != nil {
		return nil, validationError("invitee_id", "DUPLICATE", "已送出待處理的群組邀請。")
	}
	now := s.now().UTC()
	inv := &Invitation{
		ID:        ids.New(),
		GroupID:   groupID,
		InviterID: userID,
		InviteeID: inviteeID,
		Status:    InvitationPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if inv.ID == "" {
		return nil, errors.New("group: 識別碼產生失敗")
	}
	if err := s.store.CreateInvitation(ctx, inv); err != nil {
		return nil, err
	}
	g, err := s.loadGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return &InvitationEntry{Invitation: *inv, GroupName: g.Name}, nil
}

// InvitationEntry 是邀請查詢的一項：邀請列＋群組名稱（顯示用）。
type InvitationEntry struct {
	Invitation Invitation
	GroupName  string
}

// ListGroupInvitations 列出群組的待處理邀請：僅管理者可讀。
func (s *Service) ListGroupInvitations(ctx context.Context, userID, groupID string) ([]InvitationEntry, error) {
	if _, err := s.requireManager(ctx, groupID, userID); err != nil {
		return nil, err
	}
	g, err := s.loadGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListGroupInvitations(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]InvitationEntry, 0, len(rows))
	for _, inv := range rows {
		out = append(out, InvitationEntry{Invitation: *inv, GroupName: g.Name})
	}
	return out, nil
}

// ListMyInvitations 列出呼叫者收到的待處理邀請（已解散群組的
// 邀請除外）；依建立時間排序。
func (s *Service) ListMyInvitations(ctx context.Context, userID string) ([]InvitationEntry, error) {
	rows, err := s.store.ListIncomingInvitations(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]InvitationEntry, 0, len(rows))
	for _, inv := range rows {
		g, err := s.store.GetGroup(ctx, inv.GroupID)
		if err != nil {
			return nil, err
		}
		if g == nil || g.IsDisbanded {
			continue
		}
		out = append(out, InvitationEntry{Invitation: *inv, GroupName: g.Name})
	}
	return out, nil
}

// RespondInvitation 回應邀請（接受／拒絕）：僅被邀請者可回應，
// 僅待處理邀請可回應。接受時建立（或恢復）成員列並同步對話容器；
// 群組已解散或回應時出現封鎖時，邀請視為不存在（404）。
func (s *Service) RespondInvitation(ctx context.Context, userID, invitationID string, accept bool) (*InvitationEntry, error) {
	inv, err := s.store.GetInvitation(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if inv == nil || inv.InviteeID != userID {
		return nil, ErrGroupNotFound
	}
	if inv.Status != InvitationPending {
		return nil, validationError("status", "INVALID_STATE", "邀請已不在待處理狀態。")
	}
	g, err := s.loadGroup(ctx, inv.GroupID)
	if err != nil {
		return nil, err
	}
	blocked, err := s.friends.BlockedBetween(ctx, inv.InviterID, userID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrGroupNotFound
	}
	now := s.now().UTC()
	if accept {
		m, err := s.store.GetMember(ctx, inv.GroupID, userID)
		if err != nil {
			return nil, err
		}
		if m != nil && m.Active() {
			return nil, validationError("invitee_id", "DUPLICATE", "對方已是群組成員。")
		}
		if m == nil {
			m = &Member{
				GroupID:   inv.GroupID,
				UserID:    userID,
				Role:      RoleMember,
				JoinedAt:  now,
				InvitedBy: inv.InviterID,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := s.store.CreateMember(ctx, m); err != nil {
				return nil, err
			}
		} else {
			m.LeftAt = nil
			m.JoinedAt = now
			m.Role = RoleMember
			m.InvitedBy = inv.InviterID
			m.UpdatedAt = now
			if err := s.store.SaveMember(ctx, m); err != nil {
				return nil, err
			}
		}
		if s.chats != nil && g.ConversationID != "" {
			if err := s.chats.AddConversationMembers(ctx, g.ConversationID, []string{userID}); err != nil {
				return nil, err
			}
		}
		inv.Status = InvitationAccepted
	} else {
		inv.Status = InvitationDeclined
	}
	inv.RespondedAt = &now
	inv.UpdatedAt = now
	if err := s.store.SaveInvitation(ctx, inv); err != nil {
		return nil, err
	}
	return &InvitationEntry{Invitation: *inv, GroupName: g.Name}, nil
}

// CancelInvitation 取消待處理邀請：邀請者本人或管理者可取消。
func (s *Service) CancelInvitation(ctx context.Context, userID, invitationID string) (*InvitationEntry, error) {
	inv, err := s.store.GetInvitation(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, ErrGroupNotFound
	}
	if inv.InviterID != userID {
		if _, err := s.requireManager(ctx, inv.GroupID, userID); err != nil {
			return nil, err
		}
	} else if _, err := s.loadGroup(ctx, inv.GroupID); err != nil {
		return nil, err
	}
	if inv.Status != InvitationPending {
		return nil, validationError("status", "INVALID_STATE", "邀請已不在待處理狀態。")
	}
	now := s.now().UTC()
	inv.Status = InvitationCancelled
	inv.RespondedAt = &now
	inv.UpdatedAt = now
	if err := s.store.SaveInvitation(ctx, inv); err != nil {
		return nil, err
	}
	g, err := s.loadGroup(ctx, inv.GroupID)
	if err != nil {
		return nil, err
	}
	return &InvitationEntry{Invitation: *inv, GroupName: g.Name}, nil
}

// CreateAnnouncement 發佈公告：僅管理者可發佈；內容 1–1000 runes。
// 發佈時自動撤下既有現行公告（同一群組同時僅一則現行公告）。
func (s *Service) CreateAnnouncement(ctx context.Context, userID, groupID, content string) (*Announcement, error) {
	if _, err := s.requireManager(ctx, groupID, userID); err != nil {
		return nil, err
	}
	if ve := validateAnnouncementContent(content); ve != nil {
		return nil, ve
	}
	now := s.now().UTC()
	if err := s.store.RetractActiveAnnouncements(ctx, groupID, now); err != nil {
		return nil, err
	}
	a := &Announcement{
		ID:          ids.New(),
		GroupID:     groupID,
		AuthorID:    userID,
		Content:     content,
		IsActive:    true,
		PublishedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if a.ID == "" {
		return nil, errors.New("group: 識別碼產生失敗")
	}
	if err := s.store.CreateAnnouncement(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func validateAnnouncementContent(content string) *ValidationError {
	n := utf8.RuneCountInString(strings.TrimSpace(content))
	if n == 0 || n > 1000 {
		return validationError("content", "INVALID_VALUE", "公告內容須為 1 至 1000 字。")
	}
	return nil
}

// ListAnnouncements 列出群組公告（含已撤下，含 is_active 標示）：
// 僅生效成員可讀，依發佈時間倒序。
func (s *Service) ListAnnouncements(ctx context.Context, userID, groupID string) ([]*Announcement, error) {
	if _, err := s.activeMember(ctx, groupID, userID); err != nil {
		return nil, err
	}
	return s.store.ListAnnouncements(ctx, groupID)
}

// UpdateAnnouncement 編輯公告：僅管理者可編輯，僅現行公告可編輯。
// 發佈時間不變；編輯時間記於 updated_at（編輯歷史不保留，待定稿）。
func (s *Service) UpdateAnnouncement(ctx context.Context, userID, groupID, announcementID, content string) (*Announcement, error) {
	if _, err := s.requireManager(ctx, groupID, userID); err != nil {
		return nil, err
	}
	if ve := validateAnnouncementContent(content); ve != nil {
		return nil, ve
	}
	a, err := s.store.GetAnnouncement(ctx, announcementID)
	if err != nil {
		return nil, err
	}
	if a == nil || a.GroupID != groupID || !a.IsActive {
		return nil, ErrGroupNotFound
	}
	a.Content = content
	a.UpdatedAt = s.now().UTC()
	if err := s.store.SaveAnnouncement(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// RetractAnnouncement 撤下公告：僅管理者可撤下，僅現行公告可撤下。
// 撤下為狀態標記，歷史保留。
func (s *Service) RetractAnnouncement(ctx context.Context, userID, groupID, announcementID string) (*Announcement, error) {
	if _, err := s.requireManager(ctx, groupID, userID); err != nil {
		return nil, err
	}
	a, err := s.store.GetAnnouncement(ctx, announcementID)
	if err != nil {
		return nil, err
	}
	if a == nil || a.GroupID != groupID || !a.IsActive {
		return nil, ErrGroupNotFound
	}
	now := s.now().UTC()
	a.IsActive = false
	a.RetractedAt = &now
	a.UpdatedAt = now
	if err := s.store.SaveAnnouncement(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// GetSettings 讀取群組設定：僅生效成員可讀；未曾儲存時回傳預設值。
func (s *Service) GetSettings(ctx context.Context, userID, groupID string) (*Settings, error) {
	if _, err := s.activeMember(ctx, groupID, userID); err != nil {
		return nil, err
	}
	settings, err := s.store.GetSettings(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return &Settings{GroupID: groupID, InvitePolicy: InvitePolicyAdmin}, nil
	}
	return settings, nil
}

// UpdateSettings 更新群組設定：僅管理者可更新。目前僅 invite_policy
// 一項鍵（nil 表示不改）；值域為 admin／all。
func (s *Service) UpdateSettings(ctx context.Context, userID, groupID string, invitePolicy *InvitePolicy) (*Settings, error) {
	if _, err := s.requireManager(ctx, groupID, userID); err != nil {
		return nil, err
	}
	current, err := s.GetSettings(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	if invitePolicy != nil {
		if *invitePolicy != InvitePolicyAdmin && *invitePolicy != InvitePolicyAll {
			return nil, validationError("invite_policy", "INVALID_VALUE", "邀請政策須為 admin 或 all。")
		}
		current.InvitePolicy = *invitePolicy
	}
	now := s.now().UTC()
	current.UpdatedBy = userID
	current.UpdatedAt = &now
	if err := s.store.SaveSettings(ctx, current); err != nil {
		return nil, err
	}
	return current, nil
}

// MemoryStore 是 Store 的記憶體實作（開發與測試用）。
type MemoryStore struct {
	mu            sync.RWMutex
	groups        map[string]*Group
	members       map[string]map[string]*Member // groupID -> userID -> Member
	invitations   map[string]*Invitation
	announcements map[string]*Announcement
	settings      map[string]*Settings
}

// NewMemoryStore 建立空的記憶體群組儲存。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		groups:        map[string]*Group{},
		members:       map[string]map[string]*Member{},
		invitations:   map[string]*Invitation{},
		announcements: map[string]*Announcement{},
		settings:      map[string]*Settings{},
	}
}

func cloneGroup(g *Group) *Group {
	if g == nil {
		return nil
	}
	cp := *g
	if g.DisbandedAt != nil {
		t := *g.DisbandedAt
		cp.DisbandedAt = &t
	}
	return &cp
}

func cloneMember(m *Member) *Member {
	if m == nil {
		return nil
	}
	cp := *m
	if m.LeftAt != nil {
		t := *m.LeftAt
		cp.LeftAt = &t
	}
	return &cp
}

func cloneInvitation(inv *Invitation) *Invitation {
	if inv == nil {
		return nil
	}
	cp := *inv
	if inv.RespondedAt != nil {
		t := *inv.RespondedAt
		cp.RespondedAt = &t
	}
	return &cp
}

func cloneAnnouncement(a *Announcement) *Announcement {
	if a == nil {
		return nil
	}
	cp := *a
	if a.RetractedAt != nil {
		t := *a.RetractedAt
		cp.RetractedAt = &t
	}
	return &cp
}

func cloneSettings(s *Settings) *Settings {
	if s == nil {
		return nil
	}
	cp := *s
	if s.UpdatedAt != nil {
		t := *s.UpdatedAt
		cp.UpdatedAt = &t
	}
	return &cp
}

func (s *MemoryStore) GetGroup(_ context.Context, id string) (*Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneGroup(s.groups[id]), nil
}

func (s *MemoryStore) ListGroupsByUser(_ context.Context, userID string) ([]*Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Group
	for id, members := range s.members {
		if m, ok := members[userID]; ok && m.Active() {
			if g := s.groups[id]; g != nil && !g.IsDisbanded {
				out = append(out, cloneGroup(g))
			}
		}
	}
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

func (s *MemoryStore) CreateGroup(_ context.Context, g *Group) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.groups[g.ID]; exists {
		return errors.New("group: group 已存在")
	}
	s.groups[g.ID] = cloneGroup(g)
	return nil
}

func (s *MemoryStore) SaveGroup(_ context.Context, g *Group) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.groups[g.ID]; !exists {
		return ErrGroupNotFound
	}
	s.groups[g.ID] = cloneGroup(g)
	return nil
}

func (s *MemoryStore) CountActiveMembers(_ context.Context, groupID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, m := range s.members[groupID] {
		if m.Active() {
			n++
		}
	}
	return n, nil
}

func (s *MemoryStore) GetMember(_ context.Context, groupID, userID string) (*Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneMember(s.members[groupID][userID]), nil
}

func (s *MemoryStore) ListMembers(_ context.Context, groupID string) ([]*Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Member
	for _, m := range s.members[groupID] {
		if m.Active() {
			out = append(out, cloneMember(m))
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].JoinedAt.Before(out[i].JoinedAt) ||
				(out[j].JoinedAt.Equal(out[i].JoinedAt) && out[j].UserID < out[i].UserID) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (s *MemoryStore) CreateMember(_ context.Context, m *Member) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.members[m.GroupID] == nil {
		s.members[m.GroupID] = map[string]*Member{}
	}
	if _, exists := s.members[m.GroupID][m.UserID]; exists {
		return errors.New("group: member 已存在")
	}
	s.members[m.GroupID][m.UserID] = cloneMember(m)
	return nil
}

func (s *MemoryStore) SaveMember(_ context.Context, m *Member) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.members[m.GroupID] == nil || s.members[m.GroupID][m.UserID] == nil {
		return ErrGroupNotFound
	}
	s.members[m.GroupID][m.UserID] = cloneMember(m)
	return nil
}

func (s *MemoryStore) GetInvitation(_ context.Context, id string) (*Invitation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneInvitation(s.invitations[id]), nil
}

func (s *MemoryStore) GetPendingInvitation(_ context.Context, groupID, inviteeID string) (*Invitation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var best *Invitation
	for _, inv := range s.invitations {
		if inv.GroupID == groupID && inv.InviteeID == inviteeID && inv.Status == InvitationPending {
			if best == nil || inv.CreatedAt.Before(best.CreatedAt) {
				best = inv
			}
		}
	}
	return cloneInvitation(best), nil
}

func (s *MemoryStore) ListGroupInvitations(_ context.Context, groupID string) ([]*Invitation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Invitation
	for _, inv := range s.invitations {
		if inv.GroupID == groupID && inv.Status == InvitationPending {
			out = append(out, cloneInvitation(inv))
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.Before(out[i].CreatedAt) ||
				(out[j].CreatedAt.Equal(out[i].CreatedAt) && out[j].ID < out[i].ID) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (s *MemoryStore) ListIncomingInvitations(_ context.Context, userID string) ([]*Invitation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Invitation
	for _, inv := range s.invitations {
		if inv.InviteeID == userID && inv.Status == InvitationPending {
			out = append(out, cloneInvitation(inv))
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.Before(out[i].CreatedAt) ||
				(out[j].CreatedAt.Equal(out[i].CreatedAt) && out[j].ID < out[i].ID) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (s *MemoryStore) CreateInvitation(_ context.Context, inv *Invitation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.invitations[inv.ID]; exists {
		return errors.New("group: invitation 已存在")
	}
	s.invitations[inv.ID] = cloneInvitation(inv)
	return nil
}

func (s *MemoryStore) SaveInvitation(_ context.Context, inv *Invitation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.invitations[inv.ID]; !ok {
		return ErrGroupNotFound
	}
	s.invitations[inv.ID] = cloneInvitation(inv)
	return nil
}

func (s *MemoryStore) GetAnnouncement(_ context.Context, id string) (*Announcement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneAnnouncement(s.announcements[id]), nil
}

func (s *MemoryStore) ListAnnouncements(_ context.Context, groupID string) ([]*Announcement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Announcement
	for _, a := range s.announcements {
		if a.GroupID == groupID {
			out = append(out, cloneAnnouncement(a))
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].PublishedAt.After(out[i].PublishedAt) ||
				(out[j].PublishedAt.Equal(out[i].PublishedAt) && out[j].ID < out[i].ID) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (s *MemoryStore) RetractActiveAnnouncements(_ context.Context, groupID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.announcements {
		if a.GroupID == groupID && a.IsActive {
			a.IsActive = false
			t := at
			a.RetractedAt = &t
			a.UpdatedAt = at
		}
	}
	return nil
}

func (s *MemoryStore) CreateAnnouncement(_ context.Context, a *Announcement) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.announcements[a.ID]; exists {
		return errors.New("group: announcement 已存在")
	}
	s.announcements[a.ID] = cloneAnnouncement(a)
	return nil
}

func (s *MemoryStore) SaveAnnouncement(_ context.Context, a *Announcement) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.announcements[a.ID]; !ok {
		return ErrGroupNotFound
	}
	s.announcements[a.ID] = cloneAnnouncement(a)
	return nil
}

func (s *MemoryStore) GetSettings(_ context.Context, groupID string) (*Settings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSettings(s.settings[groupID]), nil
}

func (s *MemoryStore) SaveSettings(_ context.Context, st *Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings[st.GroupID] = cloneSettings(st)
	return nil
}
