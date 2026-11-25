package media

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

func TestPGStore_GetFile_MapsNulls(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "FROM media_files") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: []any{
			"f-1", "u-1", "media/u-1/f-1", "image/jpeg", int64(1000), nil, 800, 600, int64(0), nil, "pending", nil, false, fixedTime, fixedTime,
		}}
	}
	f, err := NewPGStore(fake).GetFile(context.Background(), "f-1")
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	if f == nil || f.OriginalFilename != "" || f.Checksum != "" || f.ScanCompletedAt != nil || f.ScanStatus != ScanPending {
		t.Fatalf("file = %+v", f)
	}

	fake2 := &postgrestest.FakeQuerier{}
	fake2.RowFunc = func(string, []any) postgres.Row {
		return postgrestest.FakeRow{Err: sql.ErrNoRows}
	}
	f, err = NewPGStore(fake2).GetFile(context.Background(), "missing")
	if err != nil || f != nil {
		t.Fatalf("missing = %+v, %v, want nil, nil", f, err)
	}
}

func TestPGStore_SaveFile_ZeroRowsReturnsNotFound(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return postgrestest.FakeResult{Affected: 0}, nil
	}
	err := NewPGStore(fake).SaveFile(context.Background(), &MediaFile{ID: "f-1"})
	if err != ErrMediaNotFound {
		t.Fatalf("err = %v, want ErrMediaNotFound", err)
	}
}

func TestPGStore_GetUpload_MapsNulls(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowFunc = func(query string, args []any) postgres.Row {
		if !strings.Contains(query, "FROM media_uploads") {
			t.Errorf("unexpected query: %s", query)
		}
		return postgrestest.FakeRow{Values: []any{
			"u-1", nil, "u-owner", "media/u-owner/f-9", "f-9", "ext-1", "uploading", int64(400), int64(1000), 1, nil, nil, fixedTime, fixedTime,
		}}
	}
	u, err := NewPGStore(fake).GetUpload(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("GetUpload: %v", err)
	}
	if u == nil || u.MediaFileID != "" || u.CompletedAt != nil || u.Status != UploadUploading {
		t.Fatalf("upload = %+v", u)
	}
}

func TestPGStore_ListStaleUploads_SQLAndMapping(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.RowsFunc = func(query string, args []any) (postgres.Rows, error) {
		if !strings.Contains(query, "status = 'uploading'") {
			t.Errorf("unexpected query: %s", query)
		}
		return &postgrestest.FakeRows{Rows: [][]any{{
			"u-1", nil, "u-owner", "media/u-owner/f-9", "f-9", "ext-1", "uploading", int64(0), int64(100), 0, nil, nil, fixedTime, fixedTime,
		}}}, nil
	}
	rows, err := NewPGStore(fake).ListStaleUploads(context.Background(), fixedTime)
	if err != nil {
		t.Fatalf("ListStaleUploads: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "u-1" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestPGStore_CreateThumbnail_DuplicateVariant(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	fake.ExecFunc = func(string, []any) (sql.Result, error) {
		return nil, &pq.Error{Code: "23505", Constraint: "media_thumbnails_file_variant_unique"}
	}
	err := NewPGStore(fake).CreateThumbnail(context.Background(), &Thumbnail{ID: "t-1"})
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Fields[0].Field != "variant_type" {
		t.Fatalf("err = %v, want variant_type 欄位錯誤", err)
	}

	fake2 := &postgrestest.FakeQuerier{}
	fake2.ExecFunc = func(string, []any) (sql.Result, error) {
		return nil, &pq.Error{Code: "23505", Constraint: "media_thumbnails_storage_key_key"}
	}
	if err := NewPGStore(fake2).CreateThumbnail(context.Background(), &Thumbnail{ID: "t-1"}); err == nil {
		t.Fatal("其他唯一衝突須原樣回傳，不得包裝為欄位錯誤")
	}
}

func TestPGStore_CreateAccessLog_SQL(t *testing.T) {
	fake := &postgrestest.FakeQuerier{}
	var gotQuery string
	var gotArgs []any
	fake.ExecFunc = func(query string, args []any) (sql.Result, error) {
		gotQuery, gotArgs = query, args
		return postgrestest.FakeResult{Affected: 1}, nil
	}
	err := NewPGStore(fake).CreateAccessLog(context.Background(), &AccessLog{
		ID: "l-1", MediaFileID: "f-1", AccessorID: "u-1",
		AccessType: AccessDownload, AccessResult: AccessAllowed, CreatedAt: fixedTime,
	})
	if err != nil {
		t.Fatalf("CreateAccessLog: %v", err)
	}
	if !strings.Contains(gotQuery, "INSERT INTO media_access_logs") {
		t.Fatalf("query = %s", gotQuery)
	}
	if len(gotArgs) != 6 || gotArgs[3] != "download" || gotArgs[4] != "allowed" {
		t.Fatalf("args = %v", gotArgs)
	}
}
