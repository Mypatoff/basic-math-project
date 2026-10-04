package handlers

import (
	"errors"
	"net/http"

	"mathpractice/internal/auth"
)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register creates a new user and logs them in.
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var creds credentials
	if err := decodeJSON(w, r, &creds); err != nil {
		auth.WriteJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if creds.Username == "" || len(creds.Password) < 6 {
		auth.WriteJSONError(w, http.StatusBadRequest, "username required, password must be 6+ characters")
		return
	}

	id, err := h.Auth.CreateUser(creds.Username, creds.Password)
	if err != nil {
		// SQLite reports the UNIQUE(username) violation as a generic
		// error, so we treat any failure here as "name taken".
		auth.WriteJSONError(w, http.StatusConflict, "username already taken")
		return
	}
	if err := h.Auth.CreateSession(w, id); err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": creds.Username})
}

// Login checks credentials and starts a session.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var creds credentials
	if err := decodeJSON(w, r, &creds); err != nil {
		auth.WriteJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	id, err := h.Auth.Authenticate(creds.Username, creds.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			auth.WriteJSONError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		auth.WriteJSONError(w, http.StatusInternalServerError, "login failed")
		return
	}
	if err := h.Auth.CreateSession(w, id); err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": creds.Username})
}

// Logout ends the current session.
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	h.Auth.Logout(w, r)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// Me reports who's currently logged in; the practice page uses this
// to decide whether to show the practice UI or bounce to /login.
func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r)
	username, createdAt, err := h.Auth.UserInfo(userID)
	if err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not load user")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"username":   username,
		"created_at": createdAt,
	})
}
