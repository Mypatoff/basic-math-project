package handlers

import (
	"net/http"

	"mathpractice/internal/auth"
)

// Progress summarizes a user's answered tasks plus their current
// streak of consecutive correct answers, most recent first. This is
// a rollup across every lesson/level, not just one.
func (h *Handlers) Progress(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r)

	var total, correct int
	row := h.DB.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(correct), 0) FROM tasks
		 WHERE user_id = ? AND answered = 1`,
		userID,
	)
	if err := row.Scan(&total, &correct); err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not load progress")
		return
	}

	var accuracy float64
	if total > 0 {
		accuracy = float64(correct) / float64(total)
	}

	// Walk answered tasks newest-first, counting correct answers
	// until the first wrong one to get the current streak.
	rows, err := h.DB.Query(
		`SELECT correct FROM tasks WHERE user_id = ? AND answered = 1
		 ORDER BY created_at DESC, id DESC`,
		userID,
	)
	if err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not load progress")
		return
	}
	defer rows.Close()

	streak := 0
	for rows.Next() {
		var c int
		if err := rows.Scan(&c); err != nil {
			auth.WriteJSONError(w, http.StatusInternalServerError, "could not load progress")
			return
		}
		if c == 0 {
			break
		}
		streak++
	}
	if err := rows.Err(); err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not load progress")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":    total,
		"correct":  correct,
		"accuracy": accuracy,
		"streak":   streak,
	})
}
