//go:build integration

package relay

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

// TestJetStreamPublisher_LiveNATS 驗證發佈者對真實 NATS
// JetStream 的端到端行為：訊息可被訂閱者收到、信封格式與
// Msg-Id 標頭正確。須設定 ETHER_TEST_NATS_URL（隔離環境），
// 未設定時跳過、標示環境待驗證。
func TestJetStreamPublisher_LiveNATS(t *testing.T) {
	url := os.Getenv("ETHER_TEST_NATS_URL")
	if url == "" {
		t.Skip("ETHER_TEST_NATS_URL 未設定：整合測試待隔離環境驗證")
	}
	nc, js, err := DialNATS(url)
	if err != nil {
		t.Fatalf("DialNATS: %v", err)
	}
	defer nc.Close()

	stream := "TEST_ETHER_EVENTS_" + time.Now().Format("20060102150405")
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     stream,
		Subjects: []string{DefaultSubjectPrefix + ">"},
	})
	if err != nil {
		t.Fatalf("AddStream: %v", err)
	}
	defer func() { _ = js.DeleteStream(stream) }()

	sub, err := js.SubscribeSync(DefaultSubjectPrefix + "conversation.conv-1")
	if err != nil {
		t.Fatalf("SubscribeSync: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	pub, err := NewJetStreamPublisher(js, "")
	if err != nil {
		t.Fatalf("NewJetStreamPublisher: %v", err)
	}
	if err := pub.Publish(context.Background(), Publication{
		EventID: "ev-live-1",
		Topic:   "conversation.conv-1",
		Event:   "message.received",
		Data:    json.RawMessage(`{"message_id":"m-1"}`),
	}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	msg, err := sub.NextMsg(10 * time.Second)
	if err != nil {
		t.Fatalf("NextMsg: %v", err)
	}
	if got := msg.Header.Get(nats.MsgIdHdr); got != "ev-live-1" {
		t.Fatalf("Msg-Id = %q, want ev-live-1", got)
	}
	var env Envelope
	if err := json.Unmarshal(msg.Data, &env); err != nil {
		t.Fatalf("信封解碼失敗: %v", err)
	}
	if env.EventID != "ev-live-1" || env.Event != "message.received" || string(env.Data) != `{"message_id":"m-1"}` {
		t.Fatalf("信封 = %+v, 欄位不符", env)
	}
}
