package hub

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ether.local/realtime-gateway/internal/auth"
	"ether.local/realtime-gateway/internal/broker"
)

type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, token string) (auth.Identity, error) {
	switch token {
	case "good-token":
		return auth.Identity{Subject: "user-1"}, nil
	case "good-token-2":
		return auth.Identity{Subject: "user-2"}, nil
	default:
		return auth.Identity{}, auth.ErrInvalidToken
	}
}

// rawClient 是測試用的最小 WebSocket 客戶端（與 ws 套件測試
// 相同的手動交握／遮罩實作，hub 測試自帶一份，不得 import
// 測試程式碼）。
type rawClient struct {
	conn   net.Conn
	reader *bufio.Reader
}

func dialRaw(t *testing.T, url, token string) *rawClient {
	t.Helper()
	addr := strings.TrimPrefix(url, "http://")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		t.Fatalf("rand: %v", err)
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)
	target := "/ws/v1?access_token=" + token
	req := "GET " + target + " HTTP/1.1\r\n" +
		"Host: " + addr + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}
	reader := bufio.NewReader(conn)
	status, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if !strings.Contains(status, "101") {
		t.Fatalf("status = %q, want 101", status)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read headers: %v", err)
		}
		if line == "\r\n" {
			break
		}
	}
	return &rawClient{conn: conn, reader: reader}
}

func (c *rawClient) send(payload []byte) error {
	mask := [4]byte{0x11, 0x22, 0x33, 0x44}
	header := []byte{0x81}
	n := len(payload)
	switch {
	case n <= 125:
		header = append(header, 0x80|byte(n))
	case n <= 65535:
		header = append(header, 0x80|126, byte(n>>8), byte(n))
	default:
		header = append(header, 0x80|127)
		var ext [8]byte
		binary.BigEndian.PutUint64(ext[:], uint64(n))
		header = append(header, ext[:]...)
	}
	header = append(header, mask[:]...)
	masked := make([]byte, n)
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}
	if _, err := c.conn.Write(header); err != nil {
		return err
	}
	_, err := c.conn.Write(masked)
	return err
}

func (c *rawClient) readEnvelope(t *testing.T) Envelope {
	t.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		t.Fatalf("read header: %v", err)
	}
	op := header[0] & 0x0f
	length := int64(header[1] & 0x7f)
	switch length {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(c.reader, ext[:]); err != nil {
			t.Fatalf("read ext: %v", err)
		}
		length = int64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(c.reader, ext[:]); err != nil {
			t.Fatalf("read ext: %v", err)
		}
		length = int64(binary.BigEndian.Uint64(ext[:]))
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(c.reader, payload); err != nil {
		t.Fatalf("read payload: %v", err)
	}
	if op == 0x9 {
		// 伺服器不應主動發 Ping；若收到，視為測試失敗。
		t.Fatalf("unexpected ping frame")
	}
	var env Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	return env
}

func testHub() (*Hub, *broker.MemoryBroker) {
	b := broker.NewMemoryBroker()
	h := NewHub(b, fakeVerifier{}, allowAllChecker{}, nil)
	return h, b
}

// allowAllChecker 是測試用成員檢查器：一律允許（供既有扇出／
// typing 測試，成員驗證本身由下方專項測試覆蓋）。
type allowAllChecker struct{}

func (allowAllChecker) IsMember(context.Context, string, string) (bool, error) {
	return true, nil
}

// mapChecker 是測試用成員檢查器：以 userID＋對話識別碼查表；
// err 非空時一律回傳錯誤（查詢失敗路徑）。
type mapChecker struct {
	members map[string]bool
	err     error
}

func (m mapChecker) IsMember(_ context.Context, userID, conversationID string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.members[userID+"\x00"+conversationID], nil
}

func TestHandle_RejectsUnauthenticated(t *testing.T) {
	h, _ := testHub()
	server := httptest.NewServer(http.HandlerFunc(h.Handle))
	defer server.Close()

	resp, err := http.Get(server.URL + "/ws/v1")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("無 Token status = %d, want 401", resp.StatusCode)
	}
	resp, err = http.Get(server.URL + "/ws/v1?access_token=bad")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("錯誤 Token status = %d, want 401", resp.StatusCode)
	}
}

func TestHandle_NilVerifierFailsClosed(t *testing.T) {
	b := broker.NewMemoryBroker()
	h := NewHub(b, nil, nil, nil)
	server := httptest.NewServer(http.HandlerFunc(h.Handle))
	defer server.Close()
	resp, err := http.Get(server.URL + "/ws/v1?access_token=good-token")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("無驗證器 status = %d, want 503", resp.StatusCode)
	}
}

func TestHandle_ReadyAndSubscribeFanout(t *testing.T) {
	h, b := testHub()
	server := httptest.NewServer(http.HandlerFunc(h.Handle))
	defer server.Close()

	c1 := dialRaw(t, server.URL, "good-token")
	defer func() { _ = c1.conn.Close() }()
	ready := c1.readEnvelope(t)
	if ready.Event != "connection.ready" {
		t.Fatalf("首則事件 = %q, want connection.ready", ready.Event)
	}
	var readyData struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(ready.Data, &readyData); err != nil || readyData.SessionID == "" {
		t.Fatalf("ready data = %s", ready.Data)
	}

	c2 := dialRaw(t, server.URL, "good-token-2")
	defer func() { _ = c2.conn.Close() }()
	_ = c2.readEnvelope(t) // connection.ready

	subscribe := func(c *rawClient, conv string) {
		body, _ := json.Marshal(map[string]any{
			"event_id": "e-1", "event": "subscribe", "version": "1",
			"timestamp": "2026-11-25T00:00:00Z", "correlation_id": "c-1",
			"data": map[string]any{"conversation_id": conv},
		})
		if err := c.send(body); err != nil {
			t.Fatalf("subscribe send: %v", err)
		}
		ack := c.readEnvelope(t)
		if ack.Event != "subscribe" || ack.Error != nil {
			t.Fatalf("subscribe ack = %+v", ack)
		}
	}
	subscribe(c1, "conv-1")
	subscribe(c2, "conv-1")

	// 經 Broker 發佈 message.received：兩名訂閱者皆收到。
	payload, _ := json.Marshal(map[string]any{
		"event_id": "ev-1", "event": "message.received", "version": "1",
		"timestamp": "2026-11-25T00:00:00Z", "correlation_id": "",
		"data": map[string]any{"message_id": "m-1", "conversation_id": "conv-1"},
	})
	if err := b.Publish(context.Background(), broker.Event{Topic: "conversation.conv-1", Payload: payload}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	for i, c := range []*rawClient{c1, c2} {
		env := c.readEnvelope(t)
		if env.Event != "message.received" {
			t.Fatalf("連線 %d 收到事件 = %q", i, env.Event)
		}
	}
	if h.ConnectionCount() != 2 {
		t.Fatalf("連線數 = %d, want 2", h.ConnectionCount())
	}
}

func TestHandle_TypingForwardedExceptSender(t *testing.T) {
	h, _ := testHub()
	server := httptest.NewServer(http.HandlerFunc(h.Handle))
	defer server.Close()

	c1 := dialRaw(t, server.URL, "good-token")
	defer func() { _ = c1.conn.Close() }()
	_ = c1.readEnvelope(t)
	c2 := dialRaw(t, server.URL, "good-token-2")
	defer func() { _ = c2.conn.Close() }()
	_ = c2.readEnvelope(t)

	subscribe := func(c *rawClient) {
		body, _ := json.Marshal(map[string]any{
			"event_id": "e-1", "event": "subscribe", "version": "1",
			"timestamp": "2026-11-25T00:00:00Z", "correlation_id": "",
			"data": map[string]any{"conversation_id": "conv-9"},
		})
		if err := c.send(body); err != nil {
			t.Fatalf("subscribe: %v", err)
		}
		_ = c.readEnvelope(t)
	}
	subscribe(c1)
	subscribe(c2)

	body, _ := json.Marshal(map[string]any{
		"event_id": "e-2", "event": "typing.start", "version": "1",
		"timestamp": "2026-11-25T00:00:00Z", "correlation_id": "",
		"data": map[string]any{"conversation_id": "conv-9", "user_id": "forged-user"},
	})
	if err := c1.send(body); err != nil {
		t.Fatalf("typing send: %v", err)
	}
	env := c2.readEnvelope(t)
	if env.Event != "typing.start" {
		t.Fatalf("收到事件 = %q", env.Event)
	}
	if env.SenderID != "user-1" {
		t.Fatalf("sender_id = %q, want user-1（伺服器填寫，不得信任客戶端）", env.SenderID)
	}
	var data map[string]any
	if err := json.Unmarshal(env.Data, &data); err != nil || data["user_id"] != "user-1" {
		t.Fatalf("typing data = %s", env.Data)
	}
	// c1 不應收到自己發起的 typing（FromConnID 過濾）。
	_ = c1.conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	header := make([]byte, 2)
	if _, err := io.ReadFull(c1.reader, header); err == nil {
		t.Fatal("發送端不應收到自己的 typing 事件")
	}
}

func TestHandle_MessageSendRejected(t *testing.T) {
	h, _ := testHub()
	server := httptest.NewServer(http.HandlerFunc(h.Handle))
	defer server.Close()

	c1 := dialRaw(t, server.URL, "good-token")
	defer func() { _ = c1.conn.Close() }()
	_ = c1.readEnvelope(t)
	body, _ := json.Marshal(map[string]any{
		"event_id": "e-9", "event": "message.send", "version": "1",
		"timestamp": "2026-11-25T00:00:00Z", "correlation_id": "req-1",
		"data": map[string]any{"conversation_id": "conv-1", "content": "x"},
	})
	if err := c1.send(body); err != nil {
		t.Fatalf("send: %v", err)
	}
	env := c1.readEnvelope(t)
	if env.Error == nil || env.Error.Code != "ETHER_COMMON_UNAVAILABLE" || env.CorrelationID != "req-1" {
		t.Fatalf("message.send 回覆 = %+v", env)
	}
}

func sendEnvelope(t *testing.T, c *rawClient, event, conv string, extra map[string]any) {
	t.Helper()
	data := map[string]any{"conversation_id": conv}
	for k, v := range extra {
		data[k] = v
	}
	body, _ := json.Marshal(map[string]any{
		"event_id": "e-1", "event": event, "version": "1",
		"timestamp": "2026-11-25T00:00:00Z", "correlation_id": "c-1",
		"data": data,
	})
	if err := c.send(body); err != nil {
		t.Fatalf("%s send: %v", event, err)
	}
}

func expectNoMessage(t *testing.T, c *rawClient) {
	t.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.reader, header); err == nil {
		t.Fatal("不應收到任何事件")
	}
}

func TestHandle_SubscribeDeniedForNonMember(t *testing.T) {
	b := broker.NewMemoryBroker()
	checker := mapChecker{members: map[string]bool{"user-1\x00conv-1": true}}
	h := NewHub(b, fakeVerifier{}, checker, nil)
	server := httptest.NewServer(http.HandlerFunc(h.Handle))
	defer server.Close()

	c1 := dialRaw(t, server.URL, "good-token")
	defer func() { _ = c1.conn.Close() }()
	_ = c1.readEnvelope(t)
	c2 := dialRaw(t, server.URL, "good-token-2")
	defer func() { _ = c2.conn.Close() }()
	_ = c2.readEnvelope(t)

	sendEnvelope(t, c1, "subscribe", "conv-1", nil)
	if ack := c1.readEnvelope(t); ack.Error != nil {
		t.Fatalf("成員訂閱應成功：%+v", ack)
	}
	sendEnvelope(t, c2, "subscribe", "conv-1", nil)
	denied := c2.readEnvelope(t)
	if denied.Error == nil || denied.Error.Code != "ETHER_AUTH_FORBIDDEN" {
		t.Fatalf("非成員訂閱回覆 = %+v, want ETHER_AUTH_FORBIDDEN", denied)
	}

	// 非成員不得收到扇出事件。
	payload, _ := json.Marshal(map[string]any{
		"event_id": "ev-1", "event": "message.received", "version": "1",
		"timestamp": "2026-11-25T00:00:00Z", "correlation_id": "",
		"data": map[string]any{"message_id": "m-1", "conversation_id": "conv-1"},
	})
	if err := b.Publish(context.Background(), broker.Event{Topic: "conversation.conv-1", Payload: payload}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if env := c1.readEnvelope(t); env.Event != "message.received" {
		t.Fatalf("成員收到事件 = %q", env.Event)
	}
	expectNoMessage(t, c2)
}

func TestHandle_SubscribeNilCheckerFailsClosed(t *testing.T) {
	b := broker.NewMemoryBroker()
	h := NewHub(b, fakeVerifier{}, nil, nil)
	server := httptest.NewServer(http.HandlerFunc(h.Handle))
	defer server.Close()

	c1 := dialRaw(t, server.URL, "good-token")
	defer func() { _ = c1.conn.Close() }()
	_ = c1.readEnvelope(t)
	sendEnvelope(t, c1, "subscribe", "conv-1", nil)
	env := c1.readEnvelope(t)
	if env.Error == nil || env.Error.Code != "ETHER_COMMON_UNAVAILABLE" {
		t.Fatalf("無 checker 時訂閱回覆 = %+v, want ETHER_COMMON_UNAVAILABLE", env)
	}
	sendEnvelope(t, c1, "typing.start", "conv-1", nil)
	env = c1.readEnvelope(t)
	if env.Error == nil || env.Error.Code != "ETHER_COMMON_UNAVAILABLE" {
		t.Fatalf("無 checker 時 typing 回覆 = %+v, want ETHER_COMMON_UNAVAILABLE", env)
	}
}

func TestHandle_TypingDeniedForNonMember(t *testing.T) {
	b := broker.NewMemoryBroker()
	checker := mapChecker{members: map[string]bool{"user-1\x00conv-1": true}}
	h := NewHub(b, fakeVerifier{}, checker, nil)
	server := httptest.NewServer(http.HandlerFunc(h.Handle))
	defer server.Close()

	c1 := dialRaw(t, server.URL, "good-token")
	defer func() { _ = c1.conn.Close() }()
	_ = c1.readEnvelope(t)
	c2 := dialRaw(t, server.URL, "good-token-2")
	defer func() { _ = c2.conn.Close() }()
	_ = c2.readEnvelope(t)

	sendEnvelope(t, c1, "subscribe", "conv-1", nil)
	_ = c1.readEnvelope(t)
	sendEnvelope(t, c2, "typing.start", "conv-1", map[string]any{"user_id": "user-2"})
	denied := c2.readEnvelope(t)
	if denied.Error == nil || denied.Error.Code != "ETHER_AUTH_FORBIDDEN" {
		t.Fatalf("非成員 typing 回覆 = %+v, want ETHER_AUTH_FORBIDDEN", denied)
	}
	expectNoMessage(t, c1)
}

func TestHandle_CheckerErrorFailsClosed(t *testing.T) {
	b := broker.NewMemoryBroker()
	checker := mapChecker{err: errors.New("checker: 查詢失敗（測試注入）")}
	h := NewHub(b, fakeVerifier{}, checker, nil)
	server := httptest.NewServer(http.HandlerFunc(h.Handle))
	defer server.Close()

	c1 := dialRaw(t, server.URL, "good-token")
	defer func() { _ = c1.conn.Close() }()
	_ = c1.readEnvelope(t)
	sendEnvelope(t, c1, "subscribe", "conv-1", nil)
	env := c1.readEnvelope(t)
	if env.Error == nil || env.Error.Code != "ETHER_COMMON_UNAVAILABLE" {
		t.Fatalf("checker 失敗時訂閱回覆 = %+v, want ETHER_COMMON_UNAVAILABLE", env)
	}
}
