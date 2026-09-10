package httpx

import "net/http"

// CORS returns middleware that allows a browser-based frontend hosted on
// a different origin (e.g. Netlify/Vercel) to call this API. allowedOrigin
// should be your frontend's exact origin in production (e.g.
// "https://your-frontend.vercel.app"); use "*" only for local development.
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
