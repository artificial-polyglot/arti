package accuracy

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/utility/diff"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
)

func TestResults(t *testing.T) {
	ctx := context.Background()
	tests := retrieveTestCases()
	for _, tst := range tests {
		if tst.On {
			accuracy := NewAccuracyTest(ctx, tst)
			accuracy.AccuracyCheck()
		}
	}
}

type AccuracyTest struct {
	ctx     context.Context
	test    testCase
	conn    db.DBAdapter
	silence proofing_rpt.AlignSilence
}

func NewAccuracyTest(ctx context.Context, test testCase) AccuracyTest {
	var a AccuracyTest
	a.ctx = ctx
	a.test = test
	a.conn = LoadDatabase(ctx, test.ResultsDBInput, test.ResultsDBLocal)
	a.silence = proofing_rpt.NewAlignSilence(a.conn)
	return a
}

func (a *AccuracyTest) AccuracyCheck() {
	verses, _, status := a.silence.Process()
	if status != nil {
		exit(status)
	}
	outFile, err := os.Create(a.test.MediaId + ".txt")
	if err != nil {
		exit(err)
	}
	a.checkMissingWordResults(outFile, verses)
	a.checkAddedWordResults(outFile, verses)
	_ = outFile.Close()
}

func LoadDatabase(ctx context.Context, resultsDBInput string, resultsDBLocal string) db.DBAdapter {
	client, status := s3_datastore.NewS3Client(ctx)
	if status != nil {
		exit(status)
	}
	status = client.DownloadFile(OUTPUT_BUCKET, resultsDBInput, resultsDBLocal)
	if status != nil {
		exit(status)
	}
	conn := db.NewDBAdapter(ctx, resultsDBLocal)
	return conn
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
//     every subsequent word's index. When the move lands the word directly
//     next to another instance of itself, either of the two identical,
//     adjacent words may end up carrying the low score, so a flagged word
//     next to the WordId-matching one counts too (see isAdjacentToTargetWord).
func (a *AccuracyTest) checkMissingWordResults(outFile *os.File, verses []proofing_rpt.Verse2) {
	testCases := retrieveWordSwitches(a.test.MediaId)
	var foundMissing, foundFalse, foundNot, total float64
	for _, vs := range verses {
		testWords, ok := testCases[vs.ScriptId]
		if ok {
			isFound := false
			total++
			for j, wd := range vs.Words {
				if wd.FAScore <= 0 { //1 {
					if wd.WordId == testWords.ToWordId {
						isFound = true
						foundMissing++
						//a.displayMissingError(os.Stdout, "IS FOUND", vs, wd, testWords)
					} else if isAdjacentToTargetWord(vs.Words, j, testWords.ToWordId) {
						isFound = true
						foundMissing++
						//a.displayMissingError(os.Stdout, "IS FOUND", vs, wd, testWords)
					} else {
						foundFalse++
						a.displayMissingError(outFile, "MISSING FALSE+", vs, wd, testWords)
					}
				}
			}
			if !isFound {
				foundNot++
				a.displayMissingError(os.Stdout, "NOT FOUND MISS", vs, proofing_rpt.Word2{}, testWords)
				a.displayMissingError(outFile, "NOT FOUND MISS", vs, proofing_rpt.Word2{}, testWords)
			}
		}
	}
	if total > 0 {
		pctWasMissing := foundMissing / total * 100.0
		pctFoundFalse := foundFalse / total * 100.0
		pctFoundNot := foundNot / total * 100.0
		fmt.Printf("\n*** %s Total Processed: %0.f  Pct Was Missing %.1f Pct Found False+ %.1f  Pct Not Found %1.f\n",
			a.test.MediaId, total, pctWasMissing, pctFoundFalse, pctFoundNot)
	} else {
		fmt.Println("No Missing Word test results")
	}
}

// isAdjacentToTargetWord handles the case where the moved word lands right
// next to another instance of the same word (e.g. "the the"). When that
// happens, it's ambiguous which of the two identical, adjacent words the
// alignment should flag as low-scoring, so a flagged word is also accepted
// when its neighbor carries the expected WordId and has the same text.
func isAdjacentToTargetWord(words []proofing_rpt.Word2, idx int, targetWordId int64) bool {
	wd := words[idx]
	text := strings.ToLower(strings.TrimSpace(wd.Text))
	if idx > 0 {
		prev := words[idx-1]
		if prev.WordId == targetWordId && strings.ToLower(strings.TrimSpace(prev.Text)) == text {
			return true
		}
	}
	if idx < len(words)-1 {
		next := words[idx+1]
		if next.WordId == targetWordId && strings.ToLower(strings.TrimSpace(next.Text)) == text {
			return true
		}
	}
	return false
}

func (a *AccuracyTest) displayMissingError(outFile *os.File, missingResult string, vs proofing_rpt.Verse2, wd proofing_rpt.Word2, tst wordSwitch) {
	var text []string
	for _, wd2 := range vs.Words {
		text = append(text, wd2.Text)
	}
	_, _ = fmt.Fprintf(outFile, "\n%s  %d  %s\n", vs.LineRef.Description(), vs.ScriptId, strings.Join(text, " "))
	transcript, status := a.silence.SelectTranscript(vs.ScriptId)
	if status != nil {
		exit(status)
	}
	refText := a.SelectScriptLine(vs.ScriptId)
	dif := diff.DiffReplace(strings.ToLower(refText), transcript)
	_, _ = fmt.Fprintf(outFile, "REF Script Txt: %s\n", refText)
	_, _ = fmt.Fprintf(outFile, "ASR Transcript: %s\n", transcript)
	_, _ = fmt.Fprintln(outFile, "Diff", dif)
	_, _ = fmt.Fprintf(outFile, "%s Add: %d (%d) Missing: %d (%d)\n", tst.Word, tst.FromWord, tst.FromWordId, tst.ToWord, tst.ToWordId)
	_, _ = fmt.Fprintf(outFile, "%s  %d  %s  %.3f  \n", missingResult, wd.WordId, wd.Text, wd.FAScore)
	for i, wd2 := range vs.Words {
		_, _ = fmt.Fprintf(outFile, "%d %s: (%.3f) %d [", i, wd2.Text, wd2.FAScore, wd2.WordId)
		for _, ch := range wd2.Chars {
			_, _ = fmt.Fprintf(outFile, " %s (%.3f)", string(ch.Char), ch.FAScore)
		}
		_, _ = fmt.Fprintln(outFile, "]")
	}
}

func (a *AccuracyTest) checkAddedWordResults(outFile *os.File, verses []proofing_rpt.Verse2) {
	testCases := retrieveWordSwitches(a.test.MediaId)
	var foundMissing, foundFalse, foundNot, total float64
	for _, vs := range verses {
		testWords, ok := testCases[vs.ScriptId]
		if ok {
			total++
			nonInsertedWordCount := 0
			var nonInserted = make([]int, len(vs.Words))
			var verseHasResult bool
			for i, wd := range vs.Words {
				nonInserted[i] = nonInsertedWordCount
				if wd.IsASR {
					// This is not checking that the expected word is present, because ASR might put something
					// unexpected. It is checking that something has been added to the expected place.
					// The nonInsertedWordCount is intended to step over false positives that have been added
					if testWords.FromWord <= nonInsertedWordCount &&
						nonInsertedWordCount <= testWords.FromWord+2 {
						foundMissing++
						verseHasResult = true
						//a.displayAdded(os.Stdout, nonInserted, "IS ADDED", vs, wd, testWords)
					} else {
						foundFalse++
						a.displayAdded(outFile, nonInserted, "ADDED FALSE+", vs, wd, testWords)
					}
				} else {
					nonInsertedWordCount++
				}
			}
			if !verseHasResult {
				a.displayAdded(os.Stdout, nonInserted, "NOTHING ADDED", vs, proofing_rpt.Word2{}, testWords)
				a.displayAdded(outFile, nonInserted, "NOTHING ADDED", vs, proofing_rpt.Word2{}, testWords)
			}
		}
	}
	if total > 0 {
		pctWasMissing := foundMissing / total * 100.0
		pctFoundFalse := foundFalse / total * 100.0
		pctFoundNot := foundNot / total * 100.0
		fmt.Printf("\n*** Total Processed: %0.f  Pct Was Added %.1f Pct Found False+ %.1f  Pct Not Found %1.f\n",
			total, pctWasMissing, pctFoundFalse, pctFoundNot)
	} else {
		fmt.Println("No Added Word test results")
	}
}

func (a *AccuracyTest) displayAdded(outFile *os.File, nonInserted []int, addedResult string, vs proofing_rpt.Verse2, wd proofing_rpt.Word2, tst wordSwitch) {
	var text []string
	for _, wd2 := range vs.Words {
		text = append(text, wd2.Text)
	}
	_, _ = fmt.Fprintf(outFile, "\n%s  %d  %s\n", vs.LineRef.Description(), vs.ScriptId, strings.Join(text, " "))
	transcript, status := a.silence.SelectTranscript(vs.ScriptId)
	if status != nil {
		exit(status)
	}
	refText := a.SelectScriptLine(vs.ScriptId)
	dif := diff.DiffReplace(strings.ToLower(refText), transcript)
	_, _ = fmt.Fprintf(outFile, "REF Script Txt: %s\n", refText)
	_, _ = fmt.Fprintf(outFile, "ASR Transcript: %s\n", transcript)
	_, _ = fmt.Fprintln(outFile, "Diff", dif)
	_, _ = fmt.Fprintf(outFile, "%s Add: %d (%d) Missing: %d (%d)\n", tst.Word, tst.FromWord, tst.FromWordId, tst.ToWord, tst.ToWordId)
	_, _ = fmt.Fprintf(outFile, "%s  %d  %s  %.3f  \n", addedResult, wd.WordId, wd.Text, wd.FAScore)
	for i, wd2 := range vs.Words {
		_, _ = fmt.Fprintf(outFile, "%d %d %s: (%.3f) %d [", nonInserted[i], i, wd2.Text, wd2.FAScore, wd2.WordId)
		for _, ch := range wd2.Chars {
			_, _ = fmt.Fprintf(outFile, " %s (%.3f, %.3f, %d)", string(ch.Char), ch.FAScore, ch.Silence, ch.SilenceLong)
		}
		_, _ = fmt.Fprintln(outFile, "]")
	}
}

func (a *AccuracyTest) SelectScriptLine(scriptId int64) string {
	var text string
	var query = `SELECT GROUP_CONCAT(w.word, ' ' ORDER BY w.word_id) AS text
			FROM scripts s JOIN words w ON w.script_id = s.script_id
			WHERE s.script_id = ? AND w.ttype = 'W'
			GROUP BY s.script_id`
	row := a.conn.DB.QueryRow(query, scriptId)
	err := row.Scan(&text)
	if err != nil {
		exit(err)
	}
	return text
}

func TestDisplayDifferences(t *testing.T) {
	ctx := context.Background()
	// Set this path to database created by accuracy_test
	databasePath := "/Users/gary/FCBH2024/GaryNTest/N1SKNSEC.db"
	conn := db.NewDBAdapter(ctx, databasePath)
	testCases := retrieveWordSwitches("")
	report := proofing_rpt.NewAlignSilence(conn)
	verses, _, status := report.Process()
	if status != nil {
		panic(status)
	}
	for _, vs := range verses {
		test := testCases[vs.ScriptId]
		asrText, status1 := report.SelectTranscript(vs.ScriptId)
		if status1 != nil {
			panic(status1)
		}
		refText := report.GetOriginalText(vs)
		diffSample(refText, asrText)
		fmt.Println(vs.LineRef.Description(), test)

		fmt.Print("ZER:")
		for _, wd := range vs.Words {
			//if wd.FAScore == 0.0 {
			//	fmt.Println(wd.Text)
			//}
			//for _, ch := range wd.Chars {}

			for _, ch := range wd.Chars {
				if ch.FAScore == 0.0 {
					fmt.Print(string(ch.Char), " ")
				}
			}
		}
		fmt.Println()
	}
}

func diffSample(refText string, asrText string) {
	fmt.Println("REF:", refText)
	fmt.Println("ASR:", asrText)
	diffs := diff.DiffReplace(refText, asrText)
	for _, d := range diffs {
		fmt.Print("DIF:")
		if d.Type != diff.OpEqual {
			fmt.Print(d.Type, ": |", d.Text, "|")
		}
		fmt.Println()
	}
}
