// Package broker 是 Realtime Gateway 的發佈／訂閱抽象（Phase 05）。
//
// 邊界：Gateway 不得自行寫入 messages（DATA_FLOW.md §3 寫入路徑
// 唯一）；本套件只負責行程內／NATS 的事件扇出，事件內容不得含
// E2EE 明文（事件封包只含識別碼與最小欄位，見
// MODULE_BOUNDARIES.md §3）。NATS JetStream 轉接器只留介面與
// TODO，不得在本里程碑新增 NATS 客戶端依賴；預設以後端以
// MemoryBroker 行程內扇出，跨行程扇出待 NATS 轉接器落地。
package broker

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrNotConfigured 表示 Broker 後端未設定（NATS 轉接器待落地）。
var ErrNotConfigured = errors.New("broker: backend not configured")

// Event 是扇出事件：Topic 為訂閱主題（本里程碑慣例：
// "conversation.<對話識別碼>"）；Payload 為已編碼的 WebSocket
// 信封位元組（由 hub 層組裝，本套件不解析、不改寫）；
// FromConnID 為發起連線的內部識別碼（僅供 hub 層過濾回送，
// 不寫入 Payload、不對外洩漏）。
type Event struct {
	Topic       string
	Payload     []byte
	FromConnID  string
	PublishedAt time.Time
}

// Broker 是發佈／訂閱介面。Subscribe 回傳的通道在取消訂閱函式
// 被呼叫或 Broker 關閉時關閉；Publish 不得因單一訂閱者阻塞而
// 拖住發佈者（行程內實作以有界緩衝＋滿則丟棄，見 MemoryBroker）。
type Broker interface {
	Publish(ctx context.Context, ev Event) error
	Subscribe(topic string) (<-chan Event, func(), error)
	Close() error
}

// MemoryBroker 是 Broker 的行程內實作（開發與測試用）：同一行程
// 內的 Hub 共享同一實例。訂閱通道緩衝 64 則事件；緩衝滿時丟棄
// 最舊未讀事件（不得阻塞發佈者，不得無限緩衝；丟棄時計數，
// 供監控查詢——計數本身不作送達保證）。
type MemoryBroker struct {
	mu     sync.RWMutex
	subs   map[string]map[*subscription]struct{}
	drops  map[string]uint64
	closed bool
}

type subscription struct {
	ch     chan Event
	closed bool
}

// NewMemoryBroker 建立空的行程內 Broker。
func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{
		subs:  map[string]map[*subscription]struct{}{},
		drops: map[string]uint64{},
	}
}

// Publish 將事件投遞給該主題的全部訂閱者。
func (b *MemoryBroker) Publish(_ context.Context, ev Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errors.New("broker: closed")
	}
	ev.Payload = append([]byte(nil), ev.Payload...)
	ev.PublishedAt = time.Now().UTC()
	for sub := range b.subs[ev.Topic] {
		if sub.closed {
			continue
		}
		select {
		case sub.ch <- ev:
		default:
			// 緩衝滿：丟棄最舊一則（計入丟棄）再投遞，不得阻塞發佈者。
			select {
			case <-sub.ch:
				b.drops[ev.Topic]++
			default:
			}
			select {
			case sub.ch <- ev:
			default:
				b.drops[ev.Topic]++
			}
		}
	}
	return nil
}

// Subscribe 訂閱主題，回傳接收通道與取消訂閱函式。
func (b *MemoryBroker) Subscribe(topic string) (<-chan Event, func(), error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, nil, errors.New("broker: closed")
	}
	sub := &subscription{ch: make(chan Event, 64)}
	if b.subs[topic] == nil {
		b.subs[topic] = map[*subscription]struct{}{}
	}
	b.subs[topic][sub] = struct{}{}
	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			if !sub.closed {
				sub.closed = true
				close(sub.ch)
			}
			delete(b.subs[topic], sub)
		})
	}
	return sub.ch, unsubscribe, nil
}

// Drops 回傳該主題的丟棄計數（監控用，不得解讀為送達保證）。
func (b *MemoryBroker) Drops(topic string) uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.drops[topic]
}

// Close 關閉全部訂閱通道；關閉後 Publish／Subscribe 皆回傳錯誤。
func (b *MemoryBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	for _, subs := range b.subs {
		for sub := range subs {
			if !sub.closed {
				sub.closed = true
				close(sub.ch)
			}
		}
	}
	b.subs = map[string]map[*subscription]struct{}{}
	return nil
}

// JetStreamBroker 是 NATS JetStream 轉接器的佔位實作（Phase 05
// 後續）：介面已定，後端未接。任何呼叫皆回傳 ErrNotConfigured，
// 不得假裝已連接 NATS、不得新增 NATS 客戶端依賴。
// TODO：NATS 轉接器落地時，以 JetStream 發佈／訂閱實作本介面，
// 主題對映與持久化設定待定稿（WEBSOCKET_PROTOCOL.md §10 缺口 7）。
type JetStreamBroker struct{}

// NewJetStreamBroker 建立佔位轉接器（尚未設定，永遠回傳未設定錯誤）。
func NewJetStreamBroker() *JetStreamBroker { return &JetStreamBroker{} }

func (b *JetStreamBroker) Publish(context.Context, Event) error {
	return ErrNotConfigured
}

func (b *JetStreamBroker) Subscribe(string) (<-chan Event, func(), error) {
	return nil, nil, ErrNotConfigured
}

func (b *JetStreamBroker) Close() error { return nil }
