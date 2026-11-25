package message

import (
	"context"
	"encoding/json"
	"time"

	"ether.local/api/internal/ids"
	"ether.local/api/internal/outbox"
)

// 本檔案定義訊息寫入產生的 Outbox 事件。事件名以
// WEBSOCKET_PROTOCOL.md §6.1（§21）為準，一字不得改；不得自創
// 事件名。§21 沒有編輯與僅自己刪除對應的事件名：Edit 與
// DeleteForSelf 不寫 Outbox 事件（缺口見 STATUS.md 待決定 20），
// 客戶端以 REST 歷史同步得知變更，不得以假事件名發佈。
//
// Payload 只含已公開的 wire 欄位（與 handler 的回應欄位一致）。
// E2EE 未實作前 content 為不透明字串（見 doc.go）；E2EE 落地後，
// Payload 的 content 須改為密文、不得再含明文，本檔案須同步修改。

// EventMessageReceived 是訊息送出廣播事件（WEBSOCKET_PROTOCOL.md §6.1.2）。
const EventMessageReceived = "message.received"

// EventMessageRecalled 是訊息收回廣播事件（WEBSOCKET_PROTOCOL.md §6.1.5）。
const EventMessageRecalled = "message.recalled"

// receivedPayload 是 message.received 的 data 物件（欄位名依 §6.1.2 草案）。
type receivedPayload struct {
	MessageID        string `json:"message_id"`
	ConversationID   string `json:"conversation_id"`
	SenderID         string `json:"sender_id"`
	Kind             string `json:"kind"`
	Content          string `json:"content"`
	Sequence         int64  `json:"sequence"`
	ClientMessageID  string `json:"client_message_id,omitempty"`
	CreatedAt        string `json:"created_at"`
	ReplyToMessageID string `json:"reply_to_message_id,omitempty"`
	Status           string `json:"status"`
}

// recalledPayload 是 message.recalled 的 data 物件（欄位名依 §6.1.5 草案）。
// 收回事件不得含原內容。
type recalledPayload struct {
	MessageID      string `json:"message_id"`
	ConversationID string `json:"conversation_id"`
	RecalledBy     string `json:"recalled_by"`
	RecalledAt     string `json:"recalled_at"`
}

func conversationTopic(conversationID string) string {
	return "conversation." + conversationID
}

func newReceivedEvent(m *Message) (*outbox.Event, error) {
	payload, err := json.Marshal(receivedPayload{
		MessageID:        m.ID,
		ConversationID:   m.ConversationID,
		SenderID:         m.SenderID,
		Kind:             string(m.Kind),
		Content:          m.Content,
		Sequence:         m.SequenceNumber,
		ClientMessageID:  m.ClientMessageID,
		CreatedAt:        m.CreatedAt.UTC().Format(time.RFC3339),
		ReplyToMessageID: m.ReplyToMessageID,
		Status:           string(m.Status),
	})
	if err != nil {
		return nil, err
	}
	return &outbox.Event{
		ID:             ids.New(),
		EventName:      EventMessageReceived,
		Topic:          conversationTopic(m.ConversationID),
		ConversationID: m.ConversationID,
		MessageID:      m.ID,
		Payload:        payload,
		CreatedAt:      m.CreatedAt,
	}, nil
}

func newRecalledEvent(m *Message) (*outbox.Event, error) {
	recalledAt := m.UpdatedAt
	if m.RecalledAt != nil {
		recalledAt = *m.RecalledAt
	}
	payload, err := json.Marshal(recalledPayload{
		MessageID:      m.ID,
		ConversationID: m.ConversationID,
		RecalledBy:     m.SenderID,
		RecalledAt:     recalledAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return nil, err
	}
	return &outbox.Event{
		ID:             ids.New(),
		EventName:      EventMessageRecalled,
		Topic:          conversationTopic(m.ConversationID),
		ConversationID: m.ConversationID,
		MessageID:      m.ID,
		Payload:        payload,
		CreatedAt:      m.UpdatedAt,
	}, nil
}

// EventBuilder 由儲存層在訊息列寫入後（序號已分配）呼叫，
// 依寫入後的訊息建 Outbox 事件。回傳 nil, nil 表示此次寫入
// 不產生事件（儲存層只寫訊息列）。
type EventBuilder func(m *Message) (*outbox.Event, error)

// AtomicWriter 是支援「訊息寫入＋Outbox 事件同一交易」的儲存層。
// 實作須保證：訊息列與事件列同提交、同回滾；任一寫入失敗時，
// 另一寫入不得殘留（DATA_FLOW.md §3）。未實作此介面的儲存層
// 不得搭配 Outbox 寫入（服務層 fail-closed）。
type AtomicWriter interface {
	// CreateMessageWithEvent 寫入新訊息並分配序號，建事件後與
	// 訊息列同一交易提交。
	CreateMessageWithEvent(ctx context.Context, m *Message, build EventBuilder) error
	// SaveMessageWithEvent 更新既有訊息列，建事件後與訊息列
	// 同一交易提交。
	SaveMessageWithEvent(ctx context.Context, m *Message, build EventBuilder) error
}
