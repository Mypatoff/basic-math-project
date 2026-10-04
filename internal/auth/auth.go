// Package auth handles user accounts, passwords, and login sessions.
package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidCredentials is returned when a username/password pair
// (or a session token) doesn't check out. We use one generic error
// so we never reveal whether a username exists.
var ErrInvalidCredentials = errors.New("invalid credentials")

const (
	sessionCookieName = "session"
	sessionTTL        = 7 * 24 * time.Hour
)

// Store wraps a *sql.DB with the queries auth needs.
type Store struct {
	DB *sql.DB
}

// hashPassword turns a plaintext password into a bcrypt hash that's
// safe to store. bcrypt has salting built in, so we don't manage
// salts ourselves.
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// CreateUser stores a new user with a bcrypt-hashed password.
func (s *Store) CreateUser(username, password string) (int64, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return 0, err
	}
	res, err := s.DB.Exec(
		`INSERT INTO users (username, password_hash) VALUES (?, ?)`,
		username, hash,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Authenticate checks a username/password and returns the user id on
// success.
func (s *Store) Authenticate(username, password string) (int64, error) {
	var id int64
	var hash string
	row := s.DB.QueryRow(
		`SELECT id, password_hash FROM users WHERE username = ?`, username,
	)
	if err := row.Scan(&id, &hash); err != nil {
		return 0, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return 0, ErrInvalidCredentials
	}
	return id, nil
}

// newToken makes a random, URL-safe session token.
func newToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// CreateSession makes a new session for a user and sets the cookie
// on the response.
func (s *Store) CreateSession(w http.ResponseWriter, userID int64) error {
	token, err := newToken()
	if err != nil {
		return err
	}
	expires := time.Now().Add(sessionTTL)
	_, err = s.DB.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expires,
	)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// UserID reads the session cookie from the request and returns the
// logged-in user's id.
func (s *Store) UserID(r *http.Request) (int64, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return 0, ErrInvalidCredentials
	}
	var userID int64
	var expiresAt time.Time
	row := s.DB.QueryRow(
		`SELECT user_id, expires_at FROM sessions WHERE token = ?`, cookie.Value,
	)
	if err := row.Scan(&userID, &expiresAt); err != nil {
		return 0, ErrInvalidCredentials
	}
	if time.Now().After(expiresAt) {
		return 0, ErrInvalidCredentials
	}
	return userID, nil
}

// Logout deletes the session tied to the request's cookie and clears
// it on the response.
func (s *Store) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.DB.Exec(`DELETE FROM sessions WHERE token = ?`, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// UserInfo returns a user's username and account creation time, for
// the /api/me endpoint.
func (s *Store) UserInfo(userID int64) (username string, createdAt time.Time, err error) {
	row := s.DB.QueryRow(`SELECT username, created_at FROM users WHERE id = ?`, userID)
	err = row.Scan(&username, &createdAt)
	return username, createdAt, err
}

// ChangePassword verifies the current password, stores the new one,
// and deletes every other session for this user — so a session on
// another device is logged out — while keeping the session that made
// this request alive.
func (s *Store) ChangePassword(r *http.Request, userID int64, current, newPassword string) error {
	var hash string
	row := s.DB.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, userID)
	if err := row.Scan(&hash); err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(current)) != nil {
		return ErrInvalidCredentials
	}

	newHash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	if _, err := s.DB.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, newHash, userID); err != nil {
		return err
	}

	var currentToken string
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		currentToken = cookie.Value
	}
	_, err = s.DB.Exec(`DELETE FROM sessions WHERE user_id = ? AND token != ?`, userID, currentToken)
	return err
}
