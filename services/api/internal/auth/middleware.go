// Package auth 的認證中介層：驗證 Authorization: Bearer Token，
// 將 Identity 寫入請求 context。未認證一律 401
// ETHER_AUTH_UNAUTHENTICATED（ERROR_CODES.md §3），不得回傳部分
// 資料、不得以預設身分放行、不得記錄 Token。
package auth

import (
	"context"
	"net/http"
	"strings"

	"ether.local/api/internal/transport"
)

type identityKey struct{}

// RequireAuth 包裝 handler：先驗證 Bearer Token，通過後才呼叫 next。
// verifier 為 nil（依賴未設定）時一律回 503 ETHER_COMMON_UNAVAILABLE，
// 不得在無驗證器的狀態下放行任何請求。
func RequireAuth(verifier TokenVerifier, next transport.Handler) transport.Handler {
	return func(w http.ResponseWriter, r *http.Request) {
		if verifier == nil {
			transport.WriteError(w, http.StatusServiceUnavailable,
				"ETHER_COMMON_UNAVAILABLE", "身分驗證服務尚未設定。")
			return
		}
		token, ok := bearerToken(r)
		if !ok {
			transport.WriteError(w, http.StatusUnauthorized,
				"ETHER_AUTH_UNAUTHENTICATED", "未認證。")
			return
		}
		identity, err := verifier.Verify(r.Context(), token)
		if err != nil {
			transport.WriteError(w, http.StatusUnauthorized,
				"ETHER_AUTH_UNAUTHENTICATED", "未認證。")
			return
		}
		ctx := context.WithValue(r.Context(), identityKey{}, identity)
		next(w, r.WithContext(ctx))
	}
}

// IdentityFromContext 取出中介層寫入的 Identity。未經 RequireAuth
// 的請求回傳 ok=false，呼叫方不得以零值身分繼續。
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(Identity)
	return identity, ok
}

// bearerToken 解析 Authorization: Bearer 標頭。Token 不得以查詢
// 參數、Cookie 或請求體傳遞（API_STANDARDS.md §5.1）。
func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", false
	}
	// 標頭須恰為 "Bearer <token>" 兩段；大小寫依 RFC 6750 不敏感處
	// 理 scheme，token 本身不得含空白。
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" || strings.Contains(token, " ") {
		return "", false
	}
	return token, true
}
