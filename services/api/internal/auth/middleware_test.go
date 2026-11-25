package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ether.local/api/internal/transport"
)

type fakeVerifier struct {
	identity Identity
	err      error
}

func (f fakeVerifier) Verify(context.Context, string) (Identity, error) {
	return f.identity, f.err
}

func TestRequireAuth(t *testing.T) {
	identity := Identity{Subject: "subject-1", Email: "a@example.invalid"}
	next := func(w http.ResponseWriter, r *http.Request) {
		got, ok := IdentityFromContext(r.Context())
		if !ok || got.Subject != "subject-1" {
			t.Errorf("identity in context = %+v, %v", got, ok)
			return
		}
		w.WriteHeader(http.StatusOK)
	}

	t.Run("valid token passes identity", func(t *testing.T) {
		h := RequireAuth(fakeVerifier{identity: identity}, next)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer token-value")
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})
	t.Run("missing header is 401", func(t *testing.T) {
		h := RequireAuth(fakeVerifier{identity: identity}, next)
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
	t.Run("non-bearer scheme is 401", func(t *testing.T) {
		h := RequireAuth(fakeVerifier{identity: identity}, next)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Basic abc")
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
	t.Run("invalid token is 401 without details", func(t *testing.T) {
		h := RequireAuth(fakeVerifier{err: errors.New("bad")}, next)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer bad")
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
		if got := rec.Body.String(); got == "" {
			t.Fatal("empty error body")
		}
	})
	t.Run("nil verifier is 503 fail-closed", func(t *testing.T) {
		h := RequireAuth(nil, next)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer token-value")
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
	})
}

func TestIdentityFromContext_WithoutMiddleware(t *testing.T) {
	if _, ok := IdentityFromContext(context.Background()); ok {
		t.Fatal("want ok=false without middleware")
	}
	var _ transport.Handler = RequireAuth(nil, nil)
}
