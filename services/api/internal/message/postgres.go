// PostgreSQL 儲存實作（Phase 05）：message.Store。
//
// 對映規則（schema 見 migrations/000006）：uuid 一律 ::text 讀出／
// ::uuid 傳入；可空欄位以 NULL 語義（見 internal/postgres 慣例）；
// 查詢一律具名欄位列。查無列時 GetMessage／GetByClientMessageID
// 回傳 (nil, nil)，由服務層決定 404 語義。
//
// 序號分配：INSERT 時以子查詢取 MAX(sequence_number)+1；併發寫入
// 衝突時 (conversation_id, sequence_number) 唯一約束觸發 23505，
// 本儲存層至多重試 3 次（重算序號），仍衝突則回傳錯誤（服務層
// 不靜默吞錯）。訊息列與 message_replies／message_attachments 列
// 的寫入未包在同一交易（CreateMessage 路徑，儲存介面未提供交易
// 邊界，見 STATUS.md 已知限制）：寫入中斷時至多遺失引用／附件
// 關聯，訊息列本身已持久化，不得以此假裝關聯寫入具原子性。
// CreateMessageWithEvent 路徑（Outbox 接線時）則以交易同時提交
// 訊息列、引用列、附件列與事件列。
package message

import (
	"context"
	"database/sql"
	"errors"

	"ether.local/api/internal/outbox"
	"ether.local/api/internal/postgres"
)

// PGStore 是 Store 的 PostgreSQL 實作。
type PGStore struct {
	q  postgres.Querier
	db *postgres.DB // 為空時不支援交易內原子寫入（見 NewPGStoreWithDB）。
}

// NewPGStore 建立 PGStore。
func NewPGStore(q postgres.Querier) *PGStore { return &PGStore{q: q} }

// NewPGStoreWithDB 建立帶交易支援的 PGStore：CreateMessageWithEvent／
// SaveMessageWithEvent 以 db 開啟交易，訊息列與 Outbox 事件列同一
// 交易提交（DATA_FLOW.md §3）。
func NewPGStoreWithDB(q postgres.Querier, db *postgres.DB) *PGStore {
	return &PGStore{q: q, db: db}
}

var _ AtomicWriter = (*PGStore)(nil)

// ErrDuplicateClientMessage 表示去重鍵競態：併發寫入命中
// client_message_id 部分唯一索引。服務層收到此錯誤時應重讀
// 既有列並以冪等命中回傳，不得新建。
var ErrDuplicateClientMessage = errors.New("message: duplicate client_message_id")

const messageColumns = `m.id::text, m.conversation_id::text, m.sender_id::text, m.kind, m.content, m.status, m.client_message_id, m.sequence_number, m.edited_at, m.original_content, m.recalled_at, m.deleted_for_sender_at, m.edit_count, m.created_at, m.updated_at, r.replied_to_message_id::text`

const messageFrom = `FROM messages m LEFT JOIN message_replies r ON r.message_id = m.id`

func scanMessage(row postgres.Row) (*Message, error) {
	var m Message
	var kind, status string
	var senderID, clientMessageID, originalContent, replyTo sql.NullString
	var editedAt, recalledAt, deletedAt sql.NullTime
	if err := row.Scan(&m.ID, &m.ConversationID, &senderID, &kind, &m.Content, &status,
		&clientMessageID, &m.SequenceNumber, &editedAt, &originalContent, &recalledAt,
		&deletedAt, &m.EditCount, &m.CreatedAt, &m.UpdatedAt, &replyTo); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	m.Kind = Kind(kind)
	m.Status = Status(status)
	m.SenderID = postgres.StringFromNull(senderID)
	m.ClientMessageID = postgres.StringFromNull(clientMessageID)
	m.OriginalContent = postgres.StringFromNull(originalContent)
	m.ReplyToMessageID = postgres.StringFromNull(replyTo)
	m.EditedAt = postgres.TimePtrFromNull(editedAt)
	m.RecalledAt = postgres.TimePtrFromNull(recalledAt)
	m.DeletedForSenderAt = postgres.TimePtrFromNull(deletedAt)
	return &m, nil
}

func scanMessageRows(rows postgres.Rows) ([]*Message, error) {
	defer func() { _ = rows.Close() }()
	var out []*Message
	for rows.Next() {
		var m Message
		var kind, status string
		var senderID, clientMessageID, originalContent, replyTo sql.NullString
		var editedAt, recalledAt, deletedAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.ConversationID, &senderID, &kind, &m.Content, &status,
			&clientMessageID, &m.SequenceNumber, &editedAt, &originalContent, &recalledAt,
			&deletedAt, &m.EditCount, &m.CreatedAt, &m.UpdatedAt, &replyTo); err != nil {
			return nil, err
		}
		m.Kind = Kind(kind)
		m.Status = Status(status)
		m.SenderID = postgres.StringFromNull(senderID)
		m.ClientMessageID = postgres.StringFromNull(clientMessageID)
		m.OriginalContent = postgres.StringFromNull(originalContent)
		m.ReplyToMessageID = postgres.StringFromNull(replyTo)
		m.EditedAt = postgres.TimePtrFromNull(editedAt)
		m.RecalledAt = postgres.TimePtrFromNull(recalledAt)
		m.DeletedForSenderAt = postgres.TimePtrFromNull(deletedAt)
		out = append(out, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) GetMessage(ctx context.Context, id string) (*Message, error) {
	return scanMessage(s.q.QueryRowContext(ctx,
		`SELECT `+messageColumns+` `+messageFrom+` WHERE m.id = $1::uuid`, id))
}

func (s *PGStore) GetByClientMessageID(ctx context.Context, conversationID, senderID, clientMessageID string) (*Message, error) {
	return scanMessage(s.q.QueryRowContext(ctx,
		`SELECT `+messageColumns+` `+messageFrom+`
		 WHERE m.conversation_id = $1::uuid AND m.sender_id = $2::uuid
		 AND m.client_message_id = $3`, conversationID, senderID, clientMessageID))
}

// CreateMessage 寫入訊息並分配序號（見檔頭重試規則）；引用關聯
// 寫入 message_replies（非交易，見檔頭限制）。
func (s *PGStore) CreateMessage(ctx context.Context, m *Message) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		lastErr = insertMessageRow(ctx, s.q, m)
		if lastErr == nil {
			return nil
		}
		if postgres.IsUniqueViolation(lastErr) {
			switch postgres.UniqueConstraintName(lastErr) {
			case "messages_client_message_unique_idx":
				return ErrDuplicateClientMessage
			case "messages_conversation_id_sequence_number_key":
				continue
			}
		}
		return lastErr
	}
	return lastErr
}

// insertMessageRow 以單次 INSERT 寫入訊息列並讀回序號、寫入引用
// 關聯與附件關聯（呼叫方以 q 決定是否在交易內；不重試，錯誤原樣
// 回傳）。附件列以 (message_id, media_file_id) 唯一收斂重複關聯。
func insertMessageRow(ctx context.Context, q postgres.Querier, m *Message) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO messages
		 (id, conversation_id, sender_id, kind, content, status, client_message_id,
		  sequence_number, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7,
		 (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM messages WHERE conversation_id = $2::uuid),
		 $8, $9)`,
		m.ID, m.ConversationID, postgres.NullString(m.SenderID), string(m.Kind), m.Content,
		string(m.Status), postgres.NullString(m.ClientMessageID), m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return err
	}
	// 讀回分配的序號（寫入路徑須以持久化序號回傳，不得以猜測值回應）。
	row := q.QueryRowContext(ctx,
		`SELECT sequence_number FROM messages WHERE id = $1::uuid`, m.ID)
	if err := row.Scan(&m.SequenceNumber); err != nil {
		return err
	}
	if m.ReplyToMessageID != "" {
		_, err := q.ExecContext(ctx,
			`INSERT INTO message_replies (message_id, replied_to_message_id, conversation_id, created_at)
			 VALUES ($1::uuid, $2::uuid, $3::uuid, $4)`,
			m.ID, m.ReplyToMessageID, m.ConversationID, m.CreatedAt)
		if err != nil {
			return err
		}
	}
	for _, att := range m.Attachments {
		_, err := q.ExecContext(ctx,
			`INSERT INTO message_attachments (id, message_id, media_file_id, mime_type, size_bytes, file_name, created_at)
			 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)
			 ON CONFLICT (message_id, media_file_id) DO NOTHING`,
			att.ID, m.ID, att.MediaFileID, att.MIMEType, att.SizeBytes, att.FileName, att.CreatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateMessageWithEvent 在同一交易內寫入訊息列與 Outbox 事件
// （見 AtomicWriter）。序號衝突時整個交易回滾後重試（交易內
// 錯誤不得繼續發出查詢，故重試單位為整個交易，至多 3 次）。
func (s *PGStore) CreateMessageWithEvent(ctx context.Context, m *Message, build EventBuilder) error {
	if s.db == nil {
		return errors.New("message: PG 儲存層未設定交易支援")
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		lastErr = s.createWithEventOnce(ctx, m, build)
		if lastErr == nil {
			return nil
		}
		if postgres.IsUniqueViolation(lastErr) &&
			postgres.UniqueConstraintName(lastErr) == "messages_conversation_id_sequence_number_key" {
			continue
		}
		return lastErr
	}
	return lastErr
}

func (s *PGStore) createWithEventOnce(ctx context.Context, m *Message, build EventBuilder) error {
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	qtx := tx.Querier()
	if err := insertMessageRow(ctx, qtx, m); err != nil {
		_ = tx.Rollback()
		if postgres.IsUniqueViolation(err) &&
			postgres.UniqueConstraintName(err) == "messages_client_message_unique_idx" {
			return ErrDuplicateClientMessage
		}
		return err
	}
	ev, err := build(m)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if ev != nil {
		if err := outbox.NewPGStore(qtx).Append(ctx, ev); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return err
	}
	return nil
}

func (s *PGStore) SaveMessage(ctx context.Context, m *Message) error {
	return updateMessageRow(ctx, s.q, m)
}

// updateMessageRow 更新訊息列（呼叫方以 q 決定是否在交易內）。
func updateMessageRow(ctx context.Context, q postgres.Querier, m *Message) error {
	res, err := q.ExecContext(ctx,
		`UPDATE messages
		 SET kind = $2, content = $3, status = $4, edited_at = $5,
		     original_content = $6, recalled_at = $7, deleted_for_sender_at = $8,
		     edit_count = $9, updated_at = $10
		 WHERE id = $1::uuid`,
		m.ID, string(m.Kind), m.Content, string(m.Status), postgres.NullTime(m.EditedAt),
		postgres.NullString(m.OriginalContent), postgres.NullTime(m.RecalledAt),
		postgres.NullTime(m.DeletedForSenderAt), m.EditCount, m.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrMessageNotFound
	}
	return nil
}

// SaveMessageWithEvent 在同一交易內更新訊息列與寫入 Outbox 事件
// （見 AtomicWriter）。
func (s *PGStore) SaveMessageWithEvent(ctx context.Context, m *Message, build EventBuilder) error {
	if s.db == nil {
		return errors.New("message: PG 儲存層未設定交易支援")
	}
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	qtx := tx.Querier()
	if err := updateMessageRow(ctx, qtx, m); err != nil {
		_ = tx.Rollback()
		return err
	}
	ev, err := build(m)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if ev != nil {
		if err := outbox.NewPGStore(qtx).Append(ctx, ev); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return err
	}
	return nil
}

func (s *PGStore) ListBefore(ctx context.Context, conversationID string, beforeSeq int64, limit int) ([]*Message, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+messageColumns+` `+messageFrom+`
		 WHERE m.conversation_id = $1::uuid AND m.sequence_number < $2
		 ORDER BY m.sequence_number DESC LIMIT $3`, conversationID, beforeSeq, limit)
	if err != nil {
		return nil, err
	}
	return scanMessageRows(rows)
}

func (s *PGStore) ListSince(ctx context.Context, conversationID string, sinceSeq int64, limit int) ([]*Message, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+messageColumns+` `+messageFrom+`
		 WHERE m.conversation_id = $1::uuid AND m.sequence_number > $2
		 ORDER BY m.sequence_number ASC LIMIT $3`, conversationID, sinceSeq, limit)
	if err != nil {
		return nil, err
	}
	return scanMessageRows(rows)
}

func (s *PGStore) CountAfter(ctx context.Context, conversationID, userID string, afterSeq int64) (int, error) {
	row := s.q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM messages
		 WHERE conversation_id = $1::uuid AND sequence_number > $2
		 AND sender_id IS DISTINCT FROM $3::uuid AND status <> 'recalled'`,
		conversationID, afterSeq, userID)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *PGStore) GetReplyTo(ctx context.Context, messageID string) (string, error) {
	row := s.q.QueryRowContext(ctx,
		`SELECT replied_to_message_id::text FROM message_replies WHERE message_id = $1::uuid`, messageID)
	var replyTo sql.NullString
	if err := row.Scan(&replyTo); err != nil {
		if postgres.IsNoRows(err) {
			return "", nil
		}
		return "", err
	}
	return postgres.StringFromNull(replyTo), nil
}

const receiptColumns = `id::text, message_id::text, user_id::text, device_id::text, receipt_type, created_at`

func scanReceipt(row postgres.Row) (*Receipt, error) {
	var r Receipt
	var typ string
	var deviceID sql.NullString
	if err := row.Scan(&r.ID, &r.MessageID, &r.UserID, &deviceID, &typ, &r.CreatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	r.Type = ReceiptType(typ)
	r.DeviceID = postgres.StringFromNull(deviceID)
	return &r, nil
}

func (s *PGStore) CreateReceipt(ctx context.Context, r *Receipt) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO message_receipts (id, message_id, user_id, device_id, receipt_type, created_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6)`,
		r.ID, r.MessageID, r.UserID, postgres.NullString(r.DeviceID), string(r.Type), r.CreatedAt)
	if err != nil && postgres.IsUniqueViolation(err) {
		// 併發重複寫入同一回條：呼叫方以 GetReceipt 重讀為準，
		// 此處回傳哨兵以外的一般錯誤語義不變（服務層已預檢）。
		return err
	}
	return err
}

func (s *PGStore) GetReceipt(ctx context.Context, messageID, userID string, typ ReceiptType) (*Receipt, error) {
	return scanReceipt(s.q.QueryRowContext(ctx,
		`SELECT `+receiptColumns+` FROM message_receipts
		 WHERE message_id = $1::uuid AND user_id = $2::uuid AND receipt_type = $3`,
		messageID, userID, string(typ)))
}

func (s *PGStore) ListReceipts(ctx context.Context, messageID string) ([]*Receipt, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+receiptColumns+` FROM message_receipts
		 WHERE message_id = $1::uuid ORDER BY created_at, id::text`, messageID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Receipt
	for rows.Next() {
		r, err := scanReceiptRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanReceiptRow(rows postgres.Rows) (*Receipt, error) {
	var r Receipt
	var typ string
	var deviceID sql.NullString
	if err := rows.Scan(&r.ID, &r.MessageID, &r.UserID, &deviceID, &typ, &r.CreatedAt); err != nil {
		return nil, err
	}
	r.Type = ReceiptType(typ)
	r.DeviceID = postgres.StringFromNull(deviceID)
	return &r, nil
}
