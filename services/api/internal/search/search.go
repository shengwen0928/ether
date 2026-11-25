// Package search 是 Search（搜尋）模組的訊息搜尋實作（離線可驗證部分）。
//
// 依據：MODULE_BOUNDARIES.md §5.14（Search 擁有 Meilisearch 內的搜尋
// 索引投影，無自有 PostgreSQL 表；索引遺失須可由 PostgreSQL 重建；
// 不得被其他業務模組依賴）、DATA_FLOW.md §8（索引同步是事件驅動的
// 投影，不得由業務寫入路徑同步寫索引；投影寫入須冪等；權限過濾在
// API 層、回傳前執行，不得信任索引內的可見性標記為唯一依據）、
// SPEC-SOURCE.md §15（訊息搜尋範圍：關鍵字、日期區間、發送者、
// 媒體／檔案類型、群組、一對一聊天室；結果須遵守存取權限與封鎖規則）。
//
// 本里程碑範圍（僅訊息）：記憶體索引（MemoryIndex，開發與測試用）、
// Meilisearch 轉接器（MeilisearchIndex，見 meilisearch.go）、
// 事件投影器（Projector，消費 message.received／message.recalled
// 事件，冪等）、權限過濾後的查詢服務（Service）與 HTTP 端點
// （GET /api/v1/search/messages）。其他搜尋範圍（好友、使用者、
// 群組、社群、官方帳號、檔案、貼圖、動態）不在範圍，不得預寫。
//
// 架構要點（DATA_FLOW.md §8）：本套件不掛鉤訊息寫入路徑——訊息
// 寫入不得因索引失敗而失敗，索引填充由 Worker 消費 Outbox 事件
// （經 NATS）後呼叫 Projector 完成；消費迴圈與 NATS 轉接器落地
// 屬後續里程碑，本行程不啟動任何填充迴圈。索引為空時查詢回傳
// 空結果，不得謊稱結果完整。重建入口為 Projector.Rebuild，文件
// 來源由呼叫方自 PostgreSQL 掃描提供（重建流程待維運文件定案）。
//
// 依賴方向（只經公開介面）：Search 讀取 Chat 的成員關係只經
// MembershipChecker 介面（由 main.go 以 chat.Service 接線）；不得
// 直讀 conversations／messages 資料表，不得 import 他模組的內部
// 套件，不得寫入任何業務表。
//
// 已知限制：編輯無對應事件名（§21 缺口，見 STATUS.md 待決定 20），
// 編輯後的內容不會經事件更新索引，須待事件協定定稿或全量重建；
// 僅自己刪除無事件，不影響他人索引（符合預期）；媒體／檔案類型
// 篩選未實作（kind 值域目前僅 text，附件類型篩選須待事件 payload
// 擴充，見 STATUS.md 待決定 23）；MemoryIndex 全量掃描後過濾，
// Meilisearch 落地時須改為候選頁＋補取，不得以分頁跳過過濾。
package search

import (
	"context"
	"encoding/base64"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"ether.local/api/internal/transport"
)

// ErrIndexUnavailable 表示搜尋索引後端不可用（Meilisearch 轉接器
// 未落地，或後端呼叫失敗）。查詢端點遇此錯誤時回 503，不得回
// 空結果假裝索引完整。
var ErrIndexUnavailable = errors.New("search: index unavailable")

// ErrNotConfigured 表示 Meilisearch 轉接器未設定（佔位實作）。
var ErrNotConfigured = errors.New("search: meilisearch backend not configured")

// Document 是訊息在搜尋索引內的投影文件。只含搜尋所需欄位：
// 不得含 Token、雜湊、E2EE 明文以外的私密欄位。E2EE 未實作前
// Content 為不透明字串（見 internal/message）；E2EE 落地後，
// 索引文件須改為密文或僅索引中繼資料，不得再含明文。
type Document struct {
	MessageID      string
	ConversationID string
	SenderID       string
	Kind           string
	Content        string
	SequenceNumber int64
	CreatedAt      time.Time
}

// Query 是訊息搜尋的查詢條件（Service 層）。Terms 為正規化後的
// 關鍵字詞（小寫、去空白切分）；Since／Until 為建立時間區間
// （皆含邊界，待定稿覆核）；Limit 為每頁筆數（<=0 時取預設）；
// Cursor 為上一頁回傳的不透明游標（空字串表示第一頁）。
type Query struct {
	Terms          []string
	ConversationID string
	SenderID       string
	Since          *time.Time
	Until          *time.Time
	Limit          int
	Cursor         string
}

// Page 是查詢的一頁結果：Documents 依建立時間降序；NextCursor
// 為空字串時表示已無下一頁。
type Page struct {
	Documents  []Document
	NextCursor string
}

// DefaultLimit 是預設每頁筆數（實作預設，待定稿覆核）。
const DefaultLimit = 20

// MaxLimit 是每頁筆數上限（實作預設，待定稿覆核）。
const MaxLimit = 50

// MaxQueryRunes 是關鍵字查詢字串上限（實作預設，待定稿覆核）。
const MaxQueryRunes = 200

// ValidationError 承載欄位驗證失敗清單。
type ValidationError struct {
	Fields []transport.FieldError
}

func (e *ValidationError) Error() string { return "search: validation failed" }

// FieldErrors 供 transport.WriteServiceError 寫出 details。
func (e *ValidationError) FieldErrors() []transport.FieldError { return e.Fields }

func validationError(field, issue, message string) *ValidationError {
	return &ValidationError{Fields: []transport.FieldError{{Field: field, Issue: issue, Message: message}}}
}

// MembershipChecker 是 Search 對 Chat 模組的最小依賴（依賴 DAG：
// Search --> Chat）：查詢結果回傳前，須逐筆驗證呼叫者是否為該
// 對話的生效成員。不得信任索引內的可見性標記為唯一依據。
type MembershipChecker interface {
	// IsActiveMember 回傳使用者是否為對話的生效成員。
	IsActiveMember(ctx context.Context, conversationID, userID string) (bool, error)
}

// Index 是搜尋索引的儲存介面。實作須保證：Upsert／Remove 依
// eventID 冪等（同一事件重放不得產生重複文件或重複刪除）；
// Upsert 依 SequenceNumber 擋舊版本（較舊的投影不得覆蓋較新者）。
// Search 回傳全部符合條件的文件（依建立時間降序），分頁由
// Service 在權限過濾後執行，不得以分頁為由跳過過濾。
type Index interface {
	// Upsert 寫入或更新一則投影文件。eventID 為空時不做冪等去重
	// （呼叫方須傳入事件識別碼，不得以空值呼叫）。
	Upsert(ctx context.Context, doc Document, eventID string) error
	// Remove 刪除一則投影文件（收回）。文件不存在時為冪等成功。
	Remove(ctx context.Context, messageID string, eventID string) error
	// Search 回傳符合查詢條件的全部文件（建立時間降序，時間相同
	// 時以訊息識別碼降序 tiebreak，保證游標穩定）。
	Search(ctx context.Context, q Query) ([]Document, error)
	// ReplaceAll 以給定文件全量取代索引內容（重建入口）。呼叫方
	// 須保證文件來源為 PostgreSQL 全量掃描，不得以索引反寫業務表。
	ReplaceAll(ctx context.Context, docs []Document) error
}

// normalizeQuery 將查詢字串正規化為關鍵字詞：去首尾空白、
// 小寫、依空白切分。CJK 文字以子字串比對，不做分詞（實作預設，
// 待 Meilisearch 排名／分詞定稿覆核）；停用詞不預設。
func normalizeQuery(q string) []string {
	return strings.Fields(strings.ToLower(strings.TrimSpace(q)))
}

// matches 回傳文件內容是否包含全部關鍵字詞（AND 語義，實作預設）。
// 比對大小寫不敏感：內容與詞彙皆轉小寫（Index 層亦須正規化，
// 不得依賴呼叫方已正規化）。
func matches(doc Document, terms []string) bool {
	content := strings.ToLower(doc.Content)
	for _, term := range terms {
		if !strings.Contains(content, strings.ToLower(term)) {
			return false
		}
	}
	return true
}

// sortDocuments 依建立時間降序、訊息識別碼降序排序（游標穩定的
// 全序）。
func sortDocuments(docs []Document) {
	sort.Slice(docs, func(i, j int) bool {
		if !docs[i].CreatedAt.Equal(docs[j].CreatedAt) {
			return docs[i].CreatedAt.After(docs[j].CreatedAt)
		}
		return docs[i].MessageID > docs[j].MessageID
	})
}

// cursorOf 將一則文件的排序鍵編碼為不透明游標。
func cursorOf(doc Document) string {
	raw := doc.CreatedAt.UTC().Format(time.RFC3339Nano) + "\n" + doc.MessageID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// parseCursor 解碼游標，回傳排序鍵（建立時間、訊息識別碼）。
func parseCursor(cursor string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", err
	}
	parts := strings.SplitN(string(raw), "\n", 2)
	if len(parts) != 2 || parts[1] == "" {
		return time.Time{}, "", errors.New("search: invalid cursor")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", err
	}
	return createdAt, parts[1], nil
}

// afterCursor 判斷文件是否排在游標之後（降序中的「更舊」方向）。
func afterCursor(doc Document, createdAt time.Time, messageID string) bool {
	if doc.CreatedAt.Before(createdAt) {
		return true
	}
	if doc.CreatedAt.Equal(createdAt) && doc.MessageID < messageID {
		return true
	}
	return false
}

// MemoryIndex 是 Index 的記憶體實作（開發與測試用）：行程內
// 索引，不持久，重啟後為空（須由 Worker 重建迴圈填充，見本檔
// 檔頭）。全量掃描後過濾，僅供開發與測試，不得用於正式環境。
type MemoryIndex struct {
	mu   sync.Mutex
	docs map[string]Document
	seen map[string]struct{}
}

// NewMemoryIndex 建立空的記憶體索引。
func NewMemoryIndex() *MemoryIndex {
	return &MemoryIndex{
		docs: map[string]Document{},
		seen: map[string]struct{}{},
	}
}

// Upsert 寫入或更新投影文件（見 Index 介面說明）。同一 eventID
// 重放時為 no-op；較舊 SequenceNumber 的投影不得覆蓋較新者。
func (m *MemoryIndex) Upsert(_ context.Context, doc Document, eventID string) error {
	if eventID == "" {
		return errors.New("search: eventID 不可為空")
	}
	if doc.MessageID == "" || doc.ConversationID == "" {
		return errors.New("search: 文件識別碼不可為空")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.seen[eventID]; ok {
		return nil
	}
	m.seen[eventID] = struct{}{}
	if old, ok := m.docs[doc.MessageID]; ok && doc.SequenceNumber < old.SequenceNumber {
		return nil
	}
	m.docs[doc.MessageID] = doc
	return nil
}

// Remove 刪除投影文件（見 Index 介面說明）。同一 eventID 重放時
// 為 no-op；文件不存在時為冪等成功。
func (m *MemoryIndex) Remove(_ context.Context, messageID string, eventID string) error {
	if eventID == "" {
		return errors.New("search: eventID 不可為空")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.seen[eventID]; ok {
		return nil
	}
	m.seen[eventID] = struct{}{}
	delete(m.docs, messageID)
	return nil
}

// Search 回傳符合條件的全部文件（見 Index 介面說明）。
func (m *MemoryIndex) Search(_ context.Context, q Query) ([]Document, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Document
	for _, doc := range m.docs {
		if q.ConversationID != "" && doc.ConversationID != q.ConversationID {
			continue
		}
		if q.SenderID != "" && doc.SenderID != q.SenderID {
			continue
		}
		if q.Since != nil && doc.CreatedAt.Before(*q.Since) {
			continue
		}
		if q.Until != nil && doc.CreatedAt.After(*q.Until) {
			continue
		}
		if !matches(doc, q.Terms) {
			continue
		}
		out = append(out, doc)
	}
	sortDocuments(out)
	return out, nil
}

// ReplaceAll 全量取代索引內容（見 Index 介面說明）。先完整驗證
// 後再替換，不得留下部分重建的索引。重建不經事件冪等集合
// （重建來源為全量掃描，事件集合於重建後失效——重建後重放的
// 舊事件可能被再次套用，呼叫方須保證重建期間消費端暫停或事件
// 順序，見 STATUS.md 已知限制）。
func (m *MemoryIndex) ReplaceAll(_ context.Context, docs []Document) error {
	next := map[string]Document{}
	for _, doc := range docs {
		if doc.MessageID == "" || doc.ConversationID == "" {
			return errors.New("search: 文件識別碼不可為空")
		}
		if old, ok := next[doc.MessageID]; ok && doc.SequenceNumber < old.SequenceNumber {
			continue
		}
		next[doc.MessageID] = doc
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.docs = next
	m.seen = map[string]struct{}{}
	return nil
}

// MeilisearchIndex 的實作見 meilisearch.go（DialMeilisearch
// 建構；NewMeilisearchIndex 保留未設定佔位供既有測試使用）。

// Projector 是事件驅動的索引投影器：消費 Outbox 事件
// （message.received／message.recalled，見 DATA_FLOW.md §8），
// 將投影寫入索引。呼叫方（Worker 消費迴圈，後續里程碑）須保證
// 「至少一次」投遞，本投影器以 eventID 冪等去重。本投影器不
// 在訊息寫入路徑內呼叫，不得由業務寫入同步驅動。
type Projector struct {
	idx Index
}

// NewProjector 建立投影器。
func NewProjector(idx Index) *Projector { return &Projector{idx: idx} }

// ApplyReceived 將一則 message.received 事件投影為索引文件。
// eventID 為 Outbox 事件識別碼（冪等鍵，不得為空）。
func (p *Projector) ApplyReceived(ctx context.Context, doc Document, eventID string) error {
	return p.idx.Upsert(ctx, doc, eventID)
}

// ApplyRecalled 將一則 message.recalled 事件投影為索引刪除：
// 收回訊息的內容不得殘留於索引。eventID 為 Outbox 事件識別碼
// （冪等鍵，不得為空）。
func (p *Projector) ApplyRecalled(ctx context.Context, messageID string, eventID string) error {
	return p.idx.Remove(ctx, messageID, eventID)
}

// Rebuild 以全量文件重建索引（重建入口，見本檔檔頭）。文件
// 來源須為 PostgreSQL 全量掃描，由呼叫方提供。
func (p *Projector) Rebuild(ctx context.Context, docs []Document) error {
	return p.idx.ReplaceAll(ctx, docs)
}

// Service 是訊息搜尋的查詢服務：查詢索引、權限過濾、分頁。
// 權限過濾在回傳前、逐筆經 MembershipChecker 執行；檢查失敗時
// fail-closed（回傳錯誤，不得回傳未過濾的結果）。
type Service struct {
	idx     Index
	checker MembershipChecker
}

// NewService 建立查詢服務。checker 為 nil 時，查詢一律回傳
// ErrIndexUnavailable（fail-closed，不得在無權限檢查時回傳結果）。
func NewService(idx Index, checker MembershipChecker) *Service {
	return &Service{idx: idx, checker: checker}
}

// Search 執行一次訊息搜尋：驗證查詢、分頁前先以索引條件取
// 候選文件，再逐筆驗證成員身份後回傳一頁。非成員對話的文件
// 一律不回傳，不得洩漏存在性（結果中直接略過，不報錯）。
func (s *Service) Search(ctx context.Context, userID string, q Query) (*Page, error) {
	if userID == "" {
		return nil, validationError("user_id", "REQUIRED", "使用者識別碼不可為空。")
	}
	if len(q.Terms) == 0 {
		return nil, validationError("q", "REQUIRED", "關鍵字不可為空。")
	}
	if s.checker == nil {
		return nil, ErrIndexUnavailable
	}
	limit := q.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	var cursorTime time.Time
	var cursorID string
	if q.Cursor != "" {
		t, id, err := parseCursor(q.Cursor)
		if err != nil {
			return nil, validationError("cursor", "INVALID_VALUE", "cursor 格式錯誤。")
		}
		cursorTime, cursorID = t, id
	}
	candidates, err := s.idx.Search(ctx, q)
	if err != nil {
		if errors.Is(err, ErrNotConfigured) {
			return nil, ErrIndexUnavailable
		}
		return nil, err
	}
	var permitted []Document
	for _, doc := range candidates {
		if q.Cursor != "" && !afterCursor(doc, cursorTime, cursorID) {
			continue
		}
		member, err := s.checker.IsActiveMember(ctx, doc.ConversationID, userID)
		if err != nil {
			return nil, ErrIndexUnavailable
		}
		if !member {
			continue
		}
		permitted = append(permitted, doc)
		if len(permitted) > limit {
			break
		}
	}
	page := &Page{Documents: []Document{}}
	if len(permitted) > limit {
		page.Documents = permitted[:limit]
		page.NextCursor = cursorOf(permitted[limit-1])
	} else {
		page.Documents = permitted
	}
	return page, nil
}

// ValidateQuery 驗證原始查詢字串與分頁參數，回傳正規化後的
// Query（供 handler 使用）。q 經正規化後不可為空、至多
// MaxQueryRunes runes；since／until 須為 RFC3339 時間且 until
// 不得早於 since。
func ValidateQuery(rawQuery string, limit int, cursor string, conversationID, senderID string, since, until *time.Time) (Query, error) {
	if utf8.RuneCountInString(rawQuery) > MaxQueryRunes {
		return Query{}, validationError("q", "TOO_LONG", "關鍵字過長。")
	}
	terms := normalizeQuery(rawQuery)
	if len(terms) == 0 {
		return Query{}, validationError("q", "REQUIRED", "關鍵字不可為空。")
	}
	if since != nil && until != nil && until.Before(*since) {
		return Query{}, validationError("until", "INVALID_VALUE", "until 不得早於 since。")
	}
	return Query{
		Terms:          terms,
		ConversationID: conversationID,
		SenderID:       senderID,
		Since:          since,
		Until:          until,
		Limit:          limit,
		Cursor:         cursor,
	}, nil
}
