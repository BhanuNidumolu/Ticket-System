// Command server starts the ticket system HTTP API.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"ticket-system/internal/auth"
	"ticket-system/internal/httpx"
	"ticket-system/internal/ticket"
	"ticket-system/internal/user"
)

func main() {
	port := getEnv("PORT", "8080")
	jwtSecret := getJWTSecret()
	tokenTTL := getTokenTTL()

	userStore := user.NewMemoryStore()
	ticketStore := ticket.NewMemoryStore()

	userHandler := user.NewHandler(userStore, jwtSecret, tokenTTL)
	ticketHandler := ticket.NewHandler(ticketStore)

	requireAuth := auth.RequireAuth(jwtSecret)

	mux := http.NewServeMux()

	// Public routes.
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /auth/register", userHandler.Register)
	mux.HandleFunc("POST /auth/login", userHandler.Login)

	// Protected routes (ownership-scoped ticket operations).
	mux.Handle("POST /tickets", requireAuth(http.HandlerFunc(ticketHandler.Create)))
	mux.Handle("GET /tickets", requireAuth(http.HandlerFunc(ticketHandler.List)))
	mux.Handle("GET /tickets/{id}", requireAuth(http.HandlerFunc(ticketHandler.Get)))
	mux.Handle("PATCH /tickets/{id}/status", requireAuth(http.HandlerFunc(ticketHandler.UpdateStatus)))

	addr := ":" + port
	frontendOrigin := getEnv("FRONTEND_ORIGIN", "*")
	log.Printf("ticket-system listening on %s", addr)
	if err := http.ListenAndServe(addr, logRequests(httpx.CORS(frontendOrigin)(mux))); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// logRequests is a tiny access-log middleware, useful for debugging on
// a free-tier host where you may only have log tailing to work with.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getJWTSecret reads JWT_SECRET from the environment. If it's not set,
// it generates a random one for the lifetime of the process and warns
// loudly — fine for local/dev, but tokens won't survive a restart, so
// always set JWT_SECRET explicitly in any real deployment.
func getJWTSecret() []byte {
	if v := os.Getenv("JWT_SECRET"); v != "" {
		return []byte(v)
	}

	log.Println("WARNING: JWT_SECRET not set — generating an ephemeral secret for this process only.")
	log.Println("Set JWT_SECRET in your environment for a stable secret across restarts.")

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("failed to generate fallback JWT secret: %v", err)
	}
	return []byte(hex.EncodeToString(b))
}

func getTokenTTL() time.Duration {
	minutesStr := getEnv("TOKEN_TTL_MINUTES", "60")
	minutes, err := strconv.Atoi(minutesStr)
	if err != nil || minutes <= 0 {
		log.Printf("invalid TOKEN_TTL_MINUTES=%q, defaulting to 60", minutesStr)
		minutes = 60
	}
	return time.Duration(minutes) * time.Minute
}
