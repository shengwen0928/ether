//go:build integration

// 整合測試輔助：真實 PostgreSQL 連線（需隔離資料庫）。
//
// 環境變數 ETHER_TEST_DATABASE_URL 未設定時測試跳過（標示
// 「環境待驗證」），不得以跳過冒充通過；已設定但連不上時測試
// 明確失敗（環境設定錯誤，不得靜默跳過）。
package postgrestest

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"ether.local/api/internal/postgres"
)

// OpenTestDB 開啟測試資料庫連線（未設定 DSN 時跳過）。
func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("ETHER_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("環境待驗證：未設定 ETHER_TEST_DATABASE_URL（需隔離 PostgreSQL，不得連接共用／正式資料庫）")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("開啟測試資料庫失敗: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("測試資料庫連線失敗: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// SetupTestDB 開啟測試資料庫、套用全部 migration、清空身分表，
// 回傳可直接使用的連線。每個測試各自呼叫，狀態互不干擾。
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := OpenTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if _, err := postgres.MigrateUp(ctx, db); err != nil {
		t.Fatalf("測試資料庫 migration 套用失敗: %v", err)
	}
	if _, err := db.ExecContext(ctx, `TRUNCATE session_refresh_hashes, user_security_keys, user_sessions, user_devices, user_settings, user_profiles, users CASCADE`); err != nil {
		t.Fatalf("測試資料庫清空失敗: %v", err)
	}
	return db
}
