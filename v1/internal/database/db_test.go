package database

import (
	"testing"

	"github.com/user/spotify-stego-v1/internal/stego/core"
)

func TestDatabaseWorkflow(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	track := core.Track{
		ID:          "test-1",
		Title:       "Test Song",
		Artist:      "Test Artist",
		Genre:       "electronic",
		DurationMs:  210000,
		Tempo:       128.0,
		KeyOf:       "C",
		CamelotCode: "8B",
	}

	if err := UpsertTrack(db, track); err != nil {
		t.Fatalf("UpsertTrack failed: %v", err)
	}

	tracks, err := GetTracksByGenre(db, "electronic")
	if err != nil {
		t.Fatalf("GetTracksByGenre failed: %v", err)
	}

	if len(tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(tracks))
	}

	if tracks[0].ID != track.ID {
		t.Fatalf("expected ID %q, got %q", track.ID, tracks[0].ID)
	}
}
