package accuracy

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/utility/s3_datastore"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestResults(t *testing.T) {
	ctx := context.Background()
	tests := retrieveTestCases()
	for _, tst := range tests {
		if tst.On {
			AccuracyCheck(ctx, tst)
		}
	}
}

func AccuracyCheck(ctx context.Context, test testCase) {
	conn := LoadDatabase(ctx, test.ResultsDBInput, test.ResultsDBLocal)
	report := proofing_rpt.NewAlignSilence(conn)
	verses, _, status := report.Process()
	if status != nil {
		exit(status)
	}
	outFile, err := os.Create(test.MediaId + ".txt")
	if err != nil {
		exit(err)
	}
	checkMissingWordResults(outFile, verses, test)
	checkAddedWordResults(outFile, verses, test)
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

func computeMinWordError(verses []proofing_rpt.Verse2) {
	for i, vs := range verses {
		for j, wd := range vs.Words {
			var minimum = 1.0
			for c := range wd.Chars {
				ch := wd.Chars[c]
				if len(wd.Chars) > 4 {
					if c > 0 && c < len(wd.Chars)-1 {
						if minimum > ch.FAScore {
							minimum = ch.FAScore
						}
					}
				} else {
					if minimum > ch.FAScore {
						minimum = ch.FAScore
					}
				}
			}
			verses[i].Words[j].FAScore = minimum
		}
	}
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
func checkMissingWordResults(outFile *os.File, verses []proofing_rpt.Verse2, test testCase) {
	testCases := retrieveWordSwitches(test.MediaId)
	var foundMissing, foundFalse, foundNot, total float64
	for _, vs := range verses {
		testWords, ok := testCases[vs.ScriptId]
		if ok {
			isFound := false
			total++
			for j, wd := range vs.Words {
				if wd.FAScore < 0.01 {
					if wd.WordId == testWords.ToWordId {
						isFound = true
						foundMissing++
					} else if isAdjacentToTargetWord(vs.Words, j, testWords.ToWordId) {
						isFound = true
						foundMissing++
					} else {
						foundFalse++
						displayMissingError(outFile, "MISSING FALSE+", vs, wd, testWords)
					}
				}
			}
			if !isFound {
				foundNot++
				displayMissingError(os.Stdout, "NOT FOUND MISS", vs, proofing_rpt.Word2{}, testWords)
				displayMissingError(outFile, "NOT FOUND MISS", vs, proofing_rpt.Word2{}, testWords)
			}
		}
	}
	if total > 0 {
		pctWasMissing := foundMissing / total * 100.0
		pctFoundFalse := foundFalse / total * 100.0
		pctFoundNot := foundNot / total * 100.0
		fmt.Printf("\n*** %s Total Processed: %0.f  Pct Was Missing %.1f Pct Found False+ %.1f  Pct Not Found %1.f\n",
			test.MediaId, total, pctWasMissing, pctFoundFalse, pctFoundNot)
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
			//if strings.ToLower(strings.TrimSpace(prev.Text)) == text {
			return true
		}
	}
	if idx < len(words)-1 {
		next := words[idx+1]
		if next.WordId == targetWordId && strings.ToLower(strings.TrimSpace(next.Text)) == text {
			//if strings.ToLower(strings.TrimSpace(next.Text)) == text {
			return true
		}
	}
	return false
}

func displayMissingError(outFile *os.File, missingResult string, vs proofing_rpt.Verse2, wd proofing_rpt.Word2, tst wordSwitch) {
	var text []string
	for _, wd2 := range vs.Words {
		text = append(text, wd2.Text)
	}
	_, _ = fmt.Fprintf(outFile, "\n%s  %d  %s\n", vs.LineRef.Description(), vs.ScriptId, strings.Join(text, " "))
	_, _ = fmt.Fprintf(outFile, "%s From: %d (%d) To: %d (%d)\n", tst.Word, tst.FromWord, tst.FromWordId, tst.ToWord, tst.ToWordId)
	_, _ = fmt.Fprintf(outFile, "%s  %d  %s  %.3f  [", missingResult, wd.WordId, wd.Text, wd.FAScore)
	for i, wd2 := range vs.Words {
		_, _ = fmt.Fprintf(outFile, "%d %s: (%.3f) %d [", i, wd2.Text, wd2.FAScore, wd2.WordId)
		for _, ch := range wd2.Chars {
			_, _ = fmt.Fprintf(outFile, " %s (%.3f)", string(ch.Char), ch.FAScore)
		}
		_, _ = fmt.Fprintln(outFile, "]")
	}
}

func checkAddedWordResults(outFile *os.File, verses []proofing_rpt.Verse2, test testCase) {
	testCases := retrieveWordSwitches(test.MediaId)
	var foundMissing, foundFalse, foundNot, total float64
	for _, vs := range verses {
		testWords, ok := testCases[vs.ScriptId]
		if ok {
			total++
			nonInsertedWordCount := 0
			if testWords.FromWord > testWords.ToWord {
				nonInsertedWordCount = 1
			}
			var verseHasResult bool
			for _, wd := range vs.Words {
				if wd.IsASR {
					// This is not checking that the expected word is present, because ASR might put something
					// unexpected. It is checking that something has been added to the expected place.
					// The nonInsertedWordCount is intended to step over false positives that have been added
					// Why the testWords.FromWords+2 is needed is a mystery
					if nonInsertedWordCount == testWords.FromWord ||
						nonInsertedWordCount == testWords.FromWord+2 {
						foundMissing++
						verseHasResult = true
					} else {
						foundFalse++
						displayAdded(outFile, "ADDED FALSE+", vs, wd, testWords)
					}
				} else {
					nonInsertedWordCount++
				}
			}
			if !verseHasResult {
				displayAdded(os.Stdout, "NOTHING ADDED", vs, proofing_rpt.Word2{}, testWords)
				displayAdded(outFile, "NOTHING ADDED", vs, proofing_rpt.Word2{}, testWords)
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

func displayAdded(outFile *os.File, addedResult string, vs proofing_rpt.Verse2, wd proofing_rpt.Word2, tst wordSwitch) {
	var text []string
	for _, wd2 := range vs.Words {
		text = append(text, wd2.Text)
	}
	_, _ = fmt.Fprintf(outFile, "\n%s  %d  %s\n", vs.LineRef.Description(), vs.ScriptId, strings.Join(text, " "))
	_, _ = fmt.Fprintf(outFile, "%s From: %d (%d) To: %d (%d)\n", tst.Word, tst.FromWord, tst.FromWordId, tst.ToWord, tst.ToWordId)
	_, _ = fmt.Fprintf(outFile, "%s  %d  %s  %.3f  \n", addedResult, wd.WordId, wd.Text, wd.FAScore)
	for i, wd2 := range vs.Words {
		_, _ = fmt.Fprintf(outFile, "%d %s: (%.3f) %d [", i, wd2.Text, wd2.FAScore, wd2.WordId)
		for _, ch := range wd2.Chars {
			_, _ = fmt.Fprintf(outFile, " %s (%.3f, %.3f, %d)", string(ch.Char), ch.FAScore, ch.Silence, ch.SilenceLong)
		}
		_, _ = fmt.Fprintln(outFile, "]")
	}
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
	diffMatch := diffmatchpatch.New()
	fmt.Println("REF:", refText)
	fmt.Println("ASR:", asrText)
	refText = strings.TrimSpace(refText)
	asrText = strings.TrimSpace(asrText)
	diffs := diffMatch.DiffMain(refText, asrText, false)
	diffs = diffMatch.DiffCleanupSemantic(diffs)
	for _, d := range diffs {
		fmt.Print("DIF:")
		if d.Type != diffmatchpatch.DiffEqual {
			fmt.Print(d.Type, ": |", d.Text, "|")
		}
		fmt.Println()
	}
}
