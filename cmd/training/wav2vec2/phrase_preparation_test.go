package wav2vec2

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
)

func wordRow(scriptId int64, seq int, begin float64, end float64, fa float64, word string) phraseRow {
	return phraseRow{scriptId: scriptId, audioFile: "a.mp3", wordSeq: seq, beginTS: begin, endTS: end, faScore: fa, ttype: "W", word: word}
}

func punctRow(scriptId int64, seq int) phraseRow {
	return phraseRow{scriptId: scriptId, audioFile: "a.mp3", wordSeq: seq, ttype: "P", word: ","}
}

func TestGroupPhrases(t *testing.T) {
	rows := []phraseRow{
		wordRow(1, 1, 0, 1, 0.9, "one"),
		wordRow(1, 2, 1, 1, 0.1, "zerolength"),
		wordRow(1, 3, 1, 2, 0.5, "two"),
		punctRow(1, 4),
		punctRow(1, 5),
		wordRow(1, 6, 3, 4, 0.8, "three"),
		wordRow(2, 1, 5, 6, 0.7, "four"),
	}
	got := groupPhrases(rows)
	if len(got) != 3 {
		t.Fatalf("expected 3 phrases, got %+v", got)
	}
	first := got[0]
	if first.text != "one two" || first.faScore != 0.5 || first.firstWordSeq != 1 || first.lastWordSeq != 3 {
		t.Errorf("zero-length word should be left out of text and score: %+v", first)
	}
	if got[1].text != "three" || got[1].phraseNum != 2 || got[2].phraseNum != 1 {
		t.Errorf("phrase numbers should restart in each script: %+v", got)
	}
}

func TestMergeSingleWords(t *testing.T) {
	phrases := []phrase{
		{scriptId: 1, phraseNum: 1, firstWordSeq: 1, lastWordSeq: 2, beginTS: 0, endTS: 1, text: "a b", faScore: 0.9},
		{scriptId: 1, phraseNum: 2, firstWordSeq: 4, lastWordSeq: 4, beginTS: 1.1, endTS: 2, text: "c", faScore: 0.2}, // nearer before
		{scriptId: 1, phraseNum: 3, firstWordSeq: 6, lastWordSeq: 7, beginTS: 3, endTS: 4, text: "d e", faScore: 0.8},
		{scriptId: 2, phraseNum: 1, firstWordSeq: 1, lastWordSeq: 1, beginTS: 5, endTS: 6, text: "f", faScore: 0.6}, // nearer after
		{scriptId: 2, phraseNum: 2, firstWordSeq: 3, lastWordSeq: 4, beginTS: 6.1, endTS: 7, text: "g h", faScore: 0.7},
		{scriptId: 3, phraseNum: 1, firstWordSeq: 1, lastWordSeq: 1, beginTS: 8, endTS: 9, text: "i", faScore: 0.5}, // only word in script
	}
	got := mergeSingleWords(phrases)
	if len(got) != 4 {
		t.Fatalf("expected 4 phrases, got %+v", got)
	}
	if got[0].text != "a b c" || got[0].faScore != 0.2 || got[0].lastWordSeq != 4 || got[0].endTS != 2 {
		t.Errorf("merge to previous: %+v", got[0])
	}
	if got[1].text != "d e" || got[1].phraseNum != 2 {
		t.Errorf("renumbering: %+v", got[1])
	}
	if got[2].text != "f g h" || got[2].faScore != 0.6 || got[2].beginTS != 5 || got[2].firstWordSeq != 1 {
		t.Errorf("merge to next: %+v", got[2])
	}
	if got[3].text != "i" {
		t.Errorf("single-word script: %+v", got[3])
	}
}

func TestPadPhrases(t *testing.T) {
	phrases := []phrase{
		{audioFile: "a.mp3", beginTS: 0.1, endTS: 1},
		{audioFile: "a.mp3", beginTS: 1.1, endTS: 2},
		{audioFile: "a.mp3", beginTS: 5, endTS: 6},
		{audioFile: "b.mp3", beginTS: 1, endTS: 2},
	}
	got := padPhrases(phrases)
	if got[0].beginTS != 0 || got[0].endTS != 1.05 {
		t.Errorf("begin must stop at 0 and end at half the gap: %+v", got[0])
	}
	if got[1].beginTS != 1.05 || got[1].endTS != 2.2 {
		t.Errorf("full padding when the gap is large: %+v", got[1])
	}
	if got[2].endTS != 6.2 || got[3].beginTS != 0.8 {
		t.Errorf("another audio file is not a neighbour: %+v %+v", got[2], got[3])
	}
}

func TestPhrasePruners(t *testing.T) {
	ctx := context.Background()
	log.SetOutput("stderr")
	conn := db.NewDBAdapter(ctx, filepath.Join(t.TempDir(), "phrase.db"))
	defer conn.Close()
	exec := func(query string, args ...any) {
		_, err := conn.DB.Exec(query, args...)
		if err != nil {
			t.Fatal(query, err)
		}
	}
	addScript := func(scriptId int, verseStr string) {
		exec(`INSERT INTO scripts (script_id, dataset_id, book_id, chapter_num, chapter_end, verse_str, verse_end,
			verse_num, audio_file, script_num, script_text)
			VALUES (?, 1, 'MAT', 1, 1, ?, ?, 1, 'a.mp3', ?, '')`, scriptId, verseStr, verseStr, verseStr)
	}
	addWord := func(scriptId int, seq int, ttype string, begin float64, end float64, fa float64) {
		exec(`INSERT INTO words (script_id, word_seq, verse_num, ttype, word, word_begin_ts, word_end_ts, fa_score)
			VALUES (?,?,1,?,'w',?,?,?)`, scriptId, seq, ttype, begin, end, fa)
	}
	addScript(1, "0") // heading, always pruned
	addWord(1, 1, "W", 0, 1, 0.99)
	addScript(2, "1")
	addWord(2, 1, "W", 2, 3, 0.9)
	addWord(2, 2, "W", 3, 4, 0.1) // lowest score
	addWord(2, 3, "P", 0, 0, 0)
	addWord(2, 4, "W", 9, 10, 0.8) // 5 seconds of silence before this is between phrases
	addWord(2, 5, "W", 10, 11, 0.8)
	addScript(3, "2")
	addWord(3, 1, "W", 12, 13, 0.7)
	addWord(3, 2, "W", 15, 16, 0.7) // 2 seconds of silence within a phrase
	status := PreparePhrases(conn)
	if status != nil {
		t.Fatal(status)
	}
	selectIds := func(query string) []string {
		rows, err := conn.DB.Query(query)
		if err != nil {
			t.Fatal(query, err)
		}
		defer rows.Close()
		var ids []string
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
		return ids
	}
	status = PhraseDataPruner(conn, 34)
	if status != nil {
		t.Fatal(status)
	}
	query := `SELECT s.verse_str || '.' || p.phrase_num FROM pruned_data d
		JOIN phrases p ON p.phrase_id = d.phrase_id JOIN scripts s ON s.script_id = p.script_id ORDER BY 1`
	got := selectIds(query)
	if len(got) != 2 || got[0] != "0.1" || got[1] != "1.1" {
		t.Errorf("expected heading and lowest phrase of the other 3 pruned, got %v", got)
	}
	status = PhraseSilencePruner(conn, 1)
	if status != nil {
		t.Fatal(status)
	}
	query = `SELECT s.verse_str || '.' || p.phrase_num FROM pruned_silence d
		JOIN phrases p ON p.phrase_id = d.phrase_id JOIN scripts s ON s.script_id = p.script_id`
	got = selectIds(query)
	if len(got) != 1 || got[0] != "2.1" {
		t.Errorf("expected only the phrase with silence between its own words, got %v", got)
	}
}
