package accuracy_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/db"
	"github.com/sergi/go-diff/diffmatchpatch"
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
	checkMissingWordResults(verses, testCases)
	checkAddedWordResults(verses, testCases)
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
//     every subsequent word's index.
func checkMissingWordResults(verses []proofing_rpt.Verse2, testCases map[int64]wordSwitch) {
	var foundMissing, foundFalse, foundNot, total float64
	for _, vs := range verses {
		total++
		testWords, ok := testCases[vs.ScriptId]
		if ok {
			var text []string
			for _, wd := range vs.Words {
				text = append(text, wd.Text)
			}
			fmt.Printf("\n%s  %d  %s\n", vs.LineRef.Description(), vs.ScriptId, strings.Join(text, " "))
			fmt.Printf("%s From: %d To: %d\n", testWords.Word, testWords.FromWord, testWords.ToWord)
			var missingResult string
			for _, wd := range vs.Words {
				if wd.FAScore < 0.001 {
					if wd.WordId == testWords.ToWordId {
						missingResult = "FOUND MISSING"
						foundMissing++
					} else {
						missingResult = "MISSING FALSE+"
						foundFalse++
					}
				} else {
					if wd.WordId == testWords.ToWordId {
						missingResult = "NOT FOUND MISS"
						foundNot++

					} else {
						missingResult = "OK"
					}
				}
				if wd.FAScore < 0.5 || wd.WordId == testWords.ToWordId {
					fmt.Printf("%s  %d  %s  %.2f  [", missingResult, wd.WordId, wd.Text, wd.FAScore)
					for _, ch := range wd.Chars {
						fmt.Printf(" %s (%.3f)", string(ch.Char), ch.FAScore)
					}
					fmt.Println("]")
				}
			}
		}
	}
	if total > 0 {
		pctWasMissing := foundMissing / total * 100.0
		pctFoundFalse := foundFalse / total * 100.0
		pctFoundNot := foundNot / total * 100.0
		fmt.Printf("\n*** Total Processed: %0.f  Pct Was Missing %.1f Pct Found False+ %.1f  Pct Not Found %1.f\n",
			total, pctWasMissing, pctFoundFalse, pctFoundNot)
	} else {
		fmt.Println("No Missing Word test results")
	}
}

func checkAddedWordResults(verses []proofing_rpt.Verse2, testCases map[int64]wordSwitch) {
	var foundMissing, foundFalse, foundNot, total float64
	for _, vs := range verses {
		total++
		testWords, ok := testCases[vs.ScriptId]
		if ok {
			var text []string
			for _, wd := range vs.Words {
				lastChar := wd.Chars[len(wd.Chars)-1]
				msg := fmt.Sprintf(" %s (%.2f %t)", wd.Text, lastChar.Silence, wd.IsASR)
				text = append(text, msg)
			}
			fmt.Printf("\n%s  %d  %s\n", vs.LineRef.Description(), vs.ScriptId, strings.Join(text, " "))
			fmt.Printf("%s From: %d To: %d\n", testWords.Word, testWords.FromWord, testWords.ToWord)
			var addedResult string
			var verseHasResult bool
			for _, wd := range vs.Words {
				if wd.IsASR {
					if strings.ToLower(strings.TrimSpace(wd.Text)) == strings.ToLower(testWords.Word) {
						addedResult = "FOUND ADDED"
						foundMissing++
						verseHasResult = true
						displayAdded(addedResult, testWords, wd)
					} else {
						addedResult = "ADDED FALSE+"
						foundFalse++
						verseHasResult = true
						displayAdded(addedResult, testWords, wd)
					}
				}
			}
			if !verseHasResult {
				displayAdded("NOTHING ADDED", testWords, proofing_rpt.Word2{})
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

func displayAdded(addedResult string, testWords wordSwitch, word proofing_rpt.Word2) {
	fmt.Printf("%s  %v  %s  %d  [", addedResult, testWords, word.Text, word.WordId)
	for _, ch := range word.Chars {
		fmt.Printf(" %s (%.3f, %d)", string(ch.Char), ch.Silence, ch.SilenceLong)
	}
	fmt.Println("]")
}

func TestDisplayDifferences(t *testing.T) {
	ctx := context.Background()
	// Set this path to database created by accuracy_test
	databasePath := "/Users/gary/FCBH2024/GaryNTest/N1SKNSEC.db"
	conn := db.NewDBAdapter(ctx, databasePath)
	testCases := retrieveTestCases()
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
