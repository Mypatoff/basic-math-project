package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mathpractice/internal/auth"
	"mathpractice/internal/db"
)

// newTestHandlers spins up an in-memory SQLite database with the
// real schema, so these tests exercise actual SQL, not mocks.
func newTestHandlers(t *testing.T) (*Handlers, *auth.Store) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	if err := db.Migrate(database); err != nil {
		t.Fatal(err)
	}
	store := &auth.Store{DB: database}
	return &Handlers{DB: database, Auth: store}, store
}

// loginAs creates a real session for userID and returns its cookie,
// the same way a browser would get one from a successful login.
func loginAs(t *testing.T, store *auth.Store, userID int64) *http.Cookie {
	t.Helper()
	rec := httptest.NewRecorder()
	if err := store.CreateSession(rec, userID); err != nil {
		t.Fatal(err)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("CreateSession did not set a cookie")
	}
	return cookies[0]
}

func jsonRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestNextTaskDifficultyValidation(t *testing.T) {
	h, store := newTestHandlers(t)
	userID, err := store.CreateUser("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	cookie := loginAs(t, store, userID)
	handler := store.RequireAuth(h.NextTask)

	for _, d := range []string{"0", "4", "abc", ""} {
		req := httptest.NewRequest(http.MethodGet, "/api/tasks/next?d="+d, nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("d=%q: got status %d, want 400", d, rec.Code)
		}
	}

	for _, d := range []string{"1", "2", "3"} {
		req := httptest.NewRequest(http.MethodGet, "/api/tasks/next?d="+d, nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("d=%q: got status %d, want 200", d, rec.Code)
		}
		var resp struct {
			TaskID   int64  `json:"task_id"`
			Question string `json:"question"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		if resp.TaskID == 0 || resp.Question == "" {
			t.Errorf("d=%q: got empty task_id/question: %+v", d, resp)
		}
	}
}

func TestAnswerTaskCorrectAndIncorrect(t *testing.T) {
	h, store := newTestHandlers(t)
	userID, err := store.CreateUser("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	cookie := loginAs(t, store, userID)
	answerHandler := store.RequireAuth(h.AnswerTask)

	insertTask := func(answer int) int64 {
		res, err := h.DB.Exec(
			`INSERT INTO tasks (user_id, question, answer) VALUES (?, ?, ?)`,
			userID, "2 + 2", answer,
		)
		if err != nil {
			t.Fatal(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}

	// Correct answer.
	correctTaskID := insertTask(4)
	req := jsonRequest(t, http.MethodPost, "/api/tasks/answer", map[string]any{"task_id": correctTaskID, "answer": 4})
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	answerHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("correct answer: got status %d, want 200", rec.Code)
	}
	var result struct {
		Correct       bool `json:"correct"`
		CorrectAnswer int  `json:"correct_answer"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if !result.Correct || result.CorrectAnswer != 4 {
		t.Errorf("got %+v, want correct=true correct_answer=4", result)
	}

	// Answering the same task again is a conflict.
	req2 := jsonRequest(t, http.MethodPost, "/api/tasks/answer", map[string]any{"task_id": correctTaskID, "answer": 4})
	req2.AddCookie(cookie)
	rec2 := httptest.NewRecorder()
	answerHandler(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Errorf("re-answering: got status %d, want 409", rec2.Code)
	}

	// Wrong answer on a fresh task.
	wrongTaskID := insertTask(4)
	req3 := jsonRequest(t, http.MethodPost, "/api/tasks/answer", map[string]any{"task_id": wrongTaskID, "answer": 5})
	req3.AddCookie(cookie)
	rec3 := httptest.NewRecorder()
	answerHandler(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("wrong answer: got status %d, want 200", rec3.Code)
	}
	var result3 struct {
		Correct       bool `json:"correct"`
		CorrectAnswer int  `json:"correct_answer"`
	}
	json.NewDecoder(rec3.Body).Decode(&result3)
	if result3.Correct || result3.CorrectAnswer != 4 {
		t.Errorf("got %+v, want correct=false correct_answer=4", result3)
	}

	// Non-integer answer is a bad request.
	req4 := httptest.NewRequest(http.MethodPost, "/api/tasks/answer", bytes.NewReader([]byte(`{"task_id":1,"answer":"x"}`)))
	req4.Header.Set("Content-Type", "application/json")
	req4.AddCookie(cookie)
	rec4 := httptest.NewRecorder()
	answerHandler(rec4, req4)
	if rec4.Code != http.StatusBadRequest {
		t.Errorf("non-integer answer: got status %d, want 400", rec4.Code)
	}

	// Unknown task id is not found.
	req5 := jsonRequest(t, http.MethodPost, "/api/tasks/answer", map[string]any{"task_id": 999999, "answer": 1})
	req5.AddCookie(cookie)
	rec5 := httptest.NewRecorder()
	answerHandler(rec5, req5)
	if rec5.Code != http.StatusNotFound {
		t.Errorf("unknown task: got status %d, want 404", rec5.Code)
	}
}

func TestAnswerTaskWrongOwner(t *testing.T) {
	h, store := newTestHandlers(t)
	aliceID, err := store.CreateUser("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bobID, err := store.CreateUser("bob", "password123")
	if err != nil {
		t.Fatal(err)
	}

	res, err := h.DB.Exec(
		`INSERT INTO tasks (user_id, question, answer) VALUES (?, ?, ?)`,
		aliceID, "2 + 2", 4,
	)
	if err != nil {
		t.Fatal(err)
	}
	taskID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	bobCookie := loginAs(t, store, bobID)
	answerHandler := store.RequireAuth(h.AnswerTask)

	req := jsonRequest(t, http.MethodPost, "/api/tasks/answer", map[string]any{"task_id": taskID, "answer": 4})
	req.AddCookie(bobCookie)
	rec := httptest.NewRecorder()
	answerHandler(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("bob answering alice's task: got status %d, want 404", rec.Code)
	}

	// The task must still be answerable by its actual owner.
	aliceCookie := loginAs(t, store, aliceID)
	req2 := jsonRequest(t, http.MethodPost, "/api/tasks/answer", map[string]any{"task_id": taskID, "answer": 4})
	req2.AddCookie(aliceCookie)
	rec2 := httptest.NewRecorder()
	answerHandler(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("alice answering her own task: got status %d, want 200", rec2.Code)
	}
}

func TestProgressStreak(t *testing.T) {
	h, store := newTestHandlers(t)
	userID, err := store.CreateUser("carol", "password123")
	if err != nil {
		t.Fatal(err)
	}
	cookie := loginAs(t, store, userID)
	nextHandler := store.RequireAuth(h.NextTask)
	answerHandler := store.RequireAuth(h.AnswerTask)
	progressHandler := store.RequireAuth(h.Progress)

	// wrong, correct, correct, correct -> total 4, correct 3, streak 3
	// (streak counts consecutive correct answers ending at the most recent).
	for _, wantCorrect := range []bool{false, true, true, true} {
		req := httptest.NewRequest(http.MethodGet, "/api/tasks/next?d=1", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		nextHandler(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("next task: got status %d, want 200", rec.Code)
		}
		var nt struct {
			TaskID int64 `json:"task_id"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&nt); err != nil {
			t.Fatal(err)
		}

		var correctAnswer int
		if err := h.DB.QueryRow(`SELECT answer FROM tasks WHERE id = ?`, nt.TaskID).Scan(&correctAnswer); err != nil {
			t.Fatal(err)
		}
		given := correctAnswer
		if !wantCorrect {
			given = correctAnswer + 1
		}

		req2 := jsonRequest(t, http.MethodPost, "/api/tasks/answer", map[string]any{"task_id": nt.TaskID, "answer": given})
		req2.AddCookie(cookie)
		rec2 := httptest.NewRecorder()
		answerHandler(rec2, req2)
		if rec2.Code != http.StatusOK {
			t.Fatalf("answer task: got status %d, want 200", rec2.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/progress", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	progressHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("progress: got status %d, want 200", rec.Code)
	}
	var p struct {
		Total    int     `json:"total"`
		Correct  int     `json:"correct"`
		Accuracy float64 `json:"accuracy"`
		Streak   int     `json:"streak"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.Total != 4 || p.Correct != 3 || p.Streak != 3 {
		t.Errorf("got %+v, want total=4 correct=3 streak=3", p)
	}
	if p.Accuracy != 0.75 {
		t.Errorf("got accuracy=%v, want 0.75", p.Accuracy)
	}
}
