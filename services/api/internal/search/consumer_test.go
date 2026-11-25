package search

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeDelivery 記錄 Ack／Nak 呼叫的測試投遞。
type fakeDelivery struct {
	data   []byte
	acked  int
	naked  int
	ackErr error
}

func (f *fakeDelivery) delivery() Delivery {
	return Delivery{
		Data: f.data,
		Ack: func() error {
			f.acked++
			return f.ackErr
		},
		Nak: func() error {
			f.naked++
			return nil
		},
	}
}

func envelopeJSON(eventID, event string, data json.RawMessage) []byte {
	b, _ := json.Marshal(map[string]any{
		"event_id": eventID, "event": event,
		"topic": "conversation.conv-1", "data": data,
	})
	return b
}

func receivedJSON() json.RawMessage {
	return json.RawMessage(`{"message_id":"m-1","conversation_id":"conv-1","sender_id":"u-1","kind":"text","content":"你好世界","sequence":3,"created_at":"2026-11-25T10:00:00Z"}`)
}

func TestConsumer_ReceivedProjectsAndAcks(t *testing.T) {
	idx := NewMemoryIndex()
	consumer, err := NewConsumer(NewProjector(idx), nil)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}
	ch := make(chan Delivery, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx, ch) }()
	fd := &fakeDelivery{data: envelopeJSON("ev-1", "message.received", receivedJSON())}
	ch <- fd.delivery()
	// 等待投影寫入（輪詢，避免時序假設）。
	deadline := time.Now().Add(2 * time.Second)
	for {
		docs, _ := idx.Search(context.Background(), Query{Terms: []string{"你好"}})
		if len(docs) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("投影未寫入，docs = %d", len(docs))
		}
		time.Sleep(10 * time.Millisecond)
	}
	if fd.acked != 1 || fd.naked != 0 {
		t.Fatalf("acked = %d, naked = %d, want 1/0", fd.acked, fd.naked)
	}
	// 同一事件重投：冪等，不得重複寫入（文件仍為 1 筆）。
	fd2 := &fakeDelivery{data: envelopeJSON("ev-1", "message.received", receivedJSON())}
	ch <- fd2.delivery()
	time.Sleep(50 * time.Millisecond)
	docs, _ := idx.Search(context.Background(), Query{Terms: []string{"你好"}})
	if len(docs) != 1 {
		t.Fatalf("重投後 docs = %d, want 1", len(docs))
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run 回傳錯誤: %v", err)
	}
}

func TestConsumer_RecalledRemovesAndAcks(t *testing.T) {
	idx := NewMemoryIndex()
	_ = idx.Upsert(context.Background(), Document{
		MessageID: "m-1", ConversationID: "conv-1", SenderID: "u-1",
		Kind: "text", Content: "你好", SequenceNumber: 1, CreatedAt: time.Now().UTC(),
	}, "ev-seed")
	consumer, err := NewConsumer(NewProjector(idx), nil)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}
	ch := make(chan Delivery, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx, ch) }()
	fd := &fakeDelivery{data: envelopeJSON("ev-2", "message.recalled", json.RawMessage(`{"message_id":"m-1"}`))}
	ch <- fd.delivery()
	deadline := time.Now().Add(2 * time.Second)
	for {
		docs, _ := idx.Search(context.Background(), Query{Terms: []string{"你好"}})
		if len(docs) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("收回後文件仍在，docs = %d", len(docs))
		}
		time.Sleep(10 * time.Millisecond)
	}
	if fd.acked != 1 {
		t.Fatalf("acked = %d, want 1", fd.acked)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run 回傳錯誤: %v", err)
	}
}

func TestConsumer_UnknownEventAckedAndSkipped(t *testing.T) {
	idx := NewMemoryIndex()
	consumer, err := NewConsumer(NewProjector(idx), nil)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}
	ch := make(chan Delivery, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx, ch) }()
	fd := &fakeDelivery{data: envelopeJSON("ev-9", "typing.started", json.RawMessage(`{}`))}
	ch <- fd.delivery()
	time.Sleep(50 * time.Millisecond)
	if fd.acked != 1 || fd.naked != 0 {
		t.Fatalf("acked = %d, naked = %d, want 1/0（未知事件直接 Ack 略過）", fd.acked, fd.naked)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run 回傳錯誤: %v", err)
	}
}

func TestConsumer_BadPayloadNaks(t *testing.T) {
	var reported []error
	var mu sync.Mutex
	consumer, err := NewConsumer(NewProjector(NewMemoryIndex()), func(err error) {
		mu.Lock()
		reported = append(reported, err)
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}
	ch := make(chan Delivery, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx, ch) }()
	badJSON := &fakeDelivery{data: []byte(`{not json`)}
	missingFields := &fakeDelivery{data: envelopeJSON("ev-3", "message.received", json.RawMessage(`{"message_id":""}`))}
	ch <- badJSON.delivery()
	ch <- missingFields.delivery()
	time.Sleep(100 * time.Millisecond)
	if badJSON.naked != 1 || missingFields.naked != 1 {
		t.Fatalf("naked = %d/%d, want 1/1", badJSON.naked, missingFields.naked)
	}
	if badJSON.acked != 0 || missingFields.acked != 0 {
		t.Fatalf("acked = %d/%d, want 0/0", badJSON.acked, missingFields.acked)
	}
	mu.Lock()
	n := len(reported)
	mu.Unlock()
	if n != 2 {
		t.Fatalf("回報錯誤數 = %d, want 2", n)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run 回傳錯誤: %v", err)
	}
}

func TestConsumer_ProjectorErrorNaks(t *testing.T) {
	// 索引後端回傳錯誤時，投遞須 Nak（不得 Ack 假裝已處理）。
	idx := &failingIndex{err: errors.New("search: 索引寫入失敗（測試注入）")}
	consumer, err := NewConsumer(NewProjector(idx), nil)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}
	ch := make(chan Delivery, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx, ch) }()
	fd := &fakeDelivery{data: envelopeJSON("ev-1", "message.received", receivedJSON())}
	ch <- fd.delivery()
	time.Sleep(100 * time.Millisecond)
	if fd.naked != 1 || fd.acked != 0 {
		t.Fatalf("acked = %d, naked = %d, want 0/1", fd.acked, fd.naked)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run 回傳錯誤: %v", err)
	}
}

// failingIndex 是永遠回傳寫入錯誤的 Index 替身。
type failingIndex struct {
	err error
}

func (f *failingIndex) Upsert(context.Context, Document, string) error { return f.err }
func (f *failingIndex) Remove(context.Context, string, string) error   { return f.err }
func (f *failingIndex) Search(context.Context, Query) ([]Document, error) {
	return nil, f.err
}
func (f *failingIndex) ReplaceAll(context.Context, []Document) error { return f.err }

func TestConsumer_ChannelClosedReturnsNil(t *testing.T) {
	consumer, err := NewConsumer(NewProjector(NewMemoryIndex()), nil)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}
	ch := make(chan Delivery)
	close(ch)
	if err := consumer.Run(context.Background(), ch); err != nil {
		t.Fatalf("Run 回傳錯誤: %v", err)
	}
}

func TestConsumer_NilProjectorRejected(t *testing.T) {
	if _, err := NewConsumer(nil, nil); err == nil {
		t.Fatal("nil Projector 未被拒絕")
	}
}

// fakeScanner 是 DocumentScanner 的測試替身。
type fakeScanner struct {
	docs []Document
	err  error
}

func (f *fakeScanner) Scan(_ context.Context, fn func(Document) error) error {
	if f.err != nil {
		return f.err
	}
	for _, d := range f.docs {
		if err := fn(d); err != nil {
			return err
		}
	}
	return nil
}

func TestProjector_RebuildFromScanner(t *testing.T) {
	idx := NewMemoryIndex()
	projector := NewProjector(idx)
	seed := Document{
		MessageID: "m-old", ConversationID: "conv-1", SenderID: "u-1",
		Kind: "text", Content: "舊內容", SequenceNumber: 1, CreatedAt: time.Now().UTC(),
	}
	_ = idx.Upsert(context.Background(), seed, "ev-seed")
	replacement := Document{
		MessageID: "m-new", ConversationID: "conv-1", SenderID: "u-2",
		Kind: "text", Content: "重建內容", SequenceNumber: 1, CreatedAt: time.Now().UTC(),
	}
	scanner := &fakeScanner{docs: []Document{replacement}}
	if err := projector.RebuildFromScanner(context.Background(), scanner); err != nil {
		t.Fatalf("RebuildFromScanner: %v", err)
	}
	docs, _ := idx.Search(context.Background(), Query{Terms: []string{"重建"}})
	if len(docs) != 1 || docs[0].MessageID != "m-new" {
		t.Fatalf("重建後 docs = %+v, want 僅 m-new", docs)
	}
	docs, _ = idx.Search(context.Background(), Query{Terms: []string{"舊"}})
	if len(docs) != 0 {
		t.Fatalf("重建後舊文件仍在，docs = %d", len(docs))
	}
}

func TestProjector_RebuildFromScannerErrors(t *testing.T) {
	projector := NewProjector(NewMemoryIndex())
	if err := projector.RebuildFromScanner(context.Background(), nil); err == nil {
		t.Fatal("nil 掃描來源未被拒絕")
	}
	scanner := &fakeScanner{err: errors.New("scan 失敗（測試注入）")}
	if err := projector.RebuildFromScanner(context.Background(), scanner); err == nil {
		t.Fatal("掃描失敗未被回報")
	}
}
