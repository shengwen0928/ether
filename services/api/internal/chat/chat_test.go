package chat

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeUsers struct {
	existing map[string]bool
}

func (f *fakeUsers) Exists(_ context.Context, userID string) (bool, error) {
	return f.existing[userID], nil
}

type fakeFriends struct {
	friends map[string]bool // key: a + "\x00" + b（雙向皆記）
	blocked map[string]bool
}

func pairKey(a, b string) string {
	if a < b {
		return a + "\x00" + b
	}
	return b + "\x00" + a
}

func (f *fakeFriends) AreFriends(_ context.Context, a, b string) (bool, error) {
	return f.friends[pairKey(a, b)], nil
}

func (f *fakeFriends) BlockedBetween(_ context.Context, a, b string) (bool, error) {
	return f.blocked[pairKey(a, b)], nil
}

func testService() (*Service, *fakeUsers, *fakeFriends) {
	users := &fakeUsers{existing: map[string]bool{"u-a": true, "u-b": true, "u-c": true}}
	friends := &fakeFriends{friends: map[string]bool{}, blocked: map[string]bool{}}
	fixed := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	svc := NewService(NewMemoryStore(), users, friends, func() time.Time { return fixed })
	return svc, users, friends
}

func TestCreateDirect_RequiresFriendship(t *testing.T) {
	svc, _, _ := testService()
	ctx := context.Background()
	if _, _, err := svc.CreateDirect(ctx, "u-a", "u-b"); err == nil {
		t.Fatal("非好友建一對一對話應失敗")
	} else {
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("err = %v, want ValidationError", err)
		}
	}
}

func TestCreateDirect_BlockedReturnsNotFound(t *testing.T) {
	svc, _, friends := testService()
	friends.friends[pairKey("u-a", "u-b")] = true
	friends.blocked[pairKey("u-a", "u-b")] = true
	if _, _, err := svc.CreateDirect(context.Background(), "u-a", "u-b"); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("err = %v, want ErrConversationNotFound", err)
	}
}

func TestCreateDirect_IdempotentAndRejoin(t *testing.T) {
	svc, _, friends := testService()
	friends.friends[pairKey("u-a", "u-b")] = true
	ctx := context.Background()
	conv, isNew, err := svc.CreateDirect(ctx, "u-a", "u-b")
	if err != nil || !isNew {
		t.Fatalf("CreateDirect = %+v, %v, %v", conv, isNew, err)
	}
	again, isNew, err := svc.CreateDirect(ctx, "u-b", "u-a")
	if err != nil || isNew || again.ID != conv.ID {
		t.Fatalf("重建應回傳既有對話：%+v, %v, %v", again, isNew, err)
	}
	if err := svc.Leave(ctx, "u-a", conv.ID); err != nil {
		t.Fatalf("Leave: %v", err)
	}
	if _, err := svc.GetConversation(ctx, "u-a", conv.ID); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("離開後讀取應 404，err = %v", err)
	}
	rejoined, isNew, err := svc.CreateDirect(ctx, "u-a", "u-b")
	if err != nil || isNew || rejoined.ID != conv.ID {
		t.Fatalf("重新建立應回傳既有對話並重新加入：%+v, %v, %v", rejoined, isNew, err)
	}
	if _, err := svc.GetConversation(ctx, "u-a", conv.ID); err != nil {
		t.Fatalf("重新加入後應可讀取：%v", err)
	}
}

func TestCreateGroup_ValidationAndAdmin(t *testing.T) {
	svc, _, _ := testService()
	ctx := context.Background()
	if _, err := svc.CreateGroup(ctx, "u-a", "", []string{"u-b"}); err == nil {
		t.Fatal("空標題應失敗")
	}
	if _, err := svc.CreateGroup(ctx, "u-a", "小組", nil); err == nil {
		t.Fatal("無成員應失敗")
	}
	if _, err := svc.CreateGroup(ctx, "u-a", "小組", []string{"u-ghost"}); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("不存在的成員 err = %v, want ErrConversationNotFound", err)
	}
	conv, err := svc.CreateGroup(ctx, "u-a", "小組", []string{"u-b", "u-b", "u-c"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	members, err := svc.ListMembers(ctx, "u-a", conv.ID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(members) != 3 {
		t.Fatalf("members = %d, want 3（去重後）", len(members))
	}
	for _, m := range members {
		if m.UserID == "u-a" && m.Role != RoleAdmin {
			t.Fatalf("建立者角色 = %s, want admin", m.Role)
		}
	}
}

func TestAddMembers_OnlyAdminAndGroup(t *testing.T) {
	svc, users, _ := testService()
	users.existing["u-d"] = true
	ctx := context.Background()
	conv, err := svc.CreateGroup(ctx, "u-a", "小組", []string{"u-b"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := svc.AddMembers(ctx, "u-b", conv.ID, []string{"u-c"}); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("非 admin 新增 err = %v, want ErrConversationNotFound", err)
	}
	added, err := svc.AddMembers(ctx, "u-a", conv.ID, []string{"u-c", "u-c"})
	if err != nil || len(added) != 1 {
		t.Fatalf("AddMembers = %d, %v", len(added), err)
	}
}

func TestMarkRead_NeverGoesBackwards(t *testing.T) {
	svc, _, friends := testService()
	friends.friends[pairKey("u-a", "u-b")] = true
	ctx := context.Background()
	conv, _, err := svc.CreateDirect(ctx, "u-a", "u-b")
	if err != nil {
		t.Fatalf("CreateDirect: %v", err)
	}
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	if err := svc.MarkRead(ctx, conv.ID, "u-a", "m-5", 5, now); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if err := svc.MarkRead(ctx, conv.ID, "u-a", "m-3", 3, now); err != nil {
		t.Fatalf("MarkRead 舊序號: %v", err)
	}
	seq, member, err := svc.ReadSequence(ctx, conv.ID, "u-a")
	if err != nil || !member || seq != 5 {
		t.Fatalf("水位 = %d, member = %v, err = %v, want 5", seq, member, err)
	}
}

func TestNoteMessage_UpdatesActivity(t *testing.T) {
	svc, _, friends := testService()
	friends.friends[pairKey("u-a", "u-b")] = true
	ctx := context.Background()
	conv, _, err := svc.CreateDirect(ctx, "u-a", "u-b")
	if err != nil {
		t.Fatalf("CreateDirect: %v", err)
	}
	at := time.Date(2026, 11, 25, 13, 0, 0, 0, time.UTC)
	if err := svc.NoteMessage(ctx, conv.ID, "m-1", at); err != nil {
		t.Fatalf("NoteMessage: %v", err)
	}
	got, err := svc.GetConversation(ctx, "u-a", conv.ID)
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if got.LastMessageID != "m-1" || !got.UpdatedAt.Equal(at) {
		t.Fatalf("快取未更新：%+v", got)
	}
}
