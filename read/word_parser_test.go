package read

import (
	"context"
	"fmt"
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/sergi/go-diff/diffmatchpatch"
	"strconv"
	"strings"
	"testing"
)

func TestWordParser(t *testing.T) {
	tests := []string{`ATIWBT_USXEDIT.db`} //,`ATIWBT_DBPTEXT.db`, `ATIWBT_SCRIPT.db`
	for _, test := range tests {
		testOneDatabase(test, t)
	}
}

func testOneDatabase(database string, t *testing.T) {
	ctx := context.Background()
	conn := db.NewDBAdapter(ctx, database)
	word := NewWordParser(conn)
	word.Parse()
	conn.Close()
	compareScriptAndWords(database, t)
}

func compareScriptAndWords(database string, t *testing.T) {
	var count = 0
	diffMatch := diffmatchpatch.New()
	ctx := context.Background()
	conn := db.NewDBAdapter(ctx, database)
	var words = make([]string, 0, 100)
	var records, status = conn.SelectScripts()
	if status != nil {
		t.Error(status)
	}
	for _, rec := range records {
		sql1 := `SELECT word FROM words WHERE script_id=?`
		rows, err := conn.DB.Query(sql1, rec.ScriptId)
		if err != nil {
			log.Fatal(ctx, err, sql1)
		}
		defer rows.Close()
		words = []string{}
		for rows.Next() {
			var word string
			err := rows.Scan(&word)
			if err != nil {
				log.Fatal(ctx, err, sql1)
			}
			words = append(words, word)
		}
		err = rows.Err()
		if err != nil {
			log.Fatal(ctx, err, sql1)
		}
		var wordText = strings.Join(words, ``)
		diffs := diffMatch.DiffMain(rec.ScriptText, wordText, false)
		if !isMatch(diffs) {
			ref := rec.BookId + " " + strconv.Itoa(rec.ChapterNum) + ":" + strconv.Itoa(rec.VerseNum)
			fmt.Println(ref, diffMatch.DiffPrettyText(diffs))
			fmt.Println("=============")
			count++
		}
	}
	if count > 0 {
		t.Error("The script and words did not match!, num Diffs ", count)
	}
}

func isMatch(diffs []diffmatchpatch.Diff) bool {
	for _, diff := range diffs {
		if diff.Type == diffmatchpatch.DiffInsert || diff.Type == diffmatchpatch.DiffDelete {
			if len(strings.TrimSpace(diff.Text)) > 0 {
				return false
			}
		}
	}
	return true
}
