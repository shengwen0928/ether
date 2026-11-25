package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ether.local/worker/internal/health"
)

// TestHealthz 是 healthz 的純標準庫測試骨架（placeholder）。
// 僅驗證 /healthz 回應 200 與三個欄位，不驗證任何依賴連線，
// 不驗證任何任務處理（本階段無任務）。
// 本機 VM 無 go 工具鏈，本測試尚未執行，狀態為「待 CI 驗證」。
func TestHealthz(t *testing.T) {
	server := httptest.NewServer(buildHandler())
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
