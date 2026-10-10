package middleware

import (
	"net/http"

	"kfry_shop/internal/session"
)

// RequireAuth ensures that the request comes from an authenticated user.
// If not authenticated, it redirects to /login.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if session.GetUserID(r) == 0 {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin ensures that the authenticated user has the "admin" role.
// Returns 403 Forbidden if the user is not an admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if session.GetUserRole(r) != "admin" {
			http.Error(w, "403 Forbidden — Admin access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
