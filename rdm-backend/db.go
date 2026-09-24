package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"modernc.org/sqlite"
)

type DB struct {
	db *sqlx.DB
}

func NewDB() (*DB, error) {
	db := &DB{}

	_, err := os.Stat("data.db")

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if err := db.initDB(); err != nil {
		return nil, err
	}

	if err := db.createSchema(); err != nil {
		_ = db.db.Close()
		return nil, err
	}

	return db, nil
}

func init() {
	sqlite.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, dsn string) error {
		_, err := conn.ExecContext(context.Background(), `
			PRAGMA foreign_keys = ON;
			PRAGMA busy_timeout = 5000;
			PRAGMA synchronous = NORMAL;
		`, nil)

		return err
	})
}

func (db *DB) initDB() error {
	var err error

	db.db, err = sqlx.Open("sqlite", "data.db")
	if err != nil {
		return err
	}

	db.db.SetMaxOpenConns(8)
	db.db.SetMaxIdleConns(8)
	db.db.SetConnMaxLifetime(time.Hour)

	// WAL is a database-level setting, so initialize it once.
	if _, err := db.db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		_ = db.db.Close()
		return err
	}

	if err := db.db.Ping(); err != nil {
		_ = db.db.Close()
		return err
	}

	return nil
}

func (db *DB) createSchema() error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	discord_id TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS categories (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	"order" INTEGER NOT NULL,
	description TEXT,
	criteria TEXT,
	locked INTEGER NOT NULL DEFAULT 0 CHECK (locked IN (0, 1)),
	max_suggestions INTEGER
);

CREATE TABLE IF NOT EXISTS suggestions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	category_id INTEGER NOT NULL,
	text TEXT NOT NULL,
	updated_at DATETIME NOT NULL,
	nominator_id INTEGER NOT NULL,

	FOREIGN KEY (category_id)
		REFERENCES categories(id)
		ON DELETE CASCADE,

	FOREIGN KEY (nominator_id)
		REFERENCES users(id)
		ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sessions (
	token_hash BLOB PRIMARY KEY NOT NULL,
	user_id INTEGER NOT NULL,
	created_at DATETIME NOT NULL,
	expires_at DATETIME NOT NULL,

	FOREIGN KEY (user_id)
		REFERENCES users(id)
		ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_suggestions_category_id
	ON suggestions(category_id);

CREATE INDEX IF NOT EXISTS idx_suggestions_nominator_id
	ON suggestions(nominator_id);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id
	ON sessions(user_id);

CREATE INDEX IF NOT EXISTS idx_sessions_expires_at
	ON sessions(expires_at);
`

	_, err := db.db.Exec(schema)
	return err
}

func (db *DB) Close() error {
	return db.db.Close()
}

type User struct {
	ID        int64  `db:"id"`
	DiscordID string `db:"discord_id"`
}

type Category struct {
	ID             int64   `db:"id" json:"id"`
	Name           string  `db:"name" json:"name"`
	Order          int     `db:"order" json:"order"`
	Description    *string `db:"description" json:"description"`
	Criteria       *string `db:"criteria" json:"criteria"`
	Locked         bool    `db:"locked" json:"locked"`
	MaxSuggestions *int    `db:"max_suggestions" json:"maxSuggestions"`
}

type NominationSuggestion struct {
	ID          *int64    `db:"id" json:"id,omitempty"`
	CategoryID  int64     `db:"category_id" json:"categoryId"`
	Text        string    `db:"text" json:"text"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
	NominatorID int64     `db:"nominator_id" json:"nominatorId,omitempty"`
}

type Session struct {
	TokenHash []byte    `db:"token_hash"`
	UserID    int64     `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
	ExpiresAt time.Time `db:"expires_at"`
}

func (db *DB) getUserByDiscordID(discordID string) (*User, error) {
	var user User

	err := db.db.Get(
		&user,
		"SELECT * FROM users WHERE discord_id = ?",
		discordID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (db *DB) getOrCreateUser(discordID string) (*User, error) {
	user, err := db.getUserByDiscordID(discordID)
	if err != nil {
		return nil, err
	}

	if user != nil {
		return user, nil
	}

	return db.addAndReturnUser(discordID)
}

func (db *DB) getUserByID(id int64) (*User, error) {
	var user User

	err := db.db.Get(
		&user,
		"SELECT * FROM users WHERE id = ?",
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (db *DB) addAndReturnUser(discordID string) (*User, error) {
	var user User

	err := db.db.Get(
		&user,
		"INSERT INTO users (discord_id) VALUES (?) RETURNING *",
		discordID,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (db *DB) addCategory(category Category) error {
	_, err := db.db.Exec(
		`INSERT INTO categories
			(name, "order", description, criteria, locked, max_suggestions)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		category.Name,
		category.Order,
		category.Description,
		category.Criteria,
		category.Locked,
		category.MaxSuggestions,
	)

	return err
}

func (db *DB) updateCategory(category Category) error {
	_, err := db.db.Exec(
		`UPDATE categories
		 SET name = ?, "order" = ?, description = ?, criteria = ?, locked = ?, max_suggestions = ?
		 WHERE id = ?`,
		category.Name,
		category.Order,
		category.Description,
		category.Criteria,
		category.Locked,
		category.MaxSuggestions,
		category.ID,
	)

	return err
}

func (db *DB) getCategories() ([]Category, error) {
	var categories []Category

	err := db.db.Select(
		&categories,
		"SELECT * FROM categories ORDER BY \"order\" ASC",
	)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

func (db *DB) upsertCategory(category Category) error {
	if category.ID == 0 {
		return db.addCategory(category)
	}

	return db.updateCategory(category)
}

func (db *DB) deleteCategory(categoryID int64) error {
	_, err := db.db.Exec(
		`DELETE FROM categories WHERE id = ?`,
		categoryID,
	)

	return err
}

func (db *DB) getSuggestionsFor(
	categoryID int64,
	userID int64,
) ([]NominationSuggestion, error) {
	var suggestions []NominationSuggestion

	err := db.db.Select(
		&suggestions,
		`SELECT *
		 FROM suggestions
		 WHERE category_id = ?
		   AND nominator_id = ?`,
		categoryID,
		userID,
	)
	if err != nil {
		return nil, err
	}

	return suggestions, nil
}

func (db *DB) addSuggestion(
	suggestion NominationSuggestion,
) (*NominationSuggestion, error) {
	var inserted NominationSuggestion

	err := db.db.Get(
		&inserted,
		`INSERT INTO suggestions
			(category_id, text, updated_at, nominator_id)
		 VALUES (?, ?, ?, ?)
		 RETURNING *`,
		suggestion.CategoryID,
		suggestion.Text,
		suggestion.UpdatedAt,
		suggestion.NominatorID,
	)
	if err != nil {
		return nil, err
	}

	return &inserted, nil
}

func (db *DB) updateSuggestion(
	suggestion NominationSuggestion,
	userID int64,
) (*NominationSuggestion, error) {
	var updated NominationSuggestion

	err := db.db.Get(
		&updated,
		`UPDATE suggestions
		 SET category_id = ?,
		     text = ?,
		     updated_at = ?,
		     nominator_id = ?
		 WHERE id = ?
		   AND nominator_id = ?
		 RETURNING *`,
		suggestion.CategoryID,
		suggestion.Text,
		suggestion.UpdatedAt,
		suggestion.NominatorID,
		suggestion.ID,
		userID,
	)
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

func (db *DB) upsertSuggestion(
	suggestion NominationSuggestion,
	userID int64,
) (*NominationSuggestion, error) {
	if suggestion.ID == nil || *suggestion.ID == 0 {
		return db.addSuggestion(suggestion)
	}

	return db.updateSuggestion(suggestion, userID)
}

func (db *DB) deleteSuggestion(suggestionID int64) error {
	_, err := db.db.Exec(
		"DELETE FROM suggestions WHERE id = ?",
		suggestionID,
	)

	return err
}

func (db *DB) upsertSuggestions(
	suggestions []NominationSuggestion,
	categoryID int64,
	userID int64,
) error {
	keepSuggestionIDs := make([]int64, 0, len(suggestions))

	for _, suggestion := range suggestions {
		inserted, err := db.upsertSuggestion(suggestion, userID)
		if err != nil {
			return err
		}

		keepSuggestionIDs = append(
			keepSuggestionIDs,
			*inserted.ID,
		)
	}

	// Empty list means: delete every suggestion in this category.
	if len(keepSuggestionIDs) == 0 {
		_, err := db.db.Exec(
			"DELETE FROM suggestions WHERE category_id = ?",
			categoryID,
		)
		return err
	}

	// sqlx.In expands the slice into (?, ?, ?, ...).
	query, args, err := sqlx.In(
		`DELETE FROM suggestions
		 WHERE category_id = ?
		   AND id NOT IN (?)`,
		categoryID,
		keepSuggestionIDs,
	)
	if err != nil {
		return err
	}

	// We're using SQLite, so '?' is already the correct placeholder.
	_, err = db.db.Exec(query, args...)
	return err
}

func (db *DB) getSuggestionsOf(
	userID int64,
) ([]NominationSuggestion, error) {
	var suggestions []NominationSuggestion

	err := db.db.Select(
		&suggestions,
		"SELECT * FROM suggestions WHERE nominator_id = ?",
		userID,
	)
	if err != nil {
		return nil, err
	}

	return suggestions, nil
}

func (db *DB) createSession(userID int64) (string, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}

	hash := hashToken(token)
	ts := time.Now()

	_, err = db.db.Exec(
		`INSERT INTO sessions
			(token_hash, user_id, created_at, expires_at)
		 VALUES (?, ?, ?, ?)`,
		hash,
		userID,
		ts,
		ts.Add(24*time.Hour),
	)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (db *DB) validateSession(token string) (bool, error) {
	hash := hashToken(token)

	var session Session

	err := db.db.Get(
		&session,
		`SELECT *
		 FROM sessions
		 WHERE token_hash = ?
		   AND expires_at > ?`,
		hash,
		time.Now(),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (db *DB) deleteSession(token string) error {
	hash := hashToken(token)

	_, err := db.db.Exec(
		"DELETE FROM sessions WHERE token_hash = ?",
		hash,
	)

	return err
}

func (db *DB) getSession(token string) (Session, error) {
	var session Session

	err := db.db.Get(
		&session,
		`SELECT *
		 FROM sessions
		 WHERE token_hash = ?
		   AND expires_at > ?`,
		hashToken(token),
		time.Now(),
	)
	if err != nil {
		return Session{}, err
	}

	return session, nil
}
