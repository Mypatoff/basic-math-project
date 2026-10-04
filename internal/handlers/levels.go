package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"mathpractice/internal/auth"
	"mathpractice/internal/tasks"
)

// lessonLength is how many questions make up one lesson; passingScore
// is the minimum correct answers (out of lessonLength) to pass a
// level. There's no separate "progress" table: a level counts as
// passed once the user has a finished lesson at that level scoring
// at least passingScore.
const (
	lessonLength = 5
	passingScore = 4
)

// levelPassed reports whether userID has a finished, passing lesson
// at levelID.
func (h *Handlers) levelPassed(userID int64, levelID int) (bool, error) {
	var count int
	err := h.DB.QueryRow(
		`SELECT COUNT(*) FROM lessons
		 WHERE user_id = ? AND level_id = ? AND finished_at IS NOT NULL AND score >= ?`,
		userID, levelID, passingScore,
	).Scan(&count)
	return count > 0, err
}

// Levels lists every level with this user's progress against it.
// Level 1 is always unlocked; level N unlocks once level N-1 is passed.
func (h *Handlers) Levels(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r)

	type levelResponse struct {
		ID        int    `json:"id"`
		Title     string `json:"title"`
		Unlocked  bool   `json:"unlocked"`
		Passed    bool   `json:"passed"`
		BestScore int    `json:"best_score"`
	}

	result := make([]levelResponse, 0, len(tasks.Levels))
	previousPassed := true // level 1 is always unlocked
	for _, level := range tasks.Levels {
		passed, err := h.levelPassed(userID, level.ID)
		if err != nil {
			auth.WriteJSONError(w, http.StatusInternalServerError, "could not load levels")
			return
		}

		var bestScore sql.NullInt64
		row := h.DB.QueryRow(
			`SELECT MAX(score) FROM lessons WHERE user_id = ? AND level_id = ? AND finished_at IS NOT NULL`,
			userID, level.ID,
		)
		if err := row.Scan(&bestScore); err != nil {
			auth.WriteJSONError(w, http.StatusInternalServerError, "could not load levels")
			return
		}

		result = append(result, levelResponse{
			ID:        level.ID,
			Title:     level.Title,
			Unlocked:  previousPassed,
			Passed:    passed,
			BestScore: int(bestScore.Int64),
		})
		previousPassed = passed
	}

	writeJSON(w, http.StatusOK, result)
}

// StartLevel creates a new lesson at the given level, if it's unlocked.
func (h *Handlers) StartLevel(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r)

	levelID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		auth.WriteJSONError(w, http.StatusNotFound, "level not found")
		return
	}
	level, ok := tasks.LevelByID(levelID)
	if !ok {
		auth.WriteJSONError(w, http.StatusNotFound, "level not found")
		return
	}

	unlocked := true
	if level.ID > 1 {
		unlocked, err = h.levelPassed(userID, level.ID-1)
		if err != nil {
			auth.WriteJSONError(w, http.StatusInternalServerError, "could not check level")
			return
		}
	}
	if !unlocked {
		auth.WriteJSONError(w, http.StatusForbidden, "level is locked")
		return
	}

	res, err := h.DB.Exec(`INSERT INTO lessons (user_id, level_id) VALUES (?, ?)`, userID, level.ID)
	if err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not start lesson")
		return
	}
	lessonID, err := res.LastInsertId()
	if err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not start lesson")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"lesson_id": lessonID, "total": lessonLength})
}

// NextInLesson returns the lesson's pending task if there is one
// (never creates a second task while one is unanswered), otherwise
// generates and saves the next one.
func (h *Handlers) NextInLesson(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r)

	lessonID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		auth.WriteJSONError(w, http.StatusNotFound, "lesson not found")
		return
	}

	var ownerID, levelID int64
	var finishedAt sql.NullTime
	row := h.DB.QueryRow(`SELECT user_id, level_id, finished_at FROM lessons WHERE id = ?`, lessonID)
	switch err := row.Scan(&ownerID, &levelID, &finishedAt); {
	case err == sql.ErrNoRows:
		auth.WriteJSONError(w, http.StatusNotFound, "lesson not found")
		return
	case err != nil:
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not load lesson")
		return
	case ownerID != userID:
		auth.WriteJSONError(w, http.StatusNotFound, "lesson not found")
		return
	case finishedAt.Valid:
		auth.WriteJSONError(w, http.StatusConflict, "lesson already finished")
		return
	}

	// Return the pending task if one exists, rather than creating a
	// new one. "number" is the task's 1-based position in the lesson.
	var taskID int64
	var question string
	var number int
	pendingRow := h.DB.QueryRow(
		`SELECT t.id, t.question,
		        (SELECT COUNT(*) FROM tasks WHERE lesson_id = t.lesson_id AND id <= t.id)
		   FROM tasks t WHERE t.lesson_id = ? AND t.answered = 0 LIMIT 1`,
		lessonID,
	)
	switch err := pendingRow.Scan(&taskID, &question, &number); err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]any{
			"task_id": taskID, "question": question, "number": number, "total": lessonLength,
		})
		return
	case sql.ErrNoRows:
		// Fall through to generate a new task.
	default:
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not load task")
		return
	}

	level, ok := tasks.LevelByID(int(levelID))
	if !ok {
		auth.WriteJSONError(w, http.StatusInternalServerError, "unknown level")
		return
	}
	p := level.Generate()

	res, err := h.DB.Exec(
		`INSERT INTO tasks (user_id, lesson_id, question, answer) VALUES (?, ?, ?, ?)`,
		userID, lessonID, p.Question(), p.Answer,
	)
	if err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not create task")
		return
	}
	newTaskID, err := res.LastInsertId()
	if err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not create task")
		return
	}

	var newNumber int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM tasks WHERE lesson_id = ?`, lessonID).Scan(&newNumber); err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not create task")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"task_id": newTaskID, "question": p.Question(), "number": newNumber, "total": lessonLength,
	})
}

type taskAnswerRequest struct {
	TaskID int64 `json:"task_id"`
	Answer int   `json:"answer"`
}

// AnswerTask grades an answer against the task's stored answer. The
// task must exist, belong to the caller, and be unanswered; a wrong
// owner is reported the same way as "not found" so a task id can't
// be used to probe whether it belongs to someone else. Once the
// lesson's 5th task is answered, the lesson is marked finished and
// scored, and the next level unlocks if this one was just passed.
func (h *Handlers) AnswerTask(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r)

	var body taskAnswerRequest
	if err := decodeJSON(w, r, &body); err != nil {
		auth.WriteJSONError(w, http.StatusBadRequest, "task_id and answer (integer) are required")
		return
	}

	var ownerID, lessonID, correctAnswer, answered int64
	row := h.DB.QueryRow(
		`SELECT user_id, lesson_id, answer, answered FROM tasks WHERE id = ?`, body.TaskID,
	)
	switch err := row.Scan(&ownerID, &lessonID, &correctAnswer, &answered); {
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

	var levelID int64
	if err := h.DB.QueryRow(`SELECT level_id FROM lessons WHERE id = ?`, lessonID).Scan(&levelID); err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not load lesson")
		return
	}

	var answeredCount, correctCount int
	row = h.DB.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(correct), 0) FROM tasks WHERE lesson_id = ? AND answered = 1`,
		lessonID,
	)
	if err := row.Scan(&answeredCount, &correctCount); err != nil {
		auth.WriteJSONError(w, http.StatusInternalServerError, "could not load lesson progress")
		return
	}

	finished := answeredCount >= lessonLength
	passed := false
	nextUnlocked := false
	if finished {
		passed = correctCount >= passingScore
		if _, err := h.DB.Exec(
			`UPDATE lessons SET finished_at = ?, score = ? WHERE id = ?`,
			time.Now(), correctCount, lessonID,
		); err != nil {
			auth.WriteJSONError(w, http.StatusInternalServerError, "could not finish lesson")
			return
		}
		if passed {
			if _, ok := tasks.LevelByID(int(levelID) + 1); ok {
				nextUnlocked = true
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"correct":        isCorrect,
		"correct_answer": correctAnswer,
		"answered":       answeredCount,
		"total":          lessonLength,
		"finished":       finished,
		"score":          correctCount,
		"passed":         passed,
		"next_unlocked":  nextUnlocked,
	})
}
