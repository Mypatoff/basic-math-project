# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Who's reading this

The user is a Go beginner. Prefer simple, stdlib-first code over clever
abstractions. When introducing a new stdlib feature or concept, add a short
comment explaining it (e.g. `go:embed`, `http.MaxBytesReader`, context keys).

## Stack (do not add frameworks/routers/ORMs)

- Go 1.22+, `net/http` with `"METHOD /path"` ServeMux patterns
- `modernc.org/sqlite` (driver name `"sqlite"`, pure Go, no cgo)
- `golang.org/x/crypto/bcrypt` for password hashing
- `html/template` for pages, `go:embed` to bundle `web/templates` and `web/static`
- Vanilla JS (`web/static/app.js`), one CSS file (`web/static/style.css`)

## Rules

- Always run `go version` first; stop if below 1.22.
- SQL: parameterized queries only, never string-build SQL.
- JSON errors: `{"error": "..."}` with the correct HTTP status code.
- Cap request bodies with `http.MaxBytesReader` (see `handlers.decodeJSON`).
- JS must use `textContent`, never `innerHTML`, for any server-provided data.
- `go:embed` can't reference parent directories — that's why `web/embed.go`
  lives next to `templates/` and `static/`, not in `cmd/server`.
- Before finishing a change: `go vet ./...` and `go test ./...` must pass.
- If a build/test fails twice on the same error, stop and explain rather than
  retrying blindly.

## Layout

See README.md "Layout" section — it's kept in sync with the actual package
structure.
