package message

import (
	"context"
	"errors"
	"sort"
	"time"

	"ether.local/api/internal/ids"
)

// 本檔案實作訊息反應（reactions）與置頂（pins）的領域邏輯。
//
// 邊界：反應與置頂屬 Message 模組擁有（MODULE_BOUNDARIES.md §5.7；
// 表見 migrations/000010）。反應／置頂寫入不產生 Outbox 事件：
// WEBSOCKET_PROTOCOL.md §21（§6）的 15 個事件名中沒有反應與置頂
// 對應的事件名，不得自創事件名（缺口見 STATUS.md 待決定 21）；
// 客戶端以 REST 查詢同步反應彙總與置頂清單。
// 非成員的一切查詢一律 404，不得洩漏存在性（ERROR_CODES.md §6.3）。

// AllowedEmojis 是反應允許清單（本里程碑實作預設，待端點
// Schema／產品定稿覆核；資料庫不加 CHECK，沿全專案慣例）。
// 比較以完整字串相等為準（含變體選擇符，例如 ❤️ 為兩碼點）。
var AllowedEmojis = []string{"👍", "❤️", "😂", "😮", "😢", "😡", "🙏", "🎉"}

// MaxPinsPerConversation 是每對話同時生效的置頂數上限
// （本里程碑實作預設，待定稿覆核；服務層 enforced）。
const MaxPinsPerConversation = 5

// Reaction 是 message_reactions 的一列。Emoji 對應資料庫欄位
// reaction（ERD §6.3.6）；RemovedAt 非空表示已取消（標記保留，
// 不得物理刪除歷史）。
type Reaction struct {
	ID        string
	MessageID string
	UserID    string
	Emoji     string
	CreatedAt time.Time
	UpdatedAt time.Time
	RemovedAt *time.Time
}

// Pin 是 message_pins 的一列。UnpinnedAt 非空表示已取消置頂
// （標記保留，不得物理刪除歷史）。
type Pin struct {
	ID             string
	ConversationID string
	MessageID      string
	PinnedBy       string
	PinnedAt       time.Time
	UnpinnedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ReactionCount 是單一表情的彙總：人數與呼叫者是否已反應。
type ReactionCount struct {
	Emoji       string
	Count       int
	ReactedByMe bool
}

// ReactionSummary 是某訊息的反應彙總：依人數降序、表情升序
// 排列（確定性順序，不得以 map 迭代順序回傳）。
type ReactionSummary struct {
	MessageID string
	Reactions []ReactionCount
}

func isAllowedEmoji(emoji string) bool {
	for _, allowed := range AllowedEmojis {
		if emoji == allowed {
			return true
		}
	}
	return false
}

func validateEmoji(emoji string) error {
	if emoji == "" {
		return validationError("reaction", "REQUIRED", "反應不可為空。")
	}
	if !isAllowedEmoji(emoji) {
		return validationError("reaction", "INVALID_VALUE", "不支援的反應。")
	}
	return nil
}

func summarizeReactions(messageID, viewerID string, rows []*Reaction) *ReactionSummary {
	counts := map[string]*ReactionCount{}
	for _, r := range rows {
		if r.RemovedAt != nil {
			continue
		}
		c, ok := counts[r.Emoji]
		if !ok {
			c = &ReactionCount{Emoji: r.Emoji}
			counts[r.Emoji] = c
		}
		c.Count++
		if r.UserID == viewerID {
			c.ReactedByMe = true
		}
	}
	out := make([]ReactionCount, 0, len(counts))
	for _, c := range counts {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Emoji < out[j].Emoji
	})
	return &ReactionSummary{MessageID: messageID, Reactions: out}
}

// loadReactableMessage 讀取訊息並驗證呼叫者為對話生效成員：
// 查無訊息或非成員一律 404。收回訊息不得新增反應／置頂；作者
// 僅自己刪除的訊息，作者本人不得新增反應／置頂（他人檢視不受
// 影響）。呼叫方須先以 canWrite 區分寫入與唯讀路徑：唯讀路徑
// （列出、取消）不套用收回／刪除守衛。
func (s *Service) loadReactableMessage(ctx context.Context, userID, messageID string, canWrite bool) (*Message, error) {
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
			return nil, validationError("message_id", "INVALID_VALUE", "收回的訊息不可反應或置頂。")
		}
		if m.DeletedForSenderAt != nil && m.SenderID == userID {
			return nil, validationError("message_id", "INVALID_VALUE", "已刪除的訊息不可反應或置頂。")
		}
	}
	return m, nil
}

// SetReaction 新增或更換使用者對訊息的反應：同一使用者對同一
// 訊息同時僅一則生效反應；重送相同表情為冪等成功；更換時舊列
// 標記取消。不寫 Outbox 事件（見檔頭 §21 缺口）。
func (s *Service) SetReaction(ctx context.Context, userID, messageID, emoji string) (*ReactionSummary, error) {
	if err := validateEmoji(emoji); err != nil {
		return nil, err
	}
	m, err := s.loadReactableMessage(ctx, userID, messageID, true)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	r := &Reaction{
		ID:        ids.New(),
		MessageID: m.ID,
		UserID:    userID,
		Emoji:     emoji,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if r.ID == "" {
		return nil, errors.New("message: 識別碼產生失敗")
	}
	if _, err := s.store.SetReaction(ctx, r); err != nil {
		return nil, err
	}
	return s.ListReactions(ctx, userID, messageID)
}

// RemoveReaction 取消使用者對訊息的反應（冪等；無生效反應時
// 亦成功）。收回訊息的反應仍可取消（清理路徑，不套用寫入守衛）。
func (s *Service) RemoveReaction(ctx context.Context, userID, messageID string) (*ReactionSummary, error) {
	if _, err := s.loadReactableMessage(ctx, userID, messageID, false); err != nil {
		return nil, err
	}
	if err := s.store.RemoveReaction(ctx, messageID, userID); err != nil {
		return nil, err
	}
	return s.ListReactions(ctx, userID, messageID)
}

// ListReactions 列出訊息的反應彙總：僅成員可讀。
func (s *Service) ListReactions(ctx context.Context, userID, messageID string) (*ReactionSummary, error) {
	if _, err := s.loadReactableMessage(ctx, userID, messageID, false); err != nil {
		return nil, err
	}
	rows, err := s.store.ListActiveReactions(ctx, messageID)
	if err != nil {
		return nil, err
	}
	return summarizeReactions(messageID, userID, rows), nil
}

// PinMessage 置頂訊息：僅對話生效成員可置頂；訊息須屬於該
// 對話（否則 404）；已置頂時回傳既有列（冪等）；生效置頂數達
// 上限時拒絕。收回／僅自己刪除的訊息守衛同 SetReaction。
func (s *Service) PinMessage(ctx context.Context, userID, conversationID, messageID string) (*Pin, bool, error) {
	member, err := s.chats.IsActiveMember(ctx, conversationID, userID)
	if err != nil {
		return nil, false, err
	}
	if !member {
		return nil, false, ErrMessageNotFound
	}
	m, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return nil, false, err
	}
	if m == nil || m.ConversationID != conversationID {
		return nil, false, ErrMessageNotFound
	}
	if m.Status == StatusRecalled {
		return nil, false, validationError("message_id", "INVALID_VALUE", "收回的訊息不可置頂。")
	}
	if m.DeletedForSenderAt != nil && m.SenderID == userID {
		return nil, false, validationError("message_id", "INVALID_VALUE", "已刪除的訊息不可置頂。")
	}
	count, err := s.store.CountActivePins(ctx, conversationID)
	if err != nil {
		return nil, false, err
	}
	if count >= MaxPinsPerConversation {
		// 已置頂的重送為冪等成功，不受上限影響：先查既有列。
		if existing, err := s.store.GetActivePin(ctx, conversationID, m.ID); err != nil {
			return nil, false, err
		} else if existing != nil {
			return existing, false, nil
		}
		return nil, false, validationError("message_id", "LIMIT_EXCEEDED", "此對話的置頂訊息已達上限。")
	}
	now := s.now().UTC()
	p := &Pin{
		ID:             ids.New(),
		ConversationID: conversationID,
		MessageID:      m.ID,
		PinnedBy:       userID,
		PinnedAt:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if p.ID == "" {
		return nil, false, errors.New("message: 識別碼產生失敗")
	}
	return s.store.PinMessage(ctx, p)
}

// UnpinMessage 取消置頂（冪等）：僅對話生效成員可操作；訊息須
// 屬於該對話。回傳被取消的列；已取消時回傳最近歷史列；從未
// 置頂時回傳 (nil, nil)，由 handler 轉為 pinned=false 回應。
func (s *Service) UnpinMessage(ctx context.Context, userID, conversationID, messageID string) (*Pin, error) {
	member, err := s.chats.IsActiveMember(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrMessageNotFound
	}
	m, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.ConversationID != conversationID {
		return nil, ErrMessageNotFound
	}
	return s.store.UnpinMessage(ctx, conversationID, messageID, s.now().UTC())
}

// ListPins 列出對話的生效置頂訊息（依置頂時間順序）：僅成員可讀。
func (s *Service) ListPins(ctx context.Context, userID, conversationID string) ([]*Pin, error) {
	member, err := s.chats.IsActiveMember(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrMessageNotFound
	}
	return s.store.ListActivePins(ctx, conversationID)
}

func reactionKey(messageID, userID, emoji string) string {
	return messageID + "\x00" + userID + "\x00" + emoji
}

func cloneReaction(r *Reaction) *Reaction {
	if r == nil {
		return nil
	}
	cp := *r
	return &cp
}

func clonePin(p *Pin) *Pin {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

func pinKey(conversationID, messageID string) string {
	return conversationID + "\x00" + messageID
}

// SetReaction 寫入或更換反應：同一互斥鎖內完成舊列標記與新列
// 寫入；相同表情已生效時回傳既有列（不重複寫入）。
func (s *MemoryStore) SetReaction(_ context.Context, r *Reaction) (*Reaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for _, existing := range s.reactions {
		if existing.MessageID == r.MessageID && existing.UserID == r.UserID && existing.RemovedAt == nil {
			if existing.Emoji == r.Emoji {
				return cloneReaction(existing), nil
			}
			removed := now
			existing.RemovedAt = &removed
			existing.UpdatedAt = now
		}
	}
	cp := cloneReaction(r)
	s.reactions[reactionKey(r.MessageID, r.UserID, r.Emoji)] = cp
	return cloneReaction(cp), nil
}

// RemoveReaction 取消使用者對訊息的全部生效反應（冪等）。
func (s *MemoryStore) RemoveReaction(_ context.Context, messageID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for _, existing := range s.reactions {
		if existing.MessageID == messageID && existing.UserID == userID && existing.RemovedAt == nil {
			removed := now
			existing.RemovedAt = &removed
			existing.UpdatedAt = now
		}
	}
	return nil
}

// ListActiveReactions 列出生效中的反應列（依建立時間順序）。
func (s *MemoryStore) ListActiveReactions(_ context.Context, messageID string) ([]*Reaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Reaction
	for _, r := range s.reactions {
		if r.MessageID == messageID && r.RemovedAt == nil {
			out = append(out, cloneReaction(r))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// CountActivePins 回傳對話的生效置頂數。
func (s *MemoryStore) CountActivePins(_ context.Context, conversationID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, p := range s.pins {
		if p.ConversationID == conversationID && p.UnpinnedAt == nil {
			n++
		}
	}
	return n, nil
}

// GetActivePin 讀取對話中某訊息的生效置頂列；未置頂時回傳 (nil, nil)。
func (s *MemoryStore) GetActivePin(_ context.Context, conversationID, messageID string) (*Pin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if id := s.pinActive[pinKey(conversationID, messageID)]; id != "" {
		if existing := s.pins[id]; existing != nil && existing.UnpinnedAt == nil {
			return clonePin(existing), nil
		}
	}
	return nil, nil
}

// PinMessage 寫入置頂列；已置頂時回傳既有列與 isNew=false。
func (s *MemoryStore) PinMessage(_ context.Context, p *Pin) (*Pin, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id := s.pinActive[pinKey(p.ConversationID, p.MessageID)]; id != "" {
		if existing := s.pins[id]; existing != nil && existing.UnpinnedAt == nil {
			return clonePin(existing), false, nil
		}
	}
	cp := clonePin(p)
	s.pins[cp.ID] = cp
	s.pinActive[pinKey(p.ConversationID, p.MessageID)] = cp.ID
	return clonePin(cp), true, nil
}

// UnpinMessage 取消置頂（冪等）：回傳被取消的列；已取消時回傳
// 最近一筆歷史列；從未置頂時回傳 (nil, nil)。
func (s *MemoryStore) UnpinMessage(_ context.Context, conversationID, messageID string, at time.Time) (*Pin, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id := s.pinActive[pinKey(conversationID, messageID)]; id != "" {
		if existing := s.pins[id]; existing != nil && existing.UnpinnedAt == nil {
			unpinned := at
			existing.UnpinnedAt = &unpinned
			existing.UpdatedAt = at
			delete(s.pinActive, pinKey(conversationID, messageID))
			return clonePin(existing), nil
		}
	}
	var latest *Pin
	for _, p := range s.pins {
		if p.ConversationID == conversationID && p.MessageID == messageID {
			if latest == nil || p.PinnedAt.After(latest.PinnedAt) ||
				(p.PinnedAt.Equal(latest.PinnedAt) && p.ID > latest.ID) {
				latest = p
			}
		}
	}
	return clonePin(latest), nil
}

// ListActivePins 列出生效中的置頂列（依置頂時間順序）。
func (s *MemoryStore) ListActivePins(_ context.Context, conversationID string) ([]*Pin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Pin
	for _, p := range s.pins {
		if p.ConversationID == conversationID && p.UnpinnedAt == nil {
			out = append(out, clonePin(p))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].PinnedAt.Equal(out[j].PinnedAt) {
			return out[i].PinnedAt.Before(out[j].PinnedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
