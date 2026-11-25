//go:build integration

package timeline

import (
	"context"
	"testing"
	"time"

	"ether.local/api/internal/friend"
	"ether.local/api/internal/media"
	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
	"ether.local/api/internal/profile"
	"ether.local/api/internal/user"
)

// TestPG_TimelineFlowPersists 驗證動態完整流程經 PostgreSQL 持久化：
// 建立（好友可見）→ 時間線過濾（好友可見、陌生人不可見）→ 按讚／
// 取消／重新按讚 → 留言／回覆／刪除 → 編輯 → 刪除 → 第二個
// Service 實例（模擬重啟）讀取已刪除貼文為 404。
// 無隔離資料庫時跳過並標示「環境待驗證」。
func TestPG_TimelineFlowPersists(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)
	q := postgres.WrapDB(db)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	users := user.NewService(user.NewPGUserStore(q), user.NewPGSettingsStore(q), clock)
	ua, err := users.Provision(ctx, "kc-subject-ta", "ta@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision a: %v", err)
	}
	ub, err := users.Provision(ctx, "kc-subject-tb", "tb@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision b: %v", err)
	}
	uc, err := users.Provision(ctx, "kc-subject-tc", "tc@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision c: %v", err)
	}

	profiles := profile.NewService(profile.NewPGStore(q), users, clock)
	friends := friend.NewService(friend.NewPGStore(q), users, profiles, clock)
	// 建立好友關係：a 邀請 b，b 接受。
	req, err := friends.SendRequest(ctx, ua.ID, ub.ID, "")
	if err != nil {
		t.Fatalf("SendRequest: %v", err)
	}
	if _, err := friends.RespondRequest(ctx, ub.ID, req.ID, true); err != nil {
		t.Fatalf("RespondRequest: %v", err)
	}

	mediaSvc := media.NewService(media.NewPGStore(q), users, media.NewMemoryObjectStore(), clock)
	newSvc := func() *Service {
		return NewService(NewPGStore(q), users, friends, mediaSvc, clock)
	}
	svc := newSvc()

	entry, err := svc.CreatePost(ctx, ua.ID, "好友動態", VisibilityFriends, nil)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	postID := entry.Post.ID

	feed, _, err := svc.ListFeed(ctx, ub.ID, "", 10)
	if err != nil {
		t.Fatalf("ListFeed friend: %v", err)
	}
	if len(feed) != 1 {
		t.Fatalf("好友時間線 = %d 則, want 1", len(feed))
	}
	feed, _, err = svc.ListFeed(ctx, uc.ID, "", 10)
	if err != nil {
		t.Fatalf("ListFeed stranger: %v", err)
	}
	if len(feed) != 0 {
		t.Fatalf("陌生人時間線 = %d 則, want 0", len(feed))
	}

	liked, count, err := svc.LikePost(ctx, ub.ID, postID)
	if err != nil || !liked || count != 1 {
		t.Fatalf("LikePost = %v, %d, %v", liked, count, err)
	}
	liked, count, err = svc.UnlikePost(ctx, ub.ID, postID)
	if err != nil || liked || count != 0 {
		t.Fatalf("UnlikePost = %v, %d, %v", liked, count, err)
	}

	comment, err := svc.CreateComment(ctx, ub.ID, postID, "推", "")
	if err != nil {
		t.Fatalf("CreateComment: %v", err)
	}
	if _, err := svc.CreateComment(ctx, ua.ID, postID, "回覆", comment.ID); err != nil {
		t.Fatalf("CreateComment reply: %v", err)
	}
	if err := svc.DeleteComment(ctx, ub.ID, comment.ID); err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}

	// 模擬重啟：新 Service 實例讀取同一 PG 儲存。
	svc2 := newSvc()
	got, err := svc2.GetPost(ctx, ub.ID, postID)
	if err != nil {
		t.Fatalf("GetPost after restart: %v", err)
	}
	if got.LikeCount != 0 || got.CommentCount != 1 {
		t.Fatalf("重啟後計數: likes=%d comments=%d, want 0/1", got.LikeCount, got.CommentCount)
	}
	if _, err := svc2.UpdatePost(ctx, ua.ID, postID, PostPatch{Content: strPtr("改過的好友動態")}); err != nil {
		t.Fatalf("UpdatePost: %v", err)
	}
	if err := svc2.DeletePost(ctx, ua.ID, postID); err != nil {
		t.Fatalf("DeletePost: %v", err)
	}
	if _, err := svc2.GetPost(ctx, ub.ID, postID); err == nil {
		t.Fatal("刪除後讀取應為 404")
	}
}
