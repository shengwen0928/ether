// Command worker 是 ETHER Worker 的 Phase 01 骨架。
//
// 本階段只啟動 /healthz 佔位（見 internal/health），不消費任何
// 佇列、不處理任何任務。NATS JetStream 消費者與任務定義待
// Phase 05 起落地（見 internal/jobs/doc.go）。不得寫假任務處理，
// 不得以空迴圈或定時器冒充背景工作。
//
// 埠號：WORKER_PORT（Phase 01 佔位名，compose 未為 worker 設定
// 埠號與 healthcheck），未設定時使用 8082。本監聽埠現階段不被
// compose 或 Dockerfile 探測（見 internal/health 註解與
// docs/phase01/go-notes.md）；worker 是否保留 HTTP 端點待後階段定稿。
// 本機 VM 無 go 工具鏈，編譯與測試皆為「待 CI 驗證」。
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

	"ether.local/worker/internal/health"
)

// version 是服務版本佔位符。版號規則待定稿（ROADMAP.md §4.1），
// 本值不得解讀為已發布的版本號。
const version = "0.1.0-placeholder"

// serviceName 是 /healthz 回應中的服務名，與 compose 服務名一致。
const serviceName = "worker"

// buildHandler 組裝 HTTP handler：本階段只註冊 /healthz。
// TODO: NATS JetStream 消費者待 Phase 05（見 internal/jobs/doc.go）。
func buildHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health.Handler(serviceName, version))
	return mux
}

func workerPort() string {
	if v := strings.TrimSpace(os.Getenv("WORKER_PORT")); v != "" {
		return v
	}
	return "8082"
}

func main() {
	server := &http.Server{
		Addr:              ":" + workerPort(),
		Handler:           buildHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("worker: listen: %v", err)
		}
	}()
	log.Printf("worker: listening on %s (healthz placeholder only; NATS consumer pending Phase 05)", server.Addr)

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("worker: shutdown: %v", err)
	}
}
