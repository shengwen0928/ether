package search

import (
	"errors"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// NATS 訂閱轉接器：將 JetStream 訂閱橋接為 Consumer 的投遞通道。
//
// 消費端設定（實作預設，待定稿覆核）：durable 名稱與 queue group
// 皆為 DefaultConsumerDurable（同一 durable 的多副本以 queue group
// 分工，同一訊息只由一個副本處理）；ManualAck（處理成功才 Ack，
// 失敗 Nak 重投）；AckWait 30 秒；MaxDeliver 5 次（超過上限仍未
// 成功時 NATS 不再投遞，失敗由 Consumer 的 ErrHandler 回報）。
// 訂閱主題為 DefaultConsumerSubject（ether.events.>，含全部
// Outbox 事件；非搜尋事件由 Consumer Ack 略過）。
//
// Stream 由部署方預先建立（見 relay.DefaultStreamName），轉接器
// 不自動建 Stream，訂閱不存在的 Stream 時建構失敗（fail-closed）。

// DefaultConsumerSubject 是索引消費迴圈的訂閱主題（實作預設，
// 待部署定稿覆核）。前綴須與 relay.DefaultSubjectPrefix 一致。
const DefaultConsumerSubject = "ether.events.>"

// DefaultConsumerDurable 是索引消費迴圈的 durable 名稱與 queue
// group（實作預設，待定稿覆核）。
const DefaultConsumerDurable = "ether-search-indexer"

// DefaultConsumerAckWait 是訊息的 Ack 等待逾時（實作預設）。
const DefaultConsumerAckWait = 30 * time.Second

// DefaultConsumerMaxDeliver 是單則訊息的最大投遞次數（實作預設）。
const DefaultConsumerMaxDeliver = 5

// consumerChannelBuffer 是投遞通道的緩衝大小（實作預設）。
const consumerChannelBuffer = 64

// NATSSubscription 是 Consumer 的 NATS 訂閱來源：收到的 NATS
// 訊息轉為 Delivery，Ack／Nak 對映至 NATS 訊息的 Ack／Nak。
// 回呼以非阻塞方式投遞：通道滿時不 Ack（NATS 於 AckWait 逾時
// 後重投），不得阻塞 NATS 投遞迴圈、不得無聲丟棄已 Ack 的訊息。
type NATSSubscription struct {
	sub    *nats.Subscription
	ch     chan Delivery
	mu     sync.RWMutex
	closed bool
}

// NewNATSSubscription 建立 JetStream 訂閱。nc 不可為 nil；
// subject 為空時取 DefaultConsumerSubject；durable／group 為空
// 時取 DefaultConsumerDurable。訂閱失敗時回傳錯誤，不得回傳
// 半建好的訂閱。
func NewNATSSubscription(nc *nats.Conn, subject, durable, group string) (*NATSSubscription, error) {
	if nc == nil {
		return nil, errors.New("search: NATS 連線不可為 nil")
	}
	if subject == "" {
		subject = DefaultConsumerSubject
	}
	if durable == "" {
		durable = DefaultConsumerDurable
	}
	if group == "" {
		group = DefaultConsumerDurable
	}
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	s := &NATSSubscription{ch: make(chan Delivery, consumerChannelBuffer)}
	sub, err := js.QueueSubscribe(subject, group, func(msg *nats.Msg) {
		d := Delivery{
			Data: append([]byte(nil), msg.Data...),
			Ack:  func() error { return msg.Ack() },
			Nak:  func() error { return msg.Nak() },
		}
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.closed {
			return
		}
		select {
		case s.ch <- d:
		default:
			// 通道滿：不投遞、不 Ack，NATS 逾時後重投。
		}
	}, nats.Durable(durable), nats.ManualAck(),
		nats.AckWait(DefaultConsumerAckWait), nats.MaxDeliver(DefaultConsumerMaxDeliver))
	if err != nil {
		return nil, err
	}
	s.sub = sub
	return s, nil
}

// Channel 回傳投遞通道（Close 後關閉）。
func (s *NATSSubscription) Channel() <-chan Delivery { return s.ch }

// Close 取消訂閱並關閉投遞通道（冪等）。關閉後 Consumer.Run
// 收到通道關閉訊號並回傳 nil。Close 會等待進行中的回呼完成，
// 不得與回呼投遞競態。
func (s *NATSSubscription) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	close(s.ch)
	if s.sub != nil {
		return s.sub.Unsubscribe()
	}
	return nil
}
