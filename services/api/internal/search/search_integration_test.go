//go:build integration

package search

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/nats-io/nats.go"
)

// TestMeilisearchIndex_Live 驗證轉接器對真實 Meilisearch 的
// 端到端行為：寫入、查詢、刪除。須設定
// ETHER_TEST_MEILISEARCH_URL 與 ETHER_TEST_MEILISEARCH_API_KEY
// （隔離環境），未設定時跳過、標示環境待驗證。
func TestMeilisearchIndex_Live(t *testing.T) {
	host := os.Getenv("ETHER_TEST_MEILISEARCH_URL")
	key := os.Getenv("ETHER_TEST_MEILISEARCH_API_KEY")
	if host == "" || key == "" {
		t.Skip("ETHER_TEST_MEILISEARCH_URL／KEY 未設定：整合測試待隔離環境驗證")
	}
	uid := "messages_test_" + time.Now().Format("20060102150405")
	idx, err := DialMeilisearch(host, key, uid)
	if err != nil {
		t.Fatalf("DialMeilisearch: %v", err)
	}
	client := meilisearch.New(host, meilisearch.WithAPIKey(key))
	defer func() {
		if _, err := client.DeleteIndex(context.Background(), uid); err != nil {
			t.Logf("清理測試索引失敗: %v", err)
		}
	}()
	ctx := context.Background()
	doc := Document{
		MessageID: "m-live-1", ConversationID: "conv-live", SenderID: "u-1",
		Kind: "text", Content: "整合測試內容", SequenceNumber: 1, CreatedAt: time.Now().UTC(),
	}
	if err := idx.Upsert(ctx, doc, "ev-live-1"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	docs, err := idx.Search(ctx, Query{Terms: []string{"整合"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(docs) != 1 || docs[0].MessageID != "m-live-1" {
		t.Fatalf("Search = %+v, want 1 筆 m-live-1", docs)
	}
	if err := idx.Remove(ctx, "m-live-1", "ev-live-2"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	docs, err = idx.Search(ctx, Query{Terms: []string{"整合"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("刪除後 Search = %d 筆, want 0", len(docs))
	}
}

// TestConsumer_LiveNATS 驗證消費迴圈對真實 NATS 的端到端行為：
// 發佈的信封經訂閱進入 Projector、寫入記憶體索引。須設定
// ETHER_TEST_NATS_URL（隔離環境），未設定時跳過。
func TestConsumer_LiveNATS(t *testing.T) {
	url := os.Getenv("ETHER_TEST_NATS_URL")
	if url == "" {
		t.Skip("ETHER_TEST_NATS_URL 未設定：整合測試待隔離環境驗證")
	}
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("連線 NATS: %v", err)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("JetStream: %v", err)
	}
	stream := "TEST_ETHER_SEARCH_" + time.Now().Format("20060102150405")
	if _, err := js.AddStream(&nats.StreamConfig{
		Name:     stream,
		Subjects: []string{DefaultConsumerSubject},
	}); err != nil {
		t.Fatalf("AddStream: %v", err)
	}
	defer func() { _ = js.DeleteStream(stream) }()

	idx := NewMemoryIndex()
	consumer, err := NewConsumer(NewProjector(idx), nil)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}
	durable := DefaultConsumerDurable + "_test_" + time.Now().Format("20060102150405")
	sub, err := NewNATSSubscription(nc, DefaultConsumerSubject, durable, durable)
	if err != nil {
		t.Fatalf("NewNATSSubscription: %v", err)
	}
	defer func() { _ = sub.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx, sub.Channel()) }()

	payload, _ := json.Marshal(map[string]any{
		"event_id": "ev-live-1", "event": "message.received",
		"topic": "conversation.conv-1",
		"data": map[string]any{
			"message_id": "m-live-1", "conversation_id": "conv-1", "sender_id": "u-1",
			"kind": "text", "content": "即時索引內容", "sequence": 1,
			"created_at": time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err := nc.Publish("ether.events.conversation.conv-1", payload); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		docs, _ := idx.Search(context.Background(), Query{Terms: []string{"即時"}})
		if len(docs) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("消費後索引 docs = %d, want 1", len(docs))
		}
		time.Sleep(200 * time.Millisecond)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run 回傳錯誤: %v", err)
	}
}
