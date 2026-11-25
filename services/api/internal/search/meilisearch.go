package search

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

// Meilisearch 轉接器（本里程碑落地，取代先前的佔位實作）。
//
// 依賴：github.com/meilisearch/meilisearch-go v0.37.0。
// 索引結構（實作預設，待定稿覆核）：索引 UID 見
// MessagesIndexUID；主鍵 message_id；文件欄位見 meiliDocument；
// 查詢以 created_at 降序、message_id 降序排序。索引設定
// （filterableAttributes：conversation_id、sender_id、created_at；
// sortableAttributes：created_at、message_id）由部署方預先建立，
// 轉接器不寫入設定——未設定時篩選／排序可能失敗，查詢回傳錯誤
// （fail-closed，不得回傳未篩選的結果）。
//
// 冪等：Upsert／Remove 依 eventID 去重（行程內 seen 集合，寫入
// 成功後才標記；重建後集合重置，見 MemoryIndex 相同限制）。
// Upsert 另以 SequenceNumber 擋舊版本（先讀既有文件比對）。
// 寫入為同步語義：文件任務須經 WaitForTask 確認 succeeded 才
// 回傳，失敗時回傳錯誤（呼叫方重試；NATS 消費端以 Nak 重投）。
//
// 建構失敗即 fail-closed：DialMeilisearch 在 host 或 apiKey 缺失
// 時回傳錯誤，不得靜默退回記憶體索引（索引分流會造成搜尋結果
// 不一致）。未設定的佔位建構子 NewMeilisearchIndex 保留供既有
// 測試使用，任何呼叫皆回傳 ErrNotConfigured。

// MessagesIndexUID 是訊息搜尋索引的 UID（實作預設，待定稿覆核）。
const MessagesIndexUID = "messages"

// meiliTaskPollInterval 是寫入任務的輪詢間隔（實作預設）。
const meiliTaskPollInterval = 100 * time.Millisecond

// meiliSearchPageSize 是 Search 分頁抓取的每頁筆數（實作預設）。
const meiliSearchPageSize = 1000

// meiliDocument 是 Meilisearch 內的文件形態（JSON 欄位名即
// Meilisearch 欄位名）。created_at 存 RFC3339Nano UTC 字串，
// 字串排序與時間排序一致。
type meiliDocument struct {
	MessageID      string `json:"message_id"`
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	Kind           string `json:"kind"`
	Content        string `json:"content"`
	SequenceNumber int64  `json:"sequence_number"`
	CreatedAt      string `json:"created_at"`
}

func toMeiliDocument(doc Document) meiliDocument {
	return meiliDocument{
		MessageID:      doc.MessageID,
		ConversationID: doc.ConversationID,
		SenderID:       doc.SenderID,
		Kind:           doc.Kind,
		Content:        doc.Content,
		SequenceNumber: doc.SequenceNumber,
		CreatedAt:      doc.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (d meiliDocument) toDocument() (Document, error) {
	createdAt, err := time.Parse(time.RFC3339Nano, d.CreatedAt)
	if err != nil {
		return Document{}, fmt.Errorf("search: Meilisearch 文件時間格式錯誤: %w", err)
	}
	return Document{
		MessageID:      d.MessageID,
		ConversationID: d.ConversationID,
		SenderID:       d.SenderID,
		Kind:           d.Kind,
		Content:        d.Content,
		SequenceNumber: d.SequenceNumber,
		CreatedAt:      createdAt,
	}, nil
}

// meilisearchBackend 是轉接器對 SDK 的最小依賴面：
// meilisearch.IndexManager 滿足本介面，測試以假實作注入。
type meilisearchBackend interface {
	AddDocuments(ctx context.Context, documentsPtr interface{}, opts *meilisearch.DocumentOptions) (*meilisearch.TaskInfo, error)
	DeleteDocument(ctx context.Context, identifier string, opts *meilisearch.DocumentOptions) (*meilisearch.TaskInfo, error)
	DeleteAllDocuments(ctx context.Context, opts *meilisearch.DocumentOptions) (*meilisearch.TaskInfo, error)
	GetDocument(ctx context.Context, identifier string, request *meilisearch.DocumentQuery, documentPtr interface{}) error
	Search(ctx context.Context, query string, request *meilisearch.SearchRequest) (*meilisearch.SearchResponse, error)
	WaitForTask(ctx context.Context, taskUID int64, interval time.Duration) (*meilisearch.Task, error)
}

// MeilisearchIndex 是 Index 的 Meilisearch 實作。backend 為 nil
// 時為未設定佔位（見 NewMeilisearchIndex），任何呼叫皆回傳
// ErrNotConfigured。
type MeilisearchIndex struct {
	backend meilisearchBackend
	mu      sync.Mutex
	seen    map[string]struct{}
}

// NewMeilisearchIndex 建立未設定的佔位轉接器（永遠回傳
// ErrNotConfigured）。僅供既有測試與未設定模式使用；生產接線
// 一律經 DialMeilisearch。
func NewMeilisearchIndex() *MeilisearchIndex {
	return &MeilisearchIndex{seen: map[string]struct{}{}}
}

// NewMeilisearchIndexWithBackend 以給定的後端建立轉接器（測試用）。
func NewMeilisearchIndexWithBackend(backend meilisearchBackend) *MeilisearchIndex {
	return &MeilisearchIndex{backend: backend, seen: map[string]struct{}{}}
}

// DialMeilisearch 連線至 Meilisearch 並取回指定索引的轉接器。
// host 與 apiKey 任一缺失時回傳錯誤（fail-closed，不得靜默
// 退回記憶體索引）。indexUID 為空時取 MessagesIndexUID。
func DialMeilisearch(host, apiKey, indexUID string) (*MeilisearchIndex, error) {
	if strings.TrimSpace(host) == "" {
		return nil, errors.New("search: Meilisearch 位址不可為空")
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("search: Meilisearch API 金鑰不可為空")
	}
	if indexUID == "" {
		indexUID = MessagesIndexUID
	}
	client := meilisearch.New(host, meilisearch.WithAPIKey(apiKey))
	return NewMeilisearchIndexWithBackend(client.Index(indexUID)), nil
}

func (m *MeilisearchIndex) markSeen(eventID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seen[eventID] = struct{}{}
}

func (m *MeilisearchIndex) hasSeen(eventID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.seen[eventID]
	return ok
}

// waitTask 等待寫入任務完成；任務失敗時回傳錯誤。
func (m *MeilisearchIndex) waitTask(ctx context.Context, task *meilisearch.TaskInfo) error {
	if task == nil {
		return errors.New("search: Meilisearch 未回傳任務資訊")
	}
	done, err := m.backend.WaitForTask(ctx, task.TaskUID, meiliTaskPollInterval)
	if err != nil {
		return fmt.Errorf("search: 等待 Meilisearch 任務失敗: %w", err)
	}
	if done.Status != meilisearch.TaskStatusSucceeded {
		return fmt.Errorf("search: Meilisearch 任務未成功（狀態 %s）", done.Status)
	}
	return nil
}

// isNotFound 判斷 SDK 錯誤是否為 404（文件不存在）。
func isNotFound(err error) bool {
	var apiErr *meilisearch.Error
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404
	}
	return false
}

// Upsert 寫入或更新投影文件（見 Index 介面說明）。同一 eventID
// 重放時為 no-op；較舊 SequenceNumber 的投影不得覆蓋較新者。
func (m *MeilisearchIndex) Upsert(ctx context.Context, doc Document, eventID string) error {
	if m.backend == nil {
		return ErrNotConfigured
	}
	if eventID == "" {
		return errors.New("search: eventID 不可為空")
	}
	if doc.MessageID == "" || doc.ConversationID == "" {
		return errors.New("search: 文件識別碼不可為空")
	}
	if m.hasSeen(eventID) {
		return nil
	}
	// 版本守衛：既有文件較新時跳過（不得覆蓋）。
	var existing meiliDocument
	if err := m.backend.GetDocument(ctx, doc.MessageID, nil, &existing); err != nil {
		if !isNotFound(err) {
			return fmt.Errorf("search: 讀取既有文件失敗: %w", err)
		}
	} else if doc.SequenceNumber < existing.SequenceNumber {
		m.markSeen(eventID)
		return nil
	}
	pk := "message_id"
	task, err := m.backend.AddDocuments(ctx, []meiliDocument{toMeiliDocument(doc)}, &meilisearch.DocumentOptions{PrimaryKey: &pk})
	if err != nil {
		return fmt.Errorf("search: 寫入 Meilisearch 文件失敗: %w", err)
	}
	if err := m.waitTask(ctx, task); err != nil {
		return err
	}
	m.markSeen(eventID)
	return nil
}

// Remove 刪除投影文件（見 Index 介面說明）。同一 eventID 重放
// 時為 no-op；文件不存在時為冪等成功。
func (m *MeilisearchIndex) Remove(ctx context.Context, messageID string, eventID string) error {
	if m.backend == nil {
		return ErrNotConfigured
	}
	if eventID == "" {
		return errors.New("search: eventID 不可為空")
	}
	if m.hasSeen(eventID) {
		return nil
	}
	task, err := m.backend.DeleteDocument(ctx, messageID, nil)
	if err != nil {
		return fmt.Errorf("search: 刪除 Meilisearch 文件失敗: %w", err)
	}
	if err := m.waitTask(ctx, task); err != nil {
		return err
	}
	m.markSeen(eventID)
	return nil
}

// escapeFilterValue 跳脫篩選字串值（反斜線與雙引號）。
func escapeFilterValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return `"` + v + `"`
}

// Search 回傳符合條件的全部文件（見 Index 介面說明）。以分頁
// 抓取全部候選（Meilisearch 排名不作最終排序依據，回傳前依
// 建立時間降序、訊息識別碼降序重排，保證游標穩定）。
func (m *MeilisearchIndex) Search(ctx context.Context, q Query) ([]Document, error) {
	if m.backend == nil {
		return nil, ErrNotConfigured
	}
	var filters []string
	if q.ConversationID != "" {
		filters = append(filters, "conversation_id = "+escapeFilterValue(q.ConversationID))
	}
	if q.SenderID != "" {
		filters = append(filters, "sender_id = "+escapeFilterValue(q.SenderID))
	}
	if q.Since != nil {
		filters = append(filters, "created_at >= "+escapeFilterValue(q.Since.UTC().Format(time.RFC3339Nano)))
	}
	if q.Until != nil {
		filters = append(filters, "created_at <= "+escapeFilterValue(q.Until.UTC().Format(time.RFC3339Nano)))
	}
	var filter any
	if len(filters) > 0 {
		filter = strings.Join(filters, " AND ")
	}
	var out []Document
	for offset := int64(0); ; offset += meiliSearchPageSize {
		resp, err := m.backend.Search(ctx, strings.Join(q.Terms, " "), &meilisearch.SearchRequest{
			Filter: filter,
			Sort:   []string{"created_at:desc", "message_id:desc"},
			Limit:  meiliSearchPageSize,
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("search: Meilisearch 查詢失敗: %w", err)
		}
		var hits []meiliDocument
		if err := resp.Hits.Decode(&hits); err != nil {
			return nil, fmt.Errorf("search: Meilisearch 結果解碼失敗: %w", err)
		}
		for _, h := range hits {
			doc, err := h.toDocument()
			if err != nil {
				return nil, err
			}
			out = append(out, doc)
		}
		if int64(len(hits)) < meiliSearchPageSize {
			break
		}
	}
	sortDocuments(out)
	return out, nil
}

// ReplaceAll 全量取代索引內容（見 Index 介面說明）。先刪除全部
// 文件、再寫入新文件；寫入為空時只刪除。冪等集合於重建後重置。
func (m *MeilisearchIndex) ReplaceAll(ctx context.Context, docs []Document) error {
	if m.backend == nil {
		return ErrNotConfigured
	}
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
	task, err := m.backend.DeleteAllDocuments(ctx, nil)
	if err != nil {
		return fmt.Errorf("search: 清空 Meilisearch 索引失敗: %w", err)
	}
	if err := m.waitTask(ctx, task); err != nil {
		return err
	}
	if len(next) == 0 {
		m.mu.Lock()
		m.seen = map[string]struct{}{}
		m.mu.Unlock()
		return nil
	}
	meiliDocs := make([]meiliDocument, 0, len(next))
	for _, doc := range next {
		meiliDocs = append(meiliDocs, toMeiliDocument(doc))
	}
	pk := "message_id"
	task, err = m.backend.AddDocuments(ctx, meiliDocs, &meilisearch.DocumentOptions{PrimaryKey: &pk})
	if err != nil {
		return fmt.Errorf("search: 重建 Meilisearch 索引失敗: %w", err)
	}
	if err := m.waitTask(ctx, task); err != nil {
		return err
	}
	m.mu.Lock()
	m.seen = map[string]struct{}{}
	m.mu.Unlock()
	return nil
}
