package friend

import (
	"context"
	"errors"
	"testing"
	"time"

	"ether.local/api/internal/profile"
)

type fakeUsers struct{ existing map[string]bool }

func (f fakeUsers) Exists(_ context.Context, userID string) (bool, error) {
	return f.existing[userID], nil
}

type fakeProfiles struct{ byUser map[string]*profile.Profile }

func (f fakeProfiles) GetByUserID(_ context.Context, userID string) (*profile.Profile, error) {
	p, ok := f.byUser[userID]
	if !ok {
		return nil, profile.ErrProfileNotFound
	}
	return p, nil
}

func (f fakeProfiles) GetByPublicID(_ context.Context, publicID string) (*profile.Profile, error) {
	for _, p := range f.byUser {
		if p.PublicID == publicID {
			return p, nil
		}
	}
	return nil, profile.ErrProfileNotFound
}

func testService() *Service {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	users := fakeUsers{existing: map[string]bool{"u-a": true, "u-b": true, "u-c": true}}
	profiles := fakeProfiles{byUser: map[string]*profile.Profile{
		"u-a": {UserID: "u-a", DisplayName: "阿哲", PublicID: "azhe"},
		"u-b": {UserID: "u-b", DisplayName: "小彤", PublicID: "xiaotong"},
		"u-c": {UserID: "u-c", DisplayName: "阿明", PublicID: "aming"},
	}}
	return NewService(NewMemoryStore(), users, profiles, func() time.Time { return now })
}

func TestSendAndAccept(t *testing.T) {
	ctx := context.Background()
	svc := testService()

	req, err := svc.SendRequest(ctx, "u-a", "u-b", "你好")
	if err != nil {
		t.Fatalf("SendRequest: %v", err)
	}
	if req.Status != RequestPending {
		t.Fatalf("status = %q", req.Status)
	}

	// 重複送出（同方向）→ 欄位錯誤，不得建第二筆。
	if _, err := svc.SendRequest(ctx, "u-a", "u-b", ""); !isValidation(err) {
		t.Fatalf("duplicate send err = %v, want ValidationError", err)
	}
	// 反方向送出（對方已有待處理邀請）→ 欄位錯誤。
	if _, err := svc.SendRequest(ctx, "u-b", "u-a", ""); !isValidation(err) {
		t.Fatalf("reverse send err = %v, want ValidationError", err)
	}

	// 非收件者不得回應。
	if _, err := svc.RespondRequest(ctx, "u-c", req.ID, true); !errors.Is(err, ErrRequestNotFound) {
		t.Fatalf("respond by stranger err = %v, want ErrRequestNotFound", err)
	}

	accepted, err := svc.RespondRequest(ctx, "u-b", req.ID, true)
	if err != nil {
		t.Fatalf("RespondRequest accept: %v", err)
	}
	if accepted.Status != RequestAccepted || accepted.RespondedAt == nil {
		t.Fatalf("accepted = %+v", accepted)
	}

	// 已回應的邀請不得再回應。
	if _, err := svc.RespondRequest(ctx, "u-b", req.ID, true); !isValidation(err) {
		t.Fatalf("re-respond err = %v, want ValidationError", err)
	}

	friends, err := svc.ListFriends(ctx, "u-a", false)
	if err != nil {
		t.Fatalf("ListFriends: %v", err)
	}
	if len(friends) != 1 || friends[0].Friend.UserID != "u-b" || friends[0].Friend.DisplayName != "小彤" {
		t.Fatalf("friends = %+v", friends)
	}

	// 已是好友，不得再送邀請。
	if _, err := svc.SendRequest(ctx, "u-a", "u-b", ""); !isValidation(err) {
		t.Fatalf("send to friend err = %v, want ValidationError", err)
	}
}

func TestSendValidation(t *testing.T) {
	ctx := context.Background()
	svc := testService()

	if _, err := svc.SendRequest(ctx, "u-a", "u-a", ""); !isValidation(err) {
		t.Fatalf("self request err = %v, want ValidationError", err)
	}
	if _, err := svc.SendRequest(ctx, "u-a", "u-ghost", ""); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ghost err = %v, want ErrUserNotFound", err)
	}
	long := ""
	for range 201 {
		long += "x"
	}
	if _, err := svc.SendRequest(ctx, "u-a", "u-b", long); !isValidation(err) {
		t.Fatalf("long message err = %v, want ValidationError", err)
	}
}

func TestDeclineAndCancel(t *testing.T) {
	ctx := context.Background()
	svc := testService()

	req, err := svc.SendRequest(ctx, "u-a", "u-b", "")
	if err != nil {
		t.Fatalf("SendRequest: %v", err)
	}
	declined, err := svc.RespondRequest(ctx, "u-b", req.ID, false)
	if err != nil {
		t.Fatalf("decline: %v", err)
	}
	if declined.Status != RequestDeclined {
		t.Fatalf("status = %q", declined.Status)
	}
	// 拒絕後可重新邀請。
	if _, err := svc.SendRequest(ctx, "u-a", "u-b", ""); err != nil {
		t.Fatalf("re-request after decline: %v", err)
	}

	req2, err := svc.SendRequest(ctx, "u-a", "u-c", "")
	if err != nil {
		t.Fatalf("SendRequest 2: %v", err)
	}
	// 收件者不得取消他人送出的邀請。
	if _, err := svc.CancelRequest(ctx, "u-c", req2.ID); !errors.Is(err, ErrRequestNotFound) {
		t.Fatalf("cancel by addressee err = %v, want ErrRequestNotFound", err)
	}
	cancelled, err := svc.CancelRequest(ctx, "u-a", req2.ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if cancelled.Status != RequestCancelled {
		t.Fatalf("status = %q", cancelled.Status)
	}
}

func TestBlockEndsFriendshipAndHidesExistence(t *testing.T) {
	ctx := context.Background()
	svc := testService()

	req, _ := svc.SendRequest(ctx, "u-a", "u-b", "")
	if _, err := svc.RespondRequest(ctx, "u-b", req.ID, true); err != nil {
		t.Fatalf("accept: %v", err)
	}
	blocked, isNew, err := svc.Block(ctx, "u-a", "u-b", "騷擾")
	if err != nil || !isNew {
		t.Fatalf("Block = %+v, %v, %v", blocked, isNew, err)
	}
	if blocked.Reason != "騷擾" || blocked.UnblockedAt != nil {
		t.Fatalf("block = %+v", blocked)
	}
	// 封鎖後好友關係解除。
	friends, _ := svc.ListFriends(ctx, "u-a", false)
	if len(friends) != 0 {
		t.Fatalf("friends after block = %+v", friends)
	}
	// 封鎖後雙向邀請皆以「不存在」回應，不得洩漏封鎖。
	if _, err := svc.SendRequest(ctx, "u-a", "u-b", ""); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("send after block err = %v, want ErrUserNotFound", err)
	}
	if _, err := svc.SendRequest(ctx, "u-b", "u-a", ""); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("reverse send after block err = %v, want ErrUserNotFound", err)
	}
	if _, err := svc.SearchByPublicID(ctx, "u-b", "azhe"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("search after block err = %v, want ErrUserNotFound", err)
	}
	// 重複封鎖冪等：回傳既有封鎖，isNew=false。
	again, isNew, err := svc.Block(ctx, "u-a", "u-b", "")
	if err != nil || isNew || again.ID != blocked.ID {
		t.Fatalf("re-block = %+v, %v, %v", again, isNew, err)
	}
	// 解除封鎖不恢復好友關係。
	if _, err := svc.Unblock(ctx, "u-a", "u-b"); err != nil {
		t.Fatalf("Unblock: %v", err)
	}
	friends, _ = svc.ListFriends(ctx, "u-a", false)
	if len(friends) != 0 {
		t.Fatalf("friends after unblock = %+v, want empty", friends)
	}
	// 解除後可重新邀請。
	if _, err := svc.SendRequest(ctx, "u-a", "u-b", ""); err != nil {
		t.Fatalf("re-request after unblock: %v", err)
	}
}

func TestBlockCancelsPendingRequests(t *testing.T) {
	ctx := context.Background()
	svc := testService()

	if _, err := svc.SendRequest(ctx, "u-a", "u-b", ""); err != nil {
		t.Fatalf("SendRequest: %v", err)
	}
	if _, _, err := svc.Block(ctx, "u-b", "u-a", ""); err != nil {
		t.Fatalf("Block: %v", err)
	}
	incoming, err := svc.ListRequests(ctx, "u-b", true)
	if err != nil {
		t.Fatalf("ListRequests: %v", err)
	}
	if len(incoming) != 0 {
		t.Fatalf("pending after block = %+v, want empty", incoming)
	}
}

func TestRemoveAndHide(t *testing.T) {
	ctx := context.Background()
	svc := testService()

	req, _ := svc.SendRequest(ctx, "u-a", "u-b", "")
	if _, err := svc.RespondRequest(ctx, "u-b", req.ID, true); err != nil {
		t.Fatalf("accept: %v", err)
	}
	// 隱藏後預設清單排除，includeHidden 列出並標示。
	if err := svc.SetHidden(ctx, "u-a", "u-b", true); err != nil {
		t.Fatalf("hide: %v", err)
	}
	friends, _ := svc.ListFriends(ctx, "u-a", false)
	if len(friends) != 0 {
		t.Fatalf("hidden friend still listed: %+v", friends)
	}
	friends, _ = svc.ListFriends(ctx, "u-a", true)
	if len(friends) != 1 || !friends[0].Hidden {
		t.Fatalf("includeHidden = %+v", friends)
	}
	if err := svc.SetHidden(ctx, "u-a", "u-b", false); err != nil {
		t.Fatalf("unhide: %v", err)
	}
	friends, _ = svc.ListFriends(ctx, "u-a", false)
	if len(friends) != 1 {
		t.Fatalf("friends after unhide = %+v", friends)
	}
	// 非好友不得隱藏。
	if err := svc.SetHidden(ctx, "u-a", "u-c", true); !errors.Is(err, ErrFriendshipNotFound) {
		t.Fatalf("hide non-friend err = %v", err)
	}
	// 解除好友。
	if err := svc.RemoveFriend(ctx, "u-a", "u-b"); err != nil {
		t.Fatalf("RemoveFriend: %v", err)
	}
	if err := svc.RemoveFriend(ctx, "u-a", "u-b"); !errors.Is(err, ErrFriendshipNotFound) {
		t.Fatalf("re-remove err = %v, want ErrFriendshipNotFound", err)
	}
	// 解除後可重新邀請（歷史列保留不阻擋）。
	if _, err := svc.SendRequest(ctx, "u-a", "u-b", ""); err != nil {
		t.Fatalf("re-request after remove: %v", err)
	}
}

func TestSearch(t *testing.T) {
	ctx := context.Background()
	svc := testService()

	found, err := svc.SearchByPublicID(ctx, "u-a", "xiaotong")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if found.UserID != "u-b" || found.DisplayName != "小彤" {
		t.Fatalf("found = %+v", found)
	}
	if _, err := svc.SearchByPublicID(ctx, "u-a", "nobody"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("missing err = %v", err)
	}
	if _, err := svc.SearchByPublicID(ctx, "u-a", "azhe"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("self search err = %v, want ErrUserNotFound", err)
	}
}

func isValidation(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

func TestAreFriendsAndBlockedBetween(t *testing.T) {
	ctx := context.Background()
	svc := testService()

	if ok, err := svc.AreFriends(ctx, "u-a", "u-b"); err != nil || ok {
		t.Fatalf("AreFriends 邀請前 = %v, %v, want false", ok, err)
	}
	req, err := svc.SendRequest(ctx, "u-a", "u-b", "")
	if err != nil {
		t.Fatalf("SendRequest: %v", err)
	}
	if _, err := svc.RespondRequest(ctx, "u-b", req.ID, true); err != nil {
		t.Fatalf("RespondRequest: %v", err)
	}
	if ok, err := svc.AreFriends(ctx, "u-b", "u-a"); err != nil || !ok {
		t.Fatalf("AreFriends 接受後 = %v, %v, want true", ok, err)
	}
	if blocked, err := svc.BlockedBetween(ctx, "u-a", "u-b"); err != nil || blocked {
		t.Fatalf("BlockedBetween 封鎖前 = %v, %v, want false", blocked, err)
	}
	if _, _, err := svc.Block(ctx, "u-a", "u-b", ""); err != nil {
		t.Fatalf("Block: %v", err)
	}
	if blocked, err := svc.BlockedBetween(ctx, "u-b", "u-a"); err != nil || !blocked {
		t.Fatalf("BlockedBetween 封鎖後 = %v, %v, want true", blocked, err)
	}
	if ok, err := svc.AreFriends(ctx, "u-a", "u-b"); err != nil || ok {
		t.Fatalf("AreFriends 封鎖後 = %v, %v, want false（封鎖解除好友關係）", ok, err)
	}
}
