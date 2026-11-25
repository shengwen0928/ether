// Package outbox 是 Transactional Outbox 的儲存層（訊息推播中繼）。
//
// 依據：docs/architecture/DATA_FLOW.md §3（業務寫入與 Outbox 事件
// 寫入必須在同一 PostgreSQL 交易，提交後才可發布；發布為「至少
// 一次」，消費端必須冪等）。本套件只負責事件列的持久化與認領，
// 不決定事件內容、不組裝 WebSocket 信封（信封組裝屬消費端／
// Gateway），不直接發佈至任何 Broker（發佈迴圈見 internal/relay）。
//
// 表歸屬：message_outbox 由 Message 模組擁有（MODULE_BOUNDARIES.md
// §4 一表一主原則；DATA_FLOW.md §10 缺口 1 稱 Outbox 表名待 ERD
// 定案，本表名為本里程碑實作決定，待 ERD 定稿覆核）。事件只含
// 已公開的 wire 欄位與識別碼，不得含 Token、雜湊或 E2EE 明文
// （E2EE 未實作前 content 為不透明字串，見 internal/message）。
//
// 狀態機：pending → processing → published；發佈失敗回 pending
// （attempts 遞增），達重試上限改 dead（死信，須人工介入，不得
// 靜默丟棄）。processing 加 claimed_at 租約：認領逾期的列可被
// 再次認領（Relay 崩潰後不永久卡住）。
package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"ether.local/api/internal/ids"
	"ether.local/api/internal/postgres"
)

// Status 是 Outbox 事件列的發佈狀態。值域為本里程碑實作詞彙，
// 待 ERD 定稿覆核。
type Status string

const (
	// StatusPending 是尚未認領的事件。
	StatusPending Status = "pending"
	// StatusProcessing 是已被 Relay 認領、尚未回報結果的事件。
	StatusProcessing Status = "processing"
	// StatusPublished 是已發佈的事件（終態，不得重發）。
	StatusPublished Status = "published"
	// StatusDead 是達重試上限的死信（終態，須人工介入）。
	StatusDead Status = "dead"
)

// Event 是 message_outbox 的一列。Payload 為事件 data 物件的
// JSON（WebSocket 信封的 data 欄位內容，信封其餘欄位由消費端
// 組裝）；Payload 不得為空、須為合法 JSON。
type Event struct {
	ID             string
	EventName      string // WEBSOCKET_PROTOCOL.md §21 事件名，一字不得改。
	Topic          string // 訂閱主題，慣例 conversation.<對話識別碼>。
	ConversationID string
	MessageID      string
	Payload        json.RawMessage
	Status         Status
	Attempts       int
	ClaimedAt      *time.Time
	PublishedAt    *time.Time
	CreatedAt      time.Time
}

// Store 是 Outbox 事件的儲存介面。
type Store interface {
	// Append 寫入一則待發佈事件。Event.ID 為空時由儲存層產生；
	// Payload 為空或非法 JSON 時回傳錯誤（fail-closed，不得寫入
	// 無法發佈的列）。
	Append(ctx context.Context, ev *Event) error
	// Claim 認領至多 limit 則待發佈事件（含租約逾期的 processing
	// 列），並將其改為 processing。回傳列依建立時間遞增排序。
	Claim(ctx context.Context, limit int) ([]*Event, error)
	// MarkPublished 將事件標為已發佈（終態）。重複標記為冪等成功。
	MarkPublished(ctx context.Context, id string) error
	// MarkFailed 記錄一次發佈失敗：attempts 遞增；達 maxAttempts
	// 時改為 dead 並回傳 dead=true，否則回到 pending 等待重試。
	// maxAttempts <= 0 時視為 1（單次失敗即死信，不得無限重試）。
	MarkFailed(ctx context.Context, id string, maxAttempts int) (dead bool, err error)
}

// ClaimLease 是 processing 列的認領租約（實作預設，待定稿覆核）：
// 超過此時間未回報結果的列可被再次認領。
const ClaimLease = 5 * time.Minute

// MemoryStore 是 Store 的記憶體實作（開發與測試用）。
type MemoryStore struct {
	mu     sync.Mutex
	events map[string]*Event
	order  []string
	now    func() time.Time
}

// NewMemoryStore 建立空的記憶體 Outbox 儲存。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		events: map[string]*Event{},
		now:    time.Now,
	}
}

func cloneEvent(ev *Event) *Event {
	if ev == nil {
		return nil
	}
	cp := *ev
	cp.Payload = append(json.RawMessage(nil), ev.Payload...)
	return &cp
}

func validateEvent(ev *Event) error {
	if ev == nil {
		return errors.New("outbox: 事件不可為空")
	}
	if ev.EventName == "" {
		return errors.New("outbox: event_name 不可為空")
	}
	if ev.Topic == "" {
		return errors.New("outbox: topic 不可為空")
	}
	if len(ev.Payload) == 0 || !json.Valid(ev.Payload) {
		return errors.New("outbox: payload 須為合法 JSON")
	}
	return nil
}

// Append 寫入事件（見 Store 介面說明）。
func (s *MemoryStore) Append(_ context.Context, ev *Event) error {
	if err := validateEvent(ev); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if ev.ID == "" {
		ev.ID = ids.New()
		if ev.ID == "" {
			return errors.New("outbox: 識別碼產生失敗")
		}
	}
	if _, exists := s.events[ev.ID]; exists {
		return errors.New("outbox: event already exists")
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = s.now().UTC()
	}
	cp := cloneEvent(ev)
	cp.Status = StatusPending
	s.events[cp.ID] = cp
	s.order = append(s.order, cp.ID)
	return nil
}

// Claim 認領待發佈事件（見 Store 介面說明）。
func (s *MemoryStore) Claim(_ context.Context, limit int) ([]*Event, error) {
	if limit <= 0 {
		return nil, nil
	}
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*Event
	for _, id := range s.order {
		if len(out) >= limit {
			break
		}
		ev := s.events[id]
		switch ev.Status {
		case StatusPending:
		case StatusProcessing:
			if ev.ClaimedAt == nil || !ev.ClaimedAt.Add(ClaimLease).Before(now) {
				continue
			}
		default:
			continue
		}
		claimed := now
		ev.Status = StatusProcessing
		ev.ClaimedAt = &claimed
		out = append(out, cloneEvent(ev))
	}
	return out, nil
}

// MarkPublished 標記已發佈（見 Store 介面說明）。
func (s *MemoryStore) MarkPublished(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ev := s.events[id]
	if ev == nil {
		return nil
	}
	if ev.Status == StatusPublished {
		return nil
	}
	now := s.now().UTC()
	ev.Status = StatusPublished
	ev.PublishedAt = &now
	ev.ClaimedAt = nil
	return nil
}

// MarkFailed 記錄失敗（見 Store 介面說明）。
func (s *MemoryStore) MarkFailed(_ context.Context, id string, maxAttempts int) (bool, error) {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ev := s.events[id]
	if ev == nil {
		return false, nil
	}
	if ev.Status != StatusProcessing {
		return false, nil
	}
	ev.Attempts++
	ev.ClaimedAt = nil
	if ev.Attempts >= maxAttempts {
		ev.Status = StatusDead
		return true, nil
	}
	ev.Status = StatusPending
	return false, nil
}

// PGStore 是 Store 的 PostgreSQL 實作。
type PGStore struct{ q postgres.Querier }

// NewPGStore 建立 PGStore。
func NewPGStore(q postgres.Querier) *PGStore { return &PGStore{q: q} }

const outboxColumns = `id::text, event_name, topic, conversation_id::text, message_id::text, payload, status, attempts, claimed_at, published_at, created_at`

func scanEvent(row postgres.Row) (*Event, error) {
	var ev Event
	var status string
	var payload []byte
	var claimedAt, publishedAt sql.NullTime
	if err := row.Scan(&ev.ID, &ev.EventName, &ev.Topic, &ev.ConversationID, &ev.MessageID,
		&payload, &status, &ev.Attempts, &claimedAt, &publishedAt, &ev.CreatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	ev.Payload = json.RawMessage(payload)
	ev.Status = Status(status)
	ev.ClaimedAt = postgres.TimePtrFromNull(claimedAt)
	ev.PublishedAt = postgres.TimePtrFromNull(publishedAt)
	return &ev, nil
}

// Append 寫入事件（見 Store 介面說明）。呼叫方傳入交易綁定的
// Querier 時，本寫入即與同一交易的業務寫入原子提交。
func (s *PGStore) Append(ctx context.Context, ev *Event) error {
	if err := validateEvent(ev); err != nil {
		return err
	}
	if ev.ID == "" {
		ev.ID = ids.New()
		if ev.ID == "" {
			return errors.New("outbox: 識別碼產生失敗")
		}
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	}
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO message_outbox
		 (id, event_name, topic, conversation_id, message_id, payload, status, created_at)
		 VALUES ($1::uuid, $2, $3, $4::uuid, $5::uuid, $6::jsonb, 'pending', $7)`,
		ev.ID, ev.EventName, ev.Topic, ev.ConversationID, ev.MessageID, []byte(ev.Payload), ev.CreatedAt)
	return err
}

// Claim 認領待發佈事件（見 Store 介面說明）。單一 UPDATE 陳述式
// 內以 FOR UPDATE SKIP LOCKED 搶佔，併發 Relay 互不重複認領。
func (s *PGStore) Claim(ctx context.Context, limit int) ([]*Event, error) {
	if limit <= 0 {
		return nil, nil
	}
	now := time.Now().UTC()
	rows, err := s.q.QueryContext(ctx,
		`UPDATE message_outbox SET status = 'processing', claimed_at = $1
		 WHERE id IN (
		   SELECT id FROM message_outbox
		   WHERE status = 'pending'
		      OR (status = 'processing' AND claimed_at < $2)
		   ORDER BY created_at ASC, id::text ASC
		   LIMIT $3
		   FOR UPDATE SKIP LOCKED
		 )
		 RETURNING `+outboxColumns, now, now.Add(-ClaimLease), limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Event
	for rows.Next() {
		ev, err := scanEventRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanEventRow(rows postgres.Rows) (*Event, error) {
	var ev Event
	var status string
	var payload []byte
	var claimedAt, publishedAt sql.NullTime
	if err := rows.Scan(&ev.ID, &ev.EventName, &ev.Topic, &ev.ConversationID, &ev.MessageID,
		&payload, &status, &ev.Attempts, &claimedAt, &publishedAt, &ev.CreatedAt); err != nil {
		return nil, err
	}
	ev.Payload = json.RawMessage(payload)
	ev.Status = Status(status)
	ev.ClaimedAt = postgres.TimePtrFromNull(claimedAt)
	ev.PublishedAt = postgres.TimePtrFromNull(publishedAt)
	return &ev, nil
}

// MarkPublished 標記已發佈（見 Store 介面說明）。
func (s *PGStore) MarkPublished(ctx context.Context, id string) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE message_outbox
		 SET status = 'published', published_at = $2, claimed_at = NULL
		 WHERE id = $1::uuid AND status = 'processing'`,
		id, time.Now().UTC())
	return err
}

// MarkFailed 記錄失敗（見 Store 介面說明）。
func (s *PGStore) MarkFailed(ctx context.Context, id string, maxAttempts int) (bool, error) {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	row := s.q.QueryRowContext(ctx,
		`UPDATE message_outbox
		 SET attempts = attempts + 1, claimed_at = NULL,
		     status = CASE WHEN attempts + 1 >= $2 THEN 'dead' ELSE 'pending' END
		 WHERE id = $1::uuid AND status = 'processing'
		 RETURNING status`,
		id, maxAttempts)
	var status string
	if err := row.Scan(&status); err != nil {
		if postgres.IsNoRows(err) {
			return false, nil
		}
		return false, err
	}
	return status == string(StatusDead), nil
}
