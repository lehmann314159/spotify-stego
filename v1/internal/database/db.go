package database

import (
	"database/sql"
	"fmt"

	"github.com/user/spotify-stego-v1/internal/stego/core"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	if err := Migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	return db, nil
}

func UpsertTrack(db *sql.DB, t core.Track) error {
	_, err := db.Exec(
		`INSERT OR REPLACE INTO tracks (id, title, artist, genre, duration_ms, tempo, key_of, camelot_code)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Title, t.Artist, t.Genre, t.DurationMs, t.Tempo, t.KeyOf, t.CamelotCode,
	)
	return err
}

func GetTracksByGenre(db *sql.DB, genre string) ([]core.Track, error) {
	rows, err := db.Query(
		`SELECT id, title, artist, genre, duration_ms, tempo, key_of, camelot_code
		 FROM tracks WHERE genre = ?`,
		genre,
	)
	if err != nil {
		return nil, fmt.Errorf("query tracks by genre: %w", err)
	}
	defer rows.Close()

	var tracks []core.Track
	for rows.Next() {
		var t core.Track
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Genre, &t.DurationMs, &t.Tempo, &t.KeyOf, &t.CamelotCode); err != nil {
			return nil, fmt.Errorf("scan track row: %w", err)
		}
		tracks = append(tracks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate track rows: %w", err)
	}

	return tracks, nil
}
