// Package handlers wires HTTP requests to the db/auth/tasks packages
// and to the HTML templates.
package handlers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"

	"mathpractice/internal/auth"
)

// Handlers holds everything the HTTP handlers need to do their job.
type Handlers struct {
	DB   *sql.DB
	Auth *auth.Store
	tmpl map[string]*template.Template
}

// pageNames lists every page template parsed alongside layout.html.
// Each one defines a "content" block; if we parsed them all into one
// shared *template.Template, their "content" blocks would overwrite
// each other, so New gives each page its own template set instead.
var pageNames = []string{"login.html", "register.html", "practice.html", "progress.html"}

// New parses layout.html together with each page once at startup.
// Parsing up front (instead of per-request) is both faster and
// catches template typos immediately rather than on first visit.
func New(db *sql.DB, authStore *auth.Store, templateFS fs.FS) (*Handlers, error) {
	tmpl := make(map[string]*template.Template, len(pageNames))
	for _, name := range pageNames {
		t, err := template.ParseFS(templateFS, "layout.html", name)
		if err != nil {
			return nil, err
		}
		tmpl[name] = t
	}
	return &Handlers{DB: db, Auth: authStore, tmpl: tmpl}, nil
}

// render executes the layout for a named page, e.g. "practice.html".
func (h *Handlers) render(w http.ResponseWriter, name string, data any) {
	t, ok := h.tmpl[name]
	if !ok {
		http.Error(w, "unknown template: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// writeJSON sends v as a JSON body with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// decodeJSON reads a JSON request body into dst. MaxBytesReader caps
// the body size so a huge/malicious request can't exhaust memory;
// http.Error is skipped here because the caller decides the status
// code for a bad body.
const maxBodyBytes = 1 << 20 // 1 MiB

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	return json.NewDecoder(r.Body).Decode(dst)
}
