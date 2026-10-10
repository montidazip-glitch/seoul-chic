package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"kfry_shop/internal/session"
)

const csrfSessionKey = "csrf_token"

// generateToken creates a cryptographically secure random token.
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GetCSRFToken retrieves the CSRF token from the session, generating one if absent.
func GetCSRFToken(r *http.Request, w http.ResponseWriter) string {
	sess, _ := session.Get(r)
	token, ok := sess.Values[csrfSessionKey].(string)
	if !ok || token == "" {
		token = generateToken()
		sess.Values[csrfSessionKey] = token
		sess.Save(r, w)
	}
	return token
}

// ValidateCSRF checks that the form's _csrf field matches the session token.
// Returns true if the token is valid.
func ValidateCSRF(r *http.Request) bool {
	sess, _ := session.Get(r)
	sessionToken, _ := sess.Values[csrfSessionKey].(string)
	formToken := r.FormValue("_csrf")
	return sessionToken != "" && sessionToken == formToken
}

// CSRFMiddleware validates CSRF tokens on all state-changing (POST/PUT/DELETE/PATCH) requests.
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost ||
			r.Method == http.MethodPut ||
			r.Method == http.MethodDelete ||
			r.Method == http.MethodPatch {
			if !ValidateCSRF(r) {
				http.Error(w, "403 Forbidden — CSRF token mismatch", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
