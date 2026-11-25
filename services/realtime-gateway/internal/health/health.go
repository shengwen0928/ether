// Package health 提供健康檢查端點的 handler。
//
// 契約：GET /healthz，與 infrastructure/docker/docker-compose.yml 的
// realtime-gateway healthcheck 一致（wget http://localhost:8081/healthz）。
// 回應只含 status、service、version 三個欄位（snake_case，見
// API_STANDARDS.md §4.1）；成功回應的包裝規則尚待定稿，不得以本
// 回應形式宣稱為已定合約。
//
// 誠實限制：本 handler 只證明行程存活（liveness）。依賴連線檢查是
// TODO（SYSTEM_ARCHITECTURE.md §8.3 要求 liveness／readiness 分離）；
// status 為 "ok" 不得被解讀為依賴就緒。
//
// 註：本套件與 ether.local/api 的 internal/health 內容相同，是三個
// 獨立 Go module 下有意的重複（internal 套件不得跨 module import）。
// 共用 health 套件的抽取方式（共用 module 或程式碼產生）待 Phase 01
// 後續決定，不得在本階段擅自新增第四個 module。
package health

import (
	"encoding/json"
	"net/http"
)

// Response 是 /healthz 的回應體。version 為佔位符，不得解讀為
// 已發布的版本號。
type Response struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

// Handler 回傳寫出 Response 的 handler。service 為服務名。
// 回傳型別為未命名的 func 型別，以便指派給標準庫的
// http.HandlerFunc，不得改為具名型別（具名型別間須顯式轉換）。
func Handler(service, version string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// TODO: 依賴連線檢查（PostgreSQL／Valkey／NATS）待 Phase 05 定案後
		// 在 readiness 端點實作；本 handler 不得捏造依賴狀態。
		_ = json.NewEncoder(w).Encode(Response{
			Status:  "ok",
			Service: service,
			Version: version,
		})
	}
}
