// Package relay 是 Outbox 事件的發佈迴圈（訊息推播中繼的 api 側一半）。
//
// 責任：輪詢 Outbox 儲存、經 Publisher 發佈事件、依結果標記
// 已發佈或失敗（死信）。發佈語義為「至少一次」：發佈成功後才
// 標記，標記失敗或 Relay 崩潰時租約逾期後由下次認領重發；消費
// 端必須以事件識別碼冪等（見 Publication.EventID）。
//
// 明確未落地：Relay 發佈迴圈的行程落點。生產實作
// （NATS JetStream 發佈者，見 nats.go）已落地；迴圈目前由
// api 行程依 RELAY_ENABLED 環境開關啟動（見 cmd/api/main.go）。
// Worker 行程接線須先重構模組邊界（worker 為獨立 Go module，
// 無法 import api 的 internal 套件；重複 Outbox 邏輯違反一表一主），
// 列為後續里程碑，見 STATUS.md。
// Publication.Data 為 WebSocket 信封的 data 物件；信封其餘欄位
// （event_id、version、timestamp、correlation_id）由消費端／
// Gateway 組裝，本套件不組裝信封。
package relay

import (
	"context"
	"encoding/json"
	"time"

	"ether.local/api/internal/outbox"
)

// Publication 是發佈給 Broker 的一則事件：Topic 為訂閱主題，
// Event 為 §21 事件名，Data 為信封 data 物件，EventID 為 Outbox
// 事件列識別碼（消費端冪等鍵，不得重複投遞同一 EventID 兩次
// 而產生重複顯示）。
type Publication struct {
	EventID string
	Topic   string
	Event   string
	Data    json.RawMessage
}

// Publisher 是事件發佈介面（NATS JetStream 發佈者待落地）。
type Publisher interface {
	// Publish 發佈一則事件；回傳錯誤時 Relay 記錄失敗並重試
	// （達上限後死信），不得標記為已發佈。
	Publish(ctx context.Context, p Publication) error
}

// DefaultBatchSize 是每次輪詢認領的事件數上限（實作預設，待定稿覆核）。
const DefaultBatchSize = 100

// DefaultMaxAttempts 是單一事件的發佈重試上限（實作預設，待定稿覆核）。
const DefaultMaxAttempts = 5

// DefaultPollInterval 是輪詢間隔（實作預設，待定稿覆核）。
const DefaultPollInterval = time.Second

// Relay 輪詢 Outbox 並經 Publisher 發佈。
type Relay struct {
	store       outbox.Store
	publisher   Publisher
	batchSize   int
	maxAttempts int
	interval    time.Duration
	// ErrHandler 在輪詢失敗（認領／標記錯誤）時被呼叫（可為空）；
	// 輪詢失敗不終止迴圈（下次輪詢重試），但不得靜默吞錯而不留痕跡。
	ErrHandler func(error)
}

// NewRelay 建立 Relay。batchSize／maxAttempts／interval 小於等於 0
// 時取預設值。
func NewRelay(store outbox.Store, publisher Publisher, batchSize, maxAttempts int, interval time.Duration) *Relay {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	if maxAttempts <= 0 {
		maxAttempts = DefaultMaxAttempts
	}
	if interval <= 0 {
		interval = DefaultPollInterval
	}
	return &Relay{
		store:       store,
		publisher:   publisher,
		batchSize:   batchSize,
		maxAttempts: maxAttempts,
		interval:    interval,
	}
}

// RunOnce 認領一批事件並逐一發佈，回傳成功發佈數。認領失敗時
// 回傳錯誤（本批未處理）；單一事件發佈失敗時記錄失敗並繼續
// 下一則（不得因單一事件卡住整批）；標記失敗時回傳錯誤（事件
// 保留 processing 狀態，租約逾期後重發，不得假裝已發佈）。
func (r *Relay) RunOnce(ctx context.Context) (int, error) {
	events, err := r.store.Claim(ctx, r.batchSize)
	if err != nil {
		return 0, err
	}
	published := 0
	for _, ev := range events {
		pubErr := r.publisher.Publish(ctx, Publication{
			EventID: ev.ID,
			Topic:   ev.Topic,
			Event:   ev.EventName,
			Data:    ev.Payload,
		})
		if pubErr != nil {
			if _, markErr := r.store.MarkFailed(ctx, ev.ID, r.maxAttempts); markErr != nil {
				return published, markErr
			}
			continue
		}
		if err := r.store.MarkPublished(ctx, ev.ID); err != nil {
			return published, err
		}
		published++
	}
	return published, nil
}

// Run 輪詢發佈直至 ctx 取消。輪詢錯誤經 ErrHandler 回報（未設定
// 時忽略）並繼續下一輪；ctx 取消時回傳 nil（不得將取消誤報為錯誤）。
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := r.RunOnce(ctx); err != nil && r.ErrHandler != nil {
				r.ErrHandler(err)
			}
		}
	}
}
