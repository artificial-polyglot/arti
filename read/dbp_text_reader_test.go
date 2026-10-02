package read

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/input"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

func TestDBPTextReader1(t *testing.T) {
	ctx := context.Background()
	bibleId := `ENGWEB`
	fsType := request.TextPlainEdit
	otFileset := `ENGWEBO_ET`
	ntFileset := `ENGWEBN_ET`
	testament := request.Testament{NTBooks: []string{`MAT`, `MRK`}, OTBooks: []string{`JOB`, `PSA`, `PRO`, `SNG`}}
	files, status := input.DBPDirectory(ctx, bibleId, fsType, otFileset, ntFileset) //, testament)
	if status != nil {
		t.Error(status)
	}
	var database = bibleId + `_DBPTEXT.db`
	db.DestroyDatabase(database)
	var db1 = db.NewDBAdapter(context.Background(), database)
	var req request.Request
	req.Testament = testament
	req.Testament.BuildBookMaps()
	textAdapter := NewDBPTextReader(db1, req.Testament)
	textAdapter.ProcessFiles(files)
	count, _ := db1.CountScriptRows()
	if count != 6312 {
		t.Error(`Script row count should be 1`, count)
	}
	db1.Close()
}

type TempRec struct {
	BookId     string `json:"book_id"`
	BookSeq    int
	ChapterNum int    `json:"chapter"`
	VerseStart int    `json:"verse_start"`
	VerseEnd   int    `json:"verse_end"`
	Text       string `json:"verse_text"`
}
type TempResp struct {
	Data []TempRec `json:"data"`
}

func TempTestTempLoadLaobible(t *testing.T) {
	ctx := context.Background()
	dbPath := "/Users/gary/arti2/laobible/laobible_net.db"
	conn := db.NewDBAdapter(ctx, dbPath)
	verses, status := SelectISANScript(conn)
	if status != nil {
		t.Fatal(status)
	}
	var err error
	var result TempResp
	for _, vs := range verses {
		var rec TempRec
		rec.BookId = vs.BookId
		rec.ChapterNum = vs.ChapterNum
		//parts := strings.Split(vs.VerseStr, "-")
		parts := strings.FieldsFunc(vs.VerseStr, func(r rune) bool { return r == '-' || r == '–' })
		rec.VerseStart, err = strconv.Atoi(parts[0])
		if err != nil {
			t.Fatal(status)
		}
		if len(parts) == 2 {
			rec.VerseEnd, err = strconv.Atoi(parts[1])
			if err != nil {
				t.Fatal(status)
			}
		}
		if len(parts) > 2 {
			t.Fatal("Verses has too many parts", vs.VerseStr)
		}
		rec.Text = strings.TrimSpace(vs.ScriptText)
		result.Data = append(result.Data, rec)
	}
	bytes, err1 := json.MarshalIndent(result, "", "  ")
	if err1 != nil {
		t.Fatal(err1)
	}
	err = os.WriteFile("LAO_BIBLE.json", bytes, 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func SelectISANScript(conn db.DBAdapter) ([]db.Script, *log.Status) {
	var results []db.Script
	query := `SELECT script_id, book_id, chapter_num, verse_str, isan_tts_text
		FROM scripts WHERE isan_tts_text != '' ORDER BY script_id`
	rows, err := conn.DB.Query(query)
	if err != nil {
		return results, log.Error(conn.Ctx, 500, err, "Error during select scripts")
	}
	defer rows.Close()
	for rows.Next() {
		var rec db.Script
		err = rows.Scan(&rec.ScriptId, &rec.BookId, &rec.ChapterNum, &rec.VerseStr, &rec.ScriptText)
		if err != nil {
			return results, log.Error(conn.Ctx, 500, err, "Error in SelectScripts.")
		}
		results = append(results, rec)
	}
	err = rows.Err()
	if err != nil {
		log.Warn(conn.Ctx, err, query)
	}
	return results, nil
}
