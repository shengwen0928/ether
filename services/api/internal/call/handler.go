// Package call 的 HTTP handler：/api/v1/calls 前綴下的通話端點
// （API_STANDARDS.md §2）。全部端點皆須經 RequireAuth；非參與者的
// 一切查詢皆以 404 回應，不得洩漏通話存在性（ERROR_CODES.md §6.3）。
// 回應一律 DTO（snake_case），永不回傳 Token、金鑰材料或通話內容
// （本模組不持有此類資料）。
package call

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"ether.local/api/internal/transport"
)

// Handler 持有 calls 端點所需的服務。
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

// writeCallError 將領域錯誤對應為 HTTP 回應：欄位驗證錯誤 →
// 400＋details；ErrCallNotFound → 404（訊息不可區分）；其餘 → 500。
func writeCallError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrCallNotFound) {
		transport.WriteError(w, http.StatusNotFound, "ETHER_COMMON_NOT_FOUND", "找不到資源。")
		return
	}
	transport.WriteServiceError(w, err, nil)
}

func utcTime(t time.Time) time.Time { return t.UTC() }

func utcTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

type callResponse struct {
	CallID           string     `json:"call_id"`
	ConversationID   string     `json:"conversation_id,omitempty"`
	GroupID          string     `json:"group_id,omitempty"`
	InitiatorID      string     `json:"initiator_id"`
	CallType         string     `json:"call_type"`
	Status           string     `json:"status"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	EndedAt          *time.Time `json:"ended_at,omitempty"`
	DurationSeconds  *int64     `json:"duration_seconds,omitempty"`
	EndedBy          string     `json:"ended_by,omitempty"`
	EndReason        string     `json:"end_reason,omitempty"`
	ParticipantCount int        `json:"participant_count"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func toCallResponse(entry *CallEntry) callResponse {
	c := entry.Call
	return callResponse{
		CallID:           c.ID,
		ConversationID:   c.ConversationID,
		GroupID:          c.GroupID,
		InitiatorID:      c.InitiatorID,
		CallType:         string(c.Type),
		Status:           string(c.Status),
		StartedAt:        utcTimePtr(c.StartedAt),
		EndedAt:          utcTimePtr(c.EndedAt),
		DurationSeconds:  c.Duration,
		EndedBy:          c.EndedBy,
		EndReason:        string(c.EndReason),
		ParticipantCount: entry.ParticipantCount,
		CreatedAt:        utcTime(c.CreatedAt),
		UpdatedAt:        utcTime(c.UpdatedAt),
	}
}

type participantResponse struct {
	UserID    string     `json:"user_id"`
	JoinedAt  *time.Time `json:"joined_at,omitempty"`
	LeftAt    *time.Time `json:"left_at,omitempty"`
	Result    string     `json:"result"`
	CreatedAt time.Time  `json:"created_at"`
}

func toParticipantResponse(p *Participant) participantResponse {
	return participantResponse{
		UserID:    p.UserID,
		JoinedAt:  utcTimePtr(p.JoinedAt),
		LeftAt:    utcTimePtr(p.LeftAt),
		Result:    string(p.Result),
		CreatedAt: utcTime(p.CreatedAt),
	}
}

type callEventResponse struct {
	EventID    string    `json:"event_id"`
	UserID     string    `json:"user_id,omitempty"`
	EventType  string    `json:"event_type"`
	OccurredAt time.Time `json:"occurred_at"`
}

func toCallEventResponse(e *CallEvent) callEventResponse {
	return callEventResponse{
		EventID:    e.ID,
		UserID:     e.UserID,
		EventType:  string(e.Type),
		OccurredAt: utcTime(e.OccurredAt),
	}
}

type initiateCallBody struct {
	ConversationID string `json:"conversation_id"`
	GroupID        string `json:"group_id"`
	CallType       string `json:"call_type"`
}

// InitiateCall 發起通話（POST /api/v1/calls）。成功回 201。
func (h *Handler) InitiateCall(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req initiateCallBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	entry, err := h.Service.Initiate(r.Context(), userID, InitiateInput{
		ConversationID: req.ConversationID,
		GroupID:        req.GroupID,
		Type:           CallType(req.CallType),
	})
	if err != nil {
		writeCallError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusCreated, toCallResponse(entry))
}

// ListCalls 列出呼叫者參與的通話（GET /api/v1/calls，游標分頁）。
func (h *Handler) ListCalls(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			transport.WriteError(w, http.StatusBadRequest,
				"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
			return
		}
		limit = n
	}
	entries, nextCursor, err := h.Service.ListCalls(r.Context(), userID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeCallError(w, err)
		return
	}
	out := make([]callResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toCallResponse(e))
	}
	transport.WriteJSON(w, http.StatusOK, map[string]any{
		"calls":       out,
		"next_cursor": nextCursor,
	})
}

// GetCall 讀取單一通話（GET /api/v1/calls/{call_id}）。
func (h *Handler) GetCall(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entry, err := h.Service.GetCall(r.Context(), userID, r.PathValue("call_id"))
	if err != nil {
		writeCallError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toCallResponse(entry))
}

// AnswerCall 接聽來電（POST /api/v1/calls/{call_id}/answer）。
func (h *Handler) AnswerCall(w http.ResponseWriter, r *http.Request) {
	h.respond(w, r, func(ctxUserID, callID string) (*CallEntry, error) {
		return h.Service.Answer(r.Context(), ctxUserID, callID)
	})
}

// DeclineCall 拒接來電（POST /api/v1/calls/{call_id}/decline）。
func (h *Handler) DeclineCall(w http.ResponseWriter, r *http.Request) {
	h.respond(w, r, func(ctxUserID, callID string) (*CallEntry, error) {
		return h.Service.Decline(r.Context(), ctxUserID, callID)
	})
}

// BusyCall 回報忙線（POST /api/v1/calls/{call_id}/busy）。
func (h *Handler) BusyCall(w http.ResponseWriter, r *http.Request) {
	h.respond(w, r, func(ctxUserID, callID string) (*CallEntry, error) {
		return h.Service.Busy(r.Context(), ctxUserID, callID)
	})
}

// EndCall 結束通話（POST /api/v1/calls/{call_id}/end）。
func (h *Handler) EndCall(w http.ResponseWriter, r *http.Request) {
	h.respond(w, r, func(ctxUserID, callID string) (*CallEntry, error) {
		return h.Service.End(r.Context(), ctxUserID, callID)
	})
}

func (h *Handler) respond(w http.ResponseWriter, r *http.Request, fn func(string, string) (*CallEntry, error)) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	entry, err := fn(userID, r.PathValue("call_id"))
	if err != nil {
		writeCallError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toCallResponse(entry))
}

// ListParticipants 列出通話參與者（GET /api/v1/calls/{call_id}/participants）。
func (h *Handler) ListParticipants(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	participants, err := h.Service.ListParticipants(r.Context(), userID, r.PathValue("call_id"))
	if err != nil {
		writeCallError(w, err)
		return
	}
	out := make([]participantResponse, 0, len(participants))
	for _, p := range participants {
		out = append(out, toParticipantResponse(p))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

// ListEvents 列出通話事件（GET /api/v1/calls/{call_id}/events）。
// 事件回應不含 payload（僅狀態事件的存在性與類型；payload 只供
// 內部稽核，不得經 API 外洩）。
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	events, err := h.Service.ListEvents(r.Context(), userID, r.PathValue("call_id"))
	if err != nil {
		writeCallError(w, err)
		return
	}
	out := make([]callEventResponse, 0, len(events))
	for _, e := range events {
		out = append(out, toCallEventResponse(e))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}
