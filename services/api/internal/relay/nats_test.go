package relay

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/nats-io/nats.go"
)

// fakeJetStream 是 JetStreamPublisher 的測試替身：記錄發佈的
// 訊息與 Msg-Id，不連真實 NATS。
type fakeJetStream struct {
	mu       sync.Mutex
	msgs     []*nats.Msg
	msgIDs   []string
	failNext error
}

func (f *fakeJetStream) PublishMsg(m *nats.Msg, opts ...nats.PubOpt) (*nats.PubAck, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return nil, err
	}
	// MsgId 選項寫入標頭；記錄標頭值以斷言去重鍵正確傳遞。
	f.msgs = append(f.msgs, m)
	f.msgIDs = append(f.msgIDs, m.Header.Get(nats.MsgIdHdr))
	return &nats.PubAck{}, nil
}

func TestJetStreamPublisher_PublishesEnvelope(t *testing.T) {
	js := &fakeJetStream{}
	pub, err := NewJetStreamPublisher(js, "")
	if err != nil {
		t.Fatalf("NewJetStreamPublisher: %v", err)
	}
	err = pub.Publish(context.Background(), Publication{
		EventID: "ev-1",
		Topic:   "conversation.conv-1",
		Event:   "message.received",
		Data:    json.RawMessage(`{"message_id":"m-1"}`),
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if len(js.msgs) != 1 {
		t.Fatalf("發佈數 = %d, want 1", len(js.msgs))
	}
	msg := js.msgs[0]
	if msg.Subject != "ether.events.conversation.conv-1" {
		t.Fatalf("subject = %q, want ether.events.conversation.conv-1", msg.Subject)
	}
	if js.msgIDs[0] != "ev-1" {
		t.Fatalf("Msg-Id = %q, want ev-1", js.msgIDs[0])
	}
	var env Envelope
	if err := json.Unmarshal(msg.Data, &env); err != nil {
		t.Fatalf("envelope 解碼失敗: %v", err)
	}
	if env.EventID != "ev-1" || env.Event != "message.received" || env.Topic != "conversation.conv-1" {
		t.Fatalf("envelope = %+v, 欄位不符", env)
	}
	if string(env.Data) != `{"message_id":"m-1"}` {
		t.Fatalf("envelope data = %s, want 原樣 Payload", env.Data)
	}
}

func TestJetStreamPublisher_ValidationFailsClosed(t *testing.T) {
	js := &fakeJetStream{}
	pub, err := NewJetStreamPublisher(js, "")
	if err != nil {
		t.Fatalf("NewJetStreamPublisher: %v", err)
	}
	cases := []Publication{
		{Topic: "conversation.c", Event: "message.received", Data: json.RawMessage(`{}`)},
		{EventID: "ev", Event: "message.received", Data: json.RawMessage(`{}`)},
		{EventID: "ev", Topic: "conversation.c", Event: "message.received"},
		{EventID: "ev", Topic: "conversation.c", Event: "message.received", Data: json.RawMessage(`{bad`)},
	}
	for i, c := range cases {
		if err := pub.Publish(context.Background(), c); err == nil {
			t.Fatalf("case %d: Publish 未報錯", i)
		}
	}
	if len(js.msgs) != 0 {
		t.Fatalf("發佈數 = %d, want 0（驗證失敗不得發佈）", len(js.msgs))
	}
}

func TestJetStreamPublisher_PublishErrorPropagates(t *testing.T) {
	js := &fakeJetStream{failNext: errors.New("nats: 發佈失敗（測試注入）")}
	pub, err := NewJetStreamPublisher(js, "")
	if err != nil {
		t.Fatalf("NewJetStreamPublisher: %v", err)
	}
	err = pub.Publish(context.Background(), Publication{
		EventID: "ev-1",
		Topic:   "conversation.c",
		Event:   "message.received",
		Data:    json.RawMessage(`{}`),
	})
	if err == nil || !strings.Contains(err.Error(), "JetStream 發佈失敗") {
		t.Fatalf("Publish err = %v, want 包裝後的發佈失敗", err)
	}
}

func TestJetStreamPublisher_NilBackendRejected(t *testing.T) {
	if _, err := NewJetStreamPublisher(nil, ""); err == nil {
		t.Fatal("nil JetStream 後端未被拒絕")
	}
}

func TestJetStreamPublisher_CustomPrefix(t *testing.T) {
	js := &fakeJetStream{}
	pub, err := NewJetStreamPublisher(js, "custom.")
	if err != nil {
		t.Fatalf("NewJetStreamPublisher: %v", err)
	}
	if got := pub.SubjectFor("conversation.c"); got != "custom.conversation.c" {
		t.Fatalf("SubjectFor = %q, want custom.conversation.c", got)
	}
}

func TestDialNATS_EmptyURLRejected(t *testing.T) {
	if _, _, err := DialNATS(""); err == nil {
		t.Fatal("空 NATS 位址未被拒絕")
	}
	if _, _, err := DialNATS("   "); err == nil {
		t.Fatal("空白 NATS 位址未被拒絕")
	}
}

func TestDialNATS_UnreachableRejected(t *testing.T) {
	// 127.0.0.1 未監聽的埠：連線須明確失敗，不得回傳半建好的發佈者。
	if _, _, err := DialNATS("nats://127.0.0.1:14222"); err == nil {
		t.Fatal("無法連線的 NATS 位址未被拒絕")
	}
}
