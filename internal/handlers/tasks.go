package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"mathpractice/internal/auth"
	"mathpractice/internal/tasks"
)

// NextTask generates a problem at the requested difficulty (1-3),
// saves it so AnswerTask can grade it later, and returns the
// question text. The answer itself never goes to the client.
func (h *Handlers) NextTask(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r)

	difficulty, err := strconv.Atoi(r.URL.Query().Get("d"))
	if err != nil {
		auth.WriteJSONError(w, http.StatusBadRequest, "d must be 1, 2, or 3")
		return
	}
	p, err := tasks.GenerateWithDifficulty(difficulty)
	if err != nil {
		auth.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.DB.Exec(
		`INSERT INTO tasks (user_id, question, answer) VALUES (?, ?, ?)`,
		userID, p.Question(), p.Answer,
	)
	if err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not create task")
		return
	}
	taskID, err := res.LastInsertId()
	if err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not create task")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"task_id":  taskID,
		"question": p.Question(),
	})
}

type taskAnswerRequest struct {
	TaskID int64 `json:"task_id"`
	Answer int   `json:"answer"`
}

// AnswerTask grades an answer against the task's stored answer. The
// task must exist, belong to the caller, and be unanswered; a wrong
// owner is reported the same way as "not found" so a task id can't
// be used to probe whether it belongs to someone else.
func (h *Handlers) AnswerTask(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r)

	var body taskAnswerRequest
	if err := decodeJSON(w, r, &body); err != nil {
		auth.WriteJSONError(w, http.StatusBadRequest, "task_id and answer (integer) are required")
		return
	}

	var ownerID, correctAnswer, answered int64
	row := h.DB.QueryRow(
		`SELECT user_id, answer, answered FROM tasks WHERE id = ?`, body.TaskID,
	)
	switch err := row.Scan(&ownerID, &correctAnswer, &answered); {
	case err == sql.ErrNoRows:
		auth.WriteJSONError(w, http.StatusNotFound, "task not found")
		return
	case err != nil:
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not load task")
		return
	case ownerID != userID:
		auth.WriteJSONError(w, http.StatusNotFound, "task not found")
		return
	case answered != 0:
		auth.WriteJSONError(w, http.StatusConflict, "task already answered")
		return
	}

	isCorrect := int64(body.Answer) == correctAnswer
	correctFlag := 0
	if isCorrect {
		correctFlag = 1
	}
	if _, err := h.DB.Exec(
		`UPDATE tasks SET answered = 1, correct = ? WHERE id = ?`,
		correctFlag, body.TaskID,
	); err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not save answer")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"correct":        isCorrect,
		"correct_answer": correctAnswer,
	})
}

// Progress summarizes a user's answered tasks plus their current
// streak of consecutive correct answers, most recent first.
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

	writeJSON(w, http.StatusOK, map[string]any{
		"total":    total,
		"correct":  correct,
		"accuracy": accuracy,
		"streak":   streak,
	})
}
