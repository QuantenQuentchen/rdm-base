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

var (
	ErrCategoryNotFound   = errors.New("category not found")
	ErrCategoryLocked     = errors.New("category is locked")
	ErrTooManySuggestions = errors.New("too many suggestions for this category")
	ErrSuggestionNotFound = errors.New("suggestion not found")
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

	// _txlock=immediate makes every BEGIN take the write lock up front, so a
	// transaction that reads and then writes can't fail with SQLITE_BUSY when
	// another writer commits in between. busy_timeout covers the waiting.
	db.db, err = sqlx.Open("sqlite", "file:data.db?_txlock=immediate")
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

// withTx runs fn inside a transaction (BEGIN IMMEDIATE, via _txlock).
// It commits if fn returns nil and rolls back otherwise.
func (db *DB) withTx(fn func(tx *sqlx.Tx) error) error {
	tx, err := db.db.Beginx()
	if err != nil {
		return err
	}
	// After a successful Commit this is a harmless ErrTxDone.
	defer func() { _ = tx.Rollback() }()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
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

// getOrCreateUser is a single atomic statement, so concurrent first logins
// for the same Discord ID can't race. The no-op DO UPDATE is what makes
// RETURNING yield the existing row on conflict.
func (db *DB) getOrCreateUser(discordID string) (*User, error) {
	var user User

	err := db.db.Get(
		&user,
		`INSERT INTO users (discord_id) VALUES (?)
		 ON CONFLICT(discord_id) DO UPDATE SET discord_id = excluded.discord_id
		 RETURNING *`,
		discordID,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
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

// ---------------------------------------------------------------------------
// Suggestion helpers. These take a sqlx.Ext so they work on either *sqlx.DB or
// *sqlx.Tx, and are meant to be called inside withTx.
// ---------------------------------------------------------------------------

func getCategory(q sqlx.Ext, categoryID int64) (*Category, error) {
	var category Category

	err := sqlx.Get(q, &category, "SELECT * FROM categories WHERE id = ?", categoryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}

	return &category, nil
}

// insertSuggestion always attributes the row to userID, never to whatever the
// client put in NominatorID.
func insertSuggestion(
	q sqlx.Ext,
	s NominationSuggestion,
	userID int64,
) (*NominationSuggestion, error) {
	var inserted NominationSuggestion

	err := sqlx.Get(
		q,
		&inserted,
		`INSERT INTO suggestions
			(category_id, text, updated_at, nominator_id)
		 VALUES (?, ?, ?, ?)
		 RETURNING *`,
		s.CategoryID,
		s.Text,
		s.UpdatedAt,
		userID,
	)
	if err != nil {
		return nil, err
	}

	return &inserted, nil
}

// updateSuggestion only touches text/updated_at. It can't reassign the owner
// or move the suggestion to another category (which would dodge limits/locks).
func updateSuggestion(
	q sqlx.Ext,
	s NominationSuggestion,
	userID int64,
) (*NominationSuggestion, error) {
	var updated NominationSuggestion

	err := sqlx.Get(
		q,
		&updated,
		`UPDATE suggestions
		 SET text = ?,
		     updated_at = ?
		 WHERE id = ?
		   AND nominator_id = ?
		   AND category_id = ?
		 RETURNING *`,
		s.Text,
		s.UpdatedAt,
		*s.ID,
		userID,
		s.CategoryID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSuggestionNotFound
		}
		return nil, err
	}

	return &updated, nil
}

func saveSuggestion(
	q sqlx.Ext,
	s NominationSuggestion,
	userID int64,
) (*NominationSuggestion, error) {
	if s.ID == nil || *s.ID == 0 {
		return insertSuggestion(q, s, userID)
	}

	return updateSuggestion(q, s, userID)
}

// upsertSuggestion saves a single suggestion for userID, enforcing the
// locked flag and max_suggestions (per user, per category) atomically.
func (db *DB) upsertSuggestion(
	suggestion NominationSuggestion,
	userID int64,
) (*NominationSuggestion, error) {
	var result *NominationSuggestion

	err := db.withTx(func(tx *sqlx.Tx) error {
		category, err := getCategory(tx, suggestion.CategoryID)
		if err != nil {
			return err
		}
		if category.Locked {
			return ErrCategoryLocked
		}

		isNew := suggestion.ID == nil || *suggestion.ID == 0
		if isNew && category.MaxSuggestions != nil {
			var count int
			err := sqlx.Get(
				tx,
				&count,
				`SELECT COUNT(*) FROM suggestions
				 WHERE category_id = ? AND nominator_id = ?`,
				category.ID,
				userID,
			)
			if err != nil {
				return err
			}
			if count >= *category.MaxSuggestions {
				return ErrTooManySuggestions
			}
		}

		suggestion.UpdatedAt = time.Now()

		result, err = saveSuggestion(tx, suggestion, userID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// deleteSuggestion only deletes userID's own suggestion, and only while its
// category is unlocked.
func (db *DB) deleteSuggestion(suggestionID int64, userID int64) error {
	res, err := db.db.Exec(
		`DELETE FROM suggestions
		 WHERE id = ?
		   AND nominator_id = ?
		   AND category_id IN (SELECT id FROM categories WHERE locked = 0)`,
		suggestionID,
		userID,
	)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrSuggestionNotFound
	}

	return nil
}

// upsertSuggestions replaces userID's suggestions in categoryID with the given
// list, atomically. Other users' suggestions are never touched.
func (db *DB) upsertSuggestions(
	suggestions []NominationSuggestion,
	categoryID int64,
	userID int64,
) error {
	return db.withTx(func(tx *sqlx.Tx) error {
		category, err := getCategory(tx, categoryID)
		if err != nil {
			return err
		}
		if category.Locked {
			return ErrCategoryLocked
		}
		if category.MaxSuggestions != nil && len(suggestions) > *category.MaxSuggestions {
			return ErrTooManySuggestions
		}

		now := time.Now()
		keepIDs := make([]int64, 0, len(suggestions))

		for _, s := range suggestions {
			// Never trust these from the client.
			s.CategoryID = categoryID
			s.UpdatedAt = now

			saved, err := saveSuggestion(tx, s, userID)
			if err != nil {
				return err
			}

			keepIDs = append(keepIDs, *saved.ID)
		}

		// Empty list means: delete all of *this user's* suggestions in this category.
		if len(keepIDs) == 0 {
			_, err := tx.Exec(
				`DELETE FROM suggestions
				 WHERE category_id = ?
				   AND nominator_id = ?`,
				categoryID,
				userID,
			)
			return err
		}

		// sqlx.In expands the slice into (?, ?, ?, ...).
		query, args, err := sqlx.In(
			`DELETE FROM suggestions
			 WHERE category_id = ?
			   AND nominator_id = ?
			   AND id NOT IN (?)`,
			categoryID,
			userID,
			keepIDs,
		)
		if err != nil {
			return err
		}

		_, err = tx.Exec(query, args...)
		return err
	})
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
