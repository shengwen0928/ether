// Package health 提供健康檢查端點的 handler。
//
// 契約：GET /healthz，與 infrastructure/docker/docker-compose.yml 的
// api healthcheck 一致（wget http://localhost:8080/healthz）。
// 回應只含 status、service、version 三個欄位（snake_case，見
// API_STANDARDS.md §4.1）；成功回應的包裝規則尚待定稿（同文件 §12 缺口 8），
// 本階段不預寫包裝層，不得以本回應形式宣稱為已定合約。
//
// 誠實限制：本 handler 只證明行程存活（liveness）。資料庫、Valkey、
// NATS 等依賴的連線檢查是 TODO（SYSTEM_ARCHITECTURE.md §8.3 要求
// liveness／readiness 分離，降級策略待 Phase 01 定案）；status 為 "ok"
// 不得被解讀為依賴就緒。依賴檢查落地前，不得宣稱本端點為 readiness。
package health

import (
	"encoding/json"
	"net/http"
)

// Response 是 /healthz 的回應體。version 為佔位符（見 cmd/api 的
// version 常數註解），不得解讀為已發布的版本號。
type Response struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

// Handler 回傳寫出 Response 的 handler。service 為服務名（如 "api"）。
// 回傳型別為未命名的 func 型別，以便同時指派給 transport.Handler
//（框架無關 handler 型別）與標準庫的 http.HandlerFunc，不得改為
// http.HandlerFunc 具名型別（具名型別間須顯式轉換）。
func Handler(service, version string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// TODO: 依賴連線檢查（PostgreSQL／Valkey／NATS）待 Phase 01 定案後
		// 在 readiness 端點實作；本 handler 不得捏造依賴狀態。
		_ = json.NewEncoder(w).Encode(Response{
			Status:  "ok",
			Service: service,
			Version: version,
		})
	}
}
