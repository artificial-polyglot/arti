package research

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/cmd/output/proofing_rpt"
	"github.com/artificial-polyglot/arti/db"
)

type Error struct {
	ref      string
	words    int
	total    int
	scriptId int64
}

// This program finds the maxium cutoff that finds all verses
func TestOptimizeCutoff(t *testing.T) {
	ctx := context.Background()
	errors := readErrorFile("N2XNRPMS.txt", t)
	//dbPath := filepath.Join(os.Getenv("HOME"), "Downloads", "arti_output_GaryNTest_N2XNRPMS_qa_align_00002_database_N2XNRPMS.db")
	dbPath := filepath.Join(os.Getenv("HOME"), "Downloads", "arti-output_GaryNTest_N2QAEBSP_qa_align_00004_database_N2QAEBSP.db")
	conn := db.NewDBAdapter(ctx, dbPath)
	scriptIdMap := selectReferences(conn)
	for i := range errors {
		var ok bool
		errors[i].scriptId, ok = scriptIdMap[errors[i].ref]
		if !ok {
			t.Error("Did not find ", errors[i].ref)
		}
	}

	rpt := proofing_rpt.NewProofingRpt(ctx, conn, "qae", false)
	words, status := rpt.SelectWords(1) // 1.0 is no cutoff
	if status != nil {
		t.Fatal(status)
	}
	found := testByAccuracy(words, 1)
	//found := testVersesByMinimum(words, 0.188)
	//found := testVersesByAverage(words, 0.4)
	//found := testVerseByProduct(words, 0.3)
	//found := testVerseByProductAll(words, 0.293)
	hitPct := checkCorrectness(errors, found)
	fmt.Println("PCT", hitPct, " out of", len(found))

}

func readErrorFile(filePath string, t *testing.T) []Error {
	content, _ := os.ReadFile(filePath)
	var errors []Error
	var err error
	for line := range strings.SplitSeq(string(content), "\n") {
		fmt.Println(line)
		if !strings.HasPrefix(line, "#") {
			parts := strings.Split(line, ",")
			if len(parts) != 3 {
				t.Fatal(line + " Did not parse into 3 parts")
			}
			var e Error
			e.ref = parts[0]
			e.words, err = strconv.Atoi(parts[1])
			if err != nil {
				t.Fatal(err)
			}
			e.total, err = strconv.Atoi(parts[2])
			errors = append(errors, e)
		}
	}
	return errors
}

func selectReferences(conn db.DBAdapter) map[string]int64 {
	var result = make(map[string]int64)
	scripts, status := conn.SelectScripts()
	if status != nil {
		fmt.Println(status)
		os.Exit(1)
	}
	for _, scr := range scripts {
		result[scr.BookId+" "+strconv.Itoa(scr.ChapterNum)+":"+scr.VerseStr] = int64(scr.ScriptId)
	}
	return result
}

func testByAccuracy(words [][]proofing_rpt.Word, cutoff float64) map[int64]bool {
	var scriptIds = make(map[int64]bool)
	for _, verse := range words {
		fmt.Println(verse)
		//errCode := proofing_rpt.ComputeAccuracy(verse, cutoff)
		//if errCode >= cutoff {
		//	scriptIds[verse[0].ScriptId] = true
		//}
	}
	return scriptIds
}

func testVersesByMinimum(words [][]proofing_rpt.Word, cutoff float64) map[int64]bool {
	var scriptIds = make(map[int64]bool)
	for _, verse := range words {
		for _, word := range verse {
			if word.Ttype == "W" && word.FaScore < cutoff {
				scriptIds[word.ScriptId] = true
			}
		}
	}
	return scriptIds
}

func testVersesByAverage(words [][]proofing_rpt.Word, cutoff float64) map[int64]bool {
	var scriptIds = make(map[int64]bool)
	for _, verse := range words {
		var sum float64
		var word proofing_rpt.Word
		for _, word = range verse {
			if word.Ttype == "W" {
				sum += word.FaScore
			}
		}
		avg := sum / float64(len(verse))
		if avg < cutoff {
			scriptIds[word.ScriptId] = true
		}
	}
	return scriptIds
}

func testVerseByProduct(words [][]proofing_rpt.Word, cutoff float64) map[int64]bool {
	var scriptIds = make(map[int64]bool)
	for _, verse := range words {
		var product = 1.0
		var word proofing_rpt.Word
		for _, word = range verse {
			if word.Ttype == "W" && word.FaScore < cutoff {
				product *= word.FaScore
			}
		}
		if product < cutoff {
			scriptIds[word.ScriptId] = true
		}
	}
	return scriptIds
}

func testVerseByProductAll(words [][]proofing_rpt.Word, cutoff float64) map[int64]bool {
	var scriptIds = make(map[int64]bool)
	for _, verse := range words {
		var product = 1.0
		var word proofing_rpt.Word
		for _, word = range verse {
			if word.Ttype == "W" {
				product *= word.FaScore
			}
		}
		if product < cutoff {
			scriptIds[word.ScriptId] = true
		}
	}
	return scriptIds
}

func checkCorrectness(knownErrors []Error, found map[int64]bool) float64 {
	var hits int
	for _, kn := range knownErrors {
		_, ok := found[kn.scriptId]
		if ok {
			hits += 1
		} else {
			fmt.Println("Did not find", kn)
		}
	}
	return float64(hits) / float64(len(knownErrors))
}
