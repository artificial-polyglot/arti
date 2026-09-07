package accuracy_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/artificial-polyglot/arti/generic"
)

// This is a shortcut for running the last part of accuracy_test.
// Capture the path of the database created by accuracy_test and use it here.

func TestAccuracyCheck(t *testing.T) {
	ctx := context.Background()
	// Set this path to database created by accuracy_test
	databasePath := "/Users/gary/FCBH2024/GaryNTest/N1SKNSEC.db"
	conn := db.NewDBAdapter(ctx, databasePath)
	testCases := retrieveTestCases()
	report := proofing_rpt.NewAlignSilence(conn)
	verses, _, status := report.Process()
	if status != nil {
		exit(status)
	}
	computeWordError(verses)
	checkResults(verses, testCases)
}

func computeWordError(verses []proofing_rpt.Verse2) {
	for i, vs := range verses {
		for j, wd := range vs.Words {
			var minimum = 1.0
			for _, ch := range wd.Chars {
				if minimum > ch.FAScore {
					minimum = ch.FAScore
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
//     every subsequent word's index.
func checkResults(verses []proofing_rpt.Verse2, testCases map[int64]wordSwitch) {
	var foundMissing, foundFalse, foundNot, foundAdded, total float64
	var errorMissing, errorAdded []generic.VerseRef
	for _, vs := range verses {
		testWords, ok := testCases[vs.ScriptId]
		if ok {
			var text []string
			for _, wd := range vs.Words {
				text = append(text, wd.Text)
			}
			fmt.Printf("\n%s  %d  %s\n", vs.LineRef.Description(), vs.ScriptId, strings.Join(text, " "))
			var missingResult string
			movedWord, _ := findWordById(vs, testWords.MovedWordId)
			for _, wd := range vs.Words {
				if wd.FAScore < 0.5 {
					if wd.WordId == movedWord.WordId {
						missingResult = "FOUND MISSING"
						foundMissing++
					} else {
						missingResult = "MISSING FALSE+"
						foundFalse++
					}
				} else {
					if wd.WordId == movedWord.WordId {
						missingResult = "NOT FOUND MISS"
						foundNot++

					} else {
						missingResult = "OK"
					}
				}
				if wd.FAScore < 0.5 || wd.WordId == movedWord.WordId {
					fmt.Printf("%s  %d  %s  %.2f  [", missingResult, wd.WordId, wd.Text, wd.FAScore)
					for _, ch := range wd.Chars {
						fmt.Printf(" %s (%.2f)", string(ch.Char), ch.FAScore)
					}
					fmt.Println()
				}
			}
			//var missing bool
			total++
			if hasASRWord(vs) {
				foundAdded++
			} else {
				errorAdded = append(errorAdded, vs.LineRef)
			}

			//if found && movedWord.FAScore < 0.1 {
			//	foundMissing++
			//} else {
			//	errorMissing = append(errorMissing, vs.LineRef)
			//}
			//displayVerseDetail(vs, testWords)
		}
	}
	if total > 0 {
		pctWasMissing := foundMissing / total * 100.0
		pctFoundFalse := foundFalse / total * 100.0
		pctFoundNot := foundNot / total * 100.0
		//pctWasAdded := foundAdded / total * 100.0
		fmt.Printf("Total Processed: %0.f  Pct Was Missing %.1f Pct Found False+ %.1f  Pct Not Found %1.f\n",
			total, pctWasMissing, pctFoundFalse, pctFoundNot)
		fmt.Println("Not Found Missing:", errorMissing)
		fmt.Println("Not Found Added:", errorAdded)
	} else {
		fmt.Println("No test results")
	}
}
