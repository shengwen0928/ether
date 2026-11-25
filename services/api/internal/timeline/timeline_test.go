package timeline

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"ether.local/api/internal/media"
)

// fakeFriends 是 FriendChecker 的測試替身：friends 為好友配對
// 集合（無序），blocked 為封鎖配對集合（無序，任一方向皆算）。
type fakeFriends struct {
	friends map[string]bool
	blocked map[string]bool
}

func pairKey(a, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

func (f *fakeFriends) AreFriends(_ context.Context, a, b string) (bool, error) {
	return f.friends[pairKey(a, b)], nil
}

func (f *fakeFriends) BlockedBetween(_ context.Context, a, b string) (bool, error) {
	return f.blocked[pairKey(a, b)], nil
}

// fakeMedia 是 MediaAccess 的測試替身：allowed 為可關聯的檔案
// 識別碼集合；其餘一律回 ErrMediaNotFound。
type fakeMedia struct {
	allowed map[string]bool
}

func (f *fakeMedia) CheckAttachable(_ context.Context, _, fileID string) (*media.MediaFile, error) {
	if f.allowed[fileID] {
		return &media.MediaFile{ID: fileID}, nil
	}
	return nil, media.ErrMediaNotFound
}

// fakeUsers 是 UserResolver 的測試替身。
type fakeUsers struct {
	existing map[string]bool
}

func (f *fakeUsers) Exists(_ context.Context, userID string) (bool, error) {
	return f.existing[userID], nil
}

func testClock() func() time.Time {
	base := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	n := 0
	return func() time.Time {
		n++
		return base.Add(time.Duration(n) * time.Second)
	}
}

func testService(friends *fakeFriends, m *fakeMedia) *Service {
	if friends == nil {
		friends = &fakeFriends{friends: map[string]bool{}, blocked: map[string]bool{}}
	}
	return NewService(NewMemoryStore(), &fakeUsers{existing: map[string]bool{
		"u-a": true, "u-b": true, "u-c": true, "u-d": true,
	}}, friends, m, testClock())
}

func TestCreatePost_Validation(t *testing.T) {
	ctx := context.Background()
	svc := testService(nil, &fakeMedia{allowed: map[string]bool{"f-1": true}})

	if _, err := svc.CreatePost(ctx, "u-a", "   ", VisibilityPublic, nil); err == nil {
		t.Fatal("內容與媒體皆空應為欄位錯誤")
	}
	if _, err := svc.CreatePost(ctx, "u-a", strings.Repeat("字", MaxPostContentRunes+1), VisibilityPublic, nil); err == nil {
		t.Fatal("內容過長應為欄位錯誤")
	}
	if _, err := svc.CreatePost(ctx, "u-a", "hi", "everyone", nil); err == nil {
		t.Fatal("非法能見度應為欄位錯誤")
	}
	dupIDs := make([]string, 0, MaxPostMedia+1)
	for i := 0; i <= MaxPostMedia; i++ {
		dupIDs = append(dupIDs, "f-1")
	}
	if _, err := svc.CreatePost(ctx, "u-a", "hi", VisibilityPublic, dupIDs); err != nil {
		t.Fatalf("重複媒體識別碼去重後應可建立，現為 %v", err)
	}
	many := []string{"f-1", "f-2", "f-3", "f-4", "f-5", "f-6", "f-7", "f-8", "f-9", "f-10"}
	if _, err := svc.CreatePost(ctx, "u-a", "hi", VisibilityPublic, many); err == nil {
		t.Fatal("超過媒體上限應為欄位錯誤")
	}
	if _, err := svc.CreatePost(ctx, "u-a", "hi", VisibilityPublic, []string{"f-missing"}); !errors.Is(err, media.ErrMediaNotFound) {
		t.Fatalf("未通過掃描閘門的檔案應回 Media 錯誤，現為 %v", err)
	}
	// 純媒體貼文可建立；能見度為空時預設 friends。
	entry, err := svc.CreatePost(ctx, "u-a", "", "", []string{"f-1"})
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if entry.Post.Visibility != VisibilityFriends {
		t.Fatalf("預設能見度 = %s, want friends", entry.Post.Visibility)
	}
	if len(entry.Media) != 1 || entry.Media[0].MediaFileID != "f-1" {
		t.Fatalf("media = %+v", entry.Media)
	}
}

func TestVisibilityGuards(t *testing.T) {
	ctx := context.Background()
	friends := &fakeFriends{
		friends: map[string]bool{pairKey("u-a", "u-b"): true},
		blocked: map[string]bool{pairKey("u-a", "u-c"): true},
	}
	svc := testService(friends, nil)

	pub, _ := svc.CreatePost(ctx, "u-a", "公開", VisibilityPublic, nil)
	fri, _ := svc.CreatePost(ctx, "u-a", "好友", VisibilityFriends, nil)
	own, _ := svc.CreatePost(ctx, "u-a", "自己", VisibilityOnlyMe, nil)

	// 公開貼文：陌生人可見，被封鎖者不可見。
	if _, err := svc.GetPost(ctx, "u-d", pub.Post.ID); err != nil {
		t.Fatalf("陌生人讀公開貼文: %v", err)
	}
	if _, err := svc.GetPost(ctx, "u-c", pub.Post.ID); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("被封鎖者讀公開貼文應 404，現為 %v", err)
	}
	// 好友貼文：好友可見，陌生人 404。
	if _, err := svc.GetPost(ctx, "u-b", fri.Post.ID); err != nil {
		t.Fatalf("好友讀好友貼文: %v", err)
	}
	if _, err := svc.GetPost(ctx, "u-d", fri.Post.ID); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("陌生人讀好友貼文應 404，現為 %v", err)
	}
	// 僅自己：作者可見，他人 404。
	if _, err := svc.GetPost(ctx, "u-a", own.Post.ID); err != nil {
		t.Fatalf("作者讀自己的貼文: %v", err)
	}
	if _, err := svc.GetPost(ctx, "u-b", own.Post.ID); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("好友讀僅自己貼文應 404，現為 %v", err)
	}
	// 檢查器未接線時 fail-closed（在同一服務內建公開貼文，
	// 他人讀取須回 ErrCheckerUnavailable，而非放行或 404）。
	svcNoChecker := NewService(NewMemoryStore(), &fakeUsers{existing: map[string]bool{"u-a": true}}, nil, nil, testClock())
	ncEntry, err := svcNoChecker.CreatePost(ctx, "u-a", "公開", VisibilityPublic, nil)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if _, err := svcNoChecker.GetPost(ctx, "u-b", ncEntry.Post.ID); !errors.Is(err, ErrCheckerUnavailable) {
		t.Fatalf("無檢查器應 fail-closed，現為 %v", err)
	}
}

func TestUpdateAndDeletePost(t *testing.T) {
	ctx := context.Background()
	svc := testService(nil, nil)

	entry, err := svc.CreatePost(ctx, "u-a", "原文", VisibilityPublic, nil)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	id := entry.Post.ID

	if _, err := svc.UpdatePost(ctx, "u-b", id, PostPatch{Content: strPtr("竄改")}); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("非作者編輯應 404，現為 %v", err)
	}
	updated, err := svc.UpdatePost(ctx, "u-a", id, PostPatch{Content: strPtr("改過"), Visibility: visPtr(VisibilityFriends)})
	if err != nil {
		t.Fatalf("UpdatePost: %v", err)
	}
	if updated.Post.Content != "改過" || updated.Post.Visibility != VisibilityFriends || updated.Post.EditedAt == nil {
		t.Fatalf("updated = %+v", updated.Post)
	}
	if _, err := svc.UpdatePost(ctx, "u-a", id, PostPatch{Content: strPtr("  ")}); err == nil {
		t.Fatal("無媒體貼文內容清空應為欄位錯誤")
	}
	if err := svc.DeletePost(ctx, "u-b", id); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("非作者刪除應 404，現為 %v", err)
	}
	if err := svc.DeletePost(ctx, "u-a", id); err != nil {
		t.Fatalf("DeletePost: %v", err)
	}
	if _, err := svc.GetPost(ctx, "u-a", id); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("已刪除貼文作者讀取應 404，現為 %v", err)
	}
	if _, err := svc.UpdatePost(ctx, "u-a", id, PostPatch{Content: strPtr("再改")}); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("已刪除貼文編輯應 404，現為 %v", err)
	}
}

func strPtr(s string) *string { return &s }

func visPtr(v Visibility) *Visibility { return &v }

func TestFeedPaginationAndFiltering(t *testing.T) {
	ctx := context.Background()
	friends := &fakeFriends{
		friends: map[string]bool{pairKey("u-a", "u-b"): true},
		blocked: map[string]bool{},
	}
	svc := testService(friends, nil)

	// u-b：好友貼文（可見）＋公開貼文（可見）＋僅自己（不可見）。
	// u-d：陌生人公開貼文（可見）＋好友貼文（不可見）。
	// 建立順序即時間順序（testClock 遞增）。
	if _, err := svc.CreatePost(ctx, "u-b", "b1 好友", VisibilityFriends, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreatePost(ctx, "u-d", "d1 好友", VisibilityFriends, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreatePost(ctx, "u-b", "b2 公開", VisibilityPublic, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreatePost(ctx, "u-d", "d2 公開", VisibilityPublic, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreatePost(ctx, "u-b", "b3 自己", VisibilityOnlyMe, nil); err != nil {
		t.Fatal(err)
	}

	page1, next, err := svc.ListFeed(ctx, "u-a", "", 2)
	if err != nil {
		t.Fatalf("ListFeed: %v", err)
	}
	if len(page1) != 2 || next == "" {
		t.Fatalf("page1 = %d 筆, next = %q, want 2 筆且有下一頁", len(page1), next)
	}
	if page1[0].Post.Content != "d2 公開" || page1[1].Post.Content != "b2 公開" {
		t.Fatalf("page1 順序錯誤: %q, %q", page1[0].Post.Content, page1[1].Post.Content)
	}
	page2, next2, err := svc.ListFeed(ctx, "u-a", next, 2)
	if err != nil {
		t.Fatalf("ListFeed p2: %v", err)
	}
	if len(page2) != 1 || next2 != "" {
		t.Fatalf("page2 = %d 筆, next = %q, want 1 筆且無下一頁", len(page2), next2)
	}
	if page2[0].Post.Content != "b1 好友" {
		t.Fatalf("page2 = %q, want b1 好友", page2[0].Post.Content)
	}
}

func TestListAuthorPosts(t *testing.T) {
	ctx := context.Background()
	svc := testService(nil, nil)
	if _, err := svc.CreatePost(ctx, "u-a", "a1", VisibilityPublic, nil); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.ListAuthorPosts(ctx, "u-b", "u-missing", "", 10); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("未知作者應 404，現為 %v", err)
	}
	entries, next, err := svc.ListAuthorPosts(ctx, "u-b", "u-a", "", 10)
	if err != nil {
		t.Fatalf("ListAuthorPosts: %v", err)
	}
	if len(entries) != 1 || next != "" {
		t.Fatalf("entries = %d, next = %q", len(entries), next)
	}
	if _, err := svc.CreatePost(ctx, "u-a", "a2 僅自己", VisibilityOnlyMe, nil); err != nil {
		t.Fatal(err)
	}
	entries, _, err = svc.ListAuthorPosts(ctx, "u-b", "u-a", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("陌生人應只見 1 則公開貼文，現為 %d", len(entries))
	}
}

func TestLikes(t *testing.T) {
	ctx := context.Background()
	svc := testService(nil, nil)
	entry, _ := svc.CreatePost(ctx, "u-a", "讚我", VisibilityPublic, nil)
	id := entry.Post.ID

	liked, count, err := svc.LikePost(ctx, "u-b", id)
	if err != nil || !liked || count != 1 {
		t.Fatalf("LikePost = %v, %d, %v", liked, count, err)
	}
	liked, count, err = svc.LikePost(ctx, "u-b", id)
	if err != nil || !liked || count != 1 {
		t.Fatalf("重複按讚應冪等: %v, %d, %v", liked, count, err)
	}
	liked, count, err = svc.UnlikePost(ctx, "u-b", id)
	if err != nil || liked || count != 0 {
		t.Fatalf("UnlikePost = %v, %d, %v", liked, count, err)
	}
	liked, count, err = svc.UnlikePost(ctx, "u-b", id)
	if err != nil || liked || count != 0 {
		t.Fatalf("重複取消應冪等: %v, %d, %v", liked, count, err)
	}
	liked, count, err = svc.LikePost(ctx, "u-b", id)
	if err != nil || !liked || count != 1 {
		t.Fatalf("重新按讚: %v, %d, %v", liked, count, err)
	}
	got, err := svc.GetPost(ctx, "u-b", id)
	if err != nil {
		t.Fatal(err)
	}
	if got.LikeCount != 1 || !got.LikedByMe {
		t.Fatalf("entry = %+v", got)
	}
	// 不可見貼文不得按讚。
	hidden, _ := svc.CreatePost(ctx, "u-a", "藏", VisibilityOnlyMe, nil)
	if _, _, err := svc.LikePost(ctx, "u-b", hidden.Post.ID); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("不可見貼文按讚應 404，現為 %v", err)
	}
}

func TestComments(t *testing.T) {
	ctx := context.Background()
	svc := testService(nil, nil)
	entry, _ := svc.CreatePost(ctx, "u-a", "留言區", VisibilityPublic, nil)
	id := entry.Post.ID
	other, _ := svc.CreatePost(ctx, "u-a", "另一則", VisibilityPublic, nil)

	c1, err := svc.CreateComment(ctx, "u-b", id, "第一則", "")
	if err != nil {
		t.Fatalf("CreateComment: %v", err)
	}
	if _, err := svc.CreateComment(ctx, "u-b", id, "   ", ""); err == nil {
		t.Fatal("空留言應為欄位錯誤")
	}
	if _, err := svc.CreateComment(ctx, "u-b", id, strings.Repeat("字", MaxCommentContentRunes+1), ""); err == nil {
		t.Fatal("留言過長應為欄位錯誤")
	}
	reply, err := svc.CreateComment(ctx, "u-c", id, "回覆", c1.ID)
	if err != nil {
		t.Fatalf("CreateComment reply: %v", err)
	}
	if reply.ParentCommentID != c1.ID {
		t.Fatalf("reply parent = %q", reply.ParentCommentID)
	}
	// 跨貼文回覆應拒絕。
	if _, err := svc.CreateComment(ctx, "u-b", other.Post.ID, "錯", c1.ID); err == nil {
		t.Fatal("跨貼文回覆應為欄位錯誤")
	}
	// 留言清單升序＋游標分頁。
	page1, next, err := svc.ListComments(ctx, "u-b", id, "", 1)
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if len(page1) != 1 || next == "" || page1[0].ID != c1.ID {
		t.Fatalf("page1 = %+v, next = %q", page1, next)
	}
	page2, next2, err := svc.ListComments(ctx, "u-b", id, next, 1)
	if err != nil {
		t.Fatalf("ListComments p2: %v", err)
	}
	if len(page2) != 1 || next2 != "" || page2[0].ID != reply.ID {
		t.Fatalf("page2 = %+v, next = %q", page2, next2)
	}
	// 刪除：非作者 404；作者刪除後內容遮蔽、計數排除。
	if err := svc.DeleteComment(ctx, "u-c", c1.ID); !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("非作者刪除留言應 404，現為 %v", err)
	}
	if err := svc.DeleteComment(ctx, "u-b", c1.ID); err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}
	rows, _, err := svc.ListComments(ctx, "u-b", id, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Content != "" || !rows[0].IsDeleted {
		t.Fatalf("已刪除留言應列出但內容遮蔽: %+v", rows)
	}
	got, err := svc.GetPost(ctx, "u-b", id)
	if err != nil {
		t.Fatal(err)
	}
	if got.CommentCount != 1 {
		t.Fatalf("comment count = %d, want 1", got.CommentCount)
	}
	// 不可見貼文不得留言。
	hidden, _ := svc.CreatePost(ctx, "u-a", "藏", VisibilityOnlyMe, nil)
	if _, err := svc.CreateComment(ctx, "u-b", hidden.Post.ID, "hi", ""); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("不可見貼文留言應 404，現為 %v", err)
	}
}
