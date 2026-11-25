// Package auth 的 HTTP handler：/api/v1/auth 前綴下的身分端點
// （API_STANDARDS.md §2）。全部端點皆須經 RequireAuth，不得以
// 未驗證請求存取。回應一律 snake_case、不得含 Token 明文或雜湊。
package auth

import (
	"errors"
	"net/http"
	"time"

	"ether.local/api/internal/transport"
	"ether.local/api/internal/user"
)

// Handler 持有 auth 端點所需的服務。CurrentUser 由接線方注入：
// 依請求解析出 ETHER 使用者 ID（首次見到時建置帳號）；解析失敗
// 時回傳 ok=false，handler 回 500（中介層已保證身分存在，
// 解析失敗屬伺服器設定問題）。
type Handler struct {
	Users       *user.Service
	Sessions    *SessionService
	CurrentUser func(r *http.Request) (string, bool)
}

type meResponse struct {
	UserID          string     `json:"user_id"`
	KeycloakSubject string     `json:"keycloak_subject"`
	Email           string     `json:"email"`
	EmailVerified   bool       `json:"email_verified"`
	Status          string     `json:"status"`
	RegisteredAt    time.Time  `json:"registered_at"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
}

// Me 回傳目前登入者的帳號主檔（GET /api/v1/auth/me）。
// 首次呼叫時以 Keycloak subject 建置帳號（just-in-time
// provisioning，冪等）；公開檔案屬 Profile 模組，不在此回傳。
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	identity, ok := IdentityFromContext(r.Context())
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return
	}
	u, err := h.Users.Provision(r.Context(), identity.Subject, identity.Email, identity.EmailVerified)
	if err != nil {
		transport.WriteServiceError(w, err, user.ErrUserNotFound)
		return
	}
	transport.WriteJSON(w, http.StatusOK, meResponse{
		UserID:          u.ID,
		KeycloakSubject: u.KeycloakSubject,
		Email:           u.Email,
		EmailVerified:   u.EmailVerified,
		Status:          string(u.Status),
		RegisteredAt:    u.RegisteredAt.UTC(),
		LastLoginAt:     utcPtr(u.LastLoginAt),
	})
}

type sessionResponse struct {
	SessionID         string    `json:"session_id"`
	DeviceID          string    `json:"device_id,omitempty"`
	KeycloakSessionID string    `json:"keycloak_session_id,omitempty"`
	ExpiresAt         time.Time `json:"expires_at"`
	LastActiveAt      time.Time `json:"last_active_at"`
	Revoked           bool      `json:"is_revoked"`
	RevokedReason     string    `json:"revoked_reason,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

func toSessionResponse(s *Session) sessionResponse {
	return sessionResponse{
		SessionID:         s.ID,
		DeviceID:          s.DeviceID,
		KeycloakSessionID: s.KeycloakSessionID,
		ExpiresAt:         s.ExpiresAt.UTC(),
		LastActiveAt:      s.LastActiveAt.UTC(),
		Revoked:           s.Revoked(),
		RevokedReason:     s.RevokedReason,
		CreatedAt:         s.CreatedAt.UTC(),
	}
}

// ListSessions 列出目前登入者的 Session（GET /api/v1/auth/sessions）。
// 雜湊值一律不回傳。
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.CurrentUser(r)
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return
	}
	sessions, err := h.Sessions.store.ListByUser(r.Context(), userID)
	if err != nil {
		transport.WriteServiceError(w, err, ErrSessionNotFound)
		return
	}
	out := make([]sessionResponse, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, toSessionResponse(s))
	}
	transport.WriteJSON(w, http.StatusOK, out)
}

// RevokeSession 撤銷單一 Session（POST /api/v1/auth/sessions/{session_id}/revoke，
// 遠端登出單一裝置）。非擁有者一律 404，不得洩漏他人 Session 存在性。
func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.CurrentUser(r)
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return
	}
	sessionID := r.PathValue("session_id")
	sess, err := h.Sessions.store.Get(r.Context(), sessionID)
	if err != nil {
		transport.WriteServiceError(w, err, ErrSessionNotFound)
		return
	}
	if sess.UserID != userID {
		transport.WriteError(w, http.StatusNotFound, "ETHER_COMMON_NOT_FOUND", "找不到資源。")
		return
	}
	revoked, err := h.Sessions.Revoke(r.Context(), sessionID, "user_revoked")
	if err != nil {
		transport.WriteServiceError(w, err, ErrSessionNotFound)
		return
	}
	transport.WriteJSON(w, http.StatusOK, toSessionResponse(revoked))
}

type logoutResponse struct {
	Revoked bool `json:"revoked"`
}

// Logout 撤銷目前 Token 對應的 ETHER Session（POST /api/v1/auth/logout）。
// Token 須含 Keycloak sid 聲明才能對應單一 Session；無 sid 時回 400，
// 不得擅自撤銷使用者的全部 Session。Keycloak 端的 Session 撤銷由
// Keycloak 登出流程承擔，本端點只處理 ETHER 映象（同步機制待定稿）。
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	identity, ok := IdentityFromContext(r.Context())
	if !ok {
		transport.WriteError(w, http.StatusInternalServerError,
			"ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
		return
	}
	if identity.SessionID == "" {
		transport.WriteError(w, http.StatusBadRequest,
			"ETHER_VALIDATION_INVALID_REQUEST", "Token 未含工作階段識別，無法單一登出。")
		return
	}
	sess, err := h.Sessions.store.GetByKeycloakSessionID(r.Context(), identity.SessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			// 冪等：無對應紀錄視為已登出，不得報錯。
			transport.WriteJSON(w, http.StatusOK, logoutResponse{Revoked: false})
			return
		}
		transport.WriteServiceError(w, err, ErrSessionNotFound)
		return
	}
	userID, ok := h.CurrentUser(r)
	if !ok || sess.UserID != userID {
		transport.WriteError(w, http.StatusNotFound, "ETHER_COMMON_NOT_FOUND", "找不到資源。")
		return
	}
	if _, err := h.Sessions.Revoke(r.Context(), sess.ID, "logout"); err != nil {
		transport.WriteServiceError(w, err, ErrSessionNotFound)
		return
	}
	transport.WriteJSON(w, http.StatusOK, logoutResponse{Revoked: true})
}

func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
