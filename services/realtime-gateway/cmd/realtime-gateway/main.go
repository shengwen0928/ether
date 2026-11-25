// Command realtime-gateway 是 ETHER Realtime Gateway（Phase 05）。
//
// 提供 GET /healthz 與 WebSocket 端點 /ws/v1（WEBSOCKET_PROTOCOL.md
// §2 稱端點路徑待定稿，本實作沿用 Phase 01 佔位路徑）。
//
// 認證：OIDC Access Token（查詢參數 access_token），驗證規則與
// api 服務一致（RS256、JWKS、iss／exp／aud），見 internal/auth。
// 驗證器建不出來（設定缺失或 discovery 失敗）時，WS 端點一律回
// 503、不得建立未認證連線；/healthz 不受影響。此為刻意選擇：
// Gateway 無其他業務路由，fail-closed 的 WS 不應拖垮健康檢查。
//
// 扇出：行程內 MemoryBroker（Phase 05）。NATS JetStream 轉接器
// 僅佔位（ErrNotConfigured），不得新增 NATS 客戶端依賴。
//
// 明確未實作（見 internal/hub）：WebSocket 送出訊息、Heartbeat、
// Rate Limit／Backpressure、裝置撤銷即時斷線。訂閱成員身份驗證
// 已以 MembershipChecker 介面實作，但 api 側的真實成員查詢尚未
// 整合：本服務以 nil checker 啟動，訂閱與 typing 一律拒絕
// （fail-closed，啟動日誌明示），不得假裝已驗證。訊息推播事件
// （message.received 等）須由後續中繼接線（api 寫入後經 Broker
// 發佈），本服務目前只負責訂閱與轉發。
//
// 埠號：GATEWAY_PORT，未設定時使用 8081，與
// docker-compose.yml 的 realtime-gateway healthcheck 一致。
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ether.local/realtime-gateway/internal/auth"
	"ether.local/realtime-gateway/internal/broker"
	"ether.local/realtime-gateway/internal/health"
	"ether.local/realtime-gateway/internal/hub"
)

// version 是服務版本佔位符。版號規則待定稿（ROADMAP.md §4.1），
// 本值不得解讀為已發布的版本號。
const version = "0.1.0-placeholder"

// serviceName 是 /healthz 回應中的服務名，與 compose 服務名一致。
const serviceName = "realtime-gateway"

// wsPath 是 WebSocket 連線端點。WEBSOCKET_PROTOCOL.md §2 稱端點
// 路徑待定稿，本路徑沿用 Phase 01 佔位值，不得解讀為已定合約。
const wsPath = "/ws/v1"

// buildHandler 組裝 HTTP handler：/healthz 與 /ws/v1（Hub）。
func buildHandler(h *hub.Hub) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health.Handler(serviceName, version))
	mux.HandleFunc(wsPath, h.Handle)
	return mux
}

func gatewayPort() string {
	if v := strings.TrimSpace(os.Getenv("GATEWAY_PORT")); v != "" {
		return v
	}
	return "8081"
}

// buildVerifier 由 OIDC discovery 建立 Token 驗證器（與 api 服務
// 同一流程）：KEYCLOAK_URL＋KEYCLOAK_REALM → discovery → jwks_uri。
// 失敗時回傳錯誤，呼叫方須以 nil 驗證器啟動 Hub（WS fail-closed）。
func buildVerifier(ctx context.Context) (hub.Verifier, error) {
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("KEYCLOAK_URL")), "/")
	realm := strings.TrimSpace(os.Getenv("KEYCLOAK_REALM"))
	if base == "" || realm == "" {
		return nil, errors.New("realtime-gateway: 未設定 KEYCLOAK_URL／KEYCLOAK_REALM")
	}
	issuer := base + "/realms/" + realm
	discoveryURL := issuer + "/.well-known/openid-configuration"
	discoveredIssuer, jwksURL, err := auth.DiscoverJWKSURL(ctx, discoveryURL, nil)
	if err != nil {
		return nil, err
	}
	if discoveredIssuer != issuer {
		return nil, errors.New("realtime-gateway: discovery issuer 與設定不一致")
	}
	// Audience：KEYCLOAK_AUDIENCE 有設定時強制，與 api 服務一致。
	return auth.NewVerifier(issuer, jwksURL, strings.TrimSpace(os.Getenv("KEYCLOAK_AUDIENCE")))
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	discoveryCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	verifier, err := buildVerifier(discoveryCtx)
	cancel()
	if err != nil {
		// 驗證器建不出來時 WS fail-closed（503），不得建立未認證
		// 連線；服務仍啟動以提供 /healthz。
		log.Printf("realtime-gateway: 建立身分驗證器失敗，WebSocket 端點將回 503: %v", err)
		verifier = nil
	}

	b := broker.NewMemoryBroker()
	// 成員驗證未整合（api 側查詢待後續里程碑）：nil checker，
	// 訂閱與 typing 一律拒絕（fail-closed）。真實 checker 落地前
	// 不得以放行假裝已驗證。
	log.Printf("realtime-gateway: 成員驗證未設定，訂閱與 typing 一律拒絕（fail-closed，待 api 整合）")
	h := hub.NewHub(b, verifier, nil, nil)
	defer func() {
		h.Close()
		_ = b.Close()
	}()

	server := &http.Server{
		Addr:              ":" + gatewayPort(),
		Handler:           buildHandler(h),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("realtime-gateway: listen: %v", err)
		}
	}()
	log.Printf("realtime-gateway: listening on %s", server.Addr)

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.Close()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("realtime-gateway: shutdown: %v", err)
	}
}
