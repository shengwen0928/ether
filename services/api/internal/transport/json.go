package transport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// MaxJSONBodyBytes 是 JSON 請求體的大小上限（1 MiB）。超過者
// 回傳 413，不得以無界讀取接受任意大小的請求體。
const MaxJSONBodyBytes = 1 << 20

// DecodeJSON 由請求體解碼 JSON，拒絕未知欄位（嚴格模式：
// 未定稿的欄位不得靜默接受，定稿前以明確錯誤為準，見
// API_STANDARDS.md §4.1 未知欄位處理待定稿）。
// 回傳的 error 由呼叫方對應 400 ETHER_VALIDATION_INVALID_REQUEST。
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, MaxJSONBodyBytes+1))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	// 拒絕尾隨資料（第二個 JSON 值或垃圾）。
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain a single JSON value")
		}
		return err
	}
	return nil
}

// FieldErrorProvider 是各領域 ValidationError 實作的介面：
// 攜帶可定位的欄位錯誤清單，供 WriteServiceError 寫出 details。
type FieldErrorProvider interface {
	error
	FieldErrors() []FieldError
}

// WriteServiceError 將領域服務錯誤對應為 HTTP 錯誤回應：
//   - 欄位驗證錯誤 → 400＋details；
//   - errors.Is(err, notFound) → 404 ETHER_COMMON_NOT_FOUND
//     （訊息與「不存在」不可區分，不得洩漏存在性）；
//   - 其餘 → 500 ETHER_COMMON_INTERNAL_ERROR（不回傳內部細節）。
func WriteServiceError(w http.ResponseWriter, err error, notFound error) {
	var ve FieldErrorProvider
	if errors.As(err, &ve) {
		WriteValidationError(w, ve.FieldErrors())
		return
	}
	if notFound != nil && errors.Is(err, notFound) {
		WriteError(w, http.StatusNotFound, "ETHER_COMMON_NOT_FOUND", "找不到資源。")
		return
	}
	WriteError(w, http.StatusInternalServerError, "ETHER_COMMON_INTERNAL_ERROR", "伺服器內部錯誤。")
}
