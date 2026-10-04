package handlers

import "net/http"

// pageData is the data every page template receives. LoggedIn drives
// the navbar (Levels/Account vs Login/Register); HideNav skips the
// navbar+footer entirely for the focus-mode lesson page.
type pageData struct {
	Title    string
	LoggedIn bool
	HideNav  bool
	LevelID  string
}

// loggedIn reports whether the request carries a valid session.
func (h *Handlers) loggedIn(r *http.Request) bool {
	_, err := h.Auth.UserID(r)
	return err == nil
}

// newPageData builds the data common to every normal (non-lesson) page.
func (h *Handlers) newPageData(r *http.Request, title string) pageData {
	return pageData{Title: title, LoggedIn: h.loggedIn(r)}
}

// Index shows the landing page to visitors and the levels path to
// logged-in users.
func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {
	if h.loggedIn(r) {
		h.render(w, "levels.html", h.newPageData(r, "Levels"))
		return
	}
	h.render(w, "landing.html", pageData{Title: "MathPath"})
}

func (h *Handlers) LoginPage(w http.ResponseWriter, r *http.Request) {
	if h.loggedIn(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	h.render(w, "login.html", pageData{Title: "Log In"})
}

func (h *Handlers) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if h.loggedIn(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	h.render(w, "register.html", pageData{Title: "Create Account"})
}

// LessonPage always renders; app.js drives the whole flow (start
// lesson, fetch questions, grade answers) and redirects to /login on
// a 401 or to "/" on a 403/404 (locked or unknown level).
func (h *Handlers) LessonPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "lesson.html", pageData{
		Title:   "Lesson",
		HideNav: true,
		LevelID: r.PathValue("id"),
	})
}

// AccountPage always renders; app.js fetches /api/me and
// /api/progress to fill it in.
func (h *Handlers) AccountPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "account.html", h.newPageData(r, "Account"))
}

// ProgressPage is gone; /progress now just redirects to /account.
func (h *Handlers) ProgressPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/account", http.StatusFound)
}
