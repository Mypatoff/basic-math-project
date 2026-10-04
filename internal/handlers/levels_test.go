package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

// startLevel calls StartLevel for levelID and returns the response
// status and, on success, the new lesson id.
func startLevel(t *testing.T, h *Handlers, store *auth.Store, cookie *http.Cookie, levelID int) (lessonID int64, status int) {
	t.Helper()
	handler := store.RequireAuth(h.StartLevel)
	req := httptest.NewRequest(http.MethodPost, "/api/levels/x/start", nil)
	req.SetPathValue("id", strconv.Itoa(levelID))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		return 0, rec.Code
	}
	var resp struct {
		LessonID int64 `json:"lesson_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	return resp.LessonID, rec.Code
}

// nextInLesson calls NextInLesson and returns the response status and,
// on success, the pending/new task id.
func nextInLesson(t *testing.T, h *Handlers, store *auth.Store, cookie *http.Cookie, lessonID int64) (taskID int64, status int) {
	t.Helper()
	handler := store.RequireAuth(h.NextInLesson)
	req := httptest.NewRequest(http.MethodGet, "/api/lessons/x/next", nil)
	req.SetPathValue("id", strconv.FormatInt(lessonID, 10))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		return 0, rec.Code
	}
	var resp struct {
		TaskID int64 `json:"task_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	return resp.TaskID, rec.Code
}

// answerTask calls AnswerTask and returns the response status and
// decoded JSON body.
func answerTask(t *testing.T, h *Handlers, store *auth.Store, cookie *http.Cookie, taskID int64, answer int) (body map[string]any, status int) {
	t.Helper()
	handler := store.RequireAuth(h.AnswerTask)
	req := jsonRequest(t, http.MethodPost, "/api/tasks/answer", map[string]any{"task_id": taskID, "answer": answer})
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler(rec, req)
	body = map[string]any{}
	json.NewDecoder(rec.Body).Decode(&body)
	return body, rec.Code
}

func correctAnswerFor(t *testing.T, h *Handlers, taskID int64) int {
	t.Helper()
	var answer int
	if err := h.DB.QueryRow(`SELECT answer FROM tasks WHERE id = ?`, taskID).Scan(&answer); err != nil {
		t.Fatal(err)
	}
	return answer
}

func TestStartLevelLockedAndUnknown(t *testing.T) {
	h, store := newTestHandlers(t)
	userID, err := store.CreateUser("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	cookie := loginAs(t, store, userID)

	if _, status := startLevel(t, h, store, cookie, 1); status != http.StatusOK {
		t.Fatalf("level 1 (always unlocked): got status %d, want 200", status)
	}
	if _, status := startLevel(t, h, store, cookie, 2); status != http.StatusForbidden {
		t.Fatalf("level 2 before passing level 1: got status %d, want 403", status)
	}
	if _, status := startLevel(t, h, store, cookie, 99); status != http.StatusNotFound {
		t.Fatalf("unknown level: got status %d, want 404", status)
	}
}

// playLesson starts a lesson at levelID and answers lessonLength
// questions, scoring exactly wantCorrect of them right (the rest
// deliberately wrong). It returns the final answer response.
func playLesson(t *testing.T, h *Handlers, store *auth.Store, cookie *http.Cookie, levelID, wantCorrect int) map[string]any {
	t.Helper()
	lessonID, status := startLevel(t, h, store, cookie, levelID)
	if status != http.StatusOK {
		t.Fatalf("start level %d: got status %d, want 200", levelID, status)
	}

	var last map[string]any
	for i := 0; i < lessonLength; i++ {
		taskID, status := nextInLesson(t, h, store, cookie, lessonID)
		if status != http.StatusOK {
			t.Fatalf("next task %d: got status %d, want 200", i+1, status)
		}
		correct := correctAnswerFor(t, h, taskID)
		given := correct
		if i >= wantCorrect {
			given = correct + 1 // deliberately wrong
		}
		body, status := answerTask(t, h, store, cookie, taskID, given)
		if status != http.StatusOK {
			t.Fatalf("answer task %d: got status %d, want 200", i+1, status)
		}
		last = body
	}
	return last
}

func TestLessonFinishScoringPasses(t *testing.T) {
	h, store := newTestHandlers(t)
	userID, err := store.CreateUser("bob", "password123")
	if err != nil {
		t.Fatal(err)
	}
	cookie := loginAs(t, store, userID)

	result := playLesson(t, h, store, cookie, 1, 4) // 4/5 correct

	if finished, _ := result["finished"].(bool); !finished {
		t.Errorf("got finished=%v, want true", result["finished"])
	}
	if score, _ := result["score"].(float64); score != 4 {
		t.Errorf("got score=%v, want 4", result["score"])
	}
	if passed, _ := result["passed"].(bool); !passed {
		t.Errorf("got passed=%v, want true", result["passed"])
	}
	if nextUnlocked, _ := result["next_unlocked"].(bool); !nextUnlocked {
		t.Errorf("got next_unlocked=%v, want true", result["next_unlocked"])
	}

	if _, status := startLevel(t, h, store, cookie, 2); status != http.StatusOK {
		t.Errorf("level 2 after passing level 1: got status %d, want 200", status)
	}
}

func TestLessonFinishScoringFails(t *testing.T) {
	h, store := newTestHandlers(t)
	userID, err := store.CreateUser("carol", "password123")
	if err != nil {
		t.Fatal(err)
	}
	cookie := loginAs(t, store, userID)

	result := playLesson(t, h, store, cookie, 1, 3) // 3/5 correct

	if finished, _ := result["finished"].(bool); !finished {
		t.Errorf("got finished=%v, want true", result["finished"])
	}
	if score, _ := result["score"].(float64); score != 3 {
		t.Errorf("got score=%v, want 3", result["score"])
	}
	if passed, _ := result["passed"].(bool); passed {
		t.Errorf("got passed=%v, want false", result["passed"])
	}
	if nextUnlocked, _ := result["next_unlocked"].(bool); nextUnlocked {
		t.Errorf("got next_unlocked=%v, want false", result["next_unlocked"])
	}

	if _, status := startLevel(t, h, store, cookie, 2); status != http.StatusForbidden {
		t.Errorf("level 2 after failing level 1: got status %d, want 403", status)
	}
}

func TestNextInLessonReturnsPendingTaskWithoutDuplicating(t *testing.T) {
	h, store := newTestHandlers(t)
	userID, err := store.CreateUser("dave", "password123")
	if err != nil {
		t.Fatal(err)
	}
	cookie := loginAs(t, store, userID)

	lessonID, status := startLevel(t, h, store, cookie, 1)
	if status != http.StatusOK {
		t.Fatalf("start level: got status %d, want 200", status)
	}

	firstTaskID, status := nextInLesson(t, h, store, cookie, lessonID)
	if status != http.StatusOK {
		t.Fatalf("first next: got status %d, want 200", status)
	}
	secondTaskID, status := nextInLesson(t, h, store, cookie, lessonID)
	if status != http.StatusOK {
		t.Fatalf("second next: got status %d, want 200", status)
	}
	if firstTaskID != secondTaskID {
		t.Errorf("got a new task (%d != %d) while one was still unanswered", firstTaskID, secondTaskID)
	}

	var count int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM tasks WHERE lesson_id = ?`, lessonID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("got %d tasks for the lesson, want 1", count)
	}
}

func TestLessonAndTaskOwnership(t *testing.T) {
	h, store := newTestHandlers(t)
	aliceID, err := store.CreateUser("erin", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bobID, err := store.CreateUser("frank", "password123")
	if err != nil {
		t.Fatal(err)
	}
	aliceCookie := loginAs(t, store, aliceID)
	bobCookie := loginAs(t, store, bobID)

	lessonID, status := startLevel(t, h, store, aliceCookie, 1)
	if status != http.StatusOK {
		t.Fatalf("alice start level: got status %d, want 200", status)
	}
	taskID, status := nextInLesson(t, h, store, aliceCookie, lessonID)
	if status != http.StatusOK {
		t.Fatalf("alice next task: got status %d, want 200", status)
	}

	if _, status := nextInLesson(t, h, store, bobCookie, lessonID); status != http.StatusNotFound {
		t.Errorf("bob fetching alice's lesson: got status %d, want 404", status)
	}
	if _, status := answerTask(t, h, store, bobCookie, taskID, correctAnswerFor(t, h, taskID)); status != http.StatusNotFound {
		t.Errorf("bob answering alice's task: got status %d, want 404", status)
	}

	if _, status := answerTask(t, h, store, aliceCookie, taskID, correctAnswerFor(t, h, taskID)); status != http.StatusOK {
		t.Errorf("alice answering her own task: got status %d, want 200", status)
	}
	if _, status := answerTask(t, h, store, aliceCookie, taskID, correctAnswerFor(t, h, taskID)); status != http.StatusConflict {
		t.Errorf("alice re-answering: got status %d, want 409", status)
	}
	if _, status := answerTask(t, h, store, aliceCookie, 999999, 0); status != http.StatusNotFound {
		t.Errorf("unknown task: got status %d, want 404", status)
	}
}
