package accuracy_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/cmd/speech_to_text/qa_align"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

/*
This test moves the position of a word in each test sentence.
The position the word is moved from should cause an ASR error, i.e. a word
in audio that is not in the text.  The position the word that is moved to
should cause an fa_error because the words is in the text, but not the audio.

This test should provide very accurate statistics about the accuracy
of the qa_align and proofing_rpt
*/

type wordSwitch struct {
	fromWord int
	toWord   int
}

func TestAccuracy(t *testing.T) {
	mediaId := "N1SKNSEC"
	runNum := "00004"
	var req request.Request
	req.DatasetName = mediaId
	req.Username = "GaryNTest"
	req.LanguageISO = "skn"
	req.Testament = request.Testament{NTBooks: []string{"PHM"}}
	req.Testament.BuildBookMaps()
	conn := downloadAndOpenDatabase(mediaId, runNum)
	verses := selectVersesWithoutFAError(conn, req.Testament, 0.5)
	var testCases = make(map[int64]wordSwitch)
	for _, vs := range verses {
		testWords := computeTwoRandoms(len(vs.Words))
		testCases[vs.ScriptId] = testWords
		moveFirstToSecond(vs, testWords)
	}
	status := storeAlteredData(conn, verses)
	if status != nil {
		exit(status)
	}
	_, status = qa_align.Process(conn, req)
	if status != nil {
		exit(status)
	}
	_, status = proofing_rpt.Process(conn, req)
	report := proofing_rpt.NewAlignSilence(conn)
	results, _, status := report.Process()
	if status != nil {
		exit(status)
	}
	checkResults(results, testCases)
}

func downloadAndOpenDatabase(mediaId string, runNum string) db.DBAdapter {
	objectKey := filepath.Join("GaryNTest", mediaId, "arti", runNum, "database", mediaId+".db")
	localPath := filepath.Join(os.Getenv("FCBH_DATASET_DB"), "GaryNTest", mediaId+".db")
	client, status := s3_datastore.NewS3Client(context.Background())
	if status != nil {
		exit(status)
	}
	// Remove database because it was altered in prior test.
	err := os.Remove(localPath)
	if err != nil {
		exit(err)
	}
	status = client.DownloadFile("arti-output", objectKey, localPath)
	if status != nil {
		exit(status)
	}
	ctx := context.Background()
	conn := db.NewDBAdapter(ctx, localPath)
	return conn
}

func selectVersesWithoutFAError(conn db.DBAdapter, books request.Testament, cutoff float64) []proofing_rpt.Verse2 {
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

func computeTwoRandoms(wordCnt int) wordSwitch {
	first := rand.IntN(wordCnt)
	second := rand.IntN(wordCnt)
	for first == second {
		second = rand.IntN(wordCnt)
	}
	return wordSwitch{fromWord: first, toWord: second}
}

func moveFirstToSecond(verse proofing_rpt.Verse2, tWds wordSwitch) {
	w := verse.Words[tWds.fromWord]
	if tWds.fromWord < tWds.toWord {
		// shift the gap left, closing the hole at `first`
		copy(verse.Words[tWds.fromWord:tWds.toWord], verse.Words[tWds.fromWord+1:tWds.toWord+1])
	} else {
		// shift the gap right
		copy(verse.Words[tWds.toWord+1:tWds.fromWord+1], verse.Words[tWds.toWord:tWds.fromWord])
	}
	verse.Words[tWds.toWord] = w
}

func storeAlteredData(conn db.DBAdapter, verses []proofing_rpt.Verse2) *log.Status {
	query := `UPDATE scripts set script_text = ? WHERE script_id = ?`
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
		var scriptWords []string
		for _, wd := range vs.Words {
			scriptWords = append(scriptWords, wd.Text)
			//if wd.Ttype != "W" {
			//	panic("Type: " + wd.Ttype)
			//}
		}
		scriptText := strings.Join(scriptWords, " ")
		_, err = stmt.Exec(scriptText, vs.ScriptId)
		if err != nil {
			return log.Error(conn.Ctx, 500, err, `Error while updating script text.`)
		}
	}
	err = tx.Commit()
	if err != nil {
		return log.Error(conn.Ctx, 500, err, "Error committing transaction for query:", query)
	}
	return nil
}

func checkResults(verses []proofing_rpt.Verse2, testCases map[int64]wordSwitch) {
	var foundMissing, foundAdded, total float64
	var errorMissing, errorAdded []generic.VerseRef
	for _, vs := range verses {
		testWords, ok := testCases[vs.ScriptId]
		if ok {
			displayVerseDetail(vs, testWords)
			total++
			fromWord := vs.Words[testWords.fromWord]
			if fromWord.Ttype == "ASR" {
				foundAdded++
			} else {
				errorAdded = append(errorAdded, vs.LineRef)
			}
			toWord := vs.Words[testWords.toWord]
			if toWord.FAScore < 0.1 {
				foundMissing++
			} else {
				errorMissing = append(errorMissing, vs.LineRef)
			}
		}
	}
	pctWasMissing := foundMissing / total * 100.0
	pctWasAdded := foundAdded / total * 100.0
	fmt.Printf("Total Processed: %0.f  Pct Was Missing %.1f Pct Was Added %.1f\n",
		total, pctWasMissing, pctWasAdded)
	fmt.Println("Not Found Missing:", errorMissing)
	fmt.Println("Not Found Added:", errorAdded)
}

func displayVerseDetail(verse proofing_rpt.Verse2, testCase wordSwitch) {
	fmt.Println(verse.ScriptId)
	for i, wd := range verse.Words {
		if i == testCase.fromWord {
			fmt.Print("FROM: ")
		} else if i == testCase.toWord {
			fmt.Print("TOWD: ")
		}
		fmt.Printf("%d  %s  %.2f  [", wd.WordId, wd.Text, wd.FAScore)
		for _, ch := range wd.Chars {
			fmt.Printf(" %s (%.2f)", string(ch.Char), ch.FAScore)
		}
		fmt.Println()
	}
}

func exit(err error) {
	fmt.Println("ERR", err)
	os.Exit(1)
}
