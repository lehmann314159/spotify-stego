package database

import "database/sql"

const schemaSQL = `
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

CREATE INDEX IF NOT EXISTS idx_genre_camelot ON tracks (genre, camelot_code);
CREATE INDEX IF NOT EXISTS idx_genre_tempo   ON tracks (genre, tempo);
`

func Migrate(db *sql.DB) error {
	_, err := db.Exec(schemaSQL)
	return err
}
