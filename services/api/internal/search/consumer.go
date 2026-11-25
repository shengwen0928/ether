package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// 消費迴圈：Worker 經 NATS 訂閱 Outbox 事件（Relay 發佈，見
// internal/relay/nats.go），呼叫 Projector 寫入索引
// （DATA_FLOW.md §8：索引同步是事件驅動的投影，不得由業務寫入
// 路徑同步寫索引、不得繞過 NATS 以 Outbox 輪詢直寫索引）。
//
// 線路格式與 relay.Envelope 的 JSON 欄位名須一致（見
// internal/relay/nats.go 檔頭）；本檔案以本地結構解碼，不得
// import relay 套件（模組邊界：Search 不依賴 Relay 套件）。
//
// 語義：「至少一次」投遞——處理成功才 Ack；解碼或投影失敗時
// Nak，NATS 依消費端設定重投（見 nats_consumer.go 的
// MaxDeliver 上限；超過上限仍未成功時訊息不再投遞，失敗經
// ErrHandler 回報，不得靜默丟棄）。Projector 以 eventID 冪等，
// 重投不得產生重複文件。未知事件名直接 Ack 略過（同一 Stream
// 承載多模組事件，搜尋只消費 message.received／message.recalled）。
// 收回事件不得含原內容（見 internal/message/events.go）。

// Delivery 是一則待處理的 NATS 投遞：Data 為線路承載位元組；
// Ack／Nak 由訂閱轉接器提供（可為 nil，視為 no-op）。
type Delivery struct {
	Data []byte
	Ack  func() error
	Nak  func() error
}

// Consumer 是索引填充的消費迴圈：自投遞通道取事件、經 Projector
// 寫入索引。Run 不擁有通道（關閉由訂閱轉接器負責）。
type Consumer struct {
	projector  *Projector
	errHandler func(error)
}

// NewConsumer 建立消費迴圈。projector 不可為 nil；errHandler
// 可為 nil（失敗僅計入 Nak，不另行回報）。
func NewConsumer(projector *Projector, errHandler func(error)) (*Consumer, error) {
	if projector == nil {
		return nil, errors.New("search: Projector 不可為 nil")
	}
	return &Consumer{projector: projector, errHandler: errHandler}, nil
}

// envelope 是線路承載的解碼結構（欄位名須與 relay.Envelope 一致）。
type envelope struct {
	EventID string          `json:"event_id"`
	Event   string          `json:"event"`
	Topic   string          `json:"topic"`
	Data    json.RawMessage `json:"data"`
}

// receivedData 是 message.received 的 data 物件（欄位名依
// WEBSOCKET_PROTOCOL.md §6.1.2 草案，與 internal/message
// 的 receivedPayload 一致）。
type receivedData struct {
	MessageID      string `json:"message_id"`
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	Kind           string `json:"kind"`
	Content        string `json:"content"`
	Sequence       int64  `json:"sequence"`
	CreatedAt      string `json:"created_at"`
}

// recalledData 是 message.recalled 的 data 物件（欄位名依
// WEBSOCKET_PROTOCOL.md §6.1.5 草案）。不得含原內容。
type recalledData struct {
	MessageID string `json:"message_id"`
}

// Run 消費投遞直至 ctx 取消或通道關閉，皆回傳 nil（不得將取消
// 誤報為錯誤）。單則投遞失敗時 Nak 並繼續下一則，不得因單則
// 毒訊息卡住迴圈。
func (c *Consumer) Run(ctx context.Context, deliveries <-chan Delivery) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return nil
			}
			c.handle(ctx, d)
		}
	}
}

func (c *Consumer) report(err error) {
	if c.errHandler != nil && err != nil {
		c.errHandler(err)
	}
}

func ack(d Delivery) {
	if d.Ack != nil {
		_ = d.Ack()
	}
}

// handle 處理單則投遞：成功 Ack、失敗 Nak（皆經 report 回報
// Ack／Nak 本身的錯誤）。
func (c *Consumer) handle(ctx context.Context, d Delivery) {
	var env envelope
	if err := json.Unmarshal(d.Data, &env); err != nil {
		c.report(fmt.Errorf("search: 線路訊息解碼失敗: %w", err))
		c.nak(d)
		return
	}
	switch env.Event {
	case "message.received":
		if err := c.applyReceived(ctx, env); err != nil {
			c.report(err)
			c.nak(d)
			return
		}
	case "message.recalled":
		if err := c.applyRecalled(ctx, env); err != nil {
			c.report(err)
			c.nak(d)
			return
		}
	default:
		// 非搜尋事件：Ack 略過，不得 Nak（否則無限重投）。
	}
	if d.Ack != nil {
		if err := d.Ack(); err != nil {
			c.report(fmt.Errorf("search: Ack 失敗: %w", err))
		}
	}
}

func (c *Consumer) nak(d Delivery) {
	if d.Nak != nil {
		if err := d.Nak(); err != nil {
			c.report(fmt.Errorf("search: Nak 失敗: %w", err))
		}
	}
}

func (c *Consumer) applyReceived(ctx context.Context, env envelope) error {
	if env.EventID == "" {
		return errors.New("search: 事件識別碼不可為空")
	}
	var data receivedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return fmt.Errorf("search: message.received 解碼失敗: %w", err)
	}
	if data.MessageID == "" || data.ConversationID == "" {
		return errors.New("search: message.received 缺少訊息或對話識別碼")
	}
	createdAt, err := time.Parse(time.RFC3339, data.CreatedAt)
	if err != nil {
		return fmt.Errorf("search: message.received 時間格式錯誤: %w", err)
	}
	return c.projector.ApplyReceived(ctx, Document{
		MessageID:      data.MessageID,
		ConversationID: data.ConversationID,
		SenderID:       data.SenderID,
		Kind:           data.Kind,
		Content:        data.Content,
		SequenceNumber: data.Sequence,
		CreatedAt:      createdAt,
	}, env.EventID)
}

func (c *Consumer) applyRecalled(ctx context.Context, env envelope) error {
	if env.EventID == "" {
		return errors.New("search: 事件識別碼不可為空")
	}
	var data recalledData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return fmt.Errorf("search: message.recalled 解碼失敗: %w", err)
	}
	if data.MessageID == "" {
		return errors.New("search: message.recalled 缺少訊息識別碼")
	}
	return c.projector.ApplyRecalled(ctx, data.MessageID, env.EventID)
}

// DocumentScanner 是重建流程的文件來源介面：呼叫方自 PostgreSQL
// 全量掃描提供文件（不得以索引反寫業務表）。PG 掃描實作待
// 後續里程碑（須新增 message 儲存層的掃描方法），本里程碑只定
// 介面與重建函式。
type DocumentScanner interface {
	// Scan 依序回呼每一則文件；回呼回傳錯誤時中止掃描並回傳該錯誤。
	Scan(ctx context.Context, fn func(Document) error) error
}

// RebuildFromScanner 以掃描來源的全量文件重建索引（重建入口，
// 見本套件檔頭）。重建期間消費端須暫停或保證事件順序（見
// MemoryIndex.ReplaceAll 註解），否則重建後重放的舊事件可能
// 再次套用。
func (p *Projector) RebuildFromScanner(ctx context.Context, scanner DocumentScanner) error {
	if scanner == nil {
		return errors.New("search: 掃描來源不可為 nil")
	}
	var docs []Document
	if err := scanner.Scan(ctx, func(doc Document) error {
		docs = append(docs, doc)
		return nil
	}); err != nil {
		return fmt.Errorf("search: 掃描重建來源失敗: %w", err)
	}
	return p.Rebuild(ctx, docs)
}
