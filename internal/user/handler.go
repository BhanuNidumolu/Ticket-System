package user

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"ticket-system/internal/auth"
	"ticket-system/internal/httpx"
	"ticket-system/internal/idgen"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// Handler wires the user store and JWT config into HTTP handlers.
type Handler struct {
	Store     Store
	JWTSecret []byte
	TokenTTL  time.Duration
}

func NewHandler(store Store, jwtSecret []byte, tokenTTL time.Duration) *Handler {
	return &Handler{Store: store, JWTSecret: jwtSecret, TokenTTL: tokenTTL}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Register handles POST /auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Email == "" || !emailRegex.MatchString(req.Email) {
		httpx.WriteError(w, http.StatusBadRequest, "a valid email is required")
		return
	}
	if len(req.Password) < 8 {
		httpx.WriteError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	u := &User{
		ID:           idgen.New(),
		Email:        req.Email,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}

	if err := h.Store.Create(u); err != nil {
		if errors.Is(err, ErrDuplicateEmail) {
			httpx.WriteError(w, http.StatusConflict, "email already registered")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, registerResponse{
		ID:        u.ID,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	u, err := h.Store.FindByEmail(req.Email)
	if err != nil {
		// Same error for "no such user" and "wrong password" so we
		// don't leak which emails are registered.
		httpx.WriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	ok, err := auth.VerifyPassword(req.Password, u.PasswordHash)
	if err != nil || !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := auth.GenerateToken(u.ID, h.JWTSecret, h.TokenTTL)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, loginResponse{Token: token})
}
