package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testSearchHandler(t *testing.T, idx Index, checker MembershipChecker) http.Handler {
	t.Helper()
	h := &Handler{
		Service:     NewService(idx, checker),
		CurrentUser: func(r *http.Request) (string, bool) { return "u-9", true },
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/search/messages", h.SearchMessages)
	return mux
}

func seedSearchIndex(t *testing.T, idx *MemoryIndex) {
	t.Helper()
	ctx := context.Background()
	base := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	docs := []Document{
		testDoc("m-1", "c-1", "u-1", "早安大家好", 1, base),
		testDoc("m-2", "c-1", "u-2", "早安，今天開會討論", 2, base.Add(time.Minute)),
		testDoc("m-3", "c-2", "u-1", "早安", 1, base.Add(2*time.Minute)),
	}
	for i, doc := range docs {
		if err := idx.Upsert(ctx, doc, "ev-"+string(rune('a'+i))); err != nil {
			t.Fatalf("Upsert %d: %v", i, err)
		}
	}
}

func TestHandler_SearchMessages_Flow(t *testing.T) {
	idx := NewMemoryIndex()
	seedSearchIndex(t, idx)
	checker := &stubChecker{members: map[string]bool{"c-1:u-9": true}}
	server := httptest.NewServer(testSearchHandler(t, idx, checker))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/search/messages?q=" + "%E6%97%A9%E5%AE%89")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("狀態碼應為 200，實際 %d", resp.StatusCode)
	}
	var body searchMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("解碼回應: %v", err)
	}
	// c-2 的文件須被過濾（呼叫者非成員），只剩 c-1 的 2 筆。
	if len(body.Results) != 2 {
		t.Fatalf("預期 2 筆結果，實際 %+v", body.Results)
	}
	if body.Results[0].MessageID != "m-2" || body.Results[1].MessageID != "m-1" {
		t.Fatalf("排序或內容錯誤，實際 %+v", body.Results)
	}
	if body.Results[0].Content != "早安，今天開會討論" {
		t.Fatalf("內容應原樣回傳，實際 %+v", body.Results[0])
	}
}

func TestHandler_SearchMessages_Validation(t *testing.T) {
	idx := NewMemoryIndex()
	checker := &stubChecker{members: map[string]bool{}}
	server := httptest.NewServer(testSearchHandler(t, idx, checker))
	defer server.Close()

	// 缺 q：400 欄位錯誤。
	resp, err := http.Get(server.URL + "/api/v1/search/messages")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("缺 q 應回 400，實際 %d", resp.StatusCode)
	}
	// limit 非正整數：400。
	resp2, err := http.Get(server.URL + "/api/v1/search/messages?q=x&limit=abc")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("limit 非法應回 400，實際 %d", resp2.StatusCode)
	}
	// since 非 RFC3339：400。
	resp3, err := http.Get(server.URL + "/api/v1/search/messages?q=x&since=nope")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp3.Body.Close() }()
	if resp3.StatusCode != http.StatusBadRequest {
		t.Fatalf("since 非法應回 400，實際 %d", resp3.StatusCode)
	}
}

func TestHandler_SearchMessages_IndexUnavailable(t *testing.T) {
	// Meilisearch 佔位後端：查詢回 503，不得回空結果假裝完整。
	h := &Handler{
		Service:     NewService(NewMeilisearchIndex(), &stubChecker{members: map[string]bool{}}),
		CurrentUser: func(r *http.Request) (string, bool) { return "u-9", true },
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/search/messages", h.SearchMessages)
	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/search/messages?q=x")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("索引不可用應回 503，實際 %d", resp.StatusCode)
	}
}
