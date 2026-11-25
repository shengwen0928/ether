// Package postgres 是 PostgreSQL 基礎設施層（M2b）。
//
// 內容：
//   - Querier：database/sql 的最小查詢介面（Row／Rows／Exec），
//     儲存庫只依賴此介面；單元測試以假實作驗證 SQL 與對映，
//     整合測試以真實 *sql.DB 驗證（build tag integration）。
//   - Open：以 lib/pq（純 Go、無 cgo）開啟連線池。
//   - 錯誤對映：PostgreSQL 錯誤碼 → 領域可判別的錯誤
//     （23505 唯一性違反 → 依約束名對映；其餘原樣回傳）。
//
// SQL 寫作慣例（全模組一致，不得各寫一套）：
//   - uuid 欄位一律以 ::text 讀出（掃入 Go string，避免驅動的
//     型別解碼差異）、以 ::uuid 傳入參數（
//     WHERE id = $1::uuid、INSERT VALUES ($1::uuid, …)）。
//   - timestamptz 欄位：NOT NULL 者掃入 time.Time；可空者掃入
//     sql.NullTime 再轉換，不得以零值 time.Time 冒充 NULL。
//   - 查詢一律以具名欄位列出，不得 SELECT *（schema 演進時
//     欄位順序／新增欄位不得靜默改變對映）。
//   - 可空 text 欄位：空字串寫入 NULL、讀出 NULL 轉空字串
//     （NullString／NullTime 輔助函式）；不得以空字串冒充 NULL，
//     亦不得以 NULL 冒充空字串以外的語義。
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// Row 是單列查詢結果（*sql.Row 的最小介面）。
type Row interface {
	Scan(dest ...any) error
}

// Rows 是多列查詢結果（*sql.Rows 的最小介面）。
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close() error
}

// Querier 是儲存庫依賴的查詢介面。*sql.DB 與 *sql.Tx 皆可
// 經 WrapDB／WrapTx 適配；測試以假實作替代。
type Querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) Row
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// stdQuerier 是 *sql.DB／*sql.Tx 共有的查詢方法集合。
type stdQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type querierAdapter struct{ q stdQuerier }

func (a querierAdapter) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return a.q.QueryRowContext(ctx, query, args...)
}

func (a querierAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return a.q.QueryContext(ctx, query, args...)
}

func (a querierAdapter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return a.q.ExecContext(ctx, query, args...)
}

// WrapDB 將 *sql.DB 適配為 Querier。
func WrapDB(db *sql.DB) Querier { return querierAdapter{q: db} }

// WrapTx 將 *sql.Tx 適配為 Querier（同一交易內的儲存庫呼叫）。
func WrapTx(tx *sql.Tx) Querier { return querierAdapter{q: tx} }

// DB 是 PostgreSQL 連線池持有者。Close 由呼叫方在關停時執行。
type DB struct {
	db *sql.DB
}

// Open 以 DSN 開啟連線池（lib/pq）。Open 本身不連線；Ping
// 由呼叫方在啟動時執行，失敗即明確報錯，不得靜默降級。
func Open(dsn string) (*DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: 開啟連線池失敗: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	return &DB{db: db}, nil
}

// Querier 回傳池的 Querier 介面。
func (d *DB) Querier() Querier { return WrapDB(d.db) }

// Ping 驗證資料庫可達（啟動時 fail-fast 用）。
func (d *DB) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// Begin 開始交易。呼叫方須 Commit 或 Rollback。
func (d *DB) Begin(ctx context.Context) (*sql.Tx, error) {
	return d.db.BeginTx(ctx, nil)
}

// Tx 是資料庫交易：經 Querier 發出同一交易內的查詢，結束時
// Commit 或 Rollback。呼叫方須確保兩者恰好執行其一；交易內
// 發生錯誤後不得繼續發出查詢（PostgreSQL 會中止交易）。
type Tx struct{ tx *sql.Tx }

// BeginTx 開始交易並回傳 Tx 控制代碼。
func (d *DB) BeginTx(ctx context.Context) (*Tx, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx}, nil
}

// Querier 回傳綁定此交易的 Querier。
func (t *Tx) Querier() Querier { return WrapTx(t.tx) }

// Commit 提交交易。
func (t *Tx) Commit() error { return t.tx.Commit() }

// Rollback 回滾交易；已提交的交易回滾回傳 sql.ErrTxDone，
// 呼叫方在提交失敗後的清理路徑可忽略此錯誤。
func (t *Tx) Rollback() error { return t.tx.Rollback() }

// Close 關閉連線池。
func (d *DB) Close() error { return d.db.Close() }

// IsUniqueViolation 報告 err 是否為 PostgreSQL 唯一性違反（23505）。
func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

// UniqueConstraintName 取出唯一性違反的約束名；非唯一性違反
// 時回傳空字串。呼叫方依約束名對映到欄位級錯誤。
func UniqueConstraintName(err error) string {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return pqErr.Constraint
	}
	return ""
}

// IsForeignKeyViolation 報告 err 是否為外鍵違反（23503）。
func IsForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23503"
}

// IsNoRows 報告 err 是否為查無列（sql.ErrNoRows）。
func IsNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }

// NullString 將空字串轉為 NULL 參數（可空 text 欄位寫入用）。
func NullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// NullTime 將空指標轉為 NULL 參數（可空 timestamptz 欄位寫入用）。
func NullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// StringFromNull 將 NULL 轉為空字串（可空 text 欄位讀出用）。
func StringFromNull(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}

// TimePtrFromNull 將 NULL 轉為空指標（可空 timestamptz 欄位讀出用）。
func TimePtrFromNull(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	t := nt.Time
	return &t
}
