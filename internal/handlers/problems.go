package handlers

import (
	"database/sql"
	"net/http"

	"mathpractice/internal/auth"
	"mathpractice/internal/tasks"
)

// NewProblem generates a random problem, saves it (so we know the
// right answer later), and sends the question to the client. The
// answer itself is never sent to the client.
func (h *Handlers) NewProblem(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r)
	p := tasks.Generate()

	res, err := h.DB.Exec(
		`INSERT INTO problems (user_id, operand_a, operand_b, operator, answer) VALUES (?, ?, ?, ?, ?)`,
		userID, p.A, p.B, p.Op, p.Answer,
	)
	if err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not create problem")
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not create problem")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       id,
		"question": p.Question(),
	})
}

type answerRequest struct {
	ID     int64 `json:"id"`
	Answer int   `json:"answer"`
}

// SubmitAnswer grades an answer against the problem's stored answer
// and records the attempt.
func (h *Handlers) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r)

	var body answerRequest
	if err := decodeJSON(w, r, &body); err != nil {
		auth.WriteJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var correctAnswer int
	var alreadyAnswered sql.NullInt64
	row := h.DB.QueryRow(
		`SELECT answer, user_answer FROM problems WHERE id = ? AND user_id = ?`,
		body.ID, userID,
	)
	if err := row.Scan(&correctAnswer, &alreadyAnswered); err != nil {
		auth.WriteJSONError(w, http.StatusNotFound, "problem not found")
		return
	}
	if alreadyAnswered.Valid {
		auth.WriteJSONError(w, http.StatusConflict, "problem already answered")
		return
	}

	isCorrect := body.Answer == correctAnswer
	// SQLite has no boolean type, so we store 0/1.
	correctFlag := 0
	if isCorrect {
		correctFlag = 1
	}
	_, err := h.DB.Exec(
		`UPDATE problems SET user_answer = ?, correct = ? WHERE id = ?`,
		body.Answer, correctFlag, body.ID,
	)
	if err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not save answer")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"correct": isCorrect,
		"answer":  correctAnswer,
	})
}

// Stats reports how many problems the user has answered and how
// many they got right.
func (h *Handlers) Stats(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r)

	var total, correct int
	row := h.DB.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(correct), 0) FROM problems
		 WHERE user_id = ? AND user_answer IS NOT NULL`,
		userID,
	)
	if err := row.Scan(&total, &correct); err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not load stats")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"total": total, "correct": correct})
}
