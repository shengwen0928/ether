package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ether.local/realtime-gateway/internal/auth"
	"ether.local/realtime-gateway/internal/broker"
	"ether.local/realtime-gateway/internal/health"
	"ether.local/realtime-gateway/internal/hub"
)

func testHub(t *testing.T, verifier hub.Verifier) *hub.Hub {
	t.Helper()
	b := broker.NewMemoryBroker()
	t.Cleanup(func() { _ = b.Close() })
	h := hub.NewHub(b, verifier, nil, nil)
	t.Cleanup(h.Close)
	return h
}

// TestHealthz 是 healthz 的純標準庫測試。僅驗證 /healthz 回應
// 200 與三個欄位，不驗證任何依賴連線。
func TestHealthz(t *testing.T) {
	server := httptest.NewServer(buildHandler(testHub(t, nil)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz: status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var got health.Response
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("GET /healthz: decode: %v", err)
	}
	if got.Status != "ok" || got.Service != serviceName || got.Version != version {
		t.Fatalf("GET /healthz: got %+v, want status=ok service=%s version=%s", got, serviceName, version)
	}
}

type rejectVerifier struct{}

func (rejectVerifier) Verify(context.Context, string) (auth.Identity, error) {
	return auth.Identity{}, auth.ErrInvalidToken
}

// TestWSRejectsUnauthenticated 驗證 WS 端點在驗證器存在時，
// 無 Token 的請求回 401（不得建立未認證連線）。
func TestWSRejectsUnauthenticated(t *testing.T) {
	server := httptest.NewServer(buildHandler(testHub(t, rejectVerifier{})))
	defer server.Close()

	resp, err := http.Get(server.URL + wsPath)
	if err != nil {
		t.Fatalf("GET %s: %v", wsPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET %s: status = %d, want %d", wsPath, resp.StatusCode, http.StatusUnauthorized)
	}
}

// TestWSFailsClosedWithoutVerifier 驗證驗證器為 nil 時 WS 端點
// 回 503（fail-closed），而 /healthz 不受影響。
func TestWSFailsClosedWithoutVerifier(t *testing.T) {
	server := httptest.NewServer(buildHandler(testHub(t, nil)))
	defer server.Close()

	resp, err := http.Get(server.URL + wsPath + "?access_token=any")
	if err != nil {
		t.Fatalf("GET %s: %v", wsPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GET %s: status = %d, want %d", wsPath, resp.StatusCode, http.StatusServiceUnavailable)
	}
}
