// Package db opens the SQLite database and creates tables if they
// don't exist yet ("migration" is a fancy word for that here).
package db

import (
	"database/sql"

	// The underscore import means: run this package's init() (which
	// registers the "sqlite" driver) but we don't call it by name.
	_ "modernc.org/sqlite"
)

// Open connects to the SQLite file at path. It creates the file if
// it doesn't exist.
func Open(path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := database.Ping(); err != nil {
		return nil, err
	}
	return database, nil
}

// schema holds every CREATE TABLE statement. IF NOT EXISTS makes it
// safe to run every time the server starts.
const schema = `
CREATE TABLE IF NOT EXISTS users (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	username      TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
	token      TEXT PRIMARY KEY,
	user_id    INTEGER NOT NULL REFERENCES users(id),
	expires_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS problems (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id     INTEGER NOT NULL REFERENCES users(id),
	operand_a   INTEGER NOT NULL,
	operand_b   INTEGER NOT NULL,
	operator    TEXT NOT NULL,
	answer      INTEGER NOT NULL,
	user_answer INTEGER,
	correct     INTEGER,
	created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tasks (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id    INTEGER NOT NULL REFERENCES users(id),
	question   TEXT NOT NULL,
	answer     INTEGER NOT NULL,
	answered   INTEGER NOT NULL DEFAULT 0,
	correct    INTEGER,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

// Migrate creates any missing tables.
func Migrate(database *sql.DB) error {
	_, err := database.Exec(schema)
	return err
}
