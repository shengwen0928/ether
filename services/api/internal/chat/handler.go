// Package chat 的 HTTP handler：/api/v1/conversations 前綴下的對話端點
// （API_STANDARDS.md §2）。全部端點皆須經 RequireAuth；非成員的
// 一切查詢皆以 404 回應，不得洩漏對話存在性（ERROR_CODES.md §6.3）。
package chat

import (
	"context"
	"errors"
	"net/http"
	"time"

	"ether.local/api/internal/transport"
)

// UnreadCounter 是 Message 模組提供的未讀數查詢（依賴 DAG：
// Message 讀 Chat 的成員水位，Chat 的清單回應經此介面豐富化；
// 介面由接線層注入，不得 import message 套件）。
type UnreadCounter interface {
	UnreadCount(ctx context.Context, conversationID, userID string) (int, error)
}

// Handler 持有 conversations 端點所需的服務。
type Handler struct {
	Service       *Service
	UnreadCounter UnreadCounter // 可為空：未接線時未讀數回 0。
	CurrentUser   func(r *http.Request) (string, bool)
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

func writeChatError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrConversationNotFound) {
		transport.WriteError(w, http.StatusNotFound, "ETHER_COMMON_NOT_FOUND", "找不到資源。")
		return
	}
	transport.WriteServiceError(w, err, nil)
}

type conversationResponse struct {
	ConversationID string     `json:"conversation_id"`
	Type           string     `json:"type"`
	Title          string     `json:"title"`
	CreatedBy      string     `json:"created_by"`
	LastMessageID  string     `json:"last_message_id,omitempty"`
	LastMessageAt  *time.Time `json:"last_message_at,omitempty"`
	UnreadCount    int        `json:"unread_count"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func toConversationResponse(c *Conversation, unread int) conversationResponse {
	var lastAt *time.Time
	if c.LastMessageAt != nil {
		u := c.LastMessageAt.UTC()
		lastAt = &u
	}
	return conversationResponse{
		ConversationID: c.ID,
		Type:           string(c.Type),
		Title:          c.Title,
		CreatedBy:      c.CreatedBy,
		LastMessageID:  c.LastMessageID,
		LastMessageAt:  lastAt,
		UnreadCount:    unread,
		CreatedAt:      c.CreatedAt.UTC(),
		UpdatedAt:      c.UpdatedAt.UTC(),
	}
}

func (h *Handler) unreadCount(r *http.Request, conversationID, userID string) int {
	if h.UnreadCounter == nil {
		return 0
	}
	n, err := h.UnreadCounter.UnreadCount(r.Context(), conversationID, userID)
	if err != nil {
		return 0
	}
	return n
}

// ListConversations 列出呼叫者的對話（GET /api/v1/conversations）。
func (h *Handler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	convs, err := h.Service.ListConversations(r.Context(), userID)
	if err != nil {
		writeChatError(w, err)
		return
	}
	out := make([]conversationResponse, 0, len(convs))
	for _, c := range convs {
		out = append(out, toConversationResponse(c, h.unreadCount(r, c.ID, userID)))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type createConversationBody struct {
	Type        string   `json:"type"`
	OtherUserID string   `json:"other_user_id"`
	Title       string   `json:"title"`
	MemberIDs   []string `json:"member_ids"`
}

// CreateConversation 建立對話（POST /api/v1/conversations）：
// type=direct 時須 other_user_id；type=group 時須 title＋member_ids。
// 既有一對一對話回傳 200＋既有資源，新建回傳 201。
func (h *Handler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req createConversationBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	switch req.Type {
	case string(TypeDirect):
		conv, isNew, err := h.Service.CreateDirect(r.Context(), userID, req.OtherUserID)
		if err != nil {
			writeChatError(w, err)
			return
		}
		status := http.StatusOK
		if isNew {
			status = http.StatusCreated
		}
		transport.WriteJSON(w, status, toConversationResponse(conv, 0))
	case string(TypeGroup):
		conv, err := h.Service.CreateGroup(r.Context(), userID, req.Title, req.MemberIDs)
		if err != nil {
			writeChatError(w, err)
			return
		}
		transport.WriteJSON(w, http.StatusCreated, toConversationResponse(conv, 0))
	default:
		transport.WriteValidationError(w, []transport.FieldError{{
			Field: "type", Issue: "INVALID_VALUE", Message: "type 須為 direct 或 group。",
		}})
	}
}

// GetConversation 讀取單一對話（GET /api/v1/conversations/{conversation_id}）。
func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	conv, err := h.Service.GetConversation(r.Context(), userID, r.PathValue("conversation_id"))
	if err != nil {
		writeChatError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toConversationResponse(conv, h.unreadCount(r, conv.ID, userID)))
}

type memberResponse struct {
	UserID           string     `json:"user_id"`
	Role             string     `json:"role"`
	JoinedAt         time.Time  `json:"joined_at"`
	LeftAt           *time.Time `json:"left_at,omitempty"`
	LastReadSequence int64      `json:"last_read_sequence"`
}

func toMemberResponse(m *Member) memberResponse {
	var left *time.Time
	if m.LeftAt != nil {
		u := m.LeftAt.UTC()
		left = &u
	}
	return memberResponse{
		UserID:           m.UserID,
		Role:             string(m.Role),
		JoinedAt:         m.JoinedAt.UTC(),
		LeftAt:           left,
		LastReadSequence: m.LastReadSequence,
	}
}

// ListMembers 列出對話成員（GET /api/v1/conversations/{conversation_id}/members）。
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	members, err := h.Service.ListMembers(r.Context(), userID, r.PathValue("conversation_id"))
	if err != nil {
		writeChatError(w, err)
		return
	}
	out := make([]memberResponse, 0, len(members))
	for _, m := range members {
		out = append(out, toMemberResponse(m))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type addMembersBody struct {
	MemberIDs []string `json:"member_ids"`
}

// AddMembers 新增群組成員（POST /api/v1/conversations/{conversation_id}/members）。
func (h *Handler) AddMembers(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req addMembersBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	added, err := h.Service.AddMembers(r.Context(), userID, r.PathValue("conversation_id"), req.MemberIDs)
	if err != nil {
		writeChatError(w, err)
		return
	}
	out := make([]memberResponse, 0, len(added))
	for _, m := range added {
		out = append(out, toMemberResponse(m))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

// Leave 離開對話（POST /api/v1/conversations/{conversation_id}/leave）。
func (h *Handler) Leave(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	if err := h.Service.Leave(r.Context(), userID, r.PathValue("conversation_id")); err != nil {
		writeChatError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "left"})
}
