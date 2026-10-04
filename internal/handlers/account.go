package handlers

import (
	"errors"
	"net/http"

	"mathpractice/internal/auth"
)

type changePasswordRequest struct {
	Current string `json:"current"`
	New     string `json:"new"`
}

// ChangePassword verifies the caller's current password before
// setting the new one, and logs out their other sessions.
func (h *Handlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r)

	var body changePasswordRequest
	if err := decodeJSON(w, r, &body); err != nil {
		auth.WriteJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(body.New) < 8 || len(body.New) > 72 {
		auth.WriteJSONError(w, http.StatusBadRequest, "new password must be 8-72 characters")
		return
	}

	if err := h.Auth.ChangePassword(r, userID, body.Current, body.New); err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			auth.WriteJSONError(w, http.StatusUnauthorized, "current password is incorrect")
			return
		}
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not change password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "password changed"})
}
