package group

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
)

var fixedTime = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

func TestPGStore_GetGroup_MapsNulls(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "FROM groups") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: []any{
			"g-1", "登山小隊", nil, nil, nil, "u-a", false, nil, fixedTime, fixedTime,
		}}
	}
	g, err := NewPGStore(fake).GetGroup(context.Background(), "g-1")
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if g == nil || g.Name != "登山小隊" || g.Description != "" || g.ConversationID != "" || g.DisbandedAt != nil {
		t.Fatalf("group = %+v", g)
	}

	fake2 := &postgrestest.FakeQuerier{}
	fake2.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	g, err = NewPGStore(fake2).GetGroup(context.Background(), "missing")
	if err != nil || g != nil {
		t.Fatalf("missing = %+v, %v, want nil, nil", g, err)
	}
}

func TestPGStore_SaveGroup_ZeroRowsReturnsNotFound(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return postgrestest.FakeResult{Affected: 0}, nil
	}
	err := NewPGStore(fake).SaveGroup(context.Background(), &Group{ID: "g-1", Name: "群組"})
	if err != ErrGroupNotFound {
		t.Fatalf("err = %v, want ErrGroupNotFound", err)
	}
}

func TestPGStore_ListMembers_SQLAndMapping(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "FROM group_members") || !strings.Contains(query, "left_at IS NULL") {
			t.Errorf("unexpected query: %s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{
			{"g-1", "u-a", "owner", fixedTime, nil, "u-a", fixedTime, fixedTime},
			{"g-1", "u-b", "member", fixedTime, nil, "u-a", fixedTime, fixedTime},
		}}, nil
	}
	members, err := NewPGStore(fake).ListMembers(context.Background(), "g-1")
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(members) != 2 || members[0].Role != RoleOwner || members[1].InvitedBy != "u-a" {
		t.Fatalf("members = %+v", members)
	}
}

func TestPGStore_CreateInvitation_UniqueViolationMapsToDuplicate(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return nil, &pq.Error{Code: "23505", Constraint: "group_invitations_pending_unique_idx"}
	}
	err := NewPGStore(fake).CreateInvitation(context.Background(), &Invitation{
		ID: "inv-1", GroupID: "g-1", InviterID: "u-a", InviteeID: "u-b",
		Status: InvitationPending, CreatedAt: fixedTime, UpdatedAt: fixedTime,
	})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want ValidationError DUPLICATE", err)
	}
	if ve.Fields[0].Field != "invitee_id" || ve.Fields[0].Issue != "DUPLICATE" {
		t.Fatalf("fields = %+v", ve.Fields)
	}
}

func TestPGStore_GetSettings_MapsKeyValue(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "FROM group_settings") {
			t.Errorf("unexpected query: %s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{
			{"invite_policy", "all", "u-a", fixedTime},
			{"unknown_future_key", "x", "u-a", fixedTime},
		}}, nil
	}
	st, err := NewPGStore(fake).GetSettings(context.Background(), "g-1")
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if st == nil || st.InvitePolicy != InvitePolicyAll || st.UpdatedBy != "u-a" || st.UpdatedAt == nil {
		t.Fatalf("settings = %+v", st)
	}
}

func TestPGStore_GetSettings_NoRowsReturnsNil(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	st, err := NewPGStore(fake).GetSettings(context.Background(), "g-1")
	if err != nil || st != nil {
		t.Fatalf("settings = %+v, %v, want nil, nil", st, err)
	}
}

func TestPGStore_RetractActiveAnnouncements_SQL(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	if err := NewPGStore(fake).RetractActiveAnnouncements(context.Background(), "g-1", fixedTime); err != nil {
		t.Fatalf("RetractActiveAnnouncements: %v", err)
	}
	if len(fake.Calls) != 1 || fake.Calls[0].Kind != "exec" || !strings.Contains(fake.Calls[0].Query, "is_active = false") {
		t.Fatalf("calls = %+v", fake.Calls)
	}
}

func TestPGStore_CountActiveMembers(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Values: []any{3}}
	}
	n, err := NewPGStore(fake).CountActiveMembers(context.Background(), "g-1")
	if err != nil || n != 3 {
		t.Fatalf("n = %d, err = %v", n, err)
	}
}
