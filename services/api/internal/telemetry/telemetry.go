// Package telemetry 是可觀測性接入點的註解型 stub。
//
// 本階段只定義介面與環境變數名，實作全部為 TODO，不得 import
// 任何 OpenTelemetry／Loki／GlitchTip SDK。接線方式與保留策略見
// SYSTEM_ARCHITECTURE.md §8 與 ADR-012（自架 Prometheus＋Grafana＋
// Loki＋GlitchTip＋OpenTelemetry，覆蓋 API、WebSocket、資料庫、
// Worker、Media、AI）。
//
// 環境變數名（僅名稱，尚未接線）：
//   - OTEL_SERVICE_NAME、OTEL_EXPORTER_OTLP_ENDPOINT：OpenTelemetry
//     標準變數名（服務名與 OTLP 匯出端點）。
//   - LOKI_ENDPOINT：Loki 寫入端點（Phase 01 佔位名，compose 未為
//     api 服務設定此變數，名稱待部署定稿確認，不得假裝已設定）。
//   - SENTRY_DSN：GlitchTip 的 Sentry 相容 DSN（GlitchTip 為自架
//     Sentry 相容錯誤追蹤；DSN 一律以環境注入，不得寫入程式碼）。
//
// 日誌禁令（SYSTEM_ARCHITECTURE.md §8.1）：不得輸出 Token、密碼、
// 金鑰、E2EE 明文、訊息明文。NoopLogger 丟棄一切，不得被當作
// 已接入監控的證據。
package telemetry

import (
	"context"
	"os"
	"strings"
)

// Config 是可觀測性的設定（僅名稱與值，全部可為空＝未啟用）。
type Config struct {
	ServiceName  string
	OTLPEndpoint string
	LokiEndpoint string
	SentryDSN    string
}

// LoadConfig 從環境變數讀取 Config。不報錯：全部欄位皆可為空，
// 空值表示該後端未啟用，不得以空值冒充已接線。
func LoadConfig() Config {
	return Config{
		ServiceName:  strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")),
		OTLPEndpoint: strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		LokiEndpoint: strings.TrimSpace(os.Getenv("LOKI_ENDPOINT")),
		SentryDSN:    strings.TrimSpace(os.Getenv("SENTRY_DSN")),
	}
}

// Field 是一筆結構化日誌欄位。值不得含敏感資料（見套件註解）。
type Field struct {
	Key   string
	Value string
}

// Logger 是結構化日誌的最小介面。實作接線是 TODO。
type Logger interface {
	Info(ctx context.Context, msg string, fields ...Field)
}

// NoopLogger 丟棄所有日誌。本階段唯一可用的 Logger 實作；
// 使用它不代表日誌已接入 Loki，不得對外宣稱監控已就緒。
type NoopLogger struct{}

// Info 丟棄訊息。
func (NoopLogger) Info(_ context.Context, _ string, _ ...Field) {}

// NewLogger 回傳 NoopLogger。TODO：OTLP／Loki／GlitchTip SDK 接線
// 落地後，在此依 Config 選擇實作，不得在呼叫端 import SDK。
func NewLogger(_ Config) Logger {
	return NoopLogger{}
}
