package accuracy

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/db"
	log "github.com/artificial-polyglot/arti/logger"
	req "github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

/*
This test moves the position of a word in each test sentence.
The position the word is moved from should cause an ASR error, i.e. a word
in audio that is not in the text.  The position the word that is moved to
should cause an fa_error because the words is in the text, but not the audio.

This test should provide very accurate statistics about the accuracy
of the qa_align and proofing_rpt

func TestSetup in setup_test.go creates an accuracy_test database.
func TestRun in run_test.go create a script to run the test in runpod.io.
	The script must be manually started to avoid having automated tests start runpod jobs
func TestResults in results_test.go anaylzes the results of the run,
and produces a report of problems, and run statistics
*/

func TestSetup(t *testing.T) {
	tests := CasesForTest()
	storeTestCases(tests)
	for _, tst := range tests {
		if tst.On {
			Setup(tst)
		}
	}
}

func Setup(tst testCase) {
	random := rand.New(rand.NewPCG(12, 21)) // two uint64 seeds; fixed values → same sequence every run
	conn := downloadAndOpenDatabase(tst.SetupDBInput, tst.SetupDBLocal)
	fmt.Println("Database Path", conn.DatabasePath)
	verses := selectVersesWithoutFAError(conn, tst.Testament, 0.5)
	var wordSwitches = make(map[int64]wordSwitch)
	var origWordIds = make(map[int64][]int64)
	for _, vs := range verses {
		if len(vs.Words) > 12 {
			ids := make([]int64, len(vs.Words))
			for i, wd := range vs.Words {
				ids[i] = wd.WordId
			}
			origWordIds[vs.ScriptId] = ids
			testWords := computeTwoRandoms(random, len(vs.Words))
			moveFirstToSecond(vs, &testWords)
			wordSwitches[vs.ScriptId] = testWords
		}
	}
	storeWordSwitches(tst.MediaId, wordSwitches)
	status := storeAlteredData(conn, verses, origWordIds)
	if status != nil {
		exit(status)
	}
	upoadloadDatabase(conn, tst.SetupDBOutput)
}

func downloadAndOpenDatabase(s3Path string, localPath string) db.DBAdapter {
	client, status := s3_datastore.NewS3Client(context.Background())
	if status != nil {
		exit(status)
	}
	// Remove database because it was altered in prior test.
	err := os.Remove(localPath)
	if err != nil && !os.IsNotExist(err) {
		exit(err)
	}
	status = client.DownloadFile(OUTPUT_BUCKET, s3Path, localPath)
	if status != nil {
		exit(status)
	}
	ctx := context.Background()
	conn := db.NewDBAdapter(ctx, localPath)
	return conn
}

func selectVersesWithoutFAError(conn db.DBAdapter, books req.Testament, cutoff float64) []proofing_rpt.Verse2 {
	var result []proofing_rpt.Verse2
	var query = `SELECT s.script_id, s.book_id, s.chapter_num, s.verse_str,
          w.word_id, w.word, q.fa_score
          FROM words_qa_align q JOIN words w ON q.word_id = w.word_id
          JOIN scripts s ON s.script_id = w.script_id
		  WHERE s.book_id IN (%s)
          AND w.ttype = 'W' AND s.verse_str != '0' AND w.script_id IN (
                 SELECT DISTINCT w2.script_id
                 FROM words w2 JOIN words_qa_align q2 ON w2.word_id = q2.word_id
                 WHERE q2.fa_score >= ?)
             ORDER BY s.script_id, w.word_id`
	bookStr := "'" + strings.Join(books.NTBooks, "','") + "'"
	query = strings.Replace(query, "%s", bookStr, 1)
	rows, err := conn.DB.Query(query, cutoff)
	if err != nil {
		exit(err)
	}
	defer rows.Close()

	for rows.Next() {
		var vs proofing_rpt.Verse2
		var wd proofing_rpt.Word2
		err = rows.Scan(&vs.ScriptId, &vs.LineRef.BookId, &vs.LineRef.ChapterNum,
			&vs.LineRef.VerseStr, &wd.WordId, &wd.Text, &wd.FAScore)
		if err != nil {
			exit(err)
		}
		n := len(result)
		if n == 0 || result[n-1].ScriptId != vs.ScriptId {
			// first row of a new verse: seed it with this word and append it
			vs.Words = append(vs.Words, wd)
			result = append(result, vs)
		} else {
			// same verse as the previous row: just add the word
			result[n-1].Words = append(result[n-1].Words, wd)
		}
	}
	err = rows.Err()
	if err != nil {
		exit(err)
	}
	return result
}

func computeTwoRandoms(random *rand.Rand, wordCnt int) wordSwitch {
	first := random.IntN(wordCnt)
	second := random.IntN(wordCnt)
	for math.Abs(float64(first-second)) < 6 {
		second = rand.IntN(wordCnt)
	}
	return wordSwitch{FromWord: first, ToWord: second}
}

func moveFirstToSecond(verse proofing_rpt.Verse2, tWds *wordSwitch) {
	tWds.Word = verse.Words[tWds.FromWord].Text
	//	if tWds.FromWord < tWds.ToWord {
	//		tWds.ToWord -= 1
	//	}
	tWds.ToWordId = verse.Words[tWds.ToWord].WordId
	tWds.FromWordId = verse.Words[tWds.FromWord].WordId
	w := verse.Words[tWds.FromWord]
	if tWds.FromWord < tWds.ToWord {
		// shift the gap left, closing the hole at `first`
		copy(verse.Words[tWds.FromWord:tWds.ToWord], verse.Words[tWds.FromWord+1:tWds.ToWord+1])
	} else {
		// shift the gap right
		copy(verse.Words[tWds.ToWord+1:tWds.FromWord+1], verse.Words[tWds.ToWord:tWds.FromWord])
	}
	verse.Words[tWds.ToWord] = w
}

func storeWordSwitches(mediaId string, tests map[int64]wordSwitch) {
	bytes, err := json.MarshalIndent(tests, "", "  ")
	if err != nil {
		exit(err)
	}
	filePath := fmt.Sprintf(TEST_DATA, mediaId)
	err = os.WriteFile(filePath, bytes, 0644)
	if err != nil {
		exit(err)
	}
}

func retrieveWordSwitches(mediaId string) map[int64]wordSwitch {
	var result map[int64]wordSwitch
	filePath := fmt.Sprintf(TEST_DATA, mediaId)
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		exit(err)
	}
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		exit(err)
	}
	return result
}

func storeAlteredData(conn db.DBAdapter, verses []proofing_rpt.Verse2, origWordIds map[int64][]int64) *log.Status {
	query := `UPDATE words SET word_id = ? WHERE word_id = ?`
	tx, err := conn.DB.Begin()
	if err != nil {
		return log.Error(conn.Ctx, 500, err, query)
	}
	stmt, err := tx.Prepare(query)
	if err != nil {
		return log.Error(conn.Ctx, 500, err, query)
	}
	defer stmt.Close()
	for _, vs := range verses {
		slotIds, ok := origWordIds[vs.ScriptId]
		if !ok {
			continue
		}
		for _, wd := range vs.Words {
			_, err = stmt.Exec(-wd.WordId, wd.WordId)
			if err != nil {
				return log.Error(conn.Ctx, 500, err, `Error while staging temporary word_id.`)
			}
		}
		for i, wd := range vs.Words {
			_, err = stmt.Exec(slotIds[i], -wd.WordId)
			if err != nil {
				return log.Error(conn.Ctx, 500, err, `Error while assigning final word_id.`)
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		return log.Error(conn.Ctx, 500, err, "Error committing transaction for query:", query)
	}
	return nil
}

func upoadloadDatabase(conn db.DBAdapter, s3DBPath string) {
	dbPath := conn.DatabasePath
	ctx := conn.Ctx
	conn.Close()
	client, status := s3_datastore.NewS3Client(ctx)
	if status != nil {
		exit(status)
	}
	status = client.PutFile(OUTPUT_BUCKET, s3DBPath, dbPath, "application/x-sqlite3", false)
	if status != nil {
		exit(status)
	}
}

func exit(err error) {
	fmt.Println("ERR", err)
	panic(err)
}
