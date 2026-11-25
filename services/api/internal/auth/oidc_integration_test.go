//go:build integration

// Keycloak 實機整合測試：需隔離 Keycloak 實例。
//
// 環境變數：
//   - KEYCLOAK_TEST_ISSUER：Keycloak Realm 的 issuer
//     （例如 http://localhost:8080/realms/ether）；未設定時跳過，
//     標示「環境待驗證」。已設定但連不上時明確失敗。
//   - KEYCLOAK_TEST_TOKEN：以真實使用者登入取得的 Access Token
//     （僅供本測試手動取得，不得寫入檔案或提交）；未設定時只
//     執行 discovery／JWKS 檢查，Token 驗證部分跳過。
//   - KEYCLOAK_TEST_AUDIENCE：Token 驗證時強制 的 aud 值；
//     未設定時不強制。
//
// 本檔案的測試通過只證明「與該 Keycloak 實例的整合」，Mock
// 測試（oidc_test.go）通過只證明「驗證邏輯正確」，兩者不得混淆。
package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func liveIssuer(t *testing.T) string {
	t.Helper()
	issuer := strings.TrimSpace(os.Getenv("KEYCLOAK_TEST_ISSUER"))
	if issuer == "" {
		t.Skip("環境待驗證：未設定 KEYCLOAK_TEST_ISSUER（需隔離 Keycloak 實例，不得連接正式 Keycloak）")
	}
	return strings.TrimSuffix(issuer, "/")
}

func TestLiveKeycloak_Discovery(t *testing.T) {
	issuer := liveIssuer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	discovered, jwksURL, err := DiscoverJWKSURL(ctx, issuer+"/.well-known/openid-configuration", nil)
	if err != nil {
		t.Fatalf("discovery 失敗: %v", err)
	}
	if discovered != issuer {
		t.Fatalf("discovery issuer = %q, want %q", discovered, issuer)
	}
	if jwksURL == "" {
		t.Fatal("jwks_uri 為空")
	}

	// JWKS 須含至少一把 RSA 簽名金鑰。
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		t.Fatalf("JWKS 請求建立失敗: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("JWKS 抓取失敗: %v", err)
	}
	defer resp.Body.Close()
	var jwks struct {
		Keys []struct {
			Kty string `json:"kty"`
			Use string `json:"use"`
			Alg string `json:"alg"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		t.Fatalf("JWKS 解析失敗: %v", err)
	}
	rsaKeys := 0
	for _, k := range jwks.Keys {
		if k.Kty == "RSA" && k.Use == "sig" {
			rsaKeys++
		}
	}
	if rsaKeys == 0 {
		t.Fatalf("JWKS 無 RSA 簽名金鑰: %+v", jwks.Keys)
	}
}

func TestLiveKeycloak_VerifyToken(t *testing.T) {
	issuer := liveIssuer(t)
	token := strings.TrimSpace(os.Getenv("KEYCLOAK_TEST_TOKEN"))
	if token == "" {
		t.Skip("環境待驗證：未設定 KEYCLOAK_TEST_TOKEN（需以真實使用者登入取得，不得寫入檔案）")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	discovered, jwksURL, err := DiscoverJWKSURL(ctx, issuer+"/.well-known/openid-configuration", nil)
	if err != nil {
		t.Fatalf("discovery 失敗: %v", err)
	}
	verifier, err := NewVerifier(discovered, jwksURL, strings.TrimSpace(os.Getenv("KEYCLOAK_TEST_AUDIENCE")))
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	id, err := verifier.Verify(ctx, token)
	if err != nil {
		t.Fatalf("Token 驗證失敗: %v", err)
	}
	if id.Subject == "" {
		t.Fatalf("identity subject 為空: %+v", id)
	}
}
