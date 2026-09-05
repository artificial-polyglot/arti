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
	// movedWordId is the word_id the moved word will carry in the database
	// once storeAlteredData relabels word_id to match the new arrangement -
	// i.e. the *original* word_id of the toWord slot, not the fromWord slot.
	// See storeAlteredData for why the moved word ends up wearing that id.
	movedWordId int64
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
	var origWordIds = make(map[int64][]int64)
	for _, vs := range verses {
		if len(vs.Words) > 1 {
			ids := make([]int64, len(vs.Words))
			for i, wd := range vs.Words {
				ids[i] = wd.WordId
			}
			testWords := computeTwoRandoms(len(vs.Words))
			testWords.movedWordId = vs.Words[testWords.toWord].WordId
			testCases[vs.ScriptId] = testWords
			origWordIds[vs.ScriptId] = ids
			moveFirstToSecond(vs, testWords)
		}
	}
	status := storeAlteredData(conn, verses, origWordIds)
	if status != nil {
		exit(status)
	}
	_, status = qa_align.Process(conn, req)
	if status != nil {
		exit(status)
	}
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
	if err != nil && os.IsNotExist(err) {
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

// storeAlteredData applies each verse's word rearrangement to the words
// table itself, by relabeling word_id - the column qa_align's selectWords
// and proofing_rpt's SelectFACharTimestamps actually order by (see
// fa_results.go and db_adapter.go). Every other column - word, word_punct,
// ttype, timestamps, fa_score - stays attached to its row and travels with
// it automatically, so Word2's {WordId, Text, FAScore} is all this needs;
// there's no reason to delete and reinsert rows, and no other column ever
// has to be read or rewritten.
//
// word_id is the PRIMARY KEY, so swapping two rows' ids directly would
// collide mid-statement. Each affected row is staged at a temporary negative
// id first (guaranteed free, since AUTOINCREMENT ids are never negative),
// then assigned its final id in a second pass. The final id for the row now
// sitting at position i is origWordIds[scriptId][i] - the id that slot held
// before the shuffle - which is exactly how every downstream ORDER BY
// word_id query will reconstruct the new sequence.
//
// The ttype='W' rows referenced by the ingest-time chars/word_mfcc tables
// are not touched or read anywhere in the qa_align/proofing_rpt pipeline
// this test exercises, so relabeling word_id here does not need to update
// them too.
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

// checkResults verifies two independent signals per verse:
//   - "added": qa_align/proofing_rpt spliced in a synthetic ASR word somewhere
//     in the verse, meaning it detected audio content unaccounted for by the
//     text. That synthetic word has no WordId of its own (see
//     align_compare.go InsertASRSilenceChars), so its presence anywhere in the
//     verse is the only thing that can be checked - not its position, since
//     that position shifts every later word's index in vs.Words and isn't
//     something a fixed offset computed before processing can predict.
//   - "missing": the word that was moved (identified by its original WordId,
//     which travels with it through the move) should now score a low fa_score,
//     since it no longer matches the audio at its new position. Looking it up
//     by WordId - rather than by the pre-move index into vs.Words - keeps this
//     check correct even when an ASR splice earlier in the verse has shifted
//     every subsequent word's index.
func checkResults(verses []proofing_rpt.Verse2, testCases map[int64]wordSwitch) {
	var foundMissing, foundAdded, total float64
	var errorMissing, errorAdded []generic.VerseRef
	for _, vs := range verses {
		testWords, ok := testCases[vs.ScriptId]
		if ok {
			displayVerseDetail(vs, testWords)
			total++
			if hasASRWord(vs) {
				foundAdded++
			} else {
				errorAdded = append(errorAdded, vs.LineRef)
			}
			movedWord, found := findWordById(vs, testWords.movedWordId)
			if found && movedWord.FAScore < 0.1 {
				foundMissing++
			} else {
				errorMissing = append(errorMissing, vs.LineRef)
			}
		}
	}
	if total > 0 {
		pctWasMissing := foundMissing / total * 100.0
		pctWasAdded := foundAdded / total * 100.0
		fmt.Printf("Total Processed: %0.f  Pct Was Missing %.1f Pct Was Added %.1f\n",
			total, pctWasMissing, pctWasAdded)
		fmt.Println("Not Found Missing:", errorMissing)
		fmt.Println("Not Found Added:", errorAdded)
	} else {
		fmt.Println("No test results")
	}
}

func hasASRWord(verse proofing_rpt.Verse2) bool {
	for _, wd := range verse.Words {
		if wd.Ttype == "ASR" {
			return true
		}
	}
	return false
}

func findWordById(verse proofing_rpt.Verse2, wordId int64) (proofing_rpt.Word2, bool) {
	for _, wd := range verse.Words {
		if wd.WordId == wordId {
			return wd, true
		}
	}
	return proofing_rpt.Word2{}, false
}

func displayVerseDetail(verse proofing_rpt.Verse2, testCase wordSwitch) {
	fmt.Println(verse.LineRef.Description(), verse.ScriptId)
	for _, wd := range verse.Words {
		if wd.WordId == testCase.movedWordId {
			fmt.Print("MOVED: ")
		} else if wd.Ttype == "ASR" {
			fmt.Print("ASR: ")
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
