package search

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubChecker struct {
	members map[string]bool
	err     error
}

func (s *stubChecker) IsActiveMember(_ context.Context, conversationID, userID string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.members[conversationID+":"+userID], nil
}

func testDoc(id, conv, sender, content string, seq int64, createdAt time.Time) Document {
	return Document{
		MessageID:      id,
		ConversationID: conv,
		SenderID:       sender,
		Kind:           "text",
		Content:        content,
		SequenceNumber: seq,
		CreatedAt:      createdAt,
	}
}

func TestMemoryIndex_UpsertIdempotentAndVersionGuard(t *testing.T) {
	ctx := context.Background()
	idx := NewMemoryIndex()
	now := time.Now().UTC()
	doc := testDoc("m-1", "c-1", "u-1", "Hello World", 3, now)

	if err := idx.Upsert(ctx, doc, "ev-1"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	// 同一事件重放：no-op，不得產生重複文件。
	if err := idx.Upsert(ctx, doc, "ev-1"); err != nil {
		t.Fatalf("重放 Upsert: %v", err)
	}
	// 較舊序號的投影不得覆蓋較新者。
	stale := testDoc("m-1", "c-1", "u-1", "舊內容", 2, now)
	if err := idx.Upsert(ctx, stale, "ev-2"); err != nil {
		t.Fatalf("舊版 Upsert: %v", err)
	}
	got, err := idx.Search(ctx, Query{Terms: []string{"hello"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 || got[0].Content != "Hello World" {
		t.Fatalf("預期 1 筆 Hello World，實際 %+v", got)
	}
	// 空 eventID 須拒絕（呼叫方須傳入冪等鍵）。
	if err := idx.Upsert(ctx, doc, ""); err == nil {
		t.Fatal("空 eventID 應回傳錯誤")
	}
}

func TestMemoryIndex_RemoveIdempotent(t *testing.T) {
	ctx := context.Background()
	idx := NewMemoryIndex()
	now := time.Now().UTC()
	if err := idx.Upsert(ctx, testDoc("m-1", "c-1", "u-1", "待收回", 1, now), "ev-1"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := idx.Remove(ctx, "m-1", "ev-2"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	// 重放刪除事件：no-op，不得報錯。
	if err := idx.Remove(ctx, "m-1", "ev-2"); err != nil {
		t.Fatalf("重放 Remove: %v", err)
	}
	// 不存在的文件刪除為冪等成功。
	if err := idx.Remove(ctx, "m-9", "ev-3"); err != nil {
		t.Fatalf("刪除不存在文件: %v", err)
	}
	got, err := idx.Search(ctx, Query{Terms: []string{"待收回"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("收回後索引須無殘留，實際 %+v", got)
	}
}

func TestMemoryIndex_SearchFiltersAndOrdering(t *testing.T) {
	ctx := context.Background()
	idx := NewMemoryIndex()
	base := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	docs := []Document{
		testDoc("m-1", "c-1", "u-1", "早安大家好", 1, base),
		testDoc("m-2", "c-1", "u-2", "早安，今天開會", 2, base.Add(time.Minute)),
		testDoc("m-3", "c-2", "u-1", "早安", 1, base.Add(2*time.Minute)),
		testDoc("m-4", "c-1", "u-1", "無關內容", 3, base.Add(3*time.Minute)),
		testDoc("m-5", "c-2", "u-2", "Hello World", 1, base.Add(4*time.Minute)),
	}
	for i, doc := range docs {
		if err := idx.Upsert(ctx, doc, "ev-"+string(rune('a'+i))); err != nil {
			t.Fatalf("Upsert %d: %v", i, err)
		}
	}
	// AND 語義：兩詞皆須命中。
	got, err := idx.Search(ctx, Query{Terms: []string{"早安", "開會"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 || got[0].MessageID != "m-2" {
		t.Fatalf("AND 語義錯誤，實際 %+v", got)
	}
	// 對話篩選＋降序。
	got, err = idx.Search(ctx, Query{Terms: []string{"早安"}, ConversationID: "c-1"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 2 || got[0].MessageID != "m-2" || got[1].MessageID != "m-1" {
		t.Fatalf("對話篩選或排序錯誤，實際 %+v", got)
	}
	// 發送者篩選。
	got, err = idx.Search(ctx, Query{Terms: []string{"早安"}, SenderID: "u-2"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 || got[0].MessageID != "m-2" {
		t.Fatalf("發送者篩選錯誤，實際 %+v", got)
	}
	// 時間區間（含邊界）。
	since := base.Add(time.Minute)
	until := base.Add(2 * time.Minute)
	got, err = idx.Search(ctx, Query{Terms: []string{"早安"}, Since: &since, Until: &until})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("時間區間錯誤，實際 %+v", got)
	}
	// 大小寫不敏感：大寫查詢須命中 mixed-case 內容。
	got, err = idx.Search(ctx, Query{Terms: []string{"HELLO"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 || got[0].MessageID != "m-5" {
		t.Fatalf("大小寫不敏感比對錯誤，實際 %+v", got)
	}
}

func TestMemoryIndex_ReplaceAll(t *testing.T) {
	ctx := context.Background()
	idx := NewMemoryIndex()
	now := time.Now().UTC()
	if err := idx.Upsert(ctx, testDoc("m-1", "c-1", "u-1", "舊文件", 1, now), "ev-1"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := idx.ReplaceAll(ctx, []Document{testDoc("m-2", "c-1", "u-1", "重建文件", 1, now)}); err != nil {
		t.Fatalf("ReplaceAll: %v", err)
	}
	got, err := idx.Search(ctx, Query{Terms: []string{"文件"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 || got[0].MessageID != "m-2" {
		t.Fatalf("重建後應只剩新文件，實際 %+v", got)
	}
	// 無效文件不得留下部分重建的索引。
	if err := idx.ReplaceAll(ctx, []Document{{MessageID: "", ConversationID: "c-1"}}); err == nil {
		t.Fatal("無效文件應回傳錯誤")
	}
	got, err = idx.Search(ctx, Query{Terms: []string{"重建"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("失敗的重建不得破壞既有索引，實際 %+v", got)
	}
}

func TestMeilisearchIndex_NotConfigured(t *testing.T) {
	ctx := context.Background()
	idx := NewMeilisearchIndex()
	doc := testDoc("m-1", "c-1", "u-1", "內容", 1, time.Now().UTC())
	if err := idx.Upsert(ctx, doc, "ev-1"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Upsert 應回 ErrNotConfigured，實際 %v", err)
	}
	if _, err := idx.Search(ctx, Query{Terms: []string{"內容"}}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Search 應回 ErrNotConfigured，實際 %v", err)
	}
	if err := idx.Remove(ctx, "m-1", "ev-2"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Remove 應回 ErrNotConfigured，實際 %v", err)
	}
	if err := idx.ReplaceAll(ctx, nil); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("ReplaceAll 應回 ErrNotConfigured，實際 %v", err)
	}
}

func TestProjector_ApplyAndRebuild(t *testing.T) {
	ctx := context.Background()
	idx := NewMemoryIndex()
	p := NewProjector(idx)
	now := time.Now().UTC()
	doc := testDoc("m-1", "c-1", "u-1", "投影內容", 1, now)
	if err := p.ApplyReceived(ctx, doc, "ev-1"); err != nil {
		t.Fatalf("ApplyReceived: %v", err)
	}
	if err := p.ApplyReceived(ctx, doc, "ev-1"); err != nil {
		t.Fatalf("重放 ApplyReceived: %v", err)
	}
	if err := p.ApplyRecalled(ctx, "m-1", "ev-2"); err != nil {
		t.Fatalf("ApplyRecalled: %v", err)
	}
	svc := NewService(idx, &stubChecker{members: map[string]bool{"c-1:u-1": true}})
	page, err := svc.Search(ctx, "u-1", Query{Terms: []string{"投影"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(page.Documents) != 0 {
		t.Fatalf("收回後投影須移除，實際 %+v", page.Documents)
	}
	if err := p.Rebuild(ctx, []Document{doc}); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	page, err = svc.Search(ctx, "u-1", Query{Terms: []string{"投影"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(page.Documents) != 1 {
		t.Fatalf("重建後應有 1 筆，實際 %+v", page.Documents)
	}
}

func TestService_SearchPermissionFiltering(t *testing.T) {
	ctx := context.Background()
	idx := NewMemoryIndex()
	base := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	docs := []Document{
		testDoc("m-1", "c-1", "u-1", "共同關鍵字", 1, base),
		testDoc("m-2", "c-2", "u-2", "共同關鍵字", 1, base.Add(time.Minute)),
		testDoc("m-3", "c-1", "u-2", "共同關鍵字", 2, base.Add(2*time.Minute)),
	}
	for i, doc := range docs {
		if err := idx.Upsert(ctx, doc, "ev-"+string(rune('a'+i))); err != nil {
			t.Fatalf("Upsert %d: %v", i, err)
		}
	}
	checker := &stubChecker{members: map[string]bool{"c-1:u-9": true}}
	svc := NewService(idx, checker)
	page, err := svc.Search(ctx, "u-9", Query{Terms: []string{"共同"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(page.Documents) != 2 {
		t.Fatalf("非成員對話須過濾，預期 2 筆，實際 %+v", page.Documents)
	}
	for _, doc := range page.Documents {
		if doc.ConversationID != "c-1" {
			t.Fatalf("結果含非成員對話文件: %+v", doc)
		}
	}
	// 檢查器失敗時 fail-closed，不得回傳未過濾結果。
	broken := NewService(idx, &stubChecker{err: errors.New("checker down")})
	if _, err := broken.Search(ctx, "u-9", Query{Terms: []string{"共同"}}); !errors.Is(err, ErrIndexUnavailable) {
		t.Fatalf("檢查器失敗應回 ErrIndexUnavailable，實際 %v", err)
	}
	// checker 為 nil 時 fail-closed。
	nilChecker := NewService(idx, nil)
	if _, err := nilChecker.Search(ctx, "u-9", Query{Terms: []string{"共同"}}); !errors.Is(err, ErrIndexUnavailable) {
		t.Fatalf("nil checker 應回 ErrIndexUnavailable，實際 %v", err)
	}
}

func TestService_SearchPagination(t *testing.T) {
	ctx := context.Background()
	idx := NewMemoryIndex()
	base := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		doc := testDoc(
			"m-"+string(rune('1'+i)), "c-1", "u-1", "分頁關鍵字",
			int64(i+1), base.Add(time.Duration(i)*time.Minute),
		)
		if err := idx.Upsert(ctx, doc, "ev-"+string(rune('a'+i))); err != nil {
			t.Fatalf("Upsert %d: %v", i, err)
		}
	}
	svc := NewService(idx, &stubChecker{members: map[string]bool{"c-1:u-9": true}})
	page1, err := svc.Search(ctx, "u-9", Query{Terms: []string{"分頁"}, Limit: 2})
	if err != nil {
		t.Fatalf("Search p1: %v", err)
	}
	if len(page1.Documents) != 2 || page1.NextCursor == "" {
		t.Fatalf("第一頁應有 2 筆＋游標，實際 %+v", page1)
	}
	if page1.Documents[0].MessageID != "m-5" || page1.Documents[1].MessageID != "m-4" {
		t.Fatalf("降序錯誤，實際 %+v", page1.Documents)
	}
	page2, err := svc.Search(ctx, "u-9", Query{Terms: []string{"分頁"}, Limit: 2, Cursor: page1.NextCursor})
	if err != nil {
		t.Fatalf("Search p2: %v", err)
	}
	if len(page2.Documents) != 2 || page2.NextCursor == "" {
		t.Fatalf("第二頁應有 2 筆＋游標，實際 %+v", page2)
	}
	page3, err := svc.Search(ctx, "u-9", Query{Terms: []string{"分頁"}, Limit: 2, Cursor: page2.NextCursor})
	if err != nil {
		t.Fatalf("Search p3: %v", err)
	}
	if len(page3.Documents) != 1 || page3.NextCursor != "" {
		t.Fatalf("第三頁應有 1 筆且無游標，實際 %+v", page3)
	}
	// 游標損毀時回驗證錯誤，不得靜默重頭。
	if _, err := svc.Search(ctx, "u-9", Query{Terms: []string{"分頁"}, Cursor: "!!!"}); err == nil {
		t.Fatal("損毀游標應回傳錯誤")
	}
}

func TestService_SearchValidation(t *testing.T) {
	ctx := context.Background()
	idx := NewMemoryIndex()
	svc := NewService(idx, &stubChecker{members: map[string]bool{}})
	if _, err := svc.Search(ctx, "u-9", Query{Terms: nil}); err == nil {
		t.Fatal("空關鍵字應回傳驗證錯誤")
	}
	if _, err := svc.Search(ctx, "", Query{Terms: []string{"x"}}); err == nil {
		t.Fatal("空使用者應回傳驗證錯誤")
	}
	// 索引後端未設定時回 ErrIndexUnavailable，不得回空結果假裝完整。
	unconfigured := NewService(NewMeilisearchIndex(), &stubChecker{members: map[string]bool{}})
	if _, err := unconfigured.Search(ctx, "u-9", Query{Terms: []string{"x"}}); !errors.Is(err, ErrIndexUnavailable) {
		t.Fatalf("未設定索引應回 ErrIndexUnavailable，實際 %v", err)
	}
}

func TestValidateQuery(t *testing.T) {
	q, err := ValidateQuery("  早安 開會 ", 0, "", "", "", nil, nil)
	if err != nil {
		t.Fatalf("ValidateQuery: %v", err)
	}
	if len(q.Terms) != 2 || q.Terms[0] != "早安" || q.Terms[1] != "開會" {
		t.Fatalf("正規化錯誤，實際 %+v", q.Terms)
	}
	if _, err := ValidateQuery("   ", 0, "", "", "", nil, nil); err == nil {
		t.Fatal("空白關鍵字應回傳錯誤")
	}
	long := ""
	for i := 0; i < MaxQueryRunes+1; i++ {
		long += "字"
	}
	if _, err := ValidateQuery(long, 0, "", "", "", nil, nil); err == nil {
		t.Fatal("過長關鍵字應回傳錯誤")
	}
	since := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	until := since.Add(-time.Hour)
	if _, err := ValidateQuery("x", 0, "", "", "", &since, &until); err == nil {
		t.Fatal("until 早於 since 應回傳錯誤")
	}
}
