// Package secrets 定義 Secrets Provider 介面與兩個實作：
// 環境變數實作（本機開發用）與 OpenBao 實作 stub。
//
// 規則（SPEC-SOURCE.md §27 第 11 條）：敏感金鑰不得寫入原始碼；
// Production Secrets 不入 Git，一律以 OpenBao／環境注入提供。
// 本套件不得含任何真憑證；secret 值不得寫入日誌，不得在錯誤中回傳。
// Provider 的選擇方式（env／OpenBao 的切換）待部署定稿；本階段
// main 預設使用 EnvProvider，OpenBao stub 僅供 Phase 01 審查，
// 不得假裝已接線。
package secrets

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrNotConfigured 表示 Secrets Provider 未設定：呼叫方須明確處理，
// 不得以空字串或預設值冒充已讀到 secret。
var ErrNotConfigured = errors.New("secrets: provider not configured")

// Provider 讀取具名 secret。name 為 secret 的鍵名（例如
// "MINIO_ACCESS_KEY" 一類的環境變數名，實際鍵名以部署定稿為準）。
// 找不到或未設定時回傳 error，不得回傳空字串當作值。
type Provider interface {
	Get(ctx context.Context, name string) (string, error)
}

// EnvProvider 從行程環境變數讀取 secret（本機開發用）。
// 正式環境須改用 OpenBao Provider，不得以環境變數承載正式 secret
// 而不經部署定稿。
type EnvProvider struct{}

// Get 讀取環境變數 name。未設定或為空時回傳錯誤，錯誤中不得
// 包含 secret 值。
func (EnvProvider) Get(_ context.Context, name string) (string, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return "", fmt.Errorf("secrets: secret %q is not set in the environment", name)
	}
	return v, nil
}

// OpenBaoProvider 是 OpenBao 實作的 stub：現階段永遠不回傳 secret 值。
//
// 未設定時（OPENBAO_ADDR 未設定，變數名見 .env.example 註解）：
// Get 回傳 ErrNotConfigured，呼叫方可明確判別。
// 已設定位址時：Get 仍回傳錯誤（客戶端未實作，TODO），不得假裝
// 已連上 OpenBao 讀到 secret。正式接線（Vault／OpenBao 客戶端、
// 解封流程、Token 注入）待部署定稿，不得在本 stub 中預寫。
type OpenBaoProvider struct {
	addr  string
	token string
}

// NewOpenBaoProviderFromEnv 從環境變數建立 OpenBaoProvider stub。
// 讀取 OPENBAO_ADDR 與 OPENBAO_TOKEN（.env.example 註解中的名稱）；
// 兩者皆為注入值，不得寫入程式碼或文件。
func NewOpenBaoProviderFromEnv() *OpenBaoProvider {
	return &OpenBaoProvider{
		addr:  strings.TrimSpace(os.Getenv("OPENBAO_ADDR")),
		token: strings.TrimSpace(os.Getenv("OPENBAO_TOKEN")),
	}
}

// Get 永遠回傳錯誤，絕不回傳 secret 值。
func (p *OpenBaoProvider) Get(_ context.Context, name string) (string, error) {
	if p.addr == "" {
		return "", fmt.Errorf("secrets: openbao: %w (OPENBAO_ADDR is not set)", ErrNotConfigured)
	}
	// 位址已設定不代表客戶端已實作：明確報錯，不得假裝讀到 secret。
	// p.token 在此不被使用（未建立任何連線）；保留欄位供正式接線時使用。
	_ = name
	return "", errors.New("secrets: openbao: client not implemented (TODO: OpenBao client wiring pending deployment decision)")
}
