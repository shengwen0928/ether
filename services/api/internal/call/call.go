// Package call 是 Call（通話）模組的領域實作（Phase 07 離線可驗證部分）。
//
// 責任邊界（MODULE_BOUNDARIES.md §5.12）：通話會話管理、通話紀錄、
// 參與者、通話事件。媒體面屬 LiveKit，本模組只負責會話與紀錄。
// 擁有資料表：calls、call_participants、call_events。
// 允許依賴（依賴 DAG）：User、Chat、Group、Authentication、
// Device Management。只經唯讀介面讀取他模組資料，不得直連他模組
// 資料表，不得 import 他模組的內部儲存。
//
// 禁止事項：不得儲存或記錄通話內容；不得替 LiveKit 做媒體轉發決策；
// 不得在信令中夾帶 Token 或金鑰材料；不得繞過 Friend 封鎖檢查發起
// 通話（封鎖檢查經 Chat 介面的 DirectBlocked，不得直連 Friend 表）。
//
// 範圍（本里程碑）：發起通話（一對一／群組）、接聽／拒接／忙線／
// 未接、結束通話、參與者列出、通話事件紀錄（僅狀態事件）、通話
// 紀錄查詢。超出範圍者一律未實作：LiveKit 房間建立、Access Token
// 簽發、媒體品質資料寫入、通話中即時信令（信令協調經 Gateway，
// 介面只留 SignalCoordinator 佔位，不得實作假信令）；來電推播經
// Notification 事件，屬後續里程碑。
//
// 狀態機（實作詞彙，待定稿覆核）：ringing → active（接聽）／
// declined（全員拒接）／busy（全員忙線）／missed（響鈴中結束且
// 無人接聽）；active → ended；ringing → ended（已有接聽者時結束）。
// 通話結束為終端狀態，不得重開；結束原因 end_reason 僅 hangup／
// declined／missed／busy 四值。started_at 為首次接聽時間，未接聽
// 前為空；duration_seconds 僅在有接聽期間的結束通話才有值。
package call

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"ether.local/api/internal/chat"
	"ether.local/api/internal/group"
	"ether.local/api/internal/ids"
	"ether.local/api/internal/transport"
)

// ErrCallNotFound 表示查無通話、呼叫者不是參與者，或權限不足而以
// 「不存在」回應的情形。對外一律 404，不得區分原因，不得洩漏通話
// 存在性（ERROR_CODES.md §6.3）。
var ErrCallNotFound = errors.New("call: call not found")

// CallType 是通話類型。值域為本里程碑實作詞彙 voice／video
// （WEBSOCKET_PROTOCOL.md §6.5.1 稱值域待定稿，待定稿覆核）。
type CallType string

const (
	// CallTypeVoice 是語音通話。
	CallTypeVoice CallType = "voice"
	// CallTypeVideo 是視訊通話。
	CallTypeVideo CallType = "video"
)

// Status 是通話狀態。值域為本里程碑實作詞彙（ERD §6.9.1 稱值域
// 待定稿，待定稿覆核）：ringing 響鈴中、active 通話中、declined
// 全員拒接、missed 未接、busy 全員忙線、ended 已結束。
type Status string

const (
	StatusRinging  Status = "ringing"
	StatusActive   Status = "active"
	StatusDeclined Status = "declined"
	StatusMissed   Status = "missed"
	StatusBusy     Status = "busy"
	StatusEnded    Status = "ended"
)

// IsTerminal 回傳該狀態是否為終端狀態（結束後不得再轉移）。
func (s Status) IsTerminal() bool {
	switch s {
	case StatusDeclined, StatusMissed, StatusBusy, StatusEnded:
		return true
	default:
		return false
	}
}

// EndReason 是通話結束原因。值域為本里程碑實作詞彙（WEBSOCKET
// PROTOCOL.md §6.5.3 稱值域待定稿，待定稿覆核）：hangup 掛斷、
// declined 拒接、missed 未接、busy 忙線。
type EndReason string

const (
	EndReasonHangup   EndReason = "hangup"
	EndReasonDeclined EndReason = "declined"
	EndReasonMissed   EndReason = "missed"
	EndReasonBusy     EndReason = "busy"
)

// ParticipantResult 是參與者結果。值域為本里程碑實作詞彙
// （ERD §6.9.2 稱值域待定稿，待定稿覆核）：pending 待回應、
// joined 已加入、declined 已拒接、busy 忙線、missed 未接。
type ParticipantResult string

const (
	ResultPending  ParticipantResult = "pending"
	ResultJoined   ParticipantResult = "joined"
	ResultDeclined ParticipantResult = "declined"
	ResultBusy     ParticipantResult = "busy"
	ResultMissed   ParticipantResult = "missed"
)

// EventType 是通話事件類型。值域為本里程碑實作詞彙（ERD §6.9.3
// 稱值域待 Phase 07 定稿，待定稿覆核）：initiated 發起、answered
// 接聽、declined 拒接、busy 忙線、ended 結束。事件 payload 只含
// 狀態轉移資料（from_status／to_status／reason），不得含 SDP／ICE、
// Token、金鑰材料或任何通話內容。
type EventType string

const (
	EventInitiated EventType = "initiated"
	EventAnswered  EventType = "answered"
	EventDeclined  EventType = "declined"
	EventBusy      EventType = "busy"
	EventEnded     EventType = "ended"
)

// Call 是 calls 的一列。ConversationID 與 GroupID 恰好其一有值：
// 一對一通話掛對話容器，群組通話掛群組。StartedAt 為首次接聽時間
// （未接聽前為空，不得以建立時間代替）；DurationSeconds 僅在有接聽
// 期間的結束通話才有值。通話紀錄不得物理刪除。
type Call struct {
	ID             string
	ConversationID string
	GroupID        string
	InitiatorID    string
	Type           CallType
	Status         Status
	StartedAt      *time.Time
	EndedAt        *time.Time
	Duration       *int64
	EndedBy        string
	EndReason      EndReason
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Participant 是 call_participants 的一列。每通話每使用者恰好一列
// （(call_id, user_id) 唯一）；通話結束為終端狀態，不建重新加入列。
// JoinedAt 為空表示未加入；LeftAt 為結束時已加入者的離開時間。
// 紀錄 append-only，不得物理刪除。
type Participant struct {
	ID        string
	CallID    string
	UserID    string
	JoinedAt  *time.Time
	LeftAt    *time.Time
	Result    ParticipantResult
	CreatedAt time.Time
}

// IsPending 回傳該參與者是否仍待回應（響鈴中未接聽／拒接／忙線）。
func (p *Participant) IsPending() bool { return p.Result == ResultPending }

// CallEvent 是 call_events 的一列。Payload 為 JSON 文字，只含狀態
// 轉移資料，不得含 SDP／ICE、Token、金鑰材料或通話內容。
// OccurredAt 為事件實際發生時間，不得以寫入時間代替。事件紀錄
// append-only，不得修改、不得提前刪除。
type CallEvent struct {
	ID         string
	CallID     string
	UserID     string
	Type       EventType
	Payload    string
	OccurredAt time.Time
	CreatedAt  time.Time
}

// eventPayload 組裝事件 payload：只含狀態轉移資料。
func eventPayload(from, to Status, reason EndReason) string {
	m := map[string]string{"from_status": string(from), "to_status": string(to)}
	if reason != "" {
		m["reason"] = string(reason)
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// ValidationError 承載欄位驗證失敗清單。
type ValidationError struct {
	Fields []transport.FieldError
}

func (e *ValidationError) Error() string { return "call: validation failed" }

// FieldErrors 供 transport.WriteServiceError 寫出 details。
func (e *ValidationError) FieldErrors() []transport.FieldError { return e.Fields }

func validationError(field, issue, message string) *ValidationError {
	return &ValidationError{Fields: []transport.FieldError{{Field: field, Issue: issue, Message: message}}}
}

// UserResolver 是 Call 對 User 模組的最小依賴（依賴 DAG：
// Call --> User）：僅確認使用者存在。
type UserResolver interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// ConversationAccess 是 Call 對 Chat 模組的最小依賴（依賴 DAG：
// Call --> Chat）：僅讀取對話成員關係與一對一封鎖狀態，不得寫入
// 對話表。沿 Message 模組 ConversationAccess 介面的同一組方法。
type ConversationAccess interface {
	// IsActiveMember 回傳使用者是否為對話的生效成員。
	IsActiveMember(ctx context.Context, conversationID, userID string) (bool, error)
	// IsDirect 回傳對話是否為一對一對話（不存在時回 false, nil）。
	IsDirect(ctx context.Context, conversationID string) (bool, error)
	// DirectBlocked 回傳一對一對話中另一成員與呼叫者間是否存在
	// 任一方向的生效封鎖（群組對話回 false）。
	DirectBlocked(ctx context.Context, conversationID, userID string) (bool, error)
	// ListMembers 列出對話成員（呼叫者須為生效成員，否則 404 語義
	// 由 Chat 服務層決定）；供一對一通話解析另一成員。
	ListMembers(ctx context.Context, userID, conversationID string) ([]*chat.Member, error)
}

// GroupChecker 是 Call 對 Group 模組的最小依賴（依賴 DAG：
// Call --> Group）：僅讀取群組成員清單，不得寫入群組表。呼叫者
// 須為生效成員，否則 Group 服務層以「不存在」回應。
type GroupChecker interface {
	ListMembers(ctx context.Context, userID, groupID string) ([]*group.Member, error)
}

// SignalCoordinator 是 Call 對 Gateway 信令協調的佔位介面（本里程碑
// 未接線）：Gateway 落地後實作來電／接聽／結束的即時信令扇出。
// 本里程碑 Service 不依賴此介面，不得以任何假實作冒充信令已通；
// 客戶端以 REST 查詢同步通話狀態。
type SignalCoordinator interface {
	NotifyIncoming(ctx context.Context, call *Call, inviteeIDs []string) error
	NotifyAccepted(ctx context.Context, call *Call, accepterID string) error
	NotifyEnded(ctx context.Context, call *Call, endedBy string) error
}

// Store 是 Call 三張表的儲存介面。查無列時 Get* 回傳 (nil, nil)，
// 由服務層決定 404 語義（不得洩漏存在性）。
type Store interface {
	GetCall(ctx context.Context, id string) (*Call, error)
	CreateCall(ctx context.Context, c *Call) error
	SaveCall(ctx context.Context, c *Call) error
	// FindActiveCall 回傳同一範圍（對話或群組）內進行中（響鈴／通話
	// 中）的通話；無則回傳 (nil, nil)。conversationID 與 groupID
	// 恰好其一有值。
	FindActiveCall(ctx context.Context, conversationID, groupID string) (*Call, error)
	// ListCallsByUser 列出使用者參與的通話，依建立時間倒序；cursor
	// 為 (created_at, id) 倒序鍵（空字串為第一頁），limit 為本頁上限。
	ListCallsByUser(ctx context.Context, userID, cursor string, limit int) ([]*Call, error)

	GetParticipant(ctx context.Context, callID, userID string) (*Participant, error)
	ListParticipants(ctx context.Context, callID string) ([]*Participant, error)
	CreateParticipant(ctx context.Context, p *Participant) error
	SaveParticipant(ctx context.Context, p *Participant) error

	CreateEvent(ctx context.Context, e *CallEvent) error
	ListEvents(ctx context.Context, callID string) ([]*CallEvent, error)
}

// CallEntry 是通話及其參與者計數（清單／詳情回應用）。
type CallEntry struct {
	Call             *Call
	ParticipantCount int
}

// Service 是 Call 領域服務。
type Service struct {
	store  Store
	users  UserResolver
	chats  ConversationAccess
	groups GroupChecker
	now    func() time.Time
}

// NewService 建立 Service。now 為空時用 time.Now。users／chats／
// groups 為空時對應的唯讀檢查以「不存在」回應（fail-closed，僅供
// 測試接線不全時使用；正式 main.go 一律接線）。
func NewService(store Store, users UserResolver, chats ConversationAccess, groups GroupChecker, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, users: users, chats: chats, groups: groups, now: now}
}

func validateCallType(t CallType) *ValidationError {
	switch t {
	case CallTypeVoice, CallTypeVideo:
		return nil
	default:
		return validationError("call_type", "INVALID_VALUE", "通話類型須為 voice 或 video。")
	}
}

// InitiateInput 是發起通話的輸入：ConversationID 與 GroupID 恰好
// 其一有值。
type InitiateInput struct {
	ConversationID string
	GroupID        string
	Type           CallType
}

// Initiate 發起通話：呼叫者為發起人（已加入參與者），其餘範圍成員
// 為待回應受邀者。一對一通話須為直接對話、任一方向封鎖一律 404；
// 群組通話受邀者為全部其他生效成員。同一範圍內已有進行中通話時
// 拒絕（避免重複響鈴）。新通話狀態為 ringing，寫入 initiated 事件。
func (s *Service) Initiate(ctx context.Context, callerID string, in InitiateInput) (*CallEntry, error) {
	if err := validateCallType(in.Type); err != nil {
		return nil, err
	}
	if (in.ConversationID == "" && in.GroupID == "") || (in.ConversationID != "" && in.GroupID != "") {
		return nil, validationError("conversation_id", "INVALID_VALUE", "須指定 conversation_id 或 group_id 其中之一。")
	}
	if s.users != nil {
		exists, err := s.users.Exists(ctx, callerID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrCallNotFound
		}
	}
	var inviteeIDs []string
	if in.ConversationID != "" {
		ids, err := s.directInvitees(ctx, callerID, in.ConversationID)
		if err != nil {
			return nil, err
		}
		inviteeIDs = ids
	} else {
		ids, err := s.groupInvitees(ctx, callerID, in.GroupID)
		if err != nil {
			return nil, err
		}
		inviteeIDs = ids
	}
	if len(inviteeIDs) == 0 {
		return nil, validationError("conversation_id", "INVALID_VALUE", "通話須至少有一名受邀者。")
	}
	active, err := s.store.FindActiveCall(ctx, in.ConversationID, in.GroupID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, validationError("conversation_id", "INVALID_VALUE", "同一範圍內已有進行中的通話。")
	}
	now := s.now().UTC()
	c := &Call{
		ID:             ids.New(),
		ConversationID: in.ConversationID,
		GroupID:        in.GroupID,
		InitiatorID:    callerID,
		Type:           in.Type,
		Status:         StatusRinging,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.store.CreateCall(ctx, c); err != nil {
		return nil, err
	}
	initiator := &Participant{
		ID:        ids.New(),
		CallID:    c.ID,
		UserID:    callerID,
		JoinedAt:  &now,
		Result:    ResultJoined,
		CreatedAt: now,
	}
	if err := s.store.CreateParticipant(ctx, initiator); err != nil {
		return nil, err
	}
	for _, inviteeID := range inviteeIDs {
		p := &Participant{
			ID:        ids.New(),
			CallID:    c.ID,
			UserID:    inviteeID,
			Result:    ResultPending,
			CreatedAt: now,
		}
		if err := s.store.CreateParticipant(ctx, p); err != nil {
			return nil, err
		}
	}
	if err := s.store.CreateEvent(ctx, &CallEvent{
		ID:         ids.New(),
		CallID:     c.ID,
		UserID:     callerID,
		Type:       EventInitiated,
		Payload:    eventPayload("", StatusRinging, ""),
		OccurredAt: now,
		CreatedAt:  now,
	}); err != nil {
		return nil, err
	}
	return &CallEntry{Call: c, ParticipantCount: len(inviteeIDs) + 1}, nil
}

// directInvitees 解析一對一通話的受邀者：呼叫者須為生效成員、
// 對話須為直接對話、另一成員恰好一名、任一方向封鎖一律 404。
func (s *Service) directInvitees(ctx context.Context, callerID, conversationID string) ([]string, error) {
	if s.chats == nil {
		return nil, ErrCallNotFound
	}
	member, err := s.chats.IsActiveMember(ctx, conversationID, callerID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrCallNotFound
	}
	direct, err := s.chats.IsDirect(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if !direct {
		return nil, validationError("conversation_id", "INVALID_VALUE", "群組對話請以 group_id 發起群組通話。")
	}
	blocked, err := s.chats.DirectBlocked(ctx, conversationID, callerID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrCallNotFound
	}
	members, err := s.chats.ListMembers(ctx, callerID, conversationID)
	if err != nil {
		return nil, err
	}
	var others []string
	for _, m := range members {
		if m.UserID == callerID || m.LeftAt != nil {
			continue
		}
		others = append(others, m.UserID)
	}
	if len(others) != 1 {
		return nil, ErrCallNotFound
	}
	return others, nil
}

// groupInvitees 解析群組通話的受邀者：呼叫者須為生效成員，受邀者
// 為全部其他生效成員。非成員讀取群組一律 404（Group 服務層語義）。
func (s *Service) groupInvitees(ctx context.Context, callerID, groupID string) ([]string, error) {
	if s.groups == nil {
		return nil, ErrCallNotFound
	}
	members, err := s.groups.ListMembers(ctx, callerID, groupID)
	if err != nil {
		if errors.Is(err, group.ErrGroupNotFound) {
			return nil, ErrCallNotFound
		}
		return nil, err
	}
	var invitees []string
	for _, m := range members {
		if m.UserID == callerID || !m.Active() {
			continue
		}
		invitees = append(invitees, m.UserID)
	}
	return invitees, nil
}

// loadForParticipant 讀取通話並驗證呼叫者為參與者（待回應受邀者
// 亦可讀取來電）；否則一律 404，不得洩漏存在性。
func (s *Service) loadForParticipant(ctx context.Context, userID, callID string) (*Call, *Participant, error) {
	c, err := s.store.GetCall(ctx, callID)
	if err != nil {
		return nil, nil, err
	}
	if c == nil {
		return nil, nil, ErrCallNotFound
	}
	p, err := s.store.GetParticipant(ctx, callID, userID)
	if err != nil {
		return nil, nil, err
	}
	if p == nil {
		return nil, nil, ErrCallNotFound
	}
	return c, p, nil
}

// GetCall 讀取單一通話（參與者限定）。
func (s *Service) GetCall(ctx context.Context, userID, callID string) (*CallEntry, error) {
	c, _, err := s.loadForParticipant(ctx, userID, callID)
	if err != nil {
		return nil, err
	}
	participants, err := s.store.ListParticipants(ctx, callID)
	if err != nil {
		return nil, err
	}
	return &CallEntry{Call: c, ParticipantCount: len(participants)}, nil
}

// ListCalls 列出呼叫者參與的通話（建立時間倒序游標分頁）。
func (s *Service) ListCalls(ctx context.Context, userID, cursor string, limit int) ([]*CallEntry, string, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	if cursor != "" {
		if _, _, err := decodeCursor(cursor); err != nil {
			return nil, "", err
		}
	}
	calls, err := s.store.ListCallsByUser(ctx, userID, cursor, limit)
	if err != nil {
		return nil, "", err
	}
	entries := make([]*CallEntry, 0, len(calls))
	for _, c := range calls {
		participants, err := s.store.ListParticipants(ctx, c.ID)
		if err != nil {
			return nil, "", err
		}
		entries = append(entries, &CallEntry{Call: c, ParticipantCount: len(participants)})
	}
	nextCursor := ""
	if len(calls) == limit {
		last := calls[len(calls)-1]
		nextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	return entries, nextCursor, nil
}

// ListParticipants 列出通話參與者（參與者限定，加入順序）。
func (s *Service) ListParticipants(ctx context.Context, userID, callID string) ([]*Participant, error) {
	if _, _, err := s.loadForParticipant(ctx, userID, callID); err != nil {
		return nil, err
	}
	return s.store.ListParticipants(ctx, callID)
}

// ListEvents 列出通話事件（參與者限定，發生時間升序）。
func (s *Service) ListEvents(ctx context.Context, userID, callID string) ([]*CallEvent, error) {
	if _, _, err := s.loadForParticipant(ctx, userID, callID); err != nil {
		return nil, err
	}
	return s.store.ListEvents(ctx, callID)
}

// Answer 接聽來電：僅待回應的受邀者、僅響鈴中的通話可接聽。
// 首位接聽者使通話進入 active，started_at 記為接聽時間。
func (s *Service) Answer(ctx context.Context, userID, callID string) (*CallEntry, error) {
	c, p, err := s.loadForParticipant(ctx, userID, callID)
	if err != nil {
		return nil, err
	}
	if c.Status != StatusRinging {
		return nil, validationError("status", "INVALID_VALUE", "通話已不在可接聽狀態。")
	}
	if !p.IsPending() {
		return nil, validationError("status", "INVALID_VALUE", "參與者已回應此次來電。")
	}
	if c.InitiatorID == userID {
		return nil, validationError("status", "INVALID_VALUE", "發起人無須接聽自己的來電。")
	}
	now := s.now().UTC()
	from := c.Status
	p.JoinedAt = &now
	p.Result = ResultJoined
	if err := s.store.SaveParticipant(ctx, p); err != nil {
		return nil, err
	}
	if c.StartedAt == nil {
		c.StartedAt = &now
	}
	c.Status = StatusActive
	c.UpdatedAt = now
	if err := s.store.SaveCall(ctx, c); err != nil {
		return nil, err
	}
	if err := s.recordEvent(ctx, c, userID, EventAnswered, from, c.Status, ""); err != nil {
		return nil, err
	}
	return s.entryOf(ctx, c)
}

// Decline 拒接來電：僅待回應的受邀者、僅響鈴中的通話可拒接。
// 全員皆已回應且無人接聽時，通話按回應彙總為 declined（有拒接
// 者）或 busy（全員忙線），並記結束時間。
func (s *Service) Decline(ctx context.Context, userID, callID string) (*CallEntry, error) {
	return s.respond(ctx, userID, callID, ResultDeclined, EventDeclined)
}

// Busy 回報忙線：僅待回應的受邀者、僅響鈴中的通話可回報。
// 彙總規則與拒接相同（見 Decline）。
func (s *Service) Busy(ctx context.Context, userID, callID string) (*CallEntry, error) {
	return s.respond(ctx, userID, callID, ResultBusy, EventBusy)
}

func (s *Service) respond(ctx context.Context, userID, callID string, result ParticipantResult, eventType EventType) (*CallEntry, error) {
	c, p, err := s.loadForParticipant(ctx, userID, callID)
	if err != nil {
		return nil, err
	}
	if c.Status != StatusRinging {
		return nil, validationError("status", "INVALID_VALUE", "通話已不在可回應狀態。")
	}
	if !p.IsPending() {
		return nil, validationError("status", "INVALID_VALUE", "參與者已回應此次來電。")
	}
	if c.InitiatorID == userID {
		return nil, validationError("status", "INVALID_VALUE", "發起人請以結束通話代替回應。")
	}
	now := s.now().UTC()
	from := c.Status
	p.Result = result
	if err := s.store.SaveParticipant(ctx, p); err != nil {
		return nil, err
	}
	if err := s.recordEvent(ctx, c, userID, eventType, from, c.Status, ""); err != nil {
		return nil, err
	}
	participants, err := s.store.ListParticipants(ctx, callID)
	if err != nil {
		return nil, err
	}
	if allInviteesResponded(participants, c.InitiatorID) {
		reason := EndReasonBusy
		status := StatusBusy
		for _, q := range participants {
			if q.UserID != c.InitiatorID && q.Result == ResultDeclined {
				reason = EndReasonDeclined
				status = StatusDeclined
				break
			}
		}
		c.Status = status
		c.EndedAt = &now
		c.EndReason = reason
		c.EndedBy = userID
		c.UpdatedAt = now
		if err := s.store.SaveCall(ctx, c); err != nil {
			return nil, err
		}
		if err := s.recordEvent(ctx, c, userID, EventEnded, from, c.Status, reason); err != nil {
			return nil, err
		}
	}
	return s.entryOf(ctx, c)
}

// allInviteesResponded 回傳是否全部受邀者皆已回應（無待回應者）且
// 無人加入（發起人恆為已加入，不列入判斷）。
func allInviteesResponded(participants []*Participant, initiatorID string) bool {
	for _, p := range participants {
		if p.UserID == initiatorID {
			continue
		}
		if p.IsPending() || p.Result == ResultJoined {
			return false
		}
	}
	return true
}

// End 結束通話：僅發起人或已加入的參與者可結束；響鈴中且無人接聽
// 時記為 missed，否則記為 ended（掛斷）。仍待回應的參與者記為
// missed，已加入者記離開時間。結束為終端狀態，不得重開。
func (s *Service) End(ctx context.Context, userID, callID string) (*CallEntry, error) {
	c, p, err := s.loadForParticipant(ctx, userID, callID)
	if err != nil {
		return nil, err
	}
	if c.Status.IsTerminal() {
		return nil, validationError("status", "INVALID_VALUE", "通話已結束。")
	}
	if p.Result != ResultJoined {
		return nil, ErrCallNotFound
	}
	now := s.now().UTC()
	from := c.Status
	participants, err := s.store.ListParticipants(ctx, callID)
	if err != nil {
		return nil, err
	}
	anyJoined := false
	for _, q := range participants {
		if q.UserID != c.InitiatorID && q.Result == ResultJoined {
			anyJoined = true
			break
		}
	}
	if anyJoined {
		c.Status = StatusEnded
		c.EndReason = EndReasonHangup
	} else {
		c.Status = StatusMissed
		c.EndReason = EndReasonMissed
	}
	c.EndedAt = &now
	c.EndedBy = userID
	c.UpdatedAt = now
	if c.StartedAt != nil {
		d := int64(now.Sub(*c.StartedAt) / time.Second)
		c.Duration = &d
	}
	if err := s.store.SaveCall(ctx, c); err != nil {
		return nil, err
	}
	for _, q := range participants {
		changed := false
		if q.IsPending() {
			q.Result = ResultMissed
			changed = true
		}
		if q.Result == ResultJoined && q.LeftAt == nil {
			q.LeftAt = &now
			changed = true
		}
		if changed {
			if err := s.store.SaveParticipant(ctx, q); err != nil {
				return nil, err
			}
		}
	}
	if err := s.recordEvent(ctx, c, userID, EventEnded, from, c.Status, c.EndReason); err != nil {
		return nil, err
	}
	return s.entryOf(ctx, c)
}

func (s *Service) recordEvent(ctx context.Context, c *Call, userID string, typ EventType, from, to Status, reason EndReason) error {
	now := s.now().UTC()
	return s.store.CreateEvent(ctx, &CallEvent{
		ID:         ids.New(),
		CallID:     c.ID,
		UserID:     userID,
		Type:       typ,
		Payload:    eventPayload(from, to, reason),
		OccurredAt: now,
		CreatedAt:  now,
	})
}

func (s *Service) entryOf(ctx context.Context, c *Call) (*CallEntry, error) {
	participants, err := s.store.ListParticipants(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	return &CallEntry{Call: c, ParticipantCount: len(participants)}, nil
}

// encodeCursorRaw／decodeCursorRaw 為游標的 base64url 編解碼。
func encodeCursorRaw(raw string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursorRaw(cursor string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// encodeCursor 將 (created_at, id) 倒序鍵編為不透明游標（沿
// Timeline 游標格式）。
func encodeCursor(t time.Time, id string) string {
	return encodeCursorRaw(t.UTC().Format(time.RFC3339Nano) + "|" + id)
}

// decodeCursor 解析游標：格式錯誤回欄位錯誤。
func decodeCursor(cursor string) (time.Time, string, error) {
	raw, err := decodeCursorRaw(cursor)
	if err != nil {
		return time.Time{}, "", validationError("cursor", "INVALID_VALUE", "游標格式錯誤。")
	}
	parts := strings.SplitN(raw, "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", validationError("cursor", "INVALID_VALUE", "游標格式錯誤。")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", validationError("cursor", "INVALID_VALUE", "游標格式錯誤。")
	}
	if parts[1] == "" {
		return time.Time{}, "", validationError("cursor", "INVALID_VALUE", "游標格式錯誤。")
	}
	return t, parts[1], nil
}

// MemoryStore 是 Store 的記憶體實作（開發與測試用）。單一互斥鎖
// 保護三表，寫入路徑同一鎖內完成（PG 實作分步寫入，見 postgres.go
// 已知限制）。
type MemoryStore struct {
	mu           sync.RWMutex
	calls        map[string]*Call
	participants map[string]map[string]*Participant // callID -> userID -> Participant
	events       map[string][]*CallEvent            // callID -> events（發生時間升序）
}

// NewMemoryStore 建立空的記憶體通話儲存。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		calls:        map[string]*Call{},
		participants: map[string]map[string]*Participant{},
		events:       map[string][]*CallEvent{},
	}
}

func cloneCall(c *Call) *Call {
	if c == nil {
		return nil
	}
	cp := *c
	if c.StartedAt != nil {
		t := *c.StartedAt
		cp.StartedAt = &t
	}
	if c.EndedAt != nil {
		t := *c.EndedAt
		cp.EndedAt = &t
	}
	if c.Duration != nil {
		d := *c.Duration
		cp.Duration = &d
	}
	return &cp
}

func cloneParticipant(p *Participant) *Participant {
	if p == nil {
		return nil
	}
	cp := *p
	if p.JoinedAt != nil {
		t := *p.JoinedAt
		cp.JoinedAt = &t
	}
	if p.LeftAt != nil {
		t := *p.LeftAt
		cp.LeftAt = &t
	}
	return &cp
}

func cloneEvent(e *CallEvent) *CallEvent {
	if e == nil {
		return nil
	}
	cp := *e
	return &cp
}

func (s *MemoryStore) GetCall(_ context.Context, id string) (*Call, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneCall(s.calls[id]), nil
}

func (s *MemoryStore) CreateCall(_ context.Context, c *Call) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.calls[c.ID]; exists {
		return validationError("call_id", "DUPLICATE", "通話已存在。")
	}
	s.calls[c.ID] = cloneCall(c)
	return nil
}

func (s *MemoryStore) SaveCall(_ context.Context, c *Call) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.calls[c.ID]; !exists {
		return ErrCallNotFound
	}
	s.calls[c.ID] = cloneCall(c)
	return nil
}

func (s *MemoryStore) FindActiveCall(_ context.Context, conversationID, groupID string) (*Call, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.calls {
		if c.Status != StatusRinging && c.Status != StatusActive {
			continue
		}
		if conversationID != "" && c.ConversationID == conversationID {
			return cloneCall(c), nil
		}
		if groupID != "" && c.GroupID == groupID {
			return cloneCall(c), nil
		}
	}
	return nil, nil
}

func (s *MemoryStore) ListCallsByUser(_ context.Context, userID, cursor string, limit int) ([]*Call, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var cursorTime time.Time
	var cursorID string
	if cursor != "" {
		t, id, err := decodeCursor(cursor)
		if err != nil {
			return nil, err
		}
		cursorTime, cursorID = t, id
	}
	var out []*Call
	for _, c := range s.calls {
		if _, ok := s.participants[c.ID][userID]; !ok {
			continue
		}
		if cursor != "" {
			if c.CreatedAt.After(cursorTime) {
				continue
			}
			if c.CreatedAt.Equal(cursorTime) && c.ID >= cursorID {
				continue
			}
		}
		out = append(out, cloneCall(c))
	}
	sortCallsDesc(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func sortCallsDesc(calls []*Call) {
	for i := 1; i < len(calls); i++ {
		for j := i; j > 0; j-- {
			a, b := calls[j-1], calls[j]
			if a.CreatedAt.After(b.CreatedAt) || (a.CreatedAt.Equal(b.CreatedAt) && a.ID > b.ID) {
				break
			}
			calls[j-1], calls[j] = b, a
		}
	}
}

func (s *MemoryStore) GetParticipant(_ context.Context, callID, userID string) (*Participant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneParticipant(s.participants[callID][userID]), nil
}

func (s *MemoryStore) ListParticipants(_ context.Context, callID string) ([]*Participant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Participant
	for _, p := range s.participants[callID] {
		out = append(out, cloneParticipant(p))
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0; j-- {
			if !out[j].CreatedAt.Before(out[j-1].CreatedAt) {
				break
			}
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out, nil
}

func (s *MemoryStore) CreateParticipant(_ context.Context, p *Participant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.calls[p.CallID]; !exists {
		return ErrCallNotFound
	}
	m := s.participants[p.CallID]
	if m == nil {
		m = map[string]*Participant{}
		s.participants[p.CallID] = m
	}
	if _, exists := m[p.UserID]; exists {
		return validationError("user_id", "DUPLICATE", "參與者已存在。")
	}
	m[p.UserID] = cloneParticipant(p)
	return nil
}

func (s *MemoryStore) SaveParticipant(_ context.Context, p *Participant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.participants[p.CallID]
	if m == nil || m[p.UserID] == nil {
		return ErrCallNotFound
	}
	m[p.UserID] = cloneParticipant(p)
	return nil
}

func (s *MemoryStore) CreateEvent(_ context.Context, e *CallEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.calls[e.CallID]; !exists {
		return ErrCallNotFound
	}
	if utf8.RuneCountInString(e.Payload) > 2000 {
		return validationError("event_payload", "TOO_LONG", "事件內容過長。")
	}
	s.events[e.CallID] = append(s.events[e.CallID], cloneEvent(e))
	return nil
}

func (s *MemoryStore) ListEvents(_ context.Context, callID string) ([]*CallEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*CallEvent
	for _, e := range s.events[callID] {
		out = append(out, cloneEvent(e))
	}
	return out, nil
}
