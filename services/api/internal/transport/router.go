// Package transport 隔離 HTTP 框架綁定與業務 handler。
//
// 已決議 Echo v5（≥v5.4.7），見 ADR-016；依賴尚未引入，StdRouter 維持。
// 本階段不得讓任何 handler 直接依賴框架套件。
//
// 設計：
//   - Handler 只使用 Go 標準庫型別（http.ResponseWriter、*http.Request，
//     標準 context 經 r.Context() 取得），不得 import 任何框架套件。
//   - Router 介面只定義 Handle(method, path, handler) 語義；本階段由
//     StdRouter 以 net/http ServeMux 實作。
//   - Echo Transport Adapter 未來實作同一 Router 介面，屆時替換 StdRouter，
//     不得改動 handler 簽名，不得讓 handler 直接依賴框架。
//   - 路由模式使用 Go 1.22 起 ServeMux 的 "METHOD /path" 寫法；
//     go directive 固定為 1.25（見 go.mod），待 CI 驗證。
package transport

import "net/http"

// Handler 是框架無關的 HTTP handler：只依賴標準庫型別。
type Handler func(w http.ResponseWriter, r *http.Request)

// Router 是傳輸層的唯一路由介面。
// Handle 註冊一個路由；method 為 "GET"、"POST" 等標準方法名，
// path 為絕對路徑（例如 "/healthz"；業務前綴見 API_STANDARDS.md §2，
// 本階段不得註冊業務路由）。
type Router interface {
	Handle(method, path string, h Handler)
	// Handler 回傳可交給 http.Server 的標準 handler。
	Handler() http.Handler
}

// StdRouter 以標準庫 net/http ServeMux 實作 Router。
// 這是 Echo 依賴引入前的佔位實作，選型已決議（見 ADR-016）。
type StdRouter struct {
	mux *http.ServeMux
}

// NewStdRouter 建立一個 StdRouter。
func NewStdRouter() *StdRouter {
	return &StdRouter{mux: http.NewServeMux()}
}

// Handle 註冊路由。method 與 path 拼為 ServeMux 模式 "METHOD /path"。
func (s *StdRouter) Handle(method, path string, h Handler) {
	s.mux.HandleFunc(method+" "+path, h)
}

// Handler 回傳底層 ServeMux。
func (s *StdRouter) Handler() http.Handler {
	return s.mux
}
