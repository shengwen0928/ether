// Package ws 是 Realtime Gateway 的 WebSocket 傳輸層（Phase 05）。
//
// 僅用標準庫實作 RFC 6455 交握與訊框（不得新增 x/net 等依賴）：
// 交握（Sec-WebSocket-Accept）、文字／二進位訊框、Ping／Pong、
// Close 交握、訊框分段重組、客戶端遮罩驗證。伺服器送出的訊框
// 不遮罩。訊息大小上限由呼叫方以 MaxMessageBytes 設定（預設
// 1 MiB，超過時關閉連線；上限值為 Phase 05 實作預設，待定稿）。
// 本套件只處理傳輸位元組，不解析事件信封（見 internal/hub）。
package ws

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Opcode 是 WebSocket 訊框操作碼。
type Opcode int

const (
	// OpContinuation 是分段訊框的後續段。
	OpContinuation Opcode = 0x0
	// OpText 是文字訊框。
	OpText Opcode = 0x1
	// OpBinary 是二進位訊框。
	OpBinary Opcode = 0x2
	// OpClose 是關閉訊框。
	OpClose Opcode = 0x8
	// OpPing 是 Ping 訊框。
	OpPing Opcode = 0x9
	// OpPong 是 Pong 訊框。
	OpPong Opcode = 0xA
)

// ErrNotWebSocket 表示請求不是合法的 WebSocket 交握請求。
var ErrNotWebSocket = errors.New("ws: not a websocket handshake")

// ErrMessageTooLarge 表示訊息超過 MaxMessageBytes。
var ErrMessageTooLarge = errors.New("ws: message too large")

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// DefaultMaxMessageBytes 是預設訊息大小上限（1 MiB）。
const DefaultMaxMessageBytes = 1 << 20

// Upgrader 執行 WebSocket 交握，交出 Conn。
type Upgrader struct {
	// MaxMessageBytes 為 0 時用預設值。
	MaxMessageBytes int64
	// CheckOrigin 為空時接受任何 Origin（開發預設；正式部署須
	// 限制來源，待部署文件定稿）。
	CheckOrigin func(r *http.Request) bool
}

func (u *Upgrader) maxBytes() int64 {
	if u.MaxMessageBytes <= 0 {
		return DefaultMaxMessageBytes
	}
	return u.MaxMessageBytes
}

// Upgrade 驗證交握請求、Hijack 連線、回寫 101，交出 Conn。
// 交握失敗時回寫適當的 HTTP 錯誤，不得建立半連線。
func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if !headerContains(r.Header, "Connection", "upgrade") ||
		!headerContains(r.Header, "Upgrade", "websocket") {
		http.Error(w, "not a websocket handshake", http.StatusBadRequest)
		return nil, ErrNotWebSocket
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		http.Error(w, "unsupported websocket version", http.StatusBadRequest)
		return nil, ErrNotWebSocket
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return nil, ErrNotWebSocket
	}
	if u.CheckOrigin != nil && !u.CheckOrigin(r) {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return nil, ErrNotWebSocket
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return nil, errors.New("ws: hijacking not supported")
	}
	conn, brw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	sum := sha1.Sum([]byte(key + wsGUID))
	accept := base64.StdEncoding.EncodeToString(sum[:])
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := brw.WriteString(resp); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := brw.Flush(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Conn{
		conn:     conn,
		reader:   bufio.NewReader(brw.Reader),
		maxBytes: u.maxBytes(),
	}, nil
}

func headerContains(h http.Header, name, value string) bool {
	for _, v := range h.Values(name) {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return true
			}
		}
	}
	return false
}

// CloseCode 是 WebSocket 關閉碼。
type CloseCode int

const (
	// CloseNormal 是正常關閉。
	CloseNormal CloseCode = 1000
	// CloseGoingAway 是端點離開。
	CloseGoingAway CloseCode = 1001
	// CloseProtocolError 是協定錯誤。
	CloseProtocolError CloseCode = 1002
	// CloseUnsupportedData 是不支援的資料型別。
	CloseUnsupportedData CloseCode = 1003
	// CloseMessageTooLarge 是訊息過大。
	CloseMessageTooLarge CloseCode = 1009
	// ClosePolicyViolation 是政策違規。
	ClosePolicyViolation CloseCode = 1008
)

// Conn 是一條 WebSocket 連線。ReadMessage 與 WriteMessage 不得
// 併發呼叫各自的同一方向（寫入由 hub 層以 mutex 序列化）。
type Conn struct {
	conn     net.Conn
	reader   *bufio.Reader
	maxBytes int64
}

// SetReadDeadline 設定讀取逾時（心跳逾時判定用）。
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// Close 關閉底層連線。
func (c *Conn) Close() error {
	return c.conn.Close()
}

// ReadMessage 讀取一則完整訊息（文字或二進位，分段自動重組）。
// 收到 Ping 時自動回 Pong；收到 Close 時回傳 io.EOF（呼叫方須
// 回 Close 訊框後關閉）。控制訊框的酬載不得超過 125 位元組
// （RFC 6455），違者回傳協定錯誤。
func (c *Conn) ReadMessage() (Opcode, []byte, error) {
	var payload []byte
	var messageOp Opcode
	fragmented := false
	for {
		fin, op, data, err := c.readFrame()
		if err != nil {
			return 0, nil, err
		}
		switch op {
		case OpPing:
			if err := c.writeFrame(true, OpPong, data); err != nil {
				return 0, nil, err
			}
			continue
		case OpPong:
			continue
		case OpClose:
			return 0, nil, io.EOF
		case OpText, OpBinary:
			if fragmented {
				return 0, nil, errors.New("ws: unexpected data frame during fragmentation")
			}
			messageOp = op
			payload = append(payload, data...)
			if !fin {
				fragmented = true
				continue
			}
			return messageOp, payload, nil
		case OpContinuation:
			if !fragmented {
				return 0, nil, errors.New("ws: unexpected continuation frame")
			}
			payload = append(payload, data...)
			if !fin {
				continue
			}
			return messageOp, payload, nil
		default:
			return 0, nil, fmt.Errorf("ws: unknown opcode %d", op)
		}
	}
}

// readFrame 讀取一個訊框，回傳 FIN、opcode 與解遮罩後的酬載。
func (c *Conn) readFrame() (bool, Opcode, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return false, 0, nil, err
	}
	fin := header[0]&0x80 != 0
	rsv := header[0] & 0x70
	if rsv != 0 {
		return false, 0, nil, errors.New("ws: RSV bits must be zero")
	}
	op := Opcode(header[0] & 0x0f)
	masked := header[1]&0x80 != 0
	length := int64(header[1] & 0x7f)
	switch length {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(c.reader, ext[:]); err != nil {
			return false, 0, nil, err
		}
		length = int64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(c.reader, ext[:]); err != nil {
			return false, 0, nil, err
		}
		length = int64(binary.BigEndian.Uint64(ext[:]))
		if length < 0 {
			return false, 0, nil, errors.New("ws: negative frame length")
		}
	}
	if op == OpPing || op == OpPong || op == OpClose {
		if length > 125 {
			return false, 0, nil, errors.New("ws: control frame too large")
		}
	}
	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(c.reader, maskKey[:]); err != nil {
			return false, 0, nil, err
		}
	} else {
		// 客戶端訊框必須遮罩；伺服器收到未遮罩訊框即協定錯誤。
		return false, 0, nil, errors.New("ws: client frame must be masked")
	}
	if length > c.maxBytes {
		return false, 0, nil, ErrMessageTooLarge
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(c.reader, payload); err != nil {
		return false, 0, nil, err
	}
	for i := range payload {
		payload[i] ^= maskKey[i%4]
	}
	return fin, op, payload, nil
}

// WriteMessage 寫出一則文字訊息（單一訊框、FIN 置位）。
func (c *Conn) WriteMessage(payload []byte) error {
	return c.writeFrame(true, OpText, payload)
}

// WriteControl 寫出控制訊框（Ping／Pong／Close）。
func (c *Conn) WriteControl(op Opcode, payload []byte) error {
	return c.writeFrame(true, op, payload)
}

// WriteClose 送出 Close 訊框（含關閉碼與原因）。
func (c *Conn) WriteClose(code CloseCode, reason string) error {
	payload := make([]byte, 2+len(reason))
	binary.BigEndian.PutUint16(payload, uint16(code))
	copy(payload[2:], reason)
	return c.writeFrame(true, OpClose, payload)
}

func (c *Conn) writeFrame(fin bool, op Opcode, payload []byte) error {
	var header []byte
	b0 := byte(op)
	if fin {
		b0 |= 0x80
	}
	header = append(header, b0)
	length := len(payload)
	switch {
	case length <= 125:
		header = append(header, byte(length))
	case length <= 65535:
		header = append(header, 126, byte(length>>8), byte(length))
	default:
		header = append(header, 127)
		var ext [8]byte
		binary.BigEndian.PutUint64(ext[:], uint64(length))
		header = append(header, ext[:]...)
	}
	if _, err := c.conn.Write(header); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := c.conn.Write(payload); err != nil {
			return err
		}
	}
	return nil
}
