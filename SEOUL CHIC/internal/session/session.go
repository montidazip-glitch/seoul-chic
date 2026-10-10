package session

import (
	"net/http"

	"github.com/gorilla/sessions"
)

var store *sessions.CookieStore

const sessionName = "kfry-session"

// Init initialises the cookie store with the provided secret.
// Must be called once before any request handling begins.
func Init(secret string) {
	store = sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set true when running behind HTTPS in production
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7, // 7 days
	}
}

// Get retrieves (or creates) the named session from the request.
func Get(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, sessionName)
}

// Save persists session changes to the response writer.
func Save(r *http.Request, w http.ResponseWriter) error {
	sess, err := Get(r)
	if err != nil {
		return err
	}
	return sess.Save(r, w)
}

// SetFlash stores a one-time flash message in the session.
func SetFlash(r *http.Request, w http.ResponseWriter, kind, message string) {
	sess, _ := Get(r)
	sess.AddFlash(message, kind)
	sess.Save(r, w)
}

// GetFlashes retrieves and clears flash messages of the given kind.
func GetFlashes(r *http.Request, w http.ResponseWriter, kind string) []interface{} {
	sess, _ := Get(r)
	flashes := sess.Flashes(kind)
	sess.Save(r, w)
	return flashes
}

// GetUserID returns the logged-in user's ID from the session, or 0 if not set.
func GetUserID(r *http.Request) int {
	sess, _ := Get(r)
	v, _ := sess.Values["userID"].(int)
	return v
}

// GetUserRole returns the logged-in user's role from the session.
func GetUserRole(r *http.Request) string {
	sess, _ := Get(r)
	v, _ := sess.Values["role"].(string)
	return v
}

// SetUser stores user info in the session after a successful login.
func SetUser(r *http.Request, w http.ResponseWriter, userID int, role string) {
	sess, _ := Get(r)
	sess.Values["userID"] = userID
	sess.Values["role"] = role
	sess.Save(r, w)
}

// ClearUser removes user info from the session (logout).
func ClearUser(r *http.Request, w http.ResponseWriter) {
	sess, _ := Get(r)
	delete(sess.Values, "userID")
	delete(sess.Values, "role")
	sess.Options.MaxAge = -1
	sess.Save(r, w)
}
