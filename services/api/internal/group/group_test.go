package group

import (
	"context"
	"errors"
	"fmt"
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
	blocked map[string]bool // key: a + "\x00" + b（雙向皆記）
}

func blockedKey(a, b string) string {
	if a < b {
		return a + "\x00" + b
	}
	return b + "\x00" + a
}

func (f *fakeFriends) BlockedBetween(_ context.Context, a, b string) (bool, error) {
	return f.blocked[blockedKey(a, b)], nil
}

type fakeChat struct {
	next     int
	created  []string // 建立的對話識別碼（依序）
	added    map[string][]string
	removed  map[string][]string
	failNext error
}

func (f *fakeChat) CreateGroupConversation(_ context.Context, _, _ string, _ []string) (string, error) {
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return "", err
	}
	f.next++
	id := fmt.Sprintf("conv-%d", f.next)
	f.created = append(f.created, id)
	return id, nil
}

func (f *fakeChat) AddConversationMembers(_ context.Context, conversationID string, memberIDs []string) error {
	if f.added == nil {
		f.added = map[string][]string{}
	}
	f.added[conversationID] = append(f.added[conversationID], memberIDs...)
	return nil
}

func (f *fakeChat) RemoveConversationMember(_ context.Context, conversationID, userID string) error {
	if f.removed == nil {
		f.removed = map[string][]string{}
	}
	f.removed[conversationID] = append(f.removed[conversationID], userID)
	return nil
}

func testService() (*Service, *fakeUsers, *fakeFriends, *fakeChat) {
	users := &fakeUsers{existing: map[string]bool{"u-a": true, "u-b": true, "u-c": true, "u-d": true}}
	friends := &fakeFriends{blocked: map[string]bool{}}
	chats := &fakeChat{}
	fixed := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	svc := NewService(NewMemoryStore(), users, friends, chats, func() time.Time { return fixed })
	return svc, users, friends, chats
}

func mustGroup(t *testing.T, svc *Service, ctx context.Context) *GroupEntry {
	t.Helper()
	entry, err := svc.CreateGroup(ctx, "u-a", "登山小隊", "每週爬山", "", []string{"u-b", "u-c"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	return entry
}

func isValidationErr(t *testing.T, err error) {
	t.Helper()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
}

func TestCreateGroup_RolesAndConversation(t *testing.T) {
	svc, _, _, chats := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	if entry.Group.ConversationID == "" {
		t.Fatal("ConversationID 為空，應經 Chat 介面建立容器")
	}
	if len(chats.created) != 1 || chats.created[0] != entry.Group.ConversationID {
		t.Fatalf("chats.created = %v", chats.created)
	}
	if entry.MemberCount != 3 {
		t.Fatalf("MemberCount = %d, want 3", entry.MemberCount)
	}
	members, err := svc.ListMembers(ctx, "u-a", entry.Group.ID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	roles := map[string]Role{}
	for _, m := range members {
		roles[m.UserID] = m.Role
	}
	if roles["u-a"] != RoleOwner || roles["u-b"] != RoleMember || roles["u-c"] != RoleMember {
		t.Fatalf("roles = %v", roles)
	}
}

func TestCreateGroup_RequiresOtherMember(t *testing.T) {
	svc, _, _, _ := testService()
	_, err := svc.CreateGroup(context.Background(), "u-a", "空群組", "", "", nil)
	isValidationErr(t, err)
}

func TestCreateGroup_UnknownOrBlockedMemberReturnsNotFound(t *testing.T) {
	svc, _, friends, _ := testService()
	ctx := context.Background()
	if _, err := svc.CreateGroup(ctx, "u-a", "群組", "", "", []string{"u-x"}); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("unknown member err = %v, want ErrGroupNotFound", err)
	}
	friends.blocked[blockedKey("u-a", "u-b")] = true
	if _, err := svc.CreateGroup(ctx, "u-a", "群組", "", "", []string{"u-b"}); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("blocked member err = %v, want ErrGroupNotFound", err)
	}
}

func TestCreateGroup_ValidationLimits(t *testing.T) {
	svc, _, _, _ := testService()
	ctx := context.Background()
	if _, err := svc.CreateGroup(ctx, "u-a", "", "", "", []string{"u-b"}); err == nil {
		t.Fatal("空名稱應失敗")
	} else {
		isValidationErr(t, err)
	}
	long := ""
	for i := 0; i < 501; i++ {
		long += "字"
	}
	if _, err := svc.CreateGroup(ctx, "u-a", "群組", long, "", []string{"u-b"}); err == nil {
		t.Fatal("過長介紹應失敗")
	} else {
		isValidationErr(t, err)
	}
}

func TestNonMemberReadsNotFound(t *testing.T) {
	svc, _, _, _ := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	for _, err := range []error{
		func() error { _, e := svc.GetGroup(ctx, "u-d", entry.Group.ID); return e }(),
		func() error { _, e := svc.ListMembers(ctx, "u-d", entry.Group.ID); return e }(),
		func() error { _, e := svc.ListAnnouncements(ctx, "u-d", entry.Group.ID); return e }(),
		func() error { _, e := svc.GetSettings(ctx, "u-d", entry.Group.ID); return e }(),
	} {
		if !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("outsider err = %v, want ErrGroupNotFound", err)
		}
	}
}

func TestRoleMatrix_MemberCannotManage(t *testing.T) {
	svc, _, _, _ := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	gid := entry.Group.ID
	patch := GroupPatch{}
	name := "新名稱"
	patch.Name = &name
	cases := []struct {
		name string
		call func() error
	}{
		{"update", func() error { _, e := svc.UpdateGroup(ctx, "u-b", gid, patch); return e }},
		{"add", func() error { _, e := svc.AddMembers(ctx, "u-b", gid, []string{"u-d"}); return e }},
		{"setrole", func() error { _, e := svc.SetRole(ctx, "u-b", gid, "u-c", RoleAdmin); return e }},
		{"remove", func() error { return svc.RemoveMember(ctx, "u-b", gid, "u-c") }},
		{"announce", func() error { _, e := svc.CreateAnnouncement(ctx, "u-b", gid, "公告"); return e }},
		{"settings", func() error { p := InvitePolicyAll; _, e := svc.UpdateSettings(ctx, "u-b", gid, &p); return e }},
		{"transfer", func() error { _, e := svc.TransferOwnership(ctx, "u-b", gid, "u-c"); return e }},
		{"disband", func() error { return svc.Disband(ctx, "u-b", gid) }},
	}
	for _, c := range cases {
		if err := c.call(); !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("%s: err = %v, want ErrGroupNotFound", c.name, err)
		}
	}
}

func TestAdminCanManageButNotTransferOrDisband(t *testing.T) {
	svc, _, _, _ := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	gid := entry.Group.ID
	if _, err := svc.SetRole(ctx, "u-a", gid, "u-b", RoleAdmin); err != nil {
		t.Fatalf("SetRole: %v", err)
	}
	// 管理員可新增成員、發佈公告、更新群組。
	if _, err := svc.AddMembers(ctx, "u-b", gid, []string{"u-d"}); err != nil {
		t.Fatalf("admin AddMembers: %v", err)
	}
	if _, err := svc.CreateAnnouncement(ctx, "u-b", gid, "週六集合"); err != nil {
		t.Fatalf("admin CreateAnnouncement: %v", err)
	}
	name := "登山小隊 2"
	if _, err := svc.UpdateGroup(ctx, "u-b", gid, GroupPatch{Name: &name}); err != nil {
		t.Fatalf("admin UpdateGroup: %v", err)
	}
	// 管理員不得轉移所有權、解散、變更角色。
	if _, err := svc.TransferOwnership(ctx, "u-b", gid, "u-c"); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("admin transfer err = %v, want ErrGroupNotFound", err)
	}
	if err := svc.Disband(ctx, "u-b", gid); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("admin disband err = %v, want ErrGroupNotFound", err)
	}
	if _, err := svc.SetRole(ctx, "u-b", gid, "u-c", RoleAdmin); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("admin setrole err = %v, want ErrGroupNotFound", err)
	}
}

func TestTransferOwnership(t *testing.T) {
	svc, _, _, _ := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	gid := entry.Group.ID
	if _, err := svc.TransferOwnership(ctx, "u-a", gid, "u-b"); err != nil {
		t.Fatalf("TransferOwnership: %v", err)
	}
	members, _ := svc.ListMembers(ctx, "u-a", gid)
	roles := map[string]Role{}
	for _, m := range members {
		roles[m.UserID] = m.Role
	}
	if roles["u-b"] != RoleOwner || roles["u-a"] != RoleAdmin {
		t.Fatalf("roles after transfer = %v", roles)
	}
	// 原擁有者已不得解散；新擁有者可以。
	if err := svc.Disband(ctx, "u-a", gid); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("old owner disband err = %v, want ErrGroupNotFound", err)
	}
	if err := svc.Disband(ctx, "u-b", gid); err != nil {
		t.Fatalf("new owner Disband: %v", err)
	}
	if _, err := svc.GetGroup(ctx, "u-a", gid); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("disbanded GetGroup err = %v, want ErrGroupNotFound", err)
	}
}

func TestOwnerCannotLeaveAndCannotBeRemoved(t *testing.T) {
	svc, _, _, _ := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	gid := entry.Group.ID
	if err := svc.Leave(ctx, "u-a", gid); err == nil {
		t.Fatal("擁有者直接離開應失敗")
	} else {
		isValidationErr(t, err)
	}
	if _, err := svc.SetRole(ctx, "u-a", gid, "u-b", RoleAdmin); err != nil {
		t.Fatalf("SetRole: %v", err)
	}
	if err := svc.RemoveMember(ctx, "u-b", gid, "u-a"); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("remove owner err = %v, want ErrGroupNotFound", err)
	}
	if err := svc.RemoveMember(ctx, "u-a", gid, "u-a"); err == nil {
		t.Fatal("移除自己應失敗")
	} else {
		isValidationErr(t, err)
	}
}

func TestRemoveMember_AdminCannotRemoveAdmin(t *testing.T) {
	svc, _, _, chats := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	gid := entry.Group.ID
	if _, err := svc.SetRole(ctx, "u-a", gid, "u-b", RoleAdmin); err != nil {
		t.Fatalf("SetRole: %v", err)
	}
	if _, err := svc.SetRole(ctx, "u-a", gid, "u-c", RoleAdmin); err != nil {
		t.Fatalf("SetRole: %v", err)
	}
	if err := svc.RemoveMember(ctx, "u-b", gid, "u-c"); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("admin remove admin err = %v, want ErrGroupNotFound", err)
	}
	if err := svc.RemoveMember(ctx, "u-a", gid, "u-c"); err != nil {
		t.Fatalf("owner RemoveMember: %v", err)
	}
	if len(chats.removed[entry.Group.ConversationID]) != 1 || chats.removed[entry.Group.ConversationID][0] != "u-c" {
		t.Fatalf("conversation sync removed = %v", chats.removed)
	}
	// 被移除者再讀群組為 404；重新加入後可讀。
	if _, err := svc.GetGroup(ctx, "u-c", gid); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("removed GetGroup err = %v, want ErrGroupNotFound", err)
	}
	if _, err := svc.AddMembers(ctx, "u-a", gid, []string{"u-c"}); err != nil {
		t.Fatalf("re-add: %v", err)
	}
	members, _ := svc.ListMembers(ctx, "u-a", gid)
	for _, m := range members {
		if m.UserID == "u-c" && m.Role != RoleMember {
			t.Fatalf("rejoined role = %s, want member", m.Role)
		}
	}
}

func TestLeave_SyncsConversation(t *testing.T) {
	svc, _, _, chats := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	if err := svc.Leave(ctx, "u-b", entry.Group.ID); err != nil {
		t.Fatalf("Leave: %v", err)
	}
	if len(chats.removed[entry.Group.ConversationID]) != 1 {
		t.Fatalf("conversation sync removed = %v", chats.removed)
	}
}

func TestSetRole_Rules(t *testing.T) {
	svc, _, _, _ := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	gid := entry.Group.ID
	if _, err := svc.SetRole(ctx, "u-a", gid, "u-b", RoleOwner); err == nil {
		t.Fatal("以 SetRole 授與 owner 應失敗")
	} else {
		isValidationErr(t, err)
	}
	if _, err := svc.SetRole(ctx, "u-a", gid, "u-a", RoleAdmin); err == nil {
		t.Fatal("變更自己角色應失敗")
	} else {
		isValidationErr(t, err)
	}
	if _, err := svc.SetRole(ctx, "u-a", gid, "u-d", RoleAdmin); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("non-member setrole err = %v, want ErrGroupNotFound", err)
	}
	m, err := svc.SetRole(ctx, "u-a", gid, "u-b", RoleAdmin)
	if err != nil {
		t.Fatalf("SetRole: %v", err)
	}
	if m.Role != RoleAdmin {
		t.Fatalf("role = %s", m.Role)
	}
	// 相同角色為 no-op。
	if _, err := svc.SetRole(ctx, "u-a", gid, "u-b", RoleAdmin); err != nil {
		t.Fatalf("idempotent SetRole: %v", err)
	}
}

func TestInvitationLifecycle(t *testing.T) {
	svc, _, friends, chats := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	gid := entry.Group.ID

	// 預設政策下一般成員不得邀請。
	if _, err := svc.CreateInvitation(ctx, "u-b", gid, "u-d"); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("member invite err = %v, want ErrGroupNotFound", err)
	}
	inv, err := svc.CreateInvitation(ctx, "u-a", gid, "u-d")
	if err != nil {
		t.Fatalf("CreateInvitation: %v", err)
	}
	if _, err := svc.CreateInvitation(ctx, "u-a", gid, "u-d"); err == nil {
		t.Fatal("重複邀請應失敗")
	} else {
		isValidationErr(t, err)
	}
	// 非被邀請者不得回應；管理者清單可見。
	if _, err := svc.RespondInvitation(ctx, "u-b", inv.Invitation.ID, true); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("wrong invitee err = %v, want ErrGroupNotFound", err)
	}
	rows, err := svc.ListGroupInvitations(ctx, "u-a", gid)
	if err != nil || len(rows) != 1 || rows[0].GroupName != "登山小隊" {
		t.Fatalf("ListGroupInvitations = %+v, %v", rows, err)
	}
	mine, err := svc.ListMyInvitations(ctx, "u-d")
	if err != nil || len(mine) != 1 {
		t.Fatalf("ListMyInvitations = %+v, %v", mine, err)
	}
	accepted, err := svc.RespondInvitation(ctx, "u-d", inv.Invitation.ID, true)
	if err != nil {
		t.Fatalf("RespondInvitation: %v", err)
	}
	if accepted.Invitation.Status != InvitationAccepted {
		t.Fatalf("status = %s", accepted.Invitation.Status)
	}
	if len(chats.added[entry.Group.ConversationID]) != 1 || chats.added[entry.Group.ConversationID][0] != "u-d" {
		t.Fatalf("conversation sync added = %v", chats.added)
	}
	// 已回應的邀請不得再次回應。
	if _, err := svc.RespondInvitation(ctx, "u-d", inv.Invitation.ID, false); err == nil {
		t.Fatal("重複回應應失敗")
	} else {
		isValidationErr(t, err)
	}
	// 已是成員不得再邀請。
	if _, err := svc.CreateInvitation(ctx, "u-a", gid, "u-d"); err == nil {
		t.Fatal("邀請現有成員應失敗")
	} else {
		isValidationErr(t, err)
	}
	_ = friends
}

func TestInvitation_DeclineAndCancel(t *testing.T) {
	svc, _, _, _ := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	gid := entry.Group.ID
	inv, err := svc.CreateInvitation(ctx, "u-a", gid, "u-d")
	if err != nil {
		t.Fatalf("CreateInvitation: %v", err)
	}
	declined, err := svc.RespondInvitation(ctx, "u-d", inv.Invitation.ID, false)
	if err != nil {
		t.Fatalf("decline: %v", err)
	}
	if declined.Invitation.Status != InvitationDeclined {
		t.Fatalf("status = %s", declined.Invitation.Status)
	}
	inv2, err := svc.CreateInvitation(ctx, "u-a", gid, "u-d")
	if err != nil {
		t.Fatalf("re-invite after decline: %v", err)
	}
	// 非邀請者、非管理者不得取消。
	if _, err := svc.CancelInvitation(ctx, "u-b", inv2.Invitation.ID); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("cancel by member err = %v, want ErrGroupNotFound", err)
	}
	cancelled, err := svc.CancelInvitation(ctx, "u-a", inv2.Invitation.ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if cancelled.Invitation.Status != InvitationCancelled {
		t.Fatalf("status = %s", cancelled.Invitation.Status)
	}
}

func TestInvitation_BlockedReturnsNotFound(t *testing.T) {
	svc, _, friends, _ := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	friends.blocked[blockedKey("u-a", "u-d")] = true
	if _, err := svc.CreateInvitation(ctx, "u-a", entry.Group.ID, "u-d"); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("blocked invite err = %v, want ErrGroupNotFound", err)
	}
}

func TestInvitePolicyAll_AllowsMemberInvite(t *testing.T) {
	svc, _, _, _ := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	gid := entry.Group.ID
	p := InvitePolicyAll
	if _, err := svc.UpdateSettings(ctx, "u-a", gid, &p); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if _, err := svc.CreateInvitation(ctx, "u-b", gid, "u-d"); err != nil {
		t.Fatalf("member invite with policy all: %v", err)
	}
}

func TestAnnouncement_SingleActive(t *testing.T) {
	svc, _, _, _ := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	gid := entry.Group.ID
	first, err := svc.CreateAnnouncement(ctx, "u-a", gid, "第一則")
	if err != nil {
		t.Fatalf("CreateAnnouncement: %v", err)
	}
	second, err := svc.CreateAnnouncement(ctx, "u-a", gid, "第二則")
	if err != nil {
		t.Fatalf("CreateAnnouncement 2: %v", err)
	}
	rows, err := svc.ListAnnouncements(ctx, "u-b", gid)
	if err != nil {
		t.Fatalf("ListAnnouncements: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("announcements = %d, want 2", len(rows))
	}
	active := 0
	for _, a := range rows {
		if a.IsActive {
			active++
			if a.ID != second.ID {
				t.Fatalf("active = %s, want %s", a.ID, second.ID)
			}
		}
	}
	if active != 1 {
		t.Fatalf("active count = %d, want 1", active)
	}
	// 已撤下的公告不得編輯或再次撤下。
	if _, err := svc.UpdateAnnouncement(ctx, "u-a", gid, first.ID, "改"); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("edit retracted err = %v, want ErrGroupNotFound", err)
	}
	if _, err := svc.RetractAnnouncement(ctx, "u-a", gid, first.ID); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("retract retracted err = %v, want ErrGroupNotFound", err)
	}
	// 現行公告可編輯（發佈時間不變）。
	edited, err := svc.UpdateAnnouncement(ctx, "u-a", gid, second.ID, "第二則修訂")
	if err != nil {
		t.Fatalf("UpdateAnnouncement: %v", err)
	}
	if edited.Content != "第二則修訂" || !edited.PublishedAt.Equal(second.PublishedAt) {
		t.Fatalf("edited = %+v", edited)
	}
	retracted, err := svc.RetractAnnouncement(ctx, "u-a", gid, second.ID)
	if err != nil {
		t.Fatalf("RetractAnnouncement: %v", err)
	}
	if retracted.IsActive || retracted.RetractedAt == nil {
		t.Fatalf("retracted = %+v", retracted)
	}
	if _, err := svc.CreateAnnouncement(ctx, "u-a", gid, ""); err == nil {
		t.Fatal("空公告應失敗")
	} else {
		isValidationErr(t, err)
	}
}

func TestSettings_DefaultAndValidation(t *testing.T) {
	svc, _, _, _ := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	gid := entry.Group.ID
	st, err := svc.GetSettings(ctx, "u-b", gid)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if st.InvitePolicy != InvitePolicyAdmin {
		t.Fatalf("default policy = %s", st.InvitePolicy)
	}
	bad := InvitePolicy("nobody")
	if _, err := svc.UpdateSettings(ctx, "u-a", gid, &bad); err == nil {
		t.Fatal("非法政策值應失敗")
	} else {
		isValidationErr(t, err)
	}
	p := InvitePolicyAll
	updated, err := svc.UpdateSettings(ctx, "u-a", gid, &p)
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if updated.InvitePolicy != InvitePolicyAll || updated.UpdatedBy != "u-a" || updated.UpdatedAt == nil {
		t.Fatalf("updated = %+v", updated)
	}
	again, err := svc.GetSettings(ctx, "u-b", gid)
	if err != nil || again.InvitePolicy != InvitePolicyAll {
		t.Fatalf("persisted = %+v, %v", again, err)
	}
}

func TestAddMembers_SkipsActiveAndSyncsConversation(t *testing.T) {
	svc, _, _, chats := testService()
	ctx := context.Background()
	entry := mustGroup(t, svc, ctx)
	gid := entry.Group.ID
	added, err := svc.AddMembers(ctx, "u-a", gid, []string{"u-b", "u-d", "u-d", ""})
	if err != nil {
		t.Fatalf("AddMembers: %v", err)
	}
	if len(added) != 1 || added[0].UserID != "u-d" {
		t.Fatalf("added = %+v", added)
	}
	if len(chats.added[entry.Group.ConversationID]) != 1 {
		t.Fatalf("conversation sync added = %v", chats.added)
	}
}

func TestCreateGroup_ConversationFailurePropagates(t *testing.T) {
	svc, _, _, chats := testService()
	chats.failNext = errors.New("chat: 建立容器失敗")
	if _, err := svc.CreateGroup(context.Background(), "u-a", "群組", "", "", []string{"u-b"}); err == nil {
		t.Fatal("對話容器建立失敗時群組建立應失敗")
	}
}
