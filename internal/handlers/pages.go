package handlers

import "net/http"

// pageData is the data every page template receives.
type pageData struct {
	Title string
}

// Index has no content of its own; "/" always sends visitors to the
// practice page.
func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/practice", http.StatusFound)
}

// loggedIn reports whether the request carries a valid session.
func (h *Handlers) loggedIn(r *http.Request) bool {
	_, err := h.Auth.UserID(r)
	return err == nil
}

func (h *Handlers) LoginPage(w http.ResponseWriter, r *http.Request) {
	if h.loggedIn(r) {
		http.Redirect(w, r, "/practice", http.StatusFound)
		return
	}
	h.render(w, "login.html", pageData{Title: "Log In"})
}

func (h *Handlers) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if h.loggedIn(r) {
		http.Redirect(w, r, "/practice", http.StatusFound)
		return
	}
	h.render(w, "register.html", pageData{Title: "Create Account"})
}

// PracticePage always renders; app.js checks /api/me itself and
// redirects to /login on a 401.
func (h *Handlers) PracticePage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "practice.html", pageData{Title: "Practice"})
}

// ProgressPage is a placeholder for now.
func (h *Handlers) ProgressPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "progress.html", pageData{Title: "Progress"})
}
