// Command migrate 是 ETHER API 的 migration 執行工具（M2b-2）。
//
// 用法：
//
//	migrate up              套用全部未套用的 migration
//	migrate down            回滾最近一個已套用的 migration
//	migrate down-to <版本>  回滾至指定版本（該版本保留）
//	migrate status          列出已套用／未套用版本
//
// DSN 一律取 DATABASE_URL 環境變數；程式不得印出 DSN，不得接受
// 命令列上的 DSN（避免出現在行程清單中）。回滾會遺失資料，僅限
// 開發資料庫；共用／正式環境回滾前須備份並經審查（MIGRATIONS.md §3）。
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"

	"ether.local/api/internal/postgres"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fatal("DATABASE_URL 未設定")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fatal(err.Error())
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		fatal(fmt.Sprintf("資料庫連線失敗: %v", err))
	}

	switch os.Args[1] {
	case "up":
		applied, err := postgres.MigrateUp(ctx, db)
		if err != nil {
			fatal(err.Error())
		}
		if len(applied) == 0 {
			fmt.Println("已是最新，無待套用的 migration。")
		} else {
			fmt.Printf("已套用: %v\n", applied)
		}
	case "down":
		version, err := postgres.MigrateDownOne(ctx, db)
		if err != nil {
			fatal(err.Error())
		}
		if version == "" {
			fmt.Println("無已套用的 migration，無須回滾。")
		} else {
			fmt.Printf("已回滾: %s\n", version)
		}
	case "down-to":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		rolledBack, err := postgres.MigrateDownTo(ctx, db, os.Args[2])
		if err != nil {
			fatal(err.Error())
		}
		if len(rolledBack) == 0 {
			fmt.Println("無須回滾。")
		} else {
			fmt.Printf("已回滾: %v\n", rolledBack)
		}
	case "status":
		applied, pending, err := postgres.Status(ctx, db)
		if err != nil {
			fatal(err.Error())
		}
		fmt.Printf("已套用: %v\n未套用: %v\n", applied, pending)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "用法: migrate up | down | down-to <版本> | status")
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "migrate: "+msg)
	os.Exit(1)
}
