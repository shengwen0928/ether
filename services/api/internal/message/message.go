package message

import (
	"context"
	"errors"
	"sync"
	"time"
	"unicode/utf8"

	"ether.local/api/internal/ids"
	"ether.local/api/internal/outbox"
	"ether.local/api/internal/transport"
)

// ErrMessageNotFound 表示查無訊息，或呼叫者不是該對話的生效成員，
// 或呼叫者不是訊息作者（編輯／收回／刪除路徑）。三種情形對外
// 一律 404，不得區分「不存在」與「無權存取」。
var ErrMessageNotFound = errors.New("message: message not found")

// Status 是伺服器端訊息狀態。值域為 Phase 05 實作詞彙
// sent／recalled（ERD §6.3.4 稱值域待定稿，含客戶端狀態的
// 七態對照待定稿）；本里程碑只存伺服器端狀態。
type Status string

const (
	// StatusSent 是已寫入、可被成員讀取的訊息。
	StatusSent Status = "sent"
	// StatusRecalled 是已被作者收回的訊息：列保留，內容不再回傳。
	StatusRecalled Status = "recalled"
)

// Kind 是訊息類型。值域為 Phase 05 實作詞彙：本里程碑僅 text
// （ERD §6.3.4 稱值域待定稿；圖片／語音／檔案／貼圖／系統
// 屬後續里程碑）。
type Kind string

const (
	// KindText 是純文字訊息。content 為不透明字串（見 doc.go
	// E2EE 邊界：不得以此宣稱加密）。
	KindText Kind = "text"
)

// MaxContentRunes 是訊息內容長度上限（Phase 05 實作預設，待端點
// Schema 定稿覆核）。空訊息一律拒絕。
const MaxContentRunes = 4000

// MaxClientMessageIDLength 是去重鍵長度上限（Phase 05 實作預設）。
const MaxClientMessageIDLength = 128

// Message 是 messages 的一列。Content 在收回後不得回傳（回應層
// 以空字串取代）；OriginalContent 只保留於儲存層作稽核，不得
// 經 API 回傳。Attachments 僅於寫入時攜帶待關聯的附件列（發送／
// 轉傳路徑由服務層填入，儲存層寫入後不保留於本結構；讀取路徑
// 以 ListAttachments 另查，不得以本欄位假裝附件已載入）。
type Message struct {
	ID                 string
	ConversationID     string
	SenderID           string
	Kind               Kind
	Content            string
	Status             Status
	ClientMessageID    string
	SequenceNumber     int64
	ReplyToMessageID   string
	Attachments        []*Attachment
	EditedAt           *time.Time
	OriginalContent    string
	RecalledAt         *time.Time
	DeletedForSenderAt *time.Time
	EditCount          int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ReceiptType 是回條類型。值域為 Phase 05 實作詞彙
// delivered／read。
type ReceiptType string

const (
	// ReceiptDelivered 是送達回條：訊息已投遞至接收者裝置，
	// 不表示已讀。
	ReceiptDelivered ReceiptType = "delivered"
	// ReceiptRead 是已讀回條：同時推進成員的已讀水位。
	ReceiptRead ReceiptType = "read"
)

// Receipt 是 message_receipts 的一列。
type Receipt struct {
	ID        string
	MessageID string
	UserID    string
	DeviceID  string
	Type      ReceiptType
	CreatedAt time.Time
}

// ValidationError 承載欄位驗證失敗清單。
type ValidationError struct {
	Fields []transport.FieldError
}

func (e *ValidationError) Error() string { return "message: validation failed" }

// FieldErrors 供 transport.WriteServiceError 寫出 details。
func (e *ValidationError) FieldErrors() []transport.FieldError { return e.Fields }

func validationError(field, issue, message string) *ValidationError {
	return &ValidationError{Fields: []transport.FieldError{{Field: field, Issue: issue, Message: message}}}
}

// ConversationAccess 是 Message 對 Chat 模組的最小依賴（依賴 DAG：
// Message --> Chat）：成員驗證、對話類型、封鎖檢查、最後訊息
// 快取更新、已讀水位。不得直連 conversations 表。
type ConversationAccess interface {
	// IsActiveMember 回傳使用者是否為對話的生效成員。
	IsActiveMember(ctx context.Context, conversationID, userID string) (bool, error)
	// IsDirect 回傳對話是否為一對一對話（不存在時回 false, nil）。
	IsDirect(ctx context.Context, conversationID string) (bool, error)
	// DirectBlocked 回傳一對一對話中另一成員與呼叫者間是否存在
	// 任一方向的生效封鎖（群組對話回 false）。
	DirectBlocked(ctx context.Context, conversationID, userID string) (bool, error)
	// NoteMessage 在訊息持久化成功後更新對話的最後訊息快取。
	NoteMessage(ctx context.Context, conversationID, messageID string, at time.Time) error
	// MarkRead 推進已讀水位（序號不得倒退）。
	MarkRead(ctx context.Context, conversationID, userID, messageID string, seq int64, at time.Time) error
	// ReadSequence 回傳成員的已讀序號水位與成員身份。
	ReadSequence(ctx context.Context, conversationID, userID string) (int64, bool, error)
}

// Store 是 Message 三張表的儲存介面。
type Store interface {
	GetMessage(ctx context.Context, id string) (*Message, error)
	GetByClientMessageID(ctx context.Context, conversationID, senderID, clientMessageID string) (*Message, error)
	// CreateMessage 寫入訊息並分配每對話序號（單調遞增，不跳號、
	// 不重用；併發衝突時儲存層以重試收斂）。ReplyToMessageID
	// 非空時，儲存層同時寫入 message_replies 列。
	CreateMessage(ctx context.Context, m *Message) error
	SaveMessage(ctx context.Context, m *Message) error
	// ListBefore 回傳序號小於 beforeSeq 的訊息，依序號倒序，
	// 至多 limit 筆（游標分頁：讀取較舊歷史）。
	ListBefore(ctx context.Context, conversationID string, beforeSeq int64, limit int) ([]*Message, error)
	// ListSince 回傳序號大於 sinceSeq 的訊息，依序號順序，
	// 至多 limit 筆（重連續拉：補齊離線期間的訊息）。
	ListSince(ctx context.Context, conversationID string, sinceSeq int64, limit int) ([]*Message, error)
	// CountAfter 回傳序號大於 afterSeq、且非 userID 所發的
	// 訊息數（未讀數計算用；收回訊息不計入）。
	CountAfter(ctx context.Context, conversationID, userID string, afterSeq int64) (int, error)
	GetReplyTo(ctx context.Context, messageID string) (string, error)

	CreateReceipt(ctx context.Context, r *Receipt) error
	GetReceipt(ctx context.Context, messageID, userID string, typ ReceiptType) (*Receipt, error)
	ListReceipts(ctx context.Context, messageID string) ([]*Receipt, error)

	// SetReaction 寫入或更換使用者對訊息的反應（每人每訊息至多
	// 一則生效反應；更換時舊列標記取消），回傳寫入後的生效列。
	SetReaction(ctx context.Context, r *Reaction) (*Reaction, error)
	// RemoveReaction 取消使用者對訊息的全部生效反應（冪等）。
	RemoveReaction(ctx context.Context, messageID, userID string) error
	// ListActiveReactions 列出生效中的反應列（未取消者）。
	ListActiveReactions(ctx context.Context, messageID string) ([]*Reaction, error)
	// CountActivePins 回傳對話的生效置頂數。
	CountActivePins(ctx context.Context, conversationID string) (int, error)
	// GetActivePin 讀取對話中某訊息的生效置頂列；未置頂時回傳 (nil, nil)。
	GetActivePin(ctx context.Context, conversationID, messageID string) (*Pin, error)
	// PinMessage 寫入置頂列；已置頂時回傳既有列與 isNew=false，
	// 不重複寫入（曾取消後再置頂寫入新列，保留歷史）。
	PinMessage(ctx context.Context, p *Pin) (*Pin, bool, error)
	// UnpinMessage 取消置頂（冪等）：回傳被取消的列；已取消時
	// 回傳最近一筆歷史列；從未置頂時回傳 (nil, nil)。
	UnpinMessage(ctx context.Context, conversationID, messageID string, at time.Time) (*Pin, error)
	// ListActivePins 列出生效中的置頂列（依置頂時間順序）。
	ListActivePins(ctx context.Context, conversationID string) ([]*Pin, error)

	// CreateAttachments 寫入訊息的附件關聯列（同一檔案重複關聯
	// 不重複寫入）。
	CreateAttachments(ctx context.Context, messageID string, atts []*Attachment) error
	// ListAttachments 列出訊息的附件關聯列（依建立時間順序）。
	ListAttachments(ctx context.Context, messageID string) ([]*Attachment, error)
	// GetAttachment 讀取訊息對某檔案的附件列；不存在時回傳 (nil, nil)。
	GetAttachment(ctx context.Context, messageID, mediaFileID string) (*Attachment, error)
	// CreateForward 寫入轉傳紀錄（同一新訊息重複寫入時保留首次列）。
	CreateForward(ctx context.Context, f *Forward) error
	// GetForwardByForwardedID 讀取某新訊息的轉傳紀錄；該訊息並非
	// 轉傳產物時回傳 (nil, nil)。
	GetForwardByForwardedID(ctx context.Context, forwardedMessageID string) (*Forward, error)
}

// Service 是 Message 領域服務。
type Service struct {
	store  Store
	chats  ConversationAccess
	outbox outbox.Store
	media  MediaAccess // 可為空：未接線時附件關聯 fail-closed。
	now    func() time.Time
}

// NewService 建立 Service。now 為空時用 time.Now。
func NewService(store Store, chats ConversationAccess, now func() time.Time) *Service {
	return NewServiceWithOutbox(store, chats, nil, now)
}

// NewServiceWithOutbox 建立帶 Outbox 寫入的 Service：outbox 非空時，
// Send／Recall 在訊息列寫入的同一交易寫入推播事件（儲存層須實作
// AtomicWriter，否則寫入 fail-closed）。outbox 為空時不產生事件，
// 行為與 NewService 相同（既有測試與未接線的部署）。
func NewServiceWithOutbox(store Store, chats ConversationAccess, outboxStore outbox.Store, now func() time.Time) *Service {
	return NewServiceWithDeps(store, chats, outboxStore, nil, now)
}

// NewServiceWithDeps 建立帶 Outbox 與 Media 接線的 Service：media
// 非空時，發送／新增附件經 Media 模組驗證檔案可關聯性；為空時
// 附件關聯 fail-closed（回 ErrMediaUnavailable），無附件的既有
// 路徑行為不變。
func NewServiceWithDeps(store Store, chats ConversationAccess, outboxStore outbox.Store, mediaAccess MediaAccess, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, chats: chats, outbox: outboxStore, media: mediaAccess, now: now}
}

// errOutboxUnsupported 表示 Outbox 已接線但儲存層不支援原子寫入
// （接線錯誤，fail-closed，不得以非原子寫入假裝事件不丟失）。
var errOutboxUnsupported = errors.New("message: 儲存層不支援 Outbox 原子寫入")

// createMessage 寫入新訊息：Outbox 已接線時，訊息列與
// message.received 事件同一交易寫入；未接線時只寫訊息列。
// Outbox 已接線但儲存層不支援原子寫入時 fail-closed（不得以
// 非原子寫入假裝事件不丟失）。
func (s *Service) createMessage(ctx context.Context, m *Message) error {
	if s.outbox == nil {
		return s.store.CreateMessage(ctx, m)
	}
	aw, ok := s.store.(AtomicWriter)
	if !ok {
		return errOutboxUnsupported
	}
	return aw.CreateMessageWithEvent(ctx, m, func(written *Message) (*outbox.Event, error) {
		return newReceivedEvent(written)
	})
}

// saveMessage 更新訊息列：收回路徑以同一交易寫入
// message.recalled 事件；編輯與僅自己刪除不產生事件（見 events.go）。
func (s *Service) saveMessage(ctx context.Context, m *Message) error {
	if s.outbox == nil {
		return s.store.SaveMessage(ctx, m)
	}
	aw, ok := s.store.(AtomicWriter)
	if !ok {
		return errOutboxUnsupported
	}
	return aw.SaveMessageWithEvent(ctx, m, func(written *Message) (*outbox.Event, error) {
		return newRecalledEvent(written)
	})
}

// SendResult 是 Send 的回傳：IsNew=false 表示冪等命中，回傳
// 既有訊息（不得新建、不得重發 NoteMessage）。
type SendResult struct {
	Message *Message
	IsNew   bool
}

func validateContent(content string) error {
	n := utf8.RuneCountInString(content)
	if n == 0 {
		return validationError("content", "REQUIRED", "訊息內容不可為空。")
	}
	if n > MaxContentRunes {
		return validationError("content", "TOO_LONG", "訊息內容過長。")
	}
	return nil
}

// Send 送出訊息。驗證順序：成員身份 → 類型／內容 → 去重鍵 →
// 引用 → DM 封鎖檢查 → 寫入 → NoteMessage。去重命中時回傳既有
// 訊息，不重寫、不重發快取更新。
func (s *Service) Send(ctx context.Context, senderID, conversationID, kind, content, clientMessageID, replyToID string) (*SendResult, error) {
	return s.sendInternal(ctx, senderID, conversationID, kind, content, clientMessageID, replyToID, nil)
}

// sendInternal 是 Send 與 SendWithAttachments 的共用寫入路徑：
// attachmentFileIDs 非空時，於封鎖檢查之後、寫入之前解析附件
// （見 resolveAttachments），解析出的附件列掛於訊息結構，由儲存
// 層與訊息列同一寫入路徑提交。
func (s *Service) sendInternal(ctx context.Context, senderID, conversationID, kind, content, clientMessageID, replyToID string, attachmentFileIDs []string) (*SendResult, error) {
	member, err := s.chats.IsActiveMember(ctx, conversationID, senderID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrMessageNotFound
	}
	if Kind(kind) != KindText {
		return nil, validationError("kind", "INVALID_VALUE", "本里程碑僅支援文字訊息。")
	}
	if err := validateContent(content); err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(clientMessageID) > MaxClientMessageIDLength {
		return nil, validationError("client_message_id", "TOO_LONG", "去重鍵過長。")
	}
	if clientMessageID != "" {
		existing, err := s.store.GetByClientMessageID(ctx, conversationID, senderID, clientMessageID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return &SendResult{Message: existing}, nil
		}
	}
	if replyToID != "" {
		replied, err := s.store.GetMessage(ctx, replyToID)
		if err != nil {
			return nil, err
		}
		if replied == nil || replied.ConversationID != conversationID {
			return nil, ErrMessageNotFound
		}
	}
	blocked, err := s.chats.DirectBlocked(ctx, conversationID, senderID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrMessageNotFound
	}
	now := s.now().UTC()
	attachments, err := s.resolveAttachments(ctx, senderID, attachmentFileIDs, now)
	if err != nil {
		return nil, err
	}
	m := &Message{
		ID:               ids.New(),
		ConversationID:   conversationID,
		SenderID:         senderID,
		Kind:             KindText,
		Content:          content,
		Status:           StatusSent,
		ClientMessageID:  clientMessageID,
		ReplyToMessageID: replyToID,
		Attachments:      attachments,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if m.ID == "" {
		return nil, errors.New("message: 識別碼產生失敗")
	}
	if err := s.createMessage(ctx, m); err != nil {
		if errors.Is(err, ErrDuplicateClientMessage) && clientMessageID != "" {
			// 併發重送命中去重鍵：重讀既有列，以冪等命中回傳。
			existing, readErr := s.store.GetByClientMessageID(ctx, conversationID, senderID, clientMessageID)
			if readErr != nil {
				return nil, readErr
			}
			if existing != nil {
				return &SendResult{Message: existing}, nil
			}
		}
		return nil, err
	}
	if err := s.chats.NoteMessage(ctx, conversationID, m.ID, now); err != nil {
		return nil, err
	}
	return &SendResult{Message: m, IsNew: true}, nil
}

// ListPage 是游標分頁的回傳：NextCursor 為下一頁的游標
// （before 模式為最小序號、since 模式為最大序號；無更多資料時為空）。
type ListPage struct {
	Messages   []*Message
	NextCursor string
}

// List 列出對話訊息：beforeSeq 與 sinceSeq 二選一，不得並存。
// limit 為空時預設 30、上限 100（Phase 05 實作預設，待定稿）。
// 非成員一律 404。
func (s *Service) List(ctx context.Context, userID, conversationID string, beforeSeq, sinceSeq *int64, limit int) (*ListPage, error) {
	member, err := s.chats.IsActiveMember(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrMessageNotFound
	}
	if beforeSeq != nil && sinceSeq != nil {
		return nil, validationError("cursor", "INVALID_VALUE", "before_seq 與 since_seq 不得並存。")
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	var messages []*Message
	var cursor string
	switch {
	case sinceSeq != nil:
		messages, err = s.store.ListSince(ctx, conversationID, *sinceSeq, limit)
		if err != nil {
			return nil, err
		}
		if len(messages) > 0 {
			cursor = itoa(messages[len(messages)-1].SequenceNumber)
		}
	case beforeSeq != nil:
		messages, err = s.store.ListBefore(ctx, conversationID, *beforeSeq, limit)
		if err != nil {
			return nil, err
		}
		if len(messages) > 0 {
			cursor = itoa(messages[len(messages)-1].SequenceNumber)
		}
	default:
		// 無游標：讀取最新歷史（等價於 beforeSeq = +∞，以
		// 最大 int64 實作，儲存層以 < 比較）。
		messages, err = s.store.ListBefore(ctx, conversationID, int64(^uint64(0)>>1), limit)
		if err != nil {
			return nil, err
		}
		if len(messages) > 0 {
			cursor = itoa(messages[len(messages)-1].SequenceNumber)
		}
	}
	return &ListPage{Messages: messages, NextCursor: cursor}, nil
}

// Get 讀取單一訊息（供回條／編輯等路徑複用）：非成員一律 404。
func (s *Service) Get(ctx context.Context, userID, messageID string) (*Message, error) {
	m, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrMessageNotFound
	}
	member, err := s.chats.IsActiveMember(ctx, m.ConversationID, userID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrMessageNotFound
	}
	return m, nil
}

// Edit 編輯訊息：僅作者可編輯；收回或已刪除（僅自己檢視）的
// 訊息不可編輯。首次編輯保留原值（OriginalContent），編輯次數
// 遞增、edited_at 更新。編輯不寫 Outbox 事件（§21 無對應事件名，
// 見 events.go）；客戶端以 REST 歷史同步得知變更。
func (s *Service) Edit(ctx context.Context, userID, messageID, content string) (*Message, error) {
	m, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.SenderID != userID {
		return nil, ErrMessageNotFound
	}
	member, err := s.chats.IsActiveMember(ctx, m.ConversationID, userID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrMessageNotFound
	}
	if m.Status == StatusRecalled || m.DeletedForSenderAt != nil {
		return nil, validationError("message_id", "INVALID_VALUE", "此訊息不可編輯。")
	}
	if err := validateContent(content); err != nil {
		return nil, err
	}
	if m.EditCount == 0 {
		m.OriginalContent = m.Content
	}
	m.Content = content
	m.EditCount++
	now := s.now().UTC()
	m.EditedAt = &now
	m.UpdatedAt = now
	if err := s.store.SaveMessage(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Recall 收回訊息：僅作者可收回；列保留、內容不再回傳
// （回應層以空字串取代）。重複收回為冪等成功。
func (s *Service) Recall(ctx context.Context, userID, messageID string) (*Message, error) {
	m, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.SenderID != userID {
		return nil, ErrMessageNotFound
	}
	member, err := s.chats.IsActiveMember(ctx, m.ConversationID, userID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrMessageNotFound
	}
	if m.Status == StatusRecalled {
		return m, nil
	}
	now := s.now().UTC()
	m.Status = StatusRecalled
	m.RecalledAt = &now
	m.UpdatedAt = now
	if err := s.saveMessage(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// DeleteForSelf 刪除僅自己檢視的訊息：僅作者可操作；列保留，
// 作者的檢視中內容不再回傳，他人檢視不受影響。重複操作冪等成功。
// 僅自己刪除不寫 Outbox 事件（§21 無對應事件名，見 events.go）。
func (s *Service) DeleteForSelf(ctx context.Context, userID, messageID string) (*Message, error) {
	m, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.SenderID != userID {
		return nil, ErrMessageNotFound
	}
	member, err := s.chats.IsActiveMember(ctx, m.ConversationID, userID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrMessageNotFound
	}
	if m.DeletedForSenderAt != nil {
		return m, nil
	}
	now := s.now().UTC()
	m.DeletedForSenderAt = &now
	m.UpdatedAt = now
	if err := s.store.SaveMessage(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// PostReceipt 寫入回條：僅成員可寫、不得為自己的訊息寫回條、
// 型別須為 delivered／read。同一回條重複寫入為冪等成功（回傳
// 既有列）。已讀回條同時推進已讀水位（序號不得倒退）。
func (s *Service) PostReceipt(ctx context.Context, userID, messageID string, typ ReceiptType, deviceID string) (*Receipt, error) {
	if typ != ReceiptDelivered && typ != ReceiptRead {
		return nil, validationError("receipt_type", "INVALID_VALUE", "receipt_type 須為 delivered 或 read。")
	}
	m, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrMessageNotFound
	}
	if m.SenderID == userID {
		return nil, validationError("message_id", "INVALID_VALUE", "不得為自己的訊息寫回條。")
	}
	member, err := s.chats.IsActiveMember(ctx, m.ConversationID, userID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrMessageNotFound
	}
	existing, err := s.store.GetReceipt(ctx, messageID, userID, typ)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	now := s.now().UTC()
	r := &Receipt{
		ID:        ids.New(),
		MessageID: messageID,
		UserID:    userID,
		DeviceID:  deviceID,
		Type:      typ,
		CreatedAt: now,
	}
	if r.ID == "" {
		return nil, errors.New("message: 識別碼產生失敗")
	}
	if err := s.store.CreateReceipt(ctx, r); err != nil {
		return nil, err
	}
	if typ == ReceiptRead {
		if err := s.chats.MarkRead(ctx, m.ConversationID, userID, messageID, m.SequenceNumber, now); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// ListReceipts 列出訊息的回條：僅成員可讀。
func (s *Service) ListReceipts(ctx context.Context, userID, messageID string) ([]*Receipt, error) {
	if _, err := s.Get(ctx, userID, messageID); err != nil {
		return nil, err
	}
	return s.store.ListReceipts(ctx, messageID)
}

// UnreadCount 回傳使用者的未讀數（序號大於已讀水位、非自己
// 所發、未收回的訊息數）。非成員回 0（不得洩漏存在性）。
func (s *Service) UnreadCount(ctx context.Context, conversationID, userID string) (int, error) {
	seq, member, err := s.chats.ReadSequence(ctx, conversationID, userID)
	if err != nil {
		return 0, err
	}
	if !member {
		return 0, nil
	}
	return s.store.CountAfter(ctx, conversationID, userID, seq)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// MemoryStore 是 Store 的記憶體實作（開發與測試用）。
type MemoryStore struct {
	mu        sync.RWMutex
	messages  map[string]*Message
	byClient  map[string]string // conversationID + "\x00" + senderID + "\x00" + clientMessageID -> messageID
	replies   map[string]string // messageID -> repliedToMessageID
	receipts  map[string]*Receipt
	seq       map[string]int64 // conversationID -> 下一序號
	outbox    *outbox.MemoryStore
	reactions map[string]*Reaction // messageID + "\x00" + userID + "\x00" + emoji -> 反應列（含已取消）
	pins      map[string]*Pin      // pin ID -> 置頂列（含已取消）
	pinActive map[string]string    // conversationID + "\x00" + messageID -> 生效置頂列 ID
	// attachments 以 messageID + "\x00" + mediaFileID 為鍵；forwards
	// 以轉傳產生的新訊息識別碼為鍵。
	attachments map[string]*Attachment
	forwards    map[string]*Forward
}

// NewMemoryStore 建立空的記憶體訊息儲存。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		messages:    map[string]*Message{},
		byClient:    map[string]string{},
		replies:     map[string]string{},
		receipts:    map[string]*Receipt{},
		seq:         map[string]int64{},
		reactions:   map[string]*Reaction{},
		pins:        map[string]*Pin{},
		pinActive:   map[string]string{},
		attachments: map[string]*Attachment{},
		forwards:    map[string]*Forward{},
	}
}

// NewMemoryStoreWithOutbox 建立綁定 Outbox 儲存的記憶體訊息儲存：
// 原子寫入時，訊息列與事件列在同一互斥鎖內完成，事件寫入失敗
// 時補償刪除訊息列（行程內語義等價於同一交易）。
func NewMemoryStoreWithOutbox(obs *outbox.MemoryStore) *MemoryStore {
	s := NewMemoryStore()
	s.outbox = obs
	return s
}

var _ AtomicWriter = (*MemoryStore)(nil)

func cloneMessage(m *Message) *Message {
	if m == nil {
		return nil
	}
	cp := *m
	// Attachments 為寫入時攜帶的暫存列，讀取路徑一律以
	// ListAttachments 另查，不得由讀回的訊息結構假裝已載入
	// （PG 儲存層讀取時本欄位亦為空，兩種儲存行為須一致）。
	cp.Attachments = nil
	return &cp
}

func (s *MemoryStore) GetMessage(_ context.Context, id string) (*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := cloneMessage(s.messages[id])
	if m == nil {
		return nil, nil
	}
	m.ReplyToMessageID = s.replies[id]
	return m, nil
}

func (s *MemoryStore) GetByClientMessageID(_ context.Context, conversationID, senderID, clientMessageID string) (*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id := s.byClient[conversationID+"\x00"+senderID+"\x00"+clientMessageID]
	if id == "" {
		return nil, nil
	}
	m := cloneMessage(s.messages[id])
	if m == nil {
		return nil, nil
	}
	m.ReplyToMessageID = s.replies[id]
	return m, nil
}

func (s *MemoryStore) CreateMessage(_ context.Context, m *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createLocked(m)
}

// createLocked 寫入訊息並分配序號，同時寫入引用關聯與附件關聯
// （呼叫方須持有 s.mu 寫鎖）。
func (s *MemoryStore) createLocked(m *Message) error {
	if _, exists := s.messages[m.ID]; exists {
		return errors.New("message: message already exists")
	}
	s.seq[m.ConversationID]++
	m.SequenceNumber = s.seq[m.ConversationID]
	if m.ClientMessageID != "" {
		key := m.ConversationID + "\x00" + m.SenderID + "\x00" + m.ClientMessageID
		if _, exists := s.byClient[key]; exists {
			return errors.New("message: client_message_id already exists")
		}
		s.byClient[key] = m.ID
	}
	if m.ReplyToMessageID != "" {
		s.replies[m.ID] = m.ReplyToMessageID
	}
	for _, att := range m.Attachments {
		cp := cloneAttachment(att)
		cp.MessageID = m.ID
		s.attachments[attachmentKey(m.ID, att.MediaFileID)] = cp
	}
	s.messages[m.ID] = cloneMessage(m)
	return nil
}

// CreateMessageWithEvent 原子寫入訊息列與 Outbox 事件（見
// AtomicWriter）。事件建構或寫入失敗時補償刪除訊息列，不得
// 殘留無事件的訊息列。
func (s *MemoryStore) CreateMessageWithEvent(ctx context.Context, m *Message, build EventBuilder) error {
	if s.outbox == nil {
		return errOutboxUnsupported
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.createLocked(m); err != nil {
		return err
	}
	ev, err := build(m)
	if err != nil {
		s.removeLocked(m)
		return err
	}
	if ev == nil {
		return nil
	}
	if err := s.outbox.Append(ctx, ev); err != nil {
		s.removeLocked(m)
		return err
	}
	return nil
}

// removeLocked 刪除訊息列與關聯索引及附件關聯列（呼叫方須持有
// s.mu 寫鎖，僅供 CreateMessageWithEvent 的失敗補償）。
func (s *MemoryStore) removeLocked(m *Message) {
	delete(s.messages, m.ID)
	if m.ClientMessageID != "" {
		delete(s.byClient, m.ConversationID+"\x00"+m.SenderID+"\x00"+m.ClientMessageID)
	}
	delete(s.replies, m.ID)
	for key, att := range s.attachments {
		if att.MessageID == m.ID {
			delete(s.attachments, key)
		}
	}
	if s.seq[m.ConversationID] == m.SequenceNumber {
		s.seq[m.ConversationID]--
	}
}

func (s *MemoryStore) SaveMessage(_ context.Context, m *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.saveLocked(m)
	return err
}

// saveLocked 更新訊息列（呼叫方須持有 s.mu 寫鎖），回傳更新前
// 的列副本（供原子寫入失敗時還原）。
func (s *MemoryStore) saveLocked(m *Message) (*Message, error) {
	existing, ok := s.messages[m.ID]
	if !ok {
		return nil, ErrMessageNotFound
	}
	prev := cloneMessage(existing)
	m.SequenceNumber = existing.SequenceNumber
	s.messages[m.ID] = cloneMessage(m)
	return prev, nil
}

// SaveMessageWithEvent 原子更新訊息列與 Outbox 事件（見
// AtomicWriter）。事件建構或寫入失敗時還原訊息列。
func (s *MemoryStore) SaveMessageWithEvent(ctx context.Context, m *Message, build EventBuilder) error {
	if s.outbox == nil {
		return errOutboxUnsupported
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	prev, err := s.saveLocked(m)
	if err != nil {
		return err
	}
	ev, err := build(m)
	if err != nil {
		s.messages[m.ID] = prev
		return err
	}
	if ev == nil {
		return nil
	}
	if err := s.outbox.Append(ctx, ev); err != nil {
		s.messages[m.ID] = prev
		return err
	}
	return nil
}

func (s *MemoryStore) ListBefore(_ context.Context, conversationID string, beforeSeq int64, limit int) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Message
	for _, m := range s.messages {
		if m.ConversationID == conversationID && m.SequenceNumber < beforeSeq {
			cp := cloneMessage(m)
			cp.ReplyToMessageID = s.replies[m.ID]
			out = append(out, cp)
		}
	}
	sortDesc(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) ListSince(_ context.Context, conversationID string, sinceSeq int64, limit int) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Message
	for _, m := range s.messages {
		if m.ConversationID == conversationID && m.SequenceNumber > sinceSeq {
			cp := cloneMessage(m)
			cp.ReplyToMessageID = s.replies[m.ID]
			out = append(out, cp)
		}
	}
	sortAsc(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) CountAfter(_ context.Context, conversationID, userID string, afterSeq int64) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, m := range s.messages {
		if m.ConversationID == conversationID && m.SequenceNumber > afterSeq &&
			m.SenderID != userID && m.Status != StatusRecalled {
			n++
		}
	}
	return n, nil
}

func (s *MemoryStore) GetReplyTo(_ context.Context, messageID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.replies[messageID], nil
}

func receiptKey(messageID, userID string, typ ReceiptType) string {
	return messageID + "\x00" + userID + "\x00" + string(typ)
}

func (s *MemoryStore) CreateReceipt(_ context.Context, r *Receipt) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := receiptKey(r.MessageID, r.UserID, r.Type)
	if _, exists := s.receipts[key]; exists {
		return errors.New("message: receipt already exists")
	}
	cp := *r
	s.receipts[key] = &cp
	return nil
}

func (s *MemoryStore) GetReceipt(_ context.Context, messageID, userID string, typ ReceiptType) (*Receipt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := s.receipts[receiptKey(messageID, userID, typ)]
	if r == nil {
		return nil, nil
	}
	cp := *r
	return &cp, nil
}

func (s *MemoryStore) ListReceipts(_ context.Context, messageID string) ([]*Receipt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Receipt
	for _, r := range s.receipts {
		if r.MessageID == messageID {
			cp := *r
			out = append(out, &cp)
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.Before(out[i].CreatedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func sortDesc(ms []*Message) {
	for i := 0; i < len(ms); i++ {
		for j := i + 1; j < len(ms); j++ {
			if ms[j].SequenceNumber > ms[i].SequenceNumber {
				ms[i], ms[j] = ms[j], ms[i]
			}
		}
	}
}

func sortAsc(ms []*Message) {
	for i := 0; i < len(ms); i++ {
		for j := i + 1; j < len(ms); j++ {
			if ms[j].SequenceNumber < ms[i].SequenceNumber {
				ms[i], ms[j] = ms[j], ms[i]
			}
		}
	}
}
