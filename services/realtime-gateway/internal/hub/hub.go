// Package hub 是 Realtime Gateway 的連線中樞（Phase 05）。
//
// 責任：WebSocket 連線認證（OIDC Access Token，查詢參數
// access_token——WEBSOCKET_PROTOCOL.md §3 稱傳遞方式待定稿，本
// 實作採查詢參數，Token 不得記錄於日誌）、Session Registry
// （行程內，連線中斷即移除；Valkey 持久化待定稿）、訂閱管理、
// 事件扇出（經 broker.Broker）、typing 轉發、錯誤回覆。
//
// 明確未實作：WebSocket 送出訊息（message.send 一律回錯誤，
// 請改用 REST；寫入路徑唯一，見 DATA_FLOW.md §3）；Heartbeat
// 間隔／重連退避（待定稿，本實作不主動發 Ping）；Rate Limit／
// Backpressure（待定稿）；裝置撤銷即時斷線（待 Keycloak 事件同步，
// 見 IDENTITY 缺口 12）。訂閱與 typing 的成員身份驗證已實作
// （MembershipChecker 介面注入；checker 為 nil 時 fail-closed，
// 訂閱與 typing 一律拒絕；api 側的真實成員查詢實作待後續整合，
// 不得由 Gateway 直讀 Chat 資料表）。
// 事件信封欄位依 WEBSOCKET_PROTOCOL.md §5.1（草案，待定稿）。
package hub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"ether.local/realtime-gateway/internal/auth"
	"ether.local/realtime-gateway/internal/broker"
	"ether.local/realtime-gateway/internal/ws"
)

// Verifier 是連線認證介面（OIDC 驗證器，與 api 服務同一規則）。
type Verifier interface {
	Verify(ctx context.Context, token string) (auth.Identity, error)
}

// MembershipChecker 是對話成員身份查詢介面（訂閱與 typing 的
// 驗證用）。實作須以後端認證身份為準：未知對話與非成員一律回
// (false, nil)，不得以錯誤或不同回應洩漏對話存在性；查詢失敗
// 回傳錯誤，Hub 以 fail-closed 拒絕訂閱／轉發。Gateway 不得
// 直讀 Chat 資料表：真實實作須經 api 整合（內部端點或事件投影，
// 設計待定稿），本套件只定介面。
type MembershipChecker interface {
	IsMember(ctx context.Context, userID, conversationID string) (bool, error)
}

// Envelope 是 WebSocket 事件信封（WEBSOCKET_PROTOCOL.md §5.1 草案）。
type Envelope struct {
	EventID       string          `json:"event_id"`
	Event         string          `json:"event"`
	Version       string          `json:"version"`
	Timestamp     string          `json:"timestamp"`
	CorrelationID string          `json:"correlation_id"`
	Sequence      *int64          `json:"sequence,omitempty"`
	SenderID      string          `json:"sender_id,omitempty"`
	Data          json.RawMessage `json:"data"`
	Error         *EnvelopeError  `json:"error,omitempty"`
}

// EnvelopeError 是信封內的錯誤物件（格式見 ERROR_CODES.md §5）。
type EnvelopeError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// ProtocolVersion 是本實作送出的事件協定版本（草案形式，待定稿）。
const ProtocolVersion = "1"

// Hub 管理全部 WebSocket 連線與訂閱。
type Hub struct {
	broker   broker.Broker
	verifier Verifier
	checker  MembershipChecker
	upgrader ws.Upgrader
	now      func() time.Time

	mu    sync.Mutex
	conns map[*clientConn]struct{}
}

// NewHub 建立 Hub。verifier 為 nil 時連線一律回 503
// （fail-closed，不得建立未認證連線）。checker 為 nil 時訂閱與
// typing 一律拒絕（fail-closed，不得在無成員驗證時放行）；
// 呼叫方須於啟動日誌明示此狀態。
func NewHub(b broker.Broker, verifier Verifier, checker MembershipChecker, now func() time.Time) *Hub {
	if now == nil {
		now = time.Now
	}
	return &Hub{
		broker:   b,
		verifier: verifier,
		checker:  checker,
		conns:    map[*clientConn]struct{}{},
		now:      now,
	}
}

type clientConn struct {
	hub       *Hub
	conn      *ws.Conn
	id        string
	userID    string
	sessionID string
	writeMu   sync.Mutex

	mu   sync.Mutex
	subs map[string]func()
	done chan struct{}
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unavailable"
	}
	return hex.EncodeToString(b[:])
}

// ConnectionCount 回傳目前線上連線數（監控用）。
func (h *Hub) ConnectionCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.conns)
}

// Close 關閉全部連線（優雅關停時呼叫）。
func (h *Hub) Close() {
	h.mu.Lock()
	conns := make([]*clientConn, 0, len(h.conns))
	for c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	for _, c := range conns {
		c.close()
	}
}

// Handle 升級 HTTP 請求為 WebSocket 連線並接管至斷線。
// 認證失敗時回寫 HTTP 錯誤，不得建立未認證連線。
func (h *Hub) Handle(w http.ResponseWriter, r *http.Request) {
	if h.verifier == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	// Token 只取自查詢參數，不得記錄於日誌（WEBSOCKET_PROTOCOL.md §3）。
	token := r.URL.Query().Get("access_token")
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	identity, err := h.verifier.Verify(r.Context(), token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := h.upgrader.Upgrade(w, r)
	if err != nil {
		return
	}
	c := &clientConn{
		hub:       h,
		conn:      conn,
		id:        newID(),
		userID:    identity.Subject,
		sessionID: newID(),
		subs:      map[string]func(){},
		done:      make(chan struct{}),
	}
	h.mu.Lock()
	h.conns[c] = struct{}{}
	h.mu.Unlock()

	ready, _ := json.Marshal(map[string]any{
		"session_id":         c.sessionID,
		"server_time":        h.now().UTC().Format(time.RFC3339),
		"supported_versions": []string{ProtocolVersion},
	})
	_ = c.write(h.newEnvelope("connection.ready", "", identity.Subject, ready, nil))

	h.serveConn(c)
}

func (h *Hub) newEnvelope(event, correlationID, senderID string, data json.RawMessage, err *EnvelopeError) []byte {
	if data == nil {
		data = json.RawMessage("{}")
	}
	env := Envelope{
		EventID:       newID(),
		Event:         event,
		Version:       ProtocolVersion,
		Timestamp:     h.now().UTC().Format(time.RFC3339),
		CorrelationID: correlationID,
		SenderID:      senderID,
		Data:          data,
		Error:         err,
	}
	b, _ := json.Marshal(env)
	return b
}

func (h *Hub) unregister(c *clientConn) {
	h.mu.Lock()
	delete(h.conns, c)
	h.mu.Unlock()
}

func (c *clientConn) write(payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteMessage(payload)
}

func (c *clientConn) close() {
	select {
	case <-c.done:
		return
	default:
	}
	close(c.done)
	c.mu.Lock()
	for _, unsub := range c.subs {
		unsub()
	}
	c.subs = map[string]func(){}
	c.mu.Unlock()
	c.hub.unregister(c)
	_ = c.conn.WriteClose(ws.CloseNormal, "")
	_ = c.conn.Close()
}

// serveConn 讀取客戶端事件並分發，至連線中斷。
func (h *Hub) serveConn(c *clientConn) {
	defer c.close()
	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			if err == io.EOF {
				return
			}
			return
		}
		h.dispatch(c, payload)
	}
}

// dispatch 解析一則客戶端信封並分發。
func (h *Hub) dispatch(c *clientConn, payload []byte) {
	var env Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		c.write(h.newEnvelope("connection.error", "", "", nil, &EnvelopeError{
			Code:      "ETHER_VALIDATION_INVALID_REQUEST",
			Message:   "事件格式錯誤。",
			RequestID: newID(),
		}))
		return
	}
	if env.Version != "" && env.Version != ProtocolVersion {
		c.write(h.newEnvelope(env.Event, env.CorrelationID, "", nil, &EnvelopeError{
			Code:      "ETHER_VALIDATION_INVALID_REQUEST",
			Message:   "不支援的事件協定版本。",
			RequestID: newID(),
		}))
		return
	}
	switch env.Event {
	case "subscribe", "unsubscribe":
		h.handleSubscribe(c, env)
	case "typing.start", "typing.stop":
		h.handleTyping(c, env)
	case "message.send":
		// 寫入路徑唯一（REST）：WebSocket 送出尚未接線，不得
		// 假裝寫入成功。錯誤回覆含 correlation_id 供客戶端對帳。
		c.write(h.newEnvelope(env.Event, env.CorrelationID, "", nil, &EnvelopeError{
			Code:      "ETHER_COMMON_UNAVAILABLE",
			Message:   "請改用 REST 送出訊息。",
			RequestID: newID(),
		}))
	default:
		c.write(h.newEnvelope(env.Event, env.CorrelationID, "", nil, &EnvelopeError{
			Code:      "ETHER_VALIDATION_INVALID_REQUEST",
			Message:   "未知的事件。",
			RequestID: newID(),
		}))
	}
}

type subscribeData struct {
	ConversationID string `json:"conversation_id"`
}

// checkMembership 驗證連線使用者是否為對話成員。回傳值：
// allowed 為成員身份是否成立；unavailable 為驗證服務是否不可用
// （checker 為 nil 或查詢失敗，fail-closed）。非成員與未知對話
// 皆為 allowed=false，不得區分（不洩漏存在性）。
func (h *Hub) checkMembership(ctx context.Context, userID, conversationID string) (allowed, unavailable bool) {
	if h.checker == nil {
		return false, true
	}
	ok, err := h.checker.IsMember(ctx, userID, conversationID)
	if err != nil {
		return false, true
	}
	return ok, false
}

// membershipErrorEnvelope 依驗證結果寫出訂閱／typing 的拒絕回覆。
func (h *Hub) membershipErrorEnvelope(c *clientConn, env Envelope, unavailable bool) {
	if unavailable {
		c.write(h.newEnvelope(env.Event, env.CorrelationID, "", nil, &EnvelopeError{
			Code:      "ETHER_COMMON_UNAVAILABLE",
			Message:   "訂閱服務暫時無法使用。",
			RequestID: newID(),
		}))
		return
	}
	c.write(h.newEnvelope(env.Event, env.CorrelationID, "", nil, &EnvelopeError{
		Code:      "ETHER_AUTH_FORBIDDEN",
		Message:   "無權訂閱此對話。",
		RequestID: newID(),
	}))
}

func (h *Hub) handleSubscribe(c *clientConn, env Envelope) {
	var data subscribeData
	if err := json.Unmarshal(env.Data, &data); err != nil || data.ConversationID == "" {
		c.write(h.newEnvelope(env.Event, env.CorrelationID, "", nil, &EnvelopeError{
			Code:      "ETHER_VALIDATION_INVALID_REQUEST",
			Message:   "conversation_id 不可為空。",
			RequestID: newID(),
		}))
		return
	}
	topic := "conversation." + data.ConversationID
	// 訂閱須驗證成員身份（WEBSOCKET_PROTOCOL.md §6.1.1／§6.2.1）：
	// 非成員與未知對話一律拒絕，不得靜默訂閱、不得洩漏存在性。
	// 驗證在持有訂閱鎖之前執行（checker 可能涉及網路呼叫），且
	// 含重複訂閱路徑（成員身份可能在訂閱後變更，不得以舊訂閱
	// 假裝身份仍有效）。
	if env.Event == "subscribe" {
		if allowed, unavailable := h.checkMembership(context.Background(), c.userID, data.ConversationID); !allowed {
			h.membershipErrorEnvelope(c, env, unavailable)
			return
		}
	}
	c.mu.Lock()
	if env.Event == "unsubscribe" {
		if unsub, ok := c.subs[topic]; ok {
			unsub()
			delete(c.subs, topic)
		}
		c.mu.Unlock()
		ack, _ := json.Marshal(map[string]any{
			"conversation_id": data.ConversationID,
			"subscribed":      false,
		})
		_ = c.write(h.newEnvelope(env.Event, env.CorrelationID, c.userID, ack, nil))
		return
	}
	if _, ok := c.subs[topic]; ok {
		c.mu.Unlock()
		ack, _ := json.Marshal(map[string]any{
			"conversation_id": data.ConversationID,
			"subscribed":      true,
		})
		_ = c.write(h.newEnvelope(env.Event, env.CorrelationID, c.userID, ack, nil))
		return
	}
	c.mu.Unlock()
	ch, unsub, err := h.broker.Subscribe(topic)
	if err != nil {
		c.write(h.newEnvelope(env.Event, env.CorrelationID, "", nil, &EnvelopeError{
			Code:      "ETHER_COMMON_UNAVAILABLE",
			Message:   "訂閱失敗。",
			RequestID: newID(),
		}))
		return
	}
	c.mu.Lock()
	c.subs[topic] = unsub
	c.mu.Unlock()
	go c.forward(ch)
	ack, _ := json.Marshal(map[string]any{
		"conversation_id": data.ConversationID,
		"subscribed":      true,
	})
	_ = c.write(h.newEnvelope(env.Event, env.CorrelationID, c.userID, ack, nil))
}

// forward 將 Broker 事件投遞給此連線（跳過自己發起的事件）。
func (c *clientConn) forward(ch <-chan broker.Event) {
	for {
		select {
		case <-c.done:
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if ev.FromConnID == c.id {
				continue
			}
			if err := c.write(ev.Payload); err != nil {
				return
			}
		}
	}
}

func (h *Hub) handleTyping(c *clientConn, env Envelope) {
	var data map[string]any
	if err := json.Unmarshal(env.Data, &data); err != nil {
		c.write(h.newEnvelope(env.Event, env.CorrelationID, "", nil, &EnvelopeError{
			Code:      "ETHER_VALIDATION_INVALID_REQUEST",
			Message:   "事件格式錯誤。",
			RequestID: newID(),
		}))
		return
	}
	convID, _ := data["conversation_id"].(string)
	if convID == "" {
		c.write(h.newEnvelope(env.Event, env.CorrelationID, "", nil, &EnvelopeError{
			Code:      "ETHER_VALIDATION_INVALID_REQUEST",
			Message:   "conversation_id 不可為空。",
			RequestID: newID(),
		}))
		return
	}
	// typing 轉發前同樣驗證成員身份：非成員的輸入中狀態不得
	// 轉發給對話成員（WEBSOCKET_PROTOCOL.md §6.2.1）。
	if allowed, unavailable := h.checkMembership(context.Background(), c.userID, convID); !allowed {
		h.membershipErrorEnvelope(c, env, unavailable)
		return
	}
	// sender_id 與 user_id 一律由伺服器以認證身份填寫，不得
	// 接受客戶端自填（WEBSOCKET_PROTOCOL.md §5.1）。
	data["user_id"] = c.userID
	body, _ := json.Marshal(data)
	payload := h.newEnvelope(env.Event, env.CorrelationID, c.userID, body, nil)
	_ = h.broker.Publish(context.Background(), broker.Event{
		Topic:      "conversation." + convID,
		Payload:    payload,
		FromConnID: c.id,
	})
}
