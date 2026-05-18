package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-konkol/triage/internal/auth"
)

const testSecret = "test-secret"

func TestMiddleware_MissingAuthorizationHeader(t *testing.T) {
	handler := auth.Middleware(testSecret)(okHandler())
	rr := serve(handler, newReq())
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMiddleware_WrongScheme(t *testing.T) {
	handler := auth.Middleware(testSecret)(okHandler())
	req := newReq()
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := serve(handler, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	handler := auth.Middleware(testSecret)(okHandler())
	req := newReq()
	req.Header.Set("Authorization", "Bearer bad-token-here")
	rr := serve(handler, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMiddleware_ValidToken_AllowsRequest(t *testing.T) {
	token, _ := auth.Generate("uid1", "alice", "admin", testSecret)
	handler := auth.Middleware(testSecret)(okHandler())
	req := newReq()
	req.Header.Set("Authorization", "Bearer "+token)
	rr := serve(handler, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestMiddleware_ValidToken_StoresClaimsInContext(t *testing.T) {
	token, _ := auth.Generate("uid1", "alice", "user", testSecret)

	var got *auth.Claims
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = auth.GetClaims(r)
		w.WriteHeader(http.StatusOK)
	})

	handler := auth.Middleware(testSecret)(inner)
	req := newReq()
	req.Header.Set("Authorization", "Bearer "+token)
	serve(handler, req)

	if got == nil {
		t.Fatal("expected claims in context, got nil")
	}
	if got.Username != "alice" {
		t.Errorf("Username = %q, want %q", got.Username, "alice")
	}
	if got.UserID != "uid1" {
		t.Errorf("UserID = %q, want %q", got.UserID, "uid1")
	}
}

func TestGetClaims_NilWhenNotSet(t *testing.T) {
	req := newReq()
	if c := auth.GetClaims(req); c != nil {
		t.Errorf("expected nil claims on unauthenticated request, got %+v", c)
	}
}

// helpers

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func newReq() *http.Request {
	return httptest.NewRequest(http.MethodGet, "/", nil)
}

func serve(h http.Handler, r *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	return rr
}
