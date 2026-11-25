package transport

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

// ErrorBody 是 ERROR_CODES.md §5.1 草案錯誤回應的 error 物件。
// 格式為草案、待定稿確認；request_id 的格式待定稿，本階段以
// 密碼學亂數 hex 字串佔位，不得以客戶端產生的識別碼代替。
type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// ErrorResponse 是錯誤回應的頂層結構（草案，待定稿確認）。
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// NewRequestID 產生一個伺服器端 request_id 佔位值。
// 亂數失敗時回傳 "unavailable"，不得回傳空字串冒充識別碼。
func NewRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unavailable"
	}
	return hex.EncodeToString(b[:])
}

// WriteError 依 ERROR_CODES.md §5.1 的草案格式寫出錯誤回應。
// 不得在錯誤回應中附加成功回應的資料欄位，不得回傳內部細節。
func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorBody{
			Code:      code,
			Message:   message,
			RequestID: NewRequestID(),
		},
	})
}
