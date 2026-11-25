package relay

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"ether.local/api/internal/outbox"
)

type fakePublisher struct {
	mu        sync.Mutex
	published []Publication
	failTimes int // 前 N 次發佈回傳錯誤，之後成功。
	calls     int
}

func (f *fakePublisher) Publish(_ context.Context, p Publication) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls <= f.failTimes {
		return errors.New("relay: 發佈失敗（測試注入）")
	}
	f.published = append(f.published, p)
	return nil
}

func seedEvents(t *testing.T, store *outbox.MemoryStore, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		ev := &outbox.Event{
			EventName:      "message.received",
			Topic:          "conversation.conv-1",
			ConversationID: "conv-1",
			MessageID:      "m-1",
			Payload:        json.RawMessage(`{"message_id":"m-1"}`),
		}
		if err := store.Append(context.Background(), ev); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
}

func TestRunOnce_PublishesAndMarksPublished(t *testing.T) {
	store := outbox.NewMemoryStore()
	seedEvents(t, store, 3)
	pub := &fakePublisher{}
	r := NewRelay(store, pub, 10, 5, time.Second)
	n, err := r.RunOnce(context.Background())
	if err != nil || n != 3 {
		t.Fatalf("RunOnce = %d, %v, want 3", n, err)
	}
	if len(pub.published) != 3 {
		t.Fatalf("發佈數 = %d, want 3", len(pub.published))
	}
	for _, p := range pub.published {
		if p.Event != "message.received" || p.Topic != "conversation.conv-1" || p.EventID == "" {
			t.Fatalf("Publication = %+v", p)
		}
	}
	if claimed, _ := store.Claim(context.Background(), 10); len(claimed) != 0 {
		t.Fatalf("發佈後仍有待發佈事件：%d", len(claimed))
	}
}

func TestRunOnce_PublishFailureRetriesThenDeadLetters(t *testing.T) {
	store := outbox.NewMemoryStore()
	seedEvents(t, store, 1)
	pub := &fakePublisher{failTimes: 100}
	r := NewRelay(store, pub, 10, 3, time.Second)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if n, err := r.RunOnce(ctx); err != nil || n != 0 {
			t.Fatalf("失敗輪次 RunOnce = %d, %v", n, err)
		}
	}
	// 第三次失敗達上限：死信，不再被認領。
	if n, err := r.RunOnce(ctx); err != nil || n != 0 {
		t.Fatalf("死信輪次 RunOnce = %d, %v", n, err)
	}
	if claimed, _ := store.Claim(ctx, 10); len(claimed) != 0 {
		t.Fatalf("死信後仍可認領：%d", len(claimed))
	}
	if len(pub.published) != 0 {
		t.Fatalf("失敗發佈不得計入已發佈：%d", len(pub.published))
	}
}

func TestRunOnce_EmptyStoreIsNoop(t *testing.T) {
	store := outbox.NewMemoryStore()
	r := NewRelay(store, &fakePublisher{}, 10, 5, time.Second)
	if n, err := r.RunOnce(context.Background()); err != nil || n != 0 {
		t.Fatalf("空儲存 RunOnce = %d, %v", n, err)
	}
}

func TestRun_StopsOnCancel(t *testing.T) {
	store := outbox.NewMemoryStore()
	seedEvents(t, store, 2)
	pub := &fakePublisher{}
	r := NewRelay(store, pub, 10, 5, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run 回傳錯誤：%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("取消後 Run 未返回")
	}
	if len(pub.published) != 2 {
		t.Fatalf("發佈數 = %d, want 2", len(pub.published))
	}
}

func TestNewRelay_Defaults(t *testing.T) {
	r := NewRelay(outbox.NewMemoryStore(), &fakePublisher{}, 0, 0, 0)
	if r.batchSize != DefaultBatchSize || r.maxAttempts != DefaultMaxAttempts || r.interval != DefaultPollInterval {
		t.Fatalf("預設值 = %d/%d/%v", r.batchSize, r.maxAttempts, r.interval)
	}
}
