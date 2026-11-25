// 訊息附件關聯與轉傳的領域邏輯。
//
// 邊界：附件關聯與轉傳紀錄屬 Message 模組擁有（MODULE_BOUNDARIES.md
// §5.7；表見 migrations/000011）。Message 只經 MediaAccess 介面
// 向 Media 模組查詢檔案可關聯性（依賴 DAG：Message --> Media）：
// 檔案存在性、擁有權與掃描閘門一律由 Media 模組裁決，本模組不得
// 直讀 media_files 表、不得替 Media 判斷業務權限。附件詮釋資料
// （MIME、大小、檔名）於關聯時自 Media 檔案列快照，寫入後不再
// 追隨檔案列變更（append-only）。
//
// 轉傳不複製附件：轉傳產生的新訊息不帶附件（原訊息的附件不得以
// 同一 media_file_id 逕行連結至目標對話——重新分發須重新授權，
// 參照方式與重新授權流程待定稿，見 STATUS.md 待決定 22）。轉傳
// 寫入以 message.received 事件承載新訊息（WEBSOCKET_PROTOCOL.md
// §21 有此事件名，見 events.go）；轉傳本身無獨立事件名，不得自創。
// 非成員的一切查詢一律 404，不得洩漏存在性（ERROR_CODES.md §6.3）。
package message

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"ether.local/api/internal/ids"
	"ether.local/api/internal/media"
)

// MaxAttachmentsPerMessage 是單則訊息可關聯的附件數上限
// （本里程碑實作預設，待端點 Schema／產品定稿覆核）。
const MaxAttachmentsPerMessage = 10

// MaxForwardDepth 是轉傳鏈深度上限（本里程碑實作預設，待定稿
// 覆核）：原始訊息深度為 1，轉傳一次深度加 1；新訊息深度不得
// 超過本上限。
const MaxForwardDepth = 5

// ErrMediaUnavailable 表示附件關聯未接線 Media 模組（fail-closed：
// 不得以無檢查狀態放行附件寫入）。
var ErrMediaUnavailable = errors.New("message: media access not configured")

// Attachment 是 message_attachments 的一列：訊息與媒體檔案的
// 關聯（詮釋資料快照，append-only，不得修改、不得單獨物理刪除）。
type Attachment struct {
	ID          string
	MessageID   string
	MediaFileID string
	MIMEType    string
	SizeBytes   int64
	FileName    string
	CreatedAt   time.Time
}

// Forward 是 message_forwards 的一列：轉傳紀錄（隨訊息保留，
// 不得物理刪除）。
type Forward struct {
	ID                   string
	SourceMessageID      string
	ForwardedMessageID   string
	SourceConversationID string
	TargetConversationID string
	ForwardedBy          string
	CreatedAt            time.Time
}

// MediaAccess 是 Message 對 Media 模組的最小依賴（依賴 DAG：
// Message --> Media）：僅驗證檔案可被呼叫者關聯為附件。檔案
// 存在、未刪除、為呼叫者所有、掃描已通過四項條件缺一不可，
// 任一不符回傳的錯誤語義由 Media 模組決定（不存在／非擁有者／
// 已刪除為 ErrMediaNotFound；掃描未通過為欄位錯誤）。
type MediaAccess interface {
	CheckAttachable(ctx context.Context, accessorID, fileID string) (*media.MediaFile, error)
}

// ForwardResult 是 Forward 的回傳：IsNew=false 表示冪等命中，
// 回傳既有訊息與其轉傳紀錄（不得新建、不得重發 NoteMessage）。
type ForwardResult struct {
	Message *Message
	Forward *Forward
	IsNew   bool
}

func dedupeFileIDs(fileIDs []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, id := range fileIDs {
		if id == "" {
			return nil, validationError("attachment_file_ids", "INVALID_VALUE", "附件檔案識別碼不可為空。")
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	if len(out) > MaxAttachmentsPerMessage {
		return nil, validationError("attachment_file_ids", "LIMIT_EXCEEDED", "單則訊息的附件數已達上限。")
	}
	return out, nil
}

// resolveAttachments 將檔案識別碼解析為附件列：逐一經 Media
// 模組驗證可關聯性並快照詮釋資料。Media 未接線時 fail-closed；
// 查無檔案／非擁有者／已刪除一律轉為 ErrMessageNotFound（404，
// 不得洩漏檔案存在性）；掃描未通過的欄位錯誤原樣回傳。
func (s *Service) resolveAttachments(ctx context.Context, senderID string, fileIDs []string, now time.Time) ([]*Attachment, error) {
	if len(fileIDs) == 0 {
		return nil, nil
	}
	if s.media == nil {
		return nil, ErrMediaUnavailable
	}
	var out []*Attachment
	for _, fileID := range fileIDs {
		f, err := s.media.CheckAttachable(ctx, senderID, fileID)
		if err != nil {
			if errors.Is(err, media.ErrMediaNotFound) {
				return nil, ErrMessageNotFound
			}
			return nil, err
		}
		id := ids.New()
		if id == "" {
			return nil, errors.New("message: 識別碼產生失敗")
		}
		out = append(out, &Attachment{
			ID:          id,
			MediaFileID: f.ID,
			MIMEType:    f.MIMEType,
			SizeBytes:   f.SizeBytes,
			FileName:    f.OriginalFilename,
			CreatedAt:   now,
		})
	}
	return out, nil
}

// SendWithAttachments 送出帶附件的訊息：驗證順序與 Send 相同，
// 附件解析置於封鎖檢查之後、寫入之前（未通過檢查的請求不得
// 寫入任何列）。附件列與訊息列由儲存層同一寫入路徑提交
// （Outbox 接線時同一交易，見 postgres.go insertMessageRow；
// 非交易路徑的已知限制同 message_replies，見 STATUS.md）。
func (s *Service) SendWithAttachments(ctx context.Context, senderID, conversationID, kind, content, clientMessageID, replyToID string, attachmentFileIDs []string) (*SendResult, error) {
	fileIDs, err := dedupeFileIDs(attachmentFileIDs)
	if err != nil {
		return nil, err
	}
	return s.sendInternal(ctx, senderID, conversationID, kind, content, clientMessageID, replyToID, fileIDs)
}

// loadAttachableMessage 讀取訊息並驗證呼叫者為對話生效成員：
// 查無訊息或非成員一律 404。寫入路徑（新增附件）另套用收回／
// 僅自己刪除守衛（語義同反應／置頂，見 reactions.go）；唯讀路徑
// （列出附件）不套用。
func (s *Service) loadAttachableMessage(ctx context.Context, userID, messageID string, canWrite bool) (*Message, error) {
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
	if canWrite {
		if m.Status == StatusRecalled {
			return nil, validationError("message_id", "INVALID_VALUE", "收回的訊息不可新增附件。")
		}
		if m.DeletedForSenderAt != nil && m.SenderID == userID {
			return nil, validationError("message_id", "INVALID_VALUE", "已刪除的訊息不可新增附件。")
		}
	}
	return m, nil
}

// AddAttachments 為既有訊息新增附件（upsert 語義：同一檔案重複
// 關聯不重複寫入，回傳完整附件清單）。僅作者可新增（附件屬
// 訊息內容的一部分，非作者不得事後附加）；上限與去重語義同發送
// 時關聯。
func (s *Service) AddAttachments(ctx context.Context, userID, messageID string, fileIDs []string) ([]*Attachment, error) {
	m, err := s.loadAttachableMessage(ctx, userID, messageID, true)
	if err != nil {
		return nil, err
	}
	if m.SenderID != userID {
		return nil, ErrMessageNotFound
	}
	deduped, err := dedupeFileIDs(fileIDs)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	resolved, err := s.resolveAttachments(ctx, userID, deduped, now)
	if err != nil {
		return nil, err
	}
	for _, att := range resolved {
		att.MessageID = m.ID
	}
	if err := s.store.CreateAttachments(ctx, m.ID, resolved); err != nil {
		return nil, err
	}
	return s.store.ListAttachments(ctx, m.ID)
}

// ListAttachments 列出訊息的附件：僅成員可讀。
func (s *Service) ListAttachments(ctx context.Context, userID, messageID string) ([]*Attachment, error) {
	if _, err := s.loadAttachableMessage(ctx, userID, messageID, false); err != nil {
		return nil, err
	}
	return s.store.ListAttachments(ctx, messageID)
}

// forwardDepth 回傳訊息的轉傳鏈深度（原始訊息為 1）。鏈上
// 遇環時以目前深度回傳（防禦性斷環，FK 正常情形下不應成環）。
func (s *Service) forwardDepth(ctx context.Context, messageID string) (int, error) {
	depth := 1
	visited := map[string]bool{messageID: true}
	current := messageID
	for {
		fwd, err := s.store.GetForwardByForwardedID(ctx, current)
		if err != nil {
			return 0, err
		}
		if fwd == nil {
			return depth, nil
		}
		depth++
		if visited[fwd.SourceMessageID] {
			return depth, nil
		}
		visited[fwd.SourceMessageID] = true
		current = fwd.SourceMessageID
	}
}

// Forward 將訊息轉傳至另一對話：產生新訊息（內容與類型原樣
// 承載，發送者為轉傳者），並寫入轉傳紀錄。守衛：轉傳者須為
// 來源與目標對話的生效成員（任一不符一律 404）；收回訊息與
// 作者僅自己刪除的訊息不得轉傳；目標為一對一對話且封鎖時
// 404；轉傳鏈深度不得超過上限；新訊息不帶附件、不帶引用。
// 冪等語義同 Send：同一去重鍵回傳既有訊息（不得新建、不得重發
// NoteMessage、不得重複寫入轉傳紀錄）。新訊息的推播事件為
// message.received（經 createMessage 寫入）；轉傳紀錄列寫入在
// 訊息列提交之後（與 NoteMessage 同類已知限制，見 STATUS.md）。
func (s *Service) Forward(ctx context.Context, userID, sourceMessageID, targetConversationID, clientMessageID string) (*ForwardResult, error) {
	src, err := s.store.GetMessage(ctx, sourceMessageID)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, ErrMessageNotFound
	}
	member, err := s.chats.IsActiveMember(ctx, src.ConversationID, userID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrMessageNotFound
	}
	if src.Status == StatusRecalled {
		return nil, validationError("message_id", "INVALID_VALUE", "收回的訊息不可轉傳。")
	}
	if src.DeletedForSenderAt != nil && src.SenderID == userID {
		return nil, validationError("message_id", "INVALID_VALUE", "已刪除的訊息不可轉傳。")
	}
	targetMember, err := s.chats.IsActiveMember(ctx, targetConversationID, userID)
	if err != nil {
		return nil, err
	}
	if !targetMember {
		return nil, ErrMessageNotFound
	}
	if utf8.RuneCountInString(clientMessageID) > MaxClientMessageIDLength {
		return nil, validationError("client_message_id", "TOO_LONG", "去重鍵過長。")
	}
	if clientMessageID != "" {
		existing, err := s.store.GetByClientMessageID(ctx, targetConversationID, userID, clientMessageID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			fwd, err := s.store.GetForwardByForwardedID(ctx, existing.ID)
			if err != nil {
				return nil, err
			}
			return &ForwardResult{Message: existing, Forward: fwd}, nil
		}
	}
	depth, err := s.forwardDepth(ctx, src.ID)
	if err != nil {
		return nil, err
	}
	if depth+1 > MaxForwardDepth {
		return nil, validationError("message_id", "LIMIT_EXCEEDED", "轉傳鏈深度已達上限。")
	}
	blocked, err := s.chats.DirectBlocked(ctx, targetConversationID, userID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrMessageNotFound
	}
	now := s.now().UTC()
	m := &Message{
		ID:              ids.New(),
		ConversationID:  targetConversationID,
		SenderID:        userID,
		Kind:            src.Kind,
		Content:         src.Content,
		Status:          StatusSent,
		ClientMessageID: clientMessageID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if m.ID == "" {
		return nil, errors.New("message: 識別碼產生失敗")
	}
	if err := s.createMessage(ctx, m); err != nil {
		if errors.Is(err, ErrDuplicateClientMessage) && clientMessageID != "" {
			existing, readErr := s.store.GetByClientMessageID(ctx, targetConversationID, userID, clientMessageID)
			if readErr != nil {
				return nil, readErr
			}
			if existing != nil {
				fwd, err := s.store.GetForwardByForwardedID(ctx, existing.ID)
				if err != nil {
					return nil, err
				}
				return &ForwardResult{Message: existing, Forward: fwd}, nil
			}
		}
		return nil, err
	}
	fwdID := ids.New()
	if fwdID == "" {
		return nil, errors.New("message: 識別碼產生失敗")
	}
	fwd := &Forward{
		ID:                   fwdID,
		SourceMessageID:      src.ID,
		ForwardedMessageID:   m.ID,
		SourceConversationID: src.ConversationID,
		TargetConversationID: targetConversationID,
		ForwardedBy:          userID,
		CreatedAt:            now,
	}
	if err := s.store.CreateForward(ctx, fwd); err != nil {
		return nil, err
	}
	if err := s.chats.NoteMessage(ctx, targetConversationID, m.ID, now); err != nil {
		return nil, err
	}
	return &ForwardResult{Message: m, Forward: fwd, IsNew: true}, nil
}

// GetForward 讀取某訊息的轉傳紀錄：僅該訊息所屬對話的成員
// 可讀；訊息並非轉傳產物時與查無訊息／非成員同為 404（不得
// 區分，以防洩漏）。
func (s *Service) GetForward(ctx context.Context, userID, messageID string) (*Forward, error) {
	if _, err := s.Get(ctx, userID, messageID); err != nil {
		return nil, err
	}
	fwd, err := s.store.GetForwardByForwardedID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if fwd == nil {
		return nil, ErrMessageNotFound
	}
	return fwd, nil
}

func cloneAttachment(a *Attachment) *Attachment {
	if a == nil {
		return nil
	}
	cp := *a
	return &cp
}

func cloneForward(f *Forward) *Forward {
	if f == nil {
		return nil
	}
	cp := *f
	return &cp
}

func attachmentKey(messageID, mediaFileID string) string {
	return messageID + "\x00" + mediaFileID
}

// CreateAttachments 寫入附件關聯列（同一檔案重複關聯不重複寫入）。
func (s *MemoryStore) CreateAttachments(_ context.Context, messageID string, atts []*Attachment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, att := range atts {
		key := attachmentKey(messageID, att.MediaFileID)
		if _, exists := s.attachments[key]; exists {
			continue
		}
		cp := cloneAttachment(att)
		cp.MessageID = messageID
		s.attachments[key] = cp
	}
	return nil
}

// ListAttachments 列出訊息的附件（依建立時間順序）。
func (s *MemoryStore) ListAttachments(_ context.Context, messageID string) ([]*Attachment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Attachment
	for _, att := range s.attachments {
		if att.MessageID == messageID {
			out = append(out, cloneAttachment(att))
		}
	}
	sortAttachments(out)
	return out, nil
}

// GetAttachment 讀取訊息對某檔案的附件列；不存在時回傳 (nil, nil)。
func (s *MemoryStore) GetAttachment(_ context.Context, messageID, mediaFileID string) (*Attachment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneAttachment(s.attachments[attachmentKey(messageID, mediaFileID)]), nil
}

// CreateForward 寫入轉傳紀錄（同一新訊息重複寫入時保留首次列）。
func (s *MemoryStore) CreateForward(_ context.Context, f *Forward) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.forwards[f.ForwardedMessageID]; exists {
		return nil
	}
	s.forwards[f.ForwardedMessageID] = cloneForward(f)
	return nil
}

// GetForwardByForwardedID 讀取某新訊息的轉傳紀錄；該訊息並非
// 轉傳產物時回傳 (nil, nil)。
func (s *MemoryStore) GetForwardByForwardedID(_ context.Context, forwardedMessageID string) (*Forward, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneForward(s.forwards[forwardedMessageID]), nil
}

func sortAttachments(out []*Attachment) {
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.Before(out[i].CreatedAt) ||
				(out[j].CreatedAt.Equal(out[i].CreatedAt) && out[j].ID < out[i].ID) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
}
