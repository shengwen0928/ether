// PostgreSQL 儲存實作（Phase 06）：media.Store。
//
// 對映規則（schema 見 migrations/000008）：uuid 一律 ::text 讀出／
// ::uuid 傳入；可空欄位以 NULL 語義（見 internal/postgres 慣例）；
// 查詢一律具名欄位列。查無列時 Get* 回傳 (nil, nil)，由服務層決定
// 404 語義（不得洩漏存在性）。縮圖 (media_file_id, variant_type)
// 唯一索引衝突對映為欄位錯誤（競態兜底）；其餘唯一性違反原樣回傳。
package media

import (
	"context"
	"database/sql"
	"time"

	"ether.local/api/internal/postgres"
	"ether.local/api/internal/transport"
)

// PGStore 是 Store 的 PostgreSQL 實作。
type PGStore struct{ q postgres.Querier }

// NewPGStore 建立 PGStore。
func NewPGStore(q postgres.Querier) *PGStore { return &PGStore{q: q} }

const fileColumns = `id::text, owner_id::text, storage_key, mime_type, size_bytes, original_filename, width, height, duration_seconds, checksum, scan_status, scan_completed_at, is_deleted, created_at, updated_at`

func scanFile(row postgres.Row) (*MediaFile, error) {
	var f MediaFile
	var filename, checksum sql.NullString
	var scanCompletedAt sql.NullTime
	var scanStatus string
	if err := row.Scan(&f.ID, &f.OwnerID, &f.StorageKey, &f.MIMEType, &f.SizeBytes,
		&filename, &f.Width, &f.Height, &f.DurationSeconds, &checksum,
		&scanStatus, &scanCompletedAt, &f.IsDeleted, &f.CreatedAt, &f.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	f.OriginalFilename = postgres.StringFromNull(filename)
	f.Checksum = postgres.StringFromNull(checksum)
	f.ScanStatus = ScanStatus(scanStatus)
	f.ScanCompletedAt = postgres.TimePtrFromNull(scanCompletedAt)
	return &f, nil
}

func scanFileRows(rows postgres.Rows) ([]*MediaFile, error) {
	defer func() { _ = rows.Close() }()
	var out []*MediaFile
	for rows.Next() {
		var f MediaFile
		var filename, checksum sql.NullString
		var scanCompletedAt sql.NullTime
		var scanStatus string
		if err := rows.Scan(&f.ID, &f.OwnerID, &f.StorageKey, &f.MIMEType, &f.SizeBytes,
			&filename, &f.Width, &f.Height, &f.DurationSeconds, &checksum,
			&scanStatus, &scanCompletedAt, &f.IsDeleted, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		f.OriginalFilename = postgres.StringFromNull(filename)
		f.Checksum = postgres.StringFromNull(checksum)
		f.ScanStatus = ScanStatus(scanStatus)
		f.ScanCompletedAt = postgres.TimePtrFromNull(scanCompletedAt)
		out = append(out, &f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetFile(ctx context.Context, id string) (*MediaFile, error) {
	return scanFile(s.q.QueryRowContext(ctx,
		`SELECT `+fileColumns+` FROM media_files WHERE id = $1::uuid`, id))
}

func (s *PGStore) ListFilesByOwner(ctx context.Context, ownerID string) ([]*MediaFile, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+fileColumns+` FROM media_files
		 WHERE owner_id = $1::uuid AND NOT is_deleted
		 ORDER BY created_at DESC, id`, ownerID)
	if err != nil {
		return nil, err
	}
	return scanFileRows(rows)
}

func (s *PGStore) CreateFile(ctx context.Context, f *MediaFile) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO media_files (id, owner_id, storage_key, mime_type, size_bytes, original_filename, width, height, duration_seconds, checksum, scan_status, scan_completed_at, is_deleted, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		f.ID, f.OwnerID, f.StorageKey, f.MIMEType, f.SizeBytes,
		postgres.NullString(f.OriginalFilename), f.Width, f.Height, f.DurationSeconds,
		postgres.NullString(f.Checksum), string(f.ScanStatus), postgres.NullTime(f.ScanCompletedAt),
		f.IsDeleted, f.CreatedAt, f.UpdatedAt)
	return err
}

func (s *PGStore) SaveFile(ctx context.Context, f *MediaFile) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE media_files
		 SET owner_id = $2::uuid, storage_key = $3, mime_type = $4, size_bytes = $5,
		     original_filename = $6, width = $7, height = $8, duration_seconds = $9,
		     checksum = $10, scan_status = $11, scan_completed_at = $12, is_deleted = $13,
		     updated_at = $14
		 WHERE id = $1::uuid`,
		f.ID, f.OwnerID, f.StorageKey, f.MIMEType, f.SizeBytes,
		postgres.NullString(f.OriginalFilename), f.Width, f.Height, f.DurationSeconds,
		postgres.NullString(f.Checksum), string(f.ScanStatus), postgres.NullTime(f.ScanCompletedAt),
		f.IsDeleted, f.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrMediaNotFound
	}
	return nil
}

const uploadColumns = `id::text, media_file_id::text, uploader_id::text, storage_key, reserved_file_id::text, upload_id_external, status, uploaded_bytes, total_bytes, part_count, completed_at, error_message, created_at, updated_at`

func scanUpload(row postgres.Row) (*Upload, error) {
	var u Upload
	var mediaFileID, externalID, errorMessage sql.NullString
	var completedAt sql.NullTime
	var status string
	if err := row.Scan(&u.ID, &mediaFileID, &u.UploaderID, &u.StorageKey, &u.ReservedFileID,
		&externalID, &status, &u.UploadedBytes, &u.TotalBytes, &u.PartCount,
		&completedAt, &errorMessage, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	u.MediaFileID = postgres.StringFromNull(mediaFileID)
	u.ExternalUploadID = postgres.StringFromNull(externalID)
	u.Status = UploadStatus(status)
	u.CompletedAt = postgres.TimePtrFromNull(completedAt)
	u.ErrorMessage = postgres.StringFromNull(errorMessage)
	return &u, nil
}

func scanUploadRows(rows postgres.Rows) ([]*Upload, error) {
	defer func() { _ = rows.Close() }()
	var out []*Upload
	for rows.Next() {
		var u Upload
		var mediaFileID, externalID, errorMessage sql.NullString
		var completedAt sql.NullTime
		var status string
		if err := rows.Scan(&u.ID, &mediaFileID, &u.UploaderID, &u.StorageKey, &u.ReservedFileID,
			&externalID, &status, &u.UploadedBytes, &u.TotalBytes, &u.PartCount,
			&completedAt, &errorMessage, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.MediaFileID = postgres.StringFromNull(mediaFileID)
		u.ExternalUploadID = postgres.StringFromNull(externalID)
		u.Status = UploadStatus(status)
		u.CompletedAt = postgres.TimePtrFromNull(completedAt)
		u.ErrorMessage = postgres.StringFromNull(errorMessage)
		out = append(out, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetUpload(ctx context.Context, id string) (*Upload, error) {
	return scanUpload(s.q.QueryRowContext(ctx,
		`SELECT `+uploadColumns+` FROM media_uploads WHERE id = $1::uuid`, id))
}

func (s *PGStore) ListUploadsByUploader(ctx context.Context, uploaderID string) ([]*Upload, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+uploadColumns+` FROM media_uploads
		 WHERE uploader_id = $1::uuid
		 ORDER BY created_at DESC, id`, uploaderID)
	if err != nil {
		return nil, err
	}
	return scanUploadRows(rows)
}

func (s *PGStore) ListStaleUploads(ctx context.Context, cutoff time.Time) ([]*Upload, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+uploadColumns+` FROM media_uploads
		 WHERE status = 'uploading' AND updated_at < $1
		 ORDER BY updated_at, id`, cutoff)
	if err != nil {
		return nil, err
	}
	return scanUploadRows(rows)
}

func (s *PGStore) CreateUpload(ctx context.Context, u *Upload) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO media_uploads (id, media_file_id, uploader_id, storage_key, reserved_file_id, upload_id_external, status, uploaded_bytes, total_bytes, part_count, completed_at, error_message, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5::uuid, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		u.ID, postgres.NullString(u.MediaFileID), u.UploaderID, u.StorageKey, u.ReservedFileID,
		postgres.NullString(u.ExternalUploadID), string(u.Status), u.UploadedBytes, u.TotalBytes,
		u.PartCount, postgres.NullTime(u.CompletedAt), postgres.NullString(u.ErrorMessage),
		u.CreatedAt, u.UpdatedAt)
	return err
}

func (s *PGStore) SaveUpload(ctx context.Context, u *Upload) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE media_uploads
		 SET media_file_id = $2::uuid, uploader_id = $3::uuid, storage_key = $4,
		     reserved_file_id = $5::uuid, upload_id_external = $6, status = $7,
		     uploaded_bytes = $8, total_bytes = $9, part_count = $10,
		     completed_at = $11, error_message = $12, updated_at = $13
		 WHERE id = $1::uuid`,
		u.ID, postgres.NullString(u.MediaFileID), u.UploaderID, u.StorageKey, u.ReservedFileID,
		postgres.NullString(u.ExternalUploadID), string(u.Status), u.UploadedBytes, u.TotalBytes,
		u.PartCount, postgres.NullTime(u.CompletedAt), postgres.NullString(u.ErrorMessage),
		u.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrMediaNotFound
	}
	return nil
}

const thumbnailColumns = `id::text, media_file_id::text, variant_type, storage_key, mime_type, size_bytes, width, height, created_at`

func scanThumbnail(row postgres.Row) (*Thumbnail, error) {
	var t Thumbnail
	var variant string
	if err := row.Scan(&t.ID, &t.MediaFileID, &variant, &t.StorageKey, &t.MIMEType,
		&t.SizeBytes, &t.Width, &t.Height, &t.CreatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	t.Variant = ThumbnailVariant(variant)
	return &t, nil
}

func scanThumbnailRows(rows postgres.Rows) ([]*Thumbnail, error) {
	defer func() { _ = rows.Close() }()
	var out []*Thumbnail
	for rows.Next() {
		var t Thumbnail
		var variant string
		if err := rows.Scan(&t.ID, &t.MediaFileID, &variant, &t.StorageKey, &t.MIMEType,
			&t.SizeBytes, &t.Width, &t.Height, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.Variant = ThumbnailVariant(variant)
		out = append(out, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetThumbnail(ctx context.Context, id string) (*Thumbnail, error) {
	return scanThumbnail(s.q.QueryRowContext(ctx,
		`SELECT `+thumbnailColumns+` FROM media_thumbnails WHERE id = $1::uuid`, id))
}

func (s *PGStore) ListThumbnails(ctx context.Context, mediaFileID string) ([]*Thumbnail, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+thumbnailColumns+` FROM media_thumbnails
		 WHERE media_file_id = $1::uuid
		 ORDER BY variant_type, id`, mediaFileID)
	if err != nil {
		return nil, err
	}
	return scanThumbnailRows(rows)
}

func (s *PGStore) CreateThumbnail(ctx context.Context, t *Thumbnail) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO media_thumbnails (id, media_file_id, variant_type, storage_key, mime_type, size_bytes, width, height, created_at)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9)`,
		t.ID, t.MediaFileID, string(t.Variant), t.StorageKey, t.MIMEType, t.SizeBytes,
		t.Width, t.Height, t.CreatedAt)
	if err != nil {
		if postgres.IsUniqueViolation(err) && postgres.UniqueConstraintName(err) == "media_thumbnails_file_variant_unique" {
			return &ValidationError{Fields: []transport.FieldError{{
				Field: "variant_type", Issue: "DUPLICATE", Message: "此變體的縮圖已存在。",
			}}}
		}
		return err
	}
	return nil
}

func (s *PGStore) CreateAccessLog(ctx context.Context, l *AccessLog) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO media_access_logs (id, media_file_id, accessor_id, access_type, access_result, created_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6)`,
		l.ID, l.MediaFileID, postgres.NullString(l.AccessorID), string(l.AccessType),
		string(l.AccessResult), l.CreatedAt)
	return err
}
