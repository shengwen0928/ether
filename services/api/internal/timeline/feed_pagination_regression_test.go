package timeline

import (
	"context"
	"testing"
)

// 回歸：feedPage 回傳游標取最後一筆回傳列（2026-11-25 修正）。
// 舊寫法取最後一個候選列，五筆全可見、limit 2 時合計只取回
// 四筆（第三筆被跳過）。兩案皆走完所有頁，不得漏列、不得重複。
func TestFeedPaginationAllVisible_NoSkippedRows(t *testing.T) {
	ctx := context.Background()
	friends := &fakeFriends{
		friends: map[string]bool{pairKey("u-a", "u-b"): true},
		blocked: map[string]bool{},
	}
	svc := testService(friends, nil)
	for _, c := range []string{"p1", "p2", "p3", "p4", "p5"} {
		if _, err := svc.CreatePost(ctx, "u-b", c, VisibilityPublic, nil); err != nil {
			t.Fatal(err)
		}
	}
	var got []string
	cursor := ""
	for i := 0; i < 10; i++ {
		page, next, err := svc.ListFeed(ctx, "u-a", cursor, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range page {
			got = append(got, e.Post.Content)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	want := []string{"p5", "p4", "p3", "p2", "p1"}
	if len(got) != len(want) {
		t.Fatalf("取回 %d 筆 %v，want %v", len(got), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("第 %d 筆 = %q，want %q（全序 %v）", i, got[i], want[i], got)
		}
	}
}

// 回歸：被濾除列夾在可見列之間時，分頁不得漏列、不得原地重掃。
func TestFeedPaginationFilteredMiddle_NoSkippedRows(t *testing.T) {
	ctx := context.Background()
	friends := &fakeFriends{
		friends: map[string]bool{pairKey("u-a", "u-b"): true},
		blocked: map[string]bool{},
	}
	svc := testService(friends, nil)
	// 建立順序即時間順序：q3 為僅自己（u-a 不可見），夾在中間。
	for _, tc := range []struct {
		content string
		vis     Visibility
	}{
		{"q1", VisibilityPublic},
		{"q2", VisibilityPublic},
		{"q3", VisibilityOnlyMe},
		{"q4", VisibilityPublic},
		{"q5", VisibilityPublic},
	} {
		if _, err := svc.CreatePost(ctx, "u-b", tc.content, tc.vis, nil); err != nil {
			t.Fatal(err)
		}
	}
	var got []string
	cursor := ""
	for i := 0; i < 10; i++ {
		page, next, err := svc.ListFeed(ctx, "u-a", cursor, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range page {
			got = append(got, e.Post.Content)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	want := []string{"q5", "q4", "q2", "q1"}
	if len(got) != len(want) {
		t.Fatalf("取回 %d 筆 %v，want %v", len(got), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("第 %d 筆 = %q，want %q（全序 %v）", i, got[i], want[i], got)
		}
	}
}
