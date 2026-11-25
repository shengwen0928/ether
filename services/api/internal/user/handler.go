// Package user 的 HTTP handler：/api/v1/users 前綴下的帳號設定端點
// （API_STANDARDS.md §2）。全部端點皆須經 RequireAuth。
package user

import (
	"net/http"
	"time"

	"ether.local/api/internal/transport"
)

// Handler 持有 users 端點所需的服務。
type Handler struct {
	Service     *Service
	CurrentUser func(r *http.Request) (string, bool)
}

type settingsResponse struct {
	UserID         string    `json:"user_id"`
	Language       string    `json:"language"`
	Theme          string    `json:"theme"`
	FontSize       string    `json:"font_size"`
	ChatBackground string    `json:"chat_background"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func toSettingsResponse(s *Settings) settingsResponse {
	return settingsResponse{
		UserID:         s.UserID,
		Language:       s.Language,
		Theme:          s.Theme,
		FontSize:       s.FontSize,
		ChatBackground: s.ChatBackground,
		UpdatedAt:      s.UpdatedAt.UTC(),
	}
}

// GetSettings 回傳目前登入者的帳號設定（GET /api/v1/users/me/settings）。
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.CurrentUser(r)
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return
	}
	s, err := h.Service.GetSettings(r.Context(), userID)
	if err != nil {
		transport.WriteServiceError(w, err, ErrUserNotFound)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toSettingsResponse(s))
}

type settingsRequest struct {
	Language       *string `json:"language"`
	Theme          *string `json:"theme"`
	FontSize       *string `json:"font_size"`
	ChatBackground *string `json:"chat_background"`
}

// UpdateSettings 部分更新帳號設定（PATCH /api/v1/users/me/settings）。
// 未提供欄位不更動；未知欄位拒絕（嚴格模式）。
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.CurrentUser(r)
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return
	}
	var req settingsRequest
	if err := transport.DecodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "請求格式錯誤。")
		return
	}
	updated, err := h.Service.UpdateSettings(r.Context(), userID, SettingsPatch{
		Language:       req.Language,
		Theme:          req.Theme,
		FontSize:       req.FontSize,
		ChatBackground: req.ChatBackground,
	})
	if err != nil {
		transport.WriteServiceError(w, err, ErrUserNotFound)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toSettingsResponse(updated))
}
