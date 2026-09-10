package user_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ticket-system/internal/user"
)

func newUserMux(t *testing.T) *http.ServeMux {
	t.Helper()
	store := user.NewMemoryStore()
	h := user.NewHandler(store, []byte("test-secret"), time.Hour)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/login", h.Login)
	return mux
}

func doRequest(t *testing.T, mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestRegister_Success(t *testing.T) {
	mux := newUserMux(t)
	rec := doRequest(t, mux, http.MethodPost, "/auth/register",
		`{"email":"a@example.com","password":"password123"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegister_DuplicateEmailRejected(t *testing.T) {
	mux := newUserMux(t)
	body := `{"email":"a@example.com","password":"password123"}`

	doRequest(t, mux, http.MethodPost, "/auth/register", body)
	rec := doRequest(t, mux, http.MethodPost, "/auth/register", body)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate email, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegister_InvalidEmailRejected(t *testing.T) {
	mux := newUserMux(t)
	rec := doRequest(t, mux, http.MethodPost, "/auth/register",
		`{"email":"not-an-email","password":"password123"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid email, got %d", rec.Code)
	}
}

func TestRegister_ShortPasswordRejected(t *testing.T) {
	mux := newUserMux(t)
	rec := doRequest(t, mux, http.MethodPost, "/auth/register",
		`{"email":"a@example.com","password":"short"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short password, got %d", rec.Code)
	}
}

func TestLogin_WrongPasswordRejected(t *testing.T) {
	mux := newUserMux(t)
	doRequest(t, mux, http.MethodPost, "/auth/register",
		`{"email":"a@example.com","password":"password123"}`)

	rec := doRequest(t, mux, http.MethodPost, "/auth/login",
		`{"email":"a@example.com","password":"wrong-password"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", rec.Code)
	}
}

func TestLogin_UnknownEmailRejected(t *testing.T) {
	mux := newUserMux(t)
	rec := doRequest(t, mux, http.MethodPost, "/auth/login",
		`{"email":"nobody@example.com","password":"password123"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unknown email, got %d", rec.Code)
	}
}

func TestLogin_EmailIsCaseInsensitive(t *testing.T) {
	mux := newUserMux(t)
	doRequest(t, mux, http.MethodPost, "/auth/register",
		`{"email":"Alice@Example.com","password":"password123"}`)

	rec := doRequest(t, mux, http.MethodPost, "/auth/login",
		`{"email":"alice@example.com","password":"password123"}`)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for case-insensitive email login, got %d: %s", rec.Code, rec.Body.String())
	}
}
