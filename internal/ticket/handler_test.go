package ticket_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ticket-system/internal/auth"
	"ticket-system/internal/ticket"
	"ticket-system/internal/user"
)

// newTestServer wires the same routes as cmd/server/main.go, scoped to
// what these tests need, so tests exercise real HTTP request/response
// flow instead of calling Go functions directly.
func newTestServer(t *testing.T) (mux *http.ServeMux, secret []byte) {
	t.Helper()

	secret = []byte("test-secret")
	userStore := user.NewMemoryStore()
	ticketStore := ticket.NewMemoryStore()

	userHandler := user.NewHandler(userStore, secret, time.Hour)
	ticketHandler := ticket.NewHandler(ticketStore)
	requireAuth := auth.RequireAuth(secret)

	mux = http.NewServeMux()
	mux.HandleFunc("POST /auth/register", userHandler.Register)
	mux.HandleFunc("POST /auth/login", userHandler.Login)
	mux.Handle("POST /tickets", requireAuth(http.HandlerFunc(ticketHandler.Create)))
	mux.Handle("GET /tickets", requireAuth(http.HandlerFunc(ticketHandler.List)))
	mux.Handle("GET /tickets/{id}", requireAuth(http.HandlerFunc(ticketHandler.Get)))
	mux.Handle("PATCH /tickets/{id}/status", requireAuth(http.HandlerFunc(ticketHandler.UpdateStatus)))

	return mux, secret
}

func doRequest(t *testing.T, mux *http.ServeMux, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func registerAndLogin(t *testing.T, mux *http.ServeMux, email string) string {
	t.Helper()

	body := `{"email":"` + email + `","password":"password123"}`
	rec := doRequest(t, mux, http.MethodPost, "/auth/register", "", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, mux, http.MethodPost, "/auth/login", "", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	return loginResp.Token
}

func TestTicketLifecycle_CreateListGetUpdate(t *testing.T) {
	mux, _ := newTestServer(t)
	token := registerAndLogin(t, mux, "alice@example.com")

	rec := doRequest(t, mux, http.MethodPost, "/tickets", token,
		`{"title":"Printer on fire","description":"send help"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var created ticket.Ticket
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode created ticket: %v", err)
	}
	if created.Status != ticket.StatusOpen {
		t.Errorf("new ticket status = %q, want %q", created.Status, ticket.StatusOpen)
	}

	rec = doRequest(t, mux, http.MethodGet, "/tickets", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec.Code)
	}
	var listed []ticket.Ticket
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("failed to decode ticket list: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 ticket in list, got %d", len(listed))
	}

	rec = doRequest(t, mux, http.MethodGet, "/tickets/"+created.ID, token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", rec.Code)
	}

	rec = doRequest(t, mux, http.MethodPatch, "/tickets/"+created.ID+"/status", token,
		`{"status":"in_progress"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("open->in_progress: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, mux, http.MethodPatch, "/tickets/"+created.ID+"/status", token,
		`{"status":"closed"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("in_progress->closed: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, mux, http.MethodPatch, "/tickets/"+created.ID+"/status", token,
		`{"status":"open"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("closed->open: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTicketOwnership_CannotAccessAnotherUsersTicket(t *testing.T) {
	mux, _ := newTestServer(t)

	aliceToken := registerAndLogin(t, mux, "alice@example.com")
	bobToken := registerAndLogin(t, mux, "bob@example.com")

	rec := doRequest(t, mux, http.MethodPost, "/tickets", aliceToken,
		`{"title":"Alice's private ticket","description":"secret"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", rec.Code)
	}
	var aliceTicket ticket.Ticket
	if err := json.Unmarshal(rec.Body.Bytes(), &aliceTicket); err != nil {
		t.Fatalf("failed to decode ticket: %v", err)
	}

	rec = doRequest(t, mux, http.MethodGet, "/tickets/"+aliceTicket.ID, bobToken, "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("bob GET alice's ticket: expected 404, got %d", rec.Code)
	}

	rec = doRequest(t, mux, http.MethodPatch, "/tickets/"+aliceTicket.ID+"/status", bobToken,
		`{"status":"in_progress"}`)
	if rec.Code != http.StatusNotFound {
		t.Errorf("bob update alice's ticket: expected 404, got %d", rec.Code)
	}

	rec = doRequest(t, mux, http.MethodGet, "/tickets", bobToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("bob list: expected 200, got %d", rec.Code)
	}
	var bobTickets []ticket.Ticket
	if err := json.Unmarshal(rec.Body.Bytes(), &bobTickets); err != nil {
		t.Fatalf("failed to decode bob's ticket list: %v", err)
	}
	if len(bobTickets) != 0 {
		t.Errorf("expected bob's ticket list to be empty, got %d tickets", len(bobTickets))
	}
}

func TestTicketEndpoints_RequireAuth(t *testing.T) {
	mux, _ := newTestServer(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/tickets"},
		{http.MethodGet, "/tickets"},
		{http.MethodGet, "/tickets/some-id"},
		{http.MethodPatch, "/tickets/some-id/status"},
	}

	for _, ep := range endpoints {
		rec := doRequest(t, mux, ep.method, ep.path, "" /* no token */, `{}`)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without token: expected 401, got %d", ep.method, ep.path, rec.Code)
		}
	}
}

func TestCreateTicket_MissingTitleRejected(t *testing.T) {
	mux, _ := newTestServer(t)
	token := registerAndLogin(t, mux, "alice@example.com")

	rec := doRequest(t, mux, http.MethodPost, "/tickets", token, `{"description":"no title here"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing title, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateStatus_InvalidStatusValueRejected(t *testing.T) {
	mux, _ := newTestServer(t)
	token := registerAndLogin(t, mux, "alice@example.com")

	rec := doRequest(t, mux, http.MethodPost, "/tickets", token,
		`{"title":"test","description":"test"}`)
	var created ticket.Ticket
	json.Unmarshal(rec.Body.Bytes(), &created)

	rec = doRequest(t, mux, http.MethodPatch, "/tickets/"+created.ID+"/status", token,
		`{"status":"not_a_real_status"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid status value, got %d: %s", rec.Code, rec.Body.String())
	}
}
