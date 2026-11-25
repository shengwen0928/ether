package call

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

func TestPGStore_GetCall_MapsNulls(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "FROM calls") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: []any{
			"c-1", nil, "g-1", "u-a", "voice", "ringing", nil, nil, nil, nil, nil, fixedTime, fixedTime,
		}}
	}
	c, err := NewPGStore(fake).GetCall(context.Background(), "c-1")
	if err != nil {
		t.Fatalf("GetCall: %v", err)
	}
	if c == nil || c.ConversationID != "" || c.GroupID != "g-1" || c.StartedAt != nil || c.Duration != nil || c.EndedBy != "" || c.EndReason != "" {
		t.Fatalf("call = %+v", c)
	}

	fake2 := &postgrestest.FakeQuerier{}
	fake2.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	c, err = NewPGStore(fake2).GetCall(context.Background(), "missing")
	if err != nil || c != nil {
		t.Fatalf("missing = %+v, %v, want nil, nil", c, err)
	}
}

func TestPGStore_SaveCall_ZeroRowsReturnsNotFound(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return postgrestest.FakeResult{Affected: 0}, nil
	}
	err := NewPGStore(fake).SaveCall(context.Background(), &Call{ID: "c-1", Status: StatusEnded})
	if err != ErrCallNotFound {
		t.Fatalf("err = %v, want ErrCallNotFound", err)
	}
}

func TestPGStore_CreateParticipant_UniqueViolationMapsToValidation(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return nil, &pq.Error{Code: "23505", Constraint: "call_participants_call_user_unique"}
	}
	err := NewPGStore(fake).CreateParticipant(context.Background(), &Participant{ID: "p-1", CallID: "c-1", UserID: "u-a"})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
}

func TestPGStore_ListCallsByUser_SQLAndMapping(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "JOIN call_participants") {
			t.Errorf("unexpected query: %s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{{
			"c-1", "conv-1", nil, "u-a", "video", "ended", fixedTime, fixedTime, int64(42), "u-a", "hangup", fixedTime, fixedTime,
		}}}, nil
	}
	calls, err := NewPGStore(fake).ListCallsByUser(context.Background(), "u-a", "", 20)
	if err != nil {
		t.Fatalf("ListCallsByUser: %v", err)
	}
	if len(calls) != 1 || calls[0].Duration == nil || *calls[0].Duration != 42 || calls[0].EndReason != EndReasonHangup {
		t.Fatalf("calls = %+v", calls)
	}
}

func TestPGStore_FindActiveCall_MissingReturnsNil(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	c, err := NewPGStore(fake).FindActiveCall(context.Background(), "conv-1", "")
	if err != nil || c != nil {
		t.Fatalf("call = %+v, %v, want nil, nil", c, err)
	}
}

func TestPGStore_ListEvents_Mapping(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "FROM call_events") {
			t.Errorf("unexpected query: %s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{
			{"e-1", "c-1", "u-a", "initiated", `{"to_status":"ringing"}`, fixedTime, fixedTime},
			{"e-2", "c-1", nil, "ended", `{"to_status":"ended"}`, fixedTime, fixedTime},
		}}, nil
	}
	events, err := NewPGStore(fake).ListEvents(context.Background(), "c-1")
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 2 || events[1].UserID != "" || events[0].Type != EventInitiated {
		t.Fatalf("events = %+v", events)
	}
}
