// Package auth 是 Realtime Gateway 的 OIDC Token 驗證（Phase 05）。
//
// 語義與 services/api 的 internal/auth 完全一致（同一規則，不得
// 兩套標準）：僅 RS256、JWKS（kid 選取＋未知 kid 強制重抓）、
// iss／exp（60 秒時鐘寬容）／sub 驗證、aud 僅在設定時強制。
// 兩 module 不得互相 import（internal 套件不得跨 module），故本
// 套件為有意的重複實作（與 internal/health 的重複前例相同）；
// 規則變更時兩處須同步，不得只改一處。
// 驗證失敗一律回傳 ErrInvalidToken，不得洩漏細節、不得記錄 Token。
package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
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
// kid 皆以此錯誤回傳，不得區分原因）。
var ErrInvalidToken = errors.New("auth: invalid token")

// Identity 是驗證通過的身分（僅 Gateway 需要的最小欄位）。
type Identity struct {
	Subject   string
	ExpiresAt time.Time
}

// Verifier 以 JWKS 驗證 RS256 JWT。
type Verifier struct {
	issuer   string
	jwksURL  string
	audience string // 空字串表示不強制 aud。
	client   *http.Client
	cacheTTL time.Duration

	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

// NewVerifier 建立 Verifier。issuer 與 jwksURL 不可為空。
func NewVerifier(issuer, jwksURL, audience string) (*Verifier, error) {
	if issuer == "" || jwksURL == "" {
		return nil, errors.New("auth: issuer 與 jwksURL 不可為空")
	}
	return &Verifier{
		issuer:   issuer,
		jwksURL:  jwksURL,
		audience: audience,
		client:   &http.Client{Timeout: 10 * time.Second},
		cacheTTL: 10 * time.Minute,
		keys:     map[string]*rsa.PublicKey{},
	}, nil
}

// DiscoverJWKSURL 由 OIDC discovery 文件取出 issuer 與 jwks_uri。
// 回傳的 issuer 須由呼叫方與設定值比對一致，不得盲信文件內容。
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
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("auth: discovery 回應狀態 %d", resp.StatusCode)
	}
	var doc struct {
		Issuer  string `json:"issuer"`
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", "", err
	}
	if doc.Issuer == "" || doc.JWKSURI == "" {
		return "", "", errors.New("auth: discovery 文件缺 issuer 或 jwks_uri")
	}
	return doc.Issuer, doc.JWKSURI, nil
}

type jwk struct {
	Kid string   `json:"kid"`
	Kty string   `json:"kty"`
	Use string   `json:"use"`
	Alg string   `json:"alg"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

func (v *Verifier) fetchKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth: JWKS 回應狀態 %d", resp.StatusCode)
	}
	var doc struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := parseJWK(k)
		if err != nil {
			continue
		}
		if k.Kid != "" {
			keys[k.Kid] = pub
		}
	}
	if len(keys) == 0 {
		return errors.New("auth: JWKS 無可用 RSA 金鑰")
	}
	v.mu.Lock()
	v.keys = keys
	v.fetched = time.Now()
	v.mu.Unlock()
	return nil
}

func parseJWK(k jwk) (*rsa.PublicKey, error) {
	if len(k.X5c) > 0 {
		der, err := base64.StdEncoding.DecodeString(k.X5c[0])
		if err != nil {
			return nil, err
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, err
		}
		pub, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("auth: 憑證非 RSA 金鑰")
		}
		return pub, nil
	}
	nb, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, err
	}
	eb, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nb)
	e := 0
	for _, b := range eb {
		e = e<<8 | int(b)
	}
	if e == 0 {
		return nil, errors.New("auth: RSA 指數為零")
	}
	return &rsa.PublicKey{N: n, E: e}, nil
}

func (v *Verifier) keyFor(ctx context.Context, kid string, forceRefresh bool) (*rsa.PublicKey, error) {
	v.mu.Lock()
	stale := time.Since(v.fetched) > v.cacheTTL
	key, ok := v.keys[kid]
	v.mu.Unlock()
	if ok && !stale && !forceRefresh {
		return key, nil
	}
	if err := v.fetchKeys(ctx); err != nil {
		return nil, err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	key, ok = v.keys[kid]
	if !ok {
		return nil, ErrInvalidToken
	}
	return key, nil
}

// Verify 驗證 Token，回傳身分。任何失敗皆為 ErrInvalidToken。
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
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Identity{}, ErrInvalidToken
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Identity{}, ErrInvalidToken
	}
	key, err := v.keyFor(ctx, header.Kid, false)
	if err != nil {
		return Identity{}, ErrInvalidToken
	}
	signingInput := parts[0] + "." + parts[1]
	digest := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); err != nil {
		// 未知 kid 時強制重抓一次（金鑰輪換），再驗失敗才拒絕。
		key, err = v.keyFor(ctx, header.Kid, true)
		if err != nil {
			return Identity{}, ErrInvalidToken
		}
		if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); err != nil {
			return Identity{}, ErrInvalidToken
		}
	}
	var claims struct {
		Issuer    string `json:"iss"`
		Subject   string `json:"sub"`
		ExpiresAt int64  `json:"exp"`
		Audience  any    `json:"aud"`
	}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return Identity{}, ErrInvalidToken
	}
	if claims.Issuer != v.issuer {
		return Identity{}, ErrInvalidToken
	}
	if claims.Subject == "" {
		return Identity{}, ErrInvalidToken
	}
	// 60 秒時鐘寬容（與 api 驗證器一致）。
	if time.Now().Unix() > claims.ExpiresAt+60 {
		return Identity{}, ErrInvalidToken
	}
	if v.audience != "" {
		aud, err := audienceList(claims.Audience)
		if err != nil {
			return Identity{}, ErrInvalidToken
		}
		found := false
		for _, a := range aud {
			if a == v.audience {
				found = true
				break
			}
		}
		if !found {
			return Identity{}, ErrInvalidToken
		}
	}
	return Identity{Subject: claims.Subject, ExpiresAt: time.Unix(claims.ExpiresAt, 0).UTC()}, nil
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
