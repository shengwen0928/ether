// Package relay 的 NATS JetStream 發佈者實作。
//
// 本檔案落地 Publisher 的生產實作（見 relay.go）：Relay 認領的
// Outbox 事件經 JetStream 發佈，Worker／Gateway 訂閱消費。
// 線路格式（NATS 訊息承載，JSON；消費端以此格式解碼，見
// internal/search/consumer.go 的對應解碼結構，欄位名須一致）：
//
//	{"event_id":"<Outbox 事件識別碼>","event":"<§21 事件名>",
//	 "topic":"<訂閱主題>","data":<Outbox Payload 原樣>}
//
// 主題對映：NATS subject = DefaultSubjectPrefix ＋ Publication.Topic
// （例：ether.events.conversation.<對話識別碼>）；EventID 同時寫入
// NATS Msg-Id 標頭，JetStream 以此去重（同一事件重發不產生重複
// 訊息）。Stream 名稱見 DefaultStreamName：Stream 由部署方預先
// 建立（subject 須含 ether.events.>），發佈者不自動建 Stream——
// 發佈至不存在的 Stream 時回傳錯誤，Relay 記錄失敗並重試，不得
// 靜默吞錯。Stream 建立方式與保留策略待部署文件定稿。
//
// 依賴：github.com/nats-io/nats.go v1.52.0（go 1.25 可建置的
// 最新版；v1.53＋要求 go 1.26，與本專案釘選的 go 1.25 衝突，
// 不得升級至要求更新 go 工具鏈的版本）。
package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
)

// DefaultStreamName 是 Outbox 事件的 JetStream Stream 名稱
// （實作預設，待部署定稿覆核）。Stream 由部署方預先建立。
const DefaultStreamName = "ETHER_EVENTS"

// DefaultSubjectPrefix 是發佈 subject 的前綴（實作預設，待部署
// 定稿覆核）。完整 subject 為前綴 ＋ Publication.Topic。
const DefaultSubjectPrefix = "ether.events."

// Envelope 是 NATS 訊息的承載格式（見本檔檔頭）。Data 為 Outbox
// Payload 原樣（WebSocket 信封的 data 物件），不得改寫。
type Envelope struct {
	EventID string          `json:"event_id"`
	Event   string          `json:"event"`
	Topic   string          `json:"topic"`
	Data    json.RawMessage `json:"data"`
}

// jetStream 是 JetStreamPublisher 對 nats.go 的最小依賴面：
// 只需 PublishMsg，測試以假實作注入，不得直接依賴 *nats.Conn。
type jetStream interface {
	PublishMsg(m *nats.Msg, opts ...nats.PubOpt) (*nats.PubAck, error)
}

// JetStreamPublisher 是 Publisher 的 NATS JetStream 實作。
type JetStreamPublisher struct {
	js            jetStream
	subjectPrefix string
}

// NewJetStreamPublisher 建立 JetStream 發佈者。js 不可為 nil；
// subjectPrefix 為空時取 DefaultSubjectPrefix。
func NewJetStreamPublisher(js jetStream, subjectPrefix string) (*JetStreamPublisher, error) {
	if js == nil {
		return nil, errors.New("relay: JetStream 後端不可為 nil")
	}
	if subjectPrefix == "" {
		subjectPrefix = DefaultSubjectPrefix
	}
	return &JetStreamPublisher{js: js, subjectPrefix: subjectPrefix}, nil
}

// SubjectFor 回傳 Publication 對應的 NATS subject（供測試與
// 部署文件核對）。
func (p *JetStreamPublisher) SubjectFor(topic string) string {
	return p.subjectPrefix + topic
}

// Publish 將一則事件發佈至 JetStream（見本檔檔頭）。EventID、
// Topic、Data 任一缺失或 Data 非合法 JSON 時回傳錯誤且不發佈
// （fail-closed，不得發佈無法被消費端去重或解碼的訊息）。
func (p *JetStreamPublisher) Publish(ctx context.Context, pub Publication) error {
	if pub.EventID == "" {
		return errors.New("relay: EventID 不可為空")
	}
	if pub.Topic == "" {
		return errors.New("relay: Topic 不可為空")
	}
	if len(pub.Data) == 0 || !json.Valid(pub.Data) {
		return errors.New("relay: Data 須為合法 JSON")
	}
	payload, err := json.Marshal(Envelope{
		EventID: pub.EventID,
		Event:   pub.Event,
		Topic:   pub.Topic,
		Data:    pub.Data,
	})
	if err != nil {
		return fmt.Errorf("relay: 組裝 NATS 訊息失敗: %w", err)
	}
	msg := nats.NewMsg(p.SubjectFor(pub.Topic))
	msg.Data = payload
	// Msg-Id 標頭直接寫入訊息：JetStream 以此標頭去重，與
	// nats.MsgId 發佈選項等價（選項亦只是寫入同一標頭）。
	msg.Header.Set(nats.MsgIdHdr, pub.EventID)
	if _, err := p.js.PublishMsg(msg); err != nil {
		return fmt.Errorf("relay: JetStream 發佈失敗: %w", err)
	}
	return nil
}

// DialNATS 連線至 NATS 並取回 JetStreamContext 與關閉函式。
// 關閉函式須於行程關停時呼叫（先關 JetStream 發佈、再關連線
// 由 nats.Conn.Close 一併處理）。連線失敗時回傳錯誤，不得回傳
// 半建好的發佈者。
func DialNATS(url string) (*nats.Conn, nats.JetStreamContext, error) {
	if strings.TrimSpace(url) == "" {
		return nil, nil, errors.New("relay: NATS 位址不可為空")
	}
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, nil, fmt.Errorf("relay: 連線 NATS 失敗: %w", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("relay: 取得 JetStreamContext 失敗: %w", err)
	}
	return nc, js, nil
}
