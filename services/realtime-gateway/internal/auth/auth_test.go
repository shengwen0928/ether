package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// testKey 產生 RSA 金鑰與對應的 JWKS／簽發函式（僅測試用）。
type testKey struct {
	priv *rsa.PrivateKey
	kid  string
}

func newTestKey(t *testing.T, kid string) *testKey {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return &testKey{priv: priv, kid: kid}
}

func (k *testKey) jwksJSON() []byte {
	n := base64.RawURLEncoding.EncodeToString(k.priv.PublicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.priv.PublicKey.E)).Bytes())
	doc, _ := json.Marshal(map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA", "kid": k.kid, "use": "sig", "alg": "RS256", "n": n, "e": e,
		}},
	})
	return doc
}

func (k *testKey) sign(issuer, subject string, exp int64, aud any) string {
	header, _ := json.Marshal(map[string]any{"alg": "RS256", "kid": k.kid, "typ": "JWT"})
	claims := map[string]any{"iss": issuer, "sub": subject, "exp": exp, "iat": time.Now().Unix()}
	if aud != nil {
		claims["aud"] = aud
	}
	payload, _ := json.Marshal(claims)
	enc := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	input := enc(header) + "." + enc(payload)
	digest := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, k.priv, crypto.SHA256, digest[:])
	if err != nil {
		panic(err)
	}
	return input + "." + enc(sig)
}

func TestVerify_ValidAndWrongIssuer(t *testing.T) {
	key := newTestKey(t, "key-1")
	issuer := "https://keycloak.example.invalid/realms/ether"
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(key.jwksJSON())
	}))
	defer jwks.Close()

	v, err := NewVerifier(issuer, jwks.URL, "")
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	ctx := context.Background()
	token := key.sign(issuer, "user-1", time.Now().Add(time.Hour).Unix(), nil)
	id, err := v.Verify(ctx, token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if id.Subject != "user-1" {
		t.Fatalf("subject = %q", id.Subject)
	}
	bad := key.sign("https://other.invalid/", "user-1", time.Now().Add(time.Hour).Unix(), nil)
	if _, err := v.Verify(ctx, bad); err != ErrInvalidToken {
		t.Fatalf("錯誤 issuer err = %v, want ErrInvalidToken", err)
	}
	expired := key.sign(issuer, "user-1", time.Now().Add(-2*time.Hour).Unix(), nil)
	if _, err := v.Verify(ctx, expired); err != ErrInvalidToken {
		t.Fatalf("過期 Token err = %v, want ErrInvalidToken", err)
	}
}

func TestVerify_AudienceEnforced(t *testing.T) {
	key := newTestKey(t, "key-1")
	issuer := "https://keycloak.example.invalid/realms/ether"
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(key.jwksJSON())
	}))
	defer jwks.Close()

	v, err := NewVerifier(issuer, jwks.URL, "ether-api")
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	ctx := context.Background()
	ok := key.sign(issuer, "user-1", time.Now().Add(time.Hour).Unix(), []string{"ether-api", "other"})
	if _, err := v.Verify(ctx, ok); err != nil {
		t.Fatalf("aud 含設定值應通過：%v", err)
	}
	missing := key.sign(issuer, "user-1", time.Now().Add(time.Hour).Unix(), "someone-else")
	if _, err := v.Verify(ctx, missing); err != ErrInvalidToken {
		t.Fatalf("aud 不含設定值 err = %v, want ErrInvalidToken", err)
	}
}

func TestDiscoverJWKSURL(t *testing.T) {
	issuer := "https://keycloak.example.invalid/realms/ether"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":   issuer,
			"jwks_uri": issuer + "/protocol/openid-connect/certs",
		})
	}))
	defer srv.Close()
	gotIssuer, gotURL, err := DiscoverJWKSURL(context.Background(), srv.URL, nil)
	if err != nil {
		t.Fatalf("DiscoverJWKSURL: %v", err)
	}
	if gotIssuer != issuer || gotURL != issuer+"/protocol/openid-connect/certs" {
		t.Fatalf("discovery = %q, %q", gotIssuer, gotURL)
	}
}
