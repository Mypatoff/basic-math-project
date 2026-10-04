# Math Practice

## Prerequisites

- Go 1.22 or later
- No system SQLite needed — `modernc.org/sqlite` is pure Go

## Run

```bash
go run ./cmd/server
```

Then open http://localhost:8080.

## Env vars

None. The port (`:8080`) and database path (`data/app.db`) are fixed
in `cmd/server/main.go`.
