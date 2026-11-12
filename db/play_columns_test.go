package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

// TestPlayColumnsAdded opens a database whose scripts table predates play_begin_ts and
// play_end_ts, and checks that they are added, and that UpdateScriptFATimestamps fills them.
func TestPlayColumnsAdded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	old, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = old.Exec(`CREATE TABLE scripts (script_id INTEGER PRIMARY KEY AUTOINCREMENT,
		dataset_id INTEGER NOT NULL, book_id TEXT NOT NULL, chapter_num INTEGER NOT NULL,
		chapter_end INTEGER NOT NULL, verse_str TEXT NOT NULL, verse_end TEXT NOT NULL,
		verse_num INTEGER NOT NULL, audio_file TEXT NOT NULL, script_num TEXT NOT NULL,
		usfm_style TEXT NOT NULL DEFAULT '', person TEXT NOT NULL DEFAULT '', actor TEXT NOT NULL DEFAULT '',
		script_text TEXT NOT NULL, uroman TEXT NOT NULL DEFAULT '',
		script_begin_ts REAL NOT NULL DEFAULT 0.0, script_end_ts REAL NOT NULL DEFAULT 0.0,
		fa_score REAL NOT NULL DEFAULT 0.0) STRICT`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = old.Exec(`INSERT INTO scripts (script_id, dataset_id, book_id, chapter_num, chapter_end, verse_str,
		verse_end, verse_num, audio_file, script_num, script_text) VALUES
		(1, 1, 'MAT', 1, 1, '1', '1', 1, 'a.mp3', '1', ''), (2, 1, 'MAT', 1, 1, '2', '2', 2, 'a.mp3', '2', '')`)
	if err != nil {
		t.Fatal(err)
	}
	old.Close()
	conn := NewDBAdapter(context.Background(), path)
	defer conn.Close()
	status := conn.UpdateScriptFATimestamps([]Audio{
		{ScriptId: 1, AudioFile: "a.mp3", BeginTS: 0.3, EndTS: 2.2, PlayBeginTS: 0, PlayEndTS: 2.5},
		{ScriptId: 2, AudioFile: "a.mp3", BeginTS: 2.8, EndTS: 4.2}, // no play times computed
	})
	if status != nil {
		t.Fatal(status)
	}
	rows, err := conn.DB.Query(`SELECT play_begin_ts, play_end_ts FROM scripts ORDER BY script_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got [][2]float64
	for rows.Next() {
		var b, e float64
		if err = rows.Scan(&b, &e); err != nil {
			t.Fatal(err)
		}
		got = append(got, [2]float64{b, e})
	}
	if len(got) != 2 || got[0] != [2]float64{0, 2.5} || got[1] != [2]float64{2.8, 4.2} {
		t.Errorf("play times: %v", got)
	}
}
