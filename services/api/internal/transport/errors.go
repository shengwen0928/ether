package transport

import (
	"encoding/json"
	"net/http"
)

// FieldError 是 ERROR_CODES.md §5.1 details 陣列的一項：可定位的
// 欄位驗證失敗。Issue 的值域為本階段實作詞彙（REQUIRED、
// TOO_SHORT、TOO_LONG、INVALID_FORMAT、INVALID_VALUE、DUPLICATE），
// 待 API 錯誤碼定稿時覆核，不得以本詞彙冒充已定稿的值域。
type FieldError struct {
	Field   string `json:"field"`
	Issue   string `json:"issue"`
	Message string `json:"message"`
}

// ErrorDetailsBody 是含 details 的 error 物件（ERROR_CODES.md §5.1
// 草案格式）。僅在可定位欄位的驗證失敗時使用；其餘錯誤一律用
// envelope.go 的 WriteError，不得混用。
type ErrorDetailsBody struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	RequestID string       `json:"request_id"`
	Details   []FieldError `json:"details"`
}

// ErrorDetailsResponse 是含 details 的錯誤回應頂層結構。
type ErrorDetailsResponse struct {
	Error ErrorDetailsBody `json:"error"`
}

// WriteValidationError 依 ERROR_CODES.md §5.1 草案格式寫出含 details
// 的欄位驗證錯誤（HTTP 400、ETHER_VALIDATION_INVALID_FIELD）。
// 400／422 二選一尚待定稿（ERROR_CODES.md §8 缺口 2）；本階段
// 選 400，全 API 一致，定稿後如改 422 只改此一處。
// 不得在 details 中放入未授權的欄位資訊或敏感資料。
func WriteValidationError(w http.ResponseWriter, fields []FieldError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(ErrorDetailsResponse{
		Error: ErrorDetailsBody{
			Code:      "ETHER_VALIDATION_INVALID_FIELD",
			Message:   "欄位驗證失敗。",
			RequestID: NewRequestID(),
			Details:   fields,
		},
	})
}

// WriteJSON 寫出成功回應：資源物件本身直接編碼（API_STANDARDS.md
// §4.2：不得多包一層與資源名相同的鍵；包裝規則待定稿）。
// status 由呼叫方依語義傳入（建立 201、讀取／更新 200）。
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
