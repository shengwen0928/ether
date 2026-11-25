package search

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeMeilisearch 是 Meilisearch REST API 的最小模擬：以行程內
// 文件表回應文件寫入／讀取／搜尋，並記錄收到的請求（供斷言
// 請求 shape）。任務一律立即回 succeeded。
type fakeMeilisearch struct {
	mu       sync.Mutex
	docs     map[string]meiliDocument
	requests []fakeMeilisearchRequest
	// failTask 使下一次寫入任務回 failed（測試任務失敗路徑）。
	failTask bool
	// taskStatus 是最近一次寫入任務的最終狀態（供任務輪詢回傳）。
	taskStatus string
}

type fakeMeilisearchRequest struct {
	method string
	path   string
	body   string
}

func (f *fakeMeilisearch) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.requests = append(f.requests, fakeMeilisearchRequest{method: r.Method, path: r.URL.Path, body: string(body)})
	taskStatus := f.taskStatus
	f.mu.Unlock()

	writeJSON := func(status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}
	taskJSON := func() map[string]any {
		f.mu.Lock()
		failTask := f.failTask
		f.failTask = false
		status := "succeeded"
		if failTask {
			status = "failed"
		}
		f.taskStatus = status
		f.mu.Unlock()
		return map[string]any{
			"taskUid": 1, "indexUid": MessagesIndexUID,
			"status": status, "type": "documentAdditionOrUpdate",
			"enqueuedAt": time.Now().UTC().Format(time.RFC3339),
		}
	}
	path := r.URL.Path
	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/tasks/"):
		status := taskStatus
		if status == "" {
			status = "succeeded"
		}
		writeJSON(http.StatusOK, map[string]any{
			"uid": 1, "taskUid": 1, "indexUid": MessagesIndexUID,
			"status": status, "type": "documentAdditionOrUpdate",
			"enqueuedAt": time.Now().UTC().Format(time.RFC3339),
		})
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/documents"):
		var incoming []meiliDocument
		if err := json.Unmarshal(body, &incoming); err != nil {
			writeJSON(http.StatusBadRequest, map[string]any{"message": "bad documents"})
			return
		}
		f.mu.Lock()
		for _, d := range incoming {
			f.docs[d.MessageID] = d
		}
		f.mu.Unlock()
		writeJSON(http.StatusAccepted, taskJSON())
	case r.Method == http.MethodGet && strings.Contains(path, "/documents/"):
		id := path[strings.LastIndex(path, "/")+1:]
		f.mu.Lock()
		doc, ok := f.docs[id]
		f.mu.Unlock()
		if !ok {
			writeJSON(http.StatusNotFound, map[string]any{
				"message": "Document not found", "code": "document_not_found",
				"type": "invalid_request", "link": "https://example.invalid",
			})
			return
		}
		writeJSON(http.StatusOK, doc)
	case r.Method == http.MethodDelete && strings.HasSuffix(path, "/documents"):
		f.mu.Lock()
		f.docs = map[string]meiliDocument{}
		f.mu.Unlock()
		writeJSON(http.StatusAccepted, taskJSON())
	case r.Method == http.MethodDelete && strings.Contains(path, "/documents/"):
		id := path[strings.LastIndex(path, "/")+1:]
		f.mu.Lock()
		delete(f.docs, id)
		f.mu.Unlock()
		writeJSON(http.StatusAccepted, taskJSON())
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/search"):
		var req struct {
			Query  string   `json:"q"`
			Filter any      `json:"filter"`
			Sort   []string `json:"sort"`
			Limit  int64    `json:"limit"`
			Offset int64    `json:"offset"`
		}
		_ = json.Unmarshal(body, &req)
		f.mu.Lock()
		var hits []meiliDocument
		for _, d := range f.docs {
			hits = append(hits, d)
		}
		f.mu.Unlock()
		writeJSON(http.StatusOK, map[string]any{
			"hits": hits, "query": req.Query, "processingTimeMs": 1,
			"estimatedTotalHits": len(hits), "offset": req.Offset, "limit": req.Limit,
		})
	default:
		writeJSON(http.StatusNotFound, map[string]any{"message": "unknown route"})
	}
}

func newMeilisearchTestIndex(t *testing.T) (*MeilisearchIndex, *fakeMeilisearch) {
	t.Helper()
	fake := &fakeMeilisearch{docs: map[string]meiliDocument{}}
	server := httptest.NewServer(http.HandlerFunc(fake.handler))
	t.Cleanup(server.Close)
	idx, err := DialMeilisearch(server.URL, "test-api-key", MessagesIndexUID)
	if err != nil {
		t.Fatalf("DialMeilisearch: %v", err)
	}
	return idx, fake
}

func testDocument() Document {
	return Document{
		MessageID:      "m-1",
		ConversationID: "conv-1",
		SenderID:       "u-1",
		Kind:           "text",
		Content:        "你好世界",
		SequenceNumber: 3,
		CreatedAt:      time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC),
	}
}

func TestMeilisearchIndex_UpsertAndSearch(t *testing.T) {
	idx, fake := newMeilisearchTestIndex(t)
	ctx := context.Background()
	if err := idx.Upsert(ctx, testDocument(), "ev-1"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	// 同一事件重放：不得重複寫入（第二次不應再發文件請求）。
	before := len(fake.requests)
	if err := idx.Upsert(ctx, testDocument(), "ev-1"); err != nil {
		t.Fatalf("Upsert 重放: %v", err)
	}
	fake.mu.Lock()
	after := len(fake.requests)
	fake.mu.Unlock()
	if after != before {
		t.Fatalf("重放後請求數 %d -> %d, want 不變（冪等去重）", before, after)
	}
	docs, err := idx.Search(ctx, Query{Terms: []string{"你好"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(docs) != 1 || docs[0].MessageID != "m-1" || docs[0].Content != "你好世界" {
		t.Fatalf("Search = %+v, want 1 筆 m-1", docs)
	}
}

func TestMeilisearchIndex_UpsertRejectsOlderSequence(t *testing.T) {
	idx, _ := newMeilisearchTestIndex(t)
	ctx := context.Background()
	if err := idx.Upsert(ctx, testDocument(), "ev-1"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	older := testDocument()
	older.SequenceNumber = 2
	older.Content = "舊版本內容"
	if err := idx.Upsert(ctx, older, "ev-2"); err != nil {
		t.Fatalf("Upsert 舊版本: %v", err)
	}
	docs, err := idx.Search(ctx, Query{Terms: []string{"你好"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(docs) != 1 || docs[0].Content != "你好世界" {
		t.Fatalf("Search = %+v, want 內容仍為新版本", docs)
	}
}

func TestMeilisearchIndex_Remove(t *testing.T) {
	idx, _ := newMeilisearchTestIndex(t)
	ctx := context.Background()
	if err := idx.Upsert(ctx, testDocument(), "ev-1"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := idx.Remove(ctx, "m-1", "ev-2"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	docs, err := idx.Search(ctx, Query{Terms: []string{"你好"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("Search = %d 筆, want 0（已刪除）", len(docs))
	}
	// 刪除不存在的文件為冪等成功。
	if err := idx.Remove(ctx, "m-missing", "ev-3"); err != nil {
		t.Fatalf("Remove 不存在的文件: %v", err)
	}
}

func TestMeilisearchIndex_SearchSendsFilterAndSort(t *testing.T) {
	idx, fake := newMeilisearchTestIndex(t)
	ctx := context.Background()
	since := time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)
	_, err := idx.Search(ctx, Query{
		Terms:          []string{"你好"},
		ConversationID: "conv-1",
		SenderID:       "u-1",
		Since:          &since,
		Until:          &until,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	var searchReq *fakeMeilisearchRequest
	for i := range fake.requests {
		if strings.HasSuffix(fake.requests[i].path, "/search") {
			searchReq = &fake.requests[i]
		}
	}
	if searchReq == nil {
		t.Fatal("未發出搜尋請求")
	}
	var body struct {
		Filter any      `json:"filter"`
		Sort   []string `json:"sort"`
	}
	if err := json.Unmarshal([]byte(searchReq.body), &body); err != nil {
		t.Fatalf("搜尋請求解碼失敗: %v", err)
	}
	filter, _ := body.Filter.(string)
	for _, want := range []string{`conversation_id = "conv-1"`, `sender_id = "u-1"`, "created_at >="} {
		if !strings.Contains(filter, want) {
			t.Fatalf("filter = %q, want 包含 %q", filter, want)
		}
	}
	if len(body.Sort) != 2 || body.Sort[0] != "created_at:desc" || body.Sort[1] != "message_id:desc" {
		t.Fatalf("sort = %v, want [created_at:desc message_id:desc]", body.Sort)
	}
}

func TestMeilisearchIndex_ReplaceAll(t *testing.T) {
	idx, _ := newMeilisearchTestIndex(t)
	ctx := context.Background()
	if err := idx.Upsert(ctx, testDocument(), "ev-1"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	replacement := testDocument()
	replacement.MessageID = "m-2"
	replacement.Content = "重建內容"
	if err := idx.ReplaceAll(ctx, []Document{replacement}); err != nil {
		t.Fatalf("ReplaceAll: %v", err)
	}
	docs, err := idx.Search(ctx, Query{Terms: []string{"重建"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(docs) != 1 || docs[0].MessageID != "m-2" {
		t.Fatalf("Search = %+v, want 僅 m-2（舊文件已清除）", docs)
	}
	// 重建後冪等集合重置：舊事件可再次套用。
	if err := idx.Upsert(ctx, testDocument(), "ev-1"); err != nil {
		t.Fatalf("重建後 Upsert 舊事件: %v", err)
	}
}

func TestMeilisearchIndex_TaskFailureReported(t *testing.T) {
	idx, fake := newMeilisearchTestIndex(t)
	fake.mu.Lock()
	fake.failTask = true
	fake.mu.Unlock()
	err := idx.Upsert(context.Background(), testDocument(), "ev-1")
	if err == nil {
		t.Fatal("任務失敗未被回報")
	}
}

func TestDialMeilisearch_RejectsMissingConfig(t *testing.T) {
	if _, err := DialMeilisearch("", "key", ""); err == nil {
		t.Fatal("空 host 未被拒絕")
	}
	if _, err := DialMeilisearch("http://localhost:7700", "", ""); err == nil {
		t.Fatal("空 API 金鑰未被拒絕")
	}
}

func TestMeilisearchIndex_UnconfiguredPlaceholder(t *testing.T) {
	// 未設定佔位：任何呼叫皆回傳 ErrNotConfigured（既有行為保留）。
	idx := NewMeilisearchIndex()
	ctx := context.Background()
	if err := idx.Upsert(ctx, testDocument(), "ev-1"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Upsert err = %v, want ErrNotConfigured", err)
	}
	if _, err := idx.Search(ctx, Query{Terms: []string{"x"}}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Search err = %v, want ErrNotConfigured", err)
	}
}
