package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func mustRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return key
}

func jwkFor(pub *rsa.PublicKey, kid string) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"kid": kid,
		"use": "sig",
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}

func jwksServer(t *testing.T, keys []map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": keys})
	}))
}

func signToken(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	header, _ := json.Marshal(map[string]any{"alg": "RS256", "kid": kid, "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	enc := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	sum := sha256.Sum256([]byte(enc))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return enc + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func baseClaims(issuer string) map[string]any {
	now := time.Now()
	return map[string]any{
		"sub":            "subject-1",
		"iss":            issuer,
		"aud":            "ether-web",
		"exp":            now.Add(time.Hour).Unix(),
		"iat":            now.Unix(),
		"email":          "user@example.invalid",
		"email_verified": true,
		"sid":            "kc-session-1",
	}
}

func TestVerify_Valid(t *testing.T) {
	key := mustRSAKey(t)
	srv := jwksServer(t, []map[string]any{jwkFor(&key.PublicKey, "key-1")})
	defer srv.Close()
	issuer := "https://id.example.invalid/realms/ether"
	v, err := NewVerifier(issuer, srv.URL, "ether-web")
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	token := signToken(t, key, "key-1", baseClaims(issuer))
	id, err := v.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if id.Subject != "subject-1" || id.Email != "user@example.invalid" || !id.EmailVerified || id.SessionID != "kc-session-1" {
		t.Fatalf("identity = %+v", id)
	}
}

func TestVerify_AudienceModes(t *testing.T) {
	key := mustRSAKey(t)
	srv := jwksServer(t, []map[string]any{jwkFor(&key.PublicKey, "key-1")})
	defer srv.Close()
	issuer := "https://id.example.invalid/realms/ether"

	// 未設定 audience：任何 aud 皆接受（M1 預設行為，待 Client 定稿）。
	unenforced, err := NewVerifier(issuer, srv.URL, "")
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	c := baseClaims(issuer)
	c["aud"] = "anything-else"
	if _, err := unenforced.Verify(context.Background(), signToken(t, key, "key-1", c)); err != nil {
		t.Fatalf("unenforced audience rejected: %v", err)
	}

	// 設定 audience：aud 為陣列且包含設定值時接受、不含時拒絕
	//（Keycloak 的 aud 可能是字串或陣列，兩種形態皆須處理）。
	enforced, err := NewVerifier(issuer, srv.URL, "ether-web")
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	c = baseClaims(issuer)
	c["aud"] = []string{"master-realm", "ether-web"}
	if _, err := enforced.Verify(context.Background(), signToken(t, key, "key-1", c)); err != nil {
		t.Fatalf("array aud containing value rejected: %v", err)
	}
	c = baseClaims(issuer)
	c["aud"] = []string{"master-realm", "someone-else"}
	if _, err := enforced.Verify(context.Background(), signToken(t, key, "key-1", c)); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("array aud without value err = %v, want ErrInvalidToken", err)
	}
}

func TestVerify_Rejects(t *testing.T) {
	key := mustRSAKey(t)
	other := mustRSAKey(t)
	srv := jwksServer(t, []map[string]any{jwkFor(&key.PublicKey, "key-1")})
	defer srv.Close()
	issuer := "https://id.example.invalid/realms/ether"
	v, _ := NewVerifier(issuer, srv.URL, "ether-web")
	now := time.Now()

	cases := map[string]string{
		"malformed": "not-a-jwt",
		"wrong key": signToken(t, other, "key-1", baseClaims(issuer)),
		"expired": func() string {
			c := baseClaims(issuer)
			c["exp"] = now.Add(-2 * time.Hour).Unix()
			return signToken(t, key, "key-1", c)
		}(),
		"wrong issuer": func() string {
			c := baseClaims("https://id.example.invalid/realms/other")
			return signToken(t, key, "key-1", c)
		}(),
		"empty sub": func() string {
			c := baseClaims(issuer)
			c["sub"] = ""
			return signToken(t, key, "key-1", c)
		}(),
		"wrong audience": func() string {
			c := baseClaims(issuer)
			c["aud"] = "someone-else"
			return signToken(t, key, "key-1", c)
		}(),
		"unknown kid": signToken(t, key, "no-such-kid", baseClaims(issuer)),
	}
	for name, token := range cases {
		if _, err := v.Verify(context.Background(), token); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%s: err = %v, want ErrInvalidToken", name, err)
		}
	}
	// alg 混淆：標頭 alg 改為 HS256（簽名不符且 alg 不符）必須拒絕。
	header, _ := json.Marshal(map[string]any{"alg": "HS256", "kid": "key-1", "typ": "JWT"})
	payload, _ := json.Marshal(baseClaims(issuer))
	fake := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload) + ".AAAA"
	if _, err := v.Verify(context.Background(), fake); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("HS256 header: err = %v, want ErrInvalidToken", err)
	}
}

func TestVerify_KeyRotationRefetch(t *testing.T) {
	key1 := mustRSAKey(t)
	key2 := mustRSAKey(t)
	keys := []map[string]any{jwkFor(&key1.PublicKey, "key-1")}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": keys})
	}))
	defer srv.Close()
	issuer := "https://id.example.invalid/realms/ether"
	v, _ := NewVerifier(issuer, srv.URL, "")
	// 先以 key1 驗證成功（填入快取）。
	if _, err := v.Verify(context.Background(), signToken(t, key1, "key-1", baseClaims(issuer))); err != nil {
		t.Fatalf("first verify: %v", err)
	}
	// 輪換：JWKS 改為 key2，Token 以 key2 簽署、新 kid。
	keys = []map[string]any{jwkFor(&key2.PublicKey, "key-2")}
	if _, err := v.Verify(context.Background(), signToken(t, key2, "key-2", baseClaims(issuer))); err != nil {
		t.Fatalf("rotated key verify: %v", err)
	}
}

func TestVerify_JWKSUnreachable(t *testing.T) {
	v, _ := NewVerifier("https://id.example.invalid/realms/ether", "http://127.0.0.1:1/jwks", "")
	_, err := v.Verify(context.Background(), "a.b.c")
	if err == nil {
		t.Fatal("want error for unreachable JWKS")
	}
	if _, err := v.Verify(context.Background(), "a.b.c"); err == nil {
		t.Fatal("want error on repeated unreachable JWKS")
	}
	_ = fmt.Sprint()
}

func TestDiscoverJWKSURL(t *testing.T) {
	issuer := "https://id.example.invalid/realms/ether"
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":   issuer,
			"jwks_uri": srv.URL + "/certs",
		})
	}))
	defer srv.Close()
	gotIssuer, gotJWKS, err := DiscoverJWKSURL(context.Background(), srv.URL, nil)
	if err != nil {
		t.Fatalf("DiscoverJWKSURL: %v", err)
	}
	if gotIssuer != issuer || gotJWKS != srv.URL+"/certs" {
		t.Fatalf("got issuer=%q jwks=%q", gotIssuer, gotJWKS)
	}
}
