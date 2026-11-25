// PostgreSQL 儲存實作（Phase 05 後續）：訊息附件關聯與轉傳。
//
// 對映規則（schema 見 migrations/000011）：uuid 一律 ::text 讀出／
// ::uuid 傳入；查詢一律具名欄位列。附件關聯寫入以
// (message_id, media_file_id) 唯一索引收斂重複關聯（不得重複寫入
// 同一檔案的關聯列）；轉傳紀錄寫入以 forwarded_message_id 唯一
// 索引保證一則新訊息至多一筆轉傳紀錄。
package message

import (
	"context"

	"ether.local/api/internal/postgres"
)

const attachmentColumns = `id::text, message_id::text, media_file_id::text, mime_type, size_bytes, file_name, created_at`

func scanAttachment(row postgres.Row) (*Attachment, error) {
	var a Attachment
	if err := row.Scan(&a.ID, &a.MessageID, &a.MediaFileID, &a.MIMEType, &a.SizeBytes, &a.FileName, &a.CreatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func scanAttachmentRows(rows postgres.Rows) ([]*Attachment, error) {
	defer func() { _ = rows.Close() }()
	var out []*Attachment
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.MessageID, &a.MediaFileID, &a.MIMEType, &a.SizeBytes, &a.FileName, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateAttachments 寫入附件關聯列（同一檔案重複關聯不重複寫入，
// 以唯一索引收斂）。
func (s *PGStore) CreateAttachments(ctx context.Context, messageID string, atts []*Attachment) error {
	for _, att := range atts {
		if _, err := s.q.ExecContext(ctx,
			`INSERT INTO message_attachments (id, message_id, media_file_id, mime_type, size_bytes, file_name, created_at)
			 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)
			 ON CONFLICT (message_id, media_file_id) DO NOTHING`,
			att.ID, messageID, att.MediaFileID, att.MIMEType, att.SizeBytes, att.FileName, att.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

// ListAttachments 列出訊息的附件關聯列（依建立時間順序）。
func (s *PGStore) ListAttachments(ctx context.Context, messageID string) ([]*Attachment, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+attachmentColumns+` FROM message_attachments
		 WHERE message_id = $1::uuid ORDER BY created_at, id::text`,
		messageID)
	if err != nil {
		return nil, err
	}
	return scanAttachmentRows(rows)
}

// GetAttachment 讀取訊息對某檔案的附件列；不存在時回傳 (nil, nil)。
func (s *PGStore) GetAttachment(ctx context.Context, messageID, mediaFileID string) (*Attachment, error) {
	return scanAttachment(s.q.QueryRowContext(ctx,
		`SELECT `+attachmentColumns+` FROM message_attachments
		 WHERE message_id = $1::uuid AND media_file_id = $2::uuid`,
		messageID, mediaFileID))
}

const forwardColumns = `id::text, source_message_id::text, forwarded_message_id::text, source_conversation_id::text, target_conversation_id::text, forwarded_by::text, created_at`

func scanForward(row postgres.Row) (*Forward, error) {
	var f Forward
	if err := row.Scan(&f.ID, &f.SourceMessageID, &f.ForwardedMessageID, &f.SourceConversationID, &f.TargetConversationID, &f.ForwardedBy, &f.CreatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

// CreateForward 寫入轉傳紀錄；同一新訊息重複寫入時以唯一索引
// 收斂（保留首次列，不得新建重複紀錄）。
func (s *PGStore) CreateForward(ctx context.Context, f *Forward) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO message_forwards (id, source_message_id, forwarded_message_id, source_conversation_id, target_conversation_id, forwarded_by, created_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::uuid, $6::uuid, $7)
		 ON CONFLICT (forwarded_message_id) DO NOTHING`,
		f.ID, f.SourceMessageID, f.ForwardedMessageID, f.SourceConversationID, f.TargetConversationID, f.ForwardedBy, f.CreatedAt)
	return err
}

// GetForwardByForwardedID 讀取某新訊息的轉傳紀錄；該訊息並非
// 轉傳產物時回傳 (nil, nil)。
func (s *PGStore) GetForwardByForwardedID(ctx context.Context, forwardedMessageID string) (*Forward, error) {
	return scanForward(s.q.QueryRowContext(ctx,
		`SELECT `+forwardColumns+` FROM message_forwards
		 WHERE forwarded_message_id = $1::uuid`, forwardedMessageID))
}
