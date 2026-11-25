// Package friend 是 Friend（好友）模組的 Phase 04 實作。
//
// 責任邊界（MODULE_BOUNDARIES.md §5.4）：好友邀請（發送／接受／拒絕／
// 取消／過期／紀錄）、好友關係管理（刪除）、封鎖、隱藏、備註、分類、
// 好友搜尋與排序。封鎖／刪除／隱藏是三種不同狀態，分開建模，不得
// 混為一談。擁有資料表：friendships、friend_requests、blocked_users、
// hidden_users。聯絡人同步屬 Contact 模組（contact_sync_records），
// 不在本套件實作。
//
// 依賴（依賴 DAG，只允許箭頭方向）：User（存在確認）、Profile
// （公開檔案摘要）、Authentication（授權介面，由中介層落實）。
// 不得直接讀寫他模組的資料表；公開檔案一律經 ProfileResolver
// 讀介面，不得直連 user_profiles。
//
// 儲存語義（ERD §6.2 草案，差異見 migration 000005 檔頭，待定稿覆核）：
//   - friendships：同一對使用者僅一列，以 user_id < friend_id 的
//     字典序存放無序配對；status 為 active／ended，解除為狀態標記。
//   - friend_requests：待處理邀請在同一方向僅一筆（部分唯一索引）；
//     雙向同時待處理時，後送出者收到重複錯誤，須先回應對方的邀請。
//   - blocked_users／hidden_users：解除皆為狀態標記（unblocked_at／
//     unhidden_at），歷史不物理刪除；再次封鎖／隱藏建新列。
//   - 封鎖為單向：被封鎖者的一切好友操作皆以「不存在」回應，不得
//     洩漏封鎖狀態（ERROR_CODES.md §6.3）。
package friend

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"ether.local/api/internal/ids"
	"ether.local/api/internal/profile"
	"ether.local/api/internal/transport"
)

// ErrRequestNotFound 表示查無好友邀請。對外一律 404，不得區分
// 「不存在」與「無權存取」。
var ErrRequestNotFound = errors.New("friend: request not found")

// ErrFriendshipNotFound 表示查無好友關係。
var ErrFriendshipNotFound = errors.New("friend: friendship not found")

// ErrUserNotFound 表示查無目標使用者，或目標因封鎖關係不可見。
// 兩種情形對外皆為 404，不得以錯誤差異洩漏封鎖狀態。
var ErrUserNotFound = errors.New("friend: user not found")

// RequestStatus 是好友邀請狀態。值域為 M3 實作詞彙（pending／
// accepted／declined／cancelled／expired），待端點 Schema 定稿覆核。
type RequestStatus string

const (
	RequestPending   RequestStatus = "pending"
	RequestAccepted  RequestStatus = "accepted"
	RequestDeclined  RequestStatus = "declined"
	RequestCancelled RequestStatus = "cancelled"
	RequestExpired   RequestStatus = "expired"
)

// FriendshipStatus 是好友關係狀態。值域為 M3 實作詞彙（active／
// ended），待定稿覆核。
type FriendshipStatus string

const (
	FriendshipActive FriendshipStatus = "active"
	FriendshipEnded  FriendshipStatus = "ended"
)

// FriendRequest 是 friend_requests 的一列。Message 為邀請附言
// （M3 實作上限 200 runes，待定稿覆核）；ExpiresAt 為空表示
// 不設到期（到期自動轉態須由排程／Worker 落實，本階段不自動過期）。
type FriendRequest struct {
	ID          string
	RequesterID string
	AddresseeID string
	Status      RequestStatus
	Message     string
	ExpiresAt   *time.Time
	RespondedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Friendship 是 friendships 的一列，以 user_id < friend_id 的
// 字典序存放（FirstID、SecondID）；關係本身無方向。
type Friendship struct {
	ID            string
	FirstID       string
	SecondID      string
	Status        FriendshipStatus
	EstablishedAt *time.Time
	EndedAt       *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OtherID 回傳關係中不是 userID 的一端。
func (f *Friendship) OtherID(userID string) string {
	if f.FirstID == userID {
		return f.SecondID
	}
	return f.FirstID
}

// Block 是 blocked_users 的一列（單向封鎖）。Reason 為封鎖原因
// （M3 實作上限 200 runes，值域待定稿）；UnblockedAt 為空表示
// 封鎖生效中。
type Block struct {
	ID            string
	UserID        string
	BlockedUserID string
	Reason        string
	BlockedAt     time.Time
	UnblockedAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Hidden 是 hidden_users 的一列（單向隱藏）。UnhiddenAt 為空
// 表示隱藏生效中；隱藏不影響好友關係與訊息收發。
type Hidden struct {
	ID           string
	UserID       string
	HiddenUserID string
	HiddenAt     time.Time
	UnhiddenAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ValidationError 承載欄位驗證失敗清單。
type ValidationError struct {
	Fields []transport.FieldError
}

func (e *ValidationError) Error() string { return "friend: validation failed" }

// FieldErrors 供 transport.WriteServiceError 寫出 details。
func (e *ValidationError) FieldErrors() []transport.FieldError { return e.Fields }

// UserResolver 是 Friend 對 User 模組的最小依賴（依賴 DAG：
// Friend --> User）：僅確認使用者存在。
type UserResolver interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// ProfileResolver 是 Friend 對 Profile 模組的最小依賴（依賴 DAG：
// Friend --> Profile）：僅讀取公開檔案摘要，不得讀取私密欄位。
type ProfileResolver interface {
	GetByUserID(ctx context.Context, userID string) (*profile.Profile, error)
	GetByPublicID(ctx context.Context, publicID string) (*profile.Profile, error)
}

// ProfileSummary 是公開檔案摘要（搜尋與清單豐富化用）。
type ProfileSummary struct {
	UserID        string
	DisplayName   string
	PublicID      string
	AvatarMediaID string
	StatusMessage string
}

func summaryOf(p *profile.Profile) ProfileSummary {
	if p == nil {
		return ProfileSummary{}
	}
	return ProfileSummary{
		UserID:        p.UserID,
		DisplayName:   p.DisplayName,
		PublicID:      p.PublicID,
		AvatarMediaID: p.AvatarMediaID,
		StatusMessage: p.StatusMessage,
	}
}

// Store 是 Friend 四張表的儲存介面。
type Store interface {
	GetRequest(ctx context.Context, id string) (*FriendRequest, error)
	GetPendingBetween(ctx context.Context, a, b string) (*FriendRequest, error)
	ListIncomingPending(ctx context.Context, userID string) ([]*FriendRequest, error)
	ListOutgoingPending(ctx context.Context, userID string) ([]*FriendRequest, error)
	CreateRequest(ctx context.Context, req *FriendRequest) error
	SaveRequest(ctx context.Context, req *FriendRequest) error
	CancelPendingBetween(ctx context.Context, a, b string, at time.Time) error

	GetFriendship(ctx context.Context, a, b string) (*Friendship, error)
	ListFriendships(ctx context.Context, userID string) ([]*Friendship, error)
	CreateFriendship(ctx context.Context, f *Friendship) error
	SaveFriendship(ctx context.Context, f *Friendship) error

	GetActiveBlock(ctx context.Context, userID, blockedID string) (*Block, error)
	BlockedBetween(ctx context.Context, a, b string) (bool, error)
	ListBlocks(ctx context.Context, userID string) ([]*Block, error)
	CreateBlock(ctx context.Context, b *Block) error
	SaveBlock(ctx context.Context, b *Block) error

	GetActiveHidden(ctx context.Context, userID, hiddenID string) (*Hidden, error)
	ListHiddenIDs(ctx context.Context, userID string) ([]string, error)
	CreateHidden(ctx context.Context, h *Hidden) error
	SaveHidden(ctx context.Context, h *Hidden) error
}

// Service 是 Friend 領域服務。
type Service struct {
	store    Store
	users    UserResolver
	profiles ProfileResolver
	now      func() time.Time
}

// NewService 建立 Service。now 為空時用 time.Now。
func NewService(store Store, users UserResolver, profiles ProfileResolver, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, users: users, profiles: profiles, now: now}
}

// orderedPair 回傳字典序配對（first < second）。
func orderedPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

// SendRequest 發送好友邀請。目標不存在、自我邀請、已是好友、
// 任一方向封鎖中，皆不得建立邀請：自我／已是好友／重複待處理
// 為欄位錯誤；目標不存在與封鎖皆回傳 ErrUserNotFound（404，
// 不可區分，不得洩漏封鎖狀態）。
func (s *Service) SendRequest(ctx context.Context, requesterID, addresseeID, message string) (*FriendRequest, error) {
	var fields []transport.FieldError
	if requesterID == addresseeID {
		fields = append(fields, transport.FieldError{Field: "addressee_id", Issue: "INVALID_VALUE", Message: "不得對自己發送好友邀請。"})
	}
	if len([]rune(message)) > 200 {
		fields = append(fields, transport.FieldError{Field: "message", Issue: "TOO_LONG", Message: "邀請附言過長。"})
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	ok, err := s.users.Exists(ctx, addresseeID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrUserNotFound
	}
	blocked, err := s.store.BlockedBetween(ctx, requesterID, addresseeID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrUserNotFound
	}
	friendship, err := s.store.GetFriendship(ctx, requesterID, addresseeID)
	if err != nil {
		return nil, err
	}
	if friendship != nil && friendship.Status == FriendshipActive {
		return nil, &ValidationError{Fields: []transport.FieldError{{
			Field: "addressee_id", Issue: "DUPLICATE", Message: "對方已是好友。",
		}}}
	}
	pending, err := s.store.GetPendingBetween(ctx, requesterID, addresseeID)
	if err != nil {
		return nil, err
	}
	if pending != nil {
		if pending.RequesterID == requesterID {
			return nil, &ValidationError{Fields: []transport.FieldError{{
				Field: "addressee_id", Issue: "DUPLICATE", Message: "已送出待處理的好友邀請。",
			}}}
		}
		return nil, &ValidationError{Fields: []transport.FieldError{{
			Field: "addressee_id", Issue: "DUPLICATE", Message: "對方已送出好友邀請，請先回應對方的邀請。",
		}}}
	}
	now := s.now().UTC()
	req := &FriendRequest{
		ID:          ids.New(),
		RequesterID: requesterID,
		AddresseeID: addresseeID,
		Status:      RequestPending,
		Message:     message,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if req.ID == "" {
		return nil, errors.New("friend: 識別碼產生失敗")
	}
	if err := s.store.CreateRequest(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

// RespondRequest 回應邀請（接受／拒絕）。僅收件者可回應，
// 僅待處理邀請可回應；接受時建立好友關係列。
func (s *Service) RespondRequest(ctx context.Context, userID, requestID string, accept bool) (*FriendRequest, error) {
	req, err := s.store.GetRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.AddresseeID != userID {
		return nil, ErrRequestNotFound
	}
	if req.Status != RequestPending {
		return nil, &ValidationError{Fields: []transport.FieldError{{
			Field: "status", Issue: "INVALID_STATE", Message: "邀請已不在待處理狀態。",
		}}}
	}
	now := s.now().UTC()
	if accept {
		first, second := orderedPair(req.RequesterID, req.AddresseeID)
		friendship, err := s.store.GetFriendship(ctx, req.RequesterID, req.AddresseeID)
		if err != nil {
			return nil, err
		}
		if friendship == nil {
			friendship = &Friendship{
				ID:            ids.New(),
				FirstID:       first,
				SecondID:      second,
				Status:        FriendshipActive,
				EstablishedAt: &now,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			if friendship.ID == "" {
				return nil, errors.New("friend: 識別碼產生失敗")
			}
			if err := s.store.CreateFriendship(ctx, friendship); err != nil {
				return nil, err
			}
		} else if friendship.Status != FriendshipActive {
			friendship.Status = FriendshipActive
			friendship.EstablishedAt = &now
			friendship.EndedAt = nil
			friendship.UpdatedAt = now
			if err := s.store.SaveFriendship(ctx, friendship); err != nil {
				return nil, err
			}
		}
		req.Status = RequestAccepted
	} else {
		req.Status = RequestDeclined
	}
	req.RespondedAt = &now
	req.UpdatedAt = now
	if err := s.store.SaveRequest(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

// CancelRequest 取消自己送出的待處理邀請。僅發送者可取消。
func (s *Service) CancelRequest(ctx context.Context, userID, requestID string) (*FriendRequest, error) {
	req, err := s.store.GetRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.RequesterID != userID {
		return nil, ErrRequestNotFound
	}
	if req.Status != RequestPending {
		return nil, &ValidationError{Fields: []transport.FieldError{{
			Field: "status", Issue: "INVALID_STATE", Message: "邀請已不在待處理狀態。",
		}}}
	}
	now := s.now().UTC()
	req.Status = RequestCancelled
	req.RespondedAt = &now
	req.UpdatedAt = now
	if err := s.store.SaveRequest(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

// FriendEntry 是好友清單的一項：對端公開檔案摘要＋關係狀態。
type FriendEntry struct {
	Friend  ProfileSummary
	Hidden  bool
	AddedAt time.Time
}

// ListFriends 列出使用者的有效好友（預設排除隱藏者；
// includeHidden 為真時一併列出並標示 Hidden）。回傳依建立
// 時間排序（游標分頁待 API 定稿覆核，本階段回傳完整清單）。
func (s *Service) ListFriends(ctx context.Context, userID string, includeHidden bool) ([]FriendEntry, error) {
	rows, err := s.store.ListFriendships(ctx, userID)
	if err != nil {
		return nil, err
	}
	hiddenList, err := s.store.ListHiddenIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	hiddenIDs := map[string]bool{}
	for _, id := range hiddenList {
		hiddenIDs[id] = true
	}
	var out []FriendEntry
	for _, f := range rows {
		other := f.OtherID(userID)
		if hiddenIDs[other] && !includeHidden {
			continue
		}
		entry := FriendEntry{Hidden: hiddenIDs[other]}
		if f.EstablishedAt != nil {
			entry.AddedAt = *f.EstablishedAt
		} else {
			entry.AddedAt = f.CreatedAt
		}
		p, err := s.profiles.GetByUserID(ctx, other)
		if err != nil {
			if errors.Is(err, profile.ErrProfileNotFound) {
				entry.Friend = ProfileSummary{UserID: other}
			} else {
				return nil, err
			}
		} else {
			entry.Friend = summaryOf(p)
		}
		out = append(out, entry)
	}
	return out, nil
}

// RequestEntry 是邀請清單的一項：對端公開檔案摘要＋方向。
type RequestEntry struct {
	Request FriendRequest
	Other   ProfileSummary
	// Incoming 為真表示該邀請是呼叫者收到的（呼叫者為收件者）。
	Incoming bool
}

// ListRequests 列出待處理邀請（incoming：收到的；outgoing：送出的）。
func (s *Service) ListRequests(ctx context.Context, userID string, incoming bool) ([]RequestEntry, error) {
	var rows []*FriendRequest
	var err error
	if incoming {
		rows, err = s.store.ListIncomingPending(ctx, userID)
	} else {
		rows, err = s.store.ListOutgoingPending(ctx, userID)
	}
	if err != nil {
		return nil, err
	}
	out := make([]RequestEntry, 0, len(rows))
	for _, req := range rows {
		otherID := req.AddresseeID
		if incoming {
			otherID = req.RequesterID
		}
		entry := RequestEntry{Request: *req, Incoming: incoming}
		p, err := s.profiles.GetByUserID(ctx, otherID)
		if err != nil {
			if errors.Is(err, profile.ErrProfileNotFound) {
				entry.Other = ProfileSummary{UserID: otherID}
			} else {
				return nil, err
			}
		} else {
			entry.Other = summaryOf(p)
		}
		out = append(out, entry)
	}
	return out, nil
}

// SearchByPublicID 依公開識別碼搜尋使用者。查無、或任一方向
// 封鎖中，皆回傳 ErrUserNotFound（404，不可區分，不得洩漏封鎖狀態）。
func (s *Service) SearchByPublicID(ctx context.Context, userID, publicID string) (*ProfileSummary, error) {
	p, err := s.profiles.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, profile.ErrProfileNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if p.UserID == userID {
		return nil, ErrUserNotFound
	}
	blocked, err := s.store.BlockedBetween(ctx, userID, p.UserID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrUserNotFound
	}
	summary := summaryOf(p)
	return &summary, nil
}

// RemoveFriend 解除好友關係（狀態標記，歷史保留）。非好友或
// 已解除者回傳 404；重複解除冪等回傳已解除狀態。
func (s *Service) RemoveFriend(ctx context.Context, userID, friendID string) error {
	f, err := s.store.GetFriendship(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if f == nil || f.Status != FriendshipActive {
		return ErrFriendshipNotFound
	}
	now := s.now().UTC()
	f.Status = FriendshipEnded
	f.EndedAt = &now
	f.UpdatedAt = now
	return s.store.SaveFriendship(ctx, f)
}

// Block 封鎖對方（單向）。封鎖同時解除有效好友關係、取消雙方
// 待處理邀請；解除封鎖後好友關係不自動恢復，須重新邀請。
// 目標不存在回傳 404；已封鎖中則冪等回傳既有封鎖。
func (s *Service) Block(ctx context.Context, userID, blockedID, reason string) (*Block, bool, error) {
	if userID == blockedID {
		return nil, false, &ValidationError{Fields: []transport.FieldError{{
			Field: "blocked_user_id", Issue: "INVALID_VALUE", Message: "不得封鎖自己。",
		}}}
	}
	if len([]rune(reason)) > 200 {
		return nil, false, &ValidationError{Fields: []transport.FieldError{{
			Field: "reason", Issue: "TOO_LONG", Message: "封鎖原因過長。",
		}}}
	}
	ok, err := s.users.Exists(ctx, blockedID)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, ErrUserNotFound
	}
	existing, err := s.store.GetActiveBlock(ctx, userID, blockedID)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		return existing, false, nil
	}
	now := s.now().UTC()
	b := &Block{
		ID:            ids.New(),
		UserID:        userID,
		BlockedUserID: blockedID,
		Reason:        strings.TrimSpace(reason),
		BlockedAt:     now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if b.ID == "" {
		return nil, false, errors.New("friend: 識別碼產生失敗")
	}
	if err := s.store.CreateBlock(ctx, b); err != nil {
		return nil, false, err
	}
	if f, err := s.store.GetFriendship(ctx, userID, blockedID); err != nil {
		return nil, false, err
	} else if f != nil && f.Status == FriendshipActive {
		f.Status = FriendshipEnded
		f.EndedAt = &now
		f.UpdatedAt = now
		if err := s.store.SaveFriendship(ctx, f); err != nil {
			return nil, false, err
		}
	}
	if err := s.store.CancelPendingBetween(ctx, userID, blockedID, now); err != nil {
		return nil, false, err
	}
	return b, true, nil
}

// Unblock 解除封鎖（狀態標記，歷史保留）。未封鎖者回傳 404
// （不得洩漏是否曾封鎖）。
func (s *Service) Unblock(ctx context.Context, userID, blockedID string) (*Block, error) {
	b, err := s.store.GetActiveBlock(ctx, userID, blockedID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, ErrUserNotFound
	}
	now := s.now().UTC()
	b.UnblockedAt = &now
	b.UpdatedAt = now
	if err := s.store.SaveBlock(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

// BlockEntry 是封鎖清單的一項。
type BlockEntry struct {
	Block   Block
	Blocked ProfileSummary
}

// ListBlocks 列出使用者的生效封鎖。
func (s *Service) ListBlocks(ctx context.Context, userID string) ([]BlockEntry, error) {
	rows, err := s.store.ListBlocks(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]BlockEntry, 0, len(rows))
	for _, b := range rows {
		entry := BlockEntry{Block: *b}
		p, err := s.profiles.GetByUserID(ctx, b.BlockedUserID)
		if err != nil {
			if errors.Is(err, profile.ErrProfileNotFound) {
				entry.Blocked = ProfileSummary{UserID: b.BlockedUserID}
			} else {
				return nil, err
			}
		} else {
			entry.Blocked = summaryOf(p)
		}
		out = append(out, entry)
	}
	return out, nil
}

// SetHidden 設定／取消隱藏好友。僅有效好友可隱藏；未隱藏者
// 取消隱藏回傳 404（不得洩漏隱藏狀態以外的資訊）。
func (s *Service) SetHidden(ctx context.Context, userID, friendID string, hidden bool) error {
	f, err := s.store.GetFriendship(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if f == nil || f.Status != FriendshipActive {
		return ErrFriendshipNotFound
	}
	now := s.now().UTC()
	existing, err := s.store.GetActiveHidden(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if hidden {
		if existing != nil {
			return nil
		}
		h := &Hidden{
			ID:           ids.New(),
			UserID:       userID,
			HiddenUserID: friendID,
			HiddenAt:     now,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if h.ID == "" {
			return errors.New("friend: 識別碼產生失敗")
		}
		return s.store.CreateHidden(ctx, h)
	}
	if existing == nil {
		return ErrFriendshipNotFound
	}
	existing.UnhiddenAt = &now
	existing.UpdatedAt = now
	return s.store.SaveHidden(ctx, existing)
}

// AreFriends 回傳兩人是否為有效好友（供 Chat 模組驗證一對一對話
// 的建立前提，依賴 DAG：Chat --> Friend）。唯讀，不洩漏封鎖狀態。
func (s *Service) AreFriends(ctx context.Context, a, b string) (bool, error) {
	f, err := s.store.GetFriendship(ctx, a, b)
	if err != nil {
		return false, err
	}
	return f != nil && f.Status == FriendshipActive, nil
}

// BlockedBetween 回傳兩人之間任一方向是否存在生效封鎖（供 Chat／
// Message 模組在 DM 路徑上驗證，依賴 DAG：Chat／Message --> Friend）。
// 呼叫方須以「不存在」回應，不得向被封鎖者揭露封鎖狀態。
func (s *Service) BlockedBetween(ctx context.Context, a, b string) (bool, error) {
	return s.store.BlockedBetween(ctx, a, b)
}

// MemoryStore 是 Store 的記憶體實作（開發與測試用）。
type MemoryStore struct {
	mu          sync.RWMutex
	requests    map[string]*FriendRequest
	friendships map[string]*Friendship // key：firstID + "\x00" + secondID
	blocks      map[string]*Block
	hidden      map[string]*Hidden
}

// NewMemoryStore 建立空的記憶體好友儲存。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		requests:    map[string]*FriendRequest{},
		friendships: map[string]*Friendship{},
		blocks:      map[string]*Block{},
		hidden:      map[string]*Hidden{},
	}
}

func friendshipKey(a, b string) string {
	first, second := orderedPair(a, b)
	return first + "\x00" + second
}

func cloneRequest(r *FriendRequest) *FriendRequest {
	if r == nil {
		return nil
	}
	cp := *r
	if r.ExpiresAt != nil {
		t := *r.ExpiresAt
		cp.ExpiresAt = &t
	}
	if r.RespondedAt != nil {
		t := *r.RespondedAt
		cp.RespondedAt = &t
	}
	return &cp
}

func cloneFriendship(f *Friendship) *Friendship {
	if f == nil {
		return nil
	}
	cp := *f
	if f.EstablishedAt != nil {
		t := *f.EstablishedAt
		cp.EstablishedAt = &t
	}
	if f.EndedAt != nil {
		t := *f.EndedAt
		cp.EndedAt = &t
	}
	return &cp
}

func cloneBlock(b *Block) *Block {
	if b == nil {
		return nil
	}
	cp := *b
	if b.UnblockedAt != nil {
		t := *b.UnblockedAt
		cp.UnblockedAt = &t
	}
	return &cp
}

func cloneHidden(h *Hidden) *Hidden {
	if h == nil {
		return nil
	}
	cp := *h
	if h.UnhiddenAt != nil {
		t := *h.UnhiddenAt
		cp.UnhiddenAt = &t
	}
	return &cp
}

func (m *MemoryStore) GetRequest(_ context.Context, id string) (*FriendRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.requests[id]
	if !ok {
		return nil, ErrRequestNotFound
	}
	return cloneRequest(r), nil
}

func (m *MemoryStore) GetPendingBetween(_ context.Context, a, b string) (*FriendRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.requests {
		if r.Status == RequestPending &&
			((r.RequesterID == a && r.AddresseeID == b) || (r.RequesterID == b && r.AddresseeID == a)) {
			return cloneRequest(r), nil
		}
	}
	return nil, nil
}

func (m *MemoryStore) ListIncomingPending(_ context.Context, userID string) ([]*FriendRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*FriendRequest
	for _, r := range m.requests {
		if r.AddresseeID == userID && r.Status == RequestPending {
			out = append(out, cloneRequest(r))
		}
	}
	return out, nil
}

func (m *MemoryStore) ListOutgoingPending(_ context.Context, userID string) ([]*FriendRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*FriendRequest
	for _, r := range m.requests {
		if r.RequesterID == userID && r.Status == RequestPending {
			out = append(out, cloneRequest(r))
		}
	}
	return out, nil
}

func (m *MemoryStore) CreateRequest(_ context.Context, req *FriendRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.requests[req.ID]; exists {
		return errors.New("friend: request 已存在")
	}
	m.requests[req.ID] = cloneRequest(req)
	return nil
}

func (m *MemoryStore) SaveRequest(_ context.Context, req *FriendRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.requests[req.ID]; !ok {
		return ErrRequestNotFound
	}
	m.requests[req.ID] = cloneRequest(req)
	return nil
}

func (m *MemoryStore) CancelPendingBetween(_ context.Context, a, b string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.requests {
		if r.Status == RequestPending &&
			((r.RequesterID == a && r.AddresseeID == b) || (r.RequesterID == b && r.AddresseeID == a)) {
			r.Status = RequestCancelled
			t := at
			r.RespondedAt = &t
			r.UpdatedAt = at
		}
	}
	return nil
}

func (m *MemoryStore) GetFriendship(_ context.Context, a, b string) (*Friendship, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.friendships[friendshipKey(a, b)]
	if !ok {
		return nil, nil
	}
	return cloneFriendship(f), nil
}

func (m *MemoryStore) ListFriendships(_ context.Context, userID string) ([]*Friendship, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Friendship
	for _, f := range m.friendships {
		if f.Status == FriendshipActive && (f.FirstID == userID || f.SecondID == userID) {
			out = append(out, cloneFriendship(f))
		}
	}
	return out, nil
}

func (m *MemoryStore) CreateFriendship(_ context.Context, f *Friendship) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := friendshipKey(f.FirstID, f.SecondID)
	if _, exists := m.friendships[key]; exists {
		return errors.New("friend: friendship 已存在")
	}
	m.friendships[key] = cloneFriendship(f)
	return nil
}

func (m *MemoryStore) SaveFriendship(_ context.Context, f *Friendship) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := friendshipKey(f.FirstID, f.SecondID)
	if _, ok := m.friendships[key]; !ok {
		return ErrFriendshipNotFound
	}
	m.friendships[key] = cloneFriendship(f)
	return nil
}

func (m *MemoryStore) GetActiveBlock(_ context.Context, userID, blockedID string) (*Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, b := range m.blocks {
		if b.UserID == userID && b.BlockedUserID == blockedID && b.UnblockedAt == nil {
			return cloneBlock(b), nil
		}
	}
	return nil, nil
}

func (m *MemoryStore) BlockedBetween(_ context.Context, a, b string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, blk := range m.blocks {
		if blk.UnblockedAt == nil &&
			((blk.UserID == a && blk.BlockedUserID == b) || (blk.UserID == b && blk.BlockedUserID == a)) {
			return true, nil
		}
	}
	return false, nil
}

func (m *MemoryStore) ListBlocks(_ context.Context, userID string) ([]*Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Block
	for _, b := range m.blocks {
		if b.UserID == userID && b.UnblockedAt == nil {
			out = append(out, cloneBlock(b))
		}
	}
	return out, nil
}

func (m *MemoryStore) CreateBlock(_ context.Context, b *Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.blocks[b.ID]; exists {
		return errors.New("friend: block 已存在")
	}
	m.blocks[b.ID] = cloneBlock(b)
	return nil
}

func (m *MemoryStore) SaveBlock(_ context.Context, b *Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.blocks[b.ID]; !ok {
		return ErrUserNotFound
	}
	m.blocks[b.ID] = cloneBlock(b)
	return nil
}

func (m *MemoryStore) GetActiveHidden(_ context.Context, userID, hiddenID string) (*Hidden, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, h := range m.hidden {
		if h.UserID == userID && h.HiddenUserID == hiddenID && h.UnhiddenAt == nil {
			return cloneHidden(h), nil
		}
	}
	return nil, nil
}

func (m *MemoryStore) ListHiddenIDs(_ context.Context, userID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []string
	for _, h := range m.hidden {
		if h.UserID == userID && h.UnhiddenAt == nil {
			out = append(out, h.HiddenUserID)
		}
	}
	return out, nil
}

func (m *MemoryStore) CreateHidden(_ context.Context, h *Hidden) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.hidden[h.ID]; exists {
		return errors.New("friend: hidden 已存在")
	}
	m.hidden[h.ID] = cloneHidden(h)
	return nil
}

func (m *MemoryStore) SaveHidden(_ context.Context, h *Hidden) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.hidden[h.ID]; !ok {
		return ErrFriendshipNotFound
	}
	m.hidden[h.ID] = cloneHidden(h)
	return nil
}
