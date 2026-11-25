//go:build integration

package chat

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/postgrestest"
	"ether.local/api/internal/user"
)

// TestPG_ConversationPersists 驗證對話完整流程經 PostgreSQL 持久化：
// 建一對一對話（須好友關係）→ 第二個 Service 實例（模擬重啟）仍可
// 讀取 → 離開 → 重新建立重新加入。無隔離資料庫時跳過並標示
// 「環境待驗證」。
func TestPG_ConversationPersists(t *testing.T) {
	ctx := context.Background()
	db := postgrestest.SetupTestDB(t)
	q := postgres.WrapDB(db)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	users := user.NewService(user.NewPGUserStore(q), user.NewPGSettingsStore(q), func() time.Time { return now })
	ua, err := users.Provision(ctx, "kc-subject-a", "a@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision a: %v", err)
	}
	ub, err := users.Provision(ctx, "kc-subject-b", "b@example.invalid", true)
	if err != nil {
		t.Fatalf("Provision b: %v", err)
	}
	// 好友關係須先存在：一對一對話的建立前提。直接以 SQL 寫入
	// friendships 列（friend 模組的寫入路徑屬其測試範圍，此處
	// 只建前提，不重測好友流程）。
	if _, err := db.ExecContext(ctx,
		`INSERT INTO friendships (user_id, friend_id, status, created_at, updated_at)
		 VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid), 'active', $3, $3)`,
		ua.ID, ub.ID, now); err != nil {
		t.Fatalf("建立好友前提: %v", err)
	}

	friends := &pgFriendChecker{db: db}
	svc := NewService(NewPGStore(q), users, friends, func() time.Time { return now })
	conv, isNew, err := svc.CreateDirect(ctx, ua.ID, ub.ID)
	if err != nil || !isNew {
		t.Fatalf("CreateDirect = %+v, %v, %v", conv, isNew, err)
	}

	svc2 := NewService(NewPGStore(q), users, friends, func() time.Time { return now })
	got, err := svc2.GetConversation(ctx, ub.ID, conv.ID)
	if err != nil || got == nil || got.Type != TypeDirect {
		t.Fatalf("重啟後讀取 = %+v, %v", got, err)
	}
	if err := svc2.Leave(ctx, ua.ID, conv.ID); err != nil {
		t.Fatalf("Leave: %v", err)
	}
	rejoined, isNew, err := svc2.CreateDirect(ctx, ua.ID, ub.ID)
	if err != nil || isNew || rejoined.ID != conv.ID {
		t.Fatalf("重新建立 = %+v, %v, %v", rejoined, isNew, err)
	}
}

// pgFriendChecker 是 FriendChecker 的最小 SQL 實作，僅供本整合
// 測試查詢好友／封鎖前提（直讀 friend 表；正式接線一律經
// friend.Service，不得以此替身代替）。
type pgFriendChecker struct{ db *sql.DB }

func (f *pgFriendChecker) AreFriends(ctx context.Context, a, b string) (bool, error) {
	var n int
	err := f.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM friendships
		 WHERE user_id = LEAST($1::uuid, $2::uuid)
		 AND friend_id = GREATEST($1::uuid, $2::uuid) AND status = 'active'`, a, b).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (f *pgFriendChecker) BlockedBetween(ctx context.Context, a, b string) (bool, error) {
	var n int
	err := f.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM blocked_users
		 WHERE ((user_id = $1::uuid AND blocked_user_id = $2::uuid)
		 OR (user_id = $2::uuid AND blocked_user_id = $1::uuid))
		 AND unblocked_at IS NULL`, a, b).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
