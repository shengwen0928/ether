// Package postgrestest 提供 PostgreSQL 儲存庫測試的輔助工具：
//   - FakeQuerier：不連接資料庫的 Querier 假實作，驗證儲存庫發出
//     的 SQL、參數與列對映（含 NULL 語義與錯誤對映）。
//   - OpenTestDB：整合測試用的真實資料庫連線（build tag
//     integration，見 postgrestest.go）；無 ETHER_TEST_DATABASE_URL
//     時測試跳過並標示「環境待驗證」。
//
// 本套件僅供 *_test.go 使用，不得被正式程式碼 import。
package postgrestest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"ether.local/api/internal/postgres"
)

// Call 記錄一次 FakeQuerier 呼叫。
type Call struct {
	Kind  string // "row"、"query"、"exec"
	Query string
	Args  []any
}

// FakeRow 是 postgres.Row 的假實作：依 Values 順序填入 Scan 目標。
type FakeRow struct {
	Values []any
	Err    error
}

func (r FakeRow) Scan(dest ...any) error {
	if r.Err != nil {
		return r.Err
	}
	return assignValues(dest, r.Values)
}

// FakeRows 是 postgres.Rows 的假實作。
type FakeRows struct {
	Rows    [][]any
	IterErr error
	pos     int
	begun   bool
	closed  bool
}

func (r *FakeRows) Next() bool {
	if r.closed {
		return false
	}
	if !r.begun {
		r.begun = true
		r.pos = 0
	} else {
		r.pos++
	}
	return r.pos < len(r.Rows)
}

func (r *FakeRows) Scan(dest ...any) error {
	if !r.begun || r.pos < 0 || r.pos >= len(r.Rows) {
		return errors.New("postgrestest: Scan called without a current row")
	}
	return assignValues(dest, r.Rows[r.pos])
}

func (r *FakeRows) Err() error { return r.IterErr }

func (r *FakeRows) Close() error {
	r.closed = true
	return nil
}

func assignValues(dest []any, values []any) error {
	if len(dest) != len(values) {
		return fmt.Errorf("postgrestest: Scan 目標 %d 個、值 %d 個，數量不一致", len(dest), len(values))
	}
	for i, d := range dest {
		v := values[i]
		switch t := d.(type) {
		case *string:
			switch val := v.(type) {
			case nil:
				*t = ""
			case string:
				*t = val
			case []byte:
				*t = string(val)
			default:
				return fmt.Errorf("postgrestest: 無法將 %T 指派給 *string", v)
			}
		case *bool:
			b, ok := v.(bool)
			if !ok {
				return fmt.Errorf("postgrestest: 無法將 %T 指派給 *bool", v)
			}
			*t = b
		case *int:
			switch val := v.(type) {
			case int:
				*t = val
			case int64:
				*t = int(val)
			default:
				return fmt.Errorf("postgrestest: 無法將 %T 指派給 *int", v)
			}
		case *int64:
			switch val := v.(type) {
			case int64:
				*t = val
			case int:
				*t = int64(val)
			default:
				return fmt.Errorf("postgrestest: 無法將 %T 指派給 *int64", v)
			}
		case *time.Time:
			tm, ok := v.(time.Time)
			if !ok {
				return fmt.Errorf("postgrestest: 無法將 %T 指派給 *time.Time", v)
			}
			*t = tm
		case *sql.NullString:
			if v == nil {
				*t = sql.NullString{}
			} else if s, ok := v.(string); ok {
				*t = sql.NullString{String: s, Valid: true}
			} else {
				return fmt.Errorf("postgrestest: 無法將 %T 指派給 *sql.NullString", v)
			}
		case *sql.NullTime:
			if v == nil {
				*t = sql.NullTime{}
			} else if tm, ok := v.(time.Time); ok {
				*t = sql.NullTime{Time: tm, Valid: true}
			} else {
				return fmt.Errorf("postgrestest: 無法將 %T 指派給 *sql.NullTime", v)
			}
		case *sql.NullInt64:
			if v == nil {
				*t = sql.NullInt64{}
			} else if n, ok := v.(int64); ok {
				*t = sql.NullInt64{Int64: n, Valid: true}
			} else if n, ok := v.(int); ok {
				*t = sql.NullInt64{Int64: int64(n), Valid: true}
			} else {
				return fmt.Errorf("postgrestest: 無法將 %T 指派給 *sql.NullInt64", v)
			}
		default:
			return fmt.Errorf("postgrestest: 不支援的 Scan 目標型別 %T", d)
		}
	}
	return nil
}

// FakeResult 是 sql.Result 的假實作。
type FakeResult struct{ Affected int64 }

func (r FakeResult) LastInsertId() (int64, error) {
	return 0, errors.New("postgrestest: LastInsertId 不支援")
}

func (r FakeResult) RowsAffected() (int64, error) { return r.Affected, nil }

// FakeQuerier 是 postgres.Querier 的假實作。RowFunc／RowsFunc／
// ExecFunc 未設定時分別回傳 ErrNoRows、空列集、影響 1 列。
type FakeQuerier struct {
	RowFunc  func(query string, args []any) postgres.Row
	RowsFunc func(query string, args []any) (postgres.Rows, error)
	ExecFunc func(query string, args []any) (sql.Result, error)
	Calls    []Call
}

func (f *FakeQuerier) QueryRowContext(_ context.Context, query string, args ...any) postgres.Row {
	f.Calls = append(f.Calls, Call{Kind: "row", Query: query, Args: args})
	if f.RowFunc != nil {
		return f.RowFunc(query, args)
	}
	return FakeRow{Err: sql.ErrNoRows}
}

func (f *FakeQuerier) QueryContext(_ context.Context, query string, args ...any) (postgres.Rows, error) {
	f.Calls = append(f.Calls, Call{Kind: "query", Query: query, Args: args})
	if f.RowsFunc != nil {
		return f.RowsFunc(query, args)
	}
	return &FakeRows{}, nil
}

func (f *FakeQuerier) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	f.Calls = append(f.Calls, Call{Kind: "exec", Query: query, Args: args})
	if f.ExecFunc != nil {
		return f.ExecFunc(query, args)
	}
	return FakeResult{Affected: 1}, nil
}

var _ postgres.Querier = (*FakeQuerier)(nil)
