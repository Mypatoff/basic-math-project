package auth

import (
	"context"
	"encoding/json"
	"net/http"
)

// contextKey is its own type (instead of a plain string) so our key
// can never collide with a key some other package stuffs into the
// same context.
type contextKey string

const userIDKey contextKey = "userID"

// WriteJSONError writes a JSON body like {"error":"message"} with
// the given HTTP status code. Shared by this package and the
// handlers package so error responses stay consistent.
func WriteJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// RequireAuth wraps a handler so it only runs for logged-in users. On
// success it stores the user id in the request context so the next
// handler can read it with UserIDFromContext.
func (s *Store) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := s.UserID(r)
		if err != nil {
			WriteJSONError(w, http.StatusUnauthorized, "not logged in")
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

// UserIDFromContext reads the user id a RequireAuth middleware stored
// on the request context.
func UserIDFromContext(r *http.Request) int64 {
	id, _ := r.Context().Value(userIDKey).(int64)
	return id
}
