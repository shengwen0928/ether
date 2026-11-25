// 訊息搜尋的 HTTP handler（見 handler.go 的檔頭慣例：全部端點
// 皆須經 RequireAuth；結果須經成員身份過濾，非成員對話的文件
// 一律不回傳，不得洩漏存在性）。
package search

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"ether.local/api/internal/transport"
)

// Handler 承載訊息搜尋端點的依賴。
type Handler struct {
	Service     *Service
	CurrentUser func(r *http.Request) (string, bool)
}

func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := h.CurrentUser(r)
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return "", false
	}
	return userID, true
}

func writeSearchError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrIndexUnavailable) {
		transport.WriteError(w, http.StatusServiceUnavailable,
			"ETHER_COMMON_UNAVAILABLE", "搜尋服務暫不可用。")
		return
	}
	transport.WriteServiceError(w, err, nil)
}

func parseSearchTimeParam(r *http.Request, name string) (*time.Time, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type searchResultResponse struct {
	MessageID      string    `json:"message_id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Kind           string    `json:"kind"`
	Content        string    `json:"content"`
	SequenceNumber int64     `json:"sequence_number"`
	CreatedAt      time.Time `json:"created_at"`
}

type searchMessagesResponse struct {
	Results    []searchResultResponse `json:"results"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

func toSearchResultResponse(doc Document) searchResultResponse {
	return searchResultResponse{
		MessageID:      doc.MessageID,
		ConversationID: doc.ConversationID,
		SenderID:       doc.SenderID,
		Kind:           doc.Kind,
		Content:        doc.Content,
		SequenceNumber: doc.SequenceNumber,
		CreatedAt:      doc.CreatedAt.UTC(),
	}
}

// SearchMessages 搜尋呼叫者為成員的對話內的訊息
// （GET /api/v1/search/messages?q=...）。
// 查詢參數一律 snake_case：q（必填）、conversation_id、sender_id、
// since／until（RFC3339）、limit（預設 20、上限 50）、cursor。
// 索引為空時回傳空結果，不得謊稱結果完整（索引填充狀態見
// Service／Projector，填充迴圈屬後續里程碑）。
func (h *Handler) SearchMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	limit := 0
	if raw := query.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			transport.WriteValidationError(w, []transport.FieldError{{
				Field: "limit", Issue: "INVALID_VALUE", Message: "limit 須為正整數。",
			}})
			return
		}
		limit = n
	}
	since, err := parseSearchTimeParam(r, "since")
	if err != nil {
		transport.WriteValidationError(w, []transport.FieldError{{
			Field: "since", Issue: "INVALID_VALUE", Message: "since 須為 RFC3339 時間。",
		}})
		return
	}
	until, err := parseSearchTimeParam(r, "until")
	if err != nil {
		transport.WriteValidationError(w, []transport.FieldError{{
			Field: "until", Issue: "INVALID_VALUE", Message: "until 須為 RFC3339 時間。",
		}})
		return
	}
	q, err := ValidateQuery(query.Get("q"), limit, query.Get("cursor"),
		query.Get("conversation_id"), query.Get("sender_id"), since, until)
	if err != nil {
		writeSearchError(w, err)
		return
	}
	page, err := h.Service.Search(r.Context(), userID, q)
	if err != nil {
		writeSearchError(w, err)
		return
	}
	out := make([]searchResultResponse, 0, len(page.Documents))
	for _, doc := range page.Documents {
		out = append(out, toSearchResultResponse(doc))
	}
	transport.WriteJSON(w, http.StatusOK, searchMessagesResponse{
		Results:    out,
		NextCursor: page.NextCursor,
	})
}
