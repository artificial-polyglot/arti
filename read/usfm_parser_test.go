package read

import (
	"context"
	"fmt"
	"testing"

	"github.com/faithcomesbyhearing/fcbh-dataset-io/db"
	"github.com/faithcomesbyhearing/fcbh-dataset-io/input"
)

func TestUSFMParser(t *testing.T) {
	ctx := context.Background()
	var directory = "test_data"
	var files []input.InputFile
	file1 := input.InputFile{BookId: "", Directory: directory, Filename: "43LUKCFM.SFM", MediaType: "text/plain"}
	//file2 := input.InputFile{BookId: "", Directory: directory, Filename: "43LUKDWK.SFM", MediaType: "text/plain"}
	files = append(files, file1)
	var database = directory + "/usfm_test.db"
	db.DestroyDatabase(database)
	var conn = db.NewDBAdapter(ctx, database)
	parser := NewUSFMParser(conn)
	status := parser.ProcessFiles(files)
	if status != nil {
		t.Fatal(status)
	}
	selectScripts(conn, t)
	conn.Close()
}

func selectScripts(conn db.DBAdapter, t *testing.T) {
	query := `SELECT script_id, book_id, chapter_num, verse_num, verse_str, usfm_style, script_text 
		FROM scripts ORDER BY script_id`
	rows, err := conn.DB.Query(query)
	if err != nil {
		t.Error(err)
	}
	defer rows.Close()
	for rows.Next() {
		var rec db.Script
		err = rows.Scan(&rec.ScriptId, &rec.BookId, &rec.ChapterNum,
			&rec.VerseNum, &rec.VerseStr, &rec.UsfmStyle, &rec.ScriptText)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(rec.ChapterNum, rec.VerseStr, "usfm:", rec.UsfmStyle, "[", rec.ScriptText, "]")
	}
	err = rows.Err()
	if err != nil {
		t.Error(err)
	}
}
