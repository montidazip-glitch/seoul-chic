package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// isAjax checks if the HTTP request is an AJAX / Fetch request.
func isAjax(r *http.Request) bool {
	return r.Header.Get("X-Requested-With") == "XMLHttpRequest" ||
		strings.Contains(r.Header.Get("Accept"), "application/json")
}

// muxVars is a thin wrapper so other files in this package can call gorilla/mux Vars.
func muxVars(r *http.Request) map[string]string {
	return mux.Vars(r)
}

// parseID parses an integer path variable by name from a gorilla/mux route.
// Returns 0 and false if the variable is missing or not a valid integer.
func parseID(r *http.Request, name string) (int, bool) {
	vars := mux.Vars(r)
	idStr, ok := vars[name]
	if !ok {
		return 0, false
	}
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
