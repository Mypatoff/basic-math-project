// Command server runs the math practice web app.
package main

import (
	"io/fs"
	"log"
	"net/http"

	"mathpractice/internal/auth"
	"mathpractice/internal/db"
	"mathpractice/internal/handlers"
	"mathpractice/web"
)

func main() {
	database, err := db.Open("data/app.db")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()
	if err := db.Migrate(database); err != nil {
		log.Fatalf("migrate db: %v", err)
	}

	authStore := &auth.Store{DB: database}

	// fs.Sub gives us a filesystem rooted at "templates" (or
	// "static") so the embedded files can be referenced without the
	// "templates/" prefix, same as if they lived in that folder.
	templateFS, err := fs.Sub(web.FS, "templates")
	if err != nil {
		log.Fatalf("template fs: %v", err)
	}
	staticFS, err := fs.Sub(web.FS, "static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}

	h, err := handlers.New(database, authStore, templateFS)
	if err != nil {
		log.Fatalf("parse templates: %v", err)
	}

	mux := http.NewServeMux()

	// Pages. Go 1.22's ServeMux understands "METHOD /path" patterns,
	// so we no longer need manual method checks or a routing library.
	mux.HandleFunc("GET /{$}", h.Index) // {$} matches "/" only, not every path
	mux.HandleFunc("GET /login", h.LoginPage)
	mux.HandleFunc("GET /register", h.RegisterPage)
	mux.HandleFunc("GET /level/{id}", h.LessonPage)
	mux.HandleFunc("GET /account", h.AccountPage)
	mux.HandleFunc("GET /progress", h.ProgressPage)

	// Static assets (style.css, app.js) served straight from the
	// embedded filesystem.
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

	// Auth API.
	mux.HandleFunc("POST /api/register", h.Register)
	mux.HandleFunc("POST /api/login", h.Login)
	mux.HandleFunc("POST /api/logout", h.Logout)
	mux.HandleFunc("GET /api/me", authStore.RequireAuth(h.Me))

	// Practice API, all behind RequireAuth.
	mux.HandleFunc("GET /api/problem", authStore.RequireAuth(h.NewProblem))
	mux.HandleFunc("POST /api/answer", authStore.RequireAuth(h.SubmitAnswer))
	mux.HandleFunc("GET /api/stats", authStore.RequireAuth(h.Stats))

	// Levels/lessons API.
	mux.HandleFunc("GET /api/levels", authStore.RequireAuth(h.Levels))
	mux.HandleFunc("POST /api/levels/{id}/start", authStore.RequireAuth(h.StartLevel))
	mux.HandleFunc("GET /api/lessons/{id}/next", authStore.RequireAuth(h.NextInLesson))
	mux.HandleFunc("POST /api/tasks/answer", authStore.RequireAuth(h.AnswerTask))
	mux.HandleFunc("GET /api/progress", authStore.RequireAuth(h.Progress))

	mux.HandleFunc("POST /api/account/password", authStore.RequireAuth(h.ChangePassword))

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
