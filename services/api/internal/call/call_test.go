package call

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"ether.local/api/internal/chat"
	"ether.local/api/internal/group"
)

var fixedNow = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

type fakeUsers struct{ existing map[string]bool }

func (f *fakeUsers) Exists(_ context.Context, userID string) (bool, error) {
	return f.existing[userID], nil
}

type fakeChats struct {
	members map[string][]*chat.Member
	direct  map[string]bool
	blocked map[string]bool
	err     error
	listErr error
}

func (f *fakeChats) IsActiveMember(_ context.Context, conversationID, userID string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for _, m := range f.members[conversationID] {
		if m.UserID == userID && m.LeftAt == nil {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeChats) IsDirect(_ context.Context, conversationID string) (bool, error) {
	return f.direct[conversationID], nil
}

func (f *fakeChats) DirectBlocked(_ context.Context, conversationID, _ string) (bool, error) {
	return f.blocked[conversationID], nil
}

func (f *fakeChats) ListMembers(_ context.Context, _, conversationID string) ([]*chat.Member, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.members[conversationID], nil
}

func chatMember(userID string) *chat.Member {
	return &chat.Member{UserID: userID, JoinedAt: fixedNow}
}

type fakeGroups struct {
	members map[string][]*group.Member
	err     error
}

func (f *fakeGroups) ListMembers(_ context.Context, _, groupID string) ([]*group.Member, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.members[groupID], nil
}

func groupMember(userID string) *group.Member {
	return &group.Member{UserID: userID, Role: group.RoleMember, JoinedAt: fixedNow}
}

func testService(now *time.Time, chats ConversationAccess, groups GroupChecker) *Service {
	users := &fakeUsers{existing: map[string]bool{"u-a": true, "u-b": true, "u-c": true, "u-d": true}}
	return NewService(NewMemoryStore(), users, chats, groups, func() time.Time { return *now })
}

func directChats() *fakeChats {
	return &fakeChats{
		members: map[string][]*chat.Member{
			"conv-1": {chatMember("u-a"), chatMember("u-b")},
		},
		direct: map[string]bool{"conv-1": true},
	}
}

func groupChats() *fakeGroups {
	return &fakeGroups{members: map[string][]*group.Member{
		"g-1": {groupMember("u-a"), groupMember("u-b"), groupMember("u-c")},
	}}
}

func TestInitiateDirect_Success(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	svc := testService(&now, directChats(), nil)
	entry, err := svc.Initiate(ctx, "u-a", InitiateInput{ConversationID: "conv-1", Type: CallTypeVoice})
	if err != nil {
		t.Fatalf("Initiate: %v", err)
	}
	if entry.Call.Status != StatusRinging || entry.Call.StartedAt != nil || entry.ParticipantCount != 2 {
		t.Fatalf("entry = %+v", entry.Call)
	}
	participants, err := svc.ListParticipants(ctx, "u-a", entry.Call.ID)
	if err != nil {
		t.Fatalf("ListParticipants: %v", err)
	}
	if len(participants) != 2 || participants[0].UserID != "u-a" || participants[0].Result != ResultJoined {
		t.Fatalf("participants = %+v", participants)
	}
	if participants[1].UserID != "u-b" || participants[1].Result != ResultPending || participants[1].JoinedAt != nil {
		t.Fatalf("invitee = %+v", participants[1])
	}
	events, err := svc.ListEvents(ctx, "u-a", entry.Call.ID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 || events[0].Type != EventInitiated {
		t.Fatalf("events = %+v", events)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(events[0].Payload), &payload); err != nil {
		t.Fatalf("payload JSON: %v", err)
	}
	if strings.Contains(events[0].Payload, "sdp") || strings.Contains(events[0].Payload, "ice") {
		t.Fatalf("payload 不得含信令內容: %s", events[0].Payload)
	}
}

func TestInitiate_Validation(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	svc := testService(&now, directChats(), nil)
	cases := []InitiateInput{
		{ConversationID: "conv-1", Type: "carrier-pigeon"},
		{Type: CallTypeVoice},
		{ConversationID: "conv-1", GroupID: "g-1", Type: CallTypeVoice},
	}
	for i, in := range cases {
		if _, err := svc.Initiate(ctx, "u-a", in); err == nil {
			t.Fatalf("case %d: want error", i)
		} else {
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("case %d: err = %v, want ValidationError", i, err)
			}
		}
	}
}

func TestInitiateDirect_BlockedAndNonMember404(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	blocked := directChats()
	blocked.blocked = map[string]bool{"conv-1": true}
	svc := testService(&now, blocked, nil)
	if _, err := svc.Initiate(ctx, "u-a", InitiateInput{ConversationID: "conv-1", Type: CallTypeVoice}); !errors.Is(err, ErrCallNotFound) {
		t.Fatalf("blocked: err = %v, want ErrCallNotFound", err)
	}
	svc2 := testService(&now, directChats(), nil)
	if _, err := svc2.Initiate(ctx, "u-c", InitiateInput{ConversationID: "conv-1", Type: CallTypeVoice}); !errors.Is(err, ErrCallNotFound) {
		t.Fatalf("non-member: err = %v, want ErrCallNotFound", err)
	}
}

func TestInitiateDirect_GroupConversationRejected(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	chats := directChats()
	chats.direct["conv-1"] = false
	svc := testService(&now, chats, nil)
	_, err := svc.Initiate(ctx, "u-a", InitiateInput{ConversationID: "conv-1", Type: CallTypeVoice})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
}

func TestInitiateGroup_SuccessAndNonMember(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	svc := testService(&now, nil, groupChats())
	entry, err := svc.Initiate(ctx, "u-a", InitiateInput{GroupID: "g-1", Type: CallTypeVideo})
	if err != nil {
		t.Fatalf("Initiate: %v", err)
	}
	if entry.Call.GroupID != "g-1" || entry.ParticipantCount != 3 {
		t.Fatalf("entry = %+v", entry.Call)
	}
	failing := &fakeGroups{err: group.ErrGroupNotFound}
	svc2 := testService(&now, nil, failing)
	if _, err := svc2.Initiate(ctx, "u-a", InitiateInput{GroupID: "g-1", Type: CallTypeVideo}); !errors.Is(err, ErrCallNotFound) {
		t.Fatalf("non-member: err = %v, want ErrCallNotFound", err)
	}
}

func TestInitiate_DuplicateActiveRejected(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	svc := testService(&now, directChats(), nil)
	if _, err := svc.Initiate(ctx, "u-a", InitiateInput{ConversationID: "conv-1", Type: CallTypeVoice}); err != nil {
		t.Fatalf("first Initiate: %v", err)
	}
	if _, err := svc.Initiate(ctx, "u-b", InitiateInput{ConversationID: "conv-1", Type: CallTypeVoice}); err == nil {
		t.Fatal("second Initiate: want error")
	}
}

func initiateDirect(t *testing.T, svc *Service, now *time.Time) *Call {
	t.Helper()
	entry, err := svc.Initiate(context.Background(), "u-a", InitiateInput{ConversationID: "conv-1", Type: CallTypeVoice})
	if err != nil {
		t.Fatalf("Initiate: %v", err)
	}
	_ = now
	return entry.Call
}

func TestAnswer_Flow(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	svc := testService(&now, directChats(), nil)
	c := initiateDirect(t, svc, &now)
	now = now.Add(30 * time.Second)
	entry, err := svc.Answer(ctx, "u-b", c.ID)
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if entry.Call.Status != StatusActive || entry.Call.StartedAt == nil || !entry.Call.StartedAt.Equal(now) {
		t.Fatalf("after answer = %+v", entry.Call)
	}
	if _, err := svc.Answer(ctx, "u-b", c.ID); err == nil {
		t.Fatal("second Answer: want error")
	}
	if _, err := svc.Answer(ctx, "u-a", c.ID); err == nil {
		t.Fatal("initiator Answer: want error")
	}
	events, _ := svc.ListEvents(ctx, "u-a", c.ID)
	if len(events) != 2 || events[1].Type != EventAnswered {
		t.Fatalf("events = %+v", events)
	}
}

func TestDecline_AllDeclined(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	svc := testService(&now, directChats(), nil)
	c := initiateDirect(t, svc, &now)
	entry, err := svc.Decline(ctx, "u-b", c.ID)
	if err != nil {
		t.Fatalf("Decline: %v", err)
	}
	if entry.Call.Status != StatusDeclined || entry.Call.EndReason != EndReasonDeclined || entry.Call.EndedAt == nil || entry.Call.StartedAt != nil {
		t.Fatalf("after decline = %+v", entry.Call)
	}
	if _, err := svc.Decline(ctx, "u-b", c.ID); err == nil {
		t.Fatal("decline after terminal: want error")
	}
}

func TestBusy_AllBusyAndMixed(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	svc := testService(&now, directChats(), nil)
	c := initiateDirect(t, svc, &now)
	entry, err := svc.Busy(ctx, "u-b", c.ID)
	if err != nil {
		t.Fatalf("Busy: %v", err)
	}
	if entry.Call.Status != StatusBusy || entry.Call.EndReason != EndReasonBusy {
		t.Fatalf("after busy = %+v", entry.Call)
	}

	now = fixedNow
	svc2 := testService(&now, nil, groupChats())
	g, err := svc2.Initiate(ctx, "u-a", InitiateInput{GroupID: "g-1", Type: CallTypeVoice})
	if err != nil {
		t.Fatalf("Initiate group: %v", err)
	}
	if _, err := svc2.Busy(ctx, "u-b", g.Call.ID); err != nil {
		t.Fatalf("Busy u-b: %v", err)
	}
	entry, err = svc2.Decline(ctx, "u-c", g.Call.ID)
	if err != nil {
		t.Fatalf("Decline u-c: %v", err)
	}
	if entry.Call.Status != StatusDeclined {
		t.Fatalf("mixed responses: status = %s, want declined", entry.Call.Status)
	}
}

func TestEnd_RingingNoAnswerIsMissed(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	svc := testService(&now, directChats(), nil)
	c := initiateDirect(t, svc, &now)
	entry, err := svc.End(ctx, "u-a", c.ID)
	if err != nil {
		t.Fatalf("End: %v", err)
	}
	if entry.Call.Status != StatusMissed || entry.Call.EndReason != EndReasonMissed || entry.Call.Duration != nil {
		t.Fatalf("after end = %+v", entry.Call)
	}
	participants, _ := svc.ListParticipants(ctx, "u-a", c.ID)
	for _, p := range participants {
		if p.UserID == "u-b" && p.Result != ResultMissed {
			t.Fatalf("invitee result = %s, want missed", p.Result)
		}
	}
	if _, err := svc.End(ctx, "u-a", c.ID); err == nil {
		t.Fatal("end after terminal: want error")
	}
}

func TestEnd_ActiveComputesDuration(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	svc := testService(&now, directChats(), nil)
	c := initiateDirect(t, svc, &now)
	now = now.Add(90 * time.Second)
	if _, err := svc.Answer(ctx, "u-b", c.ID); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	now = now.Add(30 * time.Second)
	entry, err := svc.End(ctx, "u-b", c.ID)
	if err != nil {
		t.Fatalf("End: %v", err)
	}
	if entry.Call.Status != StatusEnded || entry.Call.Duration == nil || *entry.Call.Duration != 30 {
		t.Fatalf("after end = %+v, duration=%v", entry.Call, entry.Call.Duration)
	}
	participants, _ := svc.ListParticipants(ctx, "u-a", c.ID)
	for _, p := range participants {
		if p.LeftAt == nil {
			t.Fatalf("participant %s LeftAt 為空", p.UserID)
		}
	}
}

func TestEnd_PendingInviteeCannotEnd(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	svc := testService(&now, directChats(), nil)
	c := initiateDirect(t, svc, &now)
	if _, err := svc.End(ctx, "u-b", c.ID); !errors.Is(err, ErrCallNotFound) {
		t.Fatalf("pending End: err = %v, want ErrCallNotFound", err)
	}
}

func TestAccess_NonParticipant404(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	svc := testService(&now, directChats(), nil)
	c := initiateDirect(t, svc, &now)
	if _, err := svc.GetCall(ctx, "u-c", c.ID); !errors.Is(err, ErrCallNotFound) {
		t.Fatalf("GetCall: err = %v, want ErrCallNotFound", err)
	}
	if _, err := svc.ListParticipants(ctx, "u-c", c.ID); !errors.Is(err, ErrCallNotFound) {
		t.Fatalf("ListParticipants: err = %v, want ErrCallNotFound", err)
	}
	if _, err := svc.ListEvents(ctx, "u-c", c.ID); !errors.Is(err, ErrCallNotFound) {
		t.Fatalf("ListEvents: err = %v, want ErrCallNotFound", err)
	}
	if _, err := svc.GetCall(ctx, "u-a", "missing"); !errors.Is(err, ErrCallNotFound) {
		t.Fatalf("missing: err = %v, want ErrCallNotFound", err)
	}
}

func TestListCalls_CursorPagination(t *testing.T) {
	ctx := context.Background()
	now := fixedNow
	svc := testService(&now, directChats(), nil)
	c1 := initiateDirect(t, svc, &now)
	if _, err := svc.End(ctx, "u-a", c1.ID); err != nil {
		t.Fatalf("End c1: %v", err)
	}
	now = now.Add(time.Second)
	c2 := initiateDirect(t, svc, &now)
	if _, err := svc.End(ctx, "u-a", c2.ID); err != nil {
		t.Fatalf("End c2: %v", err)
	}
	page1, cursor, err := svc.ListCalls(ctx, "u-a", "", 1)
	if err != nil {
		t.Fatalf("ListCalls: %v", err)
	}
	if len(page1) != 1 || page1[0].Call.ID != c2.ID || cursor == "" {
		t.Fatalf("page1 = %+v, cursor=%q", page1, cursor)
	}
	page2, cursor2, err := svc.ListCalls(ctx, "u-a", cursor, 1)
	if err != nil {
		t.Fatalf("ListCalls page2: %v", err)
	}
	if len(page2) != 1 || page2[0].Call.ID != c1.ID {
		t.Fatalf("page2 = %+v, cursor=%q", page2, cursor2)
	}
	page3, _, err := svc.ListCalls(ctx, "u-a", cursor2, 1)
	if err != nil {
		t.Fatalf("ListCalls page3: %v", err)
	}
	if len(page3) != 0 {
		t.Fatalf("page3 = %+v, want empty", page3)
	}
	if _, _, err := svc.ListCalls(ctx, "u-a", "not-a-cursor", 1); err == nil {
		t.Fatal("bad cursor: want error")
	}
}

func TestStatus_IsTerminal(t *testing.T) {
	if !StatusEnded.IsTerminal() || !StatusMissed.IsTerminal() || StatusRinging.IsTerminal() || StatusActive.IsTerminal() {
		t.Fatal("IsTerminal 語義錯誤")
	}
}
