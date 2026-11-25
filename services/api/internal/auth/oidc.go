// Package auth 的 OIDC Token 驗證：以後端驗證 Keycloak 簽發的
// Access Token 為準（IDENTITY_AND_PERMISSIONS.md §3–§4、
// API_STANDARDS.md §5.1）。Keycloak 是身分簽發者；本套件只驗證，
// 不簽發、不儲存密碼、不自寫身分儲存。
//
// 驗證規則：
//   - 僅接受 RS256 簽名（拒絕 none 與對稱式 alg，避免演算法混淆）。
//   - 簽名金鑰取自發行者的 JWKS（/.well-known 探測到的 jwks_uri），
//     依 kid 選取；遇未知 kid 時強制重新抓取一次。
//   - 驗證 iss（須等於設定值）、exp（含 60 秒時鐘寬容）、sub 非空；
//     aud 僅在設定 Audience 時強制要求包含該值（Keycloak 的 aud
//     形態依 Client 設定而異，不得假設固定值）。
//   - 金鑰快取 10 分鐘（可設定）；抓取失敗時沿用快取，首次抓取失敗
//     即回傳錯誤，不得以無金鑰狀態放行。
//   - 錯誤不含 Token 內容，不得將 Token 寫入日誌。
package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ErrInvalidToken 表示 Token 驗證失敗（格式、簽名、iss、exp、aud、
// sub 任一項不符）。呼叫方一律對應 401 ETHER_AUTH_UNAUTHENTICATED，
// 不得以錯誤細節區分失敗原因向客戶端洩漏資訊。
var ErrInvalidToken = errors.New("auth: invalid token")

// Identity 是驗證通過後的身分：Keycloak subject 為 ETHER 業務綁定
// 的穩定識別碼（不得僅以 Email 比對，見 IDENTITY §3.1）。
type Identity struct {
	Subject       string
	Email         string
	EmailVerified bool
	SessionID     string // Keycloak sid 聲明；無此聲明時為空字串。
	ExpiresAt     time.Time
	IssuedAt      time.Time
	Audience      []string
}

// TokenVerifier 是 Token 驗證介面：真實驗證（Verifier）與測試替身
// 皆實作此介面，handler 只依賴介面。
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (Identity, error)
}

// Verifier 以 JWKS 驗證 RS256 JWT。
type Verifier struct {
	issuer   string
	jwksURL  string
	audience string // 空字串表示不強制 aud。
	client   *http.Client
	cacheTTL time.Duration
	now      func() time.Time

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

// VerifierOption 設定 Verifier 的可選行為。
type VerifierOption func(*Verifier)

// WithHTTPClient 注入自訂 HTTP 客戶端（測試用 httptest 伺服器時必要）。
func WithHTTPClient(c *http.Client) VerifierOption {
	return func(v *Verifier) { v.client = c }
}

// WithCacheTTL 設定 JWKS 快取有效期（預設 10 分鐘）。
func WithCacheTTL(d time.Duration) VerifierOption {
	return func(v *Verifier) { v.cacheTTL = d }
}

// WithClock 注入時鐘（測試用）。
func WithClock(now func() time.Time) VerifierOption {
	return func(v *Verifier) { v.now = now }
}

// NewVerifier 建立 Verifier。issuer 須為發行者位址（如
// https://keycloak.example.com/realms/ether），jwksURL 須為該
// Realm 的 JWKS 端點；兩者皆不得為空。
func NewVerifier(issuer, jwksURL, audience string, opts ...VerifierOption) (*Verifier, error) {
	if strings.TrimSpace(issuer) == "" || strings.TrimSpace(jwksURL) == "" {
		return nil, errors.New("auth: issuer 與 jwksURL 皆不得為空")
	}
	v := &Verifier{
		issuer:   strings.TrimSuffix(issuer, "/"),
		jwksURL:  jwksURL,
		audience: audience,
		client:   &http.Client{Timeout: 10 * time.Second},
		cacheTTL: 10 * time.Minute,
		now:      time.Now,
	}
	for _, o := range opts {
		o(v)
	}
	return v, nil
}

// DiscoverJWKSURL 由 OIDC discovery 文件取得 jwks_uri 與 issuer。
// discoveryURL 為發行者的 .well-known/openid-configuration 位址。
// 回傳的 issuer 須與 discovery 文件所載一致，呼叫方以此建立 Verifier。
func DiscoverJWKSURL(ctx context.Context, discoveryURL string, client *http.Client) (issuer, jwksURL string, err error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return "", "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("auth: discovery 抓取失敗: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("auth: discovery 回應狀態 %d", resp.StatusCode)
	}
	var doc struct {
		Issuer  string `json:"issuer"`
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", "", fmt.Errorf("auth: discovery 解析失敗: %w", err)
	}
	if doc.Issuer == "" || doc.JWKSURI == "" {
		return "", "", errors.New("auth: discovery 文件缺少 issuer 或 jwks_uri")
	}
	return strings.TrimSuffix(doc.Issuer, "/"), doc.JWKSURI, nil
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

func (v *Verifier) fetchKeys(ctx context.Context, force bool) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !force && v.keys != nil && v.now().Sub(v.fetchedAt) < v.cacheTTL {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("auth: JWKS 抓取失敗: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth: JWKS 回應狀態 %d", resp.StatusCode)
	}
	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("auth: JWKS 解析失敗: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "RSA" || k.Use != "" && k.Use != "sig" {
			continue
		}
		pub, err := rsaPublicKey(k.N, k.E)
		if err != nil {
			continue // 單一損壞金鑰不得拖垮整組 JWKS。
		}
		if k.Kid != "" {
			keys[k.Kid] = pub
		}
	}
	v.keys = keys
	v.fetchedAt = v.now()
	return nil
}

func rsaPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	e := new(big.Int).SetBytes(eBytes).Int64()
	if e <= 0 || e > 1<<31-1 {
		return nil, errors.New("auth: 非法的 RSA 指數")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: int(e)}, nil
}

// Verify 驗證 JWT 並回傳 Identity。任何失敗皆回傳 ErrInvalidToken
// （包裝後仍以 errors.Is 判別），不得洩漏失敗細節。
func (v *Verifier) Verify(ctx context.Context, token string) (Identity, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Identity{}, ErrInvalidToken
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Identity{}, ErrInvalidToken
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return Identity{}, ErrInvalidToken
	}
	if header.Alg != "RS256" || header.Kid == "" {
		return Identity{}, ErrInvalidToken
	}

	if err := v.fetchKeys(ctx, false); err != nil {
		return Identity{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	pub := v.keyFor(header.Kid)
	if pub == nil {
		// 未知 kid：強制重新抓取一次（金鑰輪換），仍無則拒絕。
		if err := v.fetchKeys(ctx, true); err != nil {
			return Identity{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
		}
		pub = v.keyFor(header.Kid)
		if pub == nil {
			return Identity{}, ErrInvalidToken
		}
	}

	signingInput := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Identity{}, ErrInvalidToken
	}
	sum := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig); err != nil {
		return Identity{}, ErrInvalidToken
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Identity{}, ErrInvalidToken
	}
	var claims struct {
		Subject       string  `json:"sub"`
		Issuer        string  `json:"iss"`
		Audience      any     `json:"aud"`
		ExpiresAt     float64 `json:"exp"`
		IssuedAt      float64 `json:"iat"`
		Email         string  `json:"email"`
		EmailVerified bool    `json:"email_verified"`
		SessionID     string  `json:"sid"`
	}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return Identity{}, ErrInvalidToken
	}
	if claims.Subject == "" || claims.Issuer != v.issuer {
		return Identity{}, ErrInvalidToken
	}
	now := v.now()
	// 60 秒時鐘寬容：發行者與驗證者時鐘差異不得造成合法 Token 被拒。
	if claims.ExpiresAt == 0 || now.After(time.Unix(int64(claims.ExpiresAt), 0).Add(60*time.Second)) {
		return Identity{}, ErrInvalidToken
	}
	if claims.IssuedAt != 0 && time.Unix(int64(claims.IssuedAt), 0).After(now.Add(60*time.Second)) {
		return Identity{}, ErrInvalidToken
	}
	aud, err := audienceList(claims.Audience)
	if err != nil {
		return Identity{}, ErrInvalidToken
	}
	if v.audience != "" && !contains(aud, v.audience) {
		return Identity{}, ErrInvalidToken
	}
	return Identity{
		Subject:       claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		SessionID:     claims.SessionID,
		ExpiresAt:     time.Unix(int64(claims.ExpiresAt), 0),
		IssuedAt:      time.Unix(int64(claims.IssuedAt), 0),
		Audience:      aud,
	}, nil
}

func (v *Verifier) keyFor(kid string) *rsa.PublicKey {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.keys == nil {
		return nil
	}
	return v.keys[kid]
}

func audienceList(aud any) ([]string, error) {
	switch t := aud.(type) {
	case nil:
		return nil, nil
	case string:
		return []string{t}, nil
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			s, ok := e.(string)
			if !ok {
				return nil, errors.New("auth: aud 元素非字串")
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, errors.New("auth: aud 形態不明")
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
