// 訊息反應與置頂的 HTTP handler（見 handler.go 的檔頭慣例：
// 全部端點皆須經 RequireAuth；非成員的一切查詢皆以 404 回應，
// 不得洩漏存在性）。
package message

import (
	"net/http"
	"time"

	"ether.local/api/internal/transport"
)

type reactionCountResponse struct {
	Reaction    string `json:"reaction"`
	Count       int    `json:"count"`
	ReactedByMe bool   `json:"reacted_by_me"`
}

type reactionSummaryResponse struct {
	MessageID string                  `json:"message_id"`
	Reactions []reactionCountResponse `json:"reactions"`
}

func toReactionSummaryResponse(s *ReactionSummary) reactionSummaryResponse {
	out := reactionSummaryResponse{MessageID: s.MessageID, Reactions: []reactionCountResponse{}}
	for _, c := range s.Reactions {
		out.Reactions = append(out.Reactions, reactionCountResponse{
			Reaction:    c.Emoji,
			Count:       c.Count,
			ReactedByMe: c.ReactedByMe,
		})
	}
	return out
}

type setReactionBody struct {
	Reaction string `json:"reaction"`
}

// SetReaction 新增或更換反應（POST /api/v1/messages/{message_id}/reactions）。
// Upsert 語義：一律回傳 200＋更新後的反應彙總（新建、更換、重送
// 相同表情皆同）。
func (h *Handler) SetReaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req setReactionBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	summary, err := h.Service.SetReaction(r.Context(), userID, r.PathValue("message_id"), req.Reaction)
	if err != nil {
		writeMessageError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toReactionSummaryResponse(summary))
}

// RemoveReaction 取消反應（DELETE /api/v1/messages/{message_id}/reactions）。
// 冪等：無生效反應時亦回傳 200＋目前彙總。
func (h *Handler) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	summary, err := h.Service.RemoveReaction(r.Context(), userID, r.PathValue("message_id"))
	if err != nil {
		writeMessageError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toReactionSummaryResponse(summary))
}

// ListReactions 列出反應彙總（GET /api/v1/messages/{message_id}/reactions）。
func (h *Handler) ListReactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	summary, err := h.Service.ListReactions(r.Context(), userID, r.PathValue("message_id"))
	if err != nil {
		writeMessageError(w, err)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toReactionSummaryResponse(summary))
}

type pinBody struct {
	MessageID string `json:"message_id"`
}

type pinResponse struct {
	PinID          string     `json:"pin_id"`
	ConversationID string     `json:"conversation_id"`
	MessageID      string     `json:"message_id"`
	PinnedBy       string     `json:"pinned_by"`
	PinnedAt       time.Time  `json:"pinned_at"`
	UnpinnedAt     *time.Time `json:"unpinned_at,omitempty"`
}

func toPinResponse(p *Pin) pinResponse {
	var unpinned *time.Time
	if p.UnpinnedAt != nil {
		u := p.UnpinnedAt.UTC()
		unpinned = &u
	}
	return pinResponse{
		PinID:          p.ID,
		ConversationID: p.ConversationID,
		MessageID:      p.MessageID,
		PinnedBy:       p.PinnedBy,
		PinnedAt:       p.PinnedAt.UTC(),
		UnpinnedAt:     unpinned,
	}
}

type pinListResponse struct {
	Pins []pinResponse `json:"pins"`
}

// PinConversationMessage 置頂訊息（POST /api/v1/conversations/{conversation_id}/pins）。
// 新建回傳 201、已置頂回傳 200＋既有列。
func (h *Handler) PinConversationMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var req pinBody
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	pin, isNew, err := h.Service.PinMessage(r.Context(), userID,
		r.PathValue("conversation_id"), req.MessageID)
	if err != nil {
		writeMessageError(w, err)
		return
	}
	status := http.StatusOK
	if isNew {
		status = http.StatusCreated
	}
	transport.WriteJSON(w, status, toPinResponse(pin))
}

// ListPins 列出對話的生效置頂訊息（GET /api/v1/conversations/{conversation_id}/pins）。
func (h *Handler) ListPins(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	pins, err := h.Service.ListPins(r.Context(), userID, r.PathValue("conversation_id"))
	if err != nil {
		writeMessageError(w, err)
		return
	}
	out := pinListResponse{Pins: []pinResponse{}}
	for _, p := range pins {
		out.Pins = append(out.Pins, toPinResponse(p))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

type unpinResponse struct {
	ConversationID string     `json:"conversation_id"`
	MessageID      string     `json:"message_id"`
	Pinned         bool       `json:"pinned"`
	UnpinnedAt     *time.Time `json:"unpinned_at,omitempty"`
}

// UnpinConversationMessage 取消置頂（DELETE /api/v1/conversations/{conversation_id}/pins/{message_id}）。
// 冪等：一律回傳 200＋pinned=false；從未置頂時 unpinned_at 為空。
func (h *Handler) UnpinConversationMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	pin, err := h.Service.UnpinMessage(r.Context(), userID,
		r.PathValue("conversation_id"), r.PathValue("message_id"))
	if err != nil {
		writeMessageError(w, err)
		return
	}
	resp := unpinResponse{
		ConversationID: r.PathValue("conversation_id"),
		MessageID:      r.PathValue("message_id"),
		Pinned:         false,
	}
	if pin != nil && pin.UnpinnedAt != nil {
		u := pin.UnpinnedAt.UTC()
		resp.UnpinnedAt = &u
	}
	transport.WriteJSON(w, http.StatusOK, resp)
}
