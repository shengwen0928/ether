package ws

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// rawClient 是測試用的最小 WebSocket 客戶端（TCP 直連、手動交握、
// 遮罩訊框），不依賴任何 WebSocket 函式庫。
type rawClient struct {
	conn   net.Conn
	reader *bufio.Reader
}

func dialRaw(t *testing.T, url string) *rawClient {
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
	req := "GET /ws/v1 HTTP/1.1\r\n" +
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

func (c *rawClient) sendFrame(op Opcode, payload []byte) error {
	mask := [4]byte{0x11, 0x22, 0x33, 0x44}
	header := []byte{0x80 | byte(op)}
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

func (c *rawClient) readFrame() (Opcode, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return 0, nil, err
	}
	op := Opcode(header[0] & 0x0f)
	length := int64(header[1] & 0x7f)
	switch length {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(c.reader, ext[:]); err != nil {
			return 0, nil, err
		}
		length = int64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(c.reader, ext[:]); err != nil {
			return 0, nil, err
		}
		length = int64(binary.BigEndian.Uint64(ext[:]))
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(c.reader, payload); err != nil {
		return 0, nil, err
	}
	return op, payload, nil
}

func TestUpgradeAndEcho(t *testing.T) {
	upgrader := Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		for {
			op, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if op == OpText {
				_ = conn.WriteMessage(payload)
			}
		}
	}))
	defer server.Close()

	client := dialRaw(t, server.URL)
	defer func() { _ = client.conn.Close() }()
	if err := client.sendFrame(OpText, []byte("hello")); err != nil {
		t.Fatalf("send: %v", err)
	}
	_ = client.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	op, payload, err := client.readFrame()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if op != OpText || string(payload) != "hello" {
		t.Fatalf("echo = op %d %q", op, payload)
	}
}

func TestUpgradeRejectsNonWebSocket(t *testing.T) {
	upgrader := Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = upgrader.Upgrade(w, r)
	}))
	defer server.Close()
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestPingPong(t *testing.T) {
	upgrader := Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _, err = conn.ReadMessage()
		_ = err
	}))
	defer server.Close()

	client := dialRaw(t, server.URL)
	defer func() { _ = client.conn.Close() }()
	if err := client.sendFrame(OpPing, []byte("ping-data")); err != nil {
		t.Fatalf("send ping: %v", err)
	}
	_ = client.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	op, payload, err := client.readFrame()
	if err != nil {
		t.Fatalf("read pong: %v", err)
	}
	if op != OpPong || string(payload) != "ping-data" {
		t.Fatalf("pong = op %d %q", op, payload)
	}
}

func TestFragmentedMessageReassembled(t *testing.T) {
	upgrader := Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.WriteMessage(payload)
	}))
	defer server.Close()

	client := dialRaw(t, server.URL)
	defer func() { _ = client.conn.Close() }()
	// 手動送出兩個分段：FIN=0 的 Text 段＋FIN=1 的 Continuation 段。
	mask := [4]byte{0x01, 0x02, 0x03, 0x04}
	writeFrag := func(first byte, data []byte) {
		header := []byte{first, 0x80 | byte(len(data))}
		header = append(header, mask[:]...)
		masked := make([]byte, len(data))
		for i := range data {
			masked[i] = data[i] ^ mask[i%4]
		}
		_, _ = client.conn.Write(header)
		_, _ = client.conn.Write(masked)
	}
	writeFrag(0x01, []byte("hel"))
	writeFrag(0x80, []byte("lo"))
	_ = client.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	op, payload, err := client.readFrame()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if op != OpText || string(payload) != "hello" {
		t.Fatalf("reassembled = op %d %q", op, payload)
	}
}
