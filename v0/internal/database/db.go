package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS tracks (
	id           TEXT PRIMARY KEY,
	title        TEXT NOT NULL,
	artist       TEXT NOT NULL,
	genre        TEXT NOT NULL,
	duration_ms  INTEGER,
	tempo        REAL,
	key_of       TEXT,
	camelot_code TEXT
);

CREATE INDEX IF NOT EXISTS idx_genre_camelot ON tracks(genre, camelot_code);
CREATE INDEX IF NOT EXISTS idx_genre_tempo   ON tracks(genre, tempo);
`

// Open opens (or creates) the SQLite database at the given path and applies the schema.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return db, nil
}

// Track mirrors the tracks table row.
type Track struct {
	ID          string
	Title       string
	Artist      string
	Genre       string
	DurationMS  int
	Tempo       float64
	KeyOf       string
	CamelotCode string
}

// UpsertTrack inserts or replaces a track row.
func UpsertTrack(db *sql.DB, t Track) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO tracks
			(id, title, artist, genre, duration_ms, tempo, key_of, camelot_code)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Title, t.Artist, t.Genre, t.DurationMS, t.Tempo, t.KeyOf, t.CamelotCode,
	)
	return err
}

// GetTracksByGenre returns all tracks for a genre.
func GetTracksByGenre(db *sql.DB, genre string) ([]Track, error) {
	rows, err := db.Query(`
		SELECT id, title, artist, genre, duration_ms, tempo, key_of, camelot_code
		FROM tracks WHERE genre = ?`, genre)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tracks []Track
	for rows.Next() {
		var t Track
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Genre, &t.DurationMS, &t.Tempo, &t.KeyOf, &t.CamelotCode); err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}
