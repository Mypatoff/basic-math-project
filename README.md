# Math Practice

## Prerequisites

- Go 1.22 or later
- No system SQLite needed — `modernc.org/sqlite` is pure Go

## Run

```bash
go run ./cmd/server
```

Then open http://localhost:8080.

## Pages

| Path          | Logged out          | Logged in                        |
|---------------|----------------------|-----------------------------------|
| `/`           | Landing page         | Levels path                      |
| `/login`      | Login form           | redirects to `/`                 |
| `/register`   | Register form        | redirects to `/`                 |
| `/level/{id}` | renders (401 → login)| Lesson (focus mode, no navbar)   |
| `/account`    | renders (401 → login)| Profile, stats, change password  |
| `/progress`   | redirects to `/account` | redirects to `/account`       |

## Env vars

None. The port (`:8080`) and database path (`data/app.db`) are fixed
in `cmd/server/main.go`.
