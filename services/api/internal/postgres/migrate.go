// Migrator：migration 的套用、回滾與狀態追蹤（M2b-2）。
//
// 版本記帳：schema_migrations 表（version 主鍵、applied_at）。
// 000001 建立的 schema_meta 佔位表予以保留，不得刪除或挪用
// （MIGRATIONS.md §4）；版本追蹤一律以 schema_migrations 為準，
// 兩者職責不同，不得混淆。
//
// 交易語義：每個 migration 在各自的交易內套用，失敗即整批回滾；
// 版本記帳寫入同一交易，套用成功才記帳。Migration SQL 檔須為
// 交易安全（不得含 CREATE INDEX CONCURRENTLY、VACUUM 等不得
// 在交易內執行的陳述），否則套用失敗並明確報錯。
//
// 併發：同一資料庫的 migrator 以 session 級 advisory lock 序列化
// （固定鍵，見 advisoryLockKey），多個實例同時套用時只有一個能
// 進行，其餘等待；不得以「各自檢查版本」假裝序列化。
//
// 套用一律經本 migrator（cmd/migrate 或整合測試），不得以手動
// psql 套用代替版本追蹤；手動套用的資料庫視為狀態未知，須以
// Status 核對後再處理。
//
// Migration 檔案一律取自內嵌的 migrations.FS（即本目錄的 SQL 檔），
// 不接受外部傳入的檔案系統，避免套用的內容與程式碼版本脫節。
package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"ether.local/api/migrations"
)

// advisoryLockKey 是 migration 的 advisory lock 固定鍵（任意選定，
// 全專案唯一；更換此值將使舊鎖失效，須經審查）。
const advisoryLockKey = 727001

// appliedVersions 讀出已套用的版本（遞增排序）。呼叫方須持有
// advisory lock（同一連線），否則讀到的集合可能在套用中途變化。
func appliedVersions(ctx context.Context, conn *sql.Conn) (map[string]bool, error) {
	rows, err := conn.QueryContext(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("postgres: 讀取 schema_migrations 失敗: %w", err)
	}
	defer rows.Close()
	applied := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("postgres: 讀取 schema_migrations 失敗: %w", err)
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: 讀取 schema_migrations 失敗: %w", err)
	}
	return applied, nil
}

// withMigrationLock 在專用連線上持有 advisory lock 並執行 fn。
func withMigrationLock(ctx context.Context, db *sql.DB, fn func(conn *sql.Conn) error) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("postgres: 取得 migration 連線失敗: %w", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, advisoryLockKey); err != nil {
		return fmt.Errorf("postgres: 取得 migration 鎖失敗: %w", err)
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, advisoryLockKey)
	}()
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("postgres: 建立 schema_migrations 失敗: %w", err)
	}
	return fn(conn)
}

// MigrateUp 套用全部未套用的 migration，依版本遞增順序，回傳
// 本次新套用的版本清單（無待套用時為空）。
func MigrateUp(ctx context.Context, db *sql.DB) ([]string, error) {
	if err := migrations.Validate(); err != nil {
		return nil, err
	}
	list, err := migrations.List()
	if err != nil {
		return nil, err
	}
	var applied []string
	err = withMigrationLock(ctx, db, func(conn *sql.Conn) error {
		done, err := appliedVersions(ctx, conn)
		if err != nil {
			return err
		}
		for _, mig := range list {
			if done[mig.Version] {
				continue
			}
			if err := applyOne(ctx, conn, mig, true); err != nil {
				return err
			}
			applied = append(applied, mig.Version)
		}
		return nil
	})
	return applied, err
}

// MigrateDownOne 回滾最近一個已套用的 migration，回傳被回滾的
// 版本；無已套用版本時回傳空字串。
func MigrateDownOne(ctx context.Context, db *sql.DB) (string, error) {
	if err := migrations.Validate(); err != nil {
		return "", err
	}
	list, err := migrations.List()
	if err != nil {
		return "", err
	}
	byVersion := map[string]migrations.Migration{}
	for _, mig := range list {
		byVersion[mig.Version] = mig
	}
	var rolledBack string
	err = withMigrationLock(ctx, db, func(conn *sql.Conn) error {
		done, err := appliedVersions(ctx, conn)
		if err != nil {
			return err
		}
		for i := len(list) - 1; i >= 0; i-- {
			if done[list[i].Version] {
				if err := applyOne(ctx, conn, list[i], false); err != nil {
					return err
				}
				rolledBack = list[i].Version
				return nil
			}
		}
		return nil
	})
	return rolledBack, err
}

// MigrateDownTo 回滾至目標版本（含目標版本保留、其上全部回滾），
// 回傳被回滾的版本清單（遞減順序）。
func MigrateDownTo(ctx context.Context, db *sql.DB, target string) ([]string, error) {
	if err := migrations.Validate(); err != nil {
		return nil, err
	}
	list, err := migrations.List()
	if err != nil {
		return nil, err
	}
	var rolledBack []string
	err = withMigrationLock(ctx, db, func(conn *sql.Conn) error {
		done, err := appliedVersions(ctx, conn)
		if err != nil {
			return err
		}
		for i := len(list) - 1; i >= 0; i-- {
			if list[i].Version <= target {
				break
			}
			if done[list[i].Version] {
				if err := applyOne(ctx, conn, list[i], false); err != nil {
					return err
				}
				rolledBack = append(rolledBack, list[i].Version)
			}
		}
		return nil
	})
	return rolledBack, err
}

// Status 回傳（已套用版本，已知但未套用版本）。資料庫尚未建立
// schema_migrations 時，已套用為空、全部視為未套用（MigrateUp
// 會建立記帳表）。
func Status(ctx context.Context, db *sql.DB) (applied, pending []string, err error) {
	if err := migrations.Validate(); err != nil {
		return nil, nil, err
	}
	list, err := migrations.List()
	if err != nil {
		return nil, nil, err
	}
	err = withMigrationLock(ctx, db, func(conn *sql.Conn) error {
		done, err := appliedVersions(ctx, conn)
		if err != nil {
			return err
		}
		for _, mig := range list {
			if done[mig.Version] {
				applied = append(applied, mig.Version)
			} else {
				pending = append(pending, mig.Version)
			}
		}
		return nil
	})
	return applied, pending, err
}

// applyOne 在一個交易內套用（up=true）或回滾（up=false）單一
// migration，並同步版本記帳。
func applyOne(ctx context.Context, conn *sql.Conn, mig migrations.Migration, up bool) error {
	script := mig.Up
	record := `INSERT INTO schema_migrations (version) VALUES ($1)`
	if !up {
		script = mig.Down
		record = `DELETE FROM schema_migrations WHERE version = $1`
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres: migration %s 開始交易失敗: %w", mig.Version, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, script); err != nil {
		return fmt.Errorf("postgres: migration %s 執行失敗: %w", mig.Version, err)
	}
	if _, err := tx.ExecContext(ctx, record, mig.Version); err != nil {
		return fmt.Errorf("postgres: migration %s 記帳失敗: %w", mig.Version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres: migration %s 提交失敗: %w", mig.Version, err)
	}
	return nil
}
